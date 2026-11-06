package config

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// parseServerFeatureSyncEnabledForTest mirrors the production load path for
// just the serverFeatureSync section: read YAML, sanitize the raw section, then
// unmarshal. It must never fatal on malformed input.
func parseServerFeatureSyncEnabledForTest(t *testing.T, data string) bool {
	t.Helper()

	reader := viper.New()
	reader.SetConfigType("yaml")
	if err := reader.ReadConfig(strings.NewReader(data)); err != nil {
		t.Fatalf("read config: %v", err)
	}
	reader.Set("serverFeatureSync", normalizeServerFeatureSyncSection(reader.Get("serverFeatureSync")))

	var conf Config
	if err := reader.Unmarshal(&conf); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	return conf.ServerFeatureSync.Enabled
}

func reloadConfigurationForTest(t *testing.T, path string) Config {
	t.Helper()

	reader := viper.New()
	reader.SetConfigFile(path)
	if err := reader.ReadInConfig(); err != nil {
		t.Fatalf("reload configuration: %v", err)
	}

	var conf Config
	if err := reader.Unmarshal(&conf); err != nil {
		t.Fatalf("unmarshal reloaded configuration: %v", err)
	}
	return conf
}

func TestCreateConfigurationFileEnablesServerFeatureSync(t *testing.T) {
	path := useTempConfigurationFile(t)

	create()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read persisted configuration: %v", err)
	}
	if !strings.Contains(string(data), "serverFeatureSync:") {
		t.Fatalf("persisted configuration does not contain serverFeatureSync:\n%s", data)
	}

	// The typed reload is the authoritative assertion for the value.
	conf := reloadConfigurationForTest(t, path)
	if !conf.ServerFeatureSync.Enabled {
		t.Fatal("fresh configuration did not enable serverFeatureSync")
	}
}

func TestServerFeatureSyncResolvesEnabledForEveryShape(t *testing.T) {
	cases := []struct {
		name string
		data string
		want bool
	}{
		{"missing key", "proto: http\n", true},
		{"null section", "serverFeatureSync:\n", true},
		{"null leaf", "serverFeatureSync:\n  enabled:\n", true},
		{"quoted false", "serverFeatureSync:\n  enabled: \"false\"\n", true},
		{"garbage leaf", "serverFeatureSync:\n  enabled: garbage\n", true},
		{"zero leaf", "serverFeatureSync:\n  enabled: 0\n", true},
		{"float zero leaf", "serverFeatureSync:\n  enabled: 0.0\n", true},
		{"empty string leaf", "serverFeatureSync:\n  enabled: \"\"\n", true},
		{"one leaf", "serverFeatureSync:\n  enabled: 1\n", true},
		{"nested map leaf", "serverFeatureSync:\n  enabled:\n    a: b\n", true},
		{"scalar section", "serverFeatureSync: garbage\n", true},
		{"list section", "serverFeatureSync:\n  - 1\n  - 2\n", true},
		{"valid false", "serverFeatureSync:\n  enabled: false\n", false},
		{"valid true", "serverFeatureSync:\n  enabled: true\n", true},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := parseServerFeatureSyncEnabledForTest(t, testCase.data); got != testCase.want {
				t.Fatalf("serverFeatureSync.enabled = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestApplyServerFeatureSyncSettingNormalizesBeforeUnmarshal(t *testing.T) {
	cases := []struct {
		name string
		data string
		want bool
	}{
		{"missing key", "proto: http\n", true},
		{"null section", "serverFeatureSync:\n", true},
		{"null leaf", "serverFeatureSync:\n  enabled:\n", true},
		{"garbage leaf", "serverFeatureSync:\n  enabled: garbage\n", true},
		{"map leaf", "serverFeatureSync:\n  enabled: {a: b}\n", true},
		{"scalar section", "serverFeatureSync: garbage\n", true},
		{"list section", "serverFeatureSync:\n  - 1\n  - 2\n", true},
		{"valid false", "serverFeatureSync:\n  enabled: false\n", false},
		{"valid true", "serverFeatureSync:\n  enabled: true\n", true},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)

			// Mirror the production ordering in initialize(): defaults first,
			// then the file, then the sanitizer, then unmarshal.
			viper.SetDefault("serverFeatureSync.enabled", true)
			viper.SetConfigType("yaml")
			if err := viper.ReadConfig(strings.NewReader(testCase.data)); err != nil {
				t.Fatalf("read config: %v", err)
			}

			applyServerFeatureSyncSetting()

			var conf Config
			if err := viper.Unmarshal(&conf); err != nil {
				t.Fatalf("unmarshal config: %v", err)
			}
			if conf.ServerFeatureSync.Enabled != testCase.want {
				t.Fatalf("serverFeatureSync.enabled = %v, want %v",
					conf.ServerFeatureSync.Enabled, testCase.want)
			}
		})
	}
}

func TestNormalizeServerFeatureSyncSection(t *testing.T) {
	cases := []struct {
		name string
		raw  any
		want bool
	}{
		{"nil", nil, true},
		{"bool true", map[string]any{"enabled": true}, true},
		{"bool false", map[string]any{"enabled": false}, false},
		{"missing enabled", map[string]any{"other": true}, true},
		{"non-bool enabled", map[string]any{"enabled": "false"}, true},
		{"nil enabled", map[string]any{"enabled": nil}, true},
		{"nested map enabled", map[string]any{"enabled": map[string]any{"a": "b"}}, true},
		{"scalar section", "garbage", true},
		{"list section", []any{1, 2}, true},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := normalizeServerFeatureSyncSection(testCase.raw)
			if len(got) != 1 {
				t.Fatalf("normalized section = %#v, want a single enabled key", got)
			}
			if got["enabled"] != testCase.want {
				t.Fatalf("normalized enabled = %v, want %v", got["enabled"], testCase.want)
			}
			if !reflect.DeepEqual(normalizeServerFeatureSyncSection(got), got) {
				t.Fatalf("normalize is not idempotent for %#v", got)
			}
		})
	}
}

func TestDefaultConfigEnablesServerFeatureSync(t *testing.T) {
	if !defaultConfig.ServerFeatureSync.Enabled {
		t.Fatal("defaultConfig must enable serverFeatureSync")
	}
}

func TestGetServerFeatureSyncEnabledReflectsSingleton(t *testing.T) {
	useTempConfigurationFile(t)
	GetInstance()

	originalInstance := instance
	t.Cleanup(func() { instance = originalInstance })

	instance = *defaultConfig
	instance.JWT.SecretKey = "existing-secret"

	instance.ServerFeatureSync.Enabled = false
	if GetServerFeatureSyncEnabled() {
		t.Fatal("getter returned true for a disabled singleton")
	}

	instance.ServerFeatureSync.Enabled = true
	if !GetServerFeatureSyncEnabled() {
		t.Fatal("getter returned false for an enabled singleton")
	}
}

func TestServerFeatureSyncSurvivesUnrelatedSave(t *testing.T) {
	path := useTempConfigurationFile(t)
	GetInstance()

	originalInstance := instance
	t.Cleanup(func() { instance = originalInstance })

	instance = *defaultConfig
	instance.JWT.SecretKey = "preserved-secret"
	instance.ServerFeatureSync.Enabled = false

	if err := SetOnboardingCompletedVersion("v1"); err != nil {
		t.Fatalf("SetOnboardingCompletedVersion returned error: %v", err)
	}

	if GetServerFeatureSyncEnabled() {
		t.Fatal("unrelated save changed the active serverFeatureSync value")
	}

	reloaded := reloadConfigurationForTest(t, path)
	if reloaded.ServerFeatureSync.Enabled {
		t.Fatal("unrelated save dropped the manually disabled serverFeatureSync value")
	}
	if reloaded.JWT.SecretKey != "preserved-secret" {
		t.Fatalf("unrelated save changed the JWT secret to %q", reloaded.JWT.SecretKey)
	}
	if !reloaded.HDMICapture.Enabled {
		t.Fatal("unrelated save changed the HDMI capture setting")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read persisted configuration: %v", err)
	}
	if !strings.Contains(string(data), "enabled: false") {
		t.Fatalf("persisted configuration does not contain the disabled value:\n%s", data)
	}
	requireConfigurationFileMode(t, path, 0o600)
}

func TestSavePersistsNormalizedServerFeatureSync(t *testing.T) {
	path := useTempConfigurationFile(t)
	GetInstance()

	originalInstance := instance
	t.Cleanup(func() { instance = originalInstance })

	instance = *defaultConfig
	instance.JWT.SecretKey = "existing-secret"
	instance.ServerFeatureSync.Enabled = true

	if err := save(instance); err != nil {
		t.Fatalf("save configuration: %v", err)
	}

	if got := reloadConfigurationForTest(t, path).ServerFeatureSync.Enabled; !got {
		t.Fatal("save did not persist the normalized enabled value")
	}
}
