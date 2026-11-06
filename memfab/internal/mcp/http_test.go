package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"memfab/internal/capture"
	"memfab/internal/ingest"
	"memfab/internal/memory"
	"memfab/internal/paths"
	"memfab/internal/store"
)

func mcpTestDB(t *testing.T) *store.DB {
	t.Helper()
	t.Setenv("MEMFAB_ROOT", t.TempDir())
	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func loadOCRExample(t *testing.T) capture.Observation {
	t.Helper()
	data, err := os.ReadFile("../../tests/ocr_example.json")
	if err != nil {
		t.Fatal(err)
	}
	var o capture.Observation
	if err := json.Unmarshal(data, &o); err != nil {
		t.Fatal(err)
	}
	return o
}

func TestReadRecentObservationDefaultAndIncludeFlags(t *testing.T) {
	db := mcpTestDB(t)
	o := loadOCRExample(t)
	if _, err := ingest.StoreObservation(db, o); err != nil {
		t.Fatal(err)
	}

	res, err := callTool(db, "read_recent_capture", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	obs, ok := res.(capture.Observation)
	if !ok {
		t.Fatalf("result type %T", res)
	}
	if obs.OCR.Text == "" {
		t.Fatal("expected OCR text")
	}
	if obs.OCR.RawText != "" {
		t.Fatal("raw text should be omitted by default")
	}
	if len(obs.OCR.Blocks) != 0 {
		t.Fatal("blocks should be omitted by default")
	}
	if obs.OCR.BlockCount == 0 {
		t.Fatal("block_count should be preserved")
	}

	res, err = callTool(db, "read_recent_capture", json.RawMessage(`{"include_raw_text":true,"include_blocks":true}`))
	if err != nil {
		t.Fatal(err)
	}
	obs = res.(capture.Observation)
	if obs.OCR.RawText == "" {
		t.Fatal("expected raw text")
	}
	if len(obs.OCR.Blocks) == 0 {
		t.Fatal("expected blocks")
	}
}

func TestReadRecentObservationFiltersAndNoMatch(t *testing.T) {
	db := mcpTestDB(t)
	o := loadOCRExample(t)
	if _, err := ingest.StoreObservation(db, o); err != nil {
		t.Fatal(err)
	}

	res, err := callTool(db, "read_recent_capture", json.RawMessage(`{"app_name":"screen","title":"snapshot"}`))
	if err != nil {
		t.Fatal(err)
	}
	if res == nil {
		t.Fatal("expected filtered observation")
	}

	res, err = callTool(db, "read_recent_capture", json.RawMessage(`{"app_name":"missing"}`))
	if err != nil {
		t.Fatal(err)
	}
	if res != nil {
		t.Fatalf("expected nil no-match, got %#v", res)
	}
}

func TestCurrentObservationContextEmpty(t *testing.T) {
	db := mcpTestDB(t)
	res, err := callTool(db, "current_context", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	ctx := res.(map[string]any)
	if ctx["headline"] != "No recent OCR observations" {
		t.Fatalf("headline=%v", ctx["headline"])
	}
	if ctx["top_text"] != "" || ctx["latest_captured_at"] != "" || ctx["observation_count"] != 0 {
		t.Fatalf("unexpected empty context: %#v", ctx)
	}
	if len(ctx["recent_observations"].([]map[string]any)) != 0 || len(ctx["recent_timeline_blocks"].([]map[string]any)) != 0 || len(ctx["apps"].([]string)) != 0 || len(ctx["keywords"].([]string)) != 0 {
		t.Fatalf("expected empty slices: %#v", ctx)
	}
	if ctx["timeline_block_count"] != 0 {
		t.Fatalf("timeline_block_count=%v", ctx["timeline_block_count"])
	}
}

func TestCurrentObservationContextBuildsSummary(t *testing.T) {
	db := mcpTestDB(t)
	o := loadOCRExample(t)
	if _, err := ingest.StoreObservation(db, o); err != nil {
		t.Fatal(err)
	}
	res, err := callTool(db, "current_context", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	ctx := res.(map[string]any)
	if ctx["headline"] == "" {
		t.Fatal("expected headline")
	}
	if got := len(ctx["recent_observations"].([]map[string]any)); got != 1 {
		t.Fatalf("recent count=%d", got)
	}
	if !strings.Contains(ctx["top_text"].(string), "Sipeed") {
		t.Fatalf("top_text missing Sipeed: %q", ctx["top_text"])
	}
	if ctx["apps"].([]string)[0] != "ScreenOCR" {
		t.Fatalf("apps=%v", ctx["apps"])
	}
	if ctx["titles"].([]string)[0] != "OCR screen snapshot" {
		t.Fatalf("titles=%v", ctx["titles"])
	}
	if !stringSliceContains(ctx["keywords"].([]string), "Sipeed") {
		t.Fatalf("keywords=%v", ctx["keywords"])
	}
}

func TestCurrentObservationContextIncludesTimelineBlocks(t *testing.T) {
	db := mcpTestDB(t)
	if _, err := db.InsertTimelineBlock(store.TimelineBlock{ID: "tl-old", StartTime: "2026-05-14T10:00:00Z", EndTime: "2026-05-14T10:01:00Z", Timezone: "+08:00", Entries: []string{"[ScreenOCR] old block"}, AppsUsed: []string{"ScreenOCR"}, CaptureCount: 1, CreatedAt: "2026-05-14T10:01:00Z"}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.InsertTimelineBlock(store.TimelineBlock{ID: "tl-new", StartTime: "2026-05-14T10:01:00Z", EndTime: "2026-05-14T10:02:00Z", Timezone: "+08:00", Entries: []string{"[ScreenOCR] new block"}, AppsUsed: []string{"ScreenOCR", "Browser"}, CaptureCount: 2, CreatedAt: "2026-05-14T10:02:00Z"}); err != nil {
		t.Fatal(err)
	}
	res, err := callTool(db, "current_context", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	ctx := res.(map[string]any)
	blocks := ctx["recent_timeline_blocks"].([]map[string]any)
	if ctx["timeline_block_count"] != 2 || len(blocks) != 2 {
		t.Fatalf("ctx=%#v", ctx)
	}
	if blocks[0]["id"] != "tl-old" || blocks[1]["id"] != "tl-new" {
		t.Fatalf("blocks not chronological: %#v", blocks)
	}
	if blocks[1]["capture_count"] != 2 || blocks[1]["timezone"] != "+08:00" {
		t.Fatalf("block metadata=%#v", blocks[1])
	}
	entries := blocks[1]["entries"].([]string)
	if len(entries) != 1 || entries[0] != "[ScreenOCR] new block" {
		t.Fatalf("entries=%v", entries)
	}
	apps := blocks[1]["apps_used"].([]string)
	if len(apps) != 2 || apps[1] != "Browser" {
		t.Fatalf("apps=%v", apps)
	}
}

func TestCurrentObservationContextDoesNotExposeLargeDebugFields(t *testing.T) {
	db := mcpTestDB(t)
	o := loadOCRExample(t)
	if _, err := ingest.StoreObservation(db, o); err != nil {
		t.Fatal(err)
	}
	res, err := callTool(db, "current_context", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(res)
	out := string(b)
	for _, forbidden := range []string{"raw_text", "OCR Recognition Results"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("context exposed %q", forbidden)
		}
	}
}

func TestCurrentObservationContextDedupeAppsAndTitles(t *testing.T) {
	recent := []store.ObservationHit{
		{CapturedAt: "2026-05-12T21:00:00+08:00", AppName: "ScreenOCR", Title: "OCR screen snapshot", OCRText: "Sipeed NanoKVM"},
		{CapturedAt: "2026-05-12T20:00:00+08:00", AppName: "ScreenOCR", Title: "OCR screen snapshot", OCRText: "MaixCAM"},
		{CapturedAt: "2026-05-12T19:00:00+08:00", AppName: "OtherOCR", Title: "Other snapshot", OCRText: "电源 USB"},
	}
	ctx := buildCurrentObservationContext(recent, nil)
	apps := ctx["apps"].([]string)
	titles := ctx["titles"].([]string)
	if len(apps) != 2 || apps[0] != "ScreenOCR" || apps[1] != "OtherOCR" {
		t.Fatalf("apps=%v", apps)
	}
	if len(titles) != 2 || titles[0] != "OCR screen snapshot" || titles[1] != "Other snapshot" {
		t.Fatalf("titles=%v", titles)
	}
}

func TestCurrentObservationContextLimitsOutput(t *testing.T) {
	longLine := strings.Repeat("Sipeed NanoKVM MaixCAM USB RJ45 OLED ATX 电源 串口 ", 100)
	recent := []store.ObservationHit{{CapturedAt: "2026-05-12T21:00:00+08:00", AppName: "ScreenOCR", Title: "OCR screen snapshot", OCRText: longLine}}
	ctx := buildCurrentObservationContext(recent, nil)
	if len([]rune(ctx["top_text"].(string))) > 800 {
		t.Fatalf("top_text too long")
	}
	if len(ctx["keywords"].([]string)) > 12 {
		t.Fatalf("too many keywords")
	}
	if len(ctx["recent_observations"].([]map[string]any)) > 5 {
		t.Fatalf("too many observations")
	}
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestToolsListUsesCompatibleCaptureNames(t *testing.T) {
	tools := toolsList()
	names := map[string]bool{}
	for _, tool := range tools {
		names[tool.Name] = true
	}
	for _, want := range []string{"current_context", "search_captures", "read_recent_capture"} {
		if !names[want] {
			t.Fatalf("missing tool %s in %#v", want, names)
		}
	}
}

func TestMemoryTools(t *testing.T) {
	db := mcpTestDB(t)
	memoryDir := paths.MemoryDir()
	if err := os.MkdirAll(memoryDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := `---
description: Test memory
tags: [test, phase5]
status: active
created: 2026-05-14T00:00:00Z
updated: 2026-05-14T00:05:00Z
---

## [2026-05-14T00:01:00Z] {id: test-entry} #test
Memory query smoke entry.

## [2026-05-14T00:02:00Z] {id: other-entry} #other
Other entry.
`
	if err := os.WriteFile(filepath.Join(memoryDir, "project-test.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	files, indexedEntries, skipped, err := db.RebuildMemoryIndex(memoryDir)
	if err != nil {
		t.Fatal(err)
	}
	if files != 1 || indexedEntries != 2 || skipped != 0 {
		t.Fatalf("rebuild files=%d entries=%d skipped=%d", files, indexedEntries, skipped)
	}

	res, err := callTool(db, "list_memories", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	list := res.(map[string]any)
	if list["count"] != 1 {
		t.Fatalf("list=%#v", list)
	}

	res, err = callTool(db, "read_memory", json.RawMessage(`{"path":"project-test","tags":["test"],"tail_n":1}`))
	if err != nil {
		t.Fatal(err)
	}
	read := res.(map[string]any)
	entries := read["entries"].([]memory.ParsedEntry)
	if read["path"] != "project-test.md" || len(entries) != 1 || entries[0].ID != "test-entry" {
		t.Fatalf("read=%#v", read)
	}

	res, err = callTool(db, "search", json.RawMessage(`{"query":"smoke","top_k":3}`))
	if err != nil {
		t.Fatal(err)
	}
	search := res.(map[string]any)
	if len(search["results"].([]store.EntryHit)) != 1 {
		t.Fatalf("search=%#v", search)
	}

	res, err = callTool(db, "recent_activity", json.RawMessage(`{"prefix_filter":["project-"]}`))
	if err != nil {
		t.Fatal(err)
	}
	recent := res.(map[string]any)
	if recent["count"] != 2 {
		t.Fatalf("recent=%#v", recent)
	}

	res, err = callTool(db, "read_memory", json.RawMessage(`{"path":"missing"}`))
	if err != nil {
		t.Fatal(err)
	}
	missing := res.(map[string]any)
	if !strings.Contains(missing["error"].(string), "file not found") {
		t.Fatalf("missing=%#v", missing)
	}

	res, err = callTool(db, "get_schema", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	schema := res.(map[string]any)["schema"].(string)
	if !strings.Contains(schema, "Memory Organization Spec") {
		t.Fatalf("schema=%q", schema[:min(len(schema), 80)])
	}
}

func TestToolsListIncludesMemoryTools(t *testing.T) {
	tools := toolsList()
	names := map[string]bool{}
	for _, tool := range tools {
		names[tool.Name] = true
	}
	for _, want := range []string{"list_memories", "read_memory", "search", "recent_activity", "get_schema"} {
		if !names[want] {
			t.Fatalf("missing tool %s", want)
		}
	}
	if names["create_screen_timelapse"] {
		t.Fatal("create_screen_timelapse should not be registered")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
