package domain

import (
	"reflect"
	"testing"
	"time"
)

func codes(from, to int) []MCCRange { return []MCCRange{{From: from, To: to}} }

func goldProgram() Program {
	return Program{Name: "Gold", Bank: "Acme", Versions: []ProgramVersion{{
		From: day(2025, 1, 1), AnnualFee: 599_000, Cap: 300_000, Period: CapPeriodCycle,
		Rules: []CashbackRule{
			{Name: "Utilities", MCCs: codes(4900, 4900), Exclude: true},
			{Name: "Groceries", MCCs: codes(5411, 5411), Rate: 5, MinTransaction: 100_000, Cap: 200_000},
			{Name: "Groceries tier 2", MCCs: codes(5411, 5411), Rate: 10, MinTransaction: 100_000, MinCycleSpend: 5_000_000, Cap: 200_000},
			{Name: "Dining", MCCs: codes(5812, 5814), Rate: 3},
			{Name: "Others", Rate: 0.5, Fallback: true},
			{Name: "Others large", Rate: 1, MinTransaction: 1_000_000, Fallback: true},
		},
	}}}
}

func silverProgram() Program {
	return Program{Name: "Silver", Bank: "Acme", Versions: []ProgramVersion{{
		From: day(2025, 1, 1), Period: CapPeriodMonth,
		Rules: []CashbackRule{
			{Name: "Online", MCCs: codes(5311, 5311), Rate: 5, Cap: 600_000},
			{Name: "Others", Rate: 0.5, Fallback: true},
		},
	}}}
}

func platinumProgram() Program {
	return Program{Name: "Platinum", Bank: "Beta", Versions: []ProgramVersion{{
		From: day(2026, 1, 1), AnnualFee: 1_000_000, Period: CapPeriodMonth,
		Rules: []CashbackRule{{Name: "Everything", Rate: 2, Fallback: true}},
	}}}
}

func testCardConfig() CardConfig {
	return CardConfig{
		Programs: []Program{goldProgram(), silverProgram(), platinumProgram()},
		Banks:    []Bank{{Name: "Acme", SharedAccount: "Credit Card Acme", ReferenceLimit: 20_000_000}},
		Cards: []Card{
			{Name: "Gold Card", Program: "Gold", Keyword: "gold", StatementDay: 5, DueDay: 30},
			{Name: "Silver Card", Program: "Silver", Keyword: "silver", StatementDay: 5, DueDay: 30, Default: true},
		},
		Merchants: []MerchantMCC{
			{Pattern: "Coop", MCC: 5411, Confidence: MCCConfirmed},
			{Pattern: "Coop Food", MCC: 5812, Confidence: MCCAssumed},
			{Pattern: "EVN", MCC: 4900, Confidence: MCCConfirmed},
			{Pattern: "Shopee", MCC: 5311, Confidence: MCCAssumed},
		},
	}
}

// cardTxn is a payment on the Acme shared account.
func cardTxn(date time.Time, payment float64, payee string, tags ...string) Transaction {
	t := txn(date, "Shopping", CategoryTypeExpense, payment, 0)
	t.AccountName = "Credit Card Acme"
	t.Payee = payee
	t.Tags = tags
	return t
}

func cashbackRefund(date time.Time, amount float64, month string, tags ...string) Transaction {
	t := txn(date, "Refunds/Reimbursements", CategoryTypeIncome, 0, amount)
	t.AccountName = "Credit Card Acme"
	t.Payee = "Acme Cashback"
	t.Memo = text(month)
	t.Tags = tags
	return t
}

func TestPaymentCashbackPicksTheRule(t *testing.T) {
	gold := goldProgram().Versions[0]
	confirmed := func(code int) PaymentMCC { return PaymentMCC{Code: code, Status: MCCConfirmed} }
	tests := []struct {
		name        string
		amount      float64
		mcc         PaymentMCC
		periodSpend float64
		cashback    float64
		rule        string
	}{
		{"mcc rule", 200_000, confirmed(5411), 1_000_000, 10_000, "Groceries"},
		{"an assumed mcc matches too", 200_000, PaymentMCC{Code: 5411, Status: MCCAssumed}, 1_000_000, 10_000, "Groceries"},
		{"period spend reaches the higher tier", 200_000, confirmed(5411), 6_000_000, 20_000, "Groceries tier 2"},
		{"below the minimum transaction falls back", 50_000, confirmed(5411), 6_000_000, 250, "Others"},
		{"mcc inside a range", 1_000_000, confirmed(5813), 0, 30_000, "Dining"},
		{"excluded mcc earns nothing", 2_000_000, confirmed(4900), 0, 0, "Utilities"},
		{"most demanding qualifying fallback", 2_000_000, PaymentMCC{Status: MCCUnknown}, 0, 20_000, "Others large"},
		{"unknown mcc uses the fallback", 200_000, PaymentMCC{Status: MCCUnknown}, 0, 1_000, "Others"},
		{"unmatched mcc uses the fallback", 200_000, confirmed(7011), 0, 1_000, "Others"},
		{"no payment earns nothing", 0, confirmed(5411), 0, 0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cashback, rule := gold.PaymentCashback(tt.amount, tt.mcc, tt.periodSpend)
			if cashback != tt.cashback || rule != tt.rule {
				t.Fatalf("got %v %q want %v %q", cashback, rule, tt.cashback, tt.rule)
			}
		})
	}
}

func payments(merchants []MerchantMCC, transactions ...Transaction) []cardPayment {
	result := make([]cardPayment, 0, len(transactions))
	for i, t := range transactions {
		result = append(result, cardPayment{index: i, tx: t, amount: spendAmount(t), mcc: TransactionMCC(t, merchants)})
	}
	return result
}

func TestEvaluatePeriodSplitsConfirmedAndEstimated(t *testing.T) {
	config := testCardConfig()
	date := day(2026, 3, 6)
	result := evaluatePeriod(periodGroup{version: goldProgram().Versions[0], payments: payments(config.Merchants,
		cardTxn(date, 1_000_000, "Coop"),      // confirmed groceries: 50k
		cardTxn(date, 1_000_000, "Coop Food"), // assumed dining: 30k
		cardTxn(date, 200_000, "Corner shop"), // unknown, fallback: 1k
		cardTxn(date, 3_000_000, "EVN Hanoi"), // excluded
	)})
	if want := newCashbackAmount(50_000, 81_000); result.cashback != want {
		t.Fatalf("cashback %+v want %+v", result.cashback, want)
	}
	if result.spent != 5_200_000 || result.qualifyingSpend != 2_200_000 || result.unconfirmed != 2 {
		t.Fatalf("spent %v qualifying %v unconfirmed %d", result.spent, result.qualifyingSpend, result.unconfirmed)
	}
	if !reflect.DeepEqual(result.paymentRules, []string{"Groceries", "Dining", "Others", "Utilities"}) {
		t.Fatalf("rules %v", result.paymentRules)
	}
}

func TestEvaluatePeriodCapsTheConfirmedPartAndTheTotal(t *testing.T) {
	merchants := []MerchantMCC{{Pattern: "Coop", MCC: 5411, Confidence: MCCConfirmed}, {Pattern: "Mart", MCC: 5411}, {Pattern: "Pho", MCC: 5812}}
	date := day(2026, 3, 6)
	result := evaluatePeriod(periodGroup{version: goldProgram().Versions[0], payments: payments(merchants,
		cardTxn(date, 3_000_000, "Coop"), // 150k confirmed
		cardTxn(date, 1_900_000, "Mart"), // 95k assumed; the rule caps at 200k
	)})
	groceries := result.rules[0]
	if groceries.Name != "Groceries" || groceries.Earned != newCashbackAmount(150_000, 200_000) || groceries.CapLeft != 0 {
		t.Fatalf("groceries %+v", groceries)
	}
	result = evaluatePeriod(periodGroup{version: goldProgram().Versions[0], payments: payments(merchants,
		cardTxn(date, 3_000_000, "Coop"), // tier 2 reached: 300k, capped at 200k
		cardTxn(date, 9_000_000, "Pho"),  // 270k assumed; the card caps at 300k
	)})
	if result.cashback != newCashbackAmount(200_000, 300_000) {
		t.Fatalf("card cap: %+v", result.cashback)
	}
}

func TestEvaluatePeriodSharesTheCapOfACapGroup(t *testing.T) {
	version := ProgramVersion{From: day(2026, 1, 1), Period: CapPeriodCycle, Rules: []CashbackRule{
		{Name: "Supermarket", MCCs: codes(5411, 5411), Rate: 6.8, Cap: 680_000, CapGroup: "Supermarket & transport"},
		{Name: "Transport", MCCs: codes(4121, 4121), Rate: 16.8, Cap: 680_000, CapGroup: "Supermarket & transport"},
		{Name: "Others", Rate: 0.39, Fallback: true},
	}}
	merchants := []MerchantMCC{{Pattern: "Coop", MCC: 5411, Confidence: MCCConfirmed}, {Pattern: "Grab", MCC: 4121}}
	date := day(2026, 9, 6)
	result := evaluatePeriod(periodGroup{version: version, payments: payments(merchants,
		cardTxn(date, 5_000_000, "Coop"),   // 340k confirmed
		cardTxn(date, 3_000_000, "Grab"),   // 504k assumed; the group caps both at 680k
		cardTxn(date, 10_000_000, "Hotel"), // 39k, outside the group, uncapped
	)})
	if result.cashback != newCashbackAmount(340_000, 719_000) {
		t.Fatalf("cashback %+v", result.cashback)
	}
	want := []RuleStanding{
		{Name: "Supermarket & transport", Rules: []string{"Supermarket", "Transport"}, Earned: newCashbackAmount(340_000, 680_000), Cap: 680_000},
		{Name: "Others", Earned: newCashbackAmount(0, 39_000)},
	}
	if !reflect.DeepEqual(result.rules, want) {
		t.Fatalf("standings %+v", result.rules)
	}
	if !reflect.DeepEqual(result.paymentRules, []string{"Supermarket", "Transport", "Others"}) {
		t.Fatalf("payments keep their own rule: %v", result.paymentRules)
	}
}

func TestEvaluatePeriodPicksTiersFromThePeriodSpendAndNamesTheNextTier(t *testing.T) {
	merchants := testCardConfig().Merchants
	date := day(2026, 3, 6)
	below := evaluatePeriod(periodGroup{version: goldProgram().Versions[0], payments: payments(merchants,
		cardTxn(date, 4_000_000, "Coop"),
		cardTxn(date, 9_000_000, "EVN"), // excluded spend does not reach the tier
	)})
	if below.cashback.Total != 200_000 || below.paymentRules[0] != "Groceries" {
		t.Fatalf("below the tier: %+v %v", below.cashback, below.paymentRules)
	}
	if tier := below.nextTier(); tier == nil || tier.SpendNeeded != 1_000_000 || !reflect.DeepEqual(tier.Rules, []string{"Groceries tier 2"}) {
		t.Fatalf("next tier %+v", tier)
	}
	above := evaluatePeriod(periodGroup{version: goldProgram().Versions[0], payments: payments(merchants,
		cardTxn(date, 1_000_000, "Coop"),
		cardTxn(date, 4_000_000, "Coop"),
	)})
	if above.paymentRules[0] != "Groceries tier 2" || above.cashback.Total != 200_000 || above.nextTier() != nil {
		t.Fatalf("above the tier: %+v %v", above.cashback, above.paymentRules)
	}
}

func TestGroupByPeriodUsesTheVersionInEffectOnEachPayment(t *testing.T) {
	program := silverProgram()
	revised := program.Versions[0]
	revised.From = day(2026, 3, 10)
	revised.Rules = []CashbackRule{{Name: "Online", MCCs: codes(5311, 5311), Rate: 2}}
	program.Versions = append(program.Versions, revised)
	merchants := testCardConfig().Merchants
	groups := groupByPeriod(payments(merchants,
		cardTxn(day(2026, 2, 20), 1_000_000, "Shopee"),
		cardTxn(day(2026, 3, 6), 1_000_000, "Shopee"),
		cardTxn(day(2026, 3, 12), 1_000_000, "Shopee"),
	), program.VersionOn, 5)
	var earned []float64
	for _, group := range groups {
		earned = append(earned, evaluatePeriod(group).cashback.Total)
	}
	if !reflect.DeepEqual(earned, []float64{50_000, 50_000, 20_000}) || !groups[1].start.Equal(groups[2].start) {
		t.Fatalf("earned %v", earned)
	}
}
