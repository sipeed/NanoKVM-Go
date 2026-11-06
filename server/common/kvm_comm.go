package common

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"NanoKVM-Server/utils"

	"github.com/pelletier/go-toml/v2"
)

const (
	kvmCommConfigPath = "/etc/kvm/kvm_vin.toml"
	kvmVinExecPath    = "/var/lib/nanokvm-runtime/kvmcomm/vin/kvm_vin"
	kvmVinControlSock = "/run/kvm/vin_ctrl.sock"

	kvmCommConfigMode os.FileMode = 0o644

	kvmCommDeviceSection             = "device"
	kvmCommEncoderSection            = "encoder"
	kvmCommLogSection                = "log"
	kvmCommSnapshotSection           = "snapshot"
	kvmCommTimelapseSection          = "timelapse"
	kvmCommDeviceMode                = "mode"
	kvmCommIdleTimeoutMS             = "idle_timeout_ms"
	kvmCommEncoderRateControl        = "rate_control"
	kvmCommEncoderBitRate            = "bitrate"
	kvmCommEncoderGOP                = "gop"
	kvmCommEncoderFPS                = "fps"
	kvmCommSnapshotEnable            = "enable"
	kvmCommRetiredSnapshotIntervalMS = "interval_ms" // Removed when the config is next saved.
	kvmCommSnapshotPeriodicEnable    = "periodic_enable"
	kvmCommTimelapseEnable           = "enable"
	kvmCommTimelapseQP               = "qp"
	kvmCommTempFilePattern           = ".kvm_vin.*.toml"
	kvmCommDefaultIdlePollMS         = 500
	kvmCommDefaultIdleTimeoutMS      = 3000
	kvmCommEncoderBitRateMin         = 1000
	kvmCommEncoderBitRateMax         = 20000
	kvmCommDefaultEncoderBitRate     = 8000
	kvmCommEncoderGOPMin             = 1
	kvmCommEncoderGOPMax             = 300
	kvmCommDefaultEncoderGOP         = 50
	kvmCommEncoderFPSMin             = 0
	kvmCommEncoderFPSMax             = 120
	kvmCommDefaultEncoderFPS         = 0
	kvmCommDefaultLogFile            = "/var/log/kvmcomm/vin.log"
	kvmCommDefaultLogLevel           = "error"
	kvmCommDefaultLogType            = "file"
	kvmVinRestartTimeout             = 30 * time.Second
)

const (
	KvmCommVideoModeH264 = "h264"
	KvmCommVideoModeH265 = "h265"

	KvmCommEncoderRateControlCBR = "cbr"
	KvmCommEncoderRateControlVBR = "vbr"

	KvmCommTimelapseQPMin     = 36
	KvmCommTimelapseQPMax     = 51
	KvmCommTimelapseQPDefault = 42
)

var (
	ErrKvmCommLoadConfigFailed          = errors.New("failed to load kvmcomm config")
	ErrKvmCommUpdateConfigFailed        = errors.New("failed to update kvmcomm config")
	ErrKvmCommSaveConfigFailed          = errors.New("failed to save kvmcomm config")
	ErrKvmCommInvalidVideoMode          = errors.New("invalid kvmcomm video mode")
	ErrKvmCommInvalidEncoderSetting     = errors.New("invalid kvmcomm encoder setting")
	ErrKvmCommInvalidTimelapseSetting   = errors.New("invalid kvmcomm timelapse setting")
	ErrKvmCommRestartFailed             = errors.New("failed to restart kvm_vin process")
	ErrKvmCommRestartAndRestoreFailed   = errors.New("failed to restart kvm_vin process and restore config")
	ErrKvmCommRestartAfterRestoreFailed = errors.New("failed to restart kvm_vin process after restoring config")
)

var (
	kvmCommConfigFilePath = kvmCommConfigPath
	kvmCommConfigMu       sync.RWMutex
	restartWatchedProcess = utils.RestartWatchedProcess
	syncKvmCommConfigDir  = func(dir string) error {
		dirFile, err := os.Open(dir)
		if err != nil {
			return err
		}
		defer dirFile.Close()
		return dirFile.Sync()
	}
	dialKvmVinControlSock = func(ctx context.Context) error {
		var dialer net.Dialer
		conn, err := dialer.DialContext(ctx, "unix", kvmVinControlSock)
		if err != nil {
			return err
		}
		return conn.Close()
	}
)

type KvmCommConfig struct {
	Device    KvmCommDeviceConfig    `toml:"device"`
	Encoder   KvmCommEncoderConfig   `toml:"encoder"`
	Snapshot  KvmCommSnapshotConfig  `toml:"snapshot"`
	Timelapse KvmCommTimelapseConfig `toml:"timelapse"`
	Log       KvmCommLogConfig       `toml:"log"`
}

type KvmCommDeviceConfig struct {
	Mode               string `toml:"mode"`
	IdlePollMS         int    `toml:"idle_poll_ms"`
	IdlePowerOffLT7911 bool   `toml:"idle_power_off_lt7911"`
	IdleTimeoutMS      int    `toml:"idle_timeout_ms"`
}

type KvmCommEncoderConfig struct {
	RateControl string `toml:"rate_control"`
	BitRate     int    `toml:"bitrate"`
	GOP         int    `toml:"gop"`
	FPS         int    `toml:"fps"`
}

type KvmCommLogConfig struct {
	File  string `toml:"file"`
	Level string `toml:"level"`
	Type  string `toml:"type"`
}

type KvmCommSnapshotConfig struct {
	Enable         bool `toml:"enable"`
	PeriodicEnable bool `toml:"periodic_enable"`
}

type KvmCommTimelapseConfig struct {
	Enable bool `toml:"enable"`
	QP     int  `toml:"qp"`
}

type kvmCommConfig map[string]any
type kvmCommTomlTable = map[string]any

type kvmCommConfigSnapshot struct {
	data   []byte
	exists bool
	mode   os.FileMode
}

func GetKvmCommConfig() (*KvmCommConfig, error) {
	kvmCommConfigMu.RLock()
	defer kvmCommConfigMu.RUnlock()

	_, config, err := loadKvmCommConfigWithSnapshot()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrKvmCommLoadConfigFailed, err)
	}

	structured, err := config.toStruct()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrKvmCommLoadConfigFailed, err)
	}

	return structured, nil
}

func SetKvmCommEncoderRateControl(mode string) error {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode != KvmCommEncoderRateControlCBR && mode != KvmCommEncoderRateControlVBR {
		return fmt.Errorf("%w: rate_control=%q", ErrKvmCommInvalidEncoderSetting, mode)
	}

	return updateKvmCommConfig(false, 0, func(config kvmCommConfig) (bool, error) {
		encoder, err := config.table(kvmCommEncoderSection)
		if err != nil {
			return false, err
		}

		current, _ := encoder[kvmCommEncoderRateControl].(string)
		encoder[kvmCommEncoderRateControl] = mode
		return !strings.EqualFold(current, mode), nil
	})
}

func SetKvmCommEncoderBitRate(bitRate int) error {
	if bitRate < kvmCommEncoderBitRateMin || bitRate > kvmCommEncoderBitRateMax {
		return fmt.Errorf("%w: bitrate=%d", ErrKvmCommInvalidEncoderSetting, bitRate)
	}

	return updateKvmCommEncoderNumber(kvmCommEncoderBitRate, bitRate)
}

func SetKvmCommEncoderGOP(gop int) error {
	if gop < kvmCommEncoderGOPMin || gop > kvmCommEncoderGOPMax {
		return fmt.Errorf("%w: gop=%d", ErrKvmCommInvalidEncoderSetting, gop)
	}

	return updateKvmCommEncoderNumber(kvmCommEncoderGOP, gop)
}

func SetKvmCommEncoderFPS(fps int) error {
	if fps < kvmCommEncoderFPSMin || fps > kvmCommEncoderFPSMax {
		return fmt.Errorf("%w: fps=%d", ErrKvmCommInvalidEncoderSetting, fps)
	}

	return updateKvmCommEncoderNumber(kvmCommEncoderFPS, fps)
}

func updateKvmCommEncoderNumber(key string, value int) error {
	return updateKvmCommConfig(false, 0, func(config kvmCommConfig) (bool, error) {
		encoder, err := config.table(kvmCommEncoderSection)
		if err != nil {
			return false, err
		}

		current, _ := numericTomlValue[int](encoder[key])
		encoder[key] = value
		return current != value, nil
	})
}

func SetKvmCommIdleTimeoutEnabled(enabled bool, restart bool, restartDelay time.Duration) error {
	return updateKvmCommConfig(restart, restartDelay, func(config kvmCommConfig) (bool, error) {
		device, err := config.table(kvmCommDeviceSection)
		if err != nil {
			return false, err
		}

		targetTimeout := 0
		if enabled {
			targetTimeout = kvmCommDefaultIdleTimeoutMS
		}

		current, _ := numericTomlValue[int](device[kvmCommIdleTimeoutMS])
		device[kvmCommIdleTimeoutMS] = targetTimeout
		return current != targetTimeout, nil
	})
}

func SetKvmCommTimelapseEnabled(enabled bool, restart bool, restartDelay time.Duration) error {
	return updateKvmCommConfig(restart, restartDelay, func(config kvmCommConfig) (bool, error) {
		timelapse, err := config.table(kvmCommTimelapseSection)
		if err != nil {
			return false, err
		}

		current, _ := timelapse[kvmCommTimelapseEnable].(bool)
		timelapse[kvmCommTimelapseEnable] = enabled
		return current != enabled, nil
	})
}

// SetKvmCommTimelapseQP persists the dedicated Timelapse encoder fixed QP.
// It restarts kvm_vin only when Timelapse is currently enabled so disabled
// services can be configured without interrupting the live video path.
func SetKvmCommTimelapseQP(qp int, restartDelay time.Duration) (bool, bool, error) {
	if qp < KvmCommTimelapseQPMin || qp > KvmCommTimelapseQPMax {
		return false, false, fmt.Errorf("%w: qp=%d", ErrKvmCommInvalidTimelapseSetting, qp)
	}

	return updateKvmCommConfigWithRestartDecision(
		restartDelay,
		func(config kvmCommConfig) (bool, bool, error) {
			timelapse, err := config.table(kvmCommTimelapseSection)
			if err != nil {
				return false, false, err
			}

			current, _ := numericTomlValue[int](timelapse[kvmCommTimelapseQP])
			enabled, _ := timelapse[kvmCommTimelapseEnable].(bool)
			timelapse[kvmCommTimelapseQP] = qp
			return current != qp, enabled, nil
		},
	)
}

func SetKvmCommAIRuntime(snapshotEnabled bool, snapshotPeriodicEnabled bool, timelapseEnabled bool, restart bool, restartDelay time.Duration) error {
	return updateKvmCommConfig(restart, restartDelay, func(config kvmCommConfig) (bool, error) {
		snapshot, err := config.table(kvmCommSnapshotSection)
		if err != nil {
			return false, err
		}

		currentSnapshotEnabled, _ := snapshot[kvmCommSnapshotEnable].(bool)
		currentSnapshotPeriodicEnabled, _ := snapshot[kvmCommSnapshotPeriodicEnable].(bool)
		snapshot[kvmCommSnapshotEnable] = snapshotEnabled
		snapshot[kvmCommSnapshotPeriodicEnable] = snapshotPeriodicEnabled

		timelapse, err := config.table(kvmCommTimelapseSection)
		if err != nil {
			return false, err
		}
		currentTimelapseEnabled, _ := timelapse[kvmCommTimelapseEnable].(bool)
		timelapse[kvmCommTimelapseEnable] = timelapseEnabled

		return currentSnapshotEnabled != snapshotEnabled ||
			currentSnapshotPeriodicEnabled != snapshotPeriodicEnabled ||
			currentTimelapseEnabled != timelapseEnabled, nil
	})
}

func SetKvmCommVideoMode(mode string, restart bool, restartDelay time.Duration) error {
	if mode != KvmCommVideoModeH264 && mode != KvmCommVideoModeH265 {
		return fmt.Errorf("%w: %s", ErrKvmCommInvalidVideoMode, mode)
	}

	return updateKvmCommConfig(restart, restartDelay, func(config kvmCommConfig) (bool, error) {
		device, err := config.table(kvmCommDeviceSection)
		if err != nil {
			return false, err
		}

		current, _ := device[kvmCommDeviceMode].(string)
		device[kvmCommDeviceMode] = mode
		return !strings.EqualFold(current, mode), nil
	})
}

func updateKvmCommConfig(restart bool, restartDelay time.Duration, update func(kvmCommConfig) (bool, error)) error {
	_, _, err := updateKvmCommConfigWithRestartDecision(
		restartDelay,
		func(config kvmCommConfig) (bool, bool, error) {
			changed, err := update(config)
			return changed, restart, err
		},
	)
	return err
}

func updateKvmCommConfigWithRestartDecision(
	restartDelay time.Duration,
	update func(kvmCommConfig) (changed bool, restart bool, err error),
) (bool, bool, error) {
	kvmCommConfigMu.Lock()
	defer kvmCommConfigMu.Unlock()

	snapshot, config, err := loadKvmCommConfigWithSnapshot()
	if err != nil {
		return false, false, fmt.Errorf("%w: %v", ErrKvmCommLoadConfigFailed, err)
	}

	changed, restart, err := update(config)
	if err != nil {
		return false, false, fmt.Errorf("%w: %v", ErrKvmCommUpdateConfigFailed, err)
	}

	if err := saveKvmCommConfig(config, snapshot); err != nil {
		if restoreErr := snapshot.restore(); restoreErr != nil {
			return false, false, fmt.Errorf("%w: save: %v; restore: %v", ErrKvmCommSaveConfigFailed, err, restoreErr)
		}
		return false, false, fmt.Errorf("%w: %v", ErrKvmCommSaveConfigFailed, err)
	}

	if !restart || !changed {
		return changed, false, nil
	}

	if err := restartKvmVinProcess(restartDelay); err != nil {
		if restoreErr := snapshot.restore(); restoreErr != nil {
			return false, false, fmt.Errorf("%w: restart: %v; restore: %v", ErrKvmCommRestartAndRestoreFailed, err, restoreErr)
		}

		if restoreRestartErr := restartKvmVinProcess(restartDelay); restoreRestartErr != nil {
			return false, false, fmt.Errorf("%w: restart: %v; restart after restore: %v", ErrKvmCommRestartAfterRestoreFailed, err, restoreRestartErr)
		}

		return false, false, fmt.Errorf("%w: %v", ErrKvmCommRestartFailed, err)
	}

	return true, true, nil
}

func loadKvmCommConfigWithSnapshot() (kvmCommConfigSnapshot, kvmCommConfig, error) {
	snapshot, err := readKvmCommConfigSnapshot()
	if err != nil {
		return kvmCommConfigSnapshot{}, nil, fmt.Errorf("read config: %w", err)
	}

	config, err := snapshot.parse()
	if err != nil {
		return kvmCommConfigSnapshot{}, nil, fmt.Errorf("parse config: %w", err)
	}

	return snapshot, config, nil
}

func readKvmCommConfigSnapshot() (kvmCommConfigSnapshot, error) {
	data, err := os.ReadFile(kvmCommConfigFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return kvmCommConfigSnapshot{}, nil
		}
		return kvmCommConfigSnapshot{}, err
	}

	info, err := os.Stat(kvmCommConfigFilePath)
	if err != nil {
		return kvmCommConfigSnapshot{}, err
	}

	return kvmCommConfigSnapshot{
		data:   data,
		exists: true,
		mode:   info.Mode().Perm(),
	}, nil
}

func (snapshot kvmCommConfigSnapshot) parse() (kvmCommConfig, error) {
	config := kvmCommConfig{}
	if snapshot.exists {
		if err := toml.Unmarshal(snapshot.data, &config); err != nil {
			return nil, err
		}
	}

	if err := config.applyDefaults(); err != nil {
		return nil, err
	}

	return config, nil
}

func (snapshot kvmCommConfigSnapshot) writeMode() os.FileMode {
	if snapshot.exists {
		return snapshot.mode
	}
	return kvmCommConfigMode
}

func (snapshot kvmCommConfigSnapshot) restore() error {
	if !snapshot.exists {
		if err := os.Remove(kvmCommConfigFilePath); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}

	return writeKvmCommConfigData(snapshot.data, snapshot.mode)
}

func saveKvmCommConfig(config kvmCommConfig, snapshot kvmCommConfigSnapshot) error {
	if err := config.applyDefaults(); err != nil {
		return err
	}
	config.cleanupDeprecated()

	data, err := toml.Marshal(config)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if snapshot.exists && bytes.Equal(snapshot.data, data) {
		return nil
	}

	return writeKvmCommConfigData(data, snapshot.writeMode())
}

func writeKvmCommConfigData(data []byte, mode os.FileMode) error {
	dir := filepath.Dir(kvmCommConfigFilePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// Write through a temp file so readers never observe a partial TOML file.
	temp, err := os.CreateTemp(dir, kvmCommTempFilePattern)
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer func() {
		_ = os.Remove(tempPath)
	}()

	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}

	if err := temp.Chmod(mode); err != nil {
		_ = temp.Close()
		return err
	}

	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}

	if err := temp.Close(); err != nil {
		return err
	}

	if err := os.Rename(tempPath, kvmCommConfigFilePath); err != nil {
		return err
	}

	return syncKvmCommConfigDir(dir)
}

func restartKvmVinProcess(_ time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), kvmVinRestartTimeout)
	defer cancel()

	return restartWatchedProcess(ctx, utils.RestartWatchedProcessOptions{
		ExecPath:   kvmVinExecPath,
		ReadyCheck: dialKvmVinControlSock,
	})
}

func (config kvmCommConfig) toStruct() (*KvmCommConfig, error) {
	data, err := toml.Marshal(config)
	if err != nil {
		return nil, err
	}

	var structured KvmCommConfig
	if err := toml.Unmarshal(data, &structured); err != nil {
		return nil, err
	}

	return &structured, nil
}

func (config kvmCommConfig) applyDefaults() error {
	device, err := config.table(kvmCommDeviceSection)
	if err != nil {
		return err
	}
	setKvmCommTomlDefaults(device, kvmCommTomlTable{
		kvmCommDeviceMode:    KvmCommVideoModeH264,
		"idle_poll_ms":       kvmCommDefaultIdlePollMS,
		kvmCommIdleTimeoutMS: kvmCommDefaultIdleTimeoutMS,
	})

	encoder, err := config.table(kvmCommEncoderSection)
	if err != nil {
		encoder = kvmCommTomlTable{}
		config[kvmCommEncoderSection] = encoder
	}
	normalizeKvmCommEncoderConfig(encoder)

	snapshot, err := config.table(kvmCommSnapshotSection)
	if err != nil {
		return err
	}
	setKvmCommTomlDefaults(snapshot, kvmCommTomlTable{
		kvmCommSnapshotEnable:         false,
		kvmCommSnapshotPeriodicEnable: false,
	})

	timelapse, err := config.table(kvmCommTimelapseSection)
	if err != nil {
		return err
	}
	setKvmCommTomlDefaults(timelapse, kvmCommTomlTable{
		kvmCommTimelapseEnable: false,
	})
	normalizeKvmCommEncoderNumber(
		timelapse,
		kvmCommTimelapseQP,
		KvmCommTimelapseQPDefault,
		KvmCommTimelapseQPMin,
		KvmCommTimelapseQPMax,
	)

	logConfig, err := config.table(kvmCommLogSection)
	if err != nil {
		return err
	}
	setKvmCommTomlDefaults(logConfig, kvmCommTomlTable{
		"file":  kvmCommDefaultLogFile,
		"level": kvmCommDefaultLogLevel,
		"type":  kvmCommDefaultLogType,
	})

	return nil
}

func (config kvmCommConfig) cleanupDeprecated() {
	if device, err := config.table(kvmCommDeviceSection); err == nil {
		delete(device, "type")
	}
	if snapshot, err := config.table(kvmCommSnapshotSection); err == nil {
		delete(snapshot, kvmCommRetiredSnapshotIntervalMS)
	}
	if timelapse, err := config.table(kvmCommTimelapseSection); err == nil {
		delete(timelapse, "bitrate")
	}
}

func (config kvmCommConfig) table(name string) (kvmCommTomlTable, error) {
	raw, exists := config[name]
	if !exists {
		table := kvmCommTomlTable{}
		config[name] = table
		return table, nil
	}

	table, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid %s config section", name)
	}

	return table, nil
}

func setKvmCommTomlDefaults(table kvmCommTomlTable, defaults kvmCommTomlTable) {
	for key, value := range defaults {
		if _, exists := table[key]; !exists {
			table[key] = value
		}
	}
}

func normalizeKvmCommEncoderConfig(encoder kvmCommTomlTable) {
	rateControl, ok := encoder[kvmCommEncoderRateControl].(string)
	if !ok || (rateControl != KvmCommEncoderRateControlCBR && rateControl != KvmCommEncoderRateControlVBR) {
		rateControl = KvmCommEncoderRateControlVBR
	}
	encoder[kvmCommEncoderRateControl] = rateControl

	normalizeKvmCommEncoderNumber(
		encoder,
		kvmCommEncoderBitRate,
		kvmCommDefaultEncoderBitRate,
		kvmCommEncoderBitRateMin,
		kvmCommEncoderBitRateMax,
	)
	normalizeKvmCommEncoderNumber(
		encoder,
		kvmCommEncoderGOP,
		kvmCommDefaultEncoderGOP,
		kvmCommEncoderGOPMin,
		kvmCommEncoderGOPMax,
	)
	normalizeKvmCommEncoderNumber(
		encoder,
		kvmCommEncoderFPS,
		kvmCommDefaultEncoderFPS,
		kvmCommEncoderFPSMin,
		kvmCommEncoderFPSMax,
	)
}

func normalizeKvmCommEncoderNumber(
	encoder kvmCommTomlTable,
	key string,
	defaultValue int,
	minValue int,
	maxValue int,
) {
	value, ok := numericTomlValue[int](encoder[key])
	if !ok || value < minValue || value > maxValue {
		value = defaultValue
	}
	encoder[key] = value
}

func numericTomlValue[T ~int](value any) (T, bool) {
	switch v := value.(type) {
	case int:
		return T(v), true
	case int64:
		return T(v), true
	case int32:
		return T(v), true
	case uint:
		return T(v), true
	case uint64:
		return T(v), true
	case uint32:
		return T(v), true
	default:
		var zero T
		return zero, false
	}
}
