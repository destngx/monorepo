package service

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/internal/fakes"
	"errors"
	"testing"
	"time"
)

func expenseBudget(category string, monthly float64) domain.BudgetItem {
	kind := domain.CategoryTypeExpense
	return domain.BudgetItem{Category: category, CategoryType: &kind, MonthlyLimit: monthly}
}

func newAnalytics(db *fakes.Database) *analyticsService {
	return NewAnalyticsService(db, domain.CardConfig{}, func() time.Time { return fixedNow }).(*analyticsService)
}

func TestGivenMonthsAcrossYearsWhenReadingBudgetHistoryThenEachMonthUsesItsYearTab(t *testing.T) {
	kind := domain.CategoryTypeExpense
	payment := 400.0
	db := &fakes.Database{
		BudgetByYear: map[int][]domain.BudgetItem{2025: {expenseBudget("Food", 100)}, 2026: {expenseBudget("Food", 200)}},
		Transactions: []domain.Transaction{{AccountName: "Bank", Date: time.Date(2025, 12, 3, 0, 0, 0, 0, time.UTC), Category: "Food", CategoryType: &kind, Payment: &payment}},
	}

	history, err := newAnalytics(db).BudgetHistory("2026-01", 2, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(history.Months) != 2 || history.Months[0] != "2025-12" || len(history.Rows) != 1 {
		t.Fatalf("history = %+v", history)
	}
	values := history.Rows[0].Values
	if values[0] != (domain.BudgetHistoryValue{Month: "2025-12", Limit: 100, Spent: 400}) || values[1] != (domain.BudgetHistoryValue{Month: "2026-01", Limit: 200}) {
		t.Fatalf("values = %+v", values)
	}
}

func TestGivenEarlierYearWithoutTabWhenReadingBudgetHistoryThenItHasNoLimits(t *testing.T) {
	db := &fakes.Database{BudgetByYear: map[int][]domain.BudgetItem{2026: {expenseBudget("Food", 200)}}}

	history, err := newAnalytics(db).BudgetHistory("", 12, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(history.Months) != 12 || history.Months[11] != "2026-10" || history.Months[0] != "2025-11" {
		t.Fatalf("months = %v", history.Months)
	}
	if history.Rows[0].Values[0].Limit != 0 || history.Rows[0].Values[11].Limit != 200 {
		t.Fatalf("rows = %+v", history.Rows)
	}
}

func TestGivenTargetYearWithoutTabWhenReadingBudgetHistoryThenNotFound(t *testing.T) {
	db := &fakes.Database{BudgetByYear: map[int][]domain.BudgetItem{}}
	if _, err := newAnalytics(db).BudgetHistory("2026-10", 6, false); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestGivenBadParametersWhenReadingBudgetHistoryThenValidationFails(t *testing.T) {
	for _, tc := range []struct {
		month  string
		months int
	}{{"2026-13", 6}, {"2026-10", 0}, {"2026-10", 25}} {
		if _, err := newAnalytics(&fakes.Database{}).BudgetHistory(tc.month, tc.months, false); !errors.Is(err, domain.ErrValidation) {
			t.Fatalf("%+v: err = %v", tc, err)
		}
	}
}

func TestGivenAnyFailureWhenReadingBudgetHistoryThenItIsReturned(t *testing.T) {
	failure := errors.New("sheet down")
	if _, err := newAnalytics(&fakes.Database{Err: failure}).BudgetHistory("2026-10", 6, false); !errors.Is(err, failure) {
		t.Fatalf("err = %v", err)
	}
}
