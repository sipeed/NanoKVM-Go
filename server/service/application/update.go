package application

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/updateverify"
	"NanoKVM-Server/utils"
	"golang.org/x/sys/unix"
)

const maxTries = 3
const updateUnitName = "nanokvm-self-update"
const updateCacheDirPrefix = "nanokvm-go_"
const applicationDownloadSafetyMargin int64 = 32 << 20
const applicationOnlineDownloadTimeout = 30 * time.Minute

const sequentialInstallScript = `
set -u

max_tries="${NANOKVM_UPDATE_MAX_TRIES:-3}"

install_deb() {
	deb="$1"
	try=0

	while [ "$try" -lt "$max_tries" ]; do
		if [ "$try" -gt 0 ]; then
			echo "retry installing ${deb}: ${try}/${max_tries}"
			sleep "$try"
		fi

		echo "installing ${deb}"
		if /usr/bin/dpkg --force-confnew --force-confmiss -i "$deb"; then
			echo "installed ${deb}"
			return 0
		fi

		try=$((try + 1))
	done

	echo "failed to install ${deb}" >&2
	return 1
}

target_version="$1"
shift

install_failed=0
for deb in "$@"; do
	install_deb "$deb" || install_failed=1
done

if [ "$install_failed" -ne 0 ]; then
	exit 1
fi

verify_installed_version() {
	package="$1"
	installed="$(/usr/bin/dpkg-query -W -f='${Version}' "$package")" || {
		echo "failed to query installed ${package} version" >&2
		return 1
	}
	if [ "$installed" != "$target_version" ]; then
		echo "installed ${package} version ${installed} does not match ${target_version}" >&2
		return 1
	fi
}

verify_installed_version kvmcomm || exit 1
verify_installed_version nanokvm-go || exit 1
`

var (
	updateMutex sync.Mutex
	isUpdating  bool
)

type applicationUpdateDependencies struct {
	getLatest               func() (*Latest, error)
	verifyTransportArtifact func(string, updateverify.TransportMetadata) error
	validateArchive         func(string, string) error
	extract                 func(string, string) (string, error)
	currentVersion          func() string
	loadKeyRing             func(string) (updateverify.KeyRing, error)
	verifyBundle            func(string, updateverify.KeyRing, updateverify.ApplicationPolicy) (updateverify.VerifiedApplicationBundle, error)
	install                 func(context.Context, string, []string) error
	packageVersion          func(string) (string, error)
	acquireLease            func() (*applicationUpdateLease, error)
	rename                  func(string, string) error
	exchange                func(string, string) error
}

var defaultApplicationUpdateDependencies = applicationUpdateDependencies{
	getLatest:               getLatest,
	verifyTransportArtifact: updateverify.VerifyTransportArtifact,
	validateArchive:         validateApplicationArchive,
	extract:                 utils.UnTarGz,
	currentVersion:          CurrentVersion,
	loadKeyRing:             updateverify.LoadKeyRing,
	verifyBundle:            updateverify.VerifyApplicationBundle,
	install:                 runSequentialApplicationInstall,
	packageVersion:          getPackageVersion,
	acquireLease:            acquireApplicationUpdateWorkerLease,
	rename:                  os.Rename,
	exchange:                exchangePaths,
}

type publishedUpdateCache struct {
	targetDir string
	oldDir    string
	exchange  func(string, string) error
	hadTarget bool
}

func (published *publishedUpdateCache) Rollback() error {
	if published == nil {
		return nil
	}
	if !published.hadTarget {
		return os.RemoveAll(published.targetDir)
	}
	return published.exchange(published.oldDir, published.targetDir)
}

func debFileName(appName string, version string) string {
	return fmt.Sprintf("%s_%s_%s.deb", appName, version, packageArch)
}

func updateCacheDirName(version string) string {
	return updateCacheDirPrefix + version
}

func updateCacheDir(baseDir string, version string) string {
	return filepath.Join(baseDir, updateCacheDirName(version))
}

func (s *Service) Update(c *gin.Context) {
	var rsp proto.Response

	// Resolve discovery metadata before admission only to learn the candidate
	// version. It remains untrusted and is rechecked against the exact archive
	// bytes and signed bundle before installation.
	latest, err := defaultApplicationUpdateDependencies.getLatest()
	if err != nil {
		rsp.ErrRspWithData(c, -2, "update_failed", publicFailedUpdateStatus(UpdateErrorInterrupted))
		return
	}
	lease, status, err := beginApplicationUpdateAdmission(
		UpdateSourceOnline, latest.Version, UpdatePhaseDownloading,
	)
	if err != nil {
		code := UpdateErrorInterrupted
		if errors.Is(err, errApplicationUpdateBusy) {
			code = UpdateErrorBusy
		}
		rsp.ErrRspWithData(c, -1, string(code), publicFailedUpdateStatus(code))
		return
	}
	handedOff := false
	defer func() {
		if !handedOff {
			lease.Release()
		}
	}()

	if err := ensureApplicationCacheRoot(TempDir); err != nil {
		code := classifyApplicationStorageError(err)
		_ = failApplicationUpdate(status.OperationID, code)
		rsp.ErrRspWithData(c, -2, string(code), statusAfterFailure(status))
		return
	}
	workDir, err := os.MkdirTemp(TempDir, ".application-update.*")
	if err != nil {
		code := classifyApplicationStorageError(err)
		_ = failApplicationUpdate(status.OperationID, code)
		rsp.ErrRspWithData(c, -2, string(code), statusAfterFailure(status))
		return
	}
	if err := os.Chmod(workDir, 0o700); err != nil {
		_ = os.RemoveAll(workDir)
		_ = failApplicationUpdate(status.OperationID, UpdateErrorInterrupted)
		rsp.ErrRspWithData(c, -2, string(UpdateErrorInterrupted), statusAfterFailure(status))
		return
	}
	cleanupWorkDir := true
	defer func() {
		if cleanupWorkDir {
			_ = os.RemoveAll(workDir)
		}
	}()

	archivePath := filepath.Join(workDir, latest.Name)
	if err := downloadOnlineApplicationArtifact(
		latest.Url, archivePath, latest.Size, status.OperationID,
	); err != nil {
		code := classifyApplicationStorageError(err)
		_ = failApplicationUpdate(status.OperationID, code)
		rsp.ErrRspWithData(c, -2, string(code), statusAfterFailure(status))
		return
	}
	metadata := updateverify.TransportMetadata{
		Name:   latest.Name,
		SHA512: latest.Sha512,
		Size:   latest.Size,
	}
	if err := defaultApplicationUpdateDependencies.verifyTransportArtifact(archivePath, metadata); err != nil {
		_ = failApplicationUpdate(status.OperationID, UpdateErrorInvalidArchive)
		rsp.ErrRspWithData(c, -2, string(UpdateErrorInvalidArchive), statusAfterFailure(status))
		return
	}
	if err := startOnlineApplicationOperation(lease, status, archivePath, workDir); err != nil {
		_ = failApplicationUpdate(status.OperationID, UpdateErrorInstallStartFailed)
		rsp.ErrRspWithData(c, -2, string(UpdateErrorInstallStartFailed), statusAfterFailure(status))
		return
	}
	handedOff = true
	cleanupWorkDir = false
	current := applicationUpdateStatusOrFallback(status)
	rsp.OkRspWithData(c, current)
	log.Debugf("update service success")
}

func ensureApplicationCacheRoot(directory string) error {
	if !filepath.IsAbs(directory) {
		return fmt.Errorf("application cache root must be absolute")
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	info, err := os.Lstat(directory)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("application cache root is not a real directory")
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return err
	}
	return nil
}

func classifyApplicationStorageError(err error) UpdateErrorCode {
	if errors.Is(err, unix.ENOSPC) || errors.Is(err, unix.EDQUOT) {
		return UpdateErrorNoSpace
	}
	return UpdateErrorInterrupted
}

func publicFailedUpdateStatus(code UpdateErrorCode) ApplicationUpdateStatus {
	return ApplicationUpdateStatus{Phase: UpdatePhaseFailed, ErrorCode: code}
}

func statusAfterFailure(fallback ApplicationUpdateStatus) ApplicationUpdateStatus {
	current, err := currentApplicationUpdateStatus()
	if err == nil {
		return current
	}
	fallback.Phase = UpdatePhaseFailed
	fallback.ErrorCode = UpdateErrorInterrupted
	return fallback
}

func applicationUpdateStatusOrFallback(fallback ApplicationUpdateStatus) ApplicationUpdateStatus {
	current, err := currentApplicationUpdateStatus()
	if err != nil {
		return fallback
	}
	return current
}

func validateExtractedApplicationRoot(extractDir, extractedDir, version string) error {
	extractRoot, err := filepath.Abs(extractDir)
	if err != nil {
		return err
	}
	actualDir, err := filepath.Abs(extractedDir)
	if err != nil {
		return err
	}
	expectedDir := filepath.Join(extractRoot, updateCacheDirName(version))
	if actualDir != expectedDir {
		return fmt.Errorf("archive top-level directory does not match requested version")
	}
	rootInfo, err := os.Lstat(extractRoot)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("archive extraction root is not a real directory")
	}
	dirInfo, err := os.Lstat(actualDir)
	if err != nil || !dirInfo.IsDir() || dirInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("archive top-level entry is not a real directory")
	}
	entries, err := os.ReadDir(extractRoot)
	if err != nil {
		return err
	}
	if len(entries) != 1 || entries[0].Name() != updateCacheDirName(version) || !entries[0].IsDir() {
		return fmt.Errorf("archive must contain exactly one application directory")
	}
	return nil
}

func validateApplicationArchive(archivePath, version string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	closed := false
	defer func() {
		if !closed {
			_ = gzipReader.Close()
		}
	}()

	expectedRoot := updateCacheDirName(version)
	seen := make(map[string]struct{})
	kinds := make(map[string]bool)
	envelope := &io.LimitedReader{R: gzipReader, N: utils.DefaultTarMaxEnvelopeBytes + 1}
	reader := tar.NewReader(envelope)
	var entries int64
	var expanded int64
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		entries++
		if entries > utils.DefaultTarMaxEntries {
			return fmt.Errorf("application archive entry count exceeds the safety limit")
		}
		if header.Size < 0 || header.Size > utils.DefaultTarMaxFileBytes {
			return fmt.Errorf("application archive entry exceeds the file size limit")
		}
		if header.Size > utils.DefaultTarMaxExpandedBytes-expanded {
			return fmt.Errorf("application archive exceeds the expanded size limit")
		}
		expanded += header.Size
		isDirectory := header.Typeflag == tar.TypeDir
		if !isDirectory && header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			return fmt.Errorf("application archive contains an unsupported entry")
		}
		name, err := canonicalApplicationArchivePath(header.Name, isDirectory)
		if err != nil {
			return err
		}
		if name != expectedRoot && !strings.HasPrefix(name, expectedRoot+"/") {
			return fmt.Errorf("application archive contains an unexpected top-level path")
		}
		if _, exists := seen[name]; exists {
			return fmt.Errorf("application archive contains a duplicate path")
		}
		seen[name] = struct{}{}
		kinds[name] = isDirectory
	}
	if len(seen) == 0 {
		return fmt.Errorf("application archive is empty")
	}
	if directory, exists := kinds[expectedRoot]; !exists || !directory {
		return fmt.Errorf("application archive top level must be an explicit directory")
	}
	for name := range seen {
		parts := strings.Split(name, "/")
		for index := 1; index < len(parts); index++ {
			parent := strings.Join(parts[:index], "/")
			if directory, exists := kinds[parent]; exists && !directory {
				return fmt.Errorf("application archive file conflicts with a descendant")
			}
		}
	}
	if _, err := io.Copy(io.Discard, envelope); err != nil {
		return err
	}
	if envelope.N == 0 {
		return fmt.Errorf("application archive exceeds the stream size limit")
	}
	if err := gzipReader.Close(); err != nil {
		return err
	}
	closed = true
	return nil
}

func canonicalApplicationArchivePath(name string, directory bool) (string, error) {
	if name == "" || strings.HasPrefix(name, "/") || strings.ContainsAny(name, "\\\x00") {
		return "", fmt.Errorf("application archive contains an unsafe path")
	}
	canonical := name
	if directory && strings.HasSuffix(canonical, "/") {
		canonical = strings.TrimSuffix(canonical, "/")
	}
	if canonical == "" || path.IsAbs(canonical) || path.Clean(canonical) != canonical {
		return "", fmt.Errorf("application archive contains a non-canonical path")
	}
	for _, component := range strings.Split(canonical, "/") {
		if component == "" || component == "." || component == ".." {
			return "", fmt.Errorf("application archive contains a non-canonical path")
		}
	}
	return canonical, nil
}

func publishUpdateCache(
	extractedDir, version, baseDir string,
	rename func(string, string) error,
	exchange func(string, string) error,
) (string, *publishedUpdateCache, error) {
	targetDir := updateCacheDir(baseDir, version)
	published := &publishedUpdateCache{targetDir: targetDir, oldDir: extractedDir, exchange: exchange}
	if _, err := os.Lstat(targetDir); err == nil {
		if err := exchange(extractedDir, targetDir); err != nil {
			return "", nil, err
		}
		published.hadTarget = true
		return targetDir, published, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", nil, err
	}

	if err := rename(extractedDir, targetDir); err != nil {
		return "", nil, err
	}

	return targetDir, published, nil
}

func exchangePaths(left, right string) error {
	return unix.Renameat2(unix.AT_FDCWD, left, unix.AT_FDCWD, right, unix.RENAME_EXCHANGE)
}

func downloadApplicationArtifact(
	req *http.Request,
	target string,
	expectedSize int64,
	client *http.Client,
	callback func(int64, int64),
) (err error) {
	if expectedSize <= 0 {
		return fmt.Errorf("invalid expected application archive size")
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return err
	}
	budget, err := applicationDownloadBudget(filepath.Dir(target))
	if err != nil {
		return err
	}
	if expectedSize > budget {
		return fmt.Errorf("insufficient disk space for application update: %w", unix.ENOSPC)
	}

	response, err := doApplicationRequestWithHTTPFallback(client, req)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("application update server returned status %d", response.StatusCode)
	}
	if response.ContentLength >= 0 && response.ContentLength != expectedSize {
		return fmt.Errorf("application archive Content-Length does not match metadata")
	}

	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		if err != nil {
			_ = os.Remove(target)
		}
	}()

	limited := &io.LimitedReader{R: response.Body, N: expectedSize + 1}
	progress := &applicationDownloadProgress{
		total:    expectedSize,
		callback: callback,
	}
	written, err := io.Copy(file, io.TeeReader(limited, progress))
	if err != nil {
		return err
	}
	if written != expectedSize || limited.N == 0 {
		return fmt.Errorf("application archive size does not match metadata")
	}
	if err := file.Sync(); err != nil {
		return err
	}
	return nil
}

type applicationDownloadProgress struct {
	total      int64
	downloaded int64
	callback   func(int64, int64)
}

func (progress *applicationDownloadProgress) Write(data []byte) (int, error) {
	progress.downloaded += int64(len(data))
	if progress.callback != nil {
		progress.callback(progress.total, progress.downloaded)
	}
	return len(data), nil
}

func applicationDownloadBudget(directory string) (int64, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(directory, &stat); err != nil {
		return 0, err
	}
	available := uint64(stat.Bavail)
	blockSize := uint64(stat.Bsize)
	if blockSize != 0 && available > ^uint64(0)/blockSize {
		return 0, fmt.Errorf("available disk space overflows")
	}
	bytesAvailable := available * blockSize
	if bytesAvailable <= uint64(applicationDownloadSafetyMargin) {
		return 0, fmt.Errorf("insufficient disk space for application update: %w", unix.ENOSPC)
	}
	budget := bytesAvailable - uint64(applicationDownloadSafetyMargin)
	if budget > uint64(^uint64(0)>>1) {
		return int64(^uint64(0) >> 1), nil
	}
	return int64(budget), nil
}

func applicationDebPaths(dir, verifiedVersion string) []string {
	return []string{
		filepath.Join(dir, debFileName(kvmCommAppName, verifiedVersion)),
		filepath.Join(dir, debFileName(mainAppName, verifiedVersion)),
	}
}
