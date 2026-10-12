package port

import "apps/wealth-management/domain"

// DatabaseService reads and writes the Google Sheet system of record. Errors
// wrap domain.ErrValidation, ErrNotFound or ErrConflict where applicable.
type DatabaseService interface {
	ListAccounts(forceFresh bool) ([]domain.Account, error)
	CreateAccount(input domain.AccountCreateInput) error
	UpdateAccount(name string, input domain.AccountUpdateInput) (domain.Account, error)
	DeleteAccount(name string) error

	ListTransactions(forceFresh bool) ([]domain.Transaction, error)
	CreateTransaction(input domain.TransactionCreateInput) (domain.TransactionWriteResult, error)
	CreateTransactions(input domain.TransactionBatchInput) (domain.TransactionBatchResult, error)
	UpdateTransaction(id string, input domain.TransactionUpdateInput) (domain.TransactionWriteResult, error)
	DeleteTransaction(id string, etag string) error

	ListCategories(forceFresh bool) ([]domain.Category, error)
	// ListBudget reads the budget tab of a year; 0 means the current year.
	ListBudget(year int, forceFresh bool) ([]domain.BudgetItem, error)
	UpdateBudgetLimit(input domain.BudgetLimitUpdate) error
	// ReallocateBudget moves part of a month's limit between categories,
	// re-reading the sheet first (WM-BUD-7).
	ReallocateBudget(input domain.BudgetReallocation) (domain.BudgetReallocationResult, error)
	ListGoals(forceFresh bool) ([]domain.Goal, error)
	ListLoans(forceFresh bool) ([]domain.Loan, error)
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
	SyncCache() (domain.SyncResult, error)
	// SheetInfo returns the spreadsheet id and title and the time of the
	// last successful SyncCache (WM-SET-2).
	SheetInfo() (domain.SheetInfo, error)
	// Ping checks the spreadsheet is reachable.
	Ping() error
}

type DatabaseClient interface {
	ReadSheet(rangeValue string) ([][]string, error)
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
