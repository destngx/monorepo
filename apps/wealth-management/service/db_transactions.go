package service

import (
	"apps/wealth-management/domain"
	"fmt"
	"slices"
	"strings"
)

var transactionWritePatterns = []string{"transactions:*", "budget:*", "accounts:*"}

// ListTransactions reads the transactions and the categories that type them
// in one request when neither is cached.
func (s *dbService) ListTransactions(forceFresh bool) ([]domain.Transaction, error) {
	var categories []domain.Category
	var transactions []domain.Transaction
	err := s.batchRead(forceFresh, categoriesDataset(s, &categories), transactionsDataset(s, &transactions, &categories))
	return transactions, err
}

// ReadLedger fills a LedgerSnapshot with one request for whatever is not
// cached (WM-NFR-3).
func (s *dbService) ReadLedger(forceFresh bool) (domain.LedgerSnapshot, error) {
	var ledger domain.LedgerSnapshot
	var categories []domain.Category
	err := s.batchRead(forceFresh,
		dataset(s, cacheKeyAccounts, &ledger.Accounts, func(rows [][][]string) ([]domain.Account, error) {
			return parseAccounts(rows[0]), nil
		}, accountsSheetRange),
		dataset(s, cacheKeyLoans, &ledger.Loans, func(rows [][][]string) ([]domain.Loan, error) {
			return parseLoans(rows[0]), nil
		}, loansSheetRange),
		categoriesDataset(s, &categories),
		transactionsDataset(s, &ledger.Transactions, &categories),
	)
	if err != nil {
		return domain.LedgerSnapshot{}, err
	}
	return ledger, nil
}

func categoriesDataset(s *dbService, target *[]domain.Category) sheetDataset {
	return dataset(s, cacheKeyCats, target, func(rows [][][]string) ([]domain.Category, error) {
		return parseCategories(rows[0]), nil
	}, categoriesSheetRange)
}

// transactionsDataset types each transaction from *categories, so it must
// come after the categories dataset in the same batch.
func transactionsDataset(s *dbService, target *[]domain.Transaction, categories *[]domain.Category) sheetDataset {
	return dataset(s, cacheKeyTxs, target, func(rows [][][]string) ([]domain.Transaction, error) {
		return parseTransactions(rows[0], *categories), nil
	}, transactionsSheetRange)
}

func parseTransactions(rows [][]string, categories []domain.Category) []domain.Transaction {
	categoryIndex := buildCategoryIndex(categories)
	transactions := make([]domain.Transaction, 0, len(rows))
	for i, row := range rows {
		transaction, ok := mapTransaction(row, transactionsFirstRow+i)
		if !ok {
			continue
		}
		if categoryType, ok := categoryIndex[normalizeKey(transaction.Category)]; ok {
			transaction.CategoryType = categoryType
		}
		transactions = append(transactions, transaction)
	}
	sortTransactionsNewestFirst(transactions)
	return transactions
}

func (s *dbService) CreateTransaction(input domain.TransactionCreateInput) (domain.TransactionWriteResult, error) {
	if err := input.Validate(); err != nil {
		return domain.TransactionWriteResult{}, err
	}
	currencies, err := s.accountCurrencies()
	if err != nil {
		return domain.TransactionWriteResult{}, err
	}
	values, err := transactionRowValues(input, currencies.of(input.AccountName))
	if err != nil {
		return domain.TransactionWriteResult{}, err
	}
	row, err := s.client.WriteRowsAtFirstEmptyRow("Transactions", transactionsAnchorRange, [][]any{values})
	if err != nil {
		return domain.TransactionWriteResult{}, fmt.Errorf("write transaction: %w", err)
	}
	s.invalidate(transactionWritePatterns...)
	return domain.TransactionWriteResult{ID: transactionID(row)}, nil
}

// CreateTransactions writes every valid entry in one append and reports each
// entry's outcome. Replaying a BatchID returns the stored first result
// without writing again (WM-TX-10).
func (s *dbService) CreateTransactions(input domain.TransactionBatchInput) (domain.TransactionBatchResult, error) {
	if err := input.Validate(); err != nil {
		return domain.TransactionBatchResult{}, err
	}
	currencies, err := s.accountCurrencies()
	if err != nil {
		return domain.TransactionBatchResult{}, err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if previous, ok := s.batchReplay(input.BatchID); ok {
		previous.Replayed = true
		return previous, nil
	}

	result := domain.TransactionBatchResult{BatchID: input.BatchID, Rows: make([]domain.TransactionBatchRow, len(input.Transactions))}
	var values [][]any
	var written []int
	for i, entry := range input.Transactions {
		result.Rows[i].Index = i
		rowValues, err := validTransactionRow(entry, currencies.of(entry.AccountName))
		if err != nil {
			result.Rows[i].Error = err.Error()
			result.Failed++
			continue
		}
		values = append(values, rowValues)
		written = append(written, i)
	}
	if len(values) > 0 {
		firstRow, err := s.client.WriteRowsAtFirstEmptyRow("Transactions", transactionsAnchorRange, values)
		if err != nil {
			return domain.TransactionBatchResult{}, fmt.Errorf("write transaction batch: %w", err)
		}
		for offset, index := range written {
			result.Rows[index].ID = transactionID(firstRow + offset)
		}
		result.Created = len(written)
		s.invalidate(transactionWritePatterns...)
	}
	s.rememberBatch(result)
	return result, nil
}

func validTransactionRow(input domain.TransactionCreateInput, currency domain.Currency) ([]any, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	return transactionRowValues(input, currency)
}

// accountCurrencyIndex maps trimmed account names to their currency.
type accountCurrencyIndex map[string]domain.Currency

// of returns the account's currency; an unknown account is written as
// entered, in VND.
func (index accountCurrencyIndex) of(accountName string) domain.Currency {
	if currency, ok := index[strings.TrimSpace(accountName)]; ok {
		return currency
	}
	return domain.CurrencyVND
}

func (s *dbService) accountCurrencies() (accountCurrencyIndex, error) {
	accounts, err := s.ListAccounts(false)
	if err != nil {
		return nil, fmt.Errorf("read account currencies: %w", err)
	}
	index := make(accountCurrencyIndex, len(accounts))
	for _, account := range accounts {
		index[strings.TrimSpace(account.Name)] = account.Currency
	}
	return index, nil
}

// UpdateTransaction replaces columns A-J when the row still matches the ETag
// the caller read; otherwise it returns ErrConflict (WM-TX-8).
func (s *dbService) UpdateTransaction(id string, input domain.TransactionUpdateInput) (domain.TransactionWriteResult, error) {
	if err := input.Validate(); err != nil {
		return domain.TransactionWriteResult{}, err
	}
	currencies, err := s.accountCurrencies()
	if err != nil {
		return domain.TransactionWriteResult{}, err
	}
	values, err := transactionRowValues(input.TransactionCreateInput, currencies.of(input.AccountName))
	if err != nil {
		return domain.TransactionWriteResult{}, err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	rowRange, err := s.checkTransactionETag(id, input.ETag)
	if err != nil {
		return domain.TransactionWriteResult{}, err
	}
	if err := s.client.UpdateRow(rowRange, values); err != nil {
		return domain.TransactionWriteResult{}, fmt.Errorf("update transaction: %w", err)
	}
	s.invalidate(transactionWritePatterns...)
	return domain.TransactionWriteResult{ID: id}, nil
}

// DeleteTransaction clears columns A-J (K-M hold sheet formulas). Row numbers
// of other transactions do not shift, so their IDs stay valid.
func (s *dbService) DeleteTransaction(id string, etag string) error {
	if strings.TrimSpace(etag) == "" {
		return fmt.Errorf("%w: etag is required", domain.ErrValidation)
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	rowRange, err := s.checkTransactionETag(id, etag)
	if err != nil {
		return err
	}
	// Empty strings clear cells; nil values would leave them unchanged.
	blank := make([]any, transactionWriteColumns)
	for i := range blank {
		blank[i] = ""
	}
	if err := s.client.UpdateRow(rowRange, blank); err != nil {
		return fmt.Errorf("delete transaction: %w", err)
	}
	s.invalidate(transactionWritePatterns...)
	return nil
}

// checkTransactionETag re-reads the row and returns its A-J range when the
// row exists and is unchanged. Callers must hold writeMu.
func (s *dbService) checkTransactionETag(id string, etag string) (string, error) {
	row, err := parseTransactionID(id)
	if err != nil {
		return "", err
	}
	rowRange := fmt.Sprintf("Transactions!A%d:%s%d", row, columnLetter(transactionWriteColumns-1), row)
	rows, err := s.client.ReadSheet(rowRange)
	if err != nil {
		return "", fmt.Errorf("read transaction %s: %w", id, err)
	}
	current := rowAt(rows, 0)
	if strings.TrimSpace(strings.Join(current, "")) == "" {
		return "", fmt.Errorf("%w: transaction %s", domain.ErrNotFound, id)
	}
	if rowETag(current, transactionWriteColumns) != strings.TrimSpace(etag) {
		return "", fmt.Errorf("%w: transaction %s changed since it was read; reload and retry", domain.ErrConflict, id)
	}
	return rowRange, nil
}

func batchCacheKey(batchID string) string {
	return "txbatch:" + batchID
}

// batchReplay finds a stored batch result in process memory or the cache.
func (s *dbService) batchReplay(batchID string) (domain.TransactionBatchResult, bool) {
	if result, ok := s.localReplays[batchID]; ok {
		return result, true
	}
	return readCached[domain.TransactionBatchResult](s, batchCacheKey(batchID))
}

// rememberBatch stores the result in the cache and in a bounded in-process
// map, so a retry is recognised even when the cache is down.
func (s *dbService) rememberBatch(result domain.TransactionBatchResult) {
	stored := result
	stored.Rows = slices.Clone(result.Rows)
	s.writeCache(batchCacheKey(result.BatchID), stored, batchReplayTTL)
	if _, exists := s.localReplays[result.BatchID]; !exists {
		s.replayHistory = append(s.replayHistory, result.BatchID)
	}
	s.localReplays[result.BatchID] = stored
	for len(s.replayHistory) > maxLocalReplays {
		delete(s.localReplays, s.replayHistory[0])
		s.replayHistory = s.replayHistory[1:]
	}
}
