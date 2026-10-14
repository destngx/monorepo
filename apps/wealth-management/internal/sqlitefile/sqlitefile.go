// Package sqlitefile opens a SQLite database file private to the user: the
// directory is 0700 and the file 0600 (SQLite gives its -wal and -shm files
// the same mode), with WAL and a busy timeout so several engine processes
// can share one file.
package sqlitefile

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// Open creates the file and its directory when missing and returns a pool
// limited to one connection, which serialises writers inside the process;
// the busy timeout covers other processes.
func Open(path string) (*sql.DB, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("missing SQLite file path")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create sqlite directory: %w", err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return nil, fmt.Errorf("restrict sqlite directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, fmt.Errorf("create sqlite file: %w", err)
	}
	if err := file.Close(); err != nil {
		return nil, fmt.Errorf("create sqlite file: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return nil, fmt.Errorf("restrict sqlite file: %w", err)
	}

	db, err := sql.Open("sqlite", dsn(path))
	if err != nil {
		return nil, fmt.Errorf("open sqlite file: %w", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("open sqlite file: %w", err)
	}
	return db, nil
}

func dsn(path string) string {
	query := url.Values{}
	query.Add("_pragma", "busy_timeout(5000)")
	query.Add("_pragma", "journal_mode(WAL)")
	query.Add("_pragma", "synchronous(NORMAL)")
	return "file:" + (&url.URL{Path: path}).EscapedPath() + "?" + query.Encode()
}
