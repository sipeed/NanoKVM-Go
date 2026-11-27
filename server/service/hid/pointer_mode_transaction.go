package hid

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"NanoKVM-Server/storageguard"
	"NanoKVM-Server/usbgadget"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

const (
	pointerModeGadgetRoot       = "/sys/kernel/config/usb_gadget/g0"
	pointerModeSwitchTimeout    = 30 * time.Second
	pointerModeRollbackTimeout  = 15 * time.Second
	pointerModeSettleDelay      = 3 * time.Second
	pointerModeVerifyPoll       = 25 * time.Millisecond
	pointerModeScriptKillWait   = 2 * time.Second
	pointerModeMarkerPermission = fs.FileMode(0o644)
)

var (
	ErrPointerEndpointCapacity      = errors.New("Cannot restore both pointers. Disable another USB device to free an IN endpoint.")
	ErrPointerModeSwitchInvalidMode = errors.New("invalid pointer mode")
	ErrPointerModeSwitchInProgress  = errors.New("pointer mode switch already in progress")
	ErrPointerModeSwitchLock        = errors.New("failed to acquire USB gadget mutation lock")
	ErrPointerModeSwitchQuiesce     = errors.New("failed to quiesce input router")
	ErrPointerModeSwitchHID         = errors.New("failed to prepare HID endpoints")
	ErrPointerModeSwitchPersistence = errors.New("pointer mode persistence failed")
	ErrPointerModeSwitchScript      = errors.New("USB gadget rebuild script failed")
	ErrPointerModeSwitchValidation  = errors.New("USB gadget validation failed")
	ErrPointerModeSwitchOpen        = errors.New("failed to reopen HID endpoints")
	ErrPointerModeSwitchResume      = errors.New("failed to resume input router")
	ErrPointerModeSwitchRollback    = errors.New("pointer mode rollback failed")
	ErrPointerModeSwitchDegraded    = errors.New("pointer mode switch left device degraded")
)

// PointerModeSwitchStage identifies the phase that reported an error. It is
// intentionally stable so a later HTTP handler can map stages to clear API
// responses without parsing log strings.
type PointerModeSwitchStage string

const (
	PointerModeSwitchStageLock        PointerModeSwitchStage = "lock"
	PointerModeSwitchStageSnapshot    PointerModeSwitchStage = "snapshot"
	PointerModeSwitchStageQuiesce     PointerModeSwitchStage = "quiesce"
	PointerModeSwitchStageHIDClose    PointerModeSwitchStage = "hid-close"
	PointerModeSwitchStagePersistence PointerModeSwitchStage = "persistence"
	PointerModeSwitchStageScript      PointerModeSwitchStage = "script"
	PointerModeSwitchStageValidation  PointerModeSwitchStage = "validation"
	PointerModeSwitchStageHIDOpen     PointerModeSwitchStage = "hid-open"
	PointerModeSwitchStageResume      PointerModeSwitchStage = "resume"
)

// PointerModeSwitchError preserves the failing stage and, when applicable, a
// separate rollback error. errors.Is traverses both errors through Unwrap.
type PointerModeSwitchError struct {
	Stage       PointerModeSwitchStage
	Err         error
	RollbackErr error
	Degraded    bool
}

func (err *PointerModeSwitchError) Error() string {
	if err == nil {
		return "<nil>"
	}
	message := fmt.Sprintf("pointer mode switch %s failed: %v", err.Stage, err.Err)
	if err.RollbackErr != nil {
		message += fmt.Sprintf("; rollback failed: %v", err.RollbackErr)
	}
	if err.Degraded {
		message += "; device is degraded and input remains quiesced"
	}
	return message
}

func (err *PointerModeSwitchError) Unwrap() []error {
	if err == nil {
		return nil
	}
	result := make([]error, 0, 2)
	if err.Err != nil {
		result = append(result, err.Err)
	}
	if err.RollbackErr != nil {
		result = append(result, err.RollbackErr)
	}
	return result
}

func pointerModeSwitchError(stage PointerModeSwitchStage, category, cause error) *PointerModeSwitchError {
	if cause == nil {
		cause = errors.New("unspecified error")
	}
	if category != nil {
		cause = fmt.Errorf("%w: %w", category, cause)
	}
	return &PointerModeSwitchError{Stage: stage, Err: cause}
}

type pointerModeMarkerSnapshot struct {
	exists bool
	data   []byte
	perm   fs.FileMode
}

func snapshotPointerModeMarker(path string) (pointerModeMarkerSnapshot, error) {
	if path == "" {
		return pointerModeMarkerSnapshot{}, errors.New("pointer mode marker path is empty")
	}

	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return pointerModeMarkerSnapshot{}, nil
	}
	if err != nil {
		return pointerModeMarkerSnapshot{}, err
	}
	if !info.Mode().IsRegular() {
		return pointerModeMarkerSnapshot{}, fmt.Errorf("pointer mode marker is not a regular file: %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return pointerModeMarkerSnapshot{}, err
	}
	perm := info.Mode().Perm()
	if perm == 0 {
		perm = pointerModeMarkerPermission
	}
	return pointerModeMarkerSnapshot{exists: true, data: append([]byte(nil), data...), perm: perm}, nil
}

func writePointerModeMarker(path string, mode PointerMode) error {
	if !isKnownPointerMode(mode) {
		return fmt.Errorf("%w: %q", ErrPointerModeSwitchInvalidMode, mode)
	}
	return writeAtomicSetting(path, []byte(string(mode)+"\n"), pointerModeMarkerPermission)
}

func restorePointerModeMarker(path string, snapshot pointerModeMarkerSnapshot) error {
	if path == "" {
		return errors.New("pointer mode marker path is empty")
	}
	if !snapshot.exists {
		err := os.Remove(path)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		return syncPointerModeDirectory(filepath.Dir(path))
	}

	perm := snapshot.perm.Perm()
	if perm == 0 {
		perm = pointerModeMarkerPermission
	}
	return writeAtomicSetting(path, snapshot.data, perm)
}

func writeAtomicSetting(path string, data []byte, perm fs.FileMode) error {
	if path == "" {
		return errors.New("setting path is empty")
	}
	directory := filepath.Dir(path)
	base := filepath.Base(path)
	temporary, err := os.CreateTemp(directory, "."+base+".tmp-*")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	closed := false
	renameComplete := false
	defer func() {
		if !closed {
			_ = temporary.Close()
		}
		if !renameComplete {
			_ = os.Remove(temporaryName)
		}
	}()

	if err := temporary.Chmod(perm.Perm()); err != nil {
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		closed = true
		return err
	}
	closed = true
	if err := os.Rename(temporaryName, path); err != nil {
		return err
	}
	renameComplete = true
	return syncPointerModeDirectory(directory)
}

func syncPointerModeDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	syncErr := directory.Sync()
	closeErr := directory.Close()
	// Some boot filesystems do not implement directory fsync. The rename is
	// still atomic there; do not turn a completed marker update into a false
	// transaction failure for EINVAL/ENOTSUP.
	if syncErr != nil && (errors.Is(syncErr, syscall.EINVAL) || errors.Is(syncErr, syscall.ENOTSUP)) {
		syncErr = nil
	}
	return errors.Join(syncErr, closeErr)
}

func pointerModeFromMarkerSnapshot(snapshot pointerModeMarkerSnapshot) PointerMode {
	if !snapshot.exists {
		return PointerModeSingleAbsolute
	}
	value := strings.TrimRight(string(snapshot.data), "\n")
	mode, ok := parsePointerMode(value)
	if !ok {
		return PointerModeAbsolute
	}
	return mode
}

func pointerModeReportLength(mode PointerMode) (int, bool) {
	switch mode.method() {
	case "relative":
		return 4, true
	case PointerModeAbsolute:
		return 7, true
	case PointerModeDigitizerStylus:
		return 7, true
	case PointerModeTouchScreen:
		return TouchScreenReportSize, true
	default:
		return 0, false
	}
}

type pointerModeGadgetVerifier struct {
	gadgetRoot string
	hid2Path   string
	openHID2   func(string) error
}

func newPointerModeGadgetVerifier(gadgetRoot, hid2Path string) pointerModeGadgetVerifier {
	verifier := pointerModeGadgetVerifier{
		gadgetRoot: gadgetRoot,
		hid2Path:   hid2Path,
	}
	verifier.openHID2 = probeHID2Endpoint
	return verifier
}

func (verifier pointerModeGadgetVerifier) wait(ctx context.Context, mode PointerMode) error {
	if ctx == nil {
		ctx = context.Background()
	}
	settleTimer := time.NewTimer(pointerModeSettleDelay)
	defer settleTimer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-settleTimer.C:
	}
	var lastErr error
	ticker := time.NewTicker(pointerModeVerifyPoll)
	defer ticker.Stop()
	for {
		if err := verifier.check(mode); err == nil {
			return nil
		} else {
			lastErr = err
		}

		select {
		case <-ctx.Done():
			if lastErr == nil {
				return ctx.Err()
			}
			return fmt.Errorf("%w: last validation error: %v", ctx.Err(), lastErr)
		case <-ticker.C:
		}
	}
}

func (verifier pointerModeGadgetVerifier) check(mode PointerMode) error {
	wantReportLength, ok := pointerModeReportLength(mode)
	if !ok {
		return fmt.Errorf("%w: %q", ErrPointerModeSwitchInvalidMode, mode)
	}
	if verifier.gadgetRoot == "" {
		return errors.New("USB gadget root is empty")
	}
	udcData, err := os.ReadFile(filepath.Join(verifier.gadgetRoot, "UDC"))
	if err != nil {
		return fmt.Errorf("read UDC: %w", err)
	}
	if strings.TrimSpace(string(udcData)) == "" {
		return errors.New("gadget is not bound to a UDC")
	}

	function, path := "hid.GS2", verifier.hid2Path
	if mode == PointerModeSingleRelative {
		function, path = "hid.GS1", HID1
	}
	for link, want := range map[string]bool{"hid.GS1": mode.SupportsRelative(), "hid.GS2": mode != PointerModeSingleRelative} {
		info, err := os.Lstat(filepath.Join(verifier.gadgetRoot, "configs", "c.1", link))
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if (info != nil && info.Mode()&os.ModeSymlink != 0) != want {
			return fmt.Errorf("unexpected pointer layout for %s", link)
		}
	}
	reportLengthData, err := os.ReadFile(filepath.Join(verifier.gadgetRoot, "functions", function, "report_length"))
	if err != nil {
		return fmt.Errorf("read HID2 report_length: %w", err)
	}
	reportLength, err := strconv.Atoi(strings.TrimSpace(string(reportLengthData)))
	if err != nil {
		return fmt.Errorf("parse HID2 report_length: %w", err)
	}
	if reportLength != wantReportLength {
		return fmt.Errorf("HID2 report_length %d, want %d", reportLength, wantReportLength)
	}

	openHID2 := verifier.openHID2
	if openHID2 == nil {
		openHID2 = probeHID2Endpoint
	}
	if err := openHID2(path); err != nil {
		return fmt.Errorf("probe %s: %w", path, err)
	}
	return nil
}

func probeHID2Endpoint(path string) error {
	if path == "" {
		return errors.New("HID2 endpoint path is empty")
	}
	fd, err := unix.Open(path, unix.O_WRONLY|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	return unix.Close(fd)
}

type pointerModeTransactionSnapshot struct {
	activeMode  PointerMode
	marker      pointerModeMarkerSnapshot
	usbMode     string
	wasDegraded bool
}

// pointerModeTransactionOptions is deliberately dependency-injected. The
// production constructor wires real files, commands and HID methods; tests can
// exercise every failure/rollback branch without touching a board.
type pointerModeTransactionOptions struct {
	modeFile        string
	scriptPath      string
	gadgetRoot      string
	hid2Path        string
	timeout         time.Duration
	rollbackTimeout time.Duration

	lockUSB     func(context.Context) error
	unlockUSB   func()
	getUSBMode  func() string
	lockStorage func() (pointerModeStorageLock, error)

	beginQuiesce func(string) (uint64, error)
	endQuiesce   func(uint64, string) error
	lockHID      func()
	unlockHID    func()
	closeHID     func() error
	openHID      func() error

	snapshotMarker func() (pointerModeMarkerSnapshot, error)
	writeMarker    func(PointerMode) error
	restoreMarker  func(pointerModeMarkerSnapshot) error
	runScript      func(context.Context, string, string, pointerModeStorageLock) error
	verify         func(context.Context, PointerMode) error
}

type pointerModeStorageLock interface {
	Attach(*exec.Cmd)
	Close()
}

type hidEndpointSet interface {
	Lock()
	Unlock()
	CloseNoLock() error
	OpenNoLock() error
}

type realHIDEndpointSet struct {
	hid *Hid
}

func (endpoints realHIDEndpointSet) Lock() {
	endpoints.hid.Lock()
}

func (endpoints realHIDEndpointSet) Unlock() {
	endpoints.hid.Unlock()
}

func (endpoints realHIDEndpointSet) CloseNoLock() error {
	endpoints.hid.CloseNoLock()
	return nil
}

func (endpoints realHIDEndpointSet) OpenNoLock() error {
	return endpoints.hid.openNoLockWithError()
}

type pointerModeTransaction struct {
	manager *pointerModeManager
	opts    pointerModeTransactionOptions
}

func newPointerModeTransaction(manager *pointerModeManager, h *Hid, options pointerModeTransactionOptions) *pointerModeTransaction {
	if h == nil {
		h = GetHid()
	}
	return newPointerModeTransactionWithEndpoints(manager, realHIDEndpointSet{hid: h}, h, options)
}

func newPointerModeTransactionWithEndpoints(
	manager *pointerModeManager,
	endpoints hidEndpointSet,
	routerHID *Hid,
	options pointerModeTransactionOptions,
) *pointerModeTransaction {
	if manager == nil {
		manager = newPointerModeManager(PointerModeFile)
	}
	if endpoints == nil {
		if routerHID == nil {
			routerHID = GetHid()
		}
		endpoints = realHIDEndpointSet{hid: routerHID}
	}
	if routerHID == nil {
		routerHID = GetHid()
	}
	if options.modeFile == "" {
		options.modeFile = manager.modeFile
		if options.modeFile == "" {
			options.modeFile = PointerModeFile
		}
	}
	if options.scriptPath == "" {
		options.scriptPath = USBScript
	}
	if options.gadgetRoot == "" {
		options.gadgetRoot = pointerModeGadgetRoot
	}
	if options.hid2Path == "" {
		options.hid2Path = HID2
	}
	if options.timeout <= 0 {
		options.timeout = pointerModeSwitchTimeout
	}
	if options.rollbackTimeout <= 0 {
		options.rollbackTimeout = pointerModeRollbackTimeout
	}
	if options.lockUSB == nil {
		options.lockUSB = LockUsbGadgetMutationContext
	}
	if options.unlockUSB == nil {
		options.unlockUSB = UnlockUsbGadgetMutation
	}
	if options.getUSBMode == nil {
		options.getUSBMode = getHidMode
	}
	if options.lockStorage == nil {
		options.lockStorage = func() (pointerModeStorageLock, error) { return storageguard.Exclusive() }
	}
	if options.beginQuiesce == nil {
		options.beginQuiesce = routerHID.BeginInputRoutingQuiesce
	}
	if options.endQuiesce == nil {
		options.endQuiesce = routerHID.EndInputRoutingQuiesce
	}
	if options.lockHID == nil {
		options.lockHID = endpoints.Lock
	}
	if options.unlockHID == nil {
		options.unlockHID = endpoints.Unlock
	}
	if options.closeHID == nil {
		options.closeHID = endpoints.CloseNoLock
	}
	if options.openHID == nil {
		options.openHID = endpoints.OpenNoLock
	}
	store := pointerModeMarkerStore{path: options.modeFile}
	if options.snapshotMarker == nil {
		options.snapshotMarker = store.snapshot
	}
	if options.writeMarker == nil {
		options.writeMarker = store.write
	}
	if options.restoreMarker == nil {
		options.restoreMarker = store.restore
	}
	if options.runScript == nil {
		options.runScript = runPointerModeScript
	}
	if options.verify == nil {
		verifier := newPointerModeGadgetVerifier(options.gadgetRoot, options.hid2Path)
		options.verify = verifier.wait
	}
	return &pointerModeTransaction{manager: manager, opts: options}
}

type pointerModeMarkerStore struct {
	path string
}

func (store pointerModeMarkerStore) snapshot() (pointerModeMarkerSnapshot, error) {
	return snapshotPointerModeMarker(store.path)
}

func (store pointerModeMarkerStore) write(mode PointerMode) error {
	return writePointerModeMarker(store.path, mode)
}

func (store pointerModeMarkerStore) restore(snapshot pointerModeMarkerSnapshot) error {
	return restorePointerModeMarker(store.path, snapshot)
}

func runPointerModeScript(ctx context.Context, scriptPath, arg string, storageLock pointerModeStorageLock) error {
	if arg != "restart" && arg != "hid-only" {
		return fmt.Errorf("unsupported USB rebuild argument %q", arg)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	command := exec.CommandContext(ctx, "bash", scriptPath, arg)
	storageLock.Attach(command)
	// usbdev.sh invokes several child processes. Put the command in its own
	// process group so a transaction timeout cannot leave an orphan continuing
	// to mutate ConfigFS while rollback starts.
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error {
		if command.Process == nil {
			return os.ErrProcessDone
		}
		err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	command.WaitDelay = pointerModeScriptKillWait
	output, err := command.CombinedOutput()
	if err == nil {
		return nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		err = errors.Join(ctxErr, err)
	}
	message := strings.TrimSpace(string(output))
	if len(message) > 4096 {
		message = message[:4096] + "..."
	}
	if message == "" {
		return err
	}
	return fmt.Errorf("%w: %s", err, message)
}

func (transaction *pointerModeTransaction) run(parent context.Context, requested PointerMode) (PointerModeStatus, error) {
	if !isKnownPointerMode(requested) {
		status := transaction.manager.status()
		return status, pointerModeSwitchError(PointerModeSwitchStageSnapshot, ErrPointerModeSwitchInvalidMode, fmt.Errorf("%q", requested))
	}
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, transaction.opts.timeout)
	defer cancel()

	if err := transaction.opts.lockUSB(ctx); err != nil {
		return transaction.manager.status(), pointerModeSwitchError(PointerModeSwitchStageLock, ErrPointerModeSwitchLock, err)
	}
	defer transaction.opts.unlockUSB()
	return transaction.runUSBLocked(ctx, requested)
}

// runUSBLocked executes the transaction while the caller holds the global USB
// gadget mutation lock. Reset uses this entry point to inspect degraded state
// and recover it atomically without dropping the lock between the check and
// the rebuild.
func (transaction *pointerModeTransaction) runUSBLocked(ctx context.Context, requested PointerMode) (PointerModeStatus, error) {
	if !isKnownPointerMode(requested) {
		status := transaction.manager.status()
		return status, pointerModeSwitchError(PointerModeSwitchStageSnapshot, ErrPointerModeSwitchInvalidMode, fmt.Errorf("%q", requested))
	}
	if ctx == nil {
		ctx = context.Background()
	}

	oldStatus, noChange, err := transaction.manager.beginSwitch(requested)
	if err != nil {
		return transaction.manager.status(), pointerModeSwitchError(PointerModeSwitchStageLock, err, nil)
	}
	if noChange {
		return oldStatus, nil
	}

	// Restoring the second pointer must not evict another function during rebuild.
	if oldStatus.Mode.SingleEndpoint() && !requested.SingleEndpoint() {
		used, err := usbgadget.INEndpoints(filepath.Join(transaction.opts.gadgetRoot, "configs", "c.1"), "hid.GS1", "hid.GS2")
		if err != nil || used > usbgadget.INEndpointLimit {
			transaction.manager.abortSwitch()
			return transaction.manager.status(), ErrPointerEndpointCapacity
		}
	}

	// Reject active file transfers before quiescing input or closing endpoints.
	// The same transaction must also cover validation, rollback and input resume.
	storageLock, err := transaction.opts.lockStorage()
	if err != nil {
		transaction.manager.abortSwitch()
		return transaction.manager.status(), pointerModeSwitchError(PointerModeSwitchStageLock, ErrPointerModeSwitchLock, err)
	}
	defer storageLock.Close()

	snapshot, err := transaction.takeSnapshot(oldStatus)
	if err != nil {
		transaction.manager.abortSwitch()
		return transaction.manager.status(), pointerModeSwitchError(PointerModeSwitchStageSnapshot, ErrPointerModeSwitchPersistence, err)
	}

	epoch, alreadyQuiesced := transaction.manager.degradedQuiesceEpoch()
	if !alreadyQuiesced {
		log.Infof("pointer mode switch: target=%s stage=quiesce", requested)
		epoch, err = transaction.opts.beginQuiesce("pointer mode switch")
	}
	if epoch == 0 {
		if err == nil {
			err = errors.New("input router returned an empty quiesce epoch")
		}
		transaction.manager.abortSwitch()
		return transaction.manager.status(), pointerModeSwitchError(PointerModeSwitchStageQuiesce, ErrPointerModeSwitchQuiesce, err)
	}
	if err != nil {
		// A host may temporarily stop polling HID endpoints while a previous USB
		// re-enumeration settles. Router state is already cleared and the odd epoch
		// blocks new input, so a failed best-effort neutral report must not prevent
		// the gadget rebuild that will disconnect and reset the old USB device.
		log.Warnf("pointer mode switch: target=%s neutral release failed; continuing while quiesced: %s", requested, err)
	}

	hidLocked := false
	markerMutationAttempted := false
	rebuildAttempted := false
	transaction.opts.lockHID()
	hidLocked = true

	if err := transaction.opts.closeHID(); err != nil {
		return transaction.fail(storageLock, snapshot, epoch, hidLocked, markerMutationAttempted, rebuildAttempted,
			PointerModeSwitchStageHIDClose, ErrPointerModeSwitchHID, err)
	}

	markerMutationAttempted = true
	log.Infof("pointer mode switch: target=%s stage=persistence", requested)
	if err := transaction.opts.writeMarker(requested); err != nil {
		return transaction.fail(storageLock, snapshot, epoch, hidLocked, markerMutationAttempted, rebuildAttempted,
			PointerModeSwitchStagePersistence, ErrPointerModeSwitchPersistence, err)
	}

	rebuildAttempted = true
	log.Infof("pointer mode switch: target=%s stage=script usb-mode=%s", requested, snapshot.usbMode)
	rebuildArg := hidModeCmdMap[snapshot.usbMode]
	if err := transaction.opts.runScript(ctx, transaction.opts.scriptPath, rebuildArg, storageLock); err != nil {
		return transaction.fail(storageLock, snapshot, epoch, hidLocked, markerMutationAttempted, rebuildAttempted,
			PointerModeSwitchStageScript, ErrPointerModeSwitchScript, err)
	}

	log.Infof("pointer mode switch: target=%s stage=validation", requested)
	if err := transaction.opts.verify(ctx, requested); err != nil {
		return transaction.fail(storageLock, snapshot, epoch, hidLocked, markerMutationAttempted, rebuildAttempted,
			PointerModeSwitchStageValidation, ErrPointerModeSwitchValidation, err)
	}

	log.Infof("pointer mode switch: target=%s stage=hid-open", requested)
	if err := transaction.opts.openHID(); err != nil {
		return transaction.fail(storageLock, snapshot, epoch, hidLocked, markerMutationAttempted, rebuildAttempted,
			PointerModeSwitchStageHIDOpen, ErrPointerModeSwitchOpen, err)
	}

	transaction.manager.commitSwitch(requested)
	transaction.opts.unlockHID()
	hidLocked = false
	if err := transaction.opts.endQuiesce(epoch, "pointer mode switch committed"); err != nil {
		degradedErr := fmt.Errorf("%w: %w", ErrPointerModeSwitchResume, err)
		transaction.manager.markDegraded(degradedErr, epoch)
		return transaction.manager.status(), &PointerModeSwitchError{
			Stage:    PointerModeSwitchStageResume,
			Err:      degradedErr,
			Degraded: true,
		}
	}
	transaction.manager.finishSwitch()
	log.Infof("pointer mode switch: target=%s stage=committed", requested)
	return transaction.manager.status(), nil
}

// recoverDegradedUSBLocked rebuilds the canonical profile selected by the
// persisted marker. The global gadget mutation lock must already be held.
func (transaction *pointerModeTransaction) recoverDegradedUSBLocked(parent context.Context) (PointerModeStatus, error) {
	status := transaction.manager.status()
	if !status.Degraded {
		return status, nil
	}
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, transaction.opts.timeout)
	defer cancel()

	target := readPointerModeFile(transaction.opts.modeFile)
	status, err := transaction.runUSBLocked(ctx, target)
	// A failed first rebuild can still complete reset through the transaction's
	// verified rollback, because the reset target and the persisted old mode are
	// identical. In that case recovery succeeded even though the first attempt
	// produced a diagnostic error.
	if err != nil && !status.Degraded && status.Mode == target {
		log.Warnf("degraded HID reset recovered %s through verified rollback: %s", target, err)
		return status, nil
	}
	return status, err
}

func (transaction *pointerModeTransaction) takeSnapshot(oldStatus PointerModeStatus) (pointerModeTransactionSnapshot, error) {
	marker, err := transaction.opts.snapshotMarker()
	if err != nil {
		return pointerModeTransactionSnapshot{}, err
	}
	usbMode := transaction.opts.getUSBMode()
	if _, ok := hidModeCmdMap[usbMode]; !ok {
		return pointerModeTransactionSnapshot{}, fmt.Errorf("invalid current USB mode %q", usbMode)
	}
	activeMode := oldStatus.Mode
	if !isKnownPointerMode(activeMode) {
		activeMode = pointerModeFromMarkerSnapshot(marker)
	}
	return pointerModeTransactionSnapshot{
		activeMode:  activeMode,
		marker:      marker,
		usbMode:     usbMode,
		wasDegraded: oldStatus.Degraded,
	}, nil
}

func (transaction *pointerModeTransaction) fail(
	storageLock pointerModeStorageLock,
	snapshot pointerModeTransactionSnapshot,
	epoch uint64,
	hidLocked bool,
	markerMutationAttempted bool,
	rebuildAttempted bool,
	stage PointerModeSwitchStage,
	category error,
	cause error,
) (PointerModeStatus, error) {
	primary := pointerModeSwitchError(stage, category, cause)
	log.Warnf("pointer mode switch failed: old-mode=%s stage=%s: %s", snapshot.activeMode, stage, cause)

	rollbackErr := transaction.rollback(storageLock, snapshot, hidLocked, markerMutationAttempted, rebuildAttempted)
	if rollbackErr != nil {
		degradedCause := errors.Join(primary, rollbackErr)
		transaction.manager.markDegraded(degradedCause, epoch)
		primary.RollbackErr = fmt.Errorf("%w: %w: %w", ErrPointerModeSwitchRollback, ErrPointerModeSwitchDegraded, rollbackErr)
		primary.Degraded = true
		return transaction.manager.status(), primary
	}
	log.Infof("pointer mode switch: old-mode=%s stage=rollback-validated", snapshot.activeMode)

	transaction.manager.commitSwitch(snapshot.activeMode)
	if err := transaction.opts.endQuiesce(epoch, "pointer mode switch rolled back"); err != nil {
		degradedCause := errors.Join(primary, fmt.Errorf("%w: %w", ErrPointerModeSwitchResume, err))
		transaction.manager.markDegraded(degradedCause, epoch)
		primary.RollbackErr = fmt.Errorf("%w: %w: %w", ErrPointerModeSwitchResume, ErrPointerModeSwitchDegraded, err)
		primary.Degraded = true
		return transaction.manager.status(), primary
	}
	transaction.manager.finishSwitch()
	return transaction.manager.status(), primary
}

func (transaction *pointerModeTransaction) rollback(
	storageLock pointerModeStorageLock,
	snapshot pointerModeTransactionSnapshot,
	hidLocked bool,
	markerMutationAttempted bool,
	rebuildAttempted bool,
) error {
	rollbackCtx, cancel := context.WithTimeout(context.Background(), transaction.opts.rollbackTimeout)
	defer cancel()
	var rollbackErrors []error
	degradedRecovery := snapshot.wasDegraded

	if !hidLocked {
		transaction.opts.lockHID()
		hidLocked = true
	}
	// A failed open can leave a partial set of handles. Always close before
	// restoring the old descriptor so no writer can retain a stale endpoint.
	if err := transaction.opts.closeHID(); err != nil {
		rollbackErrors = append(rollbackErrors, fmt.Errorf("close HID for rollback: %w", err))
	}

	markerRestored := true
	if markerMutationAttempted {
		if err := transaction.opts.restoreMarker(snapshot.marker); err != nil {
			markerRestored = false
			rollbackErrors = append(rollbackErrors, fmt.Errorf("restore pointer mode marker: %w", err))
		}
	}
	if markerRestored && (rebuildAttempted || degradedRecovery) {
		rebuildArg := hidModeCmdMap[snapshot.usbMode]
		if err := transaction.opts.runScript(rollbackCtx, transaction.opts.scriptPath, rebuildArg, storageLock); err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("rebuild old USB descriptor: %w", err))
		} else if err := transaction.opts.verify(rollbackCtx, snapshot.activeMode); err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("validate rolled-back USB descriptor: %w", err))
		}
	}

	if len(rollbackErrors) == 0 {
		if err := transaction.opts.openHID(); err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("reopen old HID endpoints: %w", err))
		}
	}
	transaction.opts.unlockHID()
	return errors.Join(rollbackErrors...)
}
