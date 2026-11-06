package capture_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"memfab/internal/capture"
	"memfab/internal/config"
	"memfab/internal/paths"
	"memfab/internal/store"
)

func cleanupTestDB(t *testing.T) *store.DB {
	t.Helper()
	t.Setenv("MEMFAB_ROOT", t.TempDir())
	if err := paths.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func writeCleanupObservation(t *testing.T, db *store.DB, capturedAt, text string) string {
	t.Helper()
	obs := capture.Observation{
		CapturedAt: capturedAt,
		CaptureID:  "cleanup:" + capturedAt,
		Source:     capture.Source{Kind: "screen_ocr"},
		OCR:        capture.OCR{Text: text},
	}
	if err := obs.Normalize(func() time.Time { return time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC) }); err != nil {
		t.Fatal(err)
	}
	path, err := capture.WriteAtomic(obs)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.InsertObservation(capture.StemFromPath(path), obs); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCleanupBufferDeletesExpiredAbsorbedObservations(t *testing.T) {
	db := cleanupTestDB(t)
	oldPath := writeCleanupObservation(t, db, "2026-05-14T10:00:00Z", "old Sipeed OCR")
	newPath := writeCleanupObservation(t, db, "2026-05-14T12:00:00Z", "new Sipeed OCR")
	now := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	stats, err := capture.CleanupBuffer(db, config.CaptureConfig{BufferRetentionHours: 1, BufferMaxMB: 0}, "2026-05-14T11:00:00Z", now)
	if err != nil {
		t.Fatal(err)
	}
	if stats.DeletedObservations != 1 {
		t.Fatalf("stats=%+v", stats)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("old file still exists err=%v", err)
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("new file missing err=%v", err)
	}
	hits, err := db.SearchObservations("old", "", "", "", 10)
	if err != nil || len(hits) != 0 {
		t.Fatalf("old hits=%+v err=%v", hits, err)
	}
	hits, err = db.SearchObservations("new", "", "", "", 10)
	if err != nil || len(hits) != 1 {
		t.Fatalf("new hits=%+v err=%v", hits, err)
	}
}

func TestCleanupBufferNoopsWithoutProcessedBefore(t *testing.T) {
	db := cleanupTestDB(t)
	path := writeCleanupObservation(t, db, "2026-05-14T10:00:00Z", "old OCR")
	stats, err := capture.CleanupBuffer(db, config.CaptureConfig{BufferRetentionHours: 1}, "", time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if stats.DeletedObservations != 0 {
		t.Fatalf("stats=%+v", stats)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file missing err=%v", err)
	}
}

func TestCleanupBufferEvictsOldestAbsorbedForSizeCap(t *testing.T) {
	db := cleanupTestDB(t)
	oldPath := writeCleanupObservation(t, db, "2026-05-14T10:00:00Z", strings.Repeat("old ", 300000))
	newPath := writeCleanupObservation(t, db, "2026-05-14T10:10:00Z", "new OCR")
	stats, err := capture.CleanupBuffer(db, config.CaptureConfig{BufferRetentionHours: 999, BufferMaxMB: 1}, "2026-05-14T10:05:00Z", time.Date(2026, 5, 14, 11, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if stats.EvictedObservations != 1 {
		t.Fatalf("stats=%+v", stats)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("old absorbed file still exists err=%v", err)
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("new unabsorbed file missing err=%v", err)
	}
}
