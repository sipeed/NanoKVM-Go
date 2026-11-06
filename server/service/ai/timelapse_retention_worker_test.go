package ai

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestScreenTimelapseRetentionExecutorDeletesOldToNewAndContinuesAfterFailure(t *testing.T) {
	now := time.Date(2026, time.June, 3, 12, 0, 0, 0, time.UTC)
	root := t.TempDir()
	firstPath, firstModTime, _ := writeTimelapseMutationTestDay(t, root, "2026-06-01", now.Add(-time.Hour))
	secondPath, secondModTime, _ := writeTimelapseMutationTestDay(t, root, "2026-06-02", now.Add(-time.Hour))
	_ = firstPath
	_ = secondPath

	config := screenTimelapseRetentionTestConfig(1, 0)
	var deletedDates []string
	executor := newScreenTimelapseRetentionExecutor()
	executor.root = func() string { return root }
	executor.now = func() time.Time { return now }
	executor.loadConfig = func() (ScreenTimelapseRetentionConfig, error) { return config, nil }
	executor.cleanupTombstones = func() error { return nil }
	executor.timeSynchronized = func() (bool, error) { return true, nil }
	executor.deleteDay = func(gotRoot string, date string, gotNow time.Time, expectedConfig ScreenTimelapseRetentionConfig, expectedModTime time.Time) (int64, error) {
		if gotRoot != root || !gotNow.Equal(now) {
			t.Fatalf("unexpected deletion arguments: root=%q now=%s", gotRoot, gotNow)
		}
		if expectedConfig != config {
			t.Fatalf("expected config=%#v, want %#v", expectedConfig, config)
		}
		deletedDates = append(deletedDates, date)
		switch date {
		case "2026-06-01":
			if !expectedModTime.Equal(firstModTime) {
				t.Fatalf("first expected mtime=%s, want %s", expectedModTime, firstModTime)
			}
			return 0, errors.New("simulated first-day failure")
		case "2026-06-02":
			if !expectedModTime.Equal(secondModTime) {
				t.Fatalf("second expected mtime=%s, want %s", expectedModTime, secondModTime)
			}
			return 17, nil
		default:
			t.Fatalf("unexpected date %q", date)
			return 0, nil
		}
	}

	result := executor.execute(now)
	if got, want := strings.Join(deletedDates, ","), "2026-06-01,2026-06-02"; got != want {
		t.Fatalf("deletion order=%q, want %q", got, want)
	}
	if result.lastDeletedDays != 1 || result.lastDeletedBytes != 17 {
		t.Fatalf("deletion stats=%#v", result)
	}
	if result.err == nil || !strings.Contains(result.err.Error(), "delete 2026-06-01") {
		t.Fatalf("expected per-day error, got %v", result.err)
	}
}

func TestScreenTimelapseRetentionExecutorBlocksDeletionWhenClockIsUnsynchronized(t *testing.T) {
	now := time.Date(2026, time.June, 3, 12, 0, 0, 0, time.UTC)
	root := t.TempDir()
	writeTimelapseMutationTestDay(t, root, "2026-06-01", now.Add(-time.Hour))

	config := screenTimelapseRetentionTestConfig(1, 0)
	deleteCalls := 0
	executor := newScreenTimelapseRetentionExecutor()
	executor.root = func() string { return root }
	executor.now = func() time.Time { return now }
	executor.loadConfig = func() (ScreenTimelapseRetentionConfig, error) { return config, nil }
	executor.cleanupTombstones = func() error { return nil }
	executor.timeSynchronized = func() (bool, error) { return false, nil }
	executor.deleteDay = func(string, string, time.Time, ScreenTimelapseRetentionConfig, time.Time) (int64, error) {
		deleteCalls++
		return 0, nil
	}

	result := executor.execute(now)
	if !result.scanned || result.managedDays != 1 {
		t.Fatalf("expected inventory status, got %#v", result)
	}
	if result.blockedReason != screenTimelapseRetentionBlockedTimeUnsynchronized {
		t.Fatalf("blocked reason=%q", result.blockedReason)
	}
	if deleteCalls != 0 || result.lastDeletedDays != 0 {
		t.Fatalf("clock-unsynchronized execution deleted data: calls=%d result=%#v", deleteCalls, result)
	}
}

func TestScreenTimelapseRetentionExecutorDefaultDisabledDoesNotDelete(t *testing.T) {
	now := time.Date(2026, time.June, 3, 12, 0, 0, 0, time.UTC)
	root := t.TempDir()
	writeTimelapseMutationTestDay(t, root, "2026-06-01", now.Add(-time.Hour))

	executor := newScreenTimelapseRetentionExecutor()
	executor.root = func() string { return root }
	executor.now = func() time.Time { return now }
	executor.loadConfig = func() (ScreenTimelapseRetentionConfig, error) {
		return defaultScreenTimelapseRetentionConfig(), nil
	}
	executor.cleanupTombstones = func() error { return nil }
	executor.timeSynchronized = func() (bool, error) {
		t.Fatal("disabled cleanup must not check time synchronization")
		return false, nil
	}
	executor.deleteDay = func(string, string, time.Time, ScreenTimelapseRetentionConfig, time.Time) (int64, error) {
		t.Fatal("disabled cleanup must not delete")
		return 0, nil
	}

	result := executor.execute(now)
	if !result.scanned || result.blockedReason != screenTimelapseRetentionBlockedDisabled {
		t.Fatalf("disabled result=%#v", result)
	}
	if result.lastDeletedDays != 0 || result.lastDeletedBytes != 0 {
		t.Fatalf("disabled cleanup deleted data: %#v", result)
	}
}

func TestScreenTimelapseRetentionWorkerUsesStartupDelayAndPeriodicSchedule(t *testing.T) {
	now := time.Date(2026, time.June, 3, 12, 0, 0, 0, time.UTC)
	clock := newScreenTimelapseRetentionWorkerTestClock()
	worker := &screenTimelapseRetentionWorker{
		initialDelay: screenTimelapseRetentionInitialDelay,
		interval:     screenTimelapseRetentionRunInterval,
		now:          func() time.Time { return now },
		wait:         clock.wait,
		execute: func(runNow time.Time) screenTimelapseRetentionExecutionResult {
			return screenTimelapseRetentionExecutionResult{
				scanned:          true,
				scannedAt:        runNow,
				usageBytes:       42,
				managedDays:      2,
				overLimit:        true,
				lastCleanupAt:    runNow,
				lastDeletedDays:  1,
				lastDeletedBytes: 21,
			}
		},
	}
	worker.Start()
	worker.Start()
	t.Cleanup(worker.Stop)

	clock.expectDelay(t, screenTimelapseRetentionInitialDelay)
	clock.advance(t)
	status := waitForScreenTimelapseRetentionStatus(t, worker, func(status ScreenTimelapseRetentionStatus) bool {
		return !status.Running && status.LastTrigger == screenTimelapseRetentionTriggerStartup
	})
	if !status.ScannedAt.Equal(now) || status.UsageBytes != 42 || status.ManagedDays != 2 || !status.OverLimit || status.LastDeletedDays != 1 || status.LastDeletedBytes != 21 || !status.LastCleanupAt.Equal(now) {
		t.Fatalf("unexpected startup status: %#v", status)
	}

	clock.expectDelay(t, screenTimelapseRetentionRunInterval)
	clock.advance(t)
	status = waitForScreenTimelapseRetentionStatus(t, worker, func(status ScreenTimelapseRetentionStatus) bool {
		return !status.Running && status.LastTrigger == screenTimelapseRetentionTriggerScheduled
	})
	if status.LastError != "" {
		t.Fatalf("scheduled status has unexpected error: %#v", status)
	}

	worker.Stop()
	worker.Stop()
	if worker.Trigger(screenTimelapseRetentionTriggerManual) {
		t.Fatal("stopped worker accepted a trigger")
	}
}

func TestScreenTimelapseRetentionWorkerUsesThirtyMinuteInterval(t *testing.T) {
	if screenTimelapseRetentionRunInterval != 30*time.Minute {
		t.Fatalf("retention interval=%s, want %s", screenTimelapseRetentionRunInterval, 30*time.Minute)
	}
}

func TestScreenTimelapseRetentionWorkerCoalescesPendingTriggersAndRunsSingleFlight(t *testing.T) {
	clock := newScreenTimelapseRetentionWorkerTestClock()
	firstRunStarted := make(chan struct{})
	allowFirstRun := make(chan struct{})
	started := make(chan int, 3)
	finished := make(chan int, 3)
	var runCount atomic.Int32
	var inFlight atomic.Int32
	var maxInFlight atomic.Int32

	worker := &screenTimelapseRetentionWorker{
		initialDelay: screenTimelapseRetentionInitialDelay,
		interval:     screenTimelapseRetentionRunInterval,
		now:          time.Now,
		wait:         clock.wait,
		execute: func(time.Time) screenTimelapseRetentionExecutionResult {
			currentInFlight := inFlight.Add(1)
			for {
				previousMax := maxInFlight.Load()
				if currentInFlight <= previousMax || maxInFlight.CompareAndSwap(previousMax, currentInFlight) {
					break
				}
			}
			defer inFlight.Add(-1)

			run := int(runCount.Add(1))
			started <- run
			if run == 1 {
				close(firstRunStarted)
				<-allowFirstRun
			}
			finished <- run
			return screenTimelapseRetentionExecutionResult{scanned: true, scannedAt: time.Now()}
		},
	}
	worker.Start()
	t.Cleanup(worker.Stop)
	clock.expectDelay(t, screenTimelapseRetentionInitialDelay)

	if !worker.Trigger(screenTimelapseRetentionTriggerManual) {
		t.Fatal("manual trigger was not accepted")
	}
	waitForScreenTimelapseRetentionSignal(t, firstRunStarted, "first retention run")
	if run := waitForScreenTimelapseRetentionInt(t, started, "first start"); run != 1 {
		t.Fatalf("first run=%d", run)
	}
	if !worker.Trigger(screenTimelapseRetentionTriggerScheduled) || !worker.Trigger(screenTimelapseRetentionTriggerConfigSaved) {
		t.Fatal("pending triggers were not accepted")
	}
	close(allowFirstRun)
	if run := waitForScreenTimelapseRetentionInt(t, finished, "first completion"); run != 1 {
		t.Fatalf("first completion=%d", run)
	}
	if run := waitForScreenTimelapseRetentionInt(t, started, "coalesced follow-up start"); run != 2 {
		t.Fatalf("coalesced run=%d, want 2", run)
	}
	if run := waitForScreenTimelapseRetentionInt(t, finished, "coalesced follow-up completion"); run != 2 {
		t.Fatalf("coalesced completion=%d", run)
	}
	select {
	case run := <-started:
		t.Fatalf("unexpected third run %d", run)
	case <-time.After(50 * time.Millisecond):
	}
	if maxInFlight.Load() != 1 {
		t.Fatalf("maximum concurrent runs=%d, want 1", maxInFlight.Load())
	}
}

func TestScreenTimelapseRetentionWorkerDoesNotRestartUntilStopCompletes(t *testing.T) {
	initialWaitStarted := make(chan struct{}, 2)
	cancellationObserved := make(chan struct{}, 2)
	firstRunStarted := make(chan struct{})
	releaseFirstRun := make(chan struct{})
	var runs atomic.Int32

	worker := &screenTimelapseRetentionWorker{
		initialDelay: screenTimelapseRetentionInitialDelay,
		interval:     screenTimelapseRetentionRunInterval,
		now:          time.Now,
		wait: func(ctx context.Context, _ time.Duration) error {
			select {
			case initialWaitStarted <- struct{}{}:
			default:
			}
			<-ctx.Done()
			select {
			case cancellationObserved <- struct{}{}:
			default:
			}
			return ctx.Err()
		},
		execute: func(time.Time) screenTimelapseRetentionExecutionResult {
			if runs.Add(1) == 1 {
				close(firstRunStarted)
				<-releaseFirstRun
			}
			return screenTimelapseRetentionExecutionResult{scanned: true, scannedAt: time.Now()}
		},
	}
	worker.Start()
	t.Cleanup(worker.Stop)
	waitForScreenTimelapseRetentionSignal(t, initialWaitStarted, "initial worker wait")
	if !worker.Trigger(screenTimelapseRetentionTriggerManual) {
		t.Fatal("manual trigger was not accepted")
	}
	waitForScreenTimelapseRetentionSignal(t, firstRunStarted, "first retention run")

	stopped := make(chan struct{})
	go func() {
		worker.Stop()
		close(stopped)
	}()
	waitForScreenTimelapseRetentionSignal(t, cancellationObserved, "worker cancellation")

	startReturned := make(chan struct{})
	go func() {
		worker.Start()
		close(startReturned)
	}()
	select {
	case <-startReturned:
		t.Fatal("Start returned before the in-flight worker finished stopping")
	case <-time.After(50 * time.Millisecond):
	}

	close(releaseFirstRun)
	waitForScreenTimelapseRetentionSignal(t, stopped, "worker stop completion")
	waitForScreenTimelapseRetentionSignal(t, startReturned, "worker restart after stop")
	worker.Stop()
}

func TestScreenTimelapseRetentionConfigSaveNotifiesWorker(t *testing.T) {
	previousNotify := notifyScreenTimelapseRetentionConfigSaved
	notified := make(chan struct{}, 1)
	notifyScreenTimelapseRetentionConfigSaved = func() bool {
		notified <- struct{}{}
		return true
	}
	t.Cleanup(func() { notifyScreenTimelapseRetentionConfigSaved = previousNotify })

	store := newScreenTimelapseRetentionConfigStore(filepath.Join(t.TempDir(), "etc", "timelapse.json"))
	config := defaultScreenTimelapseRetentionConfig()
	if err := store.Save(config); err != nil {
		t.Fatalf("save retention config: %v", err)
	}
	waitForScreenTimelapseRetentionSignal(t, notified, "config-save worker notification")

	invalid := config
	invalid.Cleanup.Enabled = true
	invalid.Cleanup.RetentionDays = 0
	invalid.Cleanup.MaxStorageMiB = 0
	if err := store.Save(invalid); err == nil {
		t.Fatal("invalid save unexpectedly succeeded")
	}
	select {
	case <-notified:
		t.Fatal("failed config save notified the worker")
	default:
	}
}

type screenTimelapseRetentionWorkerTestClock struct {
	delays   chan time.Duration
	advances chan struct{}
}

func newScreenTimelapseRetentionWorkerTestClock() *screenTimelapseRetentionWorkerTestClock {
	return &screenTimelapseRetentionWorkerTestClock{
		delays:   make(chan time.Duration, 4),
		advances: make(chan struct{}, 4),
	}
}

func (clock *screenTimelapseRetentionWorkerTestClock) wait(ctx context.Context, delay time.Duration) error {
	select {
	case clock.delays <- delay:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case <-clock.advances:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (clock *screenTimelapseRetentionWorkerTestClock) expectDelay(t *testing.T, want time.Duration) {
	t.Helper()
	select {
	case got := <-clock.delays:
		if got != want {
			t.Fatalf("delay=%s, want %s", got, want)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for delay %s", want)
	}
}

func (clock *screenTimelapseRetentionWorkerTestClock) advance(t *testing.T) {
	t.Helper()
	select {
	case clock.advances <- struct{}{}:
	case <-time.After(time.Second):
		t.Fatal("timed out advancing retention worker clock")
	}
}

func waitForScreenTimelapseRetentionStatus(t *testing.T, worker *screenTimelapseRetentionWorker, ready func(ScreenTimelapseRetentionStatus) bool) ScreenTimelapseRetentionStatus {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		status := worker.Status()
		if ready(status) {
			return status
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for retention worker status: %#v", worker.Status())
	return ScreenTimelapseRetentionStatus{}
}

func waitForScreenTimelapseRetentionSignal(t *testing.T, signal <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", description)
	}
}

func waitForScreenTimelapseRetentionInt(t *testing.T, values <-chan int, description string) int {
	t.Helper()
	select {
	case value := <-values:
		return value
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", description)
		return 0
	}
}
