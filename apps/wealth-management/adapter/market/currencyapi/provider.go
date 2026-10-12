// Package currencyapi reads daily reference rates from the free
// fawazahmed0 currency-api (fiat and major coins). It is the exchange-rate
// fallback for fmarket and CoinGecko.
package currencyapi

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/adapter/market/httpjson"
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultBaseURL is the jsDelivr mirror of the latest daily snapshot.
const DefaultBaseURL = "https://cdn.jsdelivr.net/npm/@fawazahmed0/currency-api@latest"

type Provider struct {
	baseURL    string
	httpClient *http.Client
}

var _ port.ExchangeRateProvider = (*Provider)(nil)

func NewProvider(config domain.MarketDataProviderConfig, log *logger.Logger) (*Provider, error) {
	if strings.TrimSpace(config.BaseURL) == "" {
		return nil, fmt.Errorf("missing currency-api base URL")
	}
	log.LogApplicationEvent(context.Background(), "initializing currency-api market provider",
		slog.String("base_url", config.BaseURL),
		slog.String("component", "currencyapi"),
	)
	return &Provider{baseURL: strings.TrimRight(config.BaseURL, "/"), httpClient: httpjson.NewClient()}, nil
}

func (p *Provider) Name() string {
	return "currencyapi"
}

func (p *Provider) Health(ctx context.Context) (domain.MarketProviderHealth, error) {
	if _, err := p.GetExchangeRate(ctx, "USD", "VND"); err != nil {
		return domain.MarketProviderHealth{}, err
	}
	return domain.MarketProviderHealth{Provider: p.Name(), Status: "ok"}, nil
}

func (p *Provider) GetExchangeRate(ctx context.Context, from string, to string) (domain.ExchangeRate, error) {
	from, to = strings.ToLower(strings.TrimSpace(from)), strings.ToLower(strings.TrimSpace(to))
	if from == "" || to == "" {
		return domain.ExchangeRate{}, fmt.Errorf("%w: currencies are required", domain.ErrValidation)
	}
	endpoint := fmt.Sprintf("%s/v1/currencies/%s.min.json", p.baseURL, url.PathEscape(from))

	var payload map[string]json.RawMessage
	if err := httpjson.Get(ctx, p.httpClient, "currencyapi", endpoint, nil, &payload); err != nil {
		return domain.ExchangeRate{}, err
	}
	var rates map[string]float64
	if err := json.Unmarshal(payload[from], &rates); err != nil {
		return domain.ExchangeRate{}, fmt.Errorf("currencyapi: unreadable %s rates: %w", from, err)
	}
	rate := rates[to]
	if rate <= 0 {
		return domain.ExchangeRate{}, fmt.Errorf("currencyapi has no %s/%s rate", strings.ToUpper(from), strings.ToUpper(to))
	}

	var timestamp string
	var date string
	if json.Unmarshal(payload["date"], &date) == nil {
		if day, err := time.Parse(time.DateOnly, date); err == nil {
			timestamp = day.UTC().Format(time.RFC3339)
		}
	}
	return domain.ExchangeRate{
		From:      strings.ToUpper(from),
		To:        strings.ToUpper(to),
		Rate:      rate,
		Provider:  p.Name(),
		Timestamp: timestamp,
	}, nil
}
