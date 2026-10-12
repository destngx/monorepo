package domain

import (
	"testing"
	"time"
)

func healthFixture() ([]Account, []Loan, []Transaction) {
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
	return accounts, nil, transactions
}

func TestComputeFinancialHealth(t *testing.T) {
	accounts, loans, transactions := healthFixture()
	health := ComputeFinancialHealth(accounts, loans, transactions, day(2026, 4, 15))

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
	// debt 12M / 240M = 5% -> 20 x (1 - 5/70)
	if d := scores["debt"]; d.Value != 5 || d.Score != 18.57 {
		t.Fatalf("debt %+v", d)
	}
	// an even three-way split scores full diversification
	if v := scores["diversification"]; v.Score != 20 {
		t.Fatalf("diversification %+v", v)
	}
	if health.Score != 98.57 || health.Band != HealthBandGreen {
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
	health := ComputeFinancialHealth(nil, nil, nil, day(2026, 4, 15))
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
