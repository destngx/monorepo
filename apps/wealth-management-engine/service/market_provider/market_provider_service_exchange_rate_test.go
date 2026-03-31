package market_provider

import (
	"apps/wealth-management-engine/domain"
	"context"
	"testing"
)

func TestGivenCacheMissWhenGetExchangeRateThenProviderResultIsStoredWithCapabilityKey(t *testing.T) {
	cache := &fakeMarketCacheClient{store: map[string]string{}}
	provider := &fakeMarketProvider{
		name: "fmarket",
		exchangeRate: domain.ExchangeRate{
			From:     "USD",
			To:       "VND",
			Rate:     25250.5,
			Provider: "fmarket",
		},
	}

	service := NewServiceWithRouting(
		domain.MarketRoutingConfig{
			GetExchangeRate: map[string][]string{
				"USD_VND": {"fmarket"},
			},
			CacheTTLSeconds: 300,
		},
		cache,
		provider,
	)

	_, err := service.GetExchangeRate(context.Background(), "USD", "VND")
	if err != nil {
		t.Fatalf("expected exchange rate call to succeed, got error: %v", err)
	}

	if _, ok := cache.store["market:getExchangeRate:USD_VND"]; !ok {
		t.Fatalf("expected market exchange-rate cache key to be written")
	}
}
