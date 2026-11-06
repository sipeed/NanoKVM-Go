package memory

import (
	"strings"
	"testing"
	"time"
)

func TestEventDailyName(t *testing.T) {
	got := EventDailyName(time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC))
	if got != "event-2026-05-14.md" {
		t.Fatalf("got %q", got)
	}
}

func TestEnsureAndAppendEventDailyFile(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC)
	name := EventDailyName(now)
	parsed, err := EnsureEventDailyFile(root, name, now, now)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Path != name || parsed.EntryCount != 0 || parsed.Description != "Event memory for 2026-05-14" {
		t.Fatalf("parsed=%+v", parsed)
	}
	entry := ParsedEntry{ID: "20260514-1000-abcdef", Timestamp: now.Format(time.RFC3339), Tags: []string{"event", "session", "sid:sess_test"}, Body: "Reduced event body."}
	parsed, err = AppendEntry(root, name, entry, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.EntryCount != 1 || parsed.Updated != now.Add(time.Minute).Format(time.RFC3339) {
		t.Fatalf("parsed=%+v", parsed)
	}
	read, err := ReadFile(mustMemoryPath(t, root, name))
	if err != nil {
		t.Fatal(err)
	}
	if len(read.Entries) != 1 || read.Entries[0].ID != entry.ID || read.Entries[0].Body != entry.Body {
		t.Fatalf("read=%+v", read)
	}
}

func mustMemoryPath(t *testing.T, root, name string) string {
	t.Helper()
	path, err := MemoryPath(root, name)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRawAtomicHelpers(t *testing.T) {
	root := t.TempDir()
	path := mustMemoryPath(t, root, "project-raw.md")
	text := "---\ndescription: Raw\ntags: [raw]\nstatus: active\ncreated: 2026-05-14T00:00:00Z\nentry_count: 0\n---\n"
	if err := AtomicWriteText(path, text); err != nil {
		t.Fatal(err)
	}
	gotPath, gotText, err := ReadRaw(root, "project-raw.md")
	if err != nil || gotPath != path || gotText != text {
		t.Fatalf("path=%q text=%q err=%v", gotPath, gotText, err)
	}
	if got := StripMarkdownFence("```markdown\n" + text + "\n```"); got != strings.TrimSpace(text) {
		t.Fatalf("fence got=%q", got)
	}
	updated, err := SetFrontmatterBool(text, "needs_compact", false)
	if err != nil || !strings.Contains(updated, "needs_compact: false") {
		t.Fatalf("updated=%q err=%v", updated, err)
	}
	locked := false
	if err := WithFileLock(path, func() error { locked = true; return nil }); err != nil || !locked {
		t.Fatalf("locked=%v err=%v", locked, err)
	}
}

func TestDurableMemoryHelpers(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC)
	parsed, err := CreateDurableFile(root, "project-test.md", "Project test", []string{"project", "test"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Path != "project-test.md" || parsed.Description != "Project test" {
		t.Fatalf("parsed=%+v", parsed)
	}
	entry := ParsedEntry{ID: "entry-1", Timestamp: now.Format(time.RFC3339), Tags: []string{"test"}, Body: "Durable memory body."}
	parsed, err = AppendDurableEntry(root, "project-test.md", entry, now.Add(time.Minute))
	if err != nil || len(parsed.Entries) != 1 {
		t.Fatalf("parsed=%+v err=%v", parsed, err)
	}
	replacement := ParsedEntry{ID: "entry-2", Timestamp: now.Add(2 * time.Minute).Format(time.RFC3339), Tags: []string{"test"}, Body: "Replacement body."}
	parsed, err = SupersedeEntry(root, "project-test.md", "entry-1", replacement, now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Entries) != 2 || parsed.Entries[0].SupersededBy != "entry-2" || parsed.Entries[1].ID != "entry-2" {
		t.Fatalf("parsed=%+v", parsed)
	}
	parsed, err = FlagCompact(root, "project-test.md", now.Add(3*time.Minute))
	if err != nil || !parsed.NeedsCompact {
		t.Fatalf("parsed=%+v err=%v", parsed, err)
	}
	if _, err := CreateDurableFile(root, "event-2026-05-14.md", "bad", nil, now); err == nil {
		t.Fatal("expected event write rejection")
	}
	if _, err := AppendDurableEntry(root, "event-2026-05-14.md", entry, now); err == nil {
		t.Fatal("expected event append rejection")
	}
	if _, err := SupersedeEntry(root, "event-2026-05-14.md", "entry-1", replacement, now); err == nil {
		t.Fatal("expected event supersede rejection")
	}
	if _, err := FlagCompact(root, "event-2026-05-14.md", now); err == nil {
		t.Fatal("expected event compact rejection")
	}
	if _, err := CreateDurableFile(root, "bad-test.md", "bad", nil, now); err == nil {
		t.Fatal("expected invalid prefix rejection")
	}
}
