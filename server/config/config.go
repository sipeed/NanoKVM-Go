package config

import (
	"bytes"
	"errors"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const ConfigurationFile = "/etc/kvm/server.yaml"

var (
	configurationFilePath  = ConfigurationFile
	chmodConfigurationFile = func(file *os.File, mode os.FileMode) error { return file.Chmod(mode) }
	instance               Config
	once                   sync.Once
	configMu               sync.Mutex
	hdmiCaptureEnabled     atomic.Bool
)

type configurationFilePermissionError struct {
	err error
}

func (e *configurationFilePermissionError) Error() string {
	return "secure configuration file permissions: " + e.err.Error()
}

func (e *configurationFilePermissionError) Unwrap() error {
	return e.err
}

func GetInstance() *Config {
	once.Do(initialize)

	return &instance
}

func initialize() {
	viper.SetDefault("hdmiCapture.enabled", true)
	viper.SetDefault("twoFactorAuth.secretFile", "/etc/kvm/totp.secret")
	viper.SetDefault("twoFactorAuth.issuer", "NanoKVM")
	viper.SetDefault("twoFactorAuth.window", 1)
	viper.SetDefault("fingerRobot", string(FingerRobotModeNormal))
	viper.SetDefault("serverFeatureSync.enabled", true)

	if err := readByFile(); err != nil {
		var permissionErr *configurationFilePermissionError
		if errors.As(err, &permissionErr) {
			log.Fatalf("Failed to load configuration securely: %s", permissionErr)
		}

		if errors.As(err, &viper.ConfigFileNotFoundError{}) {
			create()
		}

		if err = readByDefault(); err != nil {
			log.Fatalf("Failed to read default configuration!")
		}

		log.Println("using default configuration")
	}

	if err := validate(); err != nil {
		log.Fatalf("Failed to validate configuration!")
	}

	applyServerFeatureSyncSetting()

	if err := viper.Unmarshal(&instance); err != nil {
		log.Fatalf("Failed to parse configuration: %s", err)
	}

	if err := checkDefaultValue(); err != nil {
		log.Fatalf("Failed to generate JWT secret key: %s", err)
	}
	hdmiCaptureEnabled.Store(instance.HDMICapture.Enabled)

	if instance.Authentication == "disable" {
		log.Println("NOTICE: Authentication is disabled! Please ensure your service is secure!")
	}

	log.Println("config loaded successfully")
}

func readByFile() error {
	viper.SetConfigFile(configurationFilePath)
	viper.SetConfigType("yaml")

	if err := secureConfigurationFilePermissions(configurationFilePath); err != nil {
		// Preserve Viper's existing missing-file classification so initialize
		// can create a default. Every existing but unsafe path fails closed
		// before Viper is allowed to parse it.
		if errors.Is(err, os.ErrNotExist) {
			return viper.ConfigFileNotFoundError{}
		}
		return &configurationFilePermissionError{err: err}
	}
	return viper.ReadInConfig()
}

func secureConfigurationFilePermissions(path string) error {
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("configuration path is not a regular file")
	}
	return chmodConfigurationFile(file, 0o600)
}

func readByDefault() error {
	data, err := yaml.Marshal(defaultConfig)
	if err != nil {
		log.Printf("failed to marshal default config: %s", err)
		return err
	}

	return viper.ReadConfig(bytes.NewBuffer(data))
}

// Create configuration file.
func create() {
	var (
		file *os.File
		data []byte
		err  error
	)

	dir := filepath.Dir(configurationFilePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		_ = os.MkdirAll(dir, 0o755)
	}

	file, err = os.OpenFile(configurationFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		log.Printf("open config failed: %s", err)
		return
	}
	defer func() {
		_ = file.Close()
	}()
	if err = file.Chmod(0o600); err != nil {
		log.Printf("failed to secure config: %s", err)
		return
	}

	if data, err = yaml.Marshal(defaultConfig); err != nil {
		log.Printf("failed to marshal default config: %s", err)
		return
	}

	if _, err = file.Write(data); err != nil {
		log.Printf("failed to save config: %s", err)
		return
	}

	if err = file.Sync(); err != nil {
		log.Printf("failed to sync config: %s", err)
		return
	}

	log.Printf("create file %s with default configuration", configurationFilePath)
}

func validate() error {
	if viper.GetInt("port.http") > 0 && viper.GetInt("port.https") > 0 {
		return nil
	}

	log.Println("configuration invalid, resetting to defaults")

	create()

	return readByDefault()
}

func GetFingerRobotMode() FingerRobotMode {
	return GetInstance().FingerRobot
}

func IsFingerRobotDebug() bool {
	return GetFingerRobotMode() == FingerRobotModeDebug
}

func GetHDMICaptureEnabled() bool {
	GetInstance()
	return hdmiCaptureEnabled.Load()
}

// normalizeServerFeatureSyncSection returns the sanitized serverFeatureSync
// section for Viper. It never fails: an absent, null, scalar, list, or
// malformed section collapses to enabled=true, and a map keeps an explicit
// bool `enabled` while forcing true for anything else. Sanitizing before
// viper.Unmarshal is required because a garbage bool is otherwise a fatal
// unmarshal error, and null / "" / 0 silently decode to false instead of the
// documented "invalid means true" default. The caller must not log the raw
// value.
func normalizeServerFeatureSyncSection(raw any) map[string]any {
	enabled := true
	if section, ok := raw.(map[string]any); ok {
		if value, ok := section["enabled"].(bool); ok {
			enabled = value
		}
	}
	return map[string]any{"enabled": enabled}
}

// applyServerFeatureSyncSetting replaces the raw serverFeatureSync section with
// its sanitized form before unmarshal. A section whose enabled leaf is a real
// bool is honored as-is (unknown sibling keys are dropped like every unknown
// key on save); anything else that is present at all is malformed and logs a
// fixed message. An absent or null section stays silent: it legitimately
// normalizes to the default without user error. The raw value is never logged.
func applyServerFeatureSyncSetting() {
	const key = "serverFeatureSync"

	raw := viper.Get(key)
	section := normalizeServerFeatureSyncSection(raw)
	if raw != nil {
		rawSection, ok := raw.(map[string]any)
		if _, valid := rawSection["enabled"].(bool); !ok || !valid {
			log.Printf("invalid %s setting; using enabled=true", key)
		}
	}
	viper.Set(key, section)
}

// GetServerFeatureSyncEnabled reports whether NanoKVM-Server may refresh
// /etc/kvm/server-feature.json from the CDN. It is read-only: no setter exists,
// so only a manual server.yaml edit changes it, effective on process restart.
func GetServerFeatureSyncEnabled() bool {
	GetInstance()

	configMu.Lock()
	defer configMu.Unlock()
	return instance.ServerFeatureSync.Enabled
}

// ErrInvalidOnboardingVersion rejects a completedVersion that is empty or
// falls outside the bounded safe charset. Services map it to an
// invalid-arguments response instead of a persistence failure.
var ErrInvalidOnboardingVersion = errors.New("invalid onboarding completedVersion")

var onboardingVersionPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{0,31}$`)

// IsValidOnboardingVersion reports whether version satisfies the bounded
// charset shared by the desktop and mobile onboarding namespaces.
func IsValidOnboardingVersion(version string) bool {
	return onboardingVersionPattern.MatchString(version)
}

// GetOnboardingCompletedVersion returns the device-level completed tour
// version; empty means the onboarding guide has never completed.
func GetOnboardingCompletedVersion() string {
	GetInstance()

	configMu.Lock()
	defer configMu.Unlock()
	return instance.Onboarding.CompletedVersion
}

// SetOnboardingCompletedVersion persists the desktop completed tour version
// through the same lock-copy-save-update path as SetHDMICaptureEnabled.
func SetOnboardingCompletedVersion(version string) error {
	if !IsValidOnboardingVersion(version) {
		return ErrInvalidOnboardingVersion
	}

	GetInstance()

	configMu.Lock()
	defer configMu.Unlock()

	updated := instance
	updated.Onboarding.CompletedVersion = version
	if err := save(updated); err != nil {
		return err
	}

	instance.Onboarding.CompletedVersion = version
	viper.Set("onboarding.completedVersion", version)
	return nil
}

// GetMobileOnboardingCompletedVersion returns the device-level completed mobile
// portrait tour version; empty means the mobile guide has never completed.
func GetMobileOnboardingCompletedVersion() string {
	GetInstance()

	configMu.Lock()
	defer configMu.Unlock()
	return instance.Onboarding.MobileCompletedVersion
}

// SetMobileOnboardingCompletedVersion persists the completed mobile tour
// version through the same lock-copy-save-update path as the desktop setter.
func SetMobileOnboardingCompletedVersion(version string) error {
	if !IsValidOnboardingVersion(version) {
		return ErrInvalidOnboardingVersion
	}

	GetInstance()

	configMu.Lock()
	defer configMu.Unlock()

	updated := instance
	updated.Onboarding.MobileCompletedVersion = version
	if err := save(updated); err != nil {
		return err
	}

	instance.Onboarding.MobileCompletedVersion = version
	viper.Set("onboarding.mobileCompletedVersion", version)
	return nil
}

func SetHDMICaptureEnabled(enabled bool) error {
	GetInstance()

	configMu.Lock()
	defer configMu.Unlock()

	updated := instance
	updated.HDMICapture.Enabled = enabled
	if err := save(updated); err != nil {
		return err
	}

	instance.HDMICapture.Enabled = enabled
	viper.Set("hdmiCapture.enabled", enabled)
	hdmiCaptureEnabled.Store(enabled)
	return nil
}

func save(conf Config) error {
	dir := filepath.Dir(configurationFilePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}

	file, err := os.CreateTemp(dir, ".server.yaml.*")
	if err != nil {
		return err
	}
	tmpPath := file.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, configurationFilePath); err != nil {
		return err
	}

	dirFile, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer dirFile.Close()
	return dirFile.Sync()
}
