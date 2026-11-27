package hid

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"NanoKVM-Server/storageguard"
)

type pointerModeEventRecorder struct {
	mutex  sync.Mutex
	events []string
}

func (recorder *pointerModeEventRecorder) add(event string) {
	recorder.mutex.Lock()
	recorder.events = append(recorder.events, event)
	recorder.mutex.Unlock()
}

func (recorder *pointerModeEventRecorder) snapshot() []string {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	return append([]string(nil), recorder.events...)
}

type fakePointerModeEndpoints struct {
	mutex       sync.Mutex
	recorder    *pointerModeEventRecorder
	locked      bool
	violations  []string
	closeErrors []error
	openErrors  []error
}

func (endpoints *fakePointerModeEndpoints) Lock() {
	endpoints.mutex.Lock()
	if endpoints.locked {
		endpoints.violations = append(endpoints.violations, "double lock")
	}
	endpoints.locked = true
	endpoints.mutex.Unlock()
	endpoints.recorder.add("hid-lock")
}

func (endpoints *fakePointerModeEndpoints) Unlock() {
	endpoints.mutex.Lock()
	if !endpoints.locked {
		endpoints.violations = append(endpoints.violations, "unlock without lock")
	}
	endpoints.locked = false
	endpoints.mutex.Unlock()
	endpoints.recorder.add("hid-unlock")
}

func (endpoints *fakePointerModeEndpoints) CloseNoLock() error {
	endpoints.recorder.add("hid-close")
	endpoints.mutex.Lock()
	defer endpoints.mutex.Unlock()
	if !endpoints.locked {
		endpoints.violations = append(endpoints.violations, "close without lock")
	}
	return popPointerModeTestError(&endpoints.closeErrors)
}

func (endpoints *fakePointerModeEndpoints) OpenNoLock() error {
	endpoints.recorder.add("hid-open")
	endpoints.mutex.Lock()
	defer endpoints.mutex.Unlock()
	if !endpoints.locked {
		endpoints.violations = append(endpoints.violations, "open without lock")
	}
	return popPointerModeTestError(&endpoints.openErrors)
}

func (endpoints *fakePointerModeEndpoints) state() (bool, []string) {
	endpoints.mutex.Lock()
	defer endpoints.mutex.Unlock()
	return endpoints.locked, append([]string(nil), endpoints.violations...)
}

func popPointerModeTestError(errorsList *[]error) error {
	if len(*errorsList) == 0 {
		return nil
	}
	err := (*errorsList)[0]
	*errorsList = (*errorsList)[1:]
	return err
}

type pointerModeTestLock struct {
	token chan struct{}
}

func newPointerModeTestLock() *pointerModeTestLock {
	lock := &pointerModeTestLock{token: make(chan struct{}, 1)}
	lock.token <- struct{}{}
	return lock
}

func (lock *pointerModeTestLock) acquire(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-lock.token:
		return nil
	}
}

func (lock *pointerModeTestLock) release() {
	lock.token <- struct{}{}
}

type fakePointerModeStorageLock struct {
	token    chan struct{}
	recorder *pointerModeEventRecorder
}

func (lock *fakePointerModeStorageLock) acquire() (pointerModeStorageLock, error) {
	select {
	case <-lock.token:
		lock.recorder.add("storage-lock")
		return lock, nil
	default:
		return nil, storageguard.ErrBusy
	}
}

func (lock *fakePointerModeStorageLock) Close() {
	lock.recorder.add("storage-unlock")
	lock.token <- struct{}{}
}

func (lock *fakePointerModeStorageLock) Attach(command *exec.Cmd) {
	command.Env = append(os.Environ(), "NANOKVM_POINTER_MODE_TEST_LOCK=held")
}

type pointerModeTransactionHarness struct {
	t            *testing.T
	modeFile     string
	manager      *pointerModeManager
	transaction  *pointerModeTransaction
	endpoints    *fakePointerModeEndpoints
	recorder     *pointerModeEventRecorder
	usbLock      *pointerModeTestLock
	storageLock  *fakePointerModeStorageLock
	usbMode      string
	epoch        uint64
	mutex        sync.Mutex
	beginErr     error
	endErrors    []error
	scriptErrors []error
	verifyErrors []error
	beginCalls   int
	endCalls     int
	scriptCalls  []string
	verifyCalls  []PointerMode
	scriptHook   func(int, PointerMode)
}

func newPointerModeTransactionHarness(
	t *testing.T,
	initialMode PointerMode,
	marker *string,
	usbMode string,
) *pointerModeTransactionHarness {
	t.Helper()
	modeFile := filepath.Join(t.TempDir(), "usb.hid2_mode")
	if marker != nil {
		if err := os.WriteFile(modeFile, []byte(*marker), 0o600); err != nil {
			t.Fatalf("write initial pointer marker: %v", err)
		}
	}
	recorder := &pointerModeEventRecorder{}
	endpoints := &fakePointerModeEndpoints{recorder: recorder}
	storageLock := &fakePointerModeStorageLock{token: make(chan struct{}, 1), recorder: recorder}
	storageLock.token <- struct{}{}
	harness := &pointerModeTransactionHarness{
		t:           t,
		modeFile:    modeFile,
		manager:     &pointerModeManager{modeFile: modeFile, activeMode: initialMode},
		endpoints:   endpoints,
		recorder:    recorder,
		usbLock:     newPointerModeTestLock(),
		storageLock: storageLock,
		usbMode:     usbMode,
		epoch:       17,
	}
	options := pointerModeTransactionOptions{
		modeFile:        modeFile,
		scriptPath:      "/fake/usbdev.sh",
		timeout:         time.Second,
		rollbackTimeout: time.Second,
		lockUSB:         harness.lockUSB,
		unlockUSB:       harness.unlockUSB,
		getUSBMode:      func() string { return harness.usbMode },
		lockStorage:     storageLock.acquire,
		beginQuiesce:    harness.beginQuiesce,
		endQuiesce:      harness.endQuiesce,
		runScript:       harness.runScript,
		verify:          harness.verify,
	}
	harness.transaction = newPointerModeTransactionWithEndpoints(harness.manager, endpoints, nil, options)
	return harness
}

func (harness *pointerModeTransactionHarness) lockUSB(ctx context.Context) error {
	if err := harness.usbLock.acquire(ctx); err != nil {
		return err
	}
	harness.recorder.add("usb-lock")
	return nil
}

func (harness *pointerModeTransactionHarness) unlockUSB() {
	harness.recorder.add("usb-unlock")
	harness.usbLock.release()
}

func (harness *pointerModeTransactionHarness) beginQuiesce(string) (uint64, error) {
	harness.recorder.add("quiesce")
	harness.mutex.Lock()
	defer harness.mutex.Unlock()
	harness.beginCalls++
	return harness.epoch, harness.beginErr
}

func (harness *pointerModeTransactionHarness) endQuiesce(uint64, string) error {
	harness.recorder.add("resume")
	harness.mutex.Lock()
	defer harness.mutex.Unlock()
	harness.endCalls++
	return popPointerModeTestError(&harness.endErrors)
}

func (harness *pointerModeTransactionHarness) runScript(_ context.Context, _ string, arg string, storageLock pointerModeStorageLock) error {
	if storageLock != harness.storageLock || len(harness.storageLock.token) != 0 {
		harness.t.Error("USB rebuild did not retain the storage transaction")
	}
	data, err := os.ReadFile(harness.modeFile)
	markerMode := PointerMode("missing")
	if err == nil {
		markerMode = PointerMode(strings.TrimSpace(string(data)))
	} else if !os.IsNotExist(err) {
		harness.t.Errorf("read marker in script: %v", err)
	}

	harness.mutex.Lock()
	call := len(harness.scriptCalls)
	harness.scriptCalls = append(harness.scriptCalls, arg)
	scriptErr := popPointerModeTestError(&harness.scriptErrors)
	hook := harness.scriptHook
	harness.mutex.Unlock()
	harness.recorder.add(fmt.Sprintf("script:%s:%s", arg, markerMode))
	if hook != nil {
		hook(call, markerMode)
	}
	return scriptErr
}

func (harness *pointerModeTransactionHarness) verify(_ context.Context, mode PointerMode) error {
	harness.recorder.add("verify:" + string(mode))
	harness.mutex.Lock()
	defer harness.mutex.Unlock()
	harness.verifyCalls = append(harness.verifyCalls, mode)
	return popPointerModeTestError(&harness.verifyErrors)
}

func (harness *pointerModeTransactionHarness) callSnapshot() (int, int, []string, []PointerMode) {
	harness.mutex.Lock()
	defer harness.mutex.Unlock()
	return harness.beginCalls, harness.endCalls,
		append([]string(nil), harness.scriptCalls...), append([]PointerMode(nil), harness.verifyCalls...)
}

func (harness *pointerModeTransactionHarness) assertLocksReleased(t *testing.T) {
	t.Helper()
	locked, violations := harness.endpoints.state()
	if locked || len(violations) != 0 {
		t.Fatalf("HID endpoint state locked=%t violations=%v", locked, violations)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if err := harness.usbLock.acquire(ctx); err != nil {
		t.Fatalf("USB mutation lock was not released: %v", err)
	}
	harness.usbLock.release()
	lock, err := harness.storageLock.acquire()
	if err != nil {
		t.Fatalf("storage mutation lock was not released: %v", err)
	}
	lock.Close()
}

func TestPointerModeTransactionSuccess(t *testing.T) {
	marker := "absolute\n"
	harness := newPointerModeTransactionHarness(t, PointerModeAbsolute, &marker, ModeNormal)

	status, err := harness.transaction.run(context.Background(), PointerModeDigitizerStylus)
	if err != nil {
		t.Fatalf("pointer mode transaction: %v", err)
	}
	if status.Mode != PointerModeDigitizerStylus || status.Switching || status.Degraded || status.Error != "" {
		t.Fatalf("status = %+v", status)
	}
	data, err := os.ReadFile(harness.modeFile)
	if err != nil || string(data) != "digitizer-stylus\n" {
		t.Fatalf("persisted marker = %q, %v", data, err)
	}
	wantEvents := []string{
		"usb-lock", "storage-lock", "quiesce", "hid-lock", "hid-close",
		"script:restart:digitizer-stylus", "verify:digitizer-stylus",
		"hid-open", "hid-unlock", "resume", "storage-unlock", "usb-unlock",
	}
	if events := harness.recorder.snapshot(); fmt.Sprint(events) != fmt.Sprint(wantEvents) {
		t.Fatalf("events = %v, want %v", events, wantEvents)
	}
	harness.assertLocksReleased(t)
}

func TestPointerModeTransactionRejectsTransfersBeforePausingInput(t *testing.T) {
	for _, usbMode := range []string{ModeNormal, ModeHidOnly} {
		t.Run(usbMode, func(t *testing.T) {
			marker := "absolute\n"
			harness := newPointerModeTransactionHarness(t, PointerModeAbsolute, &marker, usbMode)
			transfer, err := harness.storageLock.acquire()
			if err != nil {
				t.Fatal(err)
			}

			status, err := harness.transaction.run(context.Background(), PointerModeDigitizerStylus)
			if !errors.Is(err, storageguard.ErrBusy) {
				t.Fatalf("switch during transfer: %v", err)
			}
			assertRolledBackPointerModeStatus(t, status, PointerModeAbsolute)
			data, err := os.ReadFile(harness.modeFile)
			if err != nil || string(data) != marker {
				t.Fatalf("busy switch changed marker: %q, %v", data, err)
			}
			wantEvents := []string{"storage-lock", "usb-lock", "usb-unlock"}
			if events := harness.recorder.snapshot(); fmt.Sprint(events) != fmt.Sprint(wantEvents) {
				t.Fatalf("busy switch touched input: %v", events)
			}
			if len(harness.storageLock.token) != 0 {
				t.Fatal("busy switch released the transfer lock")
			}

			transfer.Close()
			if _, err := harness.transaction.run(context.Background(), PointerModeDigitizerStylus); err != nil {
				t.Fatalf("switch after transfer: %v", err)
			}
			harness.assertLocksReleased(t)
		})
	}
}

func TestPointerModeTransactionSameModeIsIdempotent(t *testing.T) {
	marker := "absolute\n"
	harness := newPointerModeTransactionHarness(t, PointerModeAbsolute, &marker, ModeNormal)

	status, err := harness.transaction.run(context.Background(), PointerModeAbsolute)
	if err != nil {
		t.Fatalf("idempotent transaction: %v", err)
	}
	if status.Mode != PointerModeAbsolute || status.Switching || status.Degraded {
		t.Fatalf("status = %+v", status)
	}
	if events := harness.recorder.snapshot(); fmt.Sprint(events) != fmt.Sprint([]string{"usb-lock", "usb-unlock"}) {
		t.Fatalf("idempotent events = %v", events)
	}
	begin, end, scripts, verifies := harness.callSnapshot()
	if begin != 0 || end != 0 || len(scripts) != 0 || len(verifies) != 0 {
		t.Fatalf("idempotent calls begin=%d end=%d scripts=%v verifies=%v", begin, end, scripts, verifies)
	}
	harness.assertLocksReleased(t)
}

func TestPointerModeTransactionUsesCurrentUSBModeCommand(t *testing.T) {
	for _, test := range []struct {
		usbMode string
		wantArg string
	}{
		{usbMode: ModeNormal, wantArg: "restart"},
		{usbMode: ModeHidOnly, wantArg: "hid-only"},
	} {
		t.Run(test.usbMode, func(t *testing.T) {
			marker := "absolute\n"
			harness := newPointerModeTransactionHarness(t, PointerModeAbsolute, &marker, test.usbMode)
			if _, err := harness.transaction.run(context.Background(), PointerModeTouchScreen); err != nil {
				t.Fatalf("pointer mode transaction: %v", err)
			}
			_, _, scripts, _ := harness.callSnapshot()
			if len(scripts) != 1 || scripts[0] != test.wantArg {
				t.Fatalf("script args = %v, want [%s]", scripts, test.wantArg)
			}
		})
	}
}

func TestPointerModeTransactionsAreSerialized(t *testing.T) {
	marker := "absolute\n"
	harness := newPointerModeTransactionHarness(t, PointerModeAbsolute, &marker, ModeNormal)
	firstScriptEntered := make(chan struct{})
	releaseFirstScript := make(chan struct{})
	harness.scriptHook = func(call int, _ PointerMode) {
		if call == 0 {
			close(firstScriptEntered)
			<-releaseFirstScript
		}
	}

	type result struct {
		status PointerModeStatus
		err    error
	}
	firstDone := make(chan result, 1)
	secondDone := make(chan result, 1)
	go func() {
		status, err := harness.transaction.run(context.Background(), PointerModeDigitizerStylus)
		firstDone <- result{status: status, err: err}
	}()
	select {
	case <-firstScriptEntered:
	case <-time.After(time.Second):
		t.Fatal("first transaction did not reach script stage")
	}
	go func() {
		status, err := harness.transaction.run(context.Background(), PointerModeTouchScreen)
		secondDone <- result{status: status, err: err}
	}()
	select {
	case premature := <-secondDone:
		t.Fatalf("second transaction bypassed USB lock: %+v", premature)
	case <-time.After(25 * time.Millisecond):
	}
	if status := harness.manager.status(); !status.Switching || status.Mode != PointerModeAbsolute {
		t.Fatalf("status while first transaction blocked = %+v", status)
	}
	close(releaseFirstScript)

	first := <-firstDone
	second := <-secondDone
	if first.err != nil || first.status.Mode != PointerModeDigitizerStylus {
		t.Fatalf("first result = %+v, %v", first.status, first.err)
	}
	if second.err != nil || second.status.Mode != PointerModeTouchScreen {
		t.Fatalf("second result = %+v, %v", second.status, second.err)
	}
	data, err := os.ReadFile(harness.modeFile)
	if err != nil || string(data) != "touch-screen\n" {
		t.Fatalf("final marker = %q, %v", data, err)
	}
	begin, end, scripts, verifies := harness.callSnapshot()
	if begin != 2 || end != 2 || fmt.Sprint(scripts) != fmt.Sprint([]string{"restart", "restart"}) ||
		fmt.Sprint(verifies) != fmt.Sprint([]PointerMode{PointerModeDigitizerStylus, PointerModeTouchScreen}) {
		t.Fatalf("calls begin=%d end=%d scripts=%v verifies=%v", begin, end, scripts, verifies)
	}
	harness.assertLocksReleased(t)
}

func TestPointerModeTransactionScriptFailureRollsBack(t *testing.T) {
	marker := "absolute\n"
	harness := newPointerModeTransactionHarness(t, PointerModeAbsolute, &marker, ModeNormal)
	scriptFailure := errors.New("injected script failure")
	harness.scriptErrors = []error{scriptFailure, nil}

	status, err := harness.transaction.run(context.Background(), PointerModeDigitizerStylus)
	if !errors.Is(err, ErrPointerModeSwitchScript) || !errors.Is(err, scriptFailure) {
		t.Fatalf("transaction error = %v, want script failure", err)
	}
	assertRolledBackPointerModeStatus(t, status, PointerModeAbsolute)
	data, readErr := os.ReadFile(harness.modeFile)
	if readErr != nil || string(data) != marker {
		t.Fatalf("rolled-back marker = %q, %v", data, readErr)
	}
	begin, end, scripts, verifies := harness.callSnapshot()
	if begin != 1 || end != 1 || fmt.Sprint(scripts) != fmt.Sprint([]string{"restart", "restart"}) ||
		fmt.Sprint(verifies) != fmt.Sprint([]PointerMode{PointerModeAbsolute}) {
		t.Fatalf("calls begin=%d end=%d scripts=%v verifies=%v", begin, end, scripts, verifies)
	}
	harness.assertLocksReleased(t)
}

func TestPointerModeTransactionValidationFailureRollsBack(t *testing.T) {
	marker := "absolute\n"
	harness := newPointerModeTransactionHarness(t, PointerModeAbsolute, &marker, ModeHidOnly)
	validationFailure := errors.New("injected profile validation failure")
	harness.verifyErrors = []error{validationFailure, nil}

	status, err := harness.transaction.run(context.Background(), PointerModeTouchScreen)
	if !errors.Is(err, ErrPointerModeSwitchValidation) || !errors.Is(err, validationFailure) {
		t.Fatalf("transaction error = %v, want validation failure", err)
	}
	assertRolledBackPointerModeStatus(t, status, PointerModeAbsolute)
	_, _, scripts, verifies := harness.callSnapshot()
	if fmt.Sprint(scripts) != fmt.Sprint([]string{"hid-only", "hid-only"}) ||
		fmt.Sprint(verifies) != fmt.Sprint([]PointerMode{PointerModeTouchScreen, PointerModeAbsolute}) {
		t.Fatalf("scripts=%v verifies=%v", scripts, verifies)
	}
	harness.assertLocksReleased(t)
}

func TestPointerModeTransactionRestoresMissingMarker(t *testing.T) {
	harness := newPointerModeTransactionHarness(t, PointerModeSingleAbsolute, nil, ModeNormal)
	harness.scriptErrors = []error{errors.New("injected initial rebuild failure"), nil}

	status, err := harness.transaction.run(context.Background(), PointerModeSingleStylus)
	if !errors.Is(err, ErrPointerModeSwitchScript) {
		t.Fatalf("transaction error = %v, want script failure", err)
	}
	assertRolledBackPointerModeStatus(t, status, PointerModeSingleAbsolute)
	if _, statErr := os.Stat(harness.modeFile); !os.IsNotExist(statErr) {
		t.Fatalf("missing marker rollback stat error = %v, want not exists", statErr)
	}
	_, _, _, verifies := harness.callSnapshot()
	if fmt.Sprint(verifies) != fmt.Sprint([]PointerMode{PointerModeSingleAbsolute}) {
		t.Fatalf("rollback verifies = %v, want single-absolute", verifies)
	}
	harness.assertLocksReleased(t)
}

func TestPointerModeTransactionPersistenceFailureRestoresOldMarker(t *testing.T) {
	marker := "absolute-android\n"
	harness := newPointerModeTransactionHarness(t, PointerModeDigitizerStylus, &marker, ModeNormal)
	persistenceFailure := errors.New("injected persistence failure")
	harness.transaction.opts.writeMarker = func(PointerMode) error {
		if err := os.WriteFile(harness.modeFile, []byte("partial"), 0o600); err != nil {
			t.Fatalf("write partial marker: %v", err)
		}
		return persistenceFailure
	}

	status, err := harness.transaction.run(context.Background(), PointerModeTouchScreen)
	if !errors.Is(err, ErrPointerModeSwitchPersistence) || !errors.Is(err, persistenceFailure) {
		t.Fatalf("transaction error = %v, want persistence failure", err)
	}
	assertRolledBackPointerModeStatus(t, status, PointerModeDigitizerStylus)
	data, readErr := os.ReadFile(harness.modeFile)
	if readErr != nil || string(data) != marker {
		t.Fatalf("restored legacy marker = %q, %v", data, readErr)
	}
	_, _, scripts, verifies := harness.callSnapshot()
	if len(scripts) != 0 || len(verifies) != 0 {
		t.Fatalf("persistence failure rebuilt gadget: scripts=%v verifies=%v", scripts, verifies)
	}
	harness.assertLocksReleased(t)
}

func TestPointerModeTransactionHIDOpenFailureRollsBack(t *testing.T) {
	marker := "absolute\n"
	harness := newPointerModeTransactionHarness(t, PointerModeAbsolute, &marker, ModeNormal)
	openFailure := errors.New("injected HID open failure")
	harness.endpoints.openErrors = []error{openFailure, nil}

	status, err := harness.transaction.run(context.Background(), PointerModeTouchScreen)
	if !errors.Is(err, ErrPointerModeSwitchOpen) || !errors.Is(err, openFailure) {
		t.Fatalf("transaction error = %v, want HID open failure", err)
	}
	assertRolledBackPointerModeStatus(t, status, PointerModeAbsolute)
	_, _, scripts, verifies := harness.callSnapshot()
	if fmt.Sprint(scripts) != fmt.Sprint([]string{"restart", "restart"}) ||
		fmt.Sprint(verifies) != fmt.Sprint([]PointerMode{PointerModeTouchScreen, PointerModeAbsolute}) {
		t.Fatalf("scripts=%v verifies=%v", scripts, verifies)
	}
	harness.assertLocksReleased(t)
}

func TestPointerModeTransactionContinuesAfterNeutralReleaseFailure(t *testing.T) {
	marker := "absolute\n"
	harness := newPointerModeTransactionHarness(t, PointerModeAbsolute, &marker, ModeNormal)
	harness.beginErr = errors.New("injected neutral release failure")

	status, err := harness.transaction.run(context.Background(), PointerModeDigitizerStylus)
	if err != nil {
		t.Fatalf("transaction error = %v, want best-effort release failure to continue", err)
	}
	if status.Mode != PointerModeDigitizerStylus || status.Switching || status.Degraded {
		t.Fatalf("final status = %+v, want healthy digitizer-stylus", status)
	}
	data, readErr := os.ReadFile(harness.modeFile)
	if readErr != nil || string(data) != "digitizer-stylus\n" {
		t.Fatalf("final marker = %q, %v", data, readErr)
	}
	begin, end, scripts, verifies := harness.callSnapshot()
	if begin != 1 || end != 1 || fmt.Sprint(scripts) != fmt.Sprint([]string{"restart"}) ||
		fmt.Sprint(verifies) != fmt.Sprint([]PointerMode{PointerModeDigitizerStylus}) {
		t.Fatalf("calls begin=%d end=%d scripts=%v verifies=%v", begin, end, scripts, verifies)
	}
	if events := harness.recorder.snapshot(); fmt.Sprint(events) != fmt.Sprint([]string{
		"usb-lock", "storage-lock", "quiesce", "hid-lock", "hid-close", "script:restart:digitizer-stylus",
		"verify:digitizer-stylus", "hid-open", "hid-unlock", "resume", "storage-unlock", "usb-unlock",
	}) {
		t.Fatalf("best-effort release events = %v", events)
	}
	harness.assertLocksReleased(t)
}

func TestPointerModeTransactionRejectsMissingQuiesceEpoch(t *testing.T) {
	marker := "absolute\n"
	harness := newPointerModeTransactionHarness(t, PointerModeAbsolute, &marker, ModeNormal)
	harness.epoch = 0
	harness.beginErr = errors.New("input router unavailable")

	status, err := harness.transaction.run(context.Background(), PointerModeDigitizerStylus)
	if !errors.Is(err, ErrPointerModeSwitchQuiesce) {
		t.Fatalf("transaction error = %v, want quiesce failure", err)
	}
	assertRolledBackPointerModeStatus(t, status, PointerModeAbsolute)
	begin, end, scripts, verifies := harness.callSnapshot()
	if begin != 1 || end != 0 || len(scripts) != 0 || len(verifies) != 0 {
		t.Fatalf("calls begin=%d end=%d scripts=%v verifies=%v", begin, end, scripts, verifies)
	}
	harness.assertLocksReleased(t)
}

func TestPointerModeTransactionResumeFailureMarksDegradedAndCanRecover(t *testing.T) {
	marker := "absolute\n"
	harness := newPointerModeTransactionHarness(t, PointerModeAbsolute, &marker, ModeNormal)
	resumeFailure := errors.New("injected epoch resume failure")
	harness.endErrors = []error{resumeFailure}

	status, err := harness.transaction.run(context.Background(), PointerModeDigitizerStylus)
	if !errors.Is(err, ErrPointerModeSwitchResume) || !errors.Is(err, resumeFailure) {
		t.Fatalf("transaction error = %v, want resume failure", err)
	}
	if status.Mode != PointerModeUnknown || status.Switching || !status.Degraded || status.Error == "" {
		t.Fatalf("degraded status = %+v", status)
	}
	data, readErr := os.ReadFile(harness.modeFile)
	if readErr != nil || string(data) != "digitizer-stylus\n" {
		t.Fatalf("committed marker after resume failure = %q, %v", data, readErr)
	}
	harness.assertLocksReleased(t)

	status, err = harness.transaction.run(context.Background(), PointerModeDigitizerStylus)
	if err != nil {
		t.Fatalf("recover after resume failure: %v", err)
	}
	if status.Mode != PointerModeDigitizerStylus || status.Switching || status.Degraded || status.Error != "" {
		t.Fatalf("recovered status = %+v", status)
	}
	begin, end, _, _ := harness.callSnapshot()
	if begin != 1 || end != 2 {
		t.Fatalf("recovery quiesce calls begin=%d end=%d, want 1/2", begin, end)
	}
	harness.assertLocksReleased(t)
}

func TestPointerModeTransactionRollbackFailureMarksDegradedAndCanRecover(t *testing.T) {
	marker := "absolute\n"
	harness := newPointerModeTransactionHarness(t, PointerModeAbsolute, &marker, ModeNormal)
	initialFailure := errors.New("injected initial rebuild failure")
	rollbackFailure := errors.New("injected rollback rebuild failure")
	harness.scriptErrors = []error{initialFailure, rollbackFailure}

	status, err := harness.transaction.run(context.Background(), PointerModeDigitizerStylus)
	if !errors.Is(err, ErrPointerModeSwitchScript) || !errors.Is(err, ErrPointerModeSwitchRollback) ||
		!errors.Is(err, ErrPointerModeSwitchDegraded) || !errors.Is(err, rollbackFailure) {
		t.Fatalf("transaction error = %v, want degraded rollback failure", err)
	}
	if status.Mode != PointerModeUnknown || status.Switching || !status.Degraded || status.Error == "" {
		t.Fatalf("degraded status = %+v", status)
	}
	begin, end, _, _ := harness.callSnapshot()
	if begin != 1 || end != 0 {
		t.Fatalf("degraded quiesce calls begin=%d end=%d, want 1/0", begin, end)
	}
	data, readErr := os.ReadFile(harness.modeFile)
	if readErr != nil || string(data) != marker {
		t.Fatalf("degraded marker = %q, %v", data, readErr)
	}
	harness.assertLocksReleased(t)

	// A later successful switch reuses the still-quiesced epoch instead of
	// trying to quiesce an already closed router again.
	harness.scriptErrors = nil
	status, err = harness.transaction.run(context.Background(), PointerModeTouchScreen)
	if err != nil {
		t.Fatalf("recover degraded transaction: %v", err)
	}
	if status.Mode != PointerModeTouchScreen || status.Switching || status.Degraded || status.Error != "" {
		t.Fatalf("recovered status = %+v", status)
	}
	begin, end, _, _ = harness.callSnapshot()
	if begin != 1 || end != 1 {
		t.Fatalf("recovery quiesce calls begin=%d end=%d, want 1/1", begin, end)
	}
	harness.assertLocksReleased(t)
}

func TestPointerModeTransactionDegradedSwitchFailureCanRecoverOldMode(t *testing.T) {
	marker := "absolute\n"
	harness := newPointerModeTransactionHarness(t, PointerModeUnknown, &marker, ModeNormal)
	harness.manager.markDegraded(errors.New("previous rollback failed"), harness.epoch)
	forwardFailure := errors.New("new profile rebuild failed")
	harness.scriptErrors = []error{forwardFailure, nil}

	status, err := harness.transaction.run(context.Background(), PointerModeTouchScreen)
	if !errors.Is(err, ErrPointerModeSwitchScript) || !errors.Is(err, forwardFailure) {
		t.Fatalf("transaction error = %v, want forward script failure", err)
	}
	assertRolledBackPointerModeStatus(t, status, PointerModeAbsolute)
	data, readErr := os.ReadFile(harness.modeFile)
	if readErr != nil || string(data) != marker {
		t.Fatalf("recovered marker = %q, %v", data, readErr)
	}
	begin, end, scripts, verifies := harness.callSnapshot()
	if begin != 0 || end != 1 || fmt.Sprint(scripts) != fmt.Sprint([]string{"restart", "restart"}) ||
		fmt.Sprint(verifies) != fmt.Sprint([]PointerMode{PointerModeAbsolute}) {
		t.Fatalf("calls begin=%d end=%d scripts=%v verifies=%v", begin, end, scripts, verifies)
	}
	harness.assertLocksReleased(t)
}

func TestPointerModeTransactionRejectsUnknownModeWithoutMutation(t *testing.T) {
	marker := "absolute\n"
	harness := newPointerModeTransactionHarness(t, PointerModeAbsolute, &marker, ModeNormal)

	status, err := harness.transaction.run(context.Background(), PointerMode("absolute-android"))
	if !errors.Is(err, ErrPointerModeSwitchInvalidMode) {
		t.Fatalf("transaction error = %v, want invalid mode", err)
	}
	if status.Mode != PointerModeAbsolute {
		t.Fatalf("status = %+v", status)
	}
	if events := harness.recorder.snapshot(); len(events) != 0 {
		t.Fatalf("invalid transaction events = %v", events)
	}
}

func assertRolledBackPointerModeStatus(t *testing.T, status PointerModeStatus, want PointerMode) {
	t.Helper()
	if status.Mode != want || status.Switching || status.Degraded || status.Error != "" {
		t.Fatalf("rolled-back status = %+v, want mode %s and healthy", status, want)
	}
}

func TestPointerModeMarkerSnapshotAndRestore(t *testing.T) {
	modeFile := filepath.Join(t.TempDir(), "usb.hid2_mode")
	original := []byte("absolute-android\n\n")
	if err := os.WriteFile(modeFile, original, 0o600); err != nil {
		t.Fatalf("write original marker: %v", err)
	}
	snapshot, err := snapshotPointerModeMarker(modeFile)
	if err != nil {
		t.Fatalf("snapshot marker: %v", err)
	}
	if !snapshot.exists || string(snapshot.data) != string(original) || snapshot.perm != 0o600 {
		t.Fatalf("snapshot = exists:%t data:%q perm:%#o", snapshot.exists, snapshot.data, snapshot.perm)
	}
	if err := writePointerModeMarker(modeFile, PointerModeTouchScreen); err != nil {
		t.Fatalf("write canonical marker: %v", err)
	}
	data, err := os.ReadFile(modeFile)
	if err != nil || string(data) != "touch-screen\n" {
		t.Fatalf("canonical marker = %q, %v", data, err)
	}
	if err := restorePointerModeMarker(modeFile, snapshot); err != nil {
		t.Fatalf("restore marker: %v", err)
	}
	data, err = os.ReadFile(modeFile)
	if err != nil || string(data) != string(original) {
		t.Fatalf("restored marker = %q, %v", data, err)
	}
	info, err := os.Stat(modeFile)
	if err != nil {
		t.Fatalf("stat restored marker: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("restored marker permission = %#o", info.Mode().Perm())
	}
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(modeFile), ".usb.hid2_mode.tmp-*"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("temporary marker files = %v, %v", matches, err)
	}
}

func TestPointerModeMarkerMissingSnapshotRestoresAbsence(t *testing.T) {
	modeFile := filepath.Join(t.TempDir(), "usb.hid2_mode")
	snapshot, err := snapshotPointerModeMarker(modeFile)
	if err != nil || snapshot.exists {
		t.Fatalf("missing snapshot = %+v, %v", snapshot, err)
	}
	if mode := pointerModeFromMarkerSnapshot(snapshot); mode != PointerModeSingleAbsolute {
		t.Fatalf("missing snapshot mode = %q, want %q", mode, PointerModeSingleAbsolute)
	}
	if err := writePointerModeMarker(modeFile, PointerModeDigitizerStylus); err != nil {
		t.Fatalf("write marker: %v", err)
	}
	if err := restorePointerModeMarker(modeFile, snapshot); err != nil {
		t.Fatalf("restore missing marker: %v", err)
	}
	if _, err := os.Stat(modeFile); !os.IsNotExist(err) {
		t.Fatalf("restored missing marker stat = %v, want not exists", err)
	}
}

func TestPointerModeMarkerRejectsNonRegularSnapshot(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "target")
	modeFile := filepath.Join(directory, "usb.hid2_mode")
	if err := os.WriteFile(target, []byte("absolute\n"), 0o600); err != nil {
		t.Fatalf("write symlink target: %v", err)
	}
	if err := os.Symlink(target, modeFile); err != nil {
		t.Fatalf("create marker symlink: %v", err)
	}
	if _, err := snapshotPointerModeMarker(modeFile); err == nil {
		t.Fatal("snapshotPointerModeMarker accepted a symlink")
	}
}

func TestPointerModeGadgetVerifierChecksBoundProfileAndEndpoint(t *testing.T) {
	for _, mode := range []PointerMode{
		PointerModeAbsolute,
		PointerModeDigitizerStylus,
		PointerModeTouchScreen,
	} {
		t.Run(string(mode), func(t *testing.T) {
			root := writePointerModeVerifierFixture(t, mode)
			opened := ""
			verifier := newPointerModeGadgetVerifier(root, "/fake/hidg2")
			verifier.openHID2 = func(path string) error {
				opened = path
				return nil
			}
			if err := verifier.check(mode); err != nil {
				t.Fatalf("verify profile: %v", err)
			}
			if opened != "/fake/hidg2" {
				t.Fatalf("opened endpoint = %q", opened)
			}
		})
	}
}

func TestPointerModeGadgetVerifierRejectsInvalidState(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, string, *pointerModeGadgetVerifier)
		want   string
	}{
		{
			name: "unbound UDC",
			mutate: func(t *testing.T, root string, _ *pointerModeGadgetVerifier) {
				writePointerModeTestFile(t, filepath.Join(root, "UDC"), nil)
			},
			want: "not bound",
		},
		{
			name: "wrong report length",
			mutate: func(t *testing.T, root string, _ *pointerModeGadgetVerifier) {
				writePointerModeTestFile(t, filepath.Join(root, "functions", "hid.GS2", "report_length"), []byte("62\n"))
			},
			want: "report_length",
		},
		{
			name: "endpoint unavailable",
			mutate: func(_ *testing.T, _ string, verifier *pointerModeGadgetVerifier) {
				verifier.openHID2 = func(string) error { return errors.New("endpoint missing") }
			},
			want: "endpoint missing",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := writePointerModeVerifierFixture(t, PointerModeAbsolute)
			verifier := newPointerModeGadgetVerifier(root, "/fake/hidg2")
			verifier.openHID2 = func(string) error { return nil }
			test.mutate(t, root, &verifier)
			err := verifier.check(PointerModeAbsolute)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("verify error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestPointerModeGadgetVerifierWaitIsBounded(t *testing.T) {
	root := t.TempDir()
	verifier := newPointerModeGadgetVerifier(root, "/fake/hidg2")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := verifier.wait(ctx, PointerModeAbsolute)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("wait error = %v, want deadline exceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 250*time.Millisecond {
		t.Fatalf("bounded verifier took %s", elapsed)
	}
}

func TestRunPointerModeScriptTimeoutKillsChildProcessGroup(t *testing.T) {
	script := filepath.Join(t.TempDir(), "slow-usbdev.sh")
	if err := os.WriteFile(script, []byte("#!/bin/bash\nsleep 10\n"), 0o700); err != nil {
		t.Fatalf("write slow USB script: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := runPointerModeScript(ctx, script, "restart", &fakePointerModeStorageLock{})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("script error = %v, want deadline exceeded", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("timed-out script retained a child process for %s", elapsed)
	}
}

func TestRunPointerModeScriptAttachesStorageTransaction(t *testing.T) {
	script := filepath.Join(t.TempDir(), "usbdev.sh")
	if err := os.WriteFile(script, []byte("#!/bin/bash\ntest \"$NANOKVM_POINTER_MODE_TEST_LOCK\" = held && test \"$1\" = restart\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := runPointerModeScript(context.Background(), script, "restart", &fakePointerModeStorageLock{}); err != nil {
		t.Fatalf("USB rebuild lost storage transaction: %v", err)
	}
}

func TestLockUsbGadgetMutationContextHonorsDeadline(t *testing.T) {
	LockUsbGadgetMutation()
	locked := true
	defer func() {
		if locked {
			UnlockUsbGadgetMutation()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Millisecond)
	defer cancel()
	if err := LockUsbGadgetMutationContext(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("lock error = %v, want deadline exceeded", err)
	}
	UnlockUsbGadgetMutation()
	locked = false

	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := LockUsbGadgetMutationContext(ctx); err != nil {
		t.Fatalf("lock after release: %v", err)
	}
	UnlockUsbGadgetMutation()
}

func TestLockUsbGadgetMutationContextRejectsCanceledContextWithoutKeepingLock(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := LockUsbGadgetMutationContext(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("lock error = %v, want canceled", err)
	}

	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := LockUsbGadgetMutationContext(ctx); err != nil {
		t.Fatalf("lock after canceled attempt: %v", err)
	}
	UnlockUsbGadgetMutation()
}

func writePointerModeVerifierFixture(t *testing.T, mode PointerMode) string {
	t.Helper()
	root := t.TempDir()
	function := filepath.Join(root, "functions", "hid.GS2")
	if err := os.MkdirAll(function, 0o755); err != nil {
		t.Fatalf("create verifier fixture: %v", err)
	}
	reportLength, ok := pointerModeReportLength(mode)
	if !ok {
		t.Fatalf("missing report length for %s", mode)
	}
	writePointerModeTestFile(t, filepath.Join(root, "UDC"), []byte("dummy_udc.0\n"))
	config := filepath.Join(root, "configs", "c.1")
	if err := os.MkdirAll(config, 0755); err != nil {
		t.Fatal(err)
	}
	for _, link := range []string{"hid.GS1", "hid.GS2"} {
		if err := os.Symlink(filepath.Join(root, "functions", link), filepath.Join(config, link)); err != nil {
			t.Fatal(err)
		}
	}
	writePointerModeTestFile(t, filepath.Join(function, "report_length"), []byte(fmt.Sprintf("%d\n", reportLength)))
	return root
}

func writePointerModeTestFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestRestoreBothPointersRequiresAnAvailableEndpoint(t *testing.T) {
	marker := "single-absolute\n"
	h := newPointerModeTransactionHarness(t, PointerModeSingleAbsolute, &marker, ModeNormal)
	root := t.TempDir()
	h.transaction.opts.gadgetRoot = root
	config := filepath.Join(root, "configs", "c.1")
	if err := os.MkdirAll(config, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, link := range []string{"hid.GS0", "hid.GS2", "uac2.usb0", "uvc.nomad", "ncm.usb0"} {
		if err := os.Symlink("../../functions/"+link, filepath.Join(config, link)); err != nil {
			t.Fatal(err)
		}
	}
	status, err := h.transaction.run(context.Background(), PointerModeAbsolute)
	if !errors.Is(err, ErrPointerEndpointCapacity) || status.Mode != PointerModeSingleAbsolute || status.Switching {
		t.Fatalf("full USB configuration: %+v, %v", status, err)
	}
	if err := os.Remove(filepath.Join(config, "ncm.usb0")); err != nil {
		t.Fatal(err)
	}
	status, err = h.transaction.run(context.Background(), PointerModeAbsolute)
	if err != nil || status.Mode != PointerModeAbsolute {
		t.Fatalf("restore both: %+v, %v", status, err)
	}
}
