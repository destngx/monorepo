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
	lent := Transaction{AccountName: "Bank", Date: time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC), Payee: "Person A", Category: "Lending / Repayment", CategoryType: &expense, CategoryNature: NatureLending, Payment: amount(3_000_000)}
	txs := []Transaction{
		lent,
		{AccountName: "Bank", Date: time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC), Category: "Salary", CategoryType: &income, Deposit: amount(30_000_000)},
		{AccountName: "Card", Date: time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC), Category: "Food", CategoryType: &expense, Payment: amount(6_000_000)},
		{AccountName: "Bank", Date: time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC), Category: "Salary", CategoryType: &income, Deposit: amount(25_000_000)},
		{AccountName: "Bank", Date: time.Date(2026, 9, 9, 0, 0, 0, 0, time.UTC), Category: "Food", CategoryType: &expense, Payment: amount(5_000_000)},
	}

	summary := ComputeDashboardSummary(accounts, txs, today, 12)

	// Accounts 69.6M plus 3M lent out: lending moves money, not net worth.
	if summary.BalanceSheet.NetWorth != 72_600_000 || summary.BalanceSheet.Receivables != 3_000_000 {
		t.Fatalf("net worth = %v", summary.BalanceSheet.NetWorth)
	}
	if summary.NetGrowthThisMonth != 24_000_000 {
		t.Fatalf("net growth = %v", summary.NetGrowthThisMonth)
	}
	// Month start net worth = 72.6M - 24M = 48.6M; growth 24M / 48.6M.
	if summary.NetWorthChangePercent == nil || *summary.NetWorthChangePercent != 49.38 {
		t.Fatalf("net worth change = %v", summary.NetWorthChangePercent)
	}
	if summary.ThisMonth.Income != 30_000_000 || summary.ThisMonth.Expense != 6_000_000 {
		t.Fatalf("this month = %+v", summary.ThisMonth)
	}
	if summary.LastMonth.Month != "2026-09" || summary.LastMonth.Income != 25_000_000 {
		t.Fatalf("last month = %+v", summary.LastMonth)
	}
	if len(summary.NetWorthTrend) != 30 || summary.NetWorthTrend[29].Value != 72_600_000 || summary.NetWorthTrend[0].Value != 72_600_000-24_000_000 {
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

func TestGivenMidMonthWhenComparingThenLastMonthIsCutToTheSameDays(t *testing.T) {
	today := day(2026, 10, 12)
	txs := []Transaction{
		txn(day(2026, 10, 5), "Salary", CategoryTypeIncome, 0, 20_000_000),
		txn(day(2026, 10, 8), "Food", CategoryTypeExpense, 2_000_000, 0),
		txn(day(2026, 10, 20), "Food", CategoryTypeExpense, 9_000_000, 0), // dated ahead: not yet
		txn(day(2026, 9, 5), "Salary", CategoryTypeIncome, 0, 20_000_000),
		txn(day(2026, 9, 10), "Food", CategoryTypeExpense, 1_000_000, 0),
		txn(day(2026, 9, 12), "Food", CategoryTypeExpense, 1_000_000, 0),
		txn(day(2026, 9, 13), "Food", CategoryTypeExpense, 6_000_000, 0), // after the same days
	}

	summary := ComputeDashboardSummary(nil, txs, today, 1)

	if summary.ThisMonth.Month != "2026-10" || summary.ThisMonth.Expense != 2_000_000 || summary.ThisMonth.Income != 20_000_000 {
		t.Fatalf("this month to date = %+v", summary.ThisMonth)
	}
	if summary.ComparedDays != 12 {
		t.Fatalf("compared days = %d", summary.ComparedDays)
	}
	if summary.LastMonthToDate.Month != "2026-09" || summary.LastMonthToDate.Expense != 2_000_000 || summary.LastMonthToDate.Income != 20_000_000 {
		t.Fatalf("last month to date = %+v", summary.LastMonthToDate)
	}
	if summary.LastMonth.Expense != 8_000_000 {
		t.Fatalf("full last month = %+v", summary.LastMonth)
	}
	if summary.ExpenseChangePercent == nil || *summary.ExpenseChangePercent != 0 {
		t.Fatalf("expense change = %v, want 0 against the same days", summary.ExpenseChangePercent)
	}
	if summary.IncomeChangePercent == nil || *summary.IncomeChangePercent != 0 {
		t.Fatalf("income change = %v", summary.IncomeChangePercent)
	}
}

func TestGivenADayLastMonthDidNotHaveWhenComparingThenTheWholeLastMonth(t *testing.T) {
	today := day(2026, 3, 31)
	txs := []Transaction{
		txn(day(2026, 3, 2), "Food", CategoryTypeExpense, 3_000_000, 0),
		txn(day(2026, 2, 28), "Food", CategoryTypeExpense, 2_000_000, 0),
	}

	summary := ComputeDashboardSummary(nil, txs, today, 1)

	if summary.ComparedDays != 28 || summary.LastMonthToDate.Expense != 2_000_000 {
		t.Fatalf("compared %d days, last month to date = %+v", summary.ComparedDays, summary.LastMonthToDate)
	}
	if summary.ExpenseChangePercent == nil || *summary.ExpenseChangePercent != 50 {
		t.Fatalf("expense change = %v", summary.ExpenseChangePercent)
	}
}

func TestGivenARefundForLastMonthWhenComparingThenItCountsOnTheFirstOfThatMonth(t *testing.T) {
	today := day(2026, 10, 3)
	refund := txn(day(2026, 10, 2), "Refunds/Reimbursements", CategoryTypeIncome, 0, 500_000)
	refund.Memo = text("09-2026")
	txs := []Transaction{refund, txn(day(2026, 9, 25), "Salary", CategoryTypeIncome, 0, 10_000_000)}

	summary := ComputeDashboardSummary(nil, txs, today, 1)

	if summary.ThisMonth.Income != 0 {
		t.Fatalf("this month = %+v, the refund belongs to September", summary.ThisMonth)
	}
	if summary.LastMonthToDate.Income != 500_000 || summary.LastMonth.Income != 10_500_000 {
		t.Fatalf("last month to date = %+v, full = %+v", summary.LastMonthToDate, summary.LastMonth)
	}
}

func paceCategories() []Category {
	return []Category{
		{Name: "Rent", Type: CategoryTypeExpense, Nature: NatureFixed},
		{Name: "Food", Type: CategoryTypeExpense, Nature: NatureEssential},
		{Name: "Fun", Type: CategoryTypeExpense, Nature: NatureFlexible},
	}
}

func TestGivenFixedBillsEarlyWhenProjectingExpenseThenOnlyTheRestIsExtrapolated(t *testing.T) {
	today := day(2026, 10, 10)
	txs := []Transaction{
		txn(day(2026, 10, 1), " rent ", CategoryTypeExpense, 10_000_000, 0),
		txn(day(2026, 10, 4), "Food", CategoryTypeExpense, 2_000_000, 0),
		txn(day(2026, 10, 9), "Gifts", CategoryTypeExpense, 1_000_000, 0), // untagged: extrapolated
		txn(day(2026, 10, 25), "Fun", CategoryTypeExpense, 5_000_000, 0),  // dated ahead
		txn(day(2026, 9, 1), "Rent", CategoryTypeExpense, 10_000_000, 0),
		txn(day(2026, 9, 20), "Food", CategoryTypeExpense, 9_000_000, 0),
	}

	pace := ComputeExpensePace(txs, paceCategories(), today)

	if pace.DaysElapsed != 10 || pace.DaysInMonth != 31 || pace.MonthEnd != "2026-10-31" || pace.MinDays != ExpensePaceMinDays {
		t.Fatalf("days = %+v", pace)
	}
	if pace.Spent != 13_000_000 || pace.Fixed != 10_000_000 || pace.Variable != 3_000_000 {
		t.Fatalf("split = %+v", pace)
	}
	// 10M fixed once + 3M over 10 days * 31 days.
	if pace.Projected == nil || *pace.Projected != 19_300_000 {
		t.Fatalf("projected = %v", pace.Projected)
	}
	if pace.LastMonthExpense != 19_000_000 || pace.LastMonth != "2026-09" {
		t.Fatalf("last month = %+v", pace)
	}
	if pace.ChangePercent == nil || *pace.ChangePercent != 1.58 {
		t.Fatalf("change = %v", pace.ChangePercent)
	}
}

func TestGivenTooFewDaysWhenProjectingExpenseThenNoPaceYet(t *testing.T) {
	today := day(2026, 10, ExpensePaceMinDays-1)
	txs := []Transaction{txn(day(2026, 10, 2), "Food", CategoryTypeExpense, 2_000_000, 0)}

	pace := ComputeExpensePace(txs, paceCategories(), today)

	if pace.Projected != nil || pace.ChangePercent != nil || pace.Spent != 2_000_000 {
		t.Fatalf("pace = %+v", pace)
	}
}

func TestGivenTheLastDayWhenProjectingExpenseThenThePaceIsWhatWasSpent(t *testing.T) {
	today := day(2026, 2, 28)
	txs := []Transaction{
		txn(day(2026, 2, 3), "Food", CategoryTypeExpense, 2_800_000, 0),
		txn(day(2026, 2, 1), "Rent", CategoryTypeExpense, 5_000_000, 0),
	}

	pace := ComputeExpensePace(txs, paceCategories(), today)

	if pace.Projected == nil || *pace.Projected != 7_800_000 || pace.ChangePercent != nil {
		t.Fatalf("pace = %+v", pace)
	}
}
