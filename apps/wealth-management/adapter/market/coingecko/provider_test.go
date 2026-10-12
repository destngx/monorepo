package coingecko

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func newTestProvider(t *testing.T, apiKey string, handler http.HandlerFunc) *Provider {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	provider, err := NewProvider(domain.MarketDataProviderConfig{BaseURL: server.URL}, apiKey, logger.NewTestLogger(t))
	if err != nil {
		t.Fatalf("new provider error: %v", err)
	}
	return provider
}

func TestGivenUSDTWhenGetExchangeRateToVNDThenQuotesTetherWithTimestampAndKey(t *testing.T) {
	provider := newTestProvider(t, "demo-key", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if r.URL.Path != "/simple/price" || query.Get("ids") != "tether" || query.Get("vs_currencies") != "vnd" {
			t.Fatalf("unexpected request: %s", r.URL.String())
		}
		if r.Header.Get("x-cg-demo-api-key") != "demo-key" {
			t.Fatal("missing api key header")
		}
		_, _ = w.Write([]byte(`{"tether":{"vnd":25916,"last_updated_at":1760246400}}`))
	})

	rate, err := provider.GetExchangeRate(context.Background(), "usdt", "VND")
	if err != nil {
		t.Fatalf("exchange rate error: %v", err)
	}
	if rate.From != "USDT" || rate.To != "VND" || rate.Rate != 25916 || rate.Provider != "coingecko" || rate.Timestamp != "2025-10-12T05:20:00Z" {
		t.Fatalf("unexpected rate: %#v", rate)
	}
}

func TestGivenUnpinnedSymbolWhenGetTickerThenResolvesBestRankedExactMatchOnce(t *testing.T) {
	var searches atomic.Int32
	provider := newTestProvider(t, "", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-cg-demo-api-key") != "" {
			t.Fatal("api key header sent without a key")
		}
		switch r.URL.Path {
		case "/search":
			searches.Add(1)
			_, _ = w.Write([]byte(`{"coins":[
				{"id":"pepe-fake","symbol":"PEPE","market_cap_rank":0},
				{"id":"pepe-two","symbol":"PEPE","market_cap_rank":900},
				{"id":"pepe","symbol":"PEPE","market_cap_rank":40},
				{"id":"pepecoin","symbol":"PEPECOIN","market_cap_rank":1}]}`))
		case "/simple/price":
			if r.URL.Query().Get("ids") != "pepe" {
				t.Fatalf("unexpected id: %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"pepe":{"usd":0.00001}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	})

	for range 2 {
		ticker, err := provider.GetTicker(context.Background(), "pepe", domain.TickerTypeCrypto)
		if err != nil {
			t.Fatalf("ticker error: %v", err)
		}
		if ticker.Symbol != "PEPE" || ticker.Currency != "USD" || ticker.Price != 0.00001 || ticker.Type != domain.TickerTypeCrypto {
			t.Fatalf("unexpected ticker: %#v", ticker)
		}
	}
	if searches.Load() != 1 {
		t.Fatalf("expected one search, got %d", searches.Load())
	}
}

func TestGivenMissingPriceWhenGetTickerThenFails(t *testing.T) {
	provider := newTestProvider(t, "", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})
	if _, err := provider.GetTicker(context.Background(), "BTC", domain.TickerTypeCrypto); err == nil {
		t.Fatal("expected an error without a price")
	}
	if _, err := provider.GetTicker(context.Background(), "BTC", domain.TickerTypeEquity); err == nil {
		t.Fatal("expected an error for equity tickers")
	}
}
