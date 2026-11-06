package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func useTempRecoveryCodesFile(t *testing.T) string {
	t.Helper()

	originalPath := recoveryCodesFilePath
	originalCost := recoveryCodeBcryptCost
	originalNow := recoveryCodesNow

	path := filepath.Join(t.TempDir(), "recovery_codes.json")
	recoveryCodesFilePath = path
	recoveryCodeBcryptCost = bcrypt.MinCost
	recoveryCodesNow = func() time.Time { return time.Unix(1700000000, 0) }

	t.Cleanup(func() {
		recoveryCodesFilePath = originalPath
		recoveryCodeBcryptCost = originalCost
		recoveryCodesNow = originalNow
	})

	return path
}

func TestRecoveryCodeGenerateFormat(t *testing.T) {
	codes, err := generateRecoveryCodes(8)
	if err != nil {
		t.Fatalf("generateRecoveryCodes returned error: %v", err)
	}
	if len(codes) != 8 {
		t.Fatalf("expected 8 recovery codes, got %d", len(codes))
	}

	pattern := regexp.MustCompile(`^nkv-[2346789ABCDEFGHJKMNPQRTUVWXYZ]{4}-[2346789ABCDEFGHJKMNPQRTUVWXYZ]{4}-[2346789ABCDEFGHJKMNPQRTUVWXYZ]{4}$`)
	for _, code := range codes {
		if !pattern.MatchString(code) {
			t.Fatalf("generated recovery code has invalid format: %q", code)
		}
	}

	if _, err := generateRecoveryCodes(0); err == nil {
		t.Fatal("generateRecoveryCodes accepted zero count")
	}
}

func TestRecoveryCodeNormalizeFormat(t *testing.T) {
	cases := map[string]string{
		" nkv-8h7k-2qpd-9m4x\n": "nkv-8H7K-2QPD-9M4X",
		"NKV-8H7K-2QPD-9M4X":    "nkv-8H7K-2QPD-9M4X",
		"nkv8H7K2QPD9M4X":       "nkv-8H7K-2QPD-9M4X",
	}

	for input, want := range cases {
		got, ok := normalizeRecoveryCode(input)
		if !ok {
			t.Fatalf("normalizeRecoveryCode rejected %q", input)
		}
		if got != want {
			t.Fatalf("normalizeRecoveryCode(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestRecoveryCodeInvalidFormatRejected(t *testing.T) {
	inputs := []string{
		"",
		"8H7K-2QPD-9M4X",
		"nkv-8H7K-2QPD",
		"nkv-8H7K-2QPD-9M4",
		"nkv-8H7K-2QPD-9M4XL",
		"nkv-8H7K2QPD-9M4X",
		"nkv-8H7K-2QPD-9M4O",
		"nkv-8H7K-2QPD-9M4I",
		"nkv-8H7K-2QPD-9M4S",
		"nkv-8H7K-2QPD-9M4Z-extra",
	}

	for _, input := range inputs {
		if got, ok := normalizeRecoveryCode(input); ok {
			t.Fatalf("normalizeRecoveryCode(%q) accepted %q", input, got)
		}
	}
}

func TestRecoveryCodeStorageUses0600AndNoPlaintext(t *testing.T) {
	path := useTempRecoveryCodesFile(t)
	codes := []string{"nkv-8H7K-2QPD-9M4X", "nkv-7MRW-8G2T-3A6D"}
	file, err := buildRecoveryCodesFile(codes)
	if err != nil {
		t.Fatalf("buildRecoveryCodesFile returned error: %v", err)
	}
	if err := saveRecoveryCodes(file); err != nil {
		t.Fatalf("saveRecoveryCodes returned error: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat returned error: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("expected recovery code file mode 0600, got %04o", mode)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	text := string(data)
	for _, code := range codes {
		if strings.Contains(text, code) || strings.Contains(text, strings.ReplaceAll(code, "-", "")) {
			t.Fatalf("recovery code file contains plaintext code %q: %s", code, text)
		}
	}
	var stored recoveryCodesFile
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatalf("Unmarshal recovery code file returned error: %v", err)
	}
	if len(stored.Codes) != len(codes) {
		t.Fatalf("expected %d recovery code entries, got %d", len(codes), len(stored.Codes))
	}
	for i, entry := range stored.Codes {
		if err := bcrypt.CompareHashAndPassword([]byte(entry.Hash), []byte(codes[i])); err != nil {
			t.Fatalf("stored recovery code entry %d is not a bcrypt hash for the code: %v", i, err)
		}
	}
}

func TestRecoveryCodesStatusAndClear(t *testing.T) {
	useTempRecoveryCodesFile(t)

	enabled, remaining, err := recoveryCodesStatus()
	if err != nil {
		t.Fatalf("recoveryCodesStatus missing file returned error: %v", err)
	}
	if enabled || remaining != 0 {
		t.Fatalf("missing recovery code file should be disabled, got enabled=%v remaining=%d", enabled, remaining)
	}

	file, err := buildRecoveryCodesFile([]string{"nkv-8H7K-2QPD-9M4X"})
	if err != nil {
		t.Fatalf("buildRecoveryCodesFile returned error: %v", err)
	}
	if err := saveRecoveryCodes(file); err != nil {
		t.Fatalf("saveRecoveryCodes returned error: %v", err)
	}
	enabled, remaining, err = recoveryCodesStatus()
	if err != nil {
		t.Fatalf("recoveryCodesStatus returned error: %v", err)
	}
	if !enabled || remaining != 1 {
		t.Fatalf("expected enabled recovery codes with one remaining, got enabled=%v remaining=%d", enabled, remaining)
	}

	if err := clearRecoveryCodes(); err != nil {
		t.Fatalf("clearRecoveryCodes returned error: %v", err)
	}
	enabled, remaining, err = recoveryCodesStatus()
	if err != nil {
		t.Fatalf("recoveryCodesStatus after clear returned error: %v", err)
	}
	if enabled || remaining != 0 {
		t.Fatalf("cleared recovery codes should be disabled, got enabled=%v remaining=%d", enabled, remaining)
	}

	loaded, err := loadRecoveryCodes()
	if err != nil {
		t.Fatalf("loadRecoveryCodes after clear returned error: %v", err)
	}
	if loaded.Enabled || len(loaded.Codes) != 0 {
		t.Fatalf("clearRecoveryCodes should remove active codes, got %+v", loaded)
	}

	if err := os.Remove(recoveryCodesFilePath); err != nil {
		t.Fatalf("Remove returned error: %v", err)
	}
	if err := clearRecoveryCodes(); err != nil {
		t.Fatalf("clearRecoveryCodes should be idempotent for missing file, got %v", err)
	}
}

func TestVerifyAndConsumeRecoveryCode(t *testing.T) {
	useTempRecoveryCodesFile(t)
	code := "nkv-8H7K-2QPD-9M4X"
	file, err := buildRecoveryCodesFile([]string{code})
	if err != nil {
		t.Fatalf("buildRecoveryCodesFile returned error: %v", err)
	}
	wantID := file.Codes[0].ID
	if err := saveRecoveryCodes(file); err != nil {
		t.Fatalf("saveRecoveryCodes returned error: %v", err)
	}

	used, codeID, err := verifyAndConsumeRecoveryCode("nkv8h7k2qpd9m4x")
	if err != nil {
		t.Fatalf("verifyAndConsumeRecoveryCode returned error: %v", err)
	}
	if !used || codeID != wantID {
		t.Fatalf("expected first recovery code use to succeed with id %q, got used=%v id=%q", wantID, used, codeID)
	}

	loaded, err := loadRecoveryCodes()
	if err != nil {
		t.Fatalf("loadRecoveryCodes returned error: %v", err)
	}
	if loaded.Codes[0].UsedAt == nil || *loaded.Codes[0].UsedAt == "" {
		t.Fatalf("expected consumed code to have UsedAt, got %+v", loaded.Codes[0])
	}
	enabled, remaining, err := recoveryCodesStatus()
	if err != nil {
		t.Fatalf("recoveryCodesStatus returned error: %v", err)
	}
	if !enabled || remaining != 0 {
		t.Fatalf("expected enabled recovery codes with none remaining, got enabled=%v remaining=%d", enabled, remaining)
	}

	used, codeID, err = verifyAndConsumeRecoveryCode(code)
	if err != nil {
		t.Fatalf("verifyAndConsumeRecoveryCode second use returned error: %v", err)
	}
	if used || codeID != "" {
		t.Fatalf("expected second recovery code use to fail generically, got used=%v id=%q", used, codeID)
	}
}

func TestVerifyAndConsumeRecoveryCodeFailures(t *testing.T) {
	useTempRecoveryCodesFile(t)
	code := "nkv-8H7K-2QPD-9M4X"
	file, err := buildRecoveryCodesFile([]string{code})
	if err != nil {
		t.Fatalf("buildRecoveryCodesFile returned error: %v", err)
	}
	if err := saveRecoveryCodes(file); err != nil {
		t.Fatalf("saveRecoveryCodes returned error: %v", err)
	}

	for _, input := range []string{"not-a-code", "nkv-8H7K-2QPD-9M46", "nkv-7MRW-8G2T-3A6D"} {
		used, codeID, err := verifyAndConsumeRecoveryCode(input)
		if err != nil {
			t.Fatalf("verifyAndConsumeRecoveryCode(%q) returned error: %v", input, err)
		}
		if used || codeID != "" {
			t.Fatalf("verifyAndConsumeRecoveryCode(%q) should fail generically, got used=%v id=%q", input, used, codeID)
		}
	}

	file.Enabled = false
	if err := saveRecoveryCodes(file); err != nil {
		t.Fatalf("saveRecoveryCodes disabled returned error: %v", err)
	}
	used, codeID, err := verifyAndConsumeRecoveryCode(code)
	if err != nil {
		t.Fatalf("verifyAndConsumeRecoveryCode disabled returned error: %v", err)
	}
	if used || codeID != "" {
		t.Fatalf("disabled recovery codes should fail generically, got used=%v id=%q", used, codeID)
	}
}

func TestVerifyAndConsumeRecoveryCodeSaveFailureRejectsSuccess(t *testing.T) {
	path := useTempRecoveryCodesFile(t)
	code := "nkv-8H7K-2QPD-9M4X"
	file, err := buildRecoveryCodesFile([]string{code})
	if err != nil {
		t.Fatalf("buildRecoveryCodesFile returned error: %v", err)
	}
	if err := saveRecoveryCodes(file); err != nil {
		t.Fatalf("saveRecoveryCodes returned error: %v", err)
	}

	originalPath := recoveryCodesFilePath
	recoveryCodesFilePath = filepath.Join(path, "child")
	used, codeID, err := verifyAndConsumeRecoveryCode(code)
	recoveryCodesFilePath = originalPath
	if err == nil {
		t.Fatal("expected verifyAndConsumeRecoveryCode to return save/load error")
	}
	if used || codeID != "" {
		t.Fatalf("save failure must reject auth success, got used=%v id=%q err=%v", used, codeID, err)
	}

	used, codeID, err = verifyAndConsumeRecoveryCode(code)
	if err != nil {
		t.Fatalf("verifyAndConsumeRecoveryCode after restoring path returned error: %v", err)
	}
	if !used || codeID == "" {
		t.Fatalf("code should remain usable after failed persistence attempt, got used=%v id=%q", used, codeID)
	}
}

func TestVerifySecondFactorRecoveryCode(t *testing.T) {
	useTempTOTPSecretFile(t)
	useTempRecoveryCodesFile(t)
	code := "nkv-8H7K-2QPD-9M4X"
	file, err := buildRecoveryCodesFile([]string{code})
	if err != nil {
		t.Fatalf("buildRecoveryCodesFile returned error: %v", err)
	}
	wantID := file.Codes[0].ID
	if err := saveRecoveryCodes(file); err != nil {
		t.Fatalf("saveRecoveryCodes returned error: %v", err)
	}

	result := verifySecondFactor("nkv8h7k2qpd9m4x")
	if !result.OK || !result.RecoveryCodeUsed || result.RecoveryCodeID != wantID {
		t.Fatalf("expected recovery code second factor success, got %+v", result)
	}

	result = verifySecondFactor(code)
	if result.OK || result.RecoveryCodeUsed || result.RecoveryCodeID != "" {
		t.Fatalf("used recovery code should fail, got %+v", result)
	}
}

func TestVerifySecondFactorRecoveryCodeNotEnabled(t *testing.T) {
	useTempTOTPSecretFile(t)
	useTempRecoveryCodesFile(t)

	result := verifySecondFactor("nkv-8H7K-2QPD-9M4X")
	if result.OK || result.RecoveryCodeUsed || result.RecoveryCodeID != "" {
		t.Fatalf("recovery code should fail when not enabled, got %+v", result)
	}
}

func TestVerifySecondFactorWrongRecoveryCode(t *testing.T) {
	useTempTOTPSecretFile(t)
	useTempRecoveryCodesFile(t)
	file, err := buildRecoveryCodesFile([]string{"nkv-8H7K-2QPD-9M4X"})
	if err != nil {
		t.Fatalf("buildRecoveryCodesFile returned error: %v", err)
	}
	if err := saveRecoveryCodes(file); err != nil {
		t.Fatalf("saveRecoveryCodes returned error: %v", err)
	}

	result := verifySecondFactor("nkv-7MRW-8G2T-3A6D")
	if result.OK || result.RecoveryCodeUsed || result.RecoveryCodeID != "" {
		t.Fatalf("wrong recovery code should fail, got %+v", result)
	}
}
