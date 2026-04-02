package market_provider

import (
	"apps/wealth-management-engine/adapter/logger"
	"apps/wealth-management-engine/domain"
	"apps/wealth-management-engine/port"
	"context"
	"fmt"
	"log/slog"
	"strings"
)

type marketProviderService struct {
	providers map[string]port.MarketProvider
	routing   domain.MarketRoutingConfig
	cache     port.CacheClient
	log       *logger.Logger
}

func NewService(log *logger.Logger, providers ...port.MarketProvider) port.MarketProviderService {
	return NewServiceWithRouting(domain.DefaultMarketRoutingConfig(), nil, log, providers...)
}

func NewServiceWithRouting(
	routing domain.MarketRoutingConfig,
	cache port.CacheClient,
	log *logger.Logger,
	providers ...port.MarketProvider,
) port.MarketProviderService {
	log.LogApplicationEvent(context.Background(), "initializing market provider service",
		slog.Int("provider_count", len(providers)),
	)

	registry := make(map[string]port.MarketProvider, len(providers))
	for _, provider := range providers {
		registry[strings.ToLower(provider.Name())] = provider
	}

	return &marketProviderService{
		providers: registry,
		routing:   routing,
		cache:     cache,
		log:       log,
	}
}

func NewMarketProviderService(log *logger.Logger, providers ...port.MarketProvider) port.MarketProviderService {
	return NewService(log, providers...)
}

func NewMarketProviderServiceWithRouting(
	routing domain.MarketRoutingConfig,
	cache port.CacheClient,
	log *logger.Logger,
	providers ...port.MarketProvider,
) port.MarketProviderService {
	return NewServiceWithRouting(routing, cache, log, providers...)
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
