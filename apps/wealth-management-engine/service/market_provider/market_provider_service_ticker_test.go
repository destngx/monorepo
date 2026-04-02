package market_provider

import (
	"apps/wealth-management-engine/adapter/logger"
	"apps/wealth-management-engine/domain"
	"context"
	"errors"
	"testing"
)

func TestGivenTickerTypeWhenPrimaryProviderFailsThenFallbackProviderIsUsed(t *testing.T) {
	primary := &fakeMarketProvider{
		name:      "fmarket",
		tickerErr: errors.New("temporary failure"),
	}
	secondary := &fakeMarketProvider{
		name: "vnstock",
		ticker: domain.Ticker{
			Symbol:   "ACB",
			Type:     domain.TickerTypeEquity,
			Provider: "vnstock",
		},
	}

	testLog := logger.NewTestLogger(t)
	service := NewServiceWithRouting(
		domain.MarketRoutingConfig{
			GetTicker: map[domain.TickerType][]string{
				domain.TickerTypeEquity: {"fmarket", "vnstock"},
			},
			CacheTTLSeconds: 300,
		},
		nil,
		testLog,
		primary,
		secondary,
	)

	ticker, err := service.GetTicker(context.Background(), "ACB", domain.TickerTypeEquity)
	if err != nil {
		t.Fatalf("expected fallback to succeed, got error: %v", err)
	}
	if ticker.Provider != "vnstock" {
		t.Fatalf("expected vnstock provider, got %s", ticker.Provider)
	}
	if primary.getTickerCalls != 1 || secondary.getTickerCalls != 1 {
		t.Fatalf("expected one call on each provider, got primary=%d secondary=%d", primary.getTickerCalls, secondary.getTickerCalls)
	}
}

func TestGivenCachedTickerWhenGetTickerThenReturnsFromCacheWithoutProviderCall(t *testing.T) {
	cache := &fakeMarketCacheClient{
		store: map[string]string{
			"market:getTicker:equity:ACB": `{"symbol":"ACB","type":"equity","provider":"cache"}`,
		},
	}
	provider := &fakeMarketProvider{
		name: "vnstock",
		ticker: domain.Ticker{
			Symbol:   "ACB",
			Type:     domain.TickerTypeEquity,
			Provider: "vnstock",
		},
	}

	testLog := logger.NewTestLogger(t)
	service := NewMarketProviderServiceWithRouting(
		domain.MarketRoutingConfig{
			GetTicker: map[domain.TickerType][]string{
				domain.TickerTypeEquity: {"vnstock"},
			},
			CacheTTLSeconds: 300,
		},
		cache,
		testLog,
		provider,
	)

	ticker, err := service.GetTicker(context.Background(), "ACB", domain.TickerTypeEquity)
	if err != nil {
		t.Fatalf("expected cache hit to succeed, got error: %v", err)
	}
	if ticker.Provider != "cache" {
		t.Fatalf("expected cache payload, got provider=%s", ticker.Provider)
	}
	if provider.getTickerCalls != 0 {
		t.Fatalf("expected no provider calls on cache hit, got %d", provider.getTickerCalls)
	}
}

func TestGivenSkipCacheContextWhenGetTickerThenBypassesCachedValue(t *testing.T) {
	cache := &fakeMarketCacheClient{
		store: map[string]string{
			"market:getTicker:equity:ACB": `{"symbol":"ACB","type":"equity","provider":"cache"}`,
		},
	}
	provider := &fakeMarketProvider{
		name: "vnstock",
		ticker: domain.Ticker{
			Symbol:   "ACB",
			Type:     domain.TickerTypeEquity,
			Provider: "vnstock",
		},
	}
	testLog := logger.NewTestLogger(t)
	service := NewMarketProviderServiceWithRouting(
		domain.MarketRoutingConfig{
			GetTicker: map[domain.TickerType][]string{
				domain.TickerTypeEquity: {"vnstock"},
			},
			CacheTTLSeconds: 300,
		},
		cache,
		testLog,
		provider,
	)

	ctx := domain.WithBypassCache(context.Background(), true)
	ticker, err := service.GetTicker(ctx, "ACB", domain.TickerTypeEquity)
	if err != nil {
		t.Fatalf("expected skipCache flow to pass, got error: %v", err)
	}
	if ticker.Provider != "vnstock" {
		t.Fatalf("expected provider result, got %s", ticker.Provider)
	}
	if provider.getTickerCalls != 1 {
		t.Fatalf("expected one provider call, got %d", provider.getTickerCalls)
	}
}
