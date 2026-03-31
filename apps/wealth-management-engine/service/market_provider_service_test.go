package service

import (
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

	service := NewMarketProviderServiceWithRouting(
		domain.MarketRoutingConfig{
			GetTicker: map[domain.TickerType][]string{
				domain.TickerTypeEquity: {"fmarket", "vnstock"},
			},
			CacheTTLSeconds: 300,
		},
		nil,
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

	service := NewMarketProviderServiceWithRouting(
		domain.MarketRoutingConfig{
			GetTicker: map[domain.TickerType][]string{
				domain.TickerTypeEquity: {"vnstock"},
			},
			CacheTTLSeconds: 300,
		},
		cache,
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

	service := NewMarketProviderServiceWithRouting(
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
	service := NewMarketProviderServiceWithRouting(
		domain.MarketRoutingConfig{
			GetTicker: map[domain.TickerType][]string{
				domain.TickerTypeEquity: {"vnstock"},
			},
			CacheTTLSeconds: 300,
		},
		cache,
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

type fakeMarketProvider struct {
	name            string
	ticker          domain.Ticker
	tickerErr       error
	exchangeRate    domain.ExchangeRate
	exchangeRateErr error
	priceSeries     domain.PriceSeries
	priceSeriesErr  error
	bankRates       []domain.BankRate
	bankRatesErr    error
	health          domain.MarketProviderHealth
	healthErr       error
	getTickerCalls  int
}

func (f *fakeMarketProvider) Name() string {
	return f.name
}

func (f *fakeMarketProvider) Health(_ context.Context) (domain.MarketProviderHealth, error) {
	if f.health.Provider == "" {
		f.health.Provider = f.name
	}
	if f.health.Status == "" {
		f.health.Status = "ok"
	}
	return f.health, f.healthErr
}

func (f *fakeMarketProvider) GetTicker(_ context.Context, _ string, _ domain.TickerType) (domain.Ticker, error) {
	f.getTickerCalls++
	return f.ticker, f.tickerErr
}

func (f *fakeMarketProvider) GetExchangeRate(_ context.Context, _, _ string) (domain.ExchangeRate, error) {
	return f.exchangeRate, f.exchangeRateErr
}

func (f *fakeMarketProvider) GetPriceSeries(_ context.Context, _ string, _ domain.SeriesType) (domain.PriceSeries, error) {
	return f.priceSeries, f.priceSeriesErr
}

func (f *fakeMarketProvider) GetBankInterestRate(_ context.Context) ([]domain.BankRate, error) {
	return f.bankRates, f.bankRatesErr
}

type fakeMarketCacheClient struct {
	store map[string]string
}

func (f *fakeMarketCacheClient) Set(_ context.Context, key string, value string, _ int) error {
	f.store[key] = value
	return nil
}

func (f *fakeMarketCacheClient) Get(_ context.Context, key string) (string, bool, error) {
	value, ok := f.store[key]
	return value, ok, nil
}

func (f *fakeMarketCacheClient) Keys(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}

func (f *fakeMarketCacheClient) Delete(_ context.Context, key string) error {
	delete(f.store, key)
	return nil
}

func (f *fakeMarketCacheClient) Ping(_ context.Context) error {
	return nil
}
