//go:build !linux

package vm

import (
	"fmt"
	"runtime"
)

func setupFingerRobotRealtime(cpu, priority int) (fingerRobotRealtimeCleanup, error) {
	return nil, fmt.Errorf("configure finger robot realtime on %s: %w", runtime.GOOS, errFingerRobotRealtimeUnsupported)
}

func getFingerRobotRealtimeThreadID() int {
	return -1
}

func getFingerRobotRealtimeDiagnostic() (fingerRobotRealtimeDiagnostic, error) {
	return fingerRobotRealtimeDiagnostic{}, fmt.Errorf("get finger robot realtime diagnostic on %s: %w", runtime.GOOS, errFingerRobotRealtimeUnsupported)
}
