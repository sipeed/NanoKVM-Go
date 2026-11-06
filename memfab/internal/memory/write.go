package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"memfab/internal/paths"
)

var pathLocks sync.Map

func EventDailyName(t time.Time) string {
	return "event-" + t.Format("2006-01-02") + ".md"
}

func EnsureEventDailyFile(memoryDir, name string, day time.Time, now time.Time) (ParsedFile, error) {
	path, err := MemoryPath(memoryDir, name)
	if err != nil {
		return ParsedFile{}, err
	}
	if parsed, err := ReadFile(path); err == nil {
		return parsed, nil
	} else if !os.IsNotExist(err) {
		return ParsedFile{}, err
	}
	parsed := ParsedFile{
		Path:        filepath.Base(path),
		Description: "Event memory for " + day.Format("2006-01-02"),
		Tags:        []string{"event", "session", "daily"},
		Status:      "active",
		Created:     now.Format(time.RFC3339),
		Updated:     now.Format(time.RFC3339),
		EntryCount:  0,
		Entries:     []ParsedEntry{},
	}
	return parsed, writeParsedFile(memoryDir, parsed)
}

func AppendEntry(memoryDir, name string, entry ParsedEntry, now time.Time) (ParsedFile, error) {
	path, err := MemoryPath(memoryDir, name)
	if err != nil {
		return ParsedFile{}, err
	}
	mu := lockForPath(path)
	mu.Lock()
	defer mu.Unlock()
	parsed, err := ReadFile(path)
	if err != nil {
		return ParsedFile{}, err
	}
	parsed.Entries = append(parsed.Entries, entry)
	parsed.EntryCount = len(parsed.Entries)
	parsed.Updated = now.Format(time.RFC3339)
	if parsed.Status == "" {
		parsed.Status = "active"
	}
	if err := writeParsedFile(memoryDir, parsed); err != nil {
		return ParsedFile{}, err
	}
	return parsed, nil
}

func writeParsedFile(memoryDir string, parsed ParsedFile) error {
	path, err := MemoryPath(memoryDir, parsed.Path)
	if err != nil {
		return err
	}
	return AtomicWriteText(path, renderParsedFile(parsed))
}

func AtomicWriteText(path, text string) error {
	if err := os.MkdirAll(filepath.Dir(path), paths.DirectoryMode); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".memory-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.WriteString(text); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return err
	}
	if dir, err := os.Open(filepath.Dir(path)); err == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}
	return nil
}

func WithFileLock(path string, fn func() error) error {
	mu := lockForPath(path)
	mu.Lock()
	defer mu.Unlock()
	return fn()
}

func renderParsedFile(parsed ParsedFile) string {
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "description: %s\n", parsed.Description)
	fmt.Fprintf(&b, "tags: [%s]\n", strings.Join(parsed.Tags, ", "))
	fmt.Fprintf(&b, "status: %s\n", parsed.Status)
	if parsed.Created != "" {
		fmt.Fprintf(&b, "created: %s\n", parsed.Created)
	}
	if parsed.Updated != "" {
		fmt.Fprintf(&b, "updated: %s\n", parsed.Updated)
	}
	fmt.Fprintf(&b, "entry_count: %d\n", len(parsed.Entries))
	if parsed.NeedsCompact {
		b.WriteString("needs_compact: true\n")
	}
	b.WriteString("---\n\n")
	for _, entry := range parsed.Entries {
		fmt.Fprintf(&b, "## [%s] {id: %s}", entry.Timestamp, entry.ID)
		for _, tag := range entry.Tags {
			if strings.TrimSpace(tag) != "" {
				fmt.Fprintf(&b, " #%s", strings.TrimPrefix(strings.TrimSpace(tag), "#"))
			}
		}
		b.WriteString("\n")
		b.WriteString(strings.Trim(entry.Body, "\n"))
		b.WriteString("\n\n")
	}
	return b.String()
}

func lockForPath(path string) *sync.Mutex {
	value, _ := pathLocks.LoadOrStore(path, &sync.Mutex{})
	return value.(*sync.Mutex)
}

func CreateDurableFile(memoryDir, name, description string, tags []string, now time.Time) (ParsedFile, error) {
	path, err := durableMemoryPath(memoryDir, name)
	if err != nil {
		return ParsedFile{}, err
	}
	if parsed, err := ReadFile(path); err == nil {
		return parsed, nil
	} else if !os.IsNotExist(err) {
		return ParsedFile{}, err
	}
	parsed := ParsedFile{Path: filepath.Base(path), Description: description, Tags: tags, Status: "active", Created: now.Format(time.RFC3339), Updated: now.Format(time.RFC3339), Entries: []ParsedEntry{}}
	return parsed, writeParsedFile(memoryDir, parsed)
}

func AppendDurableEntry(memoryDir, name string, entry ParsedEntry, now time.Time) (ParsedFile, error) {
	if _, err := durableMemoryPath(memoryDir, name); err != nil {
		return ParsedFile{}, err
	}
	return AppendEntry(memoryDir, name, entry, now)
}

func SupersedeEntry(memoryDir, name, oldEntryID string, replacement ParsedEntry, now time.Time) (ParsedFile, error) {
	path, err := durableMemoryPath(memoryDir, name)
	if err != nil {
		return ParsedFile{}, err
	}
	mu := lockForPath(path)
	mu.Lock()
	defer mu.Unlock()
	parsed, err := ReadFile(path)
	if err != nil {
		return ParsedFile{}, err
	}
	found := false
	for i := range parsed.Entries {
		if parsed.Entries[i].ID == oldEntryID {
			found = true
			parsed.Entries[i].SupersededBy = replacement.ID
			parsed.Entries[i].Tags = appendUnique(parsed.Entries[i].Tags, "superseded-by:"+replacement.ID)
			body := strings.TrimSpace(parsed.Entries[i].Body)
			if !strings.HasPrefix(body, "~~") {
				parsed.Entries[i].Body = "~~" + body + "~~"
			}
			break
		}
	}
	if !found {
		return ParsedFile{}, fmt.Errorf("entry not found: %s", oldEntryID)
	}
	parsed.Entries = append(parsed.Entries, replacement)
	parsed.EntryCount = len(parsed.Entries)
	parsed.Updated = now.Format(time.RFC3339)
	if err := writeParsedFile(memoryDir, parsed); err != nil {
		return ParsedFile{}, err
	}
	return parsed, nil
}

func FlagCompact(memoryDir, name string, now time.Time) (ParsedFile, error) {
	path, err := durableMemoryPath(memoryDir, name)
	if err != nil {
		return ParsedFile{}, err
	}
	mu := lockForPath(path)
	mu.Lock()
	defer mu.Unlock()
	parsed, err := ReadFile(path)
	if err != nil {
		return ParsedFile{}, err
	}
	parsed.NeedsCompact = true
	parsed.Updated = now.Format(time.RFC3339)
	if err := writeParsedFile(memoryDir, parsed); err != nil {
		return ParsedFile{}, err
	}
	return parsed, nil
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func durableMemoryPath(memoryDir, name string) (string, error) {
	path, err := MemoryPath(memoryDir, name)
	if err != nil {
		return "", err
	}
	base := filepath.Base(path)
	prefix, err := ValidatePrefix(base)
	if err != nil {
		return "", err
	}
	if prefix == "event" {
		return "", fmt.Errorf("classifier cannot write event memory: %s", name)
	}
	return path, nil
}
