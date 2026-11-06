package auth

import (
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func resetSensitiveActionChallengesForTest() {
	sensitiveActionChallenges.Lock()
	defer sensitiveActionChallenges.Unlock()
	sensitiveActionChallenges.items = make(map[string]sensitiveActionChallenge)
}

func TestSensitiveActionChallengeConsume(t *testing.T) {
	useSensitiveActionChallengeTestClock(t)

	token := createTestSensitiveActionChallenge(t)
	if token == "" {
		t.Fatal("createSensitiveActionChallenge returned empty token")
	}

	if !consumeSensitiveActionChallenge(token, "admin", sensitiveActionChangePassword, "session", "192.0.2.1", "agent") {
		t.Fatal("expected successful sensitive action challenge consume")
	}
	if consumeSensitiveActionChallenge(token, "admin", sensitiveActionChangePassword, "session", "192.0.2.1", "agent") {
		t.Fatal("sensitive action challenge was consumed twice")
	}
}

func TestSensitiveActionChallengeRejectsExpired(t *testing.T) {
	now := useSensitiveActionChallengeTestClock(t)
	token := createTestSensitiveActionChallenge(t)

	expiredAt := now.Add(sensitiveActionChallengeTTL + time.Second)
	cleanupExpiredSensitiveActionChallenges(expiredAt)
	totpNow = func() time.Time { return expiredAt }

	if consumeSensitiveActionChallenge(token, "admin", sensitiveActionChangePassword, "session", "192.0.2.1", "agent") {
		t.Fatal("expired sensitive action challenge was accepted")
	}
}

func TestSensitiveActionChallengeRejectsMismatchedAction(t *testing.T) {
	useSensitiveActionChallengeTestClock(t)
	token := createTestSensitiveActionChallenge(t)

	if consumeSensitiveActionChallenge(token, "admin", "disable_2fa", "session", "192.0.2.1", "agent") {
		t.Fatal("sensitive action challenge accepted mismatched action")
	}
	if !consumeSensitiveActionChallenge(token, "admin", sensitiveActionChangePassword, "session", "192.0.2.1", "agent") {
		t.Fatal("mismatch attempt should not consume a valid sensitive action challenge")
	}
}

func TestSensitiveActionChallengeRejectsMismatchedUsername(t *testing.T) {
	useSensitiveActionChallengeTestClock(t)
	token := createTestSensitiveActionChallenge(t)

	if consumeSensitiveActionChallenge(token, "operator", sensitiveActionChangePassword, "session", "192.0.2.1", "agent") {
		t.Fatal("sensitive action challenge accepted mismatched username")
	}
	if !consumeSensitiveActionChallenge(token, "admin", sensitiveActionChangePassword, "session", "192.0.2.1", "agent") {
		t.Fatal("mismatch attempt should not consume a valid sensitive action challenge")
	}
}

func TestSensitiveActionChallengeRejectsMismatchedSessionIPAndUserAgent(t *testing.T) {
	useSensitiveActionChallengeTestClock(t)

	tests := []struct {
		name       string
		sessionKey string
		ip         string
		userAgent  string
	}{
		{name: "session", sessionKey: "other-session", ip: "192.0.2.1", userAgent: "agent"},
		{name: "ip", sessionKey: "session", ip: "192.0.2.2", userAgent: "agent"},
		{name: "user-agent", sessionKey: "session", ip: "192.0.2.1", userAgent: "other-agent"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetSensitiveActionChallengesForTest()
			token := createTestSensitiveActionChallenge(t)

			if consumeSensitiveActionChallenge(token, "admin", sensitiveActionChangePassword, tt.sessionKey, tt.ip, tt.userAgent) {
				t.Fatalf("sensitive action challenge accepted mismatched %s", tt.name)
			}
			if !consumeSensitiveActionChallenge(token, "admin", sensitiveActionChangePassword, "session", "192.0.2.1", "agent") {
				t.Fatal("mismatch attempt should not consume a valid sensitive action challenge")
			}
		})
	}
}

func TestSensitiveActionChallengeRejectsUnsupportedAction(t *testing.T) {
	useSensitiveActionChallengeTestClock(t)

	if _, err := createSensitiveActionChallenge("admin", "disable_2fa", "session", "192.0.2.1", "agent"); err == nil {
		t.Fatal("expected unsupported sensitive action to fail")
	}
}

func TestOpenWebTerminalIsValidNonProtectionSensitiveAction(t *testing.T) {
	if !validSensitiveAction(SensitiveActionOpenWebTerminal) {
		t.Fatal("open_web_terminal must be a valid sensitive action")
	}
	if isProtectionSensitiveAction(SensitiveActionOpenWebTerminal) {
		t.Fatal("open_web_terminal must require the Web password even when 2FA is enabled")
	}
}

func TestUpdateSignatureVerificationChallengeIsSingleUseAndContextBound(t *testing.T) {
	setupAuthHandlerTest(t)
	useSensitiveActionChallengeTestClock(t)

	newContext := func(token, username, ip, userAgent string) *gin.Context {
		ctx := authTestContext(t, "POST", "/api/application/signature-verification", nil)
		ctx.Request.Header.Set("Authorization", "Bearer "+token)
		ctx.Request.Header.Set("User-Agent", userAgent)
		ctx.Request.RemoteAddr = ip + ":1234"
		ctx.Set("username", username)
		return ctx
	}
	newChallenge := func(ctx *gin.Context) string {
		token, err := createSensitiveActionChallenge(
			"admin",
			SensitiveActionUpdateSignatureVerification,
			sensitiveOperationSessionKey(ctx),
			ctx.ClientIP(),
			ctx.Request.UserAgent(),
		)
		if err != nil {
			t.Fatalf("create update signature verification challenge: %v", err)
		}
		return token
	}

	for _, test := range []struct {
		name      string
		action    string
		token     string
		username  string
		ip        string
		userAgent string
	}{
		{name: "action", action: SensitiveActionRunScript, token: "session-a", username: "admin", ip: "192.0.2.1", userAgent: "agent-a"},
		{name: "session", action: SensitiveActionUpdateSignatureVerification, token: "session-b", username: "admin", ip: "192.0.2.1", userAgent: "agent-a"},
		{name: "username", action: SensitiveActionUpdateSignatureVerification, token: "session-a", username: "operator", ip: "192.0.2.1", userAgent: "agent-a"},
		{name: "IP", action: SensitiveActionUpdateSignatureVerification, token: "session-a", username: "admin", ip: "192.0.2.2", userAgent: "agent-a"},
		{name: "user agent", action: SensitiveActionUpdateSignatureVerification, token: "session-a", username: "admin", ip: "192.0.2.1", userAgent: "agent-b"},
	} {
		t.Run(test.name, func(t *testing.T) {
			originalContext := newContext("session-a", "admin", "192.0.2.1", "agent-a")
			challenge := newChallenge(originalContext)
			mismatchedContext := newContext(test.token, test.username, test.ip, test.userAgent)
			if err := VerifySensitiveActionChallenge(mismatchedContext, challenge, test.action); err == nil {
				t.Fatalf("challenge accepted mismatched %s", test.name)
			}
			if err := VerifySensitiveActionChallenge(
				originalContext,
				challenge,
				SensitiveActionUpdateSignatureVerification,
			); err != nil {
				t.Fatalf("mismatch consumed the challenge: %v", err)
			}
			if err := VerifySensitiveActionChallenge(
				originalContext,
				challenge,
				SensitiveActionUpdateSignatureVerification,
			); err == nil {
				t.Fatal("challenge was accepted more than once")
			}
		})
	}
}

func TestVerifyMCPChallengeGrantsSharedAuthorization(t *testing.T) {
	setupAuthHandlerTest(t)
	ctx := authTestContext(t, "POST", "/api/ai/mcp/key", nil)
	sessionKey := sensitiveOperationSessionKey(ctx)
	challenge, err := createSensitiveActionChallenge(
		"admin",
		SensitiveActionMCPShowKey,
		sessionKey,
		ctx.ClientIP(),
		ctx.Request.UserAgent(),
	)
	if err != nil {
		t.Fatalf("createSensitiveActionChallenge returned error: %v", err)
	}

	if err := VerifyMCPChallenge(ctx, challenge, SensitiveActionMCPShowKey); err != nil {
		t.Fatalf("VerifyMCPChallenge returned error: %v", err)
	}
	if !hasMCPAuthorization("admin", sessionKey, ctx.ClientIP(), ctx.Request.UserAgent()) {
		t.Fatal("successful MCP verification did not create authorization")
	}
	if err := VerifyMCPChallenge(ctx, "", SensitiveActionMCPDisable); err != nil {
		t.Fatalf("shared MCP authorization did not allow another MCP action: %v", err)
	}
}

func TestMCPAuthorizationIsBoundAndExpires(t *testing.T) {
	now := useSensitiveActionChallengeTestClock(t)
	resetMCPAuthorizationsForTest()
	t.Cleanup(resetMCPAuthorizationsForTest)

	grantMCPAuthorization("admin", "session-a", "192.0.2.1", "agent-a")
	if !hasMCPAuthorization("admin", "session-a", "192.0.2.1", "agent-a") {
		t.Fatal("expected active MCP authorization")
	}
	if hasMCPAuthorization("admin", "session-b", "192.0.2.1", "agent-a") {
		t.Fatal("MCP authorization accepted another session")
	}
	if hasMCPAuthorization("admin", "session-a", "192.0.2.2", "agent-a") {
		t.Fatal("MCP authorization accepted another client IP")
	}
	if hasMCPAuthorization("admin", "session-a", "192.0.2.1", "agent-b") {
		t.Fatal("MCP authorization accepted another user agent")
	}

	expiredAt := now.Add(mcpAuthorizationTTL + time.Second)
	totpNow = func() time.Time { return expiredAt }
	if hasMCPAuthorization("admin", "session-a", "192.0.2.1", "agent-a") {
		t.Fatal("expired MCP authorization was accepted")
	}
}

func TestVerifyScreenTimelapsePlaybackChallengeGrantsAuthorization(t *testing.T) {
	setupAuthHandlerTest(t)
	if err := writeActiveTOTPSecret("JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("writeActiveTOTPSecret returned error: %v", err)
	}
	policy := defaultTwoFactorProtectionPolicy()
	policy.ScreenTimelapse = true
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		t.Fatalf("saveTwoFactorProtectionPolicy returned error: %v", err)
	}
	ctx := authTestContext(t, "POST", "/api/ai/screen-timelapse/access", nil)
	sessionKey := sensitiveOperationSessionKey(ctx)
	challenge, err := createSensitiveActionChallenge(
		"admin",
		SensitiveActionScreenTimelapsePlayback,
		sessionKey,
		ctx.ClientIP(),
		ctx.Request.UserAgent(),
	)
	if err != nil {
		t.Fatalf("createSensitiveActionChallenge returned error: %v", err)
	}

	if err := VerifyScreenTimelapsePlaybackChallenge(ctx, challenge); err != nil {
		t.Fatalf("VerifyScreenTimelapsePlaybackChallenge returned error: %v", err)
	}
	if !hasScreenTimelapsePlaybackAuthorization("admin", sessionKey, ctx.ClientIP(), ctx.Request.UserAgent()) {
		t.Fatal("successful Screen Timelapse verification did not create authorization")
	}
	if err := VerifyScreenTimelapsePlaybackChallenge(ctx, ""); err != nil {
		t.Fatalf("existing Screen Timelapse authorization did not allow playback: %v", err)
	}
}

func TestScreenTimelapsePlaybackAuthorizationIsBoundAndExpires(t *testing.T) {
	now := useSensitiveActionChallengeTestClock(t)
	resetScreenTimelapsePlaybackAuthorizationsForTest()
	t.Cleanup(resetScreenTimelapsePlaybackAuthorizationsForTest)

	grantScreenTimelapsePlaybackAuthorization("admin", "session-a", "192.0.2.1", "agent-a")
	if !hasScreenTimelapsePlaybackAuthorization("admin", "session-a", "192.0.2.1", "agent-a") {
		t.Fatal("expected active Screen Timelapse authorization")
	}
	if hasScreenTimelapsePlaybackAuthorization("admin", "session-b", "192.0.2.1", "agent-a") {
		t.Fatal("Screen Timelapse authorization accepted another session")
	}
	if hasScreenTimelapsePlaybackAuthorization("admin", "session-a", "192.0.2.2", "agent-a") {
		t.Fatal("Screen Timelapse authorization accepted another client IP")
	}
	if hasScreenTimelapsePlaybackAuthorization("admin", "session-a", "192.0.2.1", "agent-b") {
		t.Fatal("Screen Timelapse authorization accepted another user agent")
	}
	if hasMCPAuthorization("admin", "session-a", "192.0.2.1", "agent-a") {
		t.Fatal("Screen Timelapse authorization must not grant MCP access")
	}

	expiredAt := now.Add(screenTimelapsePlaybackAuthorizationTTL + time.Second)
	totpNow = func() time.Time { return expiredAt }
	if hasScreenTimelapsePlaybackAuthorization("admin", "session-a", "192.0.2.1", "agent-a") {
		t.Fatal("expired Screen Timelapse authorization was accepted")
	}
}

func createTestSensitiveActionChallenge(t *testing.T) string {
	t.Helper()
	token, err := createSensitiveActionChallenge("admin", sensitiveActionChangePassword, "session", "192.0.2.1", "agent")
	if err != nil {
		t.Fatalf("createSensitiveActionChallenge returned error: %v", err)
	}
	return token
}

func useSensitiveActionChallengeTestClock(t *testing.T) time.Time {
	t.Helper()
	resetSensitiveActionChallengesForTest()
	t.Cleanup(resetSensitiveActionChallengesForTest)
	originalNow := totpNow
	now := time.Unix(1700000000, 0)
	totpNow = func() time.Time { return now }
	t.Cleanup(func() {
		totpNow = originalNow
	})
	return now
}
