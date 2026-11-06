package ai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/application"
	"NanoKVM-Server/service/tailscalegate"
)

// These tests exercise the Tailscale login gate as it is mounted on the
// root-local state socket. They create the socket under t.TempDir() and talk to
// it over a real unix connection, never through httptest.NewServer, and they
// never read the region marker from /etc or touch a device path.

// stubGateHandlerForTest replaces the production gate seam with a closed gate
// so socket tests never build the real managed driver (its construction reads
// device state under /etc/kvm).
func stubGateHandlerForTest(t *testing.T) {
	t.Helper()
	previous := newGateHandler
	newGateHandler = func() http.Handler {
		return tailscalegate.NewHandler(tailscalegate.Deps{Driver: tailscalegate.NewUnimplementedDriver()})
	}
	t.Cleanup(func() { newGateHandler = previous })
}

// TestStateSocketGateHandlerFollowsTheProductionSeam proves that a state
// socket built without an explicitly injected gate mounts exactly the handler
// the production seam builds.
func TestStateSocketGateHandlerFollowsTheProductionSeam(t *testing.T) {
	env := newTestServiceStateEnv(t)
	marker := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":"seam"}`))
	})
	previous := newGateHandler
	newGateHandler = func() http.Handler { return marker }
	t.Cleanup(func() { newGateHandler = previous })

	socketPath := filepath.Join(t.TempDir(), "nanokvm-state.sock")
	server := newStateSocketServer(env.manager, socketPath)
	if err := server.Start(); err != nil {
		t.Fatalf("start state socket: %v", err)
	}
	t.Cleanup(func() { _ = server.Stop() })
	client := newTestUnixSocketClient(t, socketPath)

	resp := doStateSocketRequest(t, client, http.MethodGet, tailscalegate.LoginCodePath+"?heartbeat=true", nil)
	body := readStateSocketBody(t, resp)
	if string(body) != `{"code":"seam"}` {
		t.Fatalf("gate body = %s, want the seam marker", body)
	}
}

// TestProductionGateHandlerBuilds proves the production constructor never
// returns nil. DefaultService always carries the real pairing client, so the
// fail-closed branch cannot trigger on this path; it is covered by
// TestGateHandlerForDriver and the NewGateDriver constructor tests.
func TestProductionGateHandlerBuilds(t *testing.T) {
	previous := newGateDriverForSocket
	newGateDriverForSocket = func() (tailscalegate.Driver, error) {
		return nil, errors.New("must not be constructed in this test")
	}
	t.Cleanup(func() { newGateDriverForSocket = previous })
	if handler := productionGateHandler(); handler == nil {
		t.Fatal("production gate handler is nil")
	}
}

// TestGateHandlerForDriver proves both branches of the wiring assembly: a
// constructed driver is served as-is, while a construction failure fails
// closed to the not-implemented driver (heartbeat 503, action NOTIMPL)
// instead of silently running the demo.
func TestGateHandlerForDriver(t *testing.T) {
	previousRegion := gateLoadRegion
	gateLoadRegion = func() proto.ApplicationRegion { return proto.ApplicationRegionCN }
	previousState := gateLoadFeatureState
	gateLoadFeatureState = func() application.FeatureState { return application.FeatureStateEnabled }
	// The start-through-wiring assertion needs a device that already accepted
	// the privacy terms, so it must not answer CONSENT first.
	previousConsent := gateLoadConsent
	gateLoadConsent = func() bool { return true }
	t.Cleanup(func() {
		gateLoadRegion = previousRegion
		gateLoadFeatureState = previousState
		gateLoadConsent = previousConsent
	})

	t.Run("constructed driver is served", func(t *testing.T) {
		driver := &countingGateDriver{}
		handler := gateHandlerForDriver(driver, nil)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tailscalegate.LoginCodePath+"?heartbeat=true", nil))
		// The counting driver always errors, so the handler answers 503 — the
		// point is that the request reached the injected driver at all.
		if recorder.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503", recorder.Code)
		}
		if driver.callCount() == 0 {
			t.Fatal("the constructed driver was never called")
		}
	})

	t.Run("construction failure fails closed", func(t *testing.T) {
		handler := gateHandlerForDriver(nil, errors.New("managed mode unwired"))

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tailscalegate.LoginCodePath+"?heartbeat=true", nil))
		if recorder.Code != http.StatusServiceUnavailable {
			t.Fatalf("heartbeat status = %d, want 503", recorder.Code)
		}

		recorder = httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tailscalegate.LoginCodePath+"?start=true", nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("start status = %d, want 200", recorder.Code)
		}
		var action struct {
			Status  bool   `json:"status"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &action); err != nil {
			t.Fatalf("decode action: %v", err)
		}
		if action.Status || action.Message != "NOTIMPL" {
			t.Fatalf("start = %+v, want NOTIMPL failure", action)
		}
	})
}

func TestGateHandlerUsesInjectedFeatureManifestReader(t *testing.T) {
	if gateLoadFeatureState == nil {
		t.Fatal("gateLoadFeatureState = nil, want the production manifest reader")
	}

	previousRegion := gateLoadRegion
	previousState := gateLoadFeatureState
	gateLoadRegion = func() proto.ApplicationRegion { return proto.ApplicationRegionCN }
	t.Cleanup(func() {
		gateLoadRegion = previousRegion
		gateLoadFeatureState = previousState
	})

	driver := &countingGateDriver{}

	gateLoadFeatureState = func() application.FeatureState { return application.FeatureStateDisabled }
	recorder := httptest.NewRecorder()
	gateHandlerForDriver(driver, nil).ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, tailscalegate.LoginCodePath+"?heartbeat=true", nil),
	)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("disabled manifest status = %d, want 404", recorder.Code)
	}
	if calls := driver.callCount(); calls != 0 {
		t.Fatalf("disabled manifest driver calls = %d, want 0", calls)
	}

	gateLoadFeatureState = func() application.FeatureState { return application.FeatureStateEnabled }
	recorder = httptest.NewRecorder()
	gateHandlerForDriver(driver, nil).ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, tailscalegate.LoginCodePath+"?heartbeat=true", nil),
	)
	// The counting driver always errors, so 503 proves the request reached it.
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("enabled manifest status = %d, want 503", recorder.Code)
	}
	if calls := driver.callCount(); calls == 0 {
		t.Fatal("enabled manifest never reached the injected driver")
	}
}

// TestGateHandlerWiresTheConsentGatedSync proves the production assembly passes
// the tri-state reader and the sync seam through to the handler: a missing
// manifest on a consented start is healed by the injected sync, after which the
// request reaches the injected driver.
func TestGateHandlerWiresTheConsentGatedSync(t *testing.T) {
	previousRegion := gateLoadRegion
	previousState := gateLoadFeatureState
	previousSync := gateSyncFeatureManifest
	previousConsent := gateLoadConsent
	gateLoadRegion = func() proto.ApplicationRegion { return proto.ApplicationRegionCN }
	state := application.FeatureStateMissing
	gateLoadFeatureState = func() application.FeatureState { return state }
	syncCalls := 0
	gateSyncFeatureManifest = func(context.Context, proto.ApplicationRegion) error {
		syncCalls++
		state = application.FeatureStateEnabled
		return nil
	}
	gateLoadConsent = func() bool { return true }
	t.Cleanup(func() {
		gateLoadRegion = previousRegion
		gateLoadFeatureState = previousState
		gateSyncFeatureManifest = previousSync
		gateLoadConsent = previousConsent
	})

	driver := &countingGateDriver{}
	recorder := httptest.NewRecorder()
	gateHandlerForDriver(driver, nil).ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, tailscalegate.LoginCodePath+"?start=true", nil),
	)
	if syncCalls != 1 {
		t.Fatalf("sync calls = %d, want 1 for a missing manifest on a consented start", syncCalls)
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("start status = %d, want 200", recorder.Code)
	}
	if driver.callCount() == 0 {
		t.Fatal("the driver was never reached after the sync enabled the manifest")
	}
}

// startTestStateSocketServerWithGate starts a state socket and installs gate
// before Start so the mounted handler is deterministic and host-safe.
func startTestStateSocketServerWithGate(t *testing.T, env serviceStateTestEnv, gate http.Handler) (*StateSocketServer, *http.Client, string) {
	t.Helper()

	stubGateHandlerForTest(t)
	socketPath := filepath.Join(t.TempDir(), "nanokvm-state.sock")
	server := newStateSocketServer(env.manager, socketPath)
	server.gateHandler = gate
	if err := server.Start(); err != nil {
		t.Fatalf("start state socket: %v", err)
	}
	t.Cleanup(func() {
		_ = server.Stop()
	})

	client := newTestUnixSocketClient(t, socketPath)
	return server, client, socketPath
}

// readStateSocketBody reads and closes a response body, failing the test on a
// read error.
func readStateSocketBody(t *testing.T, resp *http.Response) []byte {
	t.Helper()

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	return body
}

// countingGateDriver records every call so a test can prove the region gate
// answered before the driver boundary was reached. Every method returns an
// error because a configured driver must not be called on a rejected request.
type countingGateDriver struct {
	mu    sync.Mutex
	calls int
}

func (d *countingGateDriver) record() {
	d.mu.Lock()
	d.calls++
	d.mu.Unlock()
}

func (d *countingGateDriver) callCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.calls
}

func (d *countingGateDriver) Snapshot(context.Context) (tailscalegate.Report, error) {
	d.record()
	return tailscalegate.Report{}, errors.New("gate driver must not be called")
}

func (d *countingGateDriver) Initialize(context.Context) error {
	d.record()
	return errors.New("gate driver must not be called")
}

func (d *countingGateDriver) Deinitialize(context.Context) error {
	d.record()
	return errors.New("gate driver must not be called")
}

func (d *countingGateDriver) LoginCode(context.Context) (string, error) {
	d.record()
	return "", errors.New("gate driver must not be called")
}

func (d *countingGateDriver) EndLoginSession(context.Context) error {
	d.record()
	return errors.New("gate driver must not be called")
}

// TestStateSocketTailscaleLoginCodeWithUnimplementedDriver mounts the real gate
// handler with the shipped NOT-IMPLEMENTED driver and checks the five-field
// contract on the wire: heartbeat 503, action NOTIMPL, the %off% code fallback,
// the transport-level 400, and the JSON error shape carried with an exact
// Content-Length.
func TestStateSocketTailscaleLoginCodeWithUnimplementedDriver(t *testing.T) {
	env := newTestServiceStateEnv(t)
	gate := tailscalegate.NewHandler(tailscalegate.Deps{
		Driver:           tailscalegate.NewUnimplementedDriver(),
		LoadRegion:       func() proto.ApplicationRegion { return proto.ApplicationRegionCN },
		LoadFeatureState: func() application.FeatureState { return application.FeatureStateEnabled },
	})
	_, client, _ := startTestStateSocketServerWithGate(t, env, gate)

	tests := []struct {
		name   string
		method string
		path   string
		status int
		body   string
	}{
		{
			name:   "heartbeat is unavailable for the unimplemented driver",
			method: http.MethodGet,
			path:   "/api/tailscale/login/code?heartbeat=true",
			status: http.StatusServiceUnavailable,
			body:   `{"error":"tailscale state unavailable"}`,
		},
		{
			name:   "start reports NOTIMPL for the unimplemented driver",
			method: http.MethodGet,
			// The handler is built directly here, so its consent defaults apply:
			// the explicit flag is what lets the start reach the driver.
			path:   "/api/tailscale/login/code?start=true&consent=true",
			status: http.StatusOK,
			body:   `{"status":false,"message":"NOTIMPL"}`,
		},
		{
			name:   "code falls back to the off compatibility value",
			method: http.MethodGet,
			path:   "/api/tailscale/login/code",
			status: http.StatusOK,
			body:   `{"code":"%off%"}`,
		},
		{
			name:   "non GET is rejected before the path is inspected",
			method: http.MethodPost,
			path:   "/api/tailscale/login/code",
			status: http.StatusBadRequest,
			body:   `{"error":"method not supported"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := doStateSocketRequest(t, client, tt.method, tt.path, nil)
			body := readStateSocketBody(t, resp)

			if resp.StatusCode != tt.status {
				t.Fatalf("expected status %d, got %d body=%s", tt.status, resp.StatusCode, body)
			}
			if got := resp.Header.Get("Content-Type"); got != "application/json" {
				t.Fatalf("expected application/json content type, got %q", got)
			}
			if resp.ContentLength != int64(len(body)) {
				t.Fatalf("expected Content-Length %d, got %d", len(body), resp.ContentLength)
			}
			if string(body) != tt.body {
				t.Fatalf("expected body %s, got %s", tt.body, body)
			}
		})
	}

	// A path outside the mounted route is answered by ServeMux itself. The
	// gate handler's own JSON 404 body is covered by the tailscalegate package
	// tests, which call ServeHTTP directly.
	t.Run("unknown paths are not routed to the gate", func(t *testing.T) {
		for _, path := range []string{"/api/tailscale/login/code/", "/api/tailscale/other"} {
			resp := doStateSocketRequest(t, client, http.MethodGet, path, nil)
			body := readStateSocketBody(t, resp)
			if resp.StatusCode != http.StatusNotFound {
				t.Fatalf("expected status 404 for %s, got %d body=%s", path, resp.StatusCode, body)
			}
			if resp.ContentLength != int64(len(body)) {
				t.Fatalf("expected Content-Length %d for %s, got %d", len(body), path, resp.ContentLength)
			}
		}
	})
}

// TestStateSocketTailscaleRouteDispatch proves the handler is mounted on the
// login path exactly once and that an unrelated path never reaches it: the mux
// answers the 404 itself.
func TestStateSocketTailscaleRouteDispatch(t *testing.T) {
	env := newTestServiceStateEnv(t)

	var mu sync.Mutex
	var seen []string
	recorder := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Method+" "+r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"gate":true}`)
	})

	_, client, _ := startTestStateSocketServerWithGate(t, env, recorder)

	resp := doStateSocketRequest(t, client, http.MethodGet, tailscalegate.LoginCodePath, nil)
	body := readStateSocketBody(t, resp)
	if resp.StatusCode != http.StatusOK || string(body) != `{"gate":true}` {
		t.Fatalf("expected the injected handler to answer, got status=%d body=%s", resp.StatusCode, body)
	}

	resp = doStateSocketRequest(t, client, http.MethodGet, "/api/tailscale/other", nil)
	unknownBody := readStateSocketBody(t, resp)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404 for an unmounted path, got %d body=%s", resp.StatusCode, unknownBody)
	}

	mu.Lock()
	got := append([]string(nil), seen...)
	mu.Unlock()
	if !stringSlicesEqual(got, []string{"GET " + tailscalegate.LoginCodePath}) {
		t.Fatalf("expected exactly one dispatch to the injected handler, got %#v", got)
	}
}

// A disabled manifest still refuses the gate in the global region.
func TestStateSocketTailscaleFeatureGateInGlobal(t *testing.T) {
	env := newTestServiceStateEnv(t)
	driver := &countingGateDriver{}
	gate := tailscalegate.NewHandler(tailscalegate.Deps{
		Driver:           driver,
		LoadRegion:       func() proto.ApplicationRegion { return proto.ApplicationRegionGlobal },
		LoadFeatureState: func() application.FeatureState { return application.FeatureStateDisabled },
	})
	_, client, _ := startTestStateSocketServerWithGate(t, env, gate)

	resp := doStateSocketRequest(t, client, http.MethodGet, tailscalegate.LoginCodePath+"?heartbeat=true", nil)
	body := readStateSocketBody(t, resp)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", resp.StatusCode, body)
	}
	if string(body) != `{"error":"not found"}` {
		t.Fatalf("expected the fixed not-found body, got %s", body)
	}
	if resp.ContentLength != int64(len(body)) {
		t.Fatalf("expected Content-Length %d, got %d", len(body), resp.ContentLength)
	}
	if calls := driver.callCount(); calls != 0 {
		t.Fatalf("expected the feature gate to keep the driver untouched, got %d calls", calls)
	}
}

// TestStateSocketTailscaleKeepsConnectionAlive proves two requests are served
// on one keep-alive connection. The second request carries an httptrace that
// must observe a reused connection, and both bodies must be complete.
func TestStateSocketTailscaleKeepsConnectionAlive(t *testing.T) {
	env := newTestServiceStateEnv(t)
	gate := tailscalegate.NewHandler(tailscalegate.Deps{
		Driver:           tailscalegate.NewUnimplementedDriver(),
		LoadRegion:       func() proto.ApplicationRegion { return proto.ApplicationRegionCN },
		LoadFeatureState: func() application.FeatureState { return application.FeatureStateEnabled },
	})
	_, client, _ := startTestStateSocketServerWithGate(t, env, gate)

	first := doStateSocketRequest(t, client, http.MethodGet, tailscalegate.LoginCodePath, nil)
	firstBody := readStateSocketBody(t, first)
	if first.ContentLength != int64(len(firstBody)) {
		t.Fatalf("expected first Content-Length %d, got %d", len(firstBody), first.ContentLength)
	}

	var traced bool
	var reused bool
	trace := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			traced = true
			reused = info.Reused
		},
	}
	req, err := http.NewRequestWithContext(
		httptrace.WithClientTrace(context.Background(), trace),
		http.MethodGet,
		"http://unix"+tailscalegate.LoginCodePath+"?heartbeat=true",
		nil,
	)
	if err != nil {
		t.Fatalf("create traced request: %v", err)
	}

	second, err := client.Do(req)
	if err != nil {
		t.Fatalf("do traced request: %v", err)
	}
	secondBody := readStateSocketBody(t, second)
	if second.ContentLength != int64(len(secondBody)) {
		t.Fatalf("expected second Content-Length %d, got %d", len(secondBody), second.ContentLength)
	}

	if !traced {
		t.Fatalf("httptrace did not observe a connection")
	}
	if !reused {
		t.Fatalf("expected the second request to reuse the first connection")
	}
	if second.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected heartbeat status 503, got %d body=%s", second.StatusCode, secondBody)
	}
}

// TestStateSocketExistingRoutesUnchangedWithGateMounted confirms the new mount
// does not disturb the existing /v1 surface or the Start/Stop lifecycle.
func TestStateSocketExistingRoutesUnchangedWithGateMounted(t *testing.T) {
	env := newTestServiceStateEnv(t)
	setTestStateSocketCapabilities(t, 256, chipTypeAX620QF)
	if err := saveConfigToPath(env.configPath, Config{MCPEnabled: true, MCPAPIKey: "test-key"}); err != nil {
		t.Fatalf("write legacy ai config: %v", err)
	}

	server, client, socketPath := startTestStateSocketServer(t, env)

	resp := doStateSocketRequest(t, client, http.MethodGet, "/v1/services", nil)
	body := readStateSocketBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", resp.StatusCode, body)
	}
	var data stateSocketServicesResponse
	if err := json.Unmarshal(body, &data); err != nil {
		t.Fatalf("decode services response: %v", err)
	}
	if data.Revision != 1 || !data.Services[aiServiceMCP].Enabled {
		t.Fatalf("unexpected services response: %#v", data)
	}

	resp = doStateSocketRequest(t, client, http.MethodPut, "/v1/services/mcp", []byte(`{"enabled":true}`))
	putBody := readStateSocketBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected PUT status 200, got %d body=%s", resp.StatusCode, putBody)
	}

	if err := server.Stop(); err != nil {
		t.Fatalf("stop state socket: %v", err)
	}
	if err := server.Stop(); err != nil {
		t.Fatalf("second stop state socket: %v", err)
	}
	if _, err := os.Lstat(socketPath); !os.IsNotExist(err) {
		t.Fatalf("expected socket removed after stop, stat err=%v", err)
	}
}
