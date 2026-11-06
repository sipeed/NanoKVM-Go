package ingest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"memfab/internal/capture"
	"memfab/internal/config"
	"memfab/internal/paths"
	"memfab/internal/store"
)

func TestResetEndpointDeletesData(t *testing.T) {
	db, cfg := setupIngestTest(t)
	insertIngestObservation(t, db, "2026-06-18T10:00:00Z", "reset endpoint target")

	req := httptest.NewRequest(http.MethodPost, "/v1/reset", nil)
	rec := httptest.NewRecorder()
	Handler(db, cfg).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out response
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if !out.OK || out.Reset == nil || out.Reset.DeletedObservations != 1 {
		t.Fatalf("response=%+v", out)
	}
	hits, err := db.SearchObservations("target", "", "", "", 10)
	if err != nil || len(hits) != 0 {
		t.Fatalf("hits after reset=%+v err=%v", hits, err)
	}
}

func TestResetEndpointRequiresPostAndToken(t *testing.T) {
	db, cfg := setupIngestTest(t)
	cfg.Ingest.RequireToken = true
	cfg.Ingest.Token = "secret"
	handler := Handler(db, cfg)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/reset", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/reset", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status=%d body=%s", rec.Code, rec.Body.String())
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/reset", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("authorized status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestObservationEndpointRequiresNonEmptyCaptureID(t *testing.T) {
	db, cfg := setupIngestTest(t)
	handler := Handler(db, cfg)
	for _, body := range []string{
		`{"schema_version":1,"type":"ocr_observation","captured_at":"2026-07-14T05:43:21Z","ocr":{"text":"missing capture id"}}`,
		`{"schema_version":1,"type":"ocr_observation","captured_at":"2026-07-14T05:43:21Z","capture_id":" \t ","ocr":{"text":"blank capture id"}}`,
	} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/observations", strings.NewReader(body)))
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "capture_id is required") {
			t.Fatalf("body=%s status=%d response=%s", body, rec.Code, rec.Body.String())
		}
	}
}

func setupIngestTest(t *testing.T) (*store.DB, config.Config) {
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
	return db, config.Default()
}

func insertIngestObservation(t *testing.T, db *store.DB, ts, text string) {
	t.Helper()
	o := capture.Observation{
		SchemaVersion: capture.SchemaVersion,
		Type:          capture.ObservationType,
		CapturedAt:    ts,
		CaptureID:     "ingest:" + ts,
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
