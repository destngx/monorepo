package port

import (
	"apps/wealth-management/domain"
	"context"
)

type CacheService interface {
	Set(ctx context.Context, key string, value string, ttlSeconds int) error
	Get(ctx context.Context, key string) (domain.CacheEntry, error)
	Invalidate(ctx context.Context, pattern string) (int, error)
	Ping(ctx context.Context) error
}

type CacheClient interface {
	Set(ctx context.Context, key string, value string, ttlSeconds int) error
	Get(ctx context.Context, key string) (string, bool, error)
	// Scan lists keys matching a glob pattern (cursor based, never KEYS).
	Scan(ctx context.Context, pattern string) ([]string, error)
	// Delete removes keys and returns how many existed.
	Delete(ctx context.Context, keys ...string) (int, error)
	Ping(ctx context.Context) error
}
