package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"memfab/internal/memory"
	"memfab/internal/store"
)

func toolsTestDB(t *testing.T) (*store.DB, string) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("MEMFAB_ROOT", root)
	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, filepath.Join(root, "memory")
}

func TestDispatcherCreateAppendSearchReadAndCompact(t *testing.T) {
	db, memoryDir := toolsTestDB(t)
	state := &CommitState{}
	d := Dispatcher{DB: db, MemoryDir: memoryDir, Now: time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC), State: state}

	created := d.Dispatch(context.Background(), "create", []byte(`{"path":"project-memfab.md","description":"memfab project","tags":["project","ocr"]}`))
	if created["error"] != nil || len(state.CreatedPaths) != 1 {
		t.Fatalf("created=%v state=%+v", created, state)
	}
	appended := d.Dispatch(context.Background(), "append", []byte(`{"path":"project-memfab.md","content":"memfab stage 8 uses an LLM classifier tool loop.","tags":["classifier","stage8"]}`))
	if appended["error"] != nil || len(state.WrittenIDs) != 1 {
		t.Fatalf("appended=%v state=%+v", appended, state)
	}
	hits := d.Dispatch(context.Background(), "search_memory", []byte(`{"query":"classifier tool loop","top_k":5}`))
	results, ok := hits["results"].([]store.EntryHit)
	if hits["error"] != nil || !ok || len(results) != 1 {
		t.Fatalf("hits=%v", hits)
	}
	read := d.Dispatch(context.Background(), "read_memory", []byte(`{"path":"project-memfab.md","tail_n":1}`))
	entries, ok := read["entries"].([]memory.ParsedEntry)
	if read["error"] != nil || !ok || len(entries) != 1 || !strings.Contains(entries[0].Body, "stage 8") {
		t.Fatalf("read=%v", read)
	}
	flagged := d.Dispatch(context.Background(), "flag_compact", []byte(`{"path":"project-memfab.md","reason":"test"}`))
	if flagged["error"] != nil || len(state.FlaggedCompact) != 1 {
		t.Fatalf("flagged=%v state=%+v", flagged, state)
	}
	files, err := db.ListMemoryFiles(false, false)
	if err != nil || len(files) != 1 || !files[0].NeedsCompact {
		t.Fatalf("files=%+v err=%v", files, err)
	}
}

func TestDispatcherSupersedeAndRejectEventWrites(t *testing.T) {
	db, memoryDir := toolsTestDB(t)
	state := &CommitState{}
	now := time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC)
	d := Dispatcher{DB: db, MemoryDir: memoryDir, Now: now, State: state}
	_ = d.Dispatch(context.Background(), "create", []byte(`{"path":"project-test.md","description":"Project","tags":["project"]}`))
	first := d.Dispatch(context.Background(), "append", []byte(`{"path":"project-test.md","content":"Old classifier fact.","tags":["project"]}`))
	id, _ := first["id"].(string)
	if id == "" {
		t.Fatalf("first=%v", first)
	}
	d.Now = now.Add(time.Minute)
	superseded := d.Dispatch(context.Background(), "supersede", []byte(`{"path":"project-test.md","old_entry_id":"`+id+`","new_content":"New classifier fact.","reason":"updated","tags":["project"]}`))
	if superseded["error"] != nil || len(state.WrittenIDs) != 2 {
		t.Fatalf("superseded=%v state=%+v", superseded, state)
	}
	hits, err := db.SearchMemory("Old", nil, "", "", 5, false)
	if err != nil || len(hits) != 0 {
		t.Fatalf("old hits=%+v err=%v", hits, err)
	}
	bad := d.Dispatch(context.Background(), "append", []byte(`{"path":"event-2026-05-14.md","content":"bad","tags":["event"]}`))
	if bad["error"] == nil {
		t.Fatalf("expected event rejection: %v", bad)
	}
	if _, err := os.Stat(filepath.Join(memoryDir, "event-2026-05-14.md")); !os.IsNotExist(err) {
		t.Fatalf("event file was created err=%v", err)
	}
}
