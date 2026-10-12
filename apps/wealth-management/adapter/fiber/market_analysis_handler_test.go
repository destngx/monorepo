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
	"testing"

	"github.com/gofiber/fiber/v2"
)

type fakeMarketAnalysis struct {
	err       error
	symbol    string
	timeframe domain.Timeframe
	bypass    bool
}

func (f *fakeMarketAnalysis) Pulse(ctx context.Context, timeframe domain.Timeframe) (domain.MarketPulse, error) {
	f.timeframe, f.bypass = timeframe, domain.ShouldBypassCache(ctx)
	return domain.MarketPulse{Timeframe: timeframe}, f.err
}

func (f *fakeMarketAnalysis) Analysis(ctx context.Context, symbol string, timeframe domain.Timeframe) (domain.MarketAnalysis, error) {
	f.symbol, f.timeframe, f.bypass = symbol, timeframe, domain.ShouldBypassCache(ctx)
	return domain.MarketAnalysis{Symbol: symbol, Timeframe: timeframe}, f.err
}

func marketAnalysisApp(t *testing.T, analysis *fakeMarketAnalysis) *fiber.App {
	app := fiber.New()
	handler := NewMarketAnalysisHandler(analysis, logger.NewTestLogger(t))
	app.Get("/api/market/pulse", handler.Pulse)
	app.Get("/api/market/analysis", handler.Analysis)
	return app
}

func getStatus(t *testing.T, app *fiber.App, target string) *http.Response {
	t.Helper()
	response, err := app.Test(httptest.NewRequest(http.MethodGet, target, nil), -1)
	if err != nil {
		t.Fatalf("fiber test request failed: %v", err)
	}
	t.Cleanup(func() { response.Body.Close() })
	return response
}

func TestGivenForcedPulseRequestWhenHandledThenPassesTimeframeAndBypass(t *testing.T) {
	analysis := &fakeMarketAnalysis{}
	response := getStatus(t, marketAnalysisApp(t, analysis), "/api/market/pulse?timeframe=4h&force=true")
	if response.StatusCode != http.StatusOK || analysis.timeframe != domain.Timeframe4H || !analysis.bypass {
		t.Fatalf("got status %d timeframe %s bypass %v", response.StatusCode, analysis.timeframe, analysis.bypass)
	}
	var pulse domain.MarketPulse
	if err := json.NewDecoder(response.Body).Decode(&pulse); err != nil || pulse.Timeframe != domain.Timeframe4H {
		t.Fatalf("unexpected payload %#v %v", pulse, err)
	}
}

func TestGivenAnalysisRequestWhenHandledThenDefaultsToDailyWithCache(t *testing.T) {
	analysis := &fakeMarketAnalysis{}
	response := getStatus(t, marketAnalysisApp(t, analysis), "/api/market/analysis?symbol=FPT")
	if response.StatusCode != http.StatusOK || analysis.symbol != "FPT" || analysis.timeframe != domain.Timeframe1D || analysis.bypass {
		t.Fatalf("got status %d symbol %q timeframe %s bypass %v", response.StatusCode, analysis.symbol, analysis.timeframe, analysis.bypass)
	}
}

func TestGivenInvalidInputWhenMarketAnalysisHandledThenReturnsBadRequest(t *testing.T) {
	app := marketAnalysisApp(t, &fakeMarketAnalysis{})
	for _, target := range []string{
		"/api/market/analysis",
		"/api/market/analysis?symbol=FPT&timeframe=2h",
		"/api/market/pulse?timeframe=5m",
	} {
		if response := getStatus(t, app, target); response.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", target, response.StatusCode)
		}
	}
}

func TestGivenServiceErrorWhenMarketAnalysisHandledThenMapsValidationAndUpstream(t *testing.T) {
	uncovered := &fakeMarketAnalysis{err: fmt.Errorf("%w: no provider covers SJC 1h", domain.ErrValidation)}
	if response := getStatus(t, marketAnalysisApp(t, uncovered), "/api/market/analysis?symbol=SJC&timeframe=1h"); response.StatusCode != http.StatusBadRequest {
		t.Fatalf("an uncovered symbol must be a 400, got %d", response.StatusCode)
	}
	down := &fakeMarketAnalysis{err: errors.New("vnstock: connection refused")}
	if response := getStatus(t, marketAnalysisApp(t, down), "/api/market/pulse"); response.StatusCode != http.StatusBadGateway {
		t.Fatalf("a provider outage must be a 502, got %d", response.StatusCode)
	}
}
