package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func prepareSipeedConfiguration(t *testing.T) string {
	t.Helper()
	path := useTempConfigurationFile(t)
	if err := save(*defaultConfig); err != nil {
		t.Fatal(err)
	}
	GetInstance()
	original := instance
	instance = *defaultConfig
	instance.JWT.SecretKey = "preserved-secret"
	t.Cleanup(func() { instance = original })
	return path
}

func TestSipeedDisclaimerPersistsAndReloads(t *testing.T) {
	path := prepareSipeedConfiguration(t)
	if GetSipeedDisclaimerAccepted() {
		t.Fatal("missing consent must default to false")
	}
	for _, accepted := range []bool{true, false} {
		if err := SetSipeedDisclaimerAccepted(accepted); err != nil {
			t.Fatal(err)
		}
		reader := viper.New()
		reader.SetConfigFile(path)
		if err := reader.ReadInConfig(); err != nil {
			t.Fatal(err)
		}
		var reloaded Config
		if err := reader.Unmarshal(&reloaded); err != nil {
			t.Fatal(err)
		}
		if reloaded.SipeedRemoteNetwork.DisclaimerAccepted != accepted || GetSipeedDisclaimerAccepted() != accepted {
			t.Fatalf("consent was not persisted and published: %+v", reloaded.SipeedRemoteNetwork)
		}
		if reloaded.JWT.SecretKey != "preserved-secret" || !reloaded.HDMICapture.Enabled {
			t.Fatal("saving consent changed unrelated settings")
		}
		requireConfigurationFileMode(t, path, 0o600)
	}
}

func TestSipeedDisclaimerSaveFailurePreservesConsent(t *testing.T) {
	path := prepareSipeedConfiguration(t)
	if err := SetSipeedDisclaimerAccepted(true); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	configurationFilePath = filepath.Join(path, "invalid.yaml")
	if err := SetSipeedDisclaimerAccepted(false); err == nil {
		t.Fatal("expected persistence failure")
	}
	if !GetSipeedDisclaimerAccepted() {
		t.Fatal("failed write changed active consent")
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(before) {
		t.Fatal("failed write changed persisted configuration")
	}
}
