package domain

import (
	"strings"
	"testing"
)

// billsToday is 17 October 2026.
var billsToday = day(2026, 10, 17)

func billItems() []RecurringItem {
	return []RecurringItem{
		{ID: "recurring-2", Row: 2, Name: "Salary", Kind: RecurringIncome, Amount: money(20_000_000), Frequency: FrequencyMonthly, Day: 25, Match: "Employer A", Status: RecurringActive},
		{ID: "recurring-3", Row: 3, Name: "Rent", Kind: RecurringBill, Amount: money(8_000_000), Frequency: FrequencyMonthly, Day: 1, Match: "Landlord", Status: RecurringActive},
		{ID: "recurring-4", Row: 4, Name: "Internet", Kind: RecurringBill, Amount: money(300_000), Frequency: FrequencyMonthly, Day: 10, Match: "Net Co", Status: RecurringActive},
		{ID: "recurring-5", Row: 5, Name: "Phone", Kind: RecurringSubscription, Amount: money(200_000), Frequency: FrequencyMonthly, Day: 20, Match: "Phone Co", Status: RecurringActive},
		{ID: "recurring-6", Row: 6, Name: "Gym", Kind: RecurringSubscription, Amount: money(500_000), Frequency: FrequencyMonthly, Day: 18, Match: "Gym", Status: RecurringPaused},
		{ID: "recurring-7", Row: 7, Name: "Health cover", Kind: RecurringInsurance, Amount: money(2_000_000), Frequency: FrequencyYearly, Day: 5, Month: 11, Match: "Cover Co", Status: RecurringActive},
	}
}

func billAccounts() []Account {
	return []Account{
		{Name: "Main Bank", Type: AccountTypeBank, Balance: 30_000_000},
		{Name: "Wallet", Type: AccountTypeCash, Balance: 1_000_000},
		{Name: "Long Savings", Type: AccountTypeLongHolding, Balance: 100_000_000},
		{Name: "Card B", Type: AccountTypeNegativeActive, Balance: -3_000_000},
	}
}

func billsInput() BillsInput {
	rent := flow(day(2026, 10, 1), "Landlord", "Rent", CategoryTypeExpense, 8_000_000)
	rent.ID = "row-10"
	return BillsInput{
		Sheet:        RecurringSheet{Exists: true, Items: billItems()},
		Accounts:     billAccounts(),
		Transactions: []Transaction{rent, flow(day(2026, 9, 25), "Employer A", "Salary", CategoryTypeIncome, 20_000_000)},
		Today:        billsToday,
	}
}

func occurrenceAt(t *testing.T, occurrences []BillOccurrence, name, date string) BillOccurrence {
	t.Helper()
	for _, o := range occurrences {
		if o.Name == name && o.ExpectedDate == date {
			return o
		}
	}
	t.Fatalf("no %s on %s in %+v", name, date, occurrences)
	return BillOccurrence{}
}

func TestGivenConfirmedItemsWhenComputingBillsThenTheCalendarShowsPaidDueAndOverdue(t *testing.T) {
	bills := ComputeBills(billsInput())
	calendar := bills.Calendar
	if calendar.From != "2026-10-01" || calendar.To != "2026-11-16" {
		t.Fatalf("range = %s..%s", calendar.From, calendar.To)
	}
	got := []string{}
	for _, o := range calendar.Occurrences {
		got = append(got, o.ExpectedDate+" "+o.Name+" "+string(o.Status))
	}
	want := []string{
		"2026-10-01 Rent paid", "2026-10-10 Internet overdue", "2026-10-20 Phone due", "2026-10-25 Salary due",
		"2026-10-29 Card B due", "2026-11-01 Rent due", "2026-11-05 Health cover due", "2026-11-10 Internet due",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("calendar:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	rent := occurrenceAt(t, calendar.Occurrences, "Rent", "2026-10-01")
	if rent.PaidAmount != 8_000_000 || rent.PaidDate != "2026-10-01" || rent.Remaining != 0 || len(rent.Matches) != 1 || rent.Matches[0].TransactionID != "row-10" {
		t.Fatalf("rent = %+v", rent)
	}
	card := occurrenceAt(t, calendar.Occurrences, "Card B", "2026-10-29")
	if card.Source != BillFromCard || card.ExpectedAmount != 3_000_000 || !card.Estimate || card.DaysUntil != 12 {
		t.Fatalf("card without settings uses the balance owed: %+v", card)
	}
	if calendar.UpcomingOut != 13_500_000 || calendar.UpcomingCount != 5 || calendar.UpcomingIn != 20_000_000 || calendar.OverdueCount != 1 || calendar.OverdueAmount != 300_000 || calendar.PaidOut != 8_000_000 {
		t.Fatalf("totals = %+v", calendar)
	}
}

func TestGivenTheNextIncomeWhenComputingSafeToSpendThenEveryComponentIsReturned(t *testing.T) {
	safe := ComputeBills(billsInput()).SafeToSpend
	if safe.Until != "2026-10-25" || !safe.UntilIncome || safe.IncomeName != "Salary" || safe.Days != 8 {
		t.Fatalf("horizon = %+v", safe)
	}
	if safe.ReadyMoney != 31_000_000 || len(safe.ReadyAccounts) != 2 || len(safe.ExcludedAccounts) != 0 || len(safe.Issues) != 0 {
		t.Fatalf("ready money = %+v", safe)
	}
	// Overdue internet and the phone bill before payday; the card is due after it.
	if safe.BillsDue != 500_000 || safe.CardPayments != 0 || len(safe.Counted) != 2 {
		t.Fatalf("components = %+v", safe)
	}
	if safe.Amount != 30_500_000 || safe.SafeToSpend != 30_500_000 || safe.ShortBy != 0 || safe.PerDay != 3_812_500 {
		t.Fatalf("result = %+v", safe)
	}
	if safe.ReadyRule != SafeToSpendReadyRule || !strings.Contains(safe.Explanation, "before the next income (Salary, 2026-10-25)") ||
		!strings.Contains(safe.Explanation, "on payday itself") {
		t.Fatalf("explanation = %q", safe.Explanation)
	}
}

func TestGivenAGoalPlanWhenComputingSafeToSpendThenNoGoalSetAsideIsSubtracted(t *testing.T) {
	safe := ComputeBills(billsInput()).SafeToSpend
	if safe.Amount != safe.ReadyMoney-safe.BillsDue-safe.CardPayments {
		t.Fatalf("safe-to-spend = ready money - bills - card payments: %+v", safe)
	}
	if strings.Contains(strings.ToLower(safe.Explanation), "goal") {
		t.Fatalf("explanation still mentions goals: %q", safe.Explanation)
	}
}

func TestGivenCryptoAccountsWhenComputingSafeToSpendThenTheyAreExcludedAsCrypto(t *testing.T) {
	input := billsInput()
	input.Accounts = append(input.Accounts,
		Account{Name: "Exchange C", Type: AccountTypeActiveUse, Currency: CurrencyUSDT, Balance: 9_000_000},
		Account{Name: "Coins D", Type: AccountTypeCrypto, Balance: 4_000_000},
	)
	safe := ComputeBills(input).SafeToSpend
	if safe.ReadyMoney != 31_000_000 || len(safe.ReadyAccounts) != 2 {
		t.Fatalf("crypto is not ready money: %+v", safe)
	}
	if len(safe.ExcludedAccounts) != 2 {
		t.Fatalf("excluded = %+v", safe.ExcludedAccounts)
	}
	for _, excluded := range safe.ExcludedAccounts {
		if excluded.Reason != ExcludedCrypto {
			t.Fatalf("excluded = %+v", safe.ExcludedAccounts)
		}
	}
}

func TestGivenAnAccountMarkedNotReadyWhenComputingSafeToSpendThenItIsExcludedAsSlow(t *testing.T) {
	input := billsInput()
	input.Accounts = append(input.Accounts, Account{Name: "Broker B", Type: AccountTypeActiveUse, Balance: 12_000_000})
	input.Readiness = SafeToSpendSettings{NotReadyAccounts: []string{"broker b"}, ETag: "etag-1"}
	safe := ComputeBills(input).SafeToSpend
	if safe.ReadyMoney != 31_000_000 || len(safe.ReadyAccounts) != 2 || len(safe.Issues) != 0 {
		t.Fatalf("a slow account is not ready money: %+v", safe)
	}
	if len(safe.ExcludedAccounts) != 1 || safe.ExcludedAccounts[0] != (ExcludedAccount{Name: "Broker B", Balance: 12_000_000, Reason: ExcludedNotReady}) {
		t.Fatalf("excluded = %+v", safe.ExcludedAccounts)
	}
	if safe.Readiness.ETag != "etag-1" || len(safe.Readiness.NotReadyAccounts) != 1 {
		t.Fatalf("the setting is returned for the switches: %+v", safe.Readiness)
	}
}

func TestGivenAMarkedNameThatIsNoAccountWhenComputingSafeToSpendThenItIsIgnoredAndReported(t *testing.T) {
	input := billsInput()
	input.Readiness = SafeToSpendSettings{NotReadyAccounts: []string{"Old Broker"}}
	safe := ComputeBills(input).SafeToSpend
	if safe.ReadyMoney != 31_000_000 || len(safe.ExcludedAccounts) != 0 {
		t.Fatalf("an unknown name changes nothing: %+v", safe)
	}
	if len(safe.Issues) != 1 || !strings.Contains(safe.Issues[0], `"Old Broker"`) || len(safe.UnknownNotReady) != 1 || safe.UnknownNotReady[0] != "Old Broker" {
		t.Fatalf("issues = %v", safe.Issues)
	}
}

func TestGivenABillDueOnPaydayWhenComputingSafeToSpendThenTheWagePaysItAndItIsNotCounted(t *testing.T) {
	input := billsInput()
	input.Sheet.Items = append(input.Sheet.Items,
		RecurringItem{ID: "recurring-8", Row: 8, Name: "Water", Kind: RecurringBill, Amount: money(400_000), Frequency: FrequencyMonthly, Day: 25, Match: "Water Co", Status: RecurringActive},
		RecurringItem{ID: "recurring-9", Row: 9, Name: "Power", Kind: RecurringBill, Amount: money(700_000), Frequency: FrequencyMonthly, Day: 24, Match: "Power Co", Status: RecurringActive},
	)
	safe := ComputeBills(input).SafeToSpend
	for _, o := range safe.Counted {
		if o.Name == "Water" {
			t.Fatalf("a bill due on payday is paid by the wage: %+v", safe.Counted)
		}
	}
	if power := occurrenceAt(t, safe.Counted, "Power", "2026-10-24"); power.Remaining != 700_000 {
		t.Fatalf("a bill the day before payday is counted: %+v", power)
	}
	// Overdue internet, phone and power.
	if safe.BillsDue != 1_200_000 || safe.Amount != 29_800_000 {
		t.Fatalf("safe = %+v", safe)
	}
	if internet := occurrenceAt(t, safe.Counted, "Internet", "2026-10-10"); internet.Status != BillOverdue {
		t.Fatalf("an overdue bill is counted: %+v", internet)
	}
}

func TestGivenACardPaymentDueOnPaydayWhenComputingSafeToSpendThenItIsNotCounted(t *testing.T) {
	input := billsInput()
	input.Sheet.Items[0].Day = 29
	safe := ComputeBills(input).SafeToSpend
	if safe.Until != "2026-10-29" || safe.CardPayments != 0 {
		t.Fatalf("the card due on payday is paid by the wage: %+v", safe)
	}
	input.Sheet.Items[0].Day = 30
	if safe = ComputeBills(input).SafeToSpend; safe.CardPayments != 3_000_000 {
		t.Fatalf("the card due the day before payday is counted: %+v", safe)
	}
}

func TestGivenNoIncomeItemWhenComputingSafeToSpendThenTheHorizonIs30DaysAndShortfallIsExposed(t *testing.T) {
	input := billsInput()
	input.Sheet.Items = input.Sheet.Items[1:]
	input.Sheet.Items = append(input.Sheet.Items, RecurringItem{ID: "recurring-8", Row: 8, Name: "Water", Kind: RecurringBill, Amount: money(400_000), Frequency: FrequencyMonthly, Day: 16, Match: "Water Co", Status: RecurringActive})
	input.Accounts = []Account{{Name: "Wallet", Type: AccountTypeCash, Balance: 5_000_000}, {Name: "Card B", Type: AccountTypeNegativeActive, Balance: -3_000_000}}
	safe := ComputeBills(input).SafeToSpend
	if safe.Until != "2026-11-16" || safe.UntilIncome || safe.Days != 30 {
		t.Fatalf("horizon = %+v", safe)
	}
	// The 30-day window is inclusive: water on 16 November counts, and the
	// overdue one of 16 October too.
	occurrenceAt(t, safe.Counted, "Water", "2026-11-16")
	occurrenceAt(t, safe.Counted, "Water", "2026-10-16")
	// 5M - (0.3 + 0.2 + 8 + 2 + 0.3 + 0.4 + 0.4)M bills - 3M card.
	if safe.BillsDue != 11_600_000 || safe.CardPayments != 3_000_000 || safe.Amount != -9_600_000 || safe.SafeToSpend != 0 || safe.ShortBy != 9_600_000 || safe.PerDay != 0 {
		t.Fatalf("safe = %+v", safe)
	}
	if !strings.Contains(safe.Explanation, "no income expected") || !strings.Contains(safe.Explanation, "up to and including 2026-11-16") {
		t.Fatalf("explanation = %q", safe.Explanation)
	}
}

func TestGivenSalaryAlreadyReceivedWhenComputingSafeToSpendThenTheHorizonMovesToNextPayday(t *testing.T) {
	input := billsInput()
	input.Transactions = append(input.Transactions, flow(day(2026, 10, 16), "EMPLOYER A", "Salary", CategoryTypeIncome, 20_000_000))
	bills := ComputeBills(input)
	if bills.SafeToSpend.Until != "2026-11-25" {
		t.Fatalf("until = %s", bills.SafeToSpend.Until)
	}
	salary := occurrenceAt(t, bills.Calendar.Occurrences, "Salary", "2026-10-25")
	if salary.Status != BillPaid || salary.PaidDate != "2026-10-16" {
		t.Fatalf("an early salary still pays this month's: %+v", salary)
	}
}

func TestGivenItemsWhenComputingBillsThenEachHasItsCurrentCycleAndLastMatch(t *testing.T) {
	bills := ComputeBills(billsInput())
	if len(bills.Items) != 6 {
		t.Fatalf("items = %d", len(bills.Items))
	}
	rent := bills.Items[1]
	if rent.Current == nil || rent.Current.Status != BillPaid || rent.NextDate != "2026-11-01" || rent.LastMatch == nil || rent.LastMatch.Date != "2026-10-01" || rent.MatchCount != 1 {
		t.Fatalf("rent = %+v current %+v", rent, rent.Current)
	}
	cover := bills.Items[5]
	if cover.Current == nil || cover.Current.ExpectedDate != "2026-11-05" || cover.Current.Status != BillDue || cover.NextDate != "2026-11-05" {
		t.Fatalf("a yearly item's cycle runs from the month before: %+v", cover)
	}
	march := billItems()[5]
	march.Month = 3
	if described := describeItem(march, nil, billsToday); described.Current != nil || described.NextDate != "2027-03-05" {
		t.Fatalf("a yearly item outside its cycle has no current occurrence: %+v", described)
	}
	gym := bills.Items[4]
	if gym.Current == nil || gym.Current.Status != BillDue {
		t.Fatalf("a paused item still shows its cycle on the item: %+v", gym)
	}
	summary := bills.Summary()
	if len(summary.Overdue) != 1 || summary.Overdue[0].Name != "Internet" || len(summary.DueSoon) != 1 || summary.DueSoon[0].Name != "Phone" || summary.SafeToSpend.SafeToSpend != 30_500_000 {
		t.Fatalf("summary = %+v", summary)
	}
	facts := bills.Facts()
	if facts.Items != 5 || facts.UpcomingOut != 13_500_000 || facts.OverdueCount != 1 || facts.SafeToSpend != 30_500_000 || facts.Until != "2026-10-25" ||
		facts.ReadyMoney != 31_000_000 || facts.DueBeforeUntil != 500_000 || facts.ExcludedAccounts != 0 {
		t.Fatalf("facts = %+v", facts)
	}
}

func TestGivenCardSettingsWhenComputingBillsThenTheStatementIsDueByThePayByDate(t *testing.T) {
	repay := func(amount float64) Transaction {
		t := flow(day(2026, 3, 10), "Repayment", "Transfer", CategoryTypeNonBudget, 0)
		t.Payment, t.Deposit, t.AccountName = nil, money(amount), "Credit Card Acme"
		return t
	}
	input := BillsInput{
		Accounts:     []Account{{Name: "Main Bank", Type: AccountTypeBank, Balance: 10_000_000}, {Name: "Credit Card Acme", Type: AccountTypeNegativeActive, Balance: -5_000_000}},
		Transactions: append(marchLedger(), repay(600_000)),
		Cards:        testCardConfig(),
		Today:        day(2026, 3, 20),
		Sheet: RecurringSheet{Exists: true, Items: []RecurringItem{
			{ID: "recurring-2", Row: 2, Name: "Acme cards", Kind: RecurringCard, Account: "credit card acme", Status: RecurringActive, Note: "Pay in full"},
			{ID: "recurring-3", Row: 3, Name: "Old card", Kind: RecurringCard, Account: "Closed Card", Status: RecurringActive},
		}},
	}
	bills := ComputeBills(input)
	card := occurrenceAt(t, bills.Calendar.Occurrences, "Acme cards", "2026-03-29")
	// The statement settled on 29 March covers 5 Feb - 4 Mar: the 1M paid on 1 March.
	if card.ItemID != "recurring-2" || card.Note != "Pay in full" || card.ExpectedAmount != 1_000_000 || card.Estimate || card.PaidAmount != 600_000 || card.Remaining != 400_000 || card.Status != BillDue {
		t.Fatalf("card = %+v", card)
	}
	if bills.SafeToSpend.CardPayments != 400_000 {
		t.Fatalf("safe-to-spend counts what is left on the statement: %+v", bills.SafeToSpend)
	}
	if len(bills.Issues) != 1 || bills.Issues[0].Row != 3 {
		t.Fatalf("a card row without a card account is reported: %+v", bills.Issues)
	}

	input.Transactions = append(input.Transactions, repay(400_000))
	if card := ComputeBills(input).Calendar.Occurrences[0]; card.Status != BillPaid || card.Remaining != 0 {
		t.Fatalf("repaid in full = %+v", card)
	}
	input.Sheet.Items[0].Status = RecurringPaused
	for _, o := range ComputeBills(input).Calendar.Occurrences {
		if o.Source == BillFromCard {
			t.Fatalf("a paused card row hides the payment: %+v", o)
		}
	}
}

func TestGivenAMinimumBalanceWhenComputingSafeToSpendThenOnlyTheBalanceAboveItIsReady(t *testing.T) {
	input := billsInput()
	input.Minimums = AccountMinimumSettings{Accounts: []AccountMinimum{{Account: "main bank", Minimum: 5_000_000, Note: "points"}}, ETag: "etag-m"}
	safe := ComputeBills(input).SafeToSpend
	// Main Bank 30M - 5M kept + Wallet 1M.
	if safe.ReadyMoney != 26_000_000 || safe.Reserved != 5_000_000 || safe.Amount != 25_500_000 {
		t.Fatalf("safe = %+v", safe)
	}
	main := safe.ReadyAccounts[0]
	if main != (SpendableAccount{Name: "Main Bank", Balance: 30_000_000, Reserved: 5_000_000, Spendable: 25_000_000}) {
		t.Fatalf("main = %+v", main)
	}
	if safe.ReadyAccounts[1] != (SpendableAccount{Name: "Wallet", Balance: 1_000_000, Spendable: 1_000_000}) {
		t.Fatalf("wallet = %+v", safe.ReadyAccounts[1])
	}
	if safe.MinimumBalance.ETag != "etag-m" || len(safe.MinimumBalances) != 1 || safe.MinimumBalances[0].Account != "Main Bank" {
		t.Fatalf("tracking = %+v", safe.MinimumBalances)
	}
	if !strings.Contains(safe.Explanation, "minimum balance") || !strings.Contains(safe.ReadyRule, "accounts.minimumBalance") {
		t.Fatalf("explanation = %q", safe.Explanation)
	}
	facts := ComputeBills(input).Facts()
	if facts.MinimumReserved != 5_000_000 || facts.MinimumBalanceAccounts != 1 || facts.MinimumBalanceAtRisk != 0 {
		t.Fatalf("facts = %+v", facts)
	}
}

func TestGivenABalanceBelowItsMinimumWhenComputingSafeToSpendThenTheAccountAddsNothing(t *testing.T) {
	input := billsInput()
	input.Minimums = AccountMinimumSettings{Accounts: []AccountMinimum{{Account: "Wallet", Minimum: 3_000_000}}}
	safe := ComputeBills(input).SafeToSpend
	if safe.ReadyMoney != 30_000_000 || safe.Reserved != 1_000_000 {
		t.Fatalf("safe = %+v", safe)
	}
	if wallet := safe.ReadyAccounts[1]; wallet.Spendable != 0 || wallet.Reserved != 1_000_000 {
		t.Fatalf("wallet = %+v", wallet)
	}
	if track := safe.MinimumBalances[0]; track.Status != MinimumAtRisk || track.ExtraNeeded <= 0 {
		t.Fatalf("track = %+v", track)
	}
}

func TestGivenAMinimumForANameThatIsNoAccountWhenComputingSafeToSpendThenItIsIgnoredAndReported(t *testing.T) {
	input := billsInput()
	input.Minimums = AccountMinimumSettings{Accounts: []AccountMinimum{{Account: "Old Bank", Minimum: 5_000_000}}}
	safe := ComputeBills(input).SafeToSpend
	if safe.ReadyMoney != 31_000_000 || safe.Reserved != 0 || len(safe.MinimumBalances) != 0 {
		t.Fatalf("an unknown name changes nothing: %+v", safe)
	}
	if len(safe.UnknownMinimum) != 1 || safe.UnknownMinimum[0] != "Old Bank" || len(safe.Issues) != 1 ||
		!strings.Contains(safe.Issues[0], `"Old Bank"`) || !strings.Contains(safe.Issues[0], string(PrefAccountsMinimumBalance)) {
		t.Fatalf("issues = %v", safe.Issues)
	}
}

func TestGivenAMinimumOnAnAccountNotReadyWhenComputingSafeToSpendThenItsAverageIsStillTracked(t *testing.T) {
	input := billsInput()
	input.Accounts = append(input.Accounts, Account{Name: "Slow Bank", Type: AccountTypeBank, Balance: 2_000_000})
	input.Readiness = SafeToSpendSettings{NotReadyAccounts: []string{"Slow Bank"}}
	input.Minimums = AccountMinimumSettings{Accounts: []AccountMinimum{{Account: "Slow Bank", Minimum: 5_000_000}}}
	safe := ComputeBills(input).SafeToSpend
	if safe.Reserved != 0 || safe.ReadyMoney != 31_000_000 || len(safe.MinimumBalances) != 1 || safe.MinimumBalances[0].Status != MinimumAtRisk {
		t.Fatalf("safe = %+v", safe)
	}
}
