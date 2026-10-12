package market_provider

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"fmt"
)

func (s *marketProviderService) GetTicker(ctx context.Context, symbol string, tickerType domain.TickerType) (domain.Ticker, error) {
	symbol = domain.NormaliseMarketSymbol(symbol)
	if symbol == "" {
		return domain.Ticker{}, fmt.Errorf("%w: symbol is required", domain.ErrValidation)
	}
	key := fmt.Sprintf("market:getTicker:%s:%s", tickerType, symbol)
	operation := fmt.Sprintf("get ticker %s/%s", tickerType, symbol)
	return cachedRoute(ctx, s, key, operation, s.routing.GetTicker[tickerType], func(provider port.MarketProvider) (domain.Ticker, error) {
		capable, ok := provider.(port.TickerProvider)
		if !ok {
			return domain.Ticker{}, errCapabilityMissing
		}
		return capable.GetTicker(ctx, symbol, tickerType)
	})
}
