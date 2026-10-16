// Package fakes holds in-memory port implementations shared by adapter tests.
package fakes

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"fmt"
	"sync"
	"time"
)

// Database is a port.DatabaseService that returns canned data and records
// writes. Err, when set, is returned by every call.
type Database struct {
	mu sync.Mutex

	Accounts     []domain.Account
	Transactions []domain.Transaction
	Budget       []domain.BudgetItem
	// BudgetByYear, when set, replaces Budget: a missing year is ErrNotFound.
	BudgetByYear map[int][]domain.BudgetItem
	Info         domain.SheetInfo
	Categories   []domain.Category
	GoalPlan     domain.GoalPlanSheet
	Recurring    domain.RecurringSheet
	Init         domain.InitResult
	AIContent    domain.AIContent
	CaptureRules []domain.CaptureRule
	Cards        domain.CardConfig
	// Preferences, when nil, reads as a workbook without the tab.
	Preferences *domain.Preferences
	Err         error
	// GoalPlanErr and RecurringErr fail only that read (optional context).
	GoalPlanErr  error
	RecurringErr error
	// StaleAt, when set, is reported as the time of stale data served.
	StaleAt time.Time

	CreatedAccounts     []domain.AccountCreateInput
	UpdatedAccounts     map[string]domain.AccountUpdateInput
	DeletedAccounts     []string
	CreatedTransactions []domain.TransactionCreateInput
	Batches             []domain.TransactionBatchInput
	UpdatedTransactions map[string]domain.TransactionUpdateInput
	DeletedTransactions map[string]string
	BudgetYears         []int
	BudgetUpdates       []domain.BudgetLimitUpdate
	Reallocations       []domain.BudgetReallocation
	Syncs               int
	SavedCaptureRules   [][]domain.CaptureRule
	SavedPreferences    []domain.PreferencesUpdate
	NatureUpdates       []domain.CategoryNatureUpdate
	GoalSetups          int
	CreatedGoals        []domain.GoalInput
	UpdatedGoals        map[string]domain.GoalUpdateInput
	DeletedGoals        map[string]string
	GoalOrders          []domain.GoalReorderInput
	CreatedRecurring    []domain.RecurringInput
	ConfirmedRecurring  []domain.RecurringConfirmInput
	UpdatedRecurring    map[string]domain.RecurringUpdateInput
	RecurringStatuses   map[string]domain.RecurringStatusInput
	DeletedRecurring    map[string]string
}

var _ port.DatabaseService = (*Database)(nil)

func (f *Database) ListAccounts(bool) ([]domain.Account, error) {
	return f.Accounts, f.Err
}

func (f *Database) CreateAccount(input domain.AccountCreateInput) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.CreatedAccounts = append(f.CreatedAccounts, input)
	return f.Err
}

func (f *Database) UpdateAccount(name string, input domain.AccountUpdateInput) (domain.Account, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.UpdatedAccounts == nil {
		f.UpdatedAccounts = map[string]domain.AccountUpdateInput{}
	}
	f.UpdatedAccounts[name] = input
	account := domain.Account{Name: name}
	if input.Type != nil {
		account.Type = *input.Type
	}
	return account, f.Err
}

func (f *Database) DeleteAccount(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.DeletedAccounts = append(f.DeletedAccounts, name)
	return f.Err
}

func (f *Database) ListTransactions(bool) ([]domain.Transaction, error) {
	return f.Transactions, f.Err
}

func (f *Database) CreateTransaction(input domain.TransactionCreateInput) (domain.TransactionWriteResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.CreatedTransactions = append(f.CreatedTransactions, input)
	return domain.TransactionWriteResult{ID: "row-100"}, f.Err
}

func (f *Database) CreateTransactions(input domain.TransactionBatchInput) (domain.TransactionBatchResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Batches = append(f.Batches, input)
	return domain.TransactionBatchResult{BatchID: input.BatchID, Created: len(input.Transactions)}, f.Err
}

func (f *Database) UpdateTransaction(id string, input domain.TransactionUpdateInput) (domain.TransactionWriteResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.UpdatedTransactions == nil {
		f.UpdatedTransactions = map[string]domain.TransactionUpdateInput{}
	}
	f.UpdatedTransactions[id] = input
	return domain.TransactionWriteResult{ID: id}, f.Err
}

func (f *Database) DeleteTransaction(id string, etag string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.DeletedTransactions == nil {
		f.DeletedTransactions = map[string]string{}
	}
	f.DeletedTransactions[id] = etag
	return f.Err
}

func (f *Database) ListCategories(bool) ([]domain.Category, error) {
	if f.Categories != nil {
		return f.Categories, f.Err
	}
	return []domain.Category{{Name: "Salary", Type: domain.CategoryTypeIncome}}, f.Err
}

func (f *Database) ListBudget(year int, _ bool) ([]domain.BudgetItem, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.BudgetYears = append(f.BudgetYears, year)
	if f.Err != nil || f.BudgetByYear == nil {
		return f.Budget, f.Err
	}
	items, ok := f.BudgetByYear[year]
	if !ok {
		return nil, fmt.Errorf("%w: budget tab for %d", domain.ErrNotFound, year)
	}
	return items, nil
}

func (f *Database) ReallocateBudget(input domain.BudgetReallocation) (domain.BudgetReallocationResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := input.Validate(); err != nil {
		return domain.BudgetReallocationResult{}, err
	}
	f.Reallocations = append(f.Reallocations, input)
	return domain.BudgetReallocationResult{
		From: domain.BudgetCategoryLimit{Category: input.From},
		To:   domain.BudgetCategoryLimit{Category: input.To, Limit: input.Amount},
	}, f.Err
}

func (f *Database) UpdateBudgetLimit(input domain.BudgetLimitUpdate) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.BudgetUpdates = append(f.BudgetUpdates, input)
	return f.Err
}

func (f *Database) ReadGoalPlan(bool) (domain.GoalPlanSheet, error) {
	if f.GoalPlanErr != nil {
		return domain.GoalPlanSheet{}, f.GoalPlanErr
	}
	return f.GoalPlan, f.Err
}

func (f *Database) SetupGoalPlan() (domain.GoalPlanSetupResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.GoalSetups++
	return domain.GoalPlanSetupResult{Created: 1, Skipped: []string{}}, f.Err
}

// CreateGoal validates like the service and records the input.
func (f *Database) CreateGoal(input domain.GoalInput) (domain.PlannedGoal, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := input.Validate(); err != nil {
		return domain.PlannedGoal{}, err
	}
	f.CreatedGoals = append(f.CreatedGoals, input)
	return domain.PlannedGoal{ID: fmt.Sprintf("goal-%d", len(f.CreatedGoals)+1), Name: input.Name, Target: input.Target, Status: domain.GoalActive}, f.Err
}

func (f *Database) UpdateGoal(id string, input domain.GoalUpdateInput) (domain.PlannedGoal, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := input.Validate(); err != nil {
		return domain.PlannedGoal{}, err
	}
	if f.UpdatedGoals == nil {
		f.UpdatedGoals = map[string]domain.GoalUpdateInput{}
	}
	f.UpdatedGoals[id] = input
	return domain.PlannedGoal{ID: id, Name: input.Name, Target: input.Target, Status: input.Status}, f.Err
}

func (f *Database) DeleteGoal(id string, etag string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.DeletedGoals == nil {
		f.DeletedGoals = map[string]string{}
	}
	f.DeletedGoals[id] = etag
	return f.Err
}

func (f *Database) ReorderGoals(input domain.GoalReorderInput) ([]domain.PlannedGoal, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := input.Validate(); err != nil {
		return nil, err
	}
	f.GoalOrders = append(f.GoalOrders, input)
	return nonNil(f.GoalPlan.Goals), f.Err
}

// UpdateCategoryNature validates like the service and records the update.
func (f *Database) UpdateCategoryNature(input domain.CategoryNatureUpdate) (domain.Category, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := input.Validate(); err != nil {
		return domain.Category{}, err
	}
	f.NatureUpdates = append(f.NatureUpdates, input)
	nature, _ := input.ParsedNature()
	return domain.Category{Name: input.Category, Type: domain.CategoryTypeExpense, Nature: nature}, f.Err
}

func nonNil[T any](values []T) []T {
	if values == nil {
		return []T{}
	}
	return values
}

func (f *Database) ListTags(bool) ([]string, error) { return []string{"salary"}, f.Err }

func (f *Database) GetInvestmentAssets(bool) (domain.InvestmentAssets, error) {
	return domain.InvestmentAssets{}, f.Err
}

func (f *Database) WarmAIContent(bool) (domain.InitResult, error) { return f.Init, f.Err }

func (f *Database) LoadAIContent(bool) (domain.AIContent, error) { return f.AIContent, f.Err }

func (f *Database) ListCaptureRules(bool) ([]domain.CaptureRule, error) {
	return nonNil(f.CaptureRules), f.Err
}

func (f *Database) ListPreferences(bool) (domain.Preferences, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.preferences(), f.Err
}

func (f *Database) preferences() domain.Preferences {
	if f.Preferences != nil {
		return *f.Preferences
	}
	prefs := domain.ParsePreferences(nil, 2)
	prefs.TabExists = false
	return prefs
}

// SavePreferences applies the update with the domain rules and keeps it.
func (f *Database) SavePreferences(update domain.PreferencesUpdate) (domain.Preferences, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.Err != nil {
		return domain.Preferences{}, f.Err
	}
	next, err := domain.ApplyPreferencesUpdate(f.preferences(), update)
	if err != nil {
		return domain.Preferences{}, err
	}
	next.TabExists = true
	f.Preferences = &next
	f.SavedPreferences = append(f.SavedPreferences, update)
	return next, nil
}

func (f *Database) SaveCaptureRules(rules []domain.CaptureRule) (domain.CaptureRulesSaveResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.Err != nil {
		return domain.CaptureRulesSaveResult{}, f.Err
	}
	f.SavedCaptureRules = append(f.SavedCaptureRules, rules)
	return domain.CaptureRulesSaveResult{Created: len(rules)}, nil
}

func (f *Database) ListCardConfig(bool) (domain.CardConfig, error) {
	return f.Cards, f.Err
}

func (f *Database) SyncCache() (domain.SyncResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Syncs++
	return domain.SyncResult{Success: f.Err == nil}, f.Err
}

func (f *Database) SheetInfo() (domain.SheetInfo, error) { return f.Info, f.Err }

func (f *Database) Ping() error { return f.Err }

func (f *Database) ReadLedger(bool) (domain.LedgerSnapshot, error) {
	return domain.LedgerSnapshot{Accounts: nonNil(f.Accounts), Transactions: nonNil(f.Transactions)}, f.Err
}

func (f *Database) StaleCursor() uint64 { return 0 }

func (f *Database) StaleSince(uint64) (time.Time, bool) {
	return f.StaleAt, !f.StaleAt.IsZero()
}

func (f *Database) ReadRecurring(bool) (domain.RecurringSheet, error) {
	if f.RecurringErr != nil {
		return domain.RecurringSheet{}, f.RecurringErr
	}
	return f.Recurring, f.Err
}

// CreateRecurring validates like the service and records the input.
func (f *Database) CreateRecurring(input domain.RecurringInput) (domain.RecurringItem, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := input.Validate(); err != nil {
		return domain.RecurringItem{}, err
	}
	f.CreatedRecurring = append(f.CreatedRecurring, input)
	item := input.Normalized()
	return domain.RecurringItem{ID: fmt.Sprintf("recurring-%d", len(f.CreatedRecurring)+1), Name: item.Name, Kind: item.Kind, Amount: item.Amount, Status: item.Status}, f.Err
}

func (f *Database) ConfirmRecurringSuggestion(input domain.RecurringConfirmInput) (domain.RecurringItem, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := input.Validate(); err != nil {
		return domain.RecurringItem{}, err
	}
	f.ConfirmedRecurring = append(f.ConfirmedRecurring, input)
	return domain.RecurringItem{ID: "recurring-2", Name: input.Name, Status: domain.RecurringActive}, f.Err
}

func (f *Database) UpdateRecurring(id string, input domain.RecurringUpdateInput) (domain.RecurringItem, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := input.Validate(); err != nil {
		return domain.RecurringItem{}, err
	}
	if f.UpdatedRecurring == nil {
		f.UpdatedRecurring = map[string]domain.RecurringUpdateInput{}
	}
	f.UpdatedRecurring[id] = input
	return domain.RecurringItem{ID: id, Name: input.Name, Kind: input.Kind, Status: input.Status}, f.Err
}

func (f *Database) SetRecurringStatus(id string, input domain.RecurringStatusInput) (domain.RecurringItem, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := input.Validate(); err != nil {
		return domain.RecurringItem{}, err
	}
	if f.RecurringStatuses == nil {
		f.RecurringStatuses = map[string]domain.RecurringStatusInput{}
	}
	f.RecurringStatuses[id] = input
	return domain.RecurringItem{ID: id, Status: input.Status}, f.Err
}

func (f *Database) DeleteRecurring(id string, etag string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.DeletedRecurring == nil {
		f.DeletedRecurring = map[string]string{}
	}
	f.DeletedRecurring[id] = etag
	return f.Err
}
