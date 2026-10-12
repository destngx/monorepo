package market_provider

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"errors"
	"testing"
	"time"
)

type ttlRecordingCache struct {
	fakeMarketCacheClient
	ttls map[string]int
}

func (c *ttlRecordingCache) Set(ctx context.Context, key string, value string, ttl int) error {
	c.ttls[key] = ttl
	return c.fakeMarketCacheClient.Set(ctx, key, value, ttl)
}

func newTTLCache() *ttlRecordingCache {
	return &ttlRecordingCache{fakeMarketCacheClient: fakeMarketCacheClient{store: map[string]string{}}, ttls: map[string]int{}}
}

func TestGivenTheDefaultRoutingWhenCachingEachCapabilityThenItUsesItsOwnTTL(t *testing.T) {
	cache := newTTLCache()
	provider := &fakeMarketProvider{
		name:         "fmarket",
		ticker:       domain.Ticker{Symbol: "FPT", Price: 1},
		exchangeRate: domain.ExchangeRate{From: "USD", To: "VND", Rate: 25000, Timestamp: time.Now().UTC().Format(time.RFC3339)},
		priceSeries:  domain.PriceSeries{Data: []domain.PricePoint{{Price: 1}}},
		bankRates:    []domain.BankRate{{}},
		candles:      domain.CandleSeries{Candles: []domain.Candle{{Time: time.Now().AddDate(0, 0, -1), Open: 1, High: 1, Low: 1, Close: 1}}},
	}
	routing := domain.MarketRoutingConfig{
		GetTicker:           map[domain.TickerType][]string{domain.TickerTypeEquity: {"fmarket"}},
		GetExchangeRate:     map[string][]string{"USD_VND": {"fmarket"}},
		GetPriceSeries:      map[domain.SeriesType][]string{domain.SeriesTypeEquity: {"fmarket"}},
		GetBankInterestRate: []string{"fmarket"},
		GetCandles:          []string{"fmarket"},
	}
	service := NewServiceWithRouting(routing, cache, logger.NewTestLogger(t), provider)
	ctx := context.Background()

	if _, err := service.GetTicker(ctx, "FPT", domain.TickerTypeEquity); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetExchangeRate(ctx, "USD", "VND"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetPriceSeries(ctx, "FPT", domain.SeriesTypeEquity); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetBankInterestRate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetCandles(ctx, "FPT", domain.Timeframe1D, 30); err != nil {
		t.Fatal(err)
	}

	want := map[string]int{"getTicker": 60, "getExchangeRate": 600, "getPriceSeries": 300, "getBankInterestRate": 3600, "getCandles": 300}
	for key, ttl := range cache.ttls {
		if len(key) > 6 && key[:6] == "stale:" {
			if ttl != domain.LastGoodTTLSeconds {
				t.Fatalf("%s ttl = %d, want 7 days", key, ttl)
			}
			continue
		}
		capability := key[len("market:"):]
		for name, expected := range want {
			if len(capability) >= len(name) && capability[:len(name)] == name {
				if ttl != expected {
					t.Fatalf("%s ttl = %d, want %d", key, ttl, expected)
				}
				delete(want, name)
			}
		}
	}
	if len(want) != 0 {
		t.Fatalf("capabilities not cached: %v (ttls %v)", want, cache.ttls)
	}
}

func TestGivenEveryProviderDownAndALastGoodQuoteWhenGettingATickerThenItIsServedAndMarkedStale(t *testing.T) {
	cache := newTTLCache()
	provider := &fakeMarketProvider{name: "vnstock", ticker: domain.Ticker{Symbol: "FPT", Price: 120000}}
	service := NewServiceWithRouting(
		domain.MarketRoutingConfig{GetTicker: map[domain.TickerType][]string{domain.TickerTypeEquity: {"vnstock"}}},
		cache, logger.NewTestLogger(t), provider,
	).(*marketProviderService)
	readAt := time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return readAt }
	if _, err := service.GetTicker(context.Background(), "FPT", domain.TickerTypeEquity); err != nil {
		t.Fatal(err)
	}

	provider.tickerErr = errors.New("upstream 502")
	recorder := &domain.StaleRecorder{}
	ctx := domain.WithBypassCache(domain.WithStaleRecorder(context.Background(), recorder), true)
	ticker, err := service.GetTicker(ctx, "FPT", domain.TickerTypeEquity)

	if err != nil || ticker.Price != 120000 {
		t.Fatalf("ticker = %+v err = %v, want the last good quote", ticker, err)
	}
	if asOf, stale := recorder.AsOf(); !stale || !asOf.Equal(readAt) {
		t.Fatalf("stale = %v asOf = %v, want %v", stale, asOf, readAt)
	}
}

func TestGivenEveryProviderDownAndNoLastGoodQuoteWhenGettingATickerThenItFails(t *testing.T) {
	provider := &fakeMarketProvider{name: "vnstock", tickerErr: errors.New("upstream 502")}
	service := NewServiceWithRouting(
		domain.MarketRoutingConfig{GetTicker: map[domain.TickerType][]string{domain.TickerTypeEquity: {"vnstock"}}},
		newTTLCache(), logger.NewTestLogger(t), provider,
	)
	recorder := &domain.StaleRecorder{}

	if _, err := service.GetTicker(domain.WithStaleRecorder(context.Background(), recorder), "FPT", domain.TickerTypeEquity); err == nil {
		t.Fatal("want the provider error")
	}
	if _, stale := recorder.AsOf(); stale {
		t.Fatal("nothing was served stale")
	}
}

func TestGivenNoProviderCoversASymbolWhenALastGoodCopyExistsThenTheValidationErrorIsReturned(t *testing.T) {
	cache := newTTLCache()
	cache.store[domain.LastGoodKey("market:getTicker:equity:FPT")] = `{"asOf":"2026-10-01T00:00:00Z","value":{"symbol":"FPT"}}`
	service := NewServiceWithRouting(
		domain.MarketRoutingConfig{GetTicker: map[domain.TickerType][]string{domain.TickerTypeEquity: {"vnstock"}}},
		cache, logger.NewTestLogger(t), &fakeMarketProvider{name: "vnstock", tickerErr: port.ErrUnsupported},
	)

	if _, err := service.GetTicker(context.Background(), "FPT", domain.TickerTypeEquity); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
}

func TestGivenEveryCandleProviderDownWhenALastGoodSeriesExistsThenItIsServedStale(t *testing.T) {
	cache := newTTLCache()
	provider := &fakeMarketProvider{name: "vnstock", candles: domain.CandleSeries{Candles: []domain.Candle{{Time: time.Now().AddDate(0, 0, -1), Open: 1, High: 2, Low: 1, Close: 2}}}}
	service := NewServiceWithRouting(domain.MarketRoutingConfig{GetCandles: []string{"vnstock"}}, cache, logger.NewTestLogger(t), provider)
	if _, err := service.GetCandles(context.Background(), "FPT", domain.Timeframe1D, 30); err != nil {
		t.Fatal(err)
	}
	for key := range cache.store {
		if key[:6] != "stale:" {
			delete(cache.store, key)
		}
	}

	provider.candlesErr = errors.New("upstream timeout")
	recorder := &domain.StaleRecorder{}
	series, err := service.GetCandles(domain.WithStaleRecorder(context.Background(), recorder), "FPT", domain.Timeframe1D, 30)

	if err != nil || len(series.Candles) != 1 {
		t.Fatalf("series = %+v err = %v", series, err)
	}
	if _, stale := recorder.AsOf(); !stale {
		t.Fatal("the candles were served stale")
	}
}
