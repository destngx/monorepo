package vnstock

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGivenVNStockHealthEndpointWhenHealthThenReturnsProviderStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
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
	if health.Provider != "vnstock" || health.Status != "ok" {
		t.Fatalf("unexpected health response: %#v", health)
	}
}

func TestGivenHourlyQueryWhenGetCandlesThenRequestsResolutionAndParsesLocalTimes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if r.URL.Path != "/api/v1/stocks/historical" || query.Get("symbol") != "FPT" || query.Get("resolution") != "1H" ||
			query.Get("start_date") != "2026-10-01" || query.Get("end_date") != "2026-10-12" {
			t.Fatalf("unexpected request: %s", r.URL.String())
		}
		_, _ = w.Write([]byte(`{"success":true,"data":[
			{"time":"2026-10-10 09:00:00","open":100.5,"high":101,"low":100,"close":100.8,"volume":1200},
			{"time":"2026-10-10 10:00:00","open":100.8,"high":102,"low":100.7,"close":101.9,"volume":800}
		]}`))
	}))
	defer server.Close()

	provider, err := NewProvider(domain.MarketDataProviderConfig{BaseURL: server.URL}, logger.NewTestLogger(t))
	if err != nil {
		t.Fatalf("new provider error: %v", err)
	}
	series, err := provider.GetCandles(context.Background(), "FPT", domain.CandleQuery{
		Timeframe: domain.Timeframe1H,
		From:      time.Date(2026, 10, 1, 3, 0, 0, 0, time.UTC),
		To:        time.Date(2026, 10, 12, 3, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("get candles error: %v", err)
	}
	if len(series.Candles) != 2 || series.Provider != "vnstock" || series.Currency != "VND" {
		t.Fatalf("unexpected series: %#v", series)
	}
	first := series.Candles[0]
	if !first.Time.Equal(time.Date(2026, 10, 10, 2, 0, 0, 0, time.UTC)) || first.Close != 100.8 || first.Volume != 1200 {
		t.Fatalf("unexpected first candle: %#v", first)
	}
}

func TestGivenFourHourQueryWhenGetCandlesThenRejectsUnservedTimeframe(t *testing.T) {
	provider, err := NewProvider(domain.MarketDataProviderConfig{BaseURL: "http://127.0.0.1:1"}, logger.NewTestLogger(t))
	if err != nil {
		t.Fatalf("new provider error: %v", err)
	}
	if _, err := provider.GetCandles(context.Background(), "FPT", domain.CandleQuery{Timeframe: domain.Timeframe4H}); err == nil {
		t.Fatal("expected an error for 4h candles")
	}
}
