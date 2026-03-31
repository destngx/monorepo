package cache

import (
	"apps/wealth-management-engine/domain"
	"apps/wealth-management-engine/port"
	"context"
)

type cacheService struct {
	client port.CacheClient
}

func NewCacheService(client port.CacheClient) port.CacheService {
	return &cacheService{client: client}
}

func New(client port.CacheClient) port.CacheService {
	return NewCacheService(client)
}

func (s *cacheService) Set(ctx context.Context, key string, value string, ttlSeconds int) error {
	return s.client.Set(ctx, key, value, ttlSeconds)
}

func (s *cacheService) Get(ctx context.Context, key string) (domain.CacheEntry, error) {
	value, found, err := s.client.Get(ctx, key)
	if err != nil {
		return domain.CacheEntry{}, err
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
		return 0, err
	}

	deleted := 0
	for _, key := range keys {
		if err := s.client.Delete(ctx, key); err != nil {
			return deleted, err
		}
		deleted++
	}

	return deleted, nil
}

func (s *cacheService) Ping(ctx context.Context) error {
	return s.client.Ping(ctx)
}
