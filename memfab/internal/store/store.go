package store

import (
	"database/sql"
	"encoding/json"

	_ "github.com/mattn/go-sqlite3"
	"memfab/internal/paths"
)

const baseSchema = `
DROP TRIGGER IF EXISTS captures_ai;
DROP TRIGGER IF EXISTS captures_ad;
DROP TRIGGER IF EXISTS captures_au;
DROP TABLE IF EXISTS captures_fts;
DROP TABLE IF EXISTS captures;

CREATE TABLE IF NOT EXISTS files (
    path TEXT PRIMARY KEY,
    prefix TEXT,
    description TEXT,
    tags TEXT,
    status TEXT,
    entry_count INTEGER,
    created TEXT,
    updated TEXT,
    needs_compact INTEGER DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_files_status ON files(status);
CREATE INDEX IF NOT EXISTS idx_files_prefix ON files(prefix);

CREATE TABLE IF NOT EXISTS ocr_observations (
    rowid INTEGER PRIMARY KEY AUTOINCREMENT,
    id TEXT UNIQUE NOT NULL,
    captured_at TEXT NOT NULL,
    capture_id TEXT NOT NULL,
    source_kind TEXT,
    source_device TEXT,
    source_engine TEXT,
    input_text_file TEXT,
    image_width INTEGER,
    image_height INTEGER,
    image_sha256 TEXT,
    app_name TEXT,
    title TEXT,
    url TEXT,
    event_type TEXT,
    event_reason TEXT,
    language TEXT,
    ocr_text TEXT,
    confidence REAL,
    block_count INTEGER,
    tags TEXT
);
CREATE INDEX IF NOT EXISTS idx_ocr_observations_ts ON ocr_observations(captured_at);
CREATE INDEX IF NOT EXISTS idx_ocr_observations_app ON ocr_observations(app_name);

CREATE TABLE IF NOT EXISTS timeline_blocks (
    id TEXT PRIMARY KEY,
    start_time TEXT NOT NULL,
    end_time TEXT NOT NULL,
    timezone TEXT,
    entries TEXT NOT NULL,
    apps_used TEXT NOT NULL,
    capture_count INTEGER NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE(start_time, end_time)
);
CREATE INDEX IF NOT EXISTS idx_timeline_end ON timeline_blocks(end_time);

CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    start_time TEXT NOT NULL,
    end_time TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    retry_count INTEGER NOT NULL DEFAULT 0,
    next_retry_at TEXT,
    last_error TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    flush_end TEXT,
    classified_end TEXT,
    classifier_retry_count INTEGER NOT NULL DEFAULT 0,
    classifier_next_retry_at TEXT,
    classifier_last_error TEXT
);
CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);
CREATE INDEX IF NOT EXISTS idx_sessions_start ON sessions(start_time);
CREATE INDEX IF NOT EXISTS idx_sessions_retry ON sessions(next_retry_at) WHERE status='failed';
PRAGMA user_version = 1;
`

const ftsSchema = `
CREATE VIRTUAL TABLE IF NOT EXISTS entries USING fts5(
    id UNINDEXED,
    path UNINDEXED,
    prefix UNINDEXED,
    timestamp UNINDEXED,
    tags,
    content,
    superseded UNINDEXED,
    tokenize = 'unicode61 remove_diacritics 2'
);

CREATE VIRTUAL TABLE IF NOT EXISTS ocr_observations_fts USING fts5(
    app_name, title, url, event_reason, ocr_text, tags,
    content='ocr_observations', content_rowid='rowid',
    tokenize='unicode61 remove_diacritics 2'
);
CREATE TRIGGER IF NOT EXISTS ocr_observations_ai AFTER INSERT ON ocr_observations BEGIN
    INSERT INTO ocr_observations_fts(rowid, app_name, title, url, event_reason, ocr_text, tags)
    VALUES (new.rowid, new.app_name, new.title, new.url, new.event_reason, new.ocr_text, new.tags);
END;
CREATE TRIGGER IF NOT EXISTS ocr_observations_ad AFTER DELETE ON ocr_observations BEGIN
    INSERT INTO ocr_observations_fts(ocr_observations_fts, rowid, app_name, title, url, event_reason, ocr_text, tags)
    VALUES ('delete', old.rowid, old.app_name, old.title, old.url, old.event_reason, old.ocr_text, old.tags);
END;
CREATE TRIGGER IF NOT EXISTS ocr_observations_au AFTER UPDATE ON ocr_observations BEGIN
    INSERT INTO ocr_observations_fts(ocr_observations_fts, rowid, app_name, title, url, event_reason, ocr_text, tags)
    VALUES ('delete', old.rowid, old.app_name, old.title, old.url, old.event_reason, old.ocr_text, old.tags);
    INSERT INTO ocr_observations_fts(rowid, app_name, title, url, event_reason, ocr_text, tags)
    VALUES (new.rowid, new.app_name, new.title, new.url, new.event_reason, new.ocr_text, new.tags);
END;
`

type DB struct {
	Conn    *sql.DB
	hasFTS5 bool
}

func Open() (*DB, error) {
	if err := paths.EnsureDirs(); err != nil {
		return nil, err
	}
	databaseFiles := []string{paths.IndexDB(), paths.IndexDB() + "-wal", paths.IndexDB() + "-shm"}
	if err := paths.SecureRegularFiles(databaseFiles...); err != nil {
		return nil, err
	}
	conn, err := sql.Open("sqlite3", paths.IndexDB())
	if err != nil {
		return nil, err
	}
	db := &DB{Conn: conn}
	if _, err := conn.Exec("PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL;"); err != nil {
		conn.Close()
		return nil, err
	}
	if err := paths.SecureRegularFiles(databaseFiles...); err != nil {
		conn.Close()
		return nil, err
	}
	if _, err := conn.Exec(baseSchema); err != nil {
		conn.Close()
		return nil, err
	}
	if err := ensureSessionClassifierColumns(conn); err != nil {
		conn.Close()
		return nil, err
	}
	if _, err := conn.Exec(ftsSchema); err != nil {
		conn.Close()
		return nil, err
	}
	db.hasFTS5 = true
	return db, nil
}

func (db *DB) Close() error  { return db.Conn.Close() }
func (db *DB) HasFTS5() bool { return db.hasFTS5 }

func ensureSessionClassifierColumns(conn *sql.DB) error {
	columns := []struct {
		name string
		ddl  string
	}{
		{"classifier_retry_count", "ALTER TABLE sessions ADD COLUMN classifier_retry_count INTEGER NOT NULL DEFAULT 0"},
		{"classifier_next_retry_at", "ALTER TABLE sessions ADD COLUMN classifier_next_retry_at TEXT"},
		{"classifier_last_error", "ALTER TABLE sessions ADD COLUMN classifier_last_error TEXT"},
	}
	existing := map[string]bool{}
	rows, err := conn.Query(`PRAGMA table_info(sessions)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		existing[name] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, column := range columns {
		if existing[column.name] {
			continue
		}
		if _, err := conn.Exec(column.ddl); err != nil {
			return err
		}
	}
	_, err = conn.Exec(`CREATE INDEX IF NOT EXISTS idx_sessions_classifier_retry ON sessions(classifier_next_retry_at) WHERE status='reduced'`)
	return err
}

func nullableInt(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullableFloat(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

func tagsJSON(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	b, _ := json.Marshal(tags)
	return string(b)
}
