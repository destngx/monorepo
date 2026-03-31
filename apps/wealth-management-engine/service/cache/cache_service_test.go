package cache

import (
	"context"
	"errors"
	"testing"
)

func TestGivenPatternWhenInvalidateThenDeletesAllMatchedKeys(t *testing.T) {
	client := &fakeCacheClient{
		keysToReturn: []string{"ledger:1", "ledger:2"},
	}
	cacheService := NewCacheService(client)

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
	cacheService := NewCacheService(client)

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

func (f *fakeCacheClient) Keys(_ context.Context, _ string) ([]string, error) {
	return f.keysToReturn, nil
}

func (f *fakeCacheClient) Delete(_ context.Context, _ string) error {
	return f.deleteErr
}

func (f *fakeCacheClient) Ping(_ context.Context) error {
	return nil
}
