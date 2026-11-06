package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMemoryPath(t *testing.T) {
	root := t.TempDir()
	path, err := MemoryPath(root, "project-test")
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(root, "project-test.md") {
		t.Fatalf("path=%q", path)
	}
	for _, bad := range []string{"../x", "a/b", `a\b`, "/tmp/x.md", "project-test.txt"} {
		if _, err := MemoryPath(root, bad); err == nil {
			t.Fatalf("expected unsafe path error for %q", bad)
		}
	}
}

func TestValidatePrefix(t *testing.T) {
	prefix, err := ValidatePrefix("project-test.md")
	if err != nil || prefix != "project" {
		t.Fatalf("prefix=%q err=%v", prefix, err)
	}
	if _, err := ValidatePrefix("bad-test.md"); err == nil {
		t.Fatal("expected invalid prefix")
	}
}

func TestParseFrontmatterAndEntries(t *testing.T) {
	parsed := Parse("project-test.md", `---
description: Test project
tags: [alpha, beta]
status: dormant
created: 2026-05-14T00:00:00Z
updated: 2026-05-14T00:05:00Z
entry_count: 2
needs_compact: true
---

# Title

## [2026-05-14T00:01:00Z] {id: first-entry} #alpha #superseded-by:second-entry
First body.

## [2026-05-14T00:02:00Z] {id: second-entry} #beta
Second body.
`)
	if parsed.Description != "Test project" || parsed.Status != "dormant" || !parsed.NeedsCompact || parsed.EntryCount != 2 {
		t.Fatalf("frontmatter=%+v", parsed)
	}
	if len(parsed.Tags) != 2 || parsed.Tags[0] != "alpha" || parsed.Tags[1] != "beta" {
		t.Fatalf("tags=%v", parsed.Tags)
	}
	if len(parsed.Entries) != 2 {
		t.Fatalf("entries=%+v", parsed.Entries)
	}
	if parsed.Entries[0].ID != "first-entry" || parsed.Entries[0].SupersededBy != "second-entry" || parsed.Entries[0].Body != "First body." {
		t.Fatalf("first=%+v", parsed.Entries[0])
	}
	if parsed.Entries[1].Tags[0] != "beta" || parsed.Entries[1].Body != "Second body." {
		t.Fatalf("second=%+v", parsed.Entries[1])
	}
}

func TestListMemoryFilesSkipsIndex(t *testing.T) {
	root := t.TempDir()
	write := func(name string) {
		t.Helper()
		if err := osWrite(filepath.Join(root, name), []byte("")); err != nil {
			t.Fatal(err)
		}
	}
	write("project-test.md")
	write("index.md")
	files, err := ListMemoryFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || filepath.Base(files[0]) != "project-test.md" {
		t.Fatalf("files=%v", files)
	}
}

func TestSchemaTextFallsBackToEmbeddedFullSchema(t *testing.T) {
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	temp := t.TempDir()
	if err := os.Chdir(temp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatal(err)
		}
	})

	schema := SchemaText()
	for _, want := range []string{
		"# Memory Organization Spec",
		"Prefer silence over noise",
		"When a preference entry IS justified",
		"Cold start",
	} {
		if !strings.Contains(schema, want) {
			t.Fatalf("schema missing %q: %.120q", want, schema)
		}
	}
	if strings.Contains(schema, "Valid prefixes: user-, project-") {
		t.Fatalf("schema used short fallback: %.120q", schema)
	}
}

func osWrite(path string, data []byte) error { return os.WriteFile(path, data, 0o644) }
