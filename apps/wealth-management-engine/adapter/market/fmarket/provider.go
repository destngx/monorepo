package fmarket

import (
	"apps/wealth-management-engine/adapter/logger"
	"apps/wealth-management-engine/domain"
	"apps/wealth-management-engine/port"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type Provider struct {
	baseURL    string
	httpClient *http.Client
	log        *logger.Logger
}

var _ port.MarketProvider = (*Provider)(nil)

func NewProvider(config domain.MarketDataProviderConfig, log *logger.Logger) (*Provider, error) {
	if strings.TrimSpace(config.BaseURL) == "" {
		return nil, fmt.Errorf("missing fmarket base URL")
	}

	log.LogApplicationEvent(context.Background(), "initializing fmarket market provider",
		slog.String("base_url", config.BaseURL),
		slog.String("component", "fmarket"),
	)

	return &Provider{
		baseURL: strings.TrimRight(config.BaseURL, "/"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		log: log,
	}, nil
}

func (p *Provider) Name() string {
	return "fmarket"
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
