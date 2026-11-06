package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"NanoKVM-Server/middleware"

	"github.com/gin-gonic/gin"
)

func TestInitializeFingerRobotSkipsPWMWhenSettingIsDisabled(t *testing.T) {
	originalValidate := fingerRobotProfilesValidate
	originalEnabled := fingerRobotSettingEnabled
	originalInitialize := fingerRobotControllerInitialize
	t.Cleanup(func() {
		fingerRobotProfilesValidate = originalValidate
		fingerRobotSettingEnabled = originalEnabled
		fingerRobotControllerInitialize = originalInitialize
	})

	fingerRobotProfilesValidate = func() error { return nil }
	fingerRobotSettingEnabled = func() bool { return false }
	fingerRobotControllerInitialize = func() error {
		t.Fatal("disabled finger robot must not initialize PWM")
		return nil
	}

	initializeFingerRobotIfEnabled()
}

func TestPublicHTTPServerBoundsHandshakeWithoutTimingOutStreams(t *testing.T) {
	server := newPublicHTTPServer(":443", http.NotFoundHandler())
	if server.ReadHeaderTimeout != publicReadHeaderTimeout || server.IdleTimeout != publicIdleTimeout {
		t.Fatalf("unexpected public server bounds: %#v", server)
	}
	if server.ReadTimeout != 0 || server.WriteTimeout != 0 {
		t.Fatalf("public streaming server has lifetime timeout: read=%s write=%s", server.ReadTimeout, server.WriteTimeout)
	}
	if server.MaxHeaderBytes != serverMaxHeaderBytes {
		t.Fatalf("public MaxHeaderBytes=%d", server.MaxHeaderBytes)
	}
}

func TestUIStateSocketStartupFailureIsNonFatal(t *testing.T) {
	originalStart := uiStateSocketStart
	t.Cleanup(func() {
		uiStateSocketStart = originalStart
	})

	want := errors.New("UI socket unavailable")
	calls := 0
	uiStateSocketStart = func() error {
		calls++
		return want
	}

	// The public server must remain eligible to start if the optional local
	// status transport cannot bind.
	startUIStateSocketServer()
	if calls != 1 {
		t.Fatalf("UI socket start calls = %d, want 1", calls)
	}
}

func TestUIStateSocketStopHookIsNonFatal(t *testing.T) {
	originalStop := uiStateSocketStop
	t.Cleanup(func() {
		uiStateSocketStop = originalStop
	})

	calls := 0
	uiStateSocketStop = func() error {
		calls++
		return errors.New("UI socket cleanup failed")
	}

	stopUIStateSocketServer()
	if calls != 1 {
		t.Fatalf("UI socket stop calls = %d, want 1", calls)
	}
}

func TestRedirectHTTPServerUsesFiniteRequestAndResponseBounds(t *testing.T) {
	server := newRedirectHTTPServer(":80", http.NotFoundHandler())
	if server.ReadHeaderTimeout != redirectReadHeaderTimeout ||
		server.ReadTimeout != redirectReadTimeout ||
		server.WriteTimeout != redirectWriteTimeout ||
		server.IdleTimeout != redirectIdleTimeout {
		t.Fatalf("unexpected redirect server bounds: %#v", server)
	}
	if server.MaxHeaderBytes != serverMaxHeaderBytes {
		t.Fatalf("redirect MaxHeaderBytes=%d", server.MaxHeaderBytes)
	}
}

func TestRedirectHandlerAppliesHTTPHeadersWithoutHSTS(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://nanokvm.test/path", nil)
	recorder := httptest.NewRecorder()

	redirectHandler(":8443", nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusTemporaryRedirect {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusTemporaryRedirect)
	}
	if actual := recorder.Header().Get("Location"); actual != "https://localhost:8443/path" {
		t.Fatalf("Location = %q", actual)
	}
	if actual := recorder.Header().Get("Content-Security-Policy"); actual == "" {
		t.Fatal("Content-Security-Policy is missing from HTTP redirect")
	}
	if actual := recorder.Header().Get("X-Frame-Options"); actual != "DENY" {
		t.Fatalf("X-Frame-Options = %q, want DENY", actual)
	}
	if actual := recorder.Header().Get("Strict-Transport-Security"); actual != "" {
		t.Fatalf("Strict-Transport-Security = %q on HTTP redirect, want empty", actual)
	}
}

func TestAPPortalExternalHostRedirectsToFixedGatewayWithFragmentPassword(t *testing.T) {
	password := "AP key+&?/="
	oldPath := apPortalPasswordFile
	apPortalPasswordFile = filepath.Join(t.TempDir(), "ap.pass")
	if err := os.WriteFile(apPortalPasswordFile, []byte(password+"\n"), 0o600); err != nil {
		t.Fatalf("write password fixture: %v", err)
	}
	t.Cleanup(func() {
		apPortalPasswordFile = oldPath
	})

	request := apPortalRequest(http.MethodGet, "http://connectivitycheck.example/generate_204?p=untrusted", "connectivitycheck.example")
	recorder := httptest.NewRecorder()

	redirectHandler(":8443", nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusFound)
	}

	encodedPassword := url.Values{"p": []string{password}}.Encode()
	want := "http://" + apPortalGateway + "/#/wifi?" + encodedPassword
	if actual := recorder.Header().Get("Location"); actual != want {
		t.Fatalf("Location = %q, want %q", actual, want)
	}

	target, err := url.Parse(recorder.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse redirect target: %v", err)
	}
	if target.RawQuery != "" {
		t.Fatalf("password leaked into redirect query: %q", target.RawQuery)
	}
	if actual := target.EscapedFragment(); actual != "/wifi?"+encodedPassword {
		t.Fatalf("escaped fragment = %q", actual)
	}
	if actual := recorder.Header().Get("Strict-Transport-Security"); actual != "" {
		t.Fatalf("Strict-Transport-Security = %q on AP HTTP redirect, want empty", actual)
	}
}

func TestAPPortalMissingPasswordOpensWiFiPageWithoutFragmentParameter(t *testing.T) {
	oldPath := apPortalPasswordFile
	apPortalPasswordFile = filepath.Join(t.TempDir(), "missing-ap.pass")
	t.Cleanup(func() {
		apPortalPasswordFile = oldPath
	})

	request := apPortalRequest(http.MethodGet, "http://connectivitycheck.example/generate_204", "connectivitycheck.example")
	recorder := httptest.NewRecorder()

	redirectHandler(":8443", nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusFound)
	}
	if actual := recorder.Header().Get("Location"); actual != "http://"+apPortalGateway+"/#/wifi" {
		t.Fatalf("Location = %q", actual)
	}
}

func TestReadBoundedAPPortalPassword(t *testing.T) {
	oldPath := apPortalPasswordFile
	apPortalPasswordFile = filepath.Join(t.TempDir(), "ap.pass")
	t.Cleanup(func() {
		apPortalPasswordFile = oldPath
	})

	if err := os.WriteFile(apPortalPasswordFile, []byte("valid-password\n"), 0o600); err != nil {
		t.Fatalf("write password fixture: %v", err)
	}
	password, err := readBoundedAPPortalPassword()
	if err != nil {
		t.Fatalf("read valid password: %v", err)
	}
	if password != "valid-password" {
		t.Fatalf("password = %q", password)
	}

	if err := os.WriteFile(apPortalPasswordFile, []byte(string(make([]byte, apPortalPasswordMaxBytes+1))), 0o600); err != nil {
		t.Fatalf("write oversized password fixture: %v", err)
	}
	if _, err := readBoundedAPPortalPassword(); err == nil {
		t.Fatal("oversized password was accepted")
	}

	secretPath := filepath.Join(t.TempDir(), "not-an-ap-password")
	if err := os.WriteFile(secretPath, []byte("not an AP password"), 0o600); err != nil {
		t.Fatalf("write symlink target fixture: %v", err)
	}
	if err := os.Remove(apPortalPasswordFile); err != nil {
		t.Fatalf("remove password fixture: %v", err)
	}
	if err := os.Symlink(secretPath, apPortalPasswordFile); err != nil {
		t.Fatalf("create password symlink fixture: %v", err)
	}
	if _, err := readBoundedAPPortalPassword(); err == nil {
		t.Fatal("symlinked AP password file was accepted")
	}
}

func TestAPPortalDelegatesOnlyStaticRequestsAndAPEndpoints(t *testing.T) {
	var delegated []string
	application := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		delegated = append(delegated, request.Method+" "+request.URL.Path)
		writer.Header().Set("X-AP-Portal-Delegated", "true")
		writer.WriteHeader(http.StatusNoContent)
	})

	for _, test := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/"},
		{method: http.MethodHead, path: "/assets/app.js"},
		{method: http.MethodPost, path: "/api/network/wifi/verify"},
		{method: http.MethodPost, path: "/api/network/wifi"},
		{method: http.MethodGet, path: "/api/network/wifi/scan"},
	} {
		t.Run(test.method+"_"+test.path, func(t *testing.T) {
			request := apPortalRequest(test.method, "http://"+apPortalGateway+test.path, apPortalGateway)
			recorder := httptest.NewRecorder()

			redirectHandler(":8443", application).ServeHTTP(recorder, request)

			if recorder.Code != http.StatusNoContent {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
			}
			if actual := recorder.Header().Get("X-AP-Portal-Delegated"); actual != "true" {
				t.Fatalf("delegation header = %q", actual)
			}
		})
	}

	if got, want := len(delegated), 5; got != want {
		t.Fatalf("delegated requests = %d, want %d", got, want)
	}
}

func TestAPPortalRejectsDisallowedRequests(t *testing.T) {
	delegated := false
	application := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		delegated = true
	})

	for _, test := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/network/wifi/verify"},
		{method: http.MethodPost, path: "/api/network/wifi/scan"},
		{method: http.MethodGet, path: "/api/network/wifi"},
		{method: http.MethodDelete, path: "/api/network/wifi"},
		{method: http.MethodGet, path: "/api"},
		{method: http.MethodPost, path: "/"},
		{method: http.MethodOptions, path: "/assets/app.js"},
	} {
		t.Run(test.method+"_"+test.path, func(t *testing.T) {
			delegated = false
			request := apPortalRequest(test.method, "http://"+apPortalGateway+test.path, apPortalGateway)
			recorder := httptest.NewRecorder()

			redirectHandler(":8443", application).ServeHTTP(recorder, request)

			if recorder.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
			}
			if delegated {
				t.Fatal("non-whitelisted API request reached the application handler")
			}
		})
	}
}

func TestAPPortalDelegatedHTTPResponseDoesNotEmitHSTS(t *testing.T) {
	originalGinMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() {
		gin.SetMode(originalGinMode)
	})
	application := gin.New()
	application.Use(middleware.SecurityHeaders(), middleware.Tls())
	application.GET("/", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := apPortalRequest(http.MethodGet, "http://"+apPortalGateway+"/", apPortalGateway)
	recorder := httptest.NewRecorder()

	redirectHandler(":8443", application).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if actual := recorder.Header().Get("Strict-Transport-Security"); actual != "" {
		t.Fatalf("Strict-Transport-Security = %q on delegated AP HTTP response, want empty", actual)
	}
}

func TestAPPortalHostSpoofOutsideGatewayRemainsHTTPSRedirect(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://"+apPortalGateway+"/probe", nil)
	request.Host = apPortalGateway
	request = request.WithContext(context.WithValue(request.Context(), http.LocalAddrContextKey, &net.TCPAddr{
		IP:   net.ParseIP("192.168.3.248"),
		Port: 80,
	}))
	recorder := httptest.NewRecorder()

	redirectHandler(":8443", http.NotFoundHandler()).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusTemporaryRedirect {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusTemporaryRedirect)
	}
	if actual := recorder.Header().Get("Location"); actual != "https://192.168.3.248:8443/probe" {
		t.Fatalf("Location = %q", actual)
	}
}

func TestAPPortalRequiresGatewayHTTPPort(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://"+apPortalGateway+"/probe", nil)
	request.Host = apPortalGateway
	request = request.WithContext(context.WithValue(request.Context(), http.LocalAddrContextKey, &net.TCPAddr{
		IP:   net.ParseIP(apPortalGateway),
		Port: 443,
	}))
	recorder := httptest.NewRecorder()

	redirectHandler(":8443", http.NotFoundHandler()).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusTemporaryRedirect {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusTemporaryRedirect)
	}
	if actual := recorder.Header().Get("Location"); actual != "https://"+apPortalGateway+":8443/probe" {
		t.Fatalf("Location = %q", actual)
	}
}

func apPortalRequest(method string, target string, host string) *http.Request {
	request := httptest.NewRequest(method, target, nil)
	request.Host = host
	return request.WithContext(context.WithValue(request.Context(), http.LocalAddrContextKey, &net.TCPAddr{
		IP:   net.ParseIP(apPortalGateway),
		Port: 80,
	}))
}

func TestInitializeFingerRobotRunsWhenSettingIsEnabled(t *testing.T) {
	originalValidate := fingerRobotProfilesValidate
	originalEnabled := fingerRobotSettingEnabled
	originalInitialize := fingerRobotControllerInitialize
	t.Cleanup(func() {
		fingerRobotProfilesValidate = originalValidate
		fingerRobotSettingEnabled = originalEnabled
		fingerRobotControllerInitialize = originalInitialize
	})

	fingerRobotProfilesValidate = func() error { return nil }
	fingerRobotSettingEnabled = func() bool { return true }
	var calls int
	fingerRobotControllerInitialize = func() error {
		calls++
		return errors.New("PWM unavailable")
	}

	// Hardware failure is deliberately non-fatal to Server startup.
	initializeFingerRobotIfEnabled()
	if calls != 1 {
		t.Fatalf("initialize calls = %d, want 1", calls)
	}
}

func TestInitializeFingerRobotRejectsInvalidProfilesBeforeSettingsAndPWM(t *testing.T) {
	originalValidate := fingerRobotProfilesValidate
	originalEnabled := fingerRobotSettingEnabled
	originalInitialize := fingerRobotControllerInitialize
	t.Cleanup(func() {
		fingerRobotProfilesValidate = originalValidate
		fingerRobotSettingEnabled = originalEnabled
		fingerRobotControllerInitialize = originalInitialize
	})

	fingerRobotProfilesValidate = func() error { return errors.New("invalid arm profile") }
	fingerRobotSettingEnabled = func() bool {
		t.Fatal("invalid profiles must fail before reading the enable setting")
		return true
	}
	fingerRobotControllerInitialize = func() error {
		t.Fatal("invalid profiles must not initialize PWM")
		return nil
	}

	initializeFingerRobotIfEnabled()
}

func TestRedirectTargetUsesLocalAddressInsteadOfHostHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://192.168.3.248/poc-path?x=1", nil)
	req.Host = "attacker.example"
	req = req.WithContext(context.WithValue(req.Context(), http.LocalAddrContextKey, &net.TCPAddr{
		IP:   net.ParseIP("192.168.3.248"),
		Port: 80,
	}))

	target := redirectTarget(req, ":8443")
	if target != "https://192.168.3.248:8443/poc-path?x=1" {
		t.Fatalf("unexpected redirect target: %s", target)
	}
}

func TestRedirectTargetUsesRequestHostWhenListeningOnUnspecifiedAddress(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://nanokvm.local/poc-path?x=1", nil)
	req.Host = "nanokvm.local"
	req = req.WithContext(context.WithValue(req.Context(), http.LocalAddrContextKey, &net.TCPAddr{
		IP:   net.IPv4zero,
		Port: 80,
	}))

	target := redirectTarget(req, ":443")
	if target != "https://nanokvm.local/poc-path?x=1" {
		t.Fatalf("unexpected redirect target: %s", target)
	}
}

func TestRedirectTargetStripsRequestHostPort(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://nanokvm.local:80/poc-path?x=1", nil)
	req.Host = "nanokvm.local:80"
	req = req.WithContext(context.WithValue(req.Context(), http.LocalAddrContextKey, &net.TCPAddr{
		IP:   net.IPv4zero,
		Port: 80,
	}))

	target := redirectTarget(req, ":8443")
	if target != "https://nanokvm.local:8443/poc-path?x=1" {
		t.Fatalf("unexpected redirect target: %s", target)
	}
}

func TestRedirectTargetFallsBackForInvalidRequestHost(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://nanokvm.local/poc-path?x=1", nil)
	req.Host = "attacker.example/path"
	req = req.WithContext(context.WithValue(req.Context(), http.LocalAddrContextKey, &net.TCPAddr{
		IP:   net.IPv4zero,
		Port: 80,
	}))

	target := redirectTarget(req, ":8443")
	if target != "https://localhost:8443/poc-path?x=1" {
		t.Fatalf("unexpected redirect target: %s", target)
	}
}

func TestRedirectTargetFallsBackForUntrustedExternalRequestHost(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://attacker.example/poc-path?x=1", nil)
	req.Host = "attacker.example"
	req = req.WithContext(context.WithValue(req.Context(), http.LocalAddrContextKey, &net.TCPAddr{
		IP:   net.IPv4zero,
		Port: 80,
	}))

	target := redirectTarget(req, ":8443")
	if target != "https://localhost:8443/poc-path?x=1" {
		t.Fatalf("unexpected redirect target: %s", target)
	}
}

func TestRedirectTargetAllowsTrustedExternalRequestHost(t *testing.T) {
	oldTrustedHosts := redirectTrustedHosts
	redirectTrustedHosts = func() []string {
		return []string{"kvm.example.com"}
	}
	t.Cleanup(func() {
		redirectTrustedHosts = oldTrustedHosts
	})

	req := httptest.NewRequest(http.MethodGet, "http://kvm.example.com/poc-path?x=1", nil)
	req.Host = "kvm.example.com"
	req = req.WithContext(context.WithValue(req.Context(), http.LocalAddrContextKey, &net.TCPAddr{
		IP:   net.IPv4zero,
		Port: 80,
	}))

	target := redirectTarget(req, ":443")
	if target != "https://kvm.example.com/poc-path?x=1" {
		t.Fatalf("unexpected redirect target: %s", target)
	}
}

func TestRedirectTargetFallsBackForUntrustedPublicIPv6Host(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://[2001:4860:4860::8888]/poc-path?x=1", nil)
	req.Host = "[2001:4860:4860::8888]"
	req = req.WithContext(context.WithValue(req.Context(), http.LocalAddrContextKey, &net.TCPAddr{
		IP:   net.IPv4zero,
		Port: 80,
	}))

	target := redirectTarget(req, ":8443")
	if target != "https://localhost:8443/poc-path?x=1" {
		t.Fatalf("unexpected redirect target: %s", target)
	}
}
