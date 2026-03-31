package vnstock

import (
	"apps/wealth-management-engine/domain"
	"apps/wealth-management-engine/port"
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
