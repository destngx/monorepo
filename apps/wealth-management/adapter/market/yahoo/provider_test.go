package yahoo

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

func TestGivenEquitySymbolWhenGetWeeklyCandlesThenMapsSymbolAndKeepsNullBarsForCleaning(t *testing.T) {
	provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v8/finance/chart/FPT.VN" || r.URL.Query().Get("interval") != "1wk" {
			t.Fatalf("unexpected request: %s", r.URL.String())
		}
		if !strings.Contains(r.Header.Get("User-Agent"), "Mozilla") {
			t.Fatalf("missing user agent")
		}
		_, _ = w.Write([]byte(`{"chart":{"result":[{"meta":{"currency":"VND"},"timestamp":[1760000000,1760604800],
			"indicators":{"quote":[{"open":[1700,null],"high":[1720,null],"low":[1690,null],"close":[1710,null],"volume":[100,null]}]}}],"error":null}}`))
	})

	series, err := provider.GetCandles(context.Background(), "FPT", domain.CandleQuery{Timeframe: domain.Timeframe1W, From: time.Now().AddDate(0, -1, 0), To: time.Now()})
	if err != nil {
		t.Fatalf("get candles error: %v", err)
	}
	if series.Currency != "VND" || len(series.Candles) != 2 || series.Candles[0].Close != 1710 || series.Candles[1].Close != 0 {
		t.Fatalf("unexpected series: %#v", series)
	}
	if cleaned := domain.CleanCandles(series.Candles); len(cleaned) != 1 {
		t.Fatalf("expected the null bar to be dropped, got %#v", cleaned)
	}
}

func TestGivenChartErrorWhenGetCandlesThenReturnsTheReason(t *testing.T) {
	provider := newTestProvider(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"chart":{"result":null,"error":{"code":"Not Found","description":"No data found, symbol may be delisted"}}}`))
	})

	_, err := provider.GetCandles(context.Background(), "ABC", domain.CandleQuery{Timeframe: domain.Timeframe1D})
	if err == nil || !strings.Contains(err.Error(), "delisted") {
		t.Fatalf("expected the chart error, got %v", err)
	}
}

func TestGivenSymbolsWhenMappingToYahooThenUsesVNSuffixAndRejectsVN30(t *testing.T) {
	cases := map[string]string{"FPT": "FPT.VN", "VNINDEX": "^VNINDEX.VN", "E1VFVN30.VN": "E1VFVN30.VN", "BTC-USD": "BTC-USD", "VIX": "^VIX", "USDVND": "USDVND=X", "XAUUSD": "GC=F"}
	for input, want := range cases {
		if got, err := yahooSymbol(input); err != nil || got != want {
			t.Fatalf("yahooSymbol(%s) = %s, %v; want %s", input, got, err, want)
		}
	}
	if _, err := yahooSymbol("VN30"); !errors.Is(err, port.ErrUnsupported) {
		t.Fatal("expected VN30 to be unsupported")
	}
}

func TestGivenIndexSymbolWhenGetDailyCandlesThenBuildsThemFromHourlyBars(t *testing.T) {
	to := time.Date(2026, 10, 12, 10, 0, 0, 0, time.UTC)
	provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if r.URL.Path != "/v8/finance/chart/^VNINDEX.VN" || query.Get("interval") != "1h" {
			t.Fatalf("unexpected request: %s", r.URL.String())
		}
		if query.Get("period1") != "1728813600" {
			t.Fatalf("expected the lookback clamped to Yahoo's hourly history, got %s", query.Get("period1"))
		}
		// 9 Oct 09:00 and 14:00 ICT, then 10 Oct 09:00 ICT.
		_, _ = w.Write([]byte(`{"chart":{"result":[{"meta":{"currency":"VND"},"timestamp":[1791511200,1791529200,1791597600],
			"indicators":{"quote":[{"open":[1700,1705,1712],"high":[1706,1711,1720],"low":[1698,1702,1709],"close":[1705,1710,1718],"volume":[1,2,3]}]}}]}}`))
	})

	series, err := provider.GetCandles(context.Background(), "VNINDEX", domain.CandleQuery{Timeframe: domain.Timeframe1D, From: to.AddDate(-5, 0, 0), To: to})
	if err != nil {
		t.Fatalf("get candles error: %v", err)
	}
	if len(series.Candles) != 2 {
		t.Fatalf("expected two daily bars, got %#v", series.Candles)
	}
	day := series.Candles[0]
	if day.Open != 1700 || day.High != 1711 || day.Low != 1698 || day.Close != 1710 || day.Volume != 3 {
		t.Fatalf("unexpected daily bar: %#v", day)
	}
}

func TestGivenFXPairWhenGetExchangeRateThenReturnsMarketQuote(t *testing.T) {
	provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v8/finance/chart/USDVND=X" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"chart":{"result":[{"meta":{"currency":"VND","regularMarketPrice":25934,"regularMarketTime":1791823806},
			"timestamp":[],"indicators":{"quote":[{}]}}]}}`))
	})

	rate, err := provider.GetExchangeRate(context.Background(), "usd", "vnd")
	if err != nil {
		t.Fatalf("exchange rate error: %v", err)
	}
	if rate.Rate != 25934 || rate.From != "USD" || rate.To != "VND" || rate.Timestamp != "2026-10-12T16:50:06Z" {
		t.Fatalf("unexpected rate: %#v", rate)
	}
	if _, err := provider.GetExchangeRate(context.Background(), "USDT", "VND"); err == nil {
		t.Fatal("expected coins to be unsupported")
	}
}

func TestGivenForexPairWhenGetPriceSeriesThenReturnsDailyCloses(t *testing.T) {
	provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v8/finance/chart/USDVND=X" || r.URL.Query().Get("interval") != "1d" {
			t.Fatalf("unexpected request %s", r.URL.String())
		}
		_, _ = w.Write([]byte(`{"chart":{"result":[{"meta":{"currency":"VND"},"timestamp":[1791763200,1791849600],
			"indicators":{"quote":[{"open":[25900,null],"high":[25950,null],"low":[25880,null],"close":[25934,null]}]}}]}}`))
	})

	series, err := provider.GetPriceSeries(context.Background(), "usd_vnd", domain.SeriesTypeForex)
	if err != nil {
		t.Fatalf("price series error: %v", err)
	}
	data := series.Data.(map[string]any)
	points := data["points"].([]domain.PricePoint)
	if data["pair"] != "USD_VND" || len(points) != 1 || points[0].Price != 25934 || series.Currency != "VND" {
		t.Fatalf("unexpected series: %#v", series)
	}
}
