package port

import (
	"apps/wealth-management/domain"
	"time"
)

// DatabaseService reads and writes the Google Sheet system of record. Errors
// wrap domain.ErrValidation, ErrNotFound or ErrConflict where applicable.
type DatabaseService interface {
	ListAccounts(forceFresh bool) ([]domain.Account, error)
	// ReadLedger returns accounts and transactions, reading every
	// dataset the cache misses in one batched Sheets request (WM-NFR-3).
	ReadLedger(forceFresh bool) (domain.LedgerSnapshot, error)
	CreateAccount(input domain.AccountCreateInput) error
	UpdateAccount(name string, input domain.AccountUpdateInput) (domain.Account, error)
	DeleteAccount(name string) error

	ListTransactions(forceFresh bool) ([]domain.Transaction, error)
	CreateTransaction(input domain.TransactionCreateInput) (domain.TransactionWriteResult, error)
	CreateTransactions(input domain.TransactionBatchInput) (domain.TransactionBatchResult, error)
	UpdateTransaction(id string, input domain.TransactionUpdateInput) (domain.TransactionWriteResult, error)
	DeleteTransaction(id string, etag string) error

	ListCategories(forceFresh bool) ([]domain.Category, error)
	// UpdateCategoryNature writes a category's nature to Categories column B
	// (WM-LEND-5); ErrNotFound when the name is missing, ErrConflict when it
	// appears more than once.
	UpdateCategoryNature(input domain.CategoryNatureUpdate) (domain.Category, error)
	// ListBudget reads the budget tab of a year; 0 means the current year.
	ListBudget(year int, forceFresh bool) ([]domain.BudgetItem, error)
	UpdateBudgetLimit(input domain.BudgetLimitUpdate) error
	// ReallocateBudget moves part of a month's limit between categories,
	// re-reading the sheet first (WM-BUD-7).
	ReallocateBudget(input domain.BudgetReallocation) (domain.BudgetReallocationResult, error)
	ListGoals(forceFresh bool) ([]domain.Goal, error)
	ListTags(forceFresh bool) ([]string, error)
	GetInvestmentAssets(forceFresh bool) (domain.InvestmentAssets, error)
	WarmAIContent(forceFresh bool) (domain.InitResult, error)
	// LoadAIContent returns the Prompts and Knowledge tabs (WM-AI-1).
	LoadAIContent(forceFresh bool) (domain.AIContent, error)
	// ListCaptureRules returns the Smart Capture aliases and payee rules
	// (WM-CAP-11); empty when the CaptureRules tab does not exist yet.
	ListCaptureRules(forceFresh bool) ([]domain.CaptureRule, error)
	// SaveCaptureRules upserts rules, creating the tab on first use.
	SaveCaptureRules(rules []domain.CaptureRule) (domain.CaptureRulesSaveResult, error)
	// ListCardConfig returns the card programmes, banks, cards held and
	// merchant MCCs from the Programs, ProgramRules, Banks, Cards and
	// Merchants tabs (WM-CARD-3); a missing tab contributes no rows.
	ListCardConfig(forceFresh bool) (domain.CardConfig, error)
	SyncCache() (domain.SyncResult, error)
	// SheetInfo returns the spreadsheet id and title and the time of the
	// last successful SyncCache (WM-SET-2).
	SheetInfo() (domain.SheetInfo, error)
	// Ping checks the spreadsheet is reachable.
	Ping() error
	StaleReporter
}

// StaleReporter reports reads served from a last good copy because the
// sheet failed (WM-NFR-4). Sheet reads carry no context, so a caller takes
// a cursor before its work and asks afterwards what was served stale since;
// a concurrent request in the same outage may also be counted.
type StaleReporter interface {
	StaleCursor() uint64
	// StaleSince returns the oldest data time served stale after cursor.
	StaleSince(cursor uint64) (time.Time, bool)
}

type DatabaseClient interface {
	ReadSheet(rangeValue string) ([][]string, error)
	// ReadSheets reads several ranges in one request; rows are returned in
	// the order of ranges.
	ReadSheets(ranges []string) ([][][]string, error)
	UpdateRow(rangeValue string, values []any) error
	// WriteRowsAtFirstEmptyRow writes rows below the anchor column's last value
	// and returns the first sheet row written.
	WriteRowsAtFirstEmptyRow(sheetName string, anchorRange string, rows [][]any) (int, error)
	SheetTitles() ([]string, error)
	// SpreadsheetTitle is the document title, not a tab name.
	SpreadsheetTitle() (string, error)
	// CreateSheet adds a tab with a header row.
	CreateSheet(title string, header []any) error
}
