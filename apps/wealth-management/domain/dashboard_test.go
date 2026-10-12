package domain

import (
	"testing"
	"time"
)

func TestComputeDashboardSummary(t *testing.T) {
	today := time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)
	income, expense := CategoryTypeIncome, CategoryTypeExpense
	amount := func(v float64) *float64 { return &v }
	accounts := []Account{
		{Name: "Bank", Balance: 50_000_000, Type: AccountTypeActiveUse},
		{Name: "Card", Balance: -2_000_000, Type: AccountTypeNegativeActive},
		{Name: "Fund", Balance: 20_000_000, Type: AccountTypeLongHolding},
		{Name: "Cash", Balance: 1_000_000, Type: AccountTypeCash},
		{Name: "Wallet", Balance: 500_000, Type: AccountTypeCash},
		{Name: "Old", Balance: 100_000, Type: AccountTypeDeprecated},
	}
	loans := []Loan{{Name: "Friend", YearlyRemaining: 3_000_000}}
	txs := []Transaction{
		{AccountName: "Bank", Date: time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC), Category: "Salary", CategoryType: &income, Deposit: amount(30_000_000)},
		{AccountName: "Card", Date: time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC), Category: "Food", CategoryType: &expense, Payment: amount(6_000_000)},
		{AccountName: "Bank", Date: time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC), Category: "Salary", CategoryType: &income, Deposit: amount(25_000_000)},
		{AccountName: "Bank", Date: time.Date(2026, 9, 9, 0, 0, 0, 0, time.UTC), Category: "Food", CategoryType: &expense, Payment: amount(5_000_000)},
	}

	summary := ComputeDashboardSummary(accounts, loans, txs, today, 12)

	if summary.BalanceSheet.NetWorth != 66_600_000 {
		t.Fatalf("net worth = %v", summary.BalanceSheet.NetWorth)
	}
	if summary.NetGrowthThisMonth != 24_000_000 {
		t.Fatalf("net growth = %v", summary.NetGrowthThisMonth)
	}
	// Month start net worth = 66.6M - 24M = 42.6M; growth 24M / 42.6M.
	if summary.NetWorthChangePercent == nil || *summary.NetWorthChangePercent != 56.34 {
		t.Fatalf("net worth change = %v", summary.NetWorthChangePercent)
	}
	if summary.ThisMonth.Income != 30_000_000 || summary.ThisMonth.Expense != 6_000_000 {
		t.Fatalf("this month = %+v", summary.ThisMonth)
	}
	if summary.LastMonth.Month != "2026-09" || summary.LastMonth.Income != 25_000_000 {
		t.Fatalf("last month = %+v", summary.LastMonth)
	}
	if len(summary.NetWorthTrend) != 30 || summary.NetWorthTrend[29].Value != 66_600_000 {
		t.Fatalf("trend end = %+v", summary.NetWorthTrend[len(summary.NetWorthTrend)-1])
	}
	if len(summary.CashFlowHistory) != 12 || summary.CashFlowHistory[11].Month != "2026-10" {
		t.Fatalf("history = %d months, last %+v", len(summary.CashFlowHistory), summary.CashFlowHistory[len(summary.CashFlowHistory)-1])
	}
	if len(summary.TopAccounts) != 5 || summary.TopAccounts[0].Name != "Bank" || summary.TopAccounts[2].Name != "Card" {
		t.Fatalf("top accounts = %+v", summary.TopAccounts)
	}
	for _, account := range summary.TopAccounts {
		if account.Type == AccountTypeDeprecated {
			t.Fatalf("deprecated account in top list")
		}
	}
}
