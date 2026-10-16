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
		Goals:        &GoalPlan{Check: PlanCheck{RequiredPerMonth: 5_000_000}},
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
	if safe.SpendableCash != 31_000_000 || len(safe.CashAccounts) != 2 {
		t.Fatalf("cash = %+v", safe)
	}
	// Overdue internet and the phone bill before payday; the card is due after it.
	if safe.BillsDue != 500_000 || safe.CardPayments != 0 || safe.GoalSetAside != 5_000_000 || !safe.GoalPlanUsed || len(safe.Counted) != 2 {
		t.Fatalf("components = %+v", safe)
	}
	if safe.Amount != 25_500_000 || safe.SafeToSpend != 25_500_000 || safe.ShortBy != 0 || safe.PerDay != 3_187_500 {
		t.Fatalf("result = %+v", safe)
	}
}

func TestGivenNoIncomeItemWhenComputingSafeToSpendThenTheHorizonIs30DaysAndShortfallIsExposed(t *testing.T) {
	input := billsInput()
	input.Sheet.Items = input.Sheet.Items[1:]
	input.Accounts = []Account{{Name: "Wallet", Type: AccountTypeCash, Balance: 5_000_000}, {Name: "Card B", Type: AccountTypeNegativeActive, Balance: -3_000_000}}
	safe := ComputeBills(input).SafeToSpend
	if safe.Until != "2026-11-16" || safe.UntilIncome || safe.Days != 30 {
		t.Fatalf("horizon = %+v", safe)
	}
	// 5M - (0.3 + 0.2 + 8 + 2 + 0.3)M bills - 3M card - 5M goals.
	if safe.BillsDue != 10_800_000 || safe.CardPayments != 3_000_000 || safe.Amount != -13_800_000 || safe.SafeToSpend != 0 || safe.ShortBy != 13_800_000 || safe.PerDay != 0 {
		t.Fatalf("safe = %+v", safe)
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
	if len(summary.Overdue) != 1 || summary.Overdue[0].Name != "Internet" || len(summary.DueSoon) != 1 || summary.DueSoon[0].Name != "Phone" || summary.SafeToSpend.SafeToSpend != 25_500_000 {
		t.Fatalf("summary = %+v", summary)
	}
	facts := bills.Facts()
	if facts.Items != 5 || facts.UpcomingOut != 13_500_000 || facts.OverdueCount != 1 || facts.SafeToSpend != 25_500_000 || facts.Until != "2026-10-25" {
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
