package tools

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"memfab/internal/llm"
	"memfab/internal/memory"
	"memfab/internal/store"
)

type CommitState struct {
	Committed      bool
	Summary        string
	WrittenIDs     []string
	CreatedPaths   []string
	FlaggedCompact []string
}

type Dispatcher struct {
	DB              *store.DB
	MemoryDir       string
	SoftLimitTokens int
	Now             time.Time
	State           *CommitState
}

func ToolDefinitions() []llm.ToolDefinition {
	return []llm.ToolDefinition{
		{
			Name:        "read_memory",
			Description: "Read a memory file with frontmatter and trailing entries.",
			Parameters: objectSchema(map[string]any{
				"path":   map[string]any{"type": "string", "description": "Memory file path, for example project-memfab.md"},
				"tail_n": map[string]any{"type": "integer", "default": 10},
			}, []string{"path"}),
		},
		{
			Name:        "search_memory",
			Description: "BM25 full-text search across memory entries. Use it to deduplicate before writing.",
			Parameters: objectSchema(map[string]any{
				"query":              map[string]any{"type": "string"},
				"top_k":              map[string]any{"type": "integer", "default": 5},
				"include_superseded": map[string]any{"type": "boolean", "default": false},
			}, []string{"query"}),
		},
		{
			Name:        "append",
			Description: "Append a new durable memory entry to an existing non-event memory file.",
			Parameters: objectSchema(map[string]any{
				"path":    map[string]any{"type": "string"},
				"content": map[string]any{"type": "string", "description": "1-3 sentence self-contained durable fact"},
				"tags":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "maxItems": 3},
			}, []string{"path", "content", "tags"}),
		},
		{
			Name:        "create",
			Description: "Create a new non-event memory file. Prefix must be user-, project-, tool-, topic-, person-, or org-.",
			Parameters: objectSchema(map[string]any{
				"path":        map[string]any{"type": "string"},
				"description": map[string]any{"type": "string"},
				"tags":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			}, []string{"path", "description", "tags"}),
		},
		{
			Name:        "supersede",
			Description: "Mark an old entry as superseded and append a replacement durable memory entry.",
			Parameters: objectSchema(map[string]any{
				"path":         map[string]any{"type": "string"},
				"old_entry_id": map[string]any{"type": "string"},
				"new_content":  map[string]any{"type": "string"},
				"reason":       map[string]any{"type": "string"},
				"tags":         map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			}, []string{"path", "old_entry_id", "new_content", "reason"}),
		},
		{
			Name:        "flag_compact",
			Description: "Flag a non-event memory file for a later compaction pass.",
			Parameters: objectSchema(map[string]any{
				"path":   map[string]any{"type": "string"},
				"reason": map[string]any{"type": "string"},
			}, []string{"path", "reason"}),
		},
		{
			Name:        "commit",
			Description: "Finish this classifier round. Call exactly once at the end.",
			Parameters: objectSchema(map[string]any{
				"summary": map[string]any{"type": "string"},
			}, []string{"summary"}),
		},
	}
}

func objectSchema(properties map[string]any, required []string) map[string]any {
	return map[string]any{"type": "object", "properties": properties, "required": required}
}

func (d Dispatcher) Dispatch(ctx context.Context, name string, args json.RawMessage) map[string]any {
	if d.State == nil {
		d.State = &CommitState{}
	}
	switch name {
	case "read_memory":
		var in struct {
			Path  string `json:"path"`
			TailN int    `json:"tail_n"`
		}
		if err := decodeArgs(args, &in); err != nil {
			return errorResult(err)
		}
		return d.readMemory(in.Path, in.TailN)
	case "search_memory":
		var in struct {
			Query             string `json:"query"`
			TopK              int    `json:"top_k"`
			IncludeSuperseded bool   `json:"include_superseded"`
		}
		if err := decodeArgs(args, &in); err != nil {
			return errorResult(err)
		}
		return d.searchMemory(ctx, in.Query, in.TopK, in.IncludeSuperseded)
	case "create":
		var in struct {
			Path        string   `json:"path"`
			Description string   `json:"description"`
			Tags        []string `json:"tags"`
		}
		if err := decodeArgs(args, &in); err != nil {
			return errorResult(err)
		}
		return d.create(in.Path, in.Description, in.Tags)
	case "append":
		var in struct {
			Path    string   `json:"path"`
			Content string   `json:"content"`
			Tags    []string `json:"tags"`
		}
		if err := decodeArgs(args, &in); err != nil {
			return errorResult(err)
		}
		return d.append(in.Path, in.Content, in.Tags)
	case "supersede":
		var in struct {
			Path       string   `json:"path"`
			OldEntryID string   `json:"old_entry_id"`
			NewContent string   `json:"new_content"`
			Reason     string   `json:"reason"`
			Tags       []string `json:"tags"`
		}
		if err := decodeArgs(args, &in); err != nil {
			return errorResult(err)
		}
		return d.supersede(in.Path, in.OldEntryID, in.NewContent, in.Reason, in.Tags)
	case "flag_compact":
		var in struct {
			Path   string `json:"path"`
			Reason string `json:"reason"`
		}
		if err := decodeArgs(args, &in); err != nil {
			return errorResult(err)
		}
		return d.flagCompact(in.Path, in.Reason)
	case "commit":
		var in struct {
			Summary string `json:"summary"`
		}
		if err := decodeArgs(args, &in); err != nil {
			return errorResult(err)
		}
		d.State.Committed = true
		d.State.Summary = strings.TrimSpace(in.Summary)
		return map[string]any{"ok": true}
	default:
		return map[string]any{"error": "unknown tool: " + name}
	}
}

func (d Dispatcher) readMemory(path string, tailN int) map[string]any {
	p, err := memory.MemoryPath(d.MemoryDir, path)
	if err != nil {
		return errorResult(err)
	}
	parsed, err := memory.ReadFile(p)
	if err != nil {
		return errorResult(err)
	}
	entries := parsed.Entries
	if tailN <= 0 {
		tailN = len(entries)
	}
	if tailN < len(entries) {
		entries = entries[len(entries)-tailN:]
	}
	return map[string]any{
		"path":          parsed.Path,
		"description":   parsed.Description,
		"tags":          parsed.Tags,
		"status":        parsed.Status,
		"entry_count":   parsed.EntryCount,
		"updated":       parsed.Updated,
		"needs_compact": parsed.NeedsCompact,
		"entries":       entries,
	}
}

func (d Dispatcher) searchMemory(ctx context.Context, query string, topK int, includeSuperseded bool) map[string]any {
	if err := ctx.Err(); err != nil {
		return errorResult(err)
	}
	hits, err := d.DB.SearchMemory(query, nil, "", "", topK, includeSuperseded)
	if err != nil {
		return errorResult(err)
	}
	return map[string]any{"query": query, "results": hits}
}

func (d Dispatcher) create(path, description string, tags []string) map[string]any {
	if err := rejectEvent(path); err != nil {
		return errorResult(err)
	}
	memoryPath, err := memory.MemoryPath(d.MemoryDir, path)
	if err != nil {
		return errorResult(err)
	}
	if _, err := os.Stat(memoryPath); err == nil {
		return errorResult(fmt.Errorf("file already exists: %s", filepath.Base(memoryPath)))
	} else if !os.IsNotExist(err) {
		return errorResult(err)
	}
	parsed, err := memory.CreateDurableFile(d.MemoryDir, path, strings.TrimSpace(description), cleanTags(tags), d.now())
	if err != nil {
		return errorResult(err)
	}
	if err := d.upsertFile(parsed); err != nil {
		return errorResult(err)
	}
	d.State.CreatedPaths = append(d.State.CreatedPaths, parsed.Path)
	return map[string]any{"ok": true, "path": parsed.Path}
}

func (d Dispatcher) append(path, content string, tags []string) map[string]any {
	if err := rejectEvent(path); err != nil {
		return errorResult(err)
	}
	now := d.now()
	entry := memory.ParsedEntry{ID: makeEntryID(now, path, content), Timestamp: now.Format(time.RFC3339), Tags: cleanTags(tags), Body: strings.TrimSpace(content)}
	parsed, err := memory.AppendDurableEntry(d.MemoryDir, path, entry, now)
	if err != nil {
		return errorResult(err)
	}
	if err := d.upsertFile(parsed); err != nil {
		return errorResult(err)
	}
	prefix, _ := memory.ValidatePrefix(parsed.Path)
	if err := d.DB.InsertMemoryEntry(store.MemoryEntryRow{ID: entry.ID, Path: parsed.Path, Prefix: prefix, Timestamp: entry.Timestamp, Tags: strings.Join(entry.Tags, " "), Content: entry.Body}); err != nil {
		return errorResult(err)
	}
	d.State.WrittenIDs = append(d.State.WrittenIDs, entry.ID)
	return map[string]any{"ok": true, "id": entry.ID, "path": parsed.Path}
}

func (d Dispatcher) supersede(path, oldEntryID, newContent, reason string, tags []string) map[string]any {
	if err := rejectEvent(path); err != nil {
		return errorResult(err)
	}
	now := d.now()
	body := strings.TrimSpace(newContent)
	if strings.TrimSpace(reason) != "" {
		body = body + "\n\nSupersedes " + oldEntryID + ": " + strings.TrimSpace(reason)
	}
	replacement := memory.ParsedEntry{ID: makeEntryID(now, path, body), Timestamp: now.Format(time.RFC3339), Tags: cleanTags(tags), Body: body}
	parsed, err := memory.SupersedeEntry(d.MemoryDir, path, oldEntryID, replacement, now)
	if err != nil {
		return errorResult(err)
	}
	if err := d.upsertFile(parsed); err != nil {
		return errorResult(err)
	}
	prefix, _ := memory.ValidatePrefix(parsed.Path)
	if err := d.DB.MarkMemoryEntrySuperseded(parsed.Path, oldEntryID); err != nil {
		return errorResult(err)
	}
	if err := d.DB.InsertMemoryEntry(store.MemoryEntryRow{ID: replacement.ID, Path: parsed.Path, Prefix: prefix, Timestamp: replacement.Timestamp, Tags: strings.Join(replacement.Tags, " "), Content: replacement.Body}); err != nil {
		return errorResult(err)
	}
	d.State.WrittenIDs = append(d.State.WrittenIDs, replacement.ID)
	return map[string]any{"ok": true, "new_id": replacement.ID, "path": parsed.Path}
}

func (d Dispatcher) flagCompact(path, reason string) map[string]any {
	if err := rejectEvent(path); err != nil {
		return errorResult(err)
	}
	parsed, err := memory.FlagCompact(d.MemoryDir, path, d.now())
	if err != nil {
		return errorResult(err)
	}
	if err := d.upsertFile(parsed); err != nil {
		return errorResult(err)
	}
	if err := d.DB.SetMemoryFileNeedsCompact(parsed.Path, true); err != nil {
		return errorResult(err)
	}
	d.State.FlaggedCompact = append(d.State.FlaggedCompact, parsed.Path)
	return map[string]any{"ok": true, "path": parsed.Path, "reason": strings.TrimSpace(reason)}
}

func (d Dispatcher) upsertFile(parsed memory.ParsedFile) error {
	prefix, err := memory.ValidatePrefix(parsed.Path)
	if err != nil {
		return err
	}
	return d.DB.UpsertMemoryFile(store.FileRow{Path: parsed.Path, Prefix: prefix, Description: parsed.Description, Tags: parsed.Tags, Status: parsed.Status, EntryCount: len(parsed.Entries), Created: parsed.Created, Updated: parsed.Updated, NeedsCompact: parsed.NeedsCompact})
}

func (d Dispatcher) now() time.Time {
	if d.Now.IsZero() {
		return time.Now()
	}
	return d.Now
}

func decodeArgs(data json.RawMessage, dest any) error {
	data = []byte(strings.TrimSpace(string(data)))
	if len(data) == 0 {
		data = []byte("{}")
	}
	if len(data) > 0 && data[0] == '"' {
		var text string
		if err := json.Unmarshal(data, &text); err != nil {
			return err
		}
		data = []byte(strings.TrimSpace(text))
		if len(data) == 0 {
			data = []byte("{}")
		}
	}
	return json.Unmarshal(data, dest)
}

func errorResult(err error) map[string]any {
	return map[string]any{"error": err.Error()}
}

func rejectEvent(path string) error {
	base := filepath.Base(strings.TrimSpace(path))
	if strings.HasPrefix(base, "event-") {
		return fmt.Errorf("forbidden: classifier cannot write to %s", base)
	}
	return nil
}

func cleanTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	seen := map[string]bool{}
	for _, tag := range tags {
		tag = strings.TrimSpace(strings.TrimPrefix(tag, "#"))
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		out = append(out, tag)
		if len(out) == 3 {
			break
		}
	}
	return out
}

func makeEntryID(now time.Time, path, content string) string {
	sum := sha1.Sum([]byte(now.Format(time.RFC3339Nano) + path + content))
	return now.Format("20060102-150405-") + hex.EncodeToString(sum[:3])
}
