package domain

import (
	"strings"
	"testing"
	"time"
)

// spendEachMonth records one expense per month for the given amounts, starting
// at January of year.
func spendEachMonth(year int, category string, amounts ...float64) []Transaction {
	transactions := make([]Transaction, 0, len(amounts))
	for i, amount := range amounts {
		if amount > 0 {
			transactions = append(transactions, txn(day(year, time.January+time.Month(i), 5), category, CategoryTypeExpense, amount, 0))
		}
	}
	return transactions
}

func expenseItem(category string, limit float64) BudgetItem {
	return BudgetItem{Category: category, CategoryType: categoryType(CategoryTypeExpense), MonthlyLimit: limit}
}

func statsFor(t *testing.T, facts AdvisorFacts, category string) BudgetCategoryStats {
	t.Helper()
	for _, stats := range facts.Categories {
		if stats.Category == category {
			return stats
		}
	}
	t.Fatalf("no stats for %s in %+v", category, facts.Categories)
	return BudgetCategoryStats{}
}

func TestAdvisorFactsYearToDateAndRecentWindow(t *testing.T) {
	// Given food spend from January to September and today in October
	transactions := spendEachMonth(2026, "Food", 2_000_000, 2_000_000, 2_000_000, 2_000_000, 2_000_000, 2_000_000, 3_000_000, 3_000_000, 3_000_000)
	items := []BudgetItem{expenseItem("Food", 2_000_000)}

	// When the advisor facts are computed
	facts := ComputeAdvisorFacts(items, nil, transactions, day(2026, 10, 19))

	// Then the year-to-date window is the nine full months and the recent window the last three
	if facts.YearToDateMonths != 9 {
		t.Fatalf("year-to-date months = %d, want 9", facts.YearToDateMonths)
	}
	food := statsFor(t, facts, "Food")
	if food.YearToDateAverage != 2_333_333 || food.AverageSpend != 3_000_000 {
		t.Fatalf("averages ytd=%v recent=%v", food.YearToDateAverage, food.AverageSpend)
	}
	if len(food.MonthlySpend) != 9 || food.MonthlySpend[0] != (MonthSpend{Month: "2026-01", Spent: 2_000_000}) || food.MonthlySpend[8].Month != "2026-09" {
		t.Fatalf("monthly spend %+v", food.MonthlySpend)
	}
	// And the last three months run 50% above the six before them
	if food.Trend != SpendTrendRising || food.Pattern != SpendPatternSteady {
		t.Fatalf("trend=%s pattern=%s", food.Trend, food.Pattern)
	}
	// And a steady category's baseline follows the last three months
	if food.BaselineLimit != 3_000_000 {
		t.Fatalf("baseline = %v, want 3,000,000", food.BaselineLimit)
	}
}

func TestAdvisorFactsTrendAndPattern(t *testing.T) {
	today := day(2026, 10, 19)
	tests := []struct {
		name     string
		amounts  []float64
		trend    SpendTrend
		pattern  SpendPattern
		baseline float64
	}{
		{
			name:     "falling spend keeps the lower recent average",
			amounts:  []float64{3_000_000, 3_000_000, 3_000_000, 3_000_000, 3_000_000, 3_000_000, 2_000_000, 2_000_000, 2_000_000},
			trend:    SpendTrendFalling,
			pattern:  SpendPatternSteady,
			baseline: 2_000_000,
		},
		{
			name:     "small moves count as steady",
			amounts:  []float64{1_000_000, 1_000_000, 1_000_000, 1_000_000, 1_000_000, 1_000_000, 1_100_000, 1_100_000, 1_100_000},
			trend:    SpendTrendSteady,
			pattern:  SpendPatternSteady,
			baseline: 1_100_000,
		},
		{
			name:     "a few large months set aside the year-to-date average",
			amounts:  []float64{0, 0, 6_000_000, 0, 0, 0, 0, 3_000_000, 0},
			trend:    SpendTrendSteady,
			pattern:  SpendPatternOccasional,
			baseline: 1_000_000,
		},
		{
			name:     "volatile spend takes the higher of the two averages",
			amounts:  []float64{500_000, 4_000_000, 500_000, 4_000_000, 500_000, 4_000_000, 500_000, 500_000, 1_000_000},
			trend:    SpendTrendFalling,
			pattern:  SpendPatternVariable,
			baseline: 1_730_000,
		},
		{
			name:     "spend that only started recently is rising",
			amounts:  []float64{0, 0, 0, 0, 0, 0, 900_000, 900_000, 900_000},
			trend:    SpendTrendRising,
			pattern:  SpendPatternSteady,
			baseline: 900_000,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			facts := ComputeAdvisorFacts([]BudgetItem{expenseItem("Fun", 1_000_000)}, nil, spendEachMonth(2026, "Fun", test.amounts...), today)
			fun := statsFor(t, facts, "Fun")
			if fun.Trend != test.trend || fun.Pattern != test.pattern || fun.BaselineLimit != test.baseline {
				t.Fatalf("trend=%s pattern=%s baseline=%v, want %s %s %v", fun.Trend, fun.Pattern, fun.BaselineLimit, test.trend, test.pattern, test.baseline)
			}
		})
	}
}

func TestAdvisorFactsEarlyInTheYear(t *testing.T) {
	// Given November and December last year, January this year, and today in February
	transactions := []Transaction{
		txn(day(2025, 11, 5), "Food", CategoryTypeExpense, 1_000_000, 0),
		txn(day(2025, 12, 5), "Food", CategoryTypeExpense, 2_000_000, 0),
		txn(day(2026, 1, 5), "Food", CategoryTypeExpense, 3_000_000, 0),
	}

	facts := ComputeAdvisorFacts([]BudgetItem{expenseItem("Food", 2_000_000)}, nil, transactions, day(2026, 2, 10))

	// Then the recent window reaches into last year while year to date is January only
	food := statsFor(t, facts, "Food")
	if facts.YearToDateMonths != 1 || food.YearToDateAverage != 3_000_000 || food.AverageSpend != 2_000_000 {
		t.Fatalf("ytd months=%d ytd=%v recent=%v", facts.YearToDateMonths, food.YearToDateAverage, food.AverageSpend)
	}
	if len(food.MonthlySpend) != 3 || food.MonthlySpend[0].Month != "2025-11" {
		t.Fatalf("monthly spend %+v", food.MonthlySpend)
	}
	// And there are no earlier months this year to compare with
	if food.Trend != SpendTrendUnknown {
		t.Fatalf("trend = %s, want unknown", food.Trend)
	}
}

func TestAdvisorFactsInJanuary(t *testing.T) {
	transactions := []Transaction{txn(day(2025, 12, 5), "Food", CategoryTypeExpense, 2_000_000, 0)}

	facts := ComputeAdvisorFacts([]BudgetItem{expenseItem("Food", 2_000_000)}, nil, transactions, day(2026, 1, 10))

	food := statsFor(t, facts, "Food")
	if facts.YearToDateMonths != 0 || food.YearToDateAverage != 0 || food.BaselineLimit != 670_000 {
		t.Fatalf("ytd months=%d ytd=%v baseline=%v", facts.YearToDateMonths, food.YearToDateAverage, food.BaselineLimit)
	}
}

func TestAdvisorFactsBaselineRoundsUpToTenThousand(t *testing.T) {
	transactions := spendEachMonth(2026, "Apps", 0, 0, 0, 0, 0, 0, 101_000, 101_000, 101_000)

	facts := ComputeAdvisorFacts([]BudgetItem{expenseItem("Apps", 0)}, nil, transactions, day(2026, 10, 1))

	if apps := statsFor(t, facts, "Apps"); apps.BaselineLimit != 110_000 {
		t.Fatalf("baseline = %v, want 110,000", apps.BaselineLimit)
	}
}

func TestAdvisorFactsListBudgetedCategoriesWithHistoryOnly(t *testing.T) {
	// Given a budget row that was only spent on earlier this year
	transactions := spendEachMonth(2026, "Travel", 0, 0, 0, 0, 9_000_000)

	facts := ComputeAdvisorFacts([]BudgetItem{expenseItem("Travel", 0)}, nil, transactions, day(2026, 10, 19))

	// Then it is still in the facts, so its limit can be planned
	travel := statsFor(t, facts, "Travel")
	if travel.SpentSoFar != 0 || travel.YearToDateAverage != 1_000_000 || travel.Pattern != SpendPatternOccasional {
		t.Fatalf("travel %+v", travel)
	}
}

func TestApplyFactsFillsMissingBaselineSuggestions(t *testing.T) {
	facts := AdvisorFacts{Month: "2026-10", Categories: []BudgetCategoryStats{
		{Category: "Food", CurrentLimit: 2_000_000, AverageSpend: 3_000_000, YearToDateAverage: 2_330_000, Trend: SpendTrendRising, Pattern: SpendPatternSteady, BaselineLimit: 3_000_000},
		{Category: "Coffee", CurrentLimit: 500_000, AverageSpend: 480_000, YearToDateAverage: 470_000, Trend: SpendTrendSteady, Pattern: SpendPatternSteady, BaselineLimit: 480_000},
		{Category: "Gifts", CurrentLimit: 0, AverageSpend: 0, YearToDateAverage: 1_000_000, Trend: SpendTrendFalling, Pattern: SpendPatternOccasional, BaselineLimit: 1_000_000},
		{Category: "Shoes", CurrentLimit: 1_000_000, AverageSpend: 2_000_000, Trend: SpendTrendRising, Pattern: SpendPatternVariable, BaselineLimit: 2_000_000},
		{Category: "New", CurrentLimit: 300_000, SpentSoFar: 100_000},
	}}
	// Given the model suggested Shoes only
	advice := BudgetAdvice{Briefing: "ok", SuggestedLimits: []SuggestedLimit{{Category: "shoes", SuggestedLimit: 1_500_000, Rationale: "Cut back"}}}

	if err := advice.ApplyFacts(facts); err != nil {
		t.Fatal(err)
	}

	// Then the model's suggestion stays first with the engine's figures attached
	shoes := advice.SuggestedLimits[0]
	if shoes.Category != "Shoes" || shoes.SuggestedLimit != 1_500_000 || shoes.BaselineLimit != 2_000_000 || shoes.Trend != SpendTrendRising || shoes.Pattern != SpendPatternVariable {
		t.Fatalf("shoes %+v", shoes)
	}
	// And the engine adds baselines that differ from the limit by 10% or more, largest change first
	if len(advice.SuggestedLimits) != 3 {
		t.Fatalf("suggestions %+v", advice.SuggestedLimits)
	}
	gifts, food := advice.SuggestedLimits[1], advice.SuggestedLimits[2]
	if gifts.Category != "Gifts" || gifts.SuggestedLimit != 1_000_000 || !strings.Contains(gifts.Rationale, "only some months") {
		t.Fatalf("gifts %+v", gifts)
	}
	if food.Category != "Food" || food.SuggestedLimit != 3_000_000 || food.YearToDateAverage != 2_330_000 || !strings.Contains(food.Rationale, "above earlier") {
		t.Fatalf("food %+v", food)
	}
}
