package middleware

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeadersAndTlsSetExactSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(SecurityHeaders(), Tls())
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "https://nanokvm.test/", nil)
	request.TLS = &tls.ConnectionState{}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}

	want := map[string]string{
		"Content-Security-Policy":      contentSecurityPolicy,
		"Cross-Origin-Embedder-Policy": "require-corp",
		"Cross-Origin-Opener-Policy":   "same-origin",
		"Cross-Origin-Resource-Policy": "same-origin",
		"Permissions-Policy":           permissionsPolicy,
		"Referrer-Policy":              "no-referrer",
		"Strict-Transport-Security":    "max-age=31536000; includeSubDomains",
		"X-Content-Type-Options":       "nosniff",
		"X-Frame-Options":              "DENY",
	}
	for name, expected := range want {
		if actual := recorder.Header().Get(name); actual != expected {
			t.Errorf("%s = %q, want %q", name, actual, expected)
		}
	}
}

func TestTlsKeepsHSTSRestrictedToHTTPS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(SecurityHeaders(), Tls())
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "http://nanokvm.test/", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if actual := recorder.Header().Get("Strict-Transport-Security"); actual != "" {
		t.Fatalf("Strict-Transport-Security = %q on HTTP, want empty", actual)
	}
	if actual := recorder.Header().Get("Content-Security-Policy"); actual != contentSecurityPolicy {
		t.Fatalf("Content-Security-Policy = %q, want %q", actual, contentSecurityPolicy)
	}
}

func TestSecurityHeadersProtectPlainHTTPWithoutHSTS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(SecurityHeaders())
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "http://nanokvm.test/", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if actual := recorder.Header().Get("Content-Security-Policy"); actual != contentSecurityPolicy {
		t.Fatalf("Content-Security-Policy = %q, want %q", actual, contentSecurityPolicy)
	}
	if actual := recorder.Header().Get("Strict-Transport-Security"); actual != "" {
		t.Fatalf("Strict-Transport-Security = %q on HTTP, want empty", actual)
	}
}

func TestSecurityHeadersAndHSTSSurviveEarlyRequestRejection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(SecurityHeaders(), Tls())
	router.Use(func(c *gin.Context) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": -1})
	})
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "https://nanokvm.test/", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
	if actual := recorder.Header().Get("Content-Security-Policy"); actual != contentSecurityPolicy {
		t.Fatalf("Content-Security-Policy = %q, want %q", actual, contentSecurityPolicy)
	}
	if actual := recorder.Header().Get("Strict-Transport-Security"); actual != "max-age=31536000; includeSubDomains" {
		t.Fatalf("Strict-Transport-Security = %q on rejected HTTPS request", actual)
	}
}

// The menu-bar Picture-in-Picture toggle needs same-origin access; "()"
// makes Chromium report document.pictureInPictureEnabled as false.
func TestPermissionsPolicyAllowsSameOriginPictureInPicture(t *testing.T) {
	if !strings.Contains(permissionsPolicy, "picture-in-picture=(self)") {
		t.Fatalf("permissions policy must allow same-origin picture-in-picture: %q", permissionsPolicy)
	}
}
