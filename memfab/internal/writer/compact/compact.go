package compact

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"memfab/internal/config"
	"memfab/internal/llm"
	"memfab/internal/memory"
	"memfab/internal/paths"
	"memfab/internal/store"
)

var callText = llm.CallText

type Result struct {
	Path              string
	Accepted          bool
	BeforeTokens      int
	AfterTokens       int
	BeforeUnique      int
	AfterUnique       int
	PreservationRatio float64
	Note              string
}

func RunPending(ctx context.Context, db *store.DB, cfg config.Config, now time.Time, limit int) ([]Result, error) {
	if !cfg.Compact.Enabled {
		return nil, nil
	}
	if limit <= 0 {
		limit = cfg.Compact.MaxFilesPerTick
	}
	if limit <= 0 {
		limit = 2
	}
	files, err := db.FilesNeedingCompact(limit)
	if err != nil {
		return nil, err
	}
	results := make([]Result, 0, len(files))
	for _, file := range files {
		result, err := File(ctx, db, cfg, file.Path, now)
		if err != nil {
			return results, err
		}
		results = append(results, result)
	}
	return results, nil
}

func File(ctx context.Context, db *store.DB, cfg config.Config, name string, now time.Time) (Result, error) {
	path, original, err := memory.ReadRaw(paths.MemoryDir(), name)
	result := Result{Path: filepath.Base(name), BeforeTokens: estimateTokens(original)}
	if err != nil {
		result.Note = err.Error()
		return result, nil
	}
	result.Path = filepath.Base(path)
	originalParsed := memory.Parse(path, original)
	if _, err := memory.ValidatePrefix(result.Path); err != nil {
		result.Note = err.Error()
		return result, nil
	}
	if cfg.Compact.MinEstimatedTokens > 0 && result.BeforeTokens < cfg.Compact.MinEstimatedTokens {
		result.Note = "below compact token threshold"
		return result, nil
	}
	model := cfg.Models.Compact
	if strings.TrimSpace(model.Model) == "" {
		result.Note = "compact llm is not configured"
		return result, nil
	}
	candidate, err := compactWithLLM(ctx, model, original)
	if err != nil {
		result.Note = "llm error: " + err.Error()
		return result, nil
	}
	candidate = memory.StripMarkdownFence(candidate)
	candidate, parsed, validation, err := validateAndPrepare(result.Path, original, candidate, originalParsed, cfg, now)
	result.AfterTokens = estimateTokens(candidate)
	result.BeforeUnique = validation.BeforeUnique
	result.AfterUnique = validation.AfterUnique
	result.PreservationRatio = validation.PreservationRatio
	if err != nil {
		result.Note = err.Error()
		return result, nil
	}
	if err := memory.WithFileLock(path, func() error {
		current, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if string(current) != original {
			result.Note = "file changed during compact — retry later"
			return nil
		}
		if err := memory.AtomicWriteText(path, candidate); err != nil {
			return err
		}
		if err := db.ReindexMemoryFile(paths.MemoryDir(), result.Path); err != nil {
			_ = memory.AtomicWriteText(path, original)
			_ = db.ReindexMemoryFile(paths.MemoryDir(), result.Path)
			return err
		}
		return nil
	}); err != nil {
		return result, err
	}
	if result.Note != "" {
		return result, nil
	}
	result.Accepted = true
	result.Note = fmt.Sprintf("accepted: %d entries", len(parsed.Entries))
	return result, nil
}

func compactWithLLM(ctx context.Context, model config.ModelConfig, original string) (string, error) {
	prompt := compactPrompt()
	user := "Compress this file. Output the full new Markdown including frontmatter.\n\n```markdown\n" + original + "\n```"
	return callText(ctx, model, []llm.Message{{Role: "system", Content: prompt}, {Role: "user", Content: user}})
}

func compactPrompt() string {
	for _, candidate := range []string{
		"src/memfab/prompts/compact.md",
		filepath.Join("..", "..", "src", "memfab", "prompts", "compact.md"),
	} {
		if data, err := os.ReadFile(candidate); err == nil {
			return string(data)
		}
	}
	return strings.TrimSpace(`You are the Compact module of memfab. Produce a compressed full Markdown memory file that preserves all unique facts. Preserve frontmatter, supersede chains, struck-through entries, dates, tags, and proper nouns. Merge only genuine duplicates. Output only the full new Markdown file starting with YAML frontmatter.`)
}

func estimateTokens(text string) int {
	return len(text) / 4
}
