package domain

import (
	"errors"
	"math"
	"slices"
	"testing"
	"time"
)

func TestGivenDailyReturnsWhenSeasonalityThenRanksBucketsPerType(t *testing.T) {
	// Two weeks of weekdays: Mondays rise 2%, Tuesdays fall 1%, others flat.
	closes := []float64{100}
	for day := 1; day < 15; day++ {
		last := closes[len(closes)-1]
		switch testDay.AddDate(0, 0, day).Weekday() {
		case time.Monday:
			closes = append(closes, last*1.02)
		case time.Tuesday:
			closes = append(closes, last*0.99)
		default:
			closes = append(closes, last)
		}
	}
	buckets := Seasonality(dailyCandles(closes...), time.UTC)
	var days []SeasonalityBucket
	for _, bucket := range buckets {
		if bucket.Type == SeasonalityDay {
			days = append(days, bucket)
		}
	}
	if len(days) != 7 {
		t.Fatalf("expected every weekday with two returns, got %d", len(days))
	}
	best := days[0]
	if best.Label != "Mon" || best.Rank != 1 || !closeTo(best.AvgReturn, 2) || best.WinRate != 100 || best.N != 2 {
		t.Fatalf("Monday must rank first, got %#v", best)
	}
	// A zero win rate zeroes the score, so Tuesday ties with the flat days.
	tuesday := days[slices.IndexFunc(days, func(b SeasonalityBucket) bool { return b.Label == "Tue" })]
	if !closeTo(tuesday.AvgReturn, -1) || tuesday.WinRate != 0 || tuesday.ProfitFactor != 0 || tuesday.Score != 0 {
		t.Fatalf("unexpected Tuesday %#v", tuesday)
	}
	// Score = avgRet × 100 × winRate / (1 + std): 0.02 × 100 × 100 / 1.
	if !closeTo(best.Score, 200) {
		t.Fatalf("score = %v, want 200", best.Score)
	}
}

func TestGivenPathWhenComputeRiskStatsThenReportsDrawdownAndDeterministicSimulation(t *testing.T) {
	stats, err := ComputeRiskStats("FPT", dailyCandles(100, 120, 90, 110), 0)
	if err != nil {
		t.Fatal(err)
	}
	if !closeTo(stats.MaxDrawdown, -25) || !closeTo(stats.PeriodReturn, 10) || !closeTo(stats.CurrentDrawdown, (110.0/120-1)*100) {
		t.Fatalf("unexpected stats %#v", stats)
	}
	if stats.AnnualisedReturn != nil || stats.Simulation != nil {
		t.Fatal("under a year and without enough returns there is no CAGR or simulation")
	}

	closes := []float64{100}
	for i := 1; i < 400; i++ {
		closes = append(closes, closes[i-1]*(1+0.01*math.Sin(float64(i))))
	}
	long, err := ComputeRiskStats("FPT", dailyCandles(closes...), 21)
	if err != nil || long.Simulation == nil || long.AnnualisedReturn == nil {
		t.Fatalf("expected a simulation and CAGR, got %#v %v", long, err)
	}
	again, _ := ComputeRiskStats("FPT", dailyCandles(closes...), 21)
	if *again.Simulation != *long.Simulation {
		t.Fatal("the same history must give the same simulation")
	}
	sim := long.Simulation
	if sim.Paths != MonteCarloPaths || !(sim.P10 < sim.P50 && sim.P50 < sim.P90) {
		t.Fatalf("unexpected simulation %#v", sim)
	}
	if _, err := ComputeRiskStats("FPT", dailyCandles(1), 21); !errors.Is(err, ErrValidation) {
		t.Fatalf("one candle must be rejected, got %v", err)
	}
}

func TestGivenSeriesWhenPearsonCorrelationThenDetectsDirection(t *testing.T) {
	if r, ok := PearsonCorrelation([]float64{1, 2, 3}, []float64{2, 4, 6}); !ok || !closeTo(r, 1) {
		t.Fatalf("got %v %v", r, ok)
	}
	if r, ok := PearsonCorrelation([]float64{1, 2, 3}, []float64{3, 2, 1}); !ok || !closeTo(r, -1) {
		t.Fatalf("got %v %v", r, ok)
	}
	if _, ok := PearsonCorrelation([]float64{1, 1, 1}, []float64{1, 2, 3}); ok {
		t.Fatal("a constant series has no correlation")
	}
}
