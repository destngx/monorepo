package domain

import (
	"testing"
	"time"
)

func series(symbol string, timeframe Timeframe, closes ...float64) CandleSeries {
	return CandleSeries{Symbol: symbol, Timeframe: timeframe, Provider: "test", Candles: dailyCandles(closes...)}
}

func TestGivenDailySeriesWhenQuotePulseThenComputesChangesAndMomentum(t *testing.T) {
	quote := QuotePulse(PulseAsset{"FPT", "FPT"}, series("FPT", Timeframe1D, 100, 100, 100, 100, 100, 100, 100, 100, 103))
	if !closeTo(quote.ChangePct, 3) || quote.Direction != TrendUp || quote.Momentum != MomentumFire {
		t.Fatalf("unexpected quote %#v", quote)
	}
	if quote.DayChangePct == nil || !closeTo(*quote.DayChangePct, 3) || quote.WeekChangePct == nil || !closeTo(*quote.WeekChangePct, 3) {
		t.Fatalf("expected day and week changes, got %#v", quote)
	}
	flat := QuotePulse(PulseAsset{"VCB", "VCB"}, series("VCB", Timeframe1D, 100, 100.04))
	if flat.Direction != "flat" || flat.Momentum != MomentumSleep || flat.WeekChangePct != nil {
		t.Fatalf("unexpected flat quote %#v", flat)
	}
	weekly := QuotePulse(PulseAsset{"ACB", "ACB"}, series("ACB", Timeframe1W, 100, 101))
	if weekly.DayChangePct != nil || weekly.Momentum != MomentumStable {
		t.Fatalf("weekly bars have no day change, got %#v", weekly)
	}
	if short := QuotePulse(PulseAsset{"X", "X"}, series("X", Timeframe1D, 1)); short.Error == "" {
		t.Fatal("one candle cannot be quoted")
	}
}

func TestGivenGoldUpAndIndexDownWhenBuildPulseThenDefensiveAndFallingKnife(t *testing.T) {
	data := map[string]CandleSeries{
		MarketSymbolVNIndex: series(MarketSymbolVNIndex, Timeframe1D, 1300, 1290),
		MarketSymbolVN30:    series(MarketSymbolVN30, Timeframe1D, 1400, 1370),
		MarketSymbolSJC:     series(MarketSymbolSJC, Timeframe1D, 140, 141),
		"FPT":               series("FPT", Timeframe1D, 100, 105),
	}
	pulse := BuildPulse(Timeframe1D, data, map[string]string{"VCB": "provider down"}, time.UTC, testDay)
	if len(pulse.Assets) != len(PulseAssets) {
		t.Fatalf("every pulse asset must be listed, got %d", len(pulse.Assets))
	}
	if pulse.Assets[2].Symbol != "VCB" || pulse.Assets[2].Error != "provider down" {
		t.Fatalf("failed fetches must carry their error, got %#v", pulse.Assets[2])
	}
	if pulse.CapitalFlow.Signal != FlowDefensive || pulse.Scenario.Name != "Falling knife" {
		t.Fatalf("unexpected flow %#v scenario %#v", pulse.CapitalFlow, pulse.Scenario)
	}
	if pulse.TopMovers[0].Symbol != "FPT" || pulse.Sentiment != nil || len(pulse.Notes) != 1 {
		t.Fatalf("unexpected movers/sentiment/notes %#v %#v %#v", pulse.TopMovers, pulse.Sentiment, pulse.Notes)
	}
	if pulse.Index == nil {
		t.Fatal("VN-Index technicals must be included")
	}
}

func TestGivenIndexUpAndVIXDownWhenBuildPulseThenRiskOnWithSentiment(t *testing.T) {
	closes := ramp(1200, 2, 30)
	data := map[string]CandleSeries{
		MarketSymbolVNIndex: series(MarketSymbolVNIndex, Timeframe1D, append(closes[:29:29], closes[28]*1.01)...),
		MarketSymbolVN30:    series(MarketSymbolVN30, Timeframe1D, 1400, 1401),
		MarketSymbolVIX:     series(MarketSymbolVIX, Timeframe1D, 14, 13),
	}
	pulse := BuildPulse(Timeframe1D, data, nil, time.UTC, testDay)
	if pulse.CapitalFlow.Signal != FlowRiskOn || pulse.Scenario.Name != "Normal growth" {
		t.Fatalf("unexpected flow %#v scenario %#v", pulse.CapitalFlow, pulse.Scenario)
	}
	if pulse.Sentiment == nil || len(pulse.Notes) != 0 {
		t.Fatalf("VIX and RSI give a sentiment, got %#v notes %#v", pulse.Sentiment, pulse.Notes)
	}
}

func TestGivenSeriesOnDifferentClocksWhenCorrelateThenAlignsOnBarDate(t *testing.T) {
	a := series("A", Timeframe1D, 1, 2, 1, 2, 1, 2, 1)
	b := series("B", Timeframe1D, 5, 6, 5, 6, 5, 6, 5)
	for i := range b.Candles {
		b.Candles[i].Time = b.Candles[i].Time.Add(13 * time.Hour) // a US session, same date
	}
	c := series("C", Timeframe1D, 1, 2)
	matrix := Correlate([]CandleSeries{a, b, c}, Timeframe1D, time.UTC)
	if matrix.Values[0][1] == nil || !closeTo(*matrix.Values[0][1], 1) || matrix.Values[1][0] != matrix.Values[0][1] {
		t.Fatalf("expected perfect correlation, got %#v", matrix.Values)
	}
	if matrix.Values[0][2] != nil {
		t.Fatal("too few shared returns must be nil")
	}
}

func TestGivenTimeframeTrendsWhenBuildConfluenceThenScoresAgreement(t *testing.T) {
	views := map[Timeframe]TimeframeView{
		Timeframe1W: {Timeframe: Timeframe1W, Trend: TrendUp},
		Timeframe1D: {Timeframe: Timeframe1D, Trend: TrendUp},
		Timeframe4H: {Timeframe: Timeframe4H, Trend: TrendDown},
		Timeframe1H: {Timeframe: Timeframe1H, Error: "no data"},
	}
	result := BuildConfluence(views)
	// (3 + 3 − 2) / 8 → 5.
	if result.Bias != TrendUp || result.Score != 5 || len(result.Views) != 4 || result.Views[0].Timeframe != Timeframe1W {
		t.Fatalf("unexpected confluence %#v", result)
	}
	if len(result.Pairs) != 2 || result.Pairs[0].Status != AlignmentAligned || result.Pairs[1].Status != AlignmentConflict {
		t.Fatalf("unexpected pairs %#v", result.Pairs)
	}
}
