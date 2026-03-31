package fmarket

import (
	"apps/wealth-management-engine/domain"
	"apps/wealth-management-engine/port"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Provider struct {
	baseURL    string
	httpClient *http.Client
}

var _ port.MarketProvider = (*Provider)(nil)

func NewProvider(config domain.MarketDataProviderConfig) (*Provider, error) {
	if strings.TrimSpace(config.BaseURL) == "" {
		return nil, fmt.Errorf("missing fmarket base URL")
	}

	return &Provider{
		baseURL: strings.TrimRight(config.BaseURL, "/"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

func (p *Provider) Name() string {
	return "fmarket"
}

func (p *Provider) Health(ctx context.Context) (domain.MarketProviderHealth, error) {
	response, err := p.do(ctx, http.MethodGet, "/res/bank-interest-rate", nil)
	if err != nil {
		return domain.MarketProviderHealth{}, err
	}
	defer response.Body.Close()

	var envelope fmarketEnvelope
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		return domain.MarketProviderHealth{}, err
	}
	status := "ok"
	if statusCode, ok := parseEnvelopeStatusCode(envelope.Status); ok && statusCode >= http.StatusBadRequest {
		status = "degraded"
	}

	return domain.MarketProviderHealth{
		Provider: p.Name(),
		Status:   status,
	}, nil
}

func (p *Provider) GetTicker(ctx context.Context, symbol string, tickerType domain.TickerType) (domain.Ticker, error) {
	switch tickerType {
	case domain.TickerTypeIFC:
		response, err := p.do(ctx, http.MethodGet, "/home/product/"+url.PathEscape(symbol), nil)
		if err != nil {
			return domain.Ticker{}, err
		}
		defer response.Body.Close()

		var envelope fmarketEnvelope
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			return domain.Ticker{}, err
		}
		var product struct {
			Code string  `json:"code"`
			Nav  float64 `json:"nav"`
		}
		if err := json.Unmarshal(envelope.Data, &product); err != nil {
			return domain.Ticker{}, err
		}

		return domain.Ticker{
			Symbol:   fallbackString(product.Code, symbol),
			Type:     tickerType,
			Provider: p.Name(),
			Currency: "VND",
			Price:    product.Nav,
		}, nil
	case domain.TickerTypeGold:
		series, err := p.GetPriceSeries(ctx, symbol, domain.SeriesTypeGoldUSD)
		if err != nil {
			return domain.Ticker{}, err
		}
		payload, ok := series.Data.(map[string]any)
		if !ok {
			return domain.Ticker{}, fmt.Errorf("unexpected gold series payload")
		}

		bid, _ := payload["bidSjc"].(float64)
		ask, _ := payload["askSjc"].(float64)
		timestamp, _ := payload["timestamp"].(string)
		return domain.Ticker{
			Symbol:    symbol,
			Type:      tickerType,
			Provider:  p.Name(),
			Currency:  "VND",
			Bid:       bid,
			Ask:       ask,
			Timestamp: timestamp,
		}, nil
	default:
		return domain.Ticker{}, fmt.Errorf("fmarket does not support ticker type %s", tickerType)
	}
}

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
	var points []struct {
		RateSellUSD float64 `json:"rateSellUsd"`
		ReportDate  int64   `json:"reportDate"`
	}
	if err := json.Unmarshal(envelope.Data, &points); err != nil {
		return domain.ExchangeRate{}, err
	}
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
			"points":    points,
			"bidSjc":    last.BidSJC,
			"askSjc":    last.AskSJC,
			"timestamp": time.UnixMilli(last.ReportDate).UTC().Format(time.RFC3339),
		},
	}, nil
}

func (p *Provider) GetBankInterestRate(ctx context.Context) ([]domain.BankRate, error) {
	response, err := p.do(ctx, http.MethodGet, "/res/bank-interest-rate", nil)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	var envelope fmarketEnvelope
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		return nil, err
	}

	var payload struct {
		BankList []struct {
			Name  string `json:"name"`
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"bankList"`
	}
	if err := json.Unmarshal(envelope.Data, &payload); err != nil {
		return nil, err
	}

	rates := make([]domain.BankRate, 0, len(payload.BankList))
	for _, item := range payload.BankList {
		rate, _ := parseStringFloat(item.Value)
		rates = append(rates, domain.BankRate{
			Bank:     fallbackString(item.Name, item.Key),
			Tenor:    "",
			Rate:     rate,
			Currency: "VND",
		})
	}

	return rates, nil
}

func (p *Provider) do(ctx context.Context, method string, path string, body interface{}) (*http.Response, error) {
	var requestBody io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		requestBody = bytes.NewBuffer(raw)
	}

	request, err := http.NewRequestWithContext(ctx, method, p.baseURL+path, requestBody)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 14.7; rv:123.0) Gecko/20100101 Firefox/123.0")
	request.Header.Set("Accept", "application/json, text/plain, */*")
	request.Header.Set("Origin", "https://fmarket.vn")
	request.Header.Set("Referer", "https://fmarket.vn/")
	request.Header.Set("F-Language", "vi")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := p.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= http.StatusBadRequest {
		defer response.Body.Close()
		return nil, fmt.Errorf("fmarket request failed with status %d", response.StatusCode)
	}

	return response, nil
}

type fmarketEnvelope struct {
	Status interface{}     `json:"status"`
	Data   json.RawMessage `json:"data"`
}

func fallbackString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func lastNDaysRange(days int) (fromDate string, toDate string) {
	now := time.Now()
	from := now.AddDate(0, 0, -days)
	return from.Format("2006-01-02"), now.Format("2006-01-02")
}

func parseEnvelopeStatusCode(value interface{}) (int, bool) {
	switch v := value.(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	case string:
		var parsed int
		if _, err := fmt.Sscanf(v, "%d", &parsed); err == nil {
			return parsed, true
		}
	}
	return 0, false
}

func parseStringFloat(value string) (float64, bool) {
	var parsed float64
	if _, err := fmt.Sscanf(strings.TrimSpace(value), "%f", &parsed); err == nil {
		return parsed, true
	}
	return 0, false
}
