package domain

import (
	"fmt"
	"testing"
	"time"
)

// Benchmarks for the hot domain paths (WM-NFR-1). They run over a
// deterministic synthetic ledger of benchLedgerSize transactions spread over
// two years, so numbers are comparable between runs and machines.

const benchLedgerSize = 5000

var benchToday = day(2026, 10, 13)

type benchLedger struct {
	accounts     []Account
	transactions []Transaction
	budget       []BudgetItem
}

// newBenchLedger builds the ledger with a small LCG so it needs no seed state.
func newBenchLedger(size int) benchLedger {
	expenses := []string{"Food", "Transport", "Rent", "Utilities", "Shopping", "Health", "Fun", "Education"}
	incomes := []string{"Salary", "Bonus", "Interest"}
	accounts := []Account{
		{Name: "Cash", Type: AccountTypeCash, Currency: CurrencyVND, Balance: 12_500_000},
		{Name: "VCB", Type: AccountTypeBank, Currency: CurrencyVND, Balance: 245_000_000},
		{Name: "TCB", Type: AccountTypeBank, Currency: CurrencyVND, Balance: 88_000_000},
		{Name: "Credit Card", Type: AccountTypeNegativeActive, Currency: CurrencyVND, Balance: -14_300_000},
		{Name: "Funds", Type: AccountTypeInvestment, Currency: CurrencyVND, Balance: 410_000_000},
		{Name: "Binance", Type: AccountTypeCrypto, Currency: CurrencyUSDT, Balance: 3_250},
		{Name: "USD Savings", Type: AccountTypeLongHolding, Currency: CurrencyUSD, Balance: 12_000},
		{Name: "Old Wallet", Type: AccountTypeDeprecated, Currency: CurrencyVND},
	}
	budget := make([]BudgetItem, 0, len(expenses))
	for i, category := range expenses {
		limit := float64((i + 2) * 1_000_000)
		budget = append(budget, BudgetItem{Category: category, CategoryType: categoryType(CategoryTypeExpense), MonthlyLimit: limit, YearlyLimit: limit * 12})
	}

	seed := uint64(42)
	next := func(n uint64) uint64 {
		seed = seed*6364136223846793005 + 1442695040888963407
		return (seed >> 33) % n
	}
	start := benchToday.AddDate(-2, 0, 0)
	span := uint64(benchToday.Sub(start) / (24 * time.Hour))
	transactions := make([]Transaction, 0, size)
	for i := range size {
		date := start.AddDate(0, 0, int(next(span)+1))
		account := accounts[next(5)].Name
		var t Transaction
		if next(10) == 0 {
			t = txn(date, incomes[next(uint64(len(incomes)))], CategoryTypeIncome, 0, float64((next(400)+50)*100_000))
		} else {
			t = txn(date, expenses[next(uint64(len(expenses)))], CategoryTypeExpense, float64((next(5000)+1)*1_000), 0)
		}
		t.ID = fmt.Sprintf("row-%d", i+2)
		t.AccountName = account
		transactions = append(transactions, t)
	}
	return benchLedger{accounts: accounts, transactions: transactions, budget: budget}
}

func BenchmarkComputeBudgetReport(b *testing.B) {
	ledger := newBenchLedger(benchLedgerSize)
	month := day(2026, 10, 1)
	b.ReportAllocs()
	for b.Loop() {
		ComputeBudgetReport(ledger.budget, ledger.transactions, month, benchToday)
	}
}

func BenchmarkComputeBalanceSheet(b *testing.B) {
	ledger := newBenchLedger(benchLedgerSize)
	b.ReportAllocs()
	for b.Loop() {
		ComputeBalanceSheet(ledger.accounts, ledger.transactions)
	}
}

func BenchmarkComputeDashboardSummary(b *testing.B) {
	ledger := newBenchLedger(benchLedgerSize)
	b.ReportAllocs()
	for b.Loop() {
		ComputeDashboardSummary(ledger.accounts, ledger.transactions, benchToday, 12)
	}
}

func BenchmarkComputeFinancialHealth(b *testing.B) {
	ledger := newBenchLedger(benchLedgerSize)
	b.ReportAllocs()
	for b.Loop() {
		ComputeFinancialHealth(ledger.accounts, ledger.transactions, ComputeAllocation(AllocationInputs{Accounts: ledger.accounts, Transactions: ledger.transactions}), benchToday)
	}
}
