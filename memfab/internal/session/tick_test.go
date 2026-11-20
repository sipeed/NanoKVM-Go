package session

import (
	"context"
	"testing"
	"time"

	"memfab/internal/capture"
	"memfab/internal/config"
	"memfab/internal/store"
)

func sessionTestDB(t *testing.T) *store.DB {
	t.Helper()
	t.Setenv("MEMFAB_ROOT", t.TempDir())
	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func insertSessionObservation(t *testing.T, db *store.DB, ts, app string) {
	t.Helper()
	o := capture.Observation{
		SchemaVersion: capture.SchemaVersion,
		Type:          capture.ObservationType,
		CapturedAt:    ts,
		CaptureID:     "session:" + ts,
		Source:        capture.Source{Kind: "screen_ocr", Device: "test"},
		Context:       capture.Context{AppName: app, Title: app + " title"},
		Event:         capture.Event{Type: "ocr_capture", Reason: "test"},
		OCR:           capture.OCR{Text: app + " text"},
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

func testConfig() config.Config {
	return config.Config{Session: config.SessionConfig{GapMinutes: 5, SoftCutMinutes: 3, MaxSessionHours: 2, TickSeconds: 30}}
}

func TestTickStartsActiveSession(t *testing.T) {
	db := sessionTestDB(t)
	insertSessionObservation(t, db, "2026-05-14T10:00:00Z", "AppA")
	result, err := Tick(context.Background(), db, testConfig(), mustTime(t, "2026-05-14T10:01:00Z"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Scanned != 1 || result.Started != 1 || result.Ended != 0 || result.ActiveID == "" {
		t.Fatalf("result=%+v", result)
	}
	active, err := db.GetActiveSession()
	if err != nil || active == nil || active.StartTime != "2026-05-14T10:00:00Z" {
		t.Fatalf("active=%+v err=%v", active, err)
	}
}

func TestTickEndsSessionAfterGapAndIsIdempotent(t *testing.T) {
	db := sessionTestDB(t)
	insertSessionObservation(t, db, "2026-05-14T10:00:00Z", "AppA")
	insertSessionObservation(t, db, "2026-05-14T10:01:00Z", "AppA")
	result, err := Tick(context.Background(), db, testConfig(), mustTime(t, "2026-05-14T10:07:00Z"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Started != 1 || result.Ended != 1 || result.ActiveID != "" {
		t.Fatalf("result=%+v", result)
	}
	rows, err := db.ListSessions("ended", 10)
	if err != nil || len(rows) != 1 || rows[0].EndTime != "2026-05-14T10:01:00Z" {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
	result, err = Tick(context.Background(), db, testConfig(), mustTime(t, "2026-05-14T10:08:00Z"))
	if err != nil {
		t.Fatal(err)
	}
	rows, _ = db.ListSessions("all", 10)
	if result.Started != 0 || result.Ended != 0 || len(rows) != 1 {
		t.Fatalf("second result=%+v rows=%+v", result, rows)
	}
}

func TestTickCreatesMultipleHistoricalSessions(t *testing.T) {
	db := sessionTestDB(t)
	insertSessionObservation(t, db, "2026-05-14T10:00:00Z", "AppA")
	insertSessionObservation(t, db, "2026-05-14T10:10:00Z", "AppA")
	result, err := Tick(context.Background(), db, testConfig(), mustTime(t, "2026-05-14T10:16:00Z"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Started != 2 || result.Ended != 2 {
		t.Fatalf("result=%+v", result)
	}
	rows, err := db.ListSessions("ended", 10)
	if err != nil || len(rows) != 2 {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
}

func TestTickRecoversActiveSession(t *testing.T) {
	db := sessionTestDB(t)
	insertSessionObservation(t, db, "2026-05-14T10:00:00Z", "AppA")
	_, err := db.InsertSession(store.SessionRow{ID: "sess_existing", StartTime: "2026-05-14T10:00:00Z", Status: string(store.SessionActive), CreatedAt: "2026-05-14T10:00:00Z", UpdatedAt: "2026-05-14T10:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := Tick(context.Background(), db, testConfig(), mustTime(t, "2026-05-14T10:07:00Z"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Started != 0 || result.Ended != 1 {
		t.Fatalf("result=%+v", result)
	}
	row, _ := db.GetSession("sess_existing")
	if row == nil || row.Status != string(store.SessionEnded) {
		t.Fatalf("row=%+v", row)
	}
}

func TestForceEndActiveUsesRequestedEndTime(t *testing.T) {
	db := sessionTestDB(t)
	_, err := db.InsertSession(store.SessionRow{ID: "sess_force", StartTime: "2026-05-14T10:00:00Z", Status: string(store.SessionActive), CreatedAt: "2026-05-14T10:00:00Z", UpdatedAt: "2026-05-14T10:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	now := mustTime(t, "2026-05-14T10:03:00Z")
	ended, err := ForceEndActive(context.Background(), db, now)
	if err != nil {
		t.Fatal(err)
	}
	if !ended {
		t.Fatal("expected active session to be force-ended")
	}
	row, err := db.GetSession("sess_force")
	if err != nil {
		t.Fatal(err)
	}
	if row == nil || row.Status != string(store.SessionEnded) || row.EndTime != "2026-05-14T10:03:00Z" {
		t.Fatalf("row=%+v", row)
	}
	ended, err = ForceEndActive(context.Background(), db, mustTime(t, "2026-05-14T10:04:00Z"))
	if err != nil {
		t.Fatal(err)
	}
	if ended {
		t.Fatal("expected second force-end to be a no-op")
	}
}

func mustTime(t *testing.T, ts string) time.Time {
	t.Helper()
	parsed, err := capture.ParseTimestamp(ts)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

// Stock writes legacy session bounds without fractional seconds even though
// its deterministic session ID includes the original observation's nanoseconds.
func TestTickResumesAfterLegacyReducedFractionalObservation(t *testing.T) {
	db := sessionTestDB(t)
	const legacyEnd = "2026-11-20T13:46:11Z"
	const first = "2026-11-20T13:46:11.696033177Z"
	const second = "2026-11-20T13:46:11.932573221Z"
	oldID := sessionID(mustTime(t, first))
	_, err := db.InsertSession(store.SessionRow{ID: oldID, StartTime: legacyEnd, EndTime: legacyEnd,
		Status: string(store.SessionReduced), CreatedAt: legacyEnd, UpdatedAt: legacyEnd})
	if err != nil {
		t.Fatal(err)
	}
	insertSessionObservation(t, db, first, "App")
	now := mustTime(t, "2026-11-20T13:46:32Z")
	var results []Result
	for i := 0; i < 3; i++ {
		if i == 1 {
			insertSessionObservation(t, db, second, "App")
		}
		r, err := Tick(context.Background(), db, testConfig(), now)
		if err != nil {
			t.Fatal(err)
		}
		results = append(results, r)
	}
	active, err := db.GetActiveSession()
	if err != nil {
		t.Fatal(err)
	}
	if results[0].Started != 0 || results[0].ActiveID != "" || results[1].Started != 1 || results[2].Started != 0 || active == nil || active.ID != sessionID(mustTime(t, second)) || active.StartTime != second {
		t.Fatalf("legacy fractional observation was replayed: started %d/%d/%d, active row present %t", results[0].Started, results[1].Started, results[2].Started, active != nil)
	}
	previous, err := db.GetSession(oldID)
	if err != nil || previous == nil || previous.Status != string(store.SessionReduced) || previous.EndTime != legacyEnd {
		t.Fatalf("legacy record changed: %+v %v", previous, err)
	}
}

func TestTickStoresExactFractionalBounds(t *testing.T) {
	db := sessionTestDB(t)
	const start = "2026-11-20T14:44:26.167030070Z"
	const end = "2026-11-20T14:44:47.485913100Z"
	insertSessionObservation(t, db, start, "App")
	insertSessionObservation(t, db, end, "App")
	for i := 0; i < 2; i++ {
		r, err := Tick(context.Background(), db, testConfig(), mustTime(t, "2026-11-20T15:00:00Z"))
		if err != nil {
			t.Fatal(err)
		}
		if i == 1 && (r.Scanned != 0 || r.Started != 0 || r.Ended != 0) {
			t.Fatalf("closed observation replayed: %+v", r)
		}
	}
	row, err := db.GetSession(sessionID(mustTime(t, start)))
	if err != nil || row == nil {
		t.Fatalf("missing session: %v", err)
	}
	if !mustTime(t, row.StartTime).Equal(mustTime(t, start)) || !mustTime(t, row.EndTime).Equal(mustTime(t, end)) {
		t.Fatal("stored session bounds lost precision")
	}
}
