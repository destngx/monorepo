package port

import (
	"apps/wealth-management/domain"
	"context"
	"errors"
)

// ErrUnsupported marks a request a provider does not cover (a symbol, pair
// or resolution outside its upstream). The service skips such providers
// without counting them as a failed fallback step.
var ErrUnsupported = errors.New("not covered by this provider")

// MarketProvider is the base every market data source implements. Each data
// capability is a separate interface so a provider only implements what its
// upstream actually serves; the service type-asserts per call.
type MarketProvider interface {
	Name() string
	Health(ctx context.Context) (domain.MarketProviderHealth, error)
}

type TickerProvider interface {
	MarketProvider
	GetTicker(ctx context.Context, symbol string, tickerType domain.TickerType) (domain.Ticker, error)
}

type ExchangeRateProvider interface {
	MarketProvider
	GetExchangeRate(ctx context.Context, from string, to string) (domain.ExchangeRate, error)
}

type PriceSeriesProvider interface {
	MarketProvider
	GetPriceSeries(ctx context.Context, symbol string, seriesType domain.SeriesType) (domain.PriceSeries, error)
}

type BankRateProvider interface {
	MarketProvider
	GetBankInterestRate(ctx context.Context) ([]domain.BankRate, error)
}

// CandleProvider serves OHLCV bars for one source timeframe (1h, 1d or 1w).
type CandleProvider interface {
	MarketProvider
	GetCandles(ctx context.Context, symbol string, query domain.CandleQuery) (domain.CandleSeries, error)
}

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
