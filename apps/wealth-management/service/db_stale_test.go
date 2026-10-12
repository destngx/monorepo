package service

import (
	"apps/wealth-management/domain"
	"errors"
	"strings"
	"testing"
	"time"
)

var errSheetDown = errors.New("googleapi: Error 503: backend unavailable")

// expireFresh drops the fresh cache copies, as their 300 s TTL would.
func expireFresh(cache *fakeCache) {
	for key := range cache.values {
		if !strings.HasPrefix(key, domain.LastGoodKey("")) {
			delete(cache.values, key)
		}
	}
}

func TestGivenSheetDownAndALastGoodCopyWhenListingAccountsThenServesItAndReportsItsTime(t *testing.T) {
	client := newFakeSheetClient()
	seedAccounts(client)
	cache := newFakeCache()
	svc := newTestDBService(t, client, cache)
	readAt := fixedNow
	if _, err := svc.ListAccounts(false); err != nil {
		t.Fatal(err)
	}

	expireFresh(cache)
	client.readErr[accountsSheetRange] = errSheetDown
	svc.now = func() time.Time { return fixedNow.Add(time.Hour) }
	cursor := svc.StaleCursor()
	accounts, err := svc.ListAccounts(false)

	if err != nil || len(accounts) != 2 {
		t.Fatalf("want the last good accounts, got %+v %v", accounts, err)
	}
	asOf, stale := svc.StaleSince(cursor)
	if !stale || !asOf.Equal(readAt) {
		t.Fatalf("stale = %v asOf = %v, want %v", stale, asOf, readAt)
	}
}

func TestGivenSheetDownAndNoLastGoodCopyWhenListingAccountsThenFails(t *testing.T) {
	client := newFakeSheetClient()
	client.readErr[accountsSheetRange] = errSheetDown
	svc := newTestDBService(t, client, newFakeCache())

	if _, err := svc.ListAccounts(false); !errors.Is(err, errSheetDown) {
		t.Fatalf("err = %v, want the sheet error", err)
	}
	if _, stale := svc.StaleSince(0); stale {
		t.Fatal("nothing was served stale")
	}
}

func TestGivenStaleDataWhenTheSheetRecoversThenLaterReadsAreNotStale(t *testing.T) {
	client := newFakeSheetClient()
	seedAccounts(client)
	cache := newFakeCache()
	svc := newTestDBService(t, client, cache)
	if _, err := svc.ListAccounts(false); err != nil {
		t.Fatal(err)
	}
	client.readErr[accountsSheetRange] = errSheetDown
	if _, err := svc.ListAccounts(true); err != nil {
		t.Fatal(err)
	}

	delete(client.readErr, accountsSheetRange)
	cursor := svc.StaleCursor()
	if _, err := svc.ListAccounts(true); err != nil {
		t.Fatal(err)
	}
	if _, stale := svc.StaleSince(cursor); stale {
		t.Fatal("a fresh read must not be reported stale")
	}
}

func TestGivenAStaleReadInAnEarlierRequestWhenAFreshCacheHitIsServedThenItIsNotStale(t *testing.T) {
	client := newFakeSheetClient()
	seedAccounts(client)
	client.ranges[categoriesSheetRange] = [][]string{{"Expense Categories"}, {"Food"}}
	cache := newFakeCache()
	svc := newTestDBService(t, client, cache)
	if _, err := svc.ListAccounts(false); err != nil {
		t.Fatal(err)
	}
	client.readErr[accountsSheetRange] = errSheetDown
	if _, err := svc.ListAccounts(true); err != nil {
		t.Fatal(err)
	}

	cursor := svc.StaleCursor()
	if _, err := svc.ListCategories(false); err != nil {
		t.Fatal(err)
	}
	if _, stale := svc.StaleSince(cursor); stale {
		t.Fatal("only reads served stale during the request count")
	}
}

func TestGivenAMissingBudgetTabWhenListingThenNotFoundIsNotMaskedByALastGoodCopy(t *testing.T) {
	client := newFakeSheetClient()
	client.titles = []string{"Accounts"}
	cache := newFakeCache()
	cache.values[domain.LastGoodKey(budgetCacheKey(2026))] = `{"asOf":"2026-01-01T00:00:00Z","value":[]}`
	svc := newTestDBService(t, client, cache)

	if _, err := svc.ListBudget(2026, false); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestGivenSyncWhenInvalidatingThenLastGoodCopiesSurvive(t *testing.T) {
	client := newFakeSheetClient()
	seedAccounts(client)
	cache := newFakeCache()
	svc := newTestDBService(t, client, cache)
	if _, err := svc.ListAccounts(false); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.SyncCache(); err != nil {
		t.Fatal(err)
	}
	if _, ok := cache.values[cacheKeyAccounts]; ok {
		t.Fatal("sync must drop the fresh copy")
	}
	if _, ok := cache.values[domain.LastGoodKey(cacheKeyAccounts)]; !ok {
		t.Fatal("sync must keep the last good copy")
	}
}

func seedLedger(client *fakeSheetClient) {
	seedAccounts(client)
	client.ranges[loansSheetRange] = [][]string{}
	client.ranges[categoriesSheetRange] = [][]string{{"Expense Categories"}, {"Food"}}
	client.ranges[transactionsSheetRange] = [][]string{
		{"Bank", "01/10/2026", "", "Shop", "", "", "Food", "", "50000"},
	}
}

func TestGivenAColdCacheWhenReadingTheLedgerThenOneBatchedRequestReadsEveryRange(t *testing.T) {
	client := newFakeSheetClient()
	seedLedger(client)
	svc := newTestDBService(t, client, newFakeCache())

	ledger, err := svc.ReadLedger(false)

	if err != nil {
		t.Fatal(err)
	}
	if client.requests != 1 || len(client.reads) != 4 {
		t.Fatalf("requests = %d reads = %v, want one batch of 4 ranges", client.requests, client.reads)
	}
	if len(ledger.Accounts) != 2 || len(ledger.Transactions) != 1 || ledger.Transactions[0].CategoryType == nil {
		t.Fatalf("ledger = %+v", ledger)
	}
}

func TestGivenSomeDatasetsCachedWhenReadingTheLedgerThenOnlyMissesAreRead(t *testing.T) {
	client := newFakeSheetClient()
	seedLedger(client)
	svc := newTestDBService(t, client, newFakeCache())
	if _, err := svc.ListAccounts(false); err != nil {
		t.Fatal(err)
	}
	client.reads, client.requests = nil, 0

	if _, err := svc.ReadLedger(false); err != nil {
		t.Fatal(err)
	}
	if client.requests != 1 || len(client.reads) != 3 {
		t.Fatalf("requests = %d reads = %v, want the 3 uncached ranges in one batch", client.requests, client.reads)
	}

	client.reads, client.requests = nil, 0
	if _, err := svc.ReadLedger(false); err != nil {
		t.Fatal(err)
	}
	if client.requests != 0 {
		t.Fatalf("a warm cache must not read the sheet, reads = %v", client.reads)
	}
}

func TestGivenSheetDownWhenReadingTheLedgerThenEveryDatasetFallsBackToItsLastGoodCopy(t *testing.T) {
	client := newFakeSheetClient()
	seedLedger(client)
	cache := newFakeCache()
	svc := newTestDBService(t, client, cache)
	if _, err := svc.ReadLedger(false); err != nil {
		t.Fatal(err)
	}

	client.readErr[transactionsSheetRange] = errSheetDown
	cursor := svc.StaleCursor()
	ledger, err := svc.ReadLedger(true)

	if err != nil || len(ledger.Accounts) != 2 || len(ledger.Transactions) != 1 {
		t.Fatalf("ledger = %+v err = %v", ledger, err)
	}
	if _, stale := svc.StaleSince(cursor); !stale {
		t.Fatal("the ledger was served stale")
	}
}

func TestGivenAIContentWhenLoadingThenPromptsAndKnowledgeComeFromOneRequest(t *testing.T) {
	client := newFakeSheetClient()
	client.ranges[promptsSheetRange] = [][]string{{"tasks", "review", "Review it"}}
	client.ranges[knowledgeSheetRange] = [][]string{{"rule", "Save first"}}
	svc := newTestDBService(t, client, newFakeCache())

	content, err := svc.LoadAIContent(true)

	if err != nil || content.Prompts["tasks/review"] != "Review it" || content.Knowledge["rule"] != "Save first" {
		t.Fatalf("content = %+v err = %v", content, err)
	}
	if client.requests != 1 {
		t.Fatalf("requests = %d, want 1", client.requests)
	}
}

func TestGivenInvestmentTabsWhenReadingAssetsThenCryptoAndFundsComeFromOneRequest(t *testing.T) {
	client := newFakeSheetClient()
	client.ranges[cryptoSheetRange] = [][]string{}
	client.ranges[fundsSheetRange] = [][]string{}
	svc := newTestDBService(t, client, newFakeCache())

	if _, err := svc.GetInvestmentAssets(true); err != nil {
		t.Fatal(err)
	}
	if client.requests != 1 || len(client.reads) != 2 {
		t.Fatalf("requests = %d reads = %v, want one batch", client.requests, client.reads)
	}
}
