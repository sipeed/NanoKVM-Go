package store

import (
	"database/sql"
	"fmt"
	"strings"

	"memfab/internal/capture"
)

func (db *DB) InsertObservation(id string, o capture.Observation) error {
	if strings.TrimSpace(o.CaptureID) == "" {
		return fmt.Errorf("capture_id is required")
	}
	_, err := db.Conn.Exec(`INSERT INTO ocr_observations
        (id, captured_at, capture_id, source_kind, source_device, source_engine, input_text_file,
         image_width, image_height, image_sha256,
         app_name, title, url, event_type, event_reason,
         language, ocr_text, confidence, block_count, tags)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(id) DO UPDATE SET
          captured_at=excluded.captured_at,
          capture_id=excluded.capture_id,
          source_kind=excluded.source_kind,
          source_device=excluded.source_device,
          source_engine=excluded.source_engine,
          input_text_file=excluded.input_text_file,
          image_width=excluded.image_width,
          image_height=excluded.image_height,
          image_sha256=excluded.image_sha256,
          app_name=excluded.app_name,
          title=excluded.title,
          url=excluded.url,
          event_type=excluded.event_type,
          event_reason=excluded.event_reason,
          language=excluded.language,
          ocr_text=excluded.ocr_text,
          confidence=excluded.confidence,
          block_count=excluded.block_count,
          tags=excluded.tags`,
		id, o.CapturedAt, o.CaptureID, o.Source.Kind, o.Source.Device, o.Source.Engine, o.Source.InputTextFile,
		nullableInt(o.Screen.Width), nullableInt(o.Screen.Height), o.Screen.ImageSHA256,
		o.Context.AppName, o.Context.Title, o.Context.URL, o.Event.Type, o.Event.Reason,
		o.OCR.Language, o.OCR.Text, nullableFloat(o.OCR.Confidence), o.OCR.BlockCount, tagsJSON(o.Tags))
	return err
}

func (db *DB) DeleteMissingObservations(existing map[string]bool) error {
	rows, err := db.Conn.Query(`SELECT id FROM ocr_observations`)
	if err != nil {
		return err
	}
	defer rows.Close()
	var remove []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		if !existing[id] {
			remove = append(remove, id)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, id := range remove {
		if err := db.DeleteObservation(id); err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) DeleteObservation(id string) error {
	_, err := db.Conn.Exec(`DELETE FROM ocr_observations WHERE id=?`, id)
	return err
}

type ObservationHit struct {
	ID           string   `json:"file_stem"`
	CapturedAt   string   `json:"captured_at"`
	SourceKind   string   `json:"source_kind,omitempty"`
	SourceDevice string   `json:"source_device,omitempty"`
	SourceEngine string   `json:"source_engine,omitempty"`
	AppName      string   `json:"app_name,omitempty"`
	Title        string   `json:"title,omitempty"`
	URL          string   `json:"url,omitempty"`
	ImageSHA256  string   `json:"image_sha256,omitempty"`
	EventType    string   `json:"event_type,omitempty"`
	EventReason  string   `json:"event_reason,omitempty"`
	Language     string   `json:"language,omitempty"`
	OCRText      string   `json:"-"`
	Confidence   *float64 `json:"confidence,omitempty"`
	BlockCount   int      `json:"block_count,omitempty"`
	Tags         string   `json:"tags,omitempty"`
	Snippet      string   `json:"snippet"`
	Rank         float64  `json:"rank"`
}

func escapeFTSQuery(q string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return ""
	}
	parts := strings.Fields(q)
	for i, p := range parts {
		p = strings.ReplaceAll(p, `"`, `""`)
		parts[i] = `"` + p + `"`
	}
	return strings.Join(parts, " ")
}

func (db *DB) SearchObservations(query, since, until, appName string, limit int) ([]ObservationHit, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	if !db.hasFTS5 || strings.TrimSpace(query) == "" {
		return db.searchObservationsLike(query, since, until, appName, limit)
	}
	where := []string{"ocr_observations_fts MATCH ?"}
	args := []any{escapeFTSQuery(query)}
	if since != "" {
		where = append(where, "o.captured_at >= ?")
		args = append(args, since)
	}
	if until != "" {
		where = append(where, "o.captured_at <= ?")
		args = append(args, until)
	}
	if appName != "" {
		where = append(where, "LOWER(o.app_name) LIKE ?")
		args = append(args, "%"+strings.ToLower(appName)+"%")
	}
	args = append(args, limit)
	q := fmt.Sprintf(`SELECT o.id, o.captured_at, o.source_kind, o.source_device, o.source_engine,
                 o.app_name, o.title, o.url, o.image_sha256, o.event_type, o.event_reason, o.language,
                 o.ocr_text, o.confidence, o.block_count, o.tags,
                 snippet(ocr_observations_fts, 4, '[', ']', '…', 16) AS snippet,
                 bm25(ocr_observations_fts) AS rank
            FROM ocr_observations_fts
            JOIN ocr_observations o ON o.rowid = ocr_observations_fts.rowid
           WHERE %s
        ORDER BY rank LIMIT ?`, strings.Join(where, " AND "))
	rows, err := db.Conn.Query(q, args...)
	if err != nil {
		return db.searchObservationsLike(query, since, until, appName, limit)
	}
	defer rows.Close()
	return scanObservationHits(rows)
}

func (db *DB) searchObservationsLike(query, since, until, appName string, limit int) ([]ObservationHit, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	where := []string{"1=1"}
	args := []any{}
	if query != "" {
		terms := strings.Fields(strings.ToLower(query))
		for _, term := range terms {
			where = append(where, `(LOWER(app_name) LIKE ? OR LOWER(title) LIKE ? OR LOWER(url) LIKE ? OR LOWER(event_reason) LIKE ? OR LOWER(ocr_text) LIKE ? OR LOWER(tags) LIKE ?)`)
			like := "%" + term + "%"
			args = append(args, like, like, like, like, like, like)
		}
	}
	if since != "" {
		where = append(where, "captured_at >= ?")
		args = append(args, since)
	}
	if until != "" {
		where = append(where, "captured_at <= ?")
		args = append(args, until)
	}
	if appName != "" {
		where = append(where, "LOWER(app_name) LIKE ?")
		args = append(args, "%"+strings.ToLower(appName)+"%")
	}
	args = append(args, limit)
	q := fmt.Sprintf(`SELECT id, captured_at, source_kind, source_device, source_engine,
                 app_name, title, url, image_sha256, event_type, event_reason, language,
                 ocr_text, confidence, block_count, tags
            FROM ocr_observations WHERE %s ORDER BY captured_at DESC LIMIT ?`, strings.Join(where, " AND "))
	rows, err := db.Conn.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	terms := strings.Fields(query)
	var out []ObservationHit
	for rows.Next() {
		var h ObservationHit
		if err := rows.Scan(&h.ID, &h.CapturedAt, &h.SourceKind, &h.SourceDevice, &h.SourceEngine, &h.AppName, &h.Title, &h.URL, &h.ImageSHA256, &h.EventType, &h.EventReason, &h.Language, &h.OCRText, &h.Confidence, &h.BlockCount, &h.Tags); err != nil {
			return nil, err
		}
		h.Snippet = makeSnippet(h.OCRText, terms)
		out = append(out, h)
	}
	return out, rows.Err()
}

func scanObservationHits(rows *sql.Rows) ([]ObservationHit, error) {
	var out []ObservationHit
	for rows.Next() {
		var h ObservationHit
		if err := rows.Scan(&h.ID, &h.CapturedAt, &h.SourceKind, &h.SourceDevice, &h.SourceEngine, &h.AppName, &h.Title, &h.URL, &h.ImageSHA256, &h.EventType, &h.EventReason, &h.Language, &h.OCRText, &h.Confidence, &h.BlockCount, &h.Tags, &h.Snippet, &h.Rank); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func makeSnippet(text string, terms []string) string {
	runes := []rune(text)
	if len(runes) > 240 {
		text = string(runes[:240]) + "…"
	}
	for _, term := range terms {
		if term == "" {
			continue
		}
		text = strings.ReplaceAll(text, term, "["+term+"]")
	}
	return text
}

func (db *DB) FindRecentObservationStem(at, appName, title string) (string, bool, error) {
	where := []string{"1=1"}
	args := []any{}
	if at != "" {
		if _, err := capture.ParseTimestamp(at); err != nil {
			return "", false, fmt.Errorf("invalid at timestamp: %w", err)
		}
		where = append(where, "captured_at <= ?")
		args = append(args, at)
	}
	if appName != "" {
		where = append(where, "LOWER(app_name) LIKE ?")
		args = append(args, "%"+strings.ToLower(appName)+"%")
	}
	if title != "" {
		where = append(where, "LOWER(title) LIKE ?")
		args = append(args, "%"+strings.ToLower(title)+"%")
	}
	q := fmt.Sprintf(`SELECT id FROM ocr_observations WHERE %s ORDER BY captured_at DESC LIMIT 1`, strings.Join(where, " AND "))
	var id string
	err := db.Conn.QueryRow(q, args...).Scan(&id)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return id, true, nil
}

func (db *DB) RecentObservations(limit int) ([]ObservationHit, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	rows, err := db.Conn.Query(`SELECT id, captured_at, source_kind, source_device, source_engine,
                 app_name, title, url, image_sha256, event_type, event_reason, language,
                 ocr_text, confidence, block_count, tags
        FROM ocr_observations ORDER BY captured_at DESC LIMIT ?`, limit)
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
