package domain

import (
	"errors"
	"testing"
	"time"
)

// snapshotLedger: a 60M bank account, a 20M fund account and a 5M card
// debt today (2026-11-05). October had 30M income and 18M expense; 1M was
// lent on 2026-10-20 (still owed). November so far: 2M expense on 11-03.
func snapshotLedger() ([]Account, []Transaction) {
	accounts := []Account{
		{Name: "Cash", Type: AccountTypeBank, Balance: 60_000_000},
		{Name: "Funds", Type: AccountTypeInvestment, Balance: 20_000_000},
		{Name: "Card", Type: AccountTypeNegativeActive, Balance: -5_000_000},
	}
	transactions := []Transaction{
		txn(day(2026, 10, 1), "Salary", CategoryTypeIncome, 0, 30_000_000),
		txn(day(2026, 10, 10), "Food", CategoryTypeExpense, 18_000_000, 0),
		lendingTxn("row-9", "2026-10-20", "Person A", 1_000_000, 0),
		txn(day(2026, 11, 3), "Food", CategoryTypeExpense, 2_000_000, 0),
	}
	return accounts, transactions
}

func TestGivenLastMonthWhenComputingItsSnapshotThenBalancesAreTakenBackToTheMonthEnd(t *testing.T) {
	accounts, transactions := snapshotLedger()

	snapshot := ComputeMonthSnapshot(SnapshotInputs{
		Accounts: accounts, Transactions: transactions, Month: day(2026, 10, 1), Today: day(2026, 11, 5),
	})

	// The 2M paid on 11-03 is undone: cash was 62M at the end of October.
	if snapshot.Month != "2026-10" || snapshot.AsOf != "2026-10-31" || snapshot.WrittenOn != "2026-11-05" {
		t.Fatalf("dates = %s %s %s", snapshot.Month, snapshot.AsOf, snapshot.WrittenOn)
	}
	if snapshot.NetWorth != 62_000_000+20_000_000-5_000_000+1_000_000 {
		t.Fatalf("net worth = %v", snapshot.NetWorth)
	}
	if *snapshot.Assets != 83_000_000 || *snapshot.Liabilities != 5_000_000 || *snapshot.MoneyLent != 1_000_000 {
		t.Fatalf("balance sheet = %v %v %v", *snapshot.Assets, *snapshot.Liabilities, *snapshot.MoneyLent)
	}
	if *snapshot.Cash != 62_000_000 || *snapshot.Funds != 20_000_000 || *snapshot.Crypto != 0 {
		t.Fatalf("classes = %v %v %v", *snapshot.Cash, *snapshot.Funds, *snapshot.Crypto)
	}
	if *snapshot.Income != 30_000_000 || *snapshot.Expense != 18_000_000 || *snapshot.Savings != 12_000_000 || *snapshot.SavingsRate != 40 {
		t.Fatalf("flow = %v %v %v %v", *snapshot.Income, *snapshot.Expense, *snapshot.Savings, *snapshot.SavingsRate)
	}
	// The score is the capped sum of all pillars (D44); four of them are kept.
	if snapshot.HealthScore == nil || *snapshot.HealthScore <= 0 || *snapshot.HealthScore > 100 {
		t.Fatalf("health %v", snapshot.HealthScore)
	}
	if snapshot.SavingsScore == nil || snapshot.EmergencyScore == nil || snapshot.DebtScore == nil || snapshot.DiversificationScore == nil {
		t.Fatal("pillar scores missing")
	}
	if snapshot.EmergencyMonths == nil {
		t.Fatal("emergency months missing")
	}
}

func TestGivenThisMonthWhenComputingItsSnapshotThenTheFiguresAreAsOfToday(t *testing.T) {
	accounts, transactions := snapshotLedger()

	snapshot := ComputeMonthSnapshot(SnapshotInputs{
		Accounts: accounts, Transactions: transactions, Funds: 25_000_000, Month: day(2026, 11, 1), Today: day(2026, 11, 5),
	})

	if snapshot.AsOf != "2026-11-05" || snapshot.NetWorth != 60_000_000+20_000_000-5_000_000+1_000_000 {
		t.Fatalf("snapshot = %s %v", snapshot.AsOf, snapshot.NetWorth)
	}
	if *snapshot.Expense != 2_000_000 || *snapshot.Income != 0 || *snapshot.Funds != 25_000_000 {
		t.Fatalf("month to date = %v %v, funds %v", *snapshot.Expense, *snapshot.Income, *snapshot.Funds)
	}
}

func TestGivenAMonthWhenCheckingItCanBeSnapshottedThenOnlyThisAndLastMonthAre(t *testing.T) {
	today := day(2026, 11, 20)
	for _, tc := range []struct {
		value string
		want  string
		ok    bool
	}{
		{"", "2026-11", true},
		{"2026-11", "2026-11", true},
		{"2026-10", "2026-10", true},
		{"2026-09", "", false},
		{"2026-12", "", false},
		{"11/2026", "", false},
	} {
		month, err := SnapshotMonth(tc.value, today)
		if tc.ok && (err != nil || MonthKey(month) != tc.want) {
			t.Errorf("%q: %v %v", tc.value, month, err)
		}
		if !tc.ok && !errors.Is(err, ErrValidation) {
			t.Errorf("%q must be rejected, got %v", tc.value, err)
		}
	}
	if month, err := SnapshotMonth("", day(2026, 11, 3)); err != nil || MonthKey(month) != "2026-10" {
		t.Fatalf("early in the month the default is last month: %v %v", month, err)
	}
}

func TestGivenSnapshotRowsWhenLookingUpAMonthThenTheFirstRowWins(t *testing.T) {
	sheet := SnapshotSheet{Snapshots: []MonthSnapshot{{Month: "2026-10", Row: 2}, {Month: "2026-10", Row: 5}}}
	if snapshot, ok := sheet.ForMonth("2026-10"); !ok || snapshot.Row != 2 {
		t.Fatalf("got %+v", snapshot)
	}
	if _, ok := sheet.ForMonth("2026-09"); ok {
		t.Fatal("a missing month must not be found")
	}
	if got := SnapshotAsOf(day(2026, 2, 1), time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC)); got != day(2026, 2, 28) {
		t.Fatalf("as of = %v", got)
	}
}
