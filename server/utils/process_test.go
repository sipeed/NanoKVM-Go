package utils

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"syscall"
	"testing"
	"time"
)

func TestFindWatchedProcessPIDsMatchesArgv0Exactly(t *testing.T) {
	procRoot := t.TempDir()
	target := "/var/lib/nanokvm-runtime/kvmcomm/vin/kvm_vin"

	writeFakeCmdline(t, procRoot, 101, target, "--debug")
	writeFakeCmdline(t, procRoot, 102, target+"_helper")
	writeFakeCmdline(t, procRoot, 103, "/bin/sh", target)
	writeFakeCmdline(t, procRoot, 104)
	writeFakeCmdline(t, procRoot, 105, target+"-old")
	if err := os.Mkdir(filepath.Join(procRoot, "not-a-pid"), 0o755); err != nil {
		t.Fatalf("mkdir fake non-pid proc entry: %v", err)
	}

	restore := replaceWatchedProcessProcRoot(procRoot)
	defer restore()

	pids, err := findWatchedProcessPIDs(target)
	if err != nil {
		t.Fatalf("find watched process pids: %v", err)
	}

	if !reflect.DeepEqual(pids, []int{101}) {
		t.Fatalf("expected only exact argv0 match, got %#v", pids)
	}
}

func TestRestartWatchedProcessSignalsOldPIDAndWaitsForReady(t *testing.T) {
	procRoot := t.TempDir()
	target := "/var/lib/nanokvm-runtime/kvmcomm/vin/kvm_vin"
	writeFakeCmdline(t, procRoot, 100, target)

	restoreProc := replaceWatchedProcessProcRoot(procRoot)
	defer restoreProc()

	var signaled []int
	restoreSignal := replaceWatchedProcessSignal(func(pid int, sig syscall.Signal) error {
		if sig != syscall.SIGTERM {
			t.Fatalf("expected SIGTERM, got %v", sig)
		}
		signaled = append(signaled, pid)
		removeFakeProc(t, procRoot, pid)
		writeFakeCmdline(t, procRoot, 200, target)
		return nil
	})
	defer restoreSignal()

	readyCalls := 0
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := RestartWatchedProcess(ctx, RestartWatchedProcessOptions{
		ExecPath:     target,
		PollInterval: time.Millisecond,
		ReadyCheck: func(context.Context) error {
			readyCalls++
			return nil
		},
	})
	if err != nil {
		t.Fatalf("restart watched process: %v", err)
	}

	if !reflect.DeepEqual(signaled, []int{100}) {
		t.Fatalf("expected old PID signaled once, got %#v", signaled)
	}
	if readyCalls == 0 {
		t.Fatalf("expected ready check to be called")
	}
}

func TestRestartWatchedProcessWaitsWhenOldPIDMissing(t *testing.T) {
	procRoot := t.TempDir()
	target := "/var/lib/nanokvm-runtime/kvmcomm/vin/kvm_vin"

	restoreProc := replaceWatchedProcessProcRoot(procRoot)
	defer restoreProc()

	restoreSignal := replaceWatchedProcessSignal(func(pid int, sig syscall.Signal) error {
		t.Fatalf("did not expect signal for missing old PID")
		return nil
	})
	defer restoreSignal()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	readyCalls := 0
	go func() {
		time.Sleep(5 * time.Millisecond)
		writeFakeCmdlineNoFail(procRoot, 300, target)
	}()

	err := RestartWatchedProcess(ctx, RestartWatchedProcessOptions{
		ExecPath:     target,
		PollInterval: time.Millisecond,
		ReadyCheck: func(context.Context) error {
			readyCalls++
			return nil
		},
	})
	if err != nil {
		t.Fatalf("restart watched process without old pid: %v", err)
	}
	if readyCalls == 0 {
		t.Fatalf("expected ready check after new PID appeared")
	}
}

func TestRestartWatchedProcessReturnsReadyTimeout(t *testing.T) {
	procRoot := t.TempDir()
	target := "/var/lib/nanokvm-runtime/kvmcomm/vin/kvm_vin"
	writeFakeCmdline(t, procRoot, 100, target)

	restoreProc := replaceWatchedProcessProcRoot(procRoot)
	defer restoreProc()

	restoreSignal := replaceWatchedProcessSignal(func(pid int, sig syscall.Signal) error {
		removeFakeProc(t, procRoot, pid)
		writeFakeCmdline(t, procRoot, 200, target)
		return nil
	})
	defer restoreSignal()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := RestartWatchedProcess(ctx, RestartWatchedProcessOptions{
		ExecPath:     target,
		PollInterval: time.Millisecond,
		ReadyCheck: func(context.Context) error {
			return errors.New("not ready")
		},
	})
	if err == nil {
		t.Fatalf("expected ready timeout error")
	}
}

func replaceWatchedProcessProcRoot(procRoot string) func() {
	old := watchedProcessProcRoot
	watchedProcessProcRoot = procRoot
	return func() {
		watchedProcessProcRoot = old
	}
}

func replaceWatchedProcessSignal(fn func(int, syscall.Signal) error) func() {
	old := watchedProcessSignal
	watchedProcessSignal = fn
	return func() {
		watchedProcessSignal = old
	}
}

func writeFakeCmdline(t *testing.T, procRoot string, pid int, argv ...string) {
	t.Helper()

	dir := filepath.Join(procRoot, intString(pid))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir fake proc dir: %v", err)
	}

	data := []byte{}
	for _, arg := range argv {
		data = append(data, []byte(arg)...)
		data = append(data, 0)
	}

	if err := os.WriteFile(filepath.Join(dir, "cmdline"), data, 0o644); err != nil {
		t.Fatalf("write fake cmdline: %v", err)
	}
}

func writeFakeCmdlineNoFail(procRoot string, pid int, argv ...string) {
	dir := filepath.Join(procRoot, intString(pid))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}

	data := []byte{}
	for _, arg := range argv {
		data = append(data, []byte(arg)...)
		data = append(data, 0)
	}

	_ = os.WriteFile(filepath.Join(dir, "cmdline"), data, 0o644)
}

func removeFakeProc(t *testing.T, procRoot string, pid int) {
	t.Helper()
	if err := os.RemoveAll(filepath.Join(procRoot, intString(pid))); err != nil {
		t.Fatalf("remove fake proc dir: %v", err)
	}
}

func intString(v int) string {
	return strconv.Itoa(v)
}
