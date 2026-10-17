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
	// ReadGoalPlan reads the app-managed Goal Plan tab (D31); Exists is
	// false until SetupGoalPlan creates it.
	ReadGoalPlan(forceFresh bool) (domain.GoalPlanSheet, error)
	// SetupGoalPlan creates the Goal Plan tab with an emergency fund row and
	// the rows imported from the old Goals tab, which is only read.
	// ErrConflict when the tab already exists.
	SetupGoalPlan() (domain.GoalPlanSetupResult, error)
	CreateGoal(input domain.GoalInput) (domain.PlannedGoal, error)
	// UpdateGoal replaces a goal row when it still matches the ETag
	// (ErrConflict otherwise).
	UpdateGoal(id string, input domain.GoalUpdateInput) (domain.PlannedGoal, error)
	// DeleteGoal clears a goal row when it still matches the ETag.
	DeleteGoal(id string, etag string) error
	// ReorderGoals sets priorities 1..n in the order of the IDs; unlisted
	// goals follow in their order. ErrConflict for an ID not in the tab.
	ReorderGoals(input domain.GoalReorderInput) ([]domain.PlannedGoal, error)
	// ReadRecurring reads the app-managed Recurring tab (WM-BILL-1); Exists
	// is false until the first item is saved.
	ReadRecurring(forceFresh bool) (domain.RecurringSheet, error)
	// CreateRecurring appends an item, creating the tab on first use.
	CreateRecurring(input domain.RecurringInput) (domain.RecurringItem, error)
	// ConfirmRecurringSuggestion writes the current suggestion with the
	// key, with the overrides applied (WM-BILL-2), creating the tab on first
	// use. ErrNotFound when the key is no longer suggested.
	ConfirmRecurringSuggestion(input domain.RecurringConfirmInput) (domain.RecurringItem, error)
	// UpdateRecurring replaces an item row when it still matches the ETag
	// (ErrConflict otherwise).
	UpdateRecurring(id string, input domain.RecurringUpdateInput) (domain.RecurringItem, error)
	// SetRecurringStatus pauses or resumes an item (column J only).
	SetRecurringStatus(id string, input domain.RecurringStatusInput) (domain.RecurringItem, error)
	// DeleteRecurring clears an item row when it still matches the ETag.
	DeleteRecurring(id string, etag string) error
	// ReadRecommendations reads the app-managed Recommendations tab
	// (WM-REC-1); Exists is false until the first item is written.
	ReadRecommendations(forceFresh bool) (domain.RecommendationSheet, error)
	// UpdateRecommendations runs the plan on a fresh read of the tab under
	// the write lock and writes what it returns: new rows below the last
	// one (creating the tab on first use) and the status columns H-K of
	// existing rows. No other tab is written.
	UpdateRecommendations(plan domain.RecommendationPlan) (domain.RecommendationWrite, error)
	// ReadSnapshots reads the app-managed Snapshots tab (WM-NW-1); Exists
	// is false until the first snapshot is written.
	ReadSnapshots(forceFresh bool) (domain.SnapshotSheet, error)
	// SaveSnapshot writes a month's row, updating the month's existing row
	// rather than adding a second one, creating the tab on first use.
	SaveSnapshot(snapshot domain.MonthSnapshot) (domain.SnapshotSaveResult, error)
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
	// ListPreferences reads the app-managed Preferences tab (WM-SET-6):
	// typed values with their defaults; a missing tab means every default.
	ListPreferences(forceFresh bool) (domain.Preferences, error)
	// SavePreferences writes the given keys (WM-SET-7), creating the tab on
	// the first save; it writes no other tab. ErrValidation for an unknown
	// key, a value out of range or allocation targets not adding up to 100.
	SavePreferences(update domain.PreferencesUpdate) (domain.Preferences, error)
	// SetAccountReadiness marks an account ready to use or not for
	// safe-to-spend (D33) in the safeToSpend.notReadyAccounts row.
	// ErrValidation without an account or ETag, ErrNotFound when marking an
	// unknown account not ready, ErrConflict when the setting changed.
	SetAccountReadiness(input domain.AccountReadinessInput) (domain.Preferences, error)
	// SetEmergencyAccount marks an account as an emergency fund account or
	// not for the idle cash check (D34) in the idleCash.emergencyAccounts
	// row. ErrValidation without an account or ETag, ErrNotFound when
	// marking an unknown account, ErrConflict when the setting changed.
	SetEmergencyAccount(input domain.EmergencyAccountInput) (domain.Preferences, error)
	// SetAccountMinimumBalance sets an account's minimum balance to keep
	// (D35) in the accounts.minimumBalance row; a nil or 0 minimum removes
	// it. ErrValidation without an account or ETag or for a bad minimum,
	// ErrNotFound when setting one on an unknown account, ErrConflict when
	// the setting changed.
	SetAccountMinimumBalance(input domain.AccountMinimumInput) (domain.Preferences, error)
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
