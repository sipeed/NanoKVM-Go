package store

import (
	"database/sql"
	"fmt"
)

type SessionStatus string

const (
	SessionActive  SessionStatus = "active"
	SessionEnded   SessionStatus = "ended"
	SessionReduced SessionStatus = "reduced"
	SessionFailed  SessionStatus = "failed"
)

type SessionRow struct {
	ID                    string `json:"id"`
	StartTime             string `json:"start_time"`
	EndTime               string `json:"end_time,omitempty"`
	Status                string `json:"status"`
	RetryCount            int    `json:"retry_count"`
	NextRetryAt           string `json:"next_retry_at,omitempty"`
	LastError             string `json:"last_error,omitempty"`
	CreatedAt             string `json:"created_at"`
	UpdatedAt             string `json:"updated_at"`
	FlushEnd              string `json:"flush_end,omitempty"`
	ClassifiedEnd         string `json:"classified_end,omitempty"`
	ClassifierRetryCount  int    `json:"classifier_retry_count"`
	ClassifierNextRetryAt string `json:"classifier_next_retry_at,omitempty"`
	ClassifierLastError   string `json:"classifier_last_error,omitempty"`
}

func (db *DB) InsertSession(row SessionRow) (bool, error) {
	if row.Status == "" {
		row.Status = string(SessionActive)
	}
	res, err := db.Conn.Exec(`INSERT OR IGNORE INTO sessions
	    (id, start_time, end_time, status, retry_count, next_retry_at, last_error, created_at, updated_at, flush_end, classified_end, classifier_retry_count, classifier_next_retry_at, classifier_last_error)
	    VALUES (?, ?, NULLIF(?, ''), ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, NULLIF(?, ''), NULLIF(?, ''))`,
		row.ID, row.StartTime, row.EndTime, row.Status, row.RetryCount, row.NextRetryAt, row.LastError, row.CreatedAt, row.UpdatedAt, row.FlushEnd, row.ClassifiedEnd, row.ClassifierRetryCount, row.ClassifierNextRetryAt, row.ClassifierLastError)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (db *DB) GetSession(id string) (*SessionRow, error) {
	row := db.Conn.QueryRow(`SELECT id, start_time, COALESCE(end_time,''), status, retry_count,
	       COALESCE(next_retry_at,''), COALESCE(last_error,''), created_at, updated_at,
	       COALESCE(flush_end,''), COALESCE(classified_end,''), classifier_retry_count,
	       COALESCE(classifier_next_retry_at,''), COALESCE(classifier_last_error,'')
	  FROM sessions WHERE id=?`, id)
	return scanSessionRow(row)
}

func (db *DB) GetActiveSession() (*SessionRow, error) {
	row := db.Conn.QueryRow(`SELECT id, start_time, COALESCE(end_time,''), status, retry_count,
	       COALESCE(next_retry_at,''), COALESCE(last_error,''), created_at, updated_at,
	       COALESCE(flush_end,''), COALESCE(classified_end,''), classifier_retry_count,
	       COALESCE(classifier_next_retry_at,''), COALESCE(classifier_last_error,'')
	  FROM sessions WHERE status='active' ORDER BY start_time DESC LIMIT 1`)
	return scanSessionRow(row)
}

func (db *DB) MarkSessionEnded(id, endTime, updatedAt string) error {
	_, err := db.Conn.Exec(`UPDATE sessions SET status='ended', end_time=?, updated_at=? WHERE id=? AND status='active'`, endTime, updatedAt, id)
	return err
}

func (db *DB) ListSessions(status string, limit int) ([]SessionRow, error) {
	if limit <= 0 || limit > 500 {
		limit = 20
	}
	where := "1=1"
	args := []any{}
	if status != "" && status != "all" {
		where = "status=?"
		args = append(args, status)
	}
	args = append(args, limit)
	rows, err := db.Conn.Query(fmt.Sprintf(`SELECT id, start_time, COALESCE(end_time,''), status, retry_count,
	       COALESCE(next_retry_at,''), COALESCE(last_error,''), created_at, updated_at,
	       COALESCE(flush_end,''), COALESCE(classified_end,''), classifier_retry_count,
	       COALESCE(classifier_next_retry_at,''), COALESCE(classifier_last_error,'')
	  FROM sessions WHERE %s ORDER BY start_time DESC LIMIT ?`, where), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
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

func (db *DB) LatestSessionEnd() (string, bool, error) {
	var end string
	err := db.Conn.QueryRow(`SELECT end_time FROM sessions WHERE end_time IS NOT NULL ORDER BY end_time DESC LIMIT 1`).Scan(&end)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return end, true, nil
}

func (db *DB) EarliestObservationTime() (string, bool, error) {
	var start string
	err := db.Conn.QueryRow(`SELECT captured_at FROM ocr_observations ORDER BY julianday(captured_at), rowid LIMIT 1`).Scan(&start)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return start, true, nil
}

func (db *DB) SessionObservations(since string, includeStart bool, limit int) ([]ObservationHit, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	where := "1=1"
	args := []any{}
	if since != "" {
		comparison := " > "
		if includeStart {
			comparison = " >= "
		}
		where = "julianday(captured_at)" + comparison + "julianday(?)"
		args = append(args, since)
	}
	args = append(args, limit)
	rows, err := db.Conn.Query(fmt.Sprintf(`SELECT id, captured_at, source_kind, source_device, source_engine,
	             app_name, title, url, image_sha256, event_type, event_reason, language,
	             ocr_text, confidence, block_count, tags
	        FROM ocr_observations
	       WHERE %s
	    ORDER BY julianday(captured_at) ASC, rowid ASC LIMIT ?`, where), args...)
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

type sessionScanner interface {
	Scan(dest ...any) error
}

func scanSessionRow(row sessionScanner) (*SessionRow, error) {
	session, err := scanSession(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func scanSessionRows(rows *sql.Rows) (SessionRow, error) { return scanSession(rows) }

func scanSession(row sessionScanner) (SessionRow, error) {
	var session SessionRow
	err := row.Scan(&session.ID, &session.StartTime, &session.EndTime, &session.Status, &session.RetryCount, &session.NextRetryAt, &session.LastError, &session.CreatedAt, &session.UpdatedAt, &session.FlushEnd, &session.ClassifiedEnd, &session.ClassifierRetryCount, &session.ClassifierNextRetryAt, &session.ClassifierLastError)
	return session, err
}

type MemoryEntryRow struct {
	ID         string
	Path       string
	Prefix     string
	Timestamp  string
	Tags       string
	Content    string
	Superseded bool
}

func (db *DB) ListPendingReduction(limit int) ([]SessionRow, error) {
	if limit <= 0 || limit > 500 {
		limit = 20
	}
	rows, err := db.Conn.Query(`SELECT id, start_time, COALESCE(end_time,''), status, retry_count,
	       COALESCE(next_retry_at,''), COALESCE(last_error,''), created_at, updated_at,
	       COALESCE(flush_end,''), COALESCE(classified_end,''), classifier_retry_count,
	       COALESCE(classifier_next_retry_at,''), COALESCE(classifier_last_error,'')
	  FROM sessions WHERE status IN ('ended','failed') ORDER BY start_time ASC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSessionRowsAll(rows)
}

func (db *DB) ListDueReductions(now string, limit int) ([]SessionRow, error) {
	if limit <= 0 || limit > 500 {
		limit = 20
	}
	rows, err := db.Conn.Query(`SELECT id, start_time, COALESCE(end_time,''), status, retry_count,
	       COALESCE(next_retry_at,''), COALESCE(last_error,''), created_at, updated_at,
	       COALESCE(flush_end,''), COALESCE(classified_end,''), classifier_retry_count,
	       COALESCE(classifier_next_retry_at,''), COALESCE(classifier_last_error,'')
	  FROM sessions
	 WHERE status='ended'
	    OR (status='failed' AND (next_retry_at IS NULL OR next_retry_at='' OR julianday(next_retry_at) <= julianday(?)))
	 ORDER BY start_time ASC LIMIT ?`, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSessionRowsAll(rows)
}

func (db *DB) MarkSessionFailed(id, lastError, nextRetryAt, updatedAt string, retryCount int) error {
	_, err := db.Conn.Exec(`UPDATE sessions SET status='failed', retry_count=?, next_retry_at=NULLIF(?, ''), last_error=?, updated_at=? WHERE id=?`, retryCount, nextRetryAt, lastError, updatedAt, id)
	return err
}

func (db *DB) MarkSessionReduced(id, flushEnd, updatedAt string) error {
	_, err := db.Conn.Exec(`UPDATE sessions SET status='reduced', flush_end=NULLIF(?, ''), updated_at=? WHERE id=?`, flushEnd, updatedAt, id)
	return err
}

func (db *DB) SetSessionFlushEnd(id, flushEnd, updatedAt string) error {
	_, err := db.Conn.Exec(`UPDATE sessions SET flush_end=?, updated_at=? WHERE id=?`, flushEnd, updatedAt, id)
	return err
}

func (db *DB) SetSessionClassifiedEnd(id, classifiedEnd, updatedAt string) error {
	_, err := db.Conn.Exec(`UPDATE sessions SET classified_end=?, classifier_retry_count=0, classifier_next_retry_at=NULL, classifier_last_error=NULL, updated_at=? WHERE id=?`, classifiedEnd, updatedAt, id)
	return err
}

func (db *DB) MarkSessionClassificationFailed(id, lastError, nextRetryAt, updatedAt string, retryCount int) error {
	_, err := db.Conn.Exec(`UPDATE sessions SET classifier_retry_count=?, classifier_next_retry_at=NULLIF(?, ''), classifier_last_error=?, updated_at=? WHERE id=?`, retryCount, nextRetryAt, lastError, updatedAt, id)
	return err
}

func (db *DB) ListSessionsNeedingClassification(now string, limit int) ([]SessionRow, error) {
	if limit <= 0 || limit > 500 {
		limit = 20
	}
	rows, err := db.Conn.Query(`SELECT id, start_time, COALESCE(end_time,''), status, retry_count,
	       COALESCE(next_retry_at,''), COALESCE(last_error,''), created_at, updated_at,
	       COALESCE(flush_end,''), COALESCE(classified_end,''), classifier_retry_count,
	       COALESCE(classifier_next_retry_at,''), COALESCE(classifier_last_error,'')
	  FROM sessions
	 WHERE status='reduced'
	   AND COALESCE(NULLIF(flush_end,''), end_time, '') != ''
	   AND (classified_end IS NULL OR classified_end='' OR julianday(classified_end) < julianday(COALESCE(NULLIF(flush_end,''), end_time)))
	   AND (classifier_next_retry_at IS NULL OR classifier_next_retry_at='' OR julianday(classifier_next_retry_at) <= julianday(?))
	 ORDER BY start_time ASC LIMIT ?`, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSessionRowsAll(rows)
}
