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
	// Income reads as money received, not spend clamped to zero.
	if salary, ok := lines["Salary"]; !ok || salary.Spent != 20_000_000 || salary.YearlySpent != 20_000_000 {
		t.Fatalf("expected salary to show the 20M received, got %+v", salary)
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

func TestGivenWagesWithAnExpectedAmountWhenComputingBudgetThenTheLineShowsWhatCameIn(t *testing.T) {
	items := []BudgetItem{{Category: "Wages", CategoryType: categoryType(CategoryTypeIncome), MonthlyLimit: 30_000_000}}
	transactions := []Transaction{
		txn(day(2026, 3, 5), "Wages", CategoryTypeIncome, 0, 25_000_000),
		txn(day(2026, 3, 6), "Wages", CategoryTypeIncome, 1_000_000, 0), // a correction paid back
	}
	report := ComputeBudgetReport(items, transactions, day(2026, 3, 1), day(2026, 3, 10))

	if len(report.Lines) != 1 {
		t.Fatalf("expected one line, got %+v", report.Lines)
	}
	wages := report.Lines[0]
	if wages.Spent != 24_000_000 || wages.Remaining != 6_000_000 || wages.PercentUsed != 80 {
		t.Fatalf("expected 24M of 30M received, got %+v", wages)
	}
	if report.TotalSpent != 0 || report.TotalLimit != 0 {
		t.Fatalf("income must stay out of the expense totals, got %+v", report)
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

func budgetHistoryFixture() ([]BudgetItem, []Transaction) {
	items := []BudgetItem{
		{Category: "Food", CategoryType: categoryType(CategoryTypeExpense), MonthlyLimit: 4_000_000, MonthlyLimits: map[string]float64{"2026-03": 5_000_000}},
		{Category: "Fun", CategoryType: categoryType(CategoryTypeExpense), MonthlyLimit: 1_000_000},
		{Category: "Rent", CategoryType: categoryType(CategoryTypeExpense), MonthlyLimit: 10_000_000},
		{Category: "Salary", CategoryType: categoryType(CategoryTypeIncome), MonthlyLimit: 30_000_000},
	}
	transactions := []Transaction{
		txn(day(2026, 2, 4), "Food", CategoryTypeExpense, 1_000_000, 0),
		txn(day(2026, 3, 2), "Food", CategoryTypeExpense, 2_500_000, 0),
		txn(day(2026, 3, 5), "Gifts", CategoryTypeExpense, 700_000, 0),
		txn(day(2026, 3, 9), "Rent", CategoryTypeExpense, 12_000_000, 0),
		txn(day(2026, 3, 5), "Salary", CategoryTypeIncome, 0, 20_000_000),
	}
	return items, transactions
}

func TestGivenCurrentMonthBurnWhenComputingBudgetThenBurnOutDateIsWhenTheLimitRunsOut(t *testing.T) {
	items, transactions := budgetHistoryFixture()
	items = append(items, BudgetItem{Category: "Travel", CategoryType: categoryType(CategoryTypeExpense), MonthlyLimit: 100_000_000})
	transactions = append(transactions, txn(day(2026, 3, 1), "Travel", CategoryTypeExpense, 1_000_000, 0))
	report := ComputeBudgetReport(items, transactions, day(2026, 3, 1), day(2026, 3, 10))

	burnOut := map[string]*string{}
	for _, line := range report.Lines {
		burnOut[line.Category] = line.BurnOutDate
	}
	// Food: 2.5M over 10 days = 250k/day; the 2.5M left lasts 10 more days.
	if got := burnOut["Food"]; got == nil || *got != "2026-03-20" {
		t.Fatalf("food burn-out = %v", got)
	}
	// Rent is already over its limit, so it ran out by today.
	if got := burnOut["Rent"]; got == nil || *got != "2026-03-10" {
		t.Fatalf("rent burn-out = %v", got)
	}
	for _, category := range []string{"Fun", "Gifts", "Travel", "Salary"} {
		if got := burnOut[category]; got != nil {
			t.Fatalf("%s burn-out = %s, want null (no spend, no limit, after month end or income)", category, *got)
		}
	}
}

func TestGivenPastMonthWhenComputingBudgetThenBurnOutDateIsNull(t *testing.T) {
	items, transactions := budgetHistoryFixture()
	report := ComputeBudgetReport(items, transactions, day(2026, 3, 1), day(2026, 4, 2))
	for _, line := range report.Lines {
		if line.BurnOutDate != nil {
			t.Fatalf("%s burn-out = %s in a closed month", line.Category, *line.BurnOutDate)
		}
	}
}

func TestGivenSeveralMonthsWhenComputingBudgetHistoryThenRowsHoldLimitAndSpentPerMonth(t *testing.T) {
	items, transactions := budgetHistoryFixture()
	months := BudgetHistoryMonths(day(2026, 3, 1), 2)
	history := ComputeBudgetHistory(map[int][]BudgetItem{2026: items}, transactions, months, day(2026, 3, 10))

	if len(history.Months) != 2 || history.Months[0] != "2026-02" || history.Months[1] != "2026-03" {
		t.Fatalf("months = %v", history.Months)
	}
	order := []string{}
	rows := map[string]BudgetHistoryRow{}
	for _, row := range history.Rows {
		order = append(order, row.Category)
		rows[row.Category] = row
	}
	if len(order) != 4 || order[0] != "Rent" || order[1] != "Food" || order[2] != "Gifts" || order[3] != "Fun" {
		t.Fatalf("rows ordered by total spent, got %v", order)
	}
	food := rows["Food"].Values
	if len(food) != 2 || food[0] != (BudgetHistoryValue{Month: "2026-02", Limit: 4_000_000, Spent: 1_000_000}) ||
		food[1] != (BudgetHistoryValue{Month: "2026-03", Limit: 5_000_000, Spent: 2_500_000}) {
		t.Fatalf("food = %+v", food)
	}
	if gifts := rows["Gifts"].Values; gifts[0] != (BudgetHistoryValue{Month: "2026-02"}) || gifts[1].Spent != 700_000 || gifts[1].Limit != 0 {
		t.Fatalf("unbudgeted gifts = %+v", gifts)
	}
}

func TestGivenMissingYearWhenComputingBudgetHistoryThenThatYearHasNoLimits(t *testing.T) {
	items, transactions := budgetHistoryFixture()
	transactions = append(transactions, txn(day(2025, 12, 3), "Food", CategoryTypeExpense, 300_000, 0))
	history := ComputeBudgetHistory(map[int][]BudgetItem{2026: items}, transactions, BudgetHistoryMonths(day(2026, 1, 1), 2), day(2026, 3, 10))
	if history.Months[0] != "2025-12" {
		t.Fatalf("months = %v", history.Months)
	}
	for _, row := range history.Rows {
		if row.Category == "Food" && row.Values[0] != (BudgetHistoryValue{Month: "2025-12", Spent: 300_000}) {
			t.Fatalf("food december = %+v", row.Values[0])
		}
	}
}
