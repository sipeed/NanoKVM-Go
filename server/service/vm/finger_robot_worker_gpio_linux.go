//go:build linux

package vm

// This file contains the only production GPIO capture implementation. The
// GPIO event fd is drained by the same locked OS thread that writes PWM.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/warthog618/go-gpiocdev"
	"github.com/warthog618/go-gpiocdev/uapi"
	"golang.org/x/sys/unix"
)

const fingerRobotGPIOPollQuantum = 5 * time.Millisecond

// epollFingerRobotPWMLowGate owns the requested GPIO event fd.  Its event
// methods are called by the worker thread; Connected is also safe to call from
// the controller's health-monitor goroutine.
type epollFingerRobotPWMLowGate struct {
	eventFD int
	epollFD int

	mu              sync.RWMutex
	closed          bool
	closeOnce       sync.Once
	closeErr        error
	edgeSequence    uint64
	lastFallingEdge time.Time
	now             func() time.Time
	readKernelEvent func() (uapi.EventData, error)
	readLineValue   func() (int, error)
	waitEpoll       func([]unix.EpollEvent, int) (int, error)
	closeFD         func(int) error

	epollEvents []unix.EpollEvent
}

func newRealtimeFingerRobotPWMLowGate() (fingerRobotPWMLowGate, error) {
	eventFD, err := openFingerRobotGPIOEventFD()
	if err != nil {
		return nil, err
	}
	epollFD, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		_ = unix.Close(eventFD)
		return nil, fmt.Errorf("create finger robot GPIO epoll: %w", err)
	}
	event := &unix.EpollEvent{Events: unix.EPOLLIN, Fd: int32(eventFD)}
	if err := unix.EpollCtl(epollFD, unix.EPOLL_CTL_ADD, eventFD, event); err != nil {
		_ = unix.Close(epollFD)
		_ = unix.Close(eventFD)
		return nil, fmt.Errorf("register finger robot GPIO event fd: %w", err)
	}
	gate := &epollFingerRobotPWMLowGate{
		eventFD:     eventFD,
		epollFD:     epollFD,
		now:         time.Now,
		epollEvents: make([]unix.EpollEvent, 1),
		closeFD:     unix.Close,
	}
	gate.readKernelEvent = func() (uapi.EventData, error) {
		return uapi.ReadEvent(uintptr(gate.eventFD))
	}
	gate.readLineValue = func() (int, error) {
		var values uapi.HandleData
		if err := uapi.GetLineValues(uintptr(gate.eventFD), &values); err != nil {
			return 0, err
		}
		return int(values[0]), nil
	}
	gate.waitEpoll = func(events []unix.EpollEvent, timeout int) (int, error) {
		return unix.EpollWait(gate.epollFD, events, timeout)
	}
	return gate, nil
}

// openFingerRobotGPIOEventFD requests the legacy v1 GPIO event interface
// directly.  Using gpiocdev.WithEventHandler here would create a library-owned
// watcher goroutine and put the edge-to-user-space handoff back on the normal
// scheduler.
func openFingerRobotGPIOEventFD() (int, error) {
	labelData, err := os.ReadFile(fingerRobotGPIOChipLabelPath)
	if err != nil {
		return -1, fmt.Errorf("read %s: %w", fingerRobotGPIOChipLabelPath, err)
	}
	wantedLabel := strings.TrimSpace(string(labelData))
	if wantedLabel == "" {
		return -1, fmt.Errorf("read %s: empty GPIO chip label", fingerRobotGPIOChipLabelPath)
	}

	var lookupErrors []error
	for _, chipName := range gpiocdev.Chips() {
		chipPath := filepath.Join("/dev", chipName)
		chipFile, openErr := os.OpenFile(chipPath, os.O_RDONLY|unix.O_CLOEXEC, 0)
		if openErr != nil {
			lookupErrors = append(lookupErrors, fmt.Errorf("open %s: %w", chipPath, openErr))
			continue
		}
		chipInfo, infoErr := uapi.GetChipInfo(chipFile.Fd())
		if infoErr != nil {
			_ = chipFile.Close()
			lookupErrors = append(lookupErrors, fmt.Errorf("read %s info: %w", chipPath, infoErr))
			continue
		}
		label := strings.TrimSpace(uapi.BytesToString(chipInfo.Label[:]))
		if label != wantedLabel {
			_ = chipFile.Close()
			continue
		}

		request := uapi.EventRequest{
			Offset:      fingerRobotGPIOLineOffset,
			HandleFlags: uapi.HandleRequestInput,
			EventFlags:  uapi.EventRequestFallingEdge,
		}
		copy(request.Consumer[:], fingerRobotGPIOConsumer)
		if requestErr := uapi.GetLineEvent(chipFile.Fd(), &request); requestErr != nil {
			_ = chipFile.Close()
			return -1, fmt.Errorf(
				"request %s line %d (GPIO35): %w",
				chipName,
				fingerRobotGPIOLineOffset,
				requestErr,
			)
		}
		if closeErr := chipFile.Close(); closeErr != nil {
			_ = unix.Close(int(request.Fd))
			return -1, fmt.Errorf("close %s after requesting GPIO line: %w", chipName, closeErr)
		}

		eventFD := int(request.Fd)
		unix.CloseOnExec(eventFD)
		if err := unix.SetNonblock(eventFD, true); err != nil {
			_ = unix.Close(eventFD)
			return -1, fmt.Errorf("set finger robot GPIO event fd nonblocking: %w", err)
		}
		return eventFD, nil
	}

	return -1, errors.Join(
		fmt.Errorf("GPIO chip with sysfs label %q was not found in /dev", wantedLabel),
		errors.Join(lookupErrors...),
	)
}

func (g *epollFingerRobotPWMLowGate) clockNow() time.Time {
	if g != nil && g.now != nil {
		return g.now()
	}
	return time.Now()
}

func (g *epollFingerRobotPWMLowGate) recordFallingEdge() {
	g.mu.Lock()
	g.edgeSequence++
	g.lastFallingEdge = g.clockNow()
	g.mu.Unlock()
}

func (g *epollFingerRobotPWMLowGate) edgeSnapshot() uint64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.edgeSequence
}

func (g *epollFingerRobotPWMLowGate) readEvent() (bool, error) {
	if g.readKernelEvent == nil {
		return false, errors.New("finger robot GPIO event reader is not initialized")
	}
	for {
		event, err := g.readKernelEvent()
		if err != nil {
			if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EWOULDBLOCK) {
				return false, nil
			}
			if errors.Is(err, unix.EINTR) {
				continue
			}
			return false, err
		}
		if event.ID != uapi.EventRequestFallingEdge {
			continue
		}
		g.recordFallingEdge()
		return true, nil
	}
}

// drainEvents must run on the worker thread.  It consumes every queued kernel
// event, keeping the connection-health timestamp current even while no duty
// update is waiting for a boundary.
func (g *epollFingerRobotPWMLowGate) drainEvents() error {
	for {
		falling, err := g.readEvent()
		if err != nil {
			return fmt.Errorf("read finger robot GPIO capture event: %w", err)
		}
		if !falling {
			return nil
		}
	}
}

func (g *epollFingerRobotPWMLowGate) DrainEvents() error {
	return g.drainEvents()
}

func (g *epollFingerRobotPWMLowGate) currentValue() (int, error) {
	if g.readLineValue == nil {
		return 0, errors.New("finger robot GPIO value reader is not initialized")
	}
	return g.readLineValue()
}

func (g *epollFingerRobotPWMLowGate) WaitLow(timeout time.Duration) error {
	return g.WaitLowContext(context.Background(), timeout)
}

func (g *epollFingerRobotPWMLowGate) WaitLowContext(ctx context.Context, timeout time.Duration) error {
	if g == nil {
		return errors.New("finger robot GPIO capture is not initialized")
	}
	if timeout <= 0 {
		return errors.New("finger robot GPIO low wait timeout must be positive")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	// Remove events that occurred before this duty update.  A PWM update is
	// allowed only after a fresh falling edge, preserving the original gate
	// contract while keeping all edge handling on this thread.
	if err := g.drainEvents(); err != nil {
		return err
	}
	seenSequence := g.edgeSnapshot()
	deadline := g.clockNow().Add(timeout)

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		remaining := deadline.Sub(g.clockNow())
		if remaining <= 0 {
			// Give an event which became ready exactly at the boundary one final
			// chance before reporting timeout.
			if err := g.drainEvents(); err != nil {
				return err
			}
			if g.edgeSnapshot() > seenSequence {
				if value, err := g.currentValue(); err != nil {
					return fmt.Errorf("read finger robot GPIO capture line %d: %w", fingerRobotGPIOLineOffset, err)
				} else if value == 0 {
					return nil
				}
				seenSequence = g.edgeSnapshot()
			}
			return fmt.Errorf(
				"timed out after %s waiting for finger robot GPIO capture line %d to become low: %w",
				timeout,
				fingerRobotGPIOLineOffset,
				errFingerRobotGPIOWaitTimeout,
			)
		}

		waitFor := remaining
		if waitFor > fingerRobotGPIOPollQuantum {
			waitFor = fingerRobotGPIOPollQuantum
		}
		// Round up so a sub-millisecond remainder is not converted to an
		// immediate zero-timeout epoll call.
		epollTimeout := int((waitFor + time.Millisecond - 1) / time.Millisecond)
		if epollTimeout < 1 {
			epollTimeout = 1
		}
		if g.waitEpoll == nil {
			return errors.New("finger robot GPIO epoll waiter is not initialized")
		}
		n, err := g.waitEpoll(g.epollEvents, epollTimeout)
		if err != nil {
			if err == unix.EINTR {
				continue
			}
			return fmt.Errorf("wait for finger robot GPIO capture event: %w", err)
		}
		if n == 0 {
			continue
		}

		// Drain all events reported in this wakeup and any events queued behind
		// them.  The fresh sequence is checked after each falling edge so a
		// high-level recheck cannot accidentally consume a later edge.
		for i := 0; i < n; i++ {
			if g.epollEvents[i].Fd != int32(g.eventFD) {
				continue
			}
			if g.epollEvents[i].Events&(unix.EPOLLERR|unix.EPOLLHUP) != 0 {
				return fmt.Errorf(
					"finger robot GPIO capture fd became invalid: events=0x%x",
					g.epollEvents[i].Events,
				)
			}
			for {
				falling, readErr := g.readEvent()
				if readErr != nil {
					return fmt.Errorf("read finger robot GPIO capture event: %w", readErr)
				}
				if !falling {
					break
				}
				sequence := g.edgeSnapshot()
				if sequence <= seenSequence {
					continue
				}
				seenSequence = sequence
				value, readErr := g.currentValue()
				if readErr != nil {
					return fmt.Errorf("read finger robot GPIO capture line %d: %w", fingerRobotGPIOLineOffset, readErr)
				}
				if value == 0 {
					return nil
				}
			}
		}
	}
}

func (g *epollFingerRobotPWMLowGate) Connected(maxSilence time.Duration) bool {
	if g == nil || maxSilence <= 0 {
		return false
	}
	g.mu.RLock()
	last := g.lastFallingEdge
	closed := g.closed
	g.mu.RUnlock()
	if closed || last.IsZero() {
		return false
	}
	now := g.clockNow()
	if now.Before(last) {
		return false
	}
	return now.Sub(last) <= maxSilence
}

func (g *epollFingerRobotPWMLowGate) Close() error {
	if g == nil {
		return nil
	}
	g.closeOnce.Do(func() {
		g.mu.Lock()
		g.closed = true
		g.mu.Unlock()
		var errs []error
		closeFD := g.closeFD
		if closeFD == nil {
			closeFD = unix.Close
		}
		if g.epollFD >= 0 {
			if err := closeFD(g.epollFD); err != nil {
				errs = append(errs, err)
			}
			g.epollFD = -1
		}
		if g.eventFD >= 0 {
			if err := closeFD(g.eventFD); err != nil {
				errs = append(errs, err)
			}
			g.eventFD = -1
		}
		g.closeErr = errors.Join(errs...)
	})
	return g.closeErr
}
