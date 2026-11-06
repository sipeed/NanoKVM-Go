//go:build linux

package vm

import (
	"errors"
	"fmt"
	"sync"

	"golang.org/x/sys/unix"
)

const (
	fingerRobotRealtimePolicy    = unix.SCHED_FIFO
	fingerRobotRealtimeResetFork = unix.SCHED_FLAG_RESET_ON_FORK
)

type fingerRobotRealtimeState struct {
	affinity unix.CPUSet
	sched    unix.SchedAttr
}

// setupFingerRobotRealtime must be called by the worker goroutine after it
// has called runtime.LockOSThread.  pid 0 in the scheduler APIs means the
// calling Linux thread, rather than the whole Go process.
func setupFingerRobotRealtime(cpu, priority int) (fingerRobotRealtimeCleanup, error) {
	if cpu < 0 {
		return nil, fmt.Errorf("configure finger robot realtime CPU %d: %w", cpu, errors.New("CPU must be non-negative"))
	}
	if priority < 1 || priority > 99 {
		return nil, fmt.Errorf("configure finger robot realtime priority %d: %w", priority, errors.New("SCHED_FIFO priority must be between 1 and 99"))
	}

	state, err := captureFingerRobotRealtimeState()
	if err != nil {
		return nil, fmt.Errorf("capture current thread realtime state: %w", err)
	}
	cleanup := newFingerRobotRealtimeCleanup(state)

	var affinity unix.CPUSet
	affinity.Zero()
	affinity.Set(cpu)
	if err := unix.SchedSetaffinity(0, &affinity); err != nil {
		return nil, errors.Join(
			fmt.Errorf("set finger robot realtime CPU affinity to CPU %d: %w", cpu, err),
			cleanup(),
		)
	}

	attr := &unix.SchedAttr{
		Policy:   fingerRobotRealtimePolicy,
		Flags:    fingerRobotRealtimeResetFork,
		Priority: uint32(priority),
	}
	if err := unix.SchedSetAttr(0, attr, 0); err != nil {
		return nil, errors.Join(
			fmt.Errorf("set finger robot realtime scheduler to SCHED_FIFO priority %d: %w", priority, err),
			cleanup(),
		)
	}

	diagnostic, err := getFingerRobotRealtimeDiagnostic()
	if err != nil {
		return nil, errors.Join(fmt.Errorf("verify finger robot realtime scheduler: %w", err), cleanup())
	}
	if !containsOnlyCPU(diagnostic.CPUs, cpu) {
		return nil, errors.Join(
			fmt.Errorf(
				"verify finger robot realtime CPU affinity: expected only CPU %d, got %v",
				cpu,
				diagnostic.CPUs,
			),
			cleanup(),
		)
	}
	if diagnostic.Policy != fingerRobotRealtimePolicy ||
		diagnostic.Priority != uint32(priority) ||
		diagnostic.Flags&fingerRobotRealtimeResetFork == 0 {
		return nil, errors.Join(
			fmt.Errorf(
				"verify finger robot realtime scheduler: expected policy=%d priority=%d reset-on-fork=true, got policy=%d priority=%d flags=0x%x",
				fingerRobotRealtimePolicy,
				priority,
				diagnostic.Policy,
				diagnostic.Priority,
				diagnostic.Flags,
			),
			cleanup(),
		)
	}

	return cleanup, nil
}

func captureFingerRobotRealtimeState() (fingerRobotRealtimeState, error) {
	var affinity unix.CPUSet
	if err := unix.SchedGetaffinity(0, &affinity); err != nil {
		return fingerRobotRealtimeState{}, fmt.Errorf("read thread CPU affinity: %w", err)
	}
	sched, err := unix.SchedGetAttr(0, 0)
	if err != nil {
		return fingerRobotRealtimeState{}, fmt.Errorf("read thread scheduler: %w", err)
	}
	return fingerRobotRealtimeState{affinity: affinity, sched: *sched}, nil
}

func newFingerRobotRealtimeCleanup(state fingerRobotRealtimeState) fingerRobotRealtimeCleanup {
	var once sync.Once
	var cleanupErr error
	return func() error {
		once.Do(func() {
			var errs []error
			if err := unix.SchedSetAttr(0, &state.sched, 0); err != nil {
				errs = append(errs, fmt.Errorf("restore thread scheduler: %w", err))
			}
			if err := unix.SchedSetaffinity(0, &state.affinity); err != nil {
				errs = append(errs, fmt.Errorf("restore thread CPU affinity: %w", err))
			}
			cleanupErr = errors.Join(errs...)
		})
		return cleanupErr
	}
}

func getFingerRobotRealtimeThreadID() int {
	return unix.Gettid()
}

func getFingerRobotRealtimeDiagnostic() (fingerRobotRealtimeDiagnostic, error) {
	var affinity unix.CPUSet
	if err := unix.SchedGetaffinity(0, &affinity); err != nil {
		return fingerRobotRealtimeDiagnostic{}, fmt.Errorf("read thread CPU affinity: %w", err)
	}

	attr, err := unix.SchedGetAttr(0, 0)
	if err != nil {
		return fingerRobotRealtimeDiagnostic{}, fmt.Errorf("read thread scheduler: %w", err)
	}

	return fingerRobotRealtimeDiagnostic{
		TID:      getFingerRobotRealtimeThreadID(),
		CPUs:     cpuSetCPUs(&affinity),
		Policy:   attr.Policy,
		Priority: attr.Priority,
		Flags:    attr.Flags,
	}, nil
}

func cpuSetCPUs(set *unix.CPUSet) []int {
	if set == nil {
		return nil
	}
	cpus := make([]int, 0)
	for cpu := 0; cpu < 1024; cpu++ {
		if set.IsSet(cpu) {
			cpus = append(cpus, cpu)
		}
	}
	return cpus
}

func containsOnlyCPU(cpus []int, wanted int) bool {
	return len(cpus) == 1 && cpus[0] == wanted
}
