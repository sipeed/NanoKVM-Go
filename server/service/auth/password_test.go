package auth

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
)

func TestChangePasswordTwoFactorDisabledAllowsNoChallenge(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()

	_, rsp := performChangePasswordRequest(t, service, proto.ChangePasswordReq{Username: "admin", Password: "new-secret"}, "admin")

	assertAuthCode(t, rsp, 0)
	if !CompareAccount("admin", "new-secret") {
		t.Fatal("password change did not update account password")
	}
}

func TestChangePasswordDoesNotInvokePasswd(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	binDir := t.TempDir()
	marker := filepath.Join(t.TempDir(), "passwd-invoked")
	fakePasswd := filepath.Join(binDir, "passwd")
	if err := os.WriteFile(fakePasswd, []byte("#!/bin/sh\n: > \"$NANOKVM_TEST_PASSWD_MARKER\"\n"), 0o755); err != nil {
		t.Fatalf("write fake passwd: %v", err)
	}
	t.Setenv("PATH", binDir)
	t.Setenv("NANOKVM_TEST_PASSWD_MARKER", marker)

	_, rsp := performChangePasswordRequest(t, service, proto.ChangePasswordReq{Username: "admin", Password: "web-only-secret"}, "admin")

	assertAuthCode(t, rsp, 0)
	if !CompareAccount("admin", "web-only-secret") {
		t.Fatal("password change did not update web account password")
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("web password change invoked passwd: %v", err)
	}
}

func TestChangePasswordTwoFactorEnabledRequiresChallenge(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	enableTOTPForSensitiveActionTest(t)

	_, rsp := performChangePasswordRequest(t, service, proto.ChangePasswordReq{Username: "admin", Password: "new-secret"}, "admin")

	assertAuthCode(t, rsp, -2)
	if rsp.Msg != invalidAuthMessage {
		t.Fatalf("expected generic auth failure message, got %q", rsp.Msg)
	}
	if !CompareAccount("admin", "secret") {
		t.Fatal("missing challenge should not change password")
	}
}

func TestChangePasswordBypassDirectPostWithoutChallengeFails(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	enableTOTPForSensitiveActionTest(t)

	_, rsp := performChangePasswordRequest(t, service, proto.ChangePasswordReq{Username: "admin", Password: "new-secret"}, "admin")

	assertAuthCode(t, rsp, -2)
	if rsp.Msg != invalidAuthMessage {
		t.Fatalf("expected generic auth failure message, got %q", rsp.Msg)
	}
	if !CompareAccount("admin", "secret") {
		t.Fatal("direct password change bypass should not change password")
	}
}

func TestChangePasswordBypassMissingRouterStateFails(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	enableTOTPForSensitiveActionTest(t)

	_, rsp := performChangePasswordRequest(t, service, proto.ChangePasswordReq{Username: "admin", Password: "new-secret"}, "admin")

	assertAuthCode(t, rsp, -2)
	if rsp.Msg != invalidAuthMessage {
		t.Fatalf("expected generic auth failure message, got %q", rsp.Msg)
	}
	if !CompareAccount("admin", "secret") {
		t.Fatal("missing router-state challenge should not change password")
	}
}

func TestChangePasswordRejectsMismatchedChallengeContext(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	enableTOTPForSensitiveActionTest(t)

	tests := []struct {
		name            string
		challengeUser   string
		challengeAction string
		jwtUsername     string
	}{
		{name: "wrong-action", challengeUser: "admin", challengeAction: "disable_2fa", jwtUsername: "admin"},
		{name: "wrong-user", challengeUser: "operator", challengeAction: sensitiveActionChangePassword, jwtUsername: "admin"},
		{name: "jwt-user-mismatch", challengeUser: "admin", challengeAction: sensitiveActionChangePassword, jwtUsername: "operator"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetSensitiveActionChallengesForTest()
			challenge := createPasswordChangeChallengeForRequest(t, tt.challengeUser, tt.challengeAction)
			_, rsp := performChangePasswordRequest(t, service, proto.ChangePasswordReq{Username: "admin", Password: "new-secret", Challenge: challenge}, tt.jwtUsername)

			assertAuthCode(t, rsp, -2)
			if rsp.Msg != invalidAuthMessage {
				t.Fatalf("expected generic auth failure message, got %q", rsp.Msg)
			}
			if !CompareAccount("admin", "secret") {
				t.Fatal("mismatched challenge should not change password")
			}
		})
	}
}

func TestChangePasswordChallengeWrongActionBypassFails(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	enableTOTPForSensitiveActionTest(t)
	challenge := createPasswordChangeChallengeForRequest(t, "admin", "disable_2fa")

	_, rsp := performChangePasswordRequest(t, service, proto.ChangePasswordReq{Username: "admin", Password: "new-secret", Challenge: challenge}, "admin")

	assertAuthCode(t, rsp, -2)
	if rsp.Msg != invalidAuthMessage {
		t.Fatalf("expected generic auth failure message, got %q", rsp.Msg)
	}
	if !CompareAccount("admin", "secret") {
		t.Fatal("wrong-action challenge should not change password")
	}
}

func TestChangePasswordChallengeJWTUserMismatchBypassFails(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	enableTOTPForSensitiveActionTest(t)
	challenge := createPasswordChangeChallengeForRequest(t, "admin", sensitiveActionChangePassword)

	_, rsp := performChangePasswordRequest(t, service, proto.ChangePasswordReq{Username: "admin", Password: "new-secret", Challenge: challenge}, "operator")

	assertAuthCode(t, rsp, -2)
	if rsp.Msg != invalidAuthMessage {
		t.Fatalf("expected generic auth failure message, got %q", rsp.Msg)
	}
	if !CompareAccount("admin", "secret") {
		t.Fatal("JWT user mismatch should not change password")
	}
}

func TestChangePasswordRejectsExpiredAndUsedChallenge(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	enableTOTPForSensitiveActionTest(t)

	challenge := createPasswordChangeChallengeForRequest(t, "admin", sensitiveActionChangePassword)
	expiredAt := totpNow().Add(sensitiveActionChallengeTTL + time.Second)
	cleanupExpiredSensitiveActionChallenges(expiredAt)
	totpNow = func() time.Time { return expiredAt }
	_, rsp := performChangePasswordRequest(t, service, proto.ChangePasswordReq{Username: "admin", Password: "new-secret", Challenge: challenge}, "admin")
	assertAuthCode(t, rsp, -2)

	totpNow = func() time.Time { return time.Unix(1700000000, 0) }
	challenge = createPasswordChangeChallengeForRequest(t, "admin", sensitiveActionChangePassword)
	_, rsp = performChangePasswordRequest(t, service, proto.ChangePasswordReq{Username: "admin", Password: "new-secret", Challenge: challenge}, "admin")
	assertAuthCode(t, rsp, 0)
	_, rsp = performChangePasswordRequest(t, service, proto.ChangePasswordReq{Username: "admin", Password: "again", Challenge: challenge}, "admin")
	assertAuthCode(t, rsp, -2)
	if !CompareAccount("admin", "new-secret") {
		t.Fatal("used challenge should not allow a second password change")
	}
}

func TestChangePasswordWithValidChallengeCanBeUsedOnce(t *testing.T) {
	setupAuthHandlerTest(t)
	service := NewService()
	enableTOTPForSensitiveActionTest(t)
	challenge := createPasswordChangeChallengeForRequest(t, "admin", sensitiveActionChangePassword)

	_, rsp := performChangePasswordRequest(t, service, proto.ChangePasswordReq{Username: "admin", Password: "new-secret", Challenge: challenge}, "admin")

	assertAuthCode(t, rsp, 0)
	if !CompareAccount("admin", "new-secret") {
		t.Fatal("valid challenge should change account password")
	}

	_, rsp = performChangePasswordRequest(t, service, proto.ChangePasswordReq{Username: "admin", Password: "again", Challenge: challenge}, "admin")
	assertAuthCode(t, rsp, -2)
	if !CompareAccount("admin", "new-secret") {
		t.Fatal("reusing challenge should not change account password")
	}
}

func TestChangePasswordKeepsTwoFactorState(t *testing.T) {
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
	challenge := createPasswordChangeChallengeForRequest(t, "admin", sensitiveActionChangePassword)

	_, rsp := performChangePasswordRequest(t, service, proto.ChangePasswordReq{Username: "admin", Password: "new-secret", Challenge: challenge}, "admin")

	assertAuthCode(t, rsp, 0)
	assertTOTPEnabled(t, true)
	enabled, remaining, err := recoveryCodesStatus()
	if err != nil {
		t.Fatalf("recoveryCodesStatus returned error: %v", err)
	}
	if !enabled || remaining != 1 {
		t.Fatalf("password change should keep recovery codes, got enabled=%v remaining=%d", enabled, remaining)
	}
}

func createPasswordChangeChallengeForRequest(t *testing.T, username, action string) string {
	t.Helper()
	ctx := authTestContext(t, http.MethodPost, "/api/auth/password", nil)
	ctx.Request.Header.Set("Authorization", "Bearer raw-jwt-token")
	if action != sensitiveActionChangePassword {
		return createSensitiveActionChallengeFixture(ctx, username, action)
	}
	challenge, err := createSensitiveActionChallenge(username, action, sensitiveOperationSessionKey(ctx), ctx.ClientIP(), ctx.Request.UserAgent())
	if err != nil {
		t.Fatalf("createSensitiveActionChallenge returned error: %v", err)
	}
	return challenge
}

func createSensitiveActionChallengeFixture(ctx *gin.Context, username, action string) string {
	token := "test-unsupported-action"
	sensitiveActionChallenges.Lock()
	defer sensitiveActionChallenges.Unlock()
	sensitiveActionChallenges.items[sensitiveActionChallengeKey(token)] = sensitiveActionChallenge{
		username:        username,
		action:          action,
		createdAt:       totpNow(),
		expiresAt:       totpNow().Add(sensitiveActionChallengeTTL),
		ip:              ctx.ClientIP(),
		userAgentDigest: userAgentHash(ctx.Request.UserAgent()),
		sessionKeyHash:  sensitiveActionSessionKeyHash(sensitiveOperationSessionKey(ctx)),
	}
	return token
}

func performChangePasswordRequest(t *testing.T, service *Service, req proto.ChangePasswordReq, jwtUsername string) (*gin.Context, authTestResponse) {
	t.Helper()
	recorder, ctx := authTestContextWithRecorder(t, http.MethodPost, "/api/auth/password", req)
	ctx.Request.Header.Set("Authorization", "Bearer raw-jwt-token")
	if jwtUsername != "" {
		ctx.Set("username", jwtUsername)
	}
	service.ChangePassword(ctx)

	var rsp authTestResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("Unmarshal response returned error: %v body=%s", err, recorder.Body.String())
	}
	return ctx, rsp
}
