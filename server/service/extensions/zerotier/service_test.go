package zerotier

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	s := NewService()
	dir := t.TempDir()
	s.paths = filePaths{binary: filepath.Join(dir, "zerotier-one"), unit: filepath.Join(dir, "service"), version: filepath.Join(dir, "version"), state: filepath.Join(dir, "state")}
	for _, path := range []string{s.paths.binary, s.paths.unit, s.paths.version} {
		if err := os.WriteFile(path, []byte(ZeroTierVersion), 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(s.paths.state, "networks.d"), 0700); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string]string{"authtoken.secret": "test-api-secret", "identity.public": "0123456789:0:public", "identity.secret": "test-private-identity"} {
		if err := os.WriteFile(filepath.Join(s.paths.state, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	s.system = systemService{running: func() (bool, error) { return true, nil }, enabled: func() (bool, error) { return true, nil }, start: func() error { return nil }, stop: func() error { return nil }, reload: func() error { return nil }}
	s.localEndpoint = func() (string, int) { return "https", 9443 }
	s.tunAvailable = func() bool { return true }
	s.requiresChallenge = func(string) bool { return false }
	return s
}

func invoke(t *testing.T, fn gin.HandlerFunc, value any, canceled bool) proto.Response {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(data))
	c.Request.Header.Set("Content-Type", "application/json")
	if canceled {
		ctx, cancel := context.WithCancel(c.Request.Context())
		cancel()
		c.Request = c.Request.WithContext(ctx)
	}
	fn(c)
	var rsp proto.Response
	if err := json.Unmarshal(w.Body.Bytes(), &rsp); err != nil {
		t.Fatal(err)
	}
	return rsp
}

func localAPI(t *testing.T, s *Service, fn http.HandlerFunc) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-ZT1-Auth") != "test-api-secret" {
			t.Error("local API missing authentication")
		}
		w.Header().Set("Content-Type", "application/json")
		fn(w, r)
	}))
	t.Cleanup(server.Close)
	s.apiURL = server.URL
}

func TestStatusSeparatesNodeOnlineMembershipAuthorizationAndAddresses(t *testing.T) {
	s := testService(t)
	online := true
	networks := []networkStatus{}
	localAPI(t, s, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/status" {
			_ = json.NewEncoder(w).Encode(nodeStatus{Address: "0123456789", Online: online, Version: ZeroTierVersion})
			return
		}
		_ = json.NewEncoder(w).Encode(networks)
	})
	if got := s.status(context.Background()); got.State != "noNetworks" {
		t.Fatal(got)
	}
	networks = []networkStatus{{ID: "0123456789abcdef", Status: "REQUESTING_CONFIGURATION"}}
	if got := s.status(context.Background()); got.State != "connecting" || len(got.Networks[0].AccessURLs) != 0 {
		t.Fatal(got)
	}
	networks[0].Status = "ACCESS_DENIED"
	if got := s.status(context.Background()); got.State != "connecting" || got.Networks[0].Status != "ACCESS_DENIED" {
		t.Fatal(got)
	}
	networks[0].Status = "OK"
	if got := s.status(context.Background()); got.State != "connecting" || got.Networks[0].Status != "WAITING_ADDRESS" {
		t.Fatal(got)
	}
	networks[0].Addresses = []string{"10.147.1.5/24", "fd00::5/64", "10.147.1.5/24", "127.0.0.1/8", "fe80::1/64", "javascript:alert(1)", "0.0.0.0/0"}
	got := s.status(context.Background())
	if got.State != "connected" || len(got.Networks[0].AccessURLs) != 2 || got.Networks[0].AccessURLs[0] != "https://10.147.1.5:9443" || got.Networks[0].AccessURLs[1] != "https://[fd00::5]:9443" {
		t.Fatal(got)
	}
	data, _ := json.Marshal(got)
	if strings.Contains(string(data), "test-api-secret") || strings.Contains(string(data), "test-private-identity") {
		t.Fatal("status leaked a credential")
	}
	online = false
	got = s.status(context.Background())
	if got.State != "offline" || len(got.Networks[0].AccessURLs) != 0 || len(got.Networks[0].Addresses) != 0 {
		t.Fatal(got)
	}
	online = true
	networks[0].Status = "PORT_ERROR"
	if got := s.status(context.Background()); got.State != "error" || got.Networks[0].Status != "PORT_ERROR" {
		t.Fatal(got)
	}
	// A ready network remains usable while another network awaits authorization.
	networks = append(networks, networkStatus{ID: "1123456789abcdef", Status: "OK", Addresses: []string{"10.3.0.2/24"}})
	if got := s.status(context.Background()); got.State != "connected" {
		t.Fatal(got)
	}
}

func TestJoinValidatesBeforeChallengeAndUsesRestrictedManagedSettings(t *testing.T) {
	s := testService(t)
	verified, calls := 0, 0
	s.requiresChallenge = func(string) bool { return true }
	s.verifyChallenge = func(_ *gin.Context, challenge, action string) error {
		verified++
		if challenge != "valid" || action != "zerotier_join" {
			return errors.New("denied")
		}
		return nil
	}
	localAPI(t, s, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/network/0123456789abcdef" || r.Method != http.MethodPost {
			t.Errorf("unexpected local API request: %s %s", r.Method, r.URL.Path)
		}
		var settings map[string]bool
		if json.NewDecoder(r.Body).Decode(&settings) != nil || len(settings) != 4 || !settings["allowManaged"] || settings["allowDefault"] || settings["allowGlobal"] || settings["allowDNS"] {
			t.Error("join changed the routing policy")
		}
		_, _ = w.Write([]byte(`{"status":"REQUESTING_CONFIGURATION"}`))
	})
	for _, id := range []string{"", "0000000000000000", "0123456789abcdeg", "../0123456789abcd", "0123456789abcdef/extra"} {
		if rsp := invoke(t, s.Join, proto.ZeroTierNetworkReq{NetworkID: id, Challenge: "valid"}, false); rsp.Code == 0 {
			t.Fatal("invalid ID accepted")
		}
	}
	if verified != 0 || calls != 0 {
		t.Fatal("invalid input consumed a challenge")
	}
	s.operation.Store(1)
	if rsp := invoke(t, s.Join, proto.ZeroTierNetworkReq{NetworkID: "0123456789abcdef", Challenge: "valid"}, false); rsp.Code == 0 || verified != 0 {
		t.Fatal("concurrent mutation consumed a challenge")
	}
	s.operation.Store(0)
	if rsp := invoke(t, s.Join, proto.ZeroTierNetworkReq{NetworkID: "0123456789abcdef"}, false); rsp.Code != -3 || calls != 0 {
		t.Fatal("join bypassed protection")
	}
	if rsp := invoke(t, s.Join, proto.ZeroTierNetworkReq{NetworkID: " 0123456789ABCDEF ", Challenge: "valid"}, true); rsp.Code != 0 || calls != 1 {
		t.Fatal("authorized join did not finish after caller disconnected", rsp)
	}
}

func TestOfflineLeavePreservesIdentityOtherNetworksAndUninstallRetainsMembership(t *testing.T) {
	s := testService(t)
	s.system.running = func() (bool, error) { return false, nil }
	s.system.enabled = func() (bool, error) { return false, nil }
	for _, name := range []string{"0123456789abcdef.conf", "0123456789abcdef.local.conf", "1123456789abcdef.conf"} {
		if err := os.WriteFile(filepath.Join(s.paths.state, "networks.d", name), []byte("saved"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if rsp := invoke(t, s.Join, proto.ZeroTierNetworkReq{NetworkID: "0123456789abcdef"}, false); rsp.Code == 0 {
		t.Fatal("joined with service stopped")
	}
	if got := s.status(context.Background()); got.State != "stopped" || len(got.Networks) != 2 || got.NodeID != "0123456789" {
		t.Fatal(got)
	}
	if err := s.uninstall(); err != nil {
		t.Fatal(err)
	}
	if got := s.status(context.Background()); got.State != "notInstalled" || len(got.Networks) != 2 {
		t.Fatal(got)
	}
	if rsp := invoke(t, s.Leave, proto.ZeroTierNetworkReq{NetworkID: "0123456789abcdef"}, false); rsp.Code != 0 {
		t.Fatal(rsp)
	}
	for _, name := range []string{"0123456789abcdef.conf", "0123456789abcdef.local.conf"} {
		if _, err := os.Stat(filepath.Join(s.paths.state, "networks.d", name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("membership not cleared", err)
		}
	}
	if _, err := os.Stat(filepath.Join(s.paths.state, "networks.d", "1123456789abcdef.conf")); err != nil {
		t.Fatal("another membership was removed", err)
	}
	data, err := os.ReadFile(filepath.Join(s.paths.state, "identity.secret"))
	if err != nil || string(data) != "test-private-identity" {
		t.Fatal("identity changed")
	}
}

func TestLeaveUsesDaemonAndFailureDoesNotDeleteMembership(t *testing.T) {
	s := testService(t)
	path := filepath.Join(s.paths.state, "networks.d", "0123456789abcdef.conf")
	if err := os.WriteFile(path, []byte("saved"), 0600); err != nil {
		t.Fatal(err)
	}
	localAPI(t, s, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/network/0123456789abcdef" {
			t.Error("unexpected leave request")
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	if rsp := invoke(t, s.Leave, proto.ZeroTierNetworkReq{NetworkID: "0123456789abcdef"}, false); rsp.Code == 0 {
		t.Fatal("failed API leave was accepted")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("failed leave removed membership")
	}
}

func TestServiceLifecycleAndVersionAreChecked(t *testing.T) {
	s := testService(t)
	starts, stops := 0, 0
	s.system.start = func() error { starts++; return nil }
	s.system.stop = func() error { stops++; return nil }
	s.tunAvailable = func() bool { return false }
	if rsp := invoke(t, s.Start, nil, false); rsp.Code == 0 || starts != 0 {
		t.Fatal("missing TUN accepted")
	}
	s.tunAvailable = func() bool { return true }
	for _, fn := range []gin.HandlerFunc{s.Start, s.Restart, s.Stop} {
		if rsp := invoke(t, fn, nil, false); rsp.Code != 0 {
			t.Fatal(rsp)
		}
	}
	if starts != 2 || stops != 1 {
		t.Fatal("lifecycle did not control service")
	}
	localAPI(t, s, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(nodeStatus{Address: "0123456789", Online: true, Version: "1.14.2"})
	})
	if got := s.status(context.Background()); got.Error != "versionMismatch" || got.State != "error" {
		t.Fatal(got)
	}
	if err := os.WriteFile(s.paths.version, []byte("1.14.2"), 0644); err != nil {
		t.Fatal(err)
	}
	if s.installed() {
		t.Fatal("wrong installed version accepted")
	}
}

func TestLocalAPIRejectsRedirectsMalformedAndOversizedResponses(t *testing.T) {
	s := testService(t)
	mode := 0
	localAPI(t, s, func(w http.ResponseWriter, r *http.Request) {
		switch mode {
		case 0:
			http.Redirect(w, r, "https://example.com/", http.StatusFound)
		case 1:
			_, _ = w.Write([]byte("not json"))
		case 2:
			_, _ = w.Write(bytes.Repeat([]byte(" "), (1<<20)+1))
		}
	})
	for mode = 0; mode < 3; mode++ {
		var result nodeStatus
		if s.request(context.Background(), http.MethodGet, "/status", nil, &result) == nil {
			t.Fatal("invalid response accepted", mode)
		}
	}
}
