package service

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// maxCandleFetches bounds concurrent candle calls across providers.
const maxCandleFetches = 4

// RiskHorizonDays is the simulation horizon: about one quarter of sessions.
const RiskHorizonDays = 63

// marketLocation is Vietnam time, used to align bars from different venues.
var marketLocation = time.FixedZone("ICT", 7*60*60)

type marketAnalysisService struct {
	market port.MarketProviderService
	now    func() time.Time
}

func NewMarketAnalysisService(market port.MarketProviderService, now func() time.Time) port.MarketAnalysisService {
	if now == nil {
		now = time.Now
	}
	return &marketAnalysisService{market: market, now: now}
}

// Pulse fetches every pulse instrument for the timeframe's WM-RULE-9
// lookback. Instruments that fail are listed with their error; the call
// fails only when nothing could be fetched.
func (s *marketAnalysisService) Pulse(ctx context.Context, timeframe domain.Timeframe) (domain.MarketPulse, error) {
	timeframe, err := domain.ParseTimeframe(string(timeframe))
	if err != nil {
		return domain.MarketPulse{}, err
	}
	var mu sync.Mutex
	series := map[string]domain.CandleSeries{}
	failures := map[string]string{}
	group := newBoundedGroup(maxCandleFetches)
	for _, asset := range domain.PulseAssets {
		group.Go(func() {
			result, err := s.market.GetCandles(ctx, asset.Symbol, timeframe, timeframe.PulseLookbackDays())
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				failures[asset.Symbol] = err.Error()
				return
			}
			series[asset.Symbol] = result
		})
	}
	group.Wait()
	if len(series) == 0 {
		return domain.MarketPulse{}, fmt.Errorf("market pulse: no instrument could be fetched: %s", joinFailures(failures))
	}
	return domain.BuildPulse(timeframe, series, failures, marketLocation, s.now()), nil
}

// Analysis fetches all four timeframes. The requested one must succeed; the
// others only feed the confluence matrix. Seasonality and risk need the
// daily series.
func (s *marketAnalysisService) Analysis(ctx context.Context, symbol string, timeframe domain.Timeframe) (domain.MarketAnalysis, error) {
	symbol = domain.NormaliseMarketSymbol(symbol)
	if symbol == "" {
		return domain.MarketAnalysis{}, fmt.Errorf("%w: symbol is required", domain.ErrValidation)
	}
	timeframe, err := domain.ParseTimeframe(string(timeframe))
	if err != nil {
		return domain.MarketAnalysis{}, err
	}

	var mu sync.Mutex
	series := map[domain.Timeframe]domain.CandleSeries{}
	failures := map[domain.Timeframe]error{}
	group := newBoundedGroup(maxCandleFetches)
	for _, tf := range domain.Timeframes {
		group.Go(func() {
			result, err := s.market.GetCandles(ctx, symbol, tf, tf.AnalysisLookbackDays())
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				failures[tf] = err
				return
			}
			series[tf] = result
		})
	}
	group.Wait()

	primary, ok := series[timeframe]
	if !ok {
		return domain.MarketAnalysis{}, fmt.Errorf("%s %s candles: %w", symbol, timeframe, failures[timeframe])
	}
	analysis := domain.MarketAnalysis{
		Symbol:      symbol,
		Timeframe:   timeframe,
		Provider:    primary.Provider,
		Source:      primary.Source,
		Currency:    primary.Currency,
		Technicals:  domain.ComputeTechnicals(primary.Candles),
		Seasonality: []domain.SeasonalityBucket{},
		Notes:       []string{},
		GeneratedAt: s.now(),
	}

	views := map[domain.Timeframe]domain.TimeframeView{}
	for _, tf := range domain.Timeframes {
		if err := failures[tf]; err != nil {
			views[tf] = domain.TimeframeView{Timeframe: tf, Error: err.Error()}
			continue
		}
		technicals := analysis.Technicals
		if tf != timeframe {
			technicals = domain.ComputeTechnicals(series[tf].Candles)
		}
		views[tf] = domain.TimeframeView{
			Timeframe: tf,
			Provider:  series[tf].Provider,
			Source:    series[tf].Source,
			Candles:   technicals.Candles,
			Price:     technicals.Price,
			Trend:     technicals.Trend,
			Phase:     technicals.Phase,
			RSI:       technicals.Indicators.RSI,
		}
	}
	analysis.Confluence = domain.BuildConfluence(views)

	daily, ok := series[domain.Timeframe1D]
	if !ok {
		analysis.Notes = append(analysis.Notes, "daily candles unavailable: no seasonality or risk statistics")
		return analysis, nil
	}
	analysis.Seasonality = domain.Seasonality(daily.Candles, marketLocation)
	risk, err := domain.ComputeRiskStats(symbol, daily.Candles, RiskHorizonDays)
	if err != nil {
		if !errors.Is(err, domain.ErrValidation) {
			return domain.MarketAnalysis{}, err
		}
		analysis.Notes = append(analysis.Notes, "risk statistics: "+err.Error())
		return analysis, nil
	}
	if risk.Simulation == nil {
		analysis.Notes = append(analysis.Notes, "less than six months of daily history: no simulation")
	}
	analysis.Risk = &risk
	return analysis, nil
}

func joinFailures(failures map[string]string) string {
	parts := make([]string, 0, len(failures))
	for _, asset := range domain.PulseAssets {
		if reason, ok := failures[asset.Symbol]; ok {
			parts = append(parts, asset.Symbol+": "+reason)
		}
	}
	return strings.Join(parts, "; ")
}
