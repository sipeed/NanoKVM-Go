package store

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	"memfab/internal/memory"
)

type FileRow struct {
	Path         string   `json:"path"`
	Prefix       string   `json:"prefix,omitempty"`
	Description  string   `json:"description,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Status       string   `json:"status,omitempty"`
	EntryCount   int      `json:"entry_count"`
	Created      string   `json:"created,omitempty"`
	Updated      string   `json:"updated,omitempty"`
	NeedsCompact bool     `json:"needs_compact,omitempty"`
}

type EntryHit struct {
	ID         string  `json:"id"`
	Path       string  `json:"path"`
	Prefix     string  `json:"prefix,omitempty"`
	Timestamp  string  `json:"timestamp"`
	Tags       string  `json:"tags,omitempty"`
	Content    string  `json:"content"`
	Superseded bool    `json:"superseded,omitempty"`
	Rank       float64 `json:"rank,omitempty"`
}

func (db *DB) RebuildMemoryIndex(memoryDir string) (int, int, int, error) {
	files, err := memory.ListMemoryFiles(memoryDir)
	if err != nil {
		return 0, 0, 0, err
	}
	tx, err := db.Conn.Begin()
	if err != nil {
		return 0, 0, 0, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM entries`); err != nil {
		return 0, 0, 0, err
	}
	if _, err := tx.Exec(`DELETE FROM files`); err != nil {
		return 0, 0, 0, err
	}
	fileCount, entryCount, skipped := 0, 0, 0
	for _, path := range files {
		prefix, err := memory.ValidatePrefix(path)
		if err != nil {
			skipped++
			continue
		}
		parsed, err := memory.ReadFile(path)
		if err != nil {
			skipped++
			continue
		}
		name := filepath.Base(path)
		if err := replaceMemoryIndexForParsedFileTx(tx, parsed, name, prefix); err != nil {
			return 0, 0, 0, err
		}
		entryCount += len(parsed.Entries)
		fileCount++
	}
	if err := tx.Commit(); err != nil {
		return 0, 0, 0, err
	}
	return fileCount, entryCount, skipped, nil
}

func replaceMemoryIndexForParsedFileTx(tx *sql.Tx, parsed memory.ParsedFile, name, prefix string) error {
	status := parsed.Status
	if status == "" {
		status = "active"
	}
	if _, err := tx.Exec(`INSERT INTO files(path, prefix, description, tags, status, entry_count, created, updated, needs_compact)
	    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	    ON CONFLICT(path) DO UPDATE SET
	      prefix=excluded.prefix,
	      description=excluded.description,
	      tags=excluded.tags,
	      status=excluded.status,
	      entry_count=excluded.entry_count,
	      created=excluded.created,
	      updated=excluded.updated,
	      needs_compact=excluded.needs_compact`, name, prefix, parsed.Description, strings.Join(parsed.Tags, " "), status, len(parsed.Entries), parsed.Created, parsed.Updated, boolInt(parsed.NeedsCompact)); err != nil {
		return err
	}
	for _, entry := range parsed.Entries {
		content, struck := stripStrike(entry.Body)
		superseded := entry.SupersededBy != "" || struck
		if _, err := tx.Exec(`INSERT INTO entries(id, path, prefix, timestamp, tags, content, superseded)
		    VALUES (?, ?, ?, ?, ?, ?, ?)`, entry.ID, name, prefix, entry.Timestamp, strings.Join(entry.Tags, " "), content, boolInt(superseded)); err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) ListMemoryFiles(includeDormant, includeArchived bool) ([]FileRow, error) {
	statuses := []string{"active"}
	if includeDormant {
		statuses = append(statuses, "dormant")
	}
	if includeArchived {
		statuses = append(statuses, "archived")
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(statuses)), ",")
	args := make([]any, 0, len(statuses))
	for _, status := range statuses {
		args = append(args, status)
	}
	rows, err := db.Conn.Query(fmt.Sprintf(`SELECT path, prefix, description, tags, status, entry_count, created, updated, needs_compact
	    FROM files WHERE status IN (%s) ORDER BY updated DESC, path ASC`, placeholders), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []FileRow
	for rows.Next() {
		var row FileRow
		var tags string
		var compact int
		if err := rows.Scan(&row.Path, &row.Prefix, &row.Description, &tags, &row.Status, &row.EntryCount, &row.Created, &row.Updated, &compact); err != nil {
			return nil, err
		}
		row.Tags = splitTags(tags)
		row.NeedsCompact = compact != 0
		out = append(out, row)
	}
	return out, rows.Err()
}

func (db *DB) SearchMemory(query string, paths []string, since, until string, topK int, includeSuperseded bool) ([]EntryHit, error) {
	if topK <= 0 || topK > 50 {
		topK = 5
	}
	if strings.TrimSpace(query) == "" {
		return []EntryHit{}, nil
	}
	where := []string{"entries MATCH ?"}
	args := []any{safeMemoryFTSQuery(query)}
	where, args = addMemoryFilters(where, args, paths, since, until, includeSuperseded)
	args = append(args, topK)
	q := fmt.Sprintf(`SELECT id, path, prefix, timestamp, tags, content, superseded, bm25(entries) AS rank
	    FROM entries WHERE %s ORDER BY rank LIMIT ?`, strings.Join(where, " AND "))
	rows, err := db.Conn.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEntryHits(rows)
}

func (db *DB) RecentMemory(since string, limit int, prefixFilter []string, includeSuperseded bool) ([]EntryHit, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	where := []string{"1=1"}
	args := []any{}
	if since != "" {
		where = append(where, "timestamp >= ?")
		args = append(args, since)
	}
	prefixes := normalizePrefixes(prefixFilter)
	if len(prefixes) > 0 {
		placeholders := strings.TrimRight(strings.Repeat("?,", len(prefixes)), ",")
		where = append(where, "prefix IN ("+placeholders+")")
		for _, prefix := range prefixes {
			args = append(args, prefix)
		}
	}
	if !includeSuperseded {
		where = append(where, "superseded=0")
	}
	args = append(args, limit)
	q := fmt.Sprintf(`SELECT id, path, prefix, timestamp, tags, content, superseded, 0.0 AS rank
	    FROM entries WHERE %s ORDER BY timestamp DESC LIMIT ?`, strings.Join(where, " AND "))
	rows, err := db.Conn.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEntryHits(rows)
}

func addMemoryFilters(where []string, args []any, paths []string, since, until string, includeSuperseded bool) ([]string, []any) {
	if len(paths) > 0 {
		parts := make([]string, 0, len(paths))
		for _, path := range paths {
			path = strings.TrimSpace(path)
			if path == "" {
				continue
			}
			parts = append(parts, "path GLOB ?")
			args = append(args, path)
		}
		if len(parts) > 0 {
			where = append(where, "("+strings.Join(parts, " OR ")+")")
		}
	}
	if since != "" {
		where = append(where, "timestamp >= ?")
		args = append(args, since)
	}
	if until != "" {
		where = append(where, "timestamp <= ?")
		args = append(args, until)
	}
	if !includeSuperseded {
		where = append(where, "superseded=0")
	}
	return where, args
}

func scanEntryHits(rows *sql.Rows) ([]EntryHit, error) {
	var out []EntryHit
	for rows.Next() {
		var hit EntryHit
		var superseded int
		if err := rows.Scan(&hit.ID, &hit.Path, &hit.Prefix, &hit.Timestamp, &hit.Tags, &hit.Content, &superseded, &hit.Rank); err != nil {
			return nil, err
		}
		hit.Superseded = superseded != 0
		out = append(out, hit)
	}
	return out, rows.Err()
}

func safeMemoryFTSQuery(query string) string {
	query = strings.Map(func(r rune) rune {
		switch r {
		case '"', ':', '*', '(', ')', '^', '+', '-':
			return ' '
		default:
			return r
		}
	}, query)
	parts := strings.Fields(query)
	for i, part := range parts {
		part = strings.ReplaceAll(part, `"`, `""`)
		parts[i] = `"` + part + `"`
	}
	return strings.Join(parts, " ")
}

func normalizePrefixes(prefixes []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(prefixes))
	for _, prefix := range prefixes {
		prefix = strings.TrimSuffix(strings.TrimSpace(prefix), "-")
		if prefix == "" || seen[prefix] {
			continue
		}
		seen[prefix] = true
		out = append(out, prefix)
	}
	return out
}

func splitTags(tags string) []string {
	parts := strings.Fields(tags)
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(strings.TrimPrefix(part, "#"))
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func stripStrike(body string) (string, bool) {
	trim := strings.TrimSpace(body)
	if strings.HasPrefix(trim, "~~") && strings.HasSuffix(trim, "~~") && len(trim) >= 4 {
		return strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trim, "~~"), "~~")), true
	}
	return body, false
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func (db *DB) UpsertMemoryFile(row FileRow) error {
	_, err := db.Conn.Exec(`INSERT INTO files(path, prefix, description, tags, status, entry_count, created, updated, needs_compact)
	    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	    ON CONFLICT(path) DO UPDATE SET
	      prefix=excluded.prefix,
	      description=excluded.description,
	      tags=excluded.tags,
	      status=excluded.status,
	      entry_count=excluded.entry_count,
	      created=excluded.created,
	      updated=excluded.updated,
	      needs_compact=excluded.needs_compact`,
		row.Path, row.Prefix, row.Description, strings.Join(row.Tags, " "), row.Status, row.EntryCount, row.Created, row.Updated, boolInt(row.NeedsCompact))
	return err
}

func (db *DB) InsertMemoryEntry(row MemoryEntryRow) error {
	_, err := db.Conn.Exec(`INSERT INTO entries(id, path, prefix, timestamp, tags, content, superseded)
	    VALUES (?, ?, ?, ?, ?, ?, ?)`, row.ID, row.Path, row.Prefix, row.Timestamp, row.Tags, row.Content, boolInt(row.Superseded))
	return err
}

func scanSessionRowsAll(rows *sql.Rows) ([]SessionRow, error) {
	out := make([]SessionRow, 0)
	for rows.Next() {
		session, err := scanSessionRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, session)
	}
	return out, rows.Err()
}

func (db *DB) MarkMemoryEntrySuperseded(path, oldEntryID string) error {
	_, err := db.Conn.Exec(`UPDATE entries SET superseded=1 WHERE path=? AND id=?`, path, oldEntryID)
	return err
}

func (db *DB) SetMemoryFileNeedsCompact(path string, needs bool) error {
	_, err := db.Conn.Exec(`UPDATE files SET needs_compact=? WHERE path=?`, boolInt(needs), path)
	return err
}

func (db *DB) FilesNeedingCompact(limit int) ([]FileRow, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := db.Conn.Query(`SELECT path, prefix, description, tags, status, entry_count, created, updated, needs_compact
	    FROM files
	   WHERE needs_compact=1 AND status IN ('active','dormant')
	ORDER BY updated ASC, path ASC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []FileRow
	for rows.Next() {
		var row FileRow
		var tags string
		var compact int
		if err := rows.Scan(&row.Path, &row.Prefix, &row.Description, &tags, &row.Status, &row.EntryCount, &row.Created, &row.Updated, &compact); err != nil {
			return nil, err
		}
		row.Tags = splitTags(tags)
		row.NeedsCompact = compact != 0
		out = append(out, row)
	}
	return out, rows.Err()
}

func (db *DB) ReindexMemoryFile(memoryDir, name string) error {
	path, err := memory.MemoryPath(memoryDir, name)
	if err != nil {
		return err
	}
	prefix, err := memory.ValidatePrefix(path)
	if err != nil {
		return err
	}
	parsed, err := memory.ReadFile(path)
	if err != nil {
		return err
	}
	base := filepath.Base(path)
	tx, err := db.Conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM entries WHERE path=?`, base); err != nil {
		return err
	}
	if err := replaceMemoryIndexForParsedFileTx(tx, parsed, base, prefix); err != nil {
		return err
	}
	return tx.Commit()
}
