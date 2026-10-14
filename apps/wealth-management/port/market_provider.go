package port

import (
	"apps/wealth-management/domain"
	"context"
)

type MarketProviderService interface {
	Health(ctx context.Context, provider string) (domain.MarketProviderHealth, error)
	GetTicker(ctx context.Context, symbol string, tickerType domain.TickerType) (domain.Ticker, error)
	GetExchangeRate(ctx context.Context, from string, to string) (domain.ExchangeRate, error)
	GetPriceSeries(ctx context.Context, symbol string, seriesType domain.SeriesType) (domain.PriceSeries, error)
	GetBankInterestRate(ctx context.Context) ([]domain.BankRate, error)
	// GetCandles returns about lookbackDays of bars ending now; 4h bars are
	// aggregated from 1h bars.
	GetCandles(ctx context.Context, symbol string, timeframe domain.Timeframe, lookbackDays int) (domain.CandleSeries, error)
}
