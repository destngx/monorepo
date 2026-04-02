package fmarket

import (
	"apps/wealth-management-engine/adapter/logger"
	"apps/wealth-management-engine/domain"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGivenHealthRequestWhenCallingProviderThenInjectsInstitutionalHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Origin") != "https://fmarket.vn" {
			t.Fatalf("expected fmarket origin header")
		}
		if r.Header.Get("Referer") != "https://fmarket.vn/" {
			t.Fatalf("expected fmarket referer header")
		}
		if r.Header.Get("F-Language") != "vi" {
			t.Fatalf("expected fmarket language header")
		}
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	testLog := logger.NewTestLogger(t)
	provider, err := NewProvider(domain.MarketDataProviderConfig{BaseURL: server.URL}, testLog)
	if err != nil {
		t.Fatalf("new provider error: %v", err)
	}

	health, err := provider.Health(context.Background())
	if err != nil {
		t.Fatalf("health error: %v", err)
	}
	if health.Provider != "fmarket" {
		t.Fatalf("expected provider=fmarket, got %s", health.Provider)
	}
}

func TestGivenEquityTickerWhenGetTickerThenReturnsUnsupportedError(t *testing.T) {
	testLog := logger.NewTestLogger(t)
	provider, err := NewProvider(domain.MarketDataProviderConfig{BaseURL: "http://localhost"}, testLog)
	if err != nil {
		t.Fatalf("new provider error: %v", err)
	}

	_, err = provider.GetTicker(context.Background(), "ACB", domain.TickerTypeEquity)
	if err == nil {
		t.Fatalf("expected unsupported ticker type error")
	}
}

func TestGivenGoldTickerWhenSymbolIsXAUThenReturnsWorldGoldPrice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST request, got %s", r.Method)
		}
		_, _ = w.Write([]byte(`{"data":[{"price":4200.5,"bidSjc":152500000,"askSjc":154500000,"reportDate":1764694800000}]}`))
	}))
	defer server.Close()

	provider, err := NewProvider(domain.MarketDataProviderConfig{BaseURL: server.URL}, logger.NewTestLogger(t))
	if err != nil {
		t.Fatalf("new provider error: %v", err)
	}

	ticker, err := provider.GetTicker(context.Background(), "XAU", domain.TickerTypeGold)
	if err != nil {
		t.Fatalf("gold ticker error: %v", err)
	}
	if ticker.Currency != "USD" || ticker.Price <= 0 || ticker.Symbol != "XAU" {
		t.Fatalf("expected XAU world gold ticker, got %+v", ticker)
	}
}

func TestGivenGoldTickerWhenSymbolIsSJCThenReturnsDomesticGoldBidAsk(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(`{"data":[{"price":4200.5,"bidSjc":152500000,"askSjc":154500000,"reportDate":1764694800000}]}`))
	}))
	defer server.Close()

	provider, err := NewProvider(domain.MarketDataProviderConfig{BaseURL: server.URL}, logger.NewTestLogger(t))
	if err != nil {
		t.Fatalf("new provider error: %v", err)
	}

	ticker, err := provider.GetTicker(context.Background(), "SJC", domain.TickerTypeGold)
	if err != nil {
		t.Fatalf("gold ticker error: %v", err)
	}
	if ticker.Currency != "VND" || ticker.Bid <= 0 || ticker.Ask < ticker.Bid || ticker.Symbol != "SJC" {
		t.Fatalf("expected SJC domestic gold ticker, got %+v", ticker)
	}
}
