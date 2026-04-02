package fiber

import (
	"apps/wealth-management-engine/adapter/logger"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestGivenProviderHealthRouteWhenUnknownProviderThenReturnsBadGateway(t *testing.T) {
	app := fiber.New()
	handler := NewMarketProviderHandler(&fakeMarketService{
		err: context.Canceled,
	}, logger.NewTestLogger(t))
	app.Get("/api/external/market/providers/:provider/health", handler.Health)

	request := httptest.NewRequest(http.MethodGet, "/api/external/market/providers/unknown/health", nil)
	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("fiber test request failed: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d", response.StatusCode)
	}
}
