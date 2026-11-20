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

func TestTickRestoresAppBeforeCheckingForSwitches(t *testing.T) {
	for _, tc := range []struct {
		name       string
		openingApp string
		pruned     bool
		wantEnded  int
	}{
		{name: "same app", openingApp: "Browser"},
		{name: "opening capture pruned", openingApp: "Browser", pruned: true},
		{name: "real switch", openingApp: "Terminal", wantEnded: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := sessionTestDB(t)
			start := mustTime(t, "2026-05-14T16:37:12Z")
			insertSessionObservation(t, db, start.Format(time.RFC3339), tc.openingApp)
			first, err := Tick(context.Background(), db, testConfig(), start)
			if err != nil || first.ActiveID == "" {
				t.Fatalf("start session: %+v, %v", first, err)
			}
			if tc.pruned {
				stem, err := capture.SafeFilename(start.Format(time.RFC3339))
				if err != nil {
					t.Fatal(err)
				}
				if err := db.DeleteObservation(stem); err != nil {
					t.Fatal(err)
				}
			}
			for n := 1; n <= 10; n++ {
				ts := start.Add(time.Duration(n) * 21 * time.Second)
				insertSessionObservation(t, db, ts.Format(time.RFC3339), "Browser")
			}
			resumed, err := Tick(context.Background(), db, testConfig(), start.Add(210*time.Second))
			if err != nil || resumed.Ended != tc.wantEnded {
				t.Fatalf("resumed: %+v, want %d ended, error %v", resumed, tc.wantEnded, err)
			}
			if tc.wantEnded == 0 && resumed.ActiveID != first.ActiveID {
				t.Fatalf("active session changed from %s to %s", first.ActiveID, resumed.ActiveID)
			}
		})
	}
}
