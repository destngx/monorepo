package domain

import (
	"reflect"
	"testing"
	"time"
)

// marchLedger is a month of spend on the Acme shared account; today is 20 March 2026.
func marchLedger() []Transaction {
	return []Transaction{
		cardTxn(day(2026, 3, 1), 1_000_000, "Coop", "gold"),         // gold, previous cycle
		cardTxn(day(2026, 3, 6), 1_000_000, "Coop"),                 // untagged: silver, 0.5% confirmed
		cardTxn(day(2026, 3, 6), 1_000_000, "Shopee"),               // untagged: silver online, assumed
		cardTxn(day(2026, 3, 7), 2_000_000, "Coop", "gold"),         // groceries 100k confirmed
		cardTxn(day(2026, 3, 8), 1_000_000, "Coop Food", "gold"),    // dining 30k assumed
		cardTxn(day(2026, 3, 9), 9_000_000, "EVN", "gold"),          // excluded
		cardTxn(day(2026, 3, 9), 300_000, "Corner shop", "gold"),    // unknown: fallback 1.5k
		cashbackRefund(day(2026, 3, 8), 250_000, "2026-02"),         // untagged refund: silver
		cardTxn(day(2026, 4, 1), 5_000_000, "Coop", "gold planned"), // after today
	}
}

func marchReports(t *testing.T, accounts []Account) (CardBankReport, CardReport, CardReport) {
	t.Helper()
	reports := ComputeCardReports(testCardConfig(), accounts, marchLedger(), day(2026, 3, 20))
	if len(reports) != 1 || len(reports[0].Cards) != 2 {
		t.Fatalf("expected one bank with two cards, got %+v", reports)
	}
	return reports[0], reports[0].Cards[0], reports[0].Cards[1]
}

func TestCardStandingInTheCurrentPeriod(t *testing.T) {
	_, gold, silver := marchReports(t, nil)
	standing := gold.Standing
	if standing.Start != "2026-03-05" || standing.End != "2026-04-04" || gold.CapPeriod != CapPeriodCycle || gold.VersionFrom != "2025-01-01" {
		t.Fatalf("period %+v", standing)
	}
	if standing.Spent != 17_300_000 || standing.QualifyingSpend != 8_300_000 || standing.PaymentCount != 5 || standing.UnconfirmedPayments != 2 {
		t.Fatalf("spend %+v", standing)
	}
	// The planned 1 April payment counts in this period: groceries reach tier 2.
	if standing.Cashback != newCashbackAmount(200_000, 231_500) || standing.Cap != 300_000 || standing.CapLeft != 68_500 {
		t.Fatalf("cashback %+v", standing)
	}
	if standing.NextTier != nil {
		t.Fatalf("no tier left, got %+v", standing.NextTier)
	}
	if silver.CapPeriod != CapPeriodMonth || silver.Standing.Start != "2026-03-01" || silver.Standing.Cashback != newCashbackAmount(5_000, 55_000) {
		t.Fatalf("silver %+v", silver.Standing)
	}
}

func TestCardStandingNamesTheNextTierAndCapsLeft(t *testing.T) {
	ledger := marchLedger()[:8]
	gold := ComputeCardReports(testCardConfig(), nil, ledger, day(2026, 3, 20))[0].Cards[0]
	standing := gold.Standing
	if standing.Cashback != newCashbackAmount(100_000, 131_500) || standing.NextTier == nil || standing.NextTier.SpendNeeded != 1_700_000 {
		t.Fatalf("standing %+v next %+v", standing, standing.NextTier)
	}
	groceries := standing.Rules[0]
	if groceries.Name != "Groceries" || groceries.Earned.Total != 100_000 || groceries.Cap != 200_000 || groceries.CapLeft != 100_000 {
		t.Fatalf("groceries %+v", groceries)
	}
	if len(gold.Transactions) != 4 || !gold.Transactions[0].Transaction.Date.Equal(day(2026, 3, 9)) {
		t.Fatalf("expected the period's payments newest first, got %+v", gold.Transactions)
	}
	for _, tx := range gold.Transactions {
		if tx.Transaction.Payee == "Coop Food" && (tx.Rule != "Dining" || tx.Cashback != 30_000 || tx.MCC.Status != MCCAssumed) {
			t.Fatalf("dining payment %+v", tx)
		}
	}
}

func TestBankLimitAndUsage(t *testing.T) {
	bank, gold, silver := marchReports(t, []Account{{
		Name: "Credit Card Acme", Type: AccountTypeNegativeActive, ClearedBalance: -8_000_000, Balance: -9_000_000,
	}})
	if bank.Limit != 20_000_000 || bank.Usage != 8_000_000 || bank.Utilization != 40 || bank.UtilizationBand != UtilizationFair || bank.RemainingLimit != 12_000_000 {
		t.Fatalf("bank %+v", bank)
	}
	if bank.Cashback != newCashbackAmount(205_000, 286_500) || bank.FeesPaidYTD != 0 || bank.NetEarnYTD != 250_000 {
		t.Fatalf("bank totals %+v", bank)
	}
	if gold.AccountName != "Credit Card Acme" || gold.Cycle.PayBy != "2026-03-29" {
		t.Fatalf("gold %+v", gold.Cycle)
	}
	if silver.RefundYTD != 250_000 || silver.NetEarnYTD != 250_000 || gold.RefundYTD != 0 {
		t.Fatalf("expected the untagged refund on silver, got %v / %v", silver.RefundYTD, gold.RefundYTD)
	}
}

func TestMoneyPaidInAheadIsRoomNotUsage(t *testing.T) {
	accounts := []Account{
		{Name: "Credit Card Acme", Type: AccountTypeNegativeActive, ClearedBalance: -3_000_000, CreditLimit: money(30_000_000)},
		{Name: "Card Gold", Type: AccountTypeNegativeActive, ClearedBalance: 2_000_000},
	}
	transactions := []Transaction{
		{AccountName: "Card Gold", Date: day(2026, 3, 6), Payment: money(100_000), Tags: []string{"silver"}}, // other card's tag
		{AccountName: "Card Gold", Date: day(2026, 3, 7), Payment: money(200_000)},
	}
	bank := ComputeCardReports(testCardConfig(), accounts, transactions, day(2026, 3, 20))[0]
	gold := bank.Cards[0]
	if bank.Limit != 30_000_000 || bank.Usage != 3_000_000 || bank.RemainingLimit != 27_000_000 {
		t.Fatalf("bank %+v", bank)
	}
	if gold.AccountName != "Card Gold" || gold.Standing.Spent != 200_000 || gold.LifetimeFees != 599_000 {
		t.Fatalf("gold %+v", gold)
	}
}

func TestCardHistoryReconcilesEachPeriodWithItsRefund(t *testing.T) {
	gold := func(date time.Time, payee string) Transaction {
		return cardTxn(date, 1_000_000, payee, "gold")
	}
	transactions := []Transaction{
		gold(day(2025, 12, 10), "Coop"),     // 50k, no refund
		gold(day(2026, 1, 10), "Coop"),      // 50k, refunded 50k
		gold(day(2026, 2, 10), "Coop"),      // 50k, refunded 30k
		gold(day(2026, 3, 10), "Coop Food"), // 30k estimated, refunded 40k
		gold(day(2026, 4, 10), "Coop"),      // current period
		cashbackRefund(day(2026, 2, 8), 50_000, "2026-01", "gold"),
		cashbackRefund(day(2026, 3, 8), 30_000, "2026-02", "gold"),
		cashbackRefund(day(2026, 4, 8), 40_000, "2026-03", "gold"),
	}
	history := ComputeCardReports(testCardConfig(), nil, transactions, day(2026, 4, 20))[0].Cards[0].History
	want := map[string]ReconciliationStatus{
		"2026-04": ReconciliationOpen, "2026-03": ReconciliationAbove, "2026-02": ReconciliationBelow,
		"2026-01": ReconciliationMatch, "2025-12": ReconciliationAwaiting,
	}
	if len(history) != len(want) || history[0].Month != "2026-04" {
		t.Fatalf("history %+v", history)
	}
	for _, period := range history {
		if period.Status != want[period.Month] {
			t.Fatalf("%s: status %q want %q", period.Month, period.Status, want[period.Month])
		}
	}
	january := history[3]
	if january.Start != "2026-01-05" || january.End != "2026-02-04" || january.Spent != 1_000_000 || january.Refund != 50_000 || january.Efficiency != 5 {
		t.Fatalf("january %+v", january)
	}
	if february := history[2]; february.Missed != 20_000 || february.Cashback != newCashbackAmount(50_000, 50_000) {
		t.Fatalf("february %+v", february)
	}
	if march := history[1]; march.Cashback != newCashbackAmount(0, 30_000) {
		t.Fatalf("march %+v", march)
	}
}

func TestBanksWithoutHeldCardsAreLeftOut(t *testing.T) {
	config := testCardConfig()
	config.Banks = append(config.Banks, Bank{Name: "Beta", ReferenceLimit: 1})
	if reports := ComputeCardReports(config, nil, nil, day(2026, 3, 20)); len(reports) != 1 {
		t.Fatalf("got %+v", reports)
	}
}

func TestAClosedCardKeepsItsHistoryButTakesNoNewPayments(t *testing.T) {
	config := testCardConfig()
	closed := day(2026, 3, 9)
	config.Cards[0].ClosedFrom = &closed
	// The refund for its last period arrives after the card is closed.
	transactions := append(marchLedger(),
		cashbackRefund(day(2026, 3, 12), 40_000, "2026-03", "gold"),
		cardFee(day(2026, 3, 2), 599_000, "gold"),
	)
	today := day(2026, 3, 20)

	bank := ComputeCardReports(config, nil, transactions, today)[0]
	gold, silver := bank.Cards[0], bank.Cards[1]
	if gold.ClosedFrom != "2026-03-09" || gold.Standing.PaymentCount != 0 || gold.Standing.Cashback.Total != 0 || len(gold.Transactions) != 0 || gold.Statement != nil {
		t.Fatalf("a closed card has no current period or statement to pay: %+v", gold)
	}
	if len(gold.History) != 2 || gold.History[0].Month != "2026-03" || gold.History[0].Spent != 3_000_000 || gold.History[0].Refund != 40_000 {
		t.Fatalf("payments before the closing date and later refunds stay: %+v", gold.History)
	}
	if gold.LifetimeFees != 599_000 || gold.FeesPaidYTD != 599_000 || bank.FeesPaidYTD != 599_000 {
		t.Fatalf("a card closed this year still paid this year's fee: %v / %v", gold.LifetimeFees, bank.FeesPaidYTD)
	}
	if silver.Standing.Spent != 2_000_000 {
		t.Fatalf("payments tagged for the closed card do not move to the default card: %+v", silver.Standing)
	}

	suggestions := SuggestCards(config, nil, transactions, CardPaymentQuery{Payee: "Coop", Amount: 1_000_000}, today)
	if len(suggestions) != 1 || suggestions[0].Card != "Silver Card" {
		t.Fatalf("a closed card is not suggested: %+v", suggestions)
	}
	for _, program := range ComparePrograms(config, nil, transactions, today).Programs {
		if program.Program == "Gold" && program.Held {
			t.Fatalf("a programme held only by a closed card is not held: %+v", program)
		}
	}

	nextYear := ComputeCardReports(config, nil, transactions, day(2027, 3, 20))[0]
	if nextYear.FeesPaidYTD != 0 || nextYear.Cards[0].LifetimeFees != 599_000 {
		t.Fatalf("fees stop at closing: %v / %v", nextYear.FeesPaidYTD, nextYear.Cards[0].LifetimeFees)
	}
}

// cardFee is the bank's annual fee on the Acme shared account, marked by its memo.
func cardFee(date time.Time, amount float64, tags ...string) Transaction {
	t := cardTxn(date, amount, "Acme", tags...)
	t.Category = "Bank fees"
	t.Memo = text("BankFee")
	return t
}

func TestNetEarnCountsTheAnnualFeeOnlyOncePaid(t *testing.T) {
	ledger := marchLedger()[:8]
	gold := func(transactions []Transaction, today time.Time) CardReport {
		t.Helper()
		return ComputeCardReports(testCardConfig(), nil, transactions, today)[0].Cards[0]
	}
	before := gold(ledger, day(2026, 3, 20))
	if before.FeesPaidYTD != 0 || before.NetEarnYTD != 0 || before.AnnualFee != 599_000 {
		t.Fatalf("no fee paid yet: %+v", before)
	}

	waiver := cardFee(day(2026, 3, 11), 0, "gold")
	waiver.Payment, waiver.Deposit = nil, money(59_900)
	withFee := append(append([]Transaction{}, ledger...),
		cardFee(day(2026, 3, 10), 658_900, "gold"), // the fee and its VAT
		waiver, // part of it given back
		cardFee(day(2025, 3, 10), 599_000, "gold"), // last year's
	)
	after := gold(withFee, day(2026, 3, 20))
	if after.FeesPaidYTD != 599_000 || after.NetEarnYTD != -599_000 {
		t.Fatalf("this year's fee rows, net of any given back: %+v", after)
	}
	if after.Standing.Spent != before.Standing.Spent || after.Standing.Cashback != before.Standing.Cashback {
		t.Fatalf("a fee is not spend: %+v vs %+v", after.Standing, before.Standing)
	}
}

func TestTheStatementEstimateIsWhatThePayByDateSettles(t *testing.T) {
	ledger := append(marchLedger(),
		cardFee(day(2026, 2, 20), 599_000, "gold"),
		cashbackRefund(day(2026, 2, 25), 50_000, "2026-01", "gold"),
	)
	// 20 March: pay by 29 March settles the statement of 5 February - 4 March.
	cards := ComputeCardReports(testCardConfig(), nil, ledger, day(2026, 3, 20))[0].Cards
	gold, silver := cards[0], cards[1]
	// The refund lowers the statement it was credited in; its memo names the period it was earned in.
	want := &CardStatement{
		Start: "2026-02-05", End: "2026-03-04", Spent: 1_000_000, Fees: 599_000, Cashback: 50_000, Amount: 1_549_000,
		Refunds: []StatementRefund{{Date: "2026-02-25", For: "2026-01", Amount: 50_000}},
	}
	if !reflect.DeepEqual(gold.Statement, want) {
		t.Fatalf("gold statement %+v", gold.Statement)
	}
	if silver.Statement == nil || silver.Statement.Amount != 0 {
		t.Fatalf("silver statement %+v", silver.Statement)
	}

	// 30 March: pay by 29 April settles the period still open, so it can grow.
	gold = ComputeCardReports(testCardConfig(), nil, ledger, day(2026, 3, 30))[0].Cards[0]
	want = &CardStatement{Start: "2026-03-05", End: "2026-04-04", Open: true, Spent: 17_300_000, Amount: 17_300_000}
	if !reflect.DeepEqual(gold.Statement, want) {
		t.Fatalf("open statement %+v", gold.Statement)
	}
}

func TestPayByIsUrgentWithinFiveDaysOnlyWhenTheStatementIsOwed(t *testing.T) {
	cards := ComputeCardReports(testCardConfig(), nil, marchLedger(), day(2026, 3, 24))[0].Cards
	gold, silver := cards[0], cards[1]
	if !gold.Cycle.PayByUrgent || gold.Cycle.DaysUntilPayBy != 5 {
		t.Fatalf("gold owes its statement: %+v", gold.Cycle)
	}
	if silver.Cycle.PayByUrgent {
		t.Fatalf("silver owes nothing: %+v", silver.Cycle)
	}
}

func TestACardReportCarriesItsLastFourDigits(t *testing.T) {
	config := testCardConfig()
	config.Cards[0].Last4 = "0123"
	gold := ComputeCardReports(config, nil, nil, day(2026, 3, 20))[0].Cards[0]
	if gold.Last4 != "0123" {
		t.Fatalf("last4 %q", gold.Last4)
	}
}
