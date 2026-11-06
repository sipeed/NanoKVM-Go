package application

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"NanoKVM-Server/updateverify"
	"NanoKVM-Server/utils"
	"golang.org/x/sys/unix"
)

const applicationUpdateWorkerFlag = "--application-update-worker"

const (
	applicationUpdateWorkerTimeout      = 45 * time.Minute
	applicationUpdateUnitRuntimeTimeout = applicationUpdateWorkerTimeout + time.Minute
	applicationUpdateInstallTimeout     = 30 * time.Minute
)

type applicationUpdateJob struct {
	OperationID               string
	Source                    UpdateSource
	Version                   string
	ArchivePath               string
	WorkDir                   string
	BaseDir                   string
	KeyRingPath               string
	SkipSignatureVerification bool
}

// RunUpdateWorkerCommand handles the private command used by the transient
// nanokvm-self-update.service. It is exported only because the package main
// must dispatch before starting the HTTP server.
func RunUpdateWorkerCommand(arguments []string) (bool, int) {
	if len(arguments) == 0 || arguments[0] != applicationUpdateWorkerFlag {
		return false, 0
	}
	if len(arguments) != 7 {
		log.Error("invalid application update worker arguments")
		return true, 2
	}
	job := applicationUpdateJob{
		OperationID: arguments[1],
		Source:      UpdateSource(arguments[2]),
		Version:     arguments[3],
		ArchivePath: arguments[4],
		WorkDir:     arguments[5],
		BaseDir:     workerBaseDir(),
		KeyRingPath: workerKeyRingPath(),
	}
	switch arguments[6] {
	case "verify-signatures":
		job.SkipSignatureVerification = false
	case "skip-signatures":
		job.SkipSignatureVerification = true
	default:
		convergeEarlyApplicationUpdateFailure(job, UpdateErrorInterrupted)
		log.Error("invalid application update signature-verification policy")
		return true, 2
	}
	if err := validateApplicationUpdateJob(job, true); err != nil {
		convergeEarlyApplicationUpdateFailure(job, UpdateErrorInterrupted)
		log.Errorf("refusing invalid application update worker job: %v", err)
		return true, 2
	}
	if err := runApplicationUpdateWorker(job, defaultApplicationUpdateDependencies); err != nil {
		convergeEarlyApplicationUpdateFailure(job, UpdateErrorInterrupted)
		log.Errorf("application update worker failed: %v", err)
		return true, 1
	}
	return true, 0
}

func workerBaseDir() string {
	if value := os.Getenv("NANOKVM_UPDATE_BASE_DIR"); value != "" {
		return value
	}
	return TempDir
}

func workerKeyRingPath() string {
	if value := os.Getenv("NANOKVM_UPDATE_KEYRING"); value != "" {
		return value
	}
	return updateKeyRingPath
}

func startOfflineApplicationOperation(
	lease *applicationUpdateLease,
	status ApplicationUpdateStatus,
	archivePath string,
	workDir string,
) error {
	verifySignatures, err := updateSignatureVerificationEnabled()
	if err != nil {
		return fmt.Errorf("read application update signature-verification policy: %w", err)
	}
	return startApplicationUpdateOperation(lease, applicationUpdateJob{
		OperationID:               status.OperationID,
		Source:                    UpdateSourceOffline,
		Version:                   status.Version,
		ArchivePath:               archivePath,
		WorkDir:                   workDir,
		BaseDir:                   TempDir,
		KeyRingPath:               updateKeyRingPath,
		SkipSignatureVerification: !verifySignatures,
	})
}

func startOnlineApplicationOperation(
	lease *applicationUpdateLease,
	status ApplicationUpdateStatus,
	archivePath string,
	workDir string,
) error {
	verifySignatures, err := updateSignatureVerificationEnabled()
	if err != nil {
		return fmt.Errorf("read application update signature-verification policy: %w", err)
	}
	return startApplicationUpdateOperation(lease, applicationUpdateJob{
		OperationID:               status.OperationID,
		Source:                    UpdateSourceOnline,
		Version:                   status.Version,
		ArchivePath:               archivePath,
		WorkDir:                   workDir,
		BaseDir:                   TempDir,
		KeyRingPath:               updateKeyRingPath,
		SkipSignatureVerification: !verifySignatures,
	})
}

func startApplicationUpdateOperation(lease *applicationUpdateLease, job applicationUpdateJob) error {
	if lease == nil {
		return fmt.Errorf("application update lease is missing")
	}
	if err := validateApplicationUpdateJob(job, false); err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate application server executable: %w", err)
	}
	args := applicationUpdateWorkerUnitArgs(executable, job)
	output, err := exec.Command("systemd-run", args...).CombinedOutput()
	if err != nil {
		log.Errorf("failed to start application update operation: %s", strings.TrimSpace(string(output)))
		return fmt.Errorf("start application update operation: %w", err)
	}

	// Type=exec guarantees that the independent worker image has been exec'd.
	// Releasing here cannot open a maintenance race: the fixed unit is already
	// active, and every other participant checks it around the shared flock.
	lease.Release()
	log.Infof("started application update operation %s", job.OperationID)
	return nil
}

func applicationUpdateWorkerUnitArgs(executable string, job applicationUpdateJob) []string {
	signaturePolicy := "verify-signatures"
	if job.SkipSignatureVerification {
		signaturePolicy = "skip-signatures"
	}
	return []string{
		"--unit=" + updateUnitName,
		"--collect",
		"--service-type=exec",
		"--property=RuntimeMaxSec=" + fmt.Sprint(int(applicationUpdateUnitRuntimeTimeout/time.Second)) + "s",
		"--setenv=NANOKVM_UPDATE_OPERATION_ID=" + job.OperationID,
		"--setenv=NANOKVM_UPDATE_VERSION=" + job.Version,
		"--setenv=NANOKVM_UPDATE_SOURCE=" + string(job.Source),
		"--setenv=NANOKVM_UPDATE_BASE_DIR=" + job.BaseDir,
		"--setenv=NANOKVM_UPDATE_KEYRING=" + job.KeyRingPath,
		"--setenv=DEBIAN_FRONTEND=noninteractive",
		"--setenv=NANOKVM_UPDATE_MAX_TRIES=" + fmt.Sprint(maxTries),
		executable,
		applicationUpdateWorkerFlag,
		job.OperationID,
		string(job.Source),
		job.Version,
		job.ArchivePath,
		job.WorkDir,
		signaturePolicy,
	}
}

func validateApplicationUpdateJob(job applicationUpdateJob, production bool) error {
	if job.OperationID == "" || len(job.OperationID) > 128 ||
		!validUpdateSource(job.Source) || !validPublicVersion(job.Version) {
		return fmt.Errorf("invalid application update identity")
	}
	if err := validateLatestVersion(job.Version); err != nil {
		return fmt.Errorf("invalid application update version: %w", err)
	}
	if !filepath.IsAbs(job.ArchivePath) || !filepath.IsAbs(job.WorkDir) ||
		!filepath.IsAbs(job.BaseDir) || !filepath.IsAbs(job.KeyRingPath) {
		return fmt.Errorf("application update paths must be absolute")
	}
	baseDir := filepath.Clean(job.BaseDir)
	workDir := filepath.Clean(job.WorkDir)
	archivePath := filepath.Clean(job.ArchivePath)
	if filepath.Dir(workDir) != baseDir || filepath.Dir(archivePath) != workDir ||
		filepath.Base(archivePath) == "." || filepath.Base(archivePath) == ".." {
		return fmt.Errorf("application update staging path is outside its cache root")
	}
	if production && (baseDir != filepath.Clean(workerBaseDir()) || job.KeyRingPath != workerKeyRingPath()) {
		return fmt.Errorf("application update worker paths do not match production policy")
	}
	workInfo, err := os.Lstat(workDir)
	if err != nil || !workInfo.IsDir() || workInfo.Mode()&os.ModeSymlink != 0 || workInfo.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("application update staging directory is not private")
	}
	archiveInfo, err := os.Lstat(archivePath)
	if err != nil || !archiveInfo.Mode().IsRegular() || archiveInfo.Mode()&os.ModeSymlink != 0 || archiveInfo.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("application update archive is not a private regular file")
	}
	return nil
}

// convergeEarlyApplicationUpdateFailure covers failures before the worker has
// installed its normal failure handler (invalid job, lock acquisition, or
// status identity mismatch). Cleanup is restricted to a private, direct child
// of the production cache root.
func convergeEarlyApplicationUpdateFailure(job applicationUpdateJob, code UpdateErrorCode) {
	if validOperationID(job.OperationID) {
		if status, err := currentApplicationUpdateStatus(); err == nil &&
			status.OperationID == job.OperationID && isApplicationUpdateActive(status.Phase) {
			_ = failApplicationUpdate(job.OperationID, code)
		}
	}
	if safeApplicationWorkerStaging(job) {
		if err := os.RemoveAll(filepath.Clean(job.WorkDir)); err != nil {
			log.Warnf("failed to clean early application update staging: %v", err)
		}
	}
}

func safeApplicationWorkerStaging(job applicationUpdateJob) bool {
	baseDir := filepath.Clean(job.BaseDir)
	workDir := filepath.Clean(job.WorkDir)
	archivePath := filepath.Clean(job.ArchivePath)
	if !filepath.IsAbs(baseDir) || !filepath.IsAbs(workDir) || !filepath.IsAbs(archivePath) ||
		baseDir != filepath.Clean(workerBaseDir()) || filepath.Dir(workDir) != baseDir ||
		filepath.Dir(archivePath) != workDir || workDir == baseDir {
		return false
	}
	info, err := os.Lstat(workDir)
	return err == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0 && info.Mode().Perm()&0o077 == 0
}

func runApplicationUpdateWorker(job applicationUpdateJob, deps applicationUpdateDependencies) (err error) {
	if err := validateApplicationUpdateJob(job, false); err != nil {
		return err
	}
	lease, err := deps.acquireLease()
	if err != nil {
		return err
	}
	defer lease.Release()
	workerContext, cancelWorker := context.WithTimeout(context.Background(), applicationUpdateWorkerTimeout)
	defer cancelWorker()

	preserveWorkDir := false
	defer func() {
		if preserveWorkDir {
			log.Errorf("preserving application update staging after a cache rollback failure")
			return
		}
		if removeErr := os.RemoveAll(job.WorkDir); removeErr != nil {
			log.Warnf("failed to clean application update staging: %v", removeErr)
		}
	}()

	status, statusErr := currentApplicationUpdateStatus()
	if statusErr != nil || status.OperationID != job.OperationID ||
		status.Source != job.Source || status.Version != job.Version ||
		!isApplicationUpdateActive(status.Phase) {
		return fmt.Errorf("application update worker status identity does not match")
	}

	fail := func(code UpdateErrorCode, cause error) error {
		if stateErr := failApplicationUpdate(job.OperationID, code); stateErr != nil {
			log.Errorf("failed to persist application update failure: %v", stateErr)
		}
		if current, currentErr := currentApplicationUpdateStatus(); currentErr == nil {
			broadcastApplicationUpdateStatus(current)
		}
		return cause
	}
	advance := func(phase UpdatePhase, progress int) error {
		// If status.json was malformed while this unit remained active, status
		// recovery conservatively publishes installing. Verification and staging
		// still run below, but must not regress that public status floor.
		if current, currentErr := currentApplicationUpdateStatus(); currentErr == nil &&
			current.OperationID == job.OperationID &&
			updatePhaseRank(current.Phase) > updatePhaseRank(phase) &&
			(current.Phase == UpdatePhaseInstalling || current.Phase == UpdatePhaseRestarting) {
			return nil
		}
		if err := advanceApplicationUpdate(job.OperationID, phase, progress); err != nil {
			return err
		}
		if current, currentErr := currentApplicationUpdateStatus(); currentErr == nil {
			broadcastApplicationUpdateStatus(current)
		}
		return nil
	}

	if err := advance(UpdatePhaseVerifying, 0); err != nil {
		return fail(UpdateErrorInterrupted, err)
	}
	if err := deps.validateArchive(job.ArchivePath, job.Version); err != nil {
		return fail(UpdateErrorInvalidArchive, err)
	}
	extractDir := filepath.Join(job.WorkDir, "extracted")
	bundleDir, err := deps.extract(job.ArchivePath, extractDir)
	if err != nil {
		return fail(applicationWorkerErrorCode(err, UpdateErrorInvalidArchive), err)
	}
	if err := validateExtractedApplicationRoot(extractDir, bundleDir, job.Version); err != nil {
		return fail(UpdateErrorInvalidArchive, err)
	}
	var keys updateverify.KeyRing
	if !job.SkipSignatureVerification {
		keys, err = deps.loadKeyRing(job.KeyRingPath)
		if err != nil {
			return fail(UpdateErrorSignatureInvalid, err)
		}
	}
	policy := applicationVerificationPolicy(job, deps.currentVersion())
	verified, err := deps.verifyBundle(bundleDir, keys, policy)
	if err != nil {
		return fail(applicationVerificationErrorCode(err), err)
	}
	if verified.Version != job.Version {
		return fail(UpdateErrorUnsupportedPackage, fmt.Errorf("verified application version does not match operation"))
	}
	if err := advance(UpdatePhaseVerifying, 100); err != nil {
		return fail(UpdateErrorInterrupted, err)
	}
	if err := advance(UpdatePhaseStaging, 0); err != nil {
		return fail(UpdateErrorInterrupted, err)
	}

	cacheDir, published, err := publishUpdateCache(
		bundleDir, verified.Version, job.BaseDir, deps.rename, deps.exchange,
	)
	if err != nil {
		return fail(applicationWorkerErrorCode(err, UpdateErrorInterrupted), err)
	}
	rollback := func(cause error) error {
		if rollbackErr := published.Rollback(); rollbackErr != nil {
			preserveWorkDir = true
			cause = errors.Join(cause, fmt.Errorf("restore application update cache: %w", rollbackErr))
		}
		return cause
	}
	verifiedCache, err := deps.verifyBundle(cacheDir, keys, policy)
	if err != nil {
		return fail(applicationVerificationErrorCode(err), rollback(err))
	}
	if verifiedCache.Version != job.Version {
		return fail(UpdateErrorUnsupportedPackage, rollback(fmt.Errorf("persisted application version does not match operation")))
	}
	if err := advance(UpdatePhaseStaging, 100); err != nil {
		return fail(UpdateErrorInterrupted, rollback(err))
	}

	// Once installation begins the signed canonical cache is intentionally
	// committed. A partial dpkg failure must not silently restore unrelated
	// bytes under the worker's feet; recovery will re-verify this cache.
	if err := advance(UpdatePhaseInstalling, 0); err != nil {
		return fail(UpdateErrorInterrupted, err)
	}
	installContext, cancelInstall := context.WithTimeout(workerContext, applicationUpdateInstallTimeout)
	err = deps.install(installContext, job.Version, applicationDebPaths(cacheDir, job.Version))
	cancelInstall()
	if err != nil {
		return fail(UpdateErrorInstallFailed, err)
	}
	if err := advance(UpdatePhaseInstalling, 100); err != nil {
		return fail(UpdateErrorInstallFailed, err)
	}
	if err := advance(UpdatePhaseRestarting, 100); err != nil {
		return fail(UpdateErrorInstallFailed, err)
	}
	for _, packageName := range appNames {
		installed, queryErr := deps.packageVersion(packageName)
		if queryErr != nil || normalizeVersion(installed) != job.Version {
			return fail(UpdateErrorInstallFailed, fmt.Errorf("installed application packages do not match target"))
		}
	}
	if err := succeedApplicationUpdate(job.OperationID); err != nil {
		return err
	}
	if current, currentErr := currentApplicationUpdateStatus(); currentErr == nil {
		broadcastApplicationUpdateStatus(current)
	}
	return nil
}

func applicationVerificationPolicy(job applicationUpdateJob, currentVersion string) updateverify.ApplicationPolicy {
	return updateverify.ApplicationPolicy{
		RequestedVersion:          job.Version,
		CurrentVersion:            currentVersion,
		AllowSameVersion:          job.Source == UpdateSourceOffline,
		SkipSignatureVerification: job.SkipSignatureVerification,
	}
}

func applicationWorkerErrorCode(err error, fallback UpdateErrorCode) UpdateErrorCode {
	if classifyApplicationStorageError(err) == UpdateErrorNoSpace {
		return UpdateErrorNoSpace
	}
	return fallback
}

func runSequentialApplicationInstall(ctx context.Context, version string, debPaths []string) error {
	return runApplicationInstallCommand(ctx, sequentialInstallScript, version, debPaths)
}

func runApplicationInstallCommand(ctx context.Context, script, version string, debPaths []string) error {
	arguments := append([]string{updateUnitName, version}, debPaths...)
	command := exec.CommandContext(ctx, "/bin/sh", append([]string{"-c", script}, arguments...)...)
	command.SysProcAttr = &unix.SysProcAttr{Setpgid: true}
	command.Cancel = func() error {
		if command.Process == nil {
			return os.ErrProcessDone
		}
		// The shell is the process-group leader. A negative PID targets every
		// descendant which inherited the group, including a running dpkg.
		if err := unix.Kill(-command.Process.Pid, unix.SIGKILL); err != nil && !errors.Is(err, unix.ESRCH) {
			return err
		}
		return nil
	}
	command.WaitDelay = 5 * time.Second
	command.Env = append(os.Environ(),
		fmt.Sprintf("NANOKVM_UPDATE_MAX_TRIES=%d", maxTries),
		"DEBIAN_FRONTEND=noninteractive",
	)
	output, err := command.CombinedOutput()
	if err != nil {
		log.Errorf("application package installation failed: %s", strings.TrimSpace(string(output)))
		if ctx.Err() != nil {
			return fmt.Errorf("install application packages timed out: %w", ctx.Err())
		}
		return fmt.Errorf("install application packages: %w", err)
	}
	return nil
}

func applicationVerificationErrorCode(err error) UpdateErrorCode {
	if updateverify.Kind(err) == updateverify.ErrorSignature {
		return UpdateErrorSignatureInvalid
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "same-version") {
		return UpdateErrorSameVersion
	}
	if strings.Contains(message, "package") || strings.Contains(message, "architecture") ||
		strings.Contains(message, "version") || strings.Contains(message, "file set") {
		return UpdateErrorUnsupportedPackage
	}
	return UpdateErrorInvalidArchive
}

func downloadOnlineApplicationArtifact(
	url string,
	target string,
	expectedSize int64,
	operationID string,
) error {
	downloadContext, cancelDownload := context.WithTimeout(context.Background(), applicationOnlineDownloadTimeout)
	defer cancelDownload()
	progressReporter := newOnlineApplicationDownloadProgress(func(progress int) error {
		if err := advanceApplicationUpdate(operationID, UpdatePhaseDownloading, progress); err != nil {
			return err
		}
		if current, err := currentApplicationUpdateStatus(); err == nil {
			broadcastApplicationUpdateStatus(current)
		}
		return nil
	})
	for attempt := range maxTries {
		if attempt > 0 {
			select {
			case <-time.After(3 * time.Second):
			case <-downloadContext.Done():
				return fmt.Errorf("download application archive timed out: %w", downloadContext.Err())
			}
		}
		request, err := http.NewRequestWithContext(downloadContext, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		err = downloadApplicationArtifact(
			request,
			target,
			expectedSize,
			applicationHTTPClient,
			progressReporter.report,
		)
		if err == nil {
			progressReporter.complete()
			return nil
		}
		if downloadContext.Err() != nil || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return fmt.Errorf("download application archive timed out: %w", err)
		}
		if errors.Is(err, unix.ENOSPC) || errors.Is(err, unix.EDQUOT) {
			return err
		}
		_ = os.Remove(target)
	}
	return fmt.Errorf("download application archive failed")
}

// onlineApplicationDownloadProgress keeps status persistence and WebSocket
// broadcasts out of the download hot path. Downloads and retries are serial,
// so one reporter can also suppress progress regressions when an attempt
// restarts from byte zero.
type onlineApplicationDownloadProgress struct {
	lastProgress int
	publish      func(int) error
}

func newOnlineApplicationDownloadProgress(publish func(int) error) *onlineApplicationDownloadProgress {
	return &onlineApplicationDownloadProgress{publish: publish}
}

func (reporter *onlineApplicationDownloadProgress) report(total, downloaded int64) {
	if total <= 0 || downloaded <= 0 {
		return
	}
	progress := int(float64(downloaded) / float64(total) * 100)
	if progress > 99 {
		progress = 99
	}
	reporter.publishProgress(progress)
}

func (reporter *onlineApplicationDownloadProgress) complete() {
	reporter.publishProgress(100)
}

func (reporter *onlineApplicationDownloadProgress) publishProgress(progress int) {
	if progress <= reporter.lastProgress ||
		(progress < 99 && progress-reporter.lastProgress < 5) {
		return
	}
	if reporter.publish != nil {
		if err := reporter.publish(progress); err != nil {
			return
		}
	}
	reporter.lastProgress = progress
}

func isApplicationArchiveLimit(err error) bool {
	return errors.Is(err, utils.ErrArchiveLimit)
}
