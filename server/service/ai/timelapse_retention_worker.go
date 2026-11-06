package ai

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"NanoKVM-Server/utils"

	log "github.com/sirupsen/logrus"
)

const (
	screenTimelapseRetentionInitialDelay = 5 * time.Minute
	screenTimelapseRetentionRunInterval  = 30 * time.Minute

	screenTimelapseRetentionTriggerStartup     = "startup"
	screenTimelapseRetentionTriggerScheduled   = "scheduled"
	screenTimelapseRetentionTriggerConfigSaved = "config_saved"
	screenTimelapseRetentionTriggerManual      = "manual"

	screenTimelapseRetentionBlockedDisabled           = "disabled"
	screenTimelapseRetentionBlockedTimeUnsynchronized = "time_unsynchronized"
	screenTimelapseRetentionBlockedInvalidConfig      = "invalid_config"
)

// ScreenTimelapseRetentionStatus is the latest in-memory worker snapshot.
// Step 5 exposes this value without performing another recursive inventory.
type ScreenTimelapseRetentionStatus struct {
	Running          bool      `json:"running"`
	ScannedAt        time.Time `json:"scannedAt"`
	UsageBytes       int64     `json:"usageBytes"`
	ManagedDays      int       `json:"managedDays"`
	OverLimit        bool      `json:"overLimit"`
	BlockedReason    string    `json:"blockedReason"`
	LastCleanupAt    time.Time `json:"lastCleanupAt"`
	LastTrigger      string    `json:"lastTrigger"`
	LastDeletedDays  int       `json:"lastDeletedDays"`
	LastDeletedBytes int64     `json:"lastDeletedBytes"`
	LastError        string    `json:"lastError"`
}

type screenTimelapseRetentionExecutionResult struct {
	scanned          bool
	scannedAt        time.Time
	usageBytes       int64
	managedDays      int
	overLimit        bool
	blockedReason    string
	lastCleanupAt    time.Time
	lastDeletedDays  int
	lastDeletedBytes int64
	err              error
}

type screenTimelapseRetentionExecutor struct {
	root              func() string
	now               func() time.Time
	loadConfig        func() (ScreenTimelapseRetentionConfig, error)
	inventory         func(string, time.Time) (ScreenTimelapseRetentionInventory, error)
	plan              func(ScreenTimelapseRetentionInventory, time.Time, ScreenTimelapseRetentionConfig) (ScreenTimelapseRetentionPlan, error)
	cleanupTombstones func() error
	deleteDay         func(string, string, time.Time, ScreenTimelapseRetentionConfig, time.Time) (int64, error)
	timeSynchronized  func() (bool, error)
}

func newScreenTimelapseRetentionExecutor() *screenTimelapseRetentionExecutor {
	return &screenTimelapseRetentionExecutor{
		root:              timelapseRoot,
		now:               time.Now,
		loadConfig:        loadScreenTimelapseRetentionConfig,
		inventory:         inventoryScreenTimelapseRetention,
		plan:              planScreenTimelapseRetention,
		cleanupTombstones: CleanupScreenTimelapseTombstones,
		deleteDay:         deleteScreenTimelapseRetentionDay,
		timeSynchronized:  screenTimelapseRetentionTimeSynchronized,
	}
}

func screenTimelapseRetentionTimeSynchronized() (bool, error) {
	status, err := utils.GetChronySyncStatus()
	if err != nil {
		return false, err
	}
	return status != nil && status.IsSynchronized, nil
}

func (executor *screenTimelapseRetentionExecutor) execute(now time.Time) screenTimelapseRetentionExecutionResult {
	result := screenTimelapseRetentionExecutionResult{scannedAt: now}

	if err := executor.cleanupTombstones(); err != nil {
		result.err = appendScreenTimelapseRetentionExecutionError(result.err, "clean leftover tombstones", err)
	}

	config, configErr := executor.loadConfig()
	if configErr != nil {
		result.err = appendScreenTimelapseRetentionExecutionError(result.err, "load retention config", configErr)
		config = defaultScreenTimelapseRetentionConfig()
		result.blockedReason = screenTimelapseRetentionBlockedInvalidConfig
	}

	root := executor.root()
	inventory, err := executor.inventory(root, now)
	if err != nil {
		result.err = appendScreenTimelapseRetentionExecutionError(result.err, "inventory timelapse", err)
		return result
	}
	result.scanned = true
	result.usageBytes = inventory.TotalBytes
	result.managedDays = len(inventory.Days)

	plan, err := executor.plan(inventory, now, config)
	if err != nil {
		result.err = appendScreenTimelapseRetentionExecutionError(result.err, "plan retention", err)
		return result
	}
	result.overLimit = screenTimelapseRetentionUsageOverLimit(config, result.usageBytes)

	if configErr != nil {
		return result
	}
	if !config.Cleanup.Enabled {
		result.blockedReason = screenTimelapseRetentionBlockedDisabled
		return result
	}
	if len(plan.Candidates) == 0 {
		return result
	}

	synchronized, err := executor.timeSynchronized()
	if err != nil {
		result.err = appendScreenTimelapseRetentionExecutionError(result.err, "check time synchronization", err)
		result.blockedReason = screenTimelapseRetentionBlockedTimeUnsynchronized
		return result
	}
	if !synchronized {
		result.blockedReason = screenTimelapseRetentionBlockedTimeUnsynchronized
		return result
	}

	daysByDate := make(map[string]ScreenTimelapseRetentionInventoryDay, len(inventory.Days))
	for _, day := range inventory.Days {
		daysByDate[day.Date] = day
	}
	candidates := append([]ScreenTimelapseRetentionPlanDay(nil), plan.Candidates...)
	sort.Slice(candidates, func(left, right int) bool {
		return candidates[left].Date < candidates[right].Date
	})

	for _, candidate := range candidates {
		day, ok := daysByDate[candidate.Date]
		if !ok {
			result.err = appendScreenTimelapseRetentionExecutionError(
				result.err,
				"revalidate planned day "+candidate.Date,
				errors.New("day is missing from inventory"),
			)
			continue
		}

		deleteNow := executor.now()
		deletedBytes, deleteErr := executor.deleteDay(root, candidate.Date, deleteNow, config, day.ModTime)
		if deleteErr != nil {
			result.err = appendScreenTimelapseRetentionExecutionError(
				result.err,
				"delete "+candidate.Date,
				deleteErr,
			)
			continue
		}

		result.lastDeletedDays++
		result.lastDeletedBytes += deletedBytes
	}

	if result.lastDeletedDays > 0 {
		result.lastCleanupAt = executor.now()
		result.usageBytes = subtractScreenTimelapseRetentionBytes(result.usageBytes, result.lastDeletedBytes)
		if result.lastDeletedDays >= result.managedDays {
			result.managedDays = 0
		} else {
			result.managedDays -= result.lastDeletedDays
		}
		result.overLimit = screenTimelapseRetentionUsageOverLimit(config, result.usageBytes)
	}
	return result
}

func screenTimelapseRetentionUsageOverLimit(config ScreenTimelapseRetentionConfig, usageBytes int64) bool {
	return config.Cleanup.Enabled && config.Cleanup.MaxStorageMiB > 0 && usageBytes > screenTimelapseRetentionStorageBytes(config.Cleanup.MaxStorageMiB)
}

func appendScreenTimelapseRetentionExecutionError(current error, operation string, err error) error {
	if err == nil {
		return current
	}
	message := fmt.Sprintf("%s: %v", operation, err)
	if current == nil {
		return errors.New(message)
	}
	return errors.New(current.Error() + "; " + message)
}

type screenTimelapseRetentionWorker struct {
	lifecycleMu sync.Mutex
	statusMu    sync.RWMutex

	initialDelay time.Duration
	interval     time.Duration
	now          func() time.Time
	wait         func(context.Context, time.Duration) error
	execute      func(time.Time) screenTimelapseRetentionExecutionResult

	cancel    context.CancelFunc
	done      chan struct{}
	triggerCh chan string
	status    ScreenTimelapseRetentionStatus
}

func newScreenTimelapseRetentionWorker(executor *screenTimelapseRetentionExecutor) *screenTimelapseRetentionWorker {
	return &screenTimelapseRetentionWorker{
		initialDelay: screenTimelapseRetentionInitialDelay,
		interval:     screenTimelapseRetentionRunInterval,
		now:          time.Now,
		wait:         waitScreenTimelapseRetentionWorker,
		execute:      executor.execute,
	}
}

func waitScreenTimelapseRetentionWorker(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (worker *screenTimelapseRetentionWorker) Start() {
	worker.lifecycleMu.Lock()
	if worker.cancel != nil {
		worker.lifecycleMu.Unlock()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	triggerCh := make(chan string, 1)
	worker.cancel = cancel
	worker.done = done
	worker.triggerCh = triggerCh
	worker.lifecycleMu.Unlock()

	go worker.run(ctx, done, triggerCh)
}

func (worker *screenTimelapseRetentionWorker) Stop() {
	worker.lifecycleMu.Lock()
	if worker.cancel == nil {
		worker.lifecycleMu.Unlock()
		return
	}
	cancel := worker.cancel
	done := worker.done
	cancel()
	<-done
	worker.cancel = nil
	worker.done = nil
	worker.triggerCh = nil
	worker.lifecycleMu.Unlock()
}

// Trigger enqueues at most one pending follow-up execution. A trigger that
// arrives while another one is pending is intentionally coalesced.
func (worker *screenTimelapseRetentionWorker) Trigger(source string) bool {
	if source == "" {
		return false
	}

	worker.lifecycleMu.Lock()
	triggerCh := worker.triggerCh
	if triggerCh == nil {
		worker.lifecycleMu.Unlock()
		return false
	}

	select {
	case triggerCh <- source:
	default:
	}
	worker.lifecycleMu.Unlock()
	return true
}

func (worker *screenTimelapseRetentionWorker) Status() ScreenTimelapseRetentionStatus {
	worker.statusMu.RLock()
	defer worker.statusMu.RUnlock()
	return worker.status
}

func (worker *screenTimelapseRetentionWorker) run(ctx context.Context, done chan struct{}, triggerCh chan string) {
	defer close(done)

	schedulerDone := make(chan struct{})
	go func() {
		defer close(schedulerDone)
		if worker.wait(ctx, worker.initialDelay) != nil {
			return
		}
		worker.enqueueScheduledTrigger(ctx, triggerCh, screenTimelapseRetentionTriggerStartup)
		for {
			if worker.wait(ctx, worker.interval) != nil {
				return
			}
			worker.enqueueScheduledTrigger(ctx, triggerCh, screenTimelapseRetentionTriggerScheduled)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			<-schedulerDone
			return
		default:
		}

		select {
		case <-ctx.Done():
			<-schedulerDone
			return
		case source := <-triggerCh:
			worker.executeRun(source)
		}
	}
}

func (worker *screenTimelapseRetentionWorker) enqueueScheduledTrigger(ctx context.Context, triggerCh chan string, source string) {
	select {
	case <-ctx.Done():
		return
	case triggerCh <- source:
	default:
	}
}

func (worker *screenTimelapseRetentionWorker) executeRun(source string) {
	worker.statusMu.Lock()
	worker.status.Running = true
	worker.status.LastTrigger = source
	worker.statusMu.Unlock()

	result := worker.execute(worker.now())

	worker.statusMu.Lock()
	worker.status.Running = false
	worker.status.LastTrigger = source
	worker.status.ScannedAt = result.scannedAt
	worker.status.LastDeletedDays = result.lastDeletedDays
	worker.status.LastDeletedBytes = result.lastDeletedBytes
	worker.status.BlockedReason = result.blockedReason
	if result.scanned {
		worker.status.UsageBytes = result.usageBytes
		worker.status.ManagedDays = result.managedDays
		worker.status.OverLimit = result.overLimit
	}
	if !result.lastCleanupAt.IsZero() {
		worker.status.LastCleanupAt = result.lastCleanupAt
	}
	if result.err != nil {
		worker.status.LastError = result.err.Error()
	} else {
		worker.status.LastError = ""
	}
	worker.statusMu.Unlock()

	if result.err != nil {
		log.WithError(result.err).Error("screen timelapse retention cleanup failed")
	}
}

var defaultScreenTimelapseRetentionWorker = newScreenTimelapseRetentionWorker(newScreenTimelapseRetentionExecutor())

func StartScreenTimelapseRetentionWorker() {
	defaultScreenTimelapseRetentionWorker.Start()
}

func StopScreenTimelapseRetentionWorker() {
	defaultScreenTimelapseRetentionWorker.Stop()
}

func GetScreenTimelapseRetentionStatus() ScreenTimelapseRetentionStatus {
	return defaultScreenTimelapseRetentionWorker.Status()
}

// TriggerScreenTimelapseRetentionCleanup is used by the Step 5 API after its
// sensitive-operation checks accept an immediate cleanup request.
func TriggerScreenTimelapseRetentionCleanup() bool {
	return defaultScreenTimelapseRetentionWorker.Trigger(screenTimelapseRetentionTriggerManual)
}

var notifyScreenTimelapseRetentionConfigSaved = func() bool {
	return defaultScreenTimelapseRetentionWorker.Trigger(screenTimelapseRetentionTriggerConfigSaved)
}
