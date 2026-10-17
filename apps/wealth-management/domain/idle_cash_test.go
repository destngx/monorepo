package domain

import (
	"math"
	"strings"
	"testing"
)

// idleAccounts hold 150M of ready money: a 100M bank account earning 0.5%
// by its note, a 40M one with no rate and 10M cash. Long holdings, card
// debt, crypto (D32) and an account marked not ready (D33) are not ready
// money, so never idle.
func idleAccounts() []Account {
	return []Account{
		{Name: "Bank A", Type: AccountTypeBank, Balance: 100_000_000, Note: note("Earn rate 0.5%")},
		{Name: "Bank B", Type: AccountTypeActiveUse, Balance: 40_000_000},
		{Name: "Wallet", Type: AccountTypeCash, Balance: 10_000_000},
		{Name: "Deposit", Type: AccountTypeLongHolding, Balance: 500_000_000, Note: note("lãi 6%")},
		{Name: "Card", Type: AccountTypeNegativeActive, Balance: -2_000_000},
		{Name: "Coin Wallet", Type: AccountTypeCrypto, Balance: 30_000_000},
		{Name: "Slow Bank", Type: AccountTypeBank, Balance: 25_000_000},
	}
}

// idleBills owe 8M of bills (2M of them overdue) and a 3M card payment in
// the next 30 days; a paid bill and an income are not counted. Slow Bank
// is marked not ready to use.
func idleBills() Bills {
	return Bills{
		Calendar: BillCalendar{Occurrences: []BillOccurrence{
			{Name: "Rent", Kind: RecurringBill, Source: BillFromRecurring, Status: BillDue, Remaining: 6_000_000},
			{Name: "Power", Kind: RecurringBill, Source: BillFromRecurring, Status: BillOverdue, Remaining: 2_000_000},
			{Name: "Card", Kind: RecurringCard, Source: BillFromCard, Status: BillDue, Remaining: 3_000_000},
			{Name: "Water", Kind: RecurringBill, Source: BillFromRecurring, Status: BillPaid, Remaining: 0, PaidAmount: 500_000},
			{Name: "Salary", Kind: RecurringIncome, Source: BillFromRecurring, Status: BillDue, Remaining: 20_000_000},
		}},
		SafeToSpend: SafeToSpend{Readiness: SafeToSpendSettings{NotReadyAccounts: []string{"slow bank"}}},
	}
}

// idlePlan sets 5M aside for goals this month and keeps a 10M spending
// buffer (the default day-to-day float) and a 30M emergency fund target.
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

// idleRates: the best instant-access option is Bank X's 1-month deposit;
// a 12-month and an untenored rate are higher but never count.
func idleRates() []BankRate {
	return []BankRate{
		{Bank: "Bank X", Tenor: "1m", TenorMonths: months(1), Rate: 5.5},
		{Bank: "Bank Y", Tenor: "no-term", TenorMonths: months(0), InstantAccess: true, Rate: 4.8},
		{Bank: "Bank Z", Tenor: "12m", TenorMonths: months(12), Rate: 7.2},
		{Bank: "Bank W", Rate: 7.5},
	}
}

func idleInput() IdleCashInput {
	return IdleCashInput{Accounts: idleAccounts(), Bills: idleBills(), Plan: idlePlan(), Rates: idleRates(), Today: billsToday}
}

func rateClose(a, b float64) bool { return math.Abs(a-b) < 0.001 }

func TestGivenReadyMoneyWhenCheckingIdleCashThenKeepLiquidAndTheDayToDayFloatAreTakenOff(t *testing.T) {
	idle := ComputeIdleCash(idleInput())

	c := idle.Components
	if c.ReadyMoney != 150_000_000 || c.BillsDue != 8_000_000 || c.CardPayments != 3_000_000 ||
		c.GoalSetAside != 5_000_000 || c.EmergencyFund != 30_000_000 || c.EmergencyFromReady != 30_000_000 {
		t.Fatalf("components = %+v", c)
	}
	if !c.EmergencyFromGoal || c.BufferMonths != 1 || c.AverageMonthlyExpense != 10_000_000 {
		t.Fatalf("emergency and float sources = %+v", c)
	}
	// Without a preference the float is the spending buffer: 1 month x 10M.
	if c.DayToDayFloat != 10_000_000 || c.FloatFromPreference {
		t.Fatalf("float = %+v", c)
	}
	// 150M - (8 + 3 + 5 + 30)M - 10M.
	if c.KeepLiquid != 46_000_000 || idle.Idle != 94_000_000 {
		t.Fatalf("keep liquid = %v, idle = %v", c.KeepLiquid, idle.Idle)
	}
	if len(idle.Accounts) != 3 {
		t.Fatalf("only ready money accounts are listed: %+v", idle.Accounts)
	}
}

func TestGivenCryptoAndANotReadyAccountWhenCheckingIdleCashThenTheyAreLeftOutAndNeverIdle(t *testing.T) {
	idle := ComputeIdleCash(idleInput())

	reasons := map[string]ExclusionReason{}
	for _, account := range idle.Excluded {
		reasons[account.Name] = account.Reason
	}
	if reasons["Coin Wallet"] != ExcludedCrypto || reasons["Slow Bank"] != ExcludedNotReady || len(reasons) != 2 {
		t.Fatalf("excluded = %+v", idle.Excluded)
	}
	for _, account := range idle.Accounts {
		if account.Name == "Coin Wallet" || account.Name == "Slow Bank" {
			t.Fatalf("%s must never be idle: %+v", account.Name, idle.Accounts)
		}
	}
}

func TestGivenADayToDayFloatPreferenceWhenCheckingIdleCashThenItReplacesTheDefault(t *testing.T) {
	input := idleInput()
	input.Settings.DayToDayFloat = number(30_000_000)

	idle := ComputeIdleCash(input)
	if idle.Components.DayToDayFloat != 30_000_000 || !idle.Components.FloatFromPreference || idle.Idle != 74_000_000 {
		t.Fatalf("components = %+v, idle = %v", idle.Components, idle.Idle)
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

func TestGivenABetter1MonthRateWhenCheckingIdleCashThenItIsWorthMovingAsALadder(t *testing.T) {
	idle := ComputeIdleCash(idleInput())

	if idle.Best == nil || idle.Best.Bank != "Bank X" || idle.Best.Rate != 5.5 || idle.Best.Access != DepositOneMonth || !strings.Contains(idle.Best.Basis, "proxy") {
		t.Fatalf("best = %+v", idle.Best)
	}
	// 94M x (5.5% - 0.23%) a year.
	want := math.Round(94_000_000 * (5.5 - idle.CurrentRate) / 100)
	if idle.ExtraPerYear != want || idle.Verdict != IdleCashWorthMoving {
		t.Fatalf("extra = %v (want %v), verdict = %s", idle.ExtraPerYear, want, idle.Verdict)
	}
	if idle.Ladder == nil || idle.Ladder.Parts != 3 || idle.Ladder.SpacingDays != 10 || idle.Ladder.PartAmount != math.Round(94_000_000.0/3) {
		t.Fatalf("ladder = %+v", idle.Ladder)
	}
	if len(idle.Reasons) == 0 || !strings.Contains(idle.Reasons[0], "1-month") || idle.RateRule == "" || idle.ThresholdRule == "" || idle.KeepLiquidRule == "" {
		t.Fatalf("reasons and rules must be returned: %+v", idle)
	}
}

func TestGivenANoTermRateAboveEvery1MonthRateWhenCheckingIdleCashThenNoLadderIsSuggested(t *testing.T) {
	input := idleInput()
	input.Rates = []BankRate{
		{Bank: "Bank X", Tenor: "1m", TenorMonths: months(1), Rate: 4.0},
		{Bank: "Bank Y", Tenor: "no-term", TenorMonths: months(0), InstantAccess: true, Rate: 4.5},
	}
	idle := ComputeIdleCash(input)
	if idle.Best == nil || idle.Best.Access != DepositNoTerm || idle.Best.Bank != "Bank Y" || idle.Ladder != nil {
		t.Fatalf("best = %+v ladder = %+v", idle.Best, idle.Ladder)
	}
}

func TestGivenASmallIdleAmountWhenTheBestIs1MonthThenNoLadderIsSplit(t *testing.T) {
	input := idleInput()
	input.Accounts = []Account{{Name: "Bank A", Type: AccountTypeBank, Balance: 71_000_000}}
	idle := ComputeIdleCash(input)
	// 15M idle: two parts of 7.5M are under the 10M part floor, so one deposit.
	if idle.Idle != 15_000_000 || idle.Ladder != nil {
		t.Fatalf("idle = %v ladder = %+v", idle.Idle, idle.Ladder)
	}
	input.Accounts[0].Balance = 81_000_000
	if idle = ComputeIdleCash(input); idle.Ladder == nil || idle.Ladder.Parts != 2 || idle.Ladder.SpacingDays != 15 {
		t.Fatalf("25M idle splits in two: %+v", idle.Ladder)
	}
}

func TestGivenOnlyUntenoredOrLongTermRatesWhenCheckingIdleCashThenTheVerdictIsNoRate(t *testing.T) {
	input := idleInput()
	input.Rates = []BankRate{{Bank: "Bank W", Rate: 7.5}, {Bank: "Bank Z", Tenor: "12m", TenorMonths: months(12), Rate: 7.2}}

	idle := ComputeIdleCash(input)
	if idle.Best != nil || idle.Verdict != IdleCashNoRate || idle.ExtraPerYear != 0 {
		t.Fatalf("an untenored or long-term rate must never be quoted: %+v", idle)
	}
	if !strings.Contains(idle.RatesIssue, "no-term or 1-month") || len(idle.Reasons) != 1 || !strings.Contains(idle.Reasons[0], "instant-access") {
		t.Fatalf("issue = %q reasons = %v", idle.RatesIssue, idle.Reasons)
	}
	if evaluated, findings := EvaluateRecommendationRules(RuleInputs{IdleCash: &idle}); len(evaluated) != 0 || len(findings) != 0 {
		t.Fatalf("the rule stays silent without an instant-access rate: %v %+v", evaluated, findings)
	}
}

func TestGivenKeepLiquidAboveReadyMoneyWhenCheckingIdleCashThenNothingIsIdle(t *testing.T) {
	input := idleInput()
	input.Accounts = []Account{{Name: "Bank A", Type: AccountTypeBank, Balance: 50_000_000}}

	idle := ComputeIdleCash(input)
	if idle.Idle != 0 || idle.Verdict != IdleCashNothingIdle || idle.ExtraPerYear != 0 || idle.CurrentRate != 0 || idle.Ladder != nil {
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
	c := idle.Components
	if c.EmergencyFund != 0 || c.DayToDayFloat != 0 || c.GoalSetAside != 0 || idle.Idle != 139_000_000 {
		t.Fatalf("without a plan nothing is kept for it: %+v", c)
	}
}

func TestGivenEmergencyAccountsWhenCheckingIdleCashThenTheyCoverTheTargetFirstAndAreNeverIdle(t *testing.T) {
	input := idleInput()
	input.Accounts = append(input.Accounts, Account{Name: "Rainy Day Bank", Type: AccountTypeBank, Balance: 20_000_000, Note: note("lãi 4%")})
	input.Settings.EmergencyAccounts = []string{"rainy day bank"}

	idle := ComputeIdleCash(input)
	c := idle.Components
	// 20M of the 30M target sits in the emergency account; ready money covers the other 10M.
	if c.EmergencyInAccounts != 20_000_000 || c.EmergencyCovered != 20_000_000 || c.EmergencyFromReady != 10_000_000 || c.ReadyMoney != 150_000_000 {
		t.Fatalf("components = %+v", c)
	}
	// 150M - (8 + 3 + 5 + 10)M - 10M.
	if c.KeepLiquid != 26_000_000 || idle.Idle != 114_000_000 {
		t.Fatalf("keep liquid = %v idle = %v", c.KeepLiquid, idle.Idle)
	}
	for _, account := range idle.Accounts {
		if account.Name == "Rainy Day Bank" {
			t.Fatalf("an emergency account is never idle: %+v", idle.Accounts)
		}
	}
	e := idle.Emergency
	if len(e.Accounts) != 1 || e.Accounts[0].Rate != 4 || e.Balance != 20_000_000 || e.CurrentRate != 4 {
		t.Fatalf("emergency = %+v", e)
	}
	// 5.5% - 4% = 1.5 points on 20M = 300.000 a year: past both thresholds.
	if !e.BetterRate || e.RateGapPoints != 1.5 || e.ExtraPerYear != 300_000 {
		t.Fatalf("emergency rate = %+v", e)
	}

	input.Accounts[len(input.Accounts)-1].Balance = 50_000_000
	idle = ComputeIdleCash(input)
	if idle.Components.EmergencyCovered != 30_000_000 || idle.Components.EmergencyFromReady != 0 || idle.Components.KeepLiquid != 16_000_000 {
		t.Fatalf("an emergency account above the target covers it all: %+v", idle.Components)
	}
}

func TestGivenAnEmergencyAccountMarkedNotReadyWhenCheckingIdleCashThenItStillCoversTheTarget(t *testing.T) {
	input := idleInput()
	input.Settings.EmergencyAccounts = []string{"Slow Bank"}

	idle := ComputeIdleCash(input)
	if idle.Components.EmergencyInAccounts != 25_000_000 || idle.Components.EmergencyFromReady != 5_000_000 {
		t.Fatalf("components = %+v", idle.Components)
	}
	for _, account := range idle.Excluded {
		if account.Name == "Slow Bank" {
			t.Fatalf("an emergency account is listed with the emergency fund, not as left out: %+v", idle.Excluded)
		}
	}
}

func TestGivenEmergencyNamesThatAreCryptoOrUnknownWhenCheckingIdleCashThenTheyAreReportedAndIgnored(t *testing.T) {
	input := idleInput()
	input.Settings.EmergencyAccounts = []string{"Coin Wallet", "Gone Bank"}

	idle := ComputeIdleCash(input)
	if idle.Components.EmergencyInAccounts != 0 || len(idle.Emergency.Accounts) != 0 || len(idle.Issues) != 2 {
		t.Fatalf("emergency = %+v issues = %v", idle.Emergency, idle.Issues)
	}
	if strings.Join(idle.Emergency.Unknown, "|") != "Gone Bank" || !strings.Contains(strings.Join(idle.Issues, " "), "crypto") {
		t.Fatalf("emergency = %+v issues = %v", idle.Emergency, idle.Issues)
	}
}

func TestGivenAnEmergencyAccountEarningNearTheBestWhenCheckingIdleCashThenNoBetterPlaceIsSuggested(t *testing.T) {
	input := idleInput()
	input.Accounts = append(input.Accounts, Account{Name: "Rainy Day Bank", Type: AccountTypeBank, Balance: 20_000_000, Note: note("rate 5.2%")})
	input.Settings.EmergencyAccounts = []string{"Rainy Day Bank"}
	if e := ComputeIdleCash(input).Emergency; e.BetterRate {
		t.Fatalf("a 0.3-point gap is under the threshold: %+v", e)
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
	input := idleInput()
	input.Accounts = append(input.Accounts, Account{Name: "Rainy Day Bank", Type: AccountTypeBank, Balance: 20_000_000, Note: note("lãi 4%")})
	input.Settings.EmergencyAccounts = []string{"Rainy Day Bank"}
	idle := ComputeIdleCash(input)
	facts := idle.Facts()
	if facts.Idle != idle.Idle || facts.ReadyMoney != 150_000_000 || facts.KeepLiquid != idle.Components.KeepLiquid || facts.DayToDayFloat != 10_000_000 ||
		facts.BestRate != 5.5 || facts.BestKind != DepositOneMonth || facts.LadderParts != 3 ||
		facts.ExtraPerYear != idle.ExtraPerYear || facts.Verdict != IdleCashWorthMoving || facts.AccountsHoldingIdle != 3 {
		t.Fatalf("facts = %+v", facts)
	}
	if facts.EmergencyAccounts != 1 || facts.EmergencyInAccounts != 20_000_000 || facts.EmergencyTarget != 30_000_000 || !facts.EmergencyBetterRate {
		t.Fatalf("emergency facts = %+v", facts)
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

func TestGivenSafeToSpendReadyMoneyWhenCheckingIdleCashThenBothUseTheSameAccounts(t *testing.T) {
	input := idleInput()
	bills := ComputeBills(BillsInput{Accounts: input.Accounts, Readiness: input.Bills.SafeToSpend.Readiness, Today: billsToday})
	idle := ComputeIdleCash(input)
	if bills.SafeToSpend.ReadyMoney != idle.Components.ReadyMoney || len(bills.SafeToSpend.ReadyAccounts) != len(idle.Accounts) {
		t.Fatalf("safe to spend %v (%d accounts), idle cash %v (%d accounts)",
			bills.SafeToSpend.ReadyMoney, len(bills.SafeToSpend.ReadyAccounts), idle.Components.ReadyMoney, len(idle.Accounts))
	}
}

func TestGivenAGoalPlanNotSetUpWhenCheckingIdleCashThenNothingIsSetAsideForGoals(t *testing.T) {
	input := idleInput()
	input.Plan.SetupNeeded = true
	if c := ComputeIdleCash(input).Components; c.GoalSetAside != 0 {
		t.Fatalf("components = %+v", c)
	}
}
