package service

import (
	"apps/wealth-management/domain"
	"errors"
	"testing"
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
