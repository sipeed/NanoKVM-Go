package auth

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"NanoKVM-Server/config"
	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type authTestResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func setupAuthHandlerTest(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	originalConfig := getConfig
	originalGenerateJWT := generateJWT
	originalInvalidParameterDelay := invalidParameterDelay
	originalInvalidCredentialDelay := invalidCredentialDelay
	originalTokenFailureDelay := tokenFailureDelay
	originalAuthRateLimiter := defaultAuthRateLimiter
	originalAccountFilePath := accountFilePath
	originalTOTPSecretFilePath := totpSecretFilePath
	originalTOTPIssuer := totpIssuer
	originalTOTPWindow := totpWindow
	originalNow := totpNow
	originalRecoveryCodesFilePath := recoveryCodesFilePath
	originalRecoveryCodeBcryptCost := recoveryCodeBcryptCost
	originalRecoveryCodesNow := recoveryCodesNow
	originalTwoFactorProtectionFilePath := twoFactorProtectionFilePath
	originalRegenerateSessionJWTSecretKey := regenerateSessionJWTSecretKey

	accountFilePath = filepath.Join(t.TempDir(), "pwd")
	totpSecretFilePath = filepath.Join(t.TempDir(), "totp.secret")
	recoveryCodesFilePath = filepath.Join(t.TempDir(), "recovery_codes.json")
	twoFactorProtectionFilePath = filepath.Join(t.TempDir(), "two_factor_protection.json")
	recoveryCodeBcryptCost = bcrypt.MinCost
	regenerateSessionJWTSecretKey = func() error { return nil }
	totpIssuer = "NanoKVM"
	totpWindow = 1
	totpNow = func() time.Time { return time.Unix(1700000000, 0) }
	recoveryCodesNow = func() time.Time { return time.Unix(1700000000, 0) }
	getConfig = func() *config.Config {
		return &config.Config{
			Authentication: "enable",
			JWT: config.JWT{
				SecretKey:            "test-secret",
				RefreshTokenDuration: 3600,
				RevokeTokensOnLogout: true,
			},
			TwoFactorAuth: config.TwoFactorAuth{
				SecretFile: totpSecretFilePath,
				Issuer:     "NanoKVM",
				Window:     1,
			},
		}
	}
	generateJWT = func(username string) (string, error) { return "jwt-" + username, nil }
	invalidParameterDelay = 0
	invalidCredentialDelay = 0
	tokenFailureDelay = 0
	defaultAuthRateLimiter = newAuthRateLimiter(func() time.Time { return time.Unix(1700000000, 0) })
	resetPendingTOTPSetupsForTest()
	resetLoginChallengesForTest()
	resetSensitiveActionChallengesForTest()
	resetTwoFactorProtectionAuthorizationsForTest()
	resetMCPAuthorizationsForTest()
	resetScreenTimelapsePlaybackAuthorizationsForTest()

	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword returned error: %v", err)
	}
	if err := SetAccount("admin", string(hash)); err != nil {
		t.Fatalf("SetAccount returned error: %v", err)
	}

	t.Cleanup(func() {
		getConfig = originalConfig
		generateJWT = originalGenerateJWT
		invalidParameterDelay = originalInvalidParameterDelay
		invalidCredentialDelay = originalInvalidCredentialDelay
		tokenFailureDelay = originalTokenFailureDelay
		defaultAuthRateLimiter = originalAuthRateLimiter
		accountFilePath = originalAccountFilePath
		totpSecretFilePath = originalTOTPSecretFilePath
		totpIssuer = originalTOTPIssuer
		totpWindow = originalTOTPWindow
		totpNow = originalNow
		recoveryCodesFilePath = originalRecoveryCodesFilePath
		recoveryCodeBcryptCost = originalRecoveryCodeBcryptCost
		recoveryCodesNow = originalRecoveryCodesNow
		twoFactorProtectionFilePath = originalTwoFactorProtectionFilePath
		regenerateSessionJWTSecretKey = originalRegenerateSessionJWTSecretKey
		resetPendingTOTPSetupsForTest()
		resetLoginChallengesForTest()
		resetSensitiveActionChallengesForTest()
		resetTwoFactorProtectionAuthorizationsForTest()
		resetMCPAuthorizationsForTest()
		resetScreenTimelapsePlaybackAuthorizationsForTest()
	})
}

func performAuthJSONRequest(t *testing.T, handler gin.HandlerFunc, method string, target string, body interface{}) (*httptest.ResponseRecorder, authTestResponse) {
	t.Helper()

	recorder, ctx := authTestContextWithRecorder(t, method, target, body)
	handler(ctx)

	var rsp authTestResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("Unmarshal response returned error: %v body=%s", err, recorder.Body.String())
	}

	return recorder, rsp
}

func authTestContext(t *testing.T, method string, target string, body interface{}) *gin.Context {
	t.Helper()
	_, ctx := authTestContextWithRecorder(t, method, target, body)
	return ctx
}

func authTestContextWithRecorder(t *testing.T, method string, target string, body interface{}) (*httptest.ResponseRecorder, *gin.Context) {
	t.Helper()

	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Marshal returned error: %v", err)
		}
		reader = bytes.NewReader(data)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, target, reader)
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Request.Header.Set("User-Agent", "auth-test")
	ctx.Request.RemoteAddr = "192.0.2.1:1234"
	return recorder, ctx
}

func decodeAuthData[T any](t *testing.T, rsp authTestResponse) T {
	t.Helper()
	var data T
	if err := json.Unmarshal(rsp.Data, &data); err != nil {
		t.Fatalf("Unmarshal data returned error: %v data=%s", err, string(rsp.Data))
	}
	return data
}

func currentTestTOTPCode(t *testing.T) string {
	t.Helper()
	secret, err := readActiveTOTPSecret()
	if err != nil {
		t.Fatalf("readActiveTOTPSecret returned error: %v", err)
	}
	code, err := generateTOTPCode(secret, totpNow())
	if err != nil {
		t.Fatalf("generateTOTPCode returned error: %v", err)
	}
	return code
}

func assertAuthCode(t *testing.T, rsp authTestResponse, want int) {
	t.Helper()
	if rsp.Code != want {
		t.Fatalf("expected response code %d, got %d body=%s", want, rsp.Code, string(rsp.Data))
	}
}

func assertAuthRateLimitData(t *testing.T, rsp authTestResponse) proto.AuthRateLimitRsp {
	t.Helper()
	data := decodeAuthData[proto.AuthRateLimitRsp](t, rsp)
	if !data.RateLimited || data.RetryAfter <= 0 || data.RetryAt <= 0 {
		t.Fatalf("expected rate-limit metadata, got %+v", data)
	}
	return data
}

func assertHasAuthCookie(t *testing.T, recorder *httptest.ResponseRecorder, want bool) {
	t.Helper()
	found := false
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == "nano-kvm-token" && cookie.Value != "" {
			found = true
		}
	}
	if found != want {
		t.Fatalf("expected auth cookie presence %v, got %v", want, found)
	}
}

func assertClearsAuthCookie(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == "nano-kvm-token" {
			if cookie.Value == "" && cookie.MaxAge < 0 {
				return
			}
			t.Fatalf("expected cleared auth cookie, got value=%q maxAge=%d", cookie.Value, cookie.MaxAge)
		}
	}
	t.Fatal("expected response to clear auth cookie")
}
