package service

import (
	"apps/wealth-management-engine/domain"
	"apps/wealth-management-engine/port"
)

const accountsSheetRange = "Accounts!A2:I"

type dbService struct {
	client port.DatabaseClient
}

func NewDatabaseService(client port.DatabaseClient) port.DatabaseService {
	return &dbService{client: client}
}

func (s *dbService) ReadAccounts() (domain.Accounts, error) {
	rows, err := s.client.ReadSheet(accountsSheetRange)
	if err != nil {
		return domain.Accounts{}, err
	}

	return domain.Accounts{
		Range: accountsSheetRange,
		Rows:  rows,
	}, nil
}
