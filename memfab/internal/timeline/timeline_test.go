package timeline

import (
	"context"
	"strings"
	"testing"
	"time"

	"memfab/internal/capture"
	"memfab/internal/config"
	"memfab/internal/ingest"
	"memfab/internal/llm"
	"memfab/internal/store"
)

func timelineTestDB(t *testing.T) *store.DB {
	t.Helper()
	t.Setenv("MEMFAB_ROOT", t.TempDir())
	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func timelineObservationAt(ts, app, title, text string) capture.Observation {
	return capture.Observation{
		SchemaVersion: capture.SchemaVersion,
		Type:          capture.ObservationType,
		CapturedAt:    ts,
		CaptureID:     "timeline:" + ts,
		Source:        capture.Source{Kind: "screen_ocr", Device: "test"},
		Context:       capture.Context{AppName: app, Title: title},
		Event:         capture.Event{Type: "ocr_capture", Reason: "test"},
		OCR:           capture.OCR{Text: text, Language: "en", BlockCount: 1},
		Tags:          []string{"ocr"},
	}
}

func insertTimelineObservation(t *testing.T, db *store.DB, o capture.Observation) {
	t.Helper()
	if err := o.Normalize(nil); err != nil {
		t.Fatal(err)
	}
	stem, err := capture.SafeFilename(o.CapturedAt)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.InsertObservation(stem, o); err != nil {
		t.Fatal(err)
	}
}

func TestFloorToWindow(t *testing.T) {
	when := time.Date(2026, 5, 12, 19, 31, 42, 0, time.FixedZone("CST", 8*3600))
	got := FloorToWindow(when, 1)
	if got.Minute() != 31 || got.Second() != 0 || got.Location() != when.Location() {
		t.Fatalf("got %s", got)
	}
	got = FloorToWindow(when, 5)
	if got.Minute() != 30 || got.Second() != 0 {
		t.Fatalf("got %s", got)
	}
}

func TestProduceBlockForWindowUsesLLMEntries(t *testing.T) {
	db := timelineTestDB(t)
	insertTimelineObservation(t, db, timelineObservationAt("2026-05-12T19:31:10+08:00", "ScreenOCR", "Sipeed资料站", "NanoKVM 电源控制 MaixCAM"))

	old := callJSON
	callJSON = func(ctx context.Context, cfg config.ModelConfig, messages []llm.Message) (string, error) {
		if cfg.ResolvedAPIKey() != "" {
			t.Fatalf("timeline should not require an API key, got %q", cfg.ResolvedAPIKey())
		}
		if !strings.Contains(messages[1].Content, "NanoKVM") {
			t.Fatalf("prompt did not contain OCR text: %s", messages[1].Content)
		}
		return `{"entries":["[ScreenOCR] Sipeed资料站: read NanoKVM 电源控制 MaixCAM."]}`, nil
	}
	t.Cleanup(func() { callJSON = old })

	start := time.Date(2026, 5, 12, 19, 31, 0, 0, time.FixedZone("CST", 8*3600))
	inserted, err := ProduceBlockForWindow(context.Background(), db, config.Config{
		Models:   config.ModelsConfig{Timeline: config.ModelConfig{Model: "gpt-test"}},
		Timeline: config.TimelineConfig{WindowMinutes: 1, ColdLookbackMinutes: 30},
	}, start, start.Add(time.Minute))
	if err != nil || !inserted {
		t.Fatalf("inserted=%v err=%v", inserted, err)
	}
	blocks, err := db.ListTimelineBlocks(5)
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 1 || len(blocks[0].Entries) != 1 || !strings.Contains(blocks[0].Entries[0], "NanoKVM") {
		t.Fatalf("blocks=%+v", blocks)
	}
}

func TestProduceBlockForWindowIncludesCompactOCRBlocks(t *testing.T) {
	db := timelineTestDB(t)
	obs := timelineObservationAt("2026-05-12T19:31:10+08:00", "ScreenOCR", "Sipeed资料站", "NanoKVM 电源控制")
	obs.OCR.Blocks = []capture.OCRBlock{
		{ID: 1, Text: "NanoKVM", BBox: []int{10, 20, 110, 40}, Confidence: 0.98},
		{ID: 2, Text: "电源控制", BBox: []int{120, 20, 220, 40}, Confidence: 0.97},
		{ID: 3, Text: "ignored without bbox"},
	}
	obs.OCR.BlockCount = len(obs.OCR.Blocks)
	if _, err := ingest.StoreObservation(db, obs); err != nil {
		t.Fatal(err)
	}

	old := callJSON
	callJSON = func(ctx context.Context, cfg config.ModelConfig, messages []llm.Message) (string, error) {
		prompt := messages[1].Content
		if !strings.Contains(prompt, `layout_blocks=[{"text":"NanoKVM","bbox":[10,20,110,40]}`) {
			t.Fatalf("prompt did not contain compact OCR block JSON: %s", prompt)
		}
		if strings.Contains(prompt, "ignored without bbox") {
			t.Fatalf("prompt included block without bbox: %s", prompt)
		}
		return `{"entries":["[ScreenOCR] Sipeed资料站: read NanoKVM 电源控制."]}`, nil
	}
	t.Cleanup(func() { callJSON = old })

	start := time.Date(2026, 5, 12, 19, 31, 0, 0, time.FixedZone("CST", 8*3600))
	inserted, err := ProduceBlockForWindow(context.Background(), db, config.Config{
		Models:   config.ModelsConfig{Timeline: config.ModelConfig{Model: "gpt-test", APIKey: "test-key"}},
		Timeline: config.TimelineConfig{WindowMinutes: 1, ColdLookbackMinutes: 30},
	}, start, start.Add(time.Minute))
	if err != nil || !inserted {
		t.Fatalf("inserted=%v err=%v", inserted, err)
	}
}

func TestProduceBlockForWindowFallsBackWhenRawObservationMissing(t *testing.T) {
	db := timelineTestDB(t)
	insertTimelineObservation(t, db, timelineObservationAt("2026-05-12T19:31:10+08:00", "ScreenOCR", "Missing JSON", "fallback OCR text"))

	old := callJSON
	callJSON = func(ctx context.Context, cfg config.ModelConfig, messages []llm.Message) (string, error) {
		prompt := messages[1].Content
		if strings.Contains(prompt, "layout_blocks=") {
			t.Fatalf("prompt should not contain layout blocks for missing raw JSON: %s", prompt)
		}
		if !strings.Contains(prompt, "fallback OCR text") {
			t.Fatalf("prompt did not fall back to OCR text: %s", prompt)
		}
		return `{"entries":["[ScreenOCR] Missing JSON: fallback OCR text."]}`, nil
	}
	t.Cleanup(func() { callJSON = old })

	start := time.Date(2026, 5, 12, 19, 31, 0, 0, time.FixedZone("CST", 8*3600))
	inserted, err := ProduceBlockForWindow(context.Background(), db, config.Config{
		Models:   config.ModelsConfig{Timeline: config.ModelConfig{Model: "gpt-test", APIKey: "test-key"}},
		Timeline: config.TimelineConfig{WindowMinutes: 1, ColdLookbackMinutes: 30},
	}, start, start.Add(time.Minute))
	if err != nil || !inserted {
		t.Fatalf("inserted=%v err=%v", inserted, err)
	}
}

func TestProduceBlockForWindowFallsBackAndDedupes(t *testing.T) {
	db := timelineTestDB(t)
	first := timelineObservationAt("2026-05-12T19:31:10+08:00", "ScreenOCR", "Duplicate", "same OCR text")
	second := timelineObservationAt("2026-05-12T19:31:20+08:00", "ScreenOCR", "Duplicate", "same OCR text")
	insertTimelineObservation(t, db, first)
	insertTimelineObservation(t, db, second)

	old := callJSON
	called := false
	callJSON = func(ctx context.Context, cfg config.ModelConfig, messages []llm.Message) (string, error) {
		called = true
		return `not json`, nil
	}
	t.Cleanup(func() { callJSON = old })

	start := time.Date(2026, 5, 12, 19, 31, 0, 0, time.FixedZone("CST", 8*3600))
	inserted, err := ProduceBlockForWindow(context.Background(), db, config.Config{
		Models:   config.ModelsConfig{Timeline: config.ModelConfig{Model: "gpt-test", APIKey: "test-key"}},
		Timeline: config.TimelineConfig{WindowMinutes: 1, ColdLookbackMinutes: 30},
	}, start, start.Add(time.Minute))
	if err != nil || !inserted {
		t.Fatalf("inserted=%v err=%v", inserted, err)
	}
	if !called {
		t.Fatal("expected LLM call before fallback")
	}
	blocks, err := db.ListTimelineBlocks(5)
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 1 || blocks[0].CaptureCount != 1 || !strings.Contains(blocks[0].Entries[0], "same OCR text") {
		t.Fatalf("blocks=%+v", blocks)
	}

	inserted, err = ProduceBlockForWindow(context.Background(), db, config.Config{}, start, start.Add(time.Minute))
	if err != nil || inserted {
		t.Fatalf("duplicate inserted=%v err=%v", inserted, err)
	}
}

func TestTickSkipsCurrentWindow(t *testing.T) {
	db := timelineTestDB(t)
	insertTimelineObservation(t, db, timelineObservationAt("2026-05-12T19:31:10+08:00", "ScreenOCR", "Closed", "closed window"))
	insertTimelineObservation(t, db, timelineObservationAt("2026-05-12T19:32:10+08:00", "ScreenOCR", "Current", "current window"))
	old := callJSON
	calls := 0
	callJSON = func(ctx context.Context, cfg config.ModelConfig, messages []llm.Message) (string, error) {
		calls++
		if cfg.ResolvedAPIKey() != "" {
			t.Fatalf("timeline should not require an API key, got %q", cfg.ResolvedAPIKey())
		}
		return `not json`, nil
	}
	t.Cleanup(func() { callJSON = old })

	produced, err := tickUntil(context.Background(), db, config.Config{
		Models:   config.ModelsConfig{Timeline: config.ModelConfig{Model: "gpt-test"}},
		Timeline: config.TimelineConfig{WindowMinutes: 1, ColdLookbackMinutes: 5},
	}, time.Date(2026, 5, 12, 19, 32, 30, 0, time.FixedZone("CST", 8*3600)))
	if err != nil || produced != 1 {
		t.Fatalf("produced=%d err=%v", produced, err)
	}
	if calls != 1 {
		t.Fatalf("llm calls=%d, want 1", calls)
	}
	blocks, err := db.ListTimelineBlocks(5)
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 1 || !strings.Contains(blocks[0].Entries[0], "closed window") {
		t.Fatalf("blocks=%+v", blocks)
	}
}

func TestTickWithOptionsIncludesCurrentWindow(t *testing.T) {
	db := timelineTestDB(t)
	insertTimelineObservation(t, db, timelineObservationAt("2026-05-12T19:31:10+08:00", "ScreenOCR", "Closed", "closed window"))
	insertTimelineObservation(t, db, timelineObservationAt("2026-05-12T19:32:10+08:00", "ScreenOCR", "Current", "current window"))
	old := callJSON
	calls := 0
	callJSON = func(ctx context.Context, cfg config.ModelConfig, messages []llm.Message) (string, error) {
		calls++
		if cfg.ResolvedAPIKey() != "" {
			t.Fatalf("timeline should not require an API key, got %q", cfg.ResolvedAPIKey())
		}
		return `not json`, nil
	}
	t.Cleanup(func() { callJSON = old })

	produced, err := tickWithOptionsUntil(context.Background(), db, config.Config{
		Models:   config.ModelsConfig{Timeline: config.ModelConfig{Model: "gpt-test"}},
		Timeline: config.TimelineConfig{WindowMinutes: 1, ColdLookbackMinutes: 5},
	}, TickOptions{IncludeOpen: true}, time.Date(2026, 5, 12, 19, 32, 30, 0, time.FixedZone("CST", 8*3600)))
	if err != nil || produced != 2 {
		t.Fatalf("produced=%d err=%v", produced, err)
	}
	if calls != 2 {
		t.Fatalf("llm calls=%d, want 2", calls)
	}
	blocks, err := db.ListTimelineBlocks(5)
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 2 {
		t.Fatalf("blocks=%+v", blocks)
	}
	if blocks[0].StartTime != "2026-05-12T19:31:00+08:00" || blocks[0].EndTime != "2026-05-12T19:32:00+08:00" || !strings.Contains(blocks[0].Entries[0], "closed window") {
		t.Fatalf("closed block=%+v", blocks[0])
	}
	if blocks[1].StartTime != "2026-05-12T19:32:00+08:00" || blocks[1].EndTime != "2026-05-12T19:33:00+08:00" || !strings.Contains(blocks[1].Entries[0], "current window") {
		t.Fatalf("open block=%+v", blocks[1])
	}

	produced, err = tickWithOptionsUntil(context.Background(), db, config.Config{
		Models:   config.ModelsConfig{Timeline: config.ModelConfig{Model: "gpt-test"}},
		Timeline: config.TimelineConfig{WindowMinutes: 1, ColdLookbackMinutes: 5},
	}, TickOptions{IncludeOpen: true}, time.Date(2026, 5, 12, 19, 32, 45, 0, time.FixedZone("CST", 8*3600)))
	if err != nil || produced != 0 {
		t.Fatalf("second produced=%d err=%v", produced, err)
	}
}
