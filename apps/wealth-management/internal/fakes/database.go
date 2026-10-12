// Package fakes holds in-memory port implementations shared by adapter tests.
package fakes

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"sync"
)

// Database is a port.DatabaseService that returns canned data and records
// writes. Err, when set, is returned by every call.
type Database struct {
	mu sync.Mutex

	Accounts     []domain.Account
	Transactions []domain.Transaction
	Budget       []domain.BudgetItem
	Init         domain.InitResult
	Err          error

	CreatedAccounts     []domain.AccountCreateInput
	UpdatedAccounts     map[string]domain.AccountUpdateInput
	DeletedAccounts     []string
	CreatedTransactions []domain.TransactionCreateInput
	Batches             []domain.TransactionBatchInput
	UpdatedTransactions map[string]domain.TransactionUpdateInput
	DeletedTransactions map[string]string
	BudgetYears         []int
	BudgetUpdates       []domain.BudgetLimitUpdate
	Syncs               int
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
	return []domain.Category{{Name: "Salary", Type: domain.CategoryTypeIncome}}, f.Err
}

func (f *Database) ListBudget(year int, _ bool) ([]domain.BudgetItem, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.BudgetYears = append(f.BudgetYears, year)
	return f.Budget, f.Err
}

func (f *Database) UpdateBudgetLimit(input domain.BudgetLimitUpdate) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.BudgetUpdates = append(f.BudgetUpdates, input)
	return f.Err
}

func (f *Database) ListGoals(bool) ([]domain.Goal, error) { return []domain.Goal{}, f.Err }

func (f *Database) ListLoans(bool) ([]domain.Loan, error) { return []domain.Loan{}, f.Err }

func (f *Database) ListTags(bool) ([]string, error) { return []string{"salary"}, f.Err }

func (f *Database) GetInvestmentAssets(bool) (domain.InvestmentAssets, error) {
	return domain.InvestmentAssets{}, f.Err
}

func (f *Database) WarmAIContent(bool) (domain.InitResult, error) { return f.Init, f.Err }

func (f *Database) SyncCache() (domain.SyncResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Syncs++
	return domain.SyncResult{Success: f.Err == nil}, f.Err
}

func (f *Database) Ping() error { return f.Err }
