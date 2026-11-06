package replay

import (
	"strings"
	"testing"

	"memfab/internal/capture"
	"memfab/internal/store"
)

func replaySearchTestDB(t *testing.T) *store.DB {
	t.Helper()
	t.Setenv("MEMFAB_ROOT", t.TempDir())
	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func insertReplaySearchObservation(t *testing.T, db *store.DB, ts, title, text string, captureID ...string) string {
	t.Helper()
	id := "test:" + ts
	if len(captureID) > 0 {
		id = captureID[0]
	}
	o := capture.Observation{
		SchemaVersion: capture.SchemaVersion,
		Type:          capture.ObservationType,
		CapturedAt:    ts,
		CaptureID:     id,
		Source:        capture.Source{Kind: "screen_ocr", Device: "test"},
		Context:       capture.Context{AppName: "ScreenOCR", Title: title},
		Event:         capture.Event{Type: "ocr_capture", Reason: "manual"},
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
	return stem
}

func TestSearchTimesByOCRSearchesOnlyOCRText(t *testing.T) {
	db := replaySearchTestDB(t)
	insertReplaySearchObservation(t, db, "2026-05-19T10:00:00Z", "Plain", "PicoClaw appears in OCR text", "session-id:42")
	insertReplaySearchObservation(t, db, "2026-05-19T10:01:00Z", "PicoClaw title only", "ordinary OCR text")

	result, err := SearchTimesByOCR(db, SearchOptions{Query: "PicoClaw"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Count != 1 || len(result.Results) != 1 {
		t.Fatalf("result=%+v", result)
	}
	if result.Results[0].CapturedAt != "2026-05-19T10:00:00Z" {
		t.Fatalf("hit=%+v", result.Results[0])
	}
	if result.Results[0].CaptureID != "session-id:42" {
		t.Fatalf("capture_id=%q", result.Results[0].CaptureID)
	}
	if !strings.Contains(result.Results[0].OCRPreview, "[PicoClaw]") {
		t.Fatalf("preview=%q", result.Results[0].OCRPreview)
	}
}

func TestSearchTimesByOCRHonorsTimeRangeAndLimit(t *testing.T) {
	db := replaySearchTestDB(t)
	insertReplaySearchObservation(t, db, "2026-05-19T10:00:00Z", "First", "target")
	insertReplaySearchObservation(t, db, "2026-05-19T10:01:00Z", "Second", "target")

	result, err := SearchTimesByOCR(db, SearchOptions{
		Query: "target",
		Since: "2026-05-19T10:00:30Z",
		Until: "2026-05-19T10:02:00Z",
		Limit: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Count != 1 || result.Results[0].CapturedAt != "2026-05-19T10:01:00Z" {
		t.Fatalf("result=%+v", result)
	}
}
