package domain

import (
	"strings"
	"testing"
	"time"
)

func healthFixture() ([]Account, []Transaction) {
	accounts := []Account{
		{Name: "Bank", Type: AccountTypeBank, Balance: 60_000_000},
		{Name: "Fund", Type: AccountTypeInvestment, Balance: 60_000_000},
		{Name: "Binance", Type: AccountTypeCrypto, Balance: 60_000_000},
		{Name: "Card", Type: AccountTypeNegativeActive, Balance: -12_000_000},
	}
	var transactions []Transaction
	for month := time.January; month <= time.March; month++ {
		date := day(2026, month, 5)
		transactions = append(transactions,
			txn(date, "Salary", CategoryTypeIncome, 0, 20_000_000),
			txn(date, "Food", CategoryTypeExpense, 10_000_000, 0),
		)
	}
	// the current month is excluded from the lookback
	transactions = append(transactions, txn(day(2026, 4, 2), "Food", CategoryTypeExpense, 50_000_000, 0))
	return accounts, transactions
}

func TestComputeFinancialHealth(t *testing.T) {
	accounts, transactions := healthFixture()
	health := ComputeFinancialHealth(accounts, transactions, day(2026, 4, 15))

	scores := map[string]HealthComponent{}
	for _, component := range health.Components {
		scores[component.Key] = component
	}
	// savings 50% -> full 30; emergency 60M / 10M = 6 months -> full 30
	if s := scores["savings"]; s.Value != 50 || s.Score != 30 {
		t.Fatalf("savings %+v", s)
	}
	if e := scores["emergency"]; e.Value != 6 || e.Score != 30 {
		t.Fatalf("emergency %+v", e)
	}
	// debt payments: 5% minimum of the 12M card = 600k / 20M income = 3% -> 20 x (1 - 3/70)
	if d := scores["debt"]; d.Value != 3 || d.Score != 19.14 {
		t.Fatalf("debt %+v", d)
	}
	// an even three-way split scores full diversification
	if v := scores["diversification"]; v.Score != 20 {
		t.Fatalf("diversification %+v", v)
	}
	if health.Score != 99.14 || health.Band != HealthBandGreen {
		t.Fatalf("score %v status %s", health.Score, health.Band)
	}
	if len(health.Recommendations) != 3 || health.Recommendations[0].Component != "debt" {
		t.Fatalf("expected debt as the first order, got %+v", health.Recommendations)
	}
	// net worth 168M / 10M expense
	if health.FreedomMonths == nil || *health.FreedomMonths != 16.8 {
		t.Fatalf("freedom months %v", health.FreedomMonths)
	}
}

func TestFinancialHealthWithNoData(t *testing.T) {
	health := ComputeFinancialHealth(nil, nil, day(2026, 4, 15))
	if health.Score != 20 || health.Band != HealthBandRed || !health.InsufficientIncome || health.FreedomMonths != nil {
		t.Fatalf("unexpected empty health %+v", health)
	}
}

func TestDiversificationSingleClassScoresZero(t *testing.T) {
	hhi, score := diversificationScore(map[AssetClass]float64{AssetClassCash: 100})
	if hhi != 1 || score != 0 {
		t.Fatalf("hhi %v score %v", hhi, score)
	}
}

func TestHealthBandFor(t *testing.T) {
	if HealthBandFor(49.9) != HealthBandRed || HealthBandFor(50) != HealthBandYellow || HealthBandFor(75) != HealthBandYellow || HealthBandFor(75.1) != HealthBandGreen {
		t.Fatal("unexpected bands")
	}
}

func TestProjectRealValue(t *testing.T) {
	// VND: 10% - 6% = 4% for 2 years
	if got := ProjectRealValue(100_000_000, 10, CurrencyVND, 2); got != 108_160_000 {
		t.Fatalf("got %v", got)
	}
	// USD with the 8% fallback: 8% - 2% = 6%
	if got := ProjectRealValue(1000, 0, CurrencyUSD, 1); got != 1060 {
		t.Fatalf("got %v", got)
	}
}

func TestGivenCardBalancesWhenScoringDebtThenCardMinimumsAreComparedToMonthlyIncome(t *testing.T) {
	accounts := []Account{
		{Name: "Bank", Type: AccountTypeBank, Balance: 50_000_000},
		{Name: "Card", Type: AccountTypeNegativeActive, Balance: -10_000_000},
	}
	// Money the user owes a person has no payment schedule: not a monthly payment (D16).
	transactions := []Transaction{lendingTxn("row-2", "2026-01-02", "Person A", 0, 2_000_000)}
	for month := time.January; month <= time.March; month++ {
		transactions = append(transactions, txn(day(2026, month, 5), "Salary", CategoryTypeIncome, 0, 10_000_000))
	}
	health := ComputeFinancialHealth(accounts, transactions, day(2026, 4, 15))

	var debt HealthComponent
	for _, component := range health.Components {
		if component.Key == "debt" {
			debt = component
		}
	}
	// 5% x 10M card / 10M income = 5%
	if debt.Value != 5 || health.MonthlyDebtPayments != 500_000 {
		t.Fatalf("debt %+v, payments %v", debt, health.MonthlyDebtPayments)
	}
	if !strings.Contains(debt.Formula, "card minimum") || strings.Contains(debt.Formula, "loan") || debt.Detail != "500000 / 10000000 = 5.00%" {
		t.Fatalf("formula %q detail %q", debt.Formula, debt.Detail)
	}
}

func TestGivenDebtPaymentsAndNoIncomeWhenScoringDebtThenRatioIsCapped(t *testing.T) {
	health := ComputeFinancialHealth([]Account{{Name: "Card", Balance: -20}}, nil, day(2026, 4, 15))
	for _, component := range health.Components {
		if component.Key == "debt" && (component.Value != 100 || component.Score != 0) {
			t.Fatalf("debt %+v", component)
		}
	}
}

func TestGivenNetWorthWhenComputingHealthThenProjectionUsesFallbackReturnAndVNDInflation(t *testing.T) {
	accounts, transactions := healthFixture()
	health := ComputeFinancialHealth(accounts, transactions, day(2026, 4, 15))

	// net worth 168M at 8% nominal; real at 8% - 6% = 2%
	want := []NetWorthProjection{
		{Years: 5, Nominal: 246_847_117, Real: 185_485_575},
		{Years: 10, Nominal: 362_699_400, Real: 204_791_063},
		{Years: 20, Nominal: 783_040_800, Real: 249_639_163},
	}
	if len(health.Projection) != len(want) {
		t.Fatalf("projection %+v", health.Projection)
	}
	for i := range want {
		if health.Projection[i] != want[i] {
			t.Fatalf("projection[%d] = %+v, want %+v", i, health.Projection[i], want[i])
		}
	}
}
