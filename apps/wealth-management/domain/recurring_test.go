package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRecurringInputValidation(t *testing.T) {
	valid := RecurringInput{Name: "Rent", Kind: RecurringBill, Amount: money(8_000_000), Day: 1, Match: "landlord"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid: %v", err)
	}
	card := RecurringInput{Name: "Card A", Kind: RecurringCard}
	if err := card.Validate(); err != nil {
		t.Fatalf("a card row needs no amount, day or match: %v", err)
	}
	cases := map[string]struct {
		input RecurringInput
		want  string
	}{
		"name":      {RecurringInput{Kind: RecurringBill, Amount: money(1), Day: 1, Match: "x"}, "name is required"},
		"kind":      {RecurringInput{Name: "X", Kind: "loan", Amount: money(1), Day: 1, Match: "x"}, "kind"},
		"amount":    {RecurringInput{Name: "X", Kind: RecurringBill, Day: 1, Match: "x"}, "amount is required"},
		"negative":  {RecurringInput{Name: "X", Kind: RecurringBill, Amount: money(-1), Day: 1, Match: "x"}, "amount must be positive"},
		"day":       {RecurringInput{Name: "X", Kind: RecurringBill, Amount: money(1), Day: 32, Match: "x"}, "day must be"},
		"month":     {RecurringInput{Name: "X", Kind: RecurringInsurance, Amount: money(1), Day: 1, Frequency: FrequencyYearly, Match: "x"}, "month (1-12)"},
		"frequency": {RecurringInput{Name: "X", Kind: RecurringBill, Amount: money(1), Day: 1, Frequency: "weekly", Match: "x"}, "frequency"},
		"status":    {RecurringInput{Name: "X", Kind: RecurringBill, Amount: money(1), Day: 1, Status: "gone", Match: "x"}, "status"},
		"match":     {RecurringInput{Name: "X", Kind: RecurringBill, Amount: money(1), Day: 1}, "match text or a category"},
	}
	for name, tc := range cases {
		err := tc.input.Validate()
		if !errors.Is(err, ErrValidation) || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %v should mention %q", name, err, tc.want)
		}
	}
	if err := (RecurringUpdateInput{RecurringInput: valid}).Validate(); !errors.Is(err, ErrValidation) {
		t.Fatalf("update without etag: %v", err)
	}
	if err := (RecurringStatusInput{Status: "done", ETag: "x"}).Validate(); !errors.Is(err, ErrValidation) {
		t.Fatalf("unknown status: %v", err)
	}
}

func TestGivenASuggestionWhenConfirmingThenBlankOverridesKeepTheSuggestedValues(t *testing.T) {
	suggestion := RecurringSuggestion{Key: "out|water|utilities", Name: "Water Co", Kind: RecurringBill, Amount: 200_000, Frequency: FrequencyMonthly, Day: 12, Account: "Main Bank", Category: "Utilities", Match: "Water Co"}
	item := RecurringConfirmInput{Key: suggestion.Key, Name: " Water ", Amount: money(250_000)}.Apply(suggestion)
	if item.Name != "Water" || *item.Amount != 250_000 || item.Day != 12 || item.Match != "Water Co" || item.Status != RecurringActive || item.Frequency != FrequencyMonthly {
		t.Fatalf("item = %+v", item)
	}
	dismissed := RecurringConfirmInput{Key: suggestion.Key, Status: RecurringPaused}.Apply(suggestion)
	if dismissed.Status != RecurringPaused || dismissed.Name != "Water Co" {
		t.Fatalf("dismissed = %+v", dismissed)
	}
}

func TestRecurringOccurrencesClampTheDayAndFollowTheFrequency(t *testing.T) {
	monthlyItem := RecurringItem{Frequency: FrequencyMonthly, Day: 31}
	dates := monthlyItem.occurrencesBetween(day(2026, 1, 1), day(2026, 4, 30))
	want := []string{"2026-01-31", "2026-02-28", "2026-03-31", "2026-04-30"}
	if len(dates) != len(want) {
		t.Fatalf("dates = %v", dates)
	}
	for i, date := range dates {
		if date.Format(time.DateOnly) != want[i] {
			t.Errorf("date %d = %s, want %s", i, date.Format(time.DateOnly), want[i])
		}
	}
	quarterly := RecurringItem{Frequency: FrequencyQuarterly, Day: 15, Month: 2}
	if dates := quarterly.occurrencesBetween(day(2026, 1, 1), day(2026, 12, 31)); len(dates) != 4 || dates[0].Month() != time.February || dates[3].Month() != time.November {
		t.Fatalf("quarterly = %v", dates)
	}
	yearly := RecurringItem{Frequency: FrequencyYearly, Day: 10, Month: 7}
	if dates := yearly.occurrencesBetween(day(2026, 1, 1), day(2027, 12, 31)); len(dates) != 2 || dates[1] != day(2027, 7, 10) {
		t.Fatalf("yearly = %v", dates)
	}
}

func TestRecurringMatching(t *testing.T) {
	rent := RecurringItem{Kind: RecurringBill, Amount: money(8_000_000), Match: "Landlord", Category: "Rent", Account: "Main Bank"}
	paid := flow(day(2026, 10, 1), "Transfer to LANDLORD Ltd", "Rent", CategoryTypeExpense, 8_000_000)
	if !rent.Matches(paid) {
		t.Fatal("payee containing the match text, same category and account, should match")
	}
	other := paid
	other.AccountName = "Wallet"
	if rent.Matches(other) {
		t.Fatal("another account must not match")
	}
	partial := flow(day(2026, 10, 1), "Landlords Union", "Rent", CategoryTypeExpense, 8_000_000)
	if rent.Matches(partial) {
		t.Fatal("the match text must be whole words")
	}
	deposit := flow(day(2026, 10, 1), "Landlord", "Rent", CategoryTypeIncome, 8_000_000)
	if rent.Matches(deposit) {
		t.Fatal("a deposit does not pay a bill")
	}
	lent := paid
	lent.CategoryNature = NatureLending
	if rent.Matches(lent) {
		t.Fatal("lending never matches")
	}
	byCategory := RecurringItem{Kind: RecurringBill, Amount: money(1_000_000), Category: "Internet"}
	if !byCategory.Matches(flow(day(2026, 10, 3), "", "internet", CategoryTypeExpense, 1_100_000)) {
		t.Fatal("without match text, the category and an amount within 15% match")
	}
	if byCategory.Matches(flow(day(2026, 10, 3), "", "Internet", CategoryTypeExpense, 2_000_000)) {
		t.Fatal("an amount far from the item's does not match")
	}
	salary := RecurringItem{Kind: RecurringIncome, Amount: money(20_000_000), Match: "employer a"}
	if !salary.Matches(flow(day(2026, 10, 25), "Employer Á", "Salary", CategoryTypeIncome, 20_000_000)) {
		t.Fatal("income matches deposits, accents ignored")
	}
}
