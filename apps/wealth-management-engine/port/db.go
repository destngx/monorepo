package port

import "apps/wealth-management-engine/domain"

type DatabaseService interface {
	ReadAccounts() (domain.Accounts, error)
}

type DatabaseClient interface {
	ReadSheet(rangeValue string) ([][]string, error)
	AppendRow(rangeValue string, values []any) error
}
