package cache

import (
	"apps/wealth-management/adapter/logger"
	"context"
	"errors"
	"testing"
)

func TestGivenPatternWhenInvalidateThenDeletesAllMatchedKeys(t *testing.T) {
	client := &fakeCacheClient{
		keysToReturn: []string{"ledger:1", "ledger:2"},
	}
	testLog := logger.NewTestLogger(t)
	cacheService := NewCacheService(client, testLog)

	deleted, err := cacheService.Invalidate(context.Background(), "ledger:*")
	if err != nil {
		t.Fatalf("expected invalidate to pass, got error: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("expected deleted=2, got %d", deleted)
	}
}

func TestGivenDeleteFailsWhenInvalidateThenReturnsError(t *testing.T) {
	client := &fakeCacheClient{
		keysToReturn: []string{"ledger:1"},
		deleteErr:    errors.New("delete failed"),
	}
	testLog := logger.NewTestLogger(t)
	cacheService := NewCacheService(client, testLog)

	_, err := cacheService.Invalidate(context.Background(), "ledger:*")
	if err == nil {
		t.Fatalf("expected error but got nil")
	}
}

type fakeCacheClient struct {
	keysToReturn []string
	deleteErr    error
}

func (f *fakeCacheClient) Set(_ context.Context, _ string, _ string, _ int) error {
	return nil
}

func (f *fakeCacheClient) Get(_ context.Context, _ string) (string, bool, error) {
	return "", false, nil
}

func (f *fakeCacheClient) Scan(_ context.Context, _ string) ([]string, error) {
	return f.keysToReturn, nil
}

func (f *fakeCacheClient) Delete(_ context.Context, keys ...string) (int, error) {
	if f.deleteErr != nil {
		return 0, f.deleteErr
	}
	return len(keys), nil
}

func (f *fakeCacheClient) Ping(_ context.Context) error {
	return nil
}
