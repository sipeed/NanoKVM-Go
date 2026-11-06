package cloudflare

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
)

func testService(t *testing.T) *Service {
	t.Helper()
	dir := t.TempDir()
	s := NewService()
	s.paths = filePaths{binary: filepath.Join(dir, "cloudflared"), unit: filepath.Join(dir, serviceName),
		version: filepath.Join(dir, "version"), config: filepath.Join(dir, "settings", "config.json"), token: filepath.Join(dir, "settings", "token")}
	for _, path := range []string{s.paths.binary, s.paths.unit, s.paths.version} {
		if err := os.WriteFile(path, []byte(CloudflaredVersion), 0700); err != nil {
			t.Fatal(err)
		}
	}
	s.system = systemService{
		running: func() (bool, error) { return false, nil }, enabled: func() (bool, error) { return false, nil },
		start: func() error { return nil }, stop: func() error { return nil }, restart: func() error { return nil }, reload: func() error { return nil },
	}
	s.localEndpoint = func() (string, int) { return "https", 9443 }
	s.requiresChallenge = func(string) bool { return false }
	return s
}

func request(t *testing.T, handler gin.HandlerFunc, body any) (proto.Response, string) {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(data))
	ctx.Request.Header.Set("Content-Type", "application/json")
	handler(ctx)
	var response proto.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	return response, recorder.Body.String()
}

func testToken(secret string) string {
	data, _ := json.Marshal(map[string]any{"a": "test-account", "t": "d1838763-3549-430f-a54e-692caf9f1653", "s": []byte(secret)})
	return base64.StdEncoding.EncodeToString(data)
}

func configure(t *testing.T, s *Service, secret string) {
	t.Helper()
	token := testToken(secret)
	rsp, _ := request(t, s.SetConfig, proto.SetCloudflareConfigReq{AccessURL: "https://kvm.example.com", Token: &token})
	if rsp.Code != 0 {
		t.Fatal(rsp)
	}
}

func TestConfigTokenPreservationAndPrivacy(t *testing.T) {
	s := testService(t)
	configure(t, s, "secret-one")
	rsp, _ := request(t, s.SetConfig, proto.SetCloudflareConfigReq{AccessURL: "https://second.example.com"})
	if rsp.Code != 0 {
		t.Fatal(rsp)
	}
	cfg, err := readConfig(s.paths)
	if err != nil || cfg.Token != testToken("secret-one") || cfg.AccessURL != "https://second.example.com" {
		t.Fatal("config not preserved", err)
	}
	for _, handler := range []gin.HandlerFunc{s.GetConfig, s.GetStatus} {
		_, raw := request(t, handler, nil)
		if strings.Contains(raw, cfg.Token) || strings.Contains(raw, "secret-one") {
			t.Fatal("response leaked token")
		}
	}
	_, raw := request(t, s.GetConfig, nil)
	if !strings.Contains(raw, `"tokenSet":true`) {
		t.Fatal("missing token flag")
	}
	data, _ := os.ReadFile(s.paths.config)
	if strings.Contains(string(data), cfg.Token) {
		t.Fatal("token duplicated into metadata")
	}
	for _, path := range []string{s.paths.config, s.paths.token} {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatal("file must be private", err)
		}
	}
	info, _ := os.Stat(filepath.Dir(s.paths.token))
	if info.Mode().Perm() != 0700 {
		t.Fatal("directory must be private")
	}
}

func TestInvalidInputAndConcurrentMutationsDoNotConsumeChallenge(t *testing.T) {
	s := testService(t)
	s.requiresChallenge = func(string) bool { return true }
	verified := 0
	s.verifyChallenge = func(*gin.Context, string, string) error { verified++; return errors.New("denied") }
	for _, value := range []string{"", "account-api-token", "cloudflared service install abc", "$(touch anything)", strings.Repeat("a", 4097)} {
		rsp, _ := request(t, s.SetConfig, proto.SetCloudflareConfigReq{Token: &value})
		if rsp.Code == 0 || verified != 0 {
			t.Fatal("invalid input reached challenge")
		}
	}
	token := testToken("valid")
	for _, address := range []string{"http://kvm.example.com", "https://user:pass@kvm.example.com", "https://kvm.example.com/path", "https://kvm.example.com?secret", "https://kvm.example.com/#fragment", "javascript:alert(1)", "https://kvm.example.com:99999"} {
		rsp, _ := request(t, s.SetConfig, proto.SetCloudflareConfigReq{AccessURL: address, Token: &token})
		if rsp.Code == 0 || verified != 0 {
			t.Fatalf("invalid address accepted: %s", address)
		}
	}
	s.operation.Store(1)
	rsp, _ := request(t, s.SetConfig, proto.SetCloudflareConfigReq{Token: &token})
	if rsp.Code == 0 || verified != 0 {
		t.Fatal("busy operation consumed challenge")
	}
	s.operation.Store(0)
	rsp, _ = request(t, s.SetConfig, proto.SetCloudflareConfigReq{Token: &token})
	if rsp.Code != -3 || verified != 1 {
		t.Fatal("missing challenge bypass")
	}
	if _, err := os.Stat(s.paths.token); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("unauthorized token write")
	}
}

func TestConfigRollbackRestoresTokenAndChangingURLDoesNotRestart(t *testing.T) {
	s := testService(t)
	configure(t, s, "old")
	s.system.running = func() (bool, error) { return true, nil }
	restarts := 0
	s.system.restart = func() error {
		restarts++
		if restarts == 1 {
			return errors.New("failed")
		}
		return nil
	}
	newToken := testToken("new")
	rsp, _ := request(t, s.SetConfig, proto.SetCloudflareConfigReq{AccessURL: "https://other.example.com", Token: &newToken})
	if rsp.Code == 0 || restarts != 2 {
		t.Fatal("restart failure did not roll back", rsp)
	}
	cfg, err := readConfig(s.paths)
	if err != nil || cfg.Token != testToken("old") || cfg.AccessURL != "https://kvm.example.com" {
		t.Fatal("rollback lost settings", err)
	}
	rsp, _ = request(t, s.SetConfig, proto.SetCloudflareConfigReq{AccessURL: "https://other.example.com"})
	if rsp.Code != 0 || restarts != 2 {
		t.Fatal("URL-only change interrupted tunnel")
	}
}

type roundTripper func(*http.Request) (*http.Response, error)

func (f roundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestStatusUsesEdgeReadinessAndNeverReturnsCredentials(t *testing.T) {
	s := testService(t)
	if s.status(context.Background()).State != "notConfigured" {
		t.Fatal("missing config state")
	}
	configure(t, s, "secret")
	if s.status(context.Background()).State != "stopped" {
		t.Fatal("stopped state")
	}
	s.system.enabled = func() (bool, error) { return true, nil }
	if s.status(context.Background()).State != "error" {
		t.Fatal("enabled but crashed service must not be connected")
	}
	s.system.running = func() (bool, error) { return true, nil }
	for _, tc := range []struct {
		code        int
		body, state string
	}{
		{503, `{"readyConnections":0}`, "connecting"}, {200, `{"readyConnections":0}`, "connecting"},
		{200, `{"readyConnections":2}`, "connected"}, {200, "invalid", "error"}, {302, "", "error"},
	} {
		s.client.Transport = roundTripper(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() != "http://"+metricsAddress+"/ready" || r.Header.Get("Authorization") != "" {
				t.Fatal("unsafe readiness request")
			}
			return &http.Response{StatusCode: tc.code, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
		})
		status := s.status(context.Background())
		if status.State != tc.state || status.Version != CloudflaredVersion || status.OriginURL != "https://127.0.0.1:9443" {
			t.Fatalf("status = %+v", status)
		}
	}
	s.operation.Store(2)
	if status := s.status(context.Background()); status.State != "installing" || !status.Busy {
		t.Fatal("installation should stay visible")
	}
}

func TestLifecycleAndHTTPSPrerequisite(t *testing.T) {
	s := testService(t)
	configure(t, s, "saved")
	starts, stops := 0, 0
	s.system.start = func() error { starts++; return nil }
	s.system.stop = func() error { stops++; return nil }
	s.localEndpoint = func() (string, int) { return "http", 80 }
	if rsp, _ := request(t, s.Start, nil); rsp.Code == 0 || starts != 0 {
		t.Fatal("HTTP origin accepted")
	}
	s.localEndpoint = func() (string, int) { return "https", 443 }
	for _, handler := range []gin.HandlerFunc{s.Start, s.Restart, s.Stop, s.Uninstall} {
		if rsp, _ := request(t, handler, nil); rsp.Code != 0 {
			t.Fatal(rsp)
		}
	}
	if starts != 2 || stops != 2 || s.installed() {
		t.Fatal("lifecycle calls incorrect")
	}
	if cfg, err := readConfig(s.paths); err != nil || cfg.Token != testToken("saved") {
		t.Fatal("uninstall lost configuration", err)
	}
}

func TestPinnedDownloadRejectsUnverifiedBytes(t *testing.T) {
	data := []byte("verified binary fixture")
	hash := sha256.Sum256(data)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(data) }))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "cloudflared")
	if err := downloadBinary(context.Background(), server.URL, path, strings.Repeat("0", 64)); err == nil {
		t.Fatal("checksum mismatch accepted")
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("bad download retained")
	}
	if err := downloadBinary(context.Background(), server.URL, path, hex.EncodeToString(hash[:])); err != nil {
		t.Fatal(err)
	}
	if err := downloadBinary(context.Background(), server.URL, path, hex.EncodeToString(hash[:])); err == nil {
		t.Fatal("existing file overwritten")
	}
}

func TestUnitAndArchitectureUsePinnedArmhfAsset(t *testing.T) {
	s := testService(t)
	unit := s.unitContents(modeNamed)
	for _, value := range []string{"--no-autoupdate", "--token-file " + s.paths.token, "--metrics 127.0.0.1:17401", "Restart=on-failure"} {
		if !strings.Contains(unit, value) {
			t.Fatal("missing service setting", value)
		}
	}
	if runtime.GOARCH == "arm" && packageArch() != "armhf" {
		t.Fatal("32-bit ARM must use armhf")
	}
	if _, ok := releaseChecksums["arm"]; ok {
		t.Fatal("soft-float asset must not be used")
	}
	if releaseChecksums["armhf"] != "236afbc341625f5af0a8c461577d9da8aa66e3fd123a5b34e801d18a8a5a4d95" || strings.Contains(releaseURL, "latest") {
		t.Fatal("release must be pinned")
	}
}
