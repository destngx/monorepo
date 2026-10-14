package cache

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"fmt"
)

// NewBackend opens the cache client the config selects.
func NewBackend(config domain.CacheConfig, log *logger.Logger) (port.CacheClient, error) {
	switch config.Backend {
	case domain.CacheBackendUpstash:
		return NewClient(config, log)
	case domain.CacheBackendSQLite:
		return NewSQLiteClient(config, log)
	default:
		return nil, fmt.Errorf("unknown cache backend %q", config.Backend)
	}
}
