package reset

import (
	"database/sql"
	"io/fs"
	"os"
	"path/filepath"

	"memfab/internal/paths"
	"memfab/internal/store"
)

type Stats struct {
	DeletedMemoryFiles    int  `json:"deleted_memory_files"`
	DeletedObservations   int  `json:"deleted_observations"`
	DeletedScreenshots    int  `json:"deleted_screenshots"`
	DeletedTimelineBlocks int  `json:"deleted_timeline_blocks"`
	DeletedSessions       int  `json:"deleted_sessions"`
	DeletedWriterState    bool `json:"deleted_writer_state"`
}

func Run(db *store.DB) (Stats, error) {
	if err := paths.EnsureDirs(); err != nil {
		return Stats{}, err
	}

	var stats Stats
	var err error
	if stats.DeletedObservations, err = countRows(db.Conn, "ocr_observations"); err != nil {
		return Stats{}, err
	}
	if stats.DeletedTimelineBlocks, err = countRows(db.Conn, "timeline_blocks"); err != nil {
		return Stats{}, err
	}
	if stats.DeletedSessions, err = countRows(db.Conn, "sessions"); err != nil {
		return Stats{}, err
	}

	if stats.DeletedMemoryFiles, err = removeChildren(paths.MemoryDir()); err != nil {
		return Stats{}, err
	}
	if _, err = removeChildren(paths.ObservationBufferDir()); err != nil {
		return Stats{}, err
	}
	deletedScreenshots, err := removeTreeIfExists(paths.ScreenshotsDir())
	if err != nil {
		return Stats{}, err
	}
	stats.DeletedScreenshots += deletedScreenshots
	if stats.DeletedWriterState, err = removeIfExists(paths.WriterState()); err != nil {
		return Stats{}, err
	}

	if err := resetDB(db); err != nil {
		return Stats{}, err
	}
	if err := paths.EnsureDirs(); err != nil {
		return Stats{}, err
	}
	return stats, nil
}

func countRows(db *sql.DB, table string) (int, error) {
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func resetDB(db *store.DB) error {
	tx, err := db.Conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, stmt := range []string{
		`DELETE FROM entries`,
		`DELETE FROM files`,
		`DELETE FROM ocr_observations`,
		`DELETE FROM timeline_blocks`,
		`DELETE FROM sessions`,
		`DELETE FROM sqlite_sequence WHERE name='ocr_observations'`,
	} {
		if _, err := tx.Exec(stmt); err != nil {
			return err
		}
	}
	if db.HasFTS5() {
		if _, err := tx.Exec(`INSERT INTO ocr_observations_fts(ocr_observations_fts) VALUES ('rebuild')`); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func removeChildren(root string) (int, error) {
	if err := os.MkdirAll(root, paths.DirectoryMode); err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	deletedFiles := 0
	for _, entry := range entries {
		path := filepath.Join(root, entry.Name())
		count, err := countFiles(path)
		if err != nil {
			return deletedFiles, err
		}
		if err := os.RemoveAll(path); err != nil {
			return deletedFiles, err
		}
		deletedFiles += count
	}
	return deletedFiles, os.MkdirAll(root, paths.DirectoryMode)
}

func removeTreeIfExists(root string) (int, error) {
	deletedFiles, err := countFiles(root)
	if err != nil {
		return 0, err
	}
	if err := os.RemoveAll(root); err != nil {
		return 0, err
	}
	return deletedFiles, nil
}

func countFiles(path string) (int, error) {
	count := 0
	err := filepath.WalkDir(path, func(path string, entry fs.DirEntry, err error) error {
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			count++
		}
		return nil
	})
	if os.IsNotExist(err) {
		return 0, nil
	}
	return count, err
}

func removeIfExists(path string) (bool, error) {
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
