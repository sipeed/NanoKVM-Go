package auth

import (
	"net/http"
	"os"
	"testing"
	"time"

	"NanoKVM-Server/proto"
)

func TestLoginTwoFactorDisabledCompatibility(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()

	recorder, rsp := performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Username: "admin", Password: "secret"})
	assertAuthCode(t, rsp, 0)
	data := decodeAuthData[proto.LoginRsp](t, rsp)
	if data.Token != "jwt-admin" || data.Requires2FA || data.Challenge != "" {
		t.Fatalf("unexpected login response: %+v", data)
	}
	assertHasAuthCookie(t, recorder, true)
}

func TestLoginTwoFactorPasswordStage(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}

	recorder, rsp := performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Username: "admin", Password: "secret"})
	assertAuthCode(t, rsp, 0)
	data := decodeAuthData[proto.LoginRsp](t, rsp)
	if !data.Requires2FA || data.Challenge == "" || data.Token != "" {
		t.Fatalf("expected 2FA challenge response, got %+v", data)
	}
	assertHasAuthCookie(t, recorder, false)

	_, rsp = performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Username: "admin", Password: "wrong"})
	assertAuthCode(t, rsp, -2)
	data = decodeAuthData[proto.LoginRsp](t, rsp)
	if data.Challenge != "" || data.Requires2FA {
		t.Fatalf("password failure should not reveal challenge, got %+v", data)
	}

	_, rsp = performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Username: "admin", Password: "wrong", Code: currentTestTOTPCode(t)})
	assertAuthCode(t, rsp, -1)
}

func TestLoginTwoFactorChallengeStage(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}

	_, rsp := performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Username: "admin", Password: "secret"})
	assertAuthCode(t, rsp, 0)
	challenge := decodeAuthData[proto.LoginRsp](t, rsp).Challenge
	code := currentTestTOTPCode(t)

	_, rsp = performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Challenge: challenge})
	assertAuthCode(t, rsp, -2)
	_, rsp = performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Code: code})
	assertAuthCode(t, rsp, -2)

	_, rsp = performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Username: "admin", Password: "secret"})
	assertAuthCode(t, rsp, 0)
	challenge = decodeAuthData[proto.LoginRsp](t, rsp).Challenge

	_, rsp = performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Challenge: challenge, Code: "000000"})
	assertAuthCode(t, rsp, -2)

	_, rsp = performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Username: "admin", Password: "secret"})
	assertAuthCode(t, rsp, 0)
	challenge = decodeAuthData[proto.LoginRsp](t, rsp).Challenge
	recorder, rsp := performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Challenge: challenge, Code: code})
	assertAuthCode(t, rsp, 0)
	data := decodeAuthData[proto.LoginRsp](t, rsp)
	if data.Token != "jwt-admin" || data.RecoveryCodeUsed {
		t.Fatalf("unexpected successful challenge login response: %+v", data)
	}
	assertHasAuthCookie(t, recorder, true)

	_, rsp = performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Challenge: challenge, Code: code})
	assertAuthCode(t, rsp, -2)
}

func TestDirectTwoFactorLoginStatus(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()

	_, rsp := performAuthJSONRequest(t, service.GetDirectTwoFactorLoginStatus, http.MethodGet, "/api/auth/login/2fa-direct", nil)
	assertAuthCode(t, rsp, 0)
	if decodeAuthData[proto.DirectTwoFactorLoginStatusRsp](t, rsp).Enabled {
		t.Fatal("direct login must be disabled while TOTP is disabled")
	}

	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}
	_, rsp = performAuthJSONRequest(t, service.GetDirectTwoFactorLoginStatus, http.MethodGet, "/api/auth/login/2fa-direct", nil)
	assertAuthCode(t, rsp, 0)
	if decodeAuthData[proto.DirectTwoFactorLoginStatusRsp](t, rsp).Enabled {
		t.Fatal("direct login must remain disabled until policy is enabled")
	}

	policy := defaultTwoFactorProtectionPolicy()
	policy.Login = true
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}
	_, rsp = performAuthJSONRequest(t, service.GetDirectTwoFactorLoginStatus, http.MethodGet, "/api/auth/login/2fa-direct", nil)
	assertAuthCode(t, rsp, 0)
	if !decodeAuthData[proto.DirectTwoFactorLoginStatusRsp](t, rsp).Enabled {
		t.Fatal("direct login status must be enabled with TOTP and policy enabled")
	}

	if err := os.WriteFile(twoFactorProtectionFilePath, []byte(`{"version":2}`), 0o600); err != nil {
		t.Fatalf("WriteFile malformed protection policy returned error: %v", err)
	}
	_, rsp = performAuthJSONRequest(t, service.GetDirectTwoFactorLoginStatus, http.MethodGet, "/api/auth/login/2fa-direct", nil)
	assertAuthCode(t, rsp, 0)
	if decodeAuthData[proto.DirectTwoFactorLoginStatusRsp](t, rsp).Enabled {
		t.Fatal("direct login status must fail closed for malformed policy")
	}
}

func TestDirectTwoFactorLogin(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}

	// A valid code remains rejected until the explicit convenience switch is enabled.
	_, rsp := performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Code: currentTestTOTPCode(t)})
	assertAuthCode(t, rsp, -2)

	policy := defaultTwoFactorProtectionPolicy()
	policy.Login = true
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}

	recorder, rsp := performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Code: currentTestTOTPCode(t)})
	assertAuthCode(t, rsp, 0)
	data := decodeAuthData[proto.LoginRsp](t, rsp)
	if data.Token != "jwt-admin" || data.RecoveryCodeUsed {
		t.Fatalf("unexpected direct TOTP login response: %+v", data)
	}
	assertHasAuthCookie(t, recorder, true)

	// Password-first login remains a valid, stronger fallback while direct login is on.
	_, rsp = performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Username: "admin", Password: "secret"})
	assertAuthCode(t, rsp, 0)
	if !decodeAuthData[proto.LoginRsp](t, rsp).Requires2FA {
		t.Fatal("password login must continue to require the existing challenge flow")
	}
}

func TestDirectTwoFactorLoginRecoveryCodeAndRateLimit(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}
	policy := defaultTwoFactorProtectionPolicy()
	policy.Login = true
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}
	recoveryFile, err := buildRecoveryCodesFile([]string{"nkv-8H7K-2QPD-9M4X"})
	if err != nil {
		t.Fatalf("buildRecoveryCodesFile returned error: %v", err)
	}
	if err := saveRecoveryCodes(recoveryFile); err != nil {
		t.Fatalf("saveRecoveryCodes returned error: %v", err)
	}

	recorder, rsp := performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Code: "nkv8h7k2qpd9m4x"})
	assertAuthCode(t, rsp, 0)
	data := decodeAuthData[proto.LoginRsp](t, rsp)
	if data.Token != "jwt-admin" || !data.RecoveryCodeUsed {
		t.Fatalf("expected direct recovery-code login success, got %+v", data)
	}
	assertHasAuthCookie(t, recorder, true)

	_, rsp = performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Code: "nkv-8H7K-2QPD-9M4X"})
	assertAuthCode(t, rsp, -2)

	for i := 1; i < loginUsernameIPRateLimitRule.MaxFailures; i++ {
		_, rsp = performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Code: "000000"})
		assertAuthCode(t, rsp, -2)
	}
	recorder, rsp = performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Code: currentTestTOTPCode(t)})
	assertAuthCode(t, rsp, -2)
	assertAuthRateLimitData(t, rsp)
	assertHasAuthCookie(t, recorder, false)
}

func TestLoginTwoFactorRecoveryCode(t *testing.T) {
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

	_, rsp := performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Username: "admin", Password: "secret"})
	assertAuthCode(t, rsp, 0)
	challenge := decodeAuthData[proto.LoginRsp](t, rsp).Challenge
	recorder, rsp := performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Challenge: challenge, Code: "nkv8h7k2qpd9m4x"})
	assertAuthCode(t, rsp, 0)
	data := decodeAuthData[proto.LoginRsp](t, rsp)
	if data.Token != "jwt-admin" || !data.RecoveryCodeUsed {
		t.Fatalf("expected recovery-code login success, got %+v", data)
	}
	assertHasAuthCookie(t, recorder, true)

	_, rsp = performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Username: "admin", Password: "secret"})
	assertAuthCode(t, rsp, 0)
	challenge = decodeAuthData[proto.LoginRsp](t, rsp).Challenge
	_, rsp = performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Challenge: challenge, Code: "nkv-8H7K-2QPD-9M4X"})
	assertAuthCode(t, rsp, -2)
}

func TestLoginFailureRateLimitBlocksCorrectPassword(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()

	for i := 0; i < loginUsernameIPRateLimitRule.MaxFailures; i++ {
		_, rsp := performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Username: "admin", Password: "wrong"})
		assertAuthCode(t, rsp, -2)
		if rsp.Msg != invalidAuthMessage {
			t.Fatalf("expected generic auth failure message, got %q", rsp.Msg)
		}
	}

	recorder, rsp := performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Username: "admin", Password: "secret"})
	assertAuthCode(t, rsp, -2)
	if rsp.Msg != invalidAuthMessage {
		t.Fatalf("expected generic auth failure message when rate limited, got %q", rsp.Msg)
	}
	assertAuthRateLimitData(t, rsp)
	assertHasAuthCookie(t, recorder, false)
}

func TestLoginSuccessClearsRateLimitFailures(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()

	for i := 0; i < loginUsernameIPRateLimitRule.MaxFailures-1; i++ {
		_, rsp := performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Username: "admin", Password: "wrong"})
		assertAuthCode(t, rsp, -2)
	}

	recorder, rsp := performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Username: "admin", Password: "secret"})
	assertAuthCode(t, rsp, 0)
	assertHasAuthCookie(t, recorder, true)

	for i := 0; i < loginUsernameIPRateLimitRule.MaxFailures-1; i++ {
		_, rsp = performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Username: "admin", Password: "wrong"})
		assertAuthCode(t, rsp, -2)
	}
	recorder, rsp = performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Username: "admin", Password: "secret"})
	assertAuthCode(t, rsp, 0)
	assertHasAuthCookie(t, recorder, true)
}

func TestLoginTwoFactorCodeFailuresAreRateLimited(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}

	for i := 0; i < loginUsernameIPRateLimitRule.MaxFailures; i++ {
		_, rsp := performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Username: "admin", Password: "secret"})
		assertAuthCode(t, rsp, 0)
		challenge := decodeAuthData[proto.LoginRsp](t, rsp).Challenge
		_, rsp = performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Challenge: challenge, Code: "000000"})
		assertAuthCode(t, rsp, -2)
	}

	recorder, rsp := performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Username: "admin", Password: "secret"})
	assertAuthCode(t, rsp, -2)
	if rsp.Msg != invalidAuthMessage {
		t.Fatalf("expected generic auth failure message when rate limited, got %q", rsp.Msg)
	}
	assertAuthRateLimitData(t, rsp)
	assertHasAuthCookie(t, recorder, false)
}

func TestLoginTwoFactorSameChallengeFailuresAreRateLimited(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}

	_, rsp := performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Username: "admin", Password: "secret"})
	assertAuthCode(t, rsp, 0)
	challenge := decodeAuthData[proto.LoginRsp](t, rsp).Challenge

	for i := 0; i < loginUsernameIPRateLimitRule.MaxFailures; i++ {
		_, rsp = performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Challenge: challenge, Code: "000000"})
		assertAuthCode(t, rsp, -2)
	}

	recorder, rsp := performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Challenge: challenge, Code: currentTestTOTPCode(t)})
	assertAuthCode(t, rsp, -2)
	if rsp.Msg != invalidAuthMessage {
		t.Fatalf("expected generic auth failure message when rate limited, got %q", rsp.Msg)
	}
	assertAuthRateLimitData(t, rsp)
	assertHasAuthCookie(t, recorder, false)
}

func TestLoginRecoveryCodeSameChallengeFailuresAreRateLimited(t *testing.T) {
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

	_, rsp := performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Username: "admin", Password: "secret"})
	assertAuthCode(t, rsp, 0)
	challenge := decodeAuthData[proto.LoginRsp](t, rsp).Challenge

	for i := 0; i < loginUsernameIPRateLimitRule.MaxFailures; i++ {
		_, rsp = performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Challenge: challenge, Code: "nkv-8H7K-2QPD-9M4Y"})
		assertAuthCode(t, rsp, -2)
	}

	recorder, rsp := performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Challenge: challenge, Code: "nkv-8H7K-2QPD-9M4X"})
	assertAuthCode(t, rsp, -2)
	if rsp.Msg != invalidAuthMessage {
		t.Fatalf("expected generic auth failure message when rate limited, got %q", rsp.Msg)
	}
	assertAuthRateLimitData(t, rsp)
	assertHasAuthCookie(t, recorder, false)
}

func TestLoginRecoveryCodeNotEnabledFails(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}

	_, rsp := performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Username: "admin", Password: "secret"})
	assertAuthCode(t, rsp, 0)
	challenge := decodeAuthData[proto.LoginRsp](t, rsp).Challenge
	_, rsp = performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Challenge: challenge, Code: "nkv-8H7K-2QPD-9M4X"})
	assertAuthCode(t, rsp, -2)
}

func TestLoginTwoFactorExpiredChallenge(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}

	_, rsp := performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Username: "admin", Password: "secret"})
	assertAuthCode(t, rsp, 0)
	challenge := decodeAuthData[proto.LoginRsp](t, rsp).Challenge
	expiredAt := totpNow().Add(loginChallengeTTL + time.Second)
	cleanupExpiredLoginChallenges(expiredAt)
	totpNow = func() time.Time { return expiredAt }
	code := currentTestTOTPCode(t)

	_, rsp = performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Challenge: challenge, Code: code})
	assertAuthCode(t, rsp, -2)
}

func TestDisableTwoFactorChangesLoginBehavior(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}

	_, rsp := performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Username: "admin", Password: "secret"})
	assertAuthCode(t, rsp, 0)
	if !decodeAuthData[proto.LoginRsp](t, rsp).Requires2FA {
		t.Fatal("enabled TOTP should require challenge")
	}

	_, rsp = performAuthJSONRequest(t, service.TwoFactorDisable, http.MethodPost, "/api/auth/2fa/disable", proto.TwoFactorDisableReq{Password: "wrong", Code: currentTestTOTPCode(t)})
	assertAuthCode(t, rsp, -2)
	_, rsp = performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Username: "admin", Password: "secret"})
	assertAuthCode(t, rsp, 0)
	if !decodeAuthData[proto.LoginRsp](t, rsp).Requires2FA {
		t.Fatal("failed disable should leave login in two-factor mode")
	}

	_, rsp = performAuthJSONRequest(t, service.TwoFactorDisable, http.MethodPost, "/api/auth/2fa/disable", proto.TwoFactorDisableReq{Password: "secret", Code: currentTestTOTPCode(t)})
	assertAuthCode(t, rsp, 0)
	recorder, rsp := performAuthJSONRequest(t, service.Login, http.MethodPost, "/api/auth/login", proto.LoginReq{Username: "admin", Password: "secret"})
	assertAuthCode(t, rsp, 0)
	data := decodeAuthData[proto.LoginRsp](t, rsp)
	if data.Token != "jwt-admin" || data.Requires2FA || data.Challenge != "" {
		t.Fatalf("disabled TOTP should restore direct login, got %+v", data)
	}
	assertHasAuthCookie(t, recorder, true)
}
