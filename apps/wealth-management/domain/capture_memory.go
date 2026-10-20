package domain

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

// Capture memory (WM-CAP-23): what Smart Capture read for each row the user
// wrote, kept so later captures fill in the payee the user uses for a
// counterparty ("NGUYEN VAN A" is the banh mi stall) and dedup can match
// rows entered by hand under that name.

type CaptureMemorySource string

const (
	// CaptureMemoryApproved is a row the user approved or committed.
	CaptureMemoryApproved CaptureMemorySource = "approved"
	// CaptureMemoryDuplicate is a draft rejected as a likely duplicate:
	// the sheet row it matched names the counterparty.
	CaptureMemoryDuplicate CaptureMemorySource = "duplicate"
)

type CaptureMemory struct {
	ID        string              `json:"id"`
	CreatedAt time.Time           `json:"createdAt"`
	Source    CaptureMemorySource `json:"source"`
	// Counterparty is the name as the material showed it; CounterpartyKey
	// is what it is matched by.
	Counterparty    string           `json:"counterparty"`
	CounterpartyKey string           `json:"counterpartyKey"`
	SourceApp       string           `json:"sourceApp,omitempty"`
	Quote           string           `json:"quote,omitempty"`
	Reference       string           `json:"reference,omitempty"`
	Account         string           `json:"account"`
	Direction       CaptureDirection `json:"direction"`
	Amount          float64          `json:"amount"`
	Date            string           `json:"date"`
	// Payee, Category and Tags are what the user wrote.
	Payee         string   `json:"payee"`
	Category      string   `json:"category,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	TransactionID string   `json:"transactionId,omitempty"`
}

// CaptureLearnedPayee explains a payee filled in from a counterparty rule:
// Count earlier rows were written under it, Others are the other payees the
// same counterparty was written under.
type CaptureLearnedPayee struct {
	Counterparty string   `json:"counterparty"`
	Count        int      `json:"count"`
	Others       []string `json:"others,omitempty"`
}

// CaptureWrittenRow is a reviewed row the Smart Capture page wrote to the
// sheet, sent back so the engine can learn from it.
type CaptureWrittenRow struct {
	Suggestion    CaptureSuggestion      `json:"suggestion"`
	Transaction   TransactionCreateInput `json:"transaction"`
	TransactionID string                 `json:"transactionId"`
}

func ValidateCaptureWrittenRows(rows []CaptureWrittenRow) error {
	var problems []string
	if len(rows) == 0 {
		problems = append(problems, "at least one row is required")
	}
	// One learn request follows one batch write.
	if len(rows) > MaxTransactionBatch {
		problems = append(problems, fmt.Sprintf("at most %d rows, got %d", MaxTransactionBatch, len(rows)))
	}
	for index, row := range rows {
		if strings.TrimSpace(row.TransactionID) == "" {
			problems = append(problems, fmt.Sprintf("row %d: transactionId is required", index+1))
		}
	}
	return validationError(problems)
}

// CaptureLearnResult counts what learning stored: memories and the
// counterparty rules saved to CaptureRules.
type CaptureLearnResult struct {
	Remembered int                    `json:"remembered"`
	Rules      CaptureRulesSaveResult `json:"rules"`
}

const captureUnknownPayee = "Unknown"

// CounterpartyKey folds a counterparty name for matching: no accents or
// case, no words with digits (account numbers, reference codes) and no
// generic transfer words, so "CT Nguyễn Văn A 0123" and "NGUYEN VAN A" both
// give "nguyen van a". Single letters stay: they are part of names.
func CounterpartyKey(text string) string {
	var words []string
	for _, word := range strings.Fields(FoldText(text)) {
		if captureGenericWords[word] || strings.ContainsFunc(word, unicode.IsDigit) {
			continue
		}
		words = append(words, word)
	}
	return strings.Join(words, " ")
}

// CaptureCounterparty is the counterparty a suggestion teaches about, empty
// for own-account transfers. Drafts saved before the counterparty was kept
// fall back to the payee the engine read.
func CaptureCounterparty(suggestion CaptureSuggestion) string {
	if suggestion.TransferGroup != "" {
		return ""
	}
	if suggestion.Counterparty != "" {
		return suggestion.Counterparty
	}
	if suggestion.Fee || suggestion.Transaction.Payee == captureUnknownPayee {
		return ""
	}
	return suggestion.Transaction.Payee
}

// NewCaptureMemory records a written row: false when the suggestion has no
// counterparty to learn from or nothing was written as payee.
func NewCaptureMemory(suggestion CaptureSuggestion, written TransactionCreateInput, transactionID string, source CaptureMemorySource, at time.Time, id string) (CaptureMemory, bool) {
	counterparty := strings.TrimSpace(CaptureCounterparty(suggestion))
	key := CounterpartyKey(counterparty)
	payee := strings.TrimSpace(written.Payee)
	if key == "" || payee == "" {
		return CaptureMemory{}, false
	}
	quotes := make([]string, 0, len(suggestion.Evidence))
	for _, evidence := range suggestion.Evidence {
		if quote := strings.TrimSpace(evidence.Quote); quote != "" {
			quotes = append(quotes, quote)
		}
	}
	reference := ""
	if suggestion.Transaction.ReferenceNumber != nil {
		reference = strings.TrimSpace(*suggestion.Transaction.ReferenceNumber)
	}
	return CaptureMemory{
		ID: id, CreatedAt: at, Source: source,
		Counterparty: counterparty, CounterpartyKey: key, SourceApp: suggestion.SourceApp,
		Quote: strings.Join(uniqueStrings(quotes), " | "), Reference: reference,
		Account: written.AccountName, Direction: suggestion.Direction, Amount: suggestion.Amount, Date: written.Date,
		Payee: payee, Category: written.Category, Tags: written.Tags, TransactionID: transactionID,
	}, true
}

// CounterpartyRuleUpdates are the counterparty rules to save after rows
// were written, one confirmation being enough: a counterparty written under
// another name is learned, and an existing rule follows the latest payee,
// back to the material's own name too. Later memories win. Rules whose
// value would not change are left out.
func CounterpartyRuleUpdates(memories []CaptureMemory, existing []CaptureRule) []CaptureRule {
	current := map[string]string{}
	for _, rule := range existing {
		if rule.Kind == CaptureRuleCounterparty {
			current[rule.Key()] = rule.Value
		}
	}
	var order []string
	updates := map[string]CaptureRule{}
	for _, memory := range memories {
		rule := CaptureRule{Kind: CaptureRuleCounterparty, Pattern: memory.CounterpartyKey, Value: memory.Payee}
		key := rule.Key()
		value, known := current[key]
		if update, pending := updates[key]; pending {
			value, known = update.Value, true
		}
		renamed := FoldText(memory.Payee) != FoldText(memory.Counterparty)
		if memory.CounterpartyKey == "" || (!known && !renamed) || (known && value == memory.Payee) {
			continue
		}
		if _, pending := updates[key]; !pending {
			order = append(order, key)
		}
		updates[key] = rule
	}
	out := make([]CaptureRule, 0, len(order))
	for _, key := range order {
		if updates[key].Value != current[key] {
			out = append(out, updates[key])
		}
	}
	return out
}

// ApplyLearnedPayees fills in the payee the user writes for a suggestion's
// counterparty from the counterparty rules, before categorising so the
// payee's own rules and history pick the category. Memories give how many
// rows were written under the payee and the other payees used for the same
// counterparty; any other flags the payee to check.
func ApplyLearnedPayees(suggestions []CaptureSuggestion, rules []CaptureRule, memories []CaptureMemory) {
	learned := map[string]string{}
	for _, rule := range rules {
		if rule.Kind == CaptureRuleCounterparty {
			learned[CounterpartyKey(rule.Pattern)] = rule.Value
		}
	}
	if len(learned) == 0 {
		return
	}
	for index := range suggestions {
		suggestion := &suggestions[index]
		counterparty := CaptureCounterparty(*suggestion)
		key := CounterpartyKey(counterparty)
		payee, ok := learned[key]
		if key == "" || !ok {
			continue
		}
		info := &CaptureLearnedPayee{Counterparty: counterparty}
		for _, memory := range memories {
			if memory.CounterpartyKey != key {
				continue
			}
			if FoldText(memory.Payee) == FoldText(payee) {
				info.Count++
			} else if !containsFolded(info.Others, memory.Payee) {
				info.Others = append(info.Others, memory.Payee)
			}
		}
		suggestion.Transaction.Payee = payee
		suggestion.LearnedPayee = info
		if len(info.Others) > 0 {
			suggestion.LowConfidence = uniqueStrings(append(suggestion.LowConfidence, "payee"))
		}
	}
}

func containsFolded(values []string, value string) bool {
	for _, other := range values {
		if FoldText(other) == FoldText(value) {
			return true
		}
	}
	return false
}
