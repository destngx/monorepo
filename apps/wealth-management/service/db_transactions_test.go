package service

import (
	"apps/wealth-management/domain"
	"errors"
	"testing"
)

func validTransactionInput() domain.TransactionCreateInput {
	return domain.TransactionCreateInput{
		AccountName: "Bank", Date: "2026-10-12", Payee: "Coffee", Category: "Food",
		Tags: []string{" a ", "", "b"}, Payment: ptr(50000.0),
	}
}

func seedTransactions(client *fakeSheetClient) {
	client.ranges[transactionsSheetRange] = [][]string{
		{"Bank", "46300", "", "Old", "", "", "Food", "", "1000"},
		{"Bank", "46300", "", "Same day later row", "", "", "Salary", "", "", "5000"},
		{"Bank", "garbage", "", "Bad date", "", "", "Food", "", "1"},
		{"Bank", "46301", "", "Newest", "", "", "Food", "", "2000"},
	}
	client.ranges[categoriesSheetRange] = [][]string{{"Income Categories"}, {"Salary"}, {"Expense Categories"}, {"Food"}}
}

func TestListTransactionsNewestFirstWithTypesAndIssues(t *testing.T) {
	client := newFakeSheetClient()
	seedTransactions(client)
	svc := newTestDBService(t, client, newFakeCache())

	txs, err := svc.ListTransactions(false)
	if err != nil {
		t.Fatal(err)
	}
	payees := []string{}
	for _, tx := range txs {
		payees = append(payees, tx.Payee)
	}
	want := []string{"Newest", "Same day later row", "Old", "Bad date"}
	for i := range want {
		if payees[i] != want[i] {
			t.Fatalf("order = %v, want %v", payees, want)
		}
	}
	if txs[0].ID != "row-5" || txs[0].CategoryType == nil || *txs[0].CategoryType != domain.CategoryTypeExpense {
		t.Fatalf("first = %+v", txs[0])
	}
	if len(txs[3].Issues) != 1 || !txs[3].Date.IsZero() {
		t.Fatalf("bad date row must be flagged, got %+v", txs[3])
	}
}

func TestListTransactionsDegradesWhenCacheIsDown(t *testing.T) {
	client := newFakeSheetClient()
	seedTransactions(client)
	cache := newFakeCache()
	cache.failing = true
	svc := newTestDBService(t, client, cache)

	txs, err := svc.ListTransactions(false)
	if err != nil || len(txs) != 4 {
		t.Fatalf("got %d, %v; want sheet read despite cache failure", len(txs), err)
	}
}

func TestCreateTransactionReturnsRowIDAndInvalidatesBalances(t *testing.T) {
	client := newFakeSheetClient()
	cache := newFakeCache()
	svc := newTestDBService(t, client, cache)

	result, err := svc.CreateTransaction(validTransactionInput())
	if err != nil {
		t.Fatal(err)
	}
	if result.ID != "row-100" {
		t.Fatalf("id = %s", result.ID)
	}
	row := client.appends[0].rows[0]
	if row[1] != "12/10/2026" || row[4] != "a,b" || row[8] != 50000.0 || row[9] != "" {
		t.Fatalf("row = %#v", row)
	}
	if len(cache.invalidated) != 3 || cache.invalidated[2] != "accounts:*" {
		t.Fatalf("invalidated = %v", cache.invalidated)
	}
}

func TestCreateTransactionRejectsInvalidInput(t *testing.T) {
	client := newFakeSheetClient()
	svc := newTestDBService(t, client, newFakeCache())
	input := validTransactionInput()
	input.Date = "31/02/2026"

	if _, err := svc.CreateTransaction(input); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("err = %v", err)
	}
	if len(client.appends) != 0 {
		t.Fatal("nothing must be written")
	}
}

func TestBatchCreateReportsPerRowAndIsIdempotent(t *testing.T) {
	client := newFakeSheetClient()
	cache := newFakeCache()
	svc := newTestDBService(t, client, cache)
	bad := validTransactionInput()
	bad.Payee = ""
	input := domain.TransactionBatchInput{BatchID: "b1", Transactions: []domain.TransactionCreateInput{validTransactionInput(), bad, validTransactionInput()}}

	first, err := svc.CreateTransactions(input)
	if err != nil {
		t.Fatal(err)
	}
	if first.Created != 2 || first.Failed != 1 || first.Rows[0].ID != "row-100" || first.Rows[2].ID != "row-101" || first.Rows[1].Error == "" {
		t.Fatalf("first = %+v", first)
	}

	// Replays come from the cache, then from memory when the cache is down.
	for _, cacheDown := range []bool{false, true} {
		cache.failing = cacheDown
		replay, err := svc.CreateTransactions(input)
		if err != nil {
			t.Fatal(err)
		}
		if !replay.Replayed || replay.Created != 2 || len(client.appends) != 1 {
			t.Fatalf("cacheDown=%v replay = %+v, appends = %d", cacheDown, replay, len(client.appends))
		}
	}
}

func TestBatchReplayFromCacheAfterRestart(t *testing.T) {
	client := newFakeSheetClient()
	cache := newFakeCache()
	input := domain.TransactionBatchInput{BatchID: "b2", Transactions: []domain.TransactionCreateInput{validTransactionInput()}}
	if _, err := newTestDBService(t, client, cache).CreateTransactions(input); err != nil {
		t.Fatal(err)
	}

	replay, err := newTestDBService(t, client, cache).CreateTransactions(input)
	if err != nil || !replay.Replayed || len(client.appends) != 1 {
		t.Fatalf("replay = %+v, %v, appends = %d", replay, err, len(client.appends))
	}
}

func TestUpdateTransactionChecksETag(t *testing.T) {
	stored := []string{"Bank", "46300", "", "Coffee", "", "", "Food", "", "1000"}
	etag := rowETag(stored, transactionWriteColumns)
	update := func(id string, etag string) (*fakeSheetClient, error) {
		client := newFakeSheetClient()
		client.ranges["Transactions!A7:J7"] = [][]string{stored}
		client.ranges["Transactions!A8:J8"] = [][]string{}
		svc := newTestDBService(t, client, newFakeCache())
		_, err := svc.UpdateTransaction(id, domain.TransactionUpdateInput{TransactionCreateInput: validTransactionInput(), ETag: etag})
		return client, err
	}

	client, err := update("row-7", etag)
	if err != nil || len(client.updates) != 1 || client.updates[0].rangeValue != "Transactions!A7:J7" {
		t.Fatalf("err = %v, updates = %+v", err, client.updates)
	}
	if _, err := update("row-7", "stale"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("stale etag err = %v", err)
	}
	if _, err := update("row-8", etag); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("empty row err = %v", err)
	}
	if _, err := update("7", etag); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("bad id err = %v", err)
	}
}

func TestDeleteTransactionClearsUserColumns(t *testing.T) {
	stored := []string{"Bank", "46300", "", "Coffee"}
	client := newFakeSheetClient()
	client.ranges["Transactions!A3:J3"] = [][]string{stored}
	svc := newTestDBService(t, client, newFakeCache())

	if err := svc.DeleteTransaction("row-3", rowETag(stored, transactionWriteColumns)); err != nil {
		t.Fatal(err)
	}
	values := client.updates[0].values
	if len(values) != transactionWriteColumns {
		t.Fatalf("values = %#v", values)
	}
	for _, value := range values {
		if value != "" {
			t.Fatalf("cells must be cleared with empty strings, got %#v", values)
		}
	}
	if err := svc.DeleteTransaction("row-3", ""); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("missing etag err = %v", err)
	}
}
