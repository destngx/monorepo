package port

import "apps/wealth-management-engine/domain"

type DatabaseService interface {
	ReadAccounts() (domain.Accounts, error)
	ListAccounts() ([]domain.Account, error)
	CreateAccount(input domain.AccountCreateInput) error
	DeleteAccount(name string) error

	ListTransactions(forceFresh bool) ([]domain.Transaction, error)
	CreateTransaction(input domain.TransactionCreateInput) error
	ListCategories(forceFresh bool) ([]domain.Category, error)
	ListBudget(forceFresh bool) ([]domain.BudgetItem, error)
	ListGoals(forceFresh bool) ([]domain.Goal, error)
	ListLoans(forceFresh bool) ([]domain.Loan, error)
	ListNotifications(forceFresh bool) ([]domain.EmailNotification, error)
	ListPendingNotifications(forceFresh bool) ([]domain.EmailNotification, error)
	MarkNotificationDone(rowNumber int) error
	ListTags(forceFresh bool) ([]string, error)
	GetInvestmentAssets(forceFresh bool) (domain.InvestmentAssets, error)
	WarmAIContent(forceFresh bool) (domain.InitResult, error)
	SyncCache() (domain.SyncResult, error)
}

type DatabaseClient interface {
	ReadSheet(rangeValue string) ([][]string, error)
	AppendRow(rangeValue string, values []any) error
	UpdateRow(rangeValue string, values []any) error
	WriteToFirstEmptyRow(sheetName string, columnRange string, values []any) error
}
