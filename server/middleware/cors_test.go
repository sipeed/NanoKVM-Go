package middleware

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDevelopmentCORSOriginPolicy(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		host    string
		origin  string
		tls     bool
		allowed bool
	}{
		{name: "no origin API client", target: "http://device.test/api", host: "device.test", allowed: false},
		{name: "same HTTP origin", target: "http://device.test/api", host: "device.test", origin: "http://device.test", allowed: true},
		{name: "same HTTPS origin", target: "https://device.test/api", host: "device.test", origin: "https://device.test", tls: true, allowed: true},
		{name: "wrong same-origin scheme", target: "http://device.test/api", host: "device.test", origin: "https://device.test", allowed: false},
		{name: "wrong same-origin port", target: "http://device.test:80/api", host: "device.test:80", origin: "http://device.test:81", allowed: false},
		{name: "vite localhost", target: "http://device.test/api", host: "device.test", origin: "http://localhost:3001", allowed: true},
		{name: "vite IPv4", target: "http://device.test/api", host: "device.test", origin: "http://127.0.0.1:3001", allowed: true},
		{name: "vite IPv6", target: "http://device.test/api", host: "device.test", origin: "http://[::1]:3001", allowed: true},
		{name: "wrong Vite port", target: "http://device.test/api", host: "device.test", origin: "http://localhost:5173", allowed: false},
		{name: "wrong Vite scheme", target: "http://device.test/api", host: "device.test", origin: "https://localhost:3001", allowed: false},
		{name: "other loopback alias", target: "http://device.test/api", host: "device.test", origin: "http://127.0.0.2:3001", allowed: false},
		{name: "configured exact origin", target: "http://device.test/api", host: "device.test", origin: "https://dev.example:4443", allowed: true},
		{name: "configured wrong port", target: "http://device.test/api", host: "device.test", origin: "https://dev.example:4444", allowed: false},
		{name: "arbitrary external origin", target: "http://device.test/api", host: "device.test", origin: "https://evil.example", allowed: false},
		{name: "opaque origin", target: "http://device.test/api", host: "device.test", origin: "null", allowed: false},
		{name: "origin with path", target: "http://device.test/api", host: "device.test", origin: "https://dev.example:4443/path", allowed: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handlerReached := false
			router := developmentCORSTestRouter([]string{
				"https://dev.example:4443",
				"not an origin",
				"https://ignored.example/path",
			}, &handlerReached)
			request := httptest.NewRequest(http.MethodGet, test.target, nil)
			request.Host = test.host
			if test.tls {
				request.TLS = &tls.ConnectionState{}
			}
			if test.origin != "" {
				request.Header.Set("Origin", test.origin)
			}
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			actual := recorder.Header().Get("Access-Control-Allow-Origin")
			if test.origin == "" {
				if actual != "" || recorder.Code != http.StatusNoContent || !handlerReached {
					t.Fatalf("no-Origin request changed: status=%d allow-origin=%q", recorder.Code, actual)
				}
				return
			}

			if test.allowed && actual != test.origin {
				t.Fatalf("Access-Control-Allow-Origin = %q, want %q", actual, test.origin)
			}
			if test.allowed && (recorder.Code != http.StatusNoContent || !handlerReached) {
				t.Fatalf("allowed request status=%d handlerReached=%t", recorder.Code, handlerReached)
			}
			if !test.allowed && (actual != "" || recorder.Code != http.StatusForbidden || handlerReached) {
				t.Fatalf("denied request status=%d allow-origin=%q handlerReached=%t", recorder.Code, actual, handlerReached)
			}
		})
	}
}

func TestDevelopmentCORSPreflightIsNarrow(t *testing.T) {
	handlerReached := false
	router := developmentCORSTestRouter(nil, &handlerReached)
	request := httptest.NewRequest(http.MethodOptions, "http://device.test/api", nil)
	request.Host = "device.test"
	request.Header.Set("Origin", "http://localhost:3001")
	request.Header.Set("Access-Control-Request-Method", http.MethodPut)
	request.Header.Set("Access-Control-Request-Headers", "content-type,x-ap-key")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if actual := recorder.Header().Get("Access-Control-Allow-Origin"); actual != "http://localhost:3001" {
		t.Fatalf("Access-Control-Allow-Origin = %q; headers = %#v", actual, recorder.Header())
	}
	if actual := recorder.Header().Get("Access-Control-Allow-Credentials"); actual != "" {
		t.Fatalf("Access-Control-Allow-Credentials = %q, want empty", actual)
	}
	if actual := recorder.Header().Get("Access-Control-Allow-Methods"); actual != http.MethodPut {
		t.Fatalf("Access-Control-Allow-Methods = %q, want %q", actual, http.MethodPut)
	}
	if handlerReached {
		t.Fatal("preflight reached the application handler")
	}
}

func developmentCORSTestRouter(configured []string, handlerReached *bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(DevelopmentCORS(configured))
	router.Any("/api", func(c *gin.Context) {
		*handlerReached = true
		c.Status(http.StatusNoContent)
	})
	return router
}
