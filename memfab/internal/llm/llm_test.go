package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"memfab/internal/config"
)

func TestCallJSONSendsModelVerbatim(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("auth=%q", got)
		}
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req["model"] != "qwen-portal/qwen3.6-plus" {
			t.Fatalf("model=%v", req["model"])
		}
		if rf, ok := req["response_format"].(map[string]any); !ok || rf["type"] != "json_object" {
			t.Fatalf("response_format=%v", req["response_format"])
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"entries\":[\"ok\"]}"}}]}`))
	}))
	defer server.Close()

	out, err := CallJSON(context.Background(), config.ModelConfig{Model: "qwen-portal/qwen3.6-plus", BaseURL: server.URL + "/v1", APIKey: "test-key"}, []Message{{Role: "user", Content: "hi"}})
	if err != nil {
		t.Fatal(err)
	}
	if out != `{"entries":["ok"]}` {
		t.Fatalf("out=%q", out)
	}
}

func TestCallJSONHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"bad key"}}`))
	}))
	defer server.Close()

	_, err := CallJSON(context.Background(), config.ModelConfig{Model: "gpt-test", BaseURL: server.URL}, []Message{{Role: "user", Content: "hi"}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCallToolsSendsToolDefinitions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req["model"] != "qwen-portal/qwen3.6-plus" {
			t.Fatalf("model=%v", req["model"])
		}
		if req["tool_choice"] != "auto" {
			t.Fatalf("tool_choice=%v", req["tool_choice"])
		}
		tools, ok := req["tools"].([]any)
		if !ok || len(tools) != 1 {
			t.Fatalf("tools=%v", req["tools"])
		}
		messages := req["messages"].([]any)
		toolMsg := messages[1].(map[string]any)
		if toolMsg["role"] != "tool" || toolMsg["name"] != "commit" || toolMsg["tool_call_id"] != "call_1" {
			t.Fatalf("tool message=%v", toolMsg)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"","tool_calls":[{"id":"call_2","type":"function","function":{"name":"commit","arguments":"{\"summary\":\"done\"}"}}]}}]}`))
	}))
	defer server.Close()

	out, err := CallTools(context.Background(), config.ModelConfig{Model: "qwen-portal/qwen3.6-plus", BaseURL: server.URL + "/v1"}, []Message{{Role: "user", Content: "hi"}, {Role: "tool", Name: "commit", ToolCallID: "call_1", Content: `{"ok":true}`}}, []ToolDefinition{{Name: "commit", Parameters: map[string]any{"type": "object"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.ToolCalls) != 1 || out.ToolCalls[0].Function.Name != "commit" {
		t.Fatalf("out=%+v", out)
	}
	var args struct {
		Summary string `json:"summary"`
	}
	if err := ToolArguments(out.ToolCalls[0], &args); err != nil || args.Summary != "done" {
		t.Fatalf("args=%+v err=%v", args, err)
	}
}

func TestToolArgumentsAcceptsObjectArguments(t *testing.T) {
	var call ToolCall
	call.Function.Name = "commit"
	call.Function.Arguments = []byte(`{"summary":"object"}`)
	var args struct {
		Summary string `json:"summary"`
	}
	if err := ToolArguments(call, &args); err != nil || args.Summary != "object" {
		t.Fatalf("args=%+v err=%v", args, err)
	}
}

func TestCallTextDoesNotSendResponseFormat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("auth=%q", got)
		}
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req["model"] != "qwen-portal/qwen3.6-plus" {
			t.Fatalf("model=%v", req["model"])
		}
		if _, ok := req["response_format"]; ok {
			t.Fatalf("unexpected response_format=%v", req["response_format"])
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"---\ndescription: Test\n---\n"}}]}`))
	}))
	defer server.Close()
	out, err := CallText(context.Background(), config.ModelConfig{Model: "qwen-portal/qwen3.6-plus", BaseURL: server.URL + "/v1"}, []Message{{Role: "user", Content: "markdown"}})
	if err != nil {
		t.Fatal(err)
	}
	if out != "---\ndescription: Test\n---\n" {
		t.Fatalf("out=%q", out)
	}
}

func TestLLMTraceDisabledDoesNotCreateFile(t *testing.T) {
	tracePath := t.TempDir() + "/trace.jsonl"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"entries\":[\"ok\"]}"}}]}`))
	}))
	defer server.Close()

	_, err := CallJSON(context.Background(), config.ModelConfig{
		Model:   "gpt-test",
		BaseURL: server.URL + "/v1",
		Trace:   config.LLMTraceConfig{Enabled: false, Path: tracePath},
	}, []Message{{Role: "user", Content: "hi"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(tracePath); !os.IsNotExist(err) {
		t.Fatalf("trace file stat err=%v, want not exist", err)
	}
}

func TestLLMTraceWritesFullRequestAndRawResponse(t *testing.T) {
	tracePath := t.TempDir() + "/trace.jsonl"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"entries\":[\"visible result\"]}"}}]}`))
	}))
	defer server.Close()

	_, err := CallJSON(context.Background(), config.ModelConfig{
		Model:   "qwen-portal/qwen3.6-plus",
		BaseURL: server.URL + "/v1",
		APIKey:  "secret-key",
		Stage:   "timeline",
		Trace:   config.LLMTraceConfig{Enabled: true, Path: tracePath},
	}, []Message{{Role: "user", Content: "visible OCR prompt"}})
	if err != nil {
		t.Fatal(err)
	}
	text := readTraceFile(t, tracePath)
	if strings.Contains(text, "secret-key") || strings.Contains(text, "Bearer") {
		t.Fatalf("trace leaked authorization: %s", text)
	}
	entry := oneTraceEntry(t, tracePath)
	if entry["stage"] != "timeline" || entry["method"] != "POST" || entry["model"] != "qwen-portal/qwen3.6-plus" {
		t.Fatalf("entry metadata=%+v", entry)
	}
	if entry["status_code"].(float64) != 200 || entry["authorization_present"] != true {
		t.Fatalf("entry status/auth=%+v", entry)
	}
	request := entry["request"].(map[string]any)
	messages := request["messages"].([]any)
	if !strings.Contains(messages[0].(map[string]any)["content"].(string), "visible OCR prompt") {
		t.Fatalf("request missing prompt: %+v", request)
	}
	if !strings.Contains(entry["response_raw"].(string), "visible result") {
		t.Fatalf("response_raw=%q", entry["response_raw"])
	}
	if _, ok := entry["error"]; ok {
		t.Fatalf("unexpected error in trace: %+v", entry)
	}
}

func TestLLMTraceWritesHTTPError(t *testing.T) {
	tracePath := t.TempDir() + "/trace.jsonl"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"bad key"}}`))
	}))
	defer server.Close()

	_, err := CallJSON(context.Background(), config.ModelConfig{
		Model:   "gpt-test",
		BaseURL: server.URL + "/v1",
		Stage:   "reducer",
		Trace:   config.LLMTraceConfig{Enabled: true, Path: tracePath},
	}, []Message{{Role: "user", Content: "hi"}})
	if err == nil {
		t.Fatal("expected error")
	}
	entry := oneTraceEntry(t, tracePath)
	if entry["status_code"].(float64) != 401 || !strings.Contains(entry["error"].(string), "bad key") {
		t.Fatalf("entry=%+v", entry)
	}
	if !strings.Contains(entry["response_raw"].(string), "bad key") {
		t.Fatalf("response_raw=%q", entry["response_raw"])
	}
}

func TestLLMTraceWritesDecodeError(t *testing.T) {
	tracePath := t.TempDir() + "/trace.jsonl"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer server.Close()

	_, err := CallJSON(context.Background(), config.ModelConfig{
		Model:   "gpt-test",
		BaseURL: server.URL + "/v1",
		Stage:   "timeline",
		Trace:   config.LLMTraceConfig{Enabled: true, Path: tracePath},
	}, []Message{{Role: "user", Content: "hi"}})
	if err == nil {
		t.Fatal("expected error")
	}
	entry := oneTraceEntry(t, tracePath)
	if entry["status_code"].(float64) != 200 || !strings.Contains(entry["error"].(string), "decode llm response") {
		t.Fatalf("entry=%+v", entry)
	}
	if entry["response_raw"] != "not-json" {
		t.Fatalf("response_raw=%q", entry["response_raw"])
	}
}

func TestLLMTraceWritesNetworkError(t *testing.T) {
	tracePath := t.TempDir() + "/trace.jsonl"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	baseURL := server.URL + "/v1"
	server.Close()

	_, err := CallJSON(context.Background(), config.ModelConfig{
		Model:   "gpt-test",
		BaseURL: baseURL,
		Stage:   "compact",
		Trace:   config.LLMTraceConfig{Enabled: true, Path: tracePath},
	}, []Message{{Role: "user", Content: "hi"}})
	if err == nil {
		t.Fatal("expected error")
	}
	entry := oneTraceEntry(t, tracePath)
	if entry["status_code"].(float64) != 0 || entry["response_raw"] != "" || entry["error"] == "" {
		t.Fatalf("entry=%+v", entry)
	}
}

func readTraceFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func oneTraceEntry(t *testing.T, path string) map[string]any {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(readTraceFile(t, path)), "\n")
	if len(lines) != 1 {
		t.Fatalf("trace lines=%d: %q", len(lines), strings.Join(lines, "\n"))
	}
	var entry map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatal(err)
	}
	return entry
}
