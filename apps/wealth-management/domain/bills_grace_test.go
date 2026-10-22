package domain

import (
	"strings"
	"testing"
	"time"
)

// A bill with grace days (D42), such as rent the user or the landlord may
// settle a week or two late: unpaid within the grace it is still due, it is
// overdue only once the grace has passed, and a late payment settles the
// cycle it was late for.

func graceRent(day, grace int) RecurringItem {
	return RecurringItem{ID: "recurring-3", Row: 3, Name: "Rent", Kind: RecurringBill, Amount: money(8_000_000), Frequency: FrequencyMonthly, Day: day, GraceDays: grace, Match: "Landlord", Status: RecurringActive}
}

func graceInput(today time.Time, item RecurringItem, transactions ...Transaction) BillsInput {
	return BillsInput{
		Sheet:        RecurringSheet{Exists: true, Items: []RecurringItem{item}},
		Accounts:     []Account{{Name: "Main Bank", Type: AccountTypeBank, Balance: 30_000_000}},
		Transactions: transactions,
		Today:        today,
	}
}

func rentPaid(date time.Time) Transaction {
	return flow(date, "Landlord", "Rent", CategoryTypeExpense, 8_000_000)
}

func TestGivenAnUnpaidBillWithinItsGraceWhenComputingBillsThenItIsStillDueNotOverdue(t *testing.T) {
	bills := ComputeBills(graceInput(day(2026, 10, 17), graceRent(10, 14)))

	rent := occurrenceAt(t, bills.Calendar.Occurrences, "Rent", "2026-10-10")
	if rent.Status != BillDue || rent.GraceUntil != "2026-10-24" || rent.DaysUntil != -7 || rent.Remaining != 8_000_000 {
		t.Fatalf("rent within grace = %+v", rent)
	}
	if bills.Calendar.OverdueCount != 0 || bills.Calendar.UpcomingOut != 16_000_000 || bills.Calendar.UpcomingCount != 2 {
		t.Fatalf("calendar totals = %+v", bills.Calendar)
	}
	if bills.SafeToSpend.BillsDue != 16_000_000 {
		t.Fatalf("a late bill is still committed: bills due = %v", bills.SafeToSpend.BillsDue)
	}
	if summary := bills.Summary(); len(summary.Overdue) != 0 {
		t.Fatalf("no overdue bill within grace: %+v", summary.Overdue)
	}
}

func TestGivenAnUnpaidBillPastItsGraceWhenComputingBillsThenItIsOverdue(t *testing.T) {
	bills := ComputeBills(graceInput(day(2026, 10, 25), graceRent(10, 14)))

	rent := occurrenceAt(t, bills.Calendar.Occurrences, "Rent", "2026-10-10")
	if rent.Status != BillOverdue || bills.Calendar.OverdueCount != 1 {
		t.Fatalf("rent past grace = %+v, overdue = %d", rent, bills.Calendar.OverdueCount)
	}
}

func TestGivenTheLastDayOfGraceWhenComputingBillsThenTheBillIsStillDue(t *testing.T) {
	bills := ComputeBills(graceInput(day(2026, 10, 24), graceRent(10, 14)))

	if rent := occurrenceAt(t, bills.Calendar.Occurrences, "Rent", "2026-10-10"); rent.Status != BillDue {
		t.Fatalf("rent on its last day of grace = %+v", rent)
	}
}

func TestGivenRentPaidLateInTheNextMonthWhenComputingBillsThenThePaymentSettlesTheLateCycle(t *testing.T) {
	bills := ComputeBills(graceInput(day(2026, 11, 6), graceRent(28, 14), rentPaid(day(2026, 11, 5))))

	november := occurrenceAt(t, bills.Calendar.Occurrences, "Rent", "2026-11-28")
	if november.Status != BillDue || len(november.Matches) != 0 {
		t.Fatalf("the late October payment must not settle November: %+v", november)
	}
	for _, o := range bills.Calendar.Occurrences {
		if o.ExpectedDate == "2026-10-28" {
			t.Fatalf("a paid cycle of last month is not carried over: %+v", o)
		}
	}
	current := bills.Items[0].Current
	if current == nil || current.ExpectedDate != "2026-10-28" || current.Status != BillPaid || current.PaidDate != "2026-11-05" {
		t.Fatalf("current cycle = %+v", current)
	}
}

func TestGivenRentPaidEarlyWhenComputingBillsThenThePaymentSettlesTheComingCycle(t *testing.T) {
	bills := ComputeBills(graceInput(day(2026, 10, 25), graceRent(28, 14), rentPaid(day(2026, 9, 28)), rentPaid(day(2026, 10, 20))))

	if rent := occurrenceAt(t, bills.Calendar.Occurrences, "Rent", "2026-10-28"); rent.Status != BillPaid || rent.PaidDate != "2026-10-20" {
		t.Fatalf("rent paid 8 days early = %+v", rent)
	}
}

func TestGivenLastMonthsRentUnpaidWhenComputingBillsThenItIsCarriedIntoThisMonth(t *testing.T) {
	bills := ComputeBills(graceInput(day(2026, 11, 6), graceRent(28, 14)))

	late := occurrenceAt(t, bills.Calendar.Occurrences, "Rent", "2026-10-28")
	if late.Status != BillDue || late.GraceUntil != "2026-11-11" || late.DaysUntil != -9 {
		t.Fatalf("last month's rent within grace = %+v", late)
	}
	if bills.SafeToSpend.BillsDue != 16_000_000 || len(bills.SafeToSpend.Counted) != 2 {
		t.Fatalf("both rents are committed: due = %v, counted = %+v", bills.SafeToSpend.BillsDue, bills.SafeToSpend.Counted)
	}

	overdue := ComputeBills(graceInput(day(2026, 11, 20), graceRent(28, 14)))
	if late := occurrenceAt(t, overdue.Calendar.Occurrences, "Rent", "2026-10-28"); late.Status != BillOverdue {
		t.Fatalf("last month's rent past grace = %+v", late)
	}
}

func TestGivenAGraceThatEndedLastMonthWhenComputingBillsThenTheOldCycleIsNotCarried(t *testing.T) {
	bills := ComputeBills(graceInput(day(2026, 11, 6), graceRent(10, 14)))

	got := []string{}
	for _, o := range bills.Calendar.Occurrences {
		got = append(got, o.ExpectedDate)
	}
	if strings.Join(got, ",") != "2026-11-10" {
		t.Fatalf("occurrences = %v", got)
	}
}

func TestGivenNoGraceWhenComputingBillsThenTheBillIsOverdueTheDayAfterItsDate(t *testing.T) {
	bills := ComputeBills(graceInput(day(2026, 10, 11), graceRent(10, 0)))

	rent := occurrenceAt(t, bills.Calendar.Occurrences, "Rent", "2026-10-10")
	if rent.Status != BillOverdue || rent.GraceUntil != "" {
		t.Fatalf("rent without grace = %+v", rent)
	}
}

func TestGivenGraceDaysWhenValidatingARecurringItemThenTheyMustFitTheRules(t *testing.T) {
	valid := RecurringInput{Name: "Rent", Kind: RecurringBill, Amount: money(8_000_000), Day: 28, Match: "Landlord", GraceDays: MaxGraceDays}
	if err := valid.Validate(); err != nil {
		t.Fatalf("grace of %d days: %v", MaxGraceDays, err)
	}
	for name, input := range map[string]RecurringInput{
		"negative":  {Name: "Rent", Kind: RecurringBill, Amount: money(8_000_000), Day: 28, Match: "Landlord", GraceDays: -1},
		"too long":  {Name: "Rent", Kind: RecurringBill, Amount: money(8_000_000), Day: 28, Match: "Landlord", GraceDays: MaxGraceDays + 1},
		"card rows": {Name: "Card B", Kind: RecurringCard, GraceDays: 5},
	} {
		if err := input.Validate(); err == nil || !strings.Contains(err.Error(), "grace") {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}
