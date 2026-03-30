package port

import (
	"apps/wealth-management-engine/domain"
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
	Keys(ctx context.Context, pattern string) ([]string, error)
	Delete(ctx context.Context, key string) error
	Ping(ctx context.Context) error
}
