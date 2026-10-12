package fmarket

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var _ port.CandleProvider = (*Provider)(nil)

// GetCandles serves daily (and weekly, aggregated) gold history: SJC bars
// use the SJC sell price in VND, XAUUSD the world price in USD per ounce.
// fmarket publishes one quote a day, so each bar is a single close.
func (p *Provider) GetCandles(ctx context.Context, symbol string, query domain.CandleQuery) (domain.CandleSeries, error) {
	if symbol != domain.MarketSymbolSJC && symbol != domain.MarketSymbolXAUUSD {
		return domain.CandleSeries{}, fmt.Errorf("%w: fmarket has no %s history", port.ErrUnsupported, symbol)
	}
	if query.Timeframe != domain.Timeframe1D && query.Timeframe != domain.Timeframe1W {
		return domain.CandleSeries{}, fmt.Errorf("%w: fmarket gold history is daily", port.ErrUnsupported)
	}
	body := map[string]any{"fromDate": fmarketDate(query.From), "toDate": fmarketDate(query.To), "isAllData": false}
	response, err := p.do(ctx, http.MethodPost, "/res/get-price-gold-history", body)
	if err != nil {
		return domain.CandleSeries{}, err
	}
	defer response.Body.Close()

	var envelope fmarketEnvelope
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		return domain.CandleSeries{}, fmt.Errorf("decode fmarket gold history: %w", err)
	}
	var rows []goldRow
	if err := json.Unmarshal(envelope.Data, &rows); err != nil {
		return domain.CandleSeries{}, fmt.Errorf("decode fmarket gold rows: %w", err)
	}

	series := domain.CandleSeries{Symbol: symbol, Timeframe: query.Timeframe, Provider: p.Name(), Currency: "USD"}
	if symbol == domain.MarketSymbolSJC {
		series.Currency = "VND"
	}
	candles := make([]domain.Candle, 0, len(rows))
	for _, row := range rows {
		value := row.Price
		if symbol == domain.MarketSymbolSJC {
			value = row.AskSJC
		}
		candles = append(candles, domain.Candle{Time: time.UnixMilli(row.ReportDate).UTC(), Close: value})
	}
	series.Candles = domain.CleanCandles(candles)
	if query.Timeframe == domain.Timeframe1W {
		series.Candles = domain.AggregateCandles(series.Candles, domain.Timeframe1W, vietnamTime)
	}
	return series, nil
}
