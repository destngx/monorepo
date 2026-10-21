// Package sqlite is the SQLite implementation of port.Store (HW-OPS-1).
package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"time"

	"apps/health-watcher/domain"
	"apps/health-watcher/internal/sqlitefile"
	"apps/health-watcher/port"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// querier is what repositories need from *sql.DB and *sql.Tx.
type querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Store is the database file with every repository.
type Store struct {
	repos
	db *sql.DB
}

var _ port.Store = (*Store)(nil)

// Open opens the database at path and applies pending migrations (HW-OPS-3).
func Open(ctx context.Context, path string) (*Store, error) {
	db, err := sqlitefile.Open(path)
	if err != nil {
		return nil, err
	}
	if err := migrate(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{repos: repos{q: db}, db: db}, nil
}

// Close closes the database.
func (s *Store) Close() error { return s.db.Close() }

// DB returns the underlying pool, for the backup command and tests.
func (s *Store) DB() *sql.DB { return s.db }

// InTx runs fn in one transaction, committed when fn returns nil.
func (s *Store) InTx(ctx context.Context, fn func(port.Repos) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	if err := fn(repos{q: tx}); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return errors.Join(err, fmt.Errorf("roll back: %w", rbErr))
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

type migration struct {
	version int
	name    string
	sql     string
}

func loadMigrations() ([]migration, error) {
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return nil, fmt.Errorf("read migrations: %w", err)
	}
	var out []migration
	for _, e := range entries {
		name := e.Name()
		prefix, _, ok := strings.Cut(name, "_")
		version, convErr := strconv.Atoi(prefix)
		if !ok || convErr != nil {
			return nil, fmt.Errorf("migration %s: name must start with a number and an underscore", name)
		}
		body, err := migrationFiles.ReadFile("migrations/" + name)
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", name, err)
		}
		out = append(out, migration{version: version, name: name, sql: string(body)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	for i := 1; i < len(out); i++ {
		if out[i].version == out[i-1].version {
			return nil, fmt.Errorf("migrations %s and %s share version %d", out[i-1].name, out[i].name, out[i].version)
		}
	}
	return out, nil
}

// migrate applies each migration newer than the recorded version in its own
// transaction. A failure stops the engine (HW-OPS-3).
func migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY, name TEXT NOT NULL, applied_at TEXT NOT NULL)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	var current int
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&current); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	migrations, err := loadMigrations()
	if err != nil {
		return err
	}
	for _, m := range migrations {
		if m.version <= current {
			continue
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("migration %s: %w", m.name, err)
		}
		if _, err := tx.ExecContext(ctx, m.sql); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration %s: %w", m.name, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)`,
			m.version, m.name, formatTime(time.Now())); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", m.name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("migration %s: %w", m.name, err)
		}
	}
	return nil
}

// repos implements port.Repos over a pool or a transaction.
type repos struct {
	q querier
}

// timeLayout is RFC 3339 in UTC with a fixed nine-digit fraction, so stored
// times compare correctly as text (`at >= ?`). RFC3339Nano drops trailing
// zeros, which made a whole second sort after its fractions.
const timeLayout = "2006-01-02T15:04:05.000000000Z"

func formatTime(t time.Time) string { return t.UTC().Format(timeLayout) }

func parseTime(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse stored time %q: %w", s, err)
	}
	return t, nil
}

func parseTimePtr(s sql.NullString) (*time.Time, error) {
	if !s.Valid {
		return nil, nil
	}
	t, err := parseTime(s.String)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// one maps sql.ErrNoRows to a domain not-found error.
func one(err error, what string) error {
	if errors.Is(err, sql.ErrNoRows) {
		return domain.NotFound(what)
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", what, err)
	}
	return nil
}

// affected returns a not-found error when an update or delete matched no row.
func affected(res sql.Result, err error, what string) error {
	if err != nil {
		return fmt.Errorf("write %s: %w", what, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("write %s: %w", what, err)
	}
	if n == 0 {
		return domain.NotFound(what)
	}
	return nil
}

// exec wraps a write error with what was written.
func exec(_ sql.Result, err error, what string) error {
	if err != nil {
		if isUnique(err) {
			return domain.Conflict("%s already exists", what)
		}
		return fmt.Errorf("write %s: %w", what, err)
	}
	return nil
}

func isUnique(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
