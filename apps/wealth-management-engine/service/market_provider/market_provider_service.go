package market_provider

import (
	"apps/wealth-management-engine/domain"
	"apps/wealth-management-engine/port"
	"context"
	"fmt"
	"strings"
)

type marketProviderService struct {
	providers map[string]port.MarketProvider
	routing   domain.MarketRoutingConfig
	cache     port.CacheClient
}

func NewService(providers ...port.MarketProvider) port.MarketProviderService {
	return NewServiceWithRouting(domain.DefaultMarketRoutingConfig(), nil, providers...)
}

func NewServiceWithRouting(
	routing domain.MarketRoutingConfig,
	cache port.CacheClient,
	providers ...port.MarketProvider,
) port.MarketProviderService {
	registry := make(map[string]port.MarketProvider, len(providers))
	for _, provider := range providers {
		registry[strings.ToLower(provider.Name())] = provider
	}

	return &marketProviderService{
		providers: registry,
		routing:   routing,
		cache:     cache,
	}
}

func NewMarketProviderService(providers ...port.MarketProvider) port.MarketProviderService {
	return NewService(providers...)
}

func NewMarketProviderServiceWithRouting(
	routing domain.MarketRoutingConfig,
	cache port.CacheClient,
	providers ...port.MarketProvider,
) port.MarketProviderService {
	return NewServiceWithRouting(routing, cache, providers...)
}

func (s *marketProviderService) Health(ctx context.Context, provider string) (domain.MarketProviderHealth, error) {
	selected, err := s.providerByName(provider)
	if err != nil {
		return domain.MarketProviderHealth{}, err
	}

	return selected.Health(ctx)
}

func (s *marketProviderService) providerByName(provider string) (port.MarketProvider, error) {
	key := strings.ToLower(strings.TrimSpace(provider))
	selected, ok := s.providers[key]
	if !ok {
		return nil, fmt.Errorf("unknown market provider: %s", provider)
	}

	return selected, nil
}
