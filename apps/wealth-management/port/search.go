package port

import (
	"apps/wealth-management/domain"
	"context"
)

// SearchProvider is one web search backend (WM-AI-12).
type SearchProvider interface {
	Name() string
	Search(ctx context.Context, query string, limit int) ([]domain.SearchResult, error)
}

// SearchService runs the provider chain with caching. A context marked with
// domain.WithBypassCache skips the cache read.
type SearchService interface {
	Search(ctx context.Context, query string) (domain.SearchResponse, error)
	Providers() []string
}
