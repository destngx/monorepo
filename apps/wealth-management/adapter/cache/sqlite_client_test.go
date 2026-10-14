package cache

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// testClock is a settable clock so TTL tests never sleep.
type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newTestSQLiteClient(t *testing.T, path string, prefix string, clock *testClock) *SQLiteClient {
	t.Helper()
	now := time.Now
	if clock != nil {
		now = clock.Now
	}
	client, err := newSQLiteClient(domain.CacheConfig{Backend: domain.CacheBackendSQLite, SQLitePath: path, KeyPrefix: prefix}, logger.NewTestLogger(t), now)
	if err != nil {
		t.Fatalf("open sqlite cache: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func sqlitePath(t *testing.T) string {
	return filepath.Join(t.TempDir(), "nested", "cache.db")
}

func TestGivenAMissingDirectoryWhenOpeningTheSQLiteCacheThenItIsCreatedPrivately(t *testing.T) {
	path := sqlitePath(t)
	newTestSQLiteClient(t, path, "wm:", nil)

	dir, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if perm := dir.Mode().Perm(); perm != 0o700 {
		t.Fatalf("dir perm = %o, want 700", perm)
	}
	file, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	if perm := file.Mode().Perm(); perm != 0o600 {
		t.Fatalf("file perm = %o, want 600", perm)
	}
}

func TestGivenAnEmptyPathWhenOpeningTheSQLiteCacheThenItFails(t *testing.T) {
	if _, err := NewSQLiteClient(domain.CacheConfig{Backend: domain.CacheBackendSQLite}, logger.NewTestLogger(t)); err == nil {
		t.Fatal("expected an error for an empty path")
	}
}

func TestGivenAStoredValueWhenGettingThenItIsReturnedAndUnknownKeysMiss(t *testing.T) {
	client := newTestSQLiteClient(t, sqlitePath(t), "wm:", nil)
	ctx := context.Background()
	large := strings.Repeat("x", 1_000_000)

	if err := client.Set(ctx, "accounts:all", large, 300); err != nil {
		t.Fatalf("set: %v", err)
	}
	value, found, err := client.Get(ctx, "accounts:all")
	if err != nil || !found || value != large {
		t.Fatalf("get = len %d found %v err %v", len(value), found, err)
	}
	if _, found, err := client.Get(ctx, "accounts:none"); err != nil || found {
		t.Fatalf("unknown key: found %v err %v", found, err)
	}
}

func TestGivenATTLWhenItElapsesThenTheKeyIsGone(t *testing.T) {
	clock := &testClock{now: time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC)}
	client := newTestSQLiteClient(t, sqlitePath(t), "wm:", clock)
	ctx := context.Background()

	if err := client.Set(ctx, "short", "v", 60); err != nil {
		t.Fatalf("set: %v", err)
	}
	if err := client.Set(ctx, "forever", "v", 0); err != nil {
		t.Fatalf("set: %v", err)
	}
	clock.Advance(59 * time.Second)
	if _, found, _ := client.Get(ctx, "short"); !found {
		t.Fatal("expected the key before its TTL")
	}

	clock.Advance(time.Second)
	if _, found, err := client.Get(ctx, "short"); err != nil || found {
		t.Fatalf("after TTL: found %v err %v", found, err)
	}
	if _, found, _ := client.Get(ctx, "forever"); !found {
		t.Fatal("a key without TTL must not expire")
	}
	keys, err := client.Scan(ctx, "*")
	if err != nil || !reflect.DeepEqual(keys, []string{"forever"}) {
		t.Fatalf("scan = %v err %v, expired keys must not be listed", keys, err)
	}
	if deleted, err := client.Delete(ctx, "short"); err != nil || deleted != 0 {
		t.Fatalf("delete expired = %d err %v, want 0", deleted, err)
	}
}

func TestGivenAnExistingKeyWhenSetAgainThenValueAndTTLAreReplaced(t *testing.T) {
	clock := &testClock{now: time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC)}
	client := newTestSQLiteClient(t, sqlitePath(t), "wm:", clock)
	ctx := context.Background()

	if err := client.Set(ctx, "k", "old", 10); err != nil {
		t.Fatalf("set: %v", err)
	}
	if err := client.Set(ctx, "k", "new", 0); err != nil {
		t.Fatalf("set: %v", err)
	}
	clock.Advance(time.Hour)
	value, found, err := client.Get(ctx, "k")
	if err != nil || !found || value != "new" {
		t.Fatalf("get = %q found %v err %v", value, found, err)
	}
}

func TestGivenAGlobWhenScanningThenOnlyMatchingKeysAreReturnedWithoutThePrefix(t *testing.T) {
	client := newTestSQLiteClient(t, sqlitePath(t), "wm:", nil)
	ctx := context.Background()
	for _, key := range []string{"sheets:accounts", "sheets:transactions", "market:VNM", "sheets*literal"} {
		if err := client.Set(ctx, key, "v", 0); err != nil {
			t.Fatalf("set %s: %v", key, err)
		}
	}

	keys, err := client.Scan(ctx, "sheets:*")
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	sort.Strings(keys)
	if !reflect.DeepEqual(keys, []string{"sheets:accounts", "sheets:transactions"}) {
		t.Fatalf("scan = %v", keys)
	}
	if keys, _ := client.Scan(ctx, `sheets\*literal`); !reflect.DeepEqual(keys, []string{"sheets*literal"}) {
		t.Fatalf("escaped scan = %v", keys)
	}
	if keys, _ := client.Scan(ctx, "market:VN?"); !reflect.DeepEqual(keys, []string{"market:VNM"}) {
		t.Fatalf("? scan = %v", keys)
	}
}

func TestGivenTwoPrefixesOnOneFileWhenScanningAndDeletingThenTheyNeverTouchEachOther(t *testing.T) {
	path := sqlitePath(t)
	prod := newTestSQLiteClient(t, path, "wm:", nil)
	test := newTestSQLiteClient(t, path, "wm-test:run[1]*:", nil)
	ctx := context.Background()

	if err := prod.Set(ctx, "sheets:accounts", "prod", 0); err != nil {
		t.Fatalf("set prod: %v", err)
	}
	if err := test.Set(ctx, "sheets:accounts", "test", 0); err != nil {
		t.Fatalf("set test: %v", err)
	}

	keys, err := test.Scan(ctx, "*")
	if err != nil || !reflect.DeepEqual(keys, []string{"sheets:accounts"}) {
		t.Fatalf("test scan = %v err %v", keys, err)
	}
	if deleted, err := test.Delete(ctx, keys...); err != nil || deleted != 1 {
		t.Fatalf("delete = %d err %v", deleted, err)
	}
	if value, found, _ := prod.Get(ctx, "sheets:accounts"); !found || value != "prod" {
		t.Fatalf("prod key = %q found %v, must survive the test run's invalidation", value, found)
	}
}

func TestGivenSomeMissingKeysWhenDeletingThenOnlyExistingOnesAreCounted(t *testing.T) {
	client := newTestSQLiteClient(t, sqlitePath(t), "wm:", nil)
	ctx := context.Background()
	if err := client.Set(ctx, "a", "1", 0); err != nil {
		t.Fatalf("set: %v", err)
	}

	if deleted, err := client.Delete(ctx); err != nil || deleted != 0 {
		t.Fatalf("empty delete = %d err %v", deleted, err)
	}
	if deleted, err := client.Delete(ctx, "a", "b"); err != nil || deleted != 1 {
		t.Fatalf("delete = %d err %v", deleted, err)
	}
	if _, found, _ := client.Get(ctx, "a"); found {
		t.Fatal("deleted key still readable")
	}
}

func TestGivenARestartWhenReopeningTheFileThenEntriesPersistAndExpiredOnesArePurged(t *testing.T) {
	path := sqlitePath(t)
	clock := &testClock{now: time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC)}
	first := newTestSQLiteClient(t, path, "wm:", clock)
	ctx := context.Background()
	if err := first.Set(ctx, "kept", "v", 0); err != nil {
		t.Fatalf("set: %v", err)
	}
	if err := first.Set(ctx, "expired", "v", 1); err != nil {
		t.Fatalf("set: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	clock.Advance(time.Minute)
	second := newTestSQLiteClient(t, path, "wm:", clock)
	if value, found, err := second.Get(ctx, "kept"); err != nil || !found || value != "v" {
		t.Fatalf("kept = %q found %v err %v", value, found, err)
	}
	var rows int
	if err := second.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM cache_entries`).Scan(&rows); err != nil || rows != 1 {
		t.Fatalf("rows = %d err %v, want the expired row purged on open", rows, err)
	}
}

func TestGivenConcurrentWritersWhenSettingThenEveryWriteLands(t *testing.T) {
	client := newTestSQLiteClient(t, sqlitePath(t), "wm:", nil)
	ctx := context.Background()
	var wg sync.WaitGroup
	errs := make(chan error, 50)
	for i := range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- client.Set(ctx, "k:"+string(rune('a'+i%26))+string(rune('a'+i/26)), "v", 300)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent set: %v", err)
		}
	}
	keys, err := client.Scan(ctx, "k:*")
	if err != nil || len(keys) != 50 {
		t.Fatalf("scan = %d keys err %v", len(keys), err)
	}
}

func TestGivenAnOpenCacheWhenPingingThenItSucceedsAndFailsAfterClose(t *testing.T) {
	client := newTestSQLiteClient(t, sqlitePath(t), "wm:", nil)
	if err := client.Ping(context.Background()); err != nil {
		t.Fatalf("ping: %v", err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := client.Ping(context.Background()); err == nil {
		t.Fatal("expected ping to fail on a closed cache")
	}
}
