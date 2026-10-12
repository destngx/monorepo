package fiber

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/internal/fakes"
	"net/http/httptest"
	"testing"
	"time"

	fiberLib "github.com/gofiber/fiber/v2"
)

func newStaleApp(reporter *fakes.Database, handler fiberLib.Handler) *fiberLib.App {
	stale := NewStaleDataMiddleware()
	if reporter != nil {
		stale.Watch(reporter)
	}
	app := fiberLib.New()
	app.Use(stale.Handler())
	app.Get("/data", handler)
	return app
}

func TestGivenAMarketCallServedStaleWhenRespondingThenTheHeaderCarriesItsTime(t *testing.T) {
	asOf := time.Date(2026, 10, 12, 8, 30, 0, 0, time.FixedZone("ICT", 7*3600))
	app := newStaleApp(nil, func(c *fiberLib.Ctx) error {
		domain.MarkStale(c.UserContext(), asOf)
		return c.SendString("ok")
	})

	response, err := app.Test(httptest.NewRequest("GET", "/data", nil))

	if err != nil {
		t.Fatal(err)
	}
	if got := response.Header.Get(StaleDataHeader); got != "2026-10-12T01:30:00Z" {
		t.Fatalf("%s = %q, want the UTC time of the stale data", StaleDataHeader, got)
	}
}

func TestGivenASheetReadServedStaleWhenRespondingThenTheOldestTimeIsReported(t *testing.T) {
	sheetAsOf := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	db := &fakes.Database{StaleAt: sheetAsOf}
	app := newStaleApp(db, func(c *fiberLib.Ctx) error {
		domain.MarkStale(c.UserContext(), sheetAsOf.Add(time.Hour))
		return c.SendString("ok")
	})

	response, err := app.Test(httptest.NewRequest("GET", "/data", nil))

	if err != nil {
		t.Fatal(err)
	}
	if got := response.Header.Get(StaleDataHeader); got != "2026-10-10T00:00:00Z" {
		t.Fatalf("%s = %q, want the oldest stale time", StaleDataHeader, got)
	}
}

func TestGivenFreshDataWhenRespondingThenThereIsNoStaleHeader(t *testing.T) {
	app := newStaleApp(&fakes.Database{}, func(c *fiberLib.Ctx) error { return c.SendString("ok") })

	response, err := app.Test(httptest.NewRequest("GET", "/data", nil))

	if err != nil {
		t.Fatal(err)
	}
	if got := response.Header.Get(StaleDataHeader); got != "" {
		t.Fatalf("%s = %q, want none", StaleDataHeader, got)
	}
}

func TestGivenATickerServedStaleWithSkipCacheWhenRequestedThenTheHeaderIsSet(t *testing.T) {
	asOf := time.Date(2026, 10, 13, 2, 0, 0, 0, time.UTC)
	market := &fakeMarketService{ticker: domain.Ticker{Symbol: "FPT"}, staleAt: asOf}
	stale := NewStaleDataMiddleware()
	app := fiberLib.New()
	app.Use(stale.Handler())
	app.Get("/api/market/ticker", NewMarketHandler(market, logger.NewTestLogger(t)).GetTicker)

	response, err := app.Test(httptest.NewRequest("GET", "/api/market/ticker?symbol=FPT&type=equity&skipCache=true", nil))

	if err != nil {
		t.Fatal(err)
	}
	if got := response.Header.Get(StaleDataHeader); got != "2026-10-13T02:00:00Z" || !market.lastBypassCache {
		t.Fatalf("%s = %q bypass = %v", StaleDataHeader, got, market.lastBypassCache)
	}
}
