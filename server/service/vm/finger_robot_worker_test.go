package vm

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"
)

type scriptedFingerRobotWorkerGate struct {
	mu sync.Mutex

	waitCalls    int
	waitStart    chan struct{}
	waitDone     chan struct{}
	blockOnCall  int
	waitErr      error
	connected    bool
	drainStarted chan struct{}
	drainErr     error
}

type fingerRobotPulseRecorder struct {
	mu     sync.Mutex
	pulses []int
}

func (r *fingerRobotPulseRecorder) append(pulse int) {
	r.mu.Lock()
	r.pulses = append(r.pulses, pulse)
	r.mu.Unlock()
}

func (r *fingerRobotPulseRecorder) snapshot() []int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]int(nil), r.pulses...)
}

func (g *scriptedFingerRobotWorkerGate) WaitLow(timeout time.Duration) error {
	return g.WaitLowContext(context.Background(), timeout)
}

func (g *scriptedFingerRobotWorkerGate) WaitLowContext(ctx context.Context, _ time.Duration) error {
	g.mu.Lock()
	g.waitCalls++
	callNumber := g.waitCalls
	start := g.waitStart
	done := g.waitDone
	blockOnCall := g.blockOnCall
	err := g.waitErr
	g.mu.Unlock()
	shouldBlock := start != nil || done != nil
	if blockOnCall > 0 {
		shouldBlock = callNumber == blockOnCall
	}
	if shouldBlock && start != nil {
		select {
		case start <- struct{}{}:
		default:
		}
	}
	if shouldBlock && done != nil {
		select {
		case <-done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return err
}

func (g *scriptedFingerRobotWorkerGate) Connected(time.Duration) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.connected
}

func (g *scriptedFingerRobotWorkerGate) Close() error { return nil }

func (g *scriptedFingerRobotWorkerGate) DrainEvents() error {
	g.mu.Lock()
	drainStarted := g.drainStarted
	err := g.drainErr
	g.mu.Unlock()
	if drainStarted != nil {
		select {
		case drainStarted <- struct{}{}:
		default:
		}
	}
	return err
}

func (g *scriptedFingerRobotWorkerGate) calls() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.waitCalls
}

func (g *scriptedFingerRobotWorkerGate) blockNextWait(start chan struct{}, done chan struct{}) {
	g.mu.Lock()
	g.waitCalls = 0
	g.waitStart = start
	g.waitDone = done
	g.blockOnCall = 1
	g.mu.Unlock()
}

func newTestFingerRobotWorker(t *testing.T, gate fingerRobotPWMLowGate) (*fingerRobotWorker, *fingerRobotWorkerBackend, *fingerRobotPulseRecorder) {
	t.Helper()
	recorder := &fingerRobotPulseRecorder{}
	backend := &pwmFingerRobotBackend{
		openLowGate: func() (fingerRobotPWMLowGate, error) { return gate, nil },
		initializePWM: func() ([]string, error) {
			return []string{"pwm0"}, nil
		},
		writePWM: func(_ []string, name, value string) error {
			if name != "duty_cycle" {
				return fmt.Errorf("unexpected PWM attribute %q", name)
			}
			pulseNS, err := strconv.Atoi(value)
			if err != nil {
				return err
			}
			recorder.append(pulseNS / 1000)
			return nil
		},
	}
	worker := newFingerRobotWorker(backend)
	worker.setupRealtime = func(cpu, priority int) (fingerRobotRealtimeCleanup, error) {
		if cpu != fingerRobotWorkerCPU || priority != fingerRobotWorkerPriority {
			return nil, fmt.Errorf("unexpected realtime configuration cpu=%d priority=%d", cpu, priority)
		}
		return func() error { return nil }, nil
	}
	adapter := &fingerRobotWorkerBackend{worker: worker}
	t.Cleanup(func() { _ = adapter.Close() })
	return worker, adapter, recorder
}

func TestFingerRobotWorkerStrictRealtimeSetupFailureOnlyEmergencyReleases(t *testing.T) {
	setupErr := errors.New("FIFO permission denied")
	initializeCalls := 0
	backend := &pwmFingerRobotBackend{
		initializePWM: func() ([]string, error) {
			initializeCalls++
			return []string{"pwm0"}, nil
		},
	}
	worker := newFingerRobotWorker(backend)
	emergencyCalls := 0
	worker.emergencyRelease = func() error {
		emergencyCalls++
		return nil
	}
	worker.setupRealtime = func(int, int) (fingerRobotRealtimeCleanup, error) {
		return nil, setupErr
	}
	adapter := &fingerRobotWorkerBackend{worker: worker}

	err := adapter.Initialize()
	if !errors.Is(err, setupErr) || !errors.Is(err, ErrFingerRobotUnavailable) {
		t.Fatalf("Initialize error = %v, want setup and unavailable errors", err)
	}
	if initializeCalls != 0 {
		t.Fatalf("hardware initialization ran %d times after realtime setup failure", initializeCalls)
	}
	if emergencyCalls != 1 {
		t.Fatalf("emergency safe release calls = %d, want 1", emergencyCalls)
	}
	if adapter.HardwareReady() {
		t.Fatal("worker reported hardware ready after realtime setup failure")
	}
	if got := worker.threadID(); got != 0 {
		t.Fatalf("stopped worker retained TID %d", got)
	}
}

func TestFingerRobotWorkerSerializesHardwareCommands(t *testing.T) {
	gate := &scriptedFingerRobotWorkerGate{connected: true}
	_, adapter, recorder := newTestFingerRobotWorker(t, gate)

	if err := adapter.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	const calls = 12
	errs := make(chan error, calls)
	var wg sync.WaitGroup
	for i := 0; i < calls; i++ {
		pulse := 1200 + i
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- adapter.SetPulseUS(pulse)
		}()
	}
	wgDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(wgDone)
	}()
	select {
	case <-wgDone:
	case <-time.After(time.Second):
		t.Fatal("serialized SetPulseUS calls did not complete")
	}
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("serialized SetPulseUS: %v", err)
		}
	}

	got := recorder.snapshot()
	if len(got) != calls {
		t.Fatalf("PWM writes = %d, want %d (%v)", len(got), calls, got)
	}
	if gate.calls() != calls {
		t.Fatalf("GPIO waits = %d, want %d", gate.calls(), calls)
	}
}

func TestFingerRobotWorkerCancellationStopsInFlightGPIOWait(t *testing.T) {
	waitStarted := make(chan struct{}, 1)
	waitDone := make(chan struct{})
	gate := &scriptedFingerRobotWorkerGate{
		waitStart: waitStarted,
		waitDone:  waitDone,
		connected: true,
	}
	_, adapter, recorder := newTestFingerRobotWorker(t, gate)
	var unblock sync.Once
	closeWaitDone := func() { unblock.Do(func() { close(waitDone) }) }
	t.Cleanup(closeWaitDone)
	if err := adapter.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { result <- adapter.SetPulseUSContext(ctx, 1900) }()
	select {
	case <-waitStarted:
	case <-time.After(time.Second):
		t.Fatal("worker did not enter GPIO wait")
	}
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled SetPulseUS error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled GPIO wait did not return")
	}

	if got := recorder.snapshot(); len(got) != 0 {
		t.Fatalf("cancelled operation wrote PWM values %v", got)
	}
	closeWaitDone()
}

func TestFingerRobotWorkerUngatedPulseBypassesFeedbackGate(t *testing.T) {
	gate := &scriptedFingerRobotWorkerGate{waitErr: errors.New("no falling edge")}
	_, adapter, recorder := newTestFingerRobotWorker(t, gate)

	if err := adapter.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if err := adapter.SetPulseUSUngated(FingerRobotFactoryZeroUS); err != nil {
		t.Fatalf("SetPulseUSUngated: %v", err)
	}
	if got := gate.calls(); got != 0 {
		t.Fatalf("ungated worker pulse waited for feedback %d times", got)
	}
	if got := recorder.snapshot(); !reflect.DeepEqual(got, []int{FingerRobotFactoryZeroUS}) {
		t.Fatalf("ungated worker pulses = %v, want factory zero", got)
	}
}

func TestWaitFingerRobotOnBackendDrainsFeedback(t *testing.T) {
	drainStarted := make(chan struct{}, 1)
	gate := &scriptedFingerRobotWorkerGate{drainStarted: drainStarted}
	backend := &pwmFingerRobotBackend{lowGate: gate}
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { result <- waitFingerRobotOnBackend(ctx, time.Hour, backend) }()

	select {
	case <-drainStarted:
		cancel()
	case <-time.After(time.Second):
		cancel()
		t.Fatal("timed wait did not drain GPIO feedback")
	}
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled timed wait error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled timed wait did not return")
	}
}

func TestFingerRobotControllerWorkerAdapterRunsActionOnSingleOwner(t *testing.T) {
	gate := &scriptedFingerRobotWorkerGate{connected: true}
	_, adapter, recorder := newTestFingerRobotWorker(t, gate)
	controller := newFingerRobotController(adapter)
	t.Cleanup(controller.cancel)

	if err := controller.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- controller.Press(FingerRobotPressRequest{
			ProfileID: FingerRobotArmProfileShortID,
			Strength:  1,
			PressMS:   100,
			Speed:     FingerRobotSpeedFast,
		})
	}()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Press: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Press did not complete")
	}
	status := controller.Status()
	if status.Running || status.Result != FingerRobotResultSuccess || status.State != FingerRobotStateIdle {
		t.Fatalf("unexpected completed status: %+v", status)
	}
	got := recorder.snapshot()
	if len(got) < 3 || got[len(got)-1] != FingerRobotSafeReleaseUS {
		t.Fatalf("worker adapter pulse sequence = %v, want final safe release", got)
	}
}

func TestFingerRobotControllerWorkerAdapterReleasePreemptsBoundaryWait(t *testing.T) {
	waitStarted := make(chan struct{}, 1)
	waitDone := make(chan struct{})
	gate := &scriptedFingerRobotWorkerGate{connected: true}
	_, adapter, recorder := newTestFingerRobotWorker(t, gate)
	controller := newFingerRobotController(adapter)
	t.Cleanup(controller.cancel)
	var unblock sync.Once
	closeWaitDone := func() { unblock.Do(func() { close(waitDone) }) }
	t.Cleanup(closeWaitDone)
	if err := controller.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	gate.blockNextWait(waitStarted, waitDone)

	pressDone := make(chan error, 1)
	go func() {
		pressDone <- controller.Press(FingerRobotPressRequest{
			ProfileID: FingerRobotArmProfileShortID,
			Strength:  1,
			PressMS:   100,
			Speed:     FingerRobotSpeedFast,
		})
	}()
	select {
	case <-waitStarted:
	case <-time.After(time.Second):
		t.Fatal("press did not enter the worker GPIO wait")
	}

	releaseDone := make(chan error, 1)
	go func() { releaseDone <- controller.Release() }()
	select {
	case err := <-releaseDone:
		if err != nil {
			t.Fatalf("Release: %v", err)
		}
	case <-time.After(time.Second):
		closeWaitDone()
		t.Fatal("Release did not complete")
	}
	select {
	case err := <-pressDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("preempted Press error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("preempted Press did not complete")
	}

	got := recorder.snapshot()
	if len(got) == 0 || got[len(got)-1] != FingerRobotSafeReleaseUS {
		t.Fatalf("preempted pulse sequence = %v, want final safe release", got)
	}
	closeWaitDone()
}
