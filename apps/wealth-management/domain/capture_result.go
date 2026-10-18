package domain

import (
	"fmt"
	"strings"
	"time"
)

// Smart Capture output: suggested transactions for the review screen
// (WM-CAP-8). Nothing here writes; commits go through the batch create path.

// Category sources on a suggestion.
const (
	CategorySourceRule     = "rule"
	CategorySourceHistory  = "history"
	CategorySourceAI       = "ai"
	CategorySourceTransfer = "transfer"
	CategorySourceDefault  = "default"
	// CategorySourceManual is a category the user picked on a draft.
	CategorySourceManual = "manual"
)

// CaptureMemoPrefix marks rows created by Smart Capture (WM-CAP-9).
const CaptureMemoPrefix = "[capture]"

type CaptureDuplicate struct {
	TransactionID string  `json:"transactionId"`
	Date          string  `json:"date"`
	Payee         string  `json:"payee"`
	Amount        float64 `json:"amount"`
	Reference     string  `json:"reference,omitempty"`
}

type CaptureSuggestion struct {
	ID       string `json:"id"`
	Selected bool   `json:"selected"`
	// Transaction is the editable row, ready for the batch create path.
	Transaction       TransactionCreateInput `json:"transaction"`
	Direction         CaptureDirection       `json:"direction"`
	Amount            float64                `json:"amount"`
	Currency          string                 `json:"currency"`
	OccurredAt        string                 `json:"occurredAt"`
	Confidence        float64                `json:"confidence"`
	LowConfidence     []string               `json:"lowConfidence"`
	AccountHint       string                 `json:"accountHint,omitempty"`
	AccountSource     string                 `json:"accountSource,omitempty"`
	AccountCandidates []string               `json:"accountCandidates,omitempty"`
	CategorySource    string                 `json:"categorySource,omitempty"`
	// TagOptions are the cashback rule tags of the row's card (WM-CAP-5).
	TagOptions []string          `json:"tagOptions,omitempty"`
	SourceApp  string            `json:"sourceApp,omitempty"`
	Evidence   []CaptureEvidence `json:"evidence"`
	Duplicate  *CaptureDuplicate `json:"duplicate,omitempty"`
	// RecordedThrough is the account's last recorded date in the sheet
	// (dd/mm/yyyy) when the row is dated on or before it without an exact
	// match: the sheet is already filled in up to then, so the row is
	// possibly recorded already.
	RecordedThrough string   `json:"recordedThrough,omitempty"`
	TransferGroup   string   `json:"transferGroup,omitempty"`
	Notes           []string `json:"notes,omitempty"`
}

// CaptureBalance is the latest balance a material showed for an account,
// with the sheet balance to reconcile against (WM-CAP-7).
type CaptureBalance struct {
	Account      string  `json:"account"`
	Currency     string  `json:"currency"`
	BalanceAfter float64 `json:"balanceAfter"`
	AsOf         string  `json:"asOf"`
	MaterialID   string  `json:"materialId"`
	SheetBalance float64 `json:"sheetBalance"`
}

type CaptureUnreadable struct {
	MaterialID string `json:"materialId"`
	Reason     string `json:"reason"`
}

type CaptureResult struct {
	Suggestions []CaptureSuggestion `json:"suggestions"`
	Unreadable  []CaptureUnreadable `json:"unreadable"`
	Balances    []CaptureBalance    `json:"balances"`
}

// CaptureBuildInput is everything BuildCaptureSuggestions needs.
type CaptureBuildInput struct {
	Items     []CaptureItem
	Materials []CaptureMaterial
	Accounts  []Account
	// TransferCategory is used for paired own-account transfers; empty
	// leaves them to the categoriser.
	TransferCategory string
}

const captureTimeLayout = "2006-01-02 15:04"

// BuildCaptureSuggestions turns consolidated items into editable rows:
// payee, memo with the capture source, amounts by direction, transfer
// pairs, separate fee rows and low-confidence flags. Categories other than
// transfers are filled in afterwards.
func BuildCaptureSuggestions(input CaptureBuildInput) []CaptureSuggestion {
	materials := map[string]CaptureMaterial{}
	for _, material := range input.Materials {
		materials[material.ID] = material
	}
	accounts := map[string]Account{}
	for _, account := range input.Accounts {
		accounts[account.Name] = account
	}
	transferOf := map[int]string{}
	counterpartOf := map[int]int{}
	for index, pair := range PairCaptureTransfers(input.Items) {
		group := fmt.Sprintf("t%d", index+1)
		transferOf[pair[0]], transferOf[pair[1]] = group, group
		counterpartOf[pair[0]], counterpartOf[pair[1]] = pair[1], pair[0]
	}

	suggestions := make([]CaptureSuggestion, 0, len(input.Items))
	add := func(suggestion CaptureSuggestion) {
		suggestion.ID = fmt.Sprintf("s%d", len(suggestions)+1)
		suggestions = append(suggestions, suggestion)
	}
	for index, item := range input.Items {
		event := item.Event
		suggestion := baseSuggestion(item, materials, accounts, event.Amount)
		suggestion.Transaction.Payee = capturePayee(event)
		if group, ok := transferOf[index]; ok {
			suggestion.TransferGroup = group
			suggestion.Transaction.Payee = input.Items[counterpartOf[index]].Account.Name
			if input.TransferCategory != "" {
				suggestion.Transaction.Category = input.TransferCategory
				suggestion.CategorySource = CategorySourceTransfer
			}
		}
		add(suggestion)
		if event.Fee != nil {
			fee := baseSuggestion(item, materials, accounts, *event.Fee)
			fee.Direction = CaptureOut
			fee.Transaction.Deposit = nil
			fee.Transaction.Payment = float64Pointer(*event.Fee)
			fee.Transaction.Payee = strings.TrimSpace(firstNonEmpty(event.SourceApp, item.Account.Name) + " fee")
			fee.Notes = append(fee.Notes, "Fee shown separately on the material.")
			add(fee)
		}
	}
	return suggestions
}

func baseSuggestion(item CaptureItem, materials map[string]CaptureMaterial, accounts map[string]Account, amount float64) CaptureSuggestion {
	event := item.Event
	transaction := TransactionCreateInput{
		AccountName: item.Account.Name,
		Date:        event.Date().Format(SheetDateLayout),
		Tags:        []string{},
		Memo:        captureMemo(item, materials),
		// Captured from a bank or wallet record, so already settled; manual
		// entry defaults to cleared too.
		Cleared: true,
	}
	if event.Reference != "" {
		transaction.ReferenceNumber = stringPointer(event.Reference)
	}
	if event.Direction == CaptureIn {
		transaction.Deposit = float64Pointer(amount)
	} else {
		transaction.Payment = float64Pointer(amount)
	}
	occurredAt := event.At.Format(captureTimeLayout)
	if !event.HasTime {
		occurredAt = event.At.Format("2006-01-02")
	}
	suggestion := CaptureSuggestion{
		Selected:          true,
		Transaction:       transaction,
		Direction:         event.Direction,
		Amount:            amount,
		Currency:          event.Currency,
		OccurredAt:        occurredAt,
		Confidence:        event.Confidence,
		LowConfidence:     []string{},
		AccountHint:       event.AccountHint,
		AccountSource:     item.Account.Source,
		AccountCandidates: item.Account.Candidates,
		SourceApp:         event.SourceApp,
		Evidence:          item.Evidence,
	}
	if event.Confidence < CaptureLowConfidence {
		suggestion.LowConfidence = append(suggestion.LowConfidence, "amount", "date")
	}
	if !event.HasTime {
		suggestion.Notes = append(suggestion.Notes, "No time on the material; matched by date only.")
	}
	if item.Account.Name == "" || item.Account.Source == MatchSourceName {
		suggestion.LowConfidence = append(suggestion.LowConfidence, "account")
	}
	if account, ok := accounts[item.Account.Name]; ok && account.Currency != "" && string(account.Currency) != event.Currency {
		suggestion.LowConfidence = append(suggestion.LowConfidence, "amount")
		suggestion.Notes = append(suggestion.Notes, fmt.Sprintf("Material amount is in %s but the account is in %s.", event.Currency, account.Currency))
	}
	suggestion.LowConfidence = uniqueStrings(suggestion.LowConfidence)
	return suggestion
}

func capturePayee(event CaptureEvent) string {
	return firstNonEmpty(event.Counterparty, event.Description, event.SourceApp, "Unknown")
}

// captureMemo records where the row came from, e.g. "[capture] MoMo
// screenshot - Chuyen tien an trua".
func captureMemo(item CaptureItem, materials map[string]CaptureMaterial) *string {
	labels := []string{}
	for _, evidence := range item.Evidence {
		if material, ok := materials[evidence.MaterialID]; ok {
			labels = append(labels, material.Label())
		}
	}
	memo := strings.TrimSpace(strings.Join([]string{CaptureMemoPrefix, item.Event.SourceApp, strings.Join(uniqueStrings(labels), " + ")}, " "))
	memo = strings.Join(strings.Fields(memo), " ")
	if item.Event.Description != "" && item.Event.Description != item.Event.Counterparty {
		memo += " - " + item.Event.Description
	}
	return &memo
}

// ApplyCaptureDuplicates marks suggestions likely already in the ledger
// (WM-CAP-6), then the rest dated on or before their account's last
// recorded date (WM-CAP-21), and unselects both.
func ApplyCaptureDuplicates(suggestions []CaptureSuggestion, ledger []Transaction, today time.Time) {
	candidates := make([]CaptureCandidate, len(suggestions))
	for index, suggestion := range suggestions {
		date, _ := ParseSheetDate(suggestion.Transaction.Date)
		reference := ""
		if suggestion.Transaction.ReferenceNumber != nil {
			reference = *suggestion.Transaction.ReferenceNumber
		}
		candidates[index] = CaptureCandidate{Account: suggestion.Transaction.AccountName, Direction: suggestion.Direction, Amount: suggestion.Amount, Date: date, Reference: reference}
	}
	for index, row := range FindCaptureDuplicates(candidates, ledger) {
		amount := row.PaymentAmount()
		if suggestions[index].Direction == CaptureIn {
			amount = row.DepositAmount()
		}
		suggestions[index].Duplicate = &CaptureDuplicate{TransactionID: row.ID, Date: row.Date.Format(SheetDateLayout), Payee: row.Payee, Amount: amount, Reference: referenceOf(row)}
		suggestions[index].Selected = false
	}
	lastRecorded := LastRecordedDates(ledger, today)
	for index := range suggestions {
		suggestion := &suggestions[index]
		last, found := lastRecorded[suggestion.Transaction.AccountName]
		if suggestion.Duplicate != nil || !found {
			continue
		}
		if date, err := ParseSheetDate(suggestion.Transaction.Date); err == nil && !date.After(last) {
			suggestion.RecordedThrough = last.Format(SheetDateLayout)
			suggestion.Selected = false
		}
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}

func float64Pointer(value float64) *float64 { return &value }

func stringPointer(value string) *string { return &value }
