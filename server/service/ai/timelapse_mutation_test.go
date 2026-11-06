package ai

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"NanoKVM-Server/proto"
)

func writeTimelapseMutationTestDay(t *testing.T, root string, date string, modifiedAt time.Time) (string, time.Time, int64) {
	t.Helper()

	dayPath := filepath.Join(root, date)
	if err := os.MkdirAll(filepath.Join(dayPath, "metadata"), 0o755); err != nil {
		t.Fatalf("create test timelapse day: %v", err)
	}
	files := map[string][]byte{
		filepath.Join(dayPath, date+".ts"):                 []byte("video-data"),
		filepath.Join(dayPath, date+".ts.idx"):             []byte("index-data"),
		filepath.Join(dayPath, "metadata", "capture.json"): []byte("{}"),
	}
	var bytes int64
	for path, data := range files {
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatalf("write test timelapse file %q: %v", path, err)
		}
		bytes += int64(len(data))
	}
	if err := os.Chtimes(dayPath, modifiedAt, modifiedAt); err != nil {
		t.Fatalf("set test day mtime: %v", err)
	}
	info, err := os.Lstat(dayPath)
	if err != nil {
		t.Fatalf("stat test timelapse day: %v", err)
	}
	return dayPath, info.ModTime(), bytes
}

func newTimelapseMutationTestCoordinator(t *testing.T, cache *timelapseCache, leases *timelapsePlaybackLeases) *timelapseMutationCoordinator {
	t.Helper()
	return newTimelapseMutationCoordinator(
		filepath.Join(t.TempDir(), "timelapse.lock"),
		func() *timelapseCache { return cache },
		leases,
	)
}

func waitForTimelapseMutationSignal(t *testing.T, signal <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", description)
	}
}

func TestTimelapseFileLockTimesOutWhenHeld(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run", "timelapse.lock")
	held, err := lockTimelapseFile(path, 0)
	if err != nil {
		t.Fatalf("acquire initial lock: %v", err)
	}
	t.Cleanup(func() {
		if err := held.Close(); err != nil {
			t.Errorf("release initial lock: %v", err)
		}
	})

	started := time.Now()
	contended, err := lockTimelapseFile(path, 40*time.Millisecond)
	if contended != nil {
		_ = contended.Close()
		t.Fatal("contended lock unexpectedly succeeded")
	}
	if !errors.Is(err, errTimelapseDayBusy) {
		t.Fatalf("expected busy timeout, got %v", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("lock timeout took too long: %s", elapsed)
	}
}

func TestTimelapseMutationRenamesInvalidatesCacheAndReleasesFileLock(t *testing.T) {
	root := t.TempDir()
	date := "2026-06-01"
	now := time.Date(2026, time.June, 2, 12, 0, 0, 0, time.Local)
	dayPath, _, wantBytes := writeTimelapseMutationTestDay(t, root, date, now.Add(-time.Hour))
	cache := newTimelapseCache()
	cache.now = func() time.Time { return now }
	cache.setDays(30, []proto.ScreenTimelapseDay{{Date: date}})
	cache.setFrames(date, []timelapseFrame{{CaptureID: "deleted"}})
	videoPath := filepath.Join(dayPath, date+".ts")
	cache.tsIndexes[timelapseTSIndexCacheKey(videoPath, 188)] = cacheEntry[timelapseTSIndex]{
		value:     timelapseTSIndex{Size: 188},
		expiresAt: now.Add(time.Minute),
	}

	coordinator := newTimelapseMutationTestCoordinator(t, cache, newTimelapsePlaybackLeases())
	coordinator.removeAll = func(tombstonePath string) error {
		if _, ok := timelapseTombstoneDate(filepath.Base(tombstonePath)); !ok {
			return errors.New("delete path is not a valid tombstone")
		}
		if _, err := os.Lstat(dayPath); !errors.Is(err, os.ErrNotExist) {
			return errors.New("source date remained visible after rename")
		}
		if _, ok := cache.getDays(30); ok {
			return errors.New("days cache was not invalidated before recursive delete")
		}
		if _, ok := cache.getFrames(date); ok {
			return errors.New("frames cache was not invalidated before recursive delete")
		}
		cache.mu.Lock()
		_, indexCached := cache.tsIndexes[timelapseTSIndexCacheKey(videoPath, 188)]
		cache.mu.Unlock()
		if indexCached {
			return errors.New("TS index cache was not invalidated before recursive delete")
		}

		lock, err := lockTimelapseFile(coordinator.lockPath, 0)
		if err != nil {
			return errors.New("namespace lock remained held during recursive delete")
		}
		if err := lock.Close(); err != nil {
			return err
		}
		return os.RemoveAll(tombstonePath)
	}

	deletedBytes, err := coordinator.delete(timelapseDeletionRequest{
		Root: root,
		Date: date,
		Now:  now,
		Mode: timelapseDeletionManual,
	})
	if err != nil {
		t.Fatalf("delete timelapse day: %v", err)
	}
	if deletedBytes != wantBytes {
		t.Fatalf("deleted bytes=%d, want %d", deletedBytes, wantBytes)
	}
	if _, err := os.Lstat(dayPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("source date still exists after delete: %v", err)
	}
}

func TestTimelapseMutationRejectsUnsafeDayPaths(t *testing.T) {
	root := t.TempDir()
	date := "2026-06-01"
	now := time.Date(2026, time.June, 2, 12, 0, 0, 0, time.Local)
	target := filepath.Join(t.TempDir(), "target")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, date)); err != nil {
		t.Fatalf("create date symlink: %v", err)
	}

	coordinator := newTimelapseMutationTestCoordinator(t, newTimelapseCache(), newTimelapsePlaybackLeases())
	if _, err := coordinator.delete(timelapseDeletionRequest{Root: root, Date: date, Now: now, Mode: timelapseDeletionManual}); !errors.Is(err, errTimelapseDayPathUnsafe) {
		t.Fatalf("expected unsafe-path error, got %v", err)
	}
	if info, err := os.Stat(target); err != nil || !info.IsDir() {
		t.Fatalf("symlink target was changed: info=%#v err=%v", info, err)
	}

	regularFileRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(regularFileRoot, date), []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("create date file: %v", err)
	}
	if _, err := coordinator.delete(timelapseDeletionRequest{Root: regularFileRoot, Date: date, Now: now, Mode: timelapseDeletionManual}); !errors.Is(err, errTimelapseDayPathUnsafe) {
		t.Fatalf("expected regular date file to be rejected, got %v", err)
	}
}

func TestTimelapseAutomaticDeletionRechecksPolicyAndDirectoryState(t *testing.T) {
	now := time.Date(2026, time.June, 2, 12, 0, 0, 0, time.Local)
	enabledConfig := defaultScreenTimelapseRetentionConfig()
	enabledConfig.Cleanup.Enabled = true
	enabledConfig.Cleanup.RetentionDays = 7
	enabledConfig.Cleanup.MaxStorageMiB = 0

	for _, test := range []struct {
		name           string
		expectedConfig ScreenTimelapseRetentionConfig
		config         ScreenTimelapseRetentionConfig
		modifiedAt     time.Time
		expectedMTime  func(time.Time) time.Time
		wantErr        error
		wantRemoved    bool
	}{
		{
			name:           "policy disabled after planning",
			expectedConfig: enabledConfig,
			config:         defaultScreenTimelapseRetentionConfig(),
			modifiedAt:     now.Add(-time.Hour),
			expectedMTime: func(modTime time.Time) time.Time {
				return modTime
			},
			wantErr: errTimelapseRetentionCleanupDisabled,
		},
		{
			name:           "policy limits changed after planning",
			expectedConfig: enabledConfig,
			config: ScreenTimelapseRetentionConfig{
				Version:  enabledConfig.Version,
				Revision: enabledConfig.Revision + 1,
				Cleanup: ScreenTimelapseRetentionCleanupConfig{
					Enabled:       true,
					RetentionDays: 30,
				},
			},
			modifiedAt: now.Add(-time.Hour),
			expectedMTime: func(modTime time.Time) time.Time {
				return modTime
			},
			wantErr: errTimelapseRetentionConfigChanged,
		},
		{
			name:           "directory changed after inventory",
			expectedConfig: enabledConfig,
			config:         enabledConfig,
			modifiedAt:     now.Add(-time.Hour),
			expectedMTime: func(modTime time.Time) time.Time {
				return modTime.Add(-time.Second)
			},
			wantErr: errTimelapseDayBusy,
		},
		{
			name:           "recent directory write",
			expectedConfig: enabledConfig,
			config:         enabledConfig,
			modifiedAt:     now.Add(-screenTimelapseRetentionQuietPeriod + time.Second),
			expectedMTime: func(modTime time.Time) time.Time {
				return modTime
			},
			wantErr: errTimelapseDayBusy,
		},
		{
			name:           "enabled unchanged historical day",
			expectedConfig: enabledConfig,
			config:         enabledConfig,
			modifiedAt:     now.Add(-time.Hour),
			expectedMTime: func(modTime time.Time) time.Time {
				return modTime
			},
			wantRemoved: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			date := "2026-06-01"
			dayPath, modTime, _ := writeTimelapseMutationTestDay(t, root, date, test.modifiedAt)
			coordinator := newTimelapseMutationTestCoordinator(t, newTimelapseCache(), newTimelapsePlaybackLeases())
			coordinator.loadRetentionConfig = func() (ScreenTimelapseRetentionConfig, error) {
				return test.config, nil
			}
			expectedConfig := test.expectedConfig

			_, err := coordinator.delete(timelapseDeletionRequest{
				Root:            root,
				Date:            date,
				Now:             now,
				Mode:            timelapseDeletionAutomatic,
				ExpectedConfig:  &expectedConfig,
				ExpectedModTime: ptrTime(test.expectedMTime(modTime)),
			})
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("expected error %v, got %v", test.wantErr, err)
				}
				if _, statErr := os.Lstat(dayPath); statErr != nil {
					t.Fatalf("protected day should remain: %v", statErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("automatic delete: %v", err)
			}
			if !test.wantRemoved {
				t.Fatal("test case must assert an automatic deletion outcome")
			}
			if _, statErr := os.Lstat(dayPath); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("automatic deletion left day behind: %v", statErr)
			}
		})
	}
}

func TestTimelapseAutomaticDeletionStillRejectsCurrentDate(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, time.June, 2, 12, 0, 0, 0, time.Local)
	date := now.Format("2006-01-02")
	dayPath, modTime, _ := writeTimelapseMutationTestDay(t, root, date, now.Add(-time.Hour))
	config := defaultScreenTimelapseRetentionConfig()
	config.Cleanup.Enabled = true
	config.Cleanup.RetentionDays = 7
	config.Cleanup.MaxStorageMiB = 0

	coordinator := newTimelapseMutationTestCoordinator(t, newTimelapseCache(), newTimelapsePlaybackLeases())
	coordinator.loadRetentionConfig = func() (ScreenTimelapseRetentionConfig, error) {
		return config, nil
	}
	_, err := coordinator.delete(timelapseDeletionRequest{
		Root:            root,
		Date:            date,
		Now:             now,
		Mode:            timelapseDeletionAutomatic,
		ExpectedConfig:  &config,
		ExpectedModTime: &modTime,
	})
	if !errors.Is(err, errTimelapseDayNotDeletable) {
		t.Fatalf("expected automatic current-day deletion rejection, got %v", err)
	}
	if _, statErr := os.Lstat(dayPath); statErr != nil {
		t.Fatalf("automatic cleanup changed current day: %v", statErr)
	}
}

func TestTimelapseDeletionRespectsAndReservesPlaybackLeases(t *testing.T) {
	root := t.TempDir()
	date := "2026-06-01"
	now := time.Date(2026, time.June, 2, 12, 0, 0, 0, time.Local)
	dayPath, _, _ := writeTimelapseMutationTestDay(t, root, date, now.Add(-time.Hour))
	leases := newTimelapsePlaybackLeases()
	coordinator := newTimelapseMutationTestCoordinator(t, newTimelapseCache(), leases)

	releasePlayback, ok := leases.acquire(date)
	if !ok {
		t.Fatal("acquire initial playback lease")
	}
	if _, err := coordinator.delete(timelapseDeletionRequest{Root: root, Date: date, Now: now, Mode: timelapseDeletionManual}); !errors.Is(err, errTimelapseDayBusy) {
		t.Fatalf("expected active playback to block delete, got %v", err)
	}
	if _, err := os.Lstat(dayPath); err != nil {
		t.Fatalf("active playback deletion removed the day: %v", err)
	}
	releasePlayback()

	reserved := make(chan struct{})
	allowRename := make(chan struct{})
	var reserveOnce sync.Once
	coordinator.directorySize = func(path string) (int64, error) {
		reserveOnce.Do(func() { close(reserved) })
		<-allowRename
		return timelapseDirectorySize(path)
	}
	deleteResult := make(chan error, 1)
	go func() {
		_, err := coordinator.delete(timelapseDeletionRequest{Root: root, Date: date, Now: now, Mode: timelapseDeletionManual})
		deleteResult <- err
	}()
	waitForTimelapseMutationSignal(t, reserved, "deletion reservation")
	if release, ok := leases.acquire(date); ok {
		release()
		t.Fatal("new playback lease acquired after deletion reservation")
	}
	close(allowRename)
	select {
	case err := <-deleteResult:
		if err != nil {
			t.Fatalf("delete reserved day: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for reserved deletion")
	}
	if release, ok := leases.acquire(date); !ok {
		t.Fatal("deletion reservation was not released")
	} else {
		release()
	}
}

func TestTimelapseMutationSerializesManualAndAutomaticDeletes(t *testing.T) {
	root := t.TempDir()
	date := "2026-06-01"
	now := time.Date(2026, time.June, 2, 12, 0, 0, 0, time.Local)
	_, modTime, _ := writeTimelapseMutationTestDay(t, root, date, now.Add(-time.Hour))
	coordinator := newTimelapseMutationTestCoordinator(t, newTimelapseCache(), newTimelapsePlaybackLeases())
	config := defaultScreenTimelapseRetentionConfig()
	config.Cleanup.Enabled = true
	config.Cleanup.RetentionDays = 7
	config.Cleanup.MaxStorageMiB = 0
	coordinator.loadRetentionConfig = func() (ScreenTimelapseRetentionConfig, error) { return config, nil }

	var lstatMu sync.Mutex
	lstatCalls := 0
	secondLstat := make(chan struct{})
	coordinator.lstat = func(path string) (os.FileInfo, error) {
		lstatMu.Lock()
		lstatCalls++
		if lstatCalls == 2 {
			close(secondLstat)
		}
		lstatMu.Unlock()
		return os.Lstat(path)
	}

	removeStarted := make(chan struct{})
	allowRemove := make(chan struct{})
	var removeOnce sync.Once
	coordinator.removeAll = func(path string) error {
		removeOnce.Do(func() { close(removeStarted) })
		<-allowRemove
		return os.RemoveAll(path)
	}
	t.Cleanup(func() {
		select {
		case <-allowRemove:
		default:
			close(allowRemove)
		}
	})

	manualResult := make(chan error, 1)
	go func() {
		_, err := coordinator.delete(timelapseDeletionRequest{Root: root, Date: date, Now: now, Mode: timelapseDeletionManual})
		manualResult <- err
	}()
	waitForTimelapseMutationSignal(t, removeStarted, "manual recursive delete")

	automaticStarted := make(chan struct{})
	automaticResult := make(chan struct {
		bytes int64
		err   error
	}, 1)
	go func() {
		close(automaticStarted)
		bytes, err := coordinator.delete(timelapseDeletionRequest{
			Root:            root,
			Date:            date,
			Now:             now,
			Mode:            timelapseDeletionAutomatic,
			ExpectedModTime: ptrTime(modTime),
		})
		automaticResult <- struct {
			bytes int64
			err   error
		}{bytes: bytes, err: err}
	}()
	waitForTimelapseMutationSignal(t, automaticStarted, "automatic delete start")
	select {
	case <-secondLstat:
		t.Fatal("automatic delete entered namespace while manual delete still removes tombstone")
	case <-time.After(50 * time.Millisecond):
	}

	close(allowRemove)
	if err := <-manualResult; err != nil {
		t.Fatalf("manual delete: %v", err)
	}
	select {
	case result := <-automaticResult:
		if result.err != nil || result.bytes != 0 {
			t.Fatalf("automatic delete after manual delete: bytes=%d err=%v", result.bytes, result.err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for automatic delete")
	}
}

func TestTimelapseTombstoneCleanupRemovesOnlySafeDirectories(t *testing.T) {
	root := t.TempDir()
	date := "2026-06-01"
	tombstone := filepath.Join(root, ".deleting-"+date+"-nonce")
	if err := os.MkdirAll(tombstone, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tombstone, "video.ts"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	malformed := filepath.Join(root, ".deleting-"+date+"-")
	if err := os.MkdirAll(malformed, 0o755); err != nil {
		t.Fatal(err)
	}
	regularDay := filepath.Join(root, "2026-05-31")
	if err := os.MkdirAll(regularDay, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "target")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	symlink := filepath.Join(root, ".deleting-"+date+"-symlink")
	if err := os.Symlink(target, symlink); err != nil {
		t.Fatalf("create tombstone symlink: %v", err)
	}

	cache := newTimelapseCache()
	cache.setDays(30, []proto.ScreenTimelapseDay{{Date: date}})
	cache.setFrames(date, []timelapseFrame{{CaptureID: "deleted"}})
	coordinator := newTimelapseMutationTestCoordinator(t, cache, newTimelapsePlaybackLeases())
	coordinator.removeAll = func(path string) error {
		lock, err := lockTimelapseFile(coordinator.lockPath, 0)
		if err != nil {
			return errors.New("namespace lock remained held during tombstone cleanup")
		}
		if err := lock.Close(); err != nil {
			return err
		}
		return os.RemoveAll(path)
	}

	if err := coordinator.cleanupTombstones(root); err != nil {
		t.Fatalf("cleanup tombstones: %v", err)
	}
	if _, err := os.Lstat(tombstone); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("valid tombstone was not removed: %v", err)
	}
	for _, path := range []string{malformed, regularDay, symlink, target} {
		if _, err := os.Lstat(path); err != nil {
			t.Fatalf("cleanup changed protected path %q: %v", path, err)
		}
	}
	if _, ok := cache.getDays(30); ok {
		t.Fatal("tombstone cleanup did not invalidate days cache")
	}
	if _, ok := cache.getFrames(date); ok {
		t.Fatal("tombstone cleanup did not invalidate date frames cache")
	}
	if err := coordinator.cleanupTombstones(filepath.Join(t.TempDir(), "missing")); err != nil {
		t.Fatalf("missing tombstone root must be a no-op: %v", err)
	}
}

func TestTimelapseTombstoneDateRequiresNonEmptyNonce(t *testing.T) {
	for name, want := range map[string]bool{
		".deleting-2026-06-01-nonce": true,
		".deleting-2026-06-01-":      false,
		".deleting-2026-02-30-nonce": false,
		"2026-06-01":                 false,
	} {
		_, got := timelapseTombstoneDate(name)
		if got != want {
			t.Fatalf("timelapseTombstoneDate(%q)=%t, want %t", name, got, want)
		}
	}
}

func TestTimelapseCacheGenerationDiscardsStaleDaysAndFramesLoads(t *testing.T) {
	t.Run("days", func(t *testing.T) {
		cache := newTimelapseCache()
		started := make(chan struct{})
		continueFirstLoad := make(chan struct{})
		loads := 0
		result := make(chan []proto.ScreenTimelapseDay, 1)
		errResult := make(chan error, 1)
		go func() {
			days, err := getTimelapseDaysWithCache(cache, 30, func(int) ([]proto.ScreenTimelapseDay, error) {
				loads++
				if loads == 1 {
					close(started)
					<-continueFirstLoad
					return []proto.ScreenTimelapseDay{{Date: "2026-06-01"}}, nil
				}
				return []proto.ScreenTimelapseDay{}, nil
			})
			if err != nil {
				errResult <- err
				return
			}
			result <- days
		}()
		waitForTimelapseMutationSignal(t, started, "initial days load")
		cache.invalidateDay("2026-06-01", filepath.Join(t.TempDir(), "2026-06-01"))
		close(continueFirstLoad)

		select {
		case err := <-errResult:
			t.Fatalf("load days: %v", err)
		case days := <-result:
			if len(days) != 0 || loads != 2 {
				t.Fatalf("stale days load was reused: days=%#v loads=%d", days, loads)
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for days cache retry")
		}
		if days, ok := cache.getDays(30); !ok || len(days) != 0 {
			t.Fatalf("days cache contains stale result: %#v ok=%t", days, ok)
		}
	})

	t.Run("frames", func(t *testing.T) {
		cache := newTimelapseCache()
		started := make(chan struct{})
		continueFirstLoad := make(chan struct{})
		loads := 0
		result := make(chan []timelapseFrame, 1)
		errResult := make(chan error, 1)
		go func() {
			frames, err := getTimelapseFramesWithCache(cache, "2026-06-01", func(string) ([]timelapseFrame, error) {
				loads++
				if loads == 1 {
					close(started)
					<-continueFirstLoad
					return []timelapseFrame{{CaptureID: "deleted"}}, nil
				}
				return []timelapseFrame{}, nil
			})
			if err != nil {
				errResult <- err
				return
			}
			result <- frames
		}()
		waitForTimelapseMutationSignal(t, started, "initial frames load")
		cache.invalidateDay("2026-06-01", filepath.Join(t.TempDir(), "2026-06-01"))
		close(continueFirstLoad)

		select {
		case err := <-errResult:
			t.Fatalf("load frames: %v", err)
		case frames := <-result:
			if len(frames) != 0 || loads != 2 {
				t.Fatalf("stale frames load was reused: frames=%#v loads=%d", frames, loads)
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for frames cache retry")
		}
		if frames, ok := cache.getFrames("2026-06-01"); !ok || len(frames) != 0 {
			t.Fatalf("frames cache contains stale result: %#v ok=%t", frames, ok)
		}
	})
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
