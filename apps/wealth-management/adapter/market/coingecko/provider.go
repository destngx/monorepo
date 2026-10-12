// Package coingecko prices crypto holdings in USD and quotes USDT/VND
// (WM-INV-2/3).
package coingecko

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/adapter/market/httpjson"
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// DefaultBaseURL is the public API; a demo key raises the rate limit.
const DefaultBaseURL = "https://api.coingecko.com/api/v3"

// knownCoins pins the common symbols to CoinGecko IDs so they never depend
// on search ranking, where symbols collide (many tokens are called "SOL").
var knownCoins = map[string]string{
	"BTC": "bitcoin", "ETH": "ethereum", "USDT": "tether", "USDC": "usd-coin", "FDUSD": "first-digital-usd",
	"DAI": "dai", "BNB": "binancecoin", "SOL": "solana", "XRP": "ripple", "ADA": "cardano", "DOGE": "dogecoin",
	"TON": "the-open-network", "TRX": "tron", "DOT": "polkadot", "POL": "polygon-ecosystem-token",
	"LINK": "chainlink", "AVAX": "avalanche-2", "LTC": "litecoin", "SUI": "sui", "ARB": "arbitrum", "OP": "optimism",
}

type Provider struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client

	mu       sync.Mutex
	resolved map[string]string
}

var (
	_ port.TickerProvider       = (*Provider)(nil)
	_ port.ExchangeRateProvider = (*Provider)(nil)
)

// NewProvider builds the adapter; apiKey is optional and sent as the demo
// key header, never logged.
func NewProvider(config domain.MarketDataProviderConfig, apiKey string, log *logger.Logger) (*Provider, error) {
	if strings.TrimSpace(config.BaseURL) == "" {
		return nil, fmt.Errorf("missing CoinGecko base URL")
	}
	log.LogApplicationEvent(context.Background(), "initializing coingecko market provider",
		slog.String("base_url", config.BaseURL),
		slog.Bool("api_key_configured", apiKey != ""),
		slog.String("component", "coingecko"),
	)
	return &Provider{
		baseURL:    strings.TrimRight(config.BaseURL, "/"),
		apiKey:     strings.TrimSpace(apiKey),
		httpClient: httpjson.NewClient(),
		resolved:   map[string]string{},
	}, nil
}

func (p *Provider) Name() string {
	return "coingecko"
}

func (p *Provider) Health(ctx context.Context) (domain.MarketProviderHealth, error) {
	var payload map[string]any
	if err := p.get(ctx, "/ping", nil, &payload); err != nil {
		return domain.MarketProviderHealth{}, err
	}
	return domain.MarketProviderHealth{Provider: p.Name(), Status: "ok"}, nil
}

func (p *Provider) GetTicker(ctx context.Context, symbol string, tickerType domain.TickerType) (domain.Ticker, error) {
	if tickerType != domain.TickerTypeCrypto {
		return domain.Ticker{}, fmt.Errorf("coingecko supports only %s tickers", domain.TickerTypeCrypto)
	}
	price, err := p.price(ctx, symbol, "usd")
	if err != nil {
		return domain.Ticker{}, err
	}
	return domain.Ticker{
		Symbol:    strings.ToUpper(symbol),
		Type:      tickerType,
		Provider:  p.Name(),
		Currency:  "USD",
		Price:     price.value,
		Timestamp: price.updatedAt,
	}, nil
}

// GetExchangeRate quotes a coin (from) in USD or VND (to), e.g. USDT_VND.
func (p *Provider) GetExchangeRate(ctx context.Context, from string, to string) (domain.ExchangeRate, error) {
	to = strings.ToLower(to)
	if to != "usd" && to != "vnd" {
		return domain.ExchangeRate{}, fmt.Errorf("coingecko quotes coins only in USD or VND")
	}
	price, err := p.price(ctx, from, to)
	if err != nil {
		return domain.ExchangeRate{}, err
	}
	return domain.ExchangeRate{
		From:      strings.ToUpper(from),
		To:        strings.ToUpper(to),
		Rate:      price.value,
		Provider:  p.Name(),
		Timestamp: price.updatedAt,
	}, nil
}

type quote struct {
	value     float64
	updatedAt string
}

func (p *Provider) price(ctx context.Context, symbol string, currency string) (quote, error) {
	id, err := p.coinID(ctx, symbol)
	if err != nil {
		return quote{}, err
	}
	params := url.Values{"ids": {id}, "vs_currencies": {currency}, "include_last_updated_at": {"true"}}
	var payload map[string]map[string]float64
	if err := p.get(ctx, "/simple/price", params, &payload); err != nil {
		return quote{}, err
	}
	value := payload[id][currency]
	if value <= 0 {
		return quote{}, fmt.Errorf("coingecko has no %s price for %s", strings.ToUpper(currency), id)
	}
	result := quote{value: value}
	if updated := payload[id]["last_updated_at"]; updated > 0 {
		result.updatedAt = time.Unix(int64(updated), 0).UTC().Format(time.RFC3339)
	}
	return result, nil
}

// coinID resolves a symbol: pinned map first, then the search result with
// that exact symbol and the best market-cap rank. Results are memoised.
func (p *Provider) coinID(ctx context.Context, symbol string) (string, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return "", fmt.Errorf("%w: coin symbol is required", domain.ErrValidation)
	}
	if id, ok := knownCoins[symbol]; ok {
		return id, nil
	}
	p.mu.Lock()
	id, ok := p.resolved[symbol]
	p.mu.Unlock()
	if ok {
		return id, nil
	}

	var payload struct {
		Coins []struct {
			ID            string `json:"id"`
			Symbol        string `json:"symbol"`
			MarketCapRank int    `json:"market_cap_rank"`
		} `json:"coins"`
	}
	if err := p.get(ctx, "/search", url.Values{"query": {symbol}}, &payload); err != nil {
		return "", err
	}
	bestRank := 0
	for _, coin := range payload.Coins {
		if !strings.EqualFold(coin.Symbol, symbol) || coin.MarketCapRank <= 0 {
			continue
		}
		if bestRank == 0 || coin.MarketCapRank < bestRank {
			id, bestRank = coin.ID, coin.MarketCapRank
		}
	}
	if id == "" {
		return "", fmt.Errorf("coingecko has no ranked coin with symbol %s", symbol)
	}
	p.mu.Lock()
	p.resolved[symbol] = id
	p.mu.Unlock()
	return id, nil
}

func (p *Provider) get(ctx context.Context, path string, params url.Values, out any) error {
	endpoint := p.baseURL + path
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}
	headers := map[string]string{}
	if p.apiKey != "" {
		headers["x-cg-demo-api-key"] = p.apiKey
	}
	return httpjson.Get(ctx, p.httpClient, "coingecko", endpoint, headers, out)
}
