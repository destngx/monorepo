package vnstock

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// vnstock reports bar times as naive Vietnam local time.
var vietnamTime = time.FixedZone("ICT", 7*60*60)

var resolutions = map[domain.Timeframe]string{
	domain.Timeframe1H: "1H",
	domain.Timeframe1D: "1D",
	domain.Timeframe1W: "1W",
}

func (p *Provider) GetCandles(ctx context.Context, symbol string, query domain.CandleQuery) (domain.CandleSeries, error) {
	if !domain.IsVNListing(symbol) {
		return domain.CandleSeries{}, fmt.Errorf("%w: %s is not a VN listing", port.ErrUnsupported, symbol)
	}
	resolution, ok := resolutions[query.Timeframe]
	if !ok {
		return domain.CandleSeries{}, fmt.Errorf("%w: vnstock does not serve %s candles", port.ErrUnsupported, query.Timeframe)
	}
	params := url.Values{
		"symbol":     {symbol},
		"start_date": {query.From.In(vietnamTime).Format(time.DateOnly)},
		"end_date":   {query.To.In(vietnamTime).Format(time.DateOnly)},
		"resolution": {resolution},
	}
	response, err := p.do(ctx, "/api/v1/stocks/historical?"+params.Encode())
	if err != nil {
		return domain.CandleSeries{}, err
	}
	defer response.Body.Close()

	var payload struct {
		Success bool `json:"success"`
		Data    []struct {
			Time   string `json:"time"`
			Open   any    `json:"open"`
			High   any    `json:"high"`
			Low    any    `json:"low"`
			Close  any    `json:"close"`
			Volume any    `json:"volume"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return domain.CandleSeries{}, fmt.Errorf("decode vnstock historical: %w", err)
	}
	if !payload.Success {
		return domain.CandleSeries{}, fmt.Errorf("vnstock historical request did not succeed")
	}

	candles := make([]domain.Candle, 0, len(payload.Data))
	for _, row := range payload.Data {
		at, err := parseBarTime(row.Time)
		if err != nil {
			return domain.CandleSeries{}, err
		}
		candles = append(candles, domain.Candle{
			Time:   at,
			Open:   parseFloat(row.Open),
			High:   parseFloat(row.High),
			Low:    parseFloat(row.Low),
			Close:  parseFloat(row.Close),
			Volume: parseFloat(row.Volume),
		})
	}
	return domain.CandleSeries{Symbol: symbol, Timeframe: query.Timeframe, Provider: p.Name(), Currency: "VND", Candles: candles}, nil
}

func parseBarTime(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	for _, layout := range []string{time.DateTime, time.DateOnly, time.RFC3339} {
		if at, err := time.ParseInLocation(layout, raw, vietnamTime); err == nil {
			return at, nil
		}
	}
	return time.Time{}, fmt.Errorf("vnstock returned an unreadable bar time %q", raw)
}
