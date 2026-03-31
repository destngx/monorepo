package market_provider

import (
	"apps/wealth-management-engine/domain"
	"context"
	"fmt"
	"strings"
)

func (s *marketProviderService) GetTicker(ctx context.Context, symbol string, tickerType domain.TickerType) (domain.Ticker, error) {
	cacheKey := fmt.Sprintf("market:getTicker:%s:%s", tickerType, symbol)
	if !domain.ShouldBypassCache(ctx) {
		if cached, ok, err := s.getCachedTicker(ctx, cacheKey); err == nil && ok {
			return cached, nil
		}
	}

	chain := s.routing.GetTicker[tickerType]
	if len(chain) == 0 {
		return domain.Ticker{}, fmt.Errorf("no provider chain configured for ticker type %s", tickerType)
	}

	var errors []string
	for _, name := range chain {
		provider, err := s.providerByName(name)
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}

		ticker, err := provider.GetTicker(ctx, symbol, tickerType)
		if err == nil {
			_ = s.setCache(ctx, cacheKey, ticker)
			return ticker, nil
		}
		errors = append(errors, fmt.Sprintf("%s: %v", provider.Name(), err))
	}

	return domain.Ticker{}, fmt.Errorf("get ticker failed for %s/%s: %s", tickerType, symbol, strings.Join(errors, "; "))
}
