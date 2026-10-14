package fiber

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestGivenProviderHealthErrorsWhenHealthCalledThenMapsStatus(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
	}{
		{"unknown provider", fmt.Errorf("%w: unknown market provider: ghost", domain.ErrNotFound), http.StatusNotFound},
		{"provider down", errors.New("connection refused"), http.StatusBadGateway},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()
			handler := NewMarketProviderHandler(&fakeMarketService{err: tc.err}, logger.NewTestLogger(t))
			app.Get("/api/external/market/providers/:provider/health", handler.Health)

			request := httptest.NewRequest(http.MethodGet, "/api/external/market/providers/ghost/health", nil)
			response, err := app.Test(request, -1)
			if err != nil {
				t.Fatalf("fiber test request failed: %v", err)
			}
			defer response.Body.Close()
			if response.StatusCode != tc.status {
				t.Fatalf("expected status %d, got %d", tc.status, response.StatusCode)
			}
		})
	}
}
