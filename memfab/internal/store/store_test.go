package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"memfab/internal/capture"
	"memfab/internal/paths"
)

func testDB(t *testing.T) *DB {
	t.Helper()
	t.Setenv("MEMFAB_ROOT", t.TempDir())
	db, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestOpenUsesPrivateSQLiteFiles(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MEMFAB_ROOT", root)
	db, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	for _, path := range []string{paths.IndexDB(), paths.IndexDB() + "-wal", paths.IndexDB() + "-shm"} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %s: %v", path, err)
		}
		if got := info.Mode().Perm(); got != paths.FileMode {
			t.Fatalf("%s mode=%#o, want %#o", path, got, paths.FileMode)
		}
	}
}

func observationAt(ts, app, title, text string) capture.Observation {
	return capture.Observation{
		SchemaVersion: capture.SchemaVersion,
		Type:          capture.ObservationType,
		CapturedAt:    ts,
		CaptureID:     "test:" + ts,
		Source:        capture.Source{Kind: "screen_ocr", Device: "test"},
		Context:       capture.Context{AppName: app, Title: title},
		Event:         capture.Event{Type: "ocr_capture", Reason: "test"},
		OCR:           capture.OCR{Text: text},
		Tags:          []string{"ocr"},
	}
}

func observationScreenAt(ts, app, title, text, hash string, width, height int, confidence float64) capture.Observation {
	o := observationAt(ts, app, title, text)
	o.Screen = capture.Screen{Width: &width, Height: &height, ImageSHA256: hash}
	o.OCR.Confidence = &confidence
	o.OCR.BlockCount = 2
	return o
}

func insertObservationForTest(t *testing.T, db *DB, o capture.Observation) string {
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
	return stem
}

func TestSearchOCRHitsSearchesOnlyOCRText(t *testing.T) {
	db := testDB(t)
	titleOnly := insertObservationForTest(t, db, observationScreenAt("2026-05-14T04:00:00Z", "ScreenOCR", "PicoClaw Title", "ordinary OCR text", "hash-title", 1280, 720, 0.91))
	textHit := insertObservationForTest(t, db, observationScreenAt("2026-05-14T04:01:00Z", "ScreenOCR", "Plain Title", "PicoClaw OCR text", "hash-text", 1280, 720, 0.95))

	hits, err := db.SearchOCRHits("PicoClaw", OCRSearchOptions{Since: "2026-05-14T03:59:00Z", Until: "2026-05-14T04:02:00Z", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].CapturedAt != "2026-05-14T04:01:00Z" {
		t.Fatalf("hits=%+v titleOnly=%s textHit=%s", hits, titleOnly, textHit)
	}
	if !strings.Contains(hits[0].OCRPreview, "[PicoClaw]") {
		t.Fatalf("preview=%q", hits[0].OCRPreview)
	}
}

func TestSearchOCRHitsLikeFallbackReturnsPreview(t *testing.T) {
	db := testDB(t)
	db.hasFTS5 = false
	insertObservationForTest(t, db, observationScreenAt("2026-05-14T04:01:00Z", "ScreenOCR", "Plain Title", "PicoClaw OCR text", "hash-text", 1280, 720, 0.95))

	hits, err := db.SearchOCRHits("PicoClaw", OCRSearchOptions{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || !strings.Contains(hits[0].OCRPreview, "[PicoClaw]") {
		t.Fatalf("hits=%+v", hits)
	}
}

func TestSearchOCRHitsFallsBackForHanSubstring(t *testing.T) {
	db := testDB(t)
	textHit := insertObservationForTest(t, db, observationScreenAt("2026-05-14T04:01:00Z", "ScreenOCR", "Plain Title", "蓝牙 背景 硬件传感指示器 外观", "hash-text", 1280, 720, 0.95))
	titleOnly := insertObservationForTest(t, db, observationScreenAt("2026-05-14T04:02:00Z", "ScreenOCR", "硬件 Title", "ordinary OCR text", "hash-title", 1280, 720, 0.91))

	ftsHits, err := db.SearchOCRHits("蓝牙", OCRSearchOptions{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(ftsHits) != 1 || ftsHits[0].CapturedAt != "2026-05-14T04:01:00Z" {
		t.Fatalf("ftsHits=%+v textHit=%s", ftsHits, textHit)
	}
	if !strings.Contains(ftsHits[0].OCRPreview, "[蓝牙]") {
		t.Fatalf("fts preview=%q", ftsHits[0].OCRPreview)
	}

	fallbackHits, err := db.SearchOCRHits("硬件", OCRSearchOptions{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(fallbackHits) != 1 || fallbackHits[0].CapturedAt != "2026-05-14T04:01:00Z" {
		t.Fatalf("fallbackHits=%+v textHit=%s titleOnly=%s", fallbackHits, textHit, titleOnly)
	}
	if !strings.Contains(fallbackHits[0].OCRPreview, "[硬件]传感指示器") {
		t.Fatalf("fallback preview=%q", fallbackHits[0].OCRPreview)
	}
}

func TestSearchOCRHitsFallsBackForASCIISubstring(t *testing.T) {
	db := testDB(t)
	textHit := insertObservationForTest(t, db, observationScreenAt("2026-05-14T04:01:00Z", "ScreenOCR", "Plain Title", "hidden napdUserSessionAgent text", "hash-text", 1280, 720, 0.95))
	titleOnly := insertObservationForTest(t, db, observationScreenAt("2026-05-14T04:02:00Z", "ScreenOCR", "napdUser Title", "ordinary OCR text", "hash-title", 1280, 720, 0.91))

	hits, err := db.SearchOCRHits("napdUser", OCRSearchOptions{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].CapturedAt != "2026-05-14T04:01:00Z" {
		t.Fatalf("hits=%+v textHit=%s titleOnly=%s", hits, textHit, titleOnly)
	}
	if !strings.Contains(hits[0].OCRPreview, "[napdUser]SessionAgent") {
		t.Fatalf("preview=%q", hits[0].OCRPreview)
	}
}

func TestSearchOCRHitsHonorsLimit(t *testing.T) {
	db := testDB(t)
	insertObservationForTest(t, db, observationScreenAt("2026-05-14T04:00:00Z", "ScreenOCR", "First", "needle", "hash-first", 1280, 720, 0.91))
	insertObservationForTest(t, db, observationScreenAt("2026-05-14T04:01:00Z", "ScreenOCR", "Second", "needle", "hash-second", 1280, 720, 0.95))

	hits, err := db.SearchOCRHits("needle", OCRSearchOptions{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("hits=%+v", hits)
	}
}

func TestDeleteObservationRemovesSearchHit(t *testing.T) {
	db := testDB(t)
	stem := insertObservationForTest(t, db, observationAt("2026-05-12T18:00:00+08:00", "ScreenOCR", "Delete", "delete target text"))
	hits, err := db.SearchObservations("target", "", "", "", 10)
	if err != nil || len(hits) != 1 {
		t.Fatalf("hits=%+v err=%v", hits, err)
	}
	if err := db.DeleteObservation(stem); err != nil {
		t.Fatal(err)
	}
	hits, err = db.SearchObservations("target", "", "", "", 10)
	if err != nil || len(hits) != 0 {
		t.Fatalf("hits after delete=%+v err=%v", hits, err)
	}
}

func TestFindRecentObservationStem(t *testing.T) {
	db := testDB(t)
	old := insertObservationForTest(t, db, observationAt("2026-05-12T19:00:00+08:00", "ScreenOCR", "Old Snapshot", "old"))
	newest := insertObservationForTest(t, db, observationAt("2026-05-12T20:00:00+08:00", "ScreenOCR", "New Snapshot", "new"))
	other := insertObservationForTest(t, db, observationAt("2026-05-12T21:00:00+08:00", "OtherApp", "Other Title", "other"))

	stem, found, err := db.FindRecentObservationStem("", "", "")
	if err != nil || !found || stem != other {
		t.Fatalf("default stem=%q found=%v err=%v, want %q", stem, found, err, other)
	}
	stem, found, err = db.FindRecentObservationStem("2026-05-12T20:30:00+08:00", "", "")
	if err != nil || !found || stem != newest {
		t.Fatalf("at stem=%q found=%v err=%v, want %q", stem, found, err, newest)
	}
	stem, found, err = db.FindRecentObservationStem("2026-05-12T19:30:00+08:00", "", "")
	if err != nil || !found || stem != old {
		t.Fatalf("at old stem=%q found=%v err=%v, want %q", stem, found, err, old)
	}
	stem, found, err = db.FindRecentObservationStem("", "screen", "new")
	if err != nil || !found || stem != newest {
		t.Fatalf("filters stem=%q found=%v err=%v, want %q", stem, found, err, newest)
	}
	_, found, err = db.FindRecentObservationStem("", "missing", "")
	if err != nil || found {
		t.Fatalf("missing found=%v err=%v", found, err)
	}
}

func TestGetTimelineBlock(t *testing.T) {
	db := testDB(t)
	_, err := db.InsertTimelineBlock(TimelineBlock{
		ID:           "tl_test",
		StartTime:    "2026-05-12T20:00:00+08:00",
		EndTime:      "2026-05-12T20:01:00+08:00",
		Timezone:     "+08:00",
		Entries:      []string{"[ScreenOCR] First: alpha"},
		AppsUsed:     []string{"ScreenOCR"},
		CaptureCount: 2,
		CreatedAt:    "2026-05-12T20:02:00+08:00",
	})
	if err != nil {
		t.Fatal(err)
	}
	block, err := db.GetTimelineBlock("tl_test")
	if err != nil {
		t.Fatal(err)
	}
	if block == nil || block.ID != "tl_test" || len(block.Entries) != 1 || block.Entries[0] != "[ScreenOCR] First: alpha" {
		t.Fatalf("block=%+v", block)
	}
	missing, err := db.GetTimelineBlock("missing")
	if err != nil || missing != nil {
		t.Fatalf("missing=%+v err=%v", missing, err)
	}
}

func TestMemoryEntriesByPath(t *testing.T) {
	db := testDB(t)
	if err := db.InsertMemoryEntry(MemoryEntryRow{ID: "entry-a", Path: "event-2026-05-19.md", Prefix: "event", Timestamp: "2026-05-19T10:00:00Z", Tags: "event session", Content: "first"}); err != nil {
		t.Fatal(err)
	}
	if err := db.InsertMemoryEntry(MemoryEntryRow{ID: "entry-b", Path: "event-2026-05-19.md", Prefix: "event", Timestamp: "2026-05-19T10:05:00Z", Tags: "event", Content: "second"}); err != nil {
		t.Fatal(err)
	}
	if err := db.InsertMemoryEntry(MemoryEntryRow{ID: "entry-old", Path: "event-2026-05-19.md", Prefix: "event", Timestamp: "2026-05-19T09:00:00Z", Tags: "event", Content: "old", Superseded: true}); err != nil {
		t.Fatal(err)
	}

	entries, err := db.MemoryEntriesByPath("/tmp/event-2026-05-19.md", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].ID != "entry-a" || entries[1].ID != "entry-b" {
		t.Fatalf("entries=%+v", entries)
	}
	entries, err = db.MemoryEntriesByPath("event-2026-05-19.md", "entry-b", false)
	if err != nil || len(entries) != 1 || entries[0].ID != "entry-b" {
		t.Fatalf("entry-b=%+v err=%v", entries, err)
	}
	entries, err = db.MemoryEntriesByPath("event-2026-05-19.md", "entry-old", false)
	if err != nil || len(entries) != 0 {
		t.Fatalf("superseded excluded=%+v err=%v", entries, err)
	}
	entries, err = db.MemoryEntriesByPath("event-2026-05-19.md", "entry-old", true)
	if err != nil || len(entries) != 1 || !entries[0].Superseded {
		t.Fatalf("superseded included=%+v err=%v", entries, err)
	}
}

func TestTimelineBlockStore(t *testing.T) {
	db := testDB(t)
	insertObservationForTest(t, db, observationAt("2026-05-12T20:00:10+08:00", "ScreenOCR", "First", "alpha"))
	insertObservationForTest(t, db, observationAt("2026-05-12T20:00:40+08:00", "ScreenOCR", "Second", "beta"))
	insertObservationForTest(t, db, observationAt("2026-05-12T20:01:00+08:00", "ScreenOCR", "Third", "gamma"))

	obs, err := db.ObservationsInWindow("2026-05-12T20:00:00+08:00", "2026-05-12T20:01:00+08:00", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(obs) != 2 || obs[0].Title != "First" || obs[1].Title != "Second" {
		t.Fatalf("obs=%+v", obs)
	}

	inserted, err := db.InsertTimelineBlock(TimelineBlock{
		ID:           "2026-05-12T20:00:00+08:00__2026-05-12T20:01:00+08:00",
		StartTime:    "2026-05-12T20:00:00+08:00",
		EndTime:      "2026-05-12T20:01:00+08:00",
		Timezone:     "+08:00",
		Entries:      []string{"[ScreenOCR] First: alpha"},
		AppsUsed:     []string{"ScreenOCR"},
		CaptureCount: 2,
		CreatedAt:    "2026-05-12T20:02:00+08:00",
	})
	if err != nil || !inserted {
		t.Fatalf("inserted=%v err=%v", inserted, err)
	}
	inserted, err = db.InsertTimelineBlock(TimelineBlock{
		ID:           "duplicate",
		StartTime:    "2026-05-12T20:00:00+08:00",
		EndTime:      "2026-05-12T20:01:00+08:00",
		Entries:      []string{"duplicate"},
		AppsUsed:     []string{"ScreenOCR"},
		CaptureCount: 2,
		CreatedAt:    "2026-05-12T20:02:01+08:00",
	})
	if err != nil || inserted {
		t.Fatalf("duplicate inserted=%v err=%v", inserted, err)
	}

	has, err := db.HasTimelineWindow("2026-05-12T20:00:00+08:00", "2026-05-12T20:01:00+08:00")
	if err != nil || !has {
		t.Fatalf("has=%v err=%v", has, err)
	}
	latest, found, err := db.LatestTimelineEnd()
	if err != nil || !found || latest != "2026-05-12T20:01:00+08:00" {
		t.Fatalf("latest=%q found=%v err=%v", latest, found, err)
	}
	blocks, err := db.ListTimelineBlocks(5)
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 1 || len(blocks[0].Entries) != 1 || blocks[0].Entries[0] != "[ScreenOCR] First: alpha" {
		t.Fatalf("blocks=%+v", blocks)
	}
}

func TestObservationsInWindowComparesRFC3339Offsets(t *testing.T) {
	db := testDB(t)
	insertObservationForTest(t, db, observationAt("2026-05-14T04:03:19Z", "ScreenOCR", "UTC", "utc text"))

	obs, err := db.ObservationsInWindow("2026-05-14T04:03:00Z", "2026-05-14T04:04:00Z", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(obs) != 1 || obs[0].Title != "UTC" {
		t.Fatalf("obs=%+v", obs)
	}
}

func TestMemoryIndexAndQueries(t *testing.T) {
	db := testDB(t)
	root := t.TempDir()
	writeMemory := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeMemory("project-alpha.md", `---
description: Alpha memory
tags: [alpha, test]
status: active
created: 2026-05-14T00:00:00Z
updated: 2026-05-14T00:10:00Z
---

## [2026-05-14T00:01:00Z] {id: alpha-entry} #alpha
Discussed NanoKVM-Go memory query work.

## [2026-05-14T00:02:00Z] {id: old-entry} #alpha #superseded-by:alpha-entry
Old superseded content.
`)
	writeMemory("topic-beta.md", `---
description: Beta memory
tags: [beta]
status: dormant
created: 2026-05-14T01:00:00Z
updated: 2026-05-14T01:05:00Z
---

## [2026-05-14T01:01:00Z] {id: beta-entry} #beta
~~Archived beta content.~~
`)
	writeMemory("bad-alpha.md", `---
description: Bad
---

## [2026-05-14T02:01:00Z] {id: bad-entry}
Bad prefix.
`)

	files, entries, skipped, err := db.RebuildMemoryIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	if files != 2 || entries != 3 || skipped != 1 {
		t.Fatalf("files=%d entries=%d skipped=%d", files, entries, skipped)
	}

	listed, err := db.ListMemoryFiles(false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].Path != "project-alpha.md" || listed[0].Tags[0] != "alpha" {
		t.Fatalf("listed=%+v", listed)
	}
	listed, err = db.ListMemoryFiles(true, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 2 || listed[0].Path != "topic-beta.md" {
		t.Fatalf("listed dormant=%+v", listed)
	}

	hits, err := db.SearchMemory("NanoKVM-Go", nil, "", "", 5, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].ID != "alpha-entry" {
		t.Fatalf("hits=%+v", hits)
	}
	hits, err = db.SearchMemory("superseded", []string{"project-*"}, "", "", 5, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].ID != "old-entry" || !hits[0].Superseded {
		t.Fatalf("superseded hits=%+v", hits)
	}

	recent, err := db.RecentMemory("2026-05-14T00:30:00Z", 10, []string{"topic-"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(recent) != 1 || recent[0].ID != "beta-entry" || !recent[0].Superseded || strings.Contains(recent[0].Content, "~~") {
		t.Fatalf("recent=%+v", recent)
	}
}

func TestSessionStoreCRUD(t *testing.T) {
	db := testDB(t)
	inserted, err := db.InsertSession(SessionRow{
		ID:        "sess_test",
		StartTime: "2026-05-14T00:00:00Z",
		Status:    string(SessionActive),
		CreatedAt: "2026-05-14T00:00:00Z",
		UpdatedAt: "2026-05-14T00:00:00Z",
	})
	if err != nil || !inserted {
		t.Fatalf("inserted=%v err=%v", inserted, err)
	}
	inserted, err = db.InsertSession(SessionRow{ID: "sess_test", StartTime: "2026-05-14T00:00:00Z", CreatedAt: "2026-05-14T00:00:00Z", UpdatedAt: "2026-05-14T00:00:00Z"})
	if err != nil || inserted {
		t.Fatalf("duplicate inserted=%v err=%v", inserted, err)
	}
	active, err := db.GetActiveSession()
	if err != nil || active == nil || active.ID != "sess_test" {
		t.Fatalf("active=%+v err=%v", active, err)
	}
	if err := db.MarkSessionEnded("sess_test", "2026-05-14T00:05:00Z", "2026-05-14T00:06:00Z"); err != nil {
		t.Fatal(err)
	}
	active, err = db.GetActiveSession()
	if err != nil || active != nil {
		t.Fatalf("active after end=%+v err=%v", active, err)
	}
	row, err := db.GetSession("sess_test")
	if err != nil || row == nil || row.Status != string(SessionEnded) || row.EndTime != "2026-05-14T00:05:00Z" {
		t.Fatalf("row=%+v err=%v", row, err)
	}
	latest, found, err := db.LatestSessionEnd()
	if err != nil || !found || latest != "2026-05-14T00:05:00Z" {
		t.Fatalf("latest=%q found=%v err=%v", latest, found, err)
	}
	rows, err := db.ListSessions("ended", 10)
	if err != nil || len(rows) != 1 || rows[0].ID != "sess_test" {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
}

func TestObservationScanHelpers(t *testing.T) {
	db := testDB(t)
	insertObservationForTest(t, db, observationAt("2026-05-14T00:02:00Z", "AppB", "B", "b"))
	insertObservationForTest(t, db, observationAt("2026-05-14T00:01:00Z", "AppA", "A", "a"))
	earliest, found, err := db.EarliestObservationTime()
	if err != nil || !found || earliest != "2026-05-14T00:01:00Z" {
		t.Fatalf("earliest=%q found=%v err=%v", earliest, found, err)
	}
	obs, err := db.SessionObservations("", false, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(obs) != 2 || obs[0].AppName != "AppA" || obs[1].AppName != "AppB" {
		t.Fatalf("obs=%+v", obs)
	}
	obs, err = db.SessionObservations("2026-05-14T00:01:00Z", false, 10)
	if err != nil || len(obs) != 1 || obs[0].AppName != "AppB" {
		t.Fatalf("since obs=%+v err=%v", obs, err)
	}
}

func TestReducerStoreHelpers(t *testing.T) {
	db := testDB(t)
	_, err := db.InsertTimelineBlock(TimelineBlock{ID: "tl1", StartTime: "2026-05-14T10:00:00Z", EndTime: "2026-05-14T10:01:00Z", Entries: []string{"[AppA] did A"}, AppsUsed: []string{"AppA"}, CaptureCount: 1, CreatedAt: "2026-05-14T10:01:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.InsertTimelineBlock(TimelineBlock{ID: "tl2", StartTime: "2026-05-14T10:02:00Z", EndTime: "2026-05-14T10:03:00Z", Entries: []string{"[AppB] did B"}, AppsUsed: []string{"AppB"}, CaptureCount: 1, CreatedAt: "2026-05-14T10:03:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	blocks, err := db.TimelineBlocksIntersecting("2026-05-14T10:00:30Z", "2026-05-14T10:02:30Z", 10)
	if err != nil || len(blocks) != 2 || blocks[0].ID != "tl1" || blocks[1].ID != "tl2" {
		t.Fatalf("blocks=%+v err=%v", blocks, err)
	}

	_, err = db.InsertSession(SessionRow{ID: "sess-ended", StartTime: "2026-05-14T10:00:00Z", EndTime: "2026-05-14T10:03:00Z", Status: string(SessionEnded), CreatedAt: "2026-05-14T10:03:00Z", UpdatedAt: "2026-05-14T10:03:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.InsertSession(SessionRow{ID: "sess-failed", StartTime: "2026-05-14T11:00:00Z", EndTime: "2026-05-14T11:03:00Z", Status: string(SessionFailed), RetryCount: 1, NextRetryAt: "2026-05-14T11:10:00Z", CreatedAt: "2026-05-14T11:03:00Z", UpdatedAt: "2026-05-14T11:03:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	pending, err := db.ListPendingReduction(10)
	if err != nil || len(pending) != 2 {
		t.Fatalf("pending=%+v err=%v", pending, err)
	}
	due, err := db.ListDueReductions("2026-05-14T11:09:00Z", 10)
	if err != nil || len(due) != 1 || due[0].ID != "sess-ended" {
		t.Fatalf("due early=%+v err=%v", due, err)
	}
	due, err = db.ListDueReductions("2026-05-14T11:11:00Z", 10)
	if err != nil || len(due) != 2 {
		t.Fatalf("due late=%+v err=%v", due, err)
	}
	if err := db.MarkSessionFailed("sess-ended", "bad json", "2026-05-14T10:08:00Z", "2026-05-14T10:04:00Z", 1); err != nil {
		t.Fatal(err)
	}
	row, _ := db.GetSession("sess-ended")
	if row.Status != string(SessionFailed) || row.RetryCount != 1 || row.NextRetryAt != "2026-05-14T10:08:00Z" {
		t.Fatalf("failed row=%+v", row)
	}
	if err := db.MarkSessionReduced("sess-ended", "2026-05-14T10:03:00Z", "2026-05-14T10:05:00Z"); err != nil {
		t.Fatal(err)
	}
	row, _ = db.GetSession("sess-ended")
	if row.Status != string(SessionReduced) || row.FlushEnd != "2026-05-14T10:03:00Z" {
		t.Fatalf("reduced row=%+v", row)
	}

	if err := db.UpsertMemoryFile(FileRow{Path: "event-2026-05-14.md", Prefix: "event", Description: "Event memory", Tags: []string{"event", "session"}, Status: "active", EntryCount: 1, Created: "2026-05-14T00:00:00Z", Updated: "2026-05-14T10:05:00Z"}); err != nil {
		t.Fatal(err)
	}
	if err := db.InsertMemoryEntry(MemoryEntryRow{ID: "entry1", Path: "event-2026-05-14.md", Prefix: "event", Timestamp: "2026-05-14T10:05:00Z", Tags: "event session", Content: "Reduced event content"}); err != nil {
		t.Fatal(err)
	}
	files, err := db.ListMemoryFiles(false, false)
	if err != nil || len(files) != 1 || files[0].EntryCount != 1 {
		t.Fatalf("files=%+v err=%v", files, err)
	}
	hits, err := db.SearchMemory("Reduced", nil, "", "", 5, false)
	if err != nil || len(hits) != 1 || hits[0].ID != "entry1" {
		t.Fatalf("hits=%+v err=%v", hits, err)
	}
}

func TestCompactStoreHelpers(t *testing.T) {
	db := testDB(t)
	root := t.TempDir()
	writeMemory := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeMemory("project-compact.md", `---
description: Compact memory
tags: [project]
status: active
created: 2026-05-14T00:00:00Z
updated: 2026-05-14T00:10:00Z
entry_count: 1
needs_compact: true
---

## [2026-05-14T00:01:00Z] {id: old-entry} #project
Old compact content.
`)
	if _, _, _, err := db.RebuildMemoryIndex(root); err != nil {
		t.Fatal(err)
	}
	pending, err := db.FilesNeedingCompact(10)
	if err != nil || len(pending) != 1 || pending[0].Path != "project-compact.md" {
		t.Fatalf("pending=%+v err=%v", pending, err)
	}
	writeMemory("project-compact.md", `---
description: Compact memory
tags: [project]
status: active
created: 2026-05-14T00:00:00Z
updated: 2026-05-14T00:20:00Z
entry_count: 1
needs_compact: false
---

## [2026-05-14T00:02:00Z] {id: new-entry} #project
New compact content.
`)
	if err := db.ReindexMemoryFile(root, "project-compact.md"); err != nil {
		t.Fatal(err)
	}
	hits, err := db.SearchMemory("Old", nil, "", "", 5, true)
	if err != nil || len(hits) != 0 {
		t.Fatalf("old hits=%+v err=%v", hits, err)
	}
	hits, err = db.SearchMemory("New", nil, "", "", 5, false)
	if err != nil || len(hits) != 1 || hits[0].ID != "new-entry" {
		t.Fatalf("new hits=%+v err=%v", hits, err)
	}
	pending, err = db.FilesNeedingCompact(10)
	if err != nil || len(pending) != 0 {
		t.Fatalf("pending after reindex=%+v err=%v", pending, err)
	}
}

func TestClassifierStoreHelpers(t *testing.T) {
	db := testDB(t)
	_, err := db.InsertSession(SessionRow{ID: "sess-classify", StartTime: "2026-05-14T10:00:00Z", EndTime: "2026-05-14T10:05:00Z", Status: string(SessionReduced), FlushEnd: "2026-05-14T10:05:00Z", CreatedAt: "2026-05-14T10:05:00Z", UpdatedAt: "2026-05-14T10:05:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	due, err := db.ListSessionsNeedingClassification("2026-05-14T10:06:00Z", 10)
	if err != nil || len(due) != 1 || due[0].ID != "sess-classify" {
		t.Fatalf("due=%+v err=%v", due, err)
	}
	if err := db.MarkSessionClassificationFailed("sess-classify", "classifier down", "2026-05-14T10:10:00Z", "2026-05-14T10:06:00Z", 1); err != nil {
		t.Fatal(err)
	}
	due, err = db.ListSessionsNeedingClassification("2026-05-14T10:09:00Z", 10)
	if err != nil || len(due) != 0 {
		t.Fatalf("due before classifier retry=%+v err=%v", due, err)
	}
	due, err = db.ListSessionsNeedingClassification("2026-05-14T10:11:00Z", 10)
	if err != nil || len(due) != 1 || due[0].ClassifierRetryCount != 1 {
		t.Fatalf("due after classifier retry=%+v err=%v", due, err)
	}
	if err := db.SetSessionClassifiedEnd("sess-classify", "2026-05-14T10:05:00Z", "2026-05-14T10:12:00Z"); err != nil {
		t.Fatal(err)
	}
	due, err = db.ListSessionsNeedingClassification("2026-05-14T10:13:00Z", 10)
	if err != nil || len(due) != 0 {
		t.Fatalf("due after classified=%+v err=%v", due, err)
	}
	row, _ := db.GetSession("sess-classify")
	if row.ClassifierRetryCount != 0 || row.ClassifierNextRetryAt != "" || row.ClassifierLastError != "" {
		t.Fatalf("classifier retry not cleared: %+v", row)
	}
	if err := db.UpsertMemoryFile(FileRow{Path: "project-test.md", Prefix: "project", Description: "Project", Tags: []string{"project"}, Status: "active", EntryCount: 1, Created: "2026-05-14T10:00:00Z", Updated: "2026-05-14T10:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	if err := db.InsertMemoryEntry(MemoryEntryRow{ID: "entry-1", Path: "project-test.md", Prefix: "project", Timestamp: "2026-05-14T10:00:00Z", Tags: "project", Content: "body"}); err != nil {
		t.Fatal(err)
	}
	if err := db.MarkMemoryEntrySuperseded("project-test.md", "entry-1"); err != nil {
		t.Fatal(err)
	}
	hits, err := db.SearchMemory("body", nil, "", "", 5, false)
	if err != nil || len(hits) != 0 {
		t.Fatalf("hits=%+v err=%v", hits, err)
	}
	if err := db.SetMemoryFileNeedsCompact("project-test.md", true); err != nil {
		t.Fatal(err)
	}
	files, err := db.ListMemoryFiles(false, false)
	if err != nil || len(files) != 1 || !files[0].NeedsCompact {
		t.Fatalf("files=%+v err=%v", files, err)
	}
}
