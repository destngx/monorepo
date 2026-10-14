package cache

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"apps/wealth-management/internal/sqlitefile"
)

// SQLiteClient is the local cache backend used when Upstash is not
// configured. It keeps Redis semantics the cache service relies on: TTLs
// (expired keys are invisible and purged), glob SCAN, DEL counting only keys
// that existed, and the key prefix so environments sharing one file never
// see each other's keys (WM-OPS-2b). Several engine processes (dev server,
// e2e run) can share the file, and so does apps/market-data-server by
// default (src/cache.py, keys under mds:): keep the two schemas identical.
type SQLiteClient struct {
	db     *sql.DB
	prefix string
	now    func() time.Time
}

// deleteBatchSize keeps DELETE ... IN (...) well under SQLite's variable limit.
const deleteBatchSize = 500

const sqliteSchema = `
CREATE TABLE IF NOT EXISTS cache_entries (
	key        TEXT PRIMARY KEY,
	value      TEXT NOT NULL,
	expires_at INTEGER
);
CREATE INDEX IF NOT EXISTS cache_entries_expires_at ON cache_entries (expires_at) WHERE expires_at IS NOT NULL;`

// liveCondition matches rows that have not expired; its one argument is now
// in Unix milliseconds.
const liveCondition = `(expires_at IS NULL OR expires_at > ?)`

func NewSQLiteClient(config domain.CacheConfig, log *logger.Logger) (*SQLiteClient, error) {
	return newSQLiteClient(config, log, time.Now)
}

func newSQLiteClient(config domain.CacheConfig, log *logger.Logger, now func() time.Time) (*SQLiteClient, error) {
	log.LogApplicationEvent(context.Background(), "initializing sqlite cache client",
		slog.String("path", config.SQLitePath),
		slog.String("key_prefix", config.KeyPrefix),
		slog.String("component", "sqlite_cache"),
	)
	// The cache holds copies of ledger data, so the file is private.
	db, err := sqlitefile.Open(config.SQLitePath)
	if err != nil {
		return nil, fmt.Errorf("sqlite cache: %w", err)
	}
	client := &SQLiteClient{db: db, prefix: config.KeyPrefix, now: now}

	ctx := context.Background()
	if _, err := db.ExecContext(ctx, sqliteSchema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create sqlite cache schema: %w", err)
	}
	if err := client.purgeExpired(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return client, nil
}

func (c *SQLiteClient) Set(ctx context.Context, key string, value string, ttlSeconds int) error {
	// Like SET without KEEPTTL, a write replaces both value and TTL.
	var expiresAt any
	if ttlSeconds > 0 {
		expiresAt = c.now().Add(time.Duration(ttlSeconds) * time.Second).UnixMilli()
	}
	_, err := c.db.ExecContext(ctx,
		`INSERT INTO cache_entries (key, value, expires_at) VALUES (?, ?, ?)
		 ON CONFLICT (key) DO UPDATE SET value = excluded.value, expires_at = excluded.expires_at`,
		c.prefix+key, value, expiresAt)
	if err != nil {
		return fmt.Errorf("sqlite cache SET: %w", err)
	}
	return nil
}

func (c *SQLiteClient) Get(ctx context.Context, key string) (string, bool, error) {
	var value string
	err := c.db.QueryRowContext(ctx,
		`SELECT value FROM cache_entries WHERE key = ? AND `+liveCondition,
		c.prefix+key, c.nowMillis()).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("sqlite cache GET: %w", err)
	}
	return value, true, nil
}

// Scan returns every live key matching a Redis glob pattern (without the
// prefix). Invalidation runs it on every sync, so it also purges expired rows.
func (c *SQLiteClient) Scan(ctx context.Context, pattern string) ([]string, error) {
	if err := c.purgeExpired(ctx); err != nil {
		return nil, err
	}
	rows, err := c.db.QueryContext(ctx,
		`SELECT key FROM cache_entries WHERE key GLOB ? AND `+liveCondition+` ORDER BY key`,
		escapeGlob(c.prefix)+redisGlobToSQLite(pattern), c.nowMillis())
	if err != nil {
		return nil, fmt.Errorf("sqlite cache SCAN: %w", err)
	}
	defer rows.Close()
	keys := []string{}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("sqlite cache SCAN: %w", err)
		}
		keys = append(keys, strings.TrimPrefix(key, c.prefix))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite cache SCAN: %w", err)
	}
	return keys, nil
}

// Delete removes the given keys (without the prefix) and returns how many
// existed; like Redis, an expired key does not count.
func (c *SQLiteClient) Delete(ctx context.Context, keys ...string) (int, error) {
	deleted := 0
	for start := 0; start < len(keys); start += deleteBatchSize {
		batch := keys[start:min(start+deleteBatchSize, len(keys))]
		args := make([]any, 0, len(batch)+1)
		for _, key := range batch {
			args = append(args, c.prefix+key)
		}
		args = append(args, c.nowMillis())
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(batch)), ",")
		result, err := c.db.ExecContext(ctx,
			`DELETE FROM cache_entries WHERE key IN (`+placeholders+`) AND `+liveCondition, args...)
		if err != nil {
			return deleted, fmt.Errorf("sqlite cache DEL: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return deleted, fmt.Errorf("sqlite cache DEL: %w", err)
		}
		deleted += int(affected)
	}
	return deleted, nil
}

func (c *SQLiteClient) Ping(ctx context.Context) error {
	if err := c.db.PingContext(ctx); err != nil {
		return fmt.Errorf("sqlite cache PING: %w", err)
	}
	return nil
}

func (c *SQLiteClient) Close() error {
	return c.db.Close()
}

func (c *SQLiteClient) purgeExpired(ctx context.Context) error {
	if _, err := c.db.ExecContext(ctx,
		`DELETE FROM cache_entries WHERE expires_at IS NOT NULL AND expires_at <= ?`, c.nowMillis()); err != nil {
		return fmt.Errorf("sqlite cache purge expired: %w", err)
	}
	return nil
}

func (c *SQLiteClient) nowMillis() int64 {
	return c.now().UnixMilli()
}

// escapeGlob makes every character of s literal in a SQLite GLOB.
func escapeGlob(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '*', '?', '[':
			b.WriteByte('[')
			b.WriteRune(r)
			b.WriteByte(']')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// redisGlobToSQLite converts a Redis MATCH pattern to SQLite GLOB. Both
// share *, ? and [...] (with ^ negation); Redis escapes with a backslash,
// which GLOB lacks, so an escaped character becomes a one-character class.
func redisGlobToSQLite(pattern string) string {
	var b strings.Builder
	runes := []rune(pattern)
	for i := 0; i < len(runes); i++ {
		if runes[i] == '\\' && i+1 < len(runes) {
			i++
			b.WriteString(escapeGlob(string(runes[i])))
			continue
		}
		b.WriteRune(runes[i])
	}
	return b.String()
}
