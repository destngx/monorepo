package domain

type CacheConfig struct {
	RESTURL   string
	RESTToken string
	// KeyPrefix namespaces every key, e.g. "wm:" in production and a unique
	// prefix per live test run (WM-OPS-2b, WM-TEST-4).
	KeyPrefix string
}
