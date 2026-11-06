package vm

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	fingerRobotWorkerCPU      = 1
	fingerRobotWorkerPriority = 50
	fingerRobotWorkerPoll     = 10 * time.Millisecond
)

var errFingerRobotWorkerStopped = errors.New("finger robot realtime worker is stopped")

// fingerRobotWorkerCommand is deliberately passed over an unbuffered channel.
// There is never a stale hardware command waiting behind a cancelled action;
// once a command is accepted, the caller waits for its result so shutdown and
// finalizer ordering remain explicit.
type fingerRobotWorkerCommand struct {
	ctx    context.Context
	run    func() error
	stop   bool
	result chan error
}

type fingerRobotWorker struct {
	backend *pwmFingerRobotBackend
	// setupRealtime is injectable for deterministic unit tests. Production
	// workers use setupFingerRobotRealtime, which must run after LockOSThread.
	setupRealtime    func(cpu, priority int) (fingerRobotRealtimeCleanup, error)
	emergencyRelease func() error

	lifecycleMu sync.Mutex
	started     bool
	closed      bool
	commands    chan fingerRobotWorkerCommand
	urgent      chan fingerRobotWorkerCommand
	done        chan struct{}
	lastErr     error

	ready atomic.Bool
	tid   atomic.Int64
}

// fingerRobotWorkerBackend is the production adapter exposed to the
// controller. The old pwmFingerRobotBackend remains a useful low-level fake
// and test fixture; only this adapter owns the real global controller.
type fingerRobotWorkerBackend struct {
	worker *fingerRobotWorker
}

func newFingerRobotWorkerBackend() *fingerRobotWorkerBackend {
	backend := newPWMFingerRobotBackend()
	worker := newFingerRobotWorker(backend)
	worker.emergencyRelease = forceExistingFingerRobotSafeRelease
	return &fingerRobotWorkerBackend{worker: worker}
}

func newFingerRobotWorker(backend *pwmFingerRobotBackend) *fingerRobotWorker {
	return &fingerRobotWorker{
		backend:          backend,
		setupRealtime:    setupFingerRobotRealtime,
		emergencyRelease: func() error { return nil },
	}
}

// start creates the locked realtime OS thread lazily. This keeps package
// initialization and unit tests from requiring GPIO devices or CAP_SYS_NICE;
// hardware is touched only when the settings switch explicitly enables it.
func (w *fingerRobotWorker) start() error {
	w.lifecycleMu.Lock()
	defer w.lifecycleMu.Unlock()

	if w.closed {
		return ErrFingerRobotClosed
	}
	if w.started {
		return nil
	}

	commands := make(chan fingerRobotWorkerCommand)
	urgent := make(chan fingerRobotWorkerCommand)
	done := make(chan struct{})
	ready := make(chan error, 1)
	w.started = true
	w.commands = commands
	w.urgent = urgent
	w.done = done
	w.lastErr = nil
	w.ready.Store(false)

	go w.run(commands, urgent, done, ready)
	if err := <-ready; err != nil {
		w.started = false
		w.commands = nil
		w.urgent = nil
		w.done = nil
		w.lastErr = err
		w.ready.Store(false)
		return err
	}
	return nil
}

func (w *fingerRobotWorker) run(
	commands <-chan fingerRobotWorkerCommand,
	urgent <-chan fingerRobotWorkerCommand,
	done chan struct{},
	ready chan<- error,
) {
	runtime.LockOSThread()

	setupRealtime := w.setupRealtime
	if setupRealtime == nil {
		setupRealtime = setupFingerRobotRealtime
	}
	cleanup, err := setupRealtime(fingerRobotWorkerCPU, fingerRobotWorkerPriority)
	if err != nil {
		if w.emergencyRelease != nil {
			if releaseErr := w.emergencyRelease(); releaseErr != nil {
				err = errors.Join(err, fmt.Errorf("emergency safe release after realtime setup failure: %w", releaseErr))
			}
		}
		// setupRealtime already attempted rollback. Keep the goroutine locked
		// while it returns so Go destroys this OS thread instead of ever reusing
		// a thread whose partial affinity/scheduler rollback may have failed.
		ready <- fmt.Errorf("start finger robot realtime worker: %w", err)
		close(done)
		w.markStopped(done, err)
		return
	}

	w.tid.Store(int64(getFingerRobotRealtimeThreadID()))
	if diagnostic, diagnosticErr := getFingerRobotRealtimeDiagnostic(); diagnosticErr != nil {
		log.Warnf("finger robot realtime worker started but diagnostic read failed: %v", diagnosticErr)
	} else {
		log.Infof(
			"finger robot realtime worker ready: tid=%d cpu=%v policy=%d priority=%d",
			diagnostic.TID,
			diagnostic.CPUs,
			diagnostic.Policy,
			diagnostic.Priority,
		)
	}

	defer func() {
		cleanupErr := cleanup()
		if cleanupErr != nil {
			log.Errorf("failed to restore finger robot worker thread attributes: %v", cleanupErr)
			// Deliberately do not UnlockOSThread. Per runtime.LockOSThread's
			// contract, returning while locked terminates this OS thread, so FIFO
			// or CPU1 attributes cannot leak to an unrelated goroutine.
		} else {
			runtime.UnlockOSThread()
		}
		w.ready.Store(false)
		w.tid.Store(0)
		close(done)
		w.markStopped(done, nil)
	}()

	ready <- nil
	idleTicker := time.NewTicker(fingerRobotWorkerPoll)
	defer idleTicker.Stop()

	for {
		// Prefer urgent commands whenever one is already waiting. A command
		// currently inside a cancellable GPIO wait is interrupted through its
		// context; once it returns the urgent finalizer wins before normal work.
		select {
		case command := <-urgent:
			if w.executeCommand(command) {
				return
			}
		default:
			select {
			case command := <-urgent:
				if w.executeCommand(command) {
					return
				}
			case command := <-commands:
				if w.executeCommand(command) {
					return
				}
			case <-idleTicker.C:
				w.drainGPIOEvents()
			}
		}
	}
}

func (w *fingerRobotWorker) executeCommand(command fingerRobotWorkerCommand) bool {
	if command.ctx != nil {
		if err := command.ctx.Err(); err != nil {
			command.result <- err
			return command.stop
		}
	}
	err := command.run()
	command.result <- err
	return command.stop
}

func (w *fingerRobotWorker) markStopped(done chan struct{}, runErr error) {
	w.lifecycleMu.Lock()
	defer w.lifecycleMu.Unlock()
	if w.done != done {
		return
	}
	w.started = false
	w.commands = nil
	w.urgent = nil
	w.done = nil
	if runErr != nil {
		w.lastErr = runErr
	}
}

func (w *fingerRobotWorker) drainGPIOEvents() {
	if w.backend == nil {
		return
	}
	w.backend.mu.Lock()
	gate := w.backend.lowGate
	w.backend.mu.Unlock()
	drainer, ok := gate.(fingerRobotPWMLowGateDrainer)
	if !ok {
		return
	}
	if err := drainer.DrainEvents(); err != nil {
		// A transient EAGAIN is handled by the gate. Other errors are logged;
		// the next duty wait will propagate the same failure through the normal
		// backend safety path.
		log.Warnf("finger robot GPIO event drain failed: %v", err)
	}
}

func (w *fingerRobotWorker) submit(
	ctx context.Context,
	run func() error,
	isUrgent bool,
	stop bool,
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	w.lifecycleMu.Lock()
	if !w.started {
		err := w.lastErr
		w.lifecycleMu.Unlock()
		if err != nil {
			return errors.Join(errFingerRobotWorkerStopped, err)
		}
		return errFingerRobotWorkerStopped
	}
	commands := w.commands
	urgent := w.urgent
	done := w.done
	w.lifecycleMu.Unlock()

	command := fingerRobotWorkerCommand{
		ctx:    ctx,
		run:    run,
		stop:   stop,
		result: make(chan error, 1),
	}
	queue := commands
	if isUrgent {
		queue = urgent
	}
	select {
	case queue <- command:
		// Once accepted, always collect the result. Returning solely on ctx.Done
		// could leave a hardware command executing after its caller has moved on.
		return <-command.result
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return errFingerRobotWorkerStopped
	}
}

func (w *fingerRobotWorker) close(ctx context.Context) error {
	w.lifecycleMu.Lock()
	started := w.started
	done := w.done
	w.lifecycleMu.Unlock()
	if !started {
		return nil
	}
	err := w.submit(ctx, func() error {
		if w.backend == nil {
			return nil
		}
		return w.backend.Close()
	}, true, true)
	// Do not let a concurrent Initialize observe the worker between the stop
	// command's result and the deferred scheduler/affinity cleanup.
	if done != nil {
		<-done
	}
	return err
}

func (w *fingerRobotWorker) connectionAlive(maxSilence time.Duration) bool {
	w.lifecycleMu.Lock()
	started := w.started
	w.lifecycleMu.Unlock()
	if !started || w.backend == nil {
		return false
	}
	return w.backend.ConnectionAlive(maxSilence)
}

func (w *fingerRobotWorker) threadID() int {
	return int(w.tid.Load())
}

// Initialize is the only operation which starts the worker. All subsequent
// hardware calls are strictly routed through its command owner.
func (b *fingerRobotWorkerBackend) Initialize() error {
	if b == nil || b.worker == nil {
		return ErrFingerRobotUnavailable
	}
	if err := b.worker.start(); err != nil {
		return errors.Join(ErrFingerRobotUnavailable, err)
	}
	err := b.worker.submit(context.Background(), b.worker.backend.Initialize, false, false)
	b.worker.ready.Store(err == nil && b.worker.backend.HardwareReady())
	return err
}

func (b *fingerRobotWorkerBackend) SetPulseUS(pulseUS int) error {
	return b.SetPulseUSContext(context.Background(), pulseUS)
}

func (b *fingerRobotWorkerBackend) SetPulseUSContext(ctx context.Context, pulseUS int) error {
	if b == nil || b.worker == nil {
		return ErrFingerRobotUnavailable
	}
	err := b.worker.submit(ctx, func() error {
		return b.worker.backend.SetPulseUSContext(ctx, pulseUS)
	}, false, false)
	b.worker.ready.Store(b.worker.backend.HardwareReady())
	return err
}

// SetPulseUSUngated routes the explicit Debug factory-zero write through the
// same realtime worker while intentionally bypassing GPIO feedback gating.
func (b *fingerRobotWorkerBackend) SetPulseUSUngated(pulseUS int) error {
	if b == nil || b.worker == nil {
		return ErrFingerRobotUnavailable
	}
	err := b.worker.submit(context.Background(), func() error {
		return b.worker.backend.SetPulseUSUngated(pulseUS)
	}, true, false)
	b.worker.ready.Store(b.worker.backend.HardwareReady())
	return err
}

// ExecuteFingerRobotAction runs the caller-supplied complete action on the
// worker thread with the raw backend. The controller uses this hook for
// production Press/Debug operations so ramp timers, GPIO waits, and duty
// writes share one scheduling owner instead of crossing the worker boundary
// once per 20ms step.
func (b *fingerRobotWorkerBackend) ExecuteFingerRobotAction(
	ctx context.Context,
	action func(fingerRobotBackend) error,
) error {
	if b == nil || b.worker == nil || action == nil {
		return ErrFingerRobotUnavailable
	}
	err := b.worker.submit(ctx, func() error {
		return action(b.worker.backend)
	}, false, false)
	b.worker.ready.Store(b.worker.backend.HardwareReady())
	return err
}

func (b *fingerRobotWorkerBackend) HardwareReady() bool {
	return b != nil && b.worker != nil && b.worker.ready.Load()
}

func (b *fingerRobotWorkerBackend) WaitConnected(timeout time.Duration) error {
	if b == nil || b.worker == nil {
		return ErrFingerRobotUnavailable
	}
	err := b.worker.submit(context.Background(), func() error {
		return b.worker.backend.WaitConnected(timeout)
	}, false, false)
	b.worker.ready.Store(b.worker.backend.HardwareReady())
	return err
}

func (b *fingerRobotWorkerBackend) ConnectionAlive(maxSilence time.Duration) bool {
	if b == nil || b.worker == nil {
		return false
	}
	return b.worker.connectionAlive(maxSilence)
}

func (b *fingerRobotWorkerBackend) ForceSafeRelease() error {
	if b == nil || b.worker == nil {
		return ErrFingerRobotUnavailable
	}
	err := b.worker.submit(context.Background(), b.worker.backend.ForceSafeRelease, true, false)
	b.worker.ready.Store(b.worker.backend.HardwareReady())
	return err
}

func (b *fingerRobotWorkerBackend) RestoreShutdownPins() error {
	if b == nil || b.worker == nil {
		return nil
	}
	err := b.worker.submit(context.Background(), b.worker.backend.RestoreShutdownPins, true, false)
	b.worker.ready.Store(b.worker.backend.HardwareReady())
	return err
}

func (b *fingerRobotWorkerBackend) Close() error {
	if b == nil || b.worker == nil {
		return nil
	}
	// Mark the adapter closed before submitting the stop command. This closes
	// the small concurrent Initialize/Close window in which a new caller could
	// otherwise observe started=true while the worker was already draining its
	// final command.
	b.worker.lifecycleMu.Lock()
	if b.worker.closed {
		b.worker.lifecycleMu.Unlock()
		return nil
	}
	b.worker.closed = true
	b.worker.lifecycleMu.Unlock()
	err := b.worker.close(context.Background())
	b.worker.ready.Store(false)
	return err
}
