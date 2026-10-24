package service

import (
	"apps/wealth-management/domain"
	"context"
	"sync"
	"testing"
	"time"
)

// heldSheetClient holds each read until release is closed, and is safe for
// concurrent use.
type heldSheetClient struct {
	*fakeSheetClient
	mu      sync.Mutex
	release chan struct{}
	started chan struct{}
}

func newHeldSheetClient(client *fakeSheetClient) *heldSheetClient {
	return &heldSheetClient{fakeSheetClient: client, release: make(chan struct{}), started: make(chan struct{}, 16)}
}

func (c *heldSheetClient) ReadSheet(rangeValue string) ([][]string, error) {
	c.started <- struct{}{}
	<-c.release
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.fakeSheetClient.ReadSheet(rangeValue)
}

func (c *heldSheetClient) ReadSheets(ranges []string) ([][][]string, error) {
	c.started <- struct{}{}
	<-c.release
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.fakeSheetClient.ReadSheets(ranges)
}

func (c *heldSheetClient) requestCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.requests
}

type lockedCache struct {
	mu    sync.Mutex
	inner *fakeCache
}

func (c *lockedCache) Set(ctx context.Context, key string, value string, ttl int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.inner.Set(ctx, key, value, ttl)
}

func (c *lockedCache) Get(ctx context.Context, key string) (domain.CacheEntry, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.inner.Get(ctx, key)
}

func (c *lockedCache) Invalidate(ctx context.Context, pattern string) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.inner.Invalidate(ctx, pattern)
}

func (c *lockedCache) Ping(ctx context.Context) error { return nil }

func newHeldDBService(t *testing.T, client *fakeSheetClient) (*dbService, *heldSheetClient) {
	t.Helper()
	svc := newTestDBService(t, client, newFakeCache())
	held := newHeldSheetClient(client)
	svc.client = held
	svc.cache = &lockedCache{inner: newFakeCache()}
	return svc, held
}

// readConcurrently starts the first read and holds its sheet request while
// the other reads arrive, then releases it.
func readConcurrently(t *testing.T, held *heldSheetClient, reads ...func() error) {
	t.Helper()
	errs := make(chan error, len(reads))
	go func() { errs <- reads[0]() }()
	<-held.started
	for _, read := range reads[1:] {
		go func() { errs <- read() }()
	}
	// Give the other reads time to find the tab busy.
	time.Sleep(20 * time.Millisecond)
	close(held.release)
	for range reads {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
}

func TestGivenAColdCacheWhenTheSameTabIsReadAtOnceThenOneSheetRequestServesAll(t *testing.T) {
	client := newFakeSheetClient()
	seedAccounts(client)
	svc, held := newHeldDBService(t, client)

	read := func() error {
		accounts, err := svc.ListAccounts(false)
		if err == nil && len(accounts) != 2 {
			t.Errorf("accounts = %+v", accounts)
		}
		return err
	}
	readConcurrently(t, held, read, read, read)

	if got := held.requestCount(); got != 1 {
		t.Fatalf("%d sheet requests, want 1", got)
	}
}

func TestGivenForcedReadsAtOnceWhenTheyOverlapThenTheyShareTheReadInFlight(t *testing.T) {
	client := newFakeSheetClient()
	seedTransactions(client)
	svc, held := newHeldDBService(t, client)

	transactions := func() error {
		txs, err := svc.ListTransactions(true)
		if err == nil && len(txs) == 0 {
			t.Error("no transactions")
		}
		return err
	}
	categories := func() error {
		_, err := svc.ListCategories(true)
		return err
	}
	readConcurrently(t, held, transactions, transactions, categories)

	if got := held.requestCount(); got != 1 {
		t.Fatalf("%d sheet requests, want 1", got)
	}
}

func TestGivenAReadThatFailedWhenAnotherWaitedForItThenItReadsItself(t *testing.T) {
	client := newFakeSheetClient()
	seedAccounts(client)
	svc := newTestDBService(t, client, newFakeCache())

	_, release := svc.reads.claim(cacheKeyAccounts)
	done := make(chan error)
	go func() {
		_, err := svc.ListAccounts(false)
		done <- err
	}()
	time.Sleep(20 * time.Millisecond)
	// The claimed read ends without caching anything, as a failed read does.
	release()

	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if client.requests != 1 {
		t.Fatalf("%d sheet requests, want 1", client.requests)
	}
}
