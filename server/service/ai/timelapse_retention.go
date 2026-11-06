package ai

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	ScreenTimelapseRetentionConfigFile = "/etc/kvm/timelapse.json"

	screenTimelapseRetentionConfigVersion           = 1
	screenTimelapseRetentionDefaultDays             = 180
	screenTimelapseRetentionDefaultStorageMiB       = 48 * 1024
	screenTimelapseRetentionMaxDays                 = 3650
	screenTimelapseRetentionQuietPeriod             = 10 * time.Minute
	screenTimelapseRetentionBytesPerMiB       int64 = 1024 * 1024
)

const (
	screenTimelapseRetentionReasonExpired     = "expired"
	screenTimelapseRetentionReasonQuota       = "quota"
	screenTimelapseRetentionSkipToday         = "today"
	screenTimelapseRetentionSkipFuture        = "future"
	screenTimelapseRetentionSkipRecentWrite   = "recent_write"
	screenTimelapseRetentionSkipIncomplete    = "inventory_incomplete"
	screenTimelapseRetentionWarningInvalid    = "invalid_name"
	screenTimelapseRetentionWarningNotDir     = "not_directory"
	screenTimelapseRetentionWarningSymlink    = "symlink"
	screenTimelapseRetentionWarningFuture     = "future_date"
	screenTimelapseRetentionWarningWalk       = "walk_error"
	screenTimelapseRetentionWarningNonRegular = "non_regular_file"
)

var (
	errScreenTimelapseRetentionInvalidConfig     = errors.New("invalid screen timelapse retention config")
	errScreenTimelapseRetentionRevisionExhausted = errors.New("screen timelapse retention config revision exhausted")
)

// ScreenTimelapseRetentionCleanupConfig is the persisted cleanup policy. A
// zero value is deliberately not a valid enabled policy: callers must provide
// at least one positive retention limit before enabling cleanup.
type ScreenTimelapseRetentionCleanupConfig struct {
	Enabled       bool  `json:"enabled"`
	RetentionDays int   `json:"retentionDays"`
	MaxStorageMiB int64 `json:"maxStorageMiB"`
}

// ScreenTimelapseRetentionConfig is independent from the service switch state
// in /etc/kvm/services.json and from MCP secrets in /etc/kvm/ai.json.
type ScreenTimelapseRetentionConfig struct {
	Version  int                                   `json:"version"`
	Revision int64                                 `json:"revision"`
	Cleanup  ScreenTimelapseRetentionCleanupConfig `json:"cleanup"`
}

// ScreenTimelapseRetentionInventory contains only direct, valid date
// directories below the timelapse root. Invalid entries are retained as
// warnings so callers can expose a safe, explainable scan result.
type ScreenTimelapseRetentionInventory struct {
	Days       []ScreenTimelapseRetentionInventoryDay     `json:"days"`
	TotalBytes int64                                      `json:"totalBytes"`
	Complete   bool                                       `json:"complete"`
	Warnings   []ScreenTimelapseRetentionInventoryWarning `json:"warnings"`
}

type ScreenTimelapseRetentionInventoryDay struct {
	Date         string    `json:"date"`
	Path         string    `json:"-"`
	Bytes        int64     `json:"bytes"`
	ModTime      time.Time `json:"modTime"`
	IsToday      bool      `json:"isToday"`
	IsFuture     bool      `json:"isFuture"`
	ScanComplete bool      `json:"scanComplete"`
}

type ScreenTimelapseRetentionInventoryWarning struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
	Error  string `json:"error,omitempty"`
}

// ScreenTimelapseRetentionPlan is intentionally a pure description. Step 1
// does not execute the plan; dry-run and the future executor will both use
// this same planner.
type ScreenTimelapseRetentionPlan struct {
	BeforeBytes         int64                                      `json:"beforeBytes"`
	ProjectedAfterBytes int64                                      `json:"projectedAfterBytes"`
	Candidates          []ScreenTimelapseRetentionPlanDay          `json:"candidates"`
	Skipped             []ScreenTimelapseRetentionSkippedDay       `json:"skipped"`
	OverLimit           bool                                       `json:"overLimit"`
	Warnings            []ScreenTimelapseRetentionInventoryWarning `json:"warnings"`
}

type ScreenTimelapseRetentionPlanDay struct {
	Date    string   `json:"date"`
	Bytes   int64    `json:"bytes"`
	Reasons []string `json:"reasons"`
}

type ScreenTimelapseRetentionSkippedDay struct {
	Date   string `json:"date"`
	Bytes  int64  `json:"bytes"`
	Reason string `json:"reason"`
}

type screenTimelapseRetentionConfigStore struct {
	mu   sync.Mutex
	path string
}

type screenTimelapseRetentionConfigFile struct {
	Version  *int                                   `json:"version"`
	Revision *int64                                 `json:"revision"`
	Cleanup  *screenTimelapseRetentionCleanupConfig `json:"cleanup"`
}

type screenTimelapseRetentionCleanupConfig struct {
	Enabled       *bool  `json:"enabled"`
	RetentionDays *int   `json:"retentionDays"`
	MaxStorageMiB *int64 `json:"maxStorageMiB"`
}

func newScreenTimelapseRetentionConfigStore(path string) *screenTimelapseRetentionConfigStore {
	return &screenTimelapseRetentionConfigStore{path: path}
}

var defaultScreenTimelapseRetentionConfigStore = newScreenTimelapseRetentionConfigStore(ScreenTimelapseRetentionConfigFile)

func defaultScreenTimelapseRetentionConfig() ScreenTimelapseRetentionConfig {
	return ScreenTimelapseRetentionConfig{
		Version:  screenTimelapseRetentionConfigVersion,
		Revision: 0,
		Cleanup: ScreenTimelapseRetentionCleanupConfig{
			Enabled:       false,
			RetentionDays: screenTimelapseRetentionDefaultDays,
			MaxStorageMiB: screenTimelapseRetentionDefaultStorageMiB,
		},
	}
}

func (store *screenTimelapseRetentionConfigStore) Load() (ScreenTimelapseRetentionConfig, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	config, err := loadScreenTimelapseRetentionConfigFromPath(store.path)
	if err != nil {
		log.WithError(err).Error("screen timelapse retention config is invalid; automatic cleanup remains disabled")
	}
	return config, err
}

func (store *screenTimelapseRetentionConfigStore) Save(config ScreenTimelapseRetentionConfig) error {
	store.mu.Lock()
	err := saveScreenTimelapseRetentionConfigToPath(store.path, config)
	store.mu.Unlock()
	if err != nil {
		return err
	}
	_ = notifyScreenTimelapseRetentionConfigSaved()
	return nil
}

// UpdateCleanup atomically compares the persisted revision and writes an
// updated cleanup policy. It serializes the complete read-check-write cycle
// so concurrent browser tabs cannot overwrite each other after independently
// observing the same revision.
func (store *screenTimelapseRetentionConfigStore) UpdateCleanup(expectedRevision int64, cleanup ScreenTimelapseRetentionCleanupConfig) (ScreenTimelapseRetentionConfig, bool, bool, bool, error) {
	store.mu.Lock()
	current, err := loadScreenTimelapseRetentionConfigFromPath(store.path)
	if err != nil {
		store.mu.Unlock()
		return current, false, false, false, err
	}
	if current.Revision != expectedRevision {
		store.mu.Unlock()
		return current, false, true, false, nil
	}
	if current.Cleanup == cleanup {
		store.mu.Unlock()
		return current, false, false, false, nil
	}
	if current.Revision == math.MaxInt64 {
		store.mu.Unlock()
		return current, false, false, false, errScreenTimelapseRetentionRevisionExhausted
	}

	updated := current
	updated.Revision++
	updated.Cleanup = cleanup
	if err := saveScreenTimelapseRetentionConfigToPath(store.path, updated); err != nil {
		store.mu.Unlock()
		return current, false, false, false, err
	}
	store.mu.Unlock()

	return updated, true, false, notifyScreenTimelapseRetentionConfigSaved(), nil
}

func loadScreenTimelapseRetentionConfig() (ScreenTimelapseRetentionConfig, error) {
	return defaultScreenTimelapseRetentionConfigStore.Load()
}

func saveScreenTimelapseRetentionConfig(config ScreenTimelapseRetentionConfig) error {
	return defaultScreenTimelapseRetentionConfigStore.Save(config)
}

func updateScreenTimelapseRetentionCleanup(expectedRevision int64, cleanup ScreenTimelapseRetentionCleanupConfig) (ScreenTimelapseRetentionConfig, bool, bool, bool, error) {
	return defaultScreenTimelapseRetentionConfigStore.UpdateCleanup(expectedRevision, cleanup)
}

func loadScreenTimelapseRetentionConfigFromPath(path string) (ScreenTimelapseRetentionConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return defaultScreenTimelapseRetentionConfig(), nil
		}
		return defaultScreenTimelapseRetentionConfig(), fmt.Errorf("read retention config: %w", err)
	}

	config, err := decodeScreenTimelapseRetentionConfig(data)
	if err != nil {
		return defaultScreenTimelapseRetentionConfig(), fmt.Errorf("decode retention config: %w", err)
	}
	return config, nil
}

func decodeScreenTimelapseRetentionConfig(data []byte) (ScreenTimelapseRetentionConfig, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	var stored screenTimelapseRetentionConfigFile
	if err := decoder.Decode(&stored); err != nil {
		return ScreenTimelapseRetentionConfig{}, fmt.Errorf("%w: invalid JSON: %v", errScreenTimelapseRetentionInvalidConfig, err)
	}
	var trailing interface{}
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return ScreenTimelapseRetentionConfig{}, fmt.Errorf("%w: multiple JSON values", errScreenTimelapseRetentionInvalidConfig)
		}
		return ScreenTimelapseRetentionConfig{}, fmt.Errorf("%w: trailing JSON: %v", errScreenTimelapseRetentionInvalidConfig, err)
	}

	if stored.Version == nil || stored.Revision == nil || stored.Cleanup == nil ||
		stored.Cleanup.Enabled == nil || stored.Cleanup.RetentionDays == nil || stored.Cleanup.MaxStorageMiB == nil {
		return ScreenTimelapseRetentionConfig{}, fmt.Errorf("%w: all fields are required", errScreenTimelapseRetentionInvalidConfig)
	}

	config := ScreenTimelapseRetentionConfig{
		Version:  *stored.Version,
		Revision: *stored.Revision,
		Cleanup: ScreenTimelapseRetentionCleanupConfig{
			Enabled:       *stored.Cleanup.Enabled,
			RetentionDays: *stored.Cleanup.RetentionDays,
			MaxStorageMiB: *stored.Cleanup.MaxStorageMiB,
		},
	}
	if err := validateScreenTimelapseRetentionConfig(config); err != nil {
		return ScreenTimelapseRetentionConfig{}, err
	}
	return config, nil
}

func validateScreenTimelapseRetentionConfig(config ScreenTimelapseRetentionConfig) error {
	if config.Version != screenTimelapseRetentionConfigVersion {
		return fmt.Errorf("%w: unsupported version %d", errScreenTimelapseRetentionInvalidConfig, config.Version)
	}
	if config.Revision < 0 {
		return fmt.Errorf("%w: revision must not be negative", errScreenTimelapseRetentionInvalidConfig)
	}
	if config.Cleanup.RetentionDays < 0 || config.Cleanup.RetentionDays > screenTimelapseRetentionMaxDays {
		return fmt.Errorf("%w: retentionDays must be between 0 and %d", errScreenTimelapseRetentionInvalidConfig, screenTimelapseRetentionMaxDays)
	}
	if config.Cleanup.MaxStorageMiB < 0 {
		return fmt.Errorf("%w: maxStorageMiB must not be negative", errScreenTimelapseRetentionInvalidConfig)
	}
	if config.Cleanup.Enabled && config.Cleanup.RetentionDays == 0 && config.Cleanup.MaxStorageMiB == 0 {
		return fmt.Errorf("%w: enabled cleanup requires a positive retention limit", errScreenTimelapseRetentionInvalidConfig)
	}
	return nil
}

func saveScreenTimelapseRetentionConfigToPath(path string, config ScreenTimelapseRetentionConfig) error {
	if err := validateScreenTimelapseRetentionConfig(config); err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(dir, ".timelapse.json.*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}

	dirFile, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer dirFile.Close()
	return dirFile.Sync()
}

func inventoryScreenTimelapseRetention(root string, now time.Time) (ScreenTimelapseRetentionInventory, error) {
	inventory := ScreenTimelapseRetentionInventory{
		Days:     make([]ScreenTimelapseRetentionInventoryDay, 0),
		Complete: true,
		Warnings: make([]ScreenTimelapseRetentionInventoryWarning, 0),
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return inventory, nil
		}
		return ScreenTimelapseRetentionInventory{}, err
	}

	today := now.In(now.Location()).Format("2006-01-02")
	for _, entry := range entries {
		name := entry.Name()
		path := filepath.Join(root, name)
		if !isValidTimelapseDate(name) {
			inventory.Warnings = append(inventory.Warnings, ScreenTimelapseRetentionInventoryWarning{
				Path: path, Reason: screenTimelapseRetentionWarningInvalid,
			})
			continue
		}

		info, err := os.Lstat(path)
		if err != nil {
			inventory.Complete = false
			inventory.Warnings = append(inventory.Warnings, ScreenTimelapseRetentionInventoryWarning{
				Path: path, Reason: screenTimelapseRetentionWarningWalk, Error: err.Error(),
			})
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			inventory.Warnings = append(inventory.Warnings, ScreenTimelapseRetentionInventoryWarning{
				Path: path, Reason: screenTimelapseRetentionWarningSymlink,
			})
			continue
		}
		if !info.IsDir() {
			inventory.Warnings = append(inventory.Warnings, ScreenTimelapseRetentionInventoryWarning{
				Path: path, Reason: screenTimelapseRetentionWarningNotDir,
			})
			continue
		}

		bytes, scanComplete, warnings := inventoryScreenTimelapseRetentionDirectory(path)
		inventory.Warnings = append(inventory.Warnings, warnings...)
		if !scanComplete {
			inventory.Complete = false
		}
		if bytes > 0 && inventory.TotalBytes > math.MaxInt64-bytes {
			return ScreenTimelapseRetentionInventory{}, errors.New("screen timelapse inventory size overflow")
		}
		inventory.TotalBytes += bytes

		day := ScreenTimelapseRetentionInventoryDay{
			Date:         name,
			Path:         path,
			Bytes:        bytes,
			ModTime:      info.ModTime(),
			IsToday:      name == today,
			IsFuture:     name > today,
			ScanComplete: scanComplete,
		}
		if day.IsFuture {
			inventory.Warnings = append(inventory.Warnings, ScreenTimelapseRetentionInventoryWarning{
				Path: path, Reason: screenTimelapseRetentionWarningFuture,
			})
		}
		inventory.Days = append(inventory.Days, day)
	}

	sort.Slice(inventory.Days, func(i, j int) bool {
		return inventory.Days[i].Date < inventory.Days[j].Date
	})
	return inventory, nil
}

func inventoryScreenTimelapseRetentionDirectory(path string) (int64, bool, []ScreenTimelapseRetentionInventoryWarning) {
	var (
		size     int64
		complete = true
		warnings []ScreenTimelapseRetentionInventoryWarning
	)

	err := filepath.WalkDir(path, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			complete = false
			warnings = append(warnings, ScreenTimelapseRetentionInventoryWarning{
				Path: current, Reason: screenTimelapseRetentionWarningWalk, Error: walkErr.Error(),
			})
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			warnings = append(warnings, ScreenTimelapseRetentionInventoryWarning{
				Path: current, Reason: screenTimelapseRetentionWarningSymlink,
			})
			return nil
		}
		if entry.IsDir() {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			complete = false
			warnings = append(warnings, ScreenTimelapseRetentionInventoryWarning{
				Path: current, Reason: screenTimelapseRetentionWarningWalk, Error: err.Error(),
			})
			return nil
		}
		if !info.Mode().IsRegular() {
			warnings = append(warnings, ScreenTimelapseRetentionInventoryWarning{
				Path: current, Reason: screenTimelapseRetentionWarningNonRegular,
			})
			return nil
		}
		if info.Size() > 0 && size > math.MaxInt64-info.Size() {
			return errors.New("screen timelapse day size overflow")
		}
		size += info.Size()
		return nil
	})
	if err != nil {
		complete = false
		warnings = append(warnings, ScreenTimelapseRetentionInventoryWarning{
			Path: path, Reason: screenTimelapseRetentionWarningWalk, Error: err.Error(),
		})
	}
	return size, complete, warnings
}

func planScreenTimelapseRetention(inventory ScreenTimelapseRetentionInventory, now time.Time, config ScreenTimelapseRetentionConfig) (ScreenTimelapseRetentionPlan, error) {
	if err := validateScreenTimelapseRetentionConfig(config); err != nil {
		return ScreenTimelapseRetentionPlan{}, err
	}

	days, beforeBytes, err := normalizedScreenTimelapseRetentionDays(inventory.Days)
	if err != nil {
		return ScreenTimelapseRetentionPlan{}, err
	}
	plan := ScreenTimelapseRetentionPlan{
		BeforeBytes:         beforeBytes,
		ProjectedAfterBytes: beforeBytes,
		Candidates:          make([]ScreenTimelapseRetentionPlanDay, 0),
		Skipped:             make([]ScreenTimelapseRetentionSkippedDay, 0),
		Warnings:            append([]ScreenTimelapseRetentionInventoryWarning(nil), inventory.Warnings...),
	}
	if !config.Cleanup.Enabled {
		return plan, nil
	}

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	eligible := make([]bool, len(days))
	for index, day := range days {
		dayTime, _ := time.ParseInLocation("2006-01-02", day.Date, now.Location())
		switch {
		case !inventory.Complete || !day.ScanComplete:
			plan.Skipped = append(plan.Skipped, ScreenTimelapseRetentionSkippedDay{
				Date: day.Date, Bytes: day.Bytes, Reason: screenTimelapseRetentionSkipIncomplete,
			})
		case dayTime.Equal(today):
			plan.Skipped = append(plan.Skipped, ScreenTimelapseRetentionSkippedDay{
				Date: day.Date, Bytes: day.Bytes, Reason: screenTimelapseRetentionSkipToday,
			})
		case dayTime.After(today):
			plan.Skipped = append(plan.Skipped, ScreenTimelapseRetentionSkippedDay{
				Date: day.Date, Bytes: day.Bytes, Reason: screenTimelapseRetentionSkipFuture,
			})
		case now.Sub(day.ModTime) < screenTimelapseRetentionQuietPeriod:
			plan.Skipped = append(plan.Skipped, ScreenTimelapseRetentionSkippedDay{
				Date: day.Date, Bytes: day.Bytes, Reason: screenTimelapseRetentionSkipRecentWrite,
			})
		default:
			eligible[index] = true
		}
	}

	candidates := make([]ScreenTimelapseRetentionPlanDay, len(days))
	selected := make([]bool, len(days))
	addReason := func(index int, reason string) {
		if !selected[index] {
			selected[index] = true
			candidates[index] = ScreenTimelapseRetentionPlanDay{
				Date:  days[index].Date,
				Bytes: days[index].Bytes,
			}
		}
		for _, existing := range candidates[index].Reasons {
			if existing == reason {
				return
			}
		}
		candidates[index].Reasons = append(candidates[index].Reasons, reason)
	}

	if config.Cleanup.RetentionDays > 0 {
		earliestKeptDay := today.AddDate(0, 0, -(config.Cleanup.RetentionDays - 1))
		for index, day := range days {
			if !eligible[index] {
				continue
			}
			dayTime, _ := time.ParseInLocation("2006-01-02", day.Date, now.Location())
			if dayTime.Before(earliestKeptDay) {
				addReason(index, screenTimelapseRetentionReasonExpired)
			}
		}
	}

	if config.Cleanup.MaxStorageMiB > 0 {
		quotaBytes := screenTimelapseRetentionStorageBytes(config.Cleanup.MaxStorageMiB)
		quotaRemaining := beforeBytes
		for index, day := range days {
			if quotaRemaining <= quotaBytes {
				break
			}
			if !eligible[index] {
				continue
			}
			addReason(index, screenTimelapseRetentionReasonQuota)
			quotaRemaining = subtractScreenTimelapseRetentionBytes(quotaRemaining, day.Bytes)
		}
	}

	for index, candidate := range candidates {
		if !selected[index] {
			continue
		}
		plan.Candidates = append(plan.Candidates, candidate)
		plan.ProjectedAfterBytes = subtractScreenTimelapseRetentionBytes(plan.ProjectedAfterBytes, candidate.Bytes)
	}
	if config.Cleanup.MaxStorageMiB > 0 {
		plan.OverLimit = plan.ProjectedAfterBytes > screenTimelapseRetentionStorageBytes(config.Cleanup.MaxStorageMiB)
	}
	return plan, nil
}

func normalizedScreenTimelapseRetentionDays(days []ScreenTimelapseRetentionInventoryDay) ([]ScreenTimelapseRetentionInventoryDay, int64, error) {
	normalized := append([]ScreenTimelapseRetentionInventoryDay(nil), days...)
	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].Date < normalized[j].Date
	})

	var total int64
	for index, day := range normalized {
		if !isValidTimelapseDate(day.Date) {
			return nil, 0, fmt.Errorf("invalid inventory date %q", day.Date)
		}
		if day.Bytes < 0 {
			return nil, 0, fmt.Errorf("negative inventory size for %q", day.Date)
		}
		if index > 0 && day.Date == normalized[index-1].Date {
			return nil, 0, fmt.Errorf("duplicate inventory date %q", day.Date)
		}
		if day.Bytes > 0 && total > math.MaxInt64-day.Bytes {
			return nil, 0, errors.New("screen timelapse planner size overflow")
		}
		total += day.Bytes
	}
	return normalized, total, nil
}

func screenTimelapseRetentionStorageBytes(storageMiB int64) int64 {
	if storageMiB <= 0 {
		return 0
	}
	if storageMiB > math.MaxInt64/screenTimelapseRetentionBytesPerMiB {
		return math.MaxInt64
	}
	return storageMiB * screenTimelapseRetentionBytesPerMiB
}

func subtractScreenTimelapseRetentionBytes(total int64, bytes int64) int64 {
	if bytes >= total {
		return 0
	}
	return total - bytes
}
