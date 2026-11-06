package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"NanoKVM-Server/proto"
)

func TestDefaultTwoFactorProtectionPolicyProtectsScriptsOnly(t *testing.T) {
	policy := defaultTwoFactorProtectionPolicy()
	if policy.Login || policy.WebTerminal || policy.SSH || policy.Tailscale || policy.MCP ||
		policy.TargetPower || policy.NanoKVMRestart || !policy.Scripts || policy.ScreenTimelapse ||
		policy.DeleteImage {
		t.Fatalf("unexpected scripts-only default protection policy: %+v", policy)
	}
}

func TestTwoFactorProtectionRequiresAuthorizationAndPersists(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}

	_, rsp := performAuthJSONRequest(t, service.GetTwoFactorProtection, http.MethodGet, "/api/auth/2fa/protection", nil)
	assertAuthCode(t, rsp, 0)
	status := decodeAuthData[proto.GetTwoFactorProtectionRsp](t, rsp)
	if status.AuthorizationExpiresIn != 0 || status.MCPAuthorizationExpiresIn != 0 || status.ScreenTimelapseAuthorizationExpiresIn != 0 ||
		status.Policy != defaultTwoFactorProtectionPolicy().response() {
		t.Fatalf("unexpected default protection status: %+v", status)
	}

	ctx := authTestContext(t, http.MethodGet, "/api/auth/2fa/protection", nil)
	grantMCPAuthorization("admin", sensitiveOperationSessionKey(ctx), ctx.ClientIP(), ctx.Request.UserAgent())
	_, rsp = performAuthJSONRequest(t, service.GetTwoFactorProtection, http.MethodGet, "/api/auth/2fa/protection", nil)
	assertAuthCode(t, rsp, 0)
	status = decodeAuthData[proto.GetTwoFactorProtectionRsp](t, rsp)
	if status.MCPAuthorizationExpiresIn <= 0 {
		t.Fatalf("expected active MCP authorization in protection status: %+v", status)
	}
	grantScreenTimelapsePlaybackAuthorization("admin", sensitiveOperationSessionKey(ctx), ctx.ClientIP(), ctx.Request.UserAgent())
	_, rsp = performAuthJSONRequest(t, service.GetTwoFactorProtection, http.MethodGet, "/api/auth/2fa/protection", nil)
	assertAuthCode(t, rsp, 0)
	status = decodeAuthData[proto.GetTwoFactorProtectionRsp](t, rsp)
	if status.ScreenTimelapseAuthorizationExpiresIn <= 0 {
		t.Fatalf("expected active Screen Timelapse authorization in protection status: %+v", status)
	}

	policy := twoFactorProtectionRequest(false, false, true, true, true, true, true, true, true, true)
	_, rsp = performAuthJSONRequest(t, service.SetTwoFactorProtection, http.MethodPost, "/api/auth/2fa/protection", policy)
	assertAuthCode(t, rsp, twoFactorProtectionAuthorizationNeeded)

	stored, err := loadTwoFactorProtectionPolicy()
	if err != nil {
		t.Fatalf("loadTwoFactorProtectionPolicy returned error: %v", err)
	}
	if stored != defaultTwoFactorProtectionPolicy() {
		t.Fatalf("unauthorized update changed policy: %+v", stored)
	}

	code := currentTestTOTPCode(t)
	_, rsp = performAuthJSONRequest(t, service.AuthorizeTwoFactorProtection, http.MethodPost, "/api/auth/2fa/protection/authorize", proto.AuthorizeTwoFactorProtectionReq{
		Code: code,
	})
	assertAuthCode(t, rsp, 0)
	authorization := decodeAuthData[proto.AuthorizeTwoFactorProtectionRsp](t, rsp)
	if authorization.ExpiresIn != int(twoFactorProtectionAuthorizationTTL.Seconds()) {
		t.Fatalf("unexpected protection authorization lifetime: %+v", authorization)
	}

	_, rsp = performAuthJSONRequest(t, service.SetTwoFactorProtection, http.MethodPost, "/api/auth/2fa/protection", policy)
	assertAuthCode(t, rsp, 0)
	updated := decodeAuthData[proto.GetTwoFactorProtectionRsp](t, rsp)
	if updated.Policy.Login || updated.Policy.WebTerminal || !updated.Policy.SSH || updated.AuthorizationExpiresIn <= 0 {
		t.Fatalf("unexpected updated protection policy: %+v", updated)
	}

	info, err := os.Stat(twoFactorProtectionFilePath)
	if err != nil {
		t.Fatalf("Stat protection policy returned error: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("expected protection policy mode 0600, got %04o", mode)
	}

	_, rsp = performAuthJSONRequest(t, service.GetTwoFactorProtection, http.MethodGet, "/api/auth/2fa/protection", nil)
	assertAuthCode(t, rsp, 0)
	reloaded := decodeAuthData[proto.GetTwoFactorProtectionRsp](t, rsp)
	if reloaded.Policy != updated.Policy {
		t.Fatalf("protection policy did not persist: got %+v want %+v", reloaded.Policy, updated.Policy)
	}
}

func TestTwoFactorProtectionAuthorizeIsCodeOnly(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}

	_, rsp := performAuthJSONRequest(t, service.AuthorizeTwoFactorProtection, http.MethodPost, "/api/auth/2fa/protection/authorize", proto.AuthorizeTwoFactorProtectionReq{
		Password: "wrong",
		Code:     "000000",
	})
	assertAuthCode(t, rsp, -2)
	if rsp.Msg != invalidAuthMessage {
		t.Fatalf("expected generic auth failure message, got %q", rsp.Msg)
	}

	// Wrong password must not matter when the second factor is valid.
	_, rsp = performAuthJSONRequest(t, service.AuthorizeTwoFactorProtection, http.MethodPost, "/api/auth/2fa/protection/authorize", proto.AuthorizeTwoFactorProtectionReq{
		Password: "wrong",
		Code:     currentTestTOTPCode(t),
	})
	assertAuthCode(t, rsp, 0)
}

func TestTwoFactorProtectionAuthorizationIsBoundAndExpires(t *testing.T) {
	setupAuthHandlerTest(t)
	now := time.Unix(1700000000, 0)
	totpNow = func() time.Time { return now }

	grantTwoFactorProtectionAuthorization("admin", "session-a", "192.0.2.1", "agent-a")
	if !hasTwoFactorProtectionAuthorization("admin", "session-a", "192.0.2.1", "agent-a") {
		t.Fatal("expected matching protection authorization")
	}
	if hasTwoFactorProtectionAuthorization("admin", "session-b", "192.0.2.1", "agent-a") {
		t.Fatal("authorization must not apply to a different session")
	}
	if hasTwoFactorProtectionAuthorization("admin", "session-a", "192.0.2.2", "agent-a") {
		t.Fatal("authorization must not apply to a different client IP")
	}

	now = now.Add(twoFactorProtectionAuthorizationTTL)
	if hasTwoFactorProtectionAuthorization("admin", "session-a", "192.0.2.1", "agent-a") {
		t.Fatal("expired protection authorization remained valid")
	}
}

func TestTwoFactorProtectionRejectsIncompleteStoredPolicy(t *testing.T) {
	for _, version := range []int{1, twoFactorProtectionFileVersion} {
		t.Run(fmt.Sprintf("version-%d", version), func(t *testing.T) {
			setupAuthHandlerTest(t)
			if err := os.WriteFile(twoFactorProtectionFilePath, []byte(fmt.Sprintf(`{"version":%d,"webTerminal":false}`, version)), 0o600); err != nil {
				t.Fatalf("WriteFile protection policy returned error: %v", err)
			}

			if _, err := loadTwoFactorProtectionPolicy(); err == nil {
				t.Fatal("incomplete stored protection policy must not be treated as defaults")
			}
		})
	}
}

func TestTwoFactorProtectionLoadsVersionOneWithDirectLoginDisabled(t *testing.T) {
	setupAuthHandlerTest(t)
	if err := os.WriteFile(twoFactorProtectionFilePath, []byte(`{"version":1,"webTerminal":false,"ssh":true,"tailscale":false,"mcp":true,"targetPower":false,"nanoKVMRestart":true,"scripts":false,"screenTimelapse":true}`), 0o600); err != nil {
		t.Fatalf("WriteFile version one protection policy returned error: %v", err)
	}

	policy, err := loadTwoFactorProtectionPolicy()
	if err != nil {
		t.Fatalf("loadTwoFactorProtectionPolicy returned error: %v", err)
	}
	if policy.Login || policy.WebTerminal || !policy.SSH || policy.Tailscale || !policy.MCP ||
		policy.TargetPower || !policy.NanoKVMRestart || policy.Scripts || !policy.ScreenTimelapse ||
		!policy.DeleteImage {
		t.Fatalf("unexpected version one migration policy: %+v", policy)
	}
	data, err := os.ReadFile(twoFactorProtectionFilePath)
	if err != nil {
		t.Fatalf("ReadFile version one protection policy returned error: %v", err)
	}
	var unmodified twoFactorProtectionFile
	if err := json.Unmarshal(data, &unmodified); err != nil {
		t.Fatalf("Unmarshal version one protection policy returned error: %v", err)
	}
	if unmodified.Version != 1 || unmodified.Login != nil {
		t.Fatalf("loading version one policy rewrote it: %+v", unmodified)
	}

	if err := saveTwoFactorProtectionPolicy(policy); !errors.Is(err, errTwoFactorProtectionDisabled) {
		t.Fatalf("save without TOTP returned %v, want disabled error", err)
	}
	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}

	data, err = os.ReadFile(twoFactorProtectionFilePath)
	if err != nil {
		t.Fatalf("ReadFile migrated protection policy returned error: %v", err)
	}
	var stored twoFactorProtectionFile
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatalf("Unmarshal migrated protection policy returned error: %v", err)
	}
	if stored.Version != twoFactorProtectionFileVersion || stored.Login == nil || *stored.Login {
		t.Fatalf("unexpected migrated protection file: %+v", stored)
	}
}

func TestWebTerminalRequiresSecondFactor(t *testing.T) {
	setupAuthHandlerTest(t)

	// TOTP off: historical password-only gate.
	if WebTerminalRequiresSecondFactor() {
		t.Fatal("web terminal must not require a second factor when 2FA is disabled")
	}
	if webTerminalRequiresSecondFactor(false) {
		t.Fatal("inner decision must be false when TOTP is disabled")
	}

	// TOTP off ignores even a stale stored webTerminal=true.
	if err := os.WriteFile(twoFactorProtectionFilePath, []byte(`{"version":2,"login":false,"webTerminal":true,"ssh":true,"tailscale":true,"mcp":true,"targetPower":true,"nanoKVMRestart":true,"scripts":true,"screenTimelapse":true,"deleteImage":true}`), 0o600); err != nil {
		t.Fatalf("WriteFile stale protection policy returned error: %v", err)
	}
	if WebTerminalRequiresSecondFactor() {
		t.Fatal("TOTP off must ignore a stale stored webTerminal=true")
	}
	if webTerminalRequiresSecondFactor(false) {
		t.Fatal("inner decision must ignore stored policy when TOTP is disabled")
	}

	if err := os.Remove(twoFactorProtectionFilePath); err != nil {
		t.Fatalf("Remove protection policy returned error: %v", err)
	}
	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}
	// Missing policy file → scripts-only default (webTerminal=false).
	if WebTerminalRequiresSecondFactor() {
		t.Fatal("default policy must not require a second factor for the web terminal")
	}

	policy := defaultTwoFactorProtectionPolicy()
	policy.WebTerminal = false
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}
	if WebTerminalRequiresSecondFactor() {
		t.Fatal("webTerminal=false must not require a second factor")
	}

	policy.WebTerminal = true
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}
	if !WebTerminalRequiresSecondFactor() {
		t.Fatal("webTerminal=true must require a second factor")
	}

	// Malformed stored policy under TOTP fails closed.
	if err := os.WriteFile(twoFactorProtectionFilePath, []byte("not-json"), 0o600); err != nil {
		t.Fatalf("WriteFile malformed protection policy returned error: %v", err)
	}
	if !WebTerminalRequiresSecondFactor() {
		t.Fatal("unreadable protection policy under enabled 2FA must fail closed")
	}
	if !webTerminalRequiresSecondFactor(true) {
		t.Fatal("inner decision must fail closed when the policy cannot be loaded")
	}

	// TOTP-status read error fails closed even though no policy issue exists.
	originalSecretFilePath := totpSecretFilePath
	if err := os.Remove(totpSecretFilePath); err != nil {
		t.Fatalf("Remove TOTP secret returned error: %v", err)
	}
	if err := os.Mkdir(totpSecretFilePath, 0o700); err != nil {
		t.Fatalf("Mkdir TOTP secret path returned error: %v", err)
	}
	if !WebTerminalRequiresSecondFactor() {
		t.Fatal("TOTP status read error must fail closed and require the second factor")
	}
	totpSecretFilePath = originalSecretFilePath
}

func TestScriptProtectionChallengeGates(t *testing.T) {
	setupAuthHandlerTest(t)

	// TOTP off: upload/delete ungated, execution still requires challenge.
	if ScriptUploadRequiresChallenge() {
		t.Fatal("script upload must not require challenge when 2FA is disabled")
	}
	if ScriptDeleteRequiresChallenge() {
		t.Fatal("script delete must not require challenge when 2FA is disabled")
	}
	if !ScriptExecutionRequiresChallenge() {
		t.Fatal("script execution must require challenge when 2FA is disabled")
	}

	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}
	// No policy file → default scripts=true.
	if !ScriptUploadRequiresChallenge() || !ScriptExecutionRequiresChallenge() || !ScriptDeleteRequiresChallenge() {
		t.Fatal("default policy must require challenges when 2FA is enabled")
	}

	policy := defaultTwoFactorProtectionPolicy()
	policy.Scripts = false
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}
	if ScriptUploadRequiresChallenge() || ScriptExecutionRequiresChallenge() || ScriptDeleteRequiresChallenge() {
		t.Fatal("script gates should honor scripts=false")
	}

	policy.Scripts = true
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}
	if !ScriptUploadRequiresChallenge() || !ScriptExecutionRequiresChallenge() || !ScriptDeleteRequiresChallenge() {
		t.Fatal("script gates should honor scripts=true")
	}
}

func TestSSHActionRequiresChallenge(t *testing.T) {
	setupAuthHandlerTest(t)

	// TOTP off → same as historical enable/disable (no challenge).
	if SSHActionRequiresChallenge() || SSHEnableRequiresChallenge() || SSHDisableRequiresChallenge() {
		t.Fatal("SSH enable/disable must not require challenge when 2FA is disabled")
	}

	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}
	if SSHActionRequiresChallenge() || SSHEnableRequiresChallenge() || SSHDisableRequiresChallenge() {
		t.Fatal("scripts-only default must leave SSH enable/disable ungated")
	}

	policy := defaultTwoFactorProtectionPolicy()
	policy.SSH = true
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}
	if !SSHActionRequiresChallenge() || !SSHEnableRequiresChallenge() || !SSHDisableRequiresChallenge() {
		t.Fatal("ssh=true must require challenge for SSH enable/disable")
	}

	policy.SSH = false
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}
	if SSHActionRequiresChallenge() || SSHDisableRequiresChallenge() {
		t.Fatal("SSH enable/disable with ssh=false must match 2FA-off (no challenge)")
	}

	policy.SSH = true
	policy.Tailscale = false // independence: tailscale off must not affect ssh
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}
	if !SSHActionRequiresChallenge() || !SSHDisableRequiresChallenge() {
		t.Fatal("SSH enable/disable must honor ssh=true even when tailscale=false")
	}
}

func TestTargetPowerActionRequiresChallenge(t *testing.T) {
	setupAuthHandlerTest(t)

	if TargetPowerActionRequiresChallenge() {
		t.Fatal("target power must not require challenge when 2FA is disabled")
	}

	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}
	if TargetPowerActionRequiresChallenge() {
		t.Fatal("scripts-only default must leave target power ungated")
	}

	policy := defaultTwoFactorProtectionPolicy()
	policy.TargetPower = true
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}
	if !TargetPowerActionRequiresChallenge() {
		t.Fatal("targetPower=true must require challenge")
	}

	policy.TargetPower = false
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}
	if TargetPowerActionRequiresChallenge() {
		t.Fatal("target power with targetPower=false must not require challenge")
	}

	policy.TargetPower = true
	policy.NanoKVMRestart = false
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}
	if !TargetPowerActionRequiresChallenge() {
		t.Fatal("target power must honor targetPower=true even when nanoKVMRestart=false")
	}
}

func TestNanoKVMRestartActionRequiresChallenge(t *testing.T) {
	setupAuthHandlerTest(t)

	if NanoKVMRestartActionRequiresChallenge() {
		t.Fatal("NanoKVM restart must not require challenge when 2FA is disabled")
	}

	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}
	if NanoKVMRestartActionRequiresChallenge() {
		t.Fatal("scripts-only default must leave NanoKVM restart ungated")
	}

	policy := defaultTwoFactorProtectionPolicy()
	policy.NanoKVMRestart = true
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}
	if !NanoKVMRestartActionRequiresChallenge() {
		t.Fatal("nanoKVMRestart=true must require challenge")
	}

	policy.NanoKVMRestart = false
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}
	if NanoKVMRestartActionRequiresChallenge() {
		t.Fatal("NanoKVM restart with nanoKVMRestart=false must not require challenge")
	}

	policy.NanoKVMRestart = true
	policy.TargetPower = false
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}
	if !NanoKVMRestartActionRequiresChallenge() {
		t.Fatal("NanoKVM restart must honor nanoKVMRestart=true even when targetPower=false")
	}
}

func TestDeleteImageActionRequiresChallenge(t *testing.T) {
	setupAuthHandlerTest(t)

	if DeleteImageActionRequiresChallenge() {
		t.Fatal("delete image must not require challenge when 2FA is disabled")
	}

	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}
	if DeleteImageActionRequiresChallenge() {
		t.Fatal("scripts-only default must leave image deletion ungated")
	}

	policy := defaultTwoFactorProtectionPolicy()
	policy.DeleteImage = true
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}
	if !DeleteImageActionRequiresChallenge() {
		t.Fatal("deleteImage=true must require challenge")
	}

	policy.DeleteImage = false
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}
	if DeleteImageActionRequiresChallenge() {
		t.Fatal("delete image with deleteImage=false must not require challenge")
	}

	policy.DeleteImage = true
	policy.TargetPower = false
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}
	if !DeleteImageActionRequiresChallenge() {
		t.Fatal("delete image must honor deleteImage=true even when targetPower=false")
	}
}

func TestScreenTimelapseEnableRequiresChallenge(t *testing.T) {
	setupAuthHandlerTest(t)

	if ScreenTimelapseEnableRequiresChallenge() {
		t.Fatal("screen timelapse enable must not require challenge when 2FA is disabled")
	}
	if ScreenTimelapsePlaybackRequiresChallenge() {
		t.Fatal("screen timelapse playback must not require challenge when 2FA is disabled")
	}

	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}
	if ScreenTimelapseEnableRequiresChallenge() {
		t.Fatal("scripts-only default must leave screen timelapse enable ungated")
	}
	if ScreenTimelapsePlaybackRequiresChallenge() {
		t.Fatal("scripts-only default must leave screen timelapse playback ungated")
	}

	policy := defaultTwoFactorProtectionPolicy()
	policy.ScreenTimelapse = true
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}
	if !ScreenTimelapseEnableRequiresChallenge() || !ScreenTimelapsePlaybackRequiresChallenge() {
		t.Fatal("screenTimelapse=true must require challenges")
	}

	policy.ScreenTimelapse = false
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}
	if ScreenTimelapseEnableRequiresChallenge() {
		t.Fatal("screen timelapse enable with screenTimelapse=false must not require challenge")
	}
	if ScreenTimelapsePlaybackRequiresChallenge() {
		t.Fatal("screen timelapse playback with screenTimelapse=false must not require challenge")
	}

	policy.ScreenTimelapse = true
	policy.DeleteImage = false
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}
	if !ScreenTimelapseEnableRequiresChallenge() {
		t.Fatal("screen timelapse enable must honor screenTimelapse=true even when deleteImage=false")
	}
	if !ScreenTimelapsePlaybackRequiresChallenge() {
		t.Fatal("screen timelapse playback must honor screenTimelapse=true even when deleteImage=false")
	}

	twoFactorProtectionFilePath = t.TempDir()
	if !ScreenTimelapseEnableRequiresChallenge() {
		t.Fatal("screen timelapse enable must require challenge when protection policy cannot be read")
	}
	if !ScreenTimelapsePlaybackRequiresChallenge() {
		t.Fatal("screen timelapse playback must require challenge when protection policy cannot be read")
	}
}

func TestTwoFactorProtectionLoadsVersionTwoMissingDeleteImageDefaultsEnabled(t *testing.T) {
	setupAuthHandlerTest(t)
	if err := os.WriteFile(twoFactorProtectionFilePath, []byte(`{"version":2,"login":false,"webTerminal":false,"ssh":true,"tailscale":false,"mcp":true,"targetPower":false,"nanoKVMRestart":true,"scripts":false,"screenTimelapse":true}`), 0o600); err != nil {
		t.Fatalf("WriteFile version two protection policy returned error: %v", err)
	}

	policy, err := loadTwoFactorProtectionPolicy()
	if err != nil {
		t.Fatalf("loadTwoFactorProtectionPolicy returned error: %v", err)
	}
	if policy.Login || policy.WebTerminal || !policy.SSH || policy.Tailscale || !policy.MCP ||
		policy.TargetPower || !policy.NanoKVMRestart || policy.Scripts || !policy.ScreenTimelapse ||
		!policy.DeleteImage {
		t.Fatalf("unexpected version two missing-deleteImage policy: %+v", policy)
	}
}

func TestTailscaleActionRequiresChallenge(t *testing.T) {
	setupAuthHandlerTest(t)

	listed := []string{
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
	}

	for _, action := range listed {
		if TailscaleActionRequiresChallenge(action) {
			t.Fatalf("%s must not require challenge when 2FA is disabled", action)
		}
	}

	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}
	for _, action := range listed {
		if TailscaleActionRequiresChallenge(action) {
			t.Fatalf("scripts-only default must leave %s ungated", action)
		}
	}

	policy := defaultTwoFactorProtectionPolicy()
	policy.Tailscale = true
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}
	for _, action := range listed {
		if !TailscaleActionRequiresChallenge(action) {
			t.Fatalf("tailscale=true must require challenge for %s", action)
		}
	}

	policy.Tailscale = false
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}
	for _, action := range listed {
		if TailscaleActionRequiresChallenge(action) {
			t.Fatalf("%s with tailscale=false must match 2FA-off (no challenge)", action)
		}
	}

	policy.Tailscale = true
	policy.SSH = false // independence
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}
	for _, action := range listed {
		if !TailscaleActionRequiresChallenge(action) {
			t.Fatalf("%s must honor tailscale=true even when ssh=false", action)
		}
	}
	if SSHEnableRequiresChallenge() {
		t.Fatal("ssh=false must keep SSH enable ungated while tailscale is on")
	}

	// Unknown action under TOTP: fail closed.
	if !TailscaleActionRequiresChallenge("not_a_tailscale_action") {
		t.Fatal("unknown tailscale action under TOTP must fail closed")
	}
}

func TestMCPActionRequiresChallenge(t *testing.T) {
	setupAuthHandlerTest(t)

	listed := []string{
		SensitiveActionMCPShowKey,
		SensitiveActionMCPCopyKey,
		SensitiveActionMCPRegenerateKey,
		SensitiveActionMCPEnable,
		SensitiveActionMCPDisable,
	}
	for _, action := range listed {
		if MCPActionRequiresChallenge(action) {
			t.Fatalf("%s must not require challenge when 2FA is disabled", action)
		}
	}

	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}
	for _, action := range listed {
		if MCPActionRequiresChallenge(action) {
			t.Fatalf("scripts-only default must leave %s ungated", action)
		}
	}

	policy := defaultTwoFactorProtectionPolicy()
	policy.MCP = true
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}
	for _, action := range listed {
		if !MCPActionRequiresChallenge(action) {
			t.Fatalf("mcp=true must require challenge for %s", action)
		}
	}

	policy.MCP = false
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}
	for _, action := range listed {
		if MCPActionRequiresChallenge(action) {
			t.Fatalf("%s with mcp=false must match 2FA-off (no challenge)", action)
		}
	}

	policy.MCP = true
	policy.SSH = false
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}
	for _, action := range listed {
		if !MCPActionRequiresChallenge(action) {
			t.Fatalf("%s must honor mcp=true even when ssh=false", action)
		}
	}
	if !MCPActionRequiresChallenge("not_an_mcp_action") {
		t.Fatal("unknown MCP action under TOTP must fail closed")
	}
}

func TestTwoFactorProtectionScriptsRoundTrip(t *testing.T) {
	setupAuthHandlerTest(t)
	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}

	policy := defaultTwoFactorProtectionPolicy()
	policy.Scripts = false
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}

	loaded, err := loadTwoFactorProtectionPolicy()
	if err != nil {
		t.Fatalf("loadTwoFactorProtectionPolicy returned error: %v", err)
	}
	if loaded.Scripts {
		t.Fatalf("expected scripts=false after round-trip, got %+v", loaded)
	}
	if loaded.response().Scripts {
		t.Fatal("response must expose scripts=false")
	}
}

func TestScreenTimelapseActionRequiresChallenge(t *testing.T) {
	setupAuthHandlerTest(t)

	listed := []string{
		SensitiveActionSetScreenTimelapseRetention,
		SensitiveActionTriggerScreenTimelapseCleanup,
	}
	for _, action := range listed {
		if ScreenTimelapseActionRequiresChallenge(action) {
			t.Fatalf("%s must not require a challenge when 2FA is disabled", action)
		}
	}

	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}
	for _, action := range listed {
		if ScreenTimelapseActionRequiresChallenge(action) {
			t.Fatalf("scripts-only default must leave %s ungated", action)
		}
	}

	policy := defaultTwoFactorProtectionPolicy()
	policy.ScreenTimelapse = false
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}
	for _, action := range listed {
		if ScreenTimelapseActionRequiresChallenge(action) {
			t.Fatalf("%s with screenTimelapse=false must not require a challenge", action)
		}
	}

	policy.ScreenTimelapse = true
	policy.MCP = false
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}
	for _, action := range listed {
		if !ScreenTimelapseActionRequiresChallenge(action) {
			t.Fatalf("%s must honor screenTimelapse=true independently of MCP", action)
		}
	}
	if !ScreenTimelapseActionRequiresChallenge("not_a_screen_timelapse_action") {
		t.Fatal("unknown screen timelapse action under TOTP must fail closed")
	}
}

func TestTwoFactorDisableClearsTwoFactorProtection(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}
	policy := defaultTwoFactorProtectionPolicy()
	policy.MCP = false
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}
	grantTwoFactorProtectionAuthorization("admin", "session-a", "192.0.2.1", "agent-a")
	grantMCPAuthorization("admin", "session-a", "192.0.2.1", "agent-a")
	grantScreenTimelapsePlaybackAuthorization("admin", "session-a", "192.0.2.1", "agent-a")

	_, rsp := performAuthJSONRequest(t, service.TwoFactorDisable, http.MethodPost, "/api/auth/2fa/disable", proto.TwoFactorDisableReq{
		Password: "secret",
		Code:     currentTestTOTPCode(t),
	})
	assertAuthCode(t, rsp, 0)

	if _, err := os.Stat(twoFactorProtectionFilePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected cleared protection policy file, got %v", err)
	}
	stored, err := loadTwoFactorProtectionPolicy()
	if err != nil {
		t.Fatalf("loadTwoFactorProtectionPolicy returned error: %v", err)
	}
	if stored != defaultTwoFactorProtectionPolicy() {
		t.Fatalf("expected scripts-only policy after 2FA disable, got %+v", stored)
	}
	if hasTwoFactorProtectionAuthorization("admin", "session-a", "192.0.2.1", "agent-a") {
		t.Fatal("2FA disable should clear protection authorizations")
	}
	if hasMCPAuthorization("admin", "session-a", "192.0.2.1", "agent-a") {
		t.Fatal("2FA disable should clear MCP authorizations")
	}
	if hasScreenTimelapsePlaybackAuthorization("admin", "session-a", "192.0.2.1", "agent-a") {
		t.Fatal("2FA disable should clear Screen Timelapse authorizations")
	}
	if err := saveTwoFactorProtectionPolicy(policy); !errors.Is(err, errTwoFactorProtectionDisabled) {
		t.Fatalf("saving after 2FA disable returned %v, want disabled error", err)
	}
	stored, err = loadTwoFactorProtectionPolicy()
	if err != nil {
		t.Fatalf("loadTwoFactorProtectionPolicy after rejected save returned error: %v", err)
	}
	if stored != defaultTwoFactorProtectionPolicy() {
		t.Fatalf("rejected save after 2FA disable changed policy: %+v", stored)
	}
}

func twoFactorProtectionRequest(login, webTerminal, ssh, tailscale, mcp, targetPower, nanoKVMRestart, scripts, screenTimelapse, deleteImage bool) proto.SetTwoFactorProtectionReq {
	return proto.SetTwoFactorProtectionReq{
		Login:           boolPointer(login),
		WebTerminal:     boolPointer(webTerminal),
		SSH:             boolPointer(ssh),
		Tailscale:       boolPointer(tailscale),
		MCP:             boolPointer(mcp),
		TargetPower:     boolPointer(targetPower),
		NanoKVMRestart:  boolPointer(nanoKVMRestart),
		Scripts:         boolPointer(scripts),
		ScreenTimelapse: boolPointer(screenTimelapse),
		DeleteImage:     boolPointer(deleteImage),
	}
}
