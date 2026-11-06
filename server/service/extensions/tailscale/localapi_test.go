package tailscale

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The fixtures mirror what tailscale 1.102.2 returns on the target device:
// GET /localapi/v0/status matches `tailscale status --json` and
// GET /localapi/v0/prefs matches `tailscale debug prefs`.
const (
	deviceStatusJSON = `{
  "Version": "1.102.2",
  "TUN": true,
  "BackendState": "Running",
  "AuthURL": "",
  "Self": {
    "ID": "n123456CNTRL",
    "HostName": "nanokvm",
    "DNSName": "nanokvm.tailnet-example.ts.net.",
    "OS": "linux",
    "TailscaleIPs": ["100.96.42.41", "fd7a:115c:a1e0::f231:2a2a"],
    "Online": true
  },
  "CurrentTailnet": {
    "Name": "example.com",
    "MagicDNSSuffix": "tailnet-example.ts.net",
    "MagicDNSEnabled": true
  },
  "Peer": {}
}`

	devicePrefsJSON = `{
  "RouteAll": false,
  "ExitNodeID": "",
  "ExitNodeIP": "",
  "CorpDNS": true,
  "RunSSH": false,
  "WantRunning": true,
  "LoggedOut": false,
  "AdvertiseRoutes": ["192.168.0.0/22", "10.0.0.0/24"],
  "Hostname": "nanokvm"
}`
)

func dialToTCP(address string) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, _, _ string) (net.Conn, error) {
		var dialer net.Dialer
		return dialer.DialContext(ctx, "tcp", address)
	}
}

func serveUnixSocket(t *testing.T, socketPath string, handler http.Handler) {
	t.Helper()

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen on %s: %v", socketPath, err)
	}
	server := &http.Server{Handler: handler}
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(func() {
		_ = server.Close()
	})
}

func TestLocalAPIStatusAndPrefsUseInjectedDial(t *testing.T) {
	var (
		requests []string
		hosts    []string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.RequestURI())
		hosts = append(hosts, r.Host)

		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case tailscaleLocalAPIBasePath + "/status":
			_, _ = io.WriteString(w, deviceStatusJSON)
		case tailscaleLocalAPIBasePath + "/prefs":
			_, _ = io.WriteString(w, devicePrefsJSON)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewLocalClient()
	client.localAPI = newLocalAPIClientWithDial(dialToTCP(server.Listener.Addr().String()))

	status, err := client.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status.BackendState != "Running" {
		t.Fatalf("BackendState = %q, want %q", status.BackendState, "Running")
	}
	if status.Self.HostName != "nanokvm" {
		t.Fatalf("Self.HostName = %q, want %q", status.Self.HostName, "nanokvm")
	}
	if want := []string{"100.96.42.41", "fd7a:115c:a1e0::f231:2a2a"}; !reflect.DeepEqual(status.Self.TailscaleIPs, want) {
		t.Fatalf("Self.TailscaleIPs = %#v, want %#v", status.Self.TailscaleIPs, want)
	}
	if status.CurrentTailnet.Name != "example.com" {
		t.Fatalf("CurrentTailnet.Name = %q, want %q", status.CurrentTailnet.Name, "example.com")
	}

	prefs, err := client.Prefs()
	if err != nil {
		t.Fatalf("Prefs() error = %v", err)
	}
	if prefs.LoggedOut {
		t.Fatal("LoggedOut = true, want false")
	}
	if want := []string{"192.168.0.0/22", "10.0.0.0/24"}; !reflect.DeepEqual(prefs.AdvertiseRoutes, want) {
		t.Fatalf("AdvertiseRoutes = %#v, want %#v", prefs.AdvertiseRoutes, want)
	}

	wantRequests := []string{
		"GET " + tailscaleLocalAPIBasePath + "/status",
		"GET " + tailscaleLocalAPIBasePath + "/prefs",
	}
	if !reflect.DeepEqual(requests, wantRequests) {
		t.Fatalf("requests = %#v, want %#v", requests, wantRequests)
	}
	for _, host := range hosts {
		if host != tailscaleLocalAPIHost {
			t.Fatalf("request Host = %q, want %q", host, tailscaleLocalAPIHost)
		}
	}
}

func TestLocalAPIUpAndDownPatchMaskedPrefs(t *testing.T) {
	var (
		methods []string
		hosts   []string
		bodies  []map[string]any
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		hosts = append(hosts, r.Host)

		if want := tailscaleLocalAPIBasePath + tailscaleLocalAPIPrefsPath; r.URL.Path != want {
			t.Errorf("request path = %q, want %q", r.URL.Path, want)
		}
		if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", contentType)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		bodies = append(bodies, body)
		// PATCH /prefs answers 200 with an empty body.
	}))
	defer server.Close()

	client := NewLocalClient()
	client.localAPI = newLocalAPIClientWithDial(dialToTCP(server.Listener.Addr().String()))

	if err := client.Up(); err != nil {
		t.Fatalf("Up() error = %v", err)
	}
	if err := client.Down(); err != nil {
		t.Fatalf("Down() error = %v", err)
	}

	wantBodies := []map[string]any{
		{"WantRunning": true, "WantRunningSet": true},
		{"WantRunning": false, "WantRunningSet": true},
	}
	if !reflect.DeepEqual(bodies, wantBodies) {
		t.Fatalf("request bodies = %#v, want %#v", bodies, wantBodies)
	}
	wantMethods := []string{http.MethodPatch, http.MethodPatch}
	if !reflect.DeepEqual(methods, wantMethods) {
		t.Fatalf("request methods = %#v, want %#v", methods, wantMethods)
	}
	for _, host := range hosts {
		if host != tailscaleLocalAPIHost {
			t.Fatalf("request Host = %q, want %q", host, tailscaleLocalAPIHost)
		}
	}
}

// TestLocalAPIPrefsPatchSerializesMasks pins the wire shape of the masked
// prefs PATCH body. Each written field must carry its Set companion (the
// daemon ignores unmasked fields), an empty route list must serialize as []
// instead of null, and a zero-value patch must not write any field at all.
func TestLocalAPIPrefsPatchSerializesMasks(t *testing.T) {
	wantRunning := true
	wantRunningOff := false
	corpDNS := false
	emptyRoutes := []string{}

	tests := []struct {
		name  string
		patch localAPIPrefsPatch
		want  string
	}{
		{
			name:  "want running on",
			patch: localAPIPrefsPatch{WantRunning: &wantRunning, WantRunningSet: true},
			want:  `{"WantRunning":true,"WantRunningSet":true}`,
		},
		{
			name:  "want running off",
			patch: localAPIPrefsPatch{WantRunning: &wantRunningOff, WantRunningSet: true},
			want:  `{"WantRunning":false,"WantRunningSet":true}`,
		},
		{
			name:  "corp dns off",
			patch: localAPIPrefsPatch{CorpDNS: &corpDNS, CorpDNSSet: true},
			want:  `{"CorpDNS":false,"CorpDNSSet":true}`,
		},
		{
			name:  "empty routes withdraw everything",
			patch: localAPIPrefsPatch{AdvertiseRoutes: &emptyRoutes, AdvertiseRoutesSet: true},
			want:  `{"AdvertiseRoutes":[],"AdvertiseRoutesSet":true}`,
		},
		{
			name:  "zero value patch writes nothing",
			patch: localAPIPrefsPatch{},
			want:  `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := json.Marshal(tt.patch)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}
			if got := string(encoded); got != tt.want {
				t.Fatalf("localAPIPrefsPatch = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestLocalAPIUpReportsMaskedPrefsPatchFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"error":"invalid host"}`)
	}))
	defer server.Close()

	client := NewLocalClient()
	client.localAPI = newLocalAPIClientWithDial(dialToTCP(server.Listener.Addr().String()))

	err := client.Up()
	if err == nil {
		t.Fatal("Up() error = nil, want error")
	}
	for _, want := range []string{"tailscale up failed", "403", "invalid host"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("Up() error = %v, want it to contain %q", err, want)
		}
	}
}

func TestLocalAPISetRoutingPatchesAdvertiseRoutes(t *testing.T) {
	type capturedRequest struct {
		method      string
		path        string
		host        string
		contentType string
		body        string
	}

	var requests []capturedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		requests = append(requests, capturedRequest{
			method:      r.Method,
			path:        r.URL.Path,
			host:        r.Host,
			contentType: r.Header.Get("Content-Type"),
			body:        string(body),
		})
		// PATCH /prefs answers 200 with an empty body.
	}))
	defer server.Close()

	client := NewLocalClient()
	client.localAPI = newLocalAPIClientWithDial(dialToTCP(server.Listener.Addr().String()))

	calls := []struct {
		name     string
		exitNode bool
		routes   []string
		wantBody string
	}{
		{
			name:     "exit node on",
			exitNode: true,
			routes:   []string{"192.168.0.0/22", "10.0.0.0/24", "192.168.0.0/22"},
			wantBody: `{"AdvertiseRoutes":["0.0.0.0/0","::/0","10.0.0.0/24","192.168.0.0/22"],"AdvertiseRoutesSet":true}`,
		},
		{
			name:     "exit node off",
			exitNode: false,
			routes:   []string{"192.168.0.0/22", "10.0.0.0/24"},
			wantBody: `{"AdvertiseRoutes":["10.0.0.0/24","192.168.0.0/22"],"AdvertiseRoutesSet":true}`,
		},
		{
			name:     "empty routes withdraw everything",
			exitNode: false,
			routes:   nil,
			wantBody: `{"AdvertiseRoutes":[],"AdvertiseRoutesSet":true}`,
		},
	}
	for _, call := range calls {
		if err := client.SetRouting(call.exitNode, call.routes); err != nil {
			t.Fatalf("SetRouting(%v, %#v) error = %v", call.exitNode, call.routes, err)
		}
		if got := requests[len(requests)-1].body; got != call.wantBody {
			t.Fatalf("%s: request body = %s, want %s", call.name, got, call.wantBody)
		}
	}

	if len(requests) != len(calls) {
		t.Fatalf("requests = %d, want %d", len(requests), len(calls))
	}
	for i, request := range requests {
		if request.method != http.MethodPatch {
			t.Fatalf("request %d method = %q, want %q", i, request.method, http.MethodPatch)
		}
		if want := tailscaleLocalAPIBasePath + tailscaleLocalAPIPrefsPath; request.path != want {
			t.Fatalf("request %d path = %q, want %q", i, request.path, want)
		}
		if request.host != tailscaleLocalAPIHost {
			t.Fatalf("request %d Host = %q, want %q", i, request.host, tailscaleLocalAPIHost)
		}
		if request.contentType != "application/json" {
			t.Fatalf("request %d Content-Type = %q, want application/json", i, request.contentType)
		}
	}

	// Turning the exit node off has to withdraw the default routes instead of
	// leaving them behind in the patched prefs.
	for _, route := range []string{exitNodeRouteIPv4, exitNodeRouteIPv6} {
		if strings.Contains(requests[1].body, route) {
			t.Fatalf("exit node off request body = %s, want no %s", requests[1].body, route)
		}
	}
}

func TestLocalAPISetRoutingRejectsInvalidRoutesWithoutRequest(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewLocalClient()
	client.localAPI = newLocalAPIClientWithDial(dialToTCP(server.Listener.Addr().String()))

	for _, route := range []string{"192.168.3.16/22", exitNodeRouteIPv4, exitNodeRouteIPv6, "not-a-prefix"} {
		err := client.SetRouting(false, []string{"10.0.0.0/24", route})
		if err == nil {
			t.Fatalf("SetRouting with route %q error = nil, want error", route)
		}
		if !strings.Contains(err.Error(), "tailscale set routing failed") {
			t.Fatalf("SetRouting with route %q error = %v, want the routing failure prefix", route, err)
		}
		if !strings.Contains(err.Error(), route) {
			t.Fatalf("SetRouting with route %q error = %v, want it to mention the route", route, err)
		}
		if requests != 0 {
			t.Fatalf("route %q sent %d requests, want 0", route, requests)
		}
	}
}

func TestLocalAPISetRoutingReportsDaemonFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("request method = %q, want %q", r.Method, http.MethodPatch)
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"Error":"invalid advertised routes"}`)
	}))
	defer server.Close()

	client := NewLocalClient()
	client.localAPI = newLocalAPIClientWithDial(dialToTCP(server.Listener.Addr().String()))

	err := client.SetRouting(false, []string{"10.0.0.0/24"})
	if err == nil {
		t.Fatal("SetRouting() error = nil, want error")
	}
	for _, want := range []string{"tailscale set routing failed", "400", "invalid advertised routes"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("SetRouting() error = %v, want it to contain %q", err, want)
		}
	}
}

func TestLocalAPISetRoutingPropagatesContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := NewLocalClientWithContext(ctx)
	client.localAPI = newLocalAPIClientWithDial(dialToTCP(server.Listener.Addr().String()))

	err := client.SetRouting(true, []string{"10.0.0.0/24"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("SetRouting() error = %v, want %v", err, context.Canceled)
	}
	if !strings.Contains(err.Error(), "tailscale set routing failed") {
		t.Fatalf("SetRouting() error = %v, want the routing failure prefix", err)
	}
}

func TestLocalAPILogoutPostsToLogoutEndpoint(t *testing.T) {
	type capturedRequest struct {
		method      string
		path        string
		host        string
		contentType string
		body        string
	}

	var requests []capturedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		requests = append(requests, capturedRequest{
			method:      r.Method,
			path:        r.URL.Path,
			host:        r.Host,
			contentType: r.Header.Get("Content-Type"),
			body:        string(body),
		})
		// POST /logout answers 204 with no body.
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewLocalClient()
	client.localAPI = newLocalAPIClientWithDial(dialToTCP(server.Listener.Addr().String()))

	if err := client.Logout(); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}

	if len(requests) != 1 {
		t.Fatalf("requests = %d, want 1", len(requests))
	}
	request := requests[0]
	if request.method != http.MethodPost {
		t.Fatalf("request method = %q, want %q", request.method, http.MethodPost)
	}
	if want := tailscaleLocalAPIBasePath + tailscaleLocalAPILogoutPath; request.path != want {
		t.Fatalf("request path = %q, want %q", request.path, want)
	}
	if request.host != tailscaleLocalAPIHost {
		t.Fatalf("request Host = %q, want %q", request.host, tailscaleLocalAPIHost)
	}
	if request.body != "" {
		t.Fatalf("request body = %q, want an empty body", request.body)
	}
	if request.contentType != "" {
		t.Fatalf("request Content-Type = %q, want it to be unset", request.contentType)
	}
}

func TestLocalAPILogoutReportsDaemonFailure(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		errCause string
	}{
		{
			name:     "forbidden",
			status:   http.StatusForbidden,
			body:     `{"Error":"logout is not permitted"}`,
			errCause: "logout is not permitted",
		},
		{
			name:     "server error",
			status:   http.StatusInternalServerError,
			body:     `{"Error":"backend failed"}`,
			errCause: "backend failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case tailscaleLocalAPIBasePath + tailscaleLocalAPILogoutPath:
					if r.Method != http.MethodPost {
						t.Errorf("logout request method = %q, want %q", r.Method, http.MethodPost)
					}
					w.WriteHeader(tt.status)
					_, _ = io.WriteString(w, tt.body)
				case tailscaleLocalAPIBasePath + tailscaleLocalAPIPrefsPath:
					if r.Method != http.MethodGet {
						t.Errorf("prefs request method = %q, want %q", r.Method, http.MethodGet)
					}
					_, _ = io.WriteString(w, `{"LoggedOut":false}`)
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			client := NewLocalClient()
			client.localAPI = newLocalAPIClientWithDial(dialToTCP(server.Listener.Addr().String()))

			err := client.Logout()
			if err == nil {
				t.Fatal("Logout() error = nil, want error")
			}
			for _, want := range []string{"tailscale logout failed", strconv.Itoa(tt.status), tt.errCause} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("Logout() error = %v, want it to contain %q", err, want)
				}
			}
		})
	}
}

func TestLocalAPILogoutReconcilesTimeoutFromLoggedOutPrefs(t *testing.T) {
	requests := make(chan string, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Method + " " + r.URL.Path
		switch r.URL.Path {
		case tailscaleLocalAPIBasePath + tailscaleLocalAPILogoutPath:
			<-r.Context().Done()
		case tailscaleLocalAPIBasePath + tailscaleLocalAPIPrefsPath:
			_, _ = io.WriteString(w, `{"LoggedOut":true}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewLocalClient()
	client.commandTimeout = 100 * time.Millisecond
	client.localAPI = newLocalAPIClientWithDial(dialToTCP(server.Listener.Addr().String()))

	started := time.Now()
	if err := client.Logout(); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("Logout() reconciliation took %s", elapsed)
	}

	want := []string{
		http.MethodPost + " " + tailscaleLocalAPIBasePath + tailscaleLocalAPILogoutPath,
		http.MethodGet + " " + tailscaleLocalAPIBasePath + tailscaleLocalAPIPrefsPath,
	}
	got := []string{<-requests, <-requests}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("requests = %v, want %v", got, want)
	}
}

func TestLocalAPILogoutReconcilesNoNodeKeyFromLoggedOutPrefs(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		switch r.URL.Path {
		case tailscaleLocalAPIBasePath + tailscaleLocalAPILogoutPath:
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, `no nodekey to log out`)
		case tailscaleLocalAPIBasePath + tailscaleLocalAPIPrefsPath:
			_, _ = io.WriteString(w, `{"LoggedOut":true}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewLocalClient()
	client.localAPI = newLocalAPIClientWithDial(dialToTCP(server.Listener.Addr().String()))

	if err := client.Logout(); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	want := []string{
		http.MethodPost + " " + tailscaleLocalAPIBasePath + tailscaleLocalAPILogoutPath,
		http.MethodGet + " " + tailscaleLocalAPIBasePath + tailscaleLocalAPIPrefsPath,
	}
	if !reflect.DeepEqual(requests, want) {
		t.Fatalf("requests = %v, want %v", requests, want)
	}
}

func TestLocalAPILogoutPropagatesContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := NewLocalClientWithContext(ctx)
	client.localAPI = newLocalAPIClientWithDial(dialToTCP(server.Listener.Addr().String()))

	err := client.Logout()
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Logout() error = %v, want %v", err, context.Canceled)
	}
	if !strings.Contains(err.Error(), "tailscale logout failed") {
		t.Fatalf("Logout() error = %v, want the logout failure prefix", err)
	}
}

func TestLocalAPIDefaultDialUsesUnixSocket(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "tailscaled.sock")
	serveUnixSocket(t, socketPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != tailscaleLocalAPIBasePath+"/status" {
			t.Errorf("request path = %q, want %q", r.URL.Path, tailscaleLocalAPIBasePath+"/status")
		}
		if r.Host != tailscaleLocalAPIHost {
			t.Errorf("request Host = %q, want %q", r.Host, tailscaleLocalAPIHost)
		}
		_, _ = io.WriteString(w, deviceStatusJSON)
	}))

	client := NewLocalClient()
	client.localAPI = newLocalAPIClient()
	client.localAPI.socketPath = socketPath

	status, err := client.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status.BackendState != "Running" {
		t.Fatalf("BackendState = %q, want %q", status.BackendState, "Running")
	}
}

func TestLocalAPIFallsBackToSecondarySocketPath(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "tailscaled.sock")
	serveUnixSocket(t, socketPath, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, devicePrefsJSON)
	}))

	client := NewLocalClient()
	client.localAPI = newLocalAPIClient()
	client.localAPI.socketPath = filepath.Join(t.TempDir(), "missing", "tailscaled.sock")
	client.localAPI.fallbackSocketPath = socketPath

	prefs, err := client.Prefs()
	if err != nil {
		t.Fatalf("Prefs() error = %v", err)
	}
	if want := []string{"192.168.0.0/22", "10.0.0.0/24"}; !reflect.DeepEqual(prefs.AdvertiseRoutes, want) {
		t.Fatalf("AdvertiseRoutes = %#v, want %#v", prefs.AdvertiseRoutes, want)
	}
}

func TestLocalAPIDoesNotFallBackForNonMissingSocket(t *testing.T) {
	notASocket := filepath.Join(t.TempDir(), "tailscaled.sock")
	if err := os.WriteFile(notASocket, []byte("regular file"), 0o600); err != nil {
		t.Fatalf("write %s: %v", notASocket, err)
	}

	fallbackPath := filepath.Join(t.TempDir(), "tailscaled.sock")
	serveUnixSocket(t, fallbackPath, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, deviceStatusJSON)
	}))

	client := NewLocalClient()
	client.localAPI = newLocalAPIClient()
	client.localAPI.socketPath = notASocket
	client.localAPI.fallbackSocketPath = fallbackPath

	_, err := client.Status()
	if err == nil {
		t.Fatal("Status() error = nil, want dial error")
	}
	// A fallback attempt would have succeeded instead of reporting the primary
	// path, so the error text has to point at the path that was actually dialed.
	if !strings.Contains(err.Error(), notASocket) {
		t.Fatalf("Status() error = %v, want it to mention %q", err, notASocket)
	}
}

func TestLocalAPIDialErrorKeepsUnderlyingCause(t *testing.T) {
	client := NewLocalClient()
	client.localAPI = newLocalAPIClient()
	client.localAPI.socketPath = filepath.Join(t.TempDir(), "missing", "tailscaled.sock")
	client.localAPI.fallbackSocketPath = filepath.Join(t.TempDir(), "missing", "tailscaled.sock")

	_, err := client.Status()
	if err == nil {
		t.Fatal("Status() error = nil, want dial error")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Status() error = %v, want it to wrap fs.ErrNotExist", err)
	}
}

func TestLocalAPIHandlesNeedsLoginStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"BackendState":"NeedsLogin","AuthURL":"https://login.tailscale.com/a/example"}`)
	}))
	defer server.Close()

	client := NewLocalClient()
	client.localAPI = newLocalAPIClientWithDial(dialToTCP(server.Listener.Addr().String()))

	status, err := client.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status.BackendState != "NeedsLogin" {
		t.Fatalf("BackendState = %q, want %q", status.BackendState, "NeedsLogin")
	}
}

func TestLocalAPIErrorPaths(t *testing.T) {
	t.Run("error status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = io.WriteString(w, `{"error":"invalid host"}`)
		}))
		defer server.Close()

		client := NewLocalClient()
		client.localAPI = newLocalAPIClientWithDial(dialToTCP(server.Listener.Addr().String()))

		_, err := client.Status()
		if err == nil {
			t.Fatal("Status() error = nil, want error")
		}
		for _, want := range []string{"403", "invalid host"} {
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("Status() error = %v, want it to contain %q", err, want)
			}
		}
	})

	t.Run("error status body is truncated", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, strings.Repeat("x", 4096))
		}))
		defer server.Close()

		client := NewLocalClient()
		client.localAPI = newLocalAPIClientWithDial(dialToTCP(server.Listener.Addr().String()))

		_, err := client.Status()
		if err == nil {
			t.Fatal("Status() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "500") {
			t.Fatalf("Status() error = %v, want it to contain the status code", err)
		}
		if len(err.Error()) > 600 {
			t.Fatalf("Status() error is %d bytes, want the response body truncated", len(err.Error()))
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, "<html>not json</html>")
		}))
		defer server.Close()

		client := NewLocalClient()
		client.localAPI = newLocalAPIClientWithDial(dialToTCP(server.Listener.Addr().String()))

		_, err := client.Status()
		if err == nil {
			t.Fatal("Status() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "decode") {
			t.Fatalf("Status() error = %v, want a decode error", err)
		}
	})

	t.Run("connection closed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Errorf("hijack: %v", err)
				return
			}
			_ = conn.Close()
		}))
		defer server.Close()

		client := NewLocalClient()
		client.localAPI = newLocalAPIClientWithDial(dialToTCP(server.Listener.Addr().String()))

		if _, err := client.Status(); err == nil {
			t.Fatal("Status() error = nil, want error")
		}
	})

	t.Run("context cancelled", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
		}))
		defer server.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		client := NewLocalClientWithContext(ctx)
		client.localAPI = newLocalAPIClientWithDial(dialToTCP(server.Listener.Addr().String()))

		if _, err := client.Status(); !errors.Is(err, context.Canceled) {
			t.Fatalf("Status() error = %v, want %v", err, context.Canceled)
		}
	})

	t.Run("context deadline", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
		}))
		defer server.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		client := NewLocalClientWithContext(ctx)
		client.localAPI = newLocalAPIClientWithDial(dialToTCP(server.Listener.Addr().String()))

		started := time.Now()
		_, err := client.Status()
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Status() error = %v, want %v", err, context.DeadlineExceeded)
		}
		if elapsed := time.Since(started); elapsed > 5*time.Second {
			t.Fatalf("Status() took %s, want it to stop at the context deadline", elapsed)
		}
	})
}

func TestLocalAPILoginUsesProfilesPrefsAndInteractiveEndpoints(t *testing.T) {
	type capturedRequest struct {
		method      string
		path        string
		host        string
		contentType string
		body        string
	}

	const authURL = "https://login.tailscale.com/a/1106eae601b5c5"

	var (
		requests   []capturedRequest
		statusCall int
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		requests = append(requests, capturedRequest{
			method:      r.Method,
			path:        r.URL.Path,
			host:        r.Host,
			contentType: r.Header.Get("Content-Type"),
			body:        string(body),
		})

		switch {
		case r.Method == http.MethodPut && r.URL.Path == tailscaleLocalAPIBasePath+tailscaleLocalAPIProfilesPath:
			// PUT /profiles/ answers 201 once the empty profile exists.
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodPatch && r.URL.Path == tailscaleLocalAPIBasePath+tailscaleLocalAPIPrefsPath:
			// PATCH /prefs answers 200 with an empty body.
		case r.Method == http.MethodGet && r.URL.Path == tailscaleLocalAPIBasePath+tailscaleLocalAPIStatusPath:
			statusCall++
			if statusCall < 3 {
				// The status that gates POST /start and the first poll both
				// run before the daemon publishes the URL.
				_, _ = io.WriteString(w, `{"BackendState":"NeedsLogin","AuthURL":""}`)
				return
			}
			_, _ = io.WriteString(w, `{"BackendState":"NeedsLogin","AuthURL":"`+authURL+`"}`)
		case r.Method == http.MethodPost && r.URL.Path == tailscaleLocalAPIBasePath+tailscaleLocalAPILoginInteractivePath:
			// POST /login-interactive answers 204 with no body.
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewLocalClient()
	client.localAPI = newLocalAPIClientWithDial(dialToTCP(server.Listener.Addr().String()))

	url, err := client.Login()
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if url != authURL {
		t.Fatalf("Login() = %q, want %q", url, authURL)
	}

	wantRequests := []capturedRequest{
		{
			method: http.MethodPut,
			path:   tailscaleLocalAPIBasePath + tailscaleLocalAPIProfilesPath,
			host:   tailscaleLocalAPIHost,
		},
		{
			method:      http.MethodPatch,
			path:        tailscaleLocalAPIBasePath + tailscaleLocalAPIPrefsPath,
			host:        tailscaleLocalAPIHost,
			contentType: "application/json",
			body:        `{"CorpDNS":false,"CorpDNSSet":true}`,
		},
		{
			method: http.MethodGet,
			path:   tailscaleLocalAPIBasePath + tailscaleLocalAPIStatusPath,
			host:   tailscaleLocalAPIHost,
		},
		{
			method: http.MethodPost,
			path:   tailscaleLocalAPIBasePath + tailscaleLocalAPILoginInteractivePath,
			host:   tailscaleLocalAPIHost,
		},
		{
			method: http.MethodGet,
			path:   tailscaleLocalAPIBasePath + tailscaleLocalAPIStatusPath,
			host:   tailscaleLocalAPIHost,
		},
		{
			method: http.MethodGet,
			path:   tailscaleLocalAPIBasePath + tailscaleLocalAPIStatusPath,
			host:   tailscaleLocalAPIHost,
		},
	}
	if !reflect.DeepEqual(requests, wantRequests) {
		t.Fatalf("requests = %#v, want %#v", requests, wantRequests)
	}
}

func TestLocalAPILoginRejectsUnsafeAuthURLBeforeReturningIt(t *testing.T) {
	const unsafeURL = "javascript:alert(1)"
	var statusCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == tailscaleLocalAPIBasePath+tailscaleLocalAPIProfilesPath:
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodPatch && r.URL.Path == tailscaleLocalAPIBasePath+tailscaleLocalAPIPrefsPath:
			// PATCH /prefs answers 200 with an empty body.
		case r.Method == http.MethodGet && r.URL.Path == tailscaleLocalAPIBasePath+tailscaleLocalAPIStatusPath:
			statusCalls++
			status := TsStatus{BackendState: "NeedsLogin"}
			if statusCalls > 1 {
				status.AuthURL = unsafeURL
			}
			if err := json.NewEncoder(w).Encode(status); err != nil {
				t.Errorf("encode status: %v", err)
			}
		case r.Method == http.MethodPost && r.URL.Path == tailscaleLocalAPIBasePath+tailscaleLocalAPILoginInteractivePath:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewLocalClient()
	client.localAPI = newLocalAPIClientWithDial(dialToTCP(server.Listener.Addr().String()))

	url, err := client.Login()
	if url != "" {
		t.Fatalf("Login() URL = %q, want empty URL for unsafe AuthURL", url)
	}
	if !errors.Is(err, errInvalidTailscaleLoginURL) {
		t.Fatalf("Login() error = %v, want %v", err, errInvalidTailscaleLoginURL)
	}
	if !strings.Contains(err.Error(), "tailscale login failed") {
		t.Fatalf("Login() error = %v, want the login failure prefix", err)
	}
	if strings.Contains(err.Error(), unsafeURL) {
		t.Fatalf("Login() error contains the untrusted AuthURL: %v", err)
	}
}

func TestLocalAPILoginStartsBackendOnlyForStoppedStates(t *testing.T) {
	tests := []struct {
		name         string
		backendState string
		wantStart    bool
	}{
		{name: "stopped", backendState: "Stopped", wantStart: true},
		{name: "no state", backendState: "NoState", wantStart: true},
		{name: "needs login", backendState: "NeedsLogin", wantStart: false},
		{name: "running", backendState: "Running", wantStart: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type startRequest struct {
				contentType string
				body        string
			}
			var starts []startRequest

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodPut:
					w.WriteHeader(http.StatusCreated)
				case r.Method == http.MethodPatch:
					// PATCH /prefs answers 200 with an empty body.
				case r.Method == http.MethodPost && r.URL.Path == tailscaleLocalAPIBasePath+tailscaleLocalAPIStartPath:
					body, err := io.ReadAll(r.Body)
					if err != nil {
						t.Errorf("read request body: %v", err)
					}
					starts = append(starts, startRequest{
						contentType: r.Header.Get("Content-Type"),
						body:        string(body),
					})
					// POST /start answers 204 with no body.
					w.WriteHeader(http.StatusNoContent)
				case r.Method == http.MethodPost && r.URL.Path == tailscaleLocalAPIBasePath+tailscaleLocalAPILoginInteractivePath:
					w.WriteHeader(http.StatusNoContent)
				case r.Method == http.MethodGet && r.URL.Path == tailscaleLocalAPIBasePath+tailscaleLocalAPIStatusPath:
					_, _ = io.WriteString(w, `{"BackendState":"`+tt.backendState+`","AuthURL":"https://login.tailscale.com/a/example"}`)
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			client := NewLocalClient()
			client.localAPI = newLocalAPIClientWithDial(dialToTCP(server.Listener.Addr().String()))

			if _, err := client.Login(); err != nil {
				t.Fatalf("Login() error = %v", err)
			}

			if !tt.wantStart {
				if len(starts) != 0 {
					t.Fatalf("POST /start requests = %#v, want none for BackendState %q", starts, tt.backendState)
				}
				return
			}
			if len(starts) != 1 {
				t.Fatalf("POST /start requests = %#v, want exactly one for BackendState %q", starts, tt.backendState)
			}
			if starts[0].body != "{}" {
				t.Fatalf("POST /start body = %q, want %q", starts[0].body, "{}")
			}
			if starts[0].contentType != "application/json" {
				t.Fatalf("POST /start Content-Type = %q, want application/json", starts[0].contentType)
			}
		})
	}
}

func TestLocalAPILoginReturnsEmptyURLWhenBackendReportsRunning(t *testing.T) {
	const statusPath = tailscaleLocalAPIBasePath + tailscaleLocalAPIStatusPath

	var statusCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut:
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodPatch:
			// PATCH /prefs answers 200 with an empty body.
		case r.Method == http.MethodPost && r.URL.Path == tailscaleLocalAPIBasePath+tailscaleLocalAPILoginInteractivePath:
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == statusPath:
			statusCalls++
			switch statusCalls {
			case 1:
				// Gating status: NeedsLogin keeps POST /start out of the flow.
				_, _ = io.WriteString(w, `{"BackendState":"NeedsLogin","AuthURL":""}`)
			case 2:
				_, _ = io.WriteString(w, `{"BackendState":"Starting","AuthURL":""}`)
			default:
				// The login completed without ever surfacing a browser URL.
				_, _ = io.WriteString(w, `{"BackendState":"Running","AuthURL":""}`)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewLocalClient()
	client.localAPI = newLocalAPIClientWithDial(dialToTCP(server.Listener.Addr().String()))

	url, err := client.Login()
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if url != "" {
		t.Fatalf("Login() = %q, want an empty URL once the backend reports Running", url)
	}
}

func TestLocalAPILoginTimesOutWaitingForAuthURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == tailscaleLocalAPIBasePath+tailscaleLocalAPIStatusPath {
			_, _ = io.WriteString(w, `{"BackendState":"NeedsLogin","AuthURL":""}`)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewLocalClient()
	client.localAPI = newLocalAPIClientWithDial(dialToTCP(server.Listener.Addr().String()))
	client.loginTimeout = 300 * time.Millisecond

	started := time.Now()
	_, err := client.Login()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Login() error = %v, want %v", err, context.DeadlineExceeded)
	}
	if !strings.Contains(err.Error(), "tailscale login failed") {
		t.Fatalf("Login() error = %v, want the login failure prefix", err)
	}
	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("Login() took %s, want it to stop at the login timeout", elapsed)
	}
}

func TestLocalAPILoginPropagatesContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := NewLocalClientWithContext(ctx)
	client.localAPI = newLocalAPIClientWithDial(dialToTCP(server.Listener.Addr().String()))

	_, err := client.Login()
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Login() error = %v, want %v", err, context.Canceled)
	}
	if !strings.Contains(err.Error(), "tailscale login failed") {
		t.Fatalf("Login() error = %v, want the login failure prefix", err)
	}
}

func TestLocalAPILoginStopsWhenProfileRequestFails(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"Error":"profile creation failed"}`)
	}))
	defer server.Close()

	client := NewLocalClient()
	client.localAPI = newLocalAPIClientWithDial(dialToTCP(server.Listener.Addr().String()))

	_, err := client.Login()
	if err == nil {
		t.Fatal("Login() error = nil, want error")
	}
	for _, want := range []string{"tailscale login failed", strconv.Itoa(http.StatusInternalServerError), "profile creation failed"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("Login() error = %v, want it to contain %q", err, want)
		}
	}

	wantRequests := []string{"PUT " + tailscaleLocalAPIBasePath + tailscaleLocalAPIProfilesPath}
	if !reflect.DeepEqual(requests, wantRequests) {
		t.Fatalf("requests = %#v, want %#v", requests, wantRequests)
	}
}

func TestLocalAPILoginStopsWhenPrefsPatchFails(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusCreated)
			return
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"Error":"prefs are not writable"}`)
	}))
	defer server.Close()

	client := NewLocalClient()
	client.localAPI = newLocalAPIClientWithDial(dialToTCP(server.Listener.Addr().String()))

	_, err := client.Login()
	if err == nil {
		t.Fatal("Login() error = nil, want error")
	}
	for _, want := range []string{"tailscale login failed", strconv.Itoa(http.StatusForbidden), "prefs are not writable"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("Login() error = %v, want it to contain %q", err, want)
		}
	}

	wantRequests := []string{
		"PUT " + tailscaleLocalAPIBasePath + tailscaleLocalAPIProfilesPath,
		"PATCH " + tailscaleLocalAPIBasePath + tailscaleLocalAPIPrefsPath,
	}
	if !reflect.DeepEqual(requests, wantRequests) {
		t.Fatalf("requests = %#v, want %#v", requests, wantRequests)
	}
}

func TestLocalAPILoginReportsStatusFailureWhilePolling(t *testing.T) {
	const statusPath = tailscaleLocalAPIBasePath + tailscaleLocalAPIStatusPath

	var statusCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut:
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodPatch:
			// PATCH /prefs answers 200 with an empty body.
		case r.Method == http.MethodPost && r.URL.Path == tailscaleLocalAPIBasePath+tailscaleLocalAPILoginInteractivePath:
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == statusPath:
			statusCalls++
			if statusCalls == 1 {
				_, _ = io.WriteString(w, `{"BackendState":"NeedsLogin","AuthURL":""}`)
				return
			}
			conn, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Errorf("hijack: %v", err)
				return
			}
			_ = conn.Close()
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewLocalClient()
	client.localAPI = newLocalAPIClientWithDial(dialToTCP(server.Listener.Addr().String()))

	started := time.Now()
	_, err := client.Login()
	if err == nil {
		t.Fatal("Login() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "tailscale login failed") {
		t.Fatalf("Login() error = %v, want the login failure prefix", err)
	}
	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("Login() took %s, want it to stop retrying once /status fails", elapsed)
	}
}
