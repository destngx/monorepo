package service

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/internal/fakes"
	"context"
	"errors"
	"testing"
	"time"
)

func expenseBudget(category string, monthly float64) domain.BudgetItem {
	kind := domain.CategoryTypeExpense
	return domain.BudgetItem{Category: category, CategoryType: &kind, MonthlyLimit: monthly}
}

func newAnalytics(db *fakes.Database) *analyticsService {
	return NewAnalyticsService(db, func() time.Time { return fixedNow }).(*analyticsService)
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

func TestGivenFixedCategoriesWhenReadingTheDashboardThenThePaceAddsThemOnce(t *testing.T) {
	kind := domain.CategoryTypeExpense
	rent, food := 6_000_000.0, 1_200_000.0
	db := &fakes.Database{
		Categories: []domain.Category{{Name: "Rent", Type: kind, Nature: domain.NatureFixed}},
		Transactions: []domain.Transaction{
			{AccountName: "Bank", Date: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), Category: "Rent", CategoryType: &kind, Payment: &rent},
			{AccountName: "Bank", Date: time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC), Category: "Food", CategoryType: &kind, Payment: &food},
		},
	}

	summary, err := newAnalytics(db).DashboardSummary(false)
	if err != nil {
		t.Fatal(err)
	}
	// Day 12 of 31: 6M rent once + 1.2M / 12 * 31.
	if summary.ExpensePace == nil || summary.ExpensePace.Projected == nil || *summary.ExpensePace.Projected != 9_100_000 {
		t.Fatalf("pace = %+v", summary.ExpensePace)
	}
}

func TestGivenCategoriesFailWhenReadingTheDashboardThenTheErrorIsReturned(t *testing.T) {
	failure := errors.New("sheet down")
	if _, err := newAnalytics(&fakes.Database{Err: failure}).DashboardSummary(false); !errors.Is(err, failure) {
		t.Fatalf("err = %v", err)
	}
}

func TestGivenNoYearWhenReadingYearlyCashFlowThenTheCurrentYear(t *testing.T) {
	kind := domain.CategoryTypeExpense
	payment := 400.0
	db := &fakes.Database{Transactions: []domain.Transaction{{AccountName: "Bank", Date: time.Date(2026, 2, 3, 0, 0, 0, 0, time.UTC), Category: "Food", CategoryType: &kind, Payment: &payment}}}

	report, err := newAnalytics(db).YearlyCashFlow(0, false)
	if err != nil {
		t.Fatal(err)
	}
	if report.Year != 2026 || report.MonthsElapsed != 10 || report.Months[1].Expense != 400 || report.Total.Expense != 400 {
		t.Fatalf("report = %+v", report)
	}
}

func TestGivenBadYearOrFailureWhenReadingYearlyCashFlowThenErrors(t *testing.T) {
	for _, year := range []int{-1, 1899, 10000} {
		if _, err := newAnalytics(&fakes.Database{}).YearlyCashFlow(year, false); !errors.Is(err, domain.ErrValidation) {
			t.Fatalf("year %d: err = %v", year, err)
		}
	}
	failure := errors.New("sheet down")
	if _, err := newAnalytics(&fakes.Database{Err: failure}).YearlyCashFlow(2025, false); !errors.Is(err, failure) {
		t.Fatalf("err = %v", err)
	}
}

func cardsDatabase() *fakes.Database {
	payment := 1_000_000.0
	return &fakes.Database{
		Cards: domain.CardConfig{
			Programs: []domain.Program{{Name: "Gold", Bank: "Acme", Versions: []domain.ProgramVersion{{
				From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Period: domain.CapPeriodCycle,
				Rules: []domain.CashbackRule{{Name: "Others", Rate: 1, Fallback: true}},
			}}}},
			Banks:     []domain.Bank{{Name: "Acme", SharedAccount: "Credit Card Acme", ReferenceLimit: 10_000_000}},
			Cards:     []domain.Card{{Name: "Gold Card", Program: "Gold", Keyword: "gold", StatementDay: 5, DueDay: 30, Default: true}},
			Merchants: []domain.MerchantMCC{},
		},
		Accounts:     []domain.Account{{Name: "Credit Card Acme", Type: domain.AccountTypeNegativeActive, ClearedBalance: -1_000_000}},
		Transactions: []domain.Transaction{{AccountName: "Credit Card Acme", Date: domain.Today(fixedNow), Category: "Shopping", Payment: &payment}},
	}
}

func TestGivenCardsInTheSheetWhenReadingCardReportsThenTheyAreEvaluated(t *testing.T) {
	reports, err := newAnalytics(cardsDatabase()).CardReports(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 1 || reports[0].Usage != 1_000_000 || reports[0].Cards[0].Standing.Cashback.Total != 10_000 {
		t.Fatalf("reports = %+v", reports)
	}
}

func TestGivenAPaymentWhenSuggestingCardsThenHeldCardsAreRanked(t *testing.T) {
	suggestions, err := newAnalytics(cardsDatabase()).SuggestCards(domain.CardPaymentQuery{Payee: "Shop", Amount: 2_000_000}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestions) != 1 || suggestions[0].Card != "Gold Card" || suggestions[0].Cashback.Total != 20_000 || !suggestions[0].FitsLimit {
		t.Fatalf("suggestions = %+v", suggestions)
	}
}

func TestGivenABadPaymentWhenSuggestingCardsThenValidationFails(t *testing.T) {
	negative := -1
	for name, query := range map[string]domain.CardPaymentQuery{
		"no amount":    {Payee: "Shop"},
		"negative mcc": {Payee: "Shop", Amount: 1, MCC: &negative},
		"no payee":     {Amount: 1},
	} {
		if _, err := newAnalytics(cardsDatabase()).SuggestCards(query, false); !errors.Is(err, domain.ErrValidation) {
			t.Fatalf("%s: err = %v", name, err)
		}
	}
}

func TestGivenCardsInTheSheetWhenComparingProgrammesThenTheYearIsReplayed(t *testing.T) {
	report, err := newAnalytics(cardsDatabase()).ComparePrograms(false)
	if err != nil {
		t.Fatal(err)
	}
	if report.Payments != 1 || len(report.Programs) != 1 || report.Programs[0].Cashback.Total != 10_000 {
		t.Fatalf("report = %+v", report)
	}
}

func TestGivenAnyFailureWhenAdvisingOnCardsThenItIsReturned(t *testing.T) {
	db := &fakes.Database{Err: domain.ErrValidation}
	if _, err := newAnalytics(db).SuggestCards(domain.CardPaymentQuery{Payee: "Shop", Amount: 1}, false); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("suggest err = %v", err)
	}
	if _, err := newAnalytics(db).ComparePrograms(false); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("compare err = %v", err)
	}
}

func TestGivenInvalidCardsWhenReadingCardReportsThenTheErrorIsReturned(t *testing.T) {
	db := &fakes.Database{Err: domain.ErrValidation}
	if _, err := newAnalytics(db).CardReports(false); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("err = %v", err)
	}
}

// staticAllocation is an AllocationSource with a fixed answer.
type staticAllocation struct {
	allocation domain.Allocation
	err        error
	forced     []bool
}

func (s *staticAllocation) Allocation(_ context.Context, forceFresh bool) (domain.Allocation, error) {
	s.forced = append(s.forced, forceFresh)
	return s.allocation, s.err
}

func TestGivenAnAllocationSourceWhenScoringHealthThenDiversificationUsesTheSameAllocation(t *testing.T) {
	db := &fakes.Database{Accounts: []domain.Account{{Name: "Bank", Type: domain.AccountTypeBank, Balance: 30}}}
	source := &staticAllocation{allocation: domain.ComputeAllocation(domain.AllocationInputs{
		Accounts: db.Accounts, Funds: 30, CryptoHoldings: 30,
	})}
	health, err := NewAnalyticsService(db, func() time.Time { return fixedNow }, WithAllocation(source)).FinancialHealth(true)
	if err != nil {
		t.Fatal(err)
	}
	if len(source.forced) != 1 || !source.forced[0] || health.Allocation.TargetedTotal != 90 {
		t.Fatalf("allocation %+v forced %v", health.Allocation, source.forced)
	}
	for _, component := range health.Components {
		if component.Key == "diversification" && component.Score != 20 {
			t.Fatalf("an even split scores full: %+v", component)
		}
	}
	// emergency months still count spendable cash only
	if health.LiquidAssets != 30 {
		t.Fatalf("liquid %v", health.LiquidAssets)
	}
}

func TestGivenAFailingAllocationSourceWhenScoringHealthThenTheErrorIsReturned(t *testing.T) {
	source := &staticAllocation{err: domain.ErrNotFound}
	if _, err := NewAnalyticsService(&fakes.Database{}, nil, WithAllocation(source)).FinancialHealth(false); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestGivenGoalSettingsInPreferencesWhenComputingTheGoalPlanThenThePlanUsesThem(t *testing.T) {
	prefs := domain.ParsePreferences(nil, 2)
	prefs.GoalPool = domain.GoalPoolSettings{EmergencyMonths: 3, SpendingBufferMonths: 2}
	db := &fakes.Database{Preferences: &prefs}

	plan, err := newAnalytics(db).GoalPlan(false)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Settings != prefs.GoalPool || plan.Pool.BufferMonths != 2 {
		t.Fatalf("settings %+v, buffer months %v", plan.Settings, plan.Pool.BufferMonths)
	}
}
