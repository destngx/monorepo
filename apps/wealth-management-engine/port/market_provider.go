package port

import (
	"apps/wealth-management-engine/domain"
	"context"
)

type MarketProvider interface {
	Name() string
	Health(ctx context.Context) (domain.MarketProviderHealth, error)
	GetTicker(ctx context.Context, symbol string, tickerType domain.TickerType) (domain.Ticker, error)
	GetExchangeRate(ctx context.Context, from string, to string) (domain.ExchangeRate, error)
	GetPriceSeries(ctx context.Context, symbol string, seriesType domain.SeriesType) (domain.PriceSeries, error)
	GetBankInterestRate(ctx context.Context) ([]domain.BankRate, error)
}

type MarketProviderService interface {
	Health(ctx context.Context, provider string) (domain.MarketProviderHealth, error)
	GetTicker(ctx context.Context, symbol string, tickerType domain.TickerType) (domain.Ticker, error)
	GetExchangeRate(ctx context.Context, from string, to string) (domain.ExchangeRate, error)
	GetPriceSeries(ctx context.Context, symbol string, seriesType domain.SeriesType) (domain.PriceSeries, error)
	GetBankInterestRate(ctx context.Context) ([]domain.BankRate, error)
}
