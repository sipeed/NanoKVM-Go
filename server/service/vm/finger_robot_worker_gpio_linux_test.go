//go:build linux

package vm

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/warthog618/go-gpiocdev/uapi"
	"golang.org/x/sys/unix"
)

type fingerRobotGPIOEventResult struct {
	event uapi.EventData
	err   error
}

type scriptedFingerRobotGPIOIO struct {
	events     []fingerRobotGPIOEventResult
	values     []int
	valueErr   error
	valueCalls int
	waitCalls  int
	closedFDs  []int
	closeErr   error
}

func (s *scriptedFingerRobotGPIOIO) enqueueFallingEdge() {
	s.events = append(s.events, fingerRobotGPIOEventResult{
		event: uapi.EventData{ID: uapi.EventRequestFallingEdge},
	})
}

func (s *scriptedFingerRobotGPIOIO) readEvent() (uapi.EventData, error) {
	if len(s.events) == 0 {
		return uapi.EventData{}, unix.EAGAIN
	}
	result := s.events[0]
	s.events = s.events[1:]
	return result.event, result.err
}

func (s *scriptedFingerRobotGPIOIO) readValue() (int, error) {
	s.valueCalls++
	if s.valueErr != nil {
		return 0, s.valueErr
	}
	if len(s.values) == 0 {
		return 0, nil
	}
	value := s.values[0]
	s.values = s.values[1:]
	return value, nil
}

func newScriptedFingerRobotEpollGate(now *time.Time, io *scriptedFingerRobotGPIOIO) *epollFingerRobotPWMLowGate {
	gate := &epollFingerRobotPWMLowGate{
		eventFD:         7,
		epollFD:         8,
		epollEvents:     make([]unix.EpollEvent, 1),
		readKernelEvent: io.readEvent,
		readLineValue:   io.readValue,
		closeFD: func(fd int) error {
			io.closedFDs = append(io.closedFDs, fd)
			return io.closeErr
		},
	}
	gate.now = func() time.Time { return *now }
	gate.waitEpoll = func([]unix.EpollEvent, int) (int, error) {
		io.waitCalls++
		*now = now.Add(time.Millisecond)
		return 0, nil
	}
	return gate
}

func TestRealtimeFingerRobotPWMLowGateRejectsStaleEdge(t *testing.T) {
	now := time.Unix(100, 0)
	io := &scriptedFingerRobotGPIOIO{values: []int{0}}
	io.enqueueFallingEdge()
	gate := newScriptedFingerRobotEpollGate(&now, io)

	err := gate.WaitLowContext(context.Background(), 3*time.Millisecond)
	if !errors.Is(err, errFingerRobotGPIOWaitTimeout) {
		t.Fatalf("WaitLowContext error = %v, want GPIO timeout", err)
	}
	if io.valueCalls != 0 {
		t.Fatalf("stale edge caused %d GPIO value reads", io.valueCalls)
	}
}

func TestRealtimeFingerRobotPWMLowGateWritesOnlyOnFreshLowEdge(t *testing.T) {
	now := time.Unix(100, 0)
	io := &scriptedFingerRobotGPIOIO{values: []int{1, 0}}
	gate := newScriptedFingerRobotEpollGate(&now, io)
	gate.waitEpoll = func(events []unix.EpollEvent, _ int) (int, error) {
		io.waitCalls++
		io.enqueueFallingEdge()
		events[0] = unix.EpollEvent{Fd: int32(gate.eventFD), Events: unix.EPOLLIN}
		return 1, nil
	}

	if err := gate.WaitLowContext(context.Background(), time.Second); err != nil {
		t.Fatalf("WaitLowContext: %v", err)
	}
	if io.valueCalls != 2 || io.waitCalls != 2 {
		t.Fatalf("value reads=%d epoll waits=%d, want 2 and 2", io.valueCalls, io.waitCalls)
	}
}

func TestRealtimeFingerRobotPWMLowGateBoundaryEdgeBeatsTimeout(t *testing.T) {
	now := time.Unix(100, 0)
	io := &scriptedFingerRobotGPIOIO{values: []int{0}}
	gate := newScriptedFingerRobotEpollGate(&now, io)
	gate.waitEpoll = func(_ []unix.EpollEvent, _ int) (int, error) {
		io.waitCalls++
		now = now.Add(11 * time.Millisecond)
		io.enqueueFallingEdge()
		return 0, nil
	}

	if err := gate.WaitLowContext(context.Background(), 10*time.Millisecond); err != nil {
		t.Fatalf("boundary falling edge lost to timeout: %v", err)
	}
	if io.valueCalls != 1 {
		t.Fatalf("boundary edge GPIO reads = %d, want 1", io.valueCalls)
	}
}

func TestRealtimeFingerRobotPWMLowGateCancellationAndReadError(t *testing.T) {
	now := time.Unix(100, 0)
	io := &scriptedFingerRobotGPIOIO{}
	gate := newScriptedFingerRobotEpollGate(&now, io)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := gate.WaitLowContext(ctx, time.Second); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled WaitLowContext error = %v", err)
	}

	readErr := errors.New("GPIO event read failed")
	io.events = []fingerRobotGPIOEventResult{{err: readErr}}
	if err := gate.DrainEvents(); !errors.Is(err, readErr) {
		t.Fatalf("DrainEvents error = %v, want %v", err, readErr)
	}
}

func TestRealtimeFingerRobotPWMLowGateReportsLineReadError(t *testing.T) {
	now := time.Unix(100, 0)
	readErr := errors.New("GPIO line read failed")
	io := &scriptedFingerRobotGPIOIO{valueErr: readErr}
	gate := newScriptedFingerRobotEpollGate(&now, io)
	gate.waitEpoll = func(events []unix.EpollEvent, _ int) (int, error) {
		io.enqueueFallingEdge()
		events[0] = unix.EpollEvent{Fd: int32(gate.eventFD), Events: unix.EPOLLIN}
		return 1, nil
	}

	if err := gate.WaitLowContext(context.Background(), time.Second); !errors.Is(err, readErr) {
		t.Fatalf("WaitLowContext error = %v, want %v", err, readErr)
	}
	if io.valueCalls != 1 {
		t.Fatalf("GPIO value reads = %d, want 1", io.valueCalls)
	}
}

func TestRealtimeFingerRobotPWMLowGateConnectionHealthAndClose(t *testing.T) {
	now := time.Unix(100, 0)
	closeErr := errors.New("close failed")
	io := &scriptedFingerRobotGPIOIO{closeErr: closeErr}
	io.enqueueFallingEdge()
	gate := newScriptedFingerRobotEpollGate(&now, io)

	if err := gate.DrainEvents(); err != nil {
		t.Fatalf("DrainEvents: %v", err)
	}
	if !gate.Connected(500 * time.Millisecond) {
		t.Fatal("fresh falling edge did not mark feedback connected")
	}
	now = now.Add(501 * time.Millisecond)
	if gate.Connected(500 * time.Millisecond) {
		t.Fatal("expired feedback remained connected")
	}

	if err := gate.Close(); !errors.Is(err, closeErr) {
		t.Fatalf("Close error = %v, want %v", err, closeErr)
	}
	if err := gate.Close(); !errors.Is(err, closeErr) {
		t.Fatalf("second Close error = %v, want cached %v", err, closeErr)
	}
	if !reflect.DeepEqual(io.closedFDs, []int{8, 7}) {
		t.Fatalf("closed fds = %v, want epoll then event fd", io.closedFDs)
	}
}
