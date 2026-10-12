package domain

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func day(year int, month time.Month, d int) time.Time {
	return time.Date(year, month, d, 0, 0, 0, 0, time.UTC)
}

func money(value float64) *float64 { return &value }

func text(value string) *string { return &value }

func categoryType(value CategoryType) *CategoryType { return &value }

func txn(date time.Time, category string, kind CategoryType, payment float64, deposit float64) Transaction {
	t := Transaction{AccountName: "Cash", Date: date, Category: category, CategoryType: categoryType(kind)}
	if payment != 0 {
		t.Payment = money(payment)
	}
	if deposit != 0 {
		t.Deposit = money(deposit)
	}
	return t
}

func TestEffectiveDate(t *testing.T) {
	base := txn(day(2026, 4, 10), "Refunds/Reimbursements", CategoryTypeIncome, 0, 150000)
	tests := []struct {
		name string
		memo *string
		want time.Time
	}{
		{"no memo keeps transaction date", nil, day(2026, 4, 10)},
		{"sheets serial", text("46086"), day(2026, 3, 5)},
		{"month/day/year uses month", text("3/15/2026"), day(2026, 3, 1)},
		{"month-year", text("cashback 03-2026"), day(2026, 3, 1)},
		{"year-month", text("2026-02"), day(2026, 2, 1)},
		{"quoted serial", text(`"46086"`), day(2026, 3, 5)},
		{"unrelated memo keeps transaction date", text("thanks"), day(2026, 4, 10)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := base
			tx.Memo = tt.memo
			if got := EffectiveDate(tx); !got.Equal(tt.want) {
				t.Fatalf("got %s want %s", got.Format(time.DateOnly), tt.want.Format(time.DateOnly))
			}
		})
	}
}

func TestEffectiveDateOnlyAppliesToRefundDeposits(t *testing.T) {
	payment := txn(day(2026, 4, 10), "Refunds/Reimbursements", CategoryTypeIncome, 50000, 0)
	payment.Memo = text("2026-02")
	other := txn(day(2026, 4, 10), "Salary", CategoryTypeIncome, 0, 50000)
	other.Memo = text("2026-02")
	for _, tx := range []Transaction{payment, other} {
		if got := EffectiveDate(tx); !got.Equal(day(2026, 4, 10)) {
			t.Fatalf("expected own date for %s, got %s", tx.Category, got)
		}
	}
}

func TestComputeBalanceSheetIncludesLoans(t *testing.T) {
	accounts := []Account{{Name: "Bank", Balance: 10_000_000}, {Name: "Card", Balance: -2_000_000}, {Name: "Wallet", Balance: 500_000}}
	loans := []Loan{{Name: "Family", YearlyRemaining: 3_000_000}}
	got := ComputeBalanceSheet(accounts, loans)
	want := BalanceSheet{Assets: 10_500_000, Liabilities: 5_000_000, AccountsNet: 8_500_000, LoansRemaining: 3_000_000, NetWorth: 5_500_000}
	if got != want {
		t.Fatalf("got %+v want %+v", got, want)
	}
	if got.Assets-got.Liabilities != got.NetWorth {
		t.Fatal("net worth must equal assets minus liabilities")
	}
}

func TestComputeCashFlowExcludesTransfersAndAttributesRefunds(t *testing.T) {
	march := day(2026, 3, 1)
	refund := txn(day(2026, 4, 2), "Refunds/Reimbursements", CategoryTypeIncome, 0, 100_000)
	refund.Memo = text("2026-03")
	transactions := []Transaction{
		txn(day(2026, 3, 1), "Salary", CategoryTypeIncome, 0, 20_000_000),
		txn(day(2026, 3, 5), "Food", CategoryTypeExpense, 3_000_000, 0),
		txn(day(2026, 3, 6), "Transfer", CategoryTypeNonBudget, 5_000_000, 5_000_000),
		txn(day(2026, 4, 1), "Food", CategoryTypeExpense, 1_000_000, 0),
		refund,
	}
	got := ComputeCashFlow(transactions, march)
	if got.Income != 20_100_000 || got.Expense != 3_000_000 || got.CashFlow != 17_100_000 {
		t.Fatalf("unexpected cash flow %+v", got)
	}
	if got.SavingsRate != 85.07 {
		t.Fatalf("expected savings rate 85.07, got %v", got.SavingsRate)
	}
}

func TestComputeCashFlowSkipsRowsWithIssues(t *testing.T) {
	bad := txn(day(2026, 3, 5), "Food", CategoryTypeExpense, 999, 0)
	bad.Issues = []string{"unparseable date"}
	if got := ComputeCashFlow([]Transaction{bad}, day(2026, 3, 1)); got.Expense != 0 {
		t.Fatalf("expected flagged row to be ignored, got %+v", got)
	}
}

func TestBackCalculateSeries(t *testing.T) {
	today := day(2026, 3, 10)
	transactions := []Transaction{
		txn(day(2026, 3, 10), "Food", CategoryTypeExpense, 100, 0),
		txn(day(2026, 3, 9), "Salary", CategoryTypeIncome, 0, 1000),
		txn(day(2026, 3, 1), "Food", CategoryTypeExpense, 5, 0), // outside window
	}
	got := BackCalculateSeries(2000, transactions, today, 3)
	want := []TrendPoint{{"2026-03-08", 1100}, {"2026-03-09", 2100}, {"2026-03-10", 2000}}
	if len(got) != len(want) {
		t.Fatalf("got %d points", len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("point %d: got %+v want %+v", i, got[i], want[i])
		}
	}
}

func TestPercentChange(t *testing.T) {
	if PercentChange(110, 100) == nil || *PercentChange(110, 100) != 10 {
		t.Fatal("expected +10%")
	}
	if got := PercentChange(-50, -100); got == nil || *got != 50 {
		t.Fatalf("expected +50%% when a negative base improves, got %v", got)
	}
	if PercentChange(10, 0) != nil {
		t.Fatal("expected nil for zero base")
	}
}

func TestAverageMonthlyExpenseUsesFullMonthsOnly(t *testing.T) {
	transactions := []Transaction{
		txn(day(2026, 1, 3), "Food", CategoryTypeExpense, 300, 0),
		txn(day(2026, 2, 3), "Food", CategoryTypeExpense, 600, 0),
		txn(day(2026, 3, 3), "Food", CategoryTypeExpense, 900, 0), // current month, excluded
	}
	if got := AverageMonthlyExpense(transactions, day(2026, 3, 15), 2); got != 450 {
		t.Fatalf("got %v", got)
	}
}

func TestGivenRefundWithMemoMonthWhenEncodedThenEffectiveDateIsTheAttributedDay(t *testing.T) {
	refund := txn(day(2026, 4, 10), "Refunds/Reimbursements", CategoryTypeIncome, 0, 150000)
	refund.Memo = text("03-2026")
	payload, err := json.Marshal(refund)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(payload), `"effectiveDate":"2026-03-01"`) {
		t.Fatalf("payload = %s", payload)
	}
	var decoded Transaction
	if err := json.Unmarshal(payload, &decoded); err != nil || decoded.Payee != refund.Payee || !decoded.Date.Equal(refund.Date) {
		t.Fatalf("round trip = %+v, %v", decoded, err)
	}
}

func TestGivenTransactionWithoutDateWhenEncodedThenEffectiveDateIsNull(t *testing.T) {
	payload, err := json.Marshal(Transaction{Issues: []string{"bad date"}})
	if err != nil || !strings.Contains(string(payload), `"effectiveDate":null`) {
		t.Fatalf("payload = %s, %v", payload, err)
	}
}

// Balance reconstruction undoes real cash movements, so it must use the
// booking date even for a refund attributed to an earlier month (WM-RULE-4
// applies to reporting, not to balances).
func TestGivenRefundAttributedToLastMonthWhenReconstructingBalancesThenBookingDateIsUsed(t *testing.T) {
	refund := txn(day(2026, 4, 10), "Refunds/Reimbursements", CategoryTypeIncome, 0, 100)
	refund.Memo = text("03-2026")
	if flow := MonthNetFlow([]Transaction{refund}, day(2026, 4, 15)); flow != 100 {
		t.Fatalf("april net flow = %v, want the refund counted when it arrived", flow)
	}
	series := BackCalculateSeries(1000, []Transaction{refund}, day(2026, 4, 15), 10)
	if series[0].Value != 900 || series[9].Value != 1000 {
		t.Fatalf("series = %+v", series)
	}
}
