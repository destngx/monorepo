package fmarket

import (
	"apps/wealth-management-engine/domain"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func (p *Provider) GetPriceSeries(ctx context.Context, symbol string, seriesType domain.SeriesType) (domain.PriceSeries, error) {
	if seriesType != domain.SeriesTypeGoldUSD {
		return domain.PriceSeries{}, fmt.Errorf("fmarket supports only %s for price series", domain.SeriesTypeGoldUSD)
	}

	fromDate, toDate := lastNDaysRange(30)
	body := map[string]any{
		"fromDate":  fromDate,
		"toDate":    toDate,
		"isAllData": false,
	}
	response, err := p.do(ctx, http.MethodPost, "/res/get-price-gold-history", body)
	if err != nil {
		return domain.PriceSeries{}, err
	}
	defer response.Body.Close()

	var envelope fmarketEnvelope
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		return domain.PriceSeries{}, err
	}

	var rows []struct {
		Price      float64 `json:"price"`
		BidSJC     float64 `json:"bidSjc"`
		AskSJC     float64 `json:"askSjc"`
		ReportDate int64   `json:"reportDate"`
	}
	if err := json.Unmarshal(envelope.Data, &rows); err != nil {
		return domain.PriceSeries{}, err
	}
	if len(rows) == 0 {
		return domain.PriceSeries{}, fmt.Errorf("fmarket returned empty gold price series")
	}

	points := make([]domain.PricePoint, 0, len(rows))
	for _, row := range rows {
		points = append(points, domain.PricePoint{
			Price:     row.Price,
			Timestamp: time.UnixMilli(row.ReportDate).UTC().Format(time.RFC3339),
		})
	}
	last := rows[len(rows)-1]

	return domain.PriceSeries{
		Provider:   p.Name(),
		SeriesType: seriesType,
		Currency:   "USD",
		Data: map[string]any{
			"price":     last.Price,
			"points":    points,
			"bidSjc":    last.BidSJC,
			"askSjc":    last.AskSJC,
			"timestamp": time.UnixMilli(last.ReportDate).UTC().Format(time.RFC3339),
		},
	}, nil
}
