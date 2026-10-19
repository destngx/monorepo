package domain

import (
	"strings"
	"testing"
)

// setAsideInput is billsInput (17 October, salary on the 25th, 31M ready,
// 500k of bills due) with tagged categories, a budget and some spending:
// Rent (fixed, 8M budget, paid), Utilities (fixed, 1M budget, 400k spent),
// Groceries (essential, 3M spent in September, the only completed month).
func setAsideInput() BillsInput {
	input := billsInput()
	input.Categories = []Category{
		{Name: "Rent", Type: CategoryTypeExpense, Nature: NatureFixed},
		{Name: "Utilities", Type: CategoryTypeExpense, Nature: NatureFixed},
		{Name: "Groceries", Type: CategoryTypeExpense, Nature: NatureEssential},
		{Name: "Eating out", Type: CategoryTypeExpense, Nature: NatureFlexible},
		{Name: "Salary", Type: CategoryTypeIncome},
	}
	input.Budgets = map[int][]BudgetItem{2026: {
		{Category: "Rent", MonthlyLimit: 8_000_000},
		{Category: "Utilities", MonthlyLimit: 1_000_000},
		{Category: "Groceries", MonthlyLimit: 5_000_000},
	}}
	input.Transactions = append(input.Transactions,
		flow(day(2026, 9, 10), "Market", "Groceries", CategoryTypeExpense, 3_000_000),
		flow(day(2026, 10, 5), "Power Co", "Utilities", CategoryTypeExpense, 400_000),
		flow(day(2026, 10, 3), "Cafe", "Eating out", CategoryTypeExpense, 2_000_000),
	)
	return input
}

func fixedAt(t *testing.T, safe SafeToSpend, category, month string) FixedCost {
	t.Helper()
	for _, line := range safe.Fixed {
		if line.Category == category && line.Month == month {
			return line
		}
	}
	t.Fatalf("no fixed %s for %s in %+v", category, month, safe.Fixed)
	return FixedCost{}
}

func essentialFor(t *testing.T, safe SafeToSpend, category string) EssentialNeed {
	t.Helper()
	for _, line := range safe.Essential {
		if line.Category == category {
			return line
		}
	}
	t.Fatalf("no essential %s in %+v", category, safe.Essential)
	return EssentialNeed{}
}

func TestGivenFixedAndEssentialCategoriesWhenComputingSafeToSpendThenTheyAreSetAsideBeforeFreeMoney(t *testing.T) {
	safe := ComputeBills(setAsideInput()).SafeToSpend
	if !safe.NatureTagged {
		t.Fatalf("natures are tagged: %+v", safe)
	}
	// Rent is paid in full; utilities have 600k of their budget still to pay.
	if rent := fixedAt(t, safe, "Rent", "2026-10"); rent.Expected != 8_000_000 || rent.Basis != BasisBudget || rent.Spent != 8_000_000 || rent.Amount != 0 {
		t.Fatalf("rent = %+v", rent)
	}
	if utilities := fixedAt(t, safe, "Utilities", "2026-10"); utilities.Spent != 400_000 || utilities.InBills != 0 || utilities.Amount != 600_000 {
		t.Fatalf("utilities = %+v", utilities)
	}
	// Groceries: 3M over September's 30 days, for the 8 days to payday.
	groceries := essentialFor(t, safe, "Groceries")
	if groceries.PerDay != 100_000 || groceries.Days != 8 || groceries.Basis != BasisAverage || groceries.Amount != 800_000 {
		t.Fatalf("groceries = %+v", groceries)
	}
	if len(safe.Essential) != 1 {
		t.Fatalf("flexible and untagged categories are never set aside: %+v", safe.Essential)
	}
	// 31M ready - 500k bills - 600k fixed = 29.9M left; - 800k essentials = 29.1M free.
	if safe.BillsDue != 500_000 || safe.FixedCosts != 600_000 || safe.Committed != 1_100_000 || safe.Amount != 29_900_000 {
		t.Fatalf("committed = %+v", safe)
	}
	if safe.Essentials != 800_000 || safe.SafeToSpend != 29_100_000 || safe.ShortBy != 0 || safe.TightBy != 0 || safe.PerDay != 3_637_500 {
		t.Fatalf("free = %+v", safe)
	}
	if !strings.Contains(safe.Explanation, "fixed costs") || !strings.Contains(safe.Explanation, "essentials") {
		t.Fatalf("explanation = %q", safe.Explanation)
	}
	if safe.SetAsideRule != SafeToSpendSetAsideRule {
		t.Fatalf("rule = %q", safe.SetAsideRule)
	}
}

func TestGivenARecurringBillInAFixedCategoryWhenComputingSafeToSpendThenItIsNotSetAsideTwice(t *testing.T) {
	input := setAsideInput()
	// Internet has no category; its payments say it is Utilities.
	input.Transactions = append(input.Transactions, flow(day(2026, 9, 10), "Net Co", "Utilities", CategoryTypeExpense, 300_000))
	safe := ComputeBills(input).SafeToSpend
	// The overdue 300k internet bill is in the bills already: 1M - 400k - 300k.
	if utilities := fixedAt(t, safe, "Utilities", "2026-10"); utilities.InBills != 300_000 || utilities.Amount != 300_000 {
		t.Fatalf("utilities = %+v", utilities)
	}
	if safe.BillsDue != 500_000 || safe.FixedCosts != 300_000 {
		t.Fatalf("safe = %+v", safe)
	}
}

func TestGivenAWindowIntoNextMonthWhenComputingSafeToSpendThenNextMonthsFixedCostsCountToo(t *testing.T) {
	input := setAsideInput()
	input.Sheet.Items = input.Sheet.Items[1:] // no salary: the next 30 days
	safe := ComputeBills(input).SafeToSpend
	if safe.UntilIncome || safe.Until != "2026-11-16" {
		t.Fatalf("horizon = %+v", safe)
	}
	// November's rent is a bill due in the window, so the budget holds nothing more.
	if rent := fixedAt(t, safe, "Rent", "2026-11"); rent.InBills != 8_000_000 || rent.Amount != 0 {
		t.Fatalf("rent = %+v", rent)
	}
	if utilities := fixedAt(t, safe, "Utilities", "2026-11"); utilities.Spent != 0 || utilities.Amount != 1_000_000 {
		t.Fatalf("utilities = %+v", utilities)
	}
	if safe.FixedCosts != 1_600_000 {
		t.Fatalf("fixed = %+v", safe.Fixed)
	}
}

func TestGivenAFixedCategoryWithoutABudgetWhenComputingSafeToSpendThenItsAverageIsExpected(t *testing.T) {
	input := setAsideInput()
	input.Budgets = map[int][]BudgetItem{2026: {{Category: "Rent", MonthlyLimit: 8_000_000}}}
	input.Transactions = append(input.Transactions, flow(day(2026, 9, 5), "Power Co", "Utilities", CategoryTypeExpense, 900_000))
	safe := ComputeBills(input).SafeToSpend
	if utilities := fixedAt(t, safe, "Utilities", "2026-10"); utilities.Expected != 900_000 || utilities.Basis != BasisAverage || utilities.Amount != 500_000 {
		t.Fatalf("utilities = %+v", utilities)
	}
}

func TestGivenAnEssentialCategoryWithoutHistoryWhenComputingSafeToSpendThenItsBudgetSetsThePace(t *testing.T) {
	input := setAsideInput()
	input.Transactions = input.Transactions[:2] // rent and September's salary only
	safe := ComputeBills(input).SafeToSpend
	// 5M over October's 31 days, for 8 days.
	groceries := essentialFor(t, safe, "Groceries")
	if groceries.Basis != BasisBudget || groceries.PerDay != 161_290 || groceries.Amount != 1_290_323 {
		t.Fatalf("groceries = %+v", groceries)
	}
}

func TestGivenRecurringPaymentsInAnEssentialCategoryWhenComputingSafeToSpendThenThePaceLeavesThemOut(t *testing.T) {
	input := setAsideInput()
	input.Categories[1].Nature = NatureEssential // Utilities
	input.Transactions = append(input.Transactions,
		flow(day(2026, 9, 10), "Net Co", "Utilities", CategoryTypeExpense, 300_000),
		flow(day(2026, 9, 12), "Water Co", "Utilities", CategoryTypeExpense, 600_000),
	)
	safe := ComputeBills(input).SafeToSpend
	// Internet is a bill already counted; only water sets the pace: 600k / 30 days.
	if utilities := essentialFor(t, safe, "Utilities"); utilities.PerDay != 20_000 || utilities.Amount != 160_000 {
		t.Fatalf("utilities = %+v", utilities)
	}
}

func TestGivenNoTaggedCategoriesWhenComputingSafeToSpendThenNothingIsSetAside(t *testing.T) {
	input := setAsideInput()
	input.Categories = nil
	safe := ComputeBills(input).SafeToSpend
	if safe.NatureTagged || safe.FixedCosts != 0 || safe.Essentials != 0 || len(safe.Fixed) != 0 || len(safe.Essential) != 0 {
		t.Fatalf("set aside = %+v", safe)
	}
	if safe.SafeToSpend != 30_500_000 || safe.Committed != 500_000 {
		t.Fatalf("safe = %+v", safe)
	}
}

func TestGivenCommittedCoveredButNotEssentialsWhenComputingSafeToSpendThenItIsTight(t *testing.T) {
	input := setAsideInput()
	input.Accounts = []Account{{Name: "Wallet", Type: AccountTypeCash, Balance: 1_500_000}}
	safe := ComputeBills(input).SafeToSpend
	// 1.5M - 1.1M committed = 400k for 800k of essentials.
	if safe.Amount != 400_000 || safe.ShortBy != 0 || safe.SafeToSpend != 0 || safe.TightBy != 400_000 || safe.PerDay != 0 {
		t.Fatalf("safe = %+v", safe)
	}
	facts := ComputeBills(input).Facts()
	if facts.Committed != 1_100_000 || facts.FixedCosts != 600_000 || facts.Essentials != 800_000 || facts.TightBy != 400_000 {
		t.Fatalf("facts = %+v", facts)
	}
}

func TestGivenReadyMoneyShortOfFixedCostsWhenComputingSafeToSpendThenItIsShort(t *testing.T) {
	input := setAsideInput()
	input.Accounts = []Account{{Name: "Wallet", Type: AccountTypeCash, Balance: 1_000_000}}
	safe := ComputeBills(input).SafeToSpend
	if safe.ShortBy != 100_000 || safe.TightBy != 800_000 || safe.SafeToSpend != 0 {
		t.Fatalf("safe = %+v", safe)
	}
}
