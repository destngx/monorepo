package market_provider

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"fmt"
)

func (s *marketProviderService) GetPriceSeries(
	ctx context.Context,
	symbol string,
	seriesType domain.SeriesType,
) (domain.PriceSeries, error) {
	symbol = domain.NormaliseMarketSymbol(symbol)
	key := fmt.Sprintf("market:getPriceSeries:%s:%s", seriesType, symbol)
	operation := fmt.Sprintf("get price series %s/%s", seriesType, symbol)
	return cachedRoute(ctx, s, key, s.routing.CacheTTL.PriceSeries, operation, s.routing.GetPriceSeries[seriesType], func(provider port.MarketProvider) (domain.PriceSeries, error) {
		capable, ok := provider.(port.PriceSeriesProvider)
		if !ok {
			return domain.PriceSeries{}, errCapabilityMissing
		}
		return capable.GetPriceSeries(ctx, symbol, seriesType)
	})
}
