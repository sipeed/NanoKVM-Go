package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"memfab/internal/config"
	"memfab/internal/paths"
)

const defaultBaseURL = "https://api.openai.com/v1"
const defaultTimeout = 60 * time.Second
const maxResponseBytes = 4 << 20

var traceMu sync.Mutex

type Message struct {
	Role       string     `json:"role"`
	Name       string     `json:"name,omitempty"`
	Content    string     `json:"content,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters"`
}

type toolWrapper struct {
	Type     string         `json:"type"`
	Function ToolDefinition `json:"function"`
}

type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	} `json:"function"`
}

type ToolResponse struct {
	Content   string
	ToolCalls []ToolCall
}

type chatRequest struct {
	Model          string         `json:"model"`
	Messages       []Message      `json:"messages"`
	MaxTokens      int            `json:"max_tokens,omitempty"`
	ResponseFormat map[string]any `json:"response_format,omitempty"`
	Tools          []toolWrapper  `json:"tools,omitempty"`
	ToolChoice     string         `json:"tool_choice,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content   string     `json:"content"`
			ToolCalls []ToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

type traceEntry struct {
	Timestamp            string          `json:"timestamp"`
	Stage                string          `json:"stage"`
	Method               string          `json:"method"`
	URL                  string          `json:"url"`
	Model                string          `json:"model"`
	DurationMS           int64           `json:"duration_ms"`
	StatusCode           int             `json:"status_code"`
	AuthorizationPresent bool            `json:"authorization_present"`
	Request              json.RawMessage `json:"request"`
	ResponseRaw          string          `json:"response_raw"`
	Error                string          `json:"error,omitempty"`
}

func CallJSON(ctx context.Context, cfg config.ModelConfig, messages []Message) (string, error) {
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		return "", errors.New("model is required")
	}
	payload := chatRequest{
		Model:          model,
		Messages:       messages,
		MaxTokens:      cfg.MaxTokens,
		ResponseFormat: map[string]any{"type": "json_object"},
	}
	return callContent(ctx, cfg, payload)
}

func CallText(ctx context.Context, cfg config.ModelConfig, messages []Message) (string, error) {
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		return "", errors.New("model is required")
	}
	payload := chatRequest{Model: model, Messages: messages, MaxTokens: cfg.MaxTokens}
	return callContent(ctx, cfg, payload)
}

func callContent(ctx context.Context, cfg config.ModelConfig, payload chatRequest) (string, error) {
	parsed, err := doChat(ctx, cfg, payload)
	if err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
		return "", errors.New("llm response missing content")
	}
	return parsed.Choices[0].Message.Content, nil
}

func CallTools(ctx context.Context, cfg config.ModelConfig, messages []Message, tools []ToolDefinition) (ToolResponse, error) {
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		return ToolResponse{}, errors.New("model is required")
	}
	wrappers := make([]toolWrapper, 0, len(tools))
	for _, tool := range tools {
		wrappers = append(wrappers, toolWrapper{Type: "function", Function: tool})
	}
	payload := chatRequest{
		Model:      model,
		Messages:   messages,
		MaxTokens:  cfg.MaxTokens,
		Tools:      wrappers,
		ToolChoice: "auto",
	}
	parsed, err := doChat(ctx, cfg, payload)
	if err != nil {
		return ToolResponse{}, err
	}
	if len(parsed.Choices) == 0 {
		return ToolResponse{}, errors.New("llm response missing choices")
	}
	msg := parsed.Choices[0].Message
	return ToolResponse{Content: msg.Content, ToolCalls: msg.ToolCalls}, nil
}

func ToolArguments(call ToolCall, dest any) error {
	data := bytes.TrimSpace(call.Function.Arguments)
	if len(data) == 0 {
		data = []byte("{}")
	}
	if len(data) > 0 && data[0] == '"' {
		var text string
		if err := json.Unmarshal(data, &text); err != nil {
			return err
		}
		text = strings.TrimSpace(text)
		if text == "" {
			text = "{}"
		}
		data = []byte(text)
	}
	return json.Unmarshal(data, dest)
}

func doChat(ctx context.Context, cfg config.ModelConfig, payload chatRequest) (chatResponse, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	url := baseURL + "/chat/completions"
	body, err := json.Marshal(payload)
	if err != nil {
		return chatResponse{}, err
	}
	started := time.Now()
	trace := func(statusCode int, responseRaw string, err error) {
		recordLLMTrace(cfg, traceEntry{
			Timestamp:            time.Now().Format(time.RFC3339Nano),
			Stage:                stageName(cfg.Stage),
			Method:               http.MethodPost,
			URL:                  url,
			Model:                payload.Model,
			DurationMS:           time.Since(started).Milliseconds(),
			StatusCode:           statusCode,
			AuthorizationPresent: strings.TrimSpace(cfg.ResolvedAPIKey()) != "",
			Request:              append(json.RawMessage(nil), body...),
			ResponseRaw:          responseRaw,
			Error:                errorText(err),
		})
	}
	timeout := defaultTimeout
	if cfg.TimeoutSeconds > 0 {
		timeout = time.Duration(cfg.TimeoutSeconds) * time.Second
	}
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		trace(0, "", err)
		return chatResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey := cfg.ResolvedAPIKey(); apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		trace(0, "", err)
		return chatResponse{}, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		trace(resp.StatusCode, "", err)
		return chatResponse{}, err
	}
	responseRaw := string(data)
	var parsed chatResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		decodeErr := fmt.Errorf("decode llm response: %w", err)
		trace(resp.StatusCode, responseRaw, decodeErr)
		return chatResponse{}, decodeErr
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var httpErr error
		if parsed.Error != nil && parsed.Error.Message != "" {
			httpErr = fmt.Errorf("llm http %d: %s", resp.StatusCode, parsed.Error.Message)
		} else {
			httpErr = fmt.Errorf("llm http %d", resp.StatusCode)
		}
		trace(resp.StatusCode, responseRaw, httpErr)
		return chatResponse{}, httpErr
	}
	trace(resp.StatusCode, responseRaw, nil)
	return parsed, nil
}

func recordLLMTrace(cfg config.ModelConfig, entry traceEntry) {
	if !cfg.Trace.Enabled {
		return
	}
	path := strings.TrimSpace(cfg.Trace.Path)
	if path == "" {
		path = filepath.Join(paths.LogsDir(), "llm-trace.jsonl")
	}
	data, err := json.Marshal(entry)
	if err != nil {
		log.Printf("llm trace marshal failed: %v", err)
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), paths.DirectoryMode); err != nil {
		log.Printf("llm trace mkdir failed: %v", err)
		return
	}
	traceMu.Lock()
	defer traceMu.Unlock()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		log.Printf("llm trace open failed: %v", err)
		return
	}
	defer f.Close()
	if _, err := f.Write(append(data, '\n')); err != nil {
		log.Printf("llm trace write failed: %v", err)
	}
}

func stageName(stage string) string {
	stage = strings.TrimSpace(stage)
	if stage == "" {
		return "unknown"
	}
	return stage
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
