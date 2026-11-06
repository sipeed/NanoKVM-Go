package auth

import (
	"encoding/json"
	"net/http"
	"testing"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
)

func TestVerifySensitiveActionPasswordSuccessReturnsChallengeWhenTwoFactorDisabled(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()

	ctx, rsp := performVerifySensitiveActionRequest(t, service, proto.VerifySensitiveActionReq{
		Action:   sensitiveActionChangePassword,
		Password: "secret",
	}, "admin")

	assertAuthCode(t, rsp, 0)
	data := decodeAuthData[proto.VerifySensitiveActionRsp](t, rsp)
	if data.Challenge == "" || data.ExpiresIn != int(sensitiveActionChallengeTTL.Seconds()) {
		t.Fatalf("unexpected sensitive action response: %+v", data)
	}
	if !consumeSensitiveActionChallenge(data.Challenge, "admin", sensitiveActionChangePassword, sensitiveOperationSessionKey(ctx), ctx.ClientIP(), ctx.Request.UserAgent()) {
		t.Fatal("returned sensitive action challenge was not consumable with the same context")
	}
}

func TestVerifySensitiveActionRejectsInvalidAction(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	enableTOTPForSensitiveActionTest(t)

	_, rsp := performVerifySensitiveActionRequest(t, service, proto.VerifySensitiveActionReq{
		Action:   "disable_2fa",
		Password: "secret",
		Code:     currentTestTOTPCode(t),
	}, "admin")

	assertAuthCode(t, rsp, -1)
}

func TestVerifySensitiveActionRejectsWrongPassword(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	enableTOTPForSensitiveActionTest(t)

	_, rsp := performVerifySensitiveActionRequest(t, service, proto.VerifySensitiveActionReq{
		Action:   sensitiveActionChangePassword,
		Password: "wrong",
		Code:     currentTestTOTPCode(t),
	}, "admin")

	assertAuthCode(t, rsp, -2)
	if rsp.Msg != invalidAuthMessage {
		t.Fatalf("expected generic auth failure message, got %q", rsp.Msg)
	}
}

func TestVerifySensitiveActionRejectsWrongCode(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	enableTOTPForSensitiveActionTest(t)

	_, rsp := performVerifySensitiveActionRequest(t, service, proto.VerifySensitiveActionReq{
		Action:   sensitiveActionChangePassword,
		Password: "secret",
		Code:     "000000",
	}, "admin")

	assertAuthCode(t, rsp, -2)
	if rsp.Msg != invalidAuthMessage {
		t.Fatalf("expected generic auth failure message, got %q", rsp.Msg)
	}
}

func TestVerifySensitiveActionRejectsJWTUserMismatch(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	enableTOTPForSensitiveActionTest(t)

	_, rsp := performVerifySensitiveActionRequest(t, service, proto.VerifySensitiveActionReq{
		Action:   sensitiveActionChangePassword,
		Password: "secret",
		Code:     currentTestTOTPCode(t),
	}, "operator")

	assertAuthCode(t, rsp, -2)
	if rsp.Msg != invalidAuthMessage {
		t.Fatalf("expected generic auth failure message, got %q", rsp.Msg)
	}
}

func TestVerifySensitiveActionTOTPSuccessReturnsChallenge(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	enableTOTPForSensitiveActionTest(t)

	ctx, rsp := performVerifySensitiveActionRequest(t, service, proto.VerifySensitiveActionReq{
		Action:   sensitiveActionChangePassword,
		Password: "secret",
		Code:     currentTestTOTPCode(t),
	}, "admin")

	assertAuthCode(t, rsp, 0)
	data := decodeAuthData[proto.VerifySensitiveActionRsp](t, rsp)
	if data.Challenge == "" || data.ExpiresIn != int(sensitiveActionChallengeTTL.Seconds()) {
		t.Fatalf("unexpected sensitive action response: %+v", data)
	}
	if !consumeSensitiveActionChallenge(data.Challenge, "admin", sensitiveActionChangePassword, sensitiveOperationSessionKey(ctx), ctx.ClientIP(), ctx.Request.UserAgent()) {
		t.Fatal("returned sensitive action challenge was not consumable with the same context")
	}
}

func TestVerifySensitiveActionRecoveryCodeSuccessConsumesCode(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	enableTOTPForSensitiveActionTest(t)
	recoveryFile, err := buildRecoveryCodesFile([]string{"nkv-8H7K-2QPD-9M4X"})
	if err != nil {
		t.Fatalf("buildRecoveryCodesFile returned error: %v", err)
	}
	if err := saveRecoveryCodes(recoveryFile); err != nil {
		t.Fatalf("saveRecoveryCodes returned error: %v", err)
	}

	ctx, rsp := performVerifySensitiveActionRequest(t, service, proto.VerifySensitiveActionReq{
		Action:   sensitiveActionChangePassword,
		Password: "secret",
		Code:     "nkv8h7k2qpd9m4x",
	}, "admin")

	assertAuthCode(t, rsp, 0)
	data := decodeAuthData[proto.VerifySensitiveActionRsp](t, rsp)
	if data.Challenge == "" || data.ExpiresIn != int(sensitiveActionChallengeTTL.Seconds()) {
		t.Fatalf("unexpected sensitive action response: %+v", data)
	}
	if !consumeSensitiveActionChallenge(data.Challenge, "admin", sensitiveActionChangePassword, sensitiveOperationSessionKey(ctx), ctx.ClientIP(), ctx.Request.UserAgent()) {
		t.Fatal("returned sensitive action challenge was not consumable with the same context")
	}
	used, _, err := verifyAndConsumeRecoveryCode("nkv-8H7K-2QPD-9M4X")
	if err != nil {
		t.Fatalf("verifyAndConsumeRecoveryCode returned error: %v", err)
	}
	if used {
		t.Fatal("recovery code should have been consumed by sensitive action verification")
	}
}

func TestVerifyProtectionSensitiveActionIsCodeOnlyWhenTOTPEnabled(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	enableTOTPForSensitiveActionTest(t)

	protectionActions := []string{
		SensitiveActionRunScript,
		SensitiveActionUploadScript,
		SensitiveActionDeleteScript,
		SensitiveActionEnableSSH,
		SensitiveActionDisableSSH,
		SensitiveActionTailscaleInstall,
		SensitiveActionTailscaleLogin,
		SensitiveActionTailscaleUp,
		SensitiveActionTailscaleDown,
		SensitiveActionTailscaleStart,
		SensitiveActionTailscaleStop,
		SensitiveActionTailscaleRestart,
		SensitiveActionTailscaleLogout,
		SensitiveActionTailscaleUninstall,
		SensitiveActionTailscaleSetExitNode,
		SensitiveActionTailscaleSetSubnetRoute,
		SensitiveActionTargetPower,
		SensitiveActionNanoKVMRestart,
		SensitiveActionDeleteImage,
		SensitiveActionScreenTimelapseEnable,
		SensitiveActionScreenTimelapsePlayback,
	}
	for _, action := range protectionActions {
		t.Run(action, func(t *testing.T) {
			// Empty password + valid code succeeds.
			ctx, rsp := performVerifySensitiveActionRequest(t, service, proto.VerifySensitiveActionReq{
				Action: action,
				Code:   currentTestTOTPCode(t),
			}, "admin")
			assertAuthCode(t, rsp, 0)
			data := decodeAuthData[proto.VerifySensitiveActionRsp](t, rsp)
			if data.Challenge == "" {
				t.Fatal("expected challenge for protection action")
			}
			if !consumeSensitiveActionChallenge(data.Challenge, "admin", action, sensitiveOperationSessionKey(ctx), ctx.ClientIP(), ctx.Request.UserAgent()) {
				t.Fatal("returned protection challenge was not consumable")
			}

			// Wrong password is ignored when code is valid.
			_, rsp = performVerifySensitiveActionRequest(t, service, proto.VerifySensitiveActionReq{
				Action:   action,
				Password: "wrong",
				Code:     currentTestTOTPCode(t),
			}, "admin")
			assertAuthCode(t, rsp, 0)

			// Wrong code fails even with correct password.
			_, rsp = performVerifySensitiveActionRequest(t, service, proto.VerifySensitiveActionReq{
				Action:   action,
				Password: "secret",
				Code:     "000000",
			}, "admin")
			assertAuthCode(t, rsp, -2)
			if rsp.Msg != invalidAuthMessage {
				t.Fatalf("expected generic auth failure message, got %q", rsp.Msg)
			}
		})
	}
}

func TestVerifyProtectionSensitiveActionRequiresPasswordWhenTOTPDisabled(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()

	_, rsp := performVerifySensitiveActionRequest(t, service, proto.VerifySensitiveActionReq{
		Action: SensitiveActionRunScript,
	}, "admin")
	assertAuthCode(t, rsp, -2)

	ctx, rsp := performVerifySensitiveActionRequest(t, service, proto.VerifySensitiveActionReq{
		Action:   SensitiveActionRunScript,
		Password: "secret",
	}, "admin")
	assertAuthCode(t, rsp, 0)
	data := decodeAuthData[proto.VerifySensitiveActionRsp](t, rsp)
	if !consumeSensitiveActionChallenge(data.Challenge, "admin", SensitiveActionRunScript, sensitiveOperationSessionKey(ctx), ctx.ClientIP(), ctx.Request.UserAgent()) {
		t.Fatal("returned password-only run_script challenge was not consumable")
	}
}

func TestVerifyUpdateSignatureVerificationIsCodeOnlyWhenTOTPEnabled(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	enableTOTPForSensitiveActionTest(t)

	ctx, rsp := performVerifySensitiveActionRequest(t, service, proto.VerifySensitiveActionReq{
		Action: SensitiveActionUpdateSignatureVerification,
		Code:   currentTestTOTPCode(t),
	}, "admin")
	assertAuthCode(t, rsp, 0)
	data := decodeAuthData[proto.VerifySensitiveActionRsp](t, rsp)
	if !consumeSensitiveActionChallenge(
		data.Challenge,
		"admin",
		SensitiveActionUpdateSignatureVerification,
		sensitiveOperationSessionKey(ctx),
		ctx.ClientIP(),
		ctx.Request.UserAgent(),
	) {
		t.Fatal("returned update signature verification challenge was not consumable")
	}

	_, rsp = performVerifySensitiveActionRequest(t, service, proto.VerifySensitiveActionReq{
		Action:   SensitiveActionUpdateSignatureVerification,
		Password: "secret",
		Code:     "000000",
	}, "admin")
	assertAuthCode(t, rsp, -2)

	_, rsp = performVerifySensitiveActionRequest(t, service, proto.VerifySensitiveActionReq{
		Action:   SensitiveActionUpdateSignatureVerification,
		Password: "wrong",
		Code:     currentTestTOTPCode(t),
	}, "admin")
	assertAuthCode(t, rsp, 0)
}

func TestVerifyUpdateSignatureVerificationIsPasswordOnlyWhenTOTPDisabled(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()

	_, rsp := performVerifySensitiveActionRequest(t, service, proto.VerifySensitiveActionReq{
		Action: SensitiveActionUpdateSignatureVerification,
		Code:   "000000",
	}, "admin")
	assertAuthCode(t, rsp, -2)

	ctx, rsp := performVerifySensitiveActionRequest(t, service, proto.VerifySensitiveActionReq{
		Action:   SensitiveActionUpdateSignatureVerification,
		Password: "secret",
		Code:     "000000",
	}, "admin")
	assertAuthCode(t, rsp, 0)
	data := decodeAuthData[proto.VerifySensitiveActionRsp](t, rsp)
	if !consumeSensitiveActionChallenge(
		data.Challenge,
		"admin",
		SensitiveActionUpdateSignatureVerification,
		sensitiveOperationSessionKey(ctx),
		ctx.ClientIP(),
		ctx.Request.UserAgent(),
	) {
		t.Fatal("returned update signature verification challenge was not consumable")
	}
}

func TestVerifyChangePasswordStillRequiresPasswordWhenTOTPEnabled(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	enableTOTPForSensitiveActionTest(t)

	_, rsp := performVerifySensitiveActionRequest(t, service, proto.VerifySensitiveActionReq{
		Action: sensitiveActionChangePassword,
		Code:   currentTestTOTPCode(t),
	}, "admin")
	assertAuthCode(t, rsp, -2)

	_, rsp = performVerifySensitiveActionRequest(t, service, proto.VerifySensitiveActionReq{
		Action:   sensitiveActionChangePassword,
		Password: "secret",
		Code:     currentTestTOTPCode(t),
	}, "admin")
	assertAuthCode(t, rsp, 0)
}

func TestVerifyOpenWebTerminalSecondFactorFollowsPolicy(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()

	// TOTP off: historical password-only gate; the challenge stays action-bound.
	ctx, rsp := performVerifySensitiveActionRequest(t, service, proto.VerifySensitiveActionReq{
		Action:   SensitiveActionOpenWebTerminal,
		Password: "secret",
	}, "admin")
	assertAuthCode(t, rsp, 0)
	data := decodeAuthData[proto.VerifySensitiveActionRsp](t, rsp)
	if err := VerifySensitiveActionChallenge(ctx, data.Challenge, SensitiveActionChangePassword); err == nil {
		t.Fatal("open_web_terminal challenge must not authorize another action")
	}
	if err := VerifySensitiveActionChallenge(ctx, data.Challenge, SensitiveActionOpenWebTerminal); err != nil {
		t.Fatalf("open_web_terminal challenge was not consumable: %v", err)
	}

	enableTOTPForSensitiveActionTest(t)
	validCode := currentTestTOTPCode(t)

	// TOTP on + default policy (webTerminal=false): password alone suffices and a
	// submitted code is ignored rather than verified.
	_, rsp = performVerifySensitiveActionRequest(t, service, proto.VerifySensitiveActionReq{
		Action:   SensitiveActionOpenWebTerminal,
		Password: "secret",
	}, "admin")
	assertAuthCode(t, rsp, 0)
	_, rsp = performVerifySensitiveActionRequest(t, service, proto.VerifySensitiveActionReq{
		Action:   SensitiveActionOpenWebTerminal,
		Password: "secret",
		Code:     "000000", // invalid code must be ignored, not rejected
	}, "admin")
	assertAuthCode(t, rsp, 0)

	policyOffRejections := []struct {
		name     string
		password string
		code     string
	}{
		{name: "code only rejected", code: validCode},
		{name: "wrong password rejected", password: "wrong"},
	}
	for _, tt := range policyOffRejections {
		_, got := performVerifySensitiveActionRequest(t, service, proto.VerifySensitiveActionReq{
			Action:   SensitiveActionOpenWebTerminal,
			Password: tt.password,
			Code:     tt.code,
		}, "admin")
		assertAuthCode(t, got, -2)
		if got.Msg != invalidAuthMessage {
			t.Fatalf("%s: expected generic authentication failure, got %q", tt.name, got.Msg)
		}
	}

	// TOTP on + webTerminal=true: the second factor is required on top of the password.
	policy := defaultTwoFactorProtectionPolicy()
	policy.WebTerminal = true
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}

	// Succeed first so the rejection rows below never trip the failure limiter.
	_, rsp = performVerifySensitiveActionRequest(t, service, proto.VerifySensitiveActionReq{
		Action:   SensitiveActionOpenWebTerminal,
		Password: "secret",
		Code:     validCode,
	}, "admin")
	assertAuthCode(t, rsp, 0)

	policyOnRejections := []struct {
		name     string
		password string
		code     string
	}{
		{name: "code only rejected", code: validCode},
		{name: "password only rejected", password: "secret"},
		{name: "wrong password rejected", password: "wrong", code: validCode},
	}
	for _, tt := range policyOnRejections {
		t.Run(tt.name, func(t *testing.T) {
			_, got := performVerifySensitiveActionRequest(t, service, proto.VerifySensitiveActionReq{
				Action:   SensitiveActionOpenWebTerminal,
				Password: tt.password,
				Code:     tt.code,
			}, "admin")
			assertAuthCode(t, got, -2)
			if got.Msg != invalidAuthMessage {
				t.Fatalf("expected generic authentication failure, got %q", got.Msg)
			}
		})
	}
}

func TestVerifyOpenWebTerminalAcceptsPasswordAndRecoveryCode(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	enableTOTPForSensitiveActionTest(t)
	recoveryFile, err := buildRecoveryCodesFile([]string{"nkv-8H7K-2QPD-9M4X"})
	if err != nil {
		t.Fatalf("buildRecoveryCodesFile returned error: %v", err)
	}
	if err := saveRecoveryCodes(recoveryFile); err != nil {
		t.Fatalf("saveRecoveryCodes returned error: %v", err)
	}

	// The recovery code satisfies the second factor only when the webTerminal
	// protection policy requires it.
	policy := defaultTwoFactorProtectionPolicy()
	policy.WebTerminal = true
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}

	ctx, rsp := performVerifySensitiveActionRequest(t, service, proto.VerifySensitiveActionReq{
		Action:   SensitiveActionOpenWebTerminal,
		Password: "secret",
		Code:     "nkv8h7k2qpd9m4x",
	}, "admin")
	assertAuthCode(t, rsp, 0)
	data := decodeAuthData[proto.VerifySensitiveActionRsp](t, rsp)
	if err := VerifySensitiveActionChallenge(ctx, data.Challenge, SensitiveActionOpenWebTerminal); err != nil {
		t.Fatalf("open_web_terminal recovery-code challenge was not consumable: %v", err)
	}

	used, _, err := verifyAndConsumeRecoveryCode("nkv-8H7K-2QPD-9M4X")
	if err != nil {
		t.Fatalf("verifyAndConsumeRecoveryCode returned error: %v", err)
	}
	if used {
		t.Fatal("recovery code should have been consumed by Web terminal verification")
	}
}

func TestVerifyOpenWebTerminalIgnoresCodeWhenPolicyDisablesSecondFactor(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	enableTOTPForSensitiveActionTest(t)
	recoveryFile, err := buildRecoveryCodesFile([]string{"nkv-8H7K-2QPD-9M4X"})
	if err != nil {
		t.Fatalf("buildRecoveryCodesFile returned error: %v", err)
	}
	if err := saveRecoveryCodes(recoveryFile); err != nil {
		t.Fatalf("saveRecoveryCodes returned error: %v", err)
	}

	// Default policy webTerminal=false: the password alone opens the terminal and the
	// submitted recovery code is ignored, so it must remain usable afterwards.
	_, rsp := performVerifySensitiveActionRequest(t, service, proto.VerifySensitiveActionReq{
		Action:   SensitiveActionOpenWebTerminal,
		Password: "secret",
		Code:     "nkv8h7k2qpd9m4x",
	}, "admin")
	assertAuthCode(t, rsp, 0)

	used, _, err := verifyAndConsumeRecoveryCode("nkv-8H7K-2QPD-9M4X")
	if err != nil {
		t.Fatalf("verifyAndConsumeRecoveryCode returned error: %v", err)
	}
	if !used {
		t.Fatal("recovery code must remain usable when the second factor is not required")
	}
}

func TestVerifyOpenWebTerminalRejectsMissingJWTIdentity(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()

	_, rsp := performVerifySensitiveActionRequest(t, service, proto.VerifySensitiveActionReq{
		Action:   SensitiveActionOpenWebTerminal,
		Password: "secret",
	}, "")
	assertAuthCode(t, rsp, -2)
	if rsp.Msg != invalidAuthMessage {
		t.Fatalf("expected generic authentication failure, got %q", rsp.Msg)
	}
}

func TestVerifyOpenWebTerminalFailuresAreRateLimited(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()

	for i := 0; i < sensitiveOperationSessionRateLimitRule.MaxFailures; i++ {
		_, rsp := performVerifySensitiveActionRequest(t, service, proto.VerifySensitiveActionReq{
			Action:   SensitiveActionOpenWebTerminal,
			Password: "wrong",
		}, "admin")
		assertAuthCode(t, rsp, -2)
	}

	_, rsp := performVerifySensitiveActionRequest(t, service, proto.VerifySensitiveActionReq{
		Action:   SensitiveActionOpenWebTerminal,
		Password: "secret",
	}, "admin")
	assertAuthCode(t, rsp, -2)
	assertAuthRateLimitData(t, rsp)
}

func TestVerifySensitiveActionFailuresAreRateLimited(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	enableTOTPForSensitiveActionTest(t)
	validCode := currentTestTOTPCode(t)

	for i := 0; i < sensitiveOperationSessionRateLimitRule.MaxFailures; i++ {
		_, rsp := performVerifySensitiveActionRequest(t, service, proto.VerifySensitiveActionReq{
			Action:   sensitiveActionChangePassword,
			Password: "wrong",
			Code:     validCode,
		}, "admin")
		assertAuthCode(t, rsp, -2)
	}

	_, rsp := performVerifySensitiveActionRequest(t, service, proto.VerifySensitiveActionReq{
		Action:   sensitiveActionChangePassword,
		Password: "secret",
		Code:     validCode,
	}, "admin")

	assertAuthCode(t, rsp, -2)
	if rsp.Msg != invalidAuthMessage {
		t.Fatalf("expected generic auth failure message when rate limited, got %q", rsp.Msg)
	}
	assertAuthRateLimitData(t, rsp)
}

func enableTOTPForSensitiveActionTest(t *testing.T) {
	t.Helper()
	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}
}

func performVerifySensitiveActionRequest(t *testing.T, service *Service, req proto.VerifySensitiveActionReq, jwtUsername string) (*gin.Context, authTestResponse) {
	t.Helper()
	recorder, ctx := authTestContextWithRecorder(t, http.MethodPost, "/api/auth/2fa/verify-sensitive-action", req)
	ctx.Request.Header.Set("Authorization", "Bearer raw-jwt-token")
	if jwtUsername != "" {
		ctx.Set("username", jwtUsername)
	}
	service.VerifySensitiveAction(ctx)

	var rsp authTestResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("Unmarshal response returned error: %v body=%s", err, recorder.Body.String())
	}
	return ctx, rsp
}
