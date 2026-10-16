package domain

import (
	"fmt"
	"strings"
)

// Write inputs and their validation, shared by HTTP and MCP so both reject
// the same requests (WM-MCP-3).

// Validate checks the required fields, the date and the amounts of a new transaction.
func (input TransactionCreateInput) Validate() error {
	var problems []string
	if strings.TrimSpace(input.AccountName) == "" {
		problems = append(problems, "accountName is required")
	}
	if strings.TrimSpace(input.Payee) == "" {
		problems = append(problems, "payee is required")
	}
	if strings.TrimSpace(input.Category) == "" {
		problems = append(problems, "category is required")
	}
	if strings.TrimSpace(input.Date) == "" {
		problems = append(problems, "date is required")
	} else if _, err := ParseSheetDate(input.Date); err != nil {
		problems = append(problems, fmt.Sprintf("date %q is not a valid dd/mm/yyyy or yyyy-mm-dd date", input.Date))
	}
	if input.Payment == nil && input.Deposit == nil {
		problems = append(problems, "payment or deposit is required")
	}
	if input.Payment != nil && *input.Payment < 0 {
		problems = append(problems, "payment must not be negative")
	}
	if input.Deposit != nil && *input.Deposit < 0 {
		problems = append(problems, "deposit must not be negative")
	}
	return validationError(problems)
}

// TransactionUpdateInput replaces columns A-J of a transaction row. ETag must
// match the row as last read, otherwise the update is rejected (WM-TX-8).
type TransactionUpdateInput struct {
	TransactionCreateInput
	ETag string `json:"etag"`
}

func (input TransactionUpdateInput) Validate() error {
	if strings.TrimSpace(input.ETag) == "" {
		return fmt.Errorf("%w: etag is required", ErrValidation)
	}
	return input.TransactionCreateInput.Validate()
}

// TransactionBatchInput appends several transactions at once. Retrying the
// same BatchID returns the first result instead of writing again (WM-TX-10).
type TransactionBatchInput struct {
	BatchID      string                   `json:"batchId"`
	Transactions []TransactionCreateInput `json:"transactions"`
}

const MaxTransactionBatch = 100

func (input TransactionBatchInput) Validate() error {
	var problems []string
	if strings.TrimSpace(input.BatchID) == "" {
		problems = append(problems, "batchId is required")
	}
	if len(input.Transactions) == 0 {
		problems = append(problems, "transactions must not be empty")
	}
	if len(input.Transactions) > MaxTransactionBatch {
		problems = append(problems, fmt.Sprintf("at most %d transactions per batch", MaxTransactionBatch))
	}
	return validationError(problems)
}

// TransactionBatchRow reports the outcome of one batch entry.
type TransactionBatchRow struct {
	Index int    `json:"index"`
	ID    string `json:"id,omitempty"`
	Error string `json:"error,omitempty"`
}

type TransactionBatchResult struct {
	BatchID  string                `json:"batchId"`
	Created  int                   `json:"created"`
	Failed   int                   `json:"failed"`
	Replayed bool                  `json:"replayed"`
	Rows     []TransactionBatchRow `json:"rows"`
}

// TransactionWriteResult identifies a written transaction row.
type TransactionWriteResult struct {
	ID string `json:"id"`
}

func (input AccountCreateInput) Validate() error {
	var problems []string
	if strings.TrimSpace(input.Name) == "" {
		problems = append(problems, "name is required")
	}
	if strings.HasPrefix(strings.TrimSpace(input.Name), DeletedAccountPrefix) {
		problems = append(problems, "name must not start with "+DeletedAccountPrefix)
	}
	if input.Type == "" {
		problems = append(problems, "type is required")
	} else if !input.Type.Valid() {
		problems = append(problems, fmt.Sprintf("type %q is not a known account type", input.Type))
	}
	if input.Currency != "" && !input.Currency.Valid() {
		problems = append(problems, fmt.Sprintf("currency %q is not supported", input.Currency))
	}
	return validationError(problems)
}

// AccountUpdateInput edits the account type and note. Balances are derived
// by the sheet from transactions, so they are not editable here.
type AccountUpdateInput struct {
	Type *AccountType `json:"type,omitempty"`
	Note *string      `json:"note,omitempty"`
}

func (input AccountUpdateInput) Validate() error {
	if input.Type == nil && input.Note == nil {
		return fmt.Errorf("%w: type or note is required", ErrValidation)
	}
	if input.Type != nil && !input.Type.Valid() {
		return fmt.Errorf("%w: type %q is not a known account type", ErrValidation, *input.Type)
	}
	return nil
}

// BudgetLimitUpdate sets a category's limit for one month ("YYYY-MM") or,
// with an empty Month, its yearly limit, in that year's budget tab (WM-BUD-6).
type BudgetLimitUpdate struct {
	Year     int     `json:"year"`
	Category string  `json:"category"`
	Month    string  `json:"month,omitempty"`
	Limit    float64 `json:"limit"`
}

func (input BudgetLimitUpdate) Validate() error {
	var problems []string
	if strings.TrimSpace(input.Category) == "" {
		problems = append(problems, "category is required")
	}
	if input.Limit < 0 {
		problems = append(problems, "limit must not be negative")
	}
	if input.Month != "" {
		month, err := ParseMonthKey(input.Month)
		if err != nil {
			problems = append(problems, "month must be YYYY-MM")
		} else if input.Year != 0 && month.Year() != input.Year {
			problems = append(problems, "month must be in the given year")
		}
	}
	return validationError(problems)
}

// BudgetReallocation moves Amount of one month's limit from one category to
// another, keeping the month's total unchanged (WM-BUD-7).
type BudgetReallocation struct {
	Year   int     `json:"year"`
	Month  string  `json:"month"`
	From   string  `json:"from"`
	To     string  `json:"to"`
	Amount float64 `json:"amount"`
}

// Validate checks the shape of the move; whether From has enough limit is
// checked against the sheet.
func (input BudgetReallocation) Validate() error {
	var problems []string
	if strings.TrimSpace(input.From) == "" || strings.TrimSpace(input.To) == "" {
		problems = append(problems, "from and to are required")
	} else if normalizeCategory(input.From) == normalizeCategory(input.To) {
		problems = append(problems, "from and to must be different categories")
	}
	if input.Amount <= 0 {
		problems = append(problems, "amount must be positive")
	}
	month, err := ParseMonthKey(input.Month)
	switch {
	case err != nil:
		problems = append(problems, "month must be YYYY-MM")
	case input.Year != 0 && month.Year() != input.Year:
		problems = append(problems, "month must be in the given year")
	}
	return validationError(problems)
}

// BudgetCategoryLimit is a category's monthly limit after a reallocation.
type BudgetCategoryLimit struct {
	Category string  `json:"category"`
	Limit    float64 `json:"limit"`
}

type BudgetReallocationResult struct {
	From BudgetCategoryLimit `json:"from"`
	To   BudgetCategoryLimit `json:"to"`
}

const DeletedAccountPrefix = "[DELETED]"

var accountTypes = map[AccountType]bool{
	AccountTypeActiveUse: true, AccountTypeRarelyUse: true, AccountTypeLongHolding: true, AccountTypeDeprecated: true,
	AccountTypeNegativeActive: true, AccountTypeBank: true, AccountTypeCrypto: true, AccountTypeCash: true, AccountTypeInvestment: true,
}

func (t AccountType) Valid() bool { return accountTypes[t] }

func (c Currency) Valid() bool {
	return c == CurrencyVND || c == CurrencyUSD || c == CurrencyUSDT
}

func validationError(problems []string) error {
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrValidation, strings.Join(problems, "; "))
}

// CategoryNatureNone clears a category's nature.
const CategoryNatureNone = "none"

// CategoryNatureUpdate sets a category's nature in Categories column B
// (WM-LEND-5): fixed, essential, flexible, lending, or none (or empty) to
// clear it. Category is matched by its exact name.
type CategoryNatureUpdate struct {
	Category string `json:"category"`
	Nature   string `json:"nature"`
}

func (input CategoryNatureUpdate) Validate() error {
	var problems []string
	if strings.TrimSpace(input.Category) == "" {
		problems = append(problems, "category is required")
	}
	if _, ok := input.ParsedNature(); !ok {
		problems = append(problems, fmt.Sprintf("nature %q must be one of fixed, essential, flexible, lending or none", input.Nature))
	}
	return validationError(problems)
}

// ParsedNature returns the nature to write; "" clears it. ok is false for
// an unknown value.
func (input CategoryNatureUpdate) ParsedNature() (CategoryNature, bool) {
	cell := strings.ToLower(strings.TrimSpace(input.Nature))
	if cell == "" || cell == CategoryNatureNone {
		return "", true
	}
	nature := ParseCategoryNature(cell)
	return nature, nature != ""
}
