package domain

import (
	"slices"
	"strings"
	"time"
)

type AccountType string

const (
	AccountTypeActiveUse      AccountType = "active use"
	AccountTypeRarelyUse      AccountType = "rarely use"
	AccountTypeLongHolding    AccountType = "long holding"
	AccountTypeDeprecated     AccountType = "deprecated"
	AccountTypeNegativeActive AccountType = "negative active use"
	AccountTypeBank           AccountType = "bank"
	AccountTypeCrypto         AccountType = "crypto"
	AccountTypeCash           AccountType = "cash"
	AccountTypeInvestment     AccountType = "investment"
)

type Currency string

const (
	CurrencyVND  Currency = "VND"
	CurrencyUSD  Currency = "USD"
	CurrencyUSDT Currency = "USDT"
)

type CategoryType string

const (
	CategoryTypeIncome    CategoryType = "income"
	CategoryTypeExpense   CategoryType = "expense"
	CategoryTypeNonBudget CategoryType = "non-budget"
)

type Account struct {
	Name    string  `json:"name"`
	DueDate *string `json:"dueDate"`
	// CreditLimit is Accounts column C: a card account's credit limit,
	// which overrides its bank's reference limit (WM-CARD-16). Column D
	// (the retired per-account goal progress) is no longer read (C2).
	CreditLimit *float64 `json:"creditLimit"`
	// Balances are always VND (D25), whatever the currency.
	ClearedBalance float64     `json:"clearedBalance"`
	Balance        float64     `json:"balance"`
	Type           AccountType `json:"type"`
	// Currency is the entry currency: USD/USDT amounts are written as a
	// formula that converts them to VND (WM-TX-9), so the sheet stores VND.
	Currency Currency `json:"currency"`
	Note     *string  `json:"note"`
}

type Transaction struct {
	ID              string        `json:"id"`
	AccountName     string        `json:"accountName"`
	Date            time.Time     `json:"date"`
	ReferenceNumber *string       `json:"referenceNumber"`
	Payee           string        `json:"payee"`
	Tags            []string      `json:"tags"`
	Memo            *string       `json:"memo"`
	Category        string        `json:"category"`
	CategoryType    *CategoryType `json:"categoryType,omitempty"`
	// CategoryNature is joined from Categories column B like CategoryType;
	// lending marks money lent and repaid (D30).
	CategoryNature CategoryNature `json:"categoryNature,omitempty"`
	Cleared        bool           `json:"cleared"`
	Payment        *float64       `json:"payment"`
	Deposit        *float64       `json:"deposit"`
	// ETag fingerprints the sheet row (A-J) so edits and deletes can detect
	// that the row changed or moved since it was read.
	ETag string `json:"etag"`
	// Issues lists data problems found while reading the row, e.g. an
	// unparseable date (WM-DATA-3). Rows with issues are excluded from rules.
	Issues []string `json:"issues,omitempty"`
}

type TransactionCreateInput struct {
	AccountName     string   `json:"accountName"`
	Date            string   `json:"date"`
	ReferenceNumber *string  `json:"referenceNumber,omitempty"`
	Payee           string   `json:"payee"`
	Tags            []string `json:"tags"`
	Memo            *string  `json:"memo,omitempty"`
	Category        string   `json:"category"`
	Cleared         bool     `json:"cleared"`
	Payment         *float64 `json:"payment,omitempty"`
	Deposit         *float64 `json:"deposit,omitempty"`
}

type AccountCreateInput struct {
	Name     string      `json:"name"`
	Type     AccountType `json:"type"`
	Balance  *float64    `json:"balance,omitempty"`
	Currency Currency    `json:"currency,omitempty"`
	Note     *string     `json:"note,omitempty"`
}

type Category struct {
	Name string       `json:"name"`
	Type CategoryType `json:"type"`
	// Nature tags a category fixed, essential, flexible or lending (Categories column B); empty when untagged.
	Nature CategoryNature `json:"nature,omitempty"`
}

// CategoryNature sorts expense by how far it can move: fixed (rent,
// insurance: set by a contract), essential (groceries, transport, utilities:
// needed, but the amount can be lowered) or flexible (eating out, shopping:
// wants). Lending marks money lent to and repaid by people: it is neither
// income nor expense (WM-RULE-3, D30).
type CategoryNature string

const (
	NatureFixed     CategoryNature = "fixed"
	NatureEssential CategoryNature = "essential"
	NatureFlexible  CategoryNature = "flexible"
	NatureLending   CategoryNature = "lending"
)

// CategoryNatures lists every nature a category can be tagged with.
var CategoryNatures = []CategoryNature{NatureFixed, NatureEssential, NatureFlexible, NatureLending}

// ParseCategoryNature reads a Categories column B cell, ignoring case and
// spaces; anything else is untagged.
func ParseCategoryNature(cell string) CategoryNature {
	nature := CategoryNature(strings.ToLower(strings.TrimSpace(cell)))
	if slices.Contains(CategoryNatures, nature) {
		return nature
	}
	return ""
}

type BudgetItem struct {
	Category         string             `json:"category"`
	CategoryType     *CategoryType      `json:"categoryType,omitempty"`
	MonthlyLimit     float64            `json:"monthlyLimit"`
	YearlyLimit      float64            `json:"yearlyLimit"`
	MonthlyLimits    map[string]float64 `json:"monthlyLimits,omitempty"`
	MonthlySpent     float64            `json:"monthlySpent"`
	YearlySpent      float64            `json:"yearlySpent"`
	MonthlyRemaining float64            `json:"monthlyRemaining"`
	YearlyRemaining  float64            `json:"yearlyRemaining"`
	Note             string             `json:"note,omitempty"`
}

// Holding is one typed investment position (WM-DATA-7). Value is in
// Currency; it is nil when neither the sheet nor quantity x price give one,
// in which case a live price is needed.
type Holding struct {
	Platform    string   `json:"platform"`
	Symbol      string   `json:"symbol"`
	Quantity    *float64 `json:"quantity"`
	Price       *float64 `json:"price"`
	Currency    Currency `json:"currency"`
	Value       *float64 `json:"value"`
	ValueSource string   `json:"valueSource"`
	// History is the fund tab's purchase and sale log (WM-INV-21), with
	// the entries that could not be read.
	History       []FundTrade `json:"history,omitempty"`
	HistoryIssues []string    `json:"historyIssues,omitempty"`
}

const (
	HoldingValueFromSheet         = "sheet"
	HoldingValueFromQuantityPrice = "quantity_x_price"
	// HoldingValueFromLivePrice is quantity × a live market price, used only
	// when the sheet has neither a value nor a price (WM-INV-2).
	HoldingValueFromLivePrice = "quantity_x_live_price"
)

type InvestmentHoldingSection struct {
	Headers  []string         `json:"headers"`
	Holdings []map[string]any `json:"holdings"`
	Items    []Holding        `json:"items"`
}

type InvestmentAssets struct {
	Crypto InvestmentHoldingSection `json:"crypto"`
	Funds  InvestmentHoldingSection `json:"funds"`
}

// SheetInfo describes the connected spreadsheet (WM-SET-2). LastSyncAt
// (RFC3339) is the engine's last successful sync, kept in memory only.
type SheetInfo struct {
	SpreadsheetID string  `json:"spreadsheetId"`
	Title         string  `json:"title"`
	LastSyncAt    *string `json:"lastSyncAt"`
}

type SyncResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Deleted int    `json:"deleted,omitempty"`
}

type InitResult struct {
	Ready  bool `json:"ready"`
	Loaded struct {
		Prompts   int `json:"prompts"`
		Knowledge int `json:"knowledge"`
	} `json:"loaded"`
	Error string `json:"error,omitempty"`
}
