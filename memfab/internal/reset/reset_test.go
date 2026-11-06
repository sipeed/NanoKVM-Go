package reset_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"memfab/internal/capture"
	"memfab/internal/memory"
	"memfab/internal/paths"
	ocReset "memfab/internal/reset"
	"memfab/internal/store"
)

func TestRunDeletesBusinessDataAndPreservesRuntimeFiles(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MEMFAB_ROOT", root)
	if err := paths.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := time.Date(2026, 6, 18, 10, 0, 0, 0, time.UTC)
	if _, err := memory.CreateDurableFile(paths.MemoryDir(), "project-reset.md", "Reset test", []string{"project"}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := memory.AppendDurableEntry(paths.MemoryDir(), "project-reset.md", memory.ParsedEntry{ID: "entry-1", Timestamp: now.Format(time.RFC3339), Tags: []string{"reset"}, Body: "reset removes memory files"}, now); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := db.RebuildMemoryIndex(paths.MemoryDir()); err != nil {
		t.Fatal(err)
	}

	touchTestFile(t, filepath.Join(paths.ScreenshotsDir(), "2026-06-18", "default.jpg"))
	outsideShot := touchTestFile(t, filepath.Join(t.TempDir(), "outside.jpg"))
	touchTestFile(t, filepath.Join(paths.ObservationBufferDir(), "first.json"))
	touchTestFile(t, filepath.Join(paths.ObservationBufferDir(), "second.json"))
	insertObservation(t, db, "2026-06-18T10:00:00Z", "first reset observation")
	insertObservation(t, db, "2026-06-18T10:01:00Z", "second reset observation")
	insertObservation(t, db, "2026-06-18T10:02:00Z", "third reset observation")
	if _, err := db.InsertTimelineBlock(store.TimelineBlock{ID: "tl-reset", StartTime: "2026-06-18T10:00:00Z", EndTime: "2026-06-18T10:01:00Z", Entries: []string{"[ScreenOCR] reset"}, AppsUsed: []string{"ScreenOCR"}, CaptureCount: 1, CreatedAt: "2026-06-18T10:01:00Z"}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.InsertSession(store.SessionRow{ID: "sess-reset", StartTime: "2026-06-18T10:00:00Z", EndTime: "2026-06-18T10:01:00Z", Status: string(store.SessionEnded), CreatedAt: "2026-06-18T10:01:00Z", UpdatedAt: "2026-06-18T10:01:00Z"}); err != nil {
		t.Fatal(err)
	}
	touchTestFile(t, paths.WriterState())
	touchTestFile(t, paths.ConfigFile())
	touchTestFile(t, filepath.Join(paths.LogsDir(), "memfab.log"))
	touchTestFile(t, paths.PIDFile())
	touchTestFile(t, paths.PausedFlag())

	stats, err := ocReset.Run(db)
	if err != nil {
		t.Fatal(err)
	}
	if stats.DeletedMemoryFiles != 1 || stats.DeletedObservations != 3 || stats.DeletedScreenshots != 1 || stats.DeletedTimelineBlocks != 1 || stats.DeletedSessions != 1 || !stats.DeletedWriterState {
		t.Fatalf("stats=%+v", stats)
	}
	assertDirEmpty(t, paths.MemoryDir())
	assertDirEmpty(t, paths.ObservationBufferDir())
	assertNotExists(t, paths.ScreenshotsDir())
	assertExists(t, outsideShot)
	assertNotExists(t, paths.WriterState())
	assertExists(t, paths.ConfigFile())
	assertExists(t, filepath.Join(paths.LogsDir(), "memfab.log"))
	assertExists(t, paths.PIDFile())
	assertExists(t, paths.PausedFlag())
	for _, table := range []string{"files", "entries", "ocr_observations", "timeline_blocks", "sessions"} {
		if got := countRows(t, db, table); got != 0 {
			t.Fatalf("%s rows=%d, want 0", table, got)
		}
	}
	insertObservation(t, db, "2026-06-18T10:03:00Z", "fresh after reset")
	hits, err := db.SearchObservations("fresh", "", "", "", 10)
	if err != nil || len(hits) != 1 {
		t.Fatalf("fresh search hits=%+v err=%v", hits, err)
	}
}

func TestRunDoesNotCreateMissingScreenshotsDir(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MEMFAB_ROOT", root)
	if err := paths.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	assertNotExists(t, paths.ScreenshotsDir())

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := ocReset.Run(db); err != nil {
		t.Fatal(err)
	}
	assertNotExists(t, paths.ScreenshotsDir())
}

func insertObservation(t *testing.T, db *store.DB, ts, text string) {
	t.Helper()
	o := capture.Observation{
		SchemaVersion: capture.SchemaVersion,
		Type:          capture.ObservationType,
		CapturedAt:    ts,
		CaptureID:     "reset:" + ts,
		Source:        capture.Source{Kind: "screen_ocr", Device: "test"},
		Context:       capture.Context{AppName: "ScreenOCR", Title: "Reset"},
		Event:         capture.Event{Type: "ocr_capture", Reason: "test"},
		OCR:           capture.OCR{Text: text},
		Tags:          []string{"ocr"},
	}
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

func touchTestFile(t *testing.T, path string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertDirEmpty(t *testing.T, path string) {
	t.Helper()
	entries, err := os.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("%s entries=%d, want empty", path, len(entries))
	}
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("%s should exist: %v", path, err)
	}
}

func assertNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("%s exists or unexpected err=%v", path, err)
	}
}

func countRows(t *testing.T, db *store.DB, table string) int {
	t.Helper()
	var n int
	if err := db.Conn.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
