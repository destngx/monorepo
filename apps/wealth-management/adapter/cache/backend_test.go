package cache

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"path/filepath"
	"testing"
)

func TestGivenEachBackendWhenOpeningThenTheMatchingClientIsReturned(t *testing.T) {
	log := logger.NewTestLogger(t)

	upstash, err := NewBackend(domain.CacheConfig{Backend: domain.CacheBackendUpstash, RESTURL: "https://redis.example", RESTToken: "t"}, log)
	if _, ok := upstash.(*Client); err != nil || !ok {
		t.Fatalf("upstash = %T err %v", upstash, err)
	}

	sqlite, err := NewBackend(domain.CacheConfig{Backend: domain.CacheBackendSQLite, SQLitePath: filepath.Join(t.TempDir(), "cache.db")}, log)
	client, ok := sqlite.(*SQLiteClient)
	if err != nil || !ok {
		t.Fatalf("sqlite = %T err %v", sqlite, err)
	}
	t.Cleanup(func() { _ = client.Close() })

	if _, err := NewBackend(domain.CacheConfig{Backend: "memcached"}, log); err == nil {
		t.Fatal("expected an unknown backend to fail")
	}
}
