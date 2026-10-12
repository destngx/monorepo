package service

import (
	"apps/wealth-management/domain"
	"fmt"
	"strings"
)

func (s *dbService) ListAccounts(forceFresh bool) ([]domain.Account, error) {
	return cachedRead(s, cacheKeyAccounts, sheetDataTTL, forceFresh, func() ([]domain.Account, error) {
		rows, err := s.client.ReadSheet(accountsSheetRange)
		if err != nil {
			return nil, fmt.Errorf("read accounts: %w", err)
		}
		accounts := make([]domain.Account, 0, len(rows))
		for _, row := range rows {
			if account, ok := mapAccount(row); ok {
				accounts = append(accounts, account)
			}
		}
		return accounts, nil
	})
}

func (s *dbService) CreateAccount(input domain.AccountCreateInput) error {
	if err := input.Validate(); err != nil {
		return err
	}
	name := strings.TrimSpace(input.Name)

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	rows, err := s.client.ReadSheet(accountsSheetRange)
	if err != nil {
		return fmt.Errorf("read accounts: %w", err)
	}
	if _, found := findAccountRow(rows, name); found {
		return fmt.Errorf("%w: account %q already exists", domain.ErrConflict, name)
	}
	balance := 0.0
	if input.Balance != nil {
		balance = *input.Balance
	}
	currency := input.Currency
	if currency == "" {
		currency = resolveCurrency("", name)
	}
	values := []any{name, "", "", "", 0.0, balance, string(input.Type), nilToString(input.Note), string(currency)}
	if _, err := s.client.WriteRowsAtFirstEmptyRow("Accounts", accountsAnchorRange, [][]any{values}); err != nil {
		return fmt.Errorf("write account: %w", err)
	}
	s.invalidate("accounts:*")
	return nil
}

// UpdateAccount edits the type (column G) and note (column H) of an account.
func (s *dbService) UpdateAccount(name string, input domain.AccountUpdateInput) (domain.Account, error) {
	if err := input.Validate(); err != nil {
		return domain.Account{}, err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	rows, err := s.client.ReadSheet(accountsSheetRange)
	if err != nil {
		return domain.Account{}, fmt.Errorf("read accounts: %w", err)
	}
	index, found := findAccountRow(rows, name)
	if !found {
		return domain.Account{}, fmt.Errorf("%w: account %q", domain.ErrNotFound, name)
	}
	row := append([]string(nil), rows[index]...)
	for len(row) < 9 {
		row = append(row, "")
	}
	if input.Type != nil {
		row[6] = string(*input.Type)
	}
	if input.Note != nil {
		row[7] = strings.TrimSpace(*input.Note)
	}
	sheetRow := accountsFirstRow + index
	if err := s.client.UpdateRow(fmt.Sprintf("Accounts!G%d:H%d", sheetRow, sheetRow), []any{row[6], row[7]}); err != nil {
		return domain.Account{}, fmt.Errorf("update account: %w", err)
	}
	s.invalidate("accounts:*")
	account, _ := mapAccount(row)
	return account, nil
}

// DeleteAccount soft-deletes by prefixing the name, so existing transactions
// keep their reference; reads exclude such rows (WM-DATA-5).
func (s *dbService) DeleteAccount(name string) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	rows, err := s.client.ReadSheet(accountsSheetRange)
	if err != nil {
		return fmt.Errorf("read accounts: %w", err)
	}
	index, found := findAccountRow(rows, name)
	if !found {
		return fmt.Errorf("%w: account %q", domain.ErrNotFound, name)
	}
	trimmed := strings.TrimSpace(name)
	if err := s.client.UpdateRow(fmt.Sprintf("Accounts!A%d", accountsFirstRow+index), []any{domain.DeletedAccountPrefix + " " + trimmed}); err != nil {
		return fmt.Errorf("delete account: %w", err)
	}
	s.invalidate("accounts:*")
	return nil
}

// findAccountRow returns the index in rows of the live account with the name.
func findAccountRow(rows [][]string, name string) (int, bool) {
	wanted := strings.TrimSpace(name)
	if wanted == "" || strings.HasPrefix(wanted, domain.DeletedAccountPrefix) {
		return 0, false
	}
	for i, row := range rows {
		if strings.TrimSpace(cellAt(row, 0)) == wanted {
			return i, true
		}
	}
	return 0, false
}
