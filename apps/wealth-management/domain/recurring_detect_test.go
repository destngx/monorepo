package domain

import (
	"strings"
	"testing"
	"time"
)

// detectToday is 17 October 2026: the last six completed months are
// April to September.
var detectToday = day(2026, 10, 17)

func flow(date time.Time, payee, category string, kind CategoryType, amount float64) Transaction {
	t := txn(date, category, kind, 0, 0)
	if kind == CategoryTypeIncome {
		t.Deposit = money(amount)
	} else {
		t.Payment = money(amount)
	}
	t.Payee = payee
	t.AccountName = "Main Bank"
	return t
}

func monthly(payee, category string, kind CategoryType, amounts []float64, days []int) []Transaction {
	out := []Transaction{}
	for i, amount := range amounts {
		out = append(out, flow(day(2026, time.Month(4+i), days[i]), payee, category, kind, amount))
	}
	return out
}

func findSuggestion(suggestions []RecurringSuggestion, name string) *RecurringSuggestion {
	for i := range suggestions {
		if suggestions[i].Name == name {
			return &suggestions[i]
		}
	}
	return nil
}

func TestGivenAStableMonthlyBillWhenDetectingThenItIsSuggestedWithMedianAmountAndTypicalDay(t *testing.T) {
	history := monthly("Electric Co", "Utilities", CategoryTypeExpense,
		[]float64{500_000, 520_000, 480_000, 510_000, 500_000, 490_000}, []int{5, 6, 5, 4, 5, 7})
	history = append(history, flow(day(2026, 10, 5), "ELECTRIC CO", "Utilities", CategoryTypeExpense, 900_000)) // this month: not counted

	suggestions := DetectRecurring(history, nil, detectToday)
	if len(suggestions) != 1 {
		t.Fatalf("suggestions = %+v", suggestions)
	}
	s := suggestions[0]
	if s.Name != "ELECTRIC CO" || s.Match != "ELECTRIC CO" || s.Kind != RecurringBill || s.Frequency != FrequencyMonthly || s.Category != "Utilities" {
		t.Fatalf("suggestion = %+v", s)
	}
	if s.Amount != 500_000 || s.Day != 5 || s.Account != "Main Bank" || s.LastSeen != "2026-09" || s.Seen != 6 || s.Checked != 6 {
		t.Fatalf("figures = %+v", s)
	}
	if s.Confidence != ConfidenceHigh || s.AmountSpread != 4 || s.DaySpread != 2 {
		t.Fatalf("confidence = %+v", s)
	}
	if !strings.HasPrefix(s.Key, "out|electric co|utilities") {
		t.Fatalf("key = %q", s.Key)
	}
}

func TestGivenSalaryAroundMonthEndWhenDetectingThenTheDayWrapsAcrossMonths(t *testing.T) {
	history := monthly("Employer A", "Salary", CategoryTypeIncome,
		[]float64{20_000_000, 20_000_000, 20_000_000, 20_000_000}, []int{30, 1, 29, 30})
	suggestions := DetectRecurring(history, nil, detectToday)
	s := findSuggestion(suggestions, "Employer A")
	if s == nil || s.Kind != RecurringIncome || s.Day != 30 || s.DaySpread != 2 || s.Confidence != ConfidenceMedium {
		t.Fatalf("salary = %+v", s)
	}
}

func TestGivenIrregularHistoryWhenDetectingThenNothingIsSuggested(t *testing.T) {
	cases := map[string][]Transaction{
		"two months only": monthly("Gym", "Fitness", CategoryTypeExpense, []float64{300_000, 300_000}, []int{3, 3}),
		"unstable amount": monthly("Grocer", "Food", CategoryTypeExpense, []float64{100_000, 400_000, 900_000, 250_000}, []int{3, 3, 3, 3}),
		"no typical day":  monthly("Water Co", "Utilities", CategoryTypeExpense, []float64{100_000, 100_000, 100_000, 100_000}, []int{1, 10, 20, 15}),
		"many a month": append(monthly("Cafe", "Food", CategoryTypeExpense, []float64{50_000, 50_000, 50_000}, []int{3, 3, 3}),
			flow(day(2026, 4, 10), "Cafe", "Food", CategoryTypeExpense, 50_000), flow(day(2026, 5, 10), "Cafe", "Food", CategoryTypeExpense, 50_000)),
	}
	for name, history := range cases {
		if suggestions := DetectRecurring(history, nil, detectToday); len(suggestions) != 0 {
			t.Errorf("%s: suggestions = %+v", name, suggestions)
		}
	}
}

func TestGivenLendingTransfersAndRefundsWhenDetectingThenTheyAreExcluded(t *testing.T) {
	lending := monthly("Person A", "Lending", CategoryTypeExpense, []float64{1_000_000, 1_000_000, 1_000_000, 1_000_000}, []int{2, 2, 2, 2})
	for i := range lending {
		lending[i].CategoryNature = NatureLending
	}
	transfers := monthly("Savings", "Transfer", CategoryTypeNonBudget, []float64{5_000_000, 5_000_000, 5_000_000, 5_000_000}, []int{2, 2, 2, 2})
	refunds := monthly("Shop", "Refunds/Reimbursements", CategoryTypeIncome, []float64{10_000, 10_000, 10_000, 10_000}, []int{2, 2, 2, 2})
	history := append(append(lending, transfers...), refunds...)
	if suggestions := DetectRecurring(history, nil, detectToday); len(suggestions) != 0 {
		t.Fatalf("suggestions = %+v", suggestions)
	}
}

func TestGivenAConfirmedItemWhenDetectingThenItIsNotSuggestedAgain(t *testing.T) {
	history := monthly("Streaming Plus Ltd", "Subscriptions", CategoryTypeExpense, []float64{100_000, 100_000, 100_000, 100_000}, []int{9, 9, 9, 9})
	if s := DetectRecurring(history, nil, detectToday); len(s) != 1 || s[0].Kind != RecurringSubscription {
		t.Fatalf("before confirm = %+v", s)
	}
	paused := []RecurringItem{{Name: "Streaming", Kind: RecurringSubscription, Match: "streaming plus", Status: RecurringPaused}}
	if s := DetectRecurring(history, paused, detectToday); len(s) != 0 {
		t.Fatalf("a paused (dismissed) row still covers it: %+v", s)
	}
	byName := []RecurringItem{{Name: "Streaming Plus Ltd", Kind: RecurringBill}}
	if s := DetectRecurring(history, byName, detectToday); len(s) != 0 {
		t.Fatalf("a row named after the payee covers it: %+v", s)
	}
	income := []RecurringItem{{Name: "Streaming Plus Ltd", Kind: RecurringIncome}}
	if s := DetectRecurring(history, income, detectToday); len(s) != 1 {
		t.Fatalf("an income row does not cover a bill: %+v", s)
	}
}

func TestGivenBlankPayeesWhenDetectingThenOnlyVeryStableGroupsAreSuggestedWithLowConfidence(t *testing.T) {
	stable := monthly("", "Rent", CategoryTypeExpense, []float64{8_000_000, 8_000_000, 8_000_000, 8_000_000}, []int{1, 1, 2, 1})
	s := DetectRecurring(stable, nil, detectToday)
	if len(s) != 1 || s[0].Name != "Rent" || s[0].Match != "" || s[0].Confidence != ConfidenceLow {
		t.Fatalf("blank payee = %+v", s)
	}
	loose := monthly("", "Rent", CategoryTypeExpense, []float64{8_000_000, 8_400_000, 8_000_000, 8_000_000}, []int{1, 1, 4, 1})
	if s := DetectRecurring(loose, nil, detectToday); len(s) != 0 {
		t.Fatalf("a loose blank-payee group is not suggested: %+v", s)
	}
}

func TestGivenAYearlyPremiumWhenDetectingThenItIsSuggestedFromConsecutiveYears(t *testing.T) {
	history := []Transaction{
		flow(day(2024, 3, 12), "Life Cover Co", "Insurance", CategoryTypeExpense, 10_000_000),
		flow(day(2025, 3, 10), "Life Cover Co", "Insurance", CategoryTypeExpense, 11_000_000),
		flow(day(2026, 3, 11), "Life Cover Co", "Insurance", CategoryTypeExpense, 12_000_000),
	}
	s := DetectRecurring(history, nil, detectToday)
	if len(s) != 1 {
		t.Fatalf("suggestions = %+v", s)
	}
	if s[0].Frequency != FrequencyYearly || s[0].Month != 3 || s[0].Day != 11 || s[0].Kind != RecurringInsurance || s[0].Amount != 11_000_000 || s[0].Seen != 3 || s[0].Confidence != ConfidenceMedium {
		t.Fatalf("yearly = %+v", s[0])
	}
	gap := []Transaction{history[0], history[2]}
	if s := DetectRecurring(gap, nil, detectToday); len(s) != 0 {
		t.Fatalf("years that are not consecutive are skipped: %+v", s)
	}
	stale := []Transaction{flow(day(2023, 3, 12), "Old Cover", "Insurance", CategoryTypeExpense, 1_000_000), flow(day(2024, 3, 12), "Old Cover", "Insurance", CategoryTypeExpense, 1_000_000)}
	if s := DetectRecurring(stale, nil, detectToday); len(s) != 0 {
		t.Fatalf("a premium not seen for over a year is skipped: %+v", s)
	}
}
