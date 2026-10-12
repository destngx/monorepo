package service

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"errors"
	"strings"
	"testing"
	"time"
)

// budgetRows builds a minimal budget tab: month serials in row 10 from
// column D (Jan = 46023, Feb = 46054 for 2026) and categories from row 12.
func budgetRows() [][]string {
	rows := make([][]string, 14)
	rows[budgetHeaderRowIndex] = []string{"", "", "", "46023", "46054"}
	rows[11] = []string{"Expense Categories"}
	rows[12] = []string{"Food", "groceries", "12000000", "900000", ""}
	rows[13] = []string{"Total Expenses", "", "99"}
	return rows
}

func TestListBudgetUsesYearTemplate(t *testing.T) {
	client := newFakeSheetClient()
	client.ranges["'Budget_2026'!A1:Z200"] = budgetRows()
	svc := newTestDBService(t, client, newFakeCache())

	items, err := svc.ListBudget(0, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Category != "Food" || items[0].MonthlyLimits["2026-01"] != 900000 || items[0].MonthlyLimit != 1000000 {
		t.Fatalf("items = %+v", items)
	}
}

func TestListBudgetMissingTabIsNotFound(t *testing.T) {
	client := newFakeSheetClient()
	client.titles = []string{"Accounts", "Budget_2026"}
	svc := newTestDBService(t, client, newFakeCache())

	if _, err := svc.ListBudget(2027, false); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestUpdateBudgetLimitWritesMonthOrYearCell(t *testing.T) {
	cases := []struct {
		name  string
		input domain.BudgetLimitUpdate
		cell  string
		err   error
	}{
		{"month column", domain.BudgetLimitUpdate{Category: "food", Month: "2026-02", Limit: 5}, "'Budget_2026'!E13", nil},
		{"yearly column", domain.BudgetLimitUpdate{Year: 2026, Category: "Food", Limit: 5}, "'Budget_2026'!C13", nil},
		{"unknown category", domain.BudgetLimitUpdate{Category: "Travel", Limit: 5}, "", domain.ErrNotFound},
		{"month without column", domain.BudgetLimitUpdate{Category: "Food", Month: "2026-03", Limit: 5}, "", domain.ErrNotFound},
		{"negative", domain.BudgetLimitUpdate{Category: "Food", Limit: -1}, "", domain.ErrValidation},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := newFakeSheetClient()
			client.ranges["'Budget_2026'!A1:Z200"] = budgetRows()
			cache := newFakeCache()
			svc := newTestDBService(t, client, cache)

			err := svc.UpdateBudgetLimit(tc.input)
			if tc.err != nil {
				if !errors.Is(err, tc.err) {
					t.Fatalf("err = %v, want %v", err, tc.err)
				}
				return
			}
			if err != nil || len(client.updates) != 1 || client.updates[0].rangeValue != tc.cell {
				t.Fatalf("err = %v, updates = %+v", err, client.updates)
			}
			if len(cache.invalidated) != 1 || cache.invalidated[0] != "budget:2026" {
				t.Fatalf("invalidated = %v", cache.invalidated)
			}
		})
	}
}

// reallocationRows adds a Fun row (no February cell, so its limit is the
// yearly 6M / 12 = 500k) below budgetRows' Food row (February 0 = yearly 1M).
func reallocationRows() [][]string {
	rows := budgetRows()
	rows[12] = []string{"Food", "groceries", "12000000", "900000", "2000000"}
	rows[13] = []string{"Fun", "", "6000000", "", ""}
	return append(rows, []string{"Total Expenses", "", "99"})
}

func newReallocationService(t *testing.T) (*dbService, *fakeSheetClient, *fakeCache) {
	t.Helper()
	client := newFakeSheetClient()
	client.ranges["'Budget_2026'!A1:Z200"] = reallocationRows()
	cache := newFakeCache()
	return newTestDBService(t, client, cache), client, cache
}

func TestGivenEnoughLimitWhenReallocatingThenBothMonthCellsMoveAndTheTotalIsKept(t *testing.T) {
	svc, client, cache := newReallocationService(t)

	result, err := svc.ReallocateBudget(domain.BudgetReallocation{Year: 2026, Month: "2026-02", From: "food", To: "Fun", Amount: 300000})
	if err != nil {
		t.Fatal(err)
	}
	if result.From != (domain.BudgetCategoryLimit{Category: "Food", Limit: 1700000}) || result.To != (domain.BudgetCategoryLimit{Category: "Fun", Limit: 800000}) {
		t.Fatalf("result = %+v", result)
	}
	if len(client.updates) != 2 || client.updates[0].rangeValue != "'Budget_2026'!E13" || client.updates[0].values[0] != 1700000.0 ||
		client.updates[1].rangeValue != "'Budget_2026'!E14" || client.updates[1].values[0] != 800000.0 {
		t.Fatalf("updates = %+v", client.updates)
	}
	if len(cache.invalidated) == 0 || cache.invalidated[0] != "budget:2026" {
		t.Fatalf("invalidated = %v", cache.invalidated)
	}
}

func TestGivenFreshSheetWhenReallocatingThenLimitsAreReadFromTheSheetNotTheCache(t *testing.T) {
	svc, client, cache := newReallocationService(t)
	cache.values["budget:2026"] = `[{"category":"Food","monthlyLimits":{"2026-02":99}}]`

	if _, err := svc.ReallocateBudget(domain.BudgetReallocation{Month: "2026-02", From: "Food", To: "Fun", Amount: 2000000}); err != nil {
		t.Fatalf("a stale cached limit must not block the move: %v", err)
	}
	if client.updates[0].values[0] != 0.0 {
		t.Fatalf("updates = %+v", client.updates)
	}
}

func TestGivenInvalidMoveWhenReallocatingThenValidationFailsWithoutWriting(t *testing.T) {
	cases := map[string]domain.BudgetReallocation{
		"zero amount":       {Month: "2026-02", From: "Food", To: "Fun", Amount: 0},
		"same category":     {Month: "2026-02", From: "Food", To: " food ", Amount: 1},
		"not enough limit":  {Month: "2026-02", From: "Fun", To: "Food", Amount: 500001},
		"missing month":     {From: "Food", To: "Fun", Amount: 1},
		"month out of year": {Year: 2025, Month: "2026-02", From: "Food", To: "Fun", Amount: 1},
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			svc, client, _ := newReallocationService(t)
			if _, err := svc.ReallocateBudget(input); !errors.Is(err, domain.ErrValidation) {
				t.Fatalf("err = %v", err)
			}
			if len(client.updates) != 0 {
				t.Fatalf("updates = %+v", client.updates)
			}
		})
	}
}

func TestGivenUnknownCategoryWhenReallocatingThenNotFound(t *testing.T) {
	svc, client, _ := newReallocationService(t)
	if _, err := svc.ReallocateBudget(domain.BudgetReallocation{Month: "2026-02", From: "Food", To: "Travel", Amount: 1}); !errors.Is(err, domain.ErrNotFound) || len(client.updates) != 0 {
		t.Fatalf("err = %v, updates = %+v", err, client.updates)
	}
}

func TestGivenSecondWriteFailsWhenReallocatingThenTheFirstCellIsRestoredAndTheErrorReported(t *testing.T) {
	svc, client, cache := newReallocationService(t)
	writeFailure := errors.New("quota exceeded")
	client.writeErrFor["'Budget_2026'!E14"] = writeFailure

	_, err := svc.ReallocateBudget(domain.BudgetReallocation{Month: "2026-02", From: "Food", To: "Fun", Amount: 300000})
	if !errors.Is(err, writeFailure) || !strings.Contains(err.Error(), "restored") {
		t.Fatalf("err = %v", err)
	}
	if len(client.updates) != 2 || client.updates[1].rangeValue != "'Budget_2026'!E13" || client.updates[1].values[0] != "2000000" {
		t.Fatalf("restore = %+v", client.updates)
	}
	if len(cache.invalidated) == 0 {
		t.Fatal("the cache must be dropped after a partial write")
	}
}

func TestGivenRestoreAlsoFailsWhenReallocatingThenBothErrorsAreReported(t *testing.T) {
	svc, client, _ := newReallocationService(t)
	moveFailure, restoreFailure := errors.New("quota exceeded"), errors.New("network down")
	client.writeErrFor["'Budget_2026'!E14"] = moveFailure
	svc.client = &restoreFailingClient{fakeSheetClient: client, failAfter: 1, err: restoreFailure}

	_, err := svc.ReallocateBudget(domain.BudgetReallocation{Month: "2026-02", From: "Food", To: "Fun", Amount: 300000})
	if !errors.Is(err, moveFailure) || !errors.Is(err, restoreFailure) || !strings.Contains(err.Error(), "by hand") {
		t.Fatalf("err = %v", err)
	}
}

// restoreFailingClient fails every UpdateRow after the first failAfter
// successful ones, other than ranges the wrapped client fails itself.
type restoreFailingClient struct {
	*fakeSheetClient
	failAfter int
	err       error
}

func (c *restoreFailingClient) UpdateRow(rangeValue string, values []any) error {
	if err := c.fakeSheetClient.writeErrFor[rangeValue]; err != nil {
		return err
	}
	if len(c.fakeSheetClient.updates) >= c.failAfter {
		return c.err
	}
	return c.fakeSheetClient.UpdateRow(rangeValue, values)
}

func TestGivenNoSyncYetWhenReadingSheetInfoThenLastSyncAtIsNullUntilASyncSucceeds(t *testing.T) {
	client := newFakeSheetClient()
	client.title = "Wealth 2026"
	svc := NewDatabaseService(client, newFakeCache(), logger.NewTestLogger(t), DatabaseServiceOptions{
		SpreadsheetID: "sheet-123",
		Now:           func() time.Time { return fixedNow },
	})

	info, err := svc.SheetInfo()
	if err != nil || info.SpreadsheetID != "sheet-123" || info.Title != "Wealth 2026" || info.LastSyncAt != nil {
		t.Fatalf("info = %+v, %v", info, err)
	}
	if _, err := svc.SyncCache(); err != nil {
		t.Fatal(err)
	}
	info, err = svc.SheetInfo()
	if err != nil || info.LastSyncAt == nil || *info.LastSyncAt != "2026-10-12T09:00:00Z" {
		t.Fatalf("info after sync = %+v, %v", info, err)
	}
}

func TestGivenFailedSyncWhenReadingSheetInfoThenLastSyncAtIsUnchanged(t *testing.T) {
	cache := newFakeCache()
	cache.failing = true
	client := newFakeSheetClient()
	svc := newTestDBService(t, client, cache)
	if _, err := svc.SyncCache(); err == nil {
		t.Fatal("expected the sync to fail")
	}
	client.titleErr = errors.New("invalid_grant")
	if _, err := svc.SheetInfo(); err == nil {
		t.Fatal("a title failure must surface")
	}
	if svc.lastSync() != nil {
		t.Fatal("a failed sync must not set lastSyncAt")
	}
}
