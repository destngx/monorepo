package domain

import (
	"testing"
	"time"
)

// planToday is mid-October: July, August and September are the last three
// completed months, April to September the last six.
var planToday = day(2026, 10, 17)

func natureTxn(date time.Time, category string, nature CategoryNature, payment float64) Transaction {
	t := txn(date, category, CategoryTypeExpense, payment, 0)
	t.CategoryNature = nature
	return t
}

// monthlyHistory records, for each month from April to September, 20M of
// salary and 10M of expense: 5M rent (fixed), 3M food (essential) and 2M fun
// (flexible).
func monthlyHistory() []Transaction {
	var transactions []Transaction
	for month := time.April; month <= time.September; month++ {
		transactions = append(transactions,
			txn(day(2026, month, 1), "Salary", CategoryTypeIncome, 0, 20_000_000),
			natureTxn(day(2026, month, 3), "Rent", NatureFixed, 5_000_000),
			natureTxn(day(2026, month, 10), "Food", NatureEssential, 3_000_000),
			natureTxn(day(2026, month, 20), "Fun", NatureFlexible, 2_000_000),
		)
	}
	return transactions
}

func taggedCategories() []Category {
	return []Category{
		{Name: "Salary", Type: CategoryTypeIncome},
		{Name: "Rent", Type: CategoryTypeExpense, Nature: NatureFixed},
		{Name: "Food", Type: CategoryTypeExpense, Nature: NatureEssential},
		{Name: "Fun", Type: CategoryTypeExpense, Nature: NatureFlexible},
	}
}

func poolAccounts() []Account {
	return []Account{
		{Name: "Bank A", Type: AccountTypeActiveUse, Currency: CurrencyVND, Balance: 50_000_000},
		{Name: "Savings B", Type: AccountTypeRarelyUse, Currency: CurrencyVND, Balance: 100_000_000},
		{Name: "Exchange C", Type: AccountTypeLongHolding, Currency: CurrencyUSDT, Balance: 30_000_000},
		{Name: "Old D", Type: AccountTypeDeprecated, Currency: CurrencyVND, Balance: 5_000_000},
		{Name: "Card E", Type: AccountTypeNegativeActive, Currency: CurrencyVND, Balance: -10_000_000},
	}
}

func TestGivenAccountsWhenComputingThePoolThenCryptoArchivedAndMoneyLentStayOutAndDebtAndBufferComeOff(t *testing.T) {
	transactions := append(monthlyHistory(), lendingTxn("row-90", "2026-09-05", "Person A", 7_000_000, 0))

	pool := ComputeGoalPool(poolAccounts(), transactions, DefaultGoalPoolSettings(), nil, planToday)

	if pool.Assets != 150_000_000 || pool.CardDebt != 10_000_000 {
		t.Fatalf("assets / card debt = %v / %v", pool.Assets, pool.CardDebt)
	}
	if pool.AverageMonthlyExpense != 10_000_000 || pool.SpendingBuffer != 10_000_000 || pool.BufferMonths != 1 {
		t.Fatalf("buffer = %+v", pool)
	}
	if pool.Total != 130_000_000 {
		t.Fatalf("pool = %v, want 150M - 10M - 10M", pool.Total)
	}
	if pool.Crypto != 30_000_000 || pool.Archived != 5_000_000 || pool.MoneyLent != 7_000_000 {
		t.Fatalf("excluded = crypto %v, archived %v, lent %v", pool.Crypto, pool.Archived, pool.MoneyLent)
	}
	roles := map[string]PoolRole{}
	for _, account := range pool.Accounts {
		roles[account.Name] = account.Role
	}
	if roles["Bank A"] != PoolRoleAsset || roles["Exchange C"] != PoolRoleCrypto || roles["Old D"] != PoolRoleArchived || roles["Card E"] != PoolRoleCardDebt {
		t.Fatalf("roles = %v", roles)
	}
}

func TestGivenMoreDebtThanAssetsWhenComputingThePoolThenItIsZero(t *testing.T) {
	accounts := []Account{
		{Name: "Bank", Type: AccountTypeActiveUse, Currency: CurrencyVND, Balance: 5_000_000},
		{Name: "Card", Type: AccountTypeNegativeActive, Currency: CurrencyVND, Balance: -8_000_000},
	}
	if pool := ComputeGoalPool(accounts, nil, DefaultGoalPoolSettings(), nil, planToday); pool.Total != 0 {
		t.Fatalf("pool = %v, want 0", pool.Total)
	}
}

func TestGivenTaggedCategoriesWhenComputingNeedsThenFixedAndEssentialAverageOverSixMonths(t *testing.T) {
	needs := ComputeMonthlyNeeds(monthlyHistory(), taggedCategories(), planToday)

	if needs.PerMonth != 8_000_000 || needs.Months != 6 || needs.FromTotalExpense {
		t.Fatalf("needs = %+v", needs)
	}
}

func TestGivenNoNaturesWhenComputingNeedsThenTotalExpenseIsUsedAndFlagged(t *testing.T) {
	categories := []Category{{Name: "Rent", Type: CategoryTypeExpense}, {Name: "Fun", Type: CategoryTypeExpense, Nature: NatureFlexible}}

	needs := ComputeMonthlyNeeds(monthlyHistory(), categories, planToday)

	if needs.PerMonth != 10_000_000 || !needs.FromTotalExpense {
		t.Fatalf("needs = %+v", needs)
	}
}

func TestGivenAShortHistoryWhenAveragingThenOnlyMonthsSinceTheFirstTransactionCount(t *testing.T) {
	transactions := []Transaction{
		txn(day(2026, 9, 1), "Salary", CategoryTypeIncome, 0, 20_000_000),
		natureTxn(day(2026, 9, 3), "Rent", NatureFixed, 6_000_000),
	}
	needs := ComputeMonthlyNeeds(transactions, taggedCategories(), planToday)
	savings := ComputeMonthlySavings(transactions, planToday)

	if needs.PerMonth != 6_000_000 || needs.Months != 1 {
		t.Fatalf("needs = %+v", needs)
	}
	if savings.PerMonth != 14_000_000 || savings.Months != 1 {
		t.Fatalf("savings = %+v", savings)
	}
}

func TestGivenIncomeAndExpenseWhenComputingMonthlySavingsThenLendingIsLeftOutAndTheFloorIsZero(t *testing.T) {
	transactions := append(monthlyHistory(), lendingTxn("row-91", "2026-08-05", "Person A", 9_000_000, 0))

	if savings := ComputeMonthlySavings(transactions, planToday); savings.PerMonth != 10_000_000 || savings.Months != 3 {
		t.Fatalf("savings = %+v", savings)
	}
	overspent := []Transaction{natureTxn(day(2026, 9, 3), "Rent", NatureFixed, 6_000_000)}
	if savings := ComputeMonthlySavings(overspent, planToday); savings.PerMonth != 0 {
		t.Fatalf("savings must not go below zero: %+v", savings)
	}
}

func TestGivenSlowAccountsWhenComputingThePoolThenTheyAreKeptApartFromReadyMoney(t *testing.T) {
	accounts := append(poolAccounts(),
		Account{Name: "Fund F", Type: AccountTypeInvestment, Currency: CurrencyVND, Balance: 40_000_000},
		Account{Name: "Gold G", Type: AccountTypeLongHolding, Currency: CurrencyVND, Balance: 15_000_000},
		Account{Name: "Term H", Type: AccountTypeBank, Currency: CurrencyVND, Balance: 20_000_000},
	)

	pool := ComputeGoalPool(accounts, monthlyHistory(), DefaultGoalPoolSettings(), []string{" term h "}, planToday)

	if pool.Assets != 150_000_000 || pool.Total != 130_000_000 {
		t.Fatalf("ready money = assets %v, pool %v", pool.Assets, pool.Total)
	}
	if pool.Slow != 75_000_000 {
		t.Fatalf("slow = %v, want investment, long holding and not-ready accounts", pool.Slow)
	}
	roles := map[string]PoolRole{}
	for _, account := range pool.Accounts {
		roles[account.Name] = account.Role
	}
	if roles["Fund F"] != PoolRoleSlow || roles["Gold G"] != PoolRoleSlow || roles["Term H"] != PoolRoleSlow || roles["Savings B"] != PoolRoleAsset || roles["Exchange C"] != PoolRoleCrypto {
		t.Fatalf("roles = %v", roles)
	}
}

func TestGivenANegativeBalanceOnAnAccountThatIsNotACardWhenComputingThePoolThenItIsFlaggedNotTakenOff(t *testing.T) {
	accounts := append(poolAccounts(),
		Account{Name: "Wallet F", Type: AccountTypeRarelyUse, Currency: CurrencyVND, Balance: -1_000_000},
		Account{Name: "Old card G", Type: AccountTypeDeprecated, Currency: CurrencyVND, Balance: -2_000_000},
	)

	pool := ComputeGoalPool(accounts, monthlyHistory(), DefaultGoalPoolSettings(), nil, planToday)

	if pool.CardDebt != 12_000_000 || pool.Total != 128_000_000 {
		t.Fatalf("card debt = %v, pool = %v: only cards and old cards are debt", pool.CardDebt, pool.Total)
	}
	if pool.NegativeBalances != 1_000_000 {
		t.Fatalf("negative balances = %v", pool.NegativeBalances)
	}
	roles := map[string]PoolRole{}
	for _, account := range pool.Accounts {
		roles[account.Name] = account.Role
	}
	if roles["Wallet F"] != PoolRoleNegative || roles["Old card G"] != PoolRoleCardDebt || roles["Card E"] != PoolRoleCardDebt {
		t.Fatalf("roles = %v", roles)
	}
}
