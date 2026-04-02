package cache

import (
	"apps/wealth-management-engine/adapter/logger"
	"apps/wealth-management-engine/domain"
	"apps/wealth-management-engine/port"
	"context"
	"log/slog"
)

type cacheService struct {
	client port.CacheClient
	log    *logger.Logger
}

func NewCacheService(client port.CacheClient, log *logger.Logger) port.CacheService {
	log.LogApplicationEvent(context.Background(), "initializing cache service")
	return &cacheService{client: client, log: log}
}

func New(client port.CacheClient, log *logger.Logger) port.CacheService {
	return NewCacheService(client, log)
}

func (s *cacheService) Set(ctx context.Context, key string, value string, ttlSeconds int) error {
	err := s.client.Set(ctx, key, value, ttlSeconds)
	if err != nil {
		s.log.LogError(ctx, "cache set failed", err,
			slog.String("key", key),
			slog.Int("ttl_seconds", ttlSeconds),
		)
	}
	return err
}

func (s *cacheService) Get(ctx context.Context, key string) (domain.CacheEntry, error) {
	value, found, err := s.client.Get(ctx, key)
	if err != nil {
		s.log.LogError(ctx, "cache get failed", err, slog.String("key", key))
		return domain.CacheEntry{}, err
	}

	if !found {
		s.log.LogApplicationEvent(ctx, "cache miss", slog.String("key", key))
	}

	return domain.CacheEntry{
		Key:   key,
		Value: value,
		Found: found,
	}, nil
}

func (s *cacheService) Invalidate(ctx context.Context, pattern string) (int, error) {
	keys, err := s.client.Keys(ctx, pattern)
	if err != nil {
		s.log.LogError(ctx, "cache invalidate failed", err, slog.String("pattern", pattern))
		return 0, err
	}

	deleted := 0
	for _, key := range keys {
		if err := s.client.Delete(ctx, key); err != nil {
			s.log.LogError(ctx, "cache delete failed", err, slog.String("key", key))
			return deleted, err
		}
		deleted++
	}

	s.log.LogApplicationEvent(ctx, "cache invalidated",
		slog.String("pattern", pattern),
		slog.Int("keys_deleted", deleted),
	)
	return deleted, nil
}

func (s *cacheService) Ping(ctx context.Context) error {
	return s.client.Ping(ctx)
}
