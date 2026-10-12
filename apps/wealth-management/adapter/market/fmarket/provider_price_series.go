package fmarket

import (
	"apps/wealth-management/domain"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (p *Provider) GetPriceSeries(ctx context.Context, symbol string, seriesType domain.SeriesType) (domain.PriceSeries, error) {
	switch seriesType {
	case domain.SeriesTypeGoldUSD:
		return p.goldSeries(ctx)
	case domain.SeriesTypeForex:
		return p.forexSeries(ctx, symbol)
	default:
		return domain.PriceSeries{}, fmt.Errorf("fmarket supports only %s and %s price series", domain.SeriesTypeGoldUSD, domain.SeriesTypeForex)
	}
}

// forexSeries is the USD/VND sell rate over the last 30 days (WM-MKT-3).
func (p *Provider) forexSeries(ctx context.Context, symbol string) (domain.PriceSeries, error) {
	pair := strings.ReplaceAll(strings.ToUpper(strings.TrimSpace(symbol)), "/", "_")
	if pair != "" && pair != "USD_VND" && pair != "USDVND" {
		return domain.PriceSeries{}, fmt.Errorf("fmarket has a forex series only for USD_VND")
	}
	fromDate, toDate := lastNDaysRange(30)
	body := map[string]any{"fromDate": fromDate, "toDate": toDate, "isAllData": false}
	response, err := p.do(ctx, http.MethodPost, "/res/get-usd-rate-history", body)
	if err != nil {
		return domain.PriceSeries{}, err
	}
	defer response.Body.Close()

	var envelope fmarketEnvelope
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		return domain.PriceSeries{}, err
	}
	var rows []usdRateRow
	if err := json.Unmarshal(envelope.Data, &rows); err != nil {
		return domain.PriceSeries{}, err
	}
	oldestFirst(rows, usdRateRow.date)
	points := make([]domain.PricePoint, 0, len(rows))
	for _, row := range rows {
		if row.RateSellUSD <= 0 {
			continue
		}
		points = append(points, domain.PricePoint{
			Price:     row.RateSellUSD,
			Timestamp: time.UnixMilli(row.ReportDate).UTC().Format(time.RFC3339),
		})
	}
	if len(points) == 0 {
		return domain.PriceSeries{}, fmt.Errorf("fmarket returned an empty USD/VND series")
	}
	return domain.PriceSeries{
		Provider:   p.Name(),
		SeriesType: domain.SeriesTypeForex,
		Currency:   "VND",
		Data:       map[string]any{"pair": "USD_VND", "points": points},
	}, nil
}

func (p *Provider) goldSeries(ctx context.Context) (domain.PriceSeries, error) {
	seriesType := domain.SeriesTypeGoldUSD

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

	var rows []goldRow
	if err := json.Unmarshal(envelope.Data, &rows); err != nil {
		return domain.PriceSeries{}, err
	}
	oldestFirst(rows, goldRow.date)
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

type goldRow struct {
	Price      float64 `json:"price"`
	BidSJC     float64 `json:"bidSjc"`
	AskSJC     float64 `json:"askSjc"`
	ReportDate int64   `json:"reportDate"`
}

func (row goldRow) date() int64 { return row.ReportDate }
