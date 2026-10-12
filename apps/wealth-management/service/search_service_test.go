package service

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeSearchProvider struct {
	name    string
	results []domain.SearchResult
	err     error
	queries []string
}

func (p *fakeSearchProvider) Name() string { return p.name }

func (p *fakeSearchProvider) Search(_ context.Context, query string, _ int) ([]domain.SearchResult, error) {
	p.queries = append(p.queries, query)
	return p.results, p.err
}

var vnResult = domain.SearchResult{Title: "VN-Index rallies", URL: "https://vnexpress.net/a", Source: "vnexpress.net"}

func TestGivenPrimaryFailsWhenSearchThenFallsBackAndCaches(t *testing.T) {
	tavily := &fakeSearchProvider{name: "tavily", err: errors.New("status 432")}
	exa := &fakeSearchProvider{name: "exa"}
	duck := &fakeSearchProvider{name: "duckduckgo", results: []domain.SearchResult{vnResult}}
	cache := newFakeCache()
	search := NewSearchService(cache, logger.NewTestLogger(t), tavily, exa, duck)

	response, err := search.Search(context.Background(), "  VN-Index   news ")
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if response.Provider != "duckduckgo" || response.Fallback != "tavily: status 432; exa: no results" || response.Query != "vn-index news" {
		t.Fatalf("unexpected response %#v", response)
	}
	if tavily.queries[0] != "VN-Index news" {
		t.Fatalf("providers must get the trimmed query with its case, got %q", tavily.queries[0])
	}

	again, err := search.Search(context.Background(), "vn-index NEWS")
	if err != nil || again.Provider != "duckduckgo" || len(duck.queries) != 1 {
		t.Fatalf("an equivalent query must hit the cache, got %#v %v after %d calls", again, err, len(duck.queries))
	}
	if _, err := search.Search(domain.WithBypassCache(context.Background(), true), "vn-index news"); err != nil || len(duck.queries) != 2 {
		t.Fatalf("force must skip the cache, got %v after %d calls", err, len(duck.queries))
	}
}

func TestGivenEveryProviderFailsWhenSearchThenListsFailures(t *testing.T) {
	search := NewSearchService(nil, logger.NewTestLogger(t),
		&fakeSearchProvider{name: "tavily", err: errors.New("timeout")},
		&fakeSearchProvider{name: "duckduckgo"})
	_, err := search.Search(context.Background(), "gold")
	if err == nil || !strings.Contains(err.Error(), "tavily: timeout; duckduckgo: no results") {
		t.Fatalf("expected every failure listed, got %v", err)
	}
	if _, err := search.Search(context.Background(), "  "); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("an empty query is a validation error, got %v", err)
	}
	if got := search.Providers(); strings.Join(got, ",") != "tavily,duckduckgo" {
		t.Fatalf("unexpected providers %v", got)
	}
}
