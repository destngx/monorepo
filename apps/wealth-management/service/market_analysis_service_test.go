package service

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"testing"
	"time"
)

// candleMarket serves generated candles per symbol and timeframe and
// records each request.
type candleMarket struct {
	port.MarketProviderService
	mu       sync.Mutex
	series   map[string][]domain.Candle
	failures map[string]error
	// fallback lists the providers that failed before "test", per request key.
	fallback map[string][]string
	requests []string
}

func (m *candleMarket) GetCandles(_ context.Context, symbol string, timeframe domain.Timeframe, days int) (domain.CandleSeries, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := symbol + "/" + string(timeframe)
	m.requests = append(m.requests, fmt.Sprintf("%s/%d", key, days))
	if err := m.failures[key]; err != nil {
		return domain.CandleSeries{}, err
	}
	candles, ok := m.series[key]
	if !ok {
		candles, ok = m.series[symbol]
	}
	if !ok {
		return domain.CandleSeries{}, errors.New("no data")
	}
	source := &domain.DataSource{Provider: "test", Fallback: m.fallback[key]}
	return domain.CandleSeries{Symbol: symbol, Timeframe: timeframe, Provider: "test", Currency: "VND", Candles: candles, Source: source}, nil
}

var analysisStart = time.Date(2025, 1, 1, 2, 0, 0, 0, time.UTC)

// trendingDays is a gently rising daily series with noise.
func trendingDays(n int, step time.Duration) []domain.Candle {
	candles := make([]domain.Candle, n)
	for i := range candles {
		close := 100 * math.Exp(0.001*float64(i)+0.01*math.Sin(float64(i)))
		candles[i] = domain.Candle{Time: analysisStart.Add(time.Duration(i) * step), Open: close, High: close * 1.01, Low: close * 0.99, Close: close}
	}
	return candles
}

func TestGivenEveryTimeframeWhenAnalysisThenBuildsConfluenceSeasonalityAndRisk(t *testing.T) {
	market := &candleMarket{series: map[string][]domain.Candle{"FPT": trendingDays(400, 24*time.Hour)}}
	now := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	analysis, err := NewMarketAnalysisService(market, func() time.Time { return now }).Analysis(context.Background(), " fpt ", domain.Timeframe1D)
	if err != nil {
		t.Fatalf("analysis error: %v", err)
	}
	if analysis.Symbol != "FPT" || analysis.Provider != "test" || analysis.Technicals.Candles != 400 || !analysis.GeneratedAt.Equal(now) {
		t.Fatalf("unexpected analysis header %#v", analysis)
	}
	if len(analysis.Confluence.Views) != 4 || len(analysis.Seasonality) == 0 || analysis.Risk == nil || analysis.Risk.Simulation == nil {
		t.Fatalf("expected confluence, seasonality and a simulation, got %#v", analysis)
	}
	if analysis.Risk.Simulation.Horizon != RiskHorizonDays || len(analysis.Notes) != 0 {
		t.Fatalf("unexpected risk %#v notes %#v", analysis.Risk, analysis.Notes)
	}
	if len(market.requests) != 4 {
		t.Fatalf("expected one request per timeframe, got %v", market.requests)
	}
	if analysis.Source == nil || analysis.Source.Provider != "test" {
		t.Fatalf("expected the candles' source on the analysis, got %#v", analysis.Source)
	}
	for _, request := range market.requests {
		if !strings.HasPrefix(request, "FPT/1d/730") && !strings.HasPrefix(request, "FPT/1h/60") && !strings.HasPrefix(request, "FPT/4h/180") && !strings.HasPrefix(request, "FPT/1w/1825") {
			t.Fatalf("unexpected lookback in %v", market.requests)
		}
	}
}

func TestGivenSecondaryTimeframeFailsWhenAnalysisThenMarksViewAndNotes(t *testing.T) {
	market := &candleMarket{
		series:   map[string][]domain.Candle{"VN30/1h": trendingDays(100, time.Hour)},
		failures: map[string]error{"VN30/1d": errors.New("vnstock down")},
	}
	analysis, err := NewMarketAnalysisService(market, nil).Analysis(context.Background(), "VN30", domain.Timeframe1H)
	if err != nil {
		t.Fatalf("analysis error: %v", err)
	}
	if analysis.Risk != nil || len(analysis.Notes) != 1 || len(analysis.Confluence.Views) != 4 {
		t.Fatalf("expected the daily gap noted, got %#v", analysis)
	}
	for _, view := range analysis.Confluence.Views {
		if view.Timeframe == domain.Timeframe1D && view.Error != "vnstock down" {
			t.Fatalf("expected the daily error on its view, got %#v", view)
		}
	}
}

func TestGivenRequestedTimeframeFailsWhenAnalysisThenFailsWithCause(t *testing.T) {
	cause := fmt.Errorf("%w: no provider covers SJC 1h", domain.ErrValidation)
	market := &candleMarket{failures: map[string]error{"SJC/1h": cause}}
	_, err := NewMarketAnalysisService(market, nil).Analysis(context.Background(), "SJC", domain.Timeframe1H)
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected the validation cause, got %v", err)
	}
	if _, err := NewMarketAnalysisService(market, nil).Analysis(context.Background(), " ", domain.Timeframe1D); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected a validation error for an empty symbol, got %v", err)
	}
}

func TestGivenSomeInstrumentsFailWhenPulseThenListsThemAndUsesRuleLookback(t *testing.T) {
	series := map[string][]domain.Candle{}
	for _, asset := range domain.PulseAssets {
		if asset.Symbol != domain.MarketSymbolVN30 {
			series[asset.Symbol] = trendingDays(20, 24*time.Hour)
		}
	}
	market := &candleMarket{series: series}
	pulse, err := NewMarketAnalysisService(market, nil).Pulse(context.Background(), domain.Timeframe1D)
	if err != nil {
		t.Fatalf("pulse error: %v", err)
	}
	if len(pulse.Assets) != len(domain.PulseAssets) || pulse.Assets[1].Error != "no data" {
		t.Fatalf("expected VN30 to carry its error, got %#v", pulse.Assets[1])
	}
	if len(market.requests) != len(domain.PulseAssets) || !strings.HasSuffix(market.requests[0], "/60") {
		t.Fatalf("expected one 60-day request per asset, got %v", market.requests)
	}

	empty := &candleMarket{}
	if _, err := NewMarketAnalysisService(empty, nil).Pulse(context.Background(), domain.Timeframe1D); err == nil || !strings.Contains(err.Error(), "VNINDEX: no data") {
		t.Fatalf("expected a failure listing every instrument, got %v", err)
	}
	if _, err := NewMarketAnalysisService(empty, nil).Pulse(context.Background(), domain.Timeframe("2h")); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected a validation error, got %v", err)
	}
}

func TestGivenAFallbackWhenAnalysisThenEachViewKeepsItsOwnSource(t *testing.T) {
	market := &candleMarket{
		series:   map[string][]domain.Candle{"FPT": trendingDays(400, 24*time.Hour)},
		fallback: map[string][]string{"FPT/4h": {"vnstock: quota"}},
	}
	analysis, err := NewMarketAnalysisService(market, nil).Analysis(context.Background(), "FPT", domain.Timeframe1D)
	if err != nil {
		t.Fatalf("analysis error: %v", err)
	}
	if len(analysis.Source.Fallback) != 0 {
		t.Fatalf("the daily answer did not fall back, got %#v", analysis.Source)
	}
	for _, view := range analysis.Confluence.Views {
		fellBack := view.Source != nil && len(view.Source.Fallback) == 1
		if fellBack != (view.Timeframe == domain.Timeframe4H) {
			t.Fatalf("view %s source %#v", view.Timeframe, view.Source)
		}
	}
}
