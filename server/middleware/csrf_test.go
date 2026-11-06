package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCSRFProtectionRejectsCrossOriginCookieStateChange(t *testing.T) {
	router := newCSRFRouter()
	req := httptest.NewRequest(http.MethodPost, "https://nanokvm.local/api/change", nil)
	req.Host = "nanokvm.local"
	req.Header.Set("Origin", "https://evil.example")
	req.AddCookie(&http.Cookie{Name: "nano-kvm-token", Value: "token"})
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden for cross-origin cookie request, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestCSRFProtectionAllowsSameOriginCookieStateChange(t *testing.T) {
	router := newCSRFRouter()
	req := httptest.NewRequest(http.MethodPost, "https://nanokvm.local/api/change", nil)
	req.Host = "nanokvm.local"
	req.Header.Set("Origin", "https://nanokvm.local")
	req.AddCookie(&http.Cookie{Name: "nano-kvm-token", Value: "token"})
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected same-origin request to pass, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestCSRFProtectionAllowsBearerStateChangeWithoutOrigin(t *testing.T) {
	router := newCSRFRouter()
	req := httptest.NewRequest(http.MethodPost, "https://nanokvm.local/api/change", nil)
	req.Host = "nanokvm.local"
	req.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected bearer API request to pass, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestCSRFProtectionAllowsLoginWithoutCookieOrigin(t *testing.T) {
	router := newCSRFRouter()
	req := httptest.NewRequest(http.MethodPost, "https://nanokvm.local/api/auth/login", nil)
	req.Host = "nanokvm.local"
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected unauthenticated login-style request to pass csrf middleware, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestCSRFProtectionAllowsTrustedOrigin(t *testing.T) {
	router := newCSRFRouterWithPolicy(csrfPolicy{
		TrustedOrigins: []string{"https://proxy.example"},
	})
	req := httptest.NewRequest(http.MethodPost, "https://nanokvm.local/api/change", nil)
	req.Host = "nanokvm.local"
	req.Header.Set("Origin", "https://proxy.example")
	req.AddCookie(&http.Cookie{Name: "nano-kvm-token", Value: "token"})
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected trusted origin request to pass, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestCSRFProtectionAllowsMissingOriginOnlyWhenConfigured(t *testing.T) {
	router := newCSRFRouterWithPolicy(csrfPolicy{AllowMissingOrigin: true})
	req := httptest.NewRequest(http.MethodPost, "https://nanokvm.local/api/change", nil)
	req.Host = "nanokvm.local"
	req.AddCookie(&http.Cookie{Name: "nano-kvm-token", Value: "token"})
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected configured missing-origin request to pass, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func newCSRFRouter() *gin.Engine {
	return newCSRFRouterWithPolicy(csrfPolicy{})
}

func newCSRFRouterWithPolicy(policy csrfPolicy) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(csrfProtection(policy))
	router.POST("/api/change", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.POST("/api/auth/login", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return router
}
