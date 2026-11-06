package vm

import "errors"

var errFingerRobotRealtimeUnsupported = errors.New("finger robot realtime scheduling is unsupported on this platform")

// fingerRobotRealtimeDiagnostic is intentionally an internal diagnostic
// snapshot. It is used by the worker startup log and is not part of the HTTP
// status surface.
type fingerRobotRealtimeDiagnostic struct {
	TID      int
	CPUs     []int
	Policy   uint32
	Priority uint32
	Flags    uint64
}

// fingerRobotRealtimeCleanup restores the calling thread's scheduler and
// affinity. The worker must invoke it while still locked to this OS thread.
// The returned function is idempotent.
type fingerRobotRealtimeCleanup func() error
