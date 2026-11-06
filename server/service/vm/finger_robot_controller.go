package vm

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	FingerRobotFactoryZeroUS       = 1500
	FingerRobotSafeReleaseUS       = 2050
	FingerRobotReleaseIntervalMS   = 500
	FingerRobotDebugLeaseTimeoutMS = 3000

	fingerRobotRampInterval = 20 * time.Millisecond

	fingerRobotConnectionPollInterval  = 100 * time.Millisecond
	fingerRobotConnectionLossTimeout   = 500 * time.Millisecond
	fingerRobotConnectionRetryInterval = 2 * time.Second
)

var (
	ErrFingerRobotBusy         = errors.New("finger robot is busy")
	ErrFingerRobotUnavailable  = errors.New("finger robot is unavailable")
	ErrFingerRobotClosed       = errors.New("finger robot controller is shut down")
	ErrFingerRobotInvalid      = errors.New("invalid finger robot parameters")
	ErrFingerRobotDisconnected = errors.New("finger robot PWM feedback is disconnected")
)

type FingerRobotLimits struct {
	ZeroUS                       int   `json:"zeroUs"`
	ReleaseMinUS                 int   `json:"releaseMinUs"`
	ReleaseMaxUS                 int   `json:"releaseMaxUs"`
	ReleaseDefaultUS             int   `json:"releaseDefaultUs"`
	PressMinUS                   int   `json:"pressMinUs"`
	PressMaxUS                   int   `json:"pressMaxUs"`
	PressDefaultUS               int   `json:"pressDefaultUs"`
	PressSpeedMinUSPerSecond     int   `json:"pressSpeedMinUsPerSecond"`
	PressSpeedMaxUSPerSecond     int   `json:"pressSpeedMaxUsPerSecond"`
	PressSpeedDefaultUSPerSecond int   `json:"pressSpeedDefaultUsPerSecond"`
	CountMin                     int64 `json:"countMin"`
	CountDefault                 int64 `json:"countDefault"`
	CountUnlimitedValue          int64 `json:"countUnlimitedValue"`
	PressMinMS                   int   `json:"pressMinMs"`
	PressMaxMS                   int   `json:"pressMaxMs"`
	PressDefaultMS               int   `json:"pressDefaultMs"`
	ReleaseIntervalMS            int   `json:"releaseIntervalMs"`
	LeaseTimeoutMS               int   `json:"leaseTimeoutMs"`
}

var fingerRobotLimits = FingerRobotLimits{
	ZeroUS:                       FingerRobotFactoryZeroUS,
	ReleaseMinUS:                 889,
	ReleaseMaxUS:                 FingerRobotSafeReleaseUS,
	ReleaseDefaultUS:             FingerRobotSafeReleaseUS,
	PressMinUS:                   889,
	PressMaxUS:                   2000,
	PressDefaultUS:               899,
	PressSpeedMinUSPerSecond:     200,
	PressSpeedMaxUSPerSecond:     2000,
	PressSpeedDefaultUSPerSecond: 2000,
	CountMin:                     1,
	CountDefault:                 50,
	CountUnlimitedValue:          0,
	PressMinMS:                   100,
	PressMaxMS:                   30000,
	PressDefaultMS:               100,
	ReleaseIntervalMS:            FingerRobotReleaseIntervalMS,
	LeaseTimeoutMS:               FingerRobotDebugLeaseTimeoutMS,
}

func GetFingerRobotLimits() FingerRobotLimits {
	return fingerRobotLimits
}

type FingerRobotDebugRequest struct {
	ReleaseUS             int   `json:"releaseUs"`
	PressUS               int   `json:"pressUs"`
	PressSpeedUSPerSecond int   `json:"pressSpeedUsPerSecond"`
	Count                 int64 `json:"count"`
	PressMS               int   `json:"pressMs"`
}

// FingerRobotPressRequest contains only product-level choices. The controller
// resolves the fixed profile pulse and speed internally so normal callers can
// never submit an arbitrary PWM value.
type FingerRobotPressRequest struct {
	ProfileID string
	Strength  int
	PressMS   int
	Speed     FingerRobotSpeed
}

type FingerRobotState string

const (
	FingerRobotStateIdle      FingerRobotState = "idle"
	FingerRobotStateReleasing FingerRobotState = "releasing"
	FingerRobotStatePressing  FingerRobotState = "pressing"
	FingerRobotStateHolding   FingerRobotState = "holding"
	FingerRobotStateResetting FingerRobotState = "resetting"
)

type FingerRobotResult string

const (
	FingerRobotResultIdle      FingerRobotResult = "idle"
	FingerRobotResultRunning   FingerRobotResult = "running"
	FingerRobotResultSuccess   FingerRobotResult = "success"
	FingerRobotResultCancelled FingerRobotResult = "cancelled"
	FingerRobotResultFailed    FingerRobotResult = "failed"
	FingerRobotResultReset     FingerRobotResult = "reset"
	FingerRobotResultShutdown  FingerRobotResult = "shutdown"
)

type FingerRobotStatus struct {
	Running bool              `json:"running"`
	Current int64             `json:"current"`
	Total   int64             `json:"total"`
	State   FingerRobotState  `json:"state"`
	Result  FingerRobotResult `json:"result"`
	Error   string            `json:"error"`
	debug   bool
}

type fingerRobotBackend interface {
	Initialize() error
	SetPulseUS(pulseUS int) error
}

// fingerRobotBackendContext is an optional extension used by the realtime
// backend. A cancelled operation must be able to interrupt GPIO boundary
// waiting before another intermediate duty value can be written. Existing
// test/fake backends intentionally need only the legacy SetPulseUS method.
type fingerRobotBackendContext interface {
	SetPulseUSContext(ctx context.Context, pulseUS int) error
}

// fingerRobotBackendUngated is an optional extension used only by the
// explicit Debug factory-zero action. It writes the PWM duty without waiting
// for a feedback falling edge, so the 1500 us factory-zero output can remain
// active while the servo is disconnected.
type fingerRobotBackendUngated interface {
	SetPulseUSUngated(pulseUS int) error
}

// fingerRobotActionExecutor is implemented only by the production realtime
// backend. It lets the complete press/debug state machine (including ramp and
// hold timers) run on the worker's locked OS thread while fake backends retain
// the original controller-driven path in unit tests.
type fingerRobotActionExecutor interface {
	ExecuteFingerRobotAction(ctx context.Context, action func(fingerRobotBackend) error) error
}

func setFingerRobotPulse(ctx context.Context, backend fingerRobotBackend, pulseUS int) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if contextualBackend, ok := backend.(fingerRobotBackendContext); ok {
		return contextualBackend.SetPulseUSContext(ctx, pulseUS)
	}
	return backend.SetPulseUS(pulseUS)
}

func setFingerRobotPulseUngated(backend fingerRobotBackend, pulseUS int) error {
	if ungatedBackend, ok := backend.(fingerRobotBackendUngated); ok {
		return ungatedBackend.SetPulseUSUngated(pulseUS)
	}
	return backend.SetPulseUS(pulseUS)
}

func isFingerRobotFactoryZeroHeld(status FingerRobotStatus) bool {
	return !status.Running && status.debug && status.Result == FingerRobotResultReset
}

type fingerRobotBackendCloser interface {
	Close() error
}

type fingerRobotBackendShutdownPinRestorer interface {
	RestoreShutdownPins() error
}

// fingerRobotHotplugBackend is implemented by backends whose PWM feedback
// returns through a connector. Monitoring observes the last GPIO edge without
// consuming the notification used to boundary-gate duty writes.
type fingerRobotHotplugBackend interface {
	HardwareReady() bool
	WaitConnected(timeout time.Duration) error
	ConnectionAlive(maxSilence time.Duration) bool
	ForceSafeRelease() error
}

type fingerRobotOperation struct {
	ctx           context.Context
	cancel        context.CancelFunc
	done          chan struct{}
	lease         chan struct{}
	leaseDeadline time.Time
	debug         bool
	releaseUS     int

	// externalFinalizer is protected by FingerRobotController.mu. Stop, Reset,
	// and Shutdown set it before cancellation so the operation does not write an
	// intermediate recovery pulse before the caller performs the final write.
	externalFinalizer bool
	completionErr     error
	recoveryErr       error
}

type FingerRobotController struct {
	mu      sync.Mutex
	resetMu sync.Mutex

	backend fingerRobotBackend
	ctx     context.Context
	cancel  context.CancelFunc

	active    *fingerRobotOperation
	enabled   bool
	available bool
	closed    bool
	resetting bool

	// initializationAttempted distinguishes a controller that has never touched
	// PWM hardware from one whose initialization only partly succeeded. Disable
	// and Shutdown must never initialize hardware merely to make it safe, while
	// an already-touched backend can still receive a best-effort release pulse.
	initializationAttempted bool
	status                  FingerRobotStatus

	debugLeaseTimeout time.Duration

	connectionMonitorOnce   sync.Once
	connectionPollInterval  time.Duration
	connectionLossTimeout   time.Duration
	connectionRetryInterval time.Duration
}

func newFingerRobotController(backend fingerRobotBackend) *FingerRobotController {
	ctx, cancel := context.WithCancel(context.Background())
	return &FingerRobotController{
		backend:                 backend,
		ctx:                     ctx,
		cancel:                  cancel,
		debugLeaseTimeout:       time.Duration(FingerRobotDebugLeaseTimeoutMS) * time.Millisecond,
		connectionPollInterval:  fingerRobotConnectionPollInterval,
		connectionLossTimeout:   fingerRobotConnectionLossTimeout,
		connectionRetryInterval: fingerRobotConnectionRetryInterval,
		status: FingerRobotStatus{
			State:  FingerRobotStateIdle,
			Result: FingerRobotResultIdle,
		},
	}
}

func (c *FingerRobotController) Initialize() error {
	c.startConnectionMonitor()
	return c.initialize(true)
}

func (c *FingerRobotController) initialize(enable bool) error {
	c.resetMu.Lock()
	defer c.resetMu.Unlock()

	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return ErrFingerRobotClosed
	}
	if c.resetting || c.active != nil {
		c.mu.Unlock()
		return ErrFingerRobotBusy
	}
	if enable {
		c.enabled = true
	}
	if !c.enabled {
		c.mu.Unlock()
		return ErrFingerRobotUnavailable
	}
	if c.available {
		c.mu.Unlock()
		return nil
	}
	c.initializationAttempted = true
	c.resetting = true
	c.status = FingerRobotStatus{
		State:  FingerRobotStateReleasing,
		Result: FingerRobotResultRunning,
	}
	c.mu.Unlock()

	err := c.initializeAndReleaseHardware()

	c.mu.Lock()
	c.resetting = false
	c.available = err == nil
	if err != nil {
		c.status.Result = FingerRobotResultFailed
		c.status.Error = err.Error()
	} else {
		c.status.State = FingerRobotStateIdle
		c.status.Result = FingerRobotResultIdle
		c.status.Error = ""
	}
	c.mu.Unlock()

	return err
}

func (c *FingerRobotController) IsAvailable() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.enabled && c.available && !c.closed
}

func (c *FingerRobotController) Status() FingerRobotStatus {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.status
}

func (c *FingerRobotController) startConnectionMonitor() {
	backend, ok := c.backend.(fingerRobotHotplugBackend)
	if !ok {
		return
	}
	c.connectionMonitorOnce.Do(func() {
		go c.monitorConnection(backend)
	})
}

func (c *FingerRobotController) monitorConnection(backend fingerRobotHotplugBackend) {
	pollInterval := c.connectionPollInterval
	if pollInterval <= 0 {
		pollInterval = fingerRobotConnectionPollInterval
	}
	lossTimeout := c.connectionLossTimeout
	if lossTimeout <= 0 {
		lossTimeout = fingerRobotConnectionLossTimeout
	}
	retryInterval := c.connectionRetryInterval
	if retryInterval <= 0 {
		retryInterval = fingerRobotConnectionRetryInterval
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	nextRetry := time.Now().Add(retryInterval)
	wasConnected := false

	for {
		select {
		case <-c.ctx.Done():
			return
		case now := <-ticker.C:
			c.mu.Lock()
			enabled := c.enabled
			available := c.available
			closed := c.closed
			resetting := c.resetting
			active := c.active != nil
			factoryZeroHeld := isFingerRobotFactoryZeroHeld(c.status)
			c.mu.Unlock()

			if closed {
				return
			}
			if !enabled {
				wasConnected = false
				nextRetry = now.Add(retryInterval)
				continue
			}
			if resetting || factoryZeroHeld {
				// Reset and the explicit factory-zero hold deliberately do not
				// query feedback connectivity or trigger the disconnect finalizer.
				wasConnected = false
				nextRetry = now.Add(retryInterval)
				continue
			}

			connected := backend.ConnectionAlive(lossTimeout)

			if available {
				if !connected {
					c.handleConnectionLost(backend, lossTimeout)
					nextRetry = now.Add(retryInterval)
				}
				wasConnected = connected
				continue
			}
			if active {
				wasConnected = connected
				continue
			}

			becameConnected := connected && !wasConnected
			switch {
			case backend.HardwareReady() && connected && (becameConnected || !now.Before(nextRetry)):
				c.recoverConnectedHardware(backend, lossTimeout)
				nextRetry = now.Add(retryInterval)
			case !backend.HardwareReady() && !now.Before(nextRetry):
				// Hardware initialization failures are retried slowly. Once GPIO and
				// PWM are ready, an unplugged servo is handled only by edge monitoring
				// and no longer causes repeated pinmux/PWM reconfiguration.
				_ = c.initialize(false)
				nextRetry = now.Add(retryInterval)
				wasConnected = false
				continue
			}
			wasConnected = connected
		}
	}
}

func (c *FingerRobotController) handleConnectionLost(backend fingerRobotHotplugBackend, lossTimeout time.Duration) {
	c.resetMu.Lock()
	defer c.resetMu.Unlock()

	c.mu.Lock()
	if isFingerRobotFactoryZeroHeld(c.status) {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()

	// The edge may have returned while waiting for another lifecycle operation.
	if backend.ConnectionAlive(lossTimeout) {
		return
	}

	c.mu.Lock()
	if c.closed || !c.enabled || !c.available || c.resetting || isFingerRobotFactoryZeroHeld(c.status) {
		c.mu.Unlock()
		return
	}
	c.resetting = true
	c.available = false
	op := c.active
	if op != nil {
		op.externalFinalizer = true
		op.cancel()
	}
	c.status.Running = true
	c.status.State = FingerRobotStateReleasing
	c.status.Result = FingerRobotResultRunning
	c.status.Error = ""
	c.mu.Unlock()

	// PWM retains its last duty after the connector opens. Force the output to
	// the safe release pulse before a later re-plug can reconnect that signal to
	// the servo. This deliberately bypasses edge gating only after feedback has
	// been absent for the full disconnect timeout.
	var safeErr error
	if err := backend.ForceSafeRelease(); err != nil {
		safeErr = fmt.Errorf("force safe release after disconnect: %w", err)
	}
	if op != nil {
		<-op.done
		// An operation may already have passed its cancellation check when the
		// disconnect is detected. If feedback returns during a fast re-plug, that
		// in-flight write can complete after the first forced release. Write the
		// safe point again so the external finalizer always owns the last duty.
		if err := backend.ForceSafeRelease(); err != nil {
			safeErr = errors.Join(
				safeErr,
				fmt.Errorf("finalize safe release after canceled operation: %w", err),
			)
		}
	}

	disconnectErr := errors.Join(ErrFingerRobotDisconnected, safeErr)
	c.mu.Lock()
	c.resetting = false
	c.status.Running = false
	c.status.State = FingerRobotStateReleasing
	c.status.Result = FingerRobotResultFailed
	c.status.Error = disconnectErr.Error()
	c.mu.Unlock()
}

func (c *FingerRobotController) recoverConnectedHardware(backend fingerRobotHotplugBackend, lossTimeout time.Duration) {
	c.resetMu.Lock()
	defer c.resetMu.Unlock()

	c.mu.Lock()
	if isFingerRobotFactoryZeroHeld(c.status) {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()

	if !backend.ConnectionAlive(lossTimeout) {
		return
	}
	c.mu.Lock()
	if c.closed || !c.enabled || c.available || c.resetting || c.active != nil || isFingerRobotFactoryZeroHeld(c.status) {
		c.mu.Unlock()
		return
	}
	c.resetting = true
	c.status.Running = true
	c.status.State = FingerRobotStateReleasing
	c.status.Result = FingerRobotResultRunning
	c.status.Error = ""
	c.mu.Unlock()

	err := setFingerRobotPulse(context.Background(), c.backend, FingerRobotSafeReleaseUS)
	if err != nil {
		err = fmt.Errorf("restore safe release point after reconnect: %w", err)
	}

	c.mu.Lock()
	c.resetting = false
	c.available = err == nil
	c.status.Running = false
	if err != nil {
		c.status.State = FingerRobotStateReleasing
		c.status.Result = FingerRobotResultFailed
		c.status.Error = err.Error()
	} else {
		c.status.State = FingerRobotStateIdle
		c.status.Result = FingerRobotResultIdle
		c.status.Error = ""
	}
	c.mu.Unlock()
}

// Press performs one complete normal-mode press synchronously. A concurrent
// Release call can cancel the ramp or hold and takes ownership of the final
// safe-release write.
func (c *FingerRobotController) Press(req FingerRobotPressRequest) error {
	pressUS, speedUSPerSecond, err := resolveFingerRobotPressRequest(req)
	if err != nil {
		return err
	}
	if err := c.ensureAvailable(); err != nil {
		return err
	}

	op, err := c.beginOperation(1, false, FingerRobotSafeReleaseUS)
	if err != nil {
		return err
	}

	if executor, ok := c.backend.(fingerRobotActionExecutor); ok {
		err = executor.ExecuteFingerRobotAction(op.ctx, func(backend fingerRobotBackend) error {
			return c.runPressOnBackend(op, backend, pressUS, speedUSPerSecond, time.Duration(req.PressMS)*time.Millisecond)
		})
	} else {
		err = c.runPress(op, pressUS, speedUSPerSecond, time.Duration(req.PressMS)*time.Millisecond)
	}
	c.completeOperation(op, err)
	return err
}

func (c *FingerRobotController) ensureAvailable() error {
	if c.IsAvailable() {
		return nil
	}
	if err := c.initialize(false); err != nil {
		if errors.Is(err, ErrFingerRobotClosed) {
			return err
		}
		if errors.Is(err, ErrFingerRobotUnavailable) {
			return ErrFingerRobotUnavailable
		}
		return errors.Join(ErrFingerRobotUnavailable, err)
	}
	return nil
}

func resolveFingerRobotPressRequest(req FingerRobotPressRequest) (int, int, error) {
	pressUS, err := ResolveFingerRobotPressUS(req.ProfileID, req.Strength)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: %v", ErrFingerRobotInvalid, err)
	}
	speedUSPerSecond, err := ResolveFingerRobotSpeedUSPerSecond(req.Speed)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: %v", ErrFingerRobotInvalid, err)
	}
	limits := GetFingerRobotLimits()
	if req.PressMS < limits.PressMinMS || req.PressMS > limits.PressMaxMS {
		return 0, 0, fmt.Errorf(
			"%w: pressMs must be between %d and %d",
			ErrFingerRobotInvalid,
			limits.PressMinMS,
			limits.PressMaxMS,
		)
	}
	return pressUS, speedUSPerSecond, nil
}

func (c *FingerRobotController) StartDebug(req FingerRobotDebugRequest) error {
	if err := validateFingerRobotDebugRequest(req); err != nil {
		return err
	}

	op, err := c.beginOperation(req.Count, true, req.ReleaseUS)
	if err != nil {
		return err
	}

	go c.monitorDebugLease(op)
	go func() {
		var err error
		if executor, ok := c.backend.(fingerRobotActionExecutor); ok {
			err = executor.ExecuteFingerRobotAction(op.ctx, func(backend fingerRobotBackend) error {
				return c.runDebugOnBackend(op, backend, req)
			})
		} else {
			err = c.runDebug(op, req)
		}
		c.completeOperation(op, err)
	}()

	return nil
}

func (c *FingerRobotController) TouchDebugLease() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.active == nil || c.active.lease == nil {
		return false
	}
	select {
	case <-c.active.ctx.Done():
		return false
	default:
	}
	now := time.Now()
	if !now.Before(c.active.leaseDeadline) {
		c.active.cancel()
		return false
	}
	c.active.leaseDeadline = now.Add(c.debugLeaseTimeout)
	select {
	case c.active.lease <- struct{}{}:
	default:
	}
	return true
}

func validateFingerRobotDebugRequest(req FingerRobotDebugRequest) error {
	limits := fingerRobotLimits
	if req.ReleaseUS < limits.ReleaseMinUS || req.ReleaseUS > limits.ReleaseMaxUS {
		return fmt.Errorf("%w: releaseUs must be between %d and %d", ErrFingerRobotInvalid, limits.ReleaseMinUS, limits.ReleaseMaxUS)
	}
	if req.PressUS < limits.PressMinUS || req.PressUS > limits.PressMaxUS {
		return fmt.Errorf("%w: pressUs must be between %d and %d", ErrFingerRobotInvalid, limits.PressMinUS, limits.PressMaxUS)
	}
	if req.PressUS >= req.ReleaseUS {
		return fmt.Errorf("%w: pressUs must be less than releaseUs", ErrFingerRobotInvalid)
	}
	if req.PressSpeedUSPerSecond < limits.PressSpeedMinUSPerSecond || req.PressSpeedUSPerSecond > limits.PressSpeedMaxUSPerSecond {
		return fmt.Errorf("%w: pressSpeedUsPerSecond must be between %d and %d", ErrFingerRobotInvalid, limits.PressSpeedMinUSPerSecond, limits.PressSpeedMaxUSPerSecond)
	}
	if req.Count != limits.CountUnlimitedValue && req.Count < limits.CountMin {
		return fmt.Errorf("%w: count must be %d for unlimited testing or at least %d", ErrFingerRobotInvalid, limits.CountUnlimitedValue, limits.CountMin)
	}
	if req.PressMS < limits.PressMinMS || req.PressMS > limits.PressMaxMS {
		return fmt.Errorf("%w: pressMs must be between %d and %d", ErrFingerRobotInvalid, limits.PressMinMS, limits.PressMaxMS)
	}

	return nil
}

func (c *FingerRobotController) StopDebug() error {
	c.resetMu.Lock()
	defer c.resetMu.Unlock()

	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return ErrFingerRobotClosed
	}
	if c.resetting {
		c.mu.Unlock()
		return ErrFingerRobotBusy
	}

	op := c.active
	if op == nil {
		c.mu.Unlock()
		return nil
	}
	if !op.debug {
		c.mu.Unlock()
		return ErrFingerRobotBusy
	}

	// Prevent a new operation from starting after completeOperation clears
	// active but before the configured release point has been restored.
	c.resetting = true
	op.externalFinalizer = true
	c.status.State = FingerRobotStateReleasing
	c.status.Result = FingerRobotResultRunning
	c.status.Error = ""
	op.cancel()
	c.mu.Unlock()

	<-op.done

	c.mu.Lock()
	c.status.Running = true
	c.status.State = FingerRobotStateReleasing
	c.status.Result = FingerRobotResultRunning
	c.status.Error = ""
	c.mu.Unlock()

	err := c.setSafeReleaseWithFeedbackFallback(op.releaseUS)

	c.mu.Lock()
	c.resetting = false
	c.status.Running = false
	c.status.State = FingerRobotStateIdle
	if err != nil {
		err = fmt.Errorf("restore debug release point after stop: %w", err)
		c.available = false
		c.status.Result = FingerRobotResultFailed
		c.status.Error = err.Error()
	} else {
		switch {
		case op.completionErr == nil:
			// The finite sequence completed naturally while StopDebug was
			// acquiring ownership. Treat the stop as an idempotent success.
			c.status.Result = FingerRobotResultSuccess
			c.status.Error = ""
		case errors.Is(op.completionErr, context.Canceled), errors.Is(op.completionErr, context.DeadlineExceeded):
			c.status.Result = FingerRobotResultCancelled
			c.status.Error = ""
		default:
			c.status.Result = FingerRobotResultFailed
			c.status.Error = op.completionErr.Error()
		}
	}
	c.mu.Unlock()

	return err
}

// Release cancels any active normal or Debug operation and
// restores the absolute safe release point. It is deliberately not rejected by
// the ordinary busy state: resetMu serializes it with every other controller
// finalizer, and externalFinalizer guarantees this write wins after cancellation.
func (c *FingerRobotController) Release() error {
	c.resetMu.Lock()
	defer c.resetMu.Unlock()

	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return ErrFingerRobotClosed
	}
	if !c.enabled || !c.initializationAttempted {
		c.mu.Unlock()
		// A never-enabled or explicitly disabled controller is already safe by
		// lifecycle contract. Keep Release idempotent without initializing or
		// touching PWM merely to repeat that state.
		return nil
	}

	c.resetting = true
	c.status = FingerRobotStatus{
		Running: true,
		State:   FingerRobotStateReleasing,
		Result:  FingerRobotResultRunning,
	}
	op := c.active
	if op != nil {
		op.externalFinalizer = true
		op.cancel()
	}
	c.mu.Unlock()

	if op != nil {
		<-op.done
	}

	c.mu.Lock()
	c.status = FingerRobotStatus{
		Running: true,
		State:   FingerRobotStateReleasing,
		Result:  FingerRobotResultRunning,
	}
	c.mu.Unlock()

	err := c.setSafeReleaseWithFeedbackFallback(FingerRobotSafeReleaseUS)
	if err != nil {
		err = fmt.Errorf("release finger robot to safe point: %w", err)
	}

	c.mu.Lock()
	c.resetting = false
	c.status.Running = false
	if err != nil {
		c.available = false
		c.status.State = FingerRobotStateReleasing
		c.status.Result = FingerRobotResultFailed
		c.status.Error = err.Error()
	} else {
		c.status.State = FingerRobotStateIdle
		c.status.Result = FingerRobotResultSuccess
		c.status.Error = ""
	}
	c.mu.Unlock()

	return err
}

func (c *FingerRobotController) Reset() error {
	c.resetMu.Lock()
	defer c.resetMu.Unlock()

	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return ErrFingerRobotClosed
	}
	if !c.enabled {
		c.mu.Unlock()
		return ErrFingerRobotUnavailable
	}
	wasAvailable := c.available
	if !wasAvailable {
		c.initializationAttempted = true
	}
	c.resetting = true
	c.status.State = FingerRobotStateResetting
	c.status.Result = FingerRobotResultRunning
	c.status.Error = ""
	c.status.debug = true
	op := c.active
	if op != nil {
		op.externalFinalizer = true
		op.cancel()
	}
	c.mu.Unlock()

	if op != nil {
		<-op.done
	}

	var err error
	if wasAvailable {
		err = c.setFactoryZeroWithSafetyFallback()
	} else {
		err = c.initializeAndFactoryResetHardware()
	}

	c.mu.Lock()
	c.resetting = false
	c.available = err == nil
	c.status.Running = false
	if err != nil {
		c.status.State = FingerRobotStateResetting
		c.status.Result = FingerRobotResultFailed
		c.status.Error = err.Error()
	} else {
		c.status.State = FingerRobotStateIdle
		c.status.Result = FingerRobotResultReset
		c.status.Error = ""
	}
	c.mu.Unlock()

	return err
}

// Disable makes the controller unavailable without disabling PWM output. It
// cancels any in-flight operation, restores the absolute safe release point,
// and leaves the controller reusable by a later explicit Initialize call.
func (c *FingerRobotController) Disable() error {
	c.resetMu.Lock()
	defer c.resetMu.Unlock()

	c.mu.Lock()
	if c.closed || !c.enabled {
		c.mu.Unlock()
		return nil
	}

	c.enabled = false
	c.available = false
	shouldRelease := c.initializationAttempted
	if !shouldRelease {
		c.status = FingerRobotStatus{
			State:  FingerRobotStateIdle,
			Result: FingerRobotResultIdle,
		}
		c.mu.Unlock()
		return nil
	}

	c.resetting = true
	c.status.State = FingerRobotStateReleasing
	c.status.Result = FingerRobotResultRunning
	c.status.Error = ""
	c.status.debug = false
	op := c.active
	if op != nil {
		op.externalFinalizer = true
		op.cancel()
	}
	c.mu.Unlock()

	if op != nil {
		<-op.done
	}

	c.mu.Lock()
	c.status.Running = true
	c.status.State = FingerRobotStateReleasing
	c.status.Result = FingerRobotResultRunning
	c.status.Error = ""
	c.mu.Unlock()

	err := c.setSafeReleaseWithFeedbackFallback(FingerRobotSafeReleaseUS)
	if err == ErrFingerRobotDisconnected {
		err = nil
	}
	if err != nil {
		err = fmt.Errorf("release finger robot while disabling: %w", err)
	}

	c.mu.Lock()
	c.resetting = false
	c.status.Running = false
	if err != nil {
		c.status.State = FingerRobotStateReleasing
		c.status.Result = FingerRobotResultFailed
		c.status.Error = err.Error()
	} else {
		c.status.State = FingerRobotStateIdle
		c.status.Result = FingerRobotResultIdle
		c.status.Error = ""
	}
	c.mu.Unlock()

	return err
}

func (c *FingerRobotController) Shutdown() error {
	c.resetMu.Lock()
	defer c.resetMu.Unlock()

	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	shouldRelease := c.initializationAttempted
	c.closed = true
	c.enabled = false
	c.available = false
	c.resetting = shouldRelease
	op := c.active
	c.cancel()
	if op != nil {
		op.externalFinalizer = true
		op.cancel()
	}
	if shouldRelease {
		c.status.State = FingerRobotStateReleasing
		c.status.Result = FingerRobotResultRunning
		c.status.Error = ""
	} else {
		c.status = FingerRobotStatus{
			State:  FingerRobotStateIdle,
			Result: FingerRobotResultShutdown,
		}
	}
	c.mu.Unlock()

	if op != nil {
		<-op.done
	}

	if !shouldRelease {
		if err := closeFingerRobotBackend(c.backend); err != nil {
			return fmt.Errorf("close finger robot backend during shutdown: %w", err)
		}
		return nil
	}

	err := c.setSafeReleaseWithFeedbackFallback(FingerRobotSafeReleaseUS)
	if err == ErrFingerRobotDisconnected {
		err = nil
	}
	if err != nil {
		err = fmt.Errorf("release finger robot during shutdown: %w", err)
	}
	if restoreErr := restoreFingerRobotBackendShutdownPins(c.backend); restoreErr != nil {
		err = errors.Join(err, fmt.Errorf("restore finger robot shutdown pins: %w", restoreErr))
	}
	if closeErr := closeFingerRobotBackend(c.backend); closeErr != nil {
		err = errors.Join(err, fmt.Errorf("close finger robot backend during shutdown: %w", closeErr))
	}

	c.mu.Lock()
	c.resetting = false
	c.status.Running = false
	c.status.State = FingerRobotStateReleasing
	if err != nil {
		c.status.Result = FingerRobotResultFailed
		c.status.Error = err.Error()
	} else {
		c.status.Result = FingerRobotResultShutdown
		c.status.Error = ""
	}
	c.mu.Unlock()

	return err
}

func closeFingerRobotBackend(backend fingerRobotBackend) error {
	closer, ok := backend.(fingerRobotBackendCloser)
	if !ok {
		return nil
	}
	return closer.Close()
}

func restoreFingerRobotBackendShutdownPins(backend fingerRobotBackend) error {
	restorer, ok := backend.(fingerRobotBackendShutdownPinRestorer)
	if !ok {
		return nil
	}
	return restorer.RestoreShutdownPins()
}

func (c *FingerRobotController) initializeAndReleaseHardware() error {
	initErr := c.backend.Initialize()
	if initErr == nil {
		if backend, ok := c.backend.(fingerRobotHotplugBackend); ok {
			if err := backend.WaitConnected(fingerRobotPWMConnectionWaitTimeout); err != nil {
				return errors.Join(
					ErrFingerRobotDisconnected,
					fmt.Errorf("wait for finger robot PWM feedback: %w", err),
				)
			}
		}
	}
	releaseErr := c.setSafeReleaseWithFeedbackFallback(FingerRobotSafeReleaseUS)
	if initErr != nil {
		initErr = fmt.Errorf("initialize finger robot PWM: %w", initErr)
	}
	if releaseErr != nil {
		releaseErr = fmt.Errorf("release finger robot PWM after initialization: %w", releaseErr)
	}
	return errors.Join(initErr, releaseErr)
}

func (c *FingerRobotController) initializeAndFactoryResetHardware() error {
	initErr := c.backend.Initialize()
	if initErr != nil {
		initErr = fmt.Errorf("initialize finger robot PWM for factory reset: %w", initErr)
		releaseErr := c.setSafeReleaseWithFeedbackFallback(FingerRobotSafeReleaseUS)
		if releaseErr != nil {
			releaseErr = fmt.Errorf("release finger robot PWM after failed factory reset initialization: %w", releaseErr)
		}
		return errors.Join(initErr, releaseErr)
	}

	return c.setFactoryZeroWithSafetyFallback()
}

func (c *FingerRobotController) beginOperation(total int64, debug bool, releaseUS int) (*fingerRobotOperation, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil, ErrFingerRobotClosed
	}
	if !c.enabled || !c.available {
		return nil, ErrFingerRobotUnavailable
	}
	if c.resetting || c.active != nil {
		return nil, ErrFingerRobotBusy
	}

	ctx, cancel := context.WithCancel(c.ctx)
	op := &fingerRobotOperation{
		ctx:       ctx,
		cancel:    cancel,
		done:      make(chan struct{}),
		debug:     debug,
		releaseUS: releaseUS,
	}
	if debug {
		op.lease = make(chan struct{}, 1)
		op.leaseDeadline = time.Now().Add(c.debugLeaseTimeout)
	}
	c.active = op
	c.status = FingerRobotStatus{
		Running: true,
		Total:   total,
		State:   FingerRobotStateIdle,
		Result:  FingerRobotResultRunning,
		debug:   debug,
	}
	return op, nil
}

func (c *FingerRobotController) monitorDebugLease(op *fingerRobotOperation) {
	remaining, active := c.fingerRobotDebugLeaseRemaining(op)
	if !active {
		return
	}
	timer := time.NewTimer(remaining)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			remaining, active = c.fingerRobotDebugLeaseRemaining(op)
			if !active {
				return
			}
			timer.Reset(remaining)
		case <-op.lease:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			remaining, active = c.fingerRobotDebugLeaseRemaining(op)
			if !active {
				return
			}
			timer.Reset(remaining)
		case <-op.ctx.Done():
			return
		}
	}
}

func (c *FingerRobotController) completeOperation(op *fingerRobotOperation, err error) {
	op.cancel()

	c.mu.Lock()
	op.completionErr = err
	if c.active == op {
		c.status.Running = false
		if err == nil {
			c.status.Result = FingerRobotResultSuccess
			c.status.Error = ""
		} else if op.recoveryErr != nil {
			c.status.Result = FingerRobotResultFailed
			c.status.Error = err.Error()
		} else if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			c.status.Result = FingerRobotResultCancelled
			c.status.Error = err.Error()
		} else {
			c.status.Result = FingerRobotResultFailed
			c.status.Error = err.Error()
		}
		c.active = nil
	}
	close(op.done)
	c.mu.Unlock()
}

func (c *FingerRobotController) runPress(op *fingerRobotOperation, pressUS, speedUSPerSecond int, duration time.Duration) error {
	return c.runPressOnBackend(op, c.backend, pressUS, speedUSPerSecond, duration)
}

func (c *FingerRobotController) runPressOnBackend(
	op *fingerRobotOperation,
	backend fingerRobotBackend,
	pressUS, speedUSPerSecond int,
	duration time.Duration,
) error {
	// Always establish the real release position before calculating the ramp.
	// Debug factory reset intentionally leaves the servo at 1500 us, so merely
	// assuming a 2050 us starting position could otherwise cause a sudden jump.
	c.updateProgress(op, FingerRobotStateReleasing, 0)
	if err := c.checkFingerRobotOperation(op); err != nil {
		return c.resetAfterFailureOnBackend(op, backend, err)
	}
	if err := c.setSafeReleaseWithFeedbackFallbackOnBackend(op.ctx, backend, FingerRobotSafeReleaseUS); err != nil {
		return c.resetAfterFailureOnBackend(op, backend, fmt.Errorf("set initial safe release point: %w", err))
	}

	c.updateProgress(op, FingerRobotStatePressing, 1)
	if err := c.checkFingerRobotOperation(op); err != nil {
		return c.resetAfterFailureOnBackend(op, backend, err)
	}
	if err := c.rampPulseOnBackend(op, backend, FingerRobotSafeReleaseUS, pressUS, speedUSPerSecond); err != nil {
		return c.resetAfterFailureOnBackend(op, backend, fmt.Errorf("press finger robot: %w", err))
	}

	c.updateProgress(op, FingerRobotStateHolding, 1)
	if err := c.checkFingerRobotOperation(op); err != nil {
		return c.resetAfterFailureOnBackend(op, backend, err)
	}
	if err := waitFingerRobotOnBackend(op.ctx, duration, backend); err != nil {
		return c.resetAfterFailureOnBackend(op, backend, err)
	}

	c.updateProgress(op, FingerRobotStateReleasing, 1)
	if err := c.checkFingerRobotOperation(op); err != nil {
		return c.resetAfterFailureOnBackend(op, backend, err)
	}
	if err := c.rampPulseOnBackend(op, backend, pressUS, FingerRobotSafeReleaseUS, speedUSPerSecond); err != nil {
		return c.resetAfterFailureOnBackend(op, backend, fmt.Errorf("release finger robot: %w", err))
	}
	c.updateProgress(op, FingerRobotStateIdle, 1)
	return nil
}

func (c *FingerRobotController) runDebug(op *fingerRobotOperation, req FingerRobotDebugRequest) error {
	return c.runDebugOnBackend(op, c.backend, req)
}

func (c *FingerRobotController) runDebugOnBackend(op *fingerRobotOperation, backend fingerRobotBackend, req FingerRobotDebugRequest) error {
	c.updateProgress(op, FingerRobotStateReleasing, 0)
	if err := c.checkFingerRobotOperation(op); err != nil {
		return c.resetAfterFailureOnBackend(op, backend, err)
	}
	if err := setFingerRobotPulse(op.ctx, backend, req.ReleaseUS); err != nil {
		return c.resetAfterFailureOnBackend(op, backend, fmt.Errorf("set initial release point: %w", err))
	}

	for current := int64(1); ; current++ {
		c.updateProgress(op, FingerRobotStatePressing, current)
		if err := c.checkFingerRobotOperation(op); err != nil {
			return c.resetAfterFailureOnBackend(op, backend, err)
		}
		if err := c.rampPulseOnBackend(op, backend, req.ReleaseUS, req.PressUS, req.PressSpeedUSPerSecond); err != nil {
			return c.resetAfterFailureOnBackend(op, backend, fmt.Errorf("press %d: %w", current, err))
		}
		if err := waitFingerRobotOnBackend(op.ctx, time.Duration(req.PressMS)*time.Millisecond, backend); err != nil {
			return c.resetAfterFailureOnBackend(op, backend, err)
		}

		c.updateProgress(op, FingerRobotStateReleasing, current)
		if err := c.checkFingerRobotOperation(op); err != nil {
			return c.resetAfterFailureOnBackend(op, backend, err)
		}
		if err := c.rampPulseOnBackend(op, backend, req.PressUS, req.ReleaseUS, req.PressSpeedUSPerSecond); err != nil {
			return c.resetAfterFailureOnBackend(op, backend, fmt.Errorf("release %d: %w", current, err))
		}
		if req.Count > 0 && current >= req.Count {
			return nil
		}
		if err := waitFingerRobotOnBackend(op.ctx, time.Duration(fingerRobotLimits.ReleaseIntervalMS)*time.Millisecond, backend); err != nil {
			return c.resetAfterFailureOnBackend(op, backend, err)
		}
	}
}

func (c *FingerRobotController) rampPulse(op *fingerRobotOperation, fromUS, toUS, speedUSPerSecond int) error {
	return c.rampPulseOnBackend(op, c.backend, fromUS, toUS, speedUSPerSecond)
}

func (c *FingerRobotController) rampPulseOnBackend(op *fingerRobotOperation, backend fingerRobotBackend, fromUS, toUS, speedUSPerSecond int) error {
	steps := fingerRobotRampSteps(fromUS, toUS, speedUSPerSecond)
	timer := time.NewTimer(fingerRobotRampInterval)
	defer timer.Stop()

	lastPulseUS := fromUS
	for step := 1; step <= steps; step++ {
		select {
		case <-timer.C:
		case <-op.ctx.Done():
			return op.ctx.Err()
		}

		if err := c.checkFingerRobotOperation(op); err != nil {
			return err
		}
		pulseUS := fingerRobotRampPulse(fromUS, toUS, step, steps)
		if pulseUS != lastPulseUS {
			if err := setFingerRobotPulse(op.ctx, backend, pulseUS); err != nil {
				return fmt.Errorf("set ramp pulse to %d us: %w", pulseUS, err)
			}
			lastPulseUS = pulseUS
		}

		if step < steps {
			timer.Reset(fingerRobotRampInterval)
		}
	}

	return nil
}

func fingerRobotRampSteps(fromUS, toUS, speedUSPerSecond int) int {
	distanceUS := int64(toUS) - int64(fromUS)
	if distanceUS < 0 {
		distanceUS = -distanceUS
	}
	intervalMS := int64(fingerRobotRampInterval / time.Millisecond)
	denominator := int64(speedUSPerSecond) * intervalMS
	return int((distanceUS*1000 + denominator - 1) / denominator)
}

func fingerRobotRampPulse(fromUS, toUS, step, steps int) int {
	distanceUS := int64(toUS) - int64(fromUS)
	direction := int64(1)
	if distanceUS < 0 {
		distanceUS = -distanceUS
		direction = -1
	}
	offsetUS := distanceUS * int64(step)
	if direction < 0 {
		offsetUS += int64(steps) - 1
	}
	offsetUS /= int64(steps)
	return fromUS + int(direction*offsetUS)
}

func (c *FingerRobotController) setSafeReleaseWithFeedbackFallback(pulseUS int) error {
	return c.setSafeReleaseWithFeedbackFallbackOnBackend(context.Background(), c.backend, pulseUS)
}

func (c *FingerRobotController) setSafeReleaseWithFeedbackFallbackOnBackend(
	ctx context.Context,
	backendImpl fingerRobotBackend,
	pulseUS int,
) error {
	backend, ok := backendImpl.(fingerRobotHotplugBackend)
	if ok && !backend.ConnectionAlive(c.connectionLossTimeout) {
		if err := backend.ForceSafeRelease(); err != nil {
			return errors.Join(
				ErrFingerRobotDisconnected,
				fmt.Errorf("force safe release without PWM feedback: %w", err),
			)
		}
		return ErrFingerRobotDisconnected
	}

	err := setFingerRobotPulse(ctx, backendImpl, pulseUS)
	if err == nil || !ok {
		return err
	}
	feedbackTimedOut := errors.Is(err, errFingerRobotGPIOWaitTimeout)
	if !feedbackTimedOut && backend.ConnectionAlive(c.connectionLossTimeout) {
		return err
	}
	if forceErr := backend.ForceSafeRelease(); forceErr != nil {
		return errors.Join(
			err,
			ErrFingerRobotDisconnected,
			fmt.Errorf("force safe release after feedback loss: %w", forceErr),
		)
	}
	return ErrFingerRobotDisconnected
}

func (c *FingerRobotController) setFactoryZeroWithSafetyFallback() error {
	err := setFingerRobotPulseUngated(c.backend, FingerRobotFactoryZeroUS)
	if err == nil {
		return nil
	}

	resetErr := fmt.Errorf("factory-zero finger robot PWM: %w", err)
	if safeErr := c.setSafeReleaseWithFeedbackFallback(FingerRobotSafeReleaseUS); safeErr != nil {
		return errors.Join(
			resetErr,
			fmt.Errorf("restore safe release point after failed factory reset: %w", safeErr),
		)
	}
	return resetErr
}

func (c *FingerRobotController) resetAfterFailure(op *fingerRobotOperation, operationErr error) error {
	return c.resetAfterFailureOnBackend(op, c.backend, operationErr)
}

func (c *FingerRobotController) resetAfterFailureOnBackend(
	op *fingerRobotOperation,
	backend fingerRobotBackend,
	operationErr error,
) error {
	if (errors.Is(operationErr, context.Canceled) || errors.Is(operationErr, context.DeadlineExceeded)) && c.operationHasExternalFinalizer(op) {
		return operationErr
	}

	recoveryAction := "restore safe release point after failure"
	c.updateProgress(op, FingerRobotStateReleasing, c.Status().Current)
	if err := c.setSafeReleaseWithFeedbackFallbackOnBackend(context.Background(), backend, FingerRobotSafeReleaseUS); err != nil {
		op.recoveryErr = fmt.Errorf("%s: %w", recoveryAction, err)
		c.mu.Lock()
		if c.active == op {
			c.available = false
		}
		c.mu.Unlock()
		return errors.Join(operationErr, op.recoveryErr)
	}
	return operationErr
}

func (c *FingerRobotController) operationHasExternalFinalizer(op *fingerRobotOperation) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.active == op && op.externalFinalizer
}

func (c *FingerRobotController) updateProgress(op *fingerRobotOperation, state FingerRobotState, current int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.active != op {
		return
	}
	c.status.State = state
	c.status.Current = current
}

func waitFingerRobot(ctx context.Context, duration time.Duration) error {
	return waitFingerRobotOnBackend(ctx, duration, nil)
}

func waitFingerRobotOnBackend(ctx context.Context, duration time.Duration, backend fingerRobotBackend) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	// A production realtime action owns the GPIO event fd for its complete
	// lifetime. Continue draining feedback during long hold/release intervals;
	// otherwise the ordinary connection monitor would see a stale timestamp and
	// cancel a healthy press after 500ms.
	var ticker *time.Ticker
	var tick <-chan time.Time
	if backendDrainsFingerRobotEvents(backend) {
		ticker = time.NewTicker(fingerRobotWorkerPoll)
		tick = ticker.C
		defer ticker.Stop()
	}

	for {
		select {
		case <-timer.C:
			return nil
		case <-tick:
			if err := drainFingerRobotBackendEvents(backend); err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func backendDrainsFingerRobotEvents(backend fingerRobotBackend) bool {
	pwmBackend, ok := backend.(*pwmFingerRobotBackend)
	if !ok || pwmBackend == nil {
		return false
	}
	pwmBackend.mu.Lock()
	defer pwmBackend.mu.Unlock()
	_, ok = pwmBackend.lowGate.(fingerRobotPWMLowGateDrainer)
	return ok
}

func drainFingerRobotBackendEvents(backend fingerRobotBackend) error {
	pwmBackend, ok := backend.(*pwmFingerRobotBackend)
	if !ok || pwmBackend == nil {
		return nil
	}
	pwmBackend.mu.Lock()
	gate := pwmBackend.lowGate
	pwmBackend.mu.Unlock()
	drainer, ok := gate.(fingerRobotPWMLowGateDrainer)
	if !ok {
		return nil
	}
	if err := drainer.DrainEvents(); err != nil {
		return fmt.Errorf("drain finger robot GPIO feedback during timed wait: %w", err)
	}
	return nil
}

func (c *FingerRobotController) checkFingerRobotOperation(op *fingerRobotOperation) error {
	select {
	case <-op.ctx.Done():
		return op.ctx.Err()
	default:
	}

	if op.lease == nil {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.active != op {
		return context.Canceled
	}
	select {
	case <-op.ctx.Done():
		return op.ctx.Err()
	default:
	}
	if !time.Now().Before(op.leaseDeadline) {
		op.cancel()
		return context.Canceled
	}
	return nil
}

func (c *FingerRobotController) fingerRobotDebugLeaseRemaining(op *fingerRobotOperation) (time.Duration, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.active != op || op.lease == nil {
		return 0, false
	}
	select {
	case <-op.ctx.Done():
		return 0, false
	default:
	}

	remaining := time.Until(op.leaseDeadline)
	if remaining <= 0 {
		op.cancel()
		return 0, false
	}
	return remaining, true
}

type pwmFingerRobotBackend struct {
	mu            sync.Mutex
	pwmPaths      []string
	lowGate       fingerRobotPWMLowGate
	hardwareReady bool

	openLowGate   func() (fingerRobotPWMLowGate, error)
	initializePWM func() ([]string, error)
	writePWM      func([]string, string, string) error
}

func newPWMFingerRobotBackend() *pwmFingerRobotBackend {
	return &pwmFingerRobotBackend{
		openLowGate:   newRealtimeFingerRobotPWMLowGate,
		initializePWM: initFingerRobotPWMs,
		writePWM:      writeFingerRobotPWMAll,
	}
}

func (b *pwmFingerRobotBackend) Initialize() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.hardwareReady = false

	var closeErr error
	if b.lowGate != nil {
		oldLowGate := b.lowGate
		b.lowGate = nil
		if err := oldLowGate.Close(); err != nil {
			closeErr = fmt.Errorf("close previous finger robot GPIO capture: %w", err)
		}
	}

	var gateErr error
	if closeErr == nil {
		openLowGate := b.openLowGate
		if openLowGate == nil {
			openLowGate = newRealtimeFingerRobotPWMLowGate
		}
		lowGate, err := openLowGate()
		if err != nil {
			gateErr = fmt.Errorf("initialize finger robot GPIO capture: %w", err)
		} else {
			b.lowGate = lowGate
		}
	}

	// PWM safety does not depend on the feedback GPIO being available. Always
	// attempt channel initialization so a stale duty retained by the kernel
	// after an earlier process exit is restored to 2050 us immediately.
	initializePWM := b.initializePWM
	if initializePWM == nil {
		initializePWM = initFingerRobotPWMs
	}
	pwmPaths, pwmErr := initializePWM()
	b.pwmPaths = pwmPaths
	b.hardwareReady = closeErr == nil && gateErr == nil && pwmErr == nil && b.lowGate != nil && len(pwmPaths) > 0
	return errors.Join(closeErr, gateErr, pwmErr)
}

func (b *pwmFingerRobotBackend) SetPulseUS(pulseUS int) error {
	return b.SetPulseUSContext(context.Background(), pulseUS)
}

func (b *pwmFingerRobotBackend) SetPulseUSContext(ctx context.Context, pulseUS int) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.pwmPaths) == 0 {
		return errors.New("finger robot PWM is not initialized")
	}
	if b.lowGate == nil {
		return errors.New("finger robot GPIO capture is not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	var waitErr error
	if contextualGate, ok := b.lowGate.(fingerRobotPWMLowGateContext); ok {
		waitErr = contextualGate.WaitLowContext(ctx, fingerRobotPWMLowWaitTimeout)
	} else {
		waitErr = b.lowGate.WaitLow(fingerRobotPWMLowWaitTimeout)
	}
	if err := waitErr; err != nil {
		if !errors.Is(err, errFingerRobotGPIOWaitTimeout) &&
			!errors.Is(err, context.Canceled) &&
			!errors.Is(err, context.DeadlineExceeded) {
			b.hardwareReady = false
		}
		return fmt.Errorf("wait for finger robot PWM low level: %w", err)
	}
	return b.writePulseUSLocked(pulseUS)
}

// SetPulseUSUngated is reserved for the explicit Debug factory-zero hold.
// Unlike SetPulseUSContext, it does not wait for GPIO feedback or require a
// connected servo; the PWM channel itself continues emitting the duty value.
func (b *pwmFingerRobotBackend) SetPulseUSUngated(pulseUS int) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.pwmPaths) == 0 {
		return errors.New("finger robot PWM is not initialized")
	}
	return b.writePulseUSLocked(pulseUS)
}

func (b *pwmFingerRobotBackend) writePulseUSLocked(pulseUS int) error {
	writePWM := b.writePWM
	if writePWM == nil {
		writePWM = writeFingerRobotPWMAll
	}
	err := writePWM(b.pwmPaths, "duty_cycle", fmt.Sprintf("%d", pulseUS*1000))
	if err != nil {
		b.hardwareReady = false
	}
	return err
}

func (b *pwmFingerRobotBackend) HardwareReady() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.hardwareReady
}

func (b *pwmFingerRobotBackend) WaitConnected(timeout time.Duration) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.hardwareReady || b.lowGate == nil {
		return errors.New("finger robot PWM feedback is not initialized")
	}
	if err := b.lowGate.WaitLow(timeout); err != nil {
		if !errors.Is(err, errFingerRobotGPIOWaitTimeout) {
			b.hardwareReady = false
		}
		return fmt.Errorf("wait for finger robot PWM feedback connection: %w", err)
	}
	return nil
}

func (b *pwmFingerRobotBackend) ConnectionAlive(maxSilence time.Duration) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.hardwareReady && b.lowGate != nil && b.lowGate.Connected(maxSilence)
}

func (b *pwmFingerRobotBackend) ForceSafeRelease() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.pwmPaths) == 0 {
		b.hardwareReady = false
		return errors.New("finger robot PWM is not initialized")
	}
	return b.writePulseUSLocked(FingerRobotSafeReleaseUS)
}

func (b *pwmFingerRobotBackend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.hardwareReady = false

	if b.lowGate == nil {
		return nil
	}
	err := b.lowGate.Close()
	b.lowGate = nil
	return err
}

func (b *pwmFingerRobotBackend) RestoreShutdownPins() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	var waitForLow func() error
	if b.lowGate != nil {
		waitForLow = func() error {
			return b.lowGate.WaitLow(fingerRobotPWMLowWaitTimeout)
		}
	}
	return restoreFingerRobotPWMShutdownPinsWithWait(
		fingerRobotPWMShutdownGPIOAddrs,
		readFingerRobotGPIORegister,
		writeFingerRobotGPIORegister,
		fingerRobotPWMShutdownMuxSettings,
		writeFingerRobotPWMPinMux,
		waitForLow,
	)
}

// Production hardware is owned by a dedicated realtime worker. The raw PWM
// backend remains constructible for deterministic controller/GPIO tests.
var fingerRobotController = newFingerRobotController(newFingerRobotWorkerBackend())

func InitializeFingerRobotController() error {
	if err := ValidateFingerRobotArmProfiles(); err != nil {
		return fmt.Errorf("validate finger robot arm profiles: %w", err)
	}
	return fingerRobotController.Initialize()
}

func DisableFingerRobotController() error {
	return fingerRobotController.Disable()
}

func ShutdownFingerRobotController() error {
	return fingerRobotController.Shutdown()
}

func IsFingerRobotAvailable() bool {
	return fingerRobotController.IsAvailable()
}

func GetFingerRobotStatus() FingerRobotStatus {
	return fingerRobotController.Status()
}

func PressFingerRobot(req FingerRobotPressRequest) error {
	return fingerRobotController.Press(req)
}

func ReleaseFingerRobot() error {
	return fingerRobotController.Release()
}

func StartFingerRobotDebug(req FingerRobotDebugRequest) error {
	return fingerRobotController.StartDebug(req)
}

func StopFingerRobotDebug() error {
	return fingerRobotController.StopDebug()
}

func TouchFingerRobotDebugLease() bool {
	return fingerRobotController.TouchDebugLease()
}

func ResetFingerRobot() error {
	return fingerRobotController.Reset()
}
