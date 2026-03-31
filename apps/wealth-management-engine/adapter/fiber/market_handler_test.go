package fiber

import (
	"apps/wealth-management-engine/domain"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
	})
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
	handler := NewMarketHandler(&fakeMarketService{})
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
	})
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
	handler := NewMarketHandler(fake)
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
}

func (f *fakeMarketService) Health(_ context.Context, _ string) (domain.MarketProviderHealth, error) {
	return f.health, f.err
}

func (f *fakeMarketService) GetTicker(ctx context.Context, _ string, _ domain.TickerType) (domain.Ticker, error) {
	f.lastBypassCache = domain.ShouldBypassCache(ctx)
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
