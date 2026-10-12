package domain

import "testing"

func TestBudgetStatusFor(t *testing.T) {
	tests := map[float64]BudgetStatus{0: BudgetStatusHealthy, 69.99: BudgetStatusHealthy, 70: BudgetStatusWarning, 89.9: BudgetStatusWarning, 90: BudgetStatusUrgent, 140: BudgetStatusUrgent}
	for percent, want := range tests {
		if got := BudgetStatusFor(percent); got != want {
			t.Fatalf("%v%%: got %s want %s", percent, got, want)
		}
	}
}

func TestComputeBudgetReport(t *testing.T) {
	march := day(2026, 3, 1)
	items := []BudgetItem{
		{Category: "Food", CategoryType: categoryType(CategoryTypeExpense), MonthlyLimit: 4_000_000, YearlyLimit: 48_000_000, MonthlyLimits: map[string]float64{"2026-03": 5_000_000}},
		{Category: "Fun", CategoryType: categoryType(CategoryTypeExpense), MonthlyLimit: 1_000_000, YearlyLimit: 12_000_000},
		{Category: "Unused", CategoryType: categoryType(CategoryTypeExpense)},
		{Category: "Salary", CategoryType: categoryType(CategoryTypeIncome)},
	}
	transactions := []Transaction{
		txn(day(2026, 3, 2), "food", CategoryTypeExpense, 3_000_000, 0),
		txn(day(2026, 3, 3), "Food", CategoryTypeExpense, 0, 500_000),   // a refund reduces spend
		txn(day(2026, 3, 4), "Fun", CategoryTypeExpense, 0, 2_000_000),  // net negative clamps to 0
		txn(day(2026, 2, 4), "Food", CategoryTypeExpense, 1_000_000, 0), // other month, counts yearly
		txn(day(2026, 3, 5), "Gifts", CategoryTypeExpense, 700_000, 0),  // no budget row
		txn(day(2026, 3, 5), "Salary", CategoryTypeIncome, 0, 20_000_000),
	}
	report := ComputeBudgetReport(items, transactions, march, day(2026, 3, 10))

	lines := map[string]BudgetLine{}
	for _, line := range report.Lines {
		lines[line.Category] = line
	}
	food := lines["Food"]
	if food.Limit != 5_000_000 || food.Spent != 2_500_000 || food.Remaining != 2_500_000 || food.PercentUsed != 50 {
		t.Fatalf("unexpected food line %+v", food)
	}
	if food.YearlySpent != 3_500_000 || food.YearlyRemaining != 44_500_000 {
		t.Fatalf("unexpected food yearly figures %+v", food)
	}
	if fun := lines["Fun"]; fun.Spent != 0 || fun.Remaining != 1_000_000 {
		t.Fatalf("expected fun spend clamped to zero, got %+v", fun)
	}
	if _, ok := lines["Unused"]; ok {
		t.Fatal("expected line with no limit and no spend to be hidden")
	}
	if _, ok := lines["Salary"]; ok {
		t.Fatal("expected income line with no limit and no spend to be hidden")
	}
	gifts := lines["Gifts"]
	if !gifts.Unbudgeted || gifts.Spent != 700_000 || gifts.Status != BudgetStatusUrgent {
		t.Fatalf("expected unbudgeted gifts spending to surface, got %+v", gifts)
	}

	if report.TotalLimit != 6_000_000 || report.TotalSpent != 3_200_000 {
		t.Fatalf("unexpected totals %+v", report)
	}
	if report.DaysInMonth != 31 || report.DaysElapsed != 10 || report.DaysRemaining != 21 {
		t.Fatalf("unexpected day counts %+v", report)
	}
	if report.DailyBurn != 320_000 || report.ProjectedSpend != 9_920_000 {
		t.Fatalf("unexpected burn %v / %v", report.DailyBurn, report.ProjectedSpend)
	}
	if report.Lines[0].Category != "Gifts" {
		t.Fatalf("expected lines sorted by percent used, first was %s", report.Lines[0].Category)
	}
}

func TestBudgetReportDaysForPastAndFutureMonths(t *testing.T) {
	past := ComputeBudgetReport(nil, nil, day(2026, 2, 1), day(2026, 3, 10))
	if past.DaysElapsed != 28 || past.DaysRemaining != 0 {
		t.Fatalf("past month: %+v", past)
	}
	future := ComputeBudgetReport(nil, nil, day(2026, 4, 1), day(2026, 3, 10))
	if future.DaysElapsed != 0 || future.DailyBurn != 0 {
		t.Fatalf("future month: %+v", future)
	}
}
