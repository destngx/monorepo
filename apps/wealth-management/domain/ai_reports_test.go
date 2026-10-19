package domain

import (
	"testing"
)

func TestComputeAdvisorFacts(t *testing.T) {
	today := day(2026, 10, 10)
	items := []BudgetItem{{Category: "Food", CategoryType: categoryType(CategoryTypeExpense), MonthlyLimit: 3_000_000}}
	transactions := []Transaction{
		// 3 full months at 3,000,000 food and 10,000,000 salary.
		txn(day(2026, 7, 5), "Food", CategoryTypeExpense, 3_000_000, 0),
		txn(day(2026, 8, 5), "Food", CategoryTypeExpense, 3_000_000, 0),
		txn(day(2026, 9, 5), "Food", CategoryTypeExpense, 3_000_000, 0),
		txn(day(2026, 7, 1), "Salary", CategoryTypeIncome, 0, 10_000_000),
		txn(day(2026, 8, 1), "Salary", CategoryTypeIncome, 0, 10_000_000),
		txn(day(2026, 9, 1), "Salary", CategoryTypeIncome, 0, 10_000_000),
		// 10 days in, already 2,000,000: about twice the usual pace.
		txn(day(2026, 10, 3), "Food", CategoryTypeExpense, 2_000_000, 0),
	}
	accounts := []Account{{Name: "Bank", Type: AccountTypeBank, Balance: 50_000_000}}

	facts := ComputeAdvisorFacts(items, accounts, transactions, today, 0, nil)

	if facts.Month != "2026-10" || len(facts.Categories) != 1 {
		t.Fatalf("unexpected facts %+v", facts)
	}
	food := facts.Categories[0]
	if food.AverageSpend != 3_000_000 || food.SpentSoFar != 2_000_000 || food.CurrentLimit != 3_000_000 {
		t.Fatalf("unexpected food stats %+v", food)
	}
	if len(facts.UnusualVelocity) != 1 || facts.UnusualVelocity[0].Ratio < 2 {
		t.Fatalf("expected a velocity flag, got %+v", facts.UnusualVelocity)
	}
	if facts.AverageIncome != 10_000_000 || facts.ForecastedSavings != 10_000_000-6_200_000 {
		t.Fatalf("unexpected savings forecast %v / %v", facts.AverageIncome, facts.ForecastedSavings)
	}
	if len(facts.Projection) != 30 || facts.Projection[0].Date != "2026-10-11" {
		t.Fatalf("unexpected projection start %+v", facts.Projection[0])
	}
	// Average cash flow is +7,000,000/month, so the balance rises.
	if last := facts.Projection[29].Balance; last <= 50_000_000 {
		t.Fatalf("projection should rise, got %v", last)
	}
}

func TestComputeBriefingFacts(t *testing.T) {
	today := day(2026, 10, 10)
	accounts := []Account{{Name: "Bank", Balance: 10_000_000}, {Name: "Card", Balance: -2_000_000}}
	transactions := []Transaction{
		lendingTxn("row-9", "2026-09-01", "Person A", 3_000_000, 0),
		txn(day(2026, 10, 1), "Salary", CategoryTypeIncome, 0, 8_000_000),
		txn(day(2026, 10, 2), "Food", CategoryTypeExpense, 2_000_000, 0),
	}
	facts := ComputeBriefingFacts(accounts, transactions, today)
	if facts.NetWorth != 11_000_000 || facts.CashFlow != 6_000_000 || facts.SavingsRate != 75 {
		t.Fatalf("unexpected facts %+v", facts)
	}
	if facts.MoneyLent.OwedToYou != 3_000_000 || facts.MoneyLent.PeopleOwing != 1 || *facts.MoneyLent.OldestDaysOutstanding != 39 {
		t.Fatalf("money lent facts %+v", facts.MoneyLent)
	}
}
