package store

import (
	"strings"
	"testing"
)

func TestObservationSchemaRequiresCaptureID(t *testing.T) {
	db := testDB(t)
	if _, err := db.Conn.Exec(`INSERT INTO ocr_observations (id, captured_at, capture_id, ocr_text) VALUES (?, ?, ?, ?)`,
		"missing-capture-id", "2026-07-14T05:43:21Z", nil, "missing capture id"); err == nil {
		t.Fatal("expected NOT NULL capture_id constraint")
	}
}

func TestInsertObservationRejectsEmptyCaptureID(t *testing.T) {
	db := testDB(t)
	o := observationAt("2026-07-14T05:43:21Z", "ScreenOCR", "Missing", "missing capture id")
	o.CaptureID = " \t"
	if err := db.InsertObservation("missing-capture-id", o); err == nil || !strings.Contains(err.Error(), "capture_id is required") {
		t.Fatalf("err=%v, want capture_id validation", err)
	}
}

func TestInsertObservationUpsertsCaptureIDIntoReplayRows(t *testing.T) {
	db := testDB(t)
	o := observationAt("2026-07-14T05:43:21Z", "ScreenOCR", "Capture ID", "capture id replay needle")
	o.CaptureID = "session-id:41"
	stem := insertObservationForTest(t, db, o)

	o.CaptureID = "session-id:42"
	if err := db.InsertObservation(stem, o); err != nil {
		t.Fatal(err)
	}
	var storedCaptureID string
	if err := db.Conn.QueryRow(`SELECT capture_id FROM ocr_observations WHERE id=?`, stem).Scan(&storedCaptureID); err != nil {
		t.Fatal(err)
	}
	if storedCaptureID != o.CaptureID {
		t.Fatalf("stored capture_id=%q, want %q", storedCaptureID, o.CaptureID)
	}

	ocrHits, err := db.SearchOCRHits("replay needle", OCRSearchOptions{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(ocrHits) != 1 || ocrHits[0].CaptureID != o.CaptureID {
		t.Fatalf("OCR hits=%+v", ocrHits)
	}
}
