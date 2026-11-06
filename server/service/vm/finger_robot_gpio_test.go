package vm

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"
)

type fakeFingerRobotPWMLowGate struct {
	waitErr    error
	waitCalls  []time.Duration
	connected  bool
	closeErr   error
	closeCalls int
	onWait     func()
}

func (g *fakeFingerRobotPWMLowGate) WaitLow(timeout time.Duration) error {
	g.waitCalls = append(g.waitCalls, timeout)
	if g.onWait != nil {
		g.onWait()
	}
	return g.waitErr
}

func (g *fakeFingerRobotPWMLowGate) Close() error {
	g.closeCalls++
	return g.closeErr
}

func (g *fakeFingerRobotPWMLowGate) Connected(time.Duration) bool {
	return g.connected
}

func TestFingerRobotGPIOCaptureUsesRequestedOffset(t *testing.T) {
	if fingerRobotGPIOLineOffset != 3 {
		t.Fatalf("GPIO capture offset = %d, want 3", fingerRobotGPIOLineOffset)
	}
}

func TestPWMFingerRobotBackendWaitsForLowBeforeWritingDuty(t *testing.T) {
	var order []string
	gate := &fakeFingerRobotPWMLowGate{onWait: func() { order = append(order, "wait-low") }}
	backend := &pwmFingerRobotBackend{
		openLowGate: func() (fingerRobotPWMLowGate, error) {
			order = append(order, "open-gpio")
			return gate, nil
		},
		initializePWM: func() ([]string, error) {
			order = append(order, "initialize-pwm")
			return []string{"pwm0"}, nil
		},
		writePWM: func(paths []string, name, value string) error {
			order = append(order, "write-duty")
			if !reflect.DeepEqual(paths, []string{"pwm0"}) || name != "duty_cycle" || value != "1900000" {
				t.Fatalf("unexpected PWM write: paths=%v name=%s value=%s", paths, name, value)
			}
			return nil
		},
	}

	if err := backend.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if err := backend.SetPulseUS(1900); err != nil {
		t.Fatalf("SetPulseUS: %v", err)
	}
	wantOrder := []string{"open-gpio", "initialize-pwm", "wait-low", "write-duty"}
	if !reflect.DeepEqual(order, wantOrder) {
		t.Fatalf("backend order = %v, want %v", order, wantOrder)
	}
	if !reflect.DeepEqual(gate.waitCalls, []time.Duration{fingerRobotPWMLowWaitTimeout}) {
		t.Fatalf("gate waits = %v, want %v", gate.waitCalls, []time.Duration{fingerRobotPWMLowWaitTimeout})
	}
}

func TestFingerRobotInitializationUsesSeparateFeedbackTimeouts(t *testing.T) {
	gate := &fakeFingerRobotPWMLowGate{connected: true}
	backend := &pwmFingerRobotBackend{
		openLowGate: func() (fingerRobotPWMLowGate, error) { return gate, nil },
		initializePWM: func() ([]string, error) {
			return []string{"pwm0"}, nil
		},
		writePWM: func([]string, string, string) error { return nil },
	}
	controller := newFingerRobotController(backend)
	t.Cleanup(controller.cancel)

	if err := controller.initializeAndReleaseHardware(); err != nil {
		t.Fatalf("initializeAndReleaseHardware: %v", err)
	}
	want := []time.Duration{100 * time.Millisecond, 200 * time.Millisecond}
	if !reflect.DeepEqual(gate.waitCalls, want) {
		t.Fatalf("feedback waits = %v, want %v", gate.waitCalls, want)
	}
}

func TestPWMFingerRobotBackendDoesNotWriteWhenLowWaitFails(t *testing.T) {
	waitErr := errors.New("no falling edge")
	writeCalls := 0
	backend := &pwmFingerRobotBackend{
		pwmPaths:      []string{"pwm0"},
		lowGate:       &fakeFingerRobotPWMLowGate{waitErr: waitErr},
		hardwareReady: true,
		writePWM: func([]string, string, string) error {
			writeCalls++
			return nil
		},
	}

	if err := backend.SetPulseUS(1900); !errors.Is(err, waitErr) {
		t.Fatalf("SetPulseUS error = %v, want %v", err, waitErr)
	}
	if writeCalls != 0 {
		t.Fatalf("PWM was written %d times after GPIO wait failure", writeCalls)
	}
	if backend.HardwareReady() {
		t.Fatal("non-timeout GPIO wait failure left hardware ready")
	}
}

func TestPWMFingerRobotBackendTimeoutKeepsHardwareReady(t *testing.T) {
	timeoutErr := fmt.Errorf("no new edge: %w", errFingerRobotGPIOWaitTimeout)
	backend := &pwmFingerRobotBackend{
		pwmPaths:      []string{"pwm0"},
		lowGate:       &fakeFingerRobotPWMLowGate{waitErr: timeoutErr},
		hardwareReady: true,
		writePWM: func([]string, string, string) error {
			t.Fatal("timeout must not write PWM duty")
			return nil
		},
	}

	if err := backend.SetPulseUS(1900); !errors.Is(err, errFingerRobotGPIOWaitTimeout) {
		t.Fatalf("SetPulseUS error = %v, want timeout sentinel", err)
	}
	if !backend.HardwareReady() {
		t.Fatal("feedback timeout incorrectly marked GPIO/PWM hardware unready")
	}
}

func TestFingerRobotGPIOReadFailureForcesSafeReleaseAndMarksHardwareUnready(t *testing.T) {
	readErr := errors.New("GPIO read failed")
	gate := &fakeFingerRobotPWMLowGate{waitErr: readErr, connected: true}
	var writes []string
	backend := &pwmFingerRobotBackend{
		pwmPaths:      []string{"pwm0"},
		lowGate:       gate,
		hardwareReady: true,
		writePWM: func(_ []string, name, value string) error {
			writes = append(writes, name+"="+value)
			return nil
		},
	}
	controller := newFingerRobotController(backend)
	t.Cleanup(controller.cancel)

	if err := controller.setSafeReleaseWithFeedbackFallback(FingerRobotSafeReleaseUS); !errors.Is(err, ErrFingerRobotDisconnected) {
		t.Fatalf("safe release after GPIO read failure = %v, want disconnected", err)
	}
	if backend.HardwareReady() {
		t.Fatal("GPIO read failure left hardware ready")
	}
	if !reflect.DeepEqual(writes, []string{"duty_cycle=2050000"}) {
		t.Fatalf("safe release writes = %v, want forced 2050 us", writes)
	}
}

func TestFingerRobotInitializationFailureStillForcesSafeRelease(t *testing.T) {
	initErr := errors.New("PWM initialization failed")
	gate := &fakeFingerRobotPWMLowGate{}
	var writes []string
	backend := &pwmFingerRobotBackend{
		openLowGate: func() (fingerRobotPWMLowGate, error) { return gate, nil },
		initializePWM: func() ([]string, error) {
			return []string{"pwm0"}, initErr
		},
		writePWM: func(_ []string, name, value string) error {
			writes = append(writes, name+"="+value)
			return nil
		},
	}
	controller := newFingerRobotController(backend)
	t.Cleanup(controller.cancel)

	err := controller.initializeAndReleaseHardware()
	if !errors.Is(err, initErr) || !errors.Is(err, ErrFingerRobotDisconnected) {
		t.Fatalf("initializeAndReleaseHardware error = %v, want initialization and disconnected errors", err)
	}
	if !reflect.DeepEqual(writes, []string{"duty_cycle=2050000"}) {
		t.Fatalf("initialization failure safety writes = %v, want forced 2050 us", writes)
	}
}

func TestFingerRobotGPIOOpenFailureStillInitializesAndSafesPWM(t *testing.T) {
	gateErr := errors.New("GPIO line unavailable")
	initializePWMCalls := 0
	var writes []string
	backend := &pwmFingerRobotBackend{
		openLowGate: func() (fingerRobotPWMLowGate, error) { return nil, gateErr },
		initializePWM: func() ([]string, error) {
			initializePWMCalls++
			return []string{"pwm0"}, nil
		},
		writePWM: func(_ []string, name, value string) error {
			writes = append(writes, name+"="+value)
			return nil
		},
	}
	controller := newFingerRobotController(backend)
	t.Cleanup(controller.cancel)

	err := controller.initializeAndReleaseHardware()
	if !errors.Is(err, gateErr) || !errors.Is(err, ErrFingerRobotDisconnected) {
		t.Fatalf("initializeAndReleaseHardware error = %v, want GPIO and disconnected errors", err)
	}
	if initializePWMCalls != 1 {
		t.Fatalf("PWM initialization calls = %d, want 1 despite GPIO open failure", initializePWMCalls)
	}
	if !reflect.DeepEqual(backend.pwmPaths, []string{"pwm0"}) {
		t.Fatalf("GPIO open failure discarded PWM paths: %v", backend.pwmPaths)
	}
	if !reflect.DeepEqual(writes, []string{"duty_cycle=2050000"}) {
		t.Fatalf("GPIO open failure safety writes = %v, want forced 2050 us", writes)
	}
	if backend.HardwareReady() {
		t.Fatal("GPIO open failure incorrectly marked hardware ready")
	}
}

func TestPWMFingerRobotBackendWaitConnectedReadFailureMarksHardwareUnready(t *testing.T) {
	readErr := errors.New("GPIO read failed")
	backend := &pwmFingerRobotBackend{
		pwmPaths:      []string{"pwm0"},
		lowGate:       &fakeFingerRobotPWMLowGate{waitErr: readErr},
		hardwareReady: true,
	}

	if err := backend.WaitConnected(time.Second); !errors.Is(err, readErr) {
		t.Fatalf("WaitConnected error = %v, want %v", err, readErr)
	}
	if backend.HardwareReady() {
		t.Fatal("WaitConnected GPIO read failure left hardware ready")
	}
}

func TestPWMFingerRobotBackendHotplugHealthAndForcedSafeRelease(t *testing.T) {
	gate := &fakeFingerRobotPWMLowGate{connected: true}
	var writes []string
	backend := &pwmFingerRobotBackend{
		openLowGate: func() (fingerRobotPWMLowGate, error) { return gate, nil },
		initializePWM: func() ([]string, error) {
			return []string{"pwm0"}, nil
		},
		writePWM: func(_ []string, name, value string) error {
			writes = append(writes, name+"="+value)
			return nil
		},
	}

	if err := backend.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if !backend.HardwareReady() || !backend.ConnectionAlive(500*time.Millisecond) {
		t.Fatal("initialized connected backend did not report healthy")
	}
	if err := backend.WaitConnected(time.Second); err != nil {
		t.Fatalf("WaitConnected: %v", err)
	}
	if len(gate.waitCalls) != 1 {
		t.Fatalf("connection waits = %v, want one", gate.waitCalls)
	}
	if err := backend.ForceSafeRelease(); err != nil {
		t.Fatalf("ForceSafeRelease: %v", err)
	}
	if len(gate.waitCalls) != 1 {
		t.Fatalf("forced safe release unexpectedly waited for feedback: %v", gate.waitCalls)
	}
	if !reflect.DeepEqual(writes, []string{"duty_cycle=2050000"}) {
		t.Fatalf("forced safe release writes = %v", writes)
	}
}

func TestPWMFingerRobotBackendWriteFailureMarksHardwareUnready(t *testing.T) {
	writeErr := errors.New("PWM write failed")
	backend := &pwmFingerRobotBackend{
		pwmPaths:      []string{"pwm0"},
		lowGate:       &fakeFingerRobotPWMLowGate{},
		hardwareReady: true,
		writePWM: func([]string, string, string) error {
			return writeErr
		},
	}

	if err := backend.ForceSafeRelease(); !errors.Is(err, writeErr) {
		t.Fatalf("ForceSafeRelease error = %v, want %v", err, writeErr)
	}
	if backend.HardwareReady() {
		t.Fatal("failed PWM write left hardware ready")
	}
}

func TestPWMFingerRobotBackendReinitializeClosesPreviousGPIOCapture(t *testing.T) {
	firstGate := &fakeFingerRobotPWMLowGate{}
	secondGate := &fakeFingerRobotPWMLowGate{}
	openCalls := 0
	backend := &pwmFingerRobotBackend{
		openLowGate: func() (fingerRobotPWMLowGate, error) {
			openCalls++
			if openCalls == 1 {
				return firstGate, nil
			}
			return secondGate, nil
		},
		initializePWM: func() ([]string, error) { return []string{"pwm0"}, nil },
	}

	if err := backend.Initialize(); err != nil {
		t.Fatalf("first Initialize: %v", err)
	}
	if err := backend.Initialize(); err != nil {
		t.Fatalf("second Initialize: %v", err)
	}
	if firstGate.closeCalls != 1 || secondGate.closeCalls != 0 || backend.lowGate != secondGate {
		t.Fatalf(
			"unexpected reinitialize state: first closes=%d second closes=%d active=%T",
			firstGate.closeCalls,
			secondGate.closeCalls,
			backend.lowGate,
		)
	}
	if err := backend.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if secondGate.closeCalls != 1 || backend.lowGate != nil {
		t.Fatalf("active GPIO capture was not closed: closes=%d gate=%v", secondGate.closeCalls, backend.lowGate)
	}
}

func TestPWMFingerRobotBackendRetriesAfterPreviousGPIOCloseError(t *testing.T) {
	closeErr := errors.New("GPIO close failed")
	firstGate := &fakeFingerRobotPWMLowGate{closeErr: closeErr}
	secondGate := &fakeFingerRobotPWMLowGate{}
	openCalls := 0
	backend := &pwmFingerRobotBackend{
		lowGate: firstGate,
		openLowGate: func() (fingerRobotPWMLowGate, error) {
			openCalls++
			return secondGate, nil
		},
		initializePWM: func() ([]string, error) { return []string{"pwm0"}, nil },
	}

	if err := backend.Initialize(); !errors.Is(err, closeErr) {
		t.Fatalf("Initialize close error = %v, want %v", err, closeErr)
	}
	if backend.lowGate != nil || openCalls != 0 {
		t.Fatalf("failed close retained stale gate or reopened too early: gate=%v opens=%d", backend.lowGate, openCalls)
	}
	if err := backend.Initialize(); err != nil {
		t.Fatalf("retry Initialize: %v", err)
	}
	if backend.lowGate != secondGate || openCalls != 1 || firstGate.closeCalls != 1 {
		t.Fatalf(
			"retry did not replace failed-close gate: gate=%T opens=%d old closes=%d",
			backend.lowGate,
			openCalls,
			firstGate.closeCalls,
		)
	}
}
