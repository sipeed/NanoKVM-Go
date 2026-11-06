package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"NanoKVM-Server/proto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/sys/unix"
)

const (
	applicationUpdateStatusPath   = "/run/nanokvm-update/status.json"
	applicationUnitInspectTimeout = 5 * time.Second

	updateOperationEnvironment = "NANOKVM_UPDATE_OPERATION_ID"
	updateVersionEnvironment   = "NANOKVM_UPDATE_VERSION"
	updateSourceEnvironment    = "NANOKVM_UPDATE_SOURCE"
)

type UpdateSource string

const (
	UpdateSourceOnline  UpdateSource = "online"
	UpdateSourceOffline UpdateSource = "offline"
)

type UpdatePhase string

const (
	UpdatePhaseIdle        UpdatePhase = "idle"
	UpdatePhaseDownloading UpdatePhase = "downloading"
	UpdatePhaseUploading   UpdatePhase = "uploading"
	UpdatePhaseVerifying   UpdatePhase = "verifying"
	UpdatePhaseStaging     UpdatePhase = "staging"
	UpdatePhaseInstalling  UpdatePhase = "installing"
	UpdatePhaseRestarting  UpdatePhase = "restarting"
	UpdatePhaseSucceeded   UpdatePhase = "succeeded"
	UpdatePhaseFailed      UpdatePhase = "failed"
)

// UpdateErrorCode is the complete public error vocabulary. Detailed errors
// stay in logs and are never persisted or returned by the status endpoint.
type UpdateErrorCode string

const (
	UpdateErrorNone               UpdateErrorCode = ""
	UpdateErrorBusy               UpdateErrorCode = "busy"
	UpdateErrorUploadInterrupted  UpdateErrorCode = "upload_interrupted"
	UpdateErrorNoSpace            UpdateErrorCode = "no_space"
	UpdateErrorInvalidArchive     UpdateErrorCode = "invalid_archive"
	UpdateErrorSignatureInvalid   UpdateErrorCode = "signature_invalid"
	UpdateErrorUnsupportedPackage UpdateErrorCode = "unsupported_package"
	UpdateErrorSameVersion        UpdateErrorCode = "same_version"
	UpdateErrorInstallStartFailed UpdateErrorCode = "install_start_failed"
	UpdateErrorInstallFailed      UpdateErrorCode = "install_failed"
	UpdateErrorInterrupted        UpdateErrorCode = "interrupted"
)

var (
	errApplicationUpdateBusy     = errors.New("application update is busy")
	errApplicationUpdateConflict = errors.New("application update operation does not match")
	errApplicationUpdatePhase    = errors.New("invalid application update phase transition")
	errApplicationUpdateStatus   = errors.New("invalid application update status")
)

type ApplicationUpdateStatus struct {
	OperationID string          `json:"operationId"`
	Source      UpdateSource    `json:"source"`
	Version     string          `json:"version"`
	Phase       UpdatePhase     `json:"phase"`
	Progress    int             `json:"progress"`
	ErrorCode   UpdateErrorCode `json:"errorCode"`
	StartedAt   string          `json:"startedAt"`
	UpdatedAt   string          `json:"updatedAt"`
}

type applicationUpdateUnitStatus struct {
	Exists         bool
	ActiveState    string
	SubState       string
	Result         string
	ExecMainCode   string
	ExecMainStatus int
	HasExecStatus  bool
	OperationID    string
	Version        string
	Source         UpdateSource
}

func (status applicationUpdateUnitStatus) active() bool {
	switch status.ActiveState {
	case "activating", "active", "deactivating", "reloading":
		return true
	default:
		return false
	}
}

func (status applicationUpdateUnitStatus) completed() bool {
	return status.Exists && !status.active() &&
		(status.Result != "" || status.ExecMainCode != "" || status.HasExecStatus)
}

func (status applicationUpdateUnitStatus) failed() bool {
	if status.Result != "" && status.Result != "success" {
		return true
	}
	if status.ExecMainCode != "" && status.ExecMainCode != "exited" {
		return true
	}
	return status.HasExecStatus && status.ExecMainStatus != 0
}

func (status applicationUpdateUnitStatus) validIdentity() bool {
	return validOperationID(status.OperationID) && validPublicVersion(status.Version) && validUpdateSource(status.Source)
}

func (status applicationUpdateUnitStatus) matches(update ApplicationUpdateStatus) bool {
	return status.validIdentity() && status.OperationID == update.OperationID &&
		status.Version == update.Version && status.Source == update.Source
}

type applicationUpdateStatusDependencies struct {
	now             func() time.Time
	newOperationID  func() string
	inspectUnit     func() (applicationUpdateUnitStatus, error)
	operationActive func() (bool, error)
	packageVersion  func(string) (string, error)
}

var defaultApplicationUpdateStatusDependencies = applicationUpdateStatusDependencies{
	now:             time.Now,
	newOperationID:  uuid.NewString,
	inspectUnit:     inspectApplicationUpdateUnit,
	operationActive: applicationUpdateOperationActive,
	packageVersion:  getPackageVersion,
}

type applicationUpdateStatusManager struct {
	mu       sync.Mutex
	path     string
	lockPath string
	deps     applicationUpdateStatusDependencies
}

func newApplicationUpdateStatusManager(path string, deps applicationUpdateStatusDependencies) *applicationUpdateStatusManager {
	return &applicationUpdateStatusManager{path: path, lockPath: path + ".lock", deps: deps}
}

var applicationStatus = newApplicationUpdateStatusManager(
	applicationUpdateStatusPath,
	defaultApplicationUpdateStatusDependencies,
)

func beginApplicationUpdate(source UpdateSource, version string, initialPhase UpdatePhase) (ApplicationUpdateStatus, error) {
	return applicationStatus.begin(source, version, initialPhase)
}

func setApplicationUpdateVersion(operationID string, version string) error {
	return applicationStatus.setVersion(operationID, version)
}

func advanceApplicationUpdate(operationID string, phase UpdatePhase, progress int) error {
	return applicationStatus.advance(operationID, phase, progress)
}

func failApplicationUpdate(operationID string, errorCode UpdateErrorCode) error {
	return applicationStatus.fail(operationID, errorCode)
}

func succeedApplicationUpdate(operationID string) error {
	return applicationStatus.succeed(operationID)
}

func currentApplicationUpdateStatus() (ApplicationUpdateStatus, error) {
	return applicationStatus.current()
}

func acknowledgeApplicationUpdateStatus(operationID string) (ApplicationUpdateStatus, bool, error) {
	return applicationStatus.acknowledge(operationID)
}

var broadcastAcknowledgedApplicationUpdateStatus = broadcastApplicationUpdateStatus

func (s *Service) GetUpdateStatus(c *gin.Context) {
	var rsp proto.Response
	status, err := currentApplicationUpdateStatus()
	if err != nil {
		rsp.ErrRsp(c, -1, "failed to get update status")
		return
	}
	rsp.OkRspWithData(c, status)
}

func (s *Service) AcknowledgeUpdateStatus(c *gin.Context) {
	var req proto.AcknowledgeApplicationUpdateStatusReq
	var rsp proto.Response
	if err := proto.ParseFormRequest(c, &req); err != nil || !validOperationID(req.OperationID) {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	status, changed, err := acknowledgeApplicationUpdateStatus(req.OperationID)
	if err != nil {
		switch {
		case errors.Is(err, errApplicationUpdateBusy):
			rsp.ErrRspWithData(c, -2, "application update is busy", status)
		case errors.Is(err, errApplicationUpdateConflict), errors.Is(err, errApplicationUpdatePhase):
			rsp.ErrRspWithData(c, -2, "update status cannot be acknowledged", status)
		default:
			rsp.ErrRsp(c, -3, "failed to acknowledge update status")
		}
		return
	}

	if changed {
		broadcastAcknowledgedApplicationUpdateStatus(status)
	}
	rsp.OkRspWithData(c, status)
}

func (manager *applicationUpdateStatusManager) begin(source UpdateSource, version string, initialPhase UpdatePhase) (ApplicationUpdateStatus, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	lock, err := manager.acquireFileLock()
	if err != nil {
		return ApplicationUpdateStatus{}, err
	}
	defer releaseApplicationStatusLock(lock)

	status, exists, err := manager.readStatus()
	if errors.Is(err, errApplicationUpdateStatus) {
		status, exists, err = manager.recoverInvalidStatus()
	}
	if err != nil {
		return ApplicationUpdateStatus{}, err
	}
	unit, inspectErr := manager.deps.inspectUnit()
	if inspectErr != nil {
		return ApplicationUpdateStatus{}, fmt.Errorf("inspect application update service")
	}
	if unit.active() {
		return ApplicationUpdateStatus{}, errApplicationUpdateBusy
	}
	status, exists, err = manager.reconcileForAdmission(status, exists)
	if err != nil && !errors.Is(err, errApplicationUpdateBusy) {
		return ApplicationUpdateStatus{}, err
	}
	if errors.Is(err, errApplicationUpdateBusy) || isApplicationUpdateActive(status.Phase) {
		return ApplicationUpdateStatus{}, errApplicationUpdateBusy
	}
	// Even a terminal or absent status cannot authorize a second operation
	// while an unidentified transient update unit is running.
	version = normalizeVersion(version)
	if !validUpdateSource(source) || !validOptionalPublicVersion(version) || !validInitialUpdatePhase(source, initialPhase) {
		return ApplicationUpdateStatus{}, errApplicationUpdatePhase
	}
	operationID := manager.deps.newOperationID()
	if !validOperationID(operationID) {
		return ApplicationUpdateStatus{}, fmt.Errorf("generate application update operation ID")
	}
	now := manager.timestamp("")
	next := ApplicationUpdateStatus{
		OperationID: operationID,
		Source:      source,
		Version:     version,
		Phase:       initialPhase,
		Progress:    0,
		ErrorCode:   UpdateErrorNone,
		StartedAt:   now,
		UpdatedAt:   now,
	}
	if err := manager.persist(next); err != nil {
		return ApplicationUpdateStatus{}, err
	}
	_ = exists
	return next, nil
}

// reconcileForAdmission runs only after coordination.go has acquired the
// exclusive application lease. Therefore any pre-installer status cannot still
// have a live owner; probing the same lock here would only see our own lease.
func (manager *applicationUpdateStatusManager) reconcileForAdmission(status ApplicationUpdateStatus, exists bool) (ApplicationUpdateStatus, bool, error) {
	if !exists || !isApplicationUpdateActive(status.Phase) {
		return status, exists, nil
	}
	if status.Phase == UpdatePhaseInstalling || status.Phase == UpdatePhaseRestarting {
		return manager.reconcileInstalledVersions(status)
	}
	return manager.persistRecoveredTerminal(status, UpdatePhaseFailed, UpdateErrorInterrupted)
}

func (manager *applicationUpdateStatusManager) setVersion(operationID string, version string) error {
	return manager.mutate(func(status ApplicationUpdateStatus) (ApplicationUpdateStatus, error) {
		version = normalizeVersion(version)
		if operationID == "" || operationID != status.OperationID {
			return status, errApplicationUpdateConflict
		}
		if !isApplicationUpdateActive(status.Phase) || !validPublicVersion(version) ||
			(status.Version != "" && status.Version != version) {
			return status, errApplicationUpdatePhase
		}
		if status.Version == version {
			return status, nil
		}
		status.Version = version
		status.UpdatedAt = manager.timestamp(status.UpdatedAt)
		return status, nil
	})
}

func (manager *applicationUpdateStatusManager) advance(operationID string, phase UpdatePhase, progress int) error {
	return manager.mutate(func(status ApplicationUpdateStatus) (ApplicationUpdateStatus, error) {
		if operationID == "" || operationID != status.OperationID {
			return status, errApplicationUpdateConflict
		}
		if progress < 0 || progress > 100 || !validForwardPhase(status, phase, progress) {
			return status, errApplicationUpdatePhase
		}
		status.Phase = phase
		status.Progress = progress
		status.ErrorCode = UpdateErrorNone
		status.UpdatedAt = manager.timestamp(status.UpdatedAt)
		return status, nil
	})
}

func (manager *applicationUpdateStatusManager) fail(operationID string, errorCode UpdateErrorCode) error {
	return manager.mutate(func(status ApplicationUpdateStatus) (ApplicationUpdateStatus, error) {
		if operationID == "" || operationID != status.OperationID {
			return status, errApplicationUpdateConflict
		}
		if !isApplicationUpdateActive(status.Phase) || errorCode == UpdateErrorNone || !validUpdateErrorCode(errorCode) {
			return status, errApplicationUpdatePhase
		}
		status.Phase = UpdatePhaseFailed
		status.ErrorCode = errorCode
		status.UpdatedAt = manager.timestamp(status.UpdatedAt)
		return status, nil
	})
}

func (manager *applicationUpdateStatusManager) succeed(operationID string) error {
	return manager.mutate(func(status ApplicationUpdateStatus) (ApplicationUpdateStatus, error) {
		if operationID == "" || operationID != status.OperationID {
			return status, errApplicationUpdateConflict
		}
		if status.Phase != UpdatePhaseInstalling && status.Phase != UpdatePhaseRestarting {
			return status, errApplicationUpdatePhase
		}
		status.Phase = UpdatePhaseSucceeded
		status.Progress = 100
		status.ErrorCode = UpdateErrorNone
		status.UpdatedAt = manager.timestamp(status.UpdatedAt)
		return status, nil
	})
}

func (manager *applicationUpdateStatusManager) mutate(change func(ApplicationUpdateStatus) (ApplicationUpdateStatus, error)) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	lock, err := manager.acquireFileLock()
	if err != nil {
		return err
	}
	defer releaseApplicationStatusLock(lock)
	status, exists, err := manager.readStatus()
	if err != nil {
		return err
	}
	if !exists {
		return errApplicationUpdateConflict
	}
	next, err := change(status)
	if err != nil {
		return err
	}
	if next == status {
		return nil
	}
	return manager.persist(next)
}

func (manager *applicationUpdateStatusManager) current() (ApplicationUpdateStatus, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	lock, err := manager.acquireFileLock()
	if err != nil {
		return ApplicationUpdateStatus{}, err
	}
	defer releaseApplicationStatusLock(lock)
	status, exists, err := manager.readStatus()
	if errors.Is(err, errApplicationUpdateStatus) {
		status, exists, err = manager.recoverInvalidStatus()
	}
	if err != nil {
		return ApplicationUpdateStatus{}, err
	}
	status, _, err = manager.reconcile(status, exists)
	return status, err
}

func (manager *applicationUpdateStatusManager) acknowledge(operationID string) (ApplicationUpdateStatus, bool, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	lock, err := manager.acquireFileLock()
	if err != nil {
		return ApplicationUpdateStatus{}, false, err
	}
	defer releaseApplicationStatusLock(lock)

	status, exists, err := manager.readStatus()
	if errors.Is(err, errApplicationUpdateStatus) {
		status, exists, err = manager.recoverInvalidStatus()
	}
	if err != nil {
		return ApplicationUpdateStatus{}, false, err
	}
	unit, inspectErr := manager.deps.inspectUnit()
	status, exists, err = manager.reconcileInspected(status, exists, unit, inspectErr)
	if err != nil {
		return status, false, err
	}
	if inspectErr != nil {
		return status, false, fmt.Errorf("inspect application update service")
	}
	if !validOperationID(operationID) {
		return status, false, errApplicationUpdateConflict
	}
	if status.Phase == UpdatePhaseIdle {
		// A successful acknowledge may have reached disk even if its HTTP response
		// was lost. Canonical idle has no operation identity left to compare, so a
		// valid retry is safe once the unit inspection confirms that no worker
		// can resurrect an in-flight operation.
		if unit.active() {
			return status, false, errApplicationUpdateBusy
		}
		return idleApplicationUpdateStatus(), false, nil
	}
	if !exists || operationID == "" || operationID != status.OperationID {
		return status, false, errApplicationUpdateConflict
	}
	if status.Phase != UpdatePhaseSucceeded && status.Phase != UpdatePhaseFailed {
		return status, false, errApplicationUpdatePhase
	}

	// The worker persists succeeded before its transient unit exits. Clearing the
	// terminal status while that unit is still active would let the next status
	// read recover the unit as installing. Require the fixed unit to be fully
	// inactive before atomically publishing canonical idle.
	if unit.active() {
		return status, false, errApplicationUpdateBusy
	}

	idle := idleApplicationUpdateStatus()
	if err := manager.persist(idle); err != nil {
		return status, false, err
	}
	return idle, true, nil
}

func (manager *applicationUpdateStatusManager) reconcile(status ApplicationUpdateStatus, exists bool) (ApplicationUpdateStatus, bool, error) {
	unit, err := manager.deps.inspectUnit()
	return manager.reconcileInspected(status, exists, unit, err)
}

func (manager *applicationUpdateStatusManager) reconcileInspected(
	status ApplicationUpdateStatus,
	exists bool,
	unit applicationUpdateUnitStatus,
	inspectErr error,
) (ApplicationUpdateStatus, bool, error) {
	if inspectErr != nil {
		if exists {
			return status, true, nil
		}
		return ApplicationUpdateStatus{}, false, fmt.Errorf("inspect application update service")
	}
	if !exists {
		if !unit.active() {
			return idleApplicationUpdateStatus(), false, nil
		}
		if !unit.validIdentity() {
			return ApplicationUpdateStatus{}, false, errApplicationUpdateBusy
		}
		now := manager.timestamp("")
		recovered := ApplicationUpdateStatus{
			OperationID: unit.OperationID,
			Source:      unit.Source,
			Version:     unit.Version,
			Phase:       UpdatePhaseInstalling,
			Progress:    0,
			StartedAt:   now,
			UpdatedAt:   now,
		}
		if err := manager.persist(recovered); err != nil {
			return ApplicationUpdateStatus{}, false, err
		}
		return recovered, true, nil
	}

	if !isApplicationUpdateActive(status.Phase) {
		if unit.active() && unit.validIdentity() && !unit.matches(status) {
			now := manager.timestamp("")
			recovered := ApplicationUpdateStatus{
				OperationID: unit.OperationID,
				Source:      unit.Source,
				Version:     unit.Version,
				Phase:       UpdatePhaseInstalling,
				StartedAt:   now,
				UpdatedAt:   now,
			}
			if err := manager.persist(recovered); err != nil {
				return ApplicationUpdateStatus{}, false, err
			}
			return recovered, true, nil
		}
		return status, true, nil
	}

	if unit.active() {
		// A matching active unit only proves liveness. The worker owns explicit
		// phase advancement, so do not turn uploading/verifying into installing.
		if unit.validIdentity() && !unit.matches(status) {
			return status, true, errApplicationUpdateBusy
		}
		return status, true, nil
	}
	if unit.completed() {
		if !unit.matches(status) {
			return status, true, nil
		}
		if unit.failed() {
			return manager.persistRecoveredTerminal(status, UpdatePhaseFailed, UpdateErrorInstallFailed)
		}
		return manager.reconcileInstalledVersions(status)
	}
	operationActive, activeErr := manager.deps.operationActive()
	if activeErr != nil {
		return status, true, nil
	}
	if operationActive {
		return status, true, nil
	}
	if status.Phase == UpdatePhaseInstalling || status.Phase == UpdatePhaseRestarting {
		// --collect can remove the transient unit before the replacement server
		// observes it. Both installed package versions are then decisive.
		return manager.reconcileInstalledVersions(status)
	}
	return manager.persistRecoveredTerminal(status, UpdatePhaseFailed, UpdateErrorInterrupted)
}

// recoverInvalidStatus deliberately trusts no identity from a malformed state
// file. A valid transient-unit identity is the only source from which an
// in-flight operation can be reconstructed. If no unit owns an operation, the
// malformed file is removed so it cannot permanently block future updates.
//
// The conservative recovered phase is installing. The worker still performs
// every verification and staging step; it treats this phase only as an
// already-published status floor after state-file recovery.
func (manager *applicationUpdateStatusManager) recoverInvalidStatus() (ApplicationUpdateStatus, bool, error) {
	unit, err := manager.deps.inspectUnit()
	if err != nil {
		return ApplicationUpdateStatus{}, false, fmt.Errorf("inspect application update service")
	}
	if unit.active() {
		if !unit.validIdentity() {
			return ApplicationUpdateStatus{}, false, errApplicationUpdateBusy
		}
		now := manager.timestamp("")
		recovered := ApplicationUpdateStatus{
			OperationID: unit.OperationID,
			Source:      unit.Source,
			Version:     unit.Version,
			Phase:       UpdatePhaseInstalling,
			Progress:    0,
			ErrorCode:   UpdateErrorNone,
			StartedAt:   now,
			UpdatedAt:   now,
		}
		if err := manager.persist(recovered); err != nil {
			return ApplicationUpdateStatus{}, false, err
		}
		return recovered, true, nil
	}
	if unit.completed() && unit.validIdentity() {
		now := manager.timestamp("")
		recovered := ApplicationUpdateStatus{
			OperationID: unit.OperationID,
			Source:      unit.Source,
			Version:     unit.Version,
			Phase:       UpdatePhaseInstalling,
			Progress:    0,
			ErrorCode:   UpdateErrorNone,
			StartedAt:   now,
			UpdatedAt:   now,
		}
		if unit.failed() {
			return manager.persistRecoveredTerminal(recovered, UpdatePhaseFailed, UpdateErrorInstallFailed)
		}
		return manager.reconcileInstalledVersions(recovered)
	}
	if err := manager.removeInvalidStatus(); err != nil {
		return ApplicationUpdateStatus{}, false, err
	}
	return idleApplicationUpdateStatus(), false, nil
}

func (manager *applicationUpdateStatusManager) removeInvalidStatus() error {
	if err := os.Remove(manager.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove invalid application update status: %w", err)
	}
	directory, err := os.Open(filepath.Dir(manager.path))
	if err != nil {
		return fmt.Errorf("open application update status directory: %w", err)
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return fmt.Errorf("sync application update status directory: %w", err)
	}
	return nil
}

func applicationUpdateOperationActive() (bool, error) {
	lock, err := openAndLockApplicationFile(applicationUpdateLockPath, true)
	if err == nil {
		unlockApplicationFile(lock)
		return false, nil
	}
	if errors.Is(err, errApplicationUpdateBusy) {
		return true, nil
	}
	return false, err
}

func (manager *applicationUpdateStatusManager) reconcileInstalledVersions(status ApplicationUpdateStatus) (ApplicationUpdateStatus, bool, error) {
	for _, packageName := range appNames {
		version, err := manager.deps.packageVersion(packageName)
		if err != nil || normalizeVersion(version) != status.Version {
			return manager.persistRecoveredTerminal(status, UpdatePhaseFailed, UpdateErrorInstallFailed)
		}
	}
	return manager.persistRecoveredTerminal(status, UpdatePhaseSucceeded, UpdateErrorNone)
}

func (manager *applicationUpdateStatusManager) persistRecoveredTerminal(status ApplicationUpdateStatus, phase UpdatePhase, code UpdateErrorCode) (ApplicationUpdateStatus, bool, error) {
	status.Phase = phase
	status.ErrorCode = code
	if phase == UpdatePhaseSucceeded {
		status.Progress = 100
	}
	status.UpdatedAt = manager.timestamp(status.UpdatedAt)
	if err := manager.persist(status); err != nil {
		return ApplicationUpdateStatus{}, false, err
	}
	return status, true, nil
}

func (manager *applicationUpdateStatusManager) acquireFileLock() (*os.File, error) {
	directory := filepath.Dir(manager.path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, fmt.Errorf("create application update state directory: %w", err)
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return nil, fmt.Errorf("protect application update state directory: %w", err)
	}
	fd, err := unix.Open(manager.lockPath, unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open application update state lock: %w", err)
	}
	lock := os.NewFile(uintptr(fd), manager.lockPath)
	if lock == nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("open application update state lock")
	}
	info, err := lock.Stat()
	if err != nil || !info.Mode().IsRegular() {
		_ = lock.Close()
		return nil, fmt.Errorf("invalid application update state lock")
	}
	if err := lock.Chmod(0o600); err != nil {
		_ = lock.Close()
		return nil, fmt.Errorf("protect application update state lock: %w", err)
	}
	if err := unix.Flock(int(lock.Fd()), unix.LOCK_EX); err != nil {
		_ = lock.Close()
		return nil, fmt.Errorf("lock application update state: %w", err)
	}
	return lock, nil
}

func releaseApplicationStatusLock(lock *os.File) {
	if lock == nil {
		return
	}
	_ = unix.Flock(int(lock.Fd()), unix.LOCK_UN)
	_ = lock.Close()
}

func (manager *applicationUpdateStatusManager) readStatus() (ApplicationUpdateStatus, bool, error) {
	data, err := os.ReadFile(manager.path)
	if errors.Is(err, os.ErrNotExist) {
		return idleApplicationUpdateStatus(), false, nil
	}
	if err != nil {
		return ApplicationUpdateStatus{}, false, fmt.Errorf("read application update status: %w", err)
	}
	if len(data) > 64<<10 {
		return ApplicationUpdateStatus{}, false, fmt.Errorf("%w: status is too large", errApplicationUpdateStatus)
	}
	status, err := decodeApplicationUpdateStatus(data)
	if err != nil || !validStoredApplicationUpdateStatus(status) {
		return ApplicationUpdateStatus{}, false, errApplicationUpdateStatus
	}
	return status, true, nil
}

func decodeApplicationUpdateStatus(data []byte) (ApplicationUpdateStatus, error) {
	expected := map[string]bool{
		"operationId": false, "source": false, "version": false, "phase": false,
		"progress": false, "errorCode": false, "startedAt": false, "updatedAt": false,
	}
	keys := json.NewDecoder(bytes.NewReader(data))
	token, err := keys.Token()
	if err != nil || token != json.Delim('{') {
		return ApplicationUpdateStatus{}, fmt.Errorf("status must be an object")
	}
	for keys.More() {
		keyToken, err := keys.Token()
		if err != nil {
			return ApplicationUpdateStatus{}, err
		}
		key, ok := keyToken.(string)
		if !ok || expected[key] || !hasStatusJSONKey(expected, key) {
			return ApplicationUpdateStatus{}, fmt.Errorf("unexpected or duplicate status field")
		}
		expected[key] = true
		var value json.RawMessage
		if err := keys.Decode(&value); err != nil {
			return ApplicationUpdateStatus{}, err
		}
	}
	if _, err := keys.Token(); err != nil {
		return ApplicationUpdateStatus{}, err
	}
	if token, err := keys.Token(); err != io.EOF || token != nil {
		return ApplicationUpdateStatus{}, fmt.Errorf("trailing status data")
	}
	for _, present := range expected {
		if !present {
			return ApplicationUpdateStatus{}, fmt.Errorf("missing status field")
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var status ApplicationUpdateStatus
	if err := decoder.Decode(&status); err != nil {
		return ApplicationUpdateStatus{}, err
	}
	return status, nil
}

func hasStatusJSONKey(expected map[string]bool, key string) bool {
	_, ok := expected[key]
	return ok
}

func (manager *applicationUpdateStatusManager) persist(status ApplicationUpdateStatus) (err error) {
	directory := filepath.Dir(manager.path)
	data, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("encode application update status: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".status-*")
	if err != nil {
		return fmt.Errorf("create application update status temporary file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return fmt.Errorf("protect application update status temporary file: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		return fmt.Errorf("write application update status: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync application update status: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close application update status: %w", err)
	}
	if err := os.Rename(temporaryPath, manager.path); err != nil {
		return fmt.Errorf("publish application update status: %w", err)
	}
	if err := os.Chmod(manager.path, 0o600); err != nil {
		return fmt.Errorf("protect application update status: %w", err)
	}
	directoryHandle, err := os.Open(directory)
	if err != nil {
		return fmt.Errorf("open application update status directory: %w", err)
	}
	defer directoryHandle.Close()
	if err := directoryHandle.Sync(); err != nil {
		return fmt.Errorf("sync application update status directory: %w", err)
	}
	return nil
}

func (manager *applicationUpdateStatusManager) timestamp(previous string) string {
	now := manager.deps.now().UTC()
	if prior, err := time.Parse(time.RFC3339Nano, previous); err == nil && now.Before(prior) {
		now = prior
	}
	return now.Format(time.RFC3339Nano)
}

func idleApplicationUpdateStatus() ApplicationUpdateStatus {
	return ApplicationUpdateStatus{Phase: UpdatePhaseIdle}
}

func isApplicationUpdateActive(phase UpdatePhase) bool {
	return updatePhaseRank(phase) > 0 && phase != UpdatePhaseSucceeded && phase != UpdatePhaseFailed
}

func updatePhaseRank(phase UpdatePhase) int {
	switch phase {
	case UpdatePhaseIdle:
		return 0
	case UpdatePhaseDownloading, UpdatePhaseUploading:
		return 1
	case UpdatePhaseVerifying:
		return 2
	case UpdatePhaseStaging:
		return 3
	case UpdatePhaseInstalling:
		return 4
	case UpdatePhaseRestarting:
		return 5
	case UpdatePhaseSucceeded, UpdatePhaseFailed:
		return 6
	default:
		return -1
	}
}

func validForwardPhase(current ApplicationUpdateStatus, next UpdatePhase, progress int) bool {
	if !isApplicationUpdateActive(current.Phase) || !isApplicationUpdateActive(next) {
		return false
	}
	currentRank, nextRank := updatePhaseRank(current.Phase), updatePhaseRank(next)
	if nextRank < currentRank {
		return false
	}
	if nextRank == currentRank {
		return next == current.Phase && progress >= current.Progress
	}
	if nextRank != currentRank+1 {
		return false
	}
	return nextRank < updatePhaseRank(UpdatePhaseStaging) || current.Version != ""
}

func validInitialUpdatePhase(source UpdateSource, phase UpdatePhase) bool {
	return (source == UpdateSourceOnline && phase == UpdatePhaseDownloading) ||
		(source == UpdateSourceOffline && phase == UpdatePhaseUploading)
}

func validUpdateSource(source UpdateSource) bool {
	return source == UpdateSourceOnline || source == UpdateSourceOffline
}

func validUpdateErrorCode(code UpdateErrorCode) bool {
	switch code {
	case UpdateErrorNone, UpdateErrorBusy, UpdateErrorUploadInterrupted,
		UpdateErrorNoSpace, UpdateErrorInvalidArchive, UpdateErrorSignatureInvalid,
		UpdateErrorUnsupportedPackage, UpdateErrorSameVersion,
		UpdateErrorInstallStartFailed, UpdateErrorInstallFailed, UpdateErrorInterrupted:
		return true
	default:
		return false
	}
}

func validPublicVersion(version string) bool {
	return version != "" && len(version) <= 128 && validateLatestVersion(version) == nil
}

func validOptionalPublicVersion(version string) bool {
	return version == "" || validPublicVersion(version)
}

func validOperationID(operationID string) bool {
	if operationID == "" || len(operationID) > 128 {
		return false
	}
	for _, value := range []byte(operationID) {
		if !((value >= 'a' && value <= 'z') || (value >= 'A' && value <= 'Z') ||
			(value >= '0' && value <= '9') || value == '-' || value == '_') {
			return false
		}
	}
	return true
}

func validStoredApplicationUpdateStatus(status ApplicationUpdateStatus) bool {
	if status.Phase == UpdatePhaseIdle {
		return status == idleApplicationUpdateStatus()
	}
	if !validOperationID(status.OperationID) || !validUpdateSource(status.Source) ||
		!validOptionalPublicVersion(status.Version) || updatePhaseRank(status.Phase) < 1 ||
		status.Progress < 0 || status.Progress > 100 || status.StartedAt == "" || status.UpdatedAt == "" {
		return false
	}
	startedAt, err := time.Parse(time.RFC3339Nano, status.StartedAt)
	if err != nil {
		return false
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, status.UpdatedAt)
	if err != nil || updatedAt.Before(startedAt) {
		return false
	}
	if status.Version == "" && updatePhaseRank(status.Phase) >= updatePhaseRank(UpdatePhaseStaging) {
		return false
	}
	if status.Phase == UpdatePhaseFailed {
		return status.ErrorCode != UpdateErrorNone && validUpdateErrorCode(status.ErrorCode)
	}
	if status.ErrorCode != UpdateErrorNone {
		return false
	}
	return status.Phase != UpdatePhaseSucceeded || status.Progress == 100
}

func inspectApplicationUpdateUnit() (applicationUpdateUnitStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), applicationUnitInspectTimeout)
	defer cancel()
	output, err := exec.CommandContext(
		ctx, "systemctl", "show", updateUnitName+".service", "--no-pager",
		"--property=LoadState", "--property=ActiveState", "--property=SubState",
		"--property=Result", "--property=ExecMainCode", "--property=ExecMainStatus", "--property=Environment",
	).CombinedOutput()
	if ctx.Err() != nil {
		return applicationUpdateUnitStatus{}, fmt.Errorf("inspect application update unit")
	}
	if err != nil {
		message := strings.ToLower(string(output))
		if strings.Contains(message, "not found") || strings.Contains(message, "not-found") || strings.Contains(message, "could not be found") {
			return applicationUpdateUnitStatus{}, nil
		}
		return applicationUpdateUnitStatus{}, fmt.Errorf("inspect application update unit")
	}
	properties := make(map[string]string)
	for _, line := range strings.Split(string(output), "\n") {
		key, value, found := strings.Cut(line, "=")
		if found {
			properties[key] = strings.TrimSpace(value)
		}
	}
	if properties["LoadState"] == "not-found" {
		return applicationUpdateUnitStatus{}, nil
	}
	status := applicationUpdateUnitStatus{
		Exists:       properties["LoadState"] != "",
		ActiveState:  properties["ActiveState"],
		SubState:     properties["SubState"],
		Result:       properties["Result"],
		ExecMainCode: properties["ExecMainCode"],
	}
	if raw, exists := properties["ExecMainStatus"]; exists && raw != "" {
		value, parseErr := strconv.Atoi(raw)
		if parseErr != nil {
			return applicationUpdateUnitStatus{}, fmt.Errorf("inspect application update unit")
		}
		status.ExecMainStatus = value
		status.HasExecStatus = true
	}
	environment := parseApplicationUpdateUnitEnvironment(properties["Environment"])
	status.OperationID = environment[updateOperationEnvironment]
	status.Version = normalizeVersion(environment[updateVersionEnvironment])
	status.Source = UpdateSource(environment[updateSourceEnvironment])
	return status, nil
}

func parseApplicationUpdateUnitEnvironment(value string) map[string]string {
	environment := make(map[string]string)
	for _, field := range strings.Fields(value) {
		if strings.HasPrefix(field, `"`) {
			if unquoted, err := strconv.Unquote(field); err == nil {
				field = unquoted
			}
		}
		key, item, found := strings.Cut(field, "=")
		if !found || (key != updateOperationEnvironment && key != updateVersionEnvironment && key != updateSourceEnvironment) {
			continue
		}
		if _, duplicate := environment[key]; duplicate {
			environment[key] = ""
			continue
		}
		environment[key] = item
	}
	return environment
}
