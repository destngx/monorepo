package service

import (
	"apps/wealth-management/domain"
	"fmt"
	"slices"
	"sort"
	"strings"
)

// Row mappers for each sheet tab (REQUIREMENTS §2).

const (
	transactionWriteColumns = 10 // A-J, every Transactions column
	budgetHeaderRowIndex    = 9  // row 10: month serials from column D
	budgetFirstDataRowIndex = 11 // row 12
	budgetFirstMonthColumn  = 3  // column D
	budgetYearlyColumn      = 2  // column C
)

func mapAccount(row []string) (domain.Account, bool) {
	name := strings.TrimSpace(cellAt(row, 0))
	if isJunkAccountName(name) || strings.HasPrefix(name, domain.DeletedAccountPrefix) {
		return domain.Account{}, false
	}
	typeRaw := strings.ToLower(strings.TrimSpace(cellAt(row, 6)))
	if typeRaw == "type" || typeRaw == "date to pay" {
		return domain.Account{}, false
	}
	if typeRaw == "" {
		typeRaw = string(domain.AccountTypeBank)
	}
	var dueDate *string
	if value := formatDueDate(cellAt(row, 1)); value != "" {
		dueDate = &value
	}
	return domain.Account{
		Name:           name,
		DueDate:        dueDate,
		CreditLimit:    nullableNumber(cellAt(row, 2)),
		ClearedBalance: parseNumber(cellAt(row, 4)),
		Balance:        parseNumber(cellAt(row, 5)),
		Type:           domain.AccountType(typeRaw),
		Currency:       resolveCurrency(cellAt(row, 8), name),
		Note:           nullableString(cellAt(row, 7)),
	}, true
}

// mapTransaction maps a Transactions row; rowNumber is the 1-based sheet row.
// An unparseable date is flagged in Issues instead of becoming "now" (WM-DATA-3).
func mapTransaction(row []string, rowNumber int) (domain.Transaction, bool) {
	accountName := strings.TrimSpace(cellAt(row, 0))
	if isJunkAccountName(accountName) {
		return domain.Transaction{}, false
	}
	category := strings.TrimSpace(cellAt(row, 6))
	if category == "Category" || accountName == "Account" {
		return domain.Transaction{}, false
	}
	if category == "" {
		category = "Uncategorized"
	}
	transaction := domain.Transaction{
		ID:              transactionID(rowNumber),
		AccountName:     accountName,
		ReferenceNumber: nullableString(cellAt(row, 2)),
		Payee:           strings.TrimSpace(cellAt(row, 3)),
		Tags:            splitTags(cellAt(row, 4)),
		Memo:            nullableString(cellAt(row, 5)),
		Category:        category,
		Cleared:         parseCleared(cellAt(row, 7)),
		Payment:         nullableNumber(cellAt(row, 8)),
		Deposit:         nullableNumber(cellAt(row, 9)),
		ETag:            rowETag(row, transactionWriteColumns),
	}
	date, err := domain.ParseSheetDate(cellAt(row, 1))
	if err != nil {
		transaction.Issues = append(transaction.Issues, fmt.Sprintf("unparseable date %q", strings.TrimSpace(cellAt(row, 1))))
	} else {
		transaction.Date = date
	}
	return transaction, true
}

func transactionID(rowNumber int) string {
	return fmt.Sprintf("row-%d", rowNumber)
}

// parseTransactionID returns the sheet row of a "row-N" id.
func parseTransactionID(id string) (int, error) {
	var row int
	if _, err := fmt.Sscanf(strings.TrimSpace(id), "row-%d", &row); err != nil || row < 2 || transactionID(row) != strings.TrimSpace(id) {
		return 0, fmt.Errorf("%w: transaction id %q must look like row-N", domain.ErrValidation, id)
	}
	return row, nil
}

// transactionRowValues renders columns A-J in the sheet's write format;
// currency is the account's, which decides USD conversion.
func transactionRowValues(input domain.TransactionCreateInput, currency domain.Currency) ([]any, error) {
	date, err := domain.ParseSheetDate(input.Date)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrValidation, err)
	}
	return []any{
		strings.TrimSpace(input.AccountName),
		date.Format(domain.SheetDateLayout),
		nilToString(input.ReferenceNumber),
		strings.TrimSpace(input.Payee),
		strings.Join(cleanStrings(input.Tags), ","),
		nilToString(input.Memo),
		strings.TrimSpace(input.Category),
		clearedCell(input.Cleared),
		formatTransactionAmount(input.Payment, input.AccountName, currency),
		formatTransactionAmount(input.Deposit, input.AccountName, currency),
	}, nil
}

func isJunkAccountName(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	if lower == "" {
		return true
	}
	switch lower {
	case "help", "accounts", "account":
		return true
	}
	for _, prefix := range []string{"you can track", "accounts column", "to get started"} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

// sortTransactionsNewestFirst orders by date, later sheet rows first on the
// same day (WM-TX-1). Rows arrive in sheet order.
func sortTransactionsNewestFirst(transactions []domain.Transaction) {
	slices.Reverse(transactions)
	sort.SliceStable(transactions, func(i, j int) bool {
		return transactions[i].Date.After(transactions[j].Date)
	})
}
