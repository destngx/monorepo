package domain

import (
	"strings"
	"testing"
)

func TestGivenScreenRulesWhenAskingForTheFormatThenTheModelIsAskedWhichLabelsItSees(t *testing.T) {
	instruction := CaptureFormatInstruction(captureToday, []string{"Tải sao kê"})
	if !strings.Contains(instruction, `"screenCues"`) || !strings.Contains(instruction, `"Tải sao kê"`) {
		t.Fatalf("expected the labels to look for: %q", instruction)
	}
	if strings.Contains(CaptureFormatInstruction(captureToday, nil), "screenCues") {
		t.Fatal("without screen rules the format has no screenCues")
	}
}

func TestGivenAScreenLabelWhenMatchingTheAccountThenTheRuleNamesIt(t *testing.T) {
	accounts := []Account{{Name: "Techcombank", Type: AccountTypeBank}, {Name: "Old bank", Type: AccountTypeDeprecated}}
	rules := []CaptureRule{
		{Kind: CaptureRuleScreen, Pattern: "Tải sao kê", Value: "Techcombank"},
		{Kind: CaptureRuleScreen, Pattern: "Old button", Value: "Old bank"},
		{Kind: CaptureRuleAlias, Pattern: "Xem chi tiết", Value: "Techcombank"},
	}

	match, ok := MatchCaptureScreenCue([]string{"tai sao ke"}, accounts, rules)
	if !ok || match.Name != "Techcombank" || match.Source != MatchSourceScreen || match.Cue != "Tải sao kê" {
		t.Fatalf("got %+v", match)
	}
	for _, cues := range [][]string{{"Old button"}, {"Xem chi tiết"}, {}} {
		if match, ok := MatchCaptureScreenCue(cues, accounts, rules); ok {
			t.Fatalf("%v: a deprecated account or another rule kind matches nothing: %+v", cues, match)
		}
	}
	if err := (CaptureRule{Kind: CaptureRuleScreen, Pattern: "Tải sao kê", Value: "Techcombank"}).Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestGivenAnAccountFromAScreenLabelWhenScoringThenItCostsALittle(t *testing.T) {
	suggestion := confident(0.95)
	suggestion.AccountSource, suggestion.AccountCue = MatchSourceScreen, "Tải sao kê"
	suggestions := []CaptureSuggestion{suggestion}

	ApplyCaptureConfidence(suggestions)

	if got := suggestions[0]; got.Overall != 90 || !strings.Contains(reasonTexts(got), `Account from the screen label "Tải sao kê"`) {
		t.Fatalf("overall=%d reasons=%q", got.Overall, reasonTexts(got))
	}
}

func TestGivenAnAccountFromAPossibleMatchWhenScoringThenItIsAGuessToCheck(t *testing.T) {
	suggestion := confident(0.95)
	suggestion.AccountSource = MatchSourceLedger
	suggestion.Duplicate = &CaptureDuplicate{TransactionID: "r2", Date: "15/10/2026", Payee: "Gym membership", Score: 70, Days: 2, Account: "Vietcombank"}
	suggestions := []CaptureSuggestion{suggestion}

	ApplyCaptureConfidence(suggestions)

	// 95 - 15 possible match - 15 account to check.
	if got := suggestions[0]; got.Overall != 65 || !strings.Contains(reasonTexts(got), "Account taken from a possible match in the sheet: check it") {
		t.Fatalf("overall=%d reasons=%q", got.Overall, reasonTexts(got))
	}
}
