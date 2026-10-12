package market_provider

import (
	"apps/wealth-management/domain"
	"context"
)

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
	candles         domain.CandleSeries
	candlesErr      error
	candleQueries   []domain.CandleQuery
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

func (f *fakeMarketProvider) GetCandles(_ context.Context, _ string, query domain.CandleQuery) (domain.CandleSeries, error) {
	f.candleQueries = append(f.candleQueries, query)
	return f.candles, f.candlesErr
}

// healthOnlyProvider has no data capability, to test that routing skips it.
type healthOnlyProvider struct{ name string }

func (h healthOnlyProvider) Name() string { return h.name }

func (h healthOnlyProvider) Health(context.Context) (domain.MarketProviderHealth, error) {
	return domain.MarketProviderHealth{Provider: h.name, Status: "ok"}, nil
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

func (f *fakeMarketCacheClient) Scan(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}

func (f *fakeMarketCacheClient) Delete(_ context.Context, keys ...string) (int, error) {
	for _, key := range keys {
		delete(f.store, key)
	}
	return len(keys), nil
}

func (f *fakeMarketCacheClient) Ping(_ context.Context) error {
	return nil
}
