// Package settings stores engine settings in a SQLite file in the user
// config directory, private to the user (secrets are kept as plain text,
// protected by the file mode). It replaces the env file.
package settings

import (
	"apps/wealth-management/internal/sqlitefile"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const schema = `
CREATE TABLE IF NOT EXISTS settings (
	key        TEXT PRIMARY KEY,
	value      TEXT NOT NULL,
	updated_at INTEGER NOT NULL
);`

type Store struct {
	db   *sql.DB
	path string
	now  func() time.Time
}

// DefaultPath is settings.db in the user config directory (macOS
// ~/Library/Application Support/wealth-management, Linux
// $XDG_CONFIG_HOME/wealth-management).
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate user config directory: %w", err)
	}
	return filepath.Join(dir, "wealth-management", "settings.db"), nil
}

// ResolvePath is override (WM_SETTINGS_PATH) when set, DefaultPath otherwise.
func ResolvePath(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	return DefaultPath()
}

func Open(path string) (*Store, error) {
	db, err := sqlitefile.Open(path)
	if err != nil {
		return nil, fmt.Errorf("settings store: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create settings schema: %w", err)
	}
	return &Store{db: db, path: path, now: time.Now}, nil
}

func (s *Store) Path() string { return s.path }

func (s *Store) Values(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT key, value FROM settings`)
	if err != nil {
		return nil, fmt.Errorf("read settings: %w", err)
	}
	defer rows.Close()
	values := map[string]string{}
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("read settings: %w", err)
		}
		values[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read settings: %w", err)
	}
	return values, nil
}

func (s *Store) Save(ctx context.Context, changes map[string]string) (err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("save settings: %w", err)
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, rollback(tx))
		}
	}()
	updatedAt := s.now().UnixMilli()
	for key, value := range changes {
		if value == "" {
			_, err = tx.ExecContext(ctx, `DELETE FROM settings WHERE key = ?`, key)
		} else {
			_, err = tx.ExecContext(ctx,
				`INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?)
				 ON CONFLICT (key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
				key, value, updatedAt)
		}
		if err != nil {
			return fmt.Errorf("save setting %s: %w", key, err)
		}
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("save settings: %w", err)
	}
	return nil
}

func (s *Store) Close() error { return s.db.Close() }

func rollback(tx *sql.Tx) error {
	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		return fmt.Errorf("rollback settings: %w", err)
	}
	return nil
}
