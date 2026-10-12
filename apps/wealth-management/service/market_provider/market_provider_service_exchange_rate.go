package market_provider

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"fmt"
	"strings"
	"time"
)

// maxExchangeRateAge rejects rates a feed stopped updating (fmarket's USD/VND
// history froze in 2025) so the chain falls through; 4 days covers weekends
// and holidays for daily reference rates.
const maxExchangeRateAge = 4 * 24 * time.Hour

func (s *marketProviderService) GetExchangeRate(ctx context.Context, from string, to string) (domain.ExchangeRate, error) {
	from, to = strings.ToUpper(strings.TrimSpace(from)), strings.ToUpper(strings.TrimSpace(to))
	pair := from + "_" + to
	key := "market:getExchangeRate:" + pair
	return cachedRoute(ctx, s, key, s.routing.CacheTTL.ExchangeRate, "get exchange rate "+pair, s.routing.GetExchangeRate[pair], func(provider port.MarketProvider) (domain.ExchangeRate, error) {
		capable, ok := provider.(port.ExchangeRateProvider)
		if !ok {
			return domain.ExchangeRate{}, errCapabilityMissing
		}
		rate, err := capable.GetExchangeRate(ctx, from, to)
		if err != nil {
			return domain.ExchangeRate{}, err
		}
		return rate, s.checkRate(rate)
	})
}

func (s *marketProviderService) checkRate(rate domain.ExchangeRate) error {
	if rate.Rate <= 0 {
		return fmt.Errorf("returned a non-positive rate")
	}
	at, err := time.Parse(time.RFC3339, rate.Timestamp)
	if err != nil {
		return fmt.Errorf("returned a rate without a readable timestamp %q", rate.Timestamp)
	}
	if age := s.now().Sub(at); age > maxExchangeRateAge {
		return fmt.Errorf("rate is stale: last updated %s", rate.Timestamp)
	}
	return nil
}
