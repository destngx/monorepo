package market_provider

import (
	"apps/wealth-management-engine/domain"
	"context"
	"fmt"
	"strings"
)

func (s *marketProviderService) GetExchangeRate(ctx context.Context, from string, to string) (domain.ExchangeRate, error) {
	pair := strings.ToUpper(from) + "_" + strings.ToUpper(to)
	cacheKey := fmt.Sprintf("market:getExchangeRate:%s", pair)
	if !domain.ShouldBypassCache(ctx) {
		if cached, ok, err := s.getCachedExchangeRate(ctx, cacheKey); err == nil && ok {
			return cached, nil
		}
	}

	chain := s.routing.GetExchangeRate[pair]
	if len(chain) == 0 {
		return domain.ExchangeRate{}, fmt.Errorf("no provider chain configured for exchange rate pair %s", pair)
	}

	var errors []string
	for _, name := range chain {
		provider, err := s.providerByName(name)
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}

		exchangeRate, err := provider.GetExchangeRate(ctx, from, to)
		if err == nil {
			_ = s.setCache(ctx, cacheKey, exchangeRate)
			return exchangeRate, nil
		}
		errors = append(errors, fmt.Sprintf("%s: %v", provider.Name(), err))
	}

	return domain.ExchangeRate{}, fmt.Errorf("get exchange rate failed for %s: %s", pair, strings.Join(errors, "; "))
}
