package market_provider

import (
	"apps/wealth-management-engine/domain"
	"context"
	"fmt"
	"strings"
)

func (s *marketProviderService) GetPriceSeries(
	ctx context.Context,
	symbol string,
	seriesType domain.SeriesType,
) (domain.PriceSeries, error) {
	cacheKey := fmt.Sprintf("market:getPriceSeries:%s:%s", seriesType, symbol)
	if !domain.ShouldBypassCache(ctx) {
		if cached, ok, err := s.getCachedPriceSeries(ctx, cacheKey); err == nil && ok {
			return cached, nil
		}
	}

	chain := s.routing.GetPriceSeries[seriesType]
	if len(chain) == 0 {
		return domain.PriceSeries{}, fmt.Errorf("no provider chain configured for series type %s", seriesType)
	}

	var errors []string
	for _, name := range chain {
		provider, err := s.providerByName(name)
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}

		series, err := provider.GetPriceSeries(ctx, symbol, seriesType)
		if err == nil {
			_ = s.setCache(ctx, cacheKey, series)
			return series, nil
		}
		errors = append(errors, fmt.Sprintf("%s: %v", provider.Name(), err))
	}

	return domain.PriceSeries{}, fmt.Errorf("get price series failed for %s/%s: %s", seriesType, symbol, strings.Join(errors, "; "))
}
