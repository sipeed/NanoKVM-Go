package ai

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	timelapseMutationLockFile    = "/run/kvm/timelapse.lock"
	timelapseMutationLockTimeout = 2 * time.Second
	timelapseMutationLockRetry   = 25 * time.Millisecond
)

var (
	errTimelapseDayBusy                  = errors.New("timelapse day is busy")
	errTimelapseDayPathUnsafe            = errors.New("timelapse day path is unsafe")
	errTimelapseRetentionCleanupDisabled = errors.New("screen timelapse retention cleanup is disabled")
	errTimelapseRetentionConfigChanged   = errors.New("screen timelapse retention config changed")
)

type timelapseDeletionMode uint8

const (
	timelapseDeletionManual timelapseDeletionMode = iota
	timelapseDeletionAutomatic
)

type timelapseDeletionRequest struct {
	Root            string
	Date            string
	Now             time.Time
	Mode            timelapseDeletionMode
	ExpectedConfig  *ScreenTimelapseRetentionConfig
	ExpectedModTime *time.Time
}

func isTimelapseDeletionDateAllowed(request timelapseDeletionRequest) bool {
	switch request.Mode {
	case timelapseDeletionManual:
		return isDeletableTimelapseDate(request.Date, request.Now)
	case timelapseDeletionAutomatic:
		// Retention cleanup must never remove the day still being recorded.
		return isDeletableTimelapseDate(request.Date, request.Now) &&
			!isCurrentTimelapseDate(request.Date, request.Now)
	default:
		return false
	}
}

type timelapseFileLock struct {
	file *os.File
}

func lockTimelapseFile(path string, timeout time.Duration) (*timelapseFileLock, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}

	deadline := time.Now().Add(timeout)
	for {
		err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return &timelapseFileLock{file: file}, nil
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) && !errors.Is(err, syscall.EAGAIN) {
			_ = file.Close()
			return nil, err
		}
		if timeout <= 0 || !time.Now().Before(deadline) {
			_ = file.Close()
			return nil, fmt.Errorf("%w: %s", errTimelapseDayBusy, path)
		}

		wait := time.Until(deadline)
		if wait > timelapseMutationLockRetry {
			wait = timelapseMutationLockRetry
		}
		time.Sleep(wait)
	}
}

func (lock *timelapseFileLock) Close() error {
	if lock == nil || lock.file == nil {
		return nil
	}
	file := lock.file
	lock.file = nil
	unlockErr := syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	closeErr := file.Close()
	if unlockErr != nil {
		return unlockErr
	}
	return closeErr
}

// timelapsePlaybackLeases makes the transition from "no active player" to
// "the date has been reserved for deletion" atomic. A request either owns a
// lease before it starts examining the file, or it observes a pending deletion
// and fails without racing the rename.
type timelapsePlaybackLeases struct {
	mu       sync.Mutex
	active   map[string]int
	deleting map[string]bool
}

func newTimelapsePlaybackLeases() *timelapsePlaybackLeases {
	return &timelapsePlaybackLeases{
		active:   make(map[string]int),
		deleting: make(map[string]bool),
	}
}

func (leases *timelapsePlaybackLeases) acquire(date string) (func(), bool) {
	leases.mu.Lock()
	defer leases.mu.Unlock()
	if leases.deleting[date] {
		return nil, false
	}
	leases.active[date]++

	var once sync.Once
	return func() {
		once.Do(func() {
			leases.mu.Lock()
			defer leases.mu.Unlock()
			if leases.active[date] <= 1 {
				delete(leases.active, date)
				return
			}
			leases.active[date]--
		})
	}, true
}

func (leases *timelapsePlaybackLeases) reserveDeletion(date string) (func(), bool) {
	leases.mu.Lock()
	defer leases.mu.Unlock()
	if leases.deleting[date] || leases.active[date] > 0 {
		return nil, false
	}
	leases.deleting[date] = true

	var once sync.Once
	return func() {
		once.Do(func() {
			leases.mu.Lock()
			delete(leases.deleting, date)
			leases.mu.Unlock()
		})
	}, true
}

func (leases *timelapsePlaybackLeases) isActive(date string) bool {
	leases.mu.Lock()
	defer leases.mu.Unlock()
	return leases.active[date] > 0
}

type timelapseMutationCoordinator struct {
	mutationMu sync.Mutex

	lockPath    string
	lockTimeout time.Duration
	cache       func() *timelapseCache
	leases      *timelapsePlaybackLeases

	loadRetentionConfig func() (ScreenTimelapseRetentionConfig, error)
	lstat               func(string) (os.FileInfo, error)
	rename              func(string, string) error
	removeAll           func(string) error
	directorySize       func(string) (int64, error)
}

func newTimelapseMutationCoordinator(lockPath string, cache func() *timelapseCache, leases *timelapsePlaybackLeases) *timelapseMutationCoordinator {
	return &timelapseMutationCoordinator{
		lockPath:            lockPath,
		lockTimeout:         timelapseMutationLockTimeout,
		cache:               cache,
		leases:              leases,
		loadRetentionConfig: loadScreenTimelapseRetentionConfig,
		lstat:               os.Lstat,
		rename:              os.Rename,
		removeAll:           os.RemoveAll,
		directorySize:       timelapseDirectorySize,
	}
}

var screenTimelapsePlaybackLeases = newTimelapsePlaybackLeases()

var screenTimelapseMutationCoordinator = newTimelapseMutationCoordinator(
	timelapseMutationLockFile,
	func() *timelapseCache { return screenTimelapseCache },
	screenTimelapsePlaybackLeases,
)

func deleteScreenTimelapseDay(root string, date string, now time.Time) (int64, error) {
	return screenTimelapseMutationCoordinator.delete(timelapseDeletionRequest{
		Root: root,
		Date: date,
		Now:  now,
		Mode: timelapseDeletionManual,
	})
}

// deleteScreenTimelapseRetentionDay is reserved for the Step 4 executor. It
// intentionally re-reads the persisted policy while holding the namespace
// lock, so an already-planned automatic deletion cannot run after its policy
// changes or after its scanned directory changes.
func deleteScreenTimelapseRetentionDay(root string, date string, now time.Time, expectedConfig ScreenTimelapseRetentionConfig, expectedModTime time.Time) (int64, error) {
	return screenTimelapseMutationCoordinator.delete(timelapseDeletionRequest{
		Root:            root,
		Date:            date,
		Now:             now,
		Mode:            timelapseDeletionAutomatic,
		ExpectedConfig:  &expectedConfig,
		ExpectedModTime: &expectedModTime,
	})
}

func (coordinator *timelapseMutationCoordinator) delete(request timelapseDeletionRequest) (int64, error) {
	coordinator.mutationMu.Lock()
	defer coordinator.mutationMu.Unlock()

	if !isTimelapseDeletionDateAllowed(request) {
		return 0, errTimelapseDayNotDeletable
	}

	fileLock, err := lockTimelapseFile(coordinator.lockPath, coordinator.lockTimeout)
	if err != nil {
		return 0, err
	}

	deletedBytes, tombstonePath, err := coordinator.renameDayToTombstoneLocked(request)
	closeErr := fileLock.Close()
	if err != nil {
		return deletedBytes, err
	}
	if closeErr != nil {
		return deletedBytes, closeErr
	}
	if tombstonePath == "" {
		return deletedBytes, nil
	}
	if err := coordinator.removeAll(tombstonePath); err != nil {
		return deletedBytes, err
	}
	return deletedBytes, nil
}

func (coordinator *timelapseMutationCoordinator) renameDayToTombstoneLocked(request timelapseDeletionRequest) (int64, string, error) {
	// Keep this validation inside the namespace lock as well as the fast-path
	// check in delete. A request can wait for another process to release the
	// lock, so every condition that protects a day must be checked again just
	// before its directory is renamed.
	if !isTimelapseDeletionDateAllowed(request) {
		return 0, "", errTimelapseDayNotDeletable
	}

	if request.Mode == timelapseDeletionAutomatic {
		config, err := coordinator.loadRetentionConfig()
		if err != nil {
			return 0, "", err
		}
		if !config.Cleanup.Enabled {
			return 0, "", errTimelapseRetentionCleanupDisabled
		}
		if request.ExpectedConfig != nil && config != *request.ExpectedConfig {
			return 0, "", errTimelapseRetentionConfigChanged
		}
	}

	dayPath := filepath.Join(request.Root, request.Date)
	info, err := coordinator.lstat(dayPath)
	if errors.Is(err, os.ErrNotExist) {
		return 0, "", nil
	}
	if err != nil {
		return 0, "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return 0, "", errTimelapseDayPathUnsafe
	}
	if request.ExpectedModTime != nil && !info.ModTime().Equal(*request.ExpectedModTime) {
		return 0, "", fmt.Errorf("%w: timelapse day changed", errTimelapseDayBusy)
	}
	if request.Mode == timelapseDeletionAutomatic && request.Now.Sub(info.ModTime()) < screenTimelapseRetentionQuietPeriod {
		return 0, "", fmt.Errorf("%w: timelapse day was modified recently", errTimelapseDayBusy)
	}

	releaseDeletion, ok := coordinator.leases.reserveDeletion(request.Date)
	if !ok {
		return 0, "", fmt.Errorf("%w: active playback", errTimelapseDayBusy)
	}
	defer releaseDeletion()

	deletedBytes, err := coordinator.directorySize(dayPath)
	if err != nil {
		return 0, "", err
	}
	tombstonePath, err := newTimelapseTombstonePath(request.Root, request.Date)
	if err != nil {
		return 0, "", err
	}
	if err := coordinator.rename(dayPath, tombstonePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, "", nil
		}
		return 0, "", err
	}
	if cache := coordinator.cache(); cache != nil {
		cache.invalidateDay(request.Date, dayPath)
	}
	return deletedBytes, tombstonePath, nil
}

func newTimelapseTombstonePath(root string, date string) (string, error) {
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate timelapse tombstone nonce: %w", err)
	}
	return filepath.Join(root, ".deleting-"+date+"-"+hex.EncodeToString(nonce)), nil
}

func CleanupScreenTimelapseTombstones() error {
	return screenTimelapseMutationCoordinator.cleanupTombstones(timelapseRoot())
}

func (coordinator *timelapseMutationCoordinator) cleanupTombstones(root string) error {
	coordinator.mutationMu.Lock()
	defer coordinator.mutationMu.Unlock()

	fileLock, err := lockTimelapseFile(coordinator.lockPath, coordinator.lockTimeout)
	if err != nil {
		return err
	}

	entries, readErr := os.ReadDir(root)
	if errors.Is(readErr, os.ErrNotExist) {
		return fileLock.Close()
	}
	if readErr != nil {
		_ = fileLock.Close()
		return readErr
	}

	paths := make([]string, 0)
	var result error
	for _, entry := range entries {
		date, ok := timelapseTombstoneDate(entry.Name())
		if !ok {
			continue
		}
		path := filepath.Join(root, entry.Name())
		info, statErr := coordinator.lstat(path)
		if statErr != nil {
			result = errors.Join(result, statErr)
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			continue
		}
		if cache := coordinator.cache(); cache != nil {
			cache.invalidateDay(date, filepath.Join(root, date))
		}
		paths = append(paths, path)
	}
	if closeErr := fileLock.Close(); closeErr != nil {
		return errors.Join(result, closeErr)
	}

	for _, path := range paths {
		if err := coordinator.removeAll(path); err != nil {
			result = errors.Join(result, err)
		}
	}
	return result
}

func timelapseTombstoneDate(name string) (string, bool) {
	const prefix = ".deleting-"
	if !strings.HasPrefix(name, prefix) {
		return "", false
	}
	remainder := strings.TrimPrefix(name, prefix)
	dateLength := len("2006-01-02")
	if len(remainder) <= dateLength+1 {
		return "", false
	}
	date := remainder[:dateLength]
	if remainder[dateLength] != '-' || !isValidTimelapseDate(date) {
		return "", false
	}
	return date, true
}

// deleteTimelapseDay remains a small test-facing wrapper around the same
// mutation primitive. Production requests use the shared coordinator above.
func deleteTimelapseDay(root string, date string, now time.Time) (int64, error) {
	coordinator := newTimelapseMutationCoordinator(
		filepath.Join(filepath.Dir(root), ".timelapse.lock"),
		newTimelapseCache,
		newTimelapsePlaybackLeases(),
	)
	return coordinator.delete(timelapseDeletionRequest{
		Root: root,
		Date: date,
		Now:  now,
		Mode: timelapseDeletionManual,
	})
}
