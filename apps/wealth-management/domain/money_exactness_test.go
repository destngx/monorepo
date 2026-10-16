package domain

import (
	"math"
	"math/big"
	"testing"
)

// WM-NFR-5: amounts are float64 in the domain. float64 holds every integer
// up to 2^53 (about 9.007e15 VND) exactly, so sums of whole VND amounts stay
// exact while every partial sum stays below that bound. These tests pin that
// property through the real aggregation paths.

const (
	exactnessCount     = 100_000
	exactnessMaxAmount = 10_000_000_000 // 10^10 VND per transaction
)

// largeAmounts returns deterministic whole VND amounts in [1, 10^10] and
// their exact total.
func largeAmounts(count int) ([]float64, *big.Int) {
	amounts := make([]float64, count)
	total := new(big.Int)
	seed := uint64(7)
	for i := range amounts {
		seed = seed*6364136223846793005 + 1442695040888963407
		amount := int64((seed>>11)%exactnessMaxAmount) + 1
		amounts[i] = float64(amount)
		total.Add(total, big.NewInt(amount))
	}
	return amounts, total
}

func exactInt(t *testing.T, value float64) *big.Int {
	t.Helper()
	exact, accuracy := big.NewFloat(value).Int(nil)
	if accuracy != big.Exact {
		t.Fatalf("%v is not a whole number", value)
	}
	return exact
}

func TestGivenOneHundredThousandLargeVNDExpensesWhenBuildingTheBudgetReportThenTotalsAreExact(t *testing.T) {
	amounts, want := largeAmounts(exactnessCount)
	october := day(2026, 10, 1)
	transactions := make([]Transaction, 0, len(amounts))
	for i, amount := range amounts {
		transactions = append(transactions, txn(day(2026, 10, 1+i%28), "Food", CategoryTypeExpense, amount, 0))
	}
	items := []BudgetItem{{Category: "Food", CategoryType: categoryType(CategoryTypeExpense), MonthlyLimit: 1}}

	report := ComputeBudgetReport(items, transactions, october, day(2026, 10, 31))

	if got := exactInt(t, report.TotalSpent); got.Cmp(want) != 0 {
		t.Fatalf("budget total spent = %s, want %s", got, want)
	}
	if got := exactInt(t, report.Lines[0].Spent); got.Cmp(want) != 0 {
		t.Fatalf("line spent = %s, want %s", got, want)
	}
}

func TestGivenOneHundredThousandLargeVNDFlowsWhenComputingCashFlowThenIncomeExpenseAndNetAreExact(t *testing.T) {
	expenses, wantExpense := largeAmounts(exactnessCount)
	incomes, wantIncome := largeAmounts(exactnessCount / 2)
	october := day(2026, 10, 1)
	transactions := make([]Transaction, 0, len(expenses)+len(incomes))
	for i, amount := range expenses {
		transactions = append(transactions, txn(day(2026, 10, 1+i%28), "Food", CategoryTypeExpense, amount, 0))
	}
	for i, amount := range incomes {
		transactions = append(transactions, txn(day(2026, 10, 1+i%28), "Salary", CategoryTypeIncome, 0, amount))
	}

	flow := ComputeCashFlow(transactions, october)

	if got := exactInt(t, flow.Expense); got.Cmp(wantExpense) != 0 {
		t.Fatalf("expense = %s, want %s", got, wantExpense)
	}
	if got := exactInt(t, flow.Income); got.Cmp(wantIncome) != 0 {
		t.Fatalf("income = %s, want %s", got, wantIncome)
	}
	wantNet := new(big.Int).Sub(wantIncome, wantExpense)
	if got := exactInt(t, flow.CashFlow); got.Cmp(wantNet) != 0 {
		t.Fatalf("cash flow = %s, want %s", got, wantNet)
	}
}

func TestGivenOneHundredThousandLargeBalancesWhenComputingTheBalanceSheetThenNetWorthIsExact(t *testing.T) {
	amounts, wantAssets := largeAmounts(exactnessCount)
	accounts := make([]Account, 0, len(amounts))
	for _, amount := range amounts {
		accounts = append(accounts, Account{Name: "a", Type: AccountTypeBank, Currency: CurrencyVND, Balance: amount})
	}
	owed := []Transaction{lendingTxn("row-2", "2026-01-01", "Person A", 0, 9_999_999_999)}

	sheet := ComputeBalanceSheet(accounts, owed)

	if got := exactInt(t, sheet.Assets); got.Cmp(wantAssets) != 0 {
		t.Fatalf("assets = %s, want %s", got, wantAssets)
	}
	wantNet := new(big.Int).Sub(wantAssets, big.NewInt(9_999_999_999))
	if got := exactInt(t, sheet.NetWorth); got.Cmp(wantNet) != 0 {
		t.Fatalf("net worth = %s, want %s", got, wantNet)
	}
}

func TestGivenAUSDHoldingWithCentsWhenValuingInVNDThenItRoundsToTheExactWholeVND(t *testing.T) {
	tests := []struct {
		balance float64
		rate    float64
		want    float64 // exact product rounded half up to whole VND
	}{
		{balance: 1234.56, rate: 25_432, want: 31_397_330},          // 31,397,329.92
		{balance: 0.01, rate: 26_150, want: 262},                    // 261.5 rounds up
		{balance: 987_654.32, rate: 25_999.5, want: 25_678_518_493}, // 25,678,518,492.84
		{balance: 3_250, rate: 26_180.75, want: 85_087_438},         // 85,087,437.5 rounds up
	}
	for _, test := range tests {
		valuation := ValuePortfolio(PortfolioInputs{
			Assets: InvestmentAssets{Crypto: InvestmentHoldingSection{Items: []Holding{usdHolding(test.balance)}}},
			Rates:  map[Currency]ExchangeRate{CurrencyUSD: {From: "USD", To: "VND", Rate: test.rate}},
		})
		got := valuation.Totals.CryptoHoldings
		exact := new(big.Rat).Mul(decimalRat(t, test.balance), decimalRat(t, test.rate))
		exactFloat, _ := exact.Float64()
		if math.Abs(got-exactFloat) > 0.01 {
			t.Errorf("%v USD x %v = %v, exact %v", test.balance, test.rate, got, exact.FloatString(4))
		}
		if rounded := math.Floor(got + 0.5); rounded != test.want {
			t.Errorf("%v USD x %v rounds to %v, want %v", test.balance, test.rate, rounded, test.want)
		}
	}
}

func TestGivenTenThousandUSDHoldingsWhenSummingTheirVNDValuesThenTheTotalIsWithinOneVND(t *testing.T) {
	const count = 10_000
	const rate = 25_432.5
	holdings := make([]Holding, 0, count)
	exact := new(big.Rat)
	for i := range count {
		cents := int64(i*7919%10_000_000 + 1) // up to 100,000.00 USD
		balance := float64(cents) / 100
		holdings = append(holdings, usdHolding(balance))
		exact.Add(exact, new(big.Rat).Mul(big.NewRat(cents, 100), decimalRat(t, rate)))
	}

	valuation := ValuePortfolio(PortfolioInputs{
		Assets: InvestmentAssets{Crypto: InvestmentHoldingSection{Items: holdings}},
		Rates:  map[Currency]ExchangeRate{CurrencyUSD: {From: "USD", To: "VND", Rate: rate}},
	})

	exactFloat, _ := exact.Float64()
	if drift := math.Abs(valuation.Totals.CryptoHoldings - exactFloat); drift >= 1 {
		t.Fatalf("USD total drifted %v VND from exact %s", drift, exact.FloatString(2))
	}
}

// usdHolding is a USD-valued holding: account balances are VND already
// (D25), so USD to VND conversion only happens for holdings.
func usdHolding(value float64) Holding {
	return Holding{Platform: "Exchange", Symbol: "USD", Value: &value, Currency: CurrencyUSD, ValueSource: HoldingValueFromSheet}
}

func decimalRat(t *testing.T, value float64) *big.Rat {
	t.Helper()
	rat, ok := new(big.Rat).SetString(big.NewFloat(value).Text('f', 6))
	if !ok {
		t.Fatalf("cannot parse %v", value)
	}
	return rat
}
