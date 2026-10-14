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
// coingecko or currencyapi).
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
	ticker, _, err := get[domain.Ticker](ctx, c, "/api/v2/ticker", url.Values{"symbol": {symbol}, "type": {string(tickerType)}})
	return ticker, err
}

func (c *Client) GetExchangeRate(ctx context.Context, from string, to string) (domain.ExchangeRate, error) {
	rate, _, err := get[domain.ExchangeRate](ctx, c, "/api/v2/fx", url.Values{"from": {from}, "to": {to}})
	return rate, err
}

func (c *Client) GetPriceSeries(ctx context.Context, symbol string, seriesType domain.SeriesType) (domain.PriceSeries, error) {
	series, _, err := get[domain.PriceSeries](ctx, c, "/api/v2/series", url.Values{"symbol": {symbol}, "type": {string(seriesType)}})
	return series, err
}

func (c *Client) GetBankInterestRate(ctx context.Context) ([]domain.BankRate, error) {
	rates, _, err := get[[]domain.BankRate](ctx, c, "/api/v2/bank-rates", nil)
	return rates, err
}

// GetCandles returns about lookbackDays of bars ending now; the server
// builds 4h bars from 1h bars. Fallback lists the sources that failed
// before the one that answered (WM-INV-15).
func (c *Client) GetCandles(ctx context.Context, symbol string, timeframe domain.Timeframe, lookbackDays int) (domain.CandleSeries, error) {
	query := url.Values{"symbol": {symbol}, "timeframe": {string(timeframe)}}
	if lookbackDays > 0 {
		query.Set("days", strconv.Itoa(lookbackDays))
	}
	series, meta, err := get[domain.CandleSeries](ctx, c, "/api/v2/candles", query)
	if err != nil {
		return domain.CandleSeries{}, err
	}
	series.Fallback = strings.Join(meta.Fallback, "; ")
	return series, nil
}
