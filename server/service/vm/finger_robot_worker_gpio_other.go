//go:build !linux

package vm

import (
	"errors"
	"runtime"
)

func newRealtimeFingerRobotPWMLowGate() (fingerRobotPWMLowGate, error) {
	return nil, errors.Join(errFingerRobotRealtimeUnsupported, errors.New("GPIO event capture is unavailable on "+runtime.GOOS))
}
