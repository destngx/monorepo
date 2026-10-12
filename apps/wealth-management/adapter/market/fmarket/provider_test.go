package fmarket

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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

func TestGivenForexSeriesWhenGetPriceSeriesThenReturnsUSDVNDHistory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/res/get-usd-rate-history" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{"rateSellUsd":25850,"reportDate":1764608400000},{"rateSellUsd":0,"reportDate":1764694800000},{"rateSellUsd":25890,"reportDate":1764781200000}]}`))
	}))
	defer server.Close()

	provider, err := NewProvider(domain.MarketDataProviderConfig{BaseURL: server.URL}, logger.NewTestLogger(t))
	if err != nil {
		t.Fatalf("new provider error: %v", err)
	}
	series, err := provider.GetPriceSeries(context.Background(), "USD_VND", domain.SeriesTypeForex)
	if err != nil {
		t.Fatalf("forex series error: %v", err)
	}
	data, ok := series.Data.(map[string]any)
	if !ok || series.SeriesType != domain.SeriesTypeForex || series.Currency != "VND" {
		t.Fatalf("unexpected series: %#v", series)
	}
	points := data["points"].([]domain.PricePoint)
	if len(points) != 2 || points[1].Price != 25890 {
		t.Fatalf("expected the zero rate to be dropped, got %#v", points)
	}
	if _, err := provider.GetPriceSeries(context.Background(), "EUR_VND", domain.SeriesTypeForex); err == nil {
		t.Fatal("expected EUR_VND to be unsupported")
	}
}

func TestGivenGoldHistoryWhenGetCandlesThenSendsCompactDatesAndMapsSeries(t *testing.T) {
	var request map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/res/get-price-gold-history" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&request)
		// Newest first, as fmarket answers; 2026-10-11 and 2026-10-12 00:00 ICT.
		_, _ = w.Write([]byte(`{"data":[{"price":4162.96,"bidSjc":141300000,"askSjc":144300000,"reportDate":1791738000000},{"price":4150,"bidSjc":141000000,"askSjc":144000000,"reportDate":1791651600000}]}`))
	}))
	defer server.Close()

	provider, err := NewProvider(domain.MarketDataProviderConfig{BaseURL: server.URL}, logger.NewTestLogger(t))
	if err != nil {
		t.Fatalf("new provider error: %v", err)
	}
	to := time.Date(2026, 10, 12, 20, 0, 0, 0, time.UTC) // 03:00 on the 13th in Vietnam
	query := domain.CandleQuery{Timeframe: domain.Timeframe1D, From: to.AddDate(0, 0, -30), To: to}
	series, err := provider.GetCandles(context.Background(), domain.MarketSymbolSJC, query)
	if err != nil {
		t.Fatalf("get candles error: %v", err)
	}
	if request["fromDate"] != "20260913" || request["toDate"] != "20261013" {
		t.Fatalf("fmarket needs compact Vietnam dates, sent %v", request)
	}
	if series.Currency != "VND" || len(series.Candles) != 2 || series.Candles[1].Close != 144300000 || !series.Candles[0].Time.Before(series.Candles[1].Time) {
		t.Fatalf("unexpected SJC series %#v", series)
	}
	world, err := provider.GetCandles(context.Background(), domain.MarketSymbolXAUUSD, query)
	if err != nil || world.Currency != "USD" || world.Candles[1].Close != 4162.96 {
		t.Fatalf("unexpected world gold series %#v, %v", world, err)
	}
	for _, unsupported := range []struct {
		symbol    string
		timeframe domain.Timeframe
	}{{"FPT", domain.Timeframe1D}, {domain.MarketSymbolSJC, domain.Timeframe1H}} {
		query.Timeframe = unsupported.timeframe
		if _, err := provider.GetCandles(context.Background(), unsupported.symbol, query); !errors.Is(err, port.ErrUnsupported) {
			t.Fatalf("%s %s must be unsupported, got %v", unsupported.symbol, unsupported.timeframe, err)
		}
	}
}

func TestGivenNewestFirstHistoryWhenReadingLatestThenUsesNewestReportDate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/res/get-usd-rate-history":
			_, _ = w.Write([]byte(`{"data":[{"rateSellUsd":26120,"reportDate":1791738000000},{"rateSellUsd":0,"reportDate":1791651600000},{"rateSellUsd":26070,"reportDate":1791478800000}]}`))
		case "/res/get-price-gold-history":
			_, _ = w.Write([]byte(`{"data":[{"price":4162.96,"askSjc":144300000,"reportDate":1791738000000},{"price":4170.93,"askSjc":144800000,"reportDate":1791478800000}]}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	provider, err := NewProvider(domain.MarketDataProviderConfig{BaseURL: server.URL}, logger.NewTestLogger(t))
	if err != nil {
		t.Fatalf("new provider error: %v", err)
	}
	rate, err := provider.GetExchangeRate(context.Background(), "USD", "VND")
	if err != nil || rate.Rate != 26120 || rate.Timestamp != "2026-10-11T17:00:00Z" {
		t.Fatalf("expected the newest rate, got %#v %v", rate, err)
	}
	gold, err := provider.GetPriceSeries(context.Background(), "", domain.SeriesTypeGoldUSD)
	if err != nil {
		t.Fatalf("gold series error: %v", err)
	}
	data := gold.Data.(map[string]any)
	points := data["points"].([]domain.PricePoint)
	if data["price"] != 4162.96 || data["askSjc"] != 144300000.0 || points[len(points)-1].Price != 4162.96 {
		t.Fatalf("expected the newest gold quote last, got %#v", data)
	}
}

func TestGivenBankRateTableWhenReadingThenKeepsValidRatesWithPublicationDate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":200,"data":{"updateAt":"05/10/2026","bankList":[{"key":"LPBank","value":"7.2","name":"LPBank"},{"key":"X","value":"n/a","name":""}]}}`))
	}))
	defer server.Close()
	provider, err := NewProvider(domain.MarketDataProviderConfig{BaseURL: server.URL}, logger.NewTestLogger(t))
	if err != nil {
		t.Fatalf("new provider error: %v", err)
	}
	rates, err := provider.GetBankInterestRate(context.Background())
	if err != nil {
		t.Fatalf("bank rates error: %v", err)
	}
	if len(rates) != 1 || rates[0].Bank != "LPBank" || rates[0].Rate != 7.2 || rates[0].UpdatedAt != "2026-10-05" || rates[0].Tenor != "" {
		t.Fatalf("unexpected rates %+v", rates)
	}
}
