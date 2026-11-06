package utils

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const defaultWatchedProcessPollInterval = 100 * time.Millisecond

var (
	watchedProcessProcRoot = "/proc"
	watchedProcessReadFile = os.ReadFile
	watchedProcessSignal   = func(pid int, sig syscall.Signal) error {
		return syscall.Kill(pid, sig)
	}
)

type RestartWatchedProcessOptions struct {
	ExecPath     string
	ReadyCheck   func(context.Context) error
	PollInterval time.Duration
}

func RestartWatchedProcess(ctx context.Context, opts RestartWatchedProcessOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}

	execPath := strings.TrimSpace(opts.ExecPath)
	if execPath == "" {
		return fmt.Errorf("exec path is required")
	}
	if !filepath.IsAbs(execPath) {
		return fmt.Errorf("exec path must be absolute: %s", execPath)
	}

	pollInterval := opts.PollInterval
	if pollInterval <= 0 {
		pollInterval = defaultWatchedProcessPollInterval
	}

	oldPIDs, err := findWatchedProcessPIDs(execPath)
	if err != nil {
		return fmt.Errorf("find watched process: %w", err)
	}

	oldPIDSet := makePIDSet(oldPIDs)
	for _, pid := range oldPIDs {
		if err := watchedProcessSignal(pid, syscall.SIGTERM); err != nil && !errors.Is(err, syscall.ESRCH) {
			return fmt.Errorf("signal process %d: %w", pid, err)
		}
	}

	if err := waitForWatchedProcessExit(ctx, execPath, oldPIDSet, pollInterval); err != nil {
		return err
	}

	return waitForWatchedProcessReady(ctx, execPath, oldPIDSet, opts.ReadyCheck, pollInterval)
}

func findWatchedProcessPIDs(execPath string) ([]int, error) {
	entries, err := os.ReadDir(watchedProcessProcRoot)
	if err != nil {
		return nil, err
	}

	pids := make([]int, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 {
			continue
		}

		cmdlinePath := filepath.Join(watchedProcessProcRoot, entry.Name(), "cmdline")
		data, err := watchedProcessReadFile(cmdlinePath)
		if err != nil {
			if os.IsNotExist(err) || errors.Is(err, syscall.ESRCH) {
				continue
			}
			return nil, fmt.Errorf("read %s: %w", cmdlinePath, err)
		}

		if watchedProcessArgv0(data) == execPath {
			pids = append(pids, pid)
		}
	}

	sort.Ints(pids)
	return pids, nil
}

func watchedProcessArgv0(cmdline []byte) string {
	if len(cmdline) == 0 {
		return ""
	}

	if i := bytes.IndexByte(cmdline, 0); i >= 0 {
		return string(cmdline[:i])
	}

	return string(cmdline)
}

func waitForWatchedProcessExit(ctx context.Context, execPath string, oldPIDs map[int]struct{}, pollInterval time.Duration) error {
	if len(oldPIDs) == 0 {
		return nil
	}

	for {
		pids, err := findWatchedProcessPIDs(execPath)
		if err != nil {
			return fmt.Errorf("find watched process while waiting for exit: %w", err)
		}

		if !hasAnyPID(pids, oldPIDs) {
			return nil
		}

		if err := waitWatchedProcessPoll(ctx, pollInterval); err != nil {
			return fmt.Errorf("wait for watched process %s to exit: %w", execPath, err)
		}
	}
}

func waitForWatchedProcessReady(
	ctx context.Context,
	execPath string,
	oldPIDs map[int]struct{},
	readyCheck func(context.Context) error,
	pollInterval time.Duration,
) error {
	var lastReadyErr error

	for {
		pids, err := findWatchedProcessPIDs(execPath)
		if err != nil {
			return fmt.Errorf("find restarted process: %w", err)
		}

		if hasNewPID(pids, oldPIDs) {
			if readyCheck == nil {
				return nil
			}

			if err := readyCheck(ctx); err == nil {
				return nil
			} else {
				lastReadyErr = err
			}
		}

		if err := waitWatchedProcessPoll(ctx, pollInterval); err != nil {
			if lastReadyErr != nil {
				return fmt.Errorf("wait for watched process %s ready: %w; last ready check: %v", execPath, err, lastReadyErr)
			}
			return fmt.Errorf("wait for watched process %s to restart: %w", execPath, err)
		}
	}
}

func waitWatchedProcessPoll(ctx context.Context, pollInterval time.Duration) error {
	timer := time.NewTimer(pollInterval)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func makePIDSet(pids []int) map[int]struct{} {
	set := make(map[int]struct{}, len(pids))
	for _, pid := range pids {
		set[pid] = struct{}{}
	}
	return set
}

func hasAnyPID(pids []int, set map[int]struct{}) bool {
	for _, pid := range pids {
		if _, ok := set[pid]; ok {
			return true
		}
	}
	return false
}

func hasNewPID(pids []int, oldPIDs map[int]struct{}) bool {
	for _, pid := range pids {
		if _, old := oldPIDs[pid]; !old {
			return true
		}
	}
	return false
}
