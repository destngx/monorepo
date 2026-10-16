package domain

import (
	"math"
	"strings"
	"testing"
	"time"
)

func near(got float64, want float64, tolerance float64) bool {
	return math.Abs(got-want) <= tolerance
}

func units(value float64) *float64 { return &value }

func TestGivenAHistoryCellWhenParsingThenEachEntryIsADateAndUnitsWithADecimalComma(t *testing.T) {
	trades, issues := ParseFundHistory(" 5-3-2024,10,5; 1-6-2024,4 ;10-1-2025,-2,25;; 2-2-2025,1.000,5")

	if len(issues) != 0 {
		t.Fatalf("issues = %v", issues)
	}
	want := []FundTrade{{"2024-03-05", 10.5}, {"2024-06-01", 4}, {"2025-01-10", -2.25}, {"2025-02-02", 1000.5}}
	if len(trades) != len(want) {
		t.Fatalf("trades = %+v", trades)
	}
	for i := range want {
		if trades[i] != want[i] {
			t.Fatalf("trade %d = %+v, want %+v", i, trades[i], want[i])
		}
	}
}

func TestGivenAnEmptyHistoryCellWhenParsingThenThereAreNoTradesAndNoIssues(t *testing.T) {
	trades, issues := ParseFundHistory("  ")
	if len(trades) != 0 || len(issues) != 0 {
		t.Fatalf("trades = %v, issues = %v", trades, issues)
	}
}

func TestGivenUnreadableHistoryEntriesWhenParsingThenEachIsAnIssue(t *testing.T) {
	trades, issues := ParseFundHistory("31-2-2024,1;5-3-2024;x,1;5-3-2024,abc;5-3-2024,0")
	if len(trades) != 0 || len(issues) != 5 {
		t.Fatalf("trades = %v, issues = %v", trades, issues)
	}
}

func TestGivenPublishedRatesWithoutTenorsWhenChoosingTheDepositRateThenTheBestRateIsUsedAndLabelled(t *testing.T) {
	rate, ok := DepositRateFromBankRates([]BankRate{{Bank: "Bank A", Rate: 4.6, UpdatedAt: "2026-10-10"}, {Bank: "Bank B", Rate: 5.1}, {Bank: "Bank C", Rate: 0}})
	if !ok || rate.Bank != "Bank B" || rate.Rate != 5.1 || rate.Tenor != "" || !strings.Contains(rate.Basis, "best published") {
		t.Fatalf("rate = %+v", rate)
	}
}

func TestGivenRatesWithTenorsWhenChoosingTheDepositRateThenTheBest12MonthRateWins(t *testing.T) {
	rate, ok := DepositRateFromBankRates([]BankRate{
		{Bank: "Bank A", Tenor: "6M", Rate: 5.5},
		{Bank: "Bank B", Tenor: "12M", Rate: 4.8},
		{Bank: "Bank C", Tenor: "12 tháng", Rate: 5.1},
		{Bank: "Bank D", Tenor: "24M", Rate: 6},
	})
	if !ok || rate.Bank != "Bank C" || rate.Rate != 5.1 {
		t.Fatalf("rate = %+v", rate)
	}

	rate, ok = DepositRateFromBankRates([]BankRate{{Bank: "Bank A", Tenor: "6M", Rate: 4.5}, {Bank: "Bank B", Tenor: "9M", Rate: 4.7}, {Bank: "Bank C", Tenor: "36M", Rate: 6}})
	if !ok || rate.Bank != "Bank B" || rate.Tenor != "9M" {
		t.Fatalf("nearest tenor = %+v", rate)
	}

	if _, ok := DepositRateFromBankRates(nil); ok {
		t.Fatal("no rates must give no deposit rate")
	}
}

func fundNavs(points ...NavPoint) map[string][]NavPoint {
	return map[string][]NavPoint{"FUND-A": points}
}

func TestGivenOneBuyAYearAgoWhenComputingPerformanceThenGainReturnAndDepositComparisonFollow(t *testing.T) {
	deposit := DepositRate{Bank: "Bank A", Rate: 5}
	result := ComputeFundPerformance(FundPerformanceInputs{
		Holdings: []Holding{{Platform: "Platform A", Symbol: "FUND-A", Quantity: units(10), History: []FundTrade{{"2025-01-01", 10}}}},
		NAVs:     fundNavs(NavPoint{"2024-12-31", 9_000}, NavPoint{"2025-01-01", 10_000}, NavPoint{"2026-01-01", 11_000}),
		Deposit:  &deposit,
		Now:      time.Date(2026, 1, 2, 9, 0, 0, 0, time.UTC),
	})

	if len(result.Holdings) != 1 {
		t.Fatalf("holdings = %+v", result.Holdings)
	}
	holding := result.Holdings[0]
	if holding.CostStatus != FundCostKnown || holding.NAVDate != "2026-01-01" || *holding.Value != 110_000 {
		t.Fatalf("holding = %+v", holding)
	}
	if len(holding.CashFlows) != 1 || holding.CashFlows[0].Amount != -100_000 || holding.CashFlows[0].NAV != 10_000 {
		t.Fatalf("cash flows = %+v", holding.CashFlows)
	}
	r := holding.Return
	if r == nil || r.NetInvested != 100_000 || r.Gain != 10_000 || !near(*r.SimpleReturn, 10, 1e-9) || !near(*r.AnnualisedReturn, 10, 1e-6) {
		t.Fatalf("return = %+v", r)
	}
	if !near(*r.DepositValue, 105_000, 1e-6) || !near(*r.DifferenceVND, 5_000, 1e-6) || !near(*r.DifferencePoints, 5, 1e-6) || r.VsDeposit != VsDepositAhead {
		t.Fatalf("deposit comparison = %+v", r)
	}
	if *holding.CostBasis != 100_000 || *holding.UnrealisedGain != 10_000 {
		t.Fatalf("cost basis = %v, unrealised = %v", *holding.CostBasis, *holding.UnrealisedGain)
	}
	if result.Totals.Return == nil || !near(*result.Totals.Return.AnnualisedReturn, 10, 1e-6) || result.Deposit == nil || result.Deposit.Rate != 5 {
		t.Fatalf("totals = %+v, deposit = %+v", result.Totals, result.Deposit)
	}
}

func TestGivenABuyOnADayWithoutNAVAndASaleWhenComputingThenTheNextPublishedNAVPricesEachTrade(t *testing.T) {
	deposit := DepositRate{Bank: "Bank A", Rate: 20}
	result := ComputeFundPerformance(FundPerformanceInputs{
		Holdings: []Holding{{Symbol: "FUND-A", Quantity: units(6), History: []FundTrade{{"2025-03-01", 10}, {"2025-07-01", -4}}}},
		NAVs:     fundNavs(NavPoint{"2025-03-03", 10_000}, NavPoint{"2025-07-01", 12_000}, NavPoint{"2026-03-03", 12_500}),
		Deposit:  &deposit,
		Now:      time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC),
	})

	holding := result.Holdings[0]
	flows := holding.CashFlows
	if len(flows) != 2 || flows[0].NAVDate != "2025-03-03" || flows[0].Amount != -100_000 || flows[1].Amount != 48_000 {
		t.Fatalf("flows = %+v", flows)
	}
	r := holding.Return
	// value 75,000; bought 100,000, sold 48,000: gain 23,000, simple 23%.
	if *holding.Value != 75_000 || r.NetInvested != 52_000 || r.TotalBought != 100_000 || r.TotalSold != 48_000 || r.Gain != 23_000 || !near(*r.SimpleReturn, 23, 1e-9) {
		t.Fatalf("holding = %+v, return = %+v", holding, r)
	}
	// Average cost: 6 units at 10,000 remain.
	if *holding.CostBasis != 60_000 || *holding.UnrealisedGain != 15_000 {
		t.Fatalf("cost basis = %v", *holding.CostBasis)
	}
	if r.AnnualisedReturn == nil || *r.AnnualisedReturn <= 20 || r.VsDeposit != VsDepositAhead {
		t.Fatalf("return = %+v", r)
	}
	// The deposit takes the same flows: the sale is a withdrawal.
	wantDeposit := 100_000*math.Pow(1.2, 365.0/365) - 48_000*math.Pow(1.2, 245.0/365)
	if !near(*r.DepositValue, wantDeposit, 1e-6) {
		t.Fatalf("deposit value = %v, want %v", *r.DepositValue, wantDeposit)
	}
}

func TestGivenHistoryUnitsThatDoNotAddUpToTotalUnitWhenComputingThenTheMismatchIsAnIssueAndCostIsUnknown(t *testing.T) {
	result := ComputeFundPerformance(FundPerformanceInputs{
		Holdings: []Holding{{Symbol: "FUND-A", Quantity: units(15), History: []FundTrade{{"2025-01-01", 10}}}},
		NAVs:     fundNavs(NavPoint{"2025-01-01", 10_000}, NavPoint{"2026-01-01", 11_000}),
		Now:      time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
	})

	holding := result.Holdings[0]
	if holding.CostStatus != FundCostUnitsMismatch || holding.Return != nil || holding.HistoryUnits != 10 || *holding.Value != 165_000 {
		t.Fatalf("holding = %+v", holding)
	}
	if len(holding.Issues) != 1 || !strings.Contains(holding.Issues[0], "10") || !strings.Contains(holding.Issues[0], "15") {
		t.Fatalf("issues = %v", holding.Issues)
	}
	if result.Totals.CostUnknown != 1 || result.Totals.Return != nil || result.Totals.Value != 165_000 {
		t.Fatalf("totals = %+v", result.Totals)
	}
}

func TestGivenAHoldingWithoutHistoryWhenComputingThenItHasAValueOnlyAndCostUnknown(t *testing.T) {
	result := ComputeFundPerformance(FundPerformanceInputs{
		Holdings: []Holding{{Symbol: "FUND-A", Quantity: units(2)}},
		NAVs:     fundNavs(NavPoint{"2026-01-01", 11_000}),
		Now:      time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
	})

	holding := result.Holdings[0]
	if holding.CostStatus != FundCostNoHistory || holding.Return != nil || *holding.Value != 22_000 || holding.CostBasis != nil {
		t.Fatalf("holding = %+v", holding)
	}
}

func TestGivenNoNAVHistoryWhenComputingThenTheHoldingHasNoValueAndTheReasonIsKept(t *testing.T) {
	result := ComputeFundPerformance(FundPerformanceInputs{
		Holdings:  []Holding{{Symbol: "fund-a", Quantity: units(2), History: []FundTrade{{"2025-01-01", 2}}}},
		NAVIssues: map[string]string{"FUND-A": "fund not found"},
		Now:       time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
	})

	holding := result.Holdings[0]
	if holding.CostStatus != FundCostNAVUnavailable || holding.Value != nil || holding.Return != nil || !strings.Contains(strings.Join(holding.Issues, " "), "fund not found") {
		t.Fatalf("holding = %+v", holding)
	}
}

func TestGivenATradeBeforeTheNAVHistoryWhenComputingThenCostIsUnknown(t *testing.T) {
	result := ComputeFundPerformance(FundPerformanceInputs{
		Holdings: []Holding{{Symbol: "FUND-A", Quantity: units(1), History: []FundTrade{{"2026-02-01", 1}}}},
		NAVs:     fundNavs(NavPoint{"2026-01-01", 11_000}),
		Now:      time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC),
	})
	holding := result.Holdings[0]
	if holding.CostStatus != FundCostNAVUnavailable || holding.Return != nil || *holding.Value != 11_000 {
		t.Fatalf("a trade after the last published NAV cannot be priced: %+v", holding)
	}
}

func TestGivenInvalidHistoryWhenComputingThenCostIsUnknownWithTheParseIssues(t *testing.T) {
	result := ComputeFundPerformance(FundPerformanceInputs{
		Holdings: []Holding{{Symbol: "FUND-A", Quantity: units(1), History: []FundTrade{{"2025-01-01", 1}}, HistoryIssues: []string{"entry 2 unreadable"}}},
		NAVs:     fundNavs(NavPoint{"2025-01-01", 10_000}, NavPoint{"2026-01-01", 11_000}),
		Now:      time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
	})
	holding := result.Holdings[0]
	if holding.CostStatus != FundCostHistoryInvalid || holding.Return != nil || holding.Issues[0] != "entry 2 unreadable" {
		t.Fatalf("holding = %+v", holding)
	}
}

func TestGivenTheSameFundOnTwoPlatformsWhenComputingThenTotalsAndPerFundSummariesCombineThem(t *testing.T) {
	deposit := DepositRate{Bank: "Bank A", Rate: 5}
	result := ComputeFundPerformance(FundPerformanceInputs{
		Holdings: []Holding{
			{Platform: "Platform A", Symbol: "FUND-A", Quantity: units(10), History: []FundTrade{{"2025-01-01", 10}}},
			{Platform: "Platform B", Symbol: "FUND-A", Quantity: units(5), History: []FundTrade{{"2025-01-01", 5}}},
			{Platform: "Platform B", Symbol: "FUND-B", Quantity: units(1)},
		},
		NAVs: map[string][]NavPoint{
			"FUND-A": {{"2025-01-01", 10_000}, {"2026-01-01", 9_000}},
			"FUND-B": {{"2026-01-01", 50_000}},
		},
		Deposit: &deposit,
		Now:     time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
	})

	totals := result.Totals
	if totals.Holdings != 3 || totals.CostKnown != 2 || totals.CostUnknown != 1 || totals.Value != 185_000 || totals.ValueCostKnown != 135_000 {
		t.Fatalf("totals = %+v", totals)
	}
	if totals.Return == nil || totals.Return.Gain != -15_000 || !near(*totals.Return.AnnualisedReturn, -10, 1e-6) || totals.Return.VsDeposit != VsDepositBehind {
		t.Fatalf("total return = %+v", totals.Return)
	}
	if len(result.Funds) != 2 || result.Funds[0].Symbol != "FUND-A" || result.Funds[0].Holdings != 2 || result.Funds[0].Value != 135_000 ||
		result.Funds[0].Return == nil || result.Funds[0].Return.NetInvested != 150_000 || result.Funds[1].CostStatus != FundCostNoHistory || result.Funds[1].Return != nil {
		t.Fatalf("funds = %+v", result.Funds)
	}
}

func TestGivenNoDepositRateWhenComputingThenTheComparisonIsLeftOutWithTheReason(t *testing.T) {
	result := ComputeFundPerformance(FundPerformanceInputs{
		Holdings:     []Holding{{Symbol: "FUND-A", Quantity: units(10), History: []FundTrade{{"2025-01-01", 10}}}},
		NAVs:         fundNavs(NavPoint{"2025-01-01", 10_000}, NavPoint{"2026-01-01", 11_000}),
		DepositIssue: "bank rates unavailable",
		Now:          time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
	})
	r := result.Holdings[0].Return
	if r == nil || r.DepositValue != nil || r.DifferenceVND != nil || r.VsDeposit != "" || result.DepositIssue != "bank rates unavailable" {
		t.Fatalf("return = %+v, result = %+v", r, result)
	}
}

func TestGivenARecentFirstPurchaseWhenComputingThenTheAnnualisedReturnIsLeftOut(t *testing.T) {
	deposit := DepositRate{Bank: "Bank A", Rate: 5}
	result := ComputeFundPerformance(FundPerformanceInputs{
		Holdings: []Holding{{Symbol: "FUND-A", Quantity: units(10), History: []FundTrade{{"2025-12-20", 10}}}},
		NAVs:     fundNavs(NavPoint{"2025-12-20", 10_000}, NavPoint{"2026-01-01", 10_500}),
		Deposit:  &deposit,
		Now:      time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
	})
	r := result.Holdings[0].Return
	if r == nil || r.AnnualisedReturn != nil || r.DifferencePoints != nil || r.DifferenceVND == nil || r.Gain != 5_000 {
		t.Fatalf("return = %+v", r)
	}
}

func TestXIRRSolvesKnownCashFlows(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	got, ok := xirr([]datedFlow{{date: start, amount: -1000}, {date: start.AddDate(0, 6, 0), amount: -1000}, {date: start.AddDate(1, 0, 0), amount: 2200}})
	if !ok {
		t.Fatal("no solution")
	}
	// Check by the definition: the NPV at the rate is zero.
	npv := -1000.0 - 1000/math.Pow(1+got, 181.0/365) + 2200/math.Pow(1+got, 1)
	if math.Abs(npv) > 1e-6 || got <= 0.1 || got >= 0.15 {
		t.Fatalf("xirr = %v, npv = %v", got, npv)
	}
	if _, ok := xirr([]datedFlow{{date: start, amount: -1000}}); ok {
		t.Fatal("one-sided flows have no rate")
	}
}
