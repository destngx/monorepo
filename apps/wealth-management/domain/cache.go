package domain

// CacheBackend names the store behind the cache port.
type CacheBackend string

const (
	// CacheBackendUpstash is the shared Upstash Redis REST store.
	CacheBackendUpstash CacheBackend = "upstash"
	// CacheBackendSQLite is a local SQLite file, used when Upstash is not
	// configured so the engine runs without any cache variables.
	CacheBackendSQLite CacheBackend = "sqlite"
)

type CacheConfig struct {
	Backend   CacheBackend
	RESTURL   string
	RESTToken string
	// SQLitePath is the database file of the SQLite backend.
	SQLitePath string
	// KeyPrefix namespaces every key, e.g. "wm:" in production and a unique
	// prefix per live test run (WM-OPS-2b, WM-TEST-4).
	KeyPrefix string
}
