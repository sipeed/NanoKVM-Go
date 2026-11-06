package application

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"NanoKVM-Server/proto"
	"github.com/gin-gonic/gin"
)

func applicationStatusTestDirectory(t *testing.T) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate application status test source")
	}
	parent := filepath.Join(filepath.Dir(source), "..", "..", "..", "build", "test-tmp")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	directory, err := os.MkdirTemp(parent, ".application-status-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(directory); err != nil {
			t.Errorf("remove application status test directory: %v", err)
		}
	})
	return directory
}

func applicationStatusTestManager(t *testing.T) (*applicationUpdateStatusManager, *time.Time) {
	t.Helper()
	now := time.Date(2026, 8, 12, 4, 5, 6, 0, time.UTC)
	manager := newApplicationUpdateStatusManager(
		filepath.Join(applicationStatusTestDirectory(t), "run", "status.json"),
		applicationUpdateStatusDependencies{
			now:             func() time.Time { return now },
			newOperationID:  func() string { return "operation-1" },
			inspectUnit:     func() (applicationUpdateUnitStatus, error) { return applicationUpdateUnitStatus{}, nil },
			operationActive: func() (bool, error) { return false, nil },
			packageVersion:  func(string) (string, error) { return "", errors.New("not installed") },
		},
	)
	return manager, &now
}

func TestApplicationUpdateStatusLifecycleIsStrictAndAtomic(t *testing.T) {
	manager, now := applicationStatusTestManager(t)
	status, err := manager.begin(UpdateSourceOnline, "v0.0.25", UpdatePhaseDownloading)
	if err != nil {
		t.Fatal(err)
	}
	if status.OperationID != "operation-1" || status.Version != "0.0.25" || status.Phase != UpdatePhaseDownloading {
		t.Fatalf("unexpected initial status: %+v", status)
	}
	if err := manager.advance(status.OperationID, UpdatePhaseDownloading, 37); err != nil {
		t.Fatal(err)
	}
	if err := manager.advance(status.OperationID, UpdatePhaseDownloading, 36); !errors.Is(err, errApplicationUpdatePhase) {
		t.Fatalf("progress regression error = %v", err)
	}
	if err := manager.advance(status.OperationID, UpdatePhaseStaging, 40); !errors.Is(err, errApplicationUpdatePhase) {
		t.Fatalf("phase skip error = %v", err)
	}
	*now = now.Add(time.Second)
	if err := manager.advance(status.OperationID, UpdatePhaseVerifying, 40); err != nil {
		t.Fatal(err)
	}
	if err := manager.advance(status.OperationID, UpdatePhaseDownloading, 50); !errors.Is(err, errApplicationUpdatePhase) {
		t.Fatalf("phase regression error = %v", err)
	}
	if err := manager.advance("stale-operation", UpdatePhaseStaging, 50); !errors.Is(err, errApplicationUpdateConflict) {
		t.Fatalf("stale operation error = %v", err)
	}
	if err := manager.fail(status.OperationID, UpdateErrorSignatureInvalid); err != nil {
		t.Fatal(err)
	}
	if err := manager.advance(status.OperationID, UpdatePhaseStaging, 50); !errors.Is(err, errApplicationUpdatePhase) {
		t.Fatalf("terminal transition error = %v", err)
	}

	stored, err := os.ReadFile(manager.path)
	if err != nil {
		t.Fatal(err)
	}
	var persisted ApplicationUpdateStatus
	if err := json.Unmarshal(stored, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.Phase != UpdatePhaseFailed || persisted.ErrorCode != UpdateErrorSignatureInvalid {
		t.Fatalf("unexpected persisted status: %+v", persisted)
	}
	info, err := os.Stat(manager.path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("status mode = %o, want 600", info.Mode().Perm())
	}
	entries, err := os.ReadDir(filepath.Dir(manager.path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Name() != "status.json" || entries[1].Name() != "status.json.lock" {
		t.Fatalf("status directory entries = %v", entries)
	}
	lockInfo, err := os.Stat(manager.lockPath)
	if err != nil {
		t.Fatal(err)
	}
	if lockInfo.Mode().Perm() != 0o600 {
		t.Fatalf("status lock mode = %o, want 600", lockInfo.Mode().Perm())
	}
}

func TestApplicationUpdateStatusRejectsNonCanonicalVersion(t *testing.T) {
	manager, _ := applicationStatusTestManager(t)
	for _, version := range []string{"0.00.25", "0.0.25/private", "0.0.25\nsecret"} {
		if _, err := manager.begin(UpdateSourceOnline, version, UpdatePhaseDownloading); !errors.Is(err, errApplicationUpdatePhase) {
			t.Fatalf("version %q error = %v", version, err)
		}
	}
}

func TestOfflineUpdateMayDiscoverVersionAfterUploadStarts(t *testing.T) {
	manager, _ := applicationStatusTestManager(t)
	status, err := manager.begin(UpdateSourceOffline, "", UpdatePhaseUploading)
	if err != nil {
		t.Fatal(err)
	}
	if status.Version != "" {
		t.Fatalf("initial version = %q", status.Version)
	}
	if err := manager.setVersion(status.OperationID, "v0.0.25"); err != nil {
		t.Fatal(err)
	}
	if err := manager.setVersion(status.OperationID, "0.0.26"); !errors.Is(err, errApplicationUpdatePhase) {
		t.Fatalf("version replacement error = %v", err)
	}
	if err := manager.advance(status.OperationID, UpdatePhaseVerifying, 100); err != nil {
		t.Fatal(err)
	}
	if err := manager.advance(status.OperationID, UpdatePhaseStaging, 100); err != nil {
		t.Fatal(err)
	}
}

func TestApplicationUpdateStatusBeginRejectsLivePersistedOperation(t *testing.T) {
	manager, _ := applicationStatusTestManager(t)
	status, err := manager.begin(UpdateSourceOnline, "0.0.25", UpdatePhaseDownloading)
	if err != nil {
		t.Fatal(err)
	}
	manager.deps.inspectUnit = func() (applicationUpdateUnitStatus, error) {
		return applicationUpdateUnitStatus{
			Exists: true, ActiveState: "active", OperationID: status.OperationID,
			Version: status.Version, Source: status.Source,
		}, nil
	}

	restarted := newApplicationUpdateStatusManager(manager.path, manager.deps)
	current, err := restarted.current()
	if err != nil {
		t.Fatal(err)
	}
	if current.Phase != UpdatePhaseDownloading {
		t.Fatalf("active unit changed persisted phase = %s", current.Phase)
	}
	if _, err := restarted.begin(UpdateSourceOffline, "", UpdatePhaseUploading); !errors.Is(err, errApplicationUpdateBusy) {
		t.Fatalf("second begin error = %v", err)
	}
	if current.OperationID != status.OperationID {
		t.Fatalf("operation changed from %q to %q", status.OperationID, current.OperationID)
	}
}

func TestApplicationUpdateStatusRecoveryUsesUnitAndBothInstalledVersions(t *testing.T) {
	for _, test := range []struct {
		name      string
		unit      applicationUpdateUnitStatus
		versions  map[string]string
		wantPhase UpdatePhase
		wantError UpdateErrorCode
	}{
		{
			name: "successful unit and matching packages",
			unit: applicationUpdateUnitStatus{
				Exists: true, ActiveState: "inactive", Result: "success", ExecMainStatus: 0, HasExecStatus: true,
				OperationID: "operation-1", Version: "0.0.25", Source: UpdateSourceOnline,
			},
			versions:  map[string]string{kvmCommAppName: "0.0.25", mainAppName: "0.0.25"},
			wantPhase: UpdatePhaseSucceeded,
		},
		{
			name: "unit success but one package mismatches",
			unit: applicationUpdateUnitStatus{
				Exists: true, ActiveState: "inactive", Result: "success", ExecMainStatus: 0, HasExecStatus: true,
				OperationID: "operation-1", Version: "0.0.25", Source: UpdateSourceOnline,
			},
			versions:  map[string]string{kvmCommAppName: "0.0.24", mainAppName: "0.0.25"},
			wantPhase: UpdatePhaseFailed,
			wantError: UpdateErrorInstallFailed,
		},
		{
			name: "failed unit cannot be hidden by matching package versions",
			unit: applicationUpdateUnitStatus{
				Exists: true, ActiveState: "failed", Result: "exit-code", ExecMainStatus: 1, HasExecStatus: true,
				OperationID: "operation-1", Version: "0.0.25", Source: UpdateSourceOnline,
			},
			versions:  map[string]string{kvmCommAppName: "0.0.25", mainAppName: "0.0.25"},
			wantPhase: UpdatePhaseFailed,
			wantError: UpdateErrorInstallFailed,
		},
		{
			name:      "server died before unit started",
			unit:      applicationUpdateUnitStatus{},
			versions:  map[string]string{},
			wantPhase: UpdatePhaseFailed,
			wantError: UpdateErrorInterrupted,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			manager, _ := applicationStatusTestManager(t)
			manager.deps.inspectUnit = func() (applicationUpdateUnitStatus, error) { return test.unit, nil }
			manager.deps.packageVersion = func(name string) (string, error) {
				version, ok := test.versions[name]
				if !ok {
					return "", errors.New("private dpkg details")
				}
				return version, nil
			}
			status, err := manager.begin(UpdateSourceOnline, "0.0.25", UpdatePhaseDownloading)
			if err != nil {
				t.Fatal(err)
			}
			if test.unit.completed() {
				if err := manager.advance(status.OperationID, UpdatePhaseVerifying, 100); err != nil {
					t.Fatal(err)
				}
				if err := manager.advance(status.OperationID, UpdatePhaseStaging, 100); err != nil {
					t.Fatal(err)
				}
				if err := manager.advance(status.OperationID, UpdatePhaseInstalling, 100); err != nil {
					t.Fatal(err)
				}
			}

			restarted := newApplicationUpdateStatusManager(manager.path, manager.deps)
			current, err := restarted.current()
			if err != nil {
				t.Fatal(err)
			}
			if current.Phase != test.wantPhase || current.ErrorCode != test.wantError {
				t.Fatalf("recovered status = %+v", current)
			}
			data, err := os.ReadFile(manager.path)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), "private dpkg details") {
				t.Fatal("internal package query error leaked into status file")
			}
		})
	}
}

func TestApplicationUpdateStatusInspectionFailurePreservesActiveStatus(t *testing.T) {
	manager, _ := applicationStatusTestManager(t)
	status, err := manager.begin(UpdateSourceOffline, "", UpdatePhaseUploading)
	if err != nil {
		t.Fatal(err)
	}
	manager.deps.inspectUnit = func() (applicationUpdateUnitStatus, error) {
		return applicationUpdateUnitStatus{}, errors.New("private systemd output")
	}
	restarted := newApplicationUpdateStatusManager(manager.path, manager.deps)
	current, err := restarted.current()
	if err != nil {
		t.Fatal(err)
	}
	if current.OperationID != status.OperationID || current.Phase != UpdatePhaseUploading {
		t.Fatalf("inspection failure changed status: %+v", current)
	}
}

func TestApplicationUpdateStatusPreinstallRecoveryUsesOperationLock(t *testing.T) {
	manager, _ := applicationStatusTestManager(t)
	status, err := manager.begin(UpdateSourceOffline, "", UpdatePhaseUploading)
	if err != nil {
		t.Fatal(err)
	}
	manager.deps.operationActive = func() (bool, error) { return true, nil }
	restarted := newApplicationUpdateStatusManager(manager.path, manager.deps)
	current, err := restarted.current()
	if err != nil {
		t.Fatal(err)
	}
	if current.OperationID != status.OperationID || current.Phase != UpdatePhaseUploading {
		t.Fatalf("live preinstall operation changed: %+v", current)
	}

	manager.deps.operationActive = func() (bool, error) { return false, nil }
	restarted = newApplicationUpdateStatusManager(manager.path, manager.deps)
	current, err = restarted.current()
	if err != nil {
		t.Fatal(err)
	}
	if current.Phase != UpdatePhaseFailed || current.ErrorCode != UpdateErrorInterrupted {
		t.Fatalf("orphaned preinstall operation = %+v", current)
	}
}

func TestApplicationUpdateStatusAdmissionMayReplaceOrphanWhileHoldingLease(t *testing.T) {
	manager, _ := applicationStatusTestManager(t)
	old, err := manager.begin(UpdateSourceOnline, "0.0.25", UpdatePhaseDownloading)
	if err != nil {
		t.Fatal(err)
	}
	manager.deps.operationActive = func() (bool, error) { return true, nil }
	manager.deps.newOperationID = func() string { return "operation-2" }
	// begin represents a caller which already owns the admission lease. The
	// probe would observe that caller's own flock, not a live owner of old.
	next, err := manager.begin(UpdateSourceOffline, "", UpdatePhaseUploading)
	if err != nil {
		t.Fatal(err)
	}
	if next.OperationID != "operation-2" || next.OperationID == old.OperationID {
		t.Fatalf("new admission status = %+v", next)
	}
}

func TestApplicationUpdateStatusOperationLockProbeFailureIsConservative(t *testing.T) {
	manager, _ := applicationStatusTestManager(t)
	status, err := manager.begin(UpdateSourceOnline, "0.0.25", UpdatePhaseDownloading)
	if err != nil {
		t.Fatal(err)
	}
	manager.deps.operationActive = func() (bool, error) { return false, errors.New("lock probe failed") }
	restarted := newApplicationUpdateStatusManager(manager.path, manager.deps)
	current, err := restarted.current()
	if err != nil {
		t.Fatal(err)
	}
	if current.OperationID != status.OperationID || current.Phase != UpdatePhaseDownloading {
		t.Fatalf("lock probe failure changed active status: %+v", current)
	}
}

func TestApplicationUpdateStatusReadsDiskForEveryMutation(t *testing.T) {
	manager, _ := applicationStatusTestManager(t)
	status, err := manager.begin(UpdateSourceOnline, "0.0.25", UpdatePhaseDownloading)
	if err != nil {
		t.Fatal(err)
	}
	other := newApplicationUpdateStatusManager(manager.path, manager.deps)
	if err := other.advance(status.OperationID, UpdatePhaseDownloading, 70); err != nil {
		t.Fatal(err)
	}
	if err := manager.advance(status.OperationID, UpdatePhaseDownloading, 69); !errors.Is(err, errApplicationUpdatePhase) {
		t.Fatalf("stale manager overwrote newer disk progress: %v", err)
	}
	current, err := manager.current()
	if err != nil {
		t.Fatal(err)
	}
	if current.Progress != 70 {
		t.Fatalf("progress = %d, want 70", current.Progress)
	}
}

func TestApplicationUpdateStatusRecoversActiveUnitWithoutStatusFile(t *testing.T) {
	manager, _ := applicationStatusTestManager(t)
	manager.deps.inspectUnit = func() (applicationUpdateUnitStatus, error) {
		return applicationUpdateUnitStatus{
			Exists: true, ActiveState: "active", SubState: "running",
			OperationID: "operation-from-unit", Version: "0.0.25", Source: UpdateSourceOffline,
		}, nil
	}
	status, err := manager.current()
	if err != nil {
		t.Fatal(err)
	}
	if status.OperationID != "operation-from-unit" || status.Version != "0.0.25" ||
		status.Source != UpdateSourceOffline || status.Phase != UpdatePhaseInstalling {
		t.Fatalf("recovered status = %+v", status)
	}
	if _, err := manager.begin(UpdateSourceOnline, "0.0.26", UpdatePhaseDownloading); !errors.Is(err, errApplicationUpdateBusy) {
		t.Fatalf("begin with live recovered unit error = %v", err)
	}
}

func TestApplicationUpdateStatusUnknownActiveUnitIsBusy(t *testing.T) {
	manager, _ := applicationStatusTestManager(t)
	manager.deps.inspectUnit = func() (applicationUpdateUnitStatus, error) {
		return applicationUpdateUnitStatus{Exists: true, ActiveState: "activating", SubState: "start"}, nil
	}
	if _, err := manager.current(); !errors.Is(err, errApplicationUpdateBusy) {
		t.Fatalf("unknown active unit status error = %v", err)
	}
	if _, err := manager.begin(UpdateSourceOnline, "0.0.25", UpdatePhaseDownloading); !errors.Is(err, errApplicationUpdateBusy) {
		t.Fatalf("begin with unknown active unit error = %v", err)
	}
}

func TestApplicationUpdateStatusRejectsMismatchedActiveUnitIdentity(t *testing.T) {
	manager, _ := applicationStatusTestManager(t)
	_, err := manager.begin(UpdateSourceOnline, "0.0.25", UpdatePhaseDownloading)
	if err != nil {
		t.Fatal(err)
	}
	manager.deps.inspectUnit = func() (applicationUpdateUnitStatus, error) {
		return applicationUpdateUnitStatus{
			Exists: true, ActiveState: "active", OperationID: "different-operation",
			Version: "0.0.26", Source: UpdateSourceOffline,
		}, nil
	}
	if _, err := manager.current(); !errors.Is(err, errApplicationUpdateBusy) {
		t.Fatalf("mismatched active unit error = %v", err)
	}
}

func TestApplicationUpdateStatusStrictJSONRejectsUnknownDuplicateAndMissingFields(t *testing.T) {
	valid := `{"operationId":"one","source":"online","version":"0.0.25","phase":"installing","progress":1,"errorCode":"","startedAt":"2026-08-12T00:00:00Z","updatedAt":"2026-08-12T00:00:00Z"}`
	for _, document := range []string{
		strings.TrimSuffix(valid, "}") + `,"internal":"secret"}`,
		strings.Replace(valid, `"operationId":"one"`, `"operationId":"one","operationId":"two"`, 1),
		strings.Replace(valid, `,"source":"online"`, "", 1),
	} {
		if _, err := decodeApplicationUpdateStatus([]byte(document)); err == nil {
			t.Fatalf("invalid status accepted: %s", document)
		}
	}
}

func TestApplicationUpdateSucceedRequiresInstallingOrRestarting(t *testing.T) {
	manager, _ := applicationStatusTestManager(t)
	status, err := manager.begin(UpdateSourceOnline, "0.0.25", UpdatePhaseDownloading)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.succeed(status.OperationID); !errors.Is(err, errApplicationUpdatePhase) {
		t.Fatalf("early succeed error = %v", err)
	}
	for _, phase := range []UpdatePhase{UpdatePhaseVerifying, UpdatePhaseStaging, UpdatePhaseInstalling} {
		if err := manager.advance(status.OperationID, phase, 100); err != nil {
			t.Fatal(err)
		}
	}
	if err := manager.succeed(status.OperationID); err != nil {
		t.Fatal(err)
	}
}

func TestApplicationUpdateStatusAcknowledgeClearsOnlyMatchingTerminalOperation(t *testing.T) {
	for _, phase := range []UpdatePhase{UpdatePhaseSucceeded, UpdatePhaseFailed} {
		t.Run(string(phase), func(t *testing.T) {
			manager, _ := applicationStatusTestManager(t)
			status, err := manager.begin(UpdateSourceOnline, "0.0.25", UpdatePhaseDownloading)
			if err != nil {
				t.Fatal(err)
			}
			if phase == UpdatePhaseFailed {
				if err := manager.fail(status.OperationID, UpdateErrorInterrupted); err != nil {
					t.Fatal(err)
				}
			} else {
				for _, next := range []UpdatePhase{UpdatePhaseVerifying, UpdatePhaseStaging, UpdatePhaseInstalling} {
					if err := manager.advance(status.OperationID, next, 100); err != nil {
						t.Fatal(err)
					}
				}
				if err := manager.succeed(status.OperationID); err != nil {
					t.Fatal(err)
				}
			}

			acknowledged, changed, err := manager.acknowledge(status.OperationID)
			if err != nil {
				t.Fatal(err)
			}
			if !changed || acknowledged != idleApplicationUpdateStatus() {
				t.Fatalf("acknowledged status = %+v, changed=%t", acknowledged, changed)
			}
			stored, exists, err := manager.readStatus()
			if err != nil || !exists || stored != idleApplicationUpdateStatus() {
				t.Fatalf("stored acknowledged status = %+v, exists=%t, err=%v", stored, exists, err)
			}
		})
	}
}

func TestApplicationUpdateStatusAcknowledgeIsIdempotentAfterResponseLoss(t *testing.T) {
	manager, _ := applicationStatusTestManager(t)
	status, err := manager.begin(UpdateSourceOnline, "0.0.25", UpdatePhaseDownloading)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.fail(status.OperationID, UpdateErrorInterrupted); err != nil {
		t.Fatal(err)
	}
	if acknowledged, changed, err := manager.acknowledge(status.OperationID); err != nil || !changed || acknowledged != idleApplicationUpdateStatus() {
		t.Fatalf("initial acknowledge = %+v, changed=%t, err=%v", acknowledged, changed, err)
	}
	if retry, changed, err := manager.acknowledge(status.OperationID); err != nil || changed || retry != idleApplicationUpdateStatus() {
		t.Fatalf("idempotent acknowledge = %+v, changed=%t, err=%v", retry, changed, err)
	}

	manager.deps.inspectUnit = func() (applicationUpdateUnitStatus, error) {
		return applicationUpdateUnitStatus{Exists: true, ActiveState: "activating"}, nil
	}
	if retry, changed, err := manager.acknowledge(status.OperationID); !errors.Is(err, errApplicationUpdateBusy) || changed || retry != idleApplicationUpdateStatus() {
		t.Fatalf("idle acknowledge with active unit = %+v, err=%v", retry, err)
	}
}

func TestApplicationUpdateStatusAcknowledgeHandlesIdleActiveAndStaleOperations(t *testing.T) {
	t.Run("idle retry", func(t *testing.T) {
		manager, _ := applicationStatusTestManager(t)
		if _, _, err := manager.acknowledge("operation-1"); err != nil {
			t.Fatalf("idle acknowledge error = %v", err)
		}
	})

	t.Run("active", func(t *testing.T) {
		manager, _ := applicationStatusTestManager(t)
		status, err := manager.begin(UpdateSourceOnline, "0.0.25", UpdatePhaseDownloading)
		if err != nil {
			t.Fatal(err)
		}
		manager.deps.operationActive = func() (bool, error) { return true, nil }
		if current, changed, err := manager.acknowledge(status.OperationID); !errors.Is(err, errApplicationUpdatePhase) || changed ||
			current.OperationID != status.OperationID || current.Phase != UpdatePhaseDownloading {
			t.Fatalf("active acknowledge = %+v, err=%v", current, err)
		}
	})

	t.Run("stale operation cannot clear replacement", func(t *testing.T) {
		manager, _ := applicationStatusTestManager(t)
		old, err := manager.begin(UpdateSourceOnline, "0.0.25", UpdatePhaseDownloading)
		if err != nil {
			t.Fatal(err)
		}
		if err := manager.fail(old.OperationID, UpdateErrorInterrupted); err != nil {
			t.Fatal(err)
		}
		manager.deps.newOperationID = func() string { return "operation-2" }
		current, err := manager.begin(UpdateSourceOnline, "0.0.26", UpdatePhaseDownloading)
		if err != nil {
			t.Fatal(err)
		}
		manager.deps.operationActive = func() (bool, error) { return true, nil }

		got, changed, err := manager.acknowledge(old.OperationID)
		if !errors.Is(err, errApplicationUpdateConflict) {
			t.Fatalf("stale acknowledge error = %v", err)
		}
		if changed || got.OperationID != current.OperationID || got.Phase != UpdatePhaseDownloading {
			t.Fatalf("stale acknowledge changed replacement: %+v", got)
		}
	})
}

func TestApplicationUpdateStatusAcknowledgeWaitsForTerminalUnitExit(t *testing.T) {
	manager, _ := applicationStatusTestManager(t)
	status, err := manager.begin(UpdateSourceOnline, "0.0.25", UpdatePhaseDownloading)
	if err != nil {
		t.Fatal(err)
	}
	for _, next := range []UpdatePhase{UpdatePhaseVerifying, UpdatePhaseStaging, UpdatePhaseInstalling} {
		if err := manager.advance(status.OperationID, next, 100); err != nil {
			t.Fatal(err)
		}
	}
	if err := manager.succeed(status.OperationID); err != nil {
		t.Fatal(err)
	}

	unitActive := true
	inspections := 0
	manager.deps.inspectUnit = func() (applicationUpdateUnitStatus, error) {
		inspections++
		state := applicationUpdateUnitStatus{
			Exists: true, OperationID: status.OperationID, Version: status.Version, Source: status.Source,
		}
		if unitActive {
			state.ActiveState = "deactivating"
		}
		return state, nil
	}
	current, changed, err := manager.acknowledge(status.OperationID)
	if !errors.Is(err, errApplicationUpdateBusy) || changed || current.Phase != UpdatePhaseSucceeded {
		t.Fatalf("active terminal unit acknowledge = %+v, err=%v", current, err)
	}
	if inspections != 1 {
		t.Fatalf("active terminal unit inspections = %d, want 1", inspections)
	}
	stored, _, readErr := manager.readStatus()
	if readErr != nil || stored.Phase != UpdatePhaseSucceeded {
		t.Fatalf("active terminal unit was cleared: %+v, err=%v", stored, readErr)
	}

	unitActive = false
	inspections = 0
	acknowledged, changed, err := manager.acknowledge(status.OperationID)
	if err != nil || !changed || acknowledged != idleApplicationUpdateStatus() {
		t.Fatalf("exited terminal unit acknowledge = %+v, err=%v", acknowledged, err)
	}
	if inspections != 1 {
		t.Fatalf("exited terminal unit inspections = %d, want 1", inspections)
	}
	refreshed, err := manager.current()
	if err != nil || refreshed != idleApplicationUpdateStatus() {
		t.Fatalf("acknowledged idle was resurrected: %+v, err=%v", refreshed, err)
	}
}

func TestAcknowledgeUpdateStatusHandlerReturnsAndBroadcastsCanonicalIdle(t *testing.T) {
	manager, _ := applicationStatusTestManager(t)
	status, err := manager.begin(UpdateSourceOnline, "0.0.25", UpdatePhaseDownloading)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.fail(status.OperationID, UpdateErrorInterrupted); err != nil {
		t.Fatal(err)
	}
	originalStatus := applicationStatus
	originalBroadcast := broadcastAcknowledgedApplicationUpdateStatus
	applicationStatus = manager
	var broadcasts []ApplicationUpdateStatus
	broadcastAcknowledgedApplicationUpdateStatus = func(status ApplicationUpdateStatus) {
		broadcasts = append(broadcasts, status)
	}
	t.Cleanup(func() {
		applicationStatus = originalStatus
		broadcastAcknowledgedApplicationUpdateStatus = originalBroadcast
	})

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/application/update/status/acknowledge",
		bytes.NewBufferString(`{"operationId":"`+status.OperationID+`"}`),
	)
	context.Request.Header.Set("Content-Type", "application/json")
	new(Service).AcknowledgeUpdateStatus(context)

	var response testApplicationResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != 0 {
		t.Fatalf("acknowledge response = %s", recorder.Body.String())
	}
	var acknowledged ApplicationUpdateStatus
	if err := json.Unmarshal(response.Data, &acknowledged); err != nil {
		t.Fatal(err)
	}
	if acknowledged != idleApplicationUpdateStatus() {
		t.Fatalf("handler acknowledged status = %+v", acknowledged)
	}
	if len(broadcasts) != 1 || broadcasts[0] != idleApplicationUpdateStatus() {
		t.Fatalf("acknowledge broadcasts = %+v", broadcasts)
	}

	// Retrying after a lost successful response remains successful without
	// emitting a duplicate state-change broadcast.
	retryRecorder := httptest.NewRecorder()
	retryContext, _ := gin.CreateTestContext(retryRecorder)
	retryContext.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/application/update/status/acknowledge",
		bytes.NewBufferString(`{"operationId":"`+status.OperationID+`"}`),
	)
	retryContext.Request.Header.Set("Content-Type", "application/json")
	new(Service).AcknowledgeUpdateStatus(retryContext)
	var retryResponse testApplicationResponse
	if err := json.Unmarshal(retryRecorder.Body.Bytes(), &retryResponse); err != nil {
		t.Fatal(err)
	}
	if retryResponse.Code != 0 || len(broadcasts) != 1 {
		t.Fatalf("idempotent handler response = %s, broadcasts=%+v", retryRecorder.Body.String(), broadcasts)
	}
}

func TestAcknowledgeUpdateStatusHandlerRejectsInvalidOrMismatchedOperation(t *testing.T) {
	manager, _ := applicationStatusTestManager(t)
	status, err := manager.begin(UpdateSourceOnline, "0.0.25", UpdatePhaseDownloading)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.fail(status.OperationID, UpdateErrorInterrupted); err != nil {
		t.Fatal(err)
	}
	originalStatus := applicationStatus
	originalBroadcast := broadcastAcknowledgedApplicationUpdateStatus
	applicationStatus = manager
	broadcasts := 0
	broadcastAcknowledgedApplicationUpdateStatus = func(ApplicationUpdateStatus) { broadcasts++ }
	t.Cleanup(func() {
		applicationStatus = originalStatus
		broadcastAcknowledgedApplicationUpdateStatus = originalBroadcast
	})

	perform := func(body string) testApplicationResponse {
		t.Helper()
		gin.SetMode(gin.TestMode)
		recorder := httptest.NewRecorder()
		context, _ := gin.CreateTestContext(recorder)
		context.Request = httptest.NewRequest(
			http.MethodPost,
			"/api/application/update/status/acknowledge",
			bytes.NewBufferString(body),
		)
		context.Request.Header.Set("Content-Type", "application/json")
		new(Service).AcknowledgeUpdateStatus(context)

		var response testApplicationResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		return response
	}
	if response := perform(`{}`); response.Code == 0 {
		t.Fatal("missing operation ID acknowledge succeeded")
	}
	response := perform(`{"operationId":"different-operation"}`)
	if response.Code == 0 {
		t.Fatal("mismatched operation acknowledge succeeded")
	}
	var authoritative ApplicationUpdateStatus
	if err := json.Unmarshal(response.Data, &authoritative); err != nil {
		t.Fatalf("decode authoritative conflict status: %v data=%s", err, response.Data)
	}
	if authoritative.OperationID != status.OperationID || authoritative.Phase != UpdatePhaseFailed {
		t.Fatalf("conflict status = %+v", authoritative)
	}
	if broadcasts != 0 {
		t.Fatalf("rejected acknowledge broadcasts = %d", broadcasts)
	}
	current, err := manager.current()
	if err != nil || current.Phase != UpdatePhaseFailed || current.OperationID != status.OperationID {
		t.Fatalf("rejected acknowledge changed status: %+v, err=%v", current, err)
	}
}

func TestParseApplicationUpdateUnitEnvironmentRequiresUniqueIdentity(t *testing.T) {
	environment := parseApplicationUpdateUnitEnvironment(
		"PATH=/usr/bin " + updateOperationEnvironment + "=operation-1 " +
			updateVersionEnvironment + "=0.0.25 " + updateSourceEnvironment + "=offline",
	)
	status := applicationUpdateUnitStatus{
		OperationID: environment[updateOperationEnvironment],
		Version:     environment[updateVersionEnvironment],
		Source:      UpdateSource(environment[updateSourceEnvironment]),
	}
	if !status.validIdentity() {
		t.Fatalf("environment identity invalid: %#v", environment)
	}
	duplicate := parseApplicationUpdateUnitEnvironment(
		updateOperationEnvironment + "=one " + updateOperationEnvironment + "=two " +
			updateVersionEnvironment + "=0.0.25 " + updateSourceEnvironment + "=online",
	)
	if duplicate[updateOperationEnvironment] != "" {
		t.Fatalf("duplicate operation ID accepted: %#v", duplicate)
	}
}

func TestGetUpdateStatusReturnsStablePublicEnvelope(t *testing.T) {
	manager, _ := applicationStatusTestManager(t)
	oldManager := applicationStatus
	applicationStatus = manager
	t.Cleanup(func() { applicationStatus = oldManager })

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/application/update/status", nil)
	new(Service).GetUpdateStatus(context)

	var response proto.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != 0 {
		t.Fatalf("response = %s", recorder.Body.String())
	}
	data, err := json.Marshal(response.Data)
	if err != nil {
		t.Fatal(err)
	}
	var status ApplicationUpdateStatus
	if err := json.Unmarshal(data, &status); err != nil {
		t.Fatal(err)
	}
	if status != idleApplicationUpdateStatus() {
		t.Fatalf("idle status = %+v", status)
	}
	for _, forbidden := range []string{"/run/", "dpkg", "systemctl"} {
		if strings.Contains(recorder.Body.String(), forbidden) {
			t.Fatalf("response leaked %q: %s", forbidden, recorder.Body.String())
		}
	}
}

func TestInvalidPersistedStatusInspectionFailureDoesNotOverwrite(t *testing.T) {
	manager, _ := applicationStatusTestManager(t)
	manager.deps.inspectUnit = func() (applicationUpdateUnitStatus, error) {
		return applicationUpdateUnitStatus{}, errors.New("systemd unavailable")
	}
	if err := os.MkdirAll(filepath.Dir(manager.path), 0o700); err != nil {
		t.Fatal(err)
	}
	invalid := []byte(`{"operationId":"one","source":"online","version":"0.0.25","phase":"installing","progress":1,"errorCode":"secret-path","startedAt":"2026-08-12T00:00:00Z","updatedAt":"2026-08-12T00:00:00Z"}`)
	if err := os.WriteFile(manager.path, invalid, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.current(); err == nil {
		t.Fatal("invalid persisted status was accepted without a trustworthy unit inspection")
	}
	stored, err := os.ReadFile(manager.path)
	if err != nil {
		t.Fatal(err)
	}
	if string(stored) != string(invalid) {
		t.Fatal("invalid persisted status was overwritten")
	}
}

func TestInvalidPersistedStatusRecoversOnlyFromValidUnitIdentity(t *testing.T) {
	for _, test := range []struct {
		name       string
		document   string
		unit       applicationUpdateUnitStatus
		versions   map[string]string
		wantPhase  UpdatePhase
		wantError  UpdateErrorCode
		wantExists bool
		wantBusy   bool
	}{
		{
			name:     "active unit replaces duplicate-field status",
			document: `{"operationId":"untrusted","operationId":"other","source":"online","version":"9.9.9","phase":"installing","progress":1,"errorCode":"","startedAt":"2026-08-12T00:00:00Z","updatedAt":"2026-08-12T00:00:00Z"}`,
			unit: applicationUpdateUnitStatus{
				Exists: true, ActiveState: "active", OperationID: "unit-operation",
				Version: "0.0.25", Source: UpdateSourceOffline,
			},
			wantPhase: UpdatePhaseInstalling, wantExists: true,
		},
		{
			name:     "completed successful unit uses both installed versions",
			document: `{"broken":true}`,
			unit: applicationUpdateUnitStatus{
				Exists: true, ActiveState: "inactive", Result: "success", ExecMainCode: "exited",
				ExecMainStatus: 0, HasExecStatus: true, OperationID: "unit-operation",
				Version: "0.0.25", Source: UpdateSourceOnline,
			},
			versions:  map[string]string{kvmCommAppName: "0.0.25", mainAppName: "0.0.25"},
			wantPhase: UpdatePhaseSucceeded, wantExists: true,
		},
		{
			name:      "collected unit discards oversized status",
			document:  strings.Repeat("x", (64<<10)+1),
			wantPhase: UpdatePhaseIdle,
		},
		{
			name:     "active unit without identity remains busy",
			document: `{"broken":true}`,
			unit:     applicationUpdateUnitStatus{Exists: true, ActiveState: "activating"},
			wantBusy: true, wantExists: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			manager, _ := applicationStatusTestManager(t)
			if err := os.MkdirAll(filepath.Dir(manager.path), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(manager.path, []byte(test.document), 0o600); err != nil {
				t.Fatal(err)
			}
			manager.deps.inspectUnit = func() (applicationUpdateUnitStatus, error) { return test.unit, nil }
			manager.deps.packageVersion = func(name string) (string, error) {
				version, ok := test.versions[name]
				if !ok {
					return "", errors.New("not installed")
				}
				return version, nil
			}

			status, err := manager.current()
			if test.wantBusy {
				if !errors.Is(err, errApplicationUpdateBusy) {
					t.Fatalf("current error = %v, want busy", err)
				}
				stored, readErr := os.ReadFile(manager.path)
				if readErr != nil || string(stored) != test.document {
					t.Fatal("unidentified active unit caused malformed status to be trusted or replaced")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if status.Phase != test.wantPhase || status.ErrorCode != test.wantError {
				t.Fatalf("recovered status = %+v", status)
			}
			_, statErr := os.Stat(manager.path)
			if test.wantExists && statErr != nil {
				t.Fatalf("recovered status was not persisted: %v", statErr)
			}
			if !test.wantExists && !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("malformed status was not removed: %v", statErr)
			}
			if status.OperationID == "untrusted" || status.Version == "9.9.9" {
				t.Fatalf("trusted malformed status identity: %+v", status)
			}
		})
	}
}
