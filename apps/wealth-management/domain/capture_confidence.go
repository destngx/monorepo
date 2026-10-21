package domain

import (
	"fmt"
	"math"
	"slices"
	"strings"
)

// Overall confidence (WM-CAP-26): one number per suggestion for how safe it
// is to approve as it is, with the reasons behind it. It starts from how
// sure the model was of its reading and loses points for what still needs
// a look; only suggestions above CaptureSelectScore start selected.

// CaptureReason explains part of a suggestion's overall confidence. Effect
// is the points it added or took away; 0 is context that changed nothing.
type CaptureReason struct {
	Text   string `json:"text"`
	Effect int    `json:"effect,omitempty"`
}

const (
	// CaptureSelectScore is the overall confidence a suggestion must beat
	// to start selected.
	CaptureSelectScore = 80

	confidenceNoAccount      = 40
	confidenceGuessedAccount = 15
	confidenceScreenAccount  = 5
	confidenceDateAssumed    = 20
	confidenceDateInferred   = 10
	confidenceDateHidden     = 20
	confidenceCurrency       = 25
	confidencePossibleMatch  = 15
	confidenceSameReference  = 20
	confidencePayee          = 10
	confidenceCategory       = 10
)

// ApplyCaptureConfidence sets each suggestion's overall confidence and
// reasons, and selects it only when the confidence is above
// CaptureSelectScore. Run it after ledger dedup and categorising.
func ApplyCaptureConfidence(suggestions []CaptureSuggestion) {
	for index := range suggestions {
		suggestion := &suggestions[index]
		reading := int(math.Round(suggestion.Confidence * 100))
		score := reading
		reasons := []CaptureReason{{Text: fmt.Sprintf("Read from the material with %d%% confidence", reading), Effect: reading}}
		deduct := func(points int, text string) {
			score -= points
			reasons = append(reasons, CaptureReason{Text: text, Effect: -points})
		}
		if duplicate := suggestion.Duplicate; duplicate != nil {
			where := fmt.Sprintf("%s (%s, %s), %d%% match", duplicate.TransactionID, duplicate.Payee, duplicate.Date, duplicate.Score)
			if duplicate.Account != "" {
				where = fmt.Sprintf("%s (%s, %s) on %s, %d%% match", duplicate.TransactionID, duplicate.Payee, duplicate.Date, duplicate.Account, duplicate.Score)
			}
			if duplicate.Likely {
				capped := min(score, 100-duplicate.Score)
				reasons = append(reasons, CaptureReason{Text: "Likely already in the sheet as " + where, Effect: capped - score})
				score = capped
			} else {
				if duplicate.Days > 0 {
					where += fmt.Sprintf(", %s apart", plural(duplicate.Days, "day", "days"))
				}
				deduct(confidencePossibleMatch, "Possibly in the sheet as "+where)
			}
		}
		if same := suggestion.SameReference; same != nil {
			deduct(confidenceSameReference, fmt.Sprintf("Same reference as %s (%s, %s) with another amount or date: check which one is right", same.TransactionID, same.Payee, same.Date))
		}
		switch {
		case suggestion.AccountSource == MatchSourceLedger && suggestion.Duplicate != nil && suggestion.Duplicate.Likely:
			reasons = append(reasons, CaptureReason{Text: "Account taken from the sheet row it matches"})
		case suggestion.AccountSource == MatchSourceLedger:
			deduct(confidenceGuessedAccount, "Account taken from a possible match in the sheet: check it")
		case suggestion.AccountSource == MatchSourceScreen:
			deduct(confidenceScreenAccount, fmt.Sprintf("Account from the screen label %q", suggestion.AccountCue))
		case suggestion.Transaction.AccountName == "":
			deduct(confidenceNoAccount, "No account: pick one before approving")
		case suggestion.AccountSource == MatchSourceName:
			deduct(confidenceGuessedAccount, "Account guessed from its name")
		}
		if suggestion.DateAssumed {
			deduct(confidenceDateAssumed, "No date on the material: dated today")
		}
		switch suggestion.DateCheck {
		case CaptureDateInferred:
			deduct(confidenceDateInferred, "Date worked out from the day headers around it: check it")
		case CaptureDateHidden:
			deduct(confidenceDateHidden, "Its day header is hidden: check the date")
		}
		if suggestion.CurrencyMismatch {
			deduct(confidenceCurrency, "The amount is in another currency than the account")
		}
		if slices.Contains(suggestion.LowConfidence, "payee") {
			text := "Payee to check"
			if learned := suggestion.LearnedPayee; learned != nil && len(learned.Others) > 0 {
				text += ": also written as " + strings.Join(learned.Others, ", ")
			}
			deduct(confidencePayee, text)
		}
		if slices.Contains(suggestion.LowConfidence, "category") {
			deduct(confidenceCategory, "No category found")
		}
		if learned := suggestion.LearnedPayee; learned != nil {
			text := fmt.Sprintf("Payee learned: %s is %s in CaptureRules", learned.Counterparty, suggestion.Transaction.Payee)
			if learned.Count > 0 {
				times := "once"
				if learned.Count > 1 {
					times = fmt.Sprintf("%d times", learned.Count)
				}
				text = fmt.Sprintf("Payee learned: %s was written as %s %s before", learned.Counterparty, suggestion.Transaction.Payee, times)
			}
			reasons = append(reasons, CaptureReason{Text: text})
		}
		if suggestion.RecordedThrough != "" {
			reasons = append(reasons, CaptureReason{Text: fmt.Sprintf("May be missing from the sheet: its account is recorded up to %s and no row matches", suggestion.RecordedThrough)})
		}
		suggestion.Overall = max(0, min(100, score))
		suggestion.Reasons = reasons
		suggestion.Selected = suggestion.Overall > CaptureSelectScore
	}
}
