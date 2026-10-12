package currencyapi

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestProvider(t *testing.T, handler http.HandlerFunc) *Provider {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	provider, err := NewProvider(domain.MarketDataProviderConfig{BaseURL: server.URL}, logger.NewTestLogger(t))
	if err != nil {
		t.Fatalf("new provider error: %v", err)
	}
	return provider
}

func TestGivenDailySnapshotWhenGetExchangeRateThenReturnsRateWithDate(t *testing.T) {
	provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/currencies/usdt.min.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"date":"2026-10-12","usdt":{"vnd":25901.2,"usd":1.0001}}`))
	})

	rate, err := provider.GetExchangeRate(context.Background(), "USDT", "VND")
	if err != nil {
		t.Fatalf("exchange rate error: %v", err)
	}
	if rate.From != "USDT" || rate.To != "VND" || rate.Rate != 25901.2 || rate.Timestamp != "2026-10-12T00:00:00Z" || rate.Provider != "currencyapi" {
		t.Fatalf("unexpected rate: %#v", rate)
	}
}

func TestGivenUnknownTargetWhenGetExchangeRateThenFails(t *testing.T) {
	provider := newTestProvider(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"date":"2026-10-12","usd":{"eur":0.9}}`))
	})
	if _, err := provider.GetExchangeRate(context.Background(), "USD", "VND"); err == nil {
		t.Fatal("expected an error for a missing target")
	}
}

func TestGivenUpstreamErrorWhenGetExchangeRateThenReportsStatus(t *testing.T) {
	provider := newTestProvider(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	if _, err := provider.GetExchangeRate(context.Background(), "USD", "VND"); err == nil {
		t.Fatal("expected an error for a 404")
	}
}
