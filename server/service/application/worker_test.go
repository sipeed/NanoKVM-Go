package application

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"NanoKVM-Server/updateverify"
	"golang.org/x/sys/unix"
)

func TestApplicationWorkerUnitArgsCarryIdentityAndRuntimeLimit(t *testing.T) {
	job := applicationUpdateJob{
		OperationID: "operation-25", Source: UpdateSourceOffline, Version: "0.0.25",
		ArchivePath: "/root/.kvmcache/work/archive.tar.gz", WorkDir: "/root/.kvmcache/work",
		BaseDir: TempDir, KeyRingPath: updateKeyRingPath,
	}
	arguments := applicationUpdateWorkerUnitArgs("/kvmapp/server/NanoKVM-Server", job)
	joined := strings.Join(arguments, "\n")
	for _, expected := range []string{
		"--unit=" + updateUnitName,
		"--service-type=exec",
		"--property=RuntimeMaxSec=2760s",
		"--setenv=NANOKVM_UPDATE_OPERATION_ID=operation-25",
		"--setenv=NANOKVM_UPDATE_VERSION=0.0.25",
		"--setenv=NANOKVM_UPDATE_SOURCE=offline",
		applicationUpdateWorkerFlag,
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("worker unit arguments missing %q: %#v", expected, arguments)
		}
	}
	wantTail := []string{applicationUpdateWorkerFlag, job.OperationID, string(job.Source), job.Version, job.ArchivePath, job.WorkDir, "verify-signatures"}
	if !reflect.DeepEqual(arguments[len(arguments)-len(wantTail):], wantTail) {
		t.Fatalf("worker identity arguments = %#v, want %#v", arguments[len(arguments)-len(wantTail):], wantTail)
	}
}

func TestApplicationWorkerUnitArgsCarryDisabledSignaturePolicy(t *testing.T) {
	job := applicationUpdateJob{
		OperationID: "operation-unsigned", Source: UpdateSourceOnline, Version: "0.0.25",
		ArchivePath: "/root/.kvmcache/work/archive.tar.gz", WorkDir: "/root/.kvmcache/work",
		BaseDir: TempDir, KeyRingPath: updateKeyRingPath, SkipSignatureVerification: true,
	}
	arguments := applicationUpdateWorkerUnitArgs("/kvmapp/server/NanoKVM-Server", job)
	if got := arguments[len(arguments)-1]; got != "skip-signatures" {
		t.Fatalf("worker signature policy = %q, want skip-signatures", got)
	}
}

func TestApplicationVerificationPolicyAllowsSameVersionOnlyOffline(t *testing.T) {
	tests := []struct {
		name      string
		source    UpdateSource
		allowSame bool
	}{
		{name: "offline", source: UpdateSourceOffline, allowSame: true},
		{name: "online", source: UpdateSourceOnline, allowSame: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			job := applicationUpdateJob{Source: test.source, Version: "0.0.25"}
			policy := applicationVerificationPolicy(job, "0.0.25")
			if policy.RequestedVersion != job.Version || policy.CurrentVersion != "0.0.25" {
				t.Fatalf("application policy identity = %+v", policy)
			}
			if policy.AllowSameVersion != test.allowSame {
				t.Fatalf("AllowSameVersion = %t, want %t", policy.AllowSameVersion, test.allowSame)
			}
			if policy.SkipSignatureVerification {
				t.Fatal("default worker policy unexpectedly skips signatures")
			}
		})
	}
}

func TestApplicationVerificationPolicyCarriesImmutableSignatureChoice(t *testing.T) {
	policy := applicationVerificationPolicy(applicationUpdateJob{
		Source: UpdateSourceOffline, Version: "0.0.25", SkipSignatureVerification: true,
	}, "0.0.24")
	if !policy.SkipSignatureVerification {
		t.Fatal("worker policy did not carry disabled signature verification")
	}
}

func TestOnlineApplicationDownloadProgressIsThrottled(t *testing.T) {
	var published []int
	reporter := newOnlineApplicationDownloadProgress(func(progress int) error {
		published = append(published, progress)
		return nil
	})

	for downloaded := int64(0); downloaded <= 110; downloaded++ {
		reporter.report(100, downloaded)
	}
	reporter.complete()
	reporter.complete()

	want := []int{5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55, 60, 65, 70, 75, 80, 85, 90, 95, 99, 100}
	if !reflect.DeepEqual(published, want) {
		t.Fatalf("published download progress = %v, want %v", published, want)
	}
}

func TestOnlineApplicationDownloadProgressDoesNotRegressAcrossRetries(t *testing.T) {
	var published []int
	reporter := newOnlineApplicationDownloadProgress(func(progress int) error {
		published = append(published, progress)
		return nil
	})

	for downloaded := int64(1); downloaded <= 62; downloaded++ {
		reporter.report(100, downloaded)
	}
	// A retry starts again at byte zero. The reporter belongs to the whole
	// operation, so it must suppress repeated and regressing percentages.
	for downloaded := int64(1); downloaded <= 100; downloaded++ {
		reporter.report(100, downloaded)
	}
	reporter.complete()

	for index := 1; index < len(published); index++ {
		if published[index] <= published[index-1] {
			t.Fatalf("download progress regressed or repeated: %v", published)
		}
	}
	if published[len(published)-1] != 100 {
		t.Fatalf("final download progress = %d, want 100", published[len(published)-1])
	}
}

func TestOnlineApplicationDownloadProgressRetriesAfterPublishFailure(t *testing.T) {
	attempts := 0
	reporter := newOnlineApplicationDownloadProgress(func(progress int) error {
		attempts++
		if attempts == 1 {
			return errors.New("persist status")
		}
		if progress != 6 {
			t.Fatalf("retried progress = %d, want 6", progress)
		}
		return nil
	})

	reporter.report(100, 5)
	reporter.report(100, 6)
	if attempts != 2 || reporter.lastProgress != 6 {
		t.Fatalf("publish attempts = %d, last progress = %d", attempts, reporter.lastProgress)
	}
}

func TestApplicationWorkerSkipsKeyRingOnlyWhenPolicyDisablesSignatures(t *testing.T) {
	for _, test := range []struct {
		name      string
		skip      bool
		wantLoads int
	}{
		{name: "verification enabled", wantLoads: 1},
		{name: "verification disabled", skip: true, wantLoads: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			const version = "0.0.25"
			baseDir := applicationStatusTestDirectory(t)
			workDir, err := os.MkdirTemp(baseDir, ".worker-signature-policy-")
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(workDir, 0o700); err != nil {
				t.Fatal(err)
			}
			archivePath := filepath.Join(workDir, "archive.tar.gz")
			if err := os.WriteFile(archivePath, []byte("archive"), 0o600); err != nil {
				t.Fatal(err)
			}

			manager, _ := applicationStatusTestManager(t)
			manager.deps.operationActive = func() (bool, error) { return true, nil }
			originalStatus := applicationStatus
			applicationStatus = manager
			t.Cleanup(func() { applicationStatus = originalStatus })
			status, err := manager.begin(UpdateSourceOnline, version, UpdatePhaseDownloading)
			if err != nil {
				t.Fatal(err)
			}
			job := applicationUpdateJob{
				OperationID: status.OperationID, Source: status.Source, Version: version,
				ArchivePath: archivePath, WorkDir: workDir, BaseDir: baseDir,
				KeyRingPath: filepath.Join(baseDir, "keys.json"), SkipSignatureVerification: test.skip,
			}
			loads := 0
			verifications := 0
			deps := applicationUpdateDependencies{
				validateArchive: func(string, string) error { return nil },
				extract: func(_ string, extractDir string) (string, error) {
					bundleDir := filepath.Join(extractDir, updateCacheDirName(version))
					writeBundlePlaceholders(t, bundleDir, version)
					return bundleDir, nil
				},
				currentVersion: func() string { return "0.0.24" },
				loadKeyRing: func(string) (updateverify.KeyRing, error) {
					loads++
					return updateverify.KeyRing{}, nil
				},
				verifyBundle: func(_ string, _ updateverify.KeyRing, policy updateverify.ApplicationPolicy) (updateverify.VerifiedApplicationBundle, error) {
					verifications++
					if policy.SkipSignatureVerification != test.skip {
						t.Fatalf("bundle policy skip = %t, want %t", policy.SkipSignatureVerification, test.skip)
					}
					return updateverify.VerifiedApplicationBundle{Version: version}, nil
				},
				install:        func(context.Context, string, []string) error { return nil },
				packageVersion: func(string) (string, error) { return version, nil },
				acquireLease:   func() (*applicationUpdateLease, error) { return &applicationUpdateLease{}, nil },
				rename:         os.Rename,
				exchange:       exchangePaths,
			}
			if err := runApplicationUpdateWorker(job, deps); err != nil {
				t.Fatalf("worker failed: %v", err)
			}
			if loads != test.wantLoads {
				t.Fatalf("key ring loads = %d, want %d", loads, test.wantLoads)
			}
			if verifications != 2 {
				t.Fatalf("bundle verifications = %d, want staging and cache re-verification", verifications)
			}
		})
	}
}

func TestApplicationWorkerAllowsSameVersionOnlyOffline(t *testing.T) {
	for _, test := range []struct {
		name          string
		source        UpdateSource
		initialPhase  UpdatePhase
		wantPhase     UpdatePhase
		wantErrorCode UpdateErrorCode
		wantInstall   bool
	}{
		{
			name:   "offline same-version reinstall",
			source: UpdateSourceOffline, initialPhase: UpdatePhaseUploading,
			wantPhase: UpdatePhaseSucceeded, wantInstall: true,
		},
		{
			name:   "online same-version update",
			source: UpdateSourceOnline, initialPhase: UpdatePhaseDownloading,
			wantPhase: UpdatePhaseFailed, wantErrorCode: UpdateErrorSameVersion,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			const version = "0.0.25"
			baseDir := applicationStatusTestDirectory(t)
			workDir, err := os.MkdirTemp(baseDir, ".worker-same-version-")
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(workDir, 0o700); err != nil {
				t.Fatal(err)
			}
			archivePath := filepath.Join(workDir, "archive.tar.gz")
			if err := os.WriteFile(archivePath, []byte("archive"), 0o600); err != nil {
				t.Fatal(err)
			}

			manager, _ := applicationStatusTestManager(t)
			manager.deps.operationActive = func() (bool, error) { return true, nil }
			originalStatus := applicationStatus
			applicationStatus = manager
			t.Cleanup(func() { applicationStatus = originalStatus })
			requestedVersion := version
			if test.source == UpdateSourceOffline {
				requestedVersion = ""
			}
			status, err := manager.begin(test.source, requestedVersion, test.initialPhase)
			if err != nil {
				t.Fatal(err)
			}
			if requestedVersion == "" {
				if err := manager.setVersion(status.OperationID, version); err != nil {
					t.Fatal(err)
				}
			}

			job := applicationUpdateJob{
				OperationID: status.OperationID, Source: test.source, Version: version,
				ArchivePath: archivePath, WorkDir: workDir, BaseDir: baseDir,
				KeyRingPath: filepath.Join(baseDir, "keys.json"),
			}
			installCalled := false
			deps := applicationUpdateDependencies{
				validateArchive: func(string, string) error { return nil },
				extract: func(_ string, extractDir string) (string, error) {
					bundleDir := filepath.Join(extractDir, updateCacheDirName(version))
					writeBundlePlaceholders(t, bundleDir, version)
					return bundleDir, nil
				},
				currentVersion: func() string { return version },
				loadKeyRing:    func(string) (updateverify.KeyRing, error) { return updateverify.KeyRing{}, nil },
				verifyBundle: func(_ string, _ updateverify.KeyRing, policy updateverify.ApplicationPolicy) (updateverify.VerifiedApplicationBundle, error) {
					if policy.RequestedVersion != version || policy.CurrentVersion != version {
						t.Fatalf("same-version worker policy = %+v", policy)
					}
					if !policy.AllowSameVersion {
						return updateverify.VerifiedApplicationBundle{}, errors.New("same-version application update is not allowed")
					}
					return updateverify.VerifiedApplicationBundle{Version: version}, nil
				},
				install: func(context.Context, string, []string) error {
					installCalled = true
					return nil
				},
				packageVersion: func(string) (string, error) { return version, nil },
				acquireLease:   func() (*applicationUpdateLease, error) { return &applicationUpdateLease{}, nil },
				rename:         os.Rename,
				exchange:       exchangePaths,
			}

			err = runApplicationUpdateWorker(job, deps)
			if test.wantPhase == UpdatePhaseSucceeded && err != nil {
				t.Fatalf("offline same-version worker failed: %v", err)
			}
			if test.wantPhase == UpdatePhaseFailed && err == nil {
				t.Fatal("online same-version worker unexpectedly succeeded")
			}
			current, statusErr := manager.current()
			if statusErr != nil {
				t.Fatal(statusErr)
			}
			if current.Phase != test.wantPhase || current.ErrorCode != test.wantErrorCode {
				t.Fatalf("same-version worker status = %+v, want phase %s and error %s", current, test.wantPhase, test.wantErrorCode)
			}
			if installCalled != test.wantInstall {
				t.Fatalf("install called = %t, want %t", installCalled, test.wantInstall)
			}
		})
	}
}

func TestApplicationLockHandoffWaitsForParentRelease(t *testing.T) {
	lockPath := filepath.Join(applicationStatusTestDirectory(t), "handoff.lock")
	parent, err := openAndLockApplicationFile(lockPath, true)
	if err != nil {
		t.Fatal(err)
	}
	acquired := make(chan *os.File, 1)
	errorsChannel := make(chan error, 1)
	go func() {
		worker, lockErr := openAndLockApplicationFile(lockPath, false)
		if lockErr != nil {
			errorsChannel <- lockErr
			return
		}
		acquired <- worker
	}()
	select {
	case worker := <-acquired:
		unlockApplicationFile(worker)
		t.Fatal("worker acquired lock before parent release")
	case err := <-errorsChannel:
		t.Fatalf("worker lock failed: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	unlockApplicationFile(parent)
	select {
	case worker := <-acquired:
		unlockApplicationFile(worker)
	case err := <-errorsChannel:
		t.Fatalf("worker lock failed: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not acquire lock after parent release")
	}
}

func TestApplicationInstallCommandHonorsContextDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := runApplicationInstallCommand(ctx, "sleep 10", "0.0.25", nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("install timeout error = %v", err)
	}
}

func TestApplicationInstallTimeoutKillsEntireProcessGroup(t *testing.T) {
	directory := applicationStatusTestDirectory(t)
	pidPath := filepath.Join(directory, "child.pid")
	script := `sleep 30 & child=$!; printf '%s\n' "$child" >"$1"; wait "$child"`
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	err := runApplicationInstallCommand(ctx, script, pidPath, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("install timeout error = %v", err)
	}
	rawPID, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatal(err)
	}
	childPID, err := strconv.Atoi(strings.TrimSpace(string(rawPID)))
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		err = unix.Kill(childPID, 0)
		if errors.Is(err, unix.ESRCH) {
			break
		}
		if err != nil {
			t.Fatalf("probe child process: %v", err)
		}
		if time.Now().After(deadline) {
			_ = unix.Kill(childPID, unix.SIGKILL)
			t.Fatalf("child process %d survived install timeout", childPID)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestApplicationWorkerRequiresBothInstalledVersions(t *testing.T) {
	for _, test := range []struct {
		name       string
		versions   map[string]string
		installErr error
		wantError  bool
		wantStatus UpdatePhase
	}{
		{name: "both match", versions: map[string]string{kvmCommAppName: "0.0.25", mainAppName: "0.0.25"}, wantStatus: UpdatePhaseSucceeded},
		{name: "kvmcomm mismatch", versions: map[string]string{kvmCommAppName: "0.0.24", mainAppName: "0.0.25"}, wantError: true, wantStatus: UpdatePhaseFailed},
		{name: "server mismatch", versions: map[string]string{kvmCommAppName: "0.0.25", mainAppName: "0.0.24"}, wantError: true, wantStatus: UpdatePhaseFailed},
		{name: "install timeout", versions: map[string]string{}, installErr: context.DeadlineExceeded, wantError: true, wantStatus: UpdatePhaseFailed},
	} {
		t.Run(test.name, func(t *testing.T) {
			baseDir := applicationStatusTestDirectory(t)
			workDir, err := os.MkdirTemp(baseDir, ".worker-")
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(workDir, 0o700); err != nil {
				t.Fatal(err)
			}
			archivePath := filepath.Join(workDir, "archive.tar.gz")
			if err := os.WriteFile(archivePath, []byte("archive"), 0o600); err != nil {
				t.Fatal(err)
			}

			manager, _ := applicationStatusTestManager(t)
			manager.deps.operationActive = func() (bool, error) { return true, nil }
			originalStatus := applicationStatus
			applicationStatus = manager
			t.Cleanup(func() { applicationStatus = originalStatus })
			status, err := manager.begin(UpdateSourceOnline, "0.0.25", UpdatePhaseDownloading)
			if err != nil {
				t.Fatal(err)
			}

			job := applicationUpdateJob{
				OperationID: status.OperationID, Source: status.Source, Version: status.Version,
				ArchivePath: archivePath, WorkDir: workDir, BaseDir: baseDir,
				KeyRingPath: filepath.Join(baseDir, "keys.json"),
			}
			deps := applicationUpdateDependencies{
				validateArchive: func(string, string) error { return nil },
				extract: func(_ string, extractDir string) (string, error) {
					bundleDir := filepath.Join(extractDir, updateCacheDirName(job.Version))
					writeBundlePlaceholders(t, bundleDir, job.Version)
					return bundleDir, nil
				},
				currentVersion: func() string { return "0.0.24" },
				loadKeyRing:    func(string) (updateverify.KeyRing, error) { return updateverify.KeyRing{}, nil },
				verifyBundle: func(string, updateverify.KeyRing, updateverify.ApplicationPolicy) (updateverify.VerifiedApplicationBundle, error) {
					return updateverify.VerifiedApplicationBundle{Version: job.Version}, nil
				},
				install:        func(context.Context, string, []string) error { return test.installErr },
				packageVersion: func(name string) (string, error) { return test.versions[name], nil },
				acquireLease:   func() (*applicationUpdateLease, error) { return &applicationUpdateLease{}, nil },
				rename:         os.Rename, exchange: exchangePaths,
			}
			err = runApplicationUpdateWorker(job, deps)
			if test.wantError != (err != nil) {
				t.Fatalf("worker error = %v, wantError=%v", err, test.wantError)
			}
			current, statusErr := manager.current()
			if statusErr != nil {
				t.Fatal(statusErr)
			}
			if current.Phase != test.wantStatus {
				t.Fatalf("worker status = %+v, want phase %s", current, test.wantStatus)
			}
		})
	}
}

func TestApplicationWorkerPreservesStorageFailureClassification(t *testing.T) {
	tests := []struct {
		name          string
		extractErr    error
		publishErr    error
		wantErrorCode UpdateErrorCode
	}{
		{
			name:          "extract ENOSPC",
			extractErr:    &os.PathError{Op: "write", Path: "extracted", Err: unix.ENOSPC},
			wantErrorCode: UpdateErrorNoSpace,
		},
		{
			name:          "extract EDQUOT",
			extractErr:    &os.PathError{Op: "write", Path: "extracted", Err: unix.EDQUOT},
			wantErrorCode: UpdateErrorNoSpace,
		},
		{
			name:          "invalid archive remains invalid archive",
			extractErr:    errors.New("invalid gzip stream"),
			wantErrorCode: UpdateErrorInvalidArchive,
		},
		{
			name:          "cache publish ENOSPC",
			publishErr:    &os.LinkError{Op: "rename", Old: "staging", New: "cache", Err: unix.ENOSPC},
			wantErrorCode: UpdateErrorNoSpace,
		},
		{
			name:          "cache publish EDQUOT",
			publishErr:    &os.LinkError{Op: "rename", Old: "staging", New: "cache", Err: unix.EDQUOT},
			wantErrorCode: UpdateErrorNoSpace,
		},
		{
			name:          "ordinary cache publish failure remains interrupted",
			publishErr:    errors.New("cache exchange failed"),
			wantErrorCode: UpdateErrorInterrupted,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			baseDir := applicationStatusTestDirectory(t)
			workDir, err := os.MkdirTemp(baseDir, ".worker-storage-")
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(workDir, 0o700); err != nil {
				t.Fatal(err)
			}
			archivePath := filepath.Join(workDir, "archive.tar.gz")
			if err := os.WriteFile(archivePath, []byte("archive"), 0o600); err != nil {
				t.Fatal(err)
			}

			manager, _ := applicationStatusTestManager(t)
			manager.deps.operationActive = func() (bool, error) { return true, nil }
			originalStatus := applicationStatus
			applicationStatus = manager
			t.Cleanup(func() { applicationStatus = originalStatus })
			status, err := manager.begin(UpdateSourceOffline, "0.0.25", UpdatePhaseUploading)
			if err != nil {
				t.Fatal(err)
			}

			job := applicationUpdateJob{
				OperationID: status.OperationID, Source: status.Source, Version: status.Version,
				ArchivePath: archivePath, WorkDir: workDir, BaseDir: baseDir,
				KeyRingPath: filepath.Join(baseDir, "keys.json"),
			}
			deps := applicationUpdateDependencies{
				validateArchive: func(string, string) error { return nil },
				extract: func(_ string, extractDir string) (string, error) {
					if test.extractErr != nil {
						return "", test.extractErr
					}
					bundleDir := filepath.Join(extractDir, updateCacheDirName(job.Version))
					writeBundlePlaceholders(t, bundleDir, job.Version)
					return bundleDir, nil
				},
				currentVersion: func() string { return "0.0.24" },
				loadKeyRing:    func(string) (updateverify.KeyRing, error) { return updateverify.KeyRing{}, nil },
				verifyBundle: func(string, updateverify.KeyRing, updateverify.ApplicationPolicy) (updateverify.VerifiedApplicationBundle, error) {
					return updateverify.VerifiedApplicationBundle{Version: job.Version}, nil
				},
				install:        func(context.Context, string, []string) error { return nil },
				packageVersion: func(string) (string, error) { return job.Version, nil },
				acquireLease:   func() (*applicationUpdateLease, error) { return &applicationUpdateLease{}, nil },
				rename: func(string, string) error {
					return test.publishErr
				},
				exchange: exchangePaths,
			}

			if err := runApplicationUpdateWorker(job, deps); err == nil {
				t.Fatal("expected worker failure")
			}
			current, err := manager.current()
			if err != nil {
				t.Fatal(err)
			}
			if current.Phase != UpdatePhaseFailed || current.ErrorCode != test.wantErrorCode {
				t.Fatalf("worker status = %+v, want error %s", current, test.wantErrorCode)
			}
		})
	}
}

func TestEarlyWorkerFailureMarksStatusAndCleansOnlyPrivateStaging(t *testing.T) {
	baseDir := applicationStatusTestDirectory(t)
	t.Setenv("NANOKVM_UPDATE_BASE_DIR", baseDir)
	workDir, err := os.MkdirTemp(baseDir, ".worker-early-")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(workDir, 0o700); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(workDir, "archive.tar.gz")
	if err := os.WriteFile(archivePath, []byte("archive"), 0o600); err != nil {
		t.Fatal(err)
	}
	manager, _ := applicationStatusTestManager(t)
	originalStatus := applicationStatus
	applicationStatus = manager
	t.Cleanup(func() { applicationStatus = originalStatus })
	status, err := manager.begin(UpdateSourceOffline, "", UpdatePhaseUploading)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.setVersion(status.OperationID, "0.0.25"); err != nil {
		t.Fatal(err)
	}
	job := applicationUpdateJob{
		OperationID: status.OperationID, Source: status.Source, Version: "0.0.25",
		ArchivePath: archivePath, WorkDir: workDir, BaseDir: baseDir, KeyRingPath: filepath.Join(baseDir, "keys.json"),
	}
	convergeEarlyApplicationUpdateFailure(job, UpdateErrorInterrupted)
	if _, err := os.Lstat(workDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("early failure staging still exists: %v", err)
	}
	current, err := manager.current()
	if err != nil {
		t.Fatal(err)
	}
	if current.Phase != UpdatePhaseFailed || current.ErrorCode != UpdateErrorInterrupted {
		t.Fatalf("early failure status = %+v", current)
	}
}
