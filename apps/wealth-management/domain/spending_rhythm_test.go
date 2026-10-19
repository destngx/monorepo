package domain

import (
	"testing"
	"time"
)

func withNature(t Transaction, nature CategoryNature) Transaction {
	t.CategoryNature = nature
	return t
}

// steadySpend books amount in category every day of the 180 days ending today.
func steadyDailySpend(today time.Time, category string, nature CategoryNature, amount float64) []Transaction {
	rows := []Transaction{}
	for i := 0; i < 2*SpendingBaselineDays; i++ {
		rows = append(rows, withNature(txn(today.AddDate(0, 0, -i), category, CategoryTypeExpense, amount, 0), nature))
	}
	return rows
}

func findSpendingAdvice(recommendations []SpendingRecommendation, kind SpendingRecommendationKind, category, tag string) *SpendingRecommendation {
	for i := range recommendations {
		r := recommendations[i]
		if r.Kind == kind && r.Category == category && r.Tag == tag {
			return &recommendations[i]
		}
	}
	return nil
}

func TestGivenALedgerWhenComputingSpendingWindowsThenEachWindowSplitsFixedFromDayToDay(t *testing.T) {
	// Given rent paid once in each window, 100k of groceries a day, and rows the windows must leave out
	today := day(2026, 10, 19)
	transactions := steadyDailySpend(today, "Groceries", NatureEssential, 100_000)
	transactions = append(transactions,
		withNature(txn(day(2026, 10, 1), "Rent", CategoryTypeExpense, 5_000_000, 0), NatureFixed),
		withNature(txn(day(2026, 8, 1), "Rent", CategoryTypeExpense, 5_000_000, 0), NatureFixed),
		txn(day(2026, 10, 20), "Groceries", CategoryTypeExpense, 9_000_000, 0),                       // after today
		withNature(txn(day(2026, 10, 5), "Loans", CategoryTypeExpense, 3_000_000, 0), NatureLending), // lending
		txn(day(2026, 10, 6), "Salary", CategoryTypeIncome, 0, 20_000_000),                           // income
		txn(day(2026, 10, 7), "Groceries", CategoryTypeExpense, 0, 50_000),                           // a refund in the category
	)

	// When
	got := ComputeSpendingWindows(transactions, nil, today)

	// Then the last 30 days run from 2026-09-20 and the 3 months from 2026-07-22
	recent, baseline := got.Windows[0], got.Windows[1]
	if recent.Days != 30 || recent.From != "2026-09-20" || recent.To != "2026-10-19" || baseline.From != "2026-07-22" {
		t.Fatalf("windows = %+v / %+v", recent, baseline)
	}
	if recent.Fixed != 5_000_000 || recent.DayToDay != 2_950_000 || recent.Spent != 7_950_000 {
		t.Fatalf("recent = fixed %v, day-to-day %v, spent %v", recent.Fixed, recent.DayToDay, recent.Spent)
	}
	if recent.DailyAverage != 98_333.33 || recent.MonthlyAverage != 8_056_000 {
		t.Fatalf("recent averages = %v / %v", recent.DailyAverage, recent.MonthlyAverage)
	}
	if baseline.Fixed != 10_000_000 || baseline.DayToDay != 8_950_000 {
		t.Fatalf("baseline = %+v", baseline)
	}
	if recent.Previous.From != "2026-08-21" || recent.Previous.To != "2026-09-19" || recent.Previous.DayToDay != 3_000_000 {
		t.Fatalf("previous = %+v", recent.Previous)
	}
	if len(recent.Categories) != 2 || recent.Categories[0].Category != "Rent" || recent.Categories[1].Nature != NatureEssential {
		t.Fatalf("categories = %+v", recent.Categories)
	}
	// And today is a typical day; the daily series covers 90 days
	if got.Today.DayToDay != 100_000 || got.Today.TypicalDay != 98_333.33 || len(got.Daily) != SpendingBaselineDays {
		t.Fatalf("today = %+v, daily %d", got.Today, len(got.Daily))
	}
	if last := got.Daily[len(got.Daily)-1]; last.Date != "2026-10-19" || last.DayToDay != 100_000 {
		t.Fatalf("last day = %+v", last)
	}
	// And a steady month gives no advice
	if len(got.Recommendations) != 0 {
		t.Fatalf("recommendations = %+v", got.Recommendations)
	}
}

func TestGivenSplitTagsWhenComputingSpendingWindowsThenCategoriesCarryTheirSplitShare(t *testing.T) {
	today := day(2026, 10, 19)
	categories := []Category{{Name: "Food & Groceries", Type: CategoryTypeExpense, SplitTags: []string{"eating-out"}}}
	transactions := []Transaction{
		txn(day(2026, 10, 10), "Food & Groceries", CategoryTypeExpense, 600_000, 0),
		tagged(txn(day(2026, 10, 11), "food & groceries", CategoryTypeExpense, 400_000, 0), "Eating-Out", "card-cashback"),
		tagged(txn(day(2026, 8, 1), "Food & Groceries", CategoryTypeExpense, 200_000, 0), "eating-out"),
	}

	got := ComputeSpendingWindows(transactions, categories, today)

	recent, baseline := got.Windows[0].Categories[0], got.Windows[1].Categories[0]
	if len(recent.Splits) != 1 || recent.Splits[0] != (SplitSpend{Tag: "eating-out", Spent: 400_000, PercentOfSpent: 40}) {
		t.Fatalf("recent splits = %+v", recent.Splits)
	}
	if baseline.Splits[0].Spent != 600_000 || baseline.Spent != 1_200_000 {
		t.Fatalf("baseline = %+v", baseline)
	}
}

func TestGivenFasterDayToDaySpendingWhenComputingSpendingWindowsThenItRecommendsReiningInRisingWants(t *testing.T) {
	// Given steady groceries, and shopping (a want) and eating out that doubled over the last 30 days
	today := day(2026, 10, 19)
	categories := []Category{{Name: "Food", Type: CategoryTypeExpense, SplitTags: []string{"eating-out"}}}
	transactions := steadyDailySpend(today, "Food", NatureEssential, 100_000)
	for i := 0; i < SpendingBaselineDays; i++ {
		amount := 50_000.0
		if i < SpendingRecentDays {
			amount = 150_000
		}
		transactions = append(transactions,
			withNature(txn(today.AddDate(0, 0, -i), "Shopping", CategoryTypeExpense, amount, 0), NatureFlexible),
			withNature(tagged(txn(today.AddDate(0, 0, -i), "Food", CategoryTypeExpense, amount/2, 0), "eating-out"), NatureEssential))
	}
	// And a monthly rent that is never advice
	transactions = append(transactions, withNature(txn(day(2026, 10, 1), "Rent", CategoryTypeExpense, 50_000_000, 0), NatureFixed))

	got := ComputeSpendingWindows(transactions, categories, today)

	faster := findSpendingAdvice(got.Recommendations, SpendingFaster, "", "")
	if faster == nil || faster.Tone != "warning" || faster.Current != 325_000 || faster.Usual <= 200_000 || faster.Amount <= 0 {
		t.Fatalf("faster = %+v in %+v", faster, got.Recommendations)
	}
	shopping := findSpendingAdvice(got.Recommendations, SpendingCategoryRising, "Shopping", "")
	if shopping == nil || shopping.Tone != "warning" || shopping.Current != 4_500_000 || shopping.Usual != 2_500_000 || shopping.Nature != NatureFlexible {
		t.Fatalf("shopping = %+v", shopping)
	}
	eatingOut := findSpendingAdvice(got.Recommendations, SpendingSplitRising, "Food", "eating-out")
	if eatingOut == nil || eatingOut.Tone != "warning" || eatingOut.Current != 2_250_000 {
		t.Fatalf("eating out = %+v", eatingOut)
	}
	// And the need rose only by its eating out, so it is looked at, not cut
	if food := findSpendingAdvice(got.Recommendations, SpendingCategoryRising, "Food", ""); food != nil && food.Tone != "info" {
		t.Fatalf("food = %+v", food)
	}
	if findSpendingAdvice(got.Recommendations, SpendingCategoryRising, "Rent", "") != nil {
		t.Fatal("rent was flagged")
	}
}

func TestGivenSlowerSpendingAndAHeavyDayWhenComputingSpendingWindowsThenItSaysSoAndSuggestsGoals(t *testing.T) {
	today := day(2026, 10, 19)
	transactions := []Transaction{}
	for i := 1; i < SpendingBaselineDays; i++ {
		amount := 200_000.0
		if i < SpendingRecentDays {
			amount = 100_000
		}
		transactions = append(transactions, txn(today.AddDate(0, 0, -i), "Food", CategoryTypeExpense, amount, 0))
	}
	transactions = append(transactions, txn(today, "Food", CategoryTypeExpense, 900_000, 0))

	got := ComputeSpendingWindows(transactions, nil, today)

	if slower := findSpendingAdvice(got.Recommendations, SpendingSlower, "", ""); slower == nil || slower.Tone != "good" || slower.Amount <= 0 {
		t.Fatalf("slower = %+v in %+v", slower, got.Recommendations)
	}
	heavy := findSpendingAdvice(got.Recommendations, SpendingHeavyDay, "", "")
	if heavy == nil || heavy.Current != 900_000 || heavy.Usual != got.Today.TypicalDay || *got.Today.ChangePercent <= 100 {
		t.Fatalf("heavy = %+v, today %+v", heavy, got.Today)
	}
}

func TestGivenNoLedgerWhenComputingSpendingWindowsThenEverythingIsZeroWithoutAdvice(t *testing.T) {
	got := ComputeSpendingWindows(nil, nil, day(2026, 10, 19))
	if got.Windows[0].Spent != 0 || got.Today.ChangePercent != nil || got.Windows[0].DailyAverageChangePercent != nil ||
		len(got.Recommendations) != 0 || len(got.Windows[1].Categories) != 0 {
		t.Fatalf("got = %+v", got)
	}
}
