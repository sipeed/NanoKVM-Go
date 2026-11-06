package ai

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestScreenTimelapseRetentionMissingConfigUsesDisabledDefaultWithoutWriting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "etc", "timelapse.json")
	store := newScreenTimelapseRetentionConfigStore(path)

	config, err := store.Load()
	if err != nil {
		t.Fatalf("load missing config: %v", err)
	}
	if !reflect.DeepEqual(config, defaultScreenTimelapseRetentionConfig()) {
		t.Fatalf("unexpected default config: %#v", config)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("loading a missing config must not create it, stat error=%v", err)
	}
}

func TestScreenTimelapseRetentionConfigRejectsMalformedInputFailClosed(t *testing.T) {
	validCleanup := `"cleanup":{"enabled":false,"retentionDays":30,"maxStorageMiB":4096}`
	for name, data := range map[string]string{
		"unknown root field":      `{"version":1,"revision":0,` + validCleanup + `,"extra":true}`,
		"unknown cleanup field":   `{"version":1,"revision":0,"cleanup":{"enabled":false,"retentionDays":30,"maxStorageMiB":4096,"extra":true}}`,
		"fractional capacity":     `{"version":1,"revision":0,"cleanup":{"enabled":false,"retentionDays":30,"maxStorageMiB":1.5}}`,
		"string capacity":         `{"version":1,"revision":0,"cleanup":{"enabled":false,"retentionDays":30,"maxStorageMiB":"4096"}}`,
		"missing field":           `{"version":1,"cleanup":{"enabled":false,"retentionDays":30,"maxStorageMiB":4096}}`,
		"unsupported version":     `{"version":2,"revision":0,` + validCleanup + `}`,
		"negative revision":       `{"version":1,"revision":-1,` + validCleanup + `}`,
		"too many days":           `{"version":1,"revision":0,"cleanup":{"enabled":false,"retentionDays":3651,"maxStorageMiB":4096}}`,
		"negative capacity":       `{"version":1,"revision":0,"cleanup":{"enabled":false,"retentionDays":30,"maxStorageMiB":-1}}`,
		"enabled without limit":   `{"version":1,"revision":0,"cleanup":{"enabled":true,"retentionDays":0,"maxStorageMiB":0}}`,
		"multiple JSON documents": `{"version":1,"revision":0,` + validCleanup + `} {}`,
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "timelapse.json")
			original := []byte(data)
			if err := os.WriteFile(path, original, 0o600); err != nil {
				t.Fatalf("write malformed config: %v", err)
			}

			config, err := loadScreenTimelapseRetentionConfigFromPath(path)
			if err == nil || !errors.Is(err, errScreenTimelapseRetentionInvalidConfig) {
				t.Fatalf("expected invalid config error, got config=%#v err=%v", config, err)
			}
			if config.Cleanup.Enabled {
				t.Fatalf("malformed config must fail closed: %#v", config)
			}
			stored, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatalf("read malformed config: %v", readErr)
			}
			if !bytes.Equal(stored, original) {
				t.Fatalf("load changed malformed config: got=%q want=%q", stored, original)
			}
		})
	}
}

func TestScreenTimelapseRetentionConfigSaveIsAtomicAndValidated(t *testing.T) {
	path := filepath.Join(t.TempDir(), "etc", "timelapse.json")
	store := newScreenTimelapseRetentionConfigStore(path)
	config := defaultScreenTimelapseRetentionConfig()
	config.Revision = 12
	config.Cleanup.Enabled = true
	config.Cleanup.RetentionDays = 7
	config.Cleanup.MaxStorageMiB = 0

	if err := store.Save(config); err != nil {
		t.Fatalf("save config: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat saved config: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("saved config mode=%o, want 0600", info.Mode().Perm())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved config: %v", err)
	}
	if !bytes.HasSuffix(data, []byte("\n")) {
		t.Fatalf("saved config must end with newline: %q", data)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("load saved config: %v", err)
	}
	if !reflect.DeepEqual(loaded, config) {
		t.Fatalf("loaded config=%#v, want %#v", loaded, config)
	}
	temporary, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".timelapse.json.*"))
	if err != nil {
		t.Fatalf("glob temporary configs: %v", err)
	}
	if len(temporary) != 0 {
		t.Fatalf("atomic save left temporary files: %v", temporary)
	}

	original := append([]byte(nil), data...)
	invalid := config
	invalid.Cleanup.RetentionDays = 0
	invalid.Cleanup.MaxStorageMiB = 0
	if err := store.Save(invalid); err == nil {
		t.Fatal("saving an enabled config with no limits must fail")
	}
	stored, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config after invalid save: %v", err)
	}
	if !bytes.Equal(stored, original) {
		t.Fatalf("invalid save overwrote config: got=%q want=%q", stored, original)
	}
}

func TestScreenTimelapseRetentionConfigStoreUpdateCleanupUsesCompareAndSwap(t *testing.T) {
	previousNotify := notifyScreenTimelapseRetentionConfigSaved
	notifications := 0
	notifyScreenTimelapseRetentionConfigSaved = func() bool {
		notifications++
		return true
	}
	t.Cleanup(func() { notifyScreenTimelapseRetentionConfigSaved = previousNotify })

	path := filepath.Join(t.TempDir(), "etc", "timelapse.json")
	store := newScreenTimelapseRetentionConfigStore(path)
	desired := ScreenTimelapseRetentionCleanupConfig{
		Enabled:       true,
		RetentionDays: 7,
		MaxStorageMiB: 0,
	}

	updated, changed, conflict, triggered, err := store.UpdateCleanup(0, desired)
	if err != nil || !changed || conflict || !triggered || updated.Revision != 1 || updated.Cleanup != desired || notifications != 1 {
		t.Fatalf("initial update=%#v changed=%t conflict=%t triggered=%t notifications=%d err=%v", updated, changed, conflict, triggered, notifications, err)
	}

	stale, changed, conflict, triggered, err := store.UpdateCleanup(0, defaultScreenTimelapseRetentionConfig().Cleanup)
	if err != nil || changed || !conflict || triggered || stale != updated || notifications != 1 {
		t.Fatalf("stale update=%#v changed=%t conflict=%t triggered=%t notifications=%d err=%v", stale, changed, conflict, triggered, notifications, err)
	}

	unchanged, changed, conflict, triggered, err := store.UpdateCleanup(1, desired)
	if err != nil || changed || conflict || triggered || unchanged != updated || notifications != 1 {
		t.Fatalf("unchanged update=%#v changed=%t conflict=%t triggered=%t notifications=%d err=%v", unchanged, changed, conflict, triggered, notifications, err)
	}

	loaded, err := store.Load()
	if err != nil || loaded != updated {
		t.Fatalf("persisted update=%#v err=%v, want %#v", loaded, err, updated)
	}
}

func TestScreenTimelapseRetentionDefaultMatchesContractFixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "screen_timelapse_retention_contract.json"))
	if err != nil {
		t.Fatalf("read contract fixture: %v", err)
	}
	var fixture struct {
		ConfigFile struct {
			Path    string                         `json:"path"`
			Default ScreenTimelapseRetentionConfig `json:"default"`
		} `json:"configFile"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatalf("decode contract fixture: %v", err)
	}
	if fixture.ConfigFile.Path != ScreenTimelapseRetentionConfigFile {
		t.Fatalf("contract config path=%q, want %q", fixture.ConfigFile.Path, ScreenTimelapseRetentionConfigFile)
	}
	if !reflect.DeepEqual(fixture.ConfigFile.Default, defaultScreenTimelapseRetentionConfig()) {
		t.Fatalf("contract default=%#v, implementation default=%#v", fixture.ConfigFile.Default, defaultScreenTimelapseRetentionConfig())
	}
}

func TestInventoryScreenTimelapseRetentionAcceptsOnlySafeDateDirectories(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2024, time.March, 1, 12, 0, 0, 0, time.FixedZone("CST", 8*60*60))

	oldDay := filepath.Join(root, "2024-02-29")
	writeScreenTimelapseRetentionTestFile(t, filepath.Join(oldDay, "2024-02-29.ts"), "abc")
	writeScreenTimelapseRetentionTestFile(t, filepath.Join(oldDay, "nested", "2024-02-29.ts.idx"), "12345")
	target := filepath.Join(root, "outside-target")
	writeScreenTimelapseRetentionTestFile(t, target, "do-not-count")
	if err := os.Symlink(target, filepath.Join(oldDay, "external-link")); err != nil {
		t.Fatalf("create nested symlink: %v", err)
	}
	oldModTime := now.Add(-2 * time.Hour)
	if err := os.Chtimes(oldDay, oldModTime, oldModTime); err != nil {
		t.Fatalf("set old day mod time: %v", err)
	}

	writeScreenTimelapseRetentionTestFile(t, filepath.Join(root, "2024-03-01", "2024-03-01.ts"), "1234567")
	writeScreenTimelapseRetentionTestFile(t, filepath.Join(root, "2024-03-02", "2024-03-02.ts"), "123456789")
	writeScreenTimelapseRetentionTestFile(t, filepath.Join(root, "2024-02-30", "bad.ts"), "bad")
	if err := os.WriteFile(filepath.Join(root, "2024-02-28"), []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("write date-like ordinary file: %v", err)
	}
	if err := os.Symlink(oldDay, filepath.Join(root, "2024-02-27")); err != nil {
		t.Fatalf("create date-like symlink: %v", err)
	}

	inventory, err := inventoryScreenTimelapseRetention(root, now)
	if err != nil {
		t.Fatalf("inventory timelapse root: %v", err)
	}
	if !inventory.Complete {
		t.Fatalf("safe inventory unexpectedly incomplete: %#v", inventory)
	}
	if got, want := inventory.TotalBytes, int64(3+5+7+9); got != want {
		t.Fatalf("inventory bytes=%d, want %d; nested symlink must not be followed", got, want)
	}
	if len(inventory.Days) != 3 {
		t.Fatalf("managed days=%#v", inventory.Days)
	}
	if inventory.Days[0].Date != "2024-02-29" || inventory.Days[1].Date != "2024-03-01" || inventory.Days[2].Date != "2024-03-02" {
		t.Fatalf("unexpected date order: %#v", inventory.Days)
	}
	if inventory.Days[0].Bytes != 8 || !inventory.Days[0].ModTime.Equal(oldModTime) || !inventory.Days[0].ScanComplete {
		t.Fatalf("unexpected historical day: %#v", inventory.Days[0])
	}
	if !inventory.Days[1].IsToday || inventory.Days[1].IsFuture {
		t.Fatalf("today flags=%#v", inventory.Days[1])
	}
	if inventory.Days[2].IsToday || !inventory.Days[2].IsFuture {
		t.Fatalf("future flags=%#v", inventory.Days[2])
	}
	for _, reason := range []string{
		screenTimelapseRetentionWarningInvalid,
		screenTimelapseRetentionWarningNotDir,
		screenTimelapseRetentionWarningSymlink,
		screenTimelapseRetentionWarningFuture,
	} {
		if !screenTimelapseRetentionInventoryHasWarning(inventory, reason) {
			t.Fatalf("expected %q warning, got %#v", reason, inventory.Warnings)
		}
	}
}

func TestInventoryScreenTimelapseRetentionMissingRootIsEmpty(t *testing.T) {
	root := filepath.Join(t.TempDir(), "missing")
	inventory, err := inventoryScreenTimelapseRetention(root, time.Now())
	if err != nil {
		t.Fatalf("inventory missing root: %v", err)
	}
	if !inventory.Complete || len(inventory.Days) != 0 || inventory.TotalBytes != 0 || len(inventory.Warnings) != 0 {
		t.Fatalf("unexpected missing-root inventory: %#v", inventory)
	}
}

func TestPlanScreenTimelapseRetentionCombinesDayAndCapacityRules(t *testing.T) {
	now := time.Date(2024, time.March, 1, 12, 0, 0, 0, time.UTC)
	miB := screenTimelapseRetentionBytesPerMiB
	inventory := screenTimelapseRetentionTestInventory(now,
		screenTimelapseRetentionTestDay("2024-02-27", miB, now.Add(-time.Hour)),
		screenTimelapseRetentionTestDay("2024-02-28", miB, now.Add(-time.Hour)),
		screenTimelapseRetentionTestDay("2024-02-29", miB, now.Add(-time.Hour)),
		screenTimelapseRetentionTestDay("2024-03-01", 2*miB, now),
	)
	config := screenTimelapseRetentionTestConfig(2, 2)

	plan, err := planScreenTimelapseRetention(inventory, now, config)
	if err != nil {
		t.Fatalf("plan retention: %v", err)
	}
	if plan.BeforeBytes != 5*miB || plan.ProjectedAfterBytes != 2*miB || plan.OverLimit {
		t.Fatalf("unexpected plan totals: %#v", plan)
	}
	wantCandidates := []ScreenTimelapseRetentionPlanDay{
		{Date: "2024-02-27", Bytes: miB, Reasons: []string{screenTimelapseRetentionReasonExpired, screenTimelapseRetentionReasonQuota}},
		{Date: "2024-02-28", Bytes: miB, Reasons: []string{screenTimelapseRetentionReasonExpired, screenTimelapseRetentionReasonQuota}},
		{Date: "2024-02-29", Bytes: miB, Reasons: []string{screenTimelapseRetentionReasonQuota}},
	}
	if !reflect.DeepEqual(plan.Candidates, wantCandidates) {
		t.Fatalf("candidates=%#v, want %#v", plan.Candidates, wantCandidates)
	}
	if !reflect.DeepEqual(plan.Skipped, []ScreenTimelapseRetentionSkippedDay{{Date: "2024-03-01", Bytes: 2 * miB, Reason: screenTimelapseRetentionSkipToday}}) {
		t.Fatalf("skipped=%#v", plan.Skipped)
	}
}

func TestPlanScreenTimelapseRetentionBoundariesAndProtections(t *testing.T) {
	now := time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC)
	miB := screenTimelapseRetentionBytesPerMiB
	for _, test := range []struct {
		name           string
		inventory      ScreenTimelapseRetentionInventory
		config         ScreenTimelapseRetentionConfig
		wantCandidates []ScreenTimelapseRetentionPlanDay
		wantSkipped    []ScreenTimelapseRetentionSkippedDay
		wantAfter      int64
		wantOverLimit  bool
	}{
		{
			name: "zero retention and exact capacity are no-op",
			inventory: screenTimelapseRetentionTestInventory(now,
				screenTimelapseRetentionTestDay("2024-12-31", miB, now.Add(-time.Hour)),
				screenTimelapseRetentionTestDay("2025-01-01", miB, now),
			),
			config:      screenTimelapseRetentionTestConfig(0, 2),
			wantSkipped: []ScreenTimelapseRetentionSkippedDay{{Date: "2025-01-01", Bytes: miB, Reason: screenTimelapseRetentionSkipToday}},
			wantAfter:   2 * miB,
		},
		{
			name: "one day keeps only today across year",
			inventory: screenTimelapseRetentionTestInventory(now,
				screenTimelapseRetentionTestDay("2024-12-31", miB, now.Add(-time.Hour)),
				screenTimelapseRetentionTestDay("2025-01-01", miB, now),
			),
			config: screenTimelapseRetentionTestConfig(1, 0),
			wantCandidates: []ScreenTimelapseRetentionPlanDay{{
				Date: "2024-12-31", Bytes: miB, Reasons: []string{screenTimelapseRetentionReasonExpired},
			}},
			wantSkipped: []ScreenTimelapseRetentionSkippedDay{{Date: "2025-01-01", Bytes: miB, Reason: screenTimelapseRetentionSkipToday}},
			wantAfter:   miB,
		},
		{
			name: "today alone can remain above soft capacity",
			inventory: screenTimelapseRetentionTestInventory(now,
				screenTimelapseRetentionTestDay("2025-01-01", 2*miB, now),
			),
			config:        screenTimelapseRetentionTestConfig(0, 1),
			wantSkipped:   []ScreenTimelapseRetentionSkippedDay{{Date: "2025-01-01", Bytes: 2 * miB, Reason: screenTimelapseRetentionSkipToday}},
			wantAfter:     2 * miB,
			wantOverLimit: true,
		},
		{
			name: "recent write and future date are protected",
			inventory: screenTimelapseRetentionTestInventory(now,
				screenTimelapseRetentionTestDay("2024-12-29", miB, now.Add(-5*time.Minute)),
				screenTimelapseRetentionTestDay("2025-01-01", miB, now),
				screenTimelapseRetentionTestDay("2025-01-02", miB, now.Add(-time.Hour)),
			),
			config: screenTimelapseRetentionTestConfig(1, 0),
			wantSkipped: []ScreenTimelapseRetentionSkippedDay{
				{Date: "2024-12-29", Bytes: miB, Reason: screenTimelapseRetentionSkipRecentWrite},
				{Date: "2025-01-01", Bytes: miB, Reason: screenTimelapseRetentionSkipToday},
				{Date: "2025-01-02", Bytes: miB, Reason: screenTimelapseRetentionSkipFuture},
			},
			wantAfter: 3 * miB,
		},
		{
			name: "incomplete inventory fails closed",
			inventory: ScreenTimelapseRetentionInventory{
				Complete: false,
				Days: []ScreenTimelapseRetentionInventoryDay{
					screenTimelapseRetentionTestDay("2024-12-30", miB, now.Add(-time.Hour)),
				},
			},
			config:      screenTimelapseRetentionTestConfig(1, 0),
			wantSkipped: []ScreenTimelapseRetentionSkippedDay{{Date: "2024-12-30", Bytes: miB, Reason: screenTimelapseRetentionSkipIncomplete}},
			wantAfter:   miB,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			plan, err := planScreenTimelapseRetention(test.inventory, now, test.config)
			if err != nil {
				t.Fatalf("plan retention: %v", err)
			}
			wantCandidates := test.wantCandidates
			if wantCandidates == nil {
				wantCandidates = []ScreenTimelapseRetentionPlanDay{}
			}
			if !reflect.DeepEqual(plan.Candidates, wantCandidates) {
				t.Fatalf("candidates=%#v, want %#v", plan.Candidates, wantCandidates)
			}
			if !reflect.DeepEqual(plan.Skipped, test.wantSkipped) {
				t.Fatalf("skipped=%#v, want %#v", plan.Skipped, test.wantSkipped)
			}
			if plan.ProjectedAfterBytes != test.wantAfter || plan.OverLimit != test.wantOverLimit {
				t.Fatalf("plan totals=%#v, want after=%d overLimit=%v", plan, test.wantAfter, test.wantOverLimit)
			}
		})
	}
}

func TestPlanScreenTimelapseRetentionIsPureAndRejectsUnsafeInput(t *testing.T) {
	now := time.Date(2024, time.March, 1, 12, 0, 0, 0, time.UTC)
	inventory := screenTimelapseRetentionTestInventory(now,
		screenTimelapseRetentionTestDay("2024-02-28", 10, now.Add(-time.Hour)),
		screenTimelapseRetentionTestDay("2024-02-27", 20, now.Add(-time.Hour)),
	)
	original := inventory
	original.Days = append([]ScreenTimelapseRetentionInventoryDay(nil), inventory.Days...)
	config := screenTimelapseRetentionTestConfig(1, 0)

	if _, err := planScreenTimelapseRetention(inventory, now, config); err != nil {
		t.Fatalf("plan retention: %v", err)
	}
	if !reflect.DeepEqual(inventory, original) {
		t.Fatalf("planner mutated inventory: got=%#v want=%#v", inventory, original)
	}

	unsafe := inventory
	unsafe.Days = append(unsafe.Days, screenTimelapseRetentionTestDay("2024-02-28", 1, now.Add(-time.Hour)))
	if _, err := planScreenTimelapseRetention(unsafe, now, config); err == nil {
		t.Fatal("duplicate inventory dates must be rejected")
	}
}

func writeScreenTimelapseRetentionTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create parent directory for %q: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}

func screenTimelapseRetentionInventoryHasWarning(inventory ScreenTimelapseRetentionInventory, reason string) bool {
	for _, warning := range inventory.Warnings {
		if warning.Reason == reason {
			return true
		}
	}
	return false
}

func screenTimelapseRetentionTestConfig(retentionDays int, maxStorageMiB int64) ScreenTimelapseRetentionConfig {
	config := defaultScreenTimelapseRetentionConfig()
	config.Cleanup.Enabled = true
	config.Cleanup.RetentionDays = retentionDays
	config.Cleanup.MaxStorageMiB = maxStorageMiB
	return config
}

func screenTimelapseRetentionTestInventory(now time.Time, days ...ScreenTimelapseRetentionInventoryDay) ScreenTimelapseRetentionInventory {
	return ScreenTimelapseRetentionInventory{
		Days:     days,
		Complete: true,
		Warnings: []ScreenTimelapseRetentionInventoryWarning{},
	}
}

func screenTimelapseRetentionTestDay(date string, bytes int64, modTime time.Time) ScreenTimelapseRetentionInventoryDay {
	return ScreenTimelapseRetentionInventoryDay{
		Date:         date,
		Bytes:        bytes,
		ModTime:      modTime,
		ScanComplete: true,
	}
}
