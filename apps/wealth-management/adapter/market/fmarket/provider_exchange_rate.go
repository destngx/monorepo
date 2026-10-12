package fmarket

import (
	"apps/wealth-management/domain"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"
)

func (p *Provider) GetExchangeRate(ctx context.Context, from string, to string) (domain.ExchangeRate, error) {
	if strings.ToUpper(from) != "USD" || strings.ToUpper(to) != "VND" {
		return domain.ExchangeRate{}, fmt.Errorf("fmarket supports only USD_VND exchange rate")
	}

	fromDate, toDate := lastNDaysRange(7)
	body := map[string]any{
		"fromDate":  fromDate,
		"toDate":    toDate,
		"isAllData": false,
	}
	response, err := p.do(ctx, http.MethodPost, "/res/get-usd-rate-history", body)
	if err != nil {
		return domain.ExchangeRate{}, err
	}
	defer response.Body.Close()

	var envelope fmarketEnvelope
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		return domain.ExchangeRate{}, err
	}
	var points []usdRateRow
	if err := json.Unmarshal(envelope.Data, &points); err != nil {
		return domain.ExchangeRate{}, err
	}
	points = slices.DeleteFunc(points, func(row usdRateRow) bool { return row.RateSellUSD <= 0 })
	oldestFirst(points, usdRateRow.date)
	if len(points) == 0 {
		return domain.ExchangeRate{}, fmt.Errorf("fmarket returned no exchange-rate data")
	}

	last := points[len(points)-1]
	return domain.ExchangeRate{
		From:      "USD",
		To:        "VND",
		Rate:      last.RateSellUSD,
		Provider:  p.Name(),
		Timestamp: time.UnixMilli(last.ReportDate).UTC().Format(time.RFC3339),
	}, nil
}

type usdRateRow struct {
	RateSellUSD float64 `json:"rateSellUsd"`
	ReportDate  int64   `json:"reportDate"`
}

func (row usdRateRow) date() int64 { return row.ReportDate }
