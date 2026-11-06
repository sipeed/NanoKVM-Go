package vm

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeFingerRobotBackend struct {
	mu sync.Mutex

	initCalls             int
	initErrors            []error
	pulses                []int
	failPulses            map[int]int
	pulseCh               chan int
	shutdownPinRestoreErr error
	shutdownPinRestores   int
}

type shutdownOrderFingerRobotBackend struct {
	events []string
}

func (b *shutdownOrderFingerRobotBackend) Initialize() error {
	b.events = append(b.events, "initialize")
	return nil
}

func (b *shutdownOrderFingerRobotBackend) SetPulseUS(pulseUS int) error {
	b.events = append(b.events, fmt.Sprintf("pulse:%d", pulseUS))
	return nil
}

func (b *shutdownOrderFingerRobotBackend) RestoreShutdownPins() error {
	b.events = append(b.events, "restore-pins")
	return nil
}

func (b *shutdownOrderFingerRobotBackend) Close() error {
	b.events = append(b.events, "close")
	return nil
}

func newFakeFingerRobotBackend() *fakeFingerRobotBackend {
	return &fakeFingerRobotBackend{
		failPulses: make(map[int]int),
		pulseCh:    make(chan int, 100),
	}
}

func (b *fakeFingerRobotBackend) Initialize() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.initCalls++
	if len(b.initErrors) == 0 {
		return nil
	}
	err := b.initErrors[0]
	b.initErrors = b.initErrors[1:]
	return err
}

func (b *fakeFingerRobotBackend) SetPulseUS(pulseUS int) error {
	b.mu.Lock()
	b.pulses = append(b.pulses, pulseUS)
	shouldFail := b.failPulses[pulseUS] > 0
	if shouldFail {
		b.failPulses[pulseUS]--
	}
	b.mu.Unlock()

	select {
	case b.pulseCh <- pulseUS:
	default:
	}
	if shouldFail {
		return errors.New("injected PWM write failure")
	}
	return nil
}

func (b *fakeFingerRobotBackend) RestoreShutdownPins() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.shutdownPinRestores++
	return b.shutdownPinRestoreErr
}

func (b *fakeFingerRobotBackend) setInitErrors(errs ...error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.initErrors = append([]error(nil), errs...)
}

func (b *fakeFingerRobotBackend) failNextPulse(pulseUS int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failPulses[pulseUS]++
}

func (b *fakeFingerRobotBackend) clearPulses() {
	b.mu.Lock()
	b.pulses = nil
	b.mu.Unlock()
	for {
		select {
		case <-b.pulseCh:
		default:
			return
		}
	}
}

func (b *fakeFingerRobotBackend) pulseSnapshot() []int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]int(nil), b.pulses...)
}

func (b *fakeFingerRobotBackend) initCallCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.initCalls
}

func (b *fakeFingerRobotBackend) shutdownPinRestoreCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.shutdownPinRestores
}

func (b *fakeFingerRobotBackend) setShutdownPinRestoreError(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.shutdownPinRestoreErr = err
}

type fakeFingerRobotHotplugBackend struct {
	backend *fakeFingerRobotBackend

	mu            sync.Mutex
	connected     bool
	hardwareReady bool
	forceCalls    int
	nextPulseErrs []error
	forceComplete chan struct{}
}

func newFakeFingerRobotHotplugBackend() *fakeFingerRobotHotplugBackend {
	return &fakeFingerRobotHotplugBackend{backend: newFakeFingerRobotBackend()}
}

func (b *fakeFingerRobotHotplugBackend) RestoreShutdownPins() error {
	return b.backend.RestoreShutdownPins()
}

func (b *fakeFingerRobotHotplugBackend) Initialize() error {
	err := b.backend.Initialize()
	b.mu.Lock()
	b.hardwareReady = err == nil
	b.mu.Unlock()
	return err
}

func (b *fakeFingerRobotHotplugBackend) SetPulseUS(pulseUS int) error {
	b.mu.Lock()
	var err error
	if len(b.nextPulseErrs) > 0 {
		err = b.nextPulseErrs[0]
		b.nextPulseErrs = b.nextPulseErrs[1:]
	}
	b.mu.Unlock()
	if err != nil {
		return err
	}
	return b.backend.SetPulseUS(pulseUS)
}

func (b *fakeFingerRobotHotplugBackend) SetPulseUSUngated(pulseUS int) error {
	return b.backend.SetPulseUS(pulseUS)
}

func (b *fakeFingerRobotHotplugBackend) HardwareReady() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.hardwareReady
}

func (b *fakeFingerRobotHotplugBackend) WaitConnected(time.Duration) error {
	if !b.ConnectionAlive(time.Second) {
		return ErrFingerRobotDisconnected
	}
	return nil
}

func (b *fakeFingerRobotHotplugBackend) ConnectionAlive(time.Duration) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.connected && b.hardwareReady
}

func (b *fakeFingerRobotHotplugBackend) ForceSafeRelease() error {
	b.mu.Lock()
	b.forceCalls++
	forceComplete := b.forceComplete
	b.mu.Unlock()
	err := b.backend.SetPulseUS(FingerRobotSafeReleaseUS)
	if forceComplete != nil {
		select {
		case forceComplete <- struct{}{}:
		default:
		}
	}
	return err
}

func (b *fakeFingerRobotHotplugBackend) setConnected(connected bool) {
	b.mu.Lock()
	b.connected = connected
	b.mu.Unlock()
}

func (b *fakeFingerRobotHotplugBackend) failNextSetPulse(err error) {
	b.mu.Lock()
	b.nextPulseErrs = append(b.nextPulseErrs, err)
	b.mu.Unlock()
}

func (b *fakeFingerRobotHotplugBackend) notifyForceCompletion(ch chan struct{}) {
	b.mu.Lock()
	b.forceComplete = ch
	b.mu.Unlock()
}

func (b *fakeFingerRobotHotplugBackend) forceCallCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.forceCalls
}

func TestValidateFingerRobotDebugRequest(t *testing.T) {
	if limits := GetFingerRobotLimits(); limits.LeaseTimeoutMS != FingerRobotDebugLeaseTimeoutMS ||
		limits.ZeroUS != FingerRobotFactoryZeroUS ||
		limits.ReleaseMaxUS != FingerRobotSafeReleaseUS || limits.ReleaseDefaultUS != FingerRobotSafeReleaseUS ||
		limits.PressMaxUS != 2000 || limits.PressDefaultUS != 899 ||
		limits.CountMin != 1 || limits.CountUnlimitedValue != 0 ||
		limits.PressSpeedMinUSPerSecond != 200 || limits.PressSpeedMaxUSPerSecond != 2000 ||
		limits.PressSpeedDefaultUSPerSecond != 2000 || limits.CountDefault != 50 ||
		limits.PressDefaultMS != 100 {
		t.Fatalf("unexpected debug lease timeout: %+v", limits)
	}

	valid := FingerRobotDebugRequest{
		ReleaseUS:             FingerRobotSafeReleaseUS,
		PressUS:               899,
		PressSpeedUSPerSecond: 2000,
		Count:                 50,
		PressMS:               100,
	}
	tests := []struct {
		name    string
		mutate  func(*FingerRobotDebugRequest)
		wantErr bool
	}{
		{name: "defaults"},
		{name: "minimums", mutate: func(req *FingerRobotDebugRequest) {
			req.ReleaseUS = 890
			req.PressUS = 889
			req.PressSpeedUSPerSecond = 200
			req.Count = 1
			req.PressMS = 100
		}},
		{name: "maximum pulses", mutate: func(req *FingerRobotDebugRequest) {
			req.ReleaseUS = FingerRobotSafeReleaseUS
			req.PressUS = 2000
			req.PressSpeedUSPerSecond = 2000
		}},
		{name: "maximum JavaScript safe integer count and maximum duration", mutate: func(req *FingerRobotDebugRequest) {
			req.Count = 9_007_199_254_740_991
			req.PressMS = 30000
		}},
		{name: "unlimited count", mutate: func(req *FingerRobotDebugRequest) { req.Count = 0 }},
		{name: "release below range", mutate: func(req *FingerRobotDebugRequest) { req.ReleaseUS = 888 }, wantErr: true},
		{name: "release above range", mutate: func(req *FingerRobotDebugRequest) { req.ReleaseUS = FingerRobotSafeReleaseUS + 1 }, wantErr: true},
		{name: "press below range", mutate: func(req *FingerRobotDebugRequest) { req.PressUS = 888 }, wantErr: true},
		{name: "press above range", mutate: func(req *FingerRobotDebugRequest) { req.PressUS = 2001 }, wantErr: true},
		{name: "wrong direction", mutate: func(req *FingerRobotDebugRequest) { req.PressUS = req.ReleaseUS }, wantErr: true},
		{name: "press speed below range", mutate: func(req *FingerRobotDebugRequest) { req.PressSpeedUSPerSecond = 199 }, wantErr: true},
		{name: "press speed above range", mutate: func(req *FingerRobotDebugRequest) { req.PressSpeedUSPerSecond = 2001 }, wantErr: true},
		{name: "negative count", mutate: func(req *FingerRobotDebugRequest) { req.Count = -1 }, wantErr: true},
		{name: "press duration below range", mutate: func(req *FingerRobotDebugRequest) { req.PressMS = 99 }, wantErr: true},
		{name: "press duration above range", mutate: func(req *FingerRobotDebugRequest) { req.PressMS = 30001 }, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := valid
			if test.mutate != nil {
				test.mutate(&req)
			}
			err := validateFingerRobotDebugRequest(req)
			if test.wantErr && !errors.Is(err, ErrFingerRobotInvalid) {
				t.Fatalf("expected invalid parameters error, got %v", err)
			}
			if !test.wantErr && err != nil {
				t.Fatalf("expected valid request, got %v", err)
			}
		})
	}
}

func TestResolveFingerRobotPressRequest(t *testing.T) {
	tests := []struct {
		name      string
		req       FingerRobotPressRequest
		wantPress int
		wantSpeed int
		wantErr   bool
	}{
		{
			name: "long arm light slow",
			req: FingerRobotPressRequest{
				ProfileID: FingerRobotArmProfileLongID,
				Strength:  1,
				PressMS:   100,
				Speed:     FingerRobotSpeedSlow,
			},
			wantPress: 2000,
			wantSpeed: FingerRobotSlowSpeedUSPerSecond,
		},
		{
			name: "short arm strong fast",
			req: FingerRobotPressRequest{
				ProfileID: FingerRobotArmProfileShortID,
				Strength:  5,
				PressMS:   30000,
				Speed:     FingerRobotSpeedFast,
			},
			wantPress: 1350,
			wantSpeed: FingerRobotFastSpeedUSPerSecond,
		},
		{name: "unknown profile", req: FingerRobotPressRequest{ProfileID: "unknown", Strength: 1, PressMS: 100, Speed: FingerRobotSpeedSlow}, wantErr: true},
		{name: "strength below range", req: FingerRobotPressRequest{ProfileID: FingerRobotArmProfileLongID, Strength: 0, PressMS: 100, Speed: FingerRobotSpeedSlow}, wantErr: true},
		{name: "strength above range", req: FingerRobotPressRequest{ProfileID: FingerRobotArmProfileLongID, Strength: 6, PressMS: 100, Speed: FingerRobotSpeedSlow}, wantErr: true},
		{name: "duration below range", req: FingerRobotPressRequest{ProfileID: FingerRobotArmProfileLongID, Strength: 1, PressMS: 99, Speed: FingerRobotSpeedSlow}, wantErr: true},
		{name: "duration above range", req: FingerRobotPressRequest{ProfileID: FingerRobotArmProfileLongID, Strength: 1, PressMS: 30001, Speed: FingerRobotSpeedSlow}, wantErr: true},
		{name: "unknown speed", req: FingerRobotPressRequest{ProfileID: FingerRobotArmProfileLongID, Strength: 1, PressMS: 100, Speed: "medium"}, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pressUS, speedUSPerSecond, err := resolveFingerRobotPressRequest(test.req)
			if test.wantErr {
				if !errors.Is(err, ErrFingerRobotInvalid) {
					t.Fatalf("expected invalid parameters error, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolve request: %v", err)
			}
			if pressUS != test.wantPress || speedUSPerSecond != test.wantSpeed {
				t.Fatalf("resolved request = (%d, %d), want (%d, %d)", pressUS, speedUSPerSecond, test.wantPress, test.wantSpeed)
			}
		})
	}
}

func TestFingerRobotRampMath(t *testing.T) {
	tests := []struct {
		name      string
		fromUS    int
		toUS      int
		speed     int
		wantSteps int
	}{
		{name: "press default travel", fromUS: FingerRobotSafeReleaseUS, toUS: 1900, speed: 1000, wantSteps: 8},
		{name: "release default travel", fromUS: 1900, toUS: FingerRobotSafeReleaseUS, speed: 1000, wantSteps: 8},
		{name: "press maximum travel at minimum speed", fromUS: FingerRobotSafeReleaseUS, toUS: 889, speed: 200, wantSteps: 291},
		{name: "release maximum travel at minimum speed", fromUS: 889, toUS: FingerRobotSafeReleaseUS, speed: 200, wantSteps: 291},
		{name: "single pulse unit", fromUS: 890, toUS: 889, speed: 2000, wantSteps: 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			steps := fingerRobotRampSteps(test.fromUS, test.toUS, test.speed)
			if steps != test.wantSteps {
				t.Fatalf("unexpected ramp step count: got %d want %d", steps, test.wantSteps)
			}

			direction := 1
			if test.toUS < test.fromUS {
				direction = -1
			}
			lastPulseUS := test.fromUS
			maxStepUS := (test.speed*int(fingerRobotRampInterval/time.Millisecond) + 999) / 1000
			for step := 1; step <= steps; step++ {
				pulseUS := fingerRobotRampPulse(test.fromUS, test.toUS, step, steps)
				progressUS := (pulseUS - lastPulseUS) * direction
				if progressUS <= 0 || (pulseUS-test.toUS)*direction > 0 {
					t.Fatalf("ramp is not strictly monotonic at step %d: last=%d pulse=%d", step, lastPulseUS, pulseUS)
				}
				if progressUS > maxStepUS {
					t.Fatalf("ramp exceeded speed bound at step %d: delta=%d max=%d", step, progressUS, maxStepUS)
				}
				lastPulseUS = pulseUS
			}
			if lastPulseUS != test.toUS {
				t.Fatalf("ramp did not end at the configured target: got %d want %d", lastPulseUS, test.toUS)
			}
		})
	}
}

func TestFingerRobotReleaseRampReversesPressRamp(t *testing.T) {
	const pressUS = 1901

	pressPulses := append(
		[]int{FingerRobotSafeReleaseUS},
		testFingerRobotRampPulses(FingerRobotSafeReleaseUS, pressUS, FingerRobotFastSpeedUSPerSecond)...,
	)
	wantReleasePulses := make([]int, len(pressPulses)-1)
	for index := range wantReleasePulses {
		wantReleasePulses[index] = pressPulses[len(pressPulses)-2-index]
	}

	gotReleasePulses := testFingerRobotRampPulses(pressUS, FingerRobotSafeReleaseUS, FingerRobotFastSpeedUSPerSecond)
	if !reflect.DeepEqual(gotReleasePulses, wantReleasePulses) {
		t.Fatalf("release ramp = %v, want reversed press ramp %v", gotReleasePulses, wantReleasePulses)
	}
}

func TestNormalPressUsesProfileRampHoldAndRampedSafeRelease(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	initializeControllerForTest(t, controller, backend)

	err := controller.Press(testFingerRobotPressRequest(100))
	if err != nil {
		t.Fatalf("Press: %v", err)
	}

	want := testFingerRobotCompletedPulses()
	if got := backend.pulseSnapshot(); !reflect.DeepEqual(got, want) {
		t.Fatalf("normal press pulse sequence = %v, want %v", got, want)
	}
	status := controller.Status()
	if status.Running || status.State != FingerRobotStateIdle || status.Result != FingerRobotResultSuccess || status.Error != "" {
		t.Fatalf("unexpected completed normal status: %+v", status)
	}
}

func TestReleasePreemptsNormalPressRampAndWinsFinalPulse(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	initializeControllerForTest(t, controller, backend)

	pressDone := make(chan error, 1)
	go func() {
		pressDone <- controller.Press(FingerRobotPressRequest{
			ProfileID: FingerRobotArmProfileShortID,
			Strength:  5,
			PressMS:   30000,
			Speed:     FingerRobotSpeedSlow,
		})
	}()
	waitForPulses(
		t,
		backend,
		FingerRobotSafeReleaseUS,
		testFingerRobotFirstRampPulse(FingerRobotSafeReleaseUS, 1350, FingerRobotSlowSpeedUSPerSecond),
	)

	if err := controller.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if err := <-pressDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("preempted press returned %v, want context cancellation", err)
	}
	pulses := backend.pulseSnapshot()
	if got := pulses[len(pulses)-1]; got != FingerRobotSafeReleaseUS {
		t.Fatalf("Release did not own the final safe pulse: %v", pulses)
	}
	time.Sleep(50 * time.Millisecond)
	if got := backend.pulseSnapshot(); !reflect.DeepEqual(got, pulses) {
		t.Fatalf("normal ramp continued after Release: before=%v after=%v", pulses, got)
	}
	status := controller.Status()
	if status.Running || status.State != FingerRobotStateIdle || status.Result != FingerRobotResultSuccess || status.Error != "" {
		t.Fatalf("unexpected release status: %+v", status)
	}
}

func TestReleasePreemptsNormalHoldAndSharesBusyState(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	initializeControllerForTest(t, controller, backend)

	pressDone := make(chan error, 1)
	go func() {
		pressDone <- controller.Press(testFingerRobotPressRequest(30000))
	}()
	waitForPulses(t, backend, testFingerRobotPressPulsesBeforeRelease()...)
	waitForFingerRobotState(t, controller, FingerRobotStateHolding)

	debugReq := FingerRobotDebugRequest{
		ReleaseUS:             1400,
		PressUS:               1300,
		PressSpeedUSPerSecond: 1000,
		Count:                 1,
		PressMS:               100,
	}
	if err := controller.StartDebug(debugReq); !errors.Is(err, ErrFingerRobotBusy) {
		t.Fatalf("Debug operation did not share normal busy state: %v", err)
	}
	if err := controller.Press(testFingerRobotPressRequest(100)); !errors.Is(err, ErrFingerRobotBusy) {
		t.Fatalf("second normal operation did not share busy state: %v", err)
	}

	if err := controller.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if err := <-pressDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("preempted hold returned %v, want context cancellation", err)
	}
	if got := backend.pulseSnapshot(); got[len(got)-1] != FingerRobotSafeReleaseUS {
		t.Fatalf("Release did not end hold at safe point: %v", got)
	}
}

func TestReleasePreemptsNormalReleaseRampAndWinsFinalPulse(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	initializeControllerForTest(t, controller, backend)

	pressDone := make(chan error, 1)
	go func() {
		pressDone <- controller.Press(FingerRobotPressRequest{
			ProfileID: FingerRobotArmProfileShortID,
			Strength:  5,
			PressMS:   100,
			Speed:     FingerRobotSpeedFast,
		})
	}()
	pressPulses := append(
		[]int{FingerRobotSafeReleaseUS},
		testFingerRobotRampPulses(FingerRobotSafeReleaseUS, 1350, FingerRobotFastSpeedUSPerSecond)...,
	)
	waitForPulses(t, backend, pressPulses...)
	waitForPulses(t, backend, testFingerRobotFirstRampPulse(1350, FingerRobotSafeReleaseUS, FingerRobotFastSpeedUSPerSecond))

	if err := controller.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if err := <-pressDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("preempted release ramp returned %v, want context cancellation", err)
	}
	pulses := backend.pulseSnapshot()
	if got := pulses[len(pulses)-1]; got != FingerRobotSafeReleaseUS {
		t.Fatalf("Release did not own the final safe pulse: %v", pulses)
	}
	time.Sleep(2 * fingerRobotRampInterval)
	if got := backend.pulseSnapshot(); !reflect.DeepEqual(got, pulses) {
		t.Fatalf("normal release ramp continued after Release: before=%v after=%v", pulses, got)
	}
}

func TestReleasePreemptsDebugAndOverridesSnapshotRelease(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	initializeControllerForTest(t, controller, backend)

	if err := controller.StartDebug(FingerRobotDebugRequest{
		ReleaseUS:             1400,
		PressUS:               1300,
		PressSpeedUSPerSecond: 200,
		Count:                 0,
		PressMS:               30000,
	}); err != nil {
		t.Fatalf("StartDebug: %v", err)
	}
	waitForPulses(t, backend, 1400, testFingerRobotFirstRampPulse(1400, 1300, 200))

	if err := controller.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}
	pulses := backend.pulseSnapshot()
	if got := pulses[len(pulses)-1]; got != FingerRobotSafeReleaseUS {
		t.Fatalf("normal Release restored Debug snapshot instead of absolute safe point: %v", pulses)
	}
	time.Sleep(50 * time.Millisecond)
	if got := backend.pulseSnapshot(); !reflect.DeepEqual(got, pulses) {
		t.Fatalf("Debug operation continued after Release: before=%v after=%v", pulses, got)
	}
	status := controller.Status()
	if status.Running || status.debug || status.State != FingerRobotStateIdle || status.Result != FingerRobotResultSuccess {
		t.Fatalf("unexpected status after releasing Debug operation: %+v", status)
	}
}

func TestReleaseNeverInitializesDisabledControllerAndIsIdleIdempotent(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)

	if err := controller.Release(); err != nil {
		t.Fatalf("never-initialized Release must be an idempotent success: %v", err)
	}
	if got := backend.initCallCount(); got != 0 {
		t.Fatalf("Release initialized disabled hardware: init calls=%d", got)
	}
	if pulses := backend.pulseSnapshot(); len(pulses) != 0 {
		t.Fatalf("Release touched never-initialized PWM: %v", pulses)
	}

	initializeControllerForTest(t, controller, backend)
	if err := controller.Release(); err != nil {
		t.Fatalf("first idle Release: %v", err)
	}
	if err := controller.Release(); err != nil {
		t.Fatalf("second idle Release: %v", err)
	}
	if got := backend.pulseSnapshot(); !reflect.DeepEqual(got, []int{FingerRobotSafeReleaseUS, FingerRobotSafeReleaseUS}) {
		t.Fatalf("idle Release was not safely idempotent: %v", got)
	}
	if got := backend.initCallCount(); got != 1 {
		t.Fatalf("idle Release reinitialized hardware: init calls=%d", got)
	}
}

func TestConcurrentReleaseNeverReturnsBusy(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	initializeControllerForTest(t, controller, backend)

	const calls = 8
	start := make(chan struct{})
	errs := make(chan error, calls)
	var wg sync.WaitGroup
	for index := 0; index < calls; index++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs <- controller.Release()
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent Release returned %v", err)
		}
	}
	pulses := backend.pulseSnapshot()
	if len(pulses) != calls {
		t.Fatalf("concurrent Releases wrote %d pulses, want %d: %v", len(pulses), calls, pulses)
	}
	for _, pulseUS := range pulses {
		if pulseUS != FingerRobotSafeReleaseUS {
			t.Fatalf("concurrent Release wrote unsafe pulse: %v", pulses)
		}
	}
}

func TestReleaseFailureMarksControllerUnavailable(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	initializeControllerForTest(t, controller, backend)
	backend.failNextPulse(FingerRobotSafeReleaseUS)

	if err := controller.Release(); err == nil {
		t.Fatal("expected injected Release failure")
	}
	if controller.IsAvailable() {
		t.Fatal("failed Release left controller available")
	}
	status := controller.Status()
	if status.Running || status.State != FingerRobotStateReleasing || status.Result != FingerRobotResultFailed || status.Error == "" {
		t.Fatalf("unexpected failed Release status: %+v", status)
	}
}

func TestInvalidDebugRequestDoesNotTouchHardware(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	if err := controller.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	backend.clearPulses()

	err := controller.StartDebug(FingerRobotDebugRequest{
		ReleaseUS:             1900,
		PressUS:               1900,
		PressSpeedUSPerSecond: 1000,
		Count:                 1,
		PressMS:               800,
	})
	if !errors.Is(err, ErrFingerRobotInvalid) {
		t.Fatalf("expected invalid parameters error, got %v", err)
	}
	if pulses := backend.pulseSnapshot(); len(pulses) != 0 {
		t.Fatalf("invalid request touched PWM: %v", pulses)
	}
}

func TestFingerRobotDebugSequenceEndsAtConfiguredReleasePoint(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	initializeControllerForTest(t, controller, backend)

	req := FingerRobotDebugRequest{
		ReleaseUS:             1060,
		PressUS:               1000,
		PressSpeedUSPerSecond: 1000,
		Count:                 2,
		PressMS:               100,
	}
	if err := controller.StartDebug(req); err != nil {
		t.Fatalf("StartDebug: %v", err)
	}
	status := waitForFingerRobotResult(t, controller)
	if status.Result != FingerRobotResultSuccess {
		t.Fatalf("expected success, got %+v", status)
	}
	if status.Current != 2 || status.Total != 2 || status.State != FingerRobotStateReleasing {
		t.Fatalf("unexpected final status: %+v", status)
	}

	want := []int{req.ReleaseUS}
	for count := int64(0); count < req.Count; count++ {
		want = append(want, testFingerRobotRampPulses(req.ReleaseUS, req.PressUS, req.PressSpeedUSPerSecond)...)
		want = append(want, testFingerRobotRampPulses(req.PressUS, req.ReleaseUS, req.PressSpeedUSPerSecond)...)
	}
	if got := backend.pulseSnapshot(); !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected PWM sequence: got %v want %v", got, want)
	}
}

func TestFingerRobotDebugPressDurationStartsAfterRamp(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	initializeControllerForTest(t, controller, backend)

	if err := controller.StartDebug(FingerRobotDebugRequest{
		ReleaseUS:             920,
		PressUS:               900,
		PressSpeedUSPerSecond: 1000,
		Count:                 1,
		PressMS:               100,
	}); err != nil {
		t.Fatalf("StartDebug: %v", err)
	}
	waitForPulses(t, backend, 920)

	select {
	case pulseUS := <-backend.pulseCh:
		if pulseUS != 900 {
			t.Fatalf("unexpected ramp target: got %d want 900", pulseUS)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the configured press point")
	}
	pressedAt := time.Now()

	select {
	case pulseUS := <-backend.pulseCh:
		if pulseUS != 920 {
			t.Fatalf("unexpected release pulse: got %d want 920", pulseUS)
		}
		if heldFor := time.Since(pressedAt); heldFor < 80*time.Millisecond {
			t.Fatalf("press duration started before the ramp reached its target: held for only %s", heldFor)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for release after press hold")
	}

	if status := waitForFingerRobotResult(t, controller); status.Result != FingerRobotResultSuccess {
		t.Fatalf("unexpected final status: %+v", status)
	}
}

func TestStopDebugCancelsUnlimitedTestAndRestoresSnapshotRelease(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	initializeControllerForTest(t, controller, backend)

	if err := controller.StartDebug(FingerRobotDebugRequest{
		ReleaseUS:             920,
		PressUS:               900,
		PressSpeedUSPerSecond: 1000,
		Count:                 0,
		PressMS:               100,
	}); err != nil {
		t.Fatalf("StartDebug: %v", err)
	}
	waitForPulses(t, backend, 920, 900, 920)

	if err := controller.StopDebug(); err != nil {
		t.Fatalf("StopDebug: %v", err)
	}
	pulses := backend.pulseSnapshot()
	if got := pulses[len(pulses)-1]; got != 920 {
		t.Fatalf("stop did not restore the request release point: got %d sequence=%v", got, pulses)
	}
	for index, pulseUS := range pulses {
		if pulseUS == FingerRobotFactoryZeroUS {
			t.Fatalf("stop wrote the factory zero at index %d: %v", index, pulses)
		}
	}
	status := controller.Status()
	if status.Running || status.Total != 0 || status.Current != 1 ||
		status.State != FingerRobotStateIdle || status.Result != FingerRobotResultCancelled || status.Error != "" {
		t.Fatalf("unexpected stopped unlimited status: %+v", status)
	}
}

func TestStopDebugPreemptsReleaseRampAndWinsFinalPulse(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	initializeControllerForTest(t, controller, backend)
	req := FingerRobotDebugRequest{
		ReleaseUS:             1400,
		PressUS:               900,
		PressSpeedUSPerSecond: 1000,
		Count:                 0,
		PressMS:               100,
	}

	if err := controller.StartDebug(req); err != nil {
		t.Fatalf("StartDebug: %v", err)
	}
	pressPulses := append(
		[]int{req.ReleaseUS},
		testFingerRobotRampPulses(req.ReleaseUS, req.PressUS, req.PressSpeedUSPerSecond)...,
	)
	waitForPulses(t, backend, pressPulses...)
	waitForPulses(t, backend, testFingerRobotFirstRampPulse(req.PressUS, req.ReleaseUS, req.PressSpeedUSPerSecond))

	if err := controller.StopDebug(); err != nil {
		t.Fatalf("StopDebug: %v", err)
	}
	pulses := backend.pulseSnapshot()
	if got := pulses[len(pulses)-1]; got != req.ReleaseUS {
		t.Fatalf("StopDebug did not own the final configured release pulse: %v", pulses)
	}
	time.Sleep(2 * fingerRobotRampInterval)
	if got := backend.pulseSnapshot(); !reflect.DeepEqual(got, pulses) {
		t.Fatalf("debug release ramp continued after StopDebug: before=%v after=%v", pulses, got)
	}
	status := controller.Status()
	if status.Running || status.State != FingerRobotStateIdle || status.Result != FingerRobotResultCancelled || status.Error != "" {
		t.Fatalf("unexpected stopped release ramp status: %+v", status)
	}
}

func TestStopDebugIsIdleIdempotentAndDoesNotCancelNormalPress(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	initializeControllerForTest(t, controller, backend)

	if err := controller.StopDebug(); err != nil {
		t.Fatalf("idle StopDebug: %v", err)
	}
	if pulses := backend.pulseSnapshot(); len(pulses) != 0 {
		t.Fatalf("idle stop unexpectedly touched PWM: %v", pulses)
	}

	pressDone := make(chan error, 1)
	go func() {
		pressDone <- controller.Press(testFingerRobotPressRequest(30000))
	}()
	waitForTestFingerRobotHolding(t, controller, backend)
	if err := controller.StopDebug(); !errors.Is(err, ErrFingerRobotBusy) {
		t.Fatalf("stop must not cancel a normal press operation, got %v", err)
	}
	if status := controller.Status(); !status.Running || status.debug {
		t.Fatalf("normal press operation stopped unexpectedly: %+v", status)
	}
	if err := controller.Reset(); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if err := <-pressDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cleanup reset to cancel normal press operation, got %v", err)
	}
}

func TestStopDebugReleaseFailureMarksControllerUnavailable(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	initializeControllerForTest(t, controller, backend)

	if err := controller.StartDebug(FingerRobotDebugRequest{
		ReleaseUS:             920,
		PressUS:               900,
		PressSpeedUSPerSecond: 1000,
		Count:                 0,
		PressMS:               30000,
	}); err != nil {
		t.Fatalf("StartDebug: %v", err)
	}
	waitForPulses(t, backend, 920, 900)
	backend.failNextPulse(920)

	if err := controller.StopDebug(); err == nil {
		t.Fatal("expected release write failure")
	}
	if controller.IsAvailable() {
		t.Fatal("controller must become unavailable after the stop release write fails")
	}
	status := controller.Status()
	if status.Running || status.State != FingerRobotStateIdle || status.Result != FingerRobotResultFailed || status.Error == "" {
		t.Fatalf("unexpected failed stop status: %+v", status)
	}
}

func TestResetCancelsReleaseGapWithoutAnotherPress(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	initializeControllerForTest(t, controller, backend)

	if err := controller.StartDebug(FingerRobotDebugRequest{
		ReleaseUS:             920,
		PressUS:               900,
		PressSpeedUSPerSecond: 1000,
		Count:                 2,
		PressMS:               100,
	}); err != nil {
		t.Fatalf("StartDebug: %v", err)
	}
	waitForPulses(t, backend, 920, 900, 920)

	if err := controller.Reset(); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	pulses := backend.pulseSnapshot()
	if len(pulses) < 4 {
		t.Fatalf("expected reset pulse after cancellation, got %v", pulses)
	}
	for index, pulse := range pulses[3:] {
		if pulse != FingerRobotFactoryZeroUS {
			t.Fatalf("unexpected pulse after reset at index %d: %d, full sequence %v", index+3, pulse, pulses)
		}
	}
	status := controller.Status()
	if status.Running || status.Result != FingerRobotResultReset || status.State != FingerRobotStateIdle {
		t.Fatalf("unexpected reset status: %+v", status)
	}
}

func TestResetCancelsDebugPressRampWithoutLaterSteps(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	initializeControllerForTest(t, controller, backend)

	if err := controller.StartDebug(FingerRobotDebugRequest{
		ReleaseUS:             1400,
		PressUS:               900,
		PressSpeedUSPerSecond: 200,
		Count:                 1,
		PressMS:               30000,
	}); err != nil {
		t.Fatalf("StartDebug: %v", err)
	}
	waitForPulses(t, backend, 1400, testFingerRobotFirstRampPulse(1400, 900, 200))

	if err := controller.Reset(); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	pulses := backend.pulseSnapshot()
	if len(pulses) < 3 {
		t.Fatalf("expected ramp and fixed-zero reset pulses, got %v", pulses)
	}
	for index, pulseUS := range pulses[2:] {
		if pulseUS != FingerRobotFactoryZeroUS {
			t.Fatalf("ramp continued after reset at index %d: pulse=%d sequence=%v", index+2, pulseUS, pulses)
		}
	}
}

func TestDebugAndNormalPressOperationsShareBusyState(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	initializeControllerForTest(t, controller, backend)

	req := FingerRobotDebugRequest{ReleaseUS: 1400, PressUS: 900, PressSpeedUSPerSecond: 1000, Count: 1, PressMS: 30000}
	if err := controller.StartDebug(req); err != nil {
		t.Fatalf("StartDebug: %v", err)
	}
	waitForPulses(t, backend, 1400)
	if status := controller.Status(); !status.Running || !status.debug {
		t.Fatalf("active debug operation was not marked as debug: %+v", status)
	}

	if err := controller.Press(testFingerRobotPressRequest(800)); !errors.Is(err, ErrFingerRobotBusy) {
		t.Fatalf("expected normal press busy error, got %v", err)
	}
	if err := controller.StartDebug(req); !errors.Is(err, ErrFingerRobotBusy) {
		t.Fatalf("expected debug busy error, got %v", err)
	}
	if err := controller.Reset(); err != nil {
		t.Fatalf("Reset: %v", err)
	}
}

func TestResetCancelsSynchronousNormalPressOperation(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	initializeControllerForTest(t, controller, backend)

	pressDone := make(chan error, 1)
	go func() {
		pressDone <- controller.Press(testFingerRobotPressRequest(30000))
	}()
	waitForTestFingerRobotHolding(t, controller, backend)

	if err := controller.Reset(); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	select {
	case err := <-pressDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected cancelled normal press operation, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("reset did not unblock synchronous normal press operation")
	}
	if pulses := backend.pulseSnapshot(); pulses[len(pulses)-1] != FingerRobotFactoryZeroUS {
		t.Fatalf("reset did not leave normal press operation at fixed zero: %v", pulses)
	}
}

func TestNormalPressRetriesTransientInitializationFailure(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	backend.setInitErrors(errors.New("transient init failure"), nil)
	controller := newFingerRobotController(backend)
	if err := controller.Initialize(); err == nil {
		t.Fatal("expected initial initialization failure")
	}
	if controller.IsAvailable() {
		t.Fatal("controller must be unavailable after failed initialization")
	}
	if got := backend.pulseSnapshot(); !reflect.DeepEqual(got, []int{FingerRobotSafeReleaseUS}) {
		t.Fatalf("partial initialization did not attempt the safe release point: %v", got)
	}
	backend.clearPulses()

	if err := controller.Press(testFingerRobotPressRequest(100)); err != nil {
		t.Fatalf("normal Press should recover after transient initialization failure: %v", err)
	}
	want := append([]int{FingerRobotSafeReleaseUS}, testFingerRobotCompletedPulses()...)
	if got := backend.pulseSnapshot(); !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected recovered normal press sequence: got %v want %v", got, want)
	}
}

func TestInitializeDoesNotRaceAnActiveUnavailableOperation(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	opCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	controller.mu.Lock()
	controller.active = &fingerRobotOperation{
		ctx:    opCtx,
		cancel: cancel,
		done:   make(chan struct{}),
	}
	controller.available = false
	controller.mu.Unlock()

	if err := controller.Initialize(); !errors.Is(err, ErrFingerRobotBusy) {
		t.Fatalf("expected busy while an unavailable operation is still active, got %v", err)
	}
	initCalls := backend.initCallCount()
	if initCalls != 0 {
		t.Fatalf("active operation raced hardware initialization: init calls=%d", initCalls)
	}
}

func TestResetRetriesInitializationWhenUnavailable(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	backend.setInitErrors(errors.New("transient init failure"), nil)
	controller := newFingerRobotController(backend)
	if err := controller.Initialize(); err == nil {
		t.Fatal("expected initial initialization failure")
	}
	backend.clearPulses()

	if err := controller.Reset(); err != nil {
		t.Fatalf("Reset should retry initialization: %v", err)
	}
	if !controller.IsAvailable() {
		t.Fatal("controller should become available after successful reset retry")
	}
	if got := backend.pulseSnapshot(); !reflect.DeepEqual(got, []int{FingerRobotFactoryZeroUS}) {
		t.Fatalf("unexpected reset sequence: %v", got)
	}
}

func TestDebugWriteFailureRestoresDefaultReleasePoint(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	initializeControllerForTest(t, controller, backend)
	backend.failNextPulse(900)

	if err := controller.StartDebug(FingerRobotDebugRequest{
		ReleaseUS:             1400,
		PressUS:               900,
		PressSpeedUSPerSecond: 2000,
		Count:                 1,
		PressMS:               100,
	}); err != nil {
		t.Fatalf("StartDebug: %v", err)
	}
	status := waitForFingerRobotResult(t, controller)
	if status.Result != FingerRobotResultFailed || status.Error == "" {
		t.Fatalf("expected failed status, got %+v", status)
	}
	if pulses := backend.pulseSnapshot(); pulses[len(pulses)-1] != fingerRobotLimits.ReleaseDefaultUS {
		t.Fatalf("write failure did not restore the default release point: %v", pulses)
	}
}

func TestDebugReleaseRampWriteFailureRestoresDefaultReleasePointImmediately(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	initializeControllerForTest(t, controller, backend)
	req := FingerRobotDebugRequest{
		ReleaseUS:             1060,
		PressUS:               1000,
		PressSpeedUSPerSecond: 1000,
		Count:                 1,
		PressMS:               500,
	}

	if err := controller.StartDebug(req); err != nil {
		t.Fatalf("StartDebug: %v", err)
	}
	pressPulses := append(
		[]int{req.ReleaseUS},
		testFingerRobotRampPulses(req.ReleaseUS, req.PressUS, req.PressSpeedUSPerSecond)...,
	)
	waitForPulses(t, backend, pressPulses...)
	releasePulses := testFingerRobotRampPulses(req.PressUS, req.ReleaseUS, req.PressSpeedUSPerSecond)
	backend.failNextPulse(releasePulses[0])

	status := waitForFingerRobotResult(t, controller)
	if status.Result != FingerRobotResultFailed || status.Error == "" {
		t.Fatalf("expected failed release ramp status, got %+v", status)
	}
	want := append(pressPulses, releasePulses[0], FingerRobotSafeReleaseUS)
	if got := backend.pulseSnapshot(); !reflect.DeepEqual(got, want) {
		t.Fatalf("release ramp failure did not immediately restore the safe point: got %v want %v", got, want)
	}
	if !controller.IsAvailable() {
		t.Fatal("successful safety recovery after release ramp failure must keep the controller available")
	}
}

func TestFailedSafetyResetMarksUnavailableAndNextNormalPressReinitializes(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	initializeControllerForTest(t, controller, backend)
	backend.failNextPulse(900)
	backend.failNextPulse(fingerRobotLimits.ReleaseDefaultUS)

	if err := controller.StartDebug(FingerRobotDebugRequest{
		ReleaseUS:             1400,
		PressUS:               900,
		PressSpeedUSPerSecond: 2000,
		Count:                 1,
		PressMS:               100,
	}); err != nil {
		t.Fatalf("StartDebug: %v", err)
	}
	status := waitForFingerRobotResult(t, controller)
	if status.Result != FingerRobotResultFailed || status.Error == "" {
		t.Fatalf("expected failed status, got %+v", status)
	}
	if controller.IsAvailable() {
		t.Fatal("controller must become unavailable when its safety reset write fails")
	}

	backend.clearPulses()
	if err := controller.Press(testFingerRobotPressRequest(100)); err != nil {
		t.Fatalf("normal Press should reinitialize after failed safety reset: %v", err)
	}
	want := append([]int{FingerRobotSafeReleaseUS}, testFingerRobotCompletedPulses()...)
	if got := backend.pulseSnapshot(); !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected recovered normal press sequence: got %v want %v", got, want)
	}
}

func TestDebugLeaseTimeoutRestoresDefaultReleasePoint(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	controller.debugLeaseTimeout = 20 * time.Millisecond
	initializeControllerForTest(t, controller, backend)

	if err := controller.StartDebug(FingerRobotDebugRequest{
		ReleaseUS:             1400,
		PressUS:               900,
		PressSpeedUSPerSecond: 1000,
		Count:                 1,
		PressMS:               100,
	}); err != nil {
		t.Fatalf("StartDebug: %v", err)
	}
	status := waitForFingerRobotResult(t, controller)
	if status.Result != FingerRobotResultCancelled || status.Error == "" {
		t.Fatalf("expected cancelled lease status, got %+v", status)
	}
	if pulses := backend.pulseSnapshot(); pulses[len(pulses)-1] != fingerRobotLimits.ReleaseDefaultUS {
		t.Fatalf("lease timeout did not restore the default release point: %v", pulses)
	}
	if controller.TouchDebugLease() {
		t.Fatal("completed debug operation must not accept a lease renewal")
	}
}

func TestDebugLeaseHeartbeatsAllowNormalCompletion(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	// Keep the operation longer than the lease so heartbeats are required, but
	// leave enough scheduling margin for this timing test under a busy race/full
	// test run.
	controller.debugLeaseTimeout = time.Second
	initializeControllerForTest(t, controller, backend)

	if err := controller.StartDebug(FingerRobotDebugRequest{
		ReleaseUS:             1400,
		PressUS:               900,
		PressSpeedUSPerSecond: 2000,
		Count:                 1,
		PressMS:               1200,
	}); err != nil {
		t.Fatalf("StartDebug: %v", err)
	}

	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		status := controller.Status()
		if !status.Running {
			if status.Result != FingerRobotResultSuccess || status.State != FingerRobotStateReleasing {
				t.Fatalf("unexpected heartbeat-protected result: %+v", status)
			}
			break
		}
		if !controller.TouchDebugLease() {
			status = waitForFingerRobotResult(t, controller)
			if status.Result != FingerRobotResultSuccess || status.State != FingerRobotStateReleasing {
				t.Fatalf("unexpected heartbeat-protected result: %+v", status)
			}
			break
		}
		select {
		case <-ticker.C:
		case <-deadline.C:
			t.Fatalf("timed out waiting for heartbeat-protected debug test: %+v", status)
		}
	}
	if pulses := backend.pulseSnapshot(); pulses[len(pulses)-1] != 1400 {
		t.Fatalf("heartbeat-protected test did not end at release point: %v", pulses)
	}
	if controller.TouchDebugLease() {
		t.Fatal("completed debug operation must not accept a lease renewal")
	}
}

func TestExpiredDebugLeaseCannotBeRevived(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	controller.debugLeaseTimeout = time.Second
	initializeControllerForTest(t, controller, backend)

	if err := controller.StartDebug(FingerRobotDebugRequest{
		ReleaseUS:             1400,
		PressUS:               900,
		PressSpeedUSPerSecond: 1000,
		Count:                 1,
		PressMS:               30000,
	}); err != nil {
		t.Fatalf("StartDebug: %v", err)
	}
	waitForPulses(t, backend, 1400)

	controller.mu.Lock()
	controller.active.leaseDeadline = time.Now().Add(-time.Millisecond)
	controller.mu.Unlock()
	if controller.TouchDebugLease() {
		t.Fatal("an expired debug lease must not be revived")
	}

	status := waitForFingerRobotResult(t, controller)
	if status.Result != FingerRobotResultCancelled || status.State != FingerRobotStateReleasing {
		t.Fatalf("expired lease did not cancel and release the operation: %+v", status)
	}
	if pulses := backend.pulseSnapshot(); pulses[len(pulses)-1] != fingerRobotLimits.ReleaseDefaultUS {
		t.Fatalf("expired lease did not restore the default release point: %v", pulses)
	}
}

func TestNormalPressOperationDoesNotUseDebugLease(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	controller.debugLeaseTimeout = 10 * time.Millisecond
	initializeControllerForTest(t, controller, backend)

	pressDone := make(chan error, 1)
	go func() {
		pressDone <- controller.Press(testFingerRobotPressRequest(100))
	}()
	waitForTestFingerRobotHolding(t, controller, backend)
	if controller.TouchDebugLease() {
		t.Fatal("normal press operation must not expose a debug lease")
	}

	select {
	case err := <-pressDone:
		if err != nil {
			t.Fatalf("normal press operation was unexpectedly cancelled: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for normal press operation")
	}
	if pulses := backend.pulseSnapshot(); pulses[len(pulses)-1] != FingerRobotSafeReleaseUS {
		t.Fatalf("normal press operation did not end at the safe release point: %v", pulses)
	}
}

func TestNormalPressFailuresRestoreSafeReleasePoint(t *testing.T) {
	for _, test := range []struct {
		name       string
		failPulse  int
		wantPulses []int
	}{
		{
			name:       "initial release write",
			failPulse:  FingerRobotSafeReleaseUS,
			wantPulses: []int{FingerRobotSafeReleaseUS, FingerRobotSafeReleaseUS},
		},
		{
			name:       "profile press write",
			failPulse:  2000,
			wantPulses: testFingerRobotImmediateSafeReleasePulses(),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			backend := newFakeFingerRobotBackend()
			controller := newFingerRobotController(backend)
			initializeControllerForTest(t, controller, backend)
			backend.failNextPulse(test.failPulse)

			if err := controller.Press(testFingerRobotPressRequest(100)); err == nil {
				t.Fatal("expected injected normal press operation failure")
			}
			if got := backend.pulseSnapshot(); !reflect.DeepEqual(got, test.wantPulses) {
				t.Fatalf("normal press failure did not end safely: got %v want %v", got, test.wantPulses)
			}
			if !controller.IsAvailable() {
				t.Fatal("successful safety release should keep the controller available")
			}
		})
	}
}

func TestNormalPressFinalReleaseFailureRetriesSafeReleasePoint(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	initializeControllerForTest(t, controller, backend)

	pressDone := make(chan error, 1)
	go func() {
		pressDone <- controller.Press(testFingerRobotPressRequest(100))
	}()
	waitForTestFingerRobotHolding(t, controller, backend)
	backend.failNextPulse(FingerRobotSafeReleaseUS)

	if err := <-pressDone; err == nil {
		t.Fatal("expected injected final release failure")
	}
	want := append(testFingerRobotCompletedPulses(), FingerRobotSafeReleaseUS)
	if got := backend.pulseSnapshot(); !reflect.DeepEqual(got, want) {
		t.Fatalf("final release failure did not retry safe point: got %v want %v", got, want)
	}
	if !controller.IsAvailable() {
		t.Fatal("successful recovery release should keep the controller available")
	}
}

func TestNeverInitializedDisableShutdownAndActionsDoNotTouchHardware(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	req := FingerRobotDebugRequest{
		ReleaseUS:             FingerRobotSafeReleaseUS,
		PressUS:               1900,
		PressSpeedUSPerSecond: 1000,
		Count:                 1,
		PressMS:               100,
	}

	if err := controller.Disable(); err != nil {
		t.Fatalf("Disable before Initialize: %v", err)
	}
	if err := controller.Press(testFingerRobotPressRequest(100)); !errors.Is(err, ErrFingerRobotUnavailable) {
		t.Fatalf("disabled normal Press must be unavailable, got %v", err)
	}
	if err := controller.StartDebug(req); !errors.Is(err, ErrFingerRobotUnavailable) {
		t.Fatalf("disabled StartDebug must be unavailable, got %v", err)
	}
	if err := controller.Reset(); !errors.Is(err, ErrFingerRobotUnavailable) {
		t.Fatalf("disabled Reset must be unavailable, got %v", err)
	}
	if err := controller.Shutdown(); err != nil {
		t.Fatalf("Shutdown before Initialize: %v", err)
	}
	if got := backend.initCallCount(); got != 0 {
		t.Fatalf("disabled lifecycle unexpectedly initialized hardware %d times", got)
	}
	if pulses := backend.pulseSnapshot(); len(pulses) != 0 {
		t.Fatalf("disabled lifecycle unexpectedly wrote PWM: %v", pulses)
	}
	if got := backend.shutdownPinRestoreCount(); got != 0 {
		t.Fatalf("disabled lifecycle unexpectedly restored shutdown pins %d times", got)
	}
	if err := controller.Initialize(); !errors.Is(err, ErrFingerRobotClosed) {
		t.Fatalf("Initialize after Shutdown must stay closed, got %v", err)
	}
}

func TestInitializeFingerRobotControllerRejectsInvalidProfilesWithoutTouchingHardware(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	originalController := fingerRobotController
	originalValidationErr := fingerRobotArmProfilesValidationErr
	t.Cleanup(func() {
		fingerRobotController = originalController
		fingerRobotArmProfilesValidationErr = originalValidationErr
	})

	fingerRobotController = newFingerRobotController(backend)
	fingerRobotArmProfilesValidationErr = errors.Join(ErrFingerRobotInvalid, errors.New("invalid test profile"))

	if err := InitializeFingerRobotController(); !errors.Is(err, ErrFingerRobotInvalid) {
		t.Fatalf("InitializeFingerRobotController error = %v, want ErrFingerRobotInvalid", err)
	}
	if got := backend.initCallCount(); got != 0 {
		t.Fatalf("invalid profiles initialized PWM %d times", got)
	}
	if pulses := backend.pulseSnapshot(); len(pulses) != 0 {
		t.Fatalf("invalid profiles wrote PWM: %v", pulses)
	}
}

func TestDisableCancelsNormalPressBlocksActionsAndAllowsExplicitReinitialize(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	initializeControllerForTest(t, controller, backend)

	pressDone := make(chan error, 1)
	go func() {
		pressDone <- controller.Press(testFingerRobotPressRequest(30000))
	}()
	waitForTestFingerRobotHolding(t, controller, backend)

	if err := controller.Disable(); err != nil {
		t.Fatalf("Disable: %v", err)
	}
	select {
	case err := <-pressDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("disabled normal press operation returned %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Disable did not unblock the synchronous normal press operation")
	}
	if controller.IsAvailable() {
		t.Fatal("disabled controller is still available")
	}
	if got := backend.pulseSnapshot(); !reflect.DeepEqual(got, testFingerRobotImmediateSafeReleasePulses()) {
		t.Fatalf("Disable did not own the final safe pulse: %v", got)
	}
	if got := backend.initCallCount(); got != 1 {
		t.Fatalf("Disable unexpectedly reinitialized hardware: init calls=%d", got)
	}

	backend.clearPulses()
	if err := controller.Press(testFingerRobotPressRequest(100)); !errors.Is(err, ErrFingerRobotUnavailable) {
		t.Fatalf("normal Press bypassed Disable: %v", err)
	}
	if err := controller.StartDebug(FingerRobotDebugRequest{
		ReleaseUS:             FingerRobotSafeReleaseUS,
		PressUS:               1900,
		PressSpeedUSPerSecond: 1000,
		Count:                 1,
		PressMS:               100,
	}); !errors.Is(err, ErrFingerRobotUnavailable) {
		t.Fatalf("StartDebug bypassed Disable: %v", err)
	}
	if err := controller.Reset(); !errors.Is(err, ErrFingerRobotUnavailable) {
		t.Fatalf("Reset bypassed Disable: %v", err)
	}
	if got := backend.initCallCount(); got != 1 {
		t.Fatalf("disabled action lazily initialized hardware: init calls=%d", got)
	}
	if pulses := backend.pulseSnapshot(); len(pulses) != 0 {
		t.Fatalf("disabled action wrote PWM: %v", pulses)
	}

	if err := controller.Initialize(); err != nil {
		t.Fatalf("re-Initialize after Disable: %v", err)
	}
	if !controller.IsAvailable() {
		t.Fatal("reinitialized controller is unavailable")
	}
	if got := backend.initCallCount(); got != 2 {
		t.Fatalf("explicit reinitialize did not reach backend: init calls=%d", got)
	}
	backend.clearPulses()
	if err := controller.Press(testFingerRobotPressRequest(100)); err != nil {
		t.Fatalf("normal Press after reinitialize: %v", err)
	}
	if got := backend.pulseSnapshot(); !reflect.DeepEqual(got, testFingerRobotCompletedPulses()) {
		t.Fatalf("reinitialized controller used an invalid normal press sequence: %v", got)
	}
}

func TestDisableCancelsDebugRampAndWinsFinalPulse(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	initializeControllerForTest(t, controller, backend)

	if err := controller.StartDebug(FingerRobotDebugRequest{
		ReleaseUS:             1400,
		PressUS:               900,
		PressSpeedUSPerSecond: 200,
		Count:                 0,
		PressMS:               30000,
	}); err != nil {
		t.Fatalf("StartDebug: %v", err)
	}
	waitForPulses(t, backend, 1400, testFingerRobotFirstRampPulse(1400, 900, 200))

	if err := controller.Disable(); err != nil {
		t.Fatalf("Disable: %v", err)
	}
	pulses := backend.pulseSnapshot()
	if got := pulses[len(pulses)-1]; got != FingerRobotSafeReleaseUS {
		t.Fatalf("Disable did not win the final Debug pulse: %v", pulses)
	}
	for index, pulseUS := range pulses {
		if pulseUS == FingerRobotFactoryZeroUS {
			t.Fatalf("Disable wrote the factory-zero point at index %d: %v", index, pulses)
		}
	}
	time.Sleep(50 * time.Millisecond)
	if got := backend.pulseSnapshot(); !reflect.DeepEqual(got, pulses) {
		t.Fatalf("Debug ramp continued after Disable: before=%v after=%v", pulses, got)
	}
}

func TestConcurrentInitializeAndDisableAreIdempotent(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)

	runConcurrent := func(action func() error) []error {
		const workers = 32
		start := make(chan struct{})
		errs := make(chan error, workers)
		var wg sync.WaitGroup
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				errs <- action()
			}()
		}
		close(start)
		wg.Wait()
		close(errs)
		result := make([]error, 0, workers)
		for err := range errs {
			result = append(result, err)
		}
		return result
	}

	for _, err := range runConcurrent(controller.Initialize) {
		if err != nil {
			t.Fatalf("concurrent Initialize: %v", err)
		}
	}
	if got := backend.initCallCount(); got != 1 {
		t.Fatalf("concurrent Initialize reached backend %d times", got)
	}
	if got := backend.pulseSnapshot(); !reflect.DeepEqual(got, []int{FingerRobotSafeReleaseUS}) {
		t.Fatalf("concurrent Initialize wrote unexpected pulses: %v", got)
	}

	backend.clearPulses()
	for _, err := range runConcurrent(controller.Disable) {
		if err != nil {
			t.Fatalf("concurrent Disable: %v", err)
		}
	}
	if got := backend.initCallCount(); got != 1 {
		t.Fatalf("concurrent Disable reinitialized backend: init calls=%d", got)
	}
	if got := backend.pulseSnapshot(); !reflect.DeepEqual(got, []int{FingerRobotSafeReleaseUS}) {
		t.Fatalf("concurrent Disable must write one final safe pulse: %v", got)
	}
}

func TestDisableReleaseFailureStaysDisabledAndCanRecoverExplicitly(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	initializeControllerForTest(t, controller, backend)
	backend.failNextPulse(FingerRobotSafeReleaseUS)

	if err := controller.Disable(); err == nil {
		t.Fatal("expected injected Disable release failure")
	}
	if controller.IsAvailable() {
		t.Fatal("failed Disable must remain unavailable")
	}
	backend.clearPulses()
	if err := controller.Press(testFingerRobotPressRequest(100)); !errors.Is(err, ErrFingerRobotUnavailable) {
		t.Fatalf("failed Disable was bypassed by normal Press: %v", err)
	}
	if err := controller.Reset(); !errors.Is(err, ErrFingerRobotUnavailable) {
		t.Fatalf("failed Disable was bypassed by Reset: %v", err)
	}
	if got := backend.initCallCount(); got != 1 {
		t.Fatalf("failed Disable triggered implicit initialization: init calls=%d", got)
	}

	if err := controller.Initialize(); err != nil {
		t.Fatalf("explicit recovery Initialize: %v", err)
	}
	if !controller.IsAvailable() {
		t.Fatal("explicit recovery did not restore availability")
	}
}

func TestShutdownAfterDisableDoesNotReinitializeHardware(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	initializeControllerForTest(t, controller, backend)

	if err := controller.Disable(); err != nil {
		t.Fatalf("Disable: %v", err)
	}
	backend.clearPulses()
	if err := controller.Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	if got := backend.initCallCount(); got != 1 {
		t.Fatalf("Shutdown after Disable reinitialized hardware: init calls=%d", got)
	}
	if got := backend.pulseSnapshot(); !reflect.DeepEqual(got, []int{FingerRobotSafeReleaseUS}) {
		t.Fatalf("Shutdown after Disable did not preserve safe output: %v", got)
	}
	if got := backend.shutdownPinRestoreCount(); got != 1 {
		t.Fatalf("Shutdown after Disable restored shutdown pins %d times, want 1", got)
	}
}

func TestShutdownRestoresPinsBeforeClosingBackend(t *testing.T) {
	backend := &shutdownOrderFingerRobotBackend{}
	controller := newFingerRobotController(backend)
	if err := controller.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	backend.events = nil

	if err := controller.Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	want := []string{
		"pulse:" + fmt.Sprint(FingerRobotSafeReleaseUS),
		"restore-pins",
		"close",
	}
	if !reflect.DeepEqual(backend.events, want) {
		t.Fatalf("shutdown backend order = %v, want %v", backend.events, want)
	}
}

func TestShutdownCancelsOperationReleasesAndRejectsNewActions(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	initializeControllerForTest(t, controller, backend)

	req := FingerRobotDebugRequest{ReleaseUS: 1400, PressUS: 900, PressSpeedUSPerSecond: 1000, Count: 1, PressMS: 30000}
	if err := controller.StartDebug(req); err != nil {
		t.Fatalf("StartDebug: %v", err)
	}
	waitForPulses(t, backend, 1400)
	if err := controller.Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	if controller.IsAvailable() {
		t.Fatal("shut down controller must be unavailable")
	}
	if pulses := backend.pulseSnapshot(); pulses[len(pulses)-1] != FingerRobotSafeReleaseUS {
		t.Fatalf("shutdown did not end at the safe release point: %v", pulses)
	}
	if got := backend.shutdownPinRestoreCount(); got != 1 {
		t.Fatalf("shutdown restored shutdown pins %d times, want 1", got)
	}
	if err := controller.StartDebug(req); !errors.Is(err, ErrFingerRobotClosed) {
		t.Fatalf("expected closed error for debug after shutdown, got %v", err)
	}
	if err := controller.Press(testFingerRobotPressRequest(800)); !errors.Is(err, ErrFingerRobotClosed) {
		t.Fatalf("expected closed error for normal Press after shutdown, got %v", err)
	}
	if err := controller.Shutdown(); err != nil {
		t.Fatalf("second Shutdown must be idempotent: %v", err)
	}
	if got := backend.shutdownPinRestoreCount(); got != 1 {
		t.Fatalf("second Shutdown repeated pin restore: calls=%d", got)
	}
}

func TestShutdownRestoresPinsWhenSafeReleaseFails(t *testing.T) {
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	initializeControllerForTest(t, controller, backend)

	restoreErr := errors.New("shutdown pin restore failed")
	backend.failNextPulse(FingerRobotSafeReleaseUS)
	backend.setShutdownPinRestoreError(restoreErr)

	err := controller.Shutdown()
	if err == nil || !strings.Contains(err.Error(), "release finger robot during shutdown") {
		t.Fatalf("Shutdown error = %v, want safe release failure", err)
	}
	if !errors.Is(err, restoreErr) {
		t.Fatalf("Shutdown error = %v, want shutdown pin restore error %v", err, restoreErr)
	}
	if got := backend.shutdownPinRestoreCount(); got != 1 {
		t.Fatalf("safe release failure restored shutdown pins %d times, want 1", got)
	}
}

func TestFingerRobotHotplugReconnectRestoresSafePointBeforeAvailability(t *testing.T) {
	backend := newFakeFingerRobotHotplugBackend()
	controller := newFingerRobotController(backend)
	controller.connectionPollInterval = 2 * time.Millisecond
	controller.connectionLossTimeout = 20 * time.Millisecond
	controller.connectionRetryInterval = 20 * time.Millisecond
	t.Cleanup(func() { _ = controller.Shutdown() })

	if err := controller.Initialize(); !errors.Is(err, ErrFingerRobotDisconnected) {
		t.Fatalf("Initialize without servo = %v, want disconnected", err)
	}
	if controller.IsAvailable() {
		t.Fatal("controller became available without PWM feedback")
	}
	backend.backend.clearPulses()

	backend.setConnected(true)
	waitForFingerRobotCondition(t, func() bool { return controller.IsAvailable() }, "hot-plug recovery")
	if got := backend.backend.pulseSnapshot(); !reflect.DeepEqual(got, []int{FingerRobotSafeReleaseUS}) {
		t.Fatalf("hot-plug recovery pulses = %v, want safe release", got)
	}
}

func TestFingerRobotHotUnplugCancelsActiveOperationAndForcesSafeOutput(t *testing.T) {
	backend := newFakeFingerRobotHotplugBackend()
	backend.setConnected(true)
	controller := newFingerRobotController(backend)
	controller.connectionPollInterval = 2 * time.Millisecond
	controller.connectionLossTimeout = 20 * time.Millisecond
	controller.connectionRetryInterval = 50 * time.Millisecond
	t.Cleanup(func() { _ = controller.Shutdown() })

	if err := controller.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	backend.backend.clearPulses()
	req := FingerRobotDebugRequest{
		ReleaseUS:             1400,
		PressUS:               1300,
		PressSpeedUSPerSecond: 2000,
		Count:                 1,
		PressMS:               30000,
	}
	if err := controller.StartDebug(req); err != nil {
		t.Fatalf("StartDebug: %v", err)
	}
	waitForPulses(t, backend.backend, append(
		[]int{req.ReleaseUS},
		testFingerRobotRampPulses(req.ReleaseUS, req.PressUS, req.PressSpeedUSPerSecond)...,
	)...)

	backend.setConnected(false)
	waitForFingerRobotCondition(t, func() bool {
		status := controller.Status()
		return !controller.IsAvailable() && !status.Running && strings.Contains(status.Error, ErrFingerRobotDisconnected.Error())
	}, "hot-unplug failure state")

	pulses := backend.backend.pulseSnapshot()
	if len(pulses) == 0 || pulses[len(pulses)-1] != FingerRobotSafeReleaseUS {
		t.Fatalf("hot-unplug final pulses = %v, want forced safe release", pulses)
	}
	if backend.forceCallCount() == 0 {
		t.Fatal("hot-unplug did not use the disconnected safe-release path")
	}
	time.Sleep(30 * time.Millisecond)
	if got := backend.backend.pulseSnapshot(); !reflect.DeepEqual(got, pulses) {
		t.Fatalf("operation continued after unplug: before=%v after=%v", pulses, got)
	}
}

func TestFingerRobotHotUnplugExternalFinalizerOwnsLastDuty(t *testing.T) {
	backend := newFakeFingerRobotHotplugBackend()
	backend.mu.Lock()
	backend.hardwareReady = true
	backend.mu.Unlock()
	forceComplete := make(chan struct{}, 1)
	backend.notifyForceCompletion(forceComplete)

	controller := newFingerRobotController(backend)
	t.Cleanup(controller.cancel)
	controller.mu.Lock()
	controller.enabled = true
	controller.available = true
	controller.mu.Unlock()
	op, err := controller.beginOperation(1, false, FingerRobotSafeReleaseUS)
	if err != nil {
		t.Fatalf("begin operation: %v", err)
	}

	lateWriteDone := make(chan struct{})
	go func() {
		<-forceComplete
		// Model a duty write that passed its cancellation check before the
		// monitor claimed the external finalizer, then completed after the first
		// forced release because feedback returned during a fast re-plug.
		_ = backend.backend.SetPulseUS(1900)
		controller.completeOperation(op, context.Canceled)
		close(lateWriteDone)
	}()

	controller.handleConnectionLost(backend, time.Millisecond)
	<-lateWriteDone
	if got := backend.backend.pulseSnapshot(); !reflect.DeepEqual(got, []int{
		FingerRobotSafeReleaseUS,
		1900,
		FingerRobotSafeReleaseUS,
	}) {
		t.Fatalf("disconnect finalizer pulses = %v, want safe point to own final duty", got)
	}
	if backend.forceCallCount() != 2 {
		t.Fatalf("forced safe-release calls = %d, want pre- and post-cancel writes", backend.forceCallCount())
	}
}

func TestFingerRobotDisableImmediatelyAfterFeedbackLossForcesSafeOutput(t *testing.T) {
	backend := newFakeFingerRobotHotplugBackend()
	backend.setConnected(true)
	controller := newFingerRobotController(backend)
	t.Cleanup(func() { _ = controller.Shutdown() })

	if err := controller.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	backend.backend.clearPulses()

	// Model a cable removal immediately after the last falling edge: the
	// connection-health window is still fresh, but the boundary wait sees no
	// subsequent edge and times out.
	backend.failNextSetPulse(fmt.Errorf("wait for PWM feedback: %w", errFingerRobotGPIOWaitTimeout))
	if err := controller.Disable(); err != nil {
		t.Fatalf("Disable immediately after feedback loss: %v", err)
	}
	if controller.IsAvailable() {
		t.Fatal("disabled controller remained available")
	}
	if backend.forceCallCount() != 1 {
		t.Fatalf("forced safe-release calls = %d, want 1", backend.forceCallCount())
	}
	if got := backend.backend.pulseSnapshot(); !reflect.DeepEqual(got, []int{FingerRobotSafeReleaseUS}) {
		t.Fatalf("immediate feedback-loss pulses = %v, want forced safe release", got)
	}
}

func TestFingerRobotHotUnplugResetKeepsFactoryZeroHeld(t *testing.T) {
	backend := newFakeFingerRobotHotplugBackend()
	backend.setConnected(true)
	controller := newFingerRobotController(backend)
	controller.connectionPollInterval = time.Hour
	controller.connectionLossTimeout = 20 * time.Millisecond
	t.Cleanup(func() { _ = controller.Shutdown() })

	if err := controller.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	backend.backend.clearPulses()

	// A disconnected servo must not prevent the explicit factory-zero write.
	backend.setConnected(false)
	if err := controller.Reset(); err != nil {
		t.Fatalf("Reset after unplug: %v", err)
	}
	if !controller.IsAvailable() {
		t.Fatal("factory-zero hold unexpectedly marked the controller unavailable")
	}
	if got := backend.backend.pulseSnapshot(); !reflect.DeepEqual(got, []int{FingerRobotFactoryZeroUS}) {
		t.Fatalf("reset after unplug pulses = %v, want only factory zero", got)
	}
	status := controller.Status()
	if status.Running || status.State != FingerRobotStateIdle || status.Result != FingerRobotResultReset {
		t.Fatalf("unexpected factory-zero hold status: %+v", status)
	}

	// The connection monitor must leave the held 1500 us output untouched.
	controller.handleConnectionLost(backend, controller.connectionLossTimeout)
	if backend.forceCallCount() != 0 {
		t.Fatalf("factory-zero hold triggered %d forced safe-release calls", backend.forceCallCount())
	}

	// Starting a Debug test exits the hold and restores the original gated path.
	timeoutErr := fmt.Errorf("wait for PWM feedback: %w", errFingerRobotGPIOWaitTimeout)
	backend.failNextSetPulse(timeoutErr)
	if err := controller.StartDebug(FingerRobotDebugRequest{
		ReleaseUS:             1400,
		PressUS:               1300,
		PressSpeedUSPerSecond: 2000,
		Count:                 1,
		PressMS:               100,
	}); err != nil {
		t.Fatalf("StartDebug: %v", err)
	}
	status = waitForFingerRobotResult(t, controller)
	if status.Result != FingerRobotResultFailed || status.Error == "" {
		t.Fatalf("StartDebug did not restore gated failure behavior: %+v", status)
	}
	if backend.forceCallCount() != 1 {
		t.Fatalf("gated Debug failure forced safe-release calls = %d, want 1", backend.forceCallCount())
	}
	pulses := backend.backend.pulseSnapshot()
	if len(pulses) == 0 || pulses[len(pulses)-1] != FingerRobotSafeReleaseUS {
		t.Fatalf("gated Debug failure final pulses = %v, want safe release", pulses)
	}
}

func TestFingerRobotHotplugRetriesHardwareFailureAndStopsWhileDisabled(t *testing.T) {
	backend := newFakeFingerRobotHotplugBackend()
	backend.setConnected(true)
	backend.backend.setInitErrors(errors.New("transient GPIO setup failure"), nil)
	controller := newFingerRobotController(backend)
	controller.connectionPollInterval = 2 * time.Millisecond
	controller.connectionLossTimeout = 20 * time.Millisecond
	controller.connectionRetryInterval = 10 * time.Millisecond
	t.Cleanup(func() { _ = controller.Shutdown() })

	if err := controller.Initialize(); err == nil {
		t.Fatal("initial hardware failure unexpectedly succeeded")
	}
	waitForFingerRobotCondition(t, func() bool { return controller.IsAvailable() }, "hardware retry")
	if backend.backend.initCallCount() < 2 {
		t.Fatalf("hardware initialization calls = %d, want retry", backend.backend.initCallCount())
	}

	if err := controller.Disable(); err != nil {
		t.Fatalf("Disable: %v", err)
	}
	initCalls := backend.backend.initCallCount()
	backend.setConnected(false)
	backend.setConnected(true)
	time.Sleep(30 * time.Millisecond)
	if controller.IsAvailable() {
		t.Fatal("disabled controller recovered from hot-plug")
	}
	if got := backend.backend.initCallCount(); got != initCalls {
		t.Fatalf("disabled controller retried hardware: before=%d after=%d", initCalls, got)
	}
}

func TestWaitFingerRobotCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := waitFingerRobot(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}

func testFingerRobotPressRequest(pressMS int) FingerRobotPressRequest {
	return FingerRobotPressRequest{
		ProfileID: FingerRobotArmProfileLongID,
		Strength:  1,
		PressMS:   pressMS,
		Speed:     FingerRobotSpeedFast,
	}
}

func testFingerRobotRampPulses(fromUS, toUS, speedUSPerSecond int) []int {
	steps := fingerRobotRampSteps(fromUS, toUS, speedUSPerSecond)
	pulses := make([]int, 0, steps)
	lastPulseUS := fromUS
	for step := 1; step <= steps; step++ {
		pulseUS := fingerRobotRampPulse(fromUS, toUS, step, steps)
		if pulseUS != lastPulseUS {
			pulses = append(pulses, pulseUS)
			lastPulseUS = pulseUS
		}
	}
	return pulses
}

func testFingerRobotFirstRampPulse(fromUS, toUS, speedUSPerSecond int) int {
	steps := fingerRobotRampSteps(fromUS, toUS, speedUSPerSecond)
	return fingerRobotRampPulse(fromUS, toUS, 1, steps)
}

func testFingerRobotPressPulsesBeforeRelease() []int {
	pulses := []int{FingerRobotSafeReleaseUS}
	return append(pulses, testFingerRobotRampPulses(
		FingerRobotSafeReleaseUS,
		2000,
		FingerRobotFastSpeedUSPerSecond,
	)...)
}

func testFingerRobotCompletedPulses() []int {
	pulses := testFingerRobotPressPulsesBeforeRelease()
	return append(pulses, testFingerRobotRampPulses(
		2000,
		FingerRobotSafeReleaseUS,
		FingerRobotFastSpeedUSPerSecond,
	)...)
}

func testFingerRobotImmediateSafeReleasePulses() []int {
	pulses := testFingerRobotPressPulsesBeforeRelease()
	return append(pulses, FingerRobotSafeReleaseUS)
}

func waitForTestFingerRobotHolding(t *testing.T, controller *FingerRobotController, backend *fakeFingerRobotBackend) {
	t.Helper()
	waitForPulses(t, backend, testFingerRobotPressPulsesBeforeRelease()...)
	waitForFingerRobotState(t, controller, FingerRobotStateHolding)
}

func initializeControllerForTest(t *testing.T, controller *FingerRobotController, backend *fakeFingerRobotBackend) {
	t.Helper()
	if err := controller.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if !controller.IsAvailable() {
		t.Fatal("initialized controller is unavailable")
	}
	status := controller.Status()
	if status.State != FingerRobotStateIdle || status.Result != FingerRobotResultIdle {
		t.Fatalf("unexpected initialized status: %+v", status)
	}
	if got := backend.pulseSnapshot(); !reflect.DeepEqual(got, []int{FingerRobotSafeReleaseUS}) {
		t.Fatalf("initialization did not end at the safe release point: %v", got)
	}
	backend.clearPulses()
}

func waitForFingerRobotResult(t *testing.T, controller *FingerRobotController) FingerRobotStatus {
	t.Helper()
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()

	for {
		status := controller.Status()
		if !status.Running && status.Result != FingerRobotResultRunning {
			return status
		}
		select {
		case <-deadline.C:
			t.Fatalf("timed out waiting for controller result; last status: %+v", status)
		case <-ticker.C:
		}
	}
}

func waitForFingerRobotState(t *testing.T, controller *FingerRobotController, want FingerRobotState) FingerRobotStatus {
	t.Helper()
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()

	for {
		status := controller.Status()
		if status.State == want {
			return status
		}
		select {
		case <-deadline.C:
			t.Fatalf("timed out waiting for finger robot state %q; last status: %+v", want, status)
		case <-ticker.C:
		}
	}
}

func waitForPulses(t *testing.T, backend *fakeFingerRobotBackend, expected ...int) {
	t.Helper()
	for index, want := range expected {
		select {
		case got := <-backend.pulseCh:
			if got != want {
				t.Fatalf("unexpected pulse %d: got %d want %d; full sequence %v", index, got, want, backend.pulseSnapshot())
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for pulse %d (%d); full sequence %v", index, want, backend.pulseSnapshot())
		}
	}
}

func waitForFingerRobotCondition(t *testing.T, condition func() bool, description string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", description)
		}
		time.Sleep(time.Millisecond)
	}
}
