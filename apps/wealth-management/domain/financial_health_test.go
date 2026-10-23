package domain

import (
	"strings"
	"testing"
	"time"
)

var healthToday = day(2026, 10, 15)

// healthyMonth is 20M income against 6M rent, 2M groceries and 2M dining.
func healthyMonth(month time.Month) []Transaction {
	date := day(2026, month, 5)
	return []Transaction{
		txn(date, "Salary", CategoryTypeIncome, 0, 20_000_000),
		withNature(txn(date, "Rent", CategoryTypeExpense, 6_000_000, 0), NatureFixed),
		withNature(txn(date, "Groceries", CategoryTypeExpense, 2_000_000, 0), NatureEssential),
		withNature(txn(date, "Dining", CategoryTypeExpense, 2_000_000, 0), NatureFlexible),
	}
}

// healthFixture: six full months (April to September) of healthyMonth.
func healthFixture() ([]Account, []Transaction) {
	accounts := []Account{
		{Name: "Bank", Type: AccountTypeBank, Balance: 60_000_000},
		{Name: "Fund", Type: AccountTypeInvestment, Balance: 48_000_000},
		{Name: "Binance", Type: AccountTypeCrypto, Balance: 12_000_000},
		{Name: "Card", Type: AccountTypeNegativeActive, Balance: -2_000_000},
	}
	var transactions []Transaction
	for month := time.April; month <= time.September; month++ {
		transactions = append(transactions, healthyMonth(month)...)
	}
	// the current month is outside the lookback
	transactions = append(transactions, txn(day(2026, 10, 2), "Dining", CategoryTypeExpense, 50_000_000, 0))
	return accounts, transactions
}

// healthFor scores with the allocation of the accounts alone (no holdings).
func healthFor(accounts []Account, transactions []Transaction, today time.Time) FinancialHealth {
	allocation := ComputeAllocation(AllocationInputs{Accounts: accounts, Transactions: transactions, Now: today})
	return ComputeFinancialHealth(accounts, transactions, allocation, today)
}

func pillars(health FinancialHealth) map[string]HealthComponent {
	byKey := map[string]HealthComponent{}
	for _, component := range health.Components {
		byKey[component.Key] = component
	}
	return byKey
}

func capKeys(health FinancialHealth) []string {
	keys := []string{}
	for _, c := range health.Caps {
		keys = append(keys, c.Key)
	}
	return keys
}

func TestGivenAHealthyLedgerWhenScoringThenEveryPillarIsMeasuredAndNoCapApplies(t *testing.T) {
	accounts, transactions := healthFixture()
	health := healthFor(accounts, transactions, healthToday)
	p := pillars(health)

	if len(health.Components) != 8 {
		t.Fatalf("expected 8 pillars, got %d", len(health.Components))
	}
	weights := 0.0
	for _, component := range health.Components {
		weights += component.MaxScore
	}
	if weights != 100 {
		t.Fatalf("pillar weights sum to %v", weights)
	}
	// 10M saved of 20M = 50% -> full 20
	if s := p["savings"]; s.Value != 50 || s.Score != 20 {
		t.Fatalf("savings %+v", s)
	}
	if c := p["consistency"]; c.Value != 6 || c.Score != 10 {
		t.Fatalf("consistency %+v", c)
	}
	// 60M liquid / 10M expense = 6 months -> full 20
	if e := p["emergency"]; e.Value != 6 || e.Score != 20 {
		t.Fatalf("emergency %+v", e)
	}
	// rent + groceries = 8M of 20M = 40% -> full
	if n := p["needs"]; n.Value != 40 || n.Score != 10 {
		t.Fatalf("needs %+v", n)
	}
	// 5% x 2M card = 100k / 20M = 0.5% -> full
	if d := p["debt"]; d.Value != 0.5 || d.Score != 10 {
		t.Fatalf("debt %+v", d)
	}
	if s := p["spendingTrend"]; s.Value != 0 || s.Score != 10 {
		t.Fatalf("spending trend %+v", s)
	}
	if l := p["lending"]; l.Value != 0 || l.Score != 10 {
		t.Fatalf("lending %+v", l)
	}
	// 50/40/10 split: HHI 0.42 -> 10 x (1 - 0.42) / (2/3) = 8.7
	if v := p["diversification"]; v.Value != 0.42 || v.Score != 8.7 {
		t.Fatalf("diversification %+v", v)
	}
	if len(health.Caps) != 0 || health.Score != 98.7 || health.UncappedScore != 98.7 || health.Band != HealthBandGreen {
		t.Fatalf("score %v (uncapped %v) band %s caps %v", health.Score, health.UncappedScore, health.Band, capKeys(health))
	}
	if health.LookbackMonths != 6 || len(health.Recommendations) != 3 || health.Recommendations[0].Component != "diversification" {
		t.Fatalf("orders %+v", health.Recommendations)
	}
	// net worth 118M / 10M expense
	if health.FreedomMonths == nil || *health.FreedomMonths != 11.8 {
		t.Fatalf("freedom months %v", health.FreedomMonths)
	}
}

func TestGivenNoDataWhenScoringThenNothingIsAssumedAndTheScoreIsHeldInTheRed(t *testing.T) {
	health := healthFor(nil, nil, healthToday)

	// Only the debt pillar scores: there is no debt.
	if health.UncappedScore != 10 || health.Score != 10 || health.Band != HealthBandRed || !health.InsufficientIncome || health.FreedomMonths != nil {
		t.Fatalf("unexpected empty health %+v", health)
	}
	if got := strings.Join(capKeys(health), ","); got != "noIncome,noBuffer,deficitMonths" {
		t.Fatalf("caps %s", got)
	}
	if p := pillars(health); p["spendingTrend"].Score != 0 || !strings.HasPrefix(p["spendingTrend"].Detail, "Not scored") {
		t.Fatalf("spending trend %+v", p["spendingTrend"])
	}
}

func TestGivenSpendingAboveIncomeWhenScoringThenTheScoreIsHeldAt49(t *testing.T) {
	accounts, transactions := healthFixture()
	// One 70M purchase in September: 6 months of 10M + 70M = 130M against 120M income.
	transactions = append(transactions, withNature(txn(day(2026, 9, 20), "Dining", CategoryTypeExpense, 70_000_000, 0), NatureFlexible))
	health := healthFor(accounts, transactions, healthToday)

	if health.Score != HealthCapCritical || health.UncappedScore <= HealthCapCritical || health.Band != HealthBandRed {
		t.Fatalf("score %v uncapped %v band %s", health.Score, health.UncappedScore, health.Band)
	}
	if health.Caps[0].Key != "deficit" || !strings.Contains(health.Caps[0].Reason, "spent more than you earned") {
		t.Fatalf("caps %+v", health.Caps)
	}
	if health.Recommendations[0].Component != "savings" {
		t.Fatalf("the deficit's pillar comes first: %+v", health.Recommendations)
	}
}

func TestGivenAThinBufferWhenScoringThenTheScoreCannotBeGreen(t *testing.T) {
	accounts, transactions := healthFixture()
	accounts[0].Balance = 25_000_000 // 2.5 months of expense
	health := healthFor(accounts, transactions, healthToday)

	// 20 x (2.5 - 1) / 5 = 6
	if e := pillars(health)["emergency"]; e.Value != 2.5 || e.Score != 6 {
		t.Fatalf("emergency %+v", e)
	}
	if got := strings.Join(capKeys(health), ","); got != "thinBuffer" || health.Score > HealthCapSerious || health.Band != HealthBandYellow {
		t.Fatalf("caps %s score %v band %s", got, health.Score, health.Band)
	}
}

func TestGivenLessThanAMonthOfCashWhenScoringThenTheEmergencyPillarScoresNothing(t *testing.T) {
	accounts, transactions := healthFixture()
	accounts[0].Balance = 8_000_000
	health := healthFor(accounts, transactions, healthToday)

	if e := pillars(health)["emergency"]; e.Score != 0 {
		t.Fatalf("emergency %+v", e)
	}
	if health.Caps[0].Key != "noBuffer" || health.Score > HealthCapCritical {
		t.Fatalf("caps %+v score %v", health.Caps, health.Score)
	}
}

func TestGivenUntaggedExpenseWhenScoringNeedsThenItCountsAsANeed(t *testing.T) {
	accounts, transactions := healthFixture()
	for i := range transactions {
		if transactions[i].Category == "Dining" {
			transactions[i].CategoryNature = ""
		}
	}
	// 10M of 20M = 50% -> still full at the rule of thumb
	if n := pillars(healthFor(accounts, transactions, healthToday))["needs"]; n.Value != 50 || n.Score != 10 {
		t.Fatalf("needs %+v", n)
	}
	for month := time.April; month <= time.September; month++ {
		transactions = append(transactions, txn(day(2026, month, 6), "Misc", CategoryTypeExpense, 3_000_000, 0))
	}
	// 13M of 20M = 65% -> 10 x (80 - 65) / 30 = 5
	if n := pillars(healthFor(accounts, transactions, healthToday))["needs"]; n.Value != 65 || n.Score != 5 {
		t.Fatalf("needs %+v", n)
	}
}

func TestGivenRisingSpendingWhenScoringThenTheTrendLosesPointsAndCapsAt20Percent(t *testing.T) {
	accounts, transactions := healthFixture()
	for month := time.July; month <= time.September; month++ {
		transactions = append(transactions, withNature(txn(day(2026, month, 7), "Dining", CategoryTypeExpense, 1_000_000, 0), NatureFlexible))
	}
	// 11M against 10M = +10% -> 5
	health := healthFor(accounts, transactions, healthToday)
	if s := pillars(health)["spendingTrend"]; s.Value != 10 || s.Score != 5 {
		t.Fatalf("spending trend %+v", s)
	}
	for month := time.July; month <= time.September; month++ {
		transactions = append(transactions, withNature(txn(day(2026, month, 8), "Dining", CategoryTypeExpense, 1_000_000, 0), NatureFlexible))
	}
	health = healthFor(accounts, transactions, healthToday)
	if s := pillars(health)["spendingTrend"]; s.Value != 20 || s.Score != 0 {
		t.Fatalf("spending trend %+v", s)
	}
	if got := strings.Join(capKeys(health), ","); got != "spendingGrowth" {
		t.Fatalf("caps %s", got)
	}
}

func TestGivenMoneyLentWhenScoringThenItsShareOfAssetsIsScoredAndCapped(t *testing.T) {
	accounts, transactions := healthFixture()
	// 80M lent on top of 120M: 40% of 200M
	transactions = append(transactions, lendingTxn("row-9", "2026-05-02", "Person A", 80_000_000, 0))
	health := healthFor(accounts, transactions, healthToday)

	if l := pillars(health)["lending"]; l.Value != 40 || l.Score != 0 {
		t.Fatalf("lending %+v", l)
	}
	if got := strings.Join(capKeys(health), ","); got != "lentTooMuch" {
		t.Fatalf("caps %s", got)
	}
}

func TestGivenCryptoAboveAQuarterWhenScoringThenTheScoreCannotBeGreen(t *testing.T) {
	accounts, transactions := healthFixture()
	accounts[2].Balance = 60_000_000 // 60 of 168M = 35.7%
	health := healthFor(accounts, transactions, healthToday)

	if got := strings.Join(capKeys(health), ","); got != "cryptoHeavy" || health.Band != HealthBandYellow {
		t.Fatalf("caps %s band %s", got, health.Band)
	}
	if health.Recommendations[0].Component != "diversification" {
		t.Fatalf("orders %+v", health.Recommendations)
	}
}

func TestGivenTwoDeficitMonthsWhenScoringThenConsistencyDropsAndTheScoreCannotBeGreen(t *testing.T) {
	accounts, transactions := healthFixture()
	for _, month := range []time.Month{time.May, time.August} {
		transactions = append(transactions, withNature(txn(day(2026, month, 9), "Dining", CategoryTypeExpense, 11_000_000, 0), NatureFlexible))
	}
	health := healthFor(accounts, transactions, healthToday)

	// 4 of 6 -> 10 x 4/6
	if c := pillars(health)["consistency"]; c.Value != 4 || c.Score != 6.67 {
		t.Fatalf("consistency %+v", c)
	}
	if got := capKeys(health); len(got) == 0 || got[len(got)-1] != "deficitMonths" || health.Band == HealthBandGreen {
		t.Fatalf("caps %v band %s", got, health.Band)
	}
}

func TestGivenCardBalancesWhenScoringDebtThenCardMinimumsAreComparedToMonthlyIncome(t *testing.T) {
	accounts := []Account{
		{Name: "Bank", Type: AccountTypeBank, Balance: 50_000_000},
		{Name: "Card", Type: AccountTypeNegativeActive, Balance: -40_000_000},
	}
	// Money the user owes a person has no payment schedule: not a monthly payment (D16).
	transactions := []Transaction{lendingTxn("row-2", "2026-04-02", "Person A", 0, 2_000_000)}
	for month := time.April; month <= time.September; month++ {
		transactions = append(transactions, txn(day(2026, month, 5), "Salary", CategoryTypeIncome, 0, 10_000_000))
	}
	health := healthFor(accounts, transactions, healthToday)

	debt := pillars(health)["debt"]
	// 5% x 40M card = 2M / 10M income = 20% -> 10 x (35 - 20) / 30 = 5
	if debt.Value != 20 || debt.Score != 5 || health.MonthlyDebtPayments != 2_000_000 {
		t.Fatalf("debt %+v, payments %v", debt, health.MonthlyDebtPayments)
	}
	if !strings.Contains(debt.Formula, "card minimum") || strings.Contains(debt.Formula, "loan") || debt.Detail != "2000000 / 10000000 = 20.00%" {
		t.Fatalf("formula %q detail %q", debt.Formula, debt.Detail)
	}
}

func TestGivenDebtPaymentsAndNoIncomeWhenScoringDebtThenRatioIsCapped(t *testing.T) {
	health := healthFor([]Account{{Name: "Card", Balance: -20}}, nil, healthToday)
	if d := pillars(health)["debt"]; d.Value != 100 || d.Score != 0 {
		t.Fatalf("debt %+v", d)
	}
	if got := strings.Join(capKeys(health), ","); !strings.Contains(got, "negativeNetWorth") || !strings.Contains(got, "highDebt") {
		t.Fatalf("caps %s", got)
	}
}

func TestDiversificationSingleClassScoresZero(t *testing.T) {
	hhi, score := diversificationScore(ComputeAllocation(AllocationInputs{Accounts: []Account{{Name: "Bank", Type: AccountTypeBank, Balance: 100}}}))
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

func TestGivenNetWorthWhenComputingHealthThenProjectionUsesFallbackReturnAndVNDInflation(t *testing.T) {
	accounts := []Account{
		{Name: "Bank", Type: AccountTypeBank, Balance: 60_000_000},
		{Name: "Fund", Type: AccountTypeInvestment, Balance: 60_000_000},
		{Name: "Binance", Type: AccountTypeCrypto, Balance: 60_000_000},
		{Name: "Card", Type: AccountTypeNegativeActive, Balance: -12_000_000},
	}
	health := healthFor(accounts, nil, day(2026, 4, 15))

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
