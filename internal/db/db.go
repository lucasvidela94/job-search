// Package db manages the SQLite database for application tracking.
package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // pure-Go SQLite driver
)

// DB wraps sql.DB with migration support.
type DB struct {
	*sql.DB
	dir string
}

// Open opens (or creates) the SQLite database at dir/state/candidate.db.
func Open(dir string) (*DB, error) {
	dbDir := filepath.Join(dir, "state")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	dbPath := filepath.Join(dbDir, "candidate.db")
	sqldb, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	sqldb.SetMaxOpenConns(1) // SQLite doesn't support concurrent writes

	db := &DB{DB: sqldb, dir: dir}
	if err := db.Migrate(); err != nil {
		sqldb.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

// Migrate creates tables if they don't exist.
func (db *DB) Migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS applications (
		id TEXT PRIMARY KEY,
		job_id TEXT NOT NULL,
		title TEXT NOT NULL,
		company TEXT NOT NULL,
		url TEXT NOT NULL DEFAULT '',
		source TEXT NOT NULL DEFAULT '',
		apply_url TEXT NOT NULL DEFAULT '',
		applied_at TEXT NOT NULL DEFAULT (datetime('now')),
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS events (
		id TEXT PRIMARY KEY,
		application_id TEXT NOT NULL REFERENCES applications(id),
		type TEXT NOT NULL,
		notes TEXT NOT NULL DEFAULT '',
		timestamp TEXT NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS cover_letters (
		id TEXT PRIMARY KEY,
		application_id TEXT NOT NULL REFERENCES applications(id),
		content TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	);

	CREATE INDEX IF NOT EXISTS idx_events_app_id ON events(application_id);
	CREATE INDEX IF NOT EXISTS idx_cover_app_id ON cover_letters(application_id);
	CREATE INDEX IF NOT EXISTS idx_apps_url ON applications(url);
	`

	_, err := db.Exec(schema)
	return err
}

// Dir returns the database directory.
func (db *DB) Dir() string { return filepath.Join(db.dir, "state") }
