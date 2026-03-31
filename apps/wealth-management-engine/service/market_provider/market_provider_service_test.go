package market_provider

import (
	"apps/wealth-management-engine/domain"
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
