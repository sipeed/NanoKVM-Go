package vm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sync"
)

const (
	FingerRobotConfigFile    = "/etc/kvm/finger-robot.json"
	FingerRobotConfigVersion = 1

	FingerRobotDefaultPressMS = 800

	FingerRobotSpeedSlow FingerRobotSpeed = "slow"
	FingerRobotSpeedFast FingerRobotSpeed = "fast"

	FingerRobotSlowSpeedUSPerSecond = 1000
	FingerRobotFastSpeedUSPerSecond = 2000
)

var (
	ErrFingerRobotConfigInvalid           = errors.New("invalid finger robot config")
	ErrFingerRobotConfigRevisionConflict  = errors.New("finger robot config revision conflict")
	ErrFingerRobotConfigRevisionExhausted = errors.New("finger robot config revision exhausted")
)

// FingerRobotSpeed is a product-level choice. Arbitrary pulse speeds are not
// persisted or accepted from the normal user API.
type FingerRobotSpeed string

// FingerRobotStrengthByArm keeps an independent user-selected strength for
// each replaceable arm. Its fixed shape also lets the JSON decoder reject
// unknown profile IDs instead of silently accepting arbitrary map keys.
type FingerRobotStrengthByArm struct {
	Long  int `json:"long"`
	Short int `json:"short"`
}

// FingerRobotConfig is the complete, versioned on-disk configuration.
// Revision starts at one and is increased only when user settings change.
type FingerRobotConfig struct {
	Version       int                      `json:"version"`
	Revision      int64                    `json:"revision"`
	ActiveArm     string                   `json:"activeArm"`
	StrengthByArm FingerRobotStrengthByArm `json:"strengthByArm"`
	PressMS       int                      `json:"pressMs"`
	Speed         FingerRobotSpeed         `json:"speed"`
}

// FingerRobotConfigUpdate contains every user-selectable setting. Updates are
// complete replacements rather than patches, so omitted fields fail normal
// validation instead of inheriting stale values.
type FingerRobotConfigUpdate struct {
	ActiveArm     string                   `json:"activeArm"`
	StrengthByArm FingerRobotStrengthByArm `json:"strengthByArm"`
	PressMS       int                      `json:"pressMs"`
	Speed         FingerRobotSpeed         `json:"speed"`
}

type fingerRobotConfigStore struct {
	mu   sync.Mutex
	path string
}

type fingerRobotConfigFile struct {
	Version       *int                          `json:"version"`
	Revision      *int64                        `json:"revision"`
	ActiveArm     *string                       `json:"activeArm"`
	StrengthByArm *fingerRobotStrengthByArmFile `json:"strengthByArm"`
	PressMS       *int                          `json:"pressMs"`
	Speed         *FingerRobotSpeed             `json:"speed"`
}

type fingerRobotStrengthByArmFile struct {
	Long  *int `json:"long"`
	Short *int `json:"short"`
}

var defaultFingerRobotConfigStore = newFingerRobotConfigStore(FingerRobotConfigFile)

func newFingerRobotConfigStore(path string) *fingerRobotConfigStore {
	return &fingerRobotConfigStore{path: path}
}

// DefaultFingerRobotConfig returns the safe first-boot selection. It does not
// access the filesystem.
func DefaultFingerRobotConfig() FingerRobotConfig {
	return FingerRobotConfig{
		Version:   FingerRobotConfigVersion,
		Revision:  1,
		ActiveArm: FingerRobotArmProfileShortID,
		StrengthByArm: FingerRobotStrengthByArm{
			Long:  FingerRobotDefaultStrengthLevel,
			Short: FingerRobotDefaultStrengthLevel,
		},
		PressMS: FingerRobotDefaultPressMS,
		Speed:   FingerRobotSpeedSlow,
	}
}

// LoadFingerRobotConfig reads the current configuration. On first boot it
// atomically creates and returns the safe default. An existing invalid file is
// never overwritten automatically.
func LoadFingerRobotConfig() (FingerRobotConfig, error) {
	return defaultFingerRobotConfigStore.Load()
}

// UpdateFingerRobotConfig performs one serialized read/check/write cycle.
// The caller must supply the revision it observed. A no-op succeeds without
// rewriting the file or increasing its revision.
func UpdateFingerRobotConfig(expectedRevision int64, update FingerRobotConfigUpdate) (FingerRobotConfig, bool, error) {
	return defaultFingerRobotConfigStore.Update(expectedRevision, update)
}

func (store *fingerRobotConfigStore) Load() (FingerRobotConfig, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.loadOrCreateLocked()
}

func (store *fingerRobotConfigStore) Update(expectedRevision int64, update FingerRobotConfigUpdate) (FingerRobotConfig, bool, error) {
	if expectedRevision < 1 {
		return FingerRobotConfig{}, false, fmt.Errorf("%w: expected revision must be positive", ErrFingerRobotConfigInvalid)
	}
	if err := validateFingerRobotConfigUpdate(update); err != nil {
		return FingerRobotConfig{}, false, err
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	current, err := store.loadOrCreateLocked()
	if err != nil {
		return FingerRobotConfig{}, false, err
	}
	if current.Revision != expectedRevision {
		return current, false, fmt.Errorf(
			"%w: expected %d, current %d",
			ErrFingerRobotConfigRevisionConflict,
			expectedRevision,
			current.Revision,
		)
	}
	if current.userSettings() == update {
		return current, false, nil
	}
	if current.Revision == math.MaxInt64 {
		return current, false, ErrFingerRobotConfigRevisionExhausted
	}

	updated := FingerRobotConfig{
		Version:       FingerRobotConfigVersion,
		Revision:      current.Revision + 1,
		ActiveArm:     update.ActiveArm,
		StrengthByArm: update.StrengthByArm,
		PressMS:       update.PressMS,
		Speed:         update.Speed,
	}
	if err := saveFingerRobotConfigToPath(store.path, updated); err != nil {
		return current, false, err
	}
	return updated, true, nil
}

func (store *fingerRobotConfigStore) loadOrCreateLocked() (FingerRobotConfig, error) {
	config, err := loadFingerRobotConfigFromPath(store.path)
	if err == nil {
		return config, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return FingerRobotConfig{}, err
	}

	config = DefaultFingerRobotConfig()
	if err := saveFingerRobotConfigToPath(store.path, config); err != nil {
		return FingerRobotConfig{}, fmt.Errorf("create default finger robot config: %w", err)
	}
	return config, nil
}

func (config FingerRobotConfig) userSettings() FingerRobotConfigUpdate {
	return FingerRobotConfigUpdate{
		ActiveArm:     config.ActiveArm,
		StrengthByArm: config.StrengthByArm,
		PressMS:       config.PressMS,
		Speed:         config.Speed,
	}
}

// StrengthForArm resolves the stored selection without exposing a map that a
// caller could mutate or extend with an unknown profile ID.
func (strengths FingerRobotStrengthByArm) StrengthForArm(profileID string) (int, error) {
	switch profileID {
	case FingerRobotArmProfileLongID:
		return strengths.Long, nil
	case FingerRobotArmProfileShortID:
		return strengths.Short, nil
	default:
		return 0, fmt.Errorf("%w: unknown arm profile %q", ErrFingerRobotConfigInvalid, profileID)
	}
}

// ResolveFingerRobotSpeedUSPerSecond maps the two persisted choices to fixed
// backend rates. Normal callers never accept an arbitrary numeric speed.
func ResolveFingerRobotSpeedUSPerSecond(speed FingerRobotSpeed) (int, error) {
	switch speed {
	case FingerRobotSpeedSlow:
		return FingerRobotSlowSpeedUSPerSecond, nil
	case FingerRobotSpeedFast:
		return FingerRobotFastSpeedUSPerSecond, nil
	default:
		return 0, fmt.Errorf("%w: speed must be %q or %q", ErrFingerRobotConfigInvalid, FingerRobotSpeedSlow, FingerRobotSpeedFast)
	}
}

func loadFingerRobotConfigFromPath(path string) (FingerRobotConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FingerRobotConfig{}, fmt.Errorf("read finger robot config: %w", err)
	}
	config, err := decodeFingerRobotConfig(data)
	if err != nil {
		return FingerRobotConfig{}, fmt.Errorf("decode finger robot config: %w", err)
	}
	return config, nil
}

func decodeFingerRobotConfig(data []byte) (FingerRobotConfig, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	var stored fingerRobotConfigFile
	if err := decoder.Decode(&stored); err != nil {
		return FingerRobotConfig{}, fmt.Errorf("%w: invalid JSON: %v", ErrFingerRobotConfigInvalid, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return FingerRobotConfig{}, fmt.Errorf("%w: multiple JSON values", ErrFingerRobotConfigInvalid)
		}
		return FingerRobotConfig{}, fmt.Errorf("%w: trailing JSON: %v", ErrFingerRobotConfigInvalid, err)
	}

	if stored.Version == nil || stored.Revision == nil || stored.ActiveArm == nil ||
		stored.StrengthByArm == nil || stored.StrengthByArm.Long == nil || stored.StrengthByArm.Short == nil ||
		stored.PressMS == nil || stored.Speed == nil {
		return FingerRobotConfig{}, fmt.Errorf("%w: all fields are required", ErrFingerRobotConfigInvalid)
	}

	config := FingerRobotConfig{
		Version:   *stored.Version,
		Revision:  *stored.Revision,
		ActiveArm: *stored.ActiveArm,
		StrengthByArm: FingerRobotStrengthByArm{
			Long:  *stored.StrengthByArm.Long,
			Short: *stored.StrengthByArm.Short,
		},
		PressMS: *stored.PressMS,
		Speed:   *stored.Speed,
	}
	if err := validateFingerRobotConfig(config); err != nil {
		return FingerRobotConfig{}, err
	}
	return config, nil
}

func validateFingerRobotConfig(config FingerRobotConfig) error {
	if config.Version != FingerRobotConfigVersion {
		return fmt.Errorf("%w: unsupported version %d", ErrFingerRobotConfigInvalid, config.Version)
	}
	if config.Revision < 1 {
		return fmt.Errorf("%w: revision must be positive", ErrFingerRobotConfigInvalid)
	}
	return validateFingerRobotConfigUpdate(config.userSettings())
}

func validateFingerRobotConfigUpdate(update FingerRobotConfigUpdate) error {
	if err := ValidateFingerRobotArmProfiles(); err != nil {
		return fmt.Errorf("%w: built-in arm profiles failed validation: %v", ErrFingerRobotConfigInvalid, err)
	}
	if _, err := GetFingerRobotArmProfile(update.ActiveArm); err != nil {
		return fmt.Errorf("%w: unknown activeArm %q", ErrFingerRobotConfigInvalid, update.ActiveArm)
	}
	for _, profileID := range [...]string{FingerRobotArmProfileLongID, FingerRobotArmProfileShortID} {
		strength, err := update.StrengthByArm.StrengthForArm(profileID)
		if err != nil {
			return err
		}
		if _, err := ResolveFingerRobotPressUS(profileID, strength); err != nil {
			return fmt.Errorf("%w: strength for %q must be between 1 and %d", ErrFingerRobotConfigInvalid, profileID, FingerRobotStrengthLevelCount)
		}
	}

	limits := GetFingerRobotLimits()
	if update.PressMS < limits.PressMinMS || update.PressMS > limits.PressMaxMS {
		return fmt.Errorf(
			"%w: pressMs must be between %d and %d",
			ErrFingerRobotConfigInvalid,
			limits.PressMinMS,
			limits.PressMaxMS,
		)
	}
	if _, err := ResolveFingerRobotSpeedUSPerSecond(update.Speed); err != nil {
		return err
	}
	return nil
}

func saveFingerRobotConfigToPath(path string, config FingerRobotConfig) error {
	if err := validateFingerRobotConfig(config); err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create finger robot config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("encode finger robot config: %w", err)
	}
	data = append(data, '\n')

	temporary, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("create temporary finger robot config: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() {
		_ = os.Remove(temporaryPath)
	}()

	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("set temporary finger robot config permissions: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write temporary finger robot config: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync temporary finger robot config: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary finger robot config: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace finger robot config: %w", err)
	}

	directory, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("open finger robot config directory: %w", err)
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return fmt.Errorf("sync finger robot config directory: %w", err)
	}
	return nil
}
