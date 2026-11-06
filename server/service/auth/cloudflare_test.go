package auth

import (
	"net/http"
	"os"
	"testing"

	"NanoKVM-Server/proto"
)

func TestCloudflareProtectionIsIndependentAndChallengesAreBoundToActions(t *testing.T) {
	setupAuthHandlerTest(t)
	actions := []string{SensitiveActionCloudflareInstall, SensitiveActionCloudflareUninstall, SensitiveActionCloudflareConfigure, SensitiveActionCloudflareStart, SensitiveActionCloudflareStop, SensitiveActionCloudflareRestart}
	for _, action := range actions {
		if CloudflareActionRequiresChallenge(action) || !validSensitiveAction(action) || !isProtectionSensitiveAction(action) {
			t.Fatalf("incorrect default for %s", action)
		}
	}
	enableTOTPForSensitiveActionTest(t)
	policy := defaultTwoFactorProtectionPolicy()
	policy.Cloudflare = true
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatal(err)
	}
	for _, action := range actions {
		if !CloudflareActionRequiresChallenge(action) {
			t.Fatal("Cloudflare policy bypass", action)
		}
	}
	if TailscaleActionRequiresChallenge(SensitiveActionTailscaleInstall) {
		t.Fatal("Cloudflare policy changed Tailscale")
	}
	ctx, rsp := performVerifySensitiveActionRequest(t, NewService(), proto.VerifySensitiveActionReq{Action: SensitiveActionCloudflareConfigure, Code: currentTestTOTPCode(t)}, "admin")
	assertAuthCode(t, rsp, 0)
	challenge := decodeAuthData[proto.VerifySensitiveActionRsp](t, rsp).Challenge
	if consumeSensitiveActionChallenge(challenge, "admin", SensitiveActionCloudflareStart, sensitiveOperationSessionKey(ctx), ctx.ClientIP(), ctx.Request.UserAgent()) {
		t.Fatal("challenge accepted for wrong action")
	}
	if !consumeSensitiveActionChallenge(challenge, "admin", SensitiveActionCloudflareConfigure, sensitiveOperationSessionKey(ctx), ctx.ClientIP(), ctx.Request.UserAgent()) {
		t.Fatal("valid challenge rejected")
	}
	if consumeSensitiveActionChallenge(challenge, "admin", SensitiveActionCloudflareConfigure, sensitiveOperationSessionKey(ctx), ctx.ClientIP(), ctx.Request.UserAgent()) {
		t.Fatal("challenge reused")
	}
	if err := os.WriteFile(twoFactorProtectionFilePath, []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if !CloudflareActionRequiresChallenge(SensitiveActionCloudflareStart) {
		t.Fatal("unreadable policy did not fail closed")
	}
}

func TestOldProtectionRequestsPreserveCloudflareChoice(t *testing.T) {
	setupAuthHandlerTest(t)
	enableTOTPForSensitiveActionTest(t)
	policy := defaultTwoFactorProtectionPolicy()
	policy.Cloudflare = true
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatal(err)
	}
	ctx := authTestContext(t, http.MethodPost, "/api/auth/2fa/protection", nil)
	grantTwoFactorProtectionAuthorization("admin", sensitiveOperationSessionKey(ctx), ctx.ClientIP(), ctx.Request.UserAgent())
	req := twoFactorProtectionRequest(false, false, false, false, false, false, false, true, false, false)
	_, rsp := performAuthJSONRequest(t, NewService().SetTwoFactorProtection, http.MethodPost, "/api/auth/2fa/protection", req)
	assertAuthCode(t, rsp, 0)
	stored, err := loadTwoFactorProtectionPolicy()
	if err != nil || !stored.Cloudflare {
		t.Fatal("legacy request cleared Cloudflare policy", err)
	}
	req.Cloudflare = boolPointer(false)
	_, rsp = performAuthJSONRequest(t, NewService().SetTwoFactorProtection, http.MethodPost, "/api/auth/2fa/protection", req)
	assertAuthCode(t, rsp, 0)
	stored, err = loadTwoFactorProtectionPolicy()
	if err != nil || stored.Cloudflare {
		t.Fatal("explicit Cloudflare policy was ignored", err)
	}
}
