package compact

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"memfab/internal/config"
	"memfab/internal/llm"
	"memfab/internal/memory"
	"memfab/internal/paths"
	"memfab/internal/store"
)

func compactTestDB(t *testing.T) *store.DB {
	t.Helper()
	t.Setenv("MEMFAB_ROOT", t.TempDir())
	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func compactConfig() config.Config {
	return config.Config{
		Compact: config.CompactConfig{Enabled: true, MaxFilesPerTick: 2, PreservationRatio: 0.80},
		Models:  config.ModelsConfig{Compact: config.ModelConfig{Model: "gpt-test"}},
	}
}

func writeCompactMemory(t *testing.T, name, text string) {
	t.Helper()
	path, err := memory.MemoryPath(paths.MemoryDir(), name)
	if err != nil {
		t.Fatal(err)
	}
	if err := memory.AtomicWriteText(path, text); err != nil {
		t.Fatal(err)
	}
}

func compactOriginal() string {
	return `---
description: Compact memory
tags: [project, compact]
status: active
created: 2026-05-14T00:00:00Z
updated: 2026-05-14T00:10:00Z
entry_count: 3
needs_compact: true
---

## [2026-05-14T00:01:00Z] {id: keep-1} #project #compact
memfab compact preserves durable architecture facts about classifier tool loops and remote OpenAI-compatible models.

## [2026-05-14T00:02:00Z] {id: keep-2} #project #compact
memfab compact preserves durable architecture facts about classifier tool loops and remote OpenAI-compatible models.

## [2026-05-14T00:03:00Z] {id: old-1} #project #superseded-by:new-1
~~Old remote model note superseded by new-1.~~
`
}

func compactAccepted() string {
	return `---
description: Compact memory
tags: [project, compact]
status: active
created: 2026-05-14T00:00:00Z
updated: 2026-05-14T00:20:00Z
entry_count: 2
needs_compact: true
---

## [2026-05-14T00:01:00Z] {id: keep-1} #project #compact
memfab compact preserves durable architecture facts about classifier tool loops and remote OpenAI-compatible models.

## [2026-05-14T00:03:00Z] {id: old-1} #project #superseded-by:new-1
~~Old remote model note superseded by new-1.~~
`
}

func TestFileAcceptsValidCompactionAndReindexes(t *testing.T) {
	db := compactTestDB(t)
	writeCompactMemory(t, "project-compact.md", compactOriginal())
	if err := db.ReindexMemoryFile(paths.MemoryDir(), "project-compact.md"); err != nil {
		t.Fatal(err)
	}
	old := callText
	callText = func(ctx context.Context, cfg config.ModelConfig, messages []llm.Message) (string, error) {
		if cfg.ResolvedAPIKey() != "" {
			t.Fatalf("compact should not require an API key, got %q", cfg.ResolvedAPIKey())
		}
		if !strings.Contains(messages[1].Content, "Compress this file") {
			t.Fatalf("prompt=%s", messages[1].Content)
		}
		return "```markdown\n" + compactAccepted() + "\n```", nil
	}
	t.Cleanup(func() { callText = old })
	result, err := File(context.Background(), db, compactConfig(), "project-compact.md", time.Date(2026, 5, 14, 0, 30, 0, 0, time.UTC))
	if err != nil || !result.Accepted || result.PreservationRatio <= 0 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	_, text, err := memory.ReadRaw(paths.MemoryDir(), "project-compact.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "needs_compact: false") || !strings.Contains(text, "entry_count: 2") {
		t.Fatalf("text=%s", text)
	}
	hits, err := db.SearchMemory("durable architecture", nil, "", "", 5, false)
	if err != nil || len(hits) != 1 || hits[0].ID != "keep-1" {
		t.Fatalf("hits=%+v err=%v", hits, err)
	}
}

func TestFileRejectsMissingFrontmatterAndKeepsOriginal(t *testing.T) {
	db := compactTestDB(t)
	writeCompactMemory(t, "project-compact.md", compactOriginal())
	if err := db.ReindexMemoryFile(paths.MemoryDir(), "project-compact.md"); err != nil {
		t.Fatal(err)
	}
	old := callText
	callText = func(ctx context.Context, cfg config.ModelConfig, messages []llm.Message) (string, error) {
		return "no frontmatter", nil
	}
	t.Cleanup(func() { callText = old })
	result, err := File(context.Background(), db, compactConfig(), "project-compact.md", time.Now())
	if err != nil || result.Accepted || !strings.Contains(result.Note, "frontmatter") {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	_, text, _ := memory.ReadRaw(paths.MemoryDir(), "project-compact.md")
	if text != compactOriginal() {
		t.Fatalf("original changed")
	}
}

func TestFileRejectsLowPreservation(t *testing.T) {
	db := compactTestDB(t)
	writeCompactMemory(t, "project-compact.md", compactOriginal())
	old := callText
	callText = func(ctx context.Context, cfg config.ModelConfig, messages []llm.Message) (string, error) {
		return `---
description: Compact memory
tags: [project, compact]
status: active
created: 2026-05-14T00:00:00Z
updated: 2026-05-14T00:20:00Z
entry_count: 2
---

## [2026-05-14T00:01:00Z] {id: keep-1} #project
Tiny note.

## [2026-05-14T00:03:00Z] {id: old-1} #project #superseded-by:new-1
~~Old remote model note superseded by new-1.~~
`, nil
	}
	t.Cleanup(func() { callText = old })
	cfg := compactConfig()
	cfg.Compact.PreservationRatio = 0.95
	result, err := File(context.Background(), db, cfg, "project-compact.md", time.Now())
	if err != nil || result.Accepted || !strings.Contains(result.Note, "unique-token") {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestFileRejectsStaleReadAndPreservesConcurrentChange(t *testing.T) {
	db := compactTestDB(t)
	writeCompactMemory(t, "project-compact.md", compactOriginal())
	path, _ := memory.MemoryPath(paths.MemoryDir(), "project-compact.md")
	old := callText
	callText = func(ctx context.Context, cfg config.ModelConfig, messages []llm.Message) (string, error) {
		if err := os.WriteFile(path, []byte(compactOriginal()+"\nconcurrent change\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return compactAccepted(), nil
	}
	t.Cleanup(func() { callText = old })
	result, err := File(context.Background(), db, compactConfig(), "project-compact.md", time.Now())
	if err != nil || result.Accepted || !strings.Contains(result.Note, "file changed") {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "concurrent change") {
		t.Fatalf("concurrent change lost")
	}
}

func TestFileRejectsLLMErrorAndUnconfiguredModel(t *testing.T) {
	db := compactTestDB(t)
	writeCompactMemory(t, "project-compact.md", compactOriginal())
	old := callText
	callText = func(ctx context.Context, cfg config.ModelConfig, messages []llm.Message) (string, error) {
		return "", errors.New("down")
	}
	t.Cleanup(func() { callText = old })
	result, err := File(context.Background(), db, compactConfig(), "project-compact.md", time.Now())
	if err != nil || result.Accepted || !strings.Contains(result.Note, "llm error") {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	cfg := compactConfig()
	cfg.Models.Compact.Model = ""
	result, err = File(context.Background(), db, cfg, "project-compact.md", time.Now())
	if err != nil || result.Accepted || !strings.Contains(result.Note, "not configured") {
		t.Fatalf("unconfigured result=%+v err=%v", result, err)
	}
}

func TestRunPendingUsesFilesNeedingCompact(t *testing.T) {
	db := compactTestDB(t)
	writeCompactMemory(t, "project-compact.md", compactOriginal())
	if err := db.ReindexMemoryFile(paths.MemoryDir(), "project-compact.md"); err != nil {
		t.Fatal(err)
	}
	old := callText
	callText = func(ctx context.Context, cfg config.ModelConfig, messages []llm.Message) (string, error) {
		return compactAccepted(), nil
	}
	t.Cleanup(func() { callText = old })
	results, err := RunPending(context.Background(), db, compactConfig(), time.Now(), 1)
	if err != nil || len(results) != 1 || !results[0].Accepted {
		t.Fatalf("results=%+v err=%v", results, err)
	}
}

func TestFileRejectsDroppedSupersedeHistory(t *testing.T) {
	db := compactTestDB(t)
	writeCompactMemory(t, "project-compact.md", compactOriginal())
	old := callText
	callText = func(ctx context.Context, cfg config.ModelConfig, messages []llm.Message) (string, error) {
		out := strings.ReplaceAll(compactAccepted(), "#superseded-by:new-1", "")
		return out, nil
	}
	t.Cleanup(func() { callText = old })
	result, err := File(context.Background(), db, compactConfig(), "project-compact.md", time.Now())
	if err != nil || result.Accepted || !strings.Contains(result.Note, "supersede") {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestPromptFallbackDoesNotRequirePythonFile(t *testing.T) {
	if compactPrompt() == "" {
		t.Fatal("empty prompt")
	}
	if filepath.Base(paths.Root()) == "" {
		t.Fatal("paths root empty")
	}
}
