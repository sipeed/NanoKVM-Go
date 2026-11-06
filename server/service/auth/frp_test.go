package auth

import (
	"net/http"
	"os"
	"testing"

	"NanoKVM-Server/proto"
)

func TestFRPProtectionIsIndependentAndChallengesAreBoundToActions(t *testing.T) {
	setupAuthHandlerTest(t)
	actions := []string{SensitiveActionFRPInstall, SensitiveActionFRPUninstall, SensitiveActionFRPConfigure, SensitiveActionFRPStart, SensitiveActionFRPStop, SensitiveActionFRPRestart}
	for _, action := range actions {
		if FRPActionRequiresChallenge(action) || !validSensitiveAction(action) || !isProtectionSensitiveAction(action) {
			t.Fatalf("incorrect default for %s", action)
		}
	}
	enableTOTPForSensitiveActionTest(t)
	policy := defaultTwoFactorProtectionPolicy()
	policy.FRP = true
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatal(err)
	}
	for _, action := range actions {
		if !FRPActionRequiresChallenge(action) {
			t.Fatal("FRP policy bypass", action)
		}
	}
	if TailscaleActionRequiresChallenge(SensitiveActionTailscaleInstall) {
		t.Fatal("FRP policy changed Tailscale")
	}
	ctx, rsp := performVerifySensitiveActionRequest(t, NewService(), proto.VerifySensitiveActionReq{Action: SensitiveActionFRPConfigure, Code: currentTestTOTPCode(t)}, "admin")
	assertAuthCode(t, rsp, 0)
	challenge := decodeAuthData[proto.VerifySensitiveActionRsp](t, rsp).Challenge
	if consumeSensitiveActionChallenge(challenge, "admin", SensitiveActionFRPStart, sensitiveOperationSessionKey(ctx), ctx.ClientIP(), ctx.Request.UserAgent()) {
		t.Fatal("challenge accepted for wrong action")
	}
	if !consumeSensitiveActionChallenge(challenge, "admin", SensitiveActionFRPConfigure, sensitiveOperationSessionKey(ctx), ctx.ClientIP(), ctx.Request.UserAgent()) {
		t.Fatal("valid challenge rejected")
	}
	if consumeSensitiveActionChallenge(challenge, "admin", SensitiveActionFRPConfigure, sensitiveOperationSessionKey(ctx), ctx.ClientIP(), ctx.Request.UserAgent()) {
		t.Fatal("challenge reused")
	}
	if err := os.WriteFile(twoFactorProtectionFilePath, []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if !FRPActionRequiresChallenge(SensitiveActionFRPStart) {
		t.Fatal("unreadable policy did not fail closed")
	}
}

func TestOldProtectionRequestsPreserveFRPChoice(t *testing.T) {
	setupAuthHandlerTest(t)
	enableTOTPForSensitiveActionTest(t)
	policy := defaultTwoFactorProtectionPolicy()
	policy.FRP = true
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatal(err)
	}
	ctx := authTestContext(t, http.MethodPost, "/api/auth/2fa/protection", nil)
	grantTwoFactorProtectionAuthorization("admin", sensitiveOperationSessionKey(ctx), ctx.ClientIP(), ctx.Request.UserAgent())
	req := twoFactorProtectionRequest(false, false, false, false, false, false, false, true, false, false)
	_, rsp := performAuthJSONRequest(t, NewService().SetTwoFactorProtection, http.MethodPost, "/api/auth/2fa/protection", req)
	assertAuthCode(t, rsp, 0)
	stored, err := loadTwoFactorProtectionPolicy()
	if err != nil || !stored.FRP {
		t.Fatal("legacy request cleared FRP policy", err)
	}
	req.FRP = boolPointer(false)
	_, rsp = performAuthJSONRequest(t, NewService().SetTwoFactorProtection, http.MethodPost, "/api/auth/2fa/protection", req)
	assertAuthCode(t, rsp, 0)
	stored, err = loadTwoFactorProtectionPolicy()
	if err != nil || stored.FRP {
		t.Fatal("explicit FRP policy was ignored", err)
	}
}
