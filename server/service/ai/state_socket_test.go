package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStateSocketGetServices(t *testing.T) {
	env := newTestServiceStateEnv(t)
	setTestStateSocketCapabilities(t, 256, chipTypeAX620QF)
	if err := saveConfigToPath(env.configPath, Config{MCPEnabled: true, MCPAPIKey: "test-key"}); err != nil {
		t.Fatalf("write legacy ai config: %v", err)
	}

	_, client, _ := startTestStateSocketServer(t, env)

	resp := doStateSocketRequest(t, client, http.MethodGet, "/v1/services", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var data stateSocketServicesResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("decode services response: %v", err)
	}

	if data.Revision != 1 {
		t.Fatalf("expected migrated revision 1, got %#v", data)
	}
	mcp := data.Services[aiServiceMCP]
	if !mcp.Enabled || !mcp.Supported || mcp.Source != serviceStateSourceMigration {
		t.Fatalf("unexpected mcp response: %#v", mcp)
	}
	if data.Services[aiServiceMemoryFabric].Supported {
		t.Fatalf("expected memoryFabric unsupported on 256MB tier")
	}
	if _, err := os.Stat(env.statePath); err != nil {
		t.Fatalf("expected state file created on socket start: %v", err)
	}
}

func TestStateSocketPutServiceUsesScreenSourceByDefault(t *testing.T) {
	env := newTestServiceStateEnv(t)
	setTestStateSocketCapabilities(t, 256, chipTypeAX620QF)
	var runtimeCalls []string
	setTestAIRuntime(t, nil, &runtimeCalls)
	env.manager.setRuntimeReconciler(defaultServiceStateRuntimeReconciler)

	_, client, _ := startTestStateSocketServer(t, env)

	resp := doStateSocketRequest(t, client, http.MethodPut, "/v1/services/mcp", []byte(`{"enabled":true}`))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var data stateSocketServicesResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("decode put response: %v", err)
	}
	if data.Revision != 2 {
		t.Fatalf("expected revision 2 after PUT, got %#v", data)
	}
	mcp := data.Services[aiServiceMCP]
	if !mcp.Enabled || mcp.Source != stateSocketDefaultSource {
		t.Fatalf("unexpected mcp response after PUT: %#v", mcp)
	}

	entry, err := env.manager.Get(aiServiceMCP)
	if err != nil {
		t.Fatalf("get manager mcp state: %v", err)
	}
	if !entry.Enabled || entry.Source != stateSocketDefaultSource {
		t.Fatalf("expected manager state updated from socket PUT, got %#v", entry)
	}
	if !stringSlicesEqual(runtimeCalls, []string{"runtime:true:false:false:true:0s"}) {
		t.Fatalf("expected socket PUT to apply AI runtime, got %#v", runtimeCalls)
	}
}

func TestStateSocketPutServiceAcceptsCustomSource(t *testing.T) {
	env := newTestServiceStateEnv(t)
	setTestStateSocketCapabilities(t, 512, chipTypeAX620QE)
	var runtimeCalls []string
	setTestAIRuntime(t, nil, &runtimeCalls)
	env.manager.setRuntimeReconciler(defaultServiceStateRuntimeReconciler)

	_, client, _ := startTestStateSocketServer(t, env)

	resp := doStateSocketRequest(t, client, http.MethodPut, "/v1/services/screenTimelapse", []byte(`{"enabled":true,"source":"web"}`))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var data stateSocketServicesResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("decode put response: %v", err)
	}
	state := data.Services[aiServiceScreenTimelapse]
	if !state.Enabled || !state.Supported || state.Source != "web" {
		t.Fatalf("unexpected screenTimelapse response after PUT: %#v", state)
	}
	if !stringSlicesEqual(runtimeCalls, []string{"runtime:true:true:true:true:0s"}) {
		t.Fatalf("expected AI runtime call, got %#v", runtimeCalls)
	}
}

func TestStateSocketScreenTimelapseSetFailureDoesNotWriteState(t *testing.T) {
	env := newTestServiceStateEnv(t)
	setTestStateSocketCapabilities(t, 512, chipTypeAX620QE)
	setTestAIRuntime(t, errTestAIRuntime, nil)
	env.manager.setRuntimeReconciler(defaultServiceStateRuntimeReconciler)

	_, client, _ := startTestStateSocketServer(t, env)
	before, beforeState := readServiceStateFile(t, env.statePath)

	resp := doStateSocketRequest(t, client, http.MethodPut, "/v1/services/screenTimelapse", []byte(`{"enabled":true,"source":"screen"}`))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", resp.StatusCode)
	}

	after, afterState := readServiceStateFile(t, env.statePath)
	if !bytes.Equal(before, after) {
		t.Fatalf("expected service state file unchanged after socket set failure")
	}
	if afterState.Revision != beforeState.Revision ||
		afterState.Services[aiServiceScreenTimelapse] != beforeState.Services[aiServiceScreenTimelapse] {
		t.Fatalf("expected service state unchanged, before=%#v after=%#v", beforeState, afterState)
	}
}

func TestStateSocketRequestErrors(t *testing.T) {
	env := newTestServiceStateEnv(t)
	setTestStateSocketCapabilities(t, 256, chipTypeAX620QF)

	_, client, _ := startTestStateSocketServer(t, env)

	tests := []struct {
		name   string
		method string
		path   string
		body   []byte
		status int
	}{
		{
			name:   "method not allowed for list",
			method: http.MethodPost,
			path:   "/v1/services",
			status: http.StatusMethodNotAllowed,
		},
		{
			name:   "invalid service id",
			method: http.MethodPut,
			path:   "/v1/services/unknown",
			body:   []byte(`{"enabled":true}`),
			status: http.StatusNotFound,
		},
		{
			name:   "invalid json",
			method: http.MethodPut,
			path:   "/v1/services/mcp",
			body:   []byte(`{"enabled":`),
			status: http.StatusBadRequest,
		},
		{
			name:   "missing enabled",
			method: http.MethodPut,
			path:   "/v1/services/mcp",
			body:   []byte(`{}`),
			status: http.StatusBadRequest,
		},
		{
			name:   "unknown field",
			method: http.MethodPut,
			path:   "/v1/services/mcp",
			body:   []byte(`{"enabled":true,"future":true}`),
			status: http.StatusBadRequest,
		},
		{
			name:   "multiple JSON values",
			method: http.MethodPut,
			path:   "/v1/services/mcp",
			body:   []byte(`{"enabled":true} {}`),
			status: http.StatusBadRequest,
		},
		{
			name:   "oversized body",
			method: http.MethodPut,
			path:   "/v1/services/mcp",
			body:   []byte(`{"enabled":true,"padding":"` + strings.Repeat("x", stateSocketMaxRequest) + `"}`),
			status: http.StatusBadRequest,
		},
		{
			name:   "method not allowed for service",
			method: http.MethodGet,
			path:   "/v1/services/mcp",
			status: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := doStateSocketRequest(t, client, tt.method, tt.path, tt.body)
			defer resp.Body.Close()
			if resp.StatusCode != tt.status {
				t.Fatalf("expected status %d, got %d", tt.status, resp.StatusCode)
			}
		})
	}
}

func TestStateSocketHTTPServerBoundsRequestsWithoutStreamWriteLifetime(t *testing.T) {
	server := newStateSocketHTTPServer(http.NotFoundHandler())
	if server.ReadHeaderTimeout != stateSocketReadHeader ||
		server.ReadTimeout != stateSocketReadTimeout ||
		server.IdleTimeout != stateSocketIdleTimeout ||
		server.MaxHeaderBytes != stateSocketMaxHeader {
		t.Fatalf("unexpected state socket server bounds: %#v", server)
	}
	if server.WriteTimeout != 0 {
		t.Fatalf("state event stream has global WriteTimeout %s", server.WriteTimeout)
	}
}

func TestStateSocketStartRejectsNonSocketPath(t *testing.T) {
	env := newTestServiceStateEnv(t)
	setTestStateSocketCapabilities(t, 256, chipTypeAX620QF)

	socketPath := filepath.Join(t.TempDir(), "nanokvm-state.sock")
	original := []byte("do-not-overwrite")
	if err := os.WriteFile(socketPath, original, 0o600); err != nil {
		t.Fatalf("write non-socket path: %v", err)
	}

	stubGateHandlerForTest(t)
	server := newStateSocketServer(env.manager, socketPath)
	if err := server.Start(); err == nil {
		t.Fatalf("expected non-socket path start error")
	}

	data, err := os.ReadFile(socketPath)
	if err != nil {
		t.Fatalf("read non-socket path after failed start: %v", err)
	}
	if !bytes.Equal(data, original) {
		t.Fatalf("expected non-socket path unchanged, got %q", data)
	}
}

func TestStateSocketStartReplacesOldSocket(t *testing.T) {
	env := newTestServiceStateEnv(t)
	setTestStateSocketCapabilities(t, 256, chipTypeAX620QF)

	socketPath := filepath.Join(t.TempDir(), "nanokvm-state.sock")
	oldListener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("create old socket: %v", err)
	}
	if err := oldListener.Close(); err != nil {
		t.Fatalf("close old socket: %v", err)
	}

	stubGateHandlerForTest(t)
	server := newStateSocketServer(env.manager, socketPath)
	if err := server.Start(); err != nil {
		t.Fatalf("start state socket over old socket: %v", err)
	}
	t.Cleanup(func() {
		_ = server.Stop()
	})

	info, err := os.Lstat(socketPath)
	if err != nil {
		t.Fatalf("stat recreated socket: %v", err)
	}
	if info.Mode()&os.ModeSocket == 0 {
		t.Fatalf("expected recreated socket, got mode %s", info.Mode())
	}
}

func TestStateSocketStopRemovesSocket(t *testing.T) {
	env := newTestServiceStateEnv(t)
	setTestStateSocketCapabilities(t, 256, chipTypeAX620QF)

	socketPath := filepath.Join(t.TempDir(), "nanokvm-state.sock")
	stubGateHandlerForTest(t)
	server := newStateSocketServer(env.manager, socketPath)
	if err := server.Start(); err != nil {
		t.Fatalf("start state socket: %v", err)
	}

	info, err := os.Lstat(socketPath)
	if err != nil {
		t.Fatalf("stat socket after start: %v", err)
	}
	if info.Mode().Perm() != stateSocketPermission {
		t.Fatalf("expected socket permission %o, got %o", stateSocketPermission, info.Mode().Perm())
	}

	if err := server.Stop(); err != nil {
		t.Fatalf("stop state socket: %v", err)
	}
	if _, err := os.Lstat(socketPath); !os.IsNotExist(err) {
		t.Fatalf("expected socket removed after stop, stat err=%v", err)
	}
}

func TestStateSocketEvents(t *testing.T) {
	env := newTestServiceStateEnv(t)
	setTestStateSocketCapabilities(t, 256, chipTypeAX620QF)

	_, client, _ := startTestStateSocketServer(t, env)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://unix/v1/events", nil)
	if err != nil {
		t.Fatalf("create events request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("open events stream: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	waitForServiceStateSubscribers(t, env.manager, 1)

	eventCh := make(chan ServiceStateEvent, 1)
	errCh := make(chan error, 1)
	go func() {
		line, err := bufio.NewReader(resp.Body).ReadBytes('\n')
		if err != nil {
			errCh <- err
			return
		}
		var event ServiceStateEvent
		if err := json.Unmarshal(line, &event); err != nil {
			errCh <- err
			return
		}
		eventCh <- event
	}()

	if _, err := env.manager.Set(aiServiceMCP, true, "web"); err != nil {
		t.Fatalf("set service state: %v", err)
	}

	select {
	case event := <-eventCh:
		if event.Type != serviceStateEventChanged || event.Service != aiServiceMCP || !event.Enabled ||
			event.Source != "web" || event.Revision != 2 {
			t.Fatalf("unexpected event: %#v", event)
		}
	case err := <-errCh:
		t.Fatalf("read event: %v", err)
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for event")
	}

	cancel()
	resp.Body.Close()
	waitForServiceStateSubscribers(t, env.manager, 0)
}

func startTestStateSocketServer(t *testing.T, env serviceStateTestEnv) (*StateSocketServer, *http.Client, string) {
	t.Helper()

	socketPath := filepath.Join(t.TempDir(), "nanokvm-state.sock")
	stubGateHandlerForTest(t)
	server := newStateSocketServer(env.manager, socketPath)
	if err := server.Start(); err != nil {
		t.Fatalf("start state socket: %v", err)
	}
	t.Cleanup(func() {
		_ = server.Stop()
	})

	client := newTestUnixSocketClient(t, socketPath)
	return server, client, socketPath
}

func newTestUnixSocketClient(t *testing.T, socketPath string) *http.Client {
	t.Helper()

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, "unix", socketPath)
		},
	}
	t.Cleanup(transport.CloseIdleConnections)

	return &http.Client{
		Transport: transport,
		Timeout:   2 * time.Second,
	}
}

func doStateSocketRequest(t *testing.T, client *http.Client, method string, path string, body []byte) *http.Response {
	t.Helper()

	var reqBody *bytes.Reader
	if body == nil {
		reqBody = bytes.NewReader(nil)
	} else {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, "http://unix"+path, reqBody)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do request %s %s: %v", method, path, err)
	}
	return resp
}

func setTestStateSocketCapabilities(t *testing.T, memoryMiB int, chipType string) {
	t.Helper()

	oldReadVisibleMemoryMiB := readVisibleMemoryMiB
	oldReadChipType := readChipType
	oldNewMemoryFabricManager := newMemoryFabricManager
	readVisibleMemoryMiB = func() (int, error) {
		return memoryMiB, nil
	}
	readChipType = func() string {
		return chipType
	}
	newMemoryFabricManager = func() memoryFabricManager {
		return memoryFabricManager{
			unitName:    memoryFabricUnitName,
			ocrUnitName: memoryFabricOCRUnitName,
			isRunning: func(string) (bool, error) {
				return false, nil
			},
			startService: func(string, bool) error {
				return nil
			},
			stopService: func(string, bool) error {
				return nil
			},
		}
	}

	t.Cleanup(func() {
		readVisibleMemoryMiB = oldReadVisibleMemoryMiB
		readChipType = oldReadChipType
		newMemoryFabricManager = oldNewMemoryFabricManager
	})
}

func waitForServiceStateSubscribers(t *testing.T, manager *ServiceStateManager, want int) {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	for {
		manager.mu.Lock()
		got := len(manager.subscribers)
		manager.mu.Unlock()
		if got == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected %d subscribers", want)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
