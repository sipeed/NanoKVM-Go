package cloudflare

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"NanoKVM-Server/proto"
	authservice "NanoKVM-Server/service/auth"

	"github.com/gin-gonic/gin"
)

func TestModesPreserveNamedCredentialAndLegacyConfiguration(t *testing.T) {
	s := testService(t)
	configure(t, s, "keep-private")
	legacy := []byte(`{"accessUrl":"https://kvm.example.com"}`)
	if err := os.WriteFile(s.paths.config, legacy, 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := readConfig(s.paths)
	if err != nil || cfg.Mode != modeNamed {
		t.Fatal("legacy mode", err)
	}
	token, _ := os.ReadFile(s.paths.token)
	stops, starts := 0, 0
	s.system.stop = func() error { stops++; return nil }
	s.system.start = func() error { starts++; return nil }
	for _, mode := range []string{modeQuick, modeNamed} {
		if rsp, _ := request(t, s.SetMode, proto.SetCloudflareModeReq{Mode: mode}); rsp.Code != 0 {
			t.Fatal(rsp)
		}
		cfg, err = readConfig(s.paths)
		currentToken, _ := os.ReadFile(s.paths.token)
		if err != nil || cfg.Mode != mode || cfg.AccessURL != "https://kvm.example.com" || string(currentToken) != string(token) {
			t.Fatal("mode switching lost settings", err)
		}
		if rsp, _ := request(t, s.SetMode, proto.SetCloudflareModeReq{Mode: mode}); rsp.Code != 0 {
			t.Fatal(rsp)
		}
	}
	if stops != 2 || starts != 0 {
		t.Fatal("switching must stop once and never start implicitly")
	}
}

func TestQuickModeStartsWithoutTokenAndUsesOnlyLoopbackOrigin(t *testing.T) {
	s := testService(t)
	if rsp, _ := request(t, s.SetMode, proto.SetCloudflareModeReq{Mode: modeQuick}); rsp.Code != 0 {
		t.Fatal(rsp)
	}
	if rsp, _ := request(t, s.Start, nil); rsp.Code != 0 {
		t.Fatal(rsp)
	}
	unit, _ := os.ReadFile(s.paths.unit)
	for _, required := range []string{"--url https://127.0.0.1:9443", "--no-tls-verify", "--config /dev/null", "--protocol auto", "--no-autoupdate", "--metrics " + metricsAddress} {
		if !strings.Contains(string(unit), required) {
			t.Fatal("missing quick setting", required)
		}
	}
	if strings.Contains(string(unit), "--token-file") || strings.Contains(string(unit), "ConditionPathExists=") {
		t.Fatal("quick mode depends on token")
	}
	if _, err := os.Stat(s.paths.token); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("quick mode created a token file")
	}
	s.localEndpoint = func() (string, int) { return "http", 80 }
	if rsp, _ := request(t, s.Start, nil); rsp.Code == 0 {
		t.Fatal("HTTP origin accepted")
	}
}

func TestModeCanBeSelectedBeforeInstallation(t *testing.T) {
	s := testService(t)
	configure(t, s, "keep-after-uninstall")
	for _, path := range []string{s.paths.binary, s.paths.unit, s.paths.version} {
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
	}
	s.system.stop = func() error { t.Fatal("uninstalled service must not be stopped"); return nil }
	s.system.reload = func() error { t.Fatal("uninstalled service must not be reloaded"); return nil }
	if rsp, _ := request(t, s.SetMode, proto.SetCloudflareModeReq{Mode: modeQuick}); rsp.Code != 0 {
		t.Fatal(rsp)
	}
	cfg, err := readConfig(s.paths)
	if err != nil || cfg.Mode != modeQuick || cfg.Token != testToken("keep-after-uninstall") || cfg.AccessURL != "https://kvm.example.com" {
		t.Fatal("mode selection lost saved configuration", err)
	}
	if _, err := os.Stat(s.paths.unit); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("mode selection installed a service")
	}
	if rsp, _ := request(t, s.Start, nil); rsp.Code == 0 {
		t.Fatal("mode selection bypassed installation")
	}
}

func TestQuickStatusReadsLiveHostnameOnlyWhenReady(t *testing.T) {
	s := testService(t)
	configure(t, s, "named-secret")
	if rsp, _ := request(t, s.SetMode, proto.SetCloudflareModeReq{Mode: modeQuick}); rsp.Code != 0 {
		t.Fatal(rsp)
	}
	running := true
	s.system.running = func() (bool, error) { return running, nil }
	host := "first-example.trycloudflare.com"
	ready := `{"readyConnections":1}`
	s.client.Transport = roundTripper(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != metricsAddress || r.Header.Get("Authorization") != "" {
			t.Fatal("unsafe metrics request")
		}
		body := ready
		if r.URL.Path == "/quicktunnel" {
			body = `{"hostname":"` + host + `"}`
		} else if r.URL.Path != "/ready" {
			t.Fatal(r.URL.Path)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	for _, name := range []string{"first-example.trycloudflare.com", "https://second-example.trycloudflare.com"} {
		host = name
		status := s.status(context.Background())
		if status.Mode != modeQuick || status.State != "connected" || status.AccessURL != "https://"+strings.TrimPrefix(name, "https://") {
			t.Fatal(status)
		}
	}
	for _, invalid := range []string{"", "trycloudflare.com.evil.example", "evil.example", "user@okay.trycloudflare.com", "okay.trycloudflare.com/path", "okay.trycloudflare.com:443", "nested.name.trycloudflare.com"} {
		host = invalid
		status := s.status(context.Background())
		if status.AccessURL != "" || status.State != "connecting" {
			t.Fatal("invalid hostname shown", status)
		}
	}
	host = "valid.trycloudflare.com"
	ready = `{"readyConnections":0}`
	if status := s.status(context.Background()); status.AccessURL != "" {
		t.Fatal("unready address shown")
	}
	running = false
	if status := s.status(context.Background()); status.AccessURL != "" || status.State != "stopped" {
		t.Fatal("stopped URL survived", status)
	}
	cfg, _ := readConfig(s.paths)
	if cfg.AccessURL != "https://kvm.example.com" {
		t.Fatal("temporary URL replaced saved named URL")
	}
}

func TestModeSwitchProtectionStopFailureAndRollback(t *testing.T) {
	s := testService(t)
	configure(t, s, "preserved")
	s.requiresChallenge = func(action string) bool {
		if action != authservice.SensitiveActionCloudflareConfigure {
			t.Fatal(action)
		}
		return true
	}
	verified, stops := 0, 0
	s.verifyChallenge = func(*gin.Context, string, string) error { verified++; return errors.New("denied") }
	s.system.stop = func() error { stops++; return errors.New("failed") }
	if rsp, _ := request(t, s.SetMode, proto.SetCloudflareModeReq{Mode: "unknown"}); rsp.Code == 0 || verified != 0 {
		t.Fatal("invalid mode reached authorization")
	}
	if rsp, _ := request(t, s.SetMode, proto.SetCloudflareModeReq{Mode: modeQuick}); rsp.Code != -3 || stops != 0 {
		t.Fatal("authorization bypass")
	}
	s.requiresChallenge = func(string) bool { return false }
	if rsp, _ := request(t, s.SetMode, proto.SetCloudflareModeReq{Mode: modeQuick}); rsp.Code == 0 {
		t.Fatal("stop failure ignored")
	}
	if cfg, _ := readConfig(s.paths); cfg.Mode != modeNamed {
		t.Fatal("mode changed despite stop failure")
	}
	s.system.stop = func() error { return nil }
	reloads := 0
	s.system.reload = func() error {
		reloads++
		if reloads == 1 {
			return errors.New("failed")
		}
		return nil
	}
	unit, _ := os.ReadFile(s.paths.unit)
	if rsp, _ := request(t, s.SetMode, proto.SetCloudflareModeReq{Mode: modeQuick}); rsp.Code == 0 {
		t.Fatal("reload failure ignored")
	}
	after, _ := os.ReadFile(s.paths.unit)
	cfg, err := readConfig(s.paths)
	if err != nil || cfg.Mode != modeNamed || cfg.Token != testToken("preserved") || string(after) != string(unit) {
		t.Fatal("rollback failed", err)
	}
}
