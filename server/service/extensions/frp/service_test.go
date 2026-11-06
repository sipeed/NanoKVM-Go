package frp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
)

func testService(t *testing.T) *Service {
	t.Helper()
	dir := t.TempDir()
	s := NewService()
	s.paths = filePaths{binary: filepath.Join(dir, "frpc"), unit: filepath.Join(dir, serviceName), config: filepath.Join(dir, "config", "frpc.toml")}
	for _, path := range []string{s.paths.binary, s.paths.unit} {
		if err := os.WriteFile(path, []byte("fixture"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	s.system = systemService{
		running: func() (bool, error) { return false, nil },
		enabled: func() (bool, error) { return false, nil },
		start:   func() error { return nil }, stop: func() error { return nil },
		restart: func() error { return nil }, reload: func() error { return nil },
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

func validForm() proto.SetFRPConfigReq {
	token := "a-sensitive-test-token"
	return proto.SetFRPConfigReq{ServerAddr: "frp.example.com", ServerPort: 7000, RemotePort: 8443, Token: &token}
}

func TestConfigPreservesTokenAndProxyIdentityWithoutDisclosingSecrets(t *testing.T) {
	s := testService(t)
	first := validForm()
	rsp, _ := request(t, s.SetConfig, first)
	if rsp.Code != 0 {
		t.Fatal(rsp)
	}
	old, err := readConfig(s.paths.config)
	if err != nil {
		t.Fatal(err)
	}
	if old.Proxies[0].LocalPort != 9443 || !old.Transport.TLS.Enable || !old.LoginFailExit {
		t.Fatalf("wrong generated config: %+v", old.Proxies[0])
	}
	first.Token = nil
	first.RemotePort = 8444
	rsp, _ = request(t, s.SetConfig, first)
	if rsp.Code != 0 {
		t.Fatal(rsp)
	}
	current, err := readConfig(s.paths.config)
	if err != nil {
		t.Fatal(err)
	}
	if current.Auth.Token != old.Auth.Token || current.WebServer.Password != old.WebServer.Password || current.Proxies[0].Name != old.Proxies[0].Name {
		t.Fatal("save replaced stored credentials or proxy identity")
	}
	rsp, raw := request(t, s.GetConfig, nil)
	if rsp.Code != 0 || strings.Contains(raw, old.Auth.Token) || strings.Contains(raw, old.WebServer.Password) || !strings.Contains(raw, `"tokenSet":true`) {
		t.Fatalf("unsafe config response: %s", raw)
	}
	info, err := os.Stat(s.paths.config)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("configuration must be private", err)
	}
	empty := ""
	first.Token = &empty
	rsp, _ = request(t, s.SetConfig, first)
	if rsp.Code != 0 {
		t.Fatal(rsp)
	}
	current, _ = readConfig(s.paths.config)
	if current.Auth.Token != "" {
		t.Fatal("explicit empty token did not clear it")
	}
}

func TestValidationHappensBeforeChallengeAndMutation(t *testing.T) {
	for _, mutate := range []func(*proto.SetFRPConfigReq){
		func(r *proto.SetFRPConfigReq) { r.ServerAddr = "https://example.com:7000/path" },
		func(r *proto.SetFRPConfigReq) { r.ServerAddr = "example.com\n[[proxies]]" },
		func(r *proto.SetFRPConfigReq) { r.ServerAddr = "-invalid.example" },
		func(r *proto.SetFRPConfigReq) { r.ServerPort = 0 },
		func(r *proto.SetFRPConfigReq) { r.RemotePort = 65536 },
		func(r *proto.SetFRPConfigReq) { r.RemotePort = r.ServerPort },
		func(r *proto.SetFRPConfigReq) { token := "bad\ntoken"; r.Token = &token },
		func(r *proto.SetFRPConfigReq) { token := "{{ .Envs.SECRET }}"; r.Token = &token },
	} {
		s := testService(t)
		s.requiresChallenge = func(string) bool { t.Fatal("invalid form consumed a challenge"); return true }
		form := validForm()
		mutate(&form)
		rsp, _ := request(t, s.SetConfig, form)
		if rsp.Code != -1 {
			t.Fatal(rsp)
		}
		if _, err := os.Stat(s.paths.config); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("invalid form wrote config")
		}
	}
	form := validForm()
	form.ServerAddr = "[2001:db8::1]"
	if err := validateForm(&form); err != nil || form.ServerAddr != "2001:db8::1" {
		t.Fatal(form.ServerAddr, err)
	}
	cfg, _ := makeConfig(form, nil, 443)
	if cfg.accessURL("https") != "https://[2001:db8::1]:8443" {
		t.Fatal(cfg.accessURL("https"))
	}
}

func TestRejectedConfigurationAndFailedRestartKeepPreviousConfiguration(t *testing.T) {
	s := testService(t)
	if rsp, _ := request(t, s.SetConfig, validForm()); rsp.Code != 0 {
		t.Fatal(rsp)
	}
	before, _ := os.ReadFile(s.paths.config)
	form := validForm()
	form.RemotePort = 0
	if rsp, _ := request(t, s.SetConfig, form); rsp.Code == 0 {
		t.Fatal("accepted rejected config")
	}
	after, _ := os.ReadFile(s.paths.config)
	if !bytes.Equal(before, after) {
		t.Fatal("verification failure changed configuration")
	}
	form.RemotePort = 8444
	s.system.enabled = func() (bool, error) { return true, nil }
	restarts := 0
	s.system.restart = func() error {
		restarts++
		if restarts == 1 {
			return errors.New("restart failed")
		}
		return nil
	}
	if rsp, _ := request(t, s.SetConfig, form); rsp.Code == 0 || rsp.Msg != "restartFailed" {
		t.Fatal(rsp)
	}
	after, _ = os.ReadFile(s.paths.config)
	if !bytes.Equal(before, after) || restarts != 2 {
		t.Fatal("previous configuration was not restored and restarted")
	}
}

func TestMutationsAreSerializedAndRequireTheirOwnChallenge(t *testing.T) {
	s := testService(t)
	s.operation.Store(2)
	if rsp, _ := request(t, s.Stop, nil); rsp.Msg != "busy" {
		t.Fatal(rsp)
	}
	if status := s.status(context.Background()); status.State != "installing" || !status.Busy {
		t.Fatal(status)
	}
	s.operation.Store(0)
	s.requiresChallenge = func(string) bool { return true }
	s.verifyChallenge = func(*gin.Context, string, string) error { return errors.New("missing challenge") }
	if rsp, _ := request(t, s.SetConfig, validForm()); rsp.Code != -3 {
		t.Fatal(rsp)
	}
	if s.operation.Load() != 0 {
		t.Fatal("authorization failure left service busy")
	}
	if _, err := os.Stat(s.paths.config); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("unauthorized request wrote config")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return fn(req) }

func TestStatusRequiresExpectedProxyToBeRunning(t *testing.T) {
	s := testService(t)
	if status := s.status(context.Background()); status.State != "notConfigured" {
		t.Fatal(status)
	}
	if rsp, _ := request(t, s.SetConfig, validForm()); rsp.Code != 0 {
		t.Fatal(rsp)
	}
	cfg, _ := readConfig(s.paths.config)
	if status := s.status(context.Background()); status.State != "stopped" {
		t.Fatal(status)
	}
	s.system.enabled = func() (bool, error) { return true, nil }
	if status := s.status(context.Background()); status.State != "error" || status.Error != "connectionFailed" {
		t.Fatal(status)
	}
	s.system.running = func() (bool, error) { return true, nil }
	for _, tc := range []struct{ body, state, errorCode string }{
		{`{}`, "connecting", ""},
		{`{"tcp":[{"name":"other","status":"running"}]}`, "connecting", ""},
		{`{"tcp":[{"name":"` + cfg.Proxies[0].Name + `","status":"running"}]}`, "connected", ""},
		{`{"tcp":[{"name":"` + cfg.Proxies[0].Name + `","status":"start error","err":"port denied secret-value"}]}`, "error", "proxyFailed"},
		{`{"tcp":[{"name":"` + cfg.Proxies[0].Name + `","status":"check failed"}]}`, "error", "proxyFailed"},
		{`not json`, "error", "serviceFailed"},
	} {
		s.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			user, password, ok := req.BasicAuth()
			if !ok || user != cfg.WebServer.User || password != cfg.WebServer.Password || req.URL.Hostname() != "127.0.0.1" || req.URL.Path != "/api/status" {
				t.Fatal("invalid admin API request")
			}
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
		})}
		status := s.status(context.Background())
		if status.State != tc.state || status.Error != tc.errorCode {
			t.Fatalf("body %s: %+v", tc.body, status)
		}
	}
}

func TestStartUsesCurrentLocalPortAndUninstallPreservesSettings(t *testing.T) {
	s := testService(t)
	if rsp, _ := request(t, s.SetConfig, validForm()); rsp.Code != 0 {
		t.Fatal(rsp)
	}
	s.localEndpoint = func() (string, int) { return "http", 8080 }
	started := false
	s.system.start = func() error { started = true; return nil }
	if err := s.start(context.Background()); err != nil || !started {
		t.Fatal(err)
	}
	cfg, _ := readConfig(s.paths.config)
	if cfg.Proxies[0].LocalPort != 8080 {
		t.Fatal("start reused stale local port")
	}
	before, _ := os.ReadFile(s.paths.config)
	stopped := false
	s.system.stop = func() error { stopped = true; return nil }
	if err := s.uninstall(); err != nil || !stopped {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(s.paths.config)
	if s.installed() || !bytes.Equal(before, after) {
		t.Fatal("uninstall lost settings or left executable installed")
	}
}
