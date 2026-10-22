package marketdata

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

var _ port.MarketProviderService = (*Client)(nil)

// Health checks one source on the server (vnstock, fmarket, yahoo,
// coingecko, currencyapi or algolab).
func (c *Client) Health(ctx context.Context, provider string) (domain.MarketProviderHealth, error) {
	name := strings.ToLower(strings.TrimSpace(provider))
	if name == "" {
		return domain.MarketProviderHealth{}, fmt.Errorf("%w: provider is required", domain.ErrValidation)
	}
	health, _, err := get[domain.MarketProviderHealth](ctx, c, "/api/v2/providers/"+url.PathEscape(name)+"/health", nil)
	return health, err
}

// Ping checks that the server answers, for the engine's health checks.
func (c *Client) Ping(ctx context.Context) error {
	_, _, err := get[[]map[string]any](ctx, c, "/api/v2/providers", nil)
	return err
}

func (c *Client) GetTicker(ctx context.Context, symbol string, tickerType domain.TickerType) (domain.Ticker, error) {
	ticker, meta, err := get[domain.Ticker](ctx, c, "/api/v2/ticker", url.Values{"symbol": {symbol}, "type": {string(tickerType)}})
	if err != nil {
		return domain.Ticker{}, err
	}
	ticker.Source = meta.source()
	return ticker, nil
}

func (c *Client) GetExchangeRate(ctx context.Context, from string, to string) (domain.ExchangeRate, error) {
	rate, meta, err := get[domain.ExchangeRate](ctx, c, "/api/v2/fx", url.Values{"from": {from}, "to": {to}})
	if err != nil {
		return domain.ExchangeRate{}, err
	}
	rate.Source = meta.source()
	return rate, nil
}

func (c *Client) GetPriceSeries(ctx context.Context, symbol string, seriesType domain.SeriesType) (domain.PriceSeries, error) {
	series, meta, err := get[domain.PriceSeries](ctx, c, "/api/v2/series", url.Values{"symbol": {symbol}, "type": {string(seriesType)}})
	if err != nil {
		return domain.PriceSeries{}, err
	}
	series.Source = meta.source()
	return series, nil
}

func (c *Client) GetBankInterestRate(ctx context.Context) (domain.BankRates, error) {
	rates, meta, err := get[[]domain.BankRate](ctx, c, "/api/v2/bank-rates", nil)
	if err != nil {
		return domain.BankRates{}, err
	}
	return domain.BankRates{Rates: rates, Source: meta.source()}, nil
}

// GetFlexibleSavings returns the flexible savings products; a provider the
// server could not read is listed in Failed.
func (c *Client) GetFlexibleSavings(ctx context.Context) (domain.FlexibleSavings, error) {
	savings, meta, err := get[domain.FlexibleSavings](ctx, c, "/api/v2/flexible-savings", nil)
	if err != nil {
		return domain.FlexibleSavings{}, err
	}
	if savings.Products == nil {
		savings.Products = []domain.FlexibleSaving{}
	}
	if savings.Failed == nil {
		savings.Failed = []string{}
	}
	savings.Source = meta.source()
	return savings, nil
}

// GetCandles returns about lookbackDays of bars ending now; the server
// builds 4h bars from 1h bars. Source lists the sources that failed before
// the one that answered (WM-INV-15).
func (c *Client) GetCandles(ctx context.Context, symbol string, timeframe domain.Timeframe, lookbackDays int) (domain.CandleSeries, error) {
	query := url.Values{"symbol": {symbol}, "timeframe": {string(timeframe)}}
	if lookbackDays > 0 {
		query.Set("days", strconv.Itoa(lookbackDays))
	}
	series, meta, err := get[domain.CandleSeries](ctx, c, "/api/v2/candles", query)
	if err != nil {
		return domain.CandleSeries{}, err
	}
	series.Source = meta.source()
	return series, nil
}
