package netbird

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"NanoKVM-Server/proto"
	"github.com/gin-gonic/gin"
)

type mockDaemon struct {
	service       *Service
	running       atomic.Bool
	enabled       atomic.Bool
	connected     atomic.Bool
	up            atomic.Int32
	starts        atomic.Int32
	logins        atomic.Int32
	waiting       chan struct{}
	release       chan struct{}
	options       chan map[string]any
	configuration chan map[string]any
	configured    atomic.Bool
	logoutFails   atomic.Bool
}

func newMockDaemon(t *testing.T) *mockDaemon {
	t.Helper()
	m := &mockDaemon{waiting: make(chan struct{}, 8), release: make(chan struct{}), options: make(chan map[string]any, 8), configuration: make(chan map[string]any, 16)}
	s := NewService()
	dir := t.TempDir()
	s.paths = filePaths{binary: filepath.Join(dir, "netbird"), version: filepath.Join(dir, "version"), unit: filepath.Join(dir, "netbird.service"), state: filepath.Join(dir, "state")}
	for name, body := range map[string]string{s.paths.binary: "binary", s.paths.version: NetBirdVersion, s.paths.unit: "unit"} {
		if err := os.WriteFile(name, []byte(body), 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(s.paths.state, 0700); err != nil {
		t.Fatal(err)
	}
	m.running.Store(true)
	m.enabled.Store(true)
	s.system = systemService{
		running: func() (bool, error) { return m.running.Load(), nil },
		enabled: func() (bool, error) { return m.enabled.Load(), nil },
		start:   func() error { m.starts.Add(1); m.running.Store(true); m.enabled.Store(true); return nil },
		stop:    func() error { m.running.Store(false); m.enabled.Store(false); return nil },
		reload:  func() error { return nil },
	}
	s.tunAvailable = func() bool { return true }
	s.requiresChallenge = func(string) bool { return false }
	s.localEndpoint = func() (string, int) { return "https", 8443 }
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/daemon.DaemonService/Status":
			st, ip := "NeedsLogin", ""
			if m.connected.Load() {
				st = "Connected"
				ip = "100.91.0.25/16"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"status": st, "daemonVersion": NetBirdVersion, "fullStatus": map[string]any{"managementState": map[string]bool{"connected": m.connected.Load()}, "localPeerState": map[string]string{"IP": ip, "fqdn": "nanokvm.netbird.cloud"}}})
		case "/daemon.DaemonService/SetConfig":
			m.configuration <- body
			m.configured.Store(true)
			_, _ = w.Write([]byte(`{}`))
		case "/daemon.DaemonService/Login":
			if !m.configured.Load() {
				t.Error("login started before persistent configuration")
			}
			m.logins.Add(1)
			m.options <- body
			_, _ = w.Write([]byte(`{"needsSSOLogin":true,"verificationURI":"https://app.netbird.io/activate","verificationURIComplete":"https://app.netbird.io/activate?code=ABCD","userCode":"ABCD"}`))
		case "/daemon.DaemonService/WaitSSOLogin":
			if body["userCode"] != "ABCD" {
				t.Error("wrong user code")
			}
			m.waiting <- struct{}{}
			select {
			case <-r.Context().Done():
				return
			case <-m.release:
			}
			_, _ = w.Write([]byte(`{}`))
		case "/daemon.DaemonService/Up":
			m.up.Add(1)
			m.connected.Store(true)
			_, _ = w.Write([]byte(`{}`))
		case "/daemon.DaemonService/Logout":
			if m.logoutFails.Load() {
				w.WriteHeader(503)
				_, _ = w.Write([]byte(`{"code":14}`))
				return
			}
			m.connected.Store(false)
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Errorf("unexpected RPC %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	s.client = &http.Client{Transport: rewriteTransport{base: server.Client().Transport, target: server.URL}}
	m.service = s
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = s.cancelLogin(ctx)
		server.Close()
	})
	return m
}

type rewriteTransport struct {
	base   http.RoundTripper
	target string
}

func (r rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.URL.Scheme = "http"
	clone.URL.Host = strings.TrimPrefix(r.target, "http://")
	return r.base.RoundTrip(clone)
}

func invoke(t *testing.T, handler gin.HandlerFunc, body string) proto.Response {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	handler(c)
	var rsp proto.Response
	if err := json.Unmarshal(w.Body.Bytes(), &rsp); err != nil {
		t.Fatal(err)
	}
	return rsp
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatal("condition timed out")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestOfficialSSOCompletesAndBuildsAccessAddress(t *testing.T) {
	m := newMockDaemon(t)
	s := m.service
	if rsp := invoke(t, s.Login, `{"managementUrl":"https://untrusted.invalid","setupKey":"ignored"}`); rsp.Code != 0 {
		t.Fatal(rsp)
	}
	options := <-m.options
	if options["managementUrl"] != managementURL || options["adminURL"] != adminURL || options["isUnixDesktopClient"] != false {
		t.Fatal(options)
	}
	if _, ok := options["setupKey"]; ok {
		t.Fatal("unexpected setup key")
	}
	options = <-m.configuration
	if options["profileName"] != "default" || options["managementUrl"] != managementURL || options["adminURL"] != adminURL {
		t.Fatal(options)
	}
	for _, key := range []string{"disableDns", "disableClientRoutes", "disableServerRoutes"} {
		if options[key] != true {
			t.Fatal(key, options)
		}
	}
	for _, key := range []string{"serverSSHAllowed", "remoteJobsAllowed"} {
		if options[key] != false {
			t.Fatal(key, options)
		}
	}
	select {
	case <-m.waiting:
	case <-time.After(time.Second):
		t.Fatal("no SSO waiter")
	}
	status := s.status(context.Background())
	if status.State != "authorizing" || status.AuthURL == "" || status.UserCode != "ABCD" || status.Busy || status.AccessURL != "" {
		t.Fatal(status)
	}
	if rsp := invoke(t, s.Login, `{}`); rsp.Code != 0 {
		t.Fatal(rsp)
	}
	if m.logins.Load() != 1 {
		t.Fatal("duplicate authorization")
	}
	close(m.release)
	waitFor(t, func() bool {
		return !s.status(context.Background()).Busy && s.status(context.Background()).State == "connected"
	})
	status = s.status(context.Background())
	if status.AccessURL != "https://100.91.0.25:8443" || status.AuthURL != "" || status.IP != "100.91.0.25" {
		t.Fatal(status)
	}
	if rsp := invoke(t, s.Stop, `{}`); rsp.Code != 0 {
		t.Fatal(rsp)
	}
	status = s.status(context.Background())
	if status.State != "stopped" || status.AccessURL != "" {
		t.Fatal(status)
	}
}

func TestStartWaitsForLoginWithoutIssuingAnotherUp(t *testing.T) {
	m := newMockDaemon(t)
	m.running.Store(false)
	m.enabled.Store(false)
	if rsp := invoke(t, m.service.Start, `{}`); rsp.Code != 0 {
		t.Fatal(rsp)
	}
	status := m.service.status(context.Background())
	if status.State != "notLogin" || !status.Enabled || !status.Running || m.up.Load() != 0 || !m.configured.Load() {
		t.Fatal("starting an unauthenticated daemon should offer login", status)
	}
}

func TestLoginRequiresAnEnabledRunningServiceAndNeverStartsIt(t *testing.T) {
	for _, test := range []struct {
		name             string
		enabled, running bool
	}{
		{"disabled", false, false},
		{"runningWithoutEnable", false, true},
		{"enabledButStopped", true, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			m := newMockDaemon(t)
			m.enabled.Store(test.enabled)
			m.running.Store(test.running)
			if rsp := invoke(t, m.service.Login, `{}`); rsp.Code != -1 || rsp.Msg != "startRequired" {
				t.Fatal(rsp)
			}
			if m.starts.Load() != 0 || m.logins.Load() != 0 || m.configured.Load() || m.enabled.Load() != test.enabled || m.running.Load() != test.running {
				t.Fatal("login changed the disabled service")
			}
		})
	}
	m := newMockDaemon(t)
	m.enabled.Store(false)
	m.running.Store(false)
	if rsp := invoke(t, m.service.Start, `{}`); rsp.Code != 0 {
		t.Fatal(rsp)
	}
	if rsp := invoke(t, m.service.Login, `{}`); rsp.Code != 0 {
		t.Fatal(rsp)
	}
	if m.starts.Load() != 1 || m.logins.Load() != 1 {
		t.Fatal("login restarted the enabled service")
	}
}

func TestCancelStopRestartAndUninstallNeverCompleteOldLogin(t *testing.T) {
	for _, action := range []string{"cancel", "stop", "restart", "uninstall"} {
		t.Run(action, func(t *testing.T) {
			m := newMockDaemon(t)
			s := m.service
			identity := filepath.Join(s.paths.state, "default.json")
			if err := os.WriteFile(identity, []byte("retained identity"), 0600); err != nil {
				t.Fatal(err)
			}
			if rsp := invoke(t, s.Login, `{}`); rsp.Code != 0 {
				t.Fatal(rsp)
			}
			select {
			case <-m.waiting:
			case <-time.After(time.Second):
				t.Fatal("no waiter")
			}
			handlers := map[string]gin.HandlerFunc{"cancel": s.CancelLogin, "stop": s.Stop, "restart": s.Restart, "uninstall": s.Uninstall}
			if rsp := invoke(t, handlers[action], `{}`); rsp.Code != 0 {
				t.Fatal(rsp)
			}
			close(m.release)
			if m.up.Load() != 0 {
				t.Fatal("late login reconnected")
			}
			status := s.status(context.Background())
			if status.AuthURL != "" || status.UserCode != "" || status.State == "authorizing" {
				t.Fatal(status)
			}
			if data, err := os.ReadFile(identity); err != nil || string(data) != "retained identity" {
				t.Fatal("identity removed", err)
			}
		})
	}
}

func TestExpiredLoginAndLogoutFailureRemainRetryable(t *testing.T) {
	m := newMockDaemon(t)
	s := m.service
	s.loginTimeout = 50 * time.Millisecond
	if rsp := invoke(t, s.Login, `{}`); rsp.Code != 0 {
		t.Fatal(rsp)
	}
	waitFor(t, func() bool { return s.status(context.Background()).Error == "loginExpired" })
	status := s.status(context.Background())
	if status.State != "notLogin" || status.AuthURL != "" || m.up.Load() != 0 {
		t.Fatal(status)
	}
	m.connected.Store(true)
	m.logoutFails.Store(true)
	if rsp := invoke(t, s.Logout, `{}`); rsp.Code != -1 || rsp.Msg != "logoutFailed" {
		t.Fatal(rsp)
	}
	if !m.running.Load() || !m.connected.Load() {
		t.Fatal("cloud failure lost local session")
	}
	m.logoutFails.Store(false)
	if rsp := invoke(t, s.Logout, `{}`); rsp.Code != 0 {
		t.Fatal(rsp)
	}
	if m.running.Load() || m.enabled.Load() {
		t.Fatal("logout did not disable service")
	}
}

func TestProtectionGateRunsBeforeDaemonAndRejectsBusyMutations(t *testing.T) {
	m := newMockDaemon(t)
	s := m.service
	s.requiresChallenge = func(string) bool { return true }
	s.verifyChallenge = func(*gin.Context, string, string) error { return errors.New("denied") }
	for _, handler := range []gin.HandlerFunc{s.Install, s.Uninstall, s.Start, s.Stop, s.Restart, s.Login, s.CancelLogin, s.Logout} {
		if rsp := invoke(t, handler, `{}`); rsp.Code != -3 {
			t.Fatal(rsp)
		}
	}
	if m.logins.Load() != 0 || m.up.Load() != 0 || !m.running.Load() {
		t.Fatal("gate bypass")
	}
	s.operation.Store(1)
	if rsp := invoke(t, s.Login, `{}`); rsp.Msg != "busy" {
		t.Fatal(rsp)
	}
	s.operation.Store(0)
}

func TestAuthorizationURLAndPeerAddressValidation(t *testing.T) {
	for _, uri := range []string{"javascript:alert(1)", "http://app.netbird.io/", "https://user:password@app.netbird.io/", "https:///invalid"} {
		if _, err := authorizationURL(loginResponse{VerificationURIComplete: uri}); err == nil {
			t.Fatal(uri)
		}
	}
	for _, ip := range []string{"invalid", "127.0.0.1", "0.0.0.0", "::", "fe80::1%eth0"} {
		if peerAddress(ip) != "" {
			t.Fatal(ip)
		}
	}
	if got := accessURL("https", 443, "fd00::1"); got != "https://[fd00::1]" {
		t.Fatal(got)
	}
}
