package domain

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"time"
)

// Recurring items (ROADMAP R4, WM-BILL-1): salary, rent, utilities,
// subscriptions, insurance and card payments, kept in the app-managed
// `Recurring` tab (REQUIREMENTS §2). Items are suggested from history
// (recurring_detect.go) and confirmed by the user; bills.go lays them out
// on a calendar and derives safe-to-spend.

// RecurringKind is the Recurring tab's Kind column (B).
type RecurringKind string

const (
	RecurringIncome       RecurringKind = "income"
	RecurringBill         RecurringKind = "bill"
	RecurringSubscription RecurringKind = "subscription"
	RecurringInsurance    RecurringKind = "insurance"
	RecurringCard         RecurringKind = "card"
)

// RecurringKinds lists every kind a Recurring row can have.
var RecurringKinds = []RecurringKind{RecurringIncome, RecurringBill, RecurringSubscription, RecurringInsurance, RecurringCard}

// IsIncome reports whether the item brings money in.
func (kind RecurringKind) IsIncome() bool { return kind == RecurringIncome }

// RecurringFrequency is the Frequency column (D).
type RecurringFrequency string

const (
	FrequencyMonthly   RecurringFrequency = "monthly"
	FrequencyQuarterly RecurringFrequency = "quarterly"
	FrequencyYearly    RecurringFrequency = "yearly"
)

// RecurringFrequencies lists every frequency.
var RecurringFrequencies = []RecurringFrequency{FrequencyMonthly, FrequencyQuarterly, FrequencyYearly}

// RecurringStatus is the Status column (J).
type RecurringStatus string

const (
	RecurringActive RecurringStatus = "active"
	RecurringPaused RecurringStatus = "paused"
)

// RecurringStatuses lists every status.
var RecurringStatuses = []RecurringStatus{RecurringActive, RecurringPaused}

// MaxGraceDays caps the Grace column (L, D42). A monthly item with grace
// settles payments from its date + grace + 1 a month back, so a cap of
// three weeks still lets a payment come a week early.
const MaxGraceDays = 21

// RecurringItem is one row of the Recurring tab. Amount is nil only on a
// card row, whose payment is the card's statement.
type RecurringItem struct {
	// ID is "recurring-<sheet row>"; with ETag it addresses the row.
	ID        string             `json:"id"`
	Row       int                `json:"row"`
	Name      string             `json:"name"`
	Kind      RecurringKind      `json:"kind"`
	Amount    *float64           `json:"amount"`
	Frequency RecurringFrequency `json:"frequency"`
	// Day is the day of month (1-31, clamped to short months); 0 on card rows.
	Day int `json:"day"`
	// Month (1-12) is the first month of a quarterly item and the month of
	// a yearly one; 0 for monthly items.
	Month    int             `json:"month,omitempty"`
	Account  string          `json:"account,omitempty"`
	Category string          `json:"category,omitempty"`
	Match    string          `json:"match,omitempty"`
	Status   RecurringStatus `json:"status"`
	Note     string          `json:"note,omitempty"`
	// GraceDays is how many days after its date an unpaid occurrence is
	// still due rather than overdue (D42): rent the user or the landlord
	// may settle a week or two late. 0 = overdue the day after.
	GraceDays int `json:"graceDays,omitempty"`
	// ETag fingerprints columns A-L as read.
	ETag string `json:"etag"`
}

// RecurringIssue is a Recurring row the reader rejected, or a row that
// cannot be used, with the reason.
type RecurringIssue struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

// RecurringSheet is the Recurring tab as read; Exists is false until the
// first item is saved.
type RecurringSheet struct {
	Exists bool             `json:"exists"`
	Items  []RecurringItem  `json:"items"`
	Issues []RecurringIssue `json:"issues"`
}

// RecurringInput creates or replaces a Recurring row. An empty Frequency is
// monthly and an empty Status active.
type RecurringInput struct {
	Name      string             `json:"name"`
	Kind      RecurringKind      `json:"kind"`
	Amount    *float64           `json:"amount,omitempty"`
	Frequency RecurringFrequency `json:"frequency,omitempty"`
	Day       int                `json:"day,omitempty"`
	Month     int                `json:"month,omitempty"`
	Account   string             `json:"account,omitempty"`
	Category  string             `json:"category,omitempty"`
	Match     string             `json:"match,omitempty"`
	Status    RecurringStatus    `json:"status,omitempty"`
	Note      string             `json:"note,omitempty"`
	GraceDays int                `json:"graceDays,omitempty"`
}

// Normalized trims the text fields and fills the defaults.
func (input RecurringInput) Normalized() RecurringInput {
	input.Name = strings.TrimSpace(input.Name)
	input.Kind = RecurringKind(strings.ToLower(strings.TrimSpace(string(input.Kind))))
	input.Frequency = RecurringFrequency(strings.ToLower(strings.TrimSpace(string(input.Frequency))))
	if input.Frequency == "" {
		input.Frequency = FrequencyMonthly
	}
	input.Status = RecurringStatus(strings.ToLower(strings.TrimSpace(string(input.Status))))
	if input.Status == "" {
		input.Status = RecurringActive
	}
	input.Account = strings.TrimSpace(input.Account)
	input.Category = strings.TrimSpace(input.Category)
	input.Match = strings.TrimSpace(input.Match)
	input.Note = strings.TrimSpace(input.Note)
	if input.Frequency == FrequencyMonthly {
		input.Month = 0
	}
	return input
}

func (input RecurringInput) Validate() error {
	return validationError(input.Normalized().Problems())
}

// Problems checks a normalized item and lists what is wrong. Card rows take
// their amount and date from the card statement, so they need neither.
func (input RecurringInput) Problems() []string {
	var problems []string
	if input.Name == "" {
		problems = append(problems, "name is required")
	}
	if !slices.Contains(RecurringKinds, input.Kind) {
		problems = append(problems, fmt.Sprintf("kind %q must be one of income, bill, subscription, insurance or card", input.Kind))
	}
	if !slices.Contains(RecurringFrequencies, input.Frequency) {
		problems = append(problems, fmt.Sprintf("frequency %q must be one of monthly, quarterly or yearly", input.Frequency))
	}
	if !slices.Contains(RecurringStatuses, input.Status) {
		problems = append(problems, fmt.Sprintf("status %q must be active or paused", input.Status))
	}
	isCard := input.Kind == RecurringCard
	switch {
	case input.Amount != nil && *input.Amount <= 0:
		problems = append(problems, "amount must be positive")
	case input.Amount == nil && !isCard:
		problems = append(problems, "amount is required (only card rows use the statement balance)")
	}
	if !isCard && (input.Day < 1 || input.Day > 31) {
		problems = append(problems, "day must be a day of the month from 1 to 31")
	}
	if isCard && input.Day != 0 && (input.Day < 1 || input.Day > 31) {
		problems = append(problems, "day must be blank or from 1 to 31")
	}
	if input.Frequency != FrequencyMonthly && (input.Month < 1 || input.Month > 12) {
		problems = append(problems, "month (1-12) is required for quarterly and yearly items")
	}
	switch {
	case isCard && input.GraceDays != 0:
		problems = append(problems, "grace days must be blank on card rows (the card's pay-by date applies)")
	case input.GraceDays < 0 || input.GraceDays > MaxGraceDays:
		problems = append(problems, fmt.Sprintf("grace days must be from 0 to %d", MaxGraceDays))
	}
	if input.Kind != RecurringCard && input.Match == "" && input.Category == "" {
		problems = append(problems, "match text or a category is needed to recognise payments")
	}
	return problems
}

// RecurringUpdateInput replaces every column of a row that still matches ETag.
type RecurringUpdateInput struct {
	RecurringInput
	ETag string `json:"etag"`
}

func (input RecurringUpdateInput) Validate() error {
	if err := input.RecurringInput.Validate(); err != nil {
		return err
	}
	return requireETag(input.ETag)
}

// RecurringStatusInput pauses or resumes a row that still matches ETag.
type RecurringStatusInput struct {
	Status RecurringStatus `json:"status"`
	ETag   string          `json:"etag"`
}

func (input RecurringStatusInput) Validate() error {
	if !slices.Contains(RecurringStatuses, input.Status) {
		return fmt.Errorf("%w: status %q must be active or paused", ErrValidation, input.Status)
	}
	return requireETag(input.ETag)
}

// RecurringConfirmInput confirms a suggestion by its key. Blank fields keep
// the suggested value; Status paused records a dismissed suggestion so it
// is not suggested again.
type RecurringConfirmInput struct {
	Key      string          `json:"key"`
	Name     string          `json:"name,omitempty"`
	Kind     RecurringKind   `json:"kind,omitempty"`
	Amount   *float64        `json:"amount,omitempty"`
	Day      int             `json:"day,omitempty"`
	Account  string          `json:"account,omitempty"`
	Category string          `json:"category,omitempty"`
	Match    string          `json:"match,omitempty"`
	Status   RecurringStatus `json:"status,omitempty"`
	Note     string          `json:"note,omitempty"`
}

func (input RecurringConfirmInput) Validate() error {
	if strings.TrimSpace(input.Key) == "" {
		return fmt.Errorf("%w: key is required", ErrValidation)
	}
	if input.Status != "" && !slices.Contains(RecurringStatuses, input.Status) {
		return fmt.Errorf("%w: status %q must be active or paused", ErrValidation, input.Status)
	}
	return nil
}

// Apply builds the item to write from the suggestion and the overrides.
func (input RecurringConfirmInput) Apply(suggestion RecurringSuggestion) RecurringInput {
	pick := func(override, suggested string) string {
		if strings.TrimSpace(override) != "" {
			return override
		}
		return suggested
	}
	amount := suggestion.Amount
	item := RecurringInput{
		Name:      pick(input.Name, suggestion.Name),
		Kind:      RecurringKind(pick(string(input.Kind), string(suggestion.Kind))),
		Amount:    &amount,
		Frequency: suggestion.Frequency,
		Day:       suggestion.Day,
		Month:     suggestion.Month,
		Account:   pick(input.Account, suggestion.Account),
		Category:  pick(input.Category, suggestion.Category),
		Match:     pick(input.Match, suggestion.Match),
		Status:    input.Status,
		Note:      input.Note,
	}
	if input.Amount != nil {
		item.Amount = input.Amount
	}
	if input.Day != 0 {
		item.Day = input.Day
	}
	return item.Normalized()
}

func requireETag(etag string) error {
	if strings.TrimSpace(etag) == "" {
		return fmt.Errorf("%w: etag is required", ErrValidation)
	}
	return nil
}

// occursIn reports whether the item falls due in the given month.
func (item RecurringItem) occursIn(month time.Month) bool {
	switch item.Frequency {
	case FrequencyQuarterly:
		return ((int(month)-item.Month)%3+3)%3 == 0
	case FrequencyYearly:
		return int(month) == item.Month
	default:
		return true
	}
}

// occurrencesBetween returns the expected dates in [from, to], the day
// clamped to the month's length.
func (item RecurringItem) occurrencesBetween(from, to time.Time) []time.Time {
	dates := []time.Time{}
	if item.Day < 1 {
		return dates
	}
	cursor := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
	for !cursor.After(to) {
		if item.occursIn(cursor.Month()) {
			date := clampedDate(cursor.Year(), cursor.Month(), item.Day)
			if !date.Before(from) && !date.After(to) {
				dates = append(dates, date)
			}
		}
		cursor = cursor.AddDate(0, 1, 0)
	}
	return dates
}

// cycleWindow is the span whose matching transactions settle the
// occurrence due on date: its calendar month for monthly items, or with
// grace days the month up to the end of its grace, so a payment made late
// in the next month settles the cycle it was late for; the month before to
// the month after for quarterly and yearly ones, so an early or late
// payment still counts. Windows of one item never overlap.
func (item RecurringItem) cycleWindow(date time.Time) (time.Time, time.Time) {
	first := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, time.UTC)
	if item.Frequency != FrequencyMonthly {
		return first.AddDate(0, -1, 0), first.AddDate(0, 2, 0)
	}
	if item.GraceDays <= 0 {
		return first, first.AddDate(0, 1, 0)
	}
	previous := first.AddDate(0, -1, 0)
	previous = clampedDate(previous.Year(), previous.Month(), item.Day)
	return previous.AddDate(0, 0, item.GraceDays+1), date.AddDate(0, 0, item.GraceDays+1)
}

// graceEnd is the last day an unpaid occurrence due on date is still due.
func (item RecurringItem) graceEnd(date time.Time) time.Time {
	return date.AddDate(0, 0, item.GraceDays)
}

// matchAmountTolerance is how far an amount may stray from the item's
// amount when the item has no match text and is recognised by its
// category (and account) alone.
const matchAmountTolerance = 0.15

// Matches reports whether the transaction is a payment (or, for income, a
// receipt) of the item: its payee contains the item's match text as whole
// words (case and accents ignored) and, when set, its category and account
// agree. An item without match text needs a category and an amount within
// matchAmountTolerance. Lending rows never match.
func (item RecurringItem) Matches(t Transaction) bool {
	if !t.Usable() || t.IsLending() {
		return false
	}
	amount := item.flowAmount(t)
	if amount <= 0 {
		return false
	}
	if item.Category != "" && FoldText(item.Category) != FoldText(t.Category) {
		return false
	}
	if item.Account != "" && FoldText(item.Account) != FoldText(t.AccountName) {
		return false
	}
	if match := FoldText(item.Match); match != "" {
		return containsWords(FoldText(t.Payee), match)
	}
	if item.Category == "" || item.Amount == nil || *item.Amount <= 0 {
		return false
	}
	return math.Abs(amount-*item.Amount) <= *item.Amount*matchAmountTolerance
}

// flowAmount is the money the transaction moved in the item's direction.
func (item RecurringItem) flowAmount(t Transaction) float64 {
	if item.Kind.IsIncome() {
		return t.DepositAmount()
	}
	return t.PaymentAmount()
}

// containsWords reports whether folded text contains the folded phrase as
// whole words.
func containsWords(text, phrase string) bool {
	return strings.Contains(" "+text+" ", " "+phrase+" ")
}
