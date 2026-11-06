package auth

import (
	"net/http"
	"os"
	"testing"

	"NanoKVM-Server/proto"
)

func TestNetBirdProtectionIsIndependentAndChallengesAreBoundToActions(t *testing.T) {
	setupAuthHandlerTest(t)
	actions := []string{SensitiveActionNetBirdLoginCancel, SensitiveActionNetBirdLogout, SensitiveActionNetBirdInstall, SensitiveActionNetBirdUninstall, SensitiveActionNetBirdLogin, SensitiveActionNetBirdStart, SensitiveActionNetBirdStop, SensitiveActionNetBirdRestart}
	for _, action := range actions {
		if NetBirdActionRequiresChallenge(action) || !validSensitiveAction(action) || !isProtectionSensitiveAction(action) {
			t.Fatalf("incorrect default for %s", action)
		}
	}
	enableTOTPForSensitiveActionTest(t)
	policy := defaultTwoFactorProtectionPolicy()
	policy.NetBird = true
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatal(err)
	}
	for _, action := range actions {
		if !NetBirdActionRequiresChallenge(action) {
			t.Fatal("NetBird policy bypass", action)
		}
	}
	if TailscaleActionRequiresChallenge(SensitiveActionTailscaleInstall) {
		t.Fatal("NetBird policy changed Tailscale")
	}
	ctx, rsp := performVerifySensitiveActionRequest(t, NewService(), proto.VerifySensitiveActionReq{Action: SensitiveActionNetBirdLogin, Code: currentTestTOTPCode(t)}, "admin")
	assertAuthCode(t, rsp, 0)
	challenge := decodeAuthData[proto.VerifySensitiveActionRsp](t, rsp).Challenge
	if consumeSensitiveActionChallenge(challenge, "admin", SensitiveActionNetBirdStart, sensitiveOperationSessionKey(ctx), ctx.ClientIP(), ctx.Request.UserAgent()) {
		t.Fatal("challenge accepted for wrong action")
	}
	if !consumeSensitiveActionChallenge(challenge, "admin", SensitiveActionNetBirdLogin, sensitiveOperationSessionKey(ctx), ctx.ClientIP(), ctx.Request.UserAgent()) {
		t.Fatal("valid challenge rejected")
	}
	if consumeSensitiveActionChallenge(challenge, "admin", SensitiveActionNetBirdLogin, sensitiveOperationSessionKey(ctx), ctx.ClientIP(), ctx.Request.UserAgent()) {
		t.Fatal("challenge reused")
	}
	if err := os.WriteFile(twoFactorProtectionFilePath, []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if !NetBirdActionRequiresChallenge(SensitiveActionNetBirdStart) {
		t.Fatal("unreadable policy did not fail closed")
	}
}

func TestOldProtectionRequestsPreserveNetBirdChoice(t *testing.T) {
	setupAuthHandlerTest(t)
	enableTOTPForSensitiveActionTest(t)
	policy := defaultTwoFactorProtectionPolicy()
	policy.NetBird = true
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatal(err)
	}
	ctx := authTestContext(t, http.MethodPost, "/api/auth/2fa/protection", nil)
	grantTwoFactorProtectionAuthorization("admin", sensitiveOperationSessionKey(ctx), ctx.ClientIP(), ctx.Request.UserAgent())
	req := twoFactorProtectionRequest(false, false, false, false, false, false, false, true, false, false)
	_, rsp := performAuthJSONRequest(t, NewService().SetTwoFactorProtection, http.MethodPost, "/api/auth/2fa/protection", req)
	assertAuthCode(t, rsp, 0)
	stored, err := loadTwoFactorProtectionPolicy()
	if err != nil || !stored.NetBird {
		t.Fatal("legacy request cleared NetBird policy", err)
	}
	req.NetBird = boolPointer(false)
	_, rsp = performAuthJSONRequest(t, NewService().SetTwoFactorProtection, http.MethodPost, "/api/auth/2fa/protection", req)
	assertAuthCode(t, rsp, 0)
	stored, err = loadTwoFactorProtectionPolicy()
	if err != nil || stored.NetBird {
		t.Fatal("explicit NetBird policy was ignored", err)
	}
}
