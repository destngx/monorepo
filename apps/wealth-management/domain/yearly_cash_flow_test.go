package domain

import (
	"reflect"
	"testing"
	"time"
)

func TestComputeYearlyCashFlow(t *testing.T) {
	today := day(2026, 10, 16)
	refund := txn(day(2026, 4, 2), "Refunds/Reimbursements", CategoryTypeIncome, 0, 300_000)
	refund.Memo = text("3/2026")
	broken := txn(day(2026, 5, 1), "Salary", CategoryTypeIncome, 0, 99_000_000)
	broken.Issues = []string{"bad amount"}
	txs := []Transaction{
		txn(day(2026, 1, 5), "Salary", CategoryTypeIncome, 0, 30_000_000),
		txn(day(2026, 1, 9), "Food", CategoryTypeExpense, 4_000_000, 0),
		txn(day(2026, 1, 20), "Rent", CategoryTypeExpense, 8_000_000, 0),
		txn(day(2026, 3, 5), "Salary", CategoryTypeIncome, 0, 30_000_000),
		txn(day(2026, 3, 7), "Food", CategoryTypeExpense, 6_000_000, 0),
		refund, // attributed to March (WM-RULE-4)
		broken,
		txn(day(2026, 3, 8), "Transfer", CategoryTypeNonBudget, 5_000_000, 0),
		txn(day(2026, 10, 1), "Bonus", CategoryTypeIncome, 0, 10_000_000),
		txn(day(2025, 2, 5), "Salary", CategoryTypeIncome, 0, 20_000_000),
		txn(day(2025, 2, 9), "Food", CategoryTypeExpense, 5_000_000, 0),
		txn(day(2025, 11, 9), "Food", CategoryTypeExpense, 9_000_000, 0),
		txn(day(2023, 6, 1), "Salary", CategoryTypeIncome, 0, 1_000_000),
	}

	got := ComputeYearlyCashFlow(txs, nil, 2026, today)

	if got.Year != 2026 || got.MonthsElapsed != 10 {
		t.Fatalf("year = %d, elapsed = %d", got.Year, got.MonthsElapsed)
	}
	if !reflect.DeepEqual(got.Years, []int{2026, 2025, 2023}) {
		t.Fatalf("years = %v", got.Years)
	}
	if len(got.Months) != 12 || got.Months[0].Month != "2026-01" || got.Months[11].Month != "2026-12" {
		t.Fatalf("months = %+v", got.Months)
	}
	if got.Months[2].Income != 30_300_000 || got.Months[2].Expense != 6_000_000 || got.Months[3].Income != 0 {
		t.Fatalf("march = %+v, april = %+v", got.Months[2], got.Months[3])
	}
	want := CashFlow{Month: "2026", Income: 70_300_000, Expense: 18_000_000, CashFlow: 52_300_000, SavingsRate: 74.4}
	if got.Total != want {
		t.Fatalf("total = %+v, want %+v", got.Total, want)
	}
	if got.MonthlyAverage.Income != 7_030_000 || got.MonthlyAverage.Expense != 1_800_000 || got.MonthlyAverage.SavingsRate != 74.4 {
		t.Fatalf("average = %+v", got.MonthlyAverage)
	}
	// Same span of 2025 (Jan-Oct): November's expense is left out.
	if got.PreviousYear.Month != "2025" || got.PreviousYear.Income != 20_000_000 || got.PreviousYear.Expense != 5_000_000 {
		t.Fatalf("previous year = %+v", got.PreviousYear)
	}
	if got.IncomeChangePercent == nil || *got.IncomeChangePercent != 251.5 || got.ExpenseChangePercent == nil || *got.ExpenseChangePercent != 260 {
		t.Fatalf("changes = %v, %v", got.IncomeChangePercent, got.ExpenseChangePercent)
	}

	incomeNames := categoryNames(got.Income)
	if !reflect.DeepEqual(incomeNames, []string{"Salary", "Bonus", "Refunds/Reimbursements"}) {
		t.Fatalf("income categories = %v", incomeNames)
	}
	if expenseNames := categoryNames(got.Expense); !reflect.DeepEqual(expenseNames, []string{"Food", "Rent"}) {
		t.Fatalf("expense categories = %v", expenseNames)
	}
	food := got.Expense[0]
	if food.Total != 10_000_000 || len(food.Values) != 12 || food.Values[0] != 4_000_000 || food.Values[2] != 6_000_000 {
		t.Fatalf("food = %+v", food)
	}
	for i, month := range got.Months {
		income, expense := 0.0, 0.0
		for _, row := range got.Income {
			income += row.Values[i]
		}
		for _, row := range got.Expense {
			expense += row.Values[i]
		}
		if income != month.Income || expense != month.Expense {
			t.Fatalf("%s: categories sum to %v/%v, month is %+v", month.Month, income, expense, month)
		}
	}
}

func TestComputeYearlyCashFlowPastAndFutureYears(t *testing.T) {
	today := day(2026, 10, 16)
	txs := []Transaction{
		txn(day(2025, 12, 5), "Food", CategoryTypeExpense, 2_000_000, 0),
		txn(day(2024, 12, 5), "Food", CategoryTypeExpense, 1_000_000, 0),
	}

	past := ComputeYearlyCashFlow(txs, nil, 2025, today)
	if past.MonthsElapsed != 12 || past.PreviousYear.Expense != 1_000_000 || past.MonthlyAverage.Expense != 2_000_000.0/12 {
		t.Fatalf("past = %+v", past)
	}
	if past.IncomeChangePercent != nil || past.MonthlyAverage.SavingsRate != 0 {
		t.Fatalf("no income: change = %v, rate = %v", past.IncomeChangePercent, past.MonthlyAverage.SavingsRate)
	}
	if !reflect.DeepEqual(past.Years, []int{2026, 2025, 2024}) {
		t.Fatalf("years always include the current year: %v", past.Years)
	}

	future := ComputeYearlyCashFlow(txs, nil, 2027, today)
	if future.MonthsElapsed != 0 || future.MonthlyAverage != (CashFlow{Month: "2027"}) || len(future.Income) != 0 || future.Income == nil {
		t.Fatalf("future = %+v", future)
	}
	if !reflect.DeepEqual(future.Years, []int{2027, 2026, 2025, 2024}) {
		t.Fatalf("years include the requested one: %v", future.Years)
	}
}

func categoryNames(rows []CategoryMonths) []string {
	names := make([]string, len(rows))
	for i, row := range rows {
		names[i] = row.Category
	}
	return names
}

func TestYearlyCashFlowVerdict(t *testing.T) {
	today := day(2025, 12, 31)
	year := func(income, expense float64, deficitMonth bool) YearVerdict {
		txs := []Transaction{txn(day(2025, 1, 5), "Salary", CategoryTypeIncome, 0, income), txn(day(2025, 2, 5), "Rent", CategoryTypeExpense, expense, 0)}
		if deficitMonth {
			txs = append(txs, txn(day(2025, 3, 5), "Trip", CategoryTypeExpense, 1, 0))
		}
		return ComputeYearlyCashFlow(txs, nil, 2025, today).Verdict
	}
	tests := []struct {
		name    string
		verdict YearVerdict
		want    YearVerdict
	}{
		{"30% and above is on target", year(100, 70, false), YearVerdict{Status: VerdictOnTarget, SavingsRate: 30, DeficitMonths: 1}},
		{"20% is healthy", year(100, 80, false), YearVerdict{Status: VerdictHealthy, SavingsRate: 20, DeficitMonths: 1}},
		{"10% is worth watching", year(100, 89, true), YearVerdict{Status: VerdictWatch, SavingsRate: 10, DeficitMonths: 2}},
		{"below 10% is at risk", year(100, 95, false), YearVerdict{Status: VerdictAtRisk, SavingsRate: 5, DeficitMonths: 1}},
		{"spending more than earned is at risk", year(100, 150, false), YearVerdict{Status: VerdictAtRisk, SavingsRate: -50, DeficitMonths: 1}},
		{"no income has no verdict", year(0, 50, false), YearVerdict{Status: VerdictNoData, DeficitMonths: 1}},
	}
	for _, tc := range tests {
		if tc.verdict != tc.want {
			t.Errorf("%s: got %+v, want %+v", tc.name, tc.verdict, tc.want)
		}
	}
}

func TestYearlyCashFlowProjection(t *testing.T) {
	txs := []Transaction{
		txn(day(2026, 1, 5), "Salary", CategoryTypeIncome, 0, 100),
		txn(day(2026, 1, 9), "Food", CategoryTypeExpense, 60, 0),
		txn(day(2026, 2, 5), "Salary", CategoryTypeIncome, 0, 100),
		txn(day(2026, 2, 9), "Food", CategoryTypeExpense, 40, 0),
		// March is under way: income not in yet, spending already above average.
		txn(day(2026, 3, 2), "Food", CategoryTypeExpense, 70, 0),
	}

	got := ComputeYearlyCashFlow(txs, nil, 2026, day(2026, 3, 10)).Projection
	// Completed Jan-Feb average 100 / 50. March counts as max(actual, average);
	// April-December at the average.
	want := &YearProjection{BasisMonths: 2, CashFlow: CashFlow{Month: "2026", Income: 1200, Expense: 620, CashFlow: 580, SavingsRate: 48.33}}
	if got == nil || *got != *want {
		t.Fatalf("projection = %+v, want %+v", got, want)
	}

	january := ComputeYearlyCashFlow(txs, nil, 2026, day(2026, 1, 20)).Projection
	if january == nil || january.BasisMonths != 0 || january.Income != 1200 || january.Expense != 720 {
		t.Fatalf("january uses the month so far: %+v", january)
	}
	for _, other := range []int{2025, 2027} {
		if p := ComputeYearlyCashFlow(txs, nil, other, day(2026, 3, 10)).Projection; p != nil {
			t.Fatalf("%d has a projection: %+v", other, p)
		}
	}
}

func TestYearlyCashFlowSpikes(t *testing.T) {
	txs := []Transaction{}
	for month := 1; month <= 5; month++ {
		txs = append(txs, txn(day(2025, time.Month(month), 3), "Food", CategoryTypeExpense, 100, 0))
		txs = append(txs, txn(day(2025, time.Month(month), 4), "Rent", CategoryTypeExpense, 200, 0))
	}
	txs = append(txs,
		txn(day(2025, 4, 20), "Travel", CategoryTypeExpense, 500, 0),
		txn(day(2025, 4, 21), "Food", CategoryTypeExpense, 50, 0),
		txn(day(2025, 5, 20), "Food", CategoryTypeExpense, 100, 0), // 400 vs 300: below 1.5x
	)

	got := ComputeYearlyCashFlow(txs, nil, 2025, day(2026, 1, 1)).Spikes
	want := []ExpenseSpike{{Month: "2025-04", Expense: 850, Typical: 300, Ratio: 2.83, Category: "Travel", CategoryAmount: 500, CategoryTypical: 0}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("spikes = %+v, want %+v", got, want)
	}

	few := ComputeYearlyCashFlow(txs[:6], nil, 2025, day(2026, 1, 1)).Spikes
	if few == nil || len(few) != 0 {
		t.Fatalf("three months are too few to call a month unusual: %+v", few)
	}
}

func TestParseCategoryNature(t *testing.T) {
	for cell, want := range map[string]CategoryNature{
		"Fixed": NatureFixed, " essential ": NatureEssential, "FLEXIBLE": NatureFlexible, "": "", "maybe": "",
	} {
		if got := ParseCategoryNature(cell); got != want {
			t.Errorf("ParseCategoryNature(%q) = %q, want %q", cell, got, want)
		}
	}
}

func TestYearlyCashFlowExpenseSplit(t *testing.T) {
	categories := []Category{
		{Name: "Rent", Type: CategoryTypeExpense, Nature: NatureFixed},
		{Name: "Food", Type: CategoryTypeExpense, Nature: NatureEssential},
		{Name: "Shopping", Type: CategoryTypeExpense, Nature: NatureFlexible},
		{Name: "Gifts", Type: CategoryTypeExpense},
		{Name: "Salary", Type: CategoryTypeIncome},
	}
	txs := []Transaction{
		txn(day(2026, 1, 5), "Salary", CategoryTypeIncome, 0, 100),
		txn(day(2026, 1, 6), "rent ", CategoryTypeExpense, 40, 0), // matched ignoring case and spaces
		txn(day(2026, 1, 7), "Food", CategoryTypeExpense, 20, 0),
		txn(day(2026, 2, 5), "Salary", CategoryTypeIncome, 0, 100),
		txn(day(2026, 2, 6), "Rent", CategoryTypeExpense, 40, 0),
		txn(day(2026, 2, 7), "Shopping", CategoryTypeExpense, 10, 0),
		txn(day(2026, 2, 8), "Gifts", CategoryTypeExpense, 15, 0),
		txn(day(2026, 2, 9), "Repairs", CategoryTypeExpense, 5, 0), // not in Categories at all
	}

	got := ComputeYearlyCashFlow(txs, categories, 2026, day(2026, 2, 20)).ExpenseSplit

	if got.Fixed.Total != 80 || got.Fixed.Values[0] != 40 || got.Fixed.Values[1] != 40 || len(got.Fixed.Values) != 12 {
		t.Fatalf("fixed = %+v", got.Fixed)
	}
	if got.Essential.Total != 20 || got.Flexible.Total != 10 || got.Untagged.Total != 20 || got.Untagged.Values[1] != 20 {
		t.Fatalf("essential = %+v, flexible = %+v, untagged = %+v", got.Essential, got.Flexible, got.Untagged)
	}
	// Needs are fixed plus essential: 100 of 200 income.
	if got.NeedsShareOfIncome == nil || *got.NeedsShareOfIncome != 50 || got.NeedsShareComfortable != 50 {
		t.Fatalf("needs share = %v of %v", got.NeedsShareOfIncome, got.NeedsShareComfortable)
	}
	if !reflect.DeepEqual(got.UntaggedCategories, []string{"Gifts", "Repairs"}) {
		t.Fatalf("untagged categories = %v", got.UntaggedCategories)
	}
	if got.Fixed.Category != string(NatureFixed) || got.Untagged.Category != "untagged" {
		t.Fatalf("labels = %q, %q", got.Fixed.Category, got.Untagged.Category)
	}
	if got.Room == nil || len(got.Room) != 0 {
		t.Fatalf("two months are too few to know the lean months: %+v", got.Room)
	}

	empty := ComputeYearlyCashFlow(nil, nil, 2026, day(2026, 2, 20)).ExpenseSplit
	if empty.NeedsShareOfIncome != nil || empty.UntaggedCategories == nil || empty.Room == nil || len(empty.Fixed.Values) != 12 {
		t.Fatalf("empty = %+v", empty)
	}
}

func TestYearlyCashFlowRoomToCut(t *testing.T) {
	categories := []Category{
		{Name: "Rent", Type: CategoryTypeExpense, Nature: NatureFixed},
		{Name: "Food", Type: CategoryTypeExpense, Nature: NatureEssential},
		{Name: "Shopping", Type: CategoryTypeExpense, Nature: NatureFlexible},
		{Name: "Phone", Type: CategoryTypeExpense, Nature: NatureFlexible},
		{Name: "Gifts", Type: CategoryTypeExpense},
	}
	food := []float64{100, 80, 120, 100, 60}
	shopping := []float64{70, 0, 40, 10, 60}
	txs := []Transaction{}
	for i := range food {
		month := time.Month(i + 1)
		txs = append(txs,
			txn(day(2025, month, 1), "Rent", CategoryTypeExpense, 500+float64(i)*100, 0),
			txn(day(2025, month, 2), "Food", CategoryTypeExpense, food[i], 0),
			txn(day(2025, month, 4), "Phone", CategoryTypeExpense, 10, 0),
			txn(day(2025, month, 5), "Gifts", CategoryTypeExpense, float64(i)*50, 0),
		)
		if shopping[i] > 0 {
			txs = append(txs, txn(day(2025, month, 3), "Shopping", CategoryTypeExpense, shopping[i], 0))
		}
	}

	got := ComputeYearlyCashFlow(txs, categories, 2025, day(2025, 6, 2)).ExpenseSplit

	// Typical is the median month, lean the 25th percentile: what you have
	// already lived on. Fixed and untagged categories get no room; a steady
	// category (Phone) has none.
	want := []CategoryRoom{
		{Category: "Shopping", Nature: NatureFlexible, Typical: 40, Lean: 10, PerMonth: 30, PerYear: 360},
		{Category: "Food", Nature: NatureEssential, Typical: 100, Lean: 80, PerMonth: 20, PerYear: 240},
	}
	if !reflect.DeepEqual(got.Room, want) {
		t.Fatalf("room = %+v, want %+v", got.Room, want)
	}
	// The total compares whole months, since categories rarely have their
	// lean month together: essential+flexible per month is 180, 90, 170, 120,
	// 130, so typical 130 and lean 120, not the 50 the category rows add up to.
	if got.RoomPerMonth != 10 || got.RoomPerYear != 120 {
		t.Fatalf("room = %v a month, %v a year", got.RoomPerMonth, got.RoomPerYear)
	}

	// A month under way is not a lean month yet: on 20 May only January-April count.
	partial := ComputeYearlyCashFlow(txs, categories, 2025, day(2025, 5, 20)).ExpenseSplit
	if len(partial.Room) != 2 || partial.Room[0].Category != "Shopping" || partial.Room[0].Typical != 25 || partial.Room[1].Lean != 95 {
		t.Fatalf("room on 20 May = %+v", partial.Room)
	}
}

func TestPercentile(t *testing.T) {
	values := []float64{60, 120, 80, 100, 100}
	if got := percentile(values, 0.25); got != 80 {
		t.Fatalf("p25 = %v", got)
	}
	if got := percentile([]float64{0, 10}, 0.25); got != 2.5 {
		t.Fatalf("interpolated p25 = %v", got)
	}
	if got := percentile(nil, 0.25); got != 0 {
		t.Fatalf("empty = %v", got)
	}
	if values[0] != 60 {
		t.Fatal("input was sorted in place")
	}
}
