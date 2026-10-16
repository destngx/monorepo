package domain

import (
	"strings"
	"testing"
	"time"
)

// dailyBars: one bar a day at 17:00 UTC (midnight in Vietnam) from start.
func dailyBars(start time.Time, days int, price func(day int) float64) []Candle {
	candles := make([]Candle, days)
	for i := range candles {
		candles[i] = Candle{Time: start.AddDate(0, 0, i), Close: price(i)}
	}
	return candles
}

func hedgeAllocation() Allocation {
	return ComputeAllocation(AllocationInputs{
		Accounts:       []Account{{Name: "Bank", Type: AccountTypeBank, Balance: 100_000_000}},
		Funds:          50_000_000,
		CryptoHoldings: 50_000_000,
	})
}

func TestGivenAYearOfDailyClosesWhenComputingHedgesThenEachPeriodChangeIsAgainstTheCloseAtItsStart(t *testing.T) {
	start := time.Date(2025, 10, 16, 17, 0, 0, 0, time.UTC)
	// 100 a year back, rising 1 a day
	gold := CandleSeries{Provider: "fmarket", Candles: dailyBars(start, 367, func(day int) float64 { return 100 + float64(day) })}
	hedges := ComputeHedges(HedgeInputs{Allocation: hedgeAllocation(), Gold: &gold, USDIssue: "upstream down", Now: start})

	if hedges.Cash != 100_000_000 || hedges.CashShare != 50 || hedges.VNDShare != 75 {
		t.Fatalf("hedges = %+v", hedges)
	}
	trend := hedges.Gold
	if trend == nil || trend.Asset != HedgeGold || trend.Symbol != HedgeGoldSymbol || trend.Latest != 466 || trend.LatestDate != "2026-10-18" {
		t.Fatalf("gold = %+v", trend)
	}
	year := trend.Change("1Y")
	if year == nil || year.Partial || year.FromDate != "2025-10-18" || year.From != 101 || year.ChangePct != 361.39 {
		t.Fatalf("1Y = %+v", year)
	}
	if !near(year.OnCash, 100_000_000*365.0/101, 0.01) {
		t.Fatalf("1Y on cash = %v", year.OnCash)
	}
	if quarter := trend.Change("3M"); quarter == nil || quarter.FromDate != "2026-07-18" || quarter.Partial {
		t.Fatalf("3M = %+v", quarter)
	}
	// points start at the longest period's reference close
	if trend.Points[0].Date != "2025-10-18" || trend.Points[len(trend.Points)-1].Close != 466 || trend.Source.Provider != "fmarket" {
		t.Fatalf("points %v .. source %+v", trend.Points[0], trend.Source)
	}
	if hedges.USD != nil || len(hedges.Issues) != 1 || !strings.Contains(hedges.Issues[0], "US dollar trend unavailable: upstream down") {
		t.Fatalf("usd = %+v issues %q", hedges.USD, hedges.Issues)
	}
}

func TestGivenAShortHistoryWhenComputingHedgesThenTheChangeIsPartialFromTheFirstClose(t *testing.T) {
	start := time.Date(2026, 9, 1, 2, 0, 0, 0, time.UTC)
	usd := CandleSeries{Candles: dailyBars(start, 30, func(day int) float64 { return 25_000 + float64(day)*10 })}
	// two bars on one Vietnam day: the later close wins
	usd.Candles = append(usd.Candles, Candle{Time: usd.Candles[29].Time.Add(time.Hour), Close: 26_000})
	hedges := ComputeHedges(HedgeInputs{Allocation: hedgeAllocation(), USD: &usd})

	trend := hedges.USD
	if trend == nil || trend.Latest != 26_000 || len(trend.Points) != 30 {
		t.Fatalf("usd = %+v", trend)
	}
	if quarter := trend.Change("3M"); !quarter.Partial || quarter.From != 25_000 || quarter.ChangePct != 4 {
		t.Fatalf("3M = %+v", quarter)
	}
	if trend.Source != nil {
		t.Fatalf("no provider means no source, got %+v", trend.Source)
	}
}

func TestGivenTooFewClosesWhenComputingHedgesThenTheTrendIsAnIssue(t *testing.T) {
	one := CandleSeries{Candles: []Candle{{Time: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), Close: 1}}}
	hedges := ComputeHedges(HedgeInputs{Gold: &one, USD: &one})
	if hedges.Gold != nil || hedges.USD != nil || len(hedges.Issues) != 2 || hedges.VNDShare != 0 {
		t.Fatalf("hedges = %+v", hedges)
	}
}
