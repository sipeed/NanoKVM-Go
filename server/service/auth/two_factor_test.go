package auth

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"NanoKVM-Server/proto"
)

func TestTwoFactorStatusDisabled(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()

	_, rsp := performAuthJSONRequest(t, service.TwoFactorStatus, http.MethodGet, "/api/auth/2fa/status", nil)
	assertAuthCode(t, rsp, 0)
	data := decodeAuthData[proto.TwoFactorStatusRsp](t, rsp)
	if data.Enabled || data.RecoveryEnabled || data.RecoveryRemaining != 0 {
		t.Fatalf("unexpected 2FA status: %+v", data)
	}
}

func TestTwoFactorStatusRecoveryCodes(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}
	file, err := buildRecoveryCodesFile([]string{"nkv-8H7K-2QPD-9M4X", "nkv-7MRW-8G2T-3A6D"})
	if err != nil {
		t.Fatalf("buildRecoveryCodesFile returned error: %v", err)
	}
	usedAt := recoveryCodesNow().UTC().Format(time.RFC3339)
	file.Codes[0].UsedAt = &usedAt
	if err := saveRecoveryCodes(file); err != nil {
		t.Fatalf("saveRecoveryCodes returned error: %v", err)
	}

	_, rsp := performAuthJSONRequest(t, service.TwoFactorStatus, http.MethodGet, "/api/auth/2fa/status", nil)
	assertAuthCode(t, rsp, 0)
	data := decodeAuthData[proto.TwoFactorStatusRsp](t, rsp)
	if !data.Enabled || !data.RecoveryEnabled || data.RecoveryRemaining != 1 {
		t.Fatalf("unexpected 2FA status with recovery codes: %+v", data)
	}
}

func TestTwoFactorSetupRejectsWrongPassword(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()

	_, rsp := performAuthJSONRequest(t, service.TwoFactorSetup, http.MethodPost, "/api/auth/2fa/setup", proto.TwoFactorSetupReq{Password: "wrong"})
	assertAuthCode(t, rsp, -2)
	enabled, err := isTOTPEnabled()
	if err != nil {
		t.Fatalf("isTOTPEnabled returned error: %v", err)
	}
	if enabled {
		t.Fatal("wrong setup password should not enable TOTP")
	}
}

func TestTwoFactorSetupEnableAndCancel(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()

	_, rsp := performAuthJSONRequest(t, service.TwoFactorSetup, http.MethodPost, "/api/auth/2fa/setup", proto.TwoFactorSetupReq{Password: "secret"})
	assertAuthCode(t, rsp, 0)
	setup := decodeAuthData[proto.TwoFactorSetupRsp](t, rsp)
	if setup.Secret == "" || setup.URI == "" {
		t.Fatalf("expected setup secret and URI, got %+v", setup)
	}
	enabled, err := isTOTPEnabled()
	if err != nil {
		t.Fatalf("isTOTPEnabled returned error: %v", err)
	}
	if enabled {
		t.Fatal("setup should create pending secret only")
	}

	_, rsp = performAuthJSONRequest(t, service.TwoFactorEnable, http.MethodPost, "/api/auth/2fa/enable", proto.TwoFactorEnableReq{Code: "000000"})
	assertAuthCode(t, rsp, -2)
	enabled, err = isTOTPEnabled()
	if err != nil {
		t.Fatalf("isTOTPEnabled after wrong code returned error: %v", err)
	}
	if enabled {
		t.Fatal("wrong setup code should not enable TOTP")
	}

	code, err := generateTOTPCode(setup.Secret, totpNow())
	if err != nil {
		t.Fatalf("generateTOTPCode returned error: %v", err)
	}
	_, rsp = performAuthJSONRequest(t, service.TwoFactorEnable, http.MethodPost, "/api/auth/2fa/enable", proto.TwoFactorEnableReq{Code: code})
	assertAuthCode(t, rsp, 0)
	enabled, err = isTOTPEnabled()
	if err != nil {
		t.Fatalf("isTOTPEnabled after enable returned error: %v", err)
	}
	if !enabled {
		t.Fatal("correct setup code should enable TOTP")
	}

	_, rsp = performAuthJSONRequest(t, service.TwoFactorSetup, http.MethodPost, "/api/auth/2fa/setup", proto.TwoFactorSetupReq{Password: "secret"})
	assertAuthCode(t, rsp, -2)
}

func TestTwoFactorEnableInvalidatesExistingSessions(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	originalInvalidateAllAuthSessions := invalidateAllAuthSessions
	originalInvalidateAllAuthSessionsDueTo2FAEnabled := invalidateAllAuthSessionsDueTo2FAEnabled
	genericInvalidated := false
	twoFactorInvalidated := false
	invalidateAllAuthSessions = func() {
		genericInvalidated = true
	}
	invalidateAllAuthSessionsDueTo2FAEnabled = func() error {
		twoFactorInvalidated = true
		return nil
	}
	t.Cleanup(func() {
		invalidateAllAuthSessions = originalInvalidateAllAuthSessions
		invalidateAllAuthSessionsDueTo2FAEnabled = originalInvalidateAllAuthSessionsDueTo2FAEnabled
	})

	_, rsp := performAuthJSONRequest(t, service.TwoFactorSetup, http.MethodPost, "/api/auth/2fa/setup", proto.TwoFactorSetupReq{Password: "secret"})
	assertAuthCode(t, rsp, 0)
	setup := decodeAuthData[proto.TwoFactorSetupRsp](t, rsp)
	code, err := generateTOTPCode(setup.Secret, totpNow())
	if err != nil {
		t.Fatalf("generateTOTPCode returned error: %v", err)
	}

	recorder, rsp := performAuthJSONRequest(t, service.TwoFactorEnable, http.MethodPost, "/api/auth/2fa/enable", proto.TwoFactorEnableReq{Code: code})
	assertAuthCode(t, rsp, 0)
	if genericInvalidated {
		t.Fatal("enabling 2FA should not use generic session invalidation")
	}
	if !twoFactorInvalidated {
		t.Fatal("enabling 2FA should invalidate sessions with the 2FA-enabled reason")
	}
	assertClearsAuthCookie(t, recorder)
}

func TestTwoFactorEnableFailsBeforeActivationWhenSessionInvalidationFails(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	original := invalidateAllAuthSessionsDueTo2FAEnabled
	invalidateAllAuthSessionsDueTo2FAEnabled = func() error {
		return errors.New("entropy unavailable")
	}
	t.Cleanup(func() { invalidateAllAuthSessionsDueTo2FAEnabled = original })

	_, rsp := performAuthJSONRequest(t, service.TwoFactorSetup, http.MethodPost, "/api/auth/2fa/setup", proto.TwoFactorSetupReq{Password: "secret"})
	assertAuthCode(t, rsp, 0)
	setup := decodeAuthData[proto.TwoFactorSetupRsp](t, rsp)
	code, err := generateTOTPCode(setup.Secret, totpNow())
	if err != nil {
		t.Fatalf("generateTOTPCode: %v", err)
	}

	recorder, rsp := performAuthJSONRequest(t, service.TwoFactorEnable, http.MethodPost, "/api/auth/2fa/enable", proto.TwoFactorEnableReq{Code: code})
	assertAuthCode(t, rsp, -3)
	if enabled, err := isTOTPEnabled(); err != nil || enabled {
		t.Fatalf("TOTP activated despite session invalidation failure: enabled=%t err=%v", enabled, err)
	}
	if cookies := recorder.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("failed enable unexpectedly changed auth cookies: %#v", cookies)
	}
}

func TestTwoFactorSetupCancelRejectsPendingEnable(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()

	_, rsp := performAuthJSONRequest(t, service.TwoFactorSetup, http.MethodPost, "/api/auth/2fa/setup", proto.TwoFactorSetupReq{Password: "secret"})
	assertAuthCode(t, rsp, 0)
	setup := decodeAuthData[proto.TwoFactorSetupRsp](t, rsp)
	_, rsp = performAuthJSONRequest(t, service.TwoFactorSetupCancel, http.MethodPost, "/api/auth/2fa/setup/cancel", nil)
	assertAuthCode(t, rsp, 0)
	cancelledCode, err := generateTOTPCode(setup.Secret, totpNow())
	if err != nil {
		t.Fatalf("generateTOTPCode cancelled setup returned error: %v", err)
	}
	_, rsp = performAuthJSONRequest(t, service.TwoFactorEnable, http.MethodPost, "/api/auth/2fa/enable", proto.TwoFactorEnableReq{Code: cancelledCode})
	assertAuthCode(t, rsp, -2)
	assertTOTPEnabled(t, false)
}

func TestTwoFactorEnableRejectsReplacementWhenAlreadyEnabled(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}
	originalSecret, err := readActiveTOTPSecret()
	if err != nil {
		t.Fatalf("readActiveTOTPSecret returned error: %v", err)
	}

	pendingSecret, _, err := createPendingTOTPSetup("admin")
	if err != nil {
		t.Fatalf("createPendingTOTPSetup returned error: %v", err)
	}
	pendingCode, err := generateTOTPCode(pendingSecret, totpNow())
	if err != nil {
		t.Fatalf("generateTOTPCode pending returned error: %v", err)
	}

	_, rsp := performAuthJSONRequest(t, service.TwoFactorEnable, http.MethodPost, "/api/auth/2fa/enable", proto.TwoFactorEnableReq{Code: pendingCode})
	assertAuthCode(t, rsp, -2)

	secret, err := readActiveTOTPSecret()
	if err != nil {
		t.Fatalf("readActiveTOTPSecret after rejected replacement returned error: %v", err)
	}
	if secret != originalSecret {
		t.Fatalf("active TOTP secret changed after rejected replacement: got %q want %q", secret, originalSecret)
	}
	if err := enablePendingTOTPSetup("admin", pendingCode); err == nil {
		t.Fatal("rejected replacement should cancel the pending setup")
	}
}

func TestTwoFactorDisable(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()

	_, rsp := performAuthJSONRequest(t, service.TwoFactorDisable, http.MethodPost, "/api/auth/2fa/disable", proto.TwoFactorDisableReq{Password: "secret", Code: "000000"})
	assertAuthCode(t, rsp, 0)

	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}
	validCode := currentTestTOTPCode(t)

	_, rsp = performAuthJSONRequest(t, service.TwoFactorDisable, http.MethodPost, "/api/auth/2fa/disable", proto.TwoFactorDisableReq{Password: "wrong", Code: validCode})
	assertAuthCode(t, rsp, -2)
	assertTOTPEnabled(t, true)

	_, rsp = performAuthJSONRequest(t, service.TwoFactorDisable, http.MethodPost, "/api/auth/2fa/disable", proto.TwoFactorDisableReq{Password: "secret", Code: "000000"})
	assertAuthCode(t, rsp, -2)
	assertTOTPEnabled(t, true)

	pendingSecret, _, err := createPendingTOTPSetup("admin")
	if err != nil {
		t.Fatalf("createPendingTOTPSetup returned error: %v", err)
	}
	pendingCode, err := generateTOTPCode(pendingSecret, totpNow())
	if err != nil {
		t.Fatalf("generateTOTPCode pending returned error: %v", err)
	}
	_, rsp = performAuthJSONRequest(t, service.TwoFactorDisable, http.MethodPost, "/api/auth/2fa/disable", proto.TwoFactorDisableReq{Password: "secret", Code: validCode})
	assertAuthCode(t, rsp, 0)
	assertTOTPEnabled(t, false)

	if err := enablePendingTOTPSetup("admin", pendingCode); err == nil {
		t.Fatal("disable should cancel pending TOTP setup")
	}
}

func TestTwoFactorRecoveryEnable(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()

	_, rsp := performAuthJSONRequest(t, service.TwoFactorRecoveryEnable, http.MethodPost, "/api/auth/2fa/recovery/enable", proto.TwoFactorRecoveryReq{Password: "secret", Code: "000000"})
	assertAuthCode(t, rsp, -2)

	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}
	validCode := currentTestTOTPCode(t)

	_, rsp = performAuthJSONRequest(t, service.TwoFactorRecoveryEnable, http.MethodPost, "/api/auth/2fa/recovery/enable", proto.TwoFactorRecoveryReq{Password: "wrong", Code: validCode})
	assertAuthCode(t, rsp, -2)
	enabled, remaining, err := recoveryCodesStatus()
	if err != nil {
		t.Fatalf("recoveryCodesStatus returned error: %v", err)
	}
	if enabled || remaining != 0 {
		t.Fatalf("wrong password should not enable recovery codes, got enabled=%v remaining=%d", enabled, remaining)
	}

	_, rsp = performAuthJSONRequest(t, service.TwoFactorRecoveryEnable, http.MethodPost, "/api/auth/2fa/recovery/enable", proto.TwoFactorRecoveryReq{Password: "secret", Code: validCode})
	assertAuthCode(t, rsp, 0)
	recovery := decodeAuthData[proto.TwoFactorRecoveryRsp](t, rsp)
	if len(recovery.RecoveryCodes) != defaultRecoveryCodeCount {
		t.Fatalf("expected %d recovery codes, got %+v", defaultRecoveryCodeCount, recovery.RecoveryCodes)
	}
	for _, code := range recovery.RecoveryCodes {
		if _, ok := normalizeRecoveryCode(code); !ok {
			t.Fatalf("response returned invalid recovery code %q", code)
		}
	}

	_, rsp = performAuthJSONRequest(t, service.TwoFactorStatus, http.MethodGet, "/api/auth/2fa/status", nil)
	assertAuthCode(t, rsp, 0)
	status := decodeAuthData[proto.TwoFactorStatusRsp](t, rsp)
	if !status.Enabled || !status.RecoveryEnabled || status.RecoveryRemaining != defaultRecoveryCodeCount {
		t.Fatalf("unexpected recovery status after enable: %+v", status)
	}

	_, rsp = performAuthJSONRequest(t, service.TwoFactorRecoveryEnable, http.MethodPost, "/api/auth/2fa/recovery/enable", proto.TwoFactorRecoveryReq{Password: "secret", Code: validCode})
	assertAuthCode(t, rsp, -2)
	_, remaining, err = recoveryCodesStatus()
	if err != nil {
		t.Fatalf("recoveryCodesStatus after duplicate enable returned error: %v", err)
	}
	if remaining != defaultRecoveryCodeCount {
		t.Fatalf("duplicate enable should not replace recovery codes, remaining=%d", remaining)
	}
}

func TestTwoFactorRecoveryRegenerateInvalidatesOldCodes(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}
	validCode := currentTestTOTPCode(t)

	_, rsp := performAuthJSONRequest(t, service.TwoFactorRecoveryEnable, http.MethodPost, "/api/auth/2fa/recovery/enable", proto.TwoFactorRecoveryReq{Password: "secret", Code: validCode})
	assertAuthCode(t, rsp, 0)
	oldCodes := decodeAuthData[proto.TwoFactorRecoveryRsp](t, rsp).RecoveryCodes
	if len(oldCodes) != defaultRecoveryCodeCount {
		t.Fatalf("expected initial recovery codes, got %+v", oldCodes)
	}

	_, rsp = performAuthJSONRequest(t, service.TwoFactorRecoveryRegenerate, http.MethodPost, "/api/auth/2fa/recovery/regenerate", proto.TwoFactorRecoveryReq{Password: "secret", Code: oldCodes[0]})
	assertAuthCode(t, rsp, 0)
	newCodes := decodeAuthData[proto.TwoFactorRecoveryRsp](t, rsp).RecoveryCodes
	if len(newCodes) != defaultRecoveryCodeCount {
		t.Fatalf("expected regenerated recovery codes, got %+v", newCodes)
	}

	for _, oldCode := range oldCodes {
		used, _, err := verifyAndConsumeRecoveryCode(oldCode)
		if err != nil {
			t.Fatalf("verifyAndConsumeRecoveryCode old code returned error: %v", err)
		}
		if used {
			t.Fatalf("regenerate should invalidate old recovery code %q", oldCode)
		}
	}

	enabled, remaining, err := recoveryCodesStatus()
	if err != nil {
		t.Fatalf("recoveryCodesStatus after regenerate returned error: %v", err)
	}
	if !enabled || remaining != defaultRecoveryCodeCount {
		t.Fatalf("unexpected recovery status after regenerate: enabled=%v remaining=%d", enabled, remaining)
	}
}

func TestTwoFactorDisableClearsRecoveryCodes(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}
	recoveryFile, err := buildRecoveryCodesFile([]string{"nkv-8H7K-2QPD-9M4X"})
	if err != nil {
		t.Fatalf("buildRecoveryCodesFile returned error: %v", err)
	}
	if err := saveRecoveryCodes(recoveryFile); err != nil {
		t.Fatalf("saveRecoveryCodes returned error: %v", err)
	}
	validCode := currentTestTOTPCode(t)

	_, rsp := performAuthJSONRequest(t, service.TwoFactorDisable, http.MethodPost, "/api/auth/2fa/disable", proto.TwoFactorDisableReq{Password: "wrong", Code: validCode})
	assertAuthCode(t, rsp, -2)
	enabled, remaining, err := recoveryCodesStatus()
	if err != nil {
		t.Fatalf("recoveryCodesStatus after failed disable returned error: %v", err)
	}
	if !enabled || remaining != 1 {
		t.Fatalf("failed disable should leave recovery codes intact, got enabled=%v remaining=%d", enabled, remaining)
	}

	_, rsp = performAuthJSONRequest(t, service.TwoFactorDisable, http.MethodPost, "/api/auth/2fa/disable", proto.TwoFactorDisableReq{Password: "secret", Code: validCode})
	assertAuthCode(t, rsp, 0)
	enabled, remaining, err = recoveryCodesStatus()
	if err != nil {
		t.Fatalf("recoveryCodesStatus after disable returned error: %v", err)
	}
	if enabled || remaining != 0 {
		t.Fatalf("successful disable should clear recovery codes, got enabled=%v remaining=%d", enabled, remaining)
	}

	_, rsp = performAuthJSONRequest(t, service.TwoFactorStatus, http.MethodGet, "/api/auth/2fa/status", nil)
	assertAuthCode(t, rsp, 0)
	status := decodeAuthData[proto.TwoFactorStatusRsp](t, rsp)
	if status.Enabled || status.RecoveryEnabled || status.RecoveryRemaining != 0 {
		t.Fatalf("status after disable should report no 2FA or recovery codes, got %+v", status)
	}

	used, _, err := verifyAndConsumeRecoveryCode("nkv-8H7K-2QPD-9M4X")
	if err != nil {
		t.Fatalf("verifyAndConsumeRecoveryCode after disable returned error: %v", err)
	}
	if used {
		t.Fatal("old recovery code should be invalid after disabling 2FA")
	}

	recorder, rsp := performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Username: "admin", Password: "secret"})
	assertAuthCode(t, rsp, 0)
	login := decodeAuthData[proto.LoginRsp](t, rsp)
	if login.Token != "jwt-admin" || login.Requires2FA || login.RecoveryCodeUsed {
		t.Fatalf("login after disable should be direct password login, got %+v", login)
	}
	assertHasAuthCookie(t, recorder, true)
}

func TestTwoFactorDisableAllowsRecoveryCodeWhenAuthenticatorUnavailable(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}
	recoveryFile, err := buildRecoveryCodesFile([]string{"nkv-8H7K-2QPD-9M4X"})
	if err != nil {
		t.Fatalf("buildRecoveryCodesFile returned error: %v", err)
	}
	if err := saveRecoveryCodes(recoveryFile); err != nil {
		t.Fatalf("saveRecoveryCodes returned error: %v", err)
	}

	_, rsp := performAuthJSONRequest(t, service.TwoFactorDisable, http.MethodPost, "/api/auth/2fa/disable", proto.TwoFactorDisableReq{Password: "secret", Code: "nkv-8H7K-2QPD-9M4X"})
	assertAuthCode(t, rsp, 0)
	assertTOTPEnabled(t, false)
	enabled, remaining, err := recoveryCodesStatus()
	if err != nil {
		t.Fatalf("recoveryCodesStatus returned error: %v", err)
	}
	if enabled || remaining != 0 {
		t.Fatalf("disable with recovery code should clear recovery codes, got enabled=%v remaining=%d", enabled, remaining)
	}

	used, _, err := verifyAndConsumeRecoveryCode("nkv-8H7K-2QPD-9M4X")
	if err != nil {
		t.Fatalf("verifyAndConsumeRecoveryCode returned error: %v", err)
	}
	if used {
		t.Fatal("recovery code used to disable 2FA should not remain valid")
	}
}

func TestTwoFactorDisableRateLimitBlocksCorrectCode(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}
	validCode := currentTestTOTPCode(t)

	for i := 0; i < sensitiveOperationSessionRateLimitRule.MaxFailures; i++ {
		_, rsp := performAuthJSONRequest(t, service.TwoFactorDisable, http.MethodPost, "/api/auth/2fa/disable", proto.TwoFactorDisableReq{Password: "secret", Code: "000000"})
		assertAuthCode(t, rsp, -2)
		if rsp.Msg != invalidAuthMessage {
			t.Fatalf("expected generic auth failure message, got %q", rsp.Msg)
		}
	}

	_, rsp := performAuthJSONRequest(t, service.TwoFactorDisable, http.MethodPost, "/api/auth/2fa/disable", proto.TwoFactorDisableReq{Password: "secret", Code: validCode})
	assertAuthCode(t, rsp, -2)
	if rsp.Msg != invalidAuthMessage {
		t.Fatalf("expected generic auth failure message when rate limited, got %q", rsp.Msg)
	}
	assertAuthRateLimitData(t, rsp)
	assertTOTPEnabled(t, true)
}

func TestTwoFactorSensitiveOperationSuccessClearsRateLimit(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}
	validCode := currentTestTOTPCode(t)

	for i := 0; i < sensitiveOperationSessionRateLimitRule.MaxFailures-1; i++ {
		_, rsp := performAuthJSONRequest(t, service.TwoFactorRecoveryEnable, http.MethodPost, "/api/auth/2fa/recovery/enable", proto.TwoFactorRecoveryReq{Password: "wrong", Code: validCode})
		assertAuthCode(t, rsp, -2)
	}

	_, rsp := performAuthJSONRequest(t, service.TwoFactorRecoveryEnable, http.MethodPost, "/api/auth/2fa/recovery/enable", proto.TwoFactorRecoveryReq{Password: "secret", Code: validCode})
	assertAuthCode(t, rsp, 0)

	for i := 0; i < sensitiveOperationSessionRateLimitRule.MaxFailures-1; i++ {
		_, rsp = performAuthJSONRequest(t, service.TwoFactorDisable, http.MethodPost, "/api/auth/2fa/disable", proto.TwoFactorDisableReq{Password: "secret", Code: "000000"})
		assertAuthCode(t, rsp, -2)
	}
	_, rsp = performAuthJSONRequest(t, service.TwoFactorDisable, http.MethodPost, "/api/auth/2fa/disable", proto.TwoFactorDisableReq{Password: "secret", Code: validCode})
	assertAuthCode(t, rsp, 0)
	assertTOTPEnabled(t, false)
}

func TestTwoFactorRecoveryEnableFailuresAreRateLimited(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}
	validCode := currentTestTOTPCode(t)

	for i := 0; i < sensitiveOperationSessionRateLimitRule.MaxFailures; i++ {
		_, rsp := performAuthJSONRequest(t, service.TwoFactorRecoveryEnable, http.MethodPost, "/api/auth/2fa/recovery/enable", proto.TwoFactorRecoveryReq{Password: "wrong", Code: validCode})
		assertAuthCode(t, rsp, -2)
	}

	_, rsp := performAuthJSONRequest(t, service.TwoFactorRecoveryEnable, http.MethodPost, "/api/auth/2fa/recovery/enable", proto.TwoFactorRecoveryReq{Password: "secret", Code: validCode})
	assertAuthCode(t, rsp, -2)
	assertAuthRateLimitData(t, rsp)
	enabled, remaining, err := recoveryCodesStatus()
	if err != nil {
		t.Fatalf("recoveryCodesStatus returned error: %v", err)
	}
	if enabled || remaining != 0 {
		t.Fatalf("rate-limited recovery enable should not generate codes, got enabled=%v remaining=%d", enabled, remaining)
	}
}

func TestSensitiveOperationSessionKeyUsesTokenHash(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	recorder, rsp := performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Username: "admin", Password: "secret"})
	assertAuthCode(t, rsp, 0)
	assertHasAuthCookie(t, recorder, true)

	ctx := authTestContext(t, http.MethodPost, "/api/auth/2fa/setup", nil)
	ctx.Request.Header.Set("Authorization", "Bearer raw-jwt-token")
	key := sensitiveOperationSessionKey(ctx)
	if key == "" || strings.Contains(key, "raw-jwt-token") {
		t.Fatalf("session key should be a non-empty token hash, got %q", key)
	}
	if key != hashAuthRateLimitKey("token:raw-jwt-token") {
		t.Fatalf("unexpected token session key %q", key)
	}

	ctx = authTestContext(t, http.MethodPost, "/api/auth/2fa/setup", nil)
	key = sensitiveOperationSessionKey(ctx)
	if key != hashAuthRateLimitKey("client:192.0.2.1\nauth-test") {
		t.Fatalf("unexpected fallback session key %q", key)
	}
}

func assertTOTPEnabled(t *testing.T, want bool) {
	t.Helper()
	enabled, err := isTOTPEnabled()
	if err != nil {
		t.Fatalf("isTOTPEnabled returned error: %v", err)
	}
	if enabled != want {
		t.Fatalf("expected TOTP enabled %v, got %v", want, enabled)
	}
}
