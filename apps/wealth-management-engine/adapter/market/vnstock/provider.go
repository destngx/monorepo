package vnstock

import (
	"apps/wealth-management-engine/domain"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Provider struct {
	baseURL    string
	httpClient *http.Client
}

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
