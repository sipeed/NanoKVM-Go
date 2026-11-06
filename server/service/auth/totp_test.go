package auth

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"NanoKVM-Server/config"

	log "github.com/sirupsen/logrus"
)

func useTempTOTPSecretFile(t *testing.T) string {
	t.Helper()

	originalSecretFilePath := totpSecretFilePath
	originalIssuer := totpIssuer
	originalWindow := totpWindow
	originalNow := totpNow
	originalHostnameFilePath := totpHostnameFilePath

	path := filepath.Join(t.TempDir(), "totp.secret")
	totpSecretFilePath = path
	totpIssuer = "NanoKVM"
	totpWindow = 1
	totpNow = time.Now
	totpHostnameFilePath = filepath.Join(t.TempDir(), "hostname")
	resetPendingTOTPSetupsForTest()

	t.Cleanup(func() {
		totpSecretFilePath = originalSecretFilePath
		totpIssuer = originalIssuer
		totpWindow = originalWindow
		totpNow = originalNow
		totpHostnameFilePath = originalHostnameFilePath
		resetPendingTOTPSetupsForTest()
	})

	return path
}

func TestPendingTOTPSetupUsesHostnameAsDefaultIssuer(t *testing.T) {
	useTempTOTPSecretFile(t)
	if err := os.WriteFile(totpHostnameFilePath, []byte("rack-kvm\n"), 0o644); err != nil {
		t.Fatalf("WriteFile hostname returned error: %v", err)
	}

	_, uri, err := createPendingTOTPSetup("admin")
	if err != nil {
		t.Fatalf("createPendingTOTPSetup returned error: %v", err)
	}

	if !strings.Contains(uri, "otpauth://totp/rack-kvm%3Aadmin") {
		t.Fatalf("expected TOTP label to use hostname, got %q", uri)
	}
	if !strings.Contains(uri, "issuer=rack-kvm") {
		t.Fatalf("expected TOTP issuer to use hostname, got %q", uri)
	}
	if strings.Contains(uri, "NanoKVM") {
		t.Fatalf("TOTP URI should not use the legacy shared issuer, got %q", uri)
	}
}

func resetPendingTOTPSetupsForTest() {
	pendingTOTPSetups.Lock()
	defer pendingTOTPSetups.Unlock()
	pendingTOTPSetups.items = make(map[string]pendingTOTPSetup)
}

func TestIsTOTPEnabledMissingFile(t *testing.T) {
	useTempTOTPSecretFile(t)

	enabled, err := isTOTPEnabled()
	if err != nil {
		t.Fatalf("isTOTPEnabled returned error: %v", err)
	}
	if enabled {
		t.Fatal("missing TOTP secret file should be disabled")
	}
}

func TestIsTOTPEnabledEmptyFile(t *testing.T) {
	path := useTempTOTPSecretFile(t)

	if err := os.WriteFile(path, []byte("\n\t "), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	enabled, err := isTOTPEnabled()
	if err != nil {
		t.Fatalf("isTOTPEnabled returned error: %v", err)
	}
	if enabled {
		t.Fatal("empty TOTP secret file should be disabled")
	}
}

func TestTOTPSecretEnabledWhenNonEmpty(t *testing.T) {
	path := useTempTOTPSecretFile(t)

	if err := os.WriteFile(path, []byte(" ABCDEFG \n"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	secret, err := readActiveTOTPSecret()
	if err != nil {
		t.Fatalf("readActiveTOTPSecret returned error: %v", err)
	}
	if secret != "ABCDEFG" {
		t.Fatalf("expected trimmed secret ABCDEFG, got %q", secret)
	}

	enabled, err := isTOTPEnabled()
	if err != nil {
		t.Fatalf("isTOTPEnabled returned error: %v", err)
	}
	if !enabled {
		t.Fatal("non-empty TOTP secret should be enabled")
	}
}

func TestTOTPSecretWriteUses0600Mode(t *testing.T) {
	path := useTempTOTPSecretFile(t)

	if err := writeActiveTOTPSecret("ABCDEFG"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat returned error: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("expected TOTP secret mode 0600, got %04o", mode)
	}
}

func TestClearActiveTOTPSecretDisablesTOTP(t *testing.T) {
	path := useTempTOTPSecretFile(t)

	if err := writeActiveTOTPSecret("ABCDEFG"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}
	if err := clearActiveTOTPSecret(); err != nil {
		t.Fatalf("clearActiveTOTPSecret returned error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if len(data) != 0 {
		t.Fatalf("expected clearActiveTOTPSecret to write an empty file, got %q", data)
	}

	enabled, err := isTOTPEnabled()
	if err != nil {
		t.Fatalf("isTOTPEnabled returned error: %v", err)
	}
	if enabled {
		t.Fatal("cleared TOTP secret should be disabled")
	}
}

func TestTOTPSecretWriteDoesNotLogSecret(t *testing.T) {
	useTempTOTPSecretFile(t)

	secret := "ABCDEFGH23456789"
	var buf bytes.Buffer
	originalOutput := log.StandardLogger().Out
	log.SetOutput(&buf)
	t.Cleanup(func() {
		log.SetOutput(originalOutput)
	})

	if err := writeActiveTOTPSecret(secret); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}
	if strings.Contains(buf.String(), secret) {
		t.Fatal("secret was written to logs")
	}
}

func TestTOTPSettingsUseConfigWhenNotOverridden(t *testing.T) {
	originalSecretFilePath := totpSecretFilePath
	originalIssuer := totpIssuer
	originalWindow := totpWindow
	originalConfig := getConfig
	configPath := filepath.Join(t.TempDir(), "configured.secret")
	totpSecretFilePath = defaultTOTPSecretFilePath
	totpIssuer = defaultTOTPIssuer
	totpWindow = defaultTOTPWindow
	getConfig = func() *config.Config {
		return &config.Config{TwoFactorAuth: config.TwoFactorAuth{
			SecretFile: configPath,
			Issuer:     "ConfiguredIssuer",
			Window:     2,
		}}
	}
	t.Cleanup(func() {
		totpSecretFilePath = originalSecretFilePath
		totpIssuer = originalIssuer
		totpWindow = originalWindow
		getConfig = originalConfig
	})

	settings := totpSettings()
	if settings.secretFile != configPath || settings.issuer != "ConfiguredIssuer" || settings.window != 2 {
		t.Fatalf("expected config-backed TOTP settings, got %+v", settings)
	}
	if err := writeActiveTOTPSecret("ABCDEFG"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("expected active secret at config path: %v", err)
	}
}

func TestGenerateTOTPSecret(t *testing.T) {
	secret, err := generateTOTPSecret()
	if err != nil {
		t.Fatalf("generateTOTPSecret returned error: %v", err)
	}
	if secret == "" {
		t.Fatal("generateTOTPSecret returned empty secret")
	}
	if strings.Contains(secret, "=") {
		t.Fatalf("expected secret without padding, got %q", secret)
	}
	if secret != strings.ToUpper(secret) {
		t.Fatalf("expected uppercase secret, got %q", secret)
	}
}

func TestBuildTOTPURI(t *testing.T) {
	uri := buildTOTPURI("Nano KVM", "admin@example.com", "JBSWY3DPEHPK3PXP")
	want := "otpauth://totp/Nano+KVM%3Aadmin%40example.com?secret=JBSWY3DPEHPK3PXP&issuer=Nano+KVM"
	if uri != want {
		t.Fatalf("expected URI %q, got %q", want, uri)
	}
}

func TestVerifyTOTPCode(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"
	now := time.Unix(1234567890, 0)
	code, err := generateTOTPCode(secret, now)
	if err != nil {
		t.Fatalf("generateTOTPCode returned error: %v", err)
	}

	if !verifyTOTPCode(secret, code, now, 0) {
		t.Fatal("current TOTP code was rejected")
	}
	previousCode, err := generateTOTPCode(secret, now.Add(-30*time.Second))
	if err != nil {
		t.Fatalf("generateTOTPCode previous returned error: %v", err)
	}
	if !verifyTOTPCode(secret, previousCode, now, 1) {
		t.Fatal("previous-window TOTP code was rejected with window=1")
	}
	tooOldCode, err := generateTOTPCode(secret, now.Add(-60*time.Second))
	if err != nil {
		t.Fatalf("generateTOTPCode old returned error: %v", err)
	}
	if verifyTOTPCode(secret, tooOldCode, now, 1) {
		t.Fatal("out-of-window TOTP code was accepted")
	}
	if verifyTOTPCode(secret, "12345a", now, 1) {
		t.Fatal("non-numeric TOTP code was accepted")
	}
	if verifyTOTPCode(secret, "", now, 1) {
		t.Fatal("empty TOTP code was accepted")
	}
}

func TestGenerateTOTPCodeRejectsTimeBeforeUnixEpoch(t *testing.T) {
	if _, err := generateTOTPCode("JBSWY3DPEHPK3PXP", time.Unix(-1, 0)); err == nil {
		t.Fatal("generateTOTPCode accepted a negative moving-factor timestamp")
	}
}

func TestVerifySecondFactorTOTP(t *testing.T) {
	useTempRecoveryCodesFile(t)
	path := useTempTOTPSecretFile(t)
	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}
	now := time.Unix(1700000000, 0)
	totpNow = func() time.Time { return now }
	code, err := generateTOTPCode("JBSWY3DPEHPK3PXP", now)
	if err != nil {
		t.Fatalf("generateTOTPCode returned error: %v", err)
	}
	recoveryFile, err := buildRecoveryCodesFile([]string{"nkv-8H7K-2QPD-9M4X"})
	if err != nil {
		t.Fatalf("buildRecoveryCodesFile returned error: %v", err)
	}
	if err := saveRecoveryCodes(recoveryFile); err != nil {
		t.Fatalf("saveRecoveryCodes returned error: %v", err)
	}

	result := verifySecondFactor(code)
	if !result.OK || result.RecoveryCodeUsed || result.RecoveryCodeID != "" {
		t.Fatalf("expected TOTP second factor success, got %+v", result)
	}

	loaded, err := loadRecoveryCodes()
	if err != nil {
		t.Fatalf("loadRecoveryCodes returned error: %v", err)
	}
	if loaded.Codes[0].UsedAt != nil {
		t.Fatalf("TOTP success should not consume recovery codes, got %+v", loaded.Codes[0])
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected TOTP secret to remain in place: %v", err)
	}
}

func TestGenerateTOTPCodeRFC6238Vector(t *testing.T) {
	// RFC 6238 test vector for time 59 is 94287082 with 8 digits; this
	// implementation uses the standard 6-digit display code, so the value is 287082.
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	code, err := generateTOTPCode(secret, time.Unix(59, 0))
	if err != nil {
		t.Fatalf("generateTOTPCode returned error: %v", err)
	}
	if code != "287082" {
		t.Fatalf("expected RFC 6238-derived code 287082, got %q", code)
	}
}

func TestPendingTOTPSetupEnable(t *testing.T) {
	useTempTOTPSecretFile(t)
	now := time.Unix(1700000000, 0)
	totpNow = func() time.Time { return now }

	secret, uri, err := createPendingTOTPSetup("admin")
	if err != nil {
		t.Fatalf("createPendingTOTPSetup returned error: %v", err)
	}
	if secret == "" || !strings.Contains(uri, "otpauth://totp/") {
		t.Fatalf("expected secret and URI, got secret=%q uri=%q", secret, uri)
	}
	enabled, err := isTOTPEnabled()
	if err != nil {
		t.Fatalf("isTOTPEnabled returned error: %v", err)
	}
	if enabled {
		t.Fatal("pending setup should not enable active TOTP before confirmation")
	}

	code, err := generateTOTPCode(secret, now)
	if err != nil {
		t.Fatalf("generateTOTPCode returned error: %v", err)
	}
	if err := enablePendingTOTPSetup("admin", code); err != nil {
		t.Fatalf("enablePendingTOTPSetup returned error: %v", err)
	}
	enabled, err = isTOTPEnabled()
	if err != nil {
		t.Fatalf("isTOTPEnabled after enable returned error: %v", err)
	}
	if !enabled {
		t.Fatal("confirmed pending setup should enable active TOTP")
	}
	if err := enablePendingTOTPSetup("admin", code); err == nil {
		t.Fatal("pending setup should be removed after successful enable")
	}
}

func TestPendingTOTPSetupRejectsWrongCodeCancelAndExpiry(t *testing.T) {
	useTempTOTPSecretFile(t)
	now := time.Unix(1700000000, 0)
	totpNow = func() time.Time { return now }

	if _, _, err := createPendingTOTPSetup("admin"); err != nil {
		t.Fatalf("createPendingTOTPSetup returned error: %v", err)
	}
	if err := enablePendingTOTPSetup("admin", "000000"); err == nil {
		t.Fatal("wrong TOTP code enabled pending setup")
	}
	enabled, err := isTOTPEnabled()
	if err != nil {
		t.Fatalf("isTOTPEnabled returned error: %v", err)
	}
	if enabled {
		t.Fatal("wrong code should not enable active TOTP")
	}

	cancelPendingTOTPSetup("admin")
	if err := enablePendingTOTPSetup("admin", "000000"); err == nil {
		t.Fatal("cancelled pending setup was enabled")
	}

	secret, _, err := createPendingTOTPSetup("admin")
	if err != nil {
		t.Fatalf("createPendingTOTPSetup after cancel returned error: %v", err)
	}
	expiredAt := now.Add(pendingTOTPSetupTTL + time.Second)
	cleanupExpiredPendingTOTPSetups(expiredAt)
	code, err := generateTOTPCode(secret, expiredAt)
	if err != nil {
		t.Fatalf("generateTOTPCode returned error: %v", err)
	}
	totpNow = func() time.Time { return expiredAt }
	if err := enablePendingTOTPSetup("admin", code); err == nil {
		t.Fatal("expired pending setup was enabled")
	}
}

func TestVerifyCurrentTOTP(t *testing.T) {
	useTempTOTPSecretFile(t)
	now := time.Unix(1700000000, 0)
	totpNow = func() time.Time { return now }

	if verifyCurrentTOTP("000000") {
		t.Fatal("disabled active TOTP should not verify")
	}
	secret := "JBSWY3DPEHPK3PXP"
	if err := writeActiveTOTPSecret(secret); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}
	code, err := generateTOTPCode(secret, now)
	if err != nil {
		t.Fatalf("generateTOTPCode returned error: %v", err)
	}
	if !verifyCurrentTOTP(code) {
		t.Fatal("valid active TOTP code was rejected")
	}
	if verifyCurrentTOTP("000000") {
		t.Fatal("invalid active TOTP code was accepted")
	}
}
