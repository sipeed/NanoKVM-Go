package store

import (
	"database/sql"
	"encoding/json"
)

type TimelineBlock struct {
	ID           string   `json:"id"`
	StartTime    string   `json:"start_time"`
	EndTime      string   `json:"end_time"`
	Timezone     string   `json:"timezone,omitempty"`
	Entries      []string `json:"entries"`
	AppsUsed     []string `json:"apps_used"`
	CaptureCount int      `json:"capture_count"`
	CreatedAt    string   `json:"created_at"`
}

func (db *DB) ObservationsInWindow(start, end string, limit int) ([]ObservationHit, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := db.Conn.Query(`SELECT id, captured_at, source_kind, source_device, source_engine,
	             app_name, title, url, image_sha256, event_type, event_reason, language,
	             ocr_text, confidence, block_count, tags
	        FROM ocr_observations
	       WHERE julianday(captured_at) >= julianday(?) AND julianday(captured_at) < julianday(?)
	    ORDER BY julianday(captured_at) ASC LIMIT ?`, start, end, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ObservationHit
	for rows.Next() {
		var h ObservationHit
		if err := rows.Scan(&h.ID, &h.CapturedAt, &h.SourceKind, &h.SourceDevice, &h.SourceEngine, &h.AppName, &h.Title, &h.URL, &h.ImageSHA256, &h.EventType, &h.EventReason, &h.Language, &h.OCRText, &h.Confidence, &h.BlockCount, &h.Tags); err != nil {
			return nil, err
		}
		h.Snippet = makeSnippet(h.OCRText, nil)
		out = append(out, h)
	}
	return out, rows.Err()
}

func (db *DB) HasTimelineWindow(start, end string) (bool, error) {
	var n int
	if err := db.Conn.QueryRow(`SELECT COUNT(*) FROM timeline_blocks WHERE start_time=? AND end_time=?`, start, end).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

func (db *DB) LatestTimelineEnd() (string, bool, error) {
	var end string
	err := db.Conn.QueryRow(`SELECT end_time FROM timeline_blocks ORDER BY end_time DESC LIMIT 1`).Scan(&end)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return end, true, nil
}

func (db *DB) InsertTimelineBlock(block TimelineBlock) (bool, error) {
	entries, err := json.Marshal(block.Entries)
	if err != nil {
		return false, err
	}
	apps, err := json.Marshal(block.AppsUsed)
	if err != nil {
		return false, err
	}
	res, err := db.Conn.Exec(`INSERT OR IGNORE INTO timeline_blocks
	    (id, start_time, end_time, timezone, entries, apps_used, capture_count, created_at)
	    VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		block.ID, block.StartTime, block.EndTime, block.Timezone, string(entries), string(apps), block.CaptureCount, block.CreatedAt)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (db *DB) ListTimelineBlocks(limit int) ([]TimelineBlock, error) {
	if limit <= 0 || limit > 500 {
		limit = 24
	}
	rows, err := db.Conn.Query(`SELECT id, start_time, end_time, timezone, entries, apps_used, capture_count, created_at
	      FROM (SELECT id, start_time, end_time, timezone, entries, apps_used, capture_count, created_at
	              FROM timeline_blocks ORDER BY end_time DESC LIMIT ?)
	  ORDER BY start_time ASC`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TimelineBlock
	for rows.Next() {
		var block TimelineBlock
		var entries, apps string
		if err := rows.Scan(&block.ID, &block.StartTime, &block.EndTime, &block.Timezone, &entries, &apps, &block.CaptureCount, &block.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(entries), &block.Entries); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(apps), &block.AppsUsed); err != nil {
			return nil, err
		}
		out = append(out, block)
	}
	return out, rows.Err()
}

func (db *DB) TimelineBlocksIntersecting(start, end string, limit int) ([]TimelineBlock, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := db.Conn.Query(`SELECT id, start_time, end_time, timezone, entries, apps_used, capture_count, created_at
	        FROM timeline_blocks
	       WHERE julianday(end_time) > julianday(?) AND julianday(start_time) < julianday(?)
	    ORDER BY julianday(start_time) ASC LIMIT ?`, start, end, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TimelineBlock
	for rows.Next() {
		var block TimelineBlock
		var entries, apps string
		if err := rows.Scan(&block.ID, &block.StartTime, &block.EndTime, &block.Timezone, &entries, &apps, &block.CaptureCount, &block.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(entries), &block.Entries); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(apps), &block.AppsUsed); err != nil {
			return nil, err
		}
		out = append(out, block)
	}
	return out, rows.Err()
}
