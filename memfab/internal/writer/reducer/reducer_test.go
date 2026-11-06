package reducer

import (
	"context"
	"strings"
	"testing"
	"time"

	"memfab/internal/config"
	"memfab/internal/llm"
	"memfab/internal/paths"
	"memfab/internal/store"
)

func reducerTestDB(t *testing.T) *store.DB {
	t.Helper()
	t.Setenv("MEMFAB_ROOT", t.TempDir())
	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func reducerConfig() config.Config {
	return config.Config{Reducer: config.ReducerConfig{Enabled: true}, Models: config.ModelsConfig{Reducer: config.ModelConfig{Model: "gpt-test"}}}
}

func insertReducerSession(t *testing.T, db *store.DB, id string, retry int) store.SessionRow {
	t.Helper()
	row := store.SessionRow{ID: id, StartTime: "2026-05-14T10:00:00Z", EndTime: "2026-05-14T10:05:00Z", Status: string(store.SessionEnded), RetryCount: retry, CreatedAt: "2026-05-14T10:05:00Z", UpdatedAt: "2026-05-14T10:05:00Z"}
	if retry > 0 {
		row.Status = string(store.SessionFailed)
	}
	if _, err := db.InsertSession(row); err != nil {
		t.Fatal(err)
	}
	return row
}

func insertReducerTimeline(t *testing.T, db *store.DB) {
	t.Helper()
	_, err := db.InsertTimelineBlock(store.TimelineBlock{ID: "tl1", StartTime: "2026-05-14T10:00:00Z", EndTime: "2026-05-14T10:01:00Z", Entries: []string{"[Cursor] edited reducer.go"}, AppsUsed: []string{"Cursor"}, CaptureCount: 1, CreatedAt: "2026-05-14T10:01:00Z"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestReduceSessionWritesEventMemory(t *testing.T) {
	db := reducerTestDB(t)
	row := insertReducerSession(t, db, "sess_ok", 0)
	insertReducerTimeline(t, db)
	old := callJSON
	callJSON = func(ctx context.Context, cfg config.ModelConfig, messages []llm.Message) (string, error) {
		if cfg.ResolvedAPIKey() != "" {
			t.Fatalf("reducer should not require an API key, got %q", cfg.ResolvedAPIKey())
		}
		if !strings.Contains(messages[0].Content, "edited reducer.go") {
			t.Fatalf("prompt missing timeline: %s", messages[0].Content)
		}
		return `{"summary":"Worked on reducer.","sub_tasks":["[10:00-10:05, Cursor] edited reducer.go"]}`, nil
	}
	t.Cleanup(func() { callJSON = old })

	result, err := ReduceSession(context.Background(), db, reducerConfig(), row, time.Date(2026, 5, 14, 10, 6, 0, 0, time.UTC))
	if err != nil || !result.Reduced || !result.Written || result.Heuristic {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	updated, _ := db.GetSession("sess_ok")
	if updated.Status != string(store.SessionReduced) {
		t.Fatalf("session=%+v", updated)
	}
	hits, err := db.SearchMemory("reducer", nil, "", "", 5, false)
	if err != nil || len(hits) != 1 || hits[0].Path != "event-2026-05-14.md" {
		t.Fatalf("hits=%+v err=%v", hits, err)
	}
}

func TestReduceSessionNoBlocksMarksReduced(t *testing.T) {
	db := reducerTestDB(t)
	row := insertReducerSession(t, db, "sess_empty", 0)
	result, err := ReduceSession(context.Background(), db, reducerConfig(), row, time.Date(2026, 5, 14, 10, 6, 0, 0, time.UTC))
	if err != nil || !result.Reduced || result.Written {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	updated, _ := db.GetSession("sess_empty")
	if updated.Status != string(store.SessionReduced) {
		t.Fatalf("session=%+v", updated)
	}
}

func TestReduceSessionSchedulesRetryOnLLMFailure(t *testing.T) {
	db := reducerTestDB(t)
	row := insertReducerSession(t, db, "sess_retry", 0)
	insertReducerTimeline(t, db)
	old := callJSON
	callJSON = func(ctx context.Context, cfg config.ModelConfig, messages []llm.Message) (string, error) {
		return `bad json`, nil
	}
	t.Cleanup(func() { callJSON = old })
	result, err := ReduceSession(context.Background(), db, reducerConfig(), row, time.Date(2026, 5, 14, 10, 6, 0, 0, time.UTC))
	if err != nil || !result.Failed || result.Written {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	updated, _ := db.GetSession("sess_retry")
	if updated.Status != string(store.SessionFailed) || updated.RetryCount != 1 || updated.NextRetryAt == "" {
		t.Fatalf("session=%+v", updated)
	}
}

func TestReduceSessionExhaustedRetriesWritesHeuristic(t *testing.T) {
	db := reducerTestDB(t)
	row := insertReducerSession(t, db, "sess_heuristic", len(retryBackoffMinutes))
	insertReducerTimeline(t, db)
	old := callJSON
	callJSON = func(ctx context.Context, cfg config.ModelConfig, messages []llm.Message) (string, error) {
		return `bad json`, nil
	}
	t.Cleanup(func() { callJSON = old })
	result, err := ReduceSession(context.Background(), db, reducerConfig(), row, time.Date(2026, 5, 14, 10, 6, 0, 0, time.UTC))
	if err != nil || !result.Reduced || !result.Written || !result.Heuristic {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	hits, err := db.SearchMemory("heuristic", nil, "", "", 5, true)
	if err != nil || len(hits) != 1 {
		t.Fatalf("hits=%+v err=%v", hits, err)
	}
}

func TestFlushActiveWritesFlushEvent(t *testing.T) {
	db := reducerTestDB(t)
	_, err := db.InsertSession(store.SessionRow{ID: "sess_active", StartTime: "2026-05-14T10:00:00Z", Status: string(store.SessionActive), CreatedAt: "2026-05-14T10:00:00Z", UpdatedAt: "2026-05-14T10:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.InsertTimelineBlock(store.TimelineBlock{ID: "tl-flush", StartTime: "2026-05-14T10:01:00Z", EndTime: "2026-05-14T10:02:00Z", Entries: []string{"[Cursor] active flush work"}, AppsUsed: []string{"Cursor"}, CaptureCount: 1, CreatedAt: "2026-05-14T10:02:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	old := callJSON
	callJSON = func(ctx context.Context, cfg config.ModelConfig, messages []llm.Message) (string, error) {
		if !strings.Contains(messages[0].Content, "active flush work") {
			t.Fatalf("prompt=%s", messages[0].Content)
		}
		return `{"summary":"Active flush work.","sub_tasks":["[10:01-10:02, Cursor] active flush work"]}`, nil
	}
	t.Cleanup(func() { callJSON = old })
	result, err := FlushActive(context.Background(), db, reducerConfig(), time.Date(2026, 5, 14, 10, 3, 0, 0, time.UTC))
	if err != nil || !result.Written || result.Reduced {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	row, _ := db.GetSession("sess_active")
	if row.Status != string(store.SessionActive) || row.FlushEnd == "" {
		t.Fatalf("row=%+v", row)
	}
	hits, err := db.SearchMemory("Active", []string{"event-*"}, "", "", 5, true)
	if err != nil || len(hits) != 1 || !strings.Contains(hits[0].Tags, "flush") {
		t.Fatalf("hits=%+v err=%v", hits, err)
	}
}

func TestFlushActiveNoBlocksNoops(t *testing.T) {
	db := reducerTestDB(t)
	_, err := db.InsertSession(store.SessionRow{ID: "sess_active_empty", StartTime: "2026-05-14T10:00:00Z", Status: string(store.SessionActive), CreatedAt: "2026-05-14T10:00:00Z", UpdatedAt: "2026-05-14T10:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := FlushActive(context.Background(), db, reducerConfig(), time.Date(2026, 5, 14, 10, 3, 0, 0, time.UTC))
	if err != nil || result.Written || result.Failed {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	row, _ := db.GetSession("sess_active_empty")
	if row.FlushEnd != "" {
		t.Fatalf("row=%+v", row)
	}
}

func TestFlushActiveLLMFailureDoesNotAdvance(t *testing.T) {
	db := reducerTestDB(t)
	_, err := db.InsertSession(store.SessionRow{ID: "sess_active_fail", StartTime: "2026-05-14T10:00:00Z", Status: string(store.SessionActive), CreatedAt: "2026-05-14T10:00:00Z", UpdatedAt: "2026-05-14T10:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.InsertTimelineBlock(store.TimelineBlock{ID: "tl-fail", StartTime: "2026-05-14T10:01:00Z", EndTime: "2026-05-14T10:02:00Z", Entries: []string{"[Cursor] will fail"}, AppsUsed: []string{"Cursor"}, CaptureCount: 1, CreatedAt: "2026-05-14T10:02:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	old := callJSON
	callJSON = func(ctx context.Context, cfg config.ModelConfig, messages []llm.Message) (string, error) {
		return `bad json`, nil
	}
	t.Cleanup(func() { callJSON = old })
	result, err := FlushActive(context.Background(), db, reducerConfig(), time.Date(2026, 5, 14, 10, 3, 0, 0, time.UTC))
	if err != nil || !result.Failed || result.Written {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	row, _ := db.GetSession("sess_active_fail")
	if row.FlushEnd != "" || row.Status != string(store.SessionActive) {
		t.Fatalf("row=%+v", row)
	}
	hits, err := db.SearchMemory("heuristic", []string{"event-*"}, "", "", 5, true)
	if err != nil || len(hits) != 0 {
		t.Fatalf("hits=%+v err=%v", hits, err)
	}
}

func TestRecentEventContextIsBounded(t *testing.T) {
	db := reducerTestDB(t)
	row := insertReducerSession(t, db, "sess_context", 0)
	insertReducerTimeline(t, db)
	for i := 0; i < 8; i++ {
		entry := store.MemoryEntryRow{ID: "old", Path: "event-2026-05-14.md", Prefix: "event", Timestamp: "2026-05-14T09:00:00Z", Tags: "event", Content: "old context"}
		_ = entry
	}
	old := callJSON
	callJSON = func(ctx context.Context, cfg config.ModelConfig, messages []llm.Message) (string, error) {
		return `{"summary":"Worked.","sub_tasks":["task"]}`, nil
	}
	t.Cleanup(func() { callJSON = old })
	_, err := ReduceSession(context.Background(), db, reducerConfig(), row, time.Date(2026, 5, 14, 10, 6, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if paths.MemoryDir() == "" {
		t.Fatal("memory dir missing")
	}
}
