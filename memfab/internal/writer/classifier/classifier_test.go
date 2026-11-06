package classifier

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"memfab/internal/config"
	"memfab/internal/llm"
	"memfab/internal/memory"
	"memfab/internal/paths"
	"memfab/internal/store"
)

func classifierTestDB(t *testing.T) *store.DB {
	t.Helper()
	t.Setenv("MEMFAB_ROOT", t.TempDir())
	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func classifierConfig() config.Config {
	return config.Config{
		Classifier: config.ClassifierConfig{Enabled: true, MaxIterations: 8, MaxSessionsPerTick: 4},
		Writer:     config.WriterConfig{SoftLimitTokens: 8000},
		Models:     config.ModelsConfig{Classifier: config.ModelConfig{Model: "gpt-test"}},
	}
}

func insertClassifySession(t *testing.T, db *store.DB, id string) store.SessionRow {
	t.Helper()
	row := store.SessionRow{ID: id, StartTime: "2026-05-14T10:00:00Z", EndTime: "2026-05-14T10:05:00Z", Status: string(store.SessionReduced), FlushEnd: "2026-05-14T10:05:00Z", CreatedAt: "2026-05-14T10:05:00Z", UpdatedAt: "2026-05-14T10:05:00Z"}
	if _, err := db.InsertSession(row); err != nil {
		t.Fatal(err)
	}
	return row
}

func insertClassifyEvent(t *testing.T, sessionID string) {
	t.Helper()
	now := time.Date(2026, 5, 14, 10, 5, 0, 0, time.UTC)
	name := memory.EventDailyName(now)
	if _, err := memory.EnsureEventDailyFile(paths.MemoryDir(), name, now, now); err != nil {
		t.Fatal(err)
	}
	entry := memory.ParsedEntry{ID: "event-entry", Timestamp: now.Format(time.RFC3339), Tags: []string{"event", "session", "sid:" + sessionID}, Body: "Summary: User decided memfab stage 8 uses an LLM classifier tool loop."}
	parsed, err := memory.AppendEntry(paths.MemoryDir(), name, entry, now)
	if err != nil {
		t.Fatal(err)
	}
	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.UpsertMemoryFile(store.FileRow{Path: parsed.Path, Prefix: "event", Description: parsed.Description, Tags: parsed.Tags, Status: parsed.Status, EntryCount: parsed.EntryCount, Created: parsed.Created, Updated: parsed.Updated}); err != nil {
		t.Fatal(err)
	}
	if err := db.InsertMemoryEntry(store.MemoryEntryRow{ID: entry.ID, Path: parsed.Path, Prefix: "event", Timestamp: entry.Timestamp, Tags: strings.Join(entry.Tags, " "), Content: entry.Body}); err != nil {
		t.Fatal(err)
	}
}

func TestClassifySessionWritesDurableMemoryAndAdvancesBookmark(t *testing.T) {
	db := classifierTestDB(t)
	row := insertClassifySession(t, db, "sess_ok")
	insertClassifyEvent(t, row.ID)
	_, err := db.InsertTimelineBlock(store.TimelineBlock{ID: "tl1", StartTime: "2026-05-14T10:00:00Z", EndTime: "2026-05-14T10:05:00Z", Entries: []string{"[Cursor] memfab stage 8 uses an LLM classifier tool loop."}, AppsUsed: []string{"Cursor"}, CaptureCount: 1, CreatedAt: "2026-05-14T10:05:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	old := callTools
	calls := 0
	callTools = func(ctx context.Context, cfg config.ModelConfig, messages []llm.Message, tools []llm.ToolDefinition) (llm.ToolResponse, error) {
		calls++
		if calls == 1 {
			if cfg.ResolvedAPIKey() != "" {
				t.Fatalf("classifier should not require an API key, got %q", cfg.ResolvedAPIKey())
			}
			if !strings.Contains(messages[1].Content, "stage 8 uses an LLM classifier") {
				t.Fatalf("prompt missing context: %s", messages[1].Content)
			}
			return llm.ToolResponse{ToolCalls: []llm.ToolCall{toolCall("call_1", "create", `{"path":"project-memfab.md","description":"memfab project","tags":["project"]}`), toolCall("call_2", "append", `{"path":"project-memfab.md","content":"memfab stage 8 uses an LLM classifier tool loop.","tags":["classifier","stage8"]}`), toolCall("call_3", "commit", `{"summary":"wrote classifier fact"}`)}}, nil
		}
		return llm.ToolResponse{}, nil
	}
	t.Cleanup(func() { callTools = old })

	result, err := ClassifySession(context.Background(), db, classifierConfig(), row, time.Date(2026, 5, 14, 10, 6, 0, 0, time.UTC))
	if err != nil || !result.Committed || len(result.WrittenIDs) != 1 || len(result.CreatedPaths) != 1 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	updated, _ := db.GetSession(row.ID)
	if updated.ClassifiedEnd != row.FlushEnd {
		t.Fatalf("updated=%+v", updated)
	}
	hits, err := db.SearchMemory("classifier", []string{"project-*"}, "", "", 5, false)
	if err != nil || len(hits) != 1 || hits[0].Path != "project-memfab.md" {
		t.Fatalf("hits=%+v err=%v", hits, err)
	}
}

func TestClassifySessionNoFocusAdvancesBookmark(t *testing.T) {
	db := classifierTestDB(t)
	row := insertClassifySession(t, db, "sess_empty")
	old := callTools
	callTools = func(ctx context.Context, cfg config.ModelConfig, messages []llm.Message, tools []llm.ToolDefinition) (llm.ToolResponse, error) {
		t.Fatal("llm should not be called for empty focus")
		return llm.ToolResponse{}, nil
	}
	t.Cleanup(func() { callTools = old })
	result, err := ClassifySession(context.Background(), db, classifierConfig(), row, time.Date(2026, 5, 14, 10, 6, 0, 0, time.UTC))
	if err != nil || !result.Committed || result.SkippedReason == "" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	updated, _ := db.GetSession(row.ID)
	if updated.ClassifiedEnd != row.FlushEnd {
		t.Fatalf("updated=%+v", updated)
	}
}

func TestClassifySessionFailureDoesNotAdvanceBookmark(t *testing.T) {
	db := classifierTestDB(t)
	row := insertClassifySession(t, db, "sess_fail")
	insertClassifyEvent(t, row.ID)
	old := callTools
	callTools = func(ctx context.Context, cfg config.ModelConfig, messages []llm.Message, tools []llm.ToolDefinition) (llm.ToolResponse, error) {
		return llm.ToolResponse{}, errors.New("llm down")
	}
	t.Cleanup(func() { callTools = old })
	result, err := ClassifySession(context.Background(), db, classifierConfig(), row, time.Date(2026, 5, 14, 10, 6, 0, 0, time.UTC))
	if err != nil || !result.Failed || result.Committed {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	updated, _ := db.GetSession(row.ID)
	if updated.ClassifiedEnd != "" || updated.ClassifierRetryCount != 1 || updated.ClassifierLastError == "" {
		t.Fatalf("updated=%+v", updated)
	}
}

func TestClassifySessionRejectsEventWrite(t *testing.T) {
	db := classifierTestDB(t)
	row := insertClassifySession(t, db, "sess_event_write")
	insertClassifyEvent(t, row.ID)
	old := callTools
	calls := 0
	callTools = func(ctx context.Context, cfg config.ModelConfig, messages []llm.Message, tools []llm.ToolDefinition) (llm.ToolResponse, error) {
		calls++
		if calls == 1 {
			return llm.ToolResponse{ToolCalls: []llm.ToolCall{toolCall("call_bad", "append", `{"path":"event-2026-05-14.md","content":"bad","tags":["event"]}`)}}, nil
		}
		if !strings.Contains(messages[len(messages)-1].Content, "forbidden") {
			t.Fatalf("expected forbidden tool response, got %+v", messages[len(messages)-1])
		}
		return llm.ToolResponse{ToolCalls: []llm.ToolCall{toolCall("call_commit", "commit", `{"summary":""}`)}}, nil
	}
	t.Cleanup(func() { callTools = old })
	result, err := ClassifySession(context.Background(), db, classifierConfig(), row, time.Date(2026, 5, 14, 10, 6, 0, 0, time.UTC))
	if err != nil || !result.Committed || len(result.WrittenIDs) != 0 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	hits, err := db.SearchMemory("bad", []string{"event-*"}, "", "", 5, true)
	if err != nil || len(hits) != 0 {
		t.Fatalf("event write leaked hits=%+v err=%v", hits, err)
	}
}

func toolCall(id, name, args string) llm.ToolCall {
	var call llm.ToolCall
	call.ID = id
	call.Type = "function"
	call.Function.Name = name
	call.Function.Arguments = []byte(args)
	return call
}
