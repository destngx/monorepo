package service

import (
	"apps/wealth-management/domain"
	"fmt"
	"slices"
	"strings"
)

var transactionWritePatterns = []string{"transactions:*", "budget:*", "accounts:*"}

func (s *dbService) ListTransactions(forceFresh bool) ([]domain.Transaction, error) {
	return cachedRead(s, cacheKeyTxs, sheetDataTTL, forceFresh, func() ([]domain.Transaction, error) {
		rows, err := s.client.ReadSheet(transactionsSheetRange)
		if err != nil {
			return nil, fmt.Errorf("read transactions: %w", err)
		}
		categories, err := s.ListCategories(forceFresh)
		if err != nil {
			return nil, err
		}
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
		return transactions, nil
	})
}

func (s *dbService) CreateTransaction(input domain.TransactionCreateInput) (domain.TransactionWriteResult, error) {
	if err := input.Validate(); err != nil {
		return domain.TransactionWriteResult{}, err
	}
	values, err := transactionRowValues(input)
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
		rowValues, err := validTransactionRow(entry)
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

func validTransactionRow(input domain.TransactionCreateInput) ([]any, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	return transactionRowValues(input)
}

// UpdateTransaction replaces columns A-J when the row still matches the ETag
// the caller read; otherwise it returns ErrConflict (WM-TX-8).
func (s *dbService) UpdateTransaction(id string, input domain.TransactionUpdateInput) (domain.TransactionWriteResult, error) {
	if err := input.Validate(); err != nil {
		return domain.TransactionWriteResult{}, err
	}
	values, err := transactionRowValues(input.TransactionCreateInput)
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
