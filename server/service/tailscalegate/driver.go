// The Driver boundary in this file has three implementations. The simulated
// demo driver in demo.go is selected by NewHandler when no driver is injected
// (tests and host walkthroughs). NewUnimplementedDriver is the explicit
// NOT-IMPLEMENTED driver for tests and fail-closed wiring: it answers a
// heartbeat request with 503 and every action with the NOTIMPL message, so
// the gate stays in the uninitialized stage instead of being faked as ready.
// Production uses the real managed-network driver
// (service/extensions/tailscale.NewGateDriver), injected by the state socket
// server.
package tailscalegate

import (
	"context"
	"errors"
)

// ErrUnimplemented marks every call made through the placeholder driver.
var ErrUnimplemented = errors.New("tailscalegate: driver not implemented")

// ErrLoginCodeNotReady marks a transiently unavailable login code: the managed
// identity is not enrolled yet, an upstream Retry-After window is active, or a
// pairing dependency flaked. It is not a component failure, so the handler
// logs it at debug instead of warn while still answering 503 (the client
// retries on its own backoff and keeps showing the previous code).
var ErrLoginCodeNotReady = errors.New("tailscalegate: login code not ready")

// Driver is the only boundary between the interaction layer and the Tailscale
// components.
//
// Every method takes a context and must honour its deadline: the handler
// applies the documented read, action, and disable budgets by cancelling the
// context. A driver that blocks past the deadline must return the context
// error so the caller can answer TIMEOUT instead of hanging the connection.
// Calls run at most one action at a time, so implementations do not need their
// own action serialization.
//
// NewDemoDriver is the current production default: a simulated driver that
// performs no system access at all, which keeps the interaction layer fully
// host-testable while an unmodified device can still walk the documented flow.
// NewUnimplementedDriver is retained as the explicit "not wired" driver and
// answers a stable NOTIMPL failure; the real adapter will replace the demo
// default when it lands.
type Driver interface {
	// Snapshot returns the current gate and tunnel report. A nil field means
	// that field is currently unknown.
	Snapshot(ctx context.Context) (Report, error)
	// Initialize starts the Tailscale component.
	Initialize(ctx context.Context) error
	// Deinitialize stops the Tailscale component and resets the initialized
	// gate; it is the inverse of Initialize.
	Deinitialize(ctx context.Context) error
	// LoginCode returns the current pairing code.
	LoginCode(ctx context.Context) (string, error)
	// EndLoginSession ends the client initialization session.
	EndLoginSession(ctx context.Context) error
}

// unimplementedDriver is the retained explicit NOT-IMPLEMENTED driver. It is no
// longer the default; it stays for tests and callers that want a stable failure
// on the Driver boundary. Every method fails with an operation-named error that
// unwraps to ErrUnimplemented and publishes MessageNotImplemented for the action
// wire response.
type unimplementedDriver struct{}

// NewUnimplementedDriver returns a Driver that performs no system access.
func NewUnimplementedDriver() Driver { return unimplementedDriver{} }

func (unimplementedDriver) Snapshot(context.Context) (Report, error) {
	return Report{}, unimplementedError("snapshot")
}

func (unimplementedDriver) Initialize(context.Context) error {
	return unimplementedError("initialize")
}

func (unimplementedDriver) Deinitialize(context.Context) error {
	return unimplementedError("deinitialize")
}

func (unimplementedDriver) LoginCode(context.Context) (string, error) {
	return "", unimplementedError("login code")
}

func (unimplementedDriver) EndLoginSession(context.Context) error {
	return unimplementedError("end login session")
}

// unimplementedError names the failed operation without carrying any value, so
// it is safe to log and never leaks system state.
type unimplementedError string

func (e unimplementedError) Error() string {
	return "tailscalegate: " + string(e) + " not implemented"
}

// GateMessage publishes the stable failure code for the action wire response.
func (e unimplementedError) GateMessage() string { return MessageNotImplemented }

// Unwrap exposes ErrUnimplemented for errors.Is.
func (e unimplementedError) Unwrap() error { return ErrUnimplemented }
