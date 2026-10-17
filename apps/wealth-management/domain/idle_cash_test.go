package domain

import (
	"math"
	"strings"
	"testing"
)

// idleAccounts hold 150M of spendable cash: a 100M bank account earning 0.5%
// by its note, a 40M one with no rate and 10M cash. Long holdings, crypto
// and card debt are not spendable cash.
func idleAccounts() []Account {
	return []Account{
		{Name: "Bank A", Type: AccountTypeBank, Balance: 100_000_000, Note: note("Earn rate 0.5%")},
		{Name: "Bank B", Type: AccountTypeActiveUse, Balance: 40_000_000},
		{Name: "Wallet", Type: AccountTypeCash, Balance: 10_000_000},
		{Name: "Deposit", Type: AccountTypeLongHolding, Balance: 500_000_000, Note: note("lãi 6%")},
		{Name: "Card", Type: AccountTypeNegativeActive, Balance: -2_000_000},
	}
}

// idleBills owe 8M of bills (2M of them overdue) and a 3M card payment in
// the next 30 days; a paid bill and an income are not counted.
func idleBills() Bills {
	return Bills{
		Calendar: BillCalendar{Occurrences: []BillOccurrence{
			{Name: "Rent", Kind: RecurringBill, Source: BillFromRecurring, Status: BillDue, Remaining: 6_000_000},
			{Name: "Power", Kind: RecurringBill, Source: BillFromRecurring, Status: BillOverdue, Remaining: 2_000_000},
			{Name: "Card", Kind: RecurringCard, Source: BillFromCard, Status: BillDue, Remaining: 3_000_000},
			{Name: "Water", Kind: RecurringBill, Source: BillFromRecurring, Status: BillPaid, Remaining: 0, PaidAmount: 500_000},
			{Name: "Salary", Kind: RecurringIncome, Source: BillFromRecurring, Status: BillDue, Remaining: 20_000_000},
		}},
	}
}

// idlePlan sets 5M aside for goals this month and keeps a 10M spending
// buffer and a 30M emergency fund target.
func idlePlan() *GoalPlan {
	return &GoalPlan{
		Check:    PlanCheck{RequiredPerMonth: 5_000_000},
		Settings: GoalPoolSettings{EmergencyMonths: 6, SpendingBufferMonths: 1},
		Pool:     GoalPool{SpendingBuffer: 10_000_000, BufferMonths: 1, AverageMonthlyExpense: 10_000_000},
		Needs:    MonthlyNeeds{PerMonth: 4_000_000, Months: 6},
		Goals: []GoalProgress{
			{PlannedGoal: PlannedGoal{Name: "Emergency fund", Status: GoalActive}, EmergencyFund: true, TargetAmount: 30_000_000},
			{PlannedGoal: PlannedGoal{Name: "Trip", Status: GoalActive}, TargetAmount: 50_000_000},
		},
	}
}

func idleInput() IdleCashInput {
	return IdleCashInput{
		Accounts: idleAccounts(),
		Bills:    idleBills(),
		Plan:     idlePlan(),
		Rates:    []BankRate{{Bank: "Bank X", Rate: 5.5}, {Bank: "Bank Y", Rate: 4.8}},
		Today:    billsToday,
	}
}

func rateClose(a, b float64) bool { return math.Abs(a-b) < 0.001 }

func TestGivenSpendableCashWhenCheckingIdleCashThenWhatMustStayLiquidIsTakenOffWithEveryComponent(t *testing.T) {
	idle := ComputeIdleCash(idleInput())

	c := idle.Components
	if c.SpendableCash != 150_000_000 || c.BillsDue != 8_000_000 || c.CardPayments != 3_000_000 ||
		c.GoalSetAside != 5_000_000 || c.SpendingBuffer != 10_000_000 || c.EmergencyFund != 30_000_000 {
		t.Fatalf("components = %+v", c)
	}
	if !c.EmergencyFromGoal || c.BufferMonths != 1 || c.AverageMonthlyExpense != 10_000_000 {
		t.Fatalf("emergency and buffer sources = %+v", c)
	}
	if c.KeepLiquid != 56_000_000 || idle.Idle != 94_000_000 {
		t.Fatalf("keep liquid = %v, idle = %v", c.KeepLiquid, idle.Idle)
	}
	if len(idle.Accounts) != 3 {
		t.Fatalf("only spendable cash accounts are listed: %+v", idle.Accounts)
	}
}

func TestGivenIdleCashWhenWeightingTheCurrentRateThenTheLowestYieldAccountsHoldItFirst(t *testing.T) {
	idle := ComputeIdleCash(idleInput())

	// 94M idle: 40M of Bank B and 10M of the wallet at 0%, then 44M of Bank A at 0.5%.
	byName := map[string]IdleCashAccount{}
	for _, account := range idle.Accounts {
		byName[account.Name] = account
	}
	if byName["Bank B"].Idle != 40_000_000 || byName["Wallet"].Idle != 10_000_000 || byName["Bank A"].Idle != 44_000_000 {
		t.Fatalf("idle by account = %+v", idle.Accounts)
	}
	if !byName["Bank A"].RateFromNote || byName["Bank A"].Rate != 0.5 || byName["Bank B"].RateFromNote {
		t.Fatalf("rates = %+v", idle.Accounts)
	}
	if idle.Accounts[len(idle.Accounts)-1].Name != "Bank A" {
		t.Fatalf("accounts are listed lowest rate first: %+v", idle.Accounts)
	}
	if want := 44_000_000 * 0.5 / 94_000_000; !rateClose(idle.CurrentRate, math.Round(want*100)/100) {
		t.Fatalf("current rate = %v, want %v", idle.CurrentRate, want)
	}
}

func TestGivenABetterPublishedRateWhenCheckingIdleCashThenTheExtraInterestMakesItWorthMoving(t *testing.T) {
	idle := ComputeIdleCash(idleInput())

	if idle.Best == nil || idle.Best.Bank != "Bank X" || idle.Best.Rate != 5.5 || !strings.Contains(idle.Best.Basis, "proxy") {
		t.Fatalf("best = %+v", idle.Best)
	}
	// 94M x (5.5% - 0.23%) a year.
	want := math.Round(94_000_000 * (5.5 - idle.CurrentRate) / 100)
	if idle.ExtraPerYear != want || idle.Verdict != IdleCashWorthMoving {
		t.Fatalf("extra = %v (want %v), verdict = %s", idle.ExtraPerYear, want, idle.Verdict)
	}
	if len(idle.Reasons) == 0 || idle.RateRule == "" || idle.ThresholdRule == "" || idle.KeepLiquidRule == "" {
		t.Fatalf("reasons and rules must be returned: %+v", idle)
	}
}

func TestGivenKeepLiquidAboveSpendableCashWhenCheckingIdleCashThenNothingIsIdle(t *testing.T) {
	input := idleInput()
	input.Accounts = []Account{{Name: "Bank A", Type: AccountTypeBank, Balance: 50_000_000}}

	idle := ComputeIdleCash(input)
	if idle.Idle != 0 || idle.Verdict != IdleCashNothingIdle || idle.ExtraPerYear != 0 || idle.CurrentRate != 0 {
		t.Fatalf("idle = %+v", idle)
	}
	if idle.Accounts[0].Idle != 0 || len(idle.Reasons) == 0 {
		t.Fatalf("idle = %+v", idle)
	}
}

func TestGivenASmallGapWhenCheckingIdleCashThenItIsNotWorthMoving(t *testing.T) {
	t.Run("rate gap under half a point", func(t *testing.T) {
		input := idleInput()
		input.Accounts[1].Note = note("rate 5.2%")
		input.Accounts[2].Balance = 0
		input.Accounts[0].Note = note("Interest: 5,3 %/year")
		idle := ComputeIdleCash(input)
		if idle.Verdict != IdleCashNotWorthIt || idle.ExtraPerYear <= 0 {
			t.Fatalf("verdict = %s, extra = %v, rate = %v", idle.Verdict, idle.ExtraPerYear, idle.CurrentRate)
		}
	})
	t.Run("extra interest under the floor", func(t *testing.T) {
		input := idleInput()
		input.Accounts = []Account{{Name: "Bank A", Type: AccountTypeBank, Balance: 59_000_000}}
		idle := ComputeIdleCash(input)
		// 3M idle x 5.5% = 165,000 a year.
		if idle.Idle != 3_000_000 || idle.ExtraPerYear != 165_000 || idle.Verdict != IdleCashNotWorthIt {
			t.Fatalf("idle = %v, extra = %v, verdict = %s", idle.Idle, idle.ExtraPerYear, idle.Verdict)
		}
	})
	t.Run("current rate above the best", func(t *testing.T) {
		input := idleInput()
		for i := range input.Accounts {
			input.Accounts[i].Note = note("lai suat 7%")
		}
		idle := ComputeIdleCash(input)
		if idle.Verdict != IdleCashNotWorthIt || idle.ExtraPerYear != 0 || idle.RateGapPoints >= 0 {
			t.Fatalf("idle = %+v", idle)
		}
	})
}

func TestGivenNoPublishedRateWhenCheckingIdleCashThenTheVerdictSaysSoWithTheIssue(t *testing.T) {
	input := idleInput()
	input.Rates = nil
	input.RatesIssue = "bank rates unavailable: offline"

	idle := ComputeIdleCash(input)
	if idle.Best != nil || idle.Verdict != IdleCashNoRate || idle.RatesIssue != input.RatesIssue || idle.Idle != 94_000_000 {
		t.Fatalf("idle = %+v", idle)
	}
	if idle.ExtraPerYear != 0 {
		t.Fatalf("no extra without a rate: %v", idle.ExtraPerYear)
	}
}

func TestGivenNoEmergencyGoalWhenCheckingIdleCashThenTheTargetComesFromTheSettings(t *testing.T) {
	input := idleInput()
	input.Plan.Goals = input.Plan.Goals[1:]

	idle := ComputeIdleCash(input)
	// 6 months x 4M of needs.
	if idle.Components.EmergencyFund != 24_000_000 || idle.Components.EmergencyFromGoal {
		t.Fatalf("components = %+v", idle.Components)
	}

	input.Plan = nil
	idle = ComputeIdleCash(input)
	if idle.Components.EmergencyFund != 0 || idle.Components.SpendingBuffer != 0 || idle.Components.GoalSetAside != 0 || idle.Idle != 139_000_000 {
		t.Fatalf("without a plan nothing is kept for it: %+v", idle.Components)
	}
}

func TestAccountRateFromNote(t *testing.T) {
	cases := []struct {
		note string
		rate float64
		ok   bool
	}{
		{"Earn rate 4%", 4, true},
		{"lãi 4%", 4, true},
		{"LÃI SUẤT 4,5%/năm", 4.5, true},
		{"rate 4.5%", 4.5, true},
		{"interest: 3.2 %", 3.2, true},
		{"APY 1%", 1, true},
		{"Salary account, rate 0.1% a year", 0.1, true},
		{"rate 4", 0, false},
		{"fee 2%", 0, false},
		{"rate 150%", 0, false},
		{"", 0, false},
	}
	for _, c := range cases {
		rate, ok := AccountRateFromNote(c.note)
		if ok != c.ok || !rateClose(rate, c.rate) {
			t.Errorf("%q: got %v %v, want %v %v", c.note, rate, ok, c.rate, c.ok)
		}
	}
}

func TestIdleCashFactsAreNameless(t *testing.T) {
	idle := ComputeIdleCash(idleInput())
	facts := idle.Facts()
	if facts.Idle != idle.Idle || facts.KeepLiquid != idle.Components.KeepLiquid || facts.BestRate != 5.5 ||
		facts.ExtraPerYear != idle.ExtraPerYear || facts.Verdict != IdleCashWorthMoving || facts.AccountsHoldingIdle != 3 {
		t.Fatalf("facts = %+v", facts)
	}
}

func TestGroupThousandsUsesVietnameseSeparators(t *testing.T) {
	if got := groupThousands(1_234_567); got != "1.234.567" {
		t.Fatalf("got %s", got)
	}
	if got := groupThousands(-200_000); got != "-200.000" {
		t.Fatalf("got %s", got)
	}
	if !strings.Contains(IdleCashThresholdRule, "200.000 VND") {
		t.Fatal(IdleCashThresholdRule)
	}
}

func TestGivenSafeToSpendLeavesOutCryptoAndSlowAccountsWhenCheckingIdleCashThenNothingChanges(t *testing.T) {
	before := ComputeIdleCash(idleInput())

	input := idleInput()
	bills := ComputeBills(BillsInput{
		Accounts:  input.Accounts,
		Readiness: SafeToSpendSettings{NotReadyAccounts: []string{"Bank A"}},
		Today:     billsToday,
	})
	if bills.SafeToSpend.ReadyMoney != 50_000_000 {
		t.Fatalf("safe-to-spend leaves the slow account out: %+v", bills.SafeToSpend)
	}
	bills.Calendar = input.Bills.Calendar
	input.Bills = bills
	after := ComputeIdleCash(input)

	// Idle cash keeps its own spendable cash (the liquid balance) and takes
	// the goal set-aside from the goal plan, not from safe-to-spend.
	if after.Components != before.Components || after.Idle != before.Idle || len(after.Accounts) != 3 {
		t.Fatalf("idle cash changed: before %+v after %+v", before.Components, after.Components)
	}
	if after.Components.SpendableCash != 150_000_000 || after.Components.GoalSetAside != 5_000_000 {
		t.Fatalf("components = %+v", after.Components)
	}
}

func TestGivenAGoalPlanNotSetUpWhenCheckingIdleCashThenNothingIsSetAsideForGoals(t *testing.T) {
	input := idleInput()
	input.Plan.SetupNeeded = true
	if c := ComputeIdleCash(input).Components; c.GoalSetAside != 0 {
		t.Fatalf("components = %+v", c)
	}
}
