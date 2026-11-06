package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"NanoKVM-Server/config"

	"github.com/gin-gonic/gin"
)

func TestCheckTokenSetsUsernameFromBearerToken(t *testing.T) {
	withJWTTestConfig(t, "enable")
	token, err := GenerateJWT("admin")
	if err != nil {
		t.Fatalf("GenerateJWT returned error: %v", err)
	}

	recorder, gotUsername, gotToken := performJWTMiddlewareRequest(t, func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+token)
	})

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if gotUsername != "admin" {
		t.Fatalf("expected username %q, got %q", "admin", gotUsername)
	}
	if gotToken != token {
		t.Fatalf("expected jwtToken to match bearer token")
	}
}

func TestCheckTokenSetsUsernameFromCookieToken(t *testing.T) {
	withJWTTestConfig(t, "enable")
	token, err := GenerateJWT("operator")
	if err != nil {
		t.Fatalf("GenerateJWT returned error: %v", err)
	}

	recorder, gotUsername, gotToken := performJWTMiddlewareRequest(t, func(req *http.Request) {
		req.AddCookie(&http.Cookie{Name: "nano-kvm-token", Value: token})
	})

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if gotUsername != "operator" {
		t.Fatalf("expected username %q, got %q", "operator", gotUsername)
	}
	if gotToken != token {
		t.Fatalf("expected jwtToken to match cookie token")
	}
}

func TestCheckTokenDoesNotSetUsernameWithoutTokenOnAuthDisabled(t *testing.T) {
	withJWTTestConfig(t, "disable")

	recorder, gotUsername, gotToken := performJWTMiddlewareRequest(t, nil)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if gotUsername != "" {
		t.Fatalf("expected empty username, got %q", gotUsername)
	}
	if gotToken != "" {
		t.Fatalf("expected empty jwtToken, got %q", gotToken)
	}
}

func TestCheckTokenRejectsLocalhostWithoutToken(t *testing.T) {
	withJWTTestConfig(t, "enable")

	recorder, gotUsername, gotToken := performJWTMiddlewareRequest(t, func(req *http.Request) {
		req.RemoteAddr = "127.0.0.1:1234"
	})

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if gotUsername != "" {
		t.Fatalf("expected empty username, got %q", gotUsername)
	}
	if gotToken != "" {
		t.Fatalf("expected empty jwtToken, got %q", gotToken)
	}
}

func TestCheckTokenRejectsForwardedLoopbackWithoutToken(t *testing.T) {
	withJWTTestConfig(t, "enable")

	recorder, _, _ := performJWTMiddlewareRequest(t, func(req *http.Request) {
		req.Header.Set("X-Forwarded-For", "127.0.0.1")
		req.Header.Set("X-Real-IP", "127.0.0.1")
	})

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func withJWTTestConfig(t *testing.T, authentication string) {
	t.Helper()
	originalGetConfig := getConfig
	getConfig = func() *config.Config {
		return &config.Config{
			Authentication: authentication,
			JWT: config.JWT{
				SecretKey:            "test-secret",
				RefreshTokenDuration: 3600,
			},
		}
	}
	t.Cleanup(func() {
		getConfig = originalGetConfig
	})
}

func performJWTMiddlewareRequest(t *testing.T, mutate func(*http.Request)) (*httptest.ResponseRecorder, string, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	router := gin.New()
	var gotUsername string
	var gotToken string
	router.Use(CheckToken())
	router.GET("/api/protected", func(c *gin.Context) {
		if value, ok := c.Get("username"); ok {
			gotUsername, _ = value.(string)
		}
		if value, ok := c.Get("jwtToken"); ok {
			gotToken, _ = value.(string)
		}
		c.Status(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	if mutate != nil {
		mutate(req)
	}
	router.ServeHTTP(recorder, req)

	return recorder, gotUsername, gotToken
}
