package fiber

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

func TestGivenTickerRequestWhenMarketHandlerCalledThenReturnsTickerPayload(t *testing.T) {
	app := fiber.New()
	handler := NewMarketHandler(&fakeMarketService{
		ticker: domain.Ticker{
			Symbol:   "ACB",
			Type:     domain.TickerTypeEquity,
			Provider: "vnstock",
		},
	}, logger.NewTestLogger(t))
	app.Get("/api/market/ticker", handler.GetTicker)

	request := httptest.NewRequest(http.MethodGet, "/api/market/ticker?symbol=ACB&type=equity", nil)
	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("fiber test request failed: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}
}

func TestGivenMissingTickerQueryWhenMarketHandlerCalledThenReturnsBadRequest(t *testing.T) {
	app := fiber.New()
	handler := NewMarketHandler(&fakeMarketService{}, logger.NewTestLogger(t))
	app.Get("/api/market/ticker", handler.GetTicker)

	request := httptest.NewRequest(http.MethodGet, "/api/market/ticker?symbol=ACB", nil)
	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("fiber test request failed: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.StatusCode)
	}
}

func TestGivenPriceSeriesRequestWhenMarketHandlerCalledThenReturnsMetadata(t *testing.T) {
	app := fiber.New()
	handler := NewMarketHandler(&fakeMarketService{
		priceSeries: domain.PriceSeries{
			Provider:   "fmarket",
			SeriesType: domain.SeriesTypeGoldUSD,
			Currency:   "USD",
			Data: []domain.PricePoint{
				{Price: 2000.5, Timestamp: "2026-03-30T10:00:00Z"},
			},
		},
	}, logger.NewTestLogger(t))
	app.Get("/api/market/price-series", handler.GetPriceSeries)

	request := httptest.NewRequest(http.MethodGet, "/api/market/price-series?symbol=GOLD&type=gold_usd", nil)
	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("fiber test request failed: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	var payload domain.PriceSeries
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Provider != "fmarket" || payload.SeriesType != domain.SeriesTypeGoldUSD || payload.Currency != "USD" {
		t.Fatalf("unexpected series metadata: %#v", payload)
	}
}

func TestGivenSkipCacheQueryWhenTickerCalledThenContextBypassesCache(t *testing.T) {
	fake := &fakeMarketService{
		ticker: domain.Ticker{
			Symbol:   "ACB",
			Type:     domain.TickerTypeEquity,
			Provider: "vnstock",
		},
	}
	app := fiber.New()
	handler := NewMarketHandler(fake, logger.NewTestLogger(t))
	app.Get("/api/market/ticker", handler.GetTicker)

	request := httptest.NewRequest(http.MethodGet, "/api/market/ticker?symbol=ACB&type=equity&skipCache=true", nil)
	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("fiber test request failed: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}
	if !fake.lastBypassCache {
		t.Fatalf("expected skipCache context flag to be true")
	}
}

type fakeMarketService struct {
	ticker          domain.Ticker
	exchangeRate    domain.ExchangeRate
	priceSeries     domain.PriceSeries
	bankRates       []domain.BankRate
	health          domain.MarketProviderHealth
	err             error
	lastBypassCache bool
	candleCalls     []string
	staleAt         time.Time
}

func (f *fakeMarketService) Health(_ context.Context, _ string) (domain.MarketProviderHealth, error) {
	return f.health, f.err
}

func (f *fakeMarketService) GetTicker(ctx context.Context, _ string, _ domain.TickerType) (domain.Ticker, error) {
	f.lastBypassCache = domain.ShouldBypassCache(ctx)
	if !f.staleAt.IsZero() {
		domain.MarkStale(ctx, f.staleAt)
	}
	return f.ticker, f.err
}

func (f *fakeMarketService) GetExchangeRate(_ context.Context, _, _ string) (domain.ExchangeRate, error) {
	return f.exchangeRate, f.err
}

func (f *fakeMarketService) GetPriceSeries(_ context.Context, _ string, _ domain.SeriesType) (domain.PriceSeries, error) {
	return f.priceSeries, f.err
}

func (f *fakeMarketService) GetBankInterestRate(_ context.Context) ([]domain.BankRate, error) {
	return f.bankRates, f.err
}

func (f *fakeMarketService) GetCandles(_ context.Context, symbol string, timeframe domain.Timeframe, days int) (domain.CandleSeries, error) {
	f.candleCalls = append(f.candleCalls, fmt.Sprintf("%s/%s/%d", symbol, timeframe, days))
	return domain.CandleSeries{Symbol: symbol, Timeframe: timeframe}, f.err
}

func TestGivenCandleQueriesWhenGetCandlesThenValidatesAndMapsErrors(t *testing.T) {
	cases := []struct {
		name   string
		query  string
		err    error
		status int
		call   string
	}{
		{"defaults to daily", "symbol=FPT", nil, http.StatusOK, "FPT/1d/0"},
		{"passes timeframe and days", "symbol=FPT&timeframe=4H&days=30", nil, http.StatusOK, "FPT/4h/30"},
		{"missing symbol", "timeframe=1d", nil, http.StatusBadRequest, ""},
		{"unknown timeframe", "symbol=FPT&timeframe=2h", nil, http.StatusBadRequest, ""},
		{"negative days", "symbol=FPT&days=-1", nil, http.StatusBadRequest, ""},
		{"validation from the service", "symbol=FPT", fmt.Errorf("%w: bad", domain.ErrValidation), http.StatusBadRequest, "FPT/1d/0"},
		{"every provider failed", "symbol=FPT", errors.New("all failed"), http.StatusBadGateway, "FPT/1d/0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			service := &fakeMarketService{err: tc.err}
			app := fiber.New()
			app.Get("/api/market/candles", NewMarketHandler(service, logger.NewTestLogger(t)).GetCandles)

			response, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/market/candles?"+tc.query, nil), -1)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer response.Body.Close()
			if response.StatusCode != tc.status {
				t.Fatalf("expected %d, got %d", tc.status, response.StatusCode)
			}
			if tc.call == "" && len(service.candleCalls) != 0 || tc.call != "" && (len(service.candleCalls) != 1 || service.candleCalls[0] != tc.call) {
				t.Fatalf("expected call %q, got %v", tc.call, service.candleCalls)
			}
		})
	}
}

type fakeFmarketService struct {
	err error
}

func (f *fakeFmarketService) RunAction(_ context.Context, _ domain.FmarketRequest, _ bool) (any, error) {
	return map[string]any{"data": []any{}}, f.err
}

// Errors from the market data server keep their class: its 400 and 404
// answers wrap ErrValidation and ErrNotFound, anything else is a 502.
func TestGivenServiceErrorsWhenMarketEndpointsCalledThenMapsStatus(t *testing.T) {
	errorCases := []struct {
		name   string
		err    error
		status int
	}{
		{"validation", fmt.Errorf("%w: bad symbol", domain.ErrValidation), http.StatusBadRequest},
		{"not found", fmt.Errorf("%w: no fund", domain.ErrNotFound), http.StatusNotFound},
		{"provider failure", errors.New("every source failed"), http.StatusBadGateway},
	}
	endpoints := []struct {
		method, path string
		register     func(app *fiber.App, market *fakeMarketService, fmarket *fakeFmarketService, log *logger.Logger)
		body         string
	}{
		{http.MethodGet, "/api/market/ticker?symbol=FPT&type=equity", func(app *fiber.App, m *fakeMarketService, _ *fakeFmarketService, log *logger.Logger) {
			app.Get("/api/market/ticker", NewMarketHandler(m, log).GetTicker)
		}, ""},
		{http.MethodGet, "/api/market/exchange-rate?from=USD&to=VND", func(app *fiber.App, m *fakeMarketService, _ *fakeFmarketService, log *logger.Logger) {
			app.Get("/api/market/exchange-rate", NewMarketHandler(m, log).GetExchangeRate)
		}, ""},
		{http.MethodGet, "/api/market/price-series?symbol=GOLD&type=gold_usd", func(app *fiber.App, m *fakeMarketService, _ *fakeFmarketService, log *logger.Logger) {
			app.Get("/api/market/price-series", NewMarketHandler(m, log).GetPriceSeries)
		}, ""},
		{http.MethodGet, "/api/market/bank-rates", func(app *fiber.App, m *fakeMarketService, _ *fakeFmarketService, log *logger.Logger) {
			app.Get("/api/market/bank-rates", NewMarketHandler(m, log).GetBankInterestRate)
		}, ""},
		{http.MethodPost, "/api/fmarket/action", func(app *fiber.App, m *fakeMarketService, f *fakeFmarketService, log *logger.Logger) {
			app.Post("/api/fmarket/action", NewFmarketHandler(f, m, log).Action)
		}, `{"action":"getNavHistory","params":{}}`},
		{http.MethodGet, "/api/fmarket/health", func(app *fiber.App, m *fakeMarketService, f *fakeFmarketService, log *logger.Logger) {
			app.Get("/api/fmarket/health", NewFmarketHandler(f, m, log).Health)
		}, ""},
	}
	for _, endpoint := range endpoints {
		for _, tc := range errorCases {
			t.Run(endpoint.path+"/"+tc.name, func(t *testing.T) {
				app := fiber.New()
				endpoint.register(app, &fakeMarketService{err: tc.err}, &fakeFmarketService{err: tc.err}, logger.NewTestLogger(t))
				request := httptest.NewRequest(endpoint.method, endpoint.path, strings.NewReader(endpoint.body))
				request.Header.Set("Content-Type", "application/json")

				response, err := app.Test(request, -1)
				if err != nil {
					t.Fatalf("request failed: %v", err)
				}
				defer response.Body.Close()
				if response.StatusCode != tc.status {
					t.Fatalf("expected %d, got %d", tc.status, response.StatusCode)
				}
			})
		}
	}
}
