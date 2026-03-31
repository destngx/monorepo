package vnstock

import (
	"apps/wealth-management-engine/domain"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

func (p *Provider) GetPriceSeries(ctx context.Context, symbol string, seriesType domain.SeriesType) (domain.PriceSeries, error) {
	if seriesType != domain.SeriesTypeEquity {
		return domain.PriceSeries{}, fmt.Errorf("vnstock supports only %s price series", domain.SeriesTypeEquity)
	}

	response, err := p.do(ctx, fmt.Sprintf("/api/v1/stocks/historical?symbol=%s", url.QueryEscape(symbol)))
	if err != nil {
		return domain.PriceSeries{}, err
	}
	defer response.Body.Close()

	var payload struct {
		Success bool `json:"success"`
		Data    []struct {
			Time  string      `json:"time"`
			Open  interface{} `json:"open"`
			High  interface{} `json:"high"`
			Low   interface{} `json:"low"`
			Close interface{} `json:"close"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return domain.PriceSeries{}, err
	}
	if !payload.Success {
		return domain.PriceSeries{}, fmt.Errorf("vnstock historical request did not succeed")
	}

	points := make([]map[string]any, 0, len(payload.Data))
	for _, item := range payload.Data {
		points = append(points, map[string]any{
			"time":  item.Time,
			"open":  parseFloat(item.Open),
			"high":  parseFloat(item.High),
			"low":   parseFloat(item.Low),
			"close": parseFloat(item.Close),
		})
	}

	return domain.PriceSeries{
		Provider:   p.Name(),
		SeriesType: seriesType,
		Currency:   "VND",
		Data:       points,
	}, nil
}
