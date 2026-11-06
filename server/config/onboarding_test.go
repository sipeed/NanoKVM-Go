package config

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func parseOnboardingVersionForTest(t *testing.T, data string) string {
	t.Helper()

	reader := viper.New()
	reader.SetConfigType("yaml")
	if err := reader.ReadConfig(strings.NewReader(data)); err != nil {
		t.Fatalf("read config: %v", err)
	}

	var conf Config
	if err := reader.Unmarshal(&conf); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}

	return conf.Onboarding.CompletedVersion
}

func parseMobileOnboardingVersionForTest(t *testing.T, data string) string {
	t.Helper()

	reader := viper.New()
	reader.SetConfigType("yaml")
	if err := reader.ReadConfig(strings.NewReader(data)); err != nil {
		t.Fatalf("read config: %v", err)
	}

	var conf Config
	if err := reader.Unmarshal(&conf); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}

	return conf.Onboarding.MobileCompletedVersion
}

func TestOnboardingVersionDefaultsToEmptyWhenMissing(t *testing.T) {
	got := parseOnboardingVersionForTest(t, "proto: http\n")
	if got != "" {
		t.Fatalf("legacy config without onboarding completedVersion = %q, want empty", got)
	}
}

func TestOnboardingVersionParsesWhenPresent(t *testing.T) {
	got := parseOnboardingVersionForTest(t, "onboarding:\n  completedVersion: v1.2\n")
	if got != "v1.2" {
		t.Fatalf("onboarding completedVersion = %q, want %q", got, "v1.2")
	}
}

func TestCheckDefaultValueNormalizesInvalidOnboardingVersion(t *testing.T) {
	originalInstance := instance
	t.Cleanup(func() { instance = originalInstance })

	instance = *defaultConfig
	instance.JWT.SecretKey = "existing-secret"
	instance.Onboarding.CompletedVersion = "INVALID VERSION!!"

	if err := checkDefaultValue(); err != nil {
		t.Fatalf("checkDefaultValue returned error: %v", err)
	}
	if instance.Onboarding.CompletedVersion != "" {
		t.Fatalf("invalid onboarding completedVersion = %q, want normalized empty",
			instance.Onboarding.CompletedVersion)
	}
}

func TestCheckDefaultValueKeepsValidOnboardingVersion(t *testing.T) {
	originalInstance := instance
	t.Cleanup(func() { instance = originalInstance })

	instance = *defaultConfig
	instance.JWT.SecretKey = "existing-secret"
	instance.Onboarding.CompletedVersion = "v1"

	if err := checkDefaultValue(); err != nil {
		t.Fatalf("checkDefaultValue returned error: %v", err)
	}
	if instance.Onboarding.CompletedVersion != "v1" {
		t.Fatalf("valid onboarding completedVersion = %q, want %q",
			instance.Onboarding.CompletedVersion, "v1")
	}
}

func TestSetOnboardingCompletedVersionPersistsAtomically(t *testing.T) {
	path := useTempConfigurationFile(t)
	originalInstance := instance
	t.Cleanup(func() { instance = originalInstance })

	if err := SetOnboardingCompletedVersion("v1"); err != nil {
		t.Fatalf("SetOnboardingCompletedVersion returned error: %v", err)
	}
	if got := GetOnboardingCompletedVersion(); got != "v1" {
		t.Fatalf("GetOnboardingCompletedVersion = %q, want %q", got, "v1")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read persisted configuration: %v", err)
	}
	if !strings.Contains(string(data), "completedVersion: v1") {
		t.Fatalf("persisted configuration does not contain the completed version:\n%s", data)
	}
	requireConfigurationFileMode(t, path, 0o600)
}

func TestSetOnboardingCompletedVersionPersistsAndReloads(t *testing.T) {
	path := useTempConfigurationFile(t)
	originalInstance := instance
	t.Cleanup(func() { instance = originalInstance })

	if err := SetOnboardingCompletedVersion("v1.2"); err != nil {
		t.Fatalf("SetOnboardingCompletedVersion returned error: %v", err)
	}

	// Simulate a service restart: a fresh viper read of the persisted file must
	// return the same completed version (AC8).
	reader := viper.New()
	reader.SetConfigFile(path)
	if err := reader.ReadInConfig(); err != nil {
		t.Fatalf("reload persisted configuration: %v", err)
	}
	var reloaded Config
	if err := reader.Unmarshal(&reloaded); err != nil {
		t.Fatalf("unmarshal reloaded configuration: %v", err)
	}
	if reloaded.Onboarding.CompletedVersion != "v1.2" {
		t.Fatalf("reloaded completedVersion = %q, want %q",
			reloaded.Onboarding.CompletedVersion, "v1.2")
	}
	requireConfigurationFileMode(t, path, 0o600)
}

func TestSetOnboardingCompletedVersionRejectsInvalidValues(t *testing.T) {
	useTempConfigurationFile(t)
	originalInstance := instance
	t.Cleanup(func() { instance = originalInstance })

	invalid := []string{
		"",
		"V1",
		" v1",
		"v1 ",
		".v1",
		"-v1",
		"v1;rm",
		"v1/inject",
		"v1_inject",
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", // 33 chars
	}
	for _, version := range invalid {
		if err := SetOnboardingCompletedVersion(version); !errors.Is(err, ErrInvalidOnboardingVersion) {
			t.Fatalf("SetOnboardingCompletedVersion(%q) error = %v, want ErrInvalidOnboardingVersion",
				version, err)
		}
	}

	if got := GetOnboardingCompletedVersion(); got != "" {
		t.Fatalf("rejected writes changed completedVersion to %q, want empty", got)
	}
}

func TestSetOnboardingCompletedVersionAcceptsBoundedValues(t *testing.T) {
	useTempConfigurationFile(t)
	originalInstance := instance
	t.Cleanup(func() { instance = originalInstance })

	valid := []string{"v1", "1", "v1.2.3", "tour-2026-09", "a.b-c.d"}
	for _, version := range valid {
		if err := SetOnboardingCompletedVersion(version); err != nil {
			t.Fatalf("SetOnboardingCompletedVersion(%q) returned error: %v", version, err)
		}
		if got := GetOnboardingCompletedVersion(); got != version {
			t.Fatalf("GetOnboardingCompletedVersion = %q, want %q", got, version)
		}
	}
}

func TestMobileOnboardingVersionDefaultsToEmptyWhenMissing(t *testing.T) {
	got := parseMobileOnboardingVersionForTest(t, "proto: http\n")
	if got != "" {
		t.Fatalf("legacy config without onboarding mobileCompletedVersion = %q, want empty", got)
	}
}

func TestMobileOnboardingVersionParsesWhenPresent(t *testing.T) {
	got := parseMobileOnboardingVersionForTest(t, "onboarding:\n  mobileCompletedVersion: mobile-v1\n")
	if got != "mobile-v1" {
		t.Fatalf("onboarding mobileCompletedVersion = %q, want %q", got, "mobile-v1")
	}
}

func TestCheckDefaultValueNormalizesInvalidMobileOnboardingVersion(t *testing.T) {
	originalInstance := instance
	t.Cleanup(func() { instance = originalInstance })

	instance = *defaultConfig
	instance.JWT.SecretKey = "existing-secret"
	instance.Onboarding.MobileCompletedVersion = "INVALID VERSION!!"

	if err := checkDefaultValue(); err != nil {
		t.Fatalf("checkDefaultValue returned error: %v", err)
	}
	if instance.Onboarding.MobileCompletedVersion != "" {
		t.Fatalf("invalid mobile onboarding mobileCompletedVersion = %q, want normalized empty",
			instance.Onboarding.MobileCompletedVersion)
	}
}

func TestCheckDefaultValueKeepsValidMobileOnboardingVersion(t *testing.T) {
	originalInstance := instance
	t.Cleanup(func() { instance = originalInstance })

	instance = *defaultConfig
	instance.JWT.SecretKey = "existing-secret"
	instance.Onboarding.MobileCompletedVersion = "mobile-v1"

	if err := checkDefaultValue(); err != nil {
		t.Fatalf("checkDefaultValue returned error: %v", err)
	}
	if instance.Onboarding.MobileCompletedVersion != "mobile-v1" {
		t.Fatalf("valid mobile onboarding mobileCompletedVersion = %q, want %q",
			instance.Onboarding.MobileCompletedVersion, "mobile-v1")
	}
}

func TestSetMobileOnboardingCompletedVersionPersistsAtomically(t *testing.T) {
	path := useTempConfigurationFile(t)
	originalInstance := instance
	t.Cleanup(func() { instance = originalInstance })

	instance = *defaultConfig
	instance.JWT.SecretKey = "existing-secret"

	if err := SetMobileOnboardingCompletedVersion("mobile-v1"); err != nil {
		t.Fatalf("SetMobileOnboardingCompletedVersion returned error: %v", err)
	}
	if got := GetMobileOnboardingCompletedVersion(); got != "mobile-v1" {
		t.Fatalf("GetMobileOnboardingCompletedVersion = %q, want %q", got, "mobile-v1")
	}
	if got := GetOnboardingCompletedVersion(); got != "" {
		t.Fatalf("mobile write changed desktop completedVersion to %q, want empty", got)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read persisted configuration: %v", err)
	}
	if !strings.Contains(string(data), "mobileCompletedVersion: mobile-v1") {
		t.Fatalf("persisted configuration does not contain the mobile completed version:\n%s", data)
	}
	requireConfigurationFileMode(t, path, 0o600)
}

func TestSetMobileOnboardingCompletedVersionRejectsInvalidValues(t *testing.T) {
	useTempConfigurationFile(t)
	originalInstance := instance
	t.Cleanup(func() { instance = originalInstance })

	instance = *defaultConfig
	instance.JWT.SecretKey = "existing-secret"

	invalid := []string{
		"",
		"MOBILE-v1",
		" mobile-v1",
		"mobile-v1 ",
		".mobile-v1",
		"-mobile-v1",
		"mobile-v1;rm",
		"mobile-v1/inject",
		"mobile_v1",
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", // 33 chars
	}
	for _, version := range invalid {
		if err := SetMobileOnboardingCompletedVersion(version); !errors.Is(err, ErrInvalidOnboardingVersion) {
			t.Fatalf("SetMobileOnboardingCompletedVersion(%q) error = %v, want ErrInvalidOnboardingVersion",
				version, err)
		}
	}

	if got := GetMobileOnboardingCompletedVersion(); got != "" {
		t.Fatalf("rejected writes changed mobileCompletedVersion to %q, want empty", got)
	}
}

func TestOnboardingVersionNamespacesAreIndependent(t *testing.T) {
	useTempConfigurationFile(t)
	originalInstance := instance
	t.Cleanup(func() { instance = originalInstance })

	instance = *defaultConfig
	instance.JWT.SecretKey = "existing-secret"

	if err := SetOnboardingCompletedVersion("v1"); err != nil {
		t.Fatalf("SetOnboardingCompletedVersion returned error: %v", err)
	}
	if err := SetMobileOnboardingCompletedVersion("mobile-v1"); err != nil {
		t.Fatalf("SetMobileOnboardingCompletedVersion returned error: %v", err)
	}
	if got := GetOnboardingCompletedVersion(); got != "v1" {
		t.Fatalf("desktop completedVersion = %q, want %q", got, "v1")
	}
	if got := GetMobileOnboardingCompletedVersion(); got != "mobile-v1" {
		t.Fatalf("mobile completedVersion = %q, want %q", got, "mobile-v1")
	}

	// A later desktop-only update must not disturb the mobile namespace.
	if err := SetOnboardingCompletedVersion("v2"); err != nil {
		t.Fatalf("SetOnboardingCompletedVersion returned error: %v", err)
	}
	if got := GetMobileOnboardingCompletedVersion(); got != "mobile-v1" {
		t.Fatalf("desktop update changed mobile completedVersion to %q, want %q", got, "mobile-v1")
	}
	if got := GetOnboardingCompletedVersion(); got != "v2" {
		t.Fatalf("desktop completedVersion = %q, want %q", got, "v2")
	}
}
