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
	facts := ComputeAdvisorFacts(items, nil, transactions, day(2026, 10, 19), 0, nil)

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
	// And an untagged steady category follows the last three months
	if food.RecommendedLimit != 3_000_000 || food.Direction != LimitIncrease {
		t.Fatalf("recommended %v %s, want 3,000,000 increase", food.RecommendedLimit, food.Direction)
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
			// Given no limit yet, so the recommendation is the untagged target
			facts := ComputeAdvisorFacts([]BudgetItem{expenseItem("Fun", 0)}, nil, spendEachMonth(2026, "Fun", test.amounts...), today, 0, nil)
			fun := statsFor(t, facts, "Fun")
			if fun.Trend != test.trend || fun.Pattern != test.pattern || fun.RecommendedLimit != test.baseline {
				t.Fatalf("trend=%s pattern=%s recommended=%v, want %s %s %v", fun.Trend, fun.Pattern, fun.RecommendedLimit, test.trend, test.pattern, test.baseline)
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

	facts := ComputeAdvisorFacts([]BudgetItem{expenseItem("Food", 2_000_000)}, nil, transactions, day(2026, 2, 10), 0, nil)

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

	facts := ComputeAdvisorFacts([]BudgetItem{expenseItem("Food", 2_000_000)}, nil, transactions, day(2026, 1, 10), 0, nil)

	// Then there is no year to date yet and the recommendation leans on the last three months:
	// about 667,000 a month plus the 10% cushion, rounded up
	food := statsFor(t, facts, "Food")
	if facts.YearToDateMonths != 0 || food.YearToDateAverage != 0 || food.RecommendedLimit != 740_000 || food.Direction != LimitLower {
		t.Fatalf("ytd months=%d ytd=%v recommended=%v %s", facts.YearToDateMonths, food.YearToDateAverage, food.RecommendedLimit, food.Direction)
	}
}

func TestAdvisorFactsRecommendationRoundsUpToTenThousand(t *testing.T) {
	transactions := spendEachMonth(2026, "Apps", 0, 0, 0, 0, 0, 0, 101_000, 101_000, 101_000)

	facts := ComputeAdvisorFacts([]BudgetItem{expenseItem("Apps", 0)}, nil, transactions, day(2026, 10, 1), 0, nil)

	if apps := statsFor(t, facts, "Apps"); apps.RecommendedLimit != 110_000 {
		t.Fatalf("recommended = %v, want 110,000", apps.RecommendedLimit)
	}
}

func TestAdvisorFactsListBudgetedCategoriesWithHistoryOnly(t *testing.T) {
	// Given a budget row that was only spent on earlier this year
	transactions := spendEachMonth(2026, "Travel", 0, 0, 0, 0, 9_000_000)

	facts := ComputeAdvisorFacts([]BudgetItem{expenseItem("Travel", 0)}, nil, transactions, day(2026, 10, 19), 0, nil)

	// Then it is still in the facts, so its limit can be planned
	travel := statsFor(t, facts, "Travel")
	if travel.SpentSoFar != 0 || travel.YearToDateAverage != 1_000_000 || travel.Pattern != SpendPatternOccasional {
		t.Fatalf("travel %+v", travel)
	}
}

// natured tags every transaction with the category nature.
func natured(nature CategoryNature, transactions []Transaction) []Transaction {
	for i := range transactions {
		transactions[i].CategoryNature = nature
	}
	return transactions
}

// salary records the same income in each of the last three full months before October.
func salary(amount float64) []Transaction {
	return []Transaction{
		txn(day(2026, 7, 1), "Salary", CategoryTypeIncome, 0, amount),
		txn(day(2026, 8, 1), "Salary", CategoryTypeIncome, 0, amount),
		txn(day(2026, 9, 1), "Salary", CategoryTypeIncome, 0, amount),
	}
}

func TestRecommendationsProtectNeeds(t *testing.T) {
	today := day(2026, 10, 19)
	tests := []struct {
		name      string
		nature    CategoryNature
		limit     float64
		amounts   []float64
		want      float64
		direction LimitDirection
		reason    string
	}{
		{
			name:      "an essential that costs more than its limit is raised to its usual cost, the higher average",
			nature:    NatureEssential,
			limit:     2_000_000,
			amounts:   []float64{3_000_000, 3_000_000, 3_000_000, 3_000_000, 3_000_000, 3_000_000, 2_500_000, 2_500_000, 2_500_000},
			want:      2_840_000,
			direction: LimitIncrease,
			reason:    "needs are budgeted at their usual cost",
		},
		{
			name:      "an essential is never cut below its usual cost even when recent months were lower",
			nature:    NatureEssential,
			limit:     2_900_000,
			amounts:   []float64{3_000_000, 3_000_000, 3_000_000, 3_000_000, 3_000_000, 3_000_000, 2_000_000, 2_000_000, 2_000_000},
			want:      2_900_000,
			direction: LimitKeep,
			reason:    "fits what you usually spend",
		},
		{
			name:      "an essential limit far above its usual cost is lowered to the usual cost plus a cushion",
			nature:    NatureEssential,
			limit:     5_000_000,
			amounts:   []float64{2_000_000, 2_000_000, 2_000_000, 2_000_000, 2_000_000, 2_000_000, 2_000_000, 2_000_000, 2_000_000},
			want:      2_200_000,
			direction: LimitLower,
			reason:    "slack can go to goals",
		},
		{
			name:      "a fixed cost follows what it costs now",
			nature:    NatureFixed,
			limit:     5_000_000,
			amounts:   []float64{5_000_000, 5_000_000, 5_000_000, 5_000_000, 5_000_000, 5_000_000, 6_000_000, 6_000_000, 6_000_000},
			want:      6_000_000,
			direction: LimitIncrease,
			reason:    "fixed cost above its limit",
		},
		{
			name:      "a fixed cost that came down lets the limit follow",
			nature:    NatureFixed,
			limit:     6_000_000,
			amounts:   []float64{6_000_000, 6_000_000, 6_000_000, 6_000_000, 6_000_000, 6_000_000, 4_000_000, 4_000_000, 4_000_000},
			want:      4_400_000,
			direction: LimitLower,
			reason:    "fixed cost came down",
		},
		{
			name:      "an untagged category asks to be tagged",
			limit:     1_000_000,
			amounts:   []float64{1_000_000, 1_000_000, 1_000_000, 1_000_000, 1_000_000, 1_000_000, 1_000_000, 1_000_000, 1_000_000},
			want:      1_000_000,
			direction: LimitKeep,
			reason:    "Tag it fixed, essential or flexible",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			facts := ComputeAdvisorFacts([]BudgetItem{expenseItem("Need", test.limit)}, nil, natured(test.nature, spendEachMonth(2026, "Need", test.amounts...)), today, 0, nil)
			need := statsFor(t, facts, "Need")
			if need.RecommendedLimit != test.want || need.Direction != test.direction || !strings.Contains(need.Reason, test.reason) {
				t.Fatalf("recommended %v %s %q, want %v %s containing %q", need.RecommendedLimit, need.Direction, need.Reason, test.want, test.direction, test.reason)
			}
			if need.Nature != test.nature {
				t.Fatalf("nature %q, want %q", need.Nature, test.nature)
			}
		})
	}
}

func TestRecommendationsFundWantsWhileSavingsAreOnTrack(t *testing.T) {
	today := day(2026, 10, 19)
	// Dining rose from 1,000,000 to 2,000,000 a month; the limit is still 1,000,000
	dining := natured(NatureFlexible, spendEachMonth(2026, "Dining", 1_000_000, 1_000_000, 1_000_000, 1_000_000, 1_000_000, 1_000_000, 2_000_000, 2_000_000, 2_000_000))
	rent := natured(NatureFixed, spendEachMonth(2026, "Rent", 5_000_000, 5_000_000, 5_000_000, 5_000_000, 5_000_000, 5_000_000, 5_000_000, 5_000_000, 5_000_000))
	items := []BudgetItem{expenseItem("Dining", 1_000_000), expenseItem("Rent", 5_000_000)}

	t.Run("on track: the want keeps its usual level", func(t *testing.T) {
		// Given 20,000,000 income: 30% is 6,000,000 and the usual 7,000,000 of limits leaves 13,000,000
		transactions := append(append(append([]Transaction{}, dining...), rent...), salary(20_000_000)...)

		facts := ComputeAdvisorFacts(items, nil, transactions, today, 4_000_000, nil)

		check := facts.SavingsCheck
		if !check.OnTrack || check.Required != 6_000_000 || check.UsualSpend != 7_000_000 || check.GoalNeed != 4_000_000 || check.AverageIncome != 20_000_000 {
			t.Fatalf("savings check %+v", check)
		}
		food := statsFor(t, facts, "Dining")
		if food.RecommendedLimit != 2_000_000 || food.Direction != LimitIncrease || !strings.Contains(food.Reason, "on track") {
			t.Fatalf("dining %+v", food)
		}
		if check.RecommendedSpend != 7_000_000 {
			t.Fatalf("recommended spend %v", check.RecommendedSpend)
		}
	})

	t.Run("short: a rising want falls back to its lower average", func(t *testing.T) {
		// Given 20,000,000 income but goals that need 15,000,000 a month
		transactions := append(append(append([]Transaction{}, dining...), rent...), salary(20_000_000)...)

		facts := ComputeAdvisorFacts(items, nil, transactions, today, 15_000_000, nil)

		if facts.SavingsCheck.OnTrack || facts.SavingsCheck.Required != 15_000_000 {
			t.Fatalf("savings check %+v", facts.SavingsCheck)
		}
		// Year-to-date average is 1,333,333; the limit 1,000,000 is below it, so it rises only that far
		food := statsFor(t, facts, "Dining")
		if food.RecommendedLimit != 1_340_000 || food.Direction != LimitIncrease {
			t.Fatalf("dining %+v", food)
		}
		// And needs are untouched by the savings shortfall
		if rent := statsFor(t, facts, "Rent"); rent.Direction != LimitKeep || rent.RecommendedLimit != 5_000_000 {
			t.Fatalf("rent %+v", rent)
		}
	})

	t.Run("short: a want limit above its lower average is lowered without a cushion", func(t *testing.T) {
		transactions := append(append(append([]Transaction{}, dining...), rent...), salary(20_000_000)...)

		facts := ComputeAdvisorFacts([]BudgetItem{expenseItem("Dining", 2_000_000), expenseItem("Rent", 5_000_000)}, nil, transactions, today, 15_000_000, nil)

		food := statsFor(t, facts, "Dining")
		if food.RecommendedLimit != 1_340_000 || food.Direction != LimitLower || !strings.Contains(food.Reason, "rising") {
			t.Fatalf("dining %+v", food)
		}
	})

	t.Run("no income recorded: savings are not on track", func(t *testing.T) {
		facts := ComputeAdvisorFacts(items, nil, append(append([]Transaction{}, dining...), rent...), today, 0, nil)

		if facts.SavingsCheck.OnTrack {
			t.Fatalf("savings check %+v", facts.SavingsCheck)
		}
	})
}

func TestRecommendationsKeepLimitsWithoutHistory(t *testing.T) {
	// Given an annual fee budgeted monthly but not paid in any full month this year
	facts := ComputeAdvisorFacts([]BudgetItem{expenseItem("Insurance", 500_000)}, nil, nil, day(2026, 10, 19), 0, nil)

	insurance := statsFor(t, facts, "Insurance")
	if insurance.Direction != LimitKeep || insurance.RecommendedLimit != 500_000 || !strings.Contains(insurance.Reason, "still needed") {
		t.Fatalf("insurance %+v", insurance)
	}
}

func TestApplyFactsFillsMissingRecommendations(t *testing.T) {
	facts := AdvisorFacts{Month: "2026-10", SavingsCheck: SavingsCheck{OnTrack: true, Required: 6_000_000}, Categories: []BudgetCategoryStats{
		{Category: "Food", Nature: NatureEssential, CurrentLimit: 2_000_000, AverageSpend: 3_000_000, YearToDateAverage: 2_330_000, Trend: SpendTrendRising, Pattern: SpendPatternSteady, RecommendedLimit: 3_000_000, Direction: LimitIncrease, Reason: "A need that costs more than its limit."},
		{Category: "Coffee", Nature: NatureFlexible, CurrentLimit: 500_000, AverageSpend: 480_000, RecommendedLimit: 500_000, Direction: LimitKeep, Reason: "Fits."},
		{Category: "Gifts", Nature: NatureFlexible, CurrentLimit: 0, YearToDateAverage: 1_000_000, Pattern: SpendPatternOccasional, RecommendedLimit: 1_000_000, Direction: LimitIncrease, Reason: "Spent in only some months."},
		{Category: "Shoes", Nature: NatureFlexible, CurrentLimit: 1_000_000, AverageSpend: 2_000_000, Trend: SpendTrendRising, Pattern: SpendPatternVariable, RecommendedLimit: 2_000_000, Direction: LimitIncrease},
		{Category: "Gym", CurrentLimit: 2_000_000, AverageSpend: 500_000, RecommendedLimit: 550_000, Direction: LimitLower, Reason: "Slack."},
	}}
	// Given the model suggested Shoes only, below the current limit
	advice := BudgetAdvice{Briefing: "ok", SuggestedLimits: []SuggestedLimit{{Category: "shoes", SuggestedLimit: 800_000, Rationale: "Cut back"}}}

	if err := advice.ApplyFacts(facts); err != nil {
		t.Fatal(err)
	}

	// Then the model's suggestion stays first with the engine's figures and its own direction
	shoes := advice.SuggestedLimits[0]
	if shoes.Category != "Shoes" || shoes.SuggestedLimit != 800_000 || shoes.RecommendedLimit != 2_000_000 || shoes.Direction != LimitLower || shoes.Nature != NatureFlexible || shoes.Trend != SpendTrendRising {
		t.Fatalf("shoes %+v", shoes)
	}
	// And every other move the engine recommends follows, largest relative change first; keeps are left out
	if len(advice.SuggestedLimits) != 4 {
		t.Fatalf("suggestions %+v", advice.SuggestedLimits)
	}
	gifts, gym, food := advice.SuggestedLimits[1], advice.SuggestedLimits[2], advice.SuggestedLimits[3]
	if gifts.Category != "Gifts" || gifts.Direction != LimitIncrease || gifts.Rationale != "Spent in only some months." {
		t.Fatalf("gifts %+v", gifts)
	}
	if gym.Category != "Gym" || gym.SuggestedLimit != 550_000 || gym.Direction != LimitLower {
		t.Fatalf("gym %+v", gym)
	}
	if food.Category != "Food" || food.SuggestedLimit != 3_000_000 || food.YearToDateAverage != 2_330_000 || food.Direction != LimitIncrease {
		t.Fatalf("food %+v", food)
	}
	// And the savings check is the engine's
	if !advice.SavingsCheck.OnTrack || advice.SavingsCheck.Required != 6_000_000 {
		t.Fatalf("savings check %+v", advice.SavingsCheck)
	}
}

func TestAdvisorFactsCarryEachSplitTagsTrend(t *testing.T) {
	// Given groceries split by eating-out: meals out rose from 500,000 to 1,000,000 a month while home food held
	categories := []Category{{Name: "Groceries", Type: CategoryTypeExpense, Nature: NatureEssential, SplitTags: []string{"eating-out"}}}
	transactions := spendEachMonth(2026, "Groceries", 2_000_000, 2_000_000, 2_000_000, 2_000_000, 2_000_000, 2_000_000, 2_000_000, 2_000_000, 2_000_000)
	for i, amount := range []float64{500_000, 500_000, 500_000, 500_000, 500_000, 500_000, 1_000_000, 1_000_000, 1_000_000} {
		transactions = append(transactions, tagged(txn(day(2026, time.January+time.Month(i), 20), "Groceries", CategoryTypeExpense, amount, 0), "eating-out"))
	}
	transactions = append(transactions, tagged(txn(day(2026, 10, 3), "Groceries", CategoryTypeExpense, 400_000, 0), "eating-out"))

	facts := ComputeAdvisorFacts([]BudgetItem{expenseItem("Groceries", 3_000_000)}, nil, natured(NatureEssential, transactions), day(2026, 10, 19), 0, categories)

	groceries := statsFor(t, facts, "Groceries")
	if len(groceries.Splits) != 1 {
		t.Fatalf("splits %+v", groceries.Splits)
	}
	split := groceries.Splits[0]
	if split != (SplitStats{Tag: "eating-out", SpentSoFar: 400_000, AverageSpend: 1_000_000, YearToDateAverage: 666_667, Trend: SpendTrendRising}) {
		t.Fatalf("split %+v", split)
	}
}
