package domain

import (
	"testing"
)

func allocationAccounts() []Account {
	return []Account{
		{Name: "Bank A", Type: AccountTypeActiveUse, Balance: 40_000_000},
		{Name: "Savings", Type: AccountTypeRarelyUse, Balance: 20_000_000},
		{Name: "Card", Type: AccountTypeNegativeActive, Balance: -5_000_000},
		{Name: "Exchange", Type: AccountTypeActiveUse, Currency: CurrencyUSDT, Balance: 10_000_000},
		{Name: "Old", Type: AccountTypeDeprecated, Balance: 9_000_000},
	}
}

func TestGivenAccountsHoldingsAndMoneyLentWhenComputingAllocationThenEachClassHasItsValueAndShare(t *testing.T) {
	transactions := []Transaction{lendingTxn("row-2", "2026-01-02", "Person A", 20_000_000, 0)}
	allocation := ComputeAllocation(AllocationInputs{
		Accounts: allocationAccounts(), Transactions: transactions,
		Funds: 30_000_000, CryptoHoldings: 12_000_000, Unvalued: 1,
	})

	// cash 60M (card debt not netted, archived left out); crypto max(10M, 12M) = 12M; funds 30M
	want := map[AllocationClass]float64{AllocationCash: 60_000_000, AllocationFunds: 30_000_000, AllocationCrypto: 12_000_000, AllocationMoneyLent: 20_000_000}
	for class, value := range want {
		if got := allocation.Line(class).Value; got != value {
			t.Errorf("%s = %v, want %v", class, got, value)
		}
	}
	if allocation.TargetedTotal != 102_000_000 || allocation.Total != 122_000_000 || allocation.Unvalued != 1 {
		t.Fatalf("totals %+v", allocation)
	}
	lent := allocation.Line(AllocationMoneyLent)
	if lent.Targeted || lent.Current != nil || lent.Status != AllocationNoTarget || lent.Share != 16.39 {
		t.Fatalf("money lent line %+v", lent)
	}
	cash := allocation.Line(AllocationCash)
	if cash.Current == nil || *cash.Current != 58.82 || cash.Share != 49.18 || cash.Target != nil || cash.Status != AllocationNoTarget {
		t.Fatalf("cash line %+v", cash)
	}
	if allocation.Rebalance || allocation.DriftThreshold != DriftThreshold {
		t.Fatalf("no targets must not ask to rebalance: %+v", allocation)
	}
}

func TestGivenTargetsWhenAClassDriftsBeyondTheThresholdThenRebalanceIsSuggested(t *testing.T) {
	targets := &AllocationTargets{Cash: 50, Funds: 40, Crypto: 10}
	transactions := []Transaction{lendingTxn("row-2", "2026-01-02", "Person A", 50_000_000, 0)}
	// targeted total 100M: cash 60%, funds 30%, crypto 10%; money lent is out of drift
	allocation := ComputeAllocation(AllocationInputs{
		Accounts:     []Account{{Name: "Bank", Type: AccountTypeBank, Balance: 60_000_000}, {Name: "Coins", Type: AccountTypeCrypto, Balance: 10_000_000}},
		Transactions: transactions, Funds: 30_000_000, Targets: targets,
	})
	cases := map[AllocationClass]struct {
		drift  float64
		status AllocationStatus
	}{
		AllocationCash:   {10, AllocationOver},
		AllocationFunds:  {-10, AllocationUnder},
		AllocationCrypto: {0, AllocationOnTarget},
	}
	for class, want := range cases {
		line := allocation.Line(class)
		if line.Drift == nil || *line.Drift != want.drift || line.Status != want.status || line.Target == nil {
			t.Errorf("%s line %+v, want drift %v %s", class, line, want.drift, want.status)
		}
	}
	if !allocation.Rebalance || allocation.Targets != targets {
		t.Fatalf("expected a rebalance with the targets: %+v", allocation)
	}
	if lent := allocation.Line(AllocationMoneyLent); lent.Drift != nil || lent.Target != nil {
		t.Fatalf("money lent must have no target: %+v", lent)
	}
}

func TestGivenDriftAtExactlyTheThresholdWhenComputingAllocationThenItIsOnTarget(t *testing.T) {
	allocation := ComputeAllocation(AllocationInputs{
		Accounts: []Account{{Name: "Bank", Type: AccountTypeBank, Balance: 55}},
		Funds:    45,
		Targets:  &AllocationTargets{Cash: 50, Funds: 50, Crypto: 0},
	})
	if allocation.Rebalance || allocation.Line(AllocationCash).Status != AllocationOnTarget {
		t.Fatalf("5 points is within the threshold: %+v", allocation)
	}
}

func TestGivenNothingToAllocateWhenComputingAllocationThenSharesAreZeroAndNoRebalance(t *testing.T) {
	allocation := ComputeAllocation(AllocationInputs{Targets: &AllocationTargets{Cash: 100}})
	if allocation.Total != 0 || allocation.Rebalance || len(allocation.Lines) != 4 {
		t.Fatalf("empty allocation %+v", allocation)
	}
	if line := allocation.Line(AllocationCash); line.Share != 0 || *line.Current != 0 {
		t.Fatalf("cash line %+v", line)
	}
}

func TestGivenAValuationWhenBuildingAllocationInputsThenHoldingsFiguresAndIssuesCarryOver(t *testing.T) {
	valuation := PortfolioValuation{Totals: PortfolioTotals{Funds: 7, CryptoHoldings: 3, Unvalued: 2}, Issues: []string{"USD/VND rate unavailable"}}
	in := AllocationInputsFromValuation(valuation)
	if in.Funds != 7 || in.CryptoHoldings != 3 || in.Unvalued != 2 || len(in.Issues) != 1 {
		t.Fatalf("inputs %+v", in)
	}
}

func TestGivenAccountsWhenSummingLiquidCashThenOnlySpendableAccountsCount(t *testing.T) {
	accounts := append(allocationAccounts(), Account{Name: "Long", Type: AccountTypeLongHolding, Balance: 70})
	// active 40M + rarely 20M + the USDT exchange account (active use) 10M
	if got := liquidBalance(accounts); got != 70_000_000 {
		t.Fatalf("liquid = %v", got)
	}
}
