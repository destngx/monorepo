package vnstock

import (
	"apps/wealth-management-engine/domain"
	"apps/wealth-management-engine/port"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
		return nil, fmt.Errorf("missing VNStock base URL")
	}

	return &Provider{
		baseURL: strings.TrimRight(config.BaseURL, "/"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

func (p *Provider) Name() string {
	return "vnstock"
}

func (p *Provider) Health(ctx context.Context) (domain.MarketProviderHealth, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/health", nil)
	if err != nil {
		return domain.MarketProviderHealth{}, err
	}

	response, err := p.httpClient.Do(request)
	if err != nil {
		return domain.MarketProviderHealth{}, err
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		return domain.MarketProviderHealth{}, fmt.Errorf("vnstock health request failed with status %d", response.StatusCode)
	}

	var payload struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return domain.MarketProviderHealth{}, err
	}
	if payload.Status == "" {
		payload.Status = "unknown"
	}

	return domain.MarketProviderHealth{
		Provider: p.Name(),
		Status:   payload.Status,
	}, nil
}

func (p *Provider) GetTicker(ctx context.Context, symbol string, tickerType domain.TickerType) (domain.Ticker, error) {
	if tickerType != domain.TickerTypeEquity {
		return domain.Ticker{}, fmt.Errorf("vnstock does not support ticker type %s", tickerType)
	}

	response, err := p.do(ctx, "/api/v1/stocks/quote?symbol="+url.QueryEscape(symbol))
	if err != nil {
		return domain.Ticker{}, err
	}
	defer response.Body.Close()

	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			Symbol string      `json:"symbol"`
			Price  interface{} `json:"price"`
			Time   string      `json:"time"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return domain.Ticker{}, err
	}
	if !payload.Success {
		return domain.Ticker{}, fmt.Errorf("vnstock quote request did not succeed")
	}

	return domain.Ticker{
		Symbol:    fallbackSymbol(payload.Data.Symbol, symbol),
		Type:      tickerType,
		Provider:  p.Name(),
		Currency:  "VND",
		Price:     parseFloat(payload.Data.Price),
		Timestamp: payload.Data.Time,
	}, nil
}

func (p *Provider) GetExchangeRate(ctx context.Context, from string, to string) (domain.ExchangeRate, error) {
	upperFrom := strings.ToUpper(from)
	upperTo := strings.ToUpper(to)
	if upperFrom != "USD" || upperTo != "VND" {
		return domain.ExchangeRate{}, fmt.Errorf("vnstock supports fallback exchange rate only for USD_VND")
	}

	return domain.ExchangeRate{}, fmt.Errorf("vnstock exchange-rate endpoint is unavailable in current API surface")
}

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

func (p *Provider) GetBankInterestRate(_ context.Context) ([]domain.BankRate, error) {
	return nil, errors.New("vnstock does not support bank interest rates")
}

func (p *Provider) do(ctx context.Context, path string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+path, nil)
	if err != nil {
		return nil, err
	}

	response, err := p.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= http.StatusBadRequest {
		defer response.Body.Close()
		return nil, fmt.Errorf("vnstock request failed with status %d", response.StatusCode)
	}

	return response, nil
}

func fallbackSymbol(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func parseFloat(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	default:
		return 0
	}
}
