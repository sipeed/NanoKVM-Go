package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var testMemfabToolNames = []string{
	"list_memories",
	"read_memory",
	"search",
	"recent_activity",
	"read_recent_capture",
	"search_captures",
	"current_context",
	"get_schema",
}

func TestMemfabBridgeListsAndCallsAllExistingToolsWithoutForwardingCredentials(t *testing.T) {
	var calls atomic.Int32
	bridge := newTestMemfabBridge(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Errorf("internal backend received Authorization header")
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/tools":
			_ = json.NewEncoder(w).Encode(memfabToolsResponse{Version: 1, Tools: testMemfabDescriptors()})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/tools/call":
			calls.Add(1)
			var request memfabToolCallRequest
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&request); err != nil {
				t.Errorf("decode call request: %v", err)
			}
			if request.Version != 1 || request.Name != "search" || string(request.Arguments) != "{\"query\":\"needle\"}" {
				t.Errorf("unexpected call request: %#v", request)
			}
			_, _ = w.Write([]byte("{\"version\":1,\"result\":{\"query\":\"needle\",\"results\":[]}}"))
		default:
			http.NotFound(w, r)
		}
	}))

	tools, err := bridge.listTools(context.Background())
	if err != nil {
		t.Fatalf("listTools: %v", err)
	}
	if len(tools) != len(testMemfabToolNames) {
		t.Fatalf("tool count = %d, want %d", len(tools), len(testMemfabToolNames))
	}
	for index, name := range testMemfabToolNames {
		if tools[index].Name != name {
			t.Fatalf("tool %d = %q, want %q", index, tools[index].Name, name)
		}
	}

	result, err := bridge.callTool(context.Background(), "search", json.RawMessage([]byte("{\"query\":\"needle\"}")))
	if err != nil {
		t.Fatalf("callTool: %v", err)
	}
	if string(result) != "{\"query\":\"needle\",\"results\":[]}" || calls.Load() != 1 {
		t.Fatalf("unexpected result=%s calls=%d", result, calls.Load())
	}
}

func TestValidateMemfabToolDescriptorsFailsClosed(t *testing.T) {
	valid := memfabToolDescriptor{Name: "search", Description: "Search memory.", InputSchema: json.RawMessage([]byte("{\"type\":\"object\"}"))}
	controls := map[string]struct{}{"type_text": {}}
	tests := []struct {
		name        string
		descriptors []memfabToolDescriptor
		controls    map[string]struct{}
	}{
		{name: "empty", controls: controls},
		{name: "duplicate", descriptors: []memfabToolDescriptor{valid, valid}, controls: controls},
		{name: "collision", descriptors: []memfabToolDescriptor{{Name: "type_text", Description: valid.Description, InputSchema: valid.InputSchema}}, controls: controls},
		{name: "invalid name", descriptors: []memfabToolDescriptor{{Name: "bad name", Description: valid.Description, InputSchema: valid.InputSchema}}, controls: controls},
		{name: "missing description", descriptors: []memfabToolDescriptor{{Name: "search", InputSchema: valid.InputSchema}}, controls: controls},
		{name: "missing schema", descriptors: []memfabToolDescriptor{{Name: "search", Description: valid.Description}}, controls: controls},
		{name: "non object schema", descriptors: []memfabToolDescriptor{{Name: "search", Description: valid.Description, InputSchema: json.RawMessage([]byte("{\"type\":\"array\"}"))}}, controls: controls},
		{name: "duplicate schema key", descriptors: []memfabToolDescriptor{{Name: "search", Description: valid.Description, InputSchema: json.RawMessage([]byte("{\"type\":\"object\",\"type\":\"object\"}"))}}, controls: controls},
		{name: "non JSON schema", descriptors: []memfabToolDescriptor{{Name: "search", Description: valid.Description, InputSchema: json.RawMessage([]byte("{"))}}, controls: controls},
		{name: "oversized description", descriptors: []memfabToolDescriptor{{Name: "search", Description: strings.Repeat("x", memfabMaxToolDescriptionSize+1), InputSchema: valid.InputSchema}}, controls: controls},
		{name: "oversized schema", descriptors: []memfabToolDescriptor{{Name: "search", Description: valid.Description, InputSchema: json.RawMessage([]byte("{\"type\":\"object\",\"x\":\"" + strings.Repeat("x", memfabMaxSchemaBytes) + "\"}"))}}, controls: controls},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got, err := validateMemfabToolDescriptors(test.descriptors, test.controls); err == nil || got != nil {
				t.Fatalf("validation = %#v, %v; want nil error result", got, err)
			}
		})
	}
}

func TestMCPControllerSwitchMatrixAndBackendFailurePreserveControlTools(t *testing.T) {
	var memoryEnabled atomic.Bool
	var invalid atomic.Bool
	bridge := newTestMemfabBridge(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/v1/tools" {
			http.NotFound(w, r)
			return
		}
		tools := testMemfabDescriptors()
		if invalid.Load() {
			tools[0].Name = "control"
		}
		_ = json.NewEncoder(w).Encode(memfabToolsResponse{Version: 1, Tools: tools})
	}))
	server, controller := newTestMCPController(t, bridge, func() (bool, error) {
		return memoryEnabled.Load(), nil
	})

	if err := controller.reconcile(context.Background()); err != nil {
		t.Fatalf("disabled reconcile: %v", err)
	}
	assertMCPToolSet(t, server, []string{"control"})

	memoryEnabled.Store(true)
	if err := controller.reconcile(context.Background()); err != nil {
		t.Fatalf("enabled reconcile: %v", err)
	}
	want := append([]string{"control"}, testMemfabToolNames...)
	assertMCPToolSet(t, server, want)

	invalid.Store(true)
	if err := controller.reconcile(context.Background()); err == nil {
		t.Fatal("expected collision to fail reconciliation")
	}
	assertMCPToolSet(t, server, []string{"control"})

	invalid.Store(false)
	memoryEnabled.Store(false)
	if err := controller.reconcile(context.Background()); err != nil {
		t.Fatalf("disabled reconcile after failure: %v", err)
	}
	assertMCPToolSet(t, server, []string{"control"})
}

func TestAuthenticatedMCPHTTPToolMatrix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name          string
		mcpEnabled    bool
		memoryEnabled bool
		wantConnect   bool
		wantMemory    bool
	}{
		{name: "both off"},
		{name: "MCP off memory on", memoryEnabled: true},
		{name: "MCP on memory off", mcpEnabled: true, wantConnect: true},
		{name: "both on", mcpEnabled: true, memoryEnabled: true, wantConnect: true, wantMemory: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configPath, _, _ := withAIStateFiles(t)
			if err := saveConfigToPath(configPath, Config{MCPAPIKey: "secret"}); err != nil {
				t.Fatalf("save AI config: %v", err)
			}
			writeTestMCPMatrixState(t, test.mcpEnabled, test.memoryEnabled)

			var backendCalls atomic.Int32
			bridge := newTestMemfabBridge(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				backendCalls.Add(1)
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/v1/tools":
					_ = json.NewEncoder(w).Encode(memfabToolsResponse{Version: 1, Tools: testMemfabDescriptors()})
				case "/v1/tools/call":
					_, _ = w.Write([]byte("{\"version\":1,\"result\":{\"ok\":true}}"))
				default:
					http.NotFound(w, r)
				}
			}))
			_, controller := newTestMCPController(t, bridge, getMemoryFabricEnabled)
			router := gin.New()
			router.Any("/api/mcp", MCPAPIKeyMiddleware(), gin.WrapH(controller))

			session, err := connectTestMCPHTTPClient(t, router, "secret")
			if !test.wantConnect {
				if err == nil {
					_ = session.Close()
					t.Fatal("disabled MCP endpoint unexpectedly accepted a client")
				}
				if backendCalls.Load() != 0 {
					t.Fatalf("disabled MCP reached memory backend %d times", backendCalls.Load())
				}
				return
			}
			if err != nil {
				t.Fatalf("connect MCP client: %v", err)
			}
			defer session.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			listed, err := session.ListTools(ctx, nil)
			if err != nil {
				t.Fatalf("list tools: %v", err)
			}
			hasMemory := false
			hasControl := false
			for _, tool := range listed.Tools {
				hasMemory = hasMemory || tool.Name == "search"
				hasControl = hasControl || tool.Name == "control"
			}
			if !hasControl || hasMemory != test.wantMemory {
				t.Fatalf("unexpected tool list: control=%t memory=%t tools=%#v", hasControl, hasMemory, listed.Tools)
			}
			if !test.wantMemory && backendCalls.Load() != 0 {
				t.Fatalf("disabled Memory Fabric reached backend %d times", backendCalls.Load())
			}
			if test.wantMemory {
				result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "search", Arguments: map[string]any{"query": "safe"}})
				if err != nil || result.IsError {
					t.Fatalf("call memory tool: result=%#v err=%v", result, err)
				}
			}
		})
	}
}

func TestAuthenticatedMCPHTTPRejectsMissingAndWrongKeysBeforeBackend(t *testing.T) {
	gin.SetMode(gin.TestMode)
	configPath, _, _ := withAIStateFiles(t)
	if err := saveConfigToPath(configPath, Config{MCPAPIKey: "secret"}); err != nil {
		t.Fatalf("save AI config: %v", err)
	}
	writeTestMCPMatrixState(t, true, true)
	var backendCalls atomic.Int32
	bridge := newTestMemfabBridge(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		backendCalls.Add(1)
		http.Error(w, "must not be reached", http.StatusInternalServerError)
	}))
	_, controller := newTestMCPController(t, bridge, getMemoryFabricEnabled)
	router := gin.New()
	router.Any("/api/mcp", MCPAPIKeyMiddleware(), gin.WrapH(controller))

	for _, token := range []string{"", "wrong"} {
		if session, err := connectTestMCPHTTPClient(t, router, token); err == nil {
			_ = session.Close()
			t.Fatalf("token %q unexpectedly authenticated", token)
		}
	}
	if backendCalls.Load() != 0 {
		t.Fatalf("unauthenticated requests reached backend %d times", backendCalls.Load())
	}
}

func TestMemoryToolHandlerRechecksStateAndPreservesResultShape(t *testing.T) {
	var enabled atomic.Bool
	enabled.Store(true)
	var calls atomic.Int32
	bridge := newTestMemfabBridge(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		calls.Add(1)
		_, _ = w.Write([]byte("{\"version\":1,\"result\":{\"count\":2}}"))
	}))
	_, controller := newTestMCPController(t, bridge, func() (bool, error) {
		return enabled.Load(), nil
	})
	handler := controller.memoryToolHandler("list_memories")
	request := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "list_memories", Arguments: json.RawMessage([]byte("{}"))}}

	result, err := handler(context.Background(), request)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if result.IsError || len(result.Content) != 1 {
		t.Fatalf("unexpected MCP result: %#v", result)
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok || text.Text != "{\"count\":2}" {
		t.Fatalf("unexpected text content: %#v", result.Content)
	}
	structured, ok := result.StructuredContent.(map[string]any)
	if !ok || structured["result"] != "{\"count\":2}" {
		t.Fatalf("unexpected structured content: %#v", result.StructuredContent)
	}

	enabled.Store(false)
	result, err = handler(context.Background(), request)
	if err != nil || !result.IsError || calls.Load() != 1 {
		t.Fatalf("disabled in-flight call result=%#v err=%v calls=%d", result, err, calls.Load())
	}
}

func TestMCPControllerSerializesConcurrentReconciliationAndCalls(t *testing.T) {
	var enabled atomic.Bool
	enabled.Store(true)
	bridge := newTestMemfabBridge(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/tools":
			_ = json.NewEncoder(w).Encode(memfabToolsResponse{Version: 1, Tools: testMemfabDescriptors()})
		case "/v1/tools/call":
			_, _ = w.Write([]byte("{\"version\":1,\"result\":{\"ok\":true}}"))
		default:
			http.NotFound(w, r)
		}
	}))
	server, controller := newTestMCPController(t, bridge, func() (bool, error) {
		return enabled.Load(), nil
	})
	handler := controller.memoryToolHandler("search")
	request := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "search", Arguments: json.RawMessage([]byte("{}"))}}

	var wg sync.WaitGroup
	for index := 0; index < 25; index++ {
		wg.Add(2)
		go func(index int) {
			defer wg.Done()
			if index%5 == 0 {
				enabled.Store(false)
			} else {
				enabled.Store(true)
			}
			_ = controller.reconcile(context.Background())
		}(index)
		go func() {
			defer wg.Done()
			_, _ = handler(context.Background(), request)
		}()
	}
	wg.Wait()
	enabled.Store(true)
	if err := controller.reconcile(context.Background()); err != nil {
		t.Fatalf("final reconcile: %v", err)
	}
	want := append([]string{"control"}, testMemfabToolNames...)
	assertMCPToolSet(t, server, want)
}

func TestMemfabBridgeRejectsInvalidHTTPContracts(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		contentType string
		body        string
	}{
		{name: "wrong content type", status: 200, contentType: "text/plain", body: "{\"version\":1,\"tools\":[]}"},
		{name: "unknown field", status: 200, contentType: "application/json", body: "{\"version\":1,\"tools\":[],\"future\":true}"},
		{name: "wrong version", status: 200, contentType: "application/json", body: "{\"version\":2,\"tools\":[]}"},
		{name: "duplicate field", status: 200, contentType: "application/json", body: "{\"version\":1,\"version\":1,\"tools\":[]}"},
		{name: "multiple JSON values", status: 200, contentType: "application/json", body: "{\"version\":1,\"tools\":[]} {}"},
		{name: "unexpected status", status: 503, contentType: "application/json", body: "{\"version\":1,\"error\":{\"code\":\"unavailable\",\"message\":\"hidden\"}}"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bridge := newTestMemfabBridge(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", test.contentType)
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			if _, err := bridge.listTools(context.Background()); err == nil {
				t.Fatal("expected invalid contract error")
			}
		})
	}
}

func TestMemfabBridgeRejectsInvalidCallContracts(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "success with error field", status: http.StatusOK, body: `{"version":1,"result":{},"error":null}`},
		{name: "success with duplicate result", status: http.StatusOK, body: `{"version":1,"result":{},"result":{}}`},
		{name: "error with result field", status: http.StatusInternalServerError, body: `{"version":1,"error":{"code":"tool_failed","message":"memory tool failed"},"result":null}`},
		{name: "error missing code", status: http.StatusInternalServerError, body: `{"version":1,"error":{"code":"","message":"memory tool failed"}}`},
		{name: "unexpected status", status: http.StatusServiceUnavailable, body: `{"version":1,"error":{"code":"unavailable","message":"memory tool unavailable"}}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bridge := newTestMemfabBridge(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			if _, err := bridge.callTool(context.Background(), "search", json.RawMessage([]byte(`{}`))); err == nil {
				t.Fatal("expected invalid call contract error")
			}
		})
	}
}

func TestMemfabBridgeRejectsDuplicateArgumentKeysBeforeBackend(t *testing.T) {
	var calls atomic.Int32
	bridge := newTestMemfabBridge(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, "must not be reached", http.StatusInternalServerError)
	}))
	if _, err := bridge.callTool(context.Background(), "search", json.RawMessage([]byte(`{"query":"one","query":"two"}`))); err == nil {
		t.Fatal("expected duplicate argument key error")
	}
	if calls.Load() != 0 {
		t.Fatalf("invalid arguments reached backend %d times", calls.Load())
	}
}

func testMemfabDescriptors() []memfabToolDescriptor {
	tools := make([]memfabToolDescriptor, 0, len(testMemfabToolNames))
	for _, name := range testMemfabToolNames {
		tools = append(tools, memfabToolDescriptor{
			Name:        name,
			Description: "Read-only memory tool.",
			InputSchema: json.RawMessage([]byte("{\"type\":\"object\",\"properties\":{}}")),
		})
	}
	return tools
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func newTestMemfabBridge(handler http.Handler) *memfabBridge {
	return &memfabBridge{client: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		return recorder.Result(), nil
	})}}
}

func newTestMCPController(t *testing.T, bridge *memfabBridge, memoryEnabled func() (bool, error)) (*mcp.Server, *mcpController) {
	t.Helper()
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1"}, &mcp.ServerOptions{
		Capabilities: &mcp.ServerCapabilities{Tools: &mcp.ToolCapabilities{ListChanged: true}},
	})
	server.AddTool(&mcp.Tool{Name: "control", InputSchema: json.RawMessage([]byte("{\"type\":\"object\"}"))}, func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "control"}}}, nil
	})
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)
	controller := newMCPController(server, handler, bridge, memoryEnabled, map[string]struct{}{"control": {}})
	return server, controller
}

func writeTestMCPMatrixState(t *testing.T, mcpEnabled, memoryEnabled bool) {
	t.Helper()
	state := defaultServiceStateSnapshot()
	state.Revision = 1
	state.Services[aiServiceMCP] = ServiceStateEntry{Enabled: mcpEnabled, Source: "test", UpdatedAt: "2026-08-06T00:00:00Z"}
	state.Services[aiServiceMemoryFabric] = ServiceStateEntry{Enabled: memoryEnabled, Source: "test", UpdatedAt: "2026-08-06T00:00:00Z"}
	if err := saveServiceStateToPath(serviceStateFilePath, state); err != nil {
		t.Fatalf("save service state: %v", err)
	}
	defaultServiceStateManagerMu.Lock()
	defaultServiceStateManager = nil
	defaultServiceStateManagerMu.Unlock()
}

func connectTestMCPHTTPClient(t *testing.T, handler http.Handler, token string) (*mcp.ClientSession, error) {
	t.Helper()
	httpClient := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if token != "" {
			request.Header.Set("Authorization", "Bearer "+token)
		}
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		return recorder.Result(), nil
	})}
	transport := &mcp.StreamableClientTransport{
		Endpoint:             "http://nanokvm.test/api/mcp",
		HTTPClient:           httpClient,
		MaxRetries:           -1,
		DisableStandaloneSSE: true,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1"}, nil)
	return client.Connect(ctx, transport, nil)
}

func assertMCPToolSet(t *testing.T, server *mcp.Server, want []string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("connect server: %v", err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect client: %v", err)
	}
	defer clientSession.Close()
	result, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	got := make(map[string]struct{}, len(result.Tools))
	for _, tool := range result.Tools {
		got[tool.Name] = struct{}{}
	}
	if len(got) != len(want) {
		t.Fatalf("tool set size=%d want=%d tools=%#v", len(got), len(want), got)
	}
	for _, name := range want {
		if _, ok := got[name]; !ok {
			t.Fatalf("tool %q missing from %#v", name, got)
		}
	}
}

func TestMemfabBridgeUnavailable(t *testing.T) {
	bridge := &memfabBridge{client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("dial failed: %w", errors.New("hidden socket detail"))
	})}}
	_, err := bridge.listTools(context.Background())
	if err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("expected bounded unavailable error, got %v", err)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("backend error leaked transport detail: %v", err)
	}
}
