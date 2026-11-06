package mcp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"memfab/internal/config"
)

func performToolRequest(t *testing.T, handler http.Handler, method, path, body, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}

func TestToolListWireContract(t *testing.T) {
	db := mcpTestDB(t)
	response := performToolRequest(t, Handler(db, testConfig()), http.MethodGet, "/v1/tools", "", "")
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type=%q", got)
	}
	var decoded struct {
		Version int              `json:"version"`
		Tools   []toolDescriptor `json:"tools"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Version != 1 || len(decoded.Tools) != 8 {
		t.Fatalf("response=%s", response.Body.String())
	}
	wantNames := []string{"list_memories", "read_memory", "search", "recent_activity", "read_recent_capture", "search_captures", "current_context", "get_schema"}
	seen := map[string]bool{}
	for i, tool := range decoded.Tools {
		if tool.Name != wantNames[i] {
			t.Fatalf("tool[%d]=%q, want %q", i, tool.Name, wantNames[i])
		}
		if seen[tool.Name] || len(tool.Name) > maxToolNameBytes || tool.Description == "" {
			t.Fatalf("invalid descriptor: %+v", tool)
		}
		seen[tool.Name] = true
		if tool.InputSchema["type"] != "object" {
			t.Fatalf("%s schema=%#v", tool.Name, tool.InputSchema)
		}
		if _, ok := tool.InputSchema["properties"].(map[string]any); !ok {
			t.Fatalf("%s properties=%T", tool.Name, tool.InputSchema["properties"])
		}
	}
	if strings.Contains(response.Body.String(), "jsonrpc") || strings.Contains(response.Body.String(), "Mcp-Session-Id") {
		t.Fatalf("retired MCP transport leaked into contract: %s", response.Body.String())
	}
}

func TestToolCallWireContractForAllTools(t *testing.T) {
	db := mcpTestDB(t)
	handler := Handler(db, testConfig())
	cases := map[string]string{
		"list_memories":       `{}`,
		"read_memory":         `{"path":"missing"}`,
		"search":              `{"query":"not-present"}`,
		"recent_activity":     `{}`,
		"read_recent_capture": `{}`,
		"search_captures":     `{"query":"not-present"}`,
		"current_context":     `{}`,
		"get_schema":          `{}`,
	}
	for name, arguments := range cases {
		t.Run(name, func(t *testing.T) {
			body := `{"version":1,"name":` + quoteJSON(t, name) + `,"arguments":` + arguments + `}`
			response := performToolRequest(t, handler, http.MethodPost, "/v1/tools/call", body, "application/json")
			if response.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			var decoded struct {
				Version int             `json:"version"`
				Result  json.RawMessage `json:"result"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded.Version != 1 || len(decoded.Result) == 0 || !json.Valid(decoded.Result) {
				t.Fatalf("response=%s", response.Body.String())
			}
		})
	}
}

func TestToolCallRejectsInvalidAndOversizedRequests(t *testing.T) {
	db := mcpTestDB(t)
	handler := Handler(db, testConfig())
	tests := []struct {
		name        string
		body        string
		contentType string
	}{
		{name: "missing content type", body: `{"version":1,"name":"get_schema","arguments":{}}`},
		{name: "wrong content type", body: `{"version":1,"name":"get_schema","arguments":{}}`, contentType: "text/plain"},
		{name: "invalid JSON", body: `{`, contentType: "application/json"},
		{name: "unknown field", body: `{"version":1,"name":"get_schema","arguments":{},"extra":true}`, contentType: "application/json"},
		{name: "duplicate envelope field", body: `{"version":1,"name":"get_schema","name":"search","arguments":{}}`, contentType: "application/json"},
		{name: "duplicate argument field", body: `{"version":1,"name":"search","arguments":{"query":"first","query":"second"}}`, contentType: "application/json"},
		{name: "wrong version", body: `{"version":2,"name":"get_schema","arguments":{}}`, contentType: "application/json"},
		{name: "missing arguments", body: `{"version":1,"name":"get_schema"}`, contentType: "application/json"},
		{name: "null arguments", body: `{"version":1,"name":"get_schema","arguments":null}`, contentType: "application/json"},
		{name: "array arguments", body: `{"version":1,"name":"get_schema","arguments":[]}`, contentType: "application/json"},
		{name: "trailing JSON", body: `{"version":1,"name":"get_schema","arguments":{}} {}`, contentType: "application/json"},
		{name: "long name", body: `{"version":1,"name":"` + strings.Repeat("x", maxToolNameBytes+1) + `","arguments":{}}`, contentType: "application/json"},
		{name: "oversized", body: `{"version":1,"name":"get_schema","arguments":{"padding":"` + strings.Repeat("x", maxToolRequestBytes) + `"}}`, contentType: "application/json"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			response := performToolRequest(t, handler, http.MethodPost, "/v1/tools/call", tc.body, tc.contentType)
			assertWireError(t, response, http.StatusBadRequest, "invalid_request", "invalid tool request")
			if strings.Contains(response.Body.String(), "padding") || strings.Contains(response.Body.String(), "unknown field") {
				t.Fatalf("request detail leaked: %s", response.Body.String())
			}
		})
	}
}

func TestToolCallUnknownAndFailureErrorsAreSanitized(t *testing.T) {
	db := mcpTestDB(t)
	handler := Handler(db, testConfig())
	unknown := performToolRequest(t, handler, http.MethodPost, "/v1/tools/call", `{"version":1,"name":"delete_everything","arguments":{}}`, "application/json")
	assertWireError(t, unknown, http.StatusNotFound, "unknown_tool", "unknown memory tool")

	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	failed := performToolRequest(t, handler, http.MethodPost, "/v1/tools/call", `{"version":1,"name":"current_context","arguments":{}}`, "application/json")
	assertWireError(t, failed, http.StatusInternalServerError, "tool_failed", "memory tool failed")
	for _, leaked := range []string{"sql", "closed", "index.db", "socket"} {
		if strings.Contains(strings.ToLower(failed.Body.String()), leaked) {
			t.Fatalf("error leaked %q: %s", leaked, failed.Body.String())
		}
	}
}

func TestDescriptorValidationRejectsInvalidContracts(t *testing.T) {
	valid := toolsList()
	if _, err := validateToolDescriptors(valid); err != nil {
		t.Fatalf("valid descriptors rejected: %v", err)
	}

	tests := map[string][]toolDescriptor{
		"empty":        {},
		"duplicate":    {valid[0], valid[0]},
		"long name":    {{Name: strings.Repeat("x", maxToolNameBytes+1), Description: "x", InputSchema: map[string]any{"type": "object"}}},
		"non-object":   {{Name: "bad_schema", Description: "x", InputSchema: map[string]any{"type": "array"}}},
		"large schema": {{Name: "large_schema", Description: "x", InputSchema: map[string]any{"type": "object", "padding": strings.Repeat("x", maxToolSchemaBytes)}}},
	}
	for name, descriptors := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := validateToolDescriptors(descriptors); err == nil {
				t.Fatal("invalid descriptors accepted")
			}
		})
	}
}

func TestWriteJSONBoundsResponse(t *testing.T) {
	response := httptest.NewRecorder()
	writeJSON(response, http.StatusOK, strings.Repeat("x", maxToolResponseBytes))
	assertWireError(t, response, http.StatusInternalServerError, "tool_failed", "memory tool failed")
	if response.Body.Len() >= maxToolResponseBytes {
		t.Fatalf("oversized error response: %d", response.Body.Len())
	}
}

func TestRetiredPublicMCPRouteIsAbsent(t *testing.T) {
	db := mcpTestDB(t)
	response := performToolRequest(t, Handler(db, testConfig()), http.MethodPost, "/mcp", `{}`, "application/json")
	if response.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func assertWireError(t *testing.T, response *httptest.ResponseRecorder, status int, code, message string) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	want, err := json.Marshal(errorResponse{Version: 1, Error: wireError{Code: code, Message: message}})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(response.Body.Bytes(), want) {
		t.Fatalf("body=%s, want %s", response.Body.Bytes(), want)
	}
}

func quoteJSON(t *testing.T, value string) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func testConfig() config.Config {
	return config.Default()
}
