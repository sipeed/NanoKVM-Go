package auth

import (
	"net/http"
	"os"
	"testing"

	"NanoKVM-Server/proto"
)

func TestZeroTierProtectionIsIndependentAndChallengesAreBoundToActions(t *testing.T) {
	setupAuthHandlerTest(t)
	actions := []string{SensitiveActionZeroTierLeave, SensitiveActionZeroTierInstall, SensitiveActionZeroTierUninstall, SensitiveActionZeroTierJoin, SensitiveActionZeroTierStart, SensitiveActionZeroTierStop, SensitiveActionZeroTierRestart}
	for _, action := range actions {
		if ZeroTierActionRequiresChallenge(action) || !validSensitiveAction(action) || !isProtectionSensitiveAction(action) {
			t.Fatalf("incorrect default for %s", action)
		}
	}
	enableTOTPForSensitiveActionTest(t)
	policy := defaultTwoFactorProtectionPolicy()
	policy.ZeroTier = true
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatal(err)
	}
	for _, action := range actions {
		if !ZeroTierActionRequiresChallenge(action) {
			t.Fatal("ZeroTier policy bypass", action)
		}
	}
	if TailscaleActionRequiresChallenge(SensitiveActionTailscaleInstall) {
		t.Fatal("ZeroTier policy changed Tailscale")
	}
	ctx, rsp := performVerifySensitiveActionRequest(t, NewService(), proto.VerifySensitiveActionReq{Action: SensitiveActionZeroTierJoin, Code: currentTestTOTPCode(t)}, "admin")
	assertAuthCode(t, rsp, 0)
	challenge := decodeAuthData[proto.VerifySensitiveActionRsp](t, rsp).Challenge
	if consumeSensitiveActionChallenge(challenge, "admin", SensitiveActionZeroTierStart, sensitiveOperationSessionKey(ctx), ctx.ClientIP(), ctx.Request.UserAgent()) {
		t.Fatal("challenge accepted for wrong action")
	}
	if !consumeSensitiveActionChallenge(challenge, "admin", SensitiveActionZeroTierJoin, sensitiveOperationSessionKey(ctx), ctx.ClientIP(), ctx.Request.UserAgent()) {
		t.Fatal("valid challenge rejected")
	}
	if consumeSensitiveActionChallenge(challenge, "admin", SensitiveActionZeroTierJoin, sensitiveOperationSessionKey(ctx), ctx.ClientIP(), ctx.Request.UserAgent()) {
		t.Fatal("challenge reused")
	}
	if err := os.WriteFile(twoFactorProtectionFilePath, []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if !ZeroTierActionRequiresChallenge(SensitiveActionZeroTierStart) {
		t.Fatal("unreadable policy did not fail closed")
	}
}

func TestOldProtectionRequestsPreserveZeroTierChoice(t *testing.T) {
	setupAuthHandlerTest(t)
	enableTOTPForSensitiveActionTest(t)
	policy := defaultTwoFactorProtectionPolicy()
	policy.ZeroTier = true
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatal(err)
	}
	ctx := authTestContext(t, http.MethodPost, "/api/auth/2fa/protection", nil)
	grantTwoFactorProtectionAuthorization("admin", sensitiveOperationSessionKey(ctx), ctx.ClientIP(), ctx.Request.UserAgent())
	req := twoFactorProtectionRequest(false, false, false, false, false, false, false, true, false, false)
	_, rsp := performAuthJSONRequest(t, NewService().SetTwoFactorProtection, http.MethodPost, "/api/auth/2fa/protection", req)
	assertAuthCode(t, rsp, 0)
	stored, err := loadTwoFactorProtectionPolicy()
	if err != nil || !stored.ZeroTier {
		t.Fatal("legacy request cleared ZeroTier policy", err)
	}
	req.ZeroTier = boolPointer(false)
	_, rsp = performAuthJSONRequest(t, NewService().SetTwoFactorProtection, http.MethodPost, "/api/auth/2fa/protection", req)
	assertAuthCode(t, rsp, 0)
	stored, err = loadTwoFactorProtectionPolicy()
	if err != nil || stored.ZeroTier {
		t.Fatal("explicit ZeroTier policy was ignored", err)
	}
}
