package vm

import (
	"context"
	"errors"
	"time"
)

const (
	// The legacy sysfs GPIO numbering identifies this controller as
	// gpiochip32 and the connected signal as global GPIO35.  The character
	// device API addresses the same signal as offset 35 - 32 = 3 on the chip
	// whose label is exposed by gpiochip32.
	fingerRobotGPIOLineOffset    = 3
	fingerRobotGPIOChipLabelPath = "/sys/class/gpio/gpiochip32/label"
	fingerRobotGPIOConsumer      = "nanokvm-finger-robot"

	fingerRobotPWMConnectionWaitTimeout = 100 * time.Millisecond
	fingerRobotPWMLowWaitTimeout        = 200 * time.Millisecond
)

var errFingerRobotGPIOWaitTimeout = errors.New("finger robot GPIO low wait timed out")

type fingerRobotPWMLowGate interface {
	WaitLow(timeout time.Duration) error
	Connected(maxSilence time.Duration) bool
	Close() error
}

// fingerRobotPWMLowGateContext is implemented by the realtime GPIO gate. It
// lets an in-flight duty update stop promptly when Release/Reset/Shutdown
// cancels the owning operation. Test gates may implement only
// fingerRobotPWMLowGate and continue to use the non-cancellable method.
type fingerRobotPWMLowGateContext interface {
	WaitLowContext(ctx context.Context, timeout time.Duration) error
}

// fingerRobotPWMLowGateDrainer is called by the worker while it is idle so
// feedback edges are timestamped continuously for hotplug health checks.
type fingerRobotPWMLowGateDrainer interface {
	DrainEvents() error
}
