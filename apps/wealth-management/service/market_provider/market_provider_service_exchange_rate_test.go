package market_provider

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"context"
	"strings"
	"testing"
	"time"
)

func TestGivenCacheMissWhenGetExchangeRateThenProviderResultIsStoredWithCapabilityKey(t *testing.T) {
	cache := &fakeMarketCacheClient{store: map[string]string{}}
	provider := &fakeMarketProvider{
		name: "fmarket",
		exchangeRate: domain.ExchangeRate{
			From:      "USD",
			To:        "VND",
			Rate:      25250.5,
			Provider:  "fmarket",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
	}

	testLog := logger.NewTestLogger(t)
	service := NewServiceWithRouting(
		domain.MarketRoutingConfig{
			GetExchangeRate: map[string][]string{
				"USD_VND": {"fmarket"},
			},
			CacheTTLSeconds: 300,
		},
		cache,
		testLog,
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

func TestGivenStalePrimaryRateWhenGetExchangeRateThenFallsBackToFreshProvider(t *testing.T) {
	now := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	stale := &fakeMarketProvider{name: "fmarket", exchangeRate: domain.ExchangeRate{From: "USD", To: "VND", Rate: 26347, Provider: "fmarket", Timestamp: "2025-10-30T17:00:00Z"}}
	undated := &fakeMarketProvider{name: "yahoo", exchangeRate: domain.ExchangeRate{From: "USD", To: "VND", Rate: 25934, Provider: "yahoo"}}
	fresh := &fakeMarketProvider{name: "currencyapi", exchangeRate: domain.ExchangeRate{From: "USD", To: "VND", Rate: 25894.5, Provider: "currencyapi", Timestamp: "2026-10-09T00:00:00Z"}}
	service := NewServiceWithRouting(
		domain.MarketRoutingConfig{GetExchangeRate: map[string][]string{"USD_VND": {"fmarket", "yahoo", "currencyapi"}}},
		nil,
		logger.NewTestLogger(t),
		stale, undated, fresh,
	).(*marketProviderService)
	service.now = func() time.Time { return now }

	rate, err := service.GetExchangeRate(context.Background(), "usd", "vnd")
	if err != nil {
		t.Fatalf("exchange rate error: %v", err)
	}
	if rate.Provider != "currencyapi" || rate.Rate != 25894.5 {
		t.Fatalf("expected the fresh fallback, got %#v", rate)
	}

	fresh.exchangeRate.Timestamp = "2026-10-01T00:00:00Z"
	_, err = service.GetExchangeRate(context.Background(), "USD", "VND")
	if err == nil || !strings.Contains(err.Error(), "stale") || !strings.Contains(err.Error(), "timestamp") {
		t.Fatalf("expected stale and undated reasons, got %v", err)
	}
}
