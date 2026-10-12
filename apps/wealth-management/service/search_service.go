package service

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

const searchCachePrefix = "search:"

// SearchService tries each provider in order until one returns results and
// caches the answer for 14 days (WM-AI-12).
type SearchService struct {
	providers []port.SearchProvider
	cache     port.CacheService
	log       *logger.Logger
}

var _ port.SearchService = (*SearchService)(nil)

// NewSearchService takes providers in priority order; cache may be nil.
func NewSearchService(cache port.CacheService, log *logger.Logger, providers ...port.SearchProvider) *SearchService {
	return &SearchService{providers: providers, cache: cache, log: log}
}

func (s *SearchService) Providers() []string {
	names := make([]string, 0, len(s.providers))
	for _, provider := range s.providers {
		names = append(names, provider.Name())
	}
	return names
}

func (s *SearchService) Search(ctx context.Context, query string) (domain.SearchResponse, error) {
	normalized := domain.NormalizeSearchQuery(query)
	if normalized == "" {
		return domain.SearchResponse{}, fmt.Errorf("%w: query is required", domain.ErrValidation)
	}
	if len(s.providers) == 0 {
		return domain.SearchResponse{}, errors.New("no web search provider is configured")
	}
	key := searchCacheKey(normalized)
	if !domain.ShouldBypassCache(ctx) {
		if cached, ok := s.read(ctx, key); ok {
			return cached, nil
		}
	}

	var failures []string
	for _, provider := range s.providers {
		results, err := provider.Search(ctx, strings.Join(strings.Fields(query), " "), domain.SearchResultLimit)
		if err == nil && len(results) == 0 {
			err = errors.New("no results")
		}
		if err != nil {
			if ctx.Err() != nil {
				return domain.SearchResponse{}, ctx.Err()
			}
			failures = append(failures, fmt.Sprintf("%s: %v", provider.Name(), err))
			continue
		}
		response := domain.SearchResponse{Query: normalized, Provider: provider.Name(), Results: results, Fallback: strings.Join(failures, "; ")}
		if len(failures) > 0 {
			s.log.LogApplicationEvent(ctx, "web search failed over", slog.String("provider", provider.Name()), slog.String("fallback", response.Fallback))
		}
		s.write(ctx, key, response)
		return response, nil
	}
	return domain.SearchResponse{}, fmt.Errorf("every web search provider failed: %s", strings.Join(failures, "; "))
}

func (s *SearchService) read(ctx context.Context, key string) (domain.SearchResponse, bool) {
	if s.cache == nil {
		return domain.SearchResponse{}, false
	}
	entry, err := s.cache.Get(ctx, key)
	if err != nil {
		s.log.LogError(ctx, "web search cache read failed", err)
		return domain.SearchResponse{}, false
	}
	var response domain.SearchResponse
	if !entry.Found || json.Unmarshal([]byte(entry.Value), &response) != nil {
		return domain.SearchResponse{}, false
	}
	return response, true
}

func (s *SearchService) write(ctx context.Context, key string, response domain.SearchResponse) {
	if s.cache == nil {
		return
	}
	payload, err := json.Marshal(response)
	if err != nil {
		s.log.LogError(ctx, "web search cache encode failed", err)
		return
	}
	if err := s.cache.Set(ctx, key, string(payload), domain.SearchTTLSeconds); err != nil {
		s.log.LogError(ctx, "web search cache write failed", err)
	}
}

func searchCacheKey(normalized string) string {
	sum := sha256.Sum256([]byte(normalized))
	return searchCachePrefix + hex.EncodeToString(sum[:12])
}
