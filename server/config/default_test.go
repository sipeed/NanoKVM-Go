package config

import (
	"bytes"
	"errors"
	"os"
	"testing"
)

func TestCheckDefaultValueFillsTwoFactorAuth(t *testing.T) {
	original := instance
	t.Cleanup(func() { instance = original })

	if defaultConfig.TwoFactorAuth.SecretFile != "/etc/kvm/totp.secret" {
		t.Fatalf("expected default config secret file, got %q", defaultConfig.TwoFactorAuth.SecretFile)
	}
	if defaultConfig.TwoFactorAuth.Issuer != "NanoKVM" {
		t.Fatalf("expected default config issuer, got %q", defaultConfig.TwoFactorAuth.Issuer)
	}
	if defaultConfig.TwoFactorAuth.Window != 1 {
		t.Fatalf("expected default config window 1, got %d", defaultConfig.TwoFactorAuth.Window)
	}

	instance = Config{}
	if err := checkDefaultValue(); err != nil {
		t.Fatalf("checkDefaultValue: %v", err)
	}

	if instance.TwoFactorAuth.SecretFile != "/etc/kvm/totp.secret" {
		t.Fatalf("expected default secret file, got %q", instance.TwoFactorAuth.SecretFile)
	}
	if instance.TwoFactorAuth.Issuer != "NanoKVM" {
		t.Fatalf("expected default issuer, got %q", instance.TwoFactorAuth.Issuer)
	}
}

func TestCheckDefaultValueNormalizesInvalidTwoFactorWindow(t *testing.T) {
	original := instance
	t.Cleanup(func() { instance = original })

	for _, value := range []int{-1, 3} {
		instance = Config{TwoFactorAuth: TwoFactorAuth{SecretFile: "/tmp/totp.secret", Issuer: "NanoKVM", Window: value}}
		if err := checkDefaultValue(); err != nil {
			t.Fatalf("checkDefaultValue: %v", err)
		}

		if instance.TwoFactorAuth.Window != 1 {
			t.Fatalf("expected invalid window %d to normalize to 1, got %d", value, instance.TwoFactorAuth.Window)
		}
	}
}

func TestGenerateRandomSecretKeyFailsClosedOnEntropyError(t *testing.T) {
	original := readJWTSecretRandom
	readJWTSecretRandom = func([]byte) (int, error) { return 0, errors.New("entropy unavailable") }
	t.Cleanup(func() { readJWTSecretRandom = original })

	if key, err := generateRandomSecretKey(); err == nil || key != "" {
		t.Fatalf("generateRandomSecretKey = %q, %v; want empty key and error", key, err)
	}
}

func TestRegenerateJWTSecretKeyPreservesExistingKeyOnEntropyError(t *testing.T) {
	originalInstance := instance
	originalReader := readJWTSecretRandom
	instance.JWT.SecretKey = "existing-secret"
	readJWTSecretRandom = func([]byte) (int, error) { return 0, errors.New("entropy unavailable") }
	t.Cleanup(func() {
		instance = originalInstance
		readJWTSecretRandom = originalReader
	})

	if err := RegenerateJWTSecretKey(); err == nil {
		t.Fatal("expected entropy error")
	}
	if instance.JWT.SecretKey != "existing-secret" {
		t.Fatalf("existing JWT key changed to %q", instance.JWT.SecretKey)
	}
}

func TestRegenerateJWTSecretKeyPersistsBeforePublishing(t *testing.T) {
	path := useTempConfigurationFile(t)
	originalInstance := instance
	originalReader := readJWTSecretRandom
	instance = *defaultConfig
	instance.JWT.SecretKey = "existing-secret"
	readJWTSecretRandom = func(value []byte) (int, error) {
		for i := range value {
			value[i] = byte(i)
		}
		return len(value), nil
	}
	t.Cleanup(func() {
		instance = originalInstance
		readJWTSecretRandom = originalReader
	})

	if err := RegenerateJWTSecretKey(); err != nil {
		t.Fatalf("RegenerateJWTSecretKey returned error: %v", err)
	}
	if instance.JWT.SecretKey == "" || instance.JWT.SecretKey == "existing-secret" {
		t.Fatalf("JWT secret was not rotated: %q", instance.JWT.SecretKey)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read persisted configuration: %v", err)
	}
	if !bytes.Contains(data, []byte(instance.JWT.SecretKey)) {
		t.Fatal("persisted configuration does not contain the active JWT secret")
	}
	requireConfigurationFileMode(t, path, 0o600)
}

func TestRegenerateJWTSecretKeyPreservesExistingKeyOnSaveError(t *testing.T) {
	path := useTempConfigurationFile(t)
	originalInstance := instance
	originalReader := readJWTSecretRandom
	instance = *defaultConfig
	instance.JWT.SecretKey = "existing-secret"
	readJWTSecretRandom = func(value []byte) (int, error) {
		for i := range value {
			value[i] = 1
		}
		return len(value), nil
	}
	t.Cleanup(func() {
		instance = originalInstance
		readJWTSecretRandom = originalReader
	})

	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatalf("create invalid configuration target: %v", err)
	}
	if err := RegenerateJWTSecretKey(); err == nil {
		t.Fatal("expected configuration save error")
	}
	if instance.JWT.SecretKey != "existing-secret" {
		t.Fatalf("active JWT secret changed to %q after save failure", instance.JWT.SecretKey)
	}
}
