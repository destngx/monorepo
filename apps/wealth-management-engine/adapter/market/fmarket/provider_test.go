package fmarket

import (
	"apps/wealth-management-engine/domain"
	"context"
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

	provider, err := NewProvider(domain.MarketDataProviderConfig{BaseURL: server.URL})
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
	provider, err := NewProvider(domain.MarketDataProviderConfig{BaseURL: "http://localhost"})
	if err != nil {
		t.Fatalf("new provider error: %v", err)
	}

	_, err = provider.GetTicker(context.Background(), "ACB", domain.TickerTypeEquity)
	if err == nil {
		t.Fatalf("expected unsupported ticker type error")
	}
}
