package service

import (
	"apps/wealth-management-engine/domain"
	"apps/wealth-management-engine/port"
	"context"
	"fmt"
	"strings"
)

type marketProviderService struct {
	providers map[string]port.MarketProvider
}

func NewMarketProviderService(providers ...port.MarketProvider) port.MarketProviderService {
	registry := make(map[string]port.MarketProvider, len(providers))
	for _, provider := range providers {
		registry[strings.ToLower(provider.Name())] = provider
	}

	return &marketProviderService{providers: registry}
}

func (s *marketProviderService) Health(ctx context.Context, provider string) (domain.MarketProviderHealth, error) {
	key := strings.ToLower(strings.TrimSpace(provider))
	selected, ok := s.providers[key]
	if !ok {
		return domain.MarketProviderHealth{}, fmt.Errorf("unknown market provider: %s", provider)
	}

	return selected.Health(ctx)
}
