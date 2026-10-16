package domain

import (
	"testing"
)

// historyLedger extends snapshotLedger back to August: 10M income on 08-01
// and 4M expense on 09-10. Net worth today (2026-11-05) is 76M.
func historyLedger() ([]Account, []Transaction) {
	accounts, transactions := snapshotLedger()
	transactions = append(transactions,
		txn(day(2026, 8, 1), "Salary", CategoryTypeIncome, 0, 10_000_000),
		txn(day(2026, 9, 10), "Food", CategoryTypeExpense, 4_000_000, 0),
	)
	return accounts, transactions
}

func pointsByMonth(history NetWorthHistory) map[string]NetWorthPoint {
	byMonth := map[string]NetWorthPoint{}
	for _, point := range history.Points {
		byMonth[point.Month] = point
	}
	return byMonth
}

func TestGivenNoSnapshotsWhenBuildingTheHistoryThenPastMonthsAreEstimatedFromToday(t *testing.T) {
	accounts, transactions := historyLedger()

	history := ComputeNetWorthHistory(NetWorthHistoryInputs{Accounts: accounts, Transactions: transactions, Today: day(2026, 11, 5)})

	if history.Current != 76_000_000 || len(history.Points) != 4 || history.Estimated != 3 || history.Snapshots != 0 || history.TabExists {
		t.Fatalf("history = %+v", history)
	}
	points := pointsByMonth(history)
	// Back from 76M: undo -2M (Nov), then +12M (Oct; lending left out), then -4M (Sep).
	want := map[string]float64{"2026-11": 76_000_000, "2026-10": 78_000_000, "2026-09": 66_000_000, "2026-08": 70_000_000}
	for month, value := range want {
		if points[month].NetWorth != value {
			t.Errorf("%s = %v, want %v", month, points[month].NetWorth, value)
		}
	}
	if points["2026-11"].Source != NetWorthLive || points["2026-11"].Estimated || !points["2026-10"].Estimated {
		t.Fatalf("sources = %+v", points)
	}
	if points["2026-11"].Change == nil || *points["2026-11"].Change != -2_000_000 || points["2026-11"].Savings != nil {
		t.Fatalf("a change next to an estimate is not split: %+v", points["2026-11"])
	}
}

func TestGivenSnapshotsWhenBuildingTheHistoryThenTheyAnchorTheMonthsAndSplitTheChange(t *testing.T) {
	accounts, transactions := historyLedger()
	sheet := SnapshotSheet{Exists: true, Snapshots: []MonthSnapshot{
		{Month: "2026-09", AsOf: "2026-09-30", NetWorth: 50_000_000},
		{Month: "2026-10", AsOf: "2026-10-31", NetWorth: 70_000_000},
	}}

	history := ComputeNetWorthHistory(NetWorthHistoryInputs{Accounts: accounts, Transactions: transactions, Snapshots: sheet, Today: day(2026, 11, 5)})

	points := pointsByMonth(history)
	if history.Snapshots != 2 || history.Estimated != 1 || !history.TabExists {
		t.Fatalf("counts = %d %d", history.Snapshots, history.Estimated)
	}
	// August is estimated from the September snapshot, not from today.
	if aug := points["2026-08"]; aug.NetWorth != 54_000_000 || !aug.Estimated {
		t.Fatalf("august = %+v", aug)
	}
	oct := points["2026-10"]
	if oct.Source != NetWorthFromSnapshot || *oct.Change != 20_000_000 || *oct.Savings != 12_000_000 || *oct.Market != 8_000_000 {
		t.Fatalf("october = %+v", oct)
	}
	nov := points["2026-11"]
	if nov.Source != NetWorthLive || *nov.Change != 6_000_000 || *nov.Savings != -2_000_000 || *nov.Market != 8_000_000 {
		t.Fatalf("november = %+v", nov)
	}
	if points["2026-09"].Savings != nil {
		t.Fatal("a change from an estimated month must not be split")
	}
	last := history.Milestones[len(history.Milestones)-1]
	if last.Kind != MilestoneAllTimeHigh || last.Month != "2026-11" || last.NetWorth != 76_000_000 {
		t.Fatalf("milestones = %+v", history.Milestones)
	}
}

func TestGivenAPartialSnapshotThisMonthWhenBuildingTheHistoryThenTodaysValueIsUsed(t *testing.T) {
	accounts, transactions := historyLedger()
	sheet := SnapshotSheet{Exists: true, Snapshots: []MonthSnapshot{{Month: "2026-11", AsOf: "2026-11-02", NetWorth: 1}}}

	history := ComputeNetWorthHistory(NetWorthHistoryInputs{Accounts: accounts, Transactions: transactions, Snapshots: sheet, Today: day(2026, 11, 5)})

	if nov := pointsByMonth(history)["2026-11"]; nov.NetWorth != 76_000_000 || nov.Source != NetWorthLive {
		t.Fatalf("november = %+v", nov)
	}
}

func TestGivenAHandTypedSnapshotBeforeTheLedgerWhenBuildingTheHistoryThenItStartsThere(t *testing.T) {
	accounts, transactions := historyLedger()
	sheet := SnapshotSheet{Exists: true, Snapshots: []MonthSnapshot{{Month: "2026-05", NetWorth: 40_000_000}}}

	history := ComputeNetWorthHistory(NetWorthHistoryInputs{Accounts: accounts, Transactions: transactions, Snapshots: sheet, Today: day(2026, 11, 5)})

	first := history.Points[0]
	if first.Month != "2026-05" || first.NetWorth != 40_000_000 || first.AsOf != "2026-05-31" || len(history.Points) != 7 {
		t.Fatalf("first = %+v of %d", first, len(history.Points))
	}
}

func TestGivenRisingNetWorthWhenFindingMilestonesThenFirstCrossingsAndTheHighAreMarked(t *testing.T) {
	points := []NetWorthPoint{
		{Month: "2026-01", NetWorth: 90_000_000},
		{Month: "2026-02", NetWorth: 150_000_000, Estimated: true},
		{Month: "2026-03", NetWorth: 95_000_000},
		{Month: "2026-04", NetWorth: 120_000_000},
		{Month: "2026-05", NetWorth: 210_000_000},
		{Month: "2026-06", NetWorth: 205_000_000},
	}

	milestones := netWorthMilestones(points)

	if len(milestones) != 3 {
		t.Fatalf("milestones = %+v", milestones)
	}
	if m := milestones[0]; m.Month != "2026-02" || *m.Threshold != 100_000_000 || !m.Estimated || m.Label != "Crossed 100 million VND" {
		t.Fatalf("first = %+v", m)
	}
	if m := milestones[1]; m.Month != "2026-05" || *m.Threshold != 200_000_000 || m.Kind != MilestoneThreshold {
		t.Fatalf("second = %+v", m)
	}
	if m := milestones[2]; m.Month != "2026-05" || m.Kind != MilestoneAllTimeHigh {
		t.Fatalf("high = %+v", m)
	}
	// A level the history starts above is not a crossing.
	if got := netWorthMilestones([]NetWorthPoint{{Month: "2026-01", NetWorth: 150_000_000}, {Month: "2026-02", NetWorth: 250_000_000}}); len(got) != 2 || *got[0].Threshold != 200_000_000 {
		t.Fatalf("starting above = %+v", got)
	}
}

func TestGivenAHistoryWhenSummarisingForTheBriefingThenTheYearAndHighAreNameless(t *testing.T) {
	points := make([]NetWorthPoint, 14)
	for i := range points {
		points[i] = NetWorthPoint{Month: MonthKey(day(2025, 10, 1).AddDate(0, i, 0)), NetWorth: float64(100+i) * 1_000_000}
	}
	saved, market := 1_000_000.0, 0.0
	points[13].Savings, points[13].Market = &saved, &market
	history := NetWorthHistory{Current: 113_000_000, Points: points, Milestones: netWorthMilestones(points)}

	facts := history.Facts()

	if facts.Months != 14 || *facts.YearAgo != 101_000_000 || *facts.YearChange != 12_000_000 || facts.AllTimeHighMonth != "2026-11" {
		t.Fatalf("facts = %+v", facts)
	}
	if facts.LastSplit == nil || facts.LastSplit.Month != "2026-11" {
		t.Fatalf("last split = %+v", facts.LastSplit)
	}
	briefing := BriefingFacts{}.WithNetWorthHistory(history)
	if briefing.NetWorthHistory == nil {
		t.Fatal("briefing facts missing the history")
	}
}
