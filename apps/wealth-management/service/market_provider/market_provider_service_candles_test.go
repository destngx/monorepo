package market_provider

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func hourly(start time.Time, closes ...float64) []domain.Candle {
	candles := make([]domain.Candle, len(closes))
	for i, close := range closes {
		candles[i] = domain.Candle{Time: start.Add(time.Duration(i) * time.Hour), Open: close, High: close + 1, Low: close - 1, Close: close, Volume: 10}
	}
	return candles
}

func newCandleService(t *testing.T, cache *fakeMarketCacheClient, now time.Time, providers ...*fakeMarketProvider) *marketProviderService {
	t.Helper()
	registered := []port.MarketProvider{healthOnlyProvider{name: "basic"}}
	for _, provider := range providers {
		registered = append(registered, provider)
	}
	var cacheClient port.CacheClient
	if cache != nil {
		cacheClient = cache
	}
	routing := domain.MarketRoutingConfig{GetCandles: []string{"basic", "vnstock", "yahoo"}}
	service := NewServiceWithRouting(routing, cacheClient, logger.NewTestLogger(t), registered...).(*marketProviderService)
	service.now = func() time.Time { return now }
	return service
}

func TestGivenPrimaryFailsWhenGetCandlesThenFallbackServesWithDataSourceIndicator(t *testing.T) {
	now := time.Date(2026, 10, 12, 8, 0, 0, 0, time.UTC)
	vnstock := &fakeMarketProvider{name: "vnstock", candlesErr: errors.New("connection refused")}
	yahoo := &fakeMarketProvider{name: "yahoo", candles: domain.CandleSeries{Currency: "VND", Candles: hourly(now.Add(-48*time.Hour), 10, 11, 12)}}
	service := newCandleService(t, nil, now, vnstock, yahoo)

	series, err := service.GetCandles(context.Background(), " ^vnindex ", domain.Timeframe1D, 30)
	if err != nil {
		t.Fatalf("get candles error: %v", err)
	}
	if series.Provider != "yahoo" || series.Symbol != "VNINDEX" || series.Timeframe != domain.Timeframe1D || len(series.Candles) != 3 {
		t.Fatalf("unexpected series: %#v", series)
	}
	if series.Fallback != "vnstock: connection refused" {
		t.Fatalf("expected the fallback reason, got %q", series.Fallback)
	}
	query := vnstock.candleQueries[0]
	if query.Timeframe != domain.Timeframe1D || !query.To.Equal(now) || !query.From.Equal(now.AddDate(0, 0, -30)) {
		t.Fatalf("unexpected query: %#v", query)
	}
}

func TestGivenFourHourTimeframeWhenGetCandlesThenFetchesHourlyAndAggregates(t *testing.T) {
	now := time.Date(2026, 10, 12, 8, 0, 0, 0, time.UTC)
	// 09:00-12:00 ICT is one 08-12 bucket; 13:00-14:00 ICT is the 12-16 bucket.
	start := time.Date(2026, 10, 9, 2, 0, 0, 0, time.UTC)
	candles := append(hourly(start, 10, 12, 11), hourly(start.Add(4*time.Hour), 13, 14)...)
	vnstock := &fakeMarketProvider{name: "vnstock", candles: domain.CandleSeries{Candles: candles}}
	service := newCandleService(t, nil, now, vnstock)

	series, err := service.GetCandles(context.Background(), "FPT", domain.Timeframe4H, 0)
	if err != nil {
		t.Fatalf("get candles error: %v", err)
	}
	if vnstock.candleQueries[0].Timeframe != domain.Timeframe1H {
		t.Fatalf("expected 1h source candles, got %s", vnstock.candleQueries[0].Timeframe)
	}
	if !vnstock.candleQueries[0].From.Equal(now.AddDate(0, 0, -domain.Timeframe4H.AnalysisLookbackDays())) {
		t.Fatalf("expected the default lookback, got %s", vnstock.candleQueries[0].From)
	}
	if len(series.Candles) != 2 {
		t.Fatalf("expected two 4h bars, got %#v", series.Candles)
	}
	first := series.Candles[0]
	if first.Open != 10 || first.Close != 11 || first.High != 13 || first.Low != 9 || first.Volume != 30 {
		t.Fatalf("unexpected first bar: %#v", first)
	}
	if series.Fallback != "" {
		t.Fatalf("unexpected fallback: %q", series.Fallback)
	}
}

func TestGivenEmptyCandlesWhenGetCandlesThenTriesNextProviderAndCachesByNormalisedSymbol(t *testing.T) {
	now := time.Date(2026, 10, 12, 8, 0, 0, 0, time.UTC)
	vnstock := &fakeMarketProvider{name: "vnstock", candles: domain.CandleSeries{Candles: []domain.Candle{{Time: now, Close: 0}}}}
	yahoo := &fakeMarketProvider{name: "yahoo", candles: domain.CandleSeries{Candles: hourly(now.Add(-time.Hour), 5)}}
	cache := &fakeMarketCacheClient{store: map[string]string{}}
	service := newCandleService(t, cache, now, vnstock, yahoo)

	if _, err := service.GetCandles(context.Background(), "fpt", domain.Timeframe1D, 10); err != nil {
		t.Fatalf("get candles error: %v", err)
	}
	if _, ok := cache.store["market:getCandles:1d:FPT:10"]; !ok {
		t.Fatalf("expected a normalised cache key, got %v", cache.store)
	}
	series, err := service.GetCandles(context.Background(), "FPT", domain.Timeframe1D, 10)
	if err != nil || series.Provider != "yahoo" {
		t.Fatalf("expected the cached yahoo series, got %#v, %v", series, err)
	}
	if len(yahoo.candleQueries) != 1 {
		t.Fatalf("expected the second call to hit the cache, got %d calls", len(yahoo.candleQueries))
	}
}

func TestGivenInvalidInputWhenGetCandlesThenReturnsValidationError(t *testing.T) {
	service := newCandleService(t, nil, time.Now())
	if _, err := service.GetCandles(context.Background(), "  ", domain.Timeframe1D, 0); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected a validation error for an empty symbol, got %v", err)
	}
	if _, err := service.GetCandles(context.Background(), "FPT", domain.Timeframe("2h"), 0); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected a validation error for 2h, got %v", err)
	}
}

func TestGivenEveryProviderFailsWhenGetCandlesThenErrorListsEachReason(t *testing.T) {
	vnstock := &fakeMarketProvider{name: "vnstock", candlesErr: errors.New("down")}
	yahoo := &fakeMarketProvider{name: "yahoo", candlesErr: errors.New("not found")}
	service := newCandleService(t, nil, time.Now(), vnstock, yahoo)

	_, err := service.GetCandles(context.Background(), "FPT", domain.Timeframe1D, 0)
	if err == nil || !strings.Contains(err.Error(), "vnstock: down") || !strings.Contains(err.Error(), "yahoo: not found") {
		t.Fatalf("expected both reasons, got %v", err)
	}
}

func TestGivenProvidersNotCoveringSymbolWhenGetCandlesThenSkipsThemWithoutFallbackNoise(t *testing.T) {
	now := time.Date(2026, 10, 12, 8, 0, 0, 0, time.UTC)
	unsupported := fmt.Errorf("%w: SJC is not a VN listing", port.ErrUnsupported)
	vnstock := &fakeMarketProvider{name: "vnstock", candlesErr: unsupported}
	yahoo := &fakeMarketProvider{name: "yahoo", candles: domain.CandleSeries{Candles: hourly(now.Add(-time.Hour), 5)}}
	service := newCandleService(t, nil, now, vnstock, yahoo)

	series, err := service.GetCandles(context.Background(), "SJC", domain.Timeframe1D, 10)
	if err != nil || series.Provider != "yahoo" || series.Fallback != "" {
		t.Fatalf("expected yahoo without a fallback note, got %#v, %v", series, err)
	}

	onlyUnsupported := newCandleService(t, nil, now,
		&fakeMarketProvider{name: "vnstock", candlesErr: unsupported},
		&fakeMarketProvider{name: "yahoo", candlesErr: unsupported})
	if _, err := onlyUnsupported.GetCandles(context.Background(), "SJC", domain.Timeframe1D, 10); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("no covering provider is a validation error, got %v", err)
	}
}
