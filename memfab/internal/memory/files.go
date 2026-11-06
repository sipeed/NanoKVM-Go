package memory

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var entryHeadingRE = regexp.MustCompile(`^##\s*\[([^\]]+)\]\s*\{id:\s*([a-zA-Z0-9\-]+)\}(.*)$`)

var validPrefixes = []string{"user-", "project-", "tool-", "topic-", "person-", "org-", "event-"}

//go:embed schema.md
var embeddedSchemaText string

type ParsedFile struct {
	Path         string        `json:"path"`
	Description  string        `json:"description,omitempty"`
	Tags         []string      `json:"tags,omitempty"`
	Status       string        `json:"status,omitempty"`
	Created      string        `json:"created,omitempty"`
	Updated      string        `json:"updated,omitempty"`
	EntryCount   int           `json:"entry_count,omitempty"`
	NeedsCompact bool          `json:"needs_compact,omitempty"`
	Entries      []ParsedEntry `json:"entries"`
}

type ParsedEntry struct {
	ID           string   `json:"id"`
	Timestamp    string   `json:"timestamp"`
	Tags         []string `json:"tags,omitempty"`
	Body         string   `json:"body"`
	SupersededBy string   `json:"superseded_by,omitempty"`
}

func MemoryPath(root, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("memory path is required")
	}
	if filepath.IsAbs(name) || strings.Contains(name, "/") || strings.Contains(name, `\`) || strings.Contains(name, "..") {
		return "", fmt.Errorf("unsafe memory path: %s", name)
	}
	if filepath.Ext(name) == "" {
		name += ".md"
	}
	if filepath.Ext(name) != ".md" {
		return "", fmt.Errorf("memory path must end with .md: %s", name)
	}
	return filepath.Join(root, name), nil
}

func ValidatePrefix(path string) (string, error) {
	base := filepath.Base(path)
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(base, prefix) {
			return strings.TrimSuffix(prefix, "-"), nil
		}
	}
	return "", fmt.Errorf("invalid memory file prefix: %s", base)
}

func ListMemoryFiles(root string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(root, "*.md"))
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(matches))
	for _, path := range matches {
		if filepath.Base(path) == "index.md" {
			continue
		}
		out = append(out, path)
	}
	return out, nil
}

func ReadFile(path string) (ParsedFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ParsedFile{}, err
	}
	parsed := Parse(path, string(data))
	return parsed, nil
}

func Parse(path, text string) ParsedFile {
	parsed := ParsedFile{Path: filepath.Base(path), Status: "active"}
	body := text
	if strings.HasPrefix(body, "---\n") || strings.HasPrefix(body, "---\r\n") {
		end := strings.Index(body[4:], "\n---")
		if end >= 0 {
			front := body[4 : 4+end]
			applyFrontmatter(&parsed, front)
			after := 4 + end + len("\n---")
			if after < len(body) && body[after] == '\r' {
				after++
			}
			if after < len(body) && body[after] == '\n' {
				after++
			}
			body = body[after:]
		}
	}
	parsed.Entries = ParseEntries(body)
	if parsed.EntryCount == 0 {
		parsed.EntryCount = len(parsed.Entries)
	}
	return parsed
}

func ParseEntries(body string) []ParsedEntry {
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	var entries []ParsedEntry
	var current *ParsedEntry
	var currentBody []string
	flush := func() {
		if current == nil {
			return
		}
		current.Body = strings.Trim(strings.Join(currentBody, "\n"), "\n")
		entries = append(entries, *current)
	}
	for _, line := range lines {
		m := entryHeadingRE.FindStringSubmatch(line)
		if m != nil {
			flush()
			tags, supersededBy := parseHeadingTags(m[3])
			current = &ParsedEntry{Timestamp: strings.TrimSpace(m[1]), ID: strings.TrimSpace(m[2]), Tags: tags, SupersededBy: supersededBy}
			currentBody = nil
			continue
		}
		if current != nil {
			currentBody = append(currentBody, line)
		}
	}
	flush()
	return entries
}

func applyFrontmatter(parsed *ParsedFile, front string) {
	lines := strings.Split(strings.ReplaceAll(front, "\r\n", "\n"), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = trimScalar(value)
		switch key {
		case "description":
			parsed.Description = value
		case "tags":
			parsed.Tags = parseTagList(value)
		case "status":
			parsed.Status = value
		case "created":
			parsed.Created = value
		case "updated":
			parsed.Updated = value
		case "entry_count":
			if n, err := strconv.Atoi(value); err == nil {
				parsed.EntryCount = n
			}
		case "needs_compact":
			parsed.NeedsCompact = strings.EqualFold(value, "true") || value == "1"
		}
	}
}

func trimScalar(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)
	return value
}

func parseTagList(value string) []string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		value = strings.TrimSuffix(strings.TrimPrefix(value, "["), "]")
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(strings.Trim(part, `"'`))
		part = strings.TrimPrefix(part, "#")
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func parseHeadingTags(text string) ([]string, string) {
	fields := strings.Fields(text)
	tags := make([]string, 0, len(fields))
	supersededBy := ""
	for _, field := range fields {
		field = strings.TrimSpace(strings.TrimPrefix(field, "#"))
		if field == "" {
			continue
		}
		if strings.HasPrefix(field, "superseded-by:") {
			supersededBy = strings.TrimPrefix(field, "superseded-by:")
		}
		tags = append(tags, field)
	}
	return tags, supersededBy
}

func SchemaText() string {
	for _, candidate := range []string{
		"src/memfab/prompts/schema.md",
		filepath.Join("..", "..", "src", "memfab", "prompts", "schema.md"),
	} {
		if data, err := os.ReadFile(candidate); err == nil {
			return string(data)
		}
	}
	return embeddedSchemaText
}
