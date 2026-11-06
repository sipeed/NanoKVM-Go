package mcp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"memfab/internal/capture"
	"memfab/internal/config"
	"memfab/internal/memory"
	"memfab/internal/paths"
	"memfab/internal/store"
)

const (
	contractVersion      = 1
	maxToolRequestBytes  = 256 << 10
	maxToolResponseBytes = 4 << 20
	maxToolCount         = 16
	maxToolNameBytes     = 64
	maxToolSchemaBytes   = 64 << 10
)

type toolDescriptor struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type toolCallRequest struct {
	Version   int             `json:"version"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type wireError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorResponse struct {
	Version int       `json:"version"`
	Error   wireError `json:"error"`
}

// Handler serves the private, versioned tool contract. It is deliberately not
// an MCP transport: authentication and MCP sessions terminate in NanoKVM-Server.
func Handler(db *store.DB, cfg config.Config) http.Handler {
	tools := toolsList()
	knownTools, contractErr := validateToolDescriptors(tools)
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/tools", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		if contractErr != nil {
			writeError(w, http.StatusInternalServerError, "invalid_contract", "memory tool contract unavailable")
			return
		}
		writeJSON(w, http.StatusOK, struct {
			Version int              `json:"version"`
			Tools   []toolDescriptor `json:"tools"`
		}{Version: contractVersion, Tools: tools})
	})
	mux.HandleFunc("/v1/tools/call", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || mediaType != "application/json" {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid tool request")
			return
		}

		var req toolCallRequest
		if err := decodeStrictJSON(w, r, &req); err != nil || req.Version != contractVersion || len(req.Name) == 0 || len(req.Name) > maxToolNameBytes || !isJSONObject(req.Arguments) {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid tool request")
			return
		}
		if contractErr != nil {
			writeError(w, http.StatusInternalServerError, "invalid_contract", "memory tool contract unavailable")
			return
		}
		if !knownTools[req.Name] {
			writeError(w, http.StatusNotFound, "unknown_tool", "unknown memory tool")
			return
		}

		result, err := callToolWithConfig(db, cfg, req.Name, req.Arguments)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "tool_failed", "memory tool failed")
			return
		}
		resultJSON, err := json.Marshal(result)
		if err != nil || len(resultJSON) > maxToolResponseBytes {
			writeError(w, http.StatusInternalServerError, "tool_failed", "memory tool failed")
			return
		}
		writeJSON(w, http.StatusOK, struct {
			Version int             `json:"version"`
			Result  json.RawMessage `json:"result"`
		}{Version: contractVersion, Result: resultJSON})
	})
	return mux
}

func decodeStrictJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	body := http.MaxBytesReader(w, r.Body, maxToolRequestBytes)
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("trailing JSON data")
	}
	return nil
}

func rejectDuplicateJSONKeys(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	if err := consumeJSONValue(dec); err != nil {
		return err
	}
	if _, err := dec.Token(); err != io.EOF {
		return fmt.Errorf("trailing JSON data")
	}
	return nil
}

func consumeJSONValue(dec *json.Decoder) error {
	token, err := dec.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for dec.More() {
			keyToken, err := dec.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok || seen[key] {
				return errors.New("duplicate or invalid JSON object key")
			}
			seen[key] = true
			if err := consumeJSONValue(dec); err != nil {
				return err
			}
		}
		_, err = dec.Token()
		return err
	case '[':
		for dec.More() {
			if err := consumeJSONValue(dec); err != nil {
				return err
			}
		}
		_, err = dec.Token()
		return err
	default:
		return errors.New("invalid JSON delimiter")
	}
}

func isJSONObject(raw json.RawMessage) bool {
	raw = bytes.TrimSpace(raw)
	if len(raw) < 2 || raw[0] != '{' || raw[len(raw)-1] != '}' {
		return false
	}
	var object map[string]json.RawMessage
	return rejectDuplicateJSONKeys(raw) == nil && json.Unmarshal(raw, &object) == nil
}

func validateToolDescriptors(tools []toolDescriptor) (map[string]bool, error) {
	if len(tools) == 0 || len(tools) > maxToolCount {
		return nil, errors.New("invalid tool count")
	}
	names := make(map[string]bool, len(tools))
	for _, tool := range tools {
		if tool.Name == "" || len(tool.Name) > maxToolNameBytes || strings.TrimSpace(tool.Name) != tool.Name || names[tool.Name] {
			return nil, errors.New("invalid or duplicate tool name")
		}
		if tool.Description == "" || tool.InputSchema == nil || tool.InputSchema["type"] != "object" {
			return nil, errors.New("invalid tool descriptor")
		}
		schema, err := json.Marshal(tool.InputSchema)
		if err != nil || len(schema) > maxToolSchemaBytes {
			return nil, errors.New("invalid tool schema")
		}
		names[tool.Name] = true
	}
	return names, nil
}

func toolsList() []toolDescriptor {
	mk := func(name, desc string, props map[string]any) toolDescriptor {
		return toolDescriptor{Name: name, Description: desc, InputSchema: map[string]any{"type": "object", "properties": props}}
	}
	return []toolDescriptor{
		mk("list_memories", "List memory files.", map[string]any{"include_dormant": map[string]string{"type": "boolean"}, "include_archived": map[string]string{"type": "boolean"}}),
		mk("read_memory", "Read a memory file.", map[string]any{"path": map[string]string{"type": "string"}, "since": map[string]string{"type": "string"}, "until": map[string]string{"type": "string"}, "tags": map[string]any{"type": "array", "items": map[string]string{"type": "string"}}, "tail_n": map[string]string{"type": "number"}}),
		mk("search", "Search compressed memory.", map[string]any{"query": map[string]string{"type": "string"}, "paths": map[string]any{"type": "array", "items": map[string]string{"type": "string"}}, "since": map[string]string{"type": "string"}, "until": map[string]string{"type": "string"}, "top_k": map[string]string{"type": "number"}, "include_superseded": map[string]string{"type": "boolean"}}),
		mk("recent_activity", "Recent memory activity.", map[string]any{"since": map[string]string{"type": "string"}, "limit": map[string]string{"type": "number"}, "prefix_filter": map[string]any{"type": "array", "items": map[string]string{"type": "string"}}, "include_superseded": map[string]string{"type": "boolean"}}),
		mk("read_recent_capture", "Read a recent OCR-backed capture observation.", map[string]any{"at": map[string]string{"type": "string"}, "app_name": map[string]string{"type": "string"}, "title": map[string]string{"type": "string"}, "include_raw_text": map[string]string{"type": "boolean"}, "include_blocks": map[string]string{"type": "boolean"}}),
		mk("search_captures", "Search OCR-backed capture observations.", map[string]any{"query": map[string]string{"type": "string"}}),
		mk("current_context", "Current OCR-backed screen context.", map[string]any{}),
		mk("get_schema", "Return memory schema.", map[string]any{}),
	}
}

func callTool(db *store.DB, name string, raw json.RawMessage) (any, error) {
	return callToolWithConfig(db, config.Default(), name, raw)
}

func callToolWithConfig(db *store.DB, cfg config.Config, name string, raw json.RawMessage) (any, error) {
	switch name {
	case "search_captures":
		var args struct {
			Query, Since, Until, AppName string
			Limit                        int
		}
		_ = json.Unmarshal(raw, &args)
		hits, err := db.SearchObservations(args.Query, args.Since, args.Until, args.AppName, args.Limit)
		if err != nil {
			return nil, err
		}
		results := make([]map[string]any, 0, len(hits))
		for _, h := range hits {
			results = append(results, map[string]any{
				"captured_at": h.CapturedAt, "file_stem": h.ID, "source_kind": h.SourceKind,
				"source_device": h.SourceDevice, "source_engine": h.SourceEngine, "app_name": h.AppName,
				"title": h.Title, "url": h.URL, "event_type": h.EventType, "event_reason": h.EventReason,
				"language": h.Language, "confidence": h.Confidence, "block_count": h.BlockCount,
				"tags": h.Tags, "snippet": h.Snippet, "rank": h.Rank,
			})
		}
		return map[string]any{"query": args.Query, "results": results}, nil
	case "current_context":
		recent, err := db.RecentObservations(5)
		if err != nil {
			return nil, err
		}
		blocks, err := db.ListTimelineBlocks(8)
		if err != nil {
			return nil, err
		}
		return buildCurrentObservationContext(recent, blocks), nil
	case "list_memories":
		var args struct {
			IncludeDormant  bool `json:"include_dormant"`
			IncludeArchived bool `json:"include_archived"`
		}
		_ = json.Unmarshal(raw, &args)
		files, err := db.ListMemoryFiles(args.IncludeDormant, args.IncludeArchived)
		if err != nil {
			return nil, err
		}
		return map[string]any{"count": len(files), "files": files}, nil
	case "read_memory":
		var args struct {
			Path  string   `json:"path"`
			Since string   `json:"since"`
			Until string   `json:"until"`
			Tags  []string `json:"tags"`
			TailN int      `json:"tail_n"`
		}
		_ = json.Unmarshal(raw, &args)
		return readMemoryTool(args.Path, args.Since, args.Until, args.Tags, args.TailN)
	case "search":
		var args struct {
			Query             string   `json:"query"`
			Paths             []string `json:"paths"`
			Since             string   `json:"since"`
			Until             string   `json:"until"`
			TopK              int      `json:"top_k"`
			IncludeSuperseded bool     `json:"include_superseded"`
		}
		_ = json.Unmarshal(raw, &args)
		hits, err := db.SearchMemory(args.Query, args.Paths, args.Since, args.Until, args.TopK, args.IncludeSuperseded)
		if err != nil {
			return nil, err
		}
		return map[string]any{"query": args.Query, "results": hits}, nil
	case "recent_activity":
		var args struct {
			Since             string   `json:"since"`
			Limit             int      `json:"limit"`
			PrefixFilter      []string `json:"prefix_filter"`
			IncludeSuperseded bool     `json:"include_superseded"`
		}
		_ = json.Unmarshal(raw, &args)
		hits, err := db.RecentMemory(args.Since, args.Limit, args.PrefixFilter, args.IncludeSuperseded)
		if err != nil {
			return nil, err
		}
		return map[string]any{"count": len(hits), "entries": hits}, nil
	case "read_recent_capture":
		var args struct {
			At             string `json:"at"`
			AppName        string `json:"app_name"`
			Title          string `json:"title"`
			IncludeRawText bool   `json:"include_raw_text"`
			IncludeBlocks  bool   `json:"include_blocks"`
		}
		_ = json.Unmarshal(raw, &args)
		stem, found, err := db.FindRecentObservationStem(args.At, args.AppName, args.Title)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, nil
		}
		obs, err := capture.ReadFile(filepath.Join(paths.ObservationBufferDir(), stem+".json"))
		if err != nil {
			return nil, err
		}
		if !args.IncludeRawText {
			obs.OCR.RawText = ""
		}
		if !args.IncludeBlocks {
			obs.OCR.Blocks = nil
		}
		return obs, nil
	case "get_schema":
		return map[string]any{"schema": memory.SchemaText()}, nil
	default:
		return nil, fmt.Errorf("unknown tool %s", name)
	}
}

func readMemoryTool(name, since, until string, tags []string, tailN int) (map[string]any, error) {
	path, err := memory.MemoryPath(paths.MemoryDir(), name)
	if err != nil {
		return map[string]any{"error": err.Error()}, nil
	}
	parsed, err := memory.ReadFile(path)
	if err != nil {
		return map[string]any{"error": fmt.Sprintf("file not found: %s", name)}, nil
	}
	entries := filterMemoryEntries(parsed.Entries, since, until, tags)
	if tailN > 0 && len(entries) > tailN {
		entries = entries[len(entries)-tailN:]
	}
	return map[string]any{
		"path":        parsed.Path,
		"description": parsed.Description,
		"tags":        parsed.Tags,
		"status":      parsed.Status,
		"updated":     parsed.Updated,
		"entry_count": len(entries),
		"entries":     entries,
	}, nil
}

func filterMemoryEntries(entries []memory.ParsedEntry, since, until string, tags []string) []memory.ParsedEntry {
	out := make([]memory.ParsedEntry, 0, len(entries))
	wantTags := map[string]bool{}
	for _, tag := range tags {
		tag = strings.TrimSpace(strings.TrimPrefix(tag, "#"))
		if tag != "" {
			wantTags[tag] = true
		}
	}
	for _, entry := range entries {
		if since != "" && entry.Timestamp < since {
			continue
		}
		if until != "" && entry.Timestamp > until {
			continue
		}
		if len(wantTags) > 0 && !entryHasAnyTag(entry.Tags, wantTags) {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func entryHasAnyTag(tags []string, want map[string]bool) bool {
	for _, tag := range tags {
		if want[strings.TrimPrefix(tag, "#")] {
			return true
		}
	}
	return false
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorResponse{Version: contractVersion, Error: wireError{Code: code, Message: message}})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	data, err := json.Marshal(value)
	if err != nil || len(data) > maxToolResponseBytes {
		status = http.StatusInternalServerError
		data = []byte(`{"version":1,"error":{"code":"tool_failed","message":"memory tool failed"}}`)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}
