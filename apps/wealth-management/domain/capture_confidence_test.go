package domain

import (
	"strings"
	"testing"
)

func confident(reading float64) CaptureSuggestion {
	return CaptureSuggestion{
		Confidence: reading, Direction: CaptureOut, Amount: 40000, AccountSource: MatchSourceAlias, LowConfidence: []string{},
		Transaction: TransactionCreateInput{AccountName: "Vietcombank", Date: "16/10/2026", Payee: "Com tam"},
	}
}

func reasonTexts(suggestion CaptureSuggestion) string {
	texts := make([]string, len(suggestion.Reasons))
	for index, reason := range suggestion.Reasons {
		texts[index] = reason.Text
	}
	return strings.Join(texts, " | ")
}

func TestApplyCaptureConfidence(t *testing.T) {
	noAccount := confident(0.9)
	noAccount.Transaction.AccountName, noAccount.AccountSource = "", ""
	guessed := confident(0.95)
	guessed.AccountSource = MatchSourceName
	assumed := confident(0.95)
	assumed.DateAssumed = true
	currency := confident(0.95)
	currency.CurrencyMismatch = true
	payee := confident(0.95)
	payee.LowConfidence = []string{"payee"}
	uncategorised := confident(0.95)
	uncategorised.LowConfidence = []string{"category"}
	likely := confident(0.95)
	likely.Duplicate = &CaptureDuplicate{TransactionID: "row-9", Date: "16/10/2026", Payee: "Com tam", Score: 90, Likely: true}
	possible := confident(0.95)
	possible.Duplicate = &CaptureDuplicate{TransactionID: "row-8", Date: "12/10/2026", Payee: "Com tam", Score: 65, Days: 4}
	many := confident(0.48)
	many.Transaction.AccountName, many.AccountSource = "", ""
	many.DateAssumed = true

	tests := []struct {
		name       string
		suggestion CaptureSuggestion
		overall    int
		selected   bool
		reason     string
	}{
		{"a confident reading with nothing to check", confident(0.95), 95, true, "Read from the material with 95% confidence"},
		{"exactly 80 is not above 80", confident(0.8), 80, false, ""},
		{"no account", noAccount, 50, false, "No account: pick one before approving"},
		{"an account guessed from its name", guessed, 80, false, "Account guessed from its name"},
		{"no date on the material", assumed, 75, false, "No date on the material: dated today"},
		{"another currency", currency, 70, false, "The amount is in another currency than the account"},
		{"a payee to check", payee, 85, true, "Payee to check"},
		{"no category", uncategorised, 85, true, "No category found"},
		{"a likely duplicate caps it at 100 minus the match", likely, 10, false, "Likely already in the sheet as row-9 (Com tam, 16/10/2026), 90% match"},
		{"a possible match", possible, 80, false, "Possibly in the sheet as row-8 (Com tam, 12/10/2026), 65% match, 4 days apart"},
		{"never below zero", many, 0, false, "No date on the material: dated today"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			suggestions := []CaptureSuggestion{test.suggestion}
			ApplyCaptureConfidence(suggestions)
			got := suggestions[0]
			if got.Overall != test.overall || got.Selected != test.selected || !strings.Contains(reasonTexts(got), test.reason) {
				t.Fatalf("overall=%d selected=%v reasons=%q, want %d %v containing %q", got.Overall, got.Selected, reasonTexts(got), test.overall, test.selected, test.reason)
			}
		})
	}
}

func TestCaptureConfidenceReasonsSayHowMuchEachOneCost(t *testing.T) {
	suggestion := confident(0.9)
	suggestion.Transaction.AccountName, suggestion.AccountSource = "", ""
	suggestion.RecordedThrough = "15/10/2026"
	suggestion.LearnedPayee = &CaptureLearnedPayee{Counterparty: "NGUYEN VAN A", Count: 2}
	suggestion.Transaction.Payee = "Banh mi stall"
	suggestions := []CaptureSuggestion{suggestion}

	ApplyCaptureConfidence(suggestions)

	want := []CaptureReason{
		{Text: "Read from the material with 90% confidence", Effect: 90},
		{Text: "No account: pick one before approving", Effect: -40},
		{Text: "Payee learned: NGUYEN VAN A was written as Banh mi stall 2 times before"},
		{Text: "May be missing from the sheet: its account is recorded up to 15/10/2026 and no row matches"},
	}
	got := suggestions[0].Reasons
	if len(got) != len(want) {
		t.Fatalf("reasons = %+v", got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("reason %d = %+v, want %+v", index, got[index], want[index])
		}
	}
}
