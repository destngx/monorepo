package domain

import (
	"math"
	"slices"
	"testing"
	"time"
)

var testDay = time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC) // a Monday

func closeTo(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// dailyCandles builds one bar per day from closes, with a ±1 range.
func dailyCandles(closes ...float64) []Candle {
	candles := make([]Candle, len(closes))
	for i, c := range closes {
		candles[i] = Candle{Time: testDay.AddDate(0, 0, i), Open: c, High: c + 1, Low: c - 1, Close: c}
	}
	return candles
}

func ramp(from float64, step float64, n int) []float64 {
	values := make([]float64, n)
	for i := range values {
		values[i] = from + step*float64(i)
	}
	return values
}

func TestGivenSeriesWhenMovingAveragesThenMatchHandComputedValues(t *testing.T) {
	values := []float64{1, 2, 3, 4, 5}
	if sma, ok := SMA(values, 3); !ok || !closeTo(sma, 4) {
		t.Fatalf("SMA = %v %v, want 4", sma, ok)
	}
	// Seed mean(1,2,3) = 2, k = 0.5: 4 → 3, 5 → 4.
	if ema, ok := EMA(values, 3); !ok || !closeTo(ema, 4) {
		t.Fatalf("EMA = %v %v, want 4", ema, ok)
	}
	if _, ok := EMA(values, 6); ok {
		t.Fatal("EMA must need at least period values")
	}
}

func TestGivenClosesWhenRSIThenUsesWilderSmoothing(t *testing.T) {
	// Period 2: seed gain 0.5 loss 0.5; +1 → gain 0.75 loss 0.25 → RS 3 → 75.
	if rsi, ok := RSI([]float64{1, 2, 1, 2}, 2); !ok || !closeTo(rsi, 75) {
		t.Fatalf("RSI = %v %v, want 75", rsi, ok)
	}
	if rsi, _ := RSI(ramp(1, 1, 20), 14); rsi != 100 {
		t.Fatalf("a steady rise must read 100, got %v", rsi)
	}
	if rsi, _ := RSI(ramp(5, 0, 20), 14); rsi != 50 {
		t.Fatalf("a flat series must read 50, got %v", rsi)
	}
	if _, ok := RSI(ramp(1, 1, 14), 14); ok {
		t.Fatal("RSI needs period+1 closes")
	}
}

func TestGivenCandlesWhenATRAndBollingerThenMeasureRange(t *testing.T) {
	candles := dailyCandles(slices.Repeat([]float64{10}, 20)...)
	if atr, ok := ATR(candles, 14); !ok || !closeTo(atr, 2) {
		t.Fatalf("ATR = %v %v, want 2", atr, ok)
	}
	bands, ok := BollingerBands(slices.Repeat([]float64{10}, 20), 20, 2)
	if !ok || bands.Upper != 10 || bands.Lower != 10 || bands.Middle != 10 {
		t.Fatalf("flat closes must collapse the bands, got %#v", bands)
	}
	bands, _ = BollingerBands([]float64{1, 3}, 2, 2)
	if !closeTo(bands.Upper, 4) || !closeTo(bands.Lower, 0) {
		t.Fatalf("mean 2, deviation 1, width 2: want 4/0, got %#v", bands)
	}
}

func TestGivenSwingsWhenSwingLevelsThenSplitsAroundLastClose(t *testing.T) {
	candles := dailyCandles(10, 12, 15, 12, 10, 8, 5, 8, 10, 11, 10)
	levels := SwingLevels(candles, 3)
	// Highs are close+1: swing high 16 above 10; swing low 4 below.
	if !slices.Equal(levels.Resistance, []float64{16}) || !slices.Equal(levels.Support, []float64{4}) {
		t.Fatalf("unexpected levels %#v", levels)
	}
}

func TestGivenGapsWhenOpenFairValueGapsThenSkipsFilledOnes(t *testing.T) {
	candles := []Candle{
		{Time: testDay, High: 10, Low: 9, Close: 9.5},
		{Time: testDay.AddDate(0, 0, 1), High: 13, Low: 10, Close: 12},
		{Time: testDay.AddDate(0, 0, 2), High: 14, Low: 12, Close: 13},   // bullish gap 10..12
		{Time: testDay.AddDate(0, 0, 3), High: 15, Low: 13.5, Close: 14}, // bullish gap 13..13.5
	}
	gaps := OpenFairValueGaps(candles, 3)
	if len(gaps) != 2 || gaps[0].Type != FVGBullish || gaps[0].Bottom != 13 || gaps[1].Bottom != 10 || gaps[1].Top != 12 {
		t.Fatalf("expected the two open bullish gaps newest first, got %#v", gaps)
	}
	filled := append(candles, Candle{Time: testDay.AddDate(0, 0, 4), High: 14, Low: 9.5, Close: 10})
	if gaps := OpenFairValueGaps(filled, 3); len(gaps) != 0 {
		t.Fatalf("a bar trading back through both gaps fills them, got %#v", gaps)
	}
}

func TestGivenEMAPositionsWhenWyckoffPhaseThenFollowsRuleOrder(t *testing.T) {
	cases := []struct {
		name  string
		price float64
		in    Indicators
		want  string
	}{
		{"above all", 110, Indicators{EMA20: ptr(100), EMA50: ptr(95), EMA200: ptr(90), RSI: ptr(60)}, PhaseMarkup},
		{"below all", 80, Indicators{EMA20: ptr(100), EMA50: ptr(95), EMA200: ptr(90), RSI: ptr(40)}, PhaseMarkdown},
		{"above 50 hot RSI under 200", 96, Indicators{EMA20: ptr(97), EMA50: ptr(95), EMA200: ptr(99), RSI: ptr(75)}, PhaseDistribution},
		{"below 200 cold RSI", 92, Indicators{EMA20: ptr(91), EMA50: ptr(95), EMA200: ptr(100), RSI: ptr(25)}, PhaseAccumulation},
		{"lost EMA20 above 50", 96, Indicators{EMA20: ptr(97), EMA50: ptr(95), EMA200: ptr(90), RSI: ptr(50)}, PhaseDecline},
		{"mixed", 96, Indicators{EMA20: ptr(95), EMA50: ptr(97), EMA200: ptr(90), RSI: ptr(50)}, PhaseConsolidation},
		{"no EMA200 above both", 110, Indicators{EMA20: ptr(100), EMA50: ptr(95), RSI: ptr(60)}, PhaseMarkup},
		{"no RSI", 110, Indicators{EMA20: ptr(100), EMA50: ptr(95)}, PhaseUnknown},
	}
	for _, tc := range cases {
		if got := wyckoffPhase(tc.price, tc.in); got != tc.want {
			t.Errorf("%s: got %s, want %s", tc.name, got, tc.want)
		}
	}
}

func TestGivenPhaseWhenSignalThenSetsATRLevelsOnlyForPositions(t *testing.T) {
	long := signalFor(PhaseMarkup, 100, Indicators{RSI: ptr(55), ATR: ptr(2)})
	if long.Action != ActionLong || *long.StopLoss != 97 || *long.TakeProfit != 107 || !closeTo(*long.RiskReward, 3.5/1.5) {
		t.Fatalf("unexpected long signal %#v", long)
	}
	short := signalFor(PhaseMarkdown, 100, Indicators{RSI: ptr(50), ATR: ptr(2)})
	if short.Action != ActionShort || *short.StopLoss != 103 || *short.TakeProfit != 93 {
		t.Fatalf("unexpected short signal %#v", short)
	}
	exit := signalFor(PhaseMarkdown, 100, Indicators{RSI: ptr(30), ATR: ptr(2)})
	if exit.Action != ActionExit || exit.StopLoss != nil {
		t.Fatalf("exit must carry no levels, got %#v", exit)
	}
	if profit := signalFor(PhaseMarkup, 100, Indicators{RSI: ptr(70), ATR: ptr(2)}); profit.Action != ActionTakeProfit {
		t.Fatalf("hot markup must take profit, got %#v", profit)
	}
}

func TestGivenRisingSeriesWhenComputeTechnicalsThenReadsMarkupWithNotes(t *testing.T) {
	result := ComputeTechnicals(dailyCandles(ramp(100, 1, 60)...))
	if result.Phase != PhaseMarkup || result.Trend != TrendUp || result.Signal.Action != ActionTakeProfit {
		t.Fatalf("a steady rise reads markup with RSI 100, got %s %s %s", result.Phase, result.Trend, result.Signal.Action)
	}
	if result.Indicators.EMA200 != nil || len(result.Notes) != 1 {
		t.Fatalf("60 candles have no EMA200 and must say so, got %#v", result.Notes)
	}
	if empty := ComputeTechnicals(nil); empty.Phase != PhaseUnknown || len(empty.Notes) != 1 {
		t.Fatalf("no candles must be unknown, got %#v", empty)
	}
}

func TestGivenVIXAndRSIWhenSentimentThenAppliesRule(t *testing.T) {
	cases := []struct {
		vix, rsi float64
		score    int
		label    string
	}{
		{20, 50, 50, "Neutral"},
		{10, 70, 95, "Greed"},
		{40, 40, 0, "Fear"},
	}
	for _, tc := range cases {
		if got := SentimentScore(tc.vix, tc.rsi); got.Score != tc.score || got.Label != tc.label {
			t.Errorf("SentimentScore(%v, %v) = %#v", tc.vix, tc.rsi, got)
		}
	}
	if Direction(0.04) != "flat" || Direction(0.06) != TrendUp || Direction(-0.06) != TrendDown {
		t.Fatal("direction band is ±0.05%")
	}
}
