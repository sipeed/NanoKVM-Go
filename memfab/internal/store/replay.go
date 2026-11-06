package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

type OCRSearchOptions struct {
	Since string
	Until string
	Limit int
}

type OCRSearchRow struct {
	CapturedAt string
	CaptureID  string
	OCRPreview string
}

func (db *DB) SearchOCRHits(query string, opts OCRSearchOptions) ([]OCRSearchRow, error) {
	if strings.TrimSpace(query) == "" {
		return []OCRSearchRow{}, nil
	}
	if !db.hasFTS5 {
		return db.searchOCRHitsLike(query, opts)
	}
	limit := normalizeOCRSearchLimit(opts.Limit)
	where := []string{"ocr_observations_fts MATCH ?"}
	args := []any{"{ocr_text}: " + escapeFTSQuery(query)}
	if opts.Since != "" {
		where = append(where, "julianday(o.captured_at) >= julianday(?)")
		args = append(args, opts.Since)
	}
	if opts.Until != "" {
		where = append(where, "julianday(o.captured_at) <= julianday(?)")
		args = append(args, opts.Until)
	}
	args = append(args, limit)
	q := fmt.Sprintf(`SELECT o.captured_at, o.capture_id,
	             snippet(ocr_observations_fts, 4, '[', ']', '…', 16) AS snippet,
	             bm25(ocr_observations_fts) AS rank, o.rowid
	        FROM ocr_observations_fts
	        JOIN ocr_observations o ON o.rowid = ocr_observations_fts.rowid
	       WHERE %s
	    ORDER BY rank, julianday(o.captured_at) ASC, o.rowid ASC LIMIT ?`, strings.Join(where, " AND "))
	rows, err := db.Conn.Query(q, args...)
	if err != nil {
		return db.searchOCRHitsLike(query, opts)
	}
	defer rows.Close()
	hits, err := scanOCRSearchRows(rows, true, nil)
	if err != nil {
		return nil, err
	}
	if len(hits) == 0 {
		return db.searchOCRHitsLike(query, opts)
	}
	return hits, nil
}

func (db *DB) searchOCRHitsLike(query string, opts OCRSearchOptions) ([]OCRSearchRow, error) {
	limit := normalizeOCRSearchLimit(opts.Limit)
	where := []string{"1=1"}
	args := []any{}
	terms := strings.Fields(query)
	for _, term := range terms {
		where = append(where, "LOWER(ocr_text) LIKE ?")
		args = append(args, "%"+strings.ToLower(term)+"%")
	}
	if opts.Since != "" {
		where = append(where, "julianday(captured_at) >= julianday(?)")
		args = append(args, opts.Since)
	}
	if opts.Until != "" {
		where = append(where, "julianday(captured_at) <= julianday(?)")
		args = append(args, opts.Until)
	}
	args = append(args, limit)
	q := fmt.Sprintf(`SELECT captured_at, capture_id, ocr_text
	        FROM ocr_observations
	       WHERE %s
	    ORDER BY julianday(captured_at) ASC, rowid ASC LIMIT ?`, strings.Join(where, " AND "))
	rows, err := db.Conn.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOCRSearchRows(rows, false, terms)
}

func (db *DB) GetTimelineBlock(id string) (*TimelineBlock, error) {
	row := db.Conn.QueryRow(`SELECT id, start_time, end_time, timezone, entries, apps_used, capture_count, created_at
	      FROM timeline_blocks WHERE id=?`, id)
	var block TimelineBlock
	var entries, apps string
	if err := row.Scan(&block.ID, &block.StartTime, &block.EndTime, &block.Timezone, &entries, &apps, &block.CaptureCount, &block.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if err := jsonUnmarshalTimelineBlock(&block, entries, apps); err != nil {
		return nil, err
	}
	return &block, nil
}

func (db *DB) MemoryEntriesByPath(path, entryID string, includeSuperseded bool) ([]EntryHit, error) {
	name := filepath.Base(strings.TrimSpace(path))
	if name == "." || name == string(filepath.Separator) || name == "" {
		return nil, fmt.Errorf("memory path is required")
	}
	where := []string{"path=?"}
	args := []any{name}
	if entryID != "" {
		where = append(where, "id=?")
		args = append(args, entryID)
	}
	if !includeSuperseded {
		where = append(where, "superseded=0")
	}
	q := fmt.Sprintf(`SELECT id, path, prefix, timestamp, tags, content, superseded, 0.0 AS rank
	    FROM entries WHERE %s ORDER BY julianday(timestamp) ASC, id ASC`, strings.Join(where, " AND "))
	rows, err := db.Conn.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEntryHits(rows)
}

func normalizeOCRSearchLimit(limit int) int {
	if limit <= 0 || limit > 5000 {
		return 5000
	}
	return limit
}

func scanOCRSearchRows(rows *sql.Rows, withRank bool, terms []string) ([]OCRSearchRow, error) {
	var out []OCRSearchRow
	for rows.Next() {
		var row OCRSearchRow
		if withRank {
			var rank float64
			var rowID int64
			if err := rows.Scan(&row.CapturedAt, &row.CaptureID, &row.OCRPreview, &rank, &rowID); err != nil {
				return nil, err
			}
		} else {
			var ocrText string
			if err := rows.Scan(&row.CapturedAt, &row.CaptureID, &ocrText); err != nil {
				return nil, err
			}
			row.OCRPreview = makeSnippet(ocrText, terms)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func jsonUnmarshalTimelineBlock(block *TimelineBlock, entries, apps string) error {
	if err := json.Unmarshal([]byte(entries), &block.Entries); err != nil {
		return err
	}
	return json.Unmarshal([]byte(apps), &block.AppsUsed)
}
