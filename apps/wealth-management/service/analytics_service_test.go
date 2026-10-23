package service

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/internal/fakes"
	"context"
	"errors"
	"slices"
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

func TestGivenTheLedgerWhenReadingSpendingWindowsThenFixedCategoriesAndSplitsComeFromTheCategories(t *testing.T) {
	kind := domain.CategoryTypeExpense
	rent, meal := 6_000_000.0, 300_000.0
	db := &fakes.Database{
		Categories: []domain.Category{
			{Name: "Rent", Type: kind, Nature: domain.NatureFixed},
			{Name: "Food", Type: kind, Nature: domain.NatureEssential, SplitTags: []string{"eating-out"}},
		},
		Transactions: []domain.Transaction{
			{AccountName: "Bank", Date: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), Category: "Rent", CategoryType: &kind, Payment: &rent},
			{AccountName: "Bank", Date: time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC), Category: "Food", CategoryType: &kind, Payment: &meal, Tags: []string{"eating-out"}},
		},
	}

	spending, err := newAnalytics(db).SpendingWindows(false)
	if err != nil {
		t.Fatal(err)
	}
	recent := spending.Windows[0]
	if spending.AsOf != "2026-10-12" || recent.Fixed != rent || recent.DayToDay != meal || spending.Today.DayToDay != meal {
		t.Fatalf("spending = %+v", spending)
	}
	if food := recent.Categories[1]; food.Nature != domain.NatureEssential || len(food.Splits) != 1 || food.Splits[0].Spent != meal {
		t.Fatalf("food = %+v", food)
	}
	if _, err := newAnalytics(&fakes.Database{Err: errors.New("sheet down")}).SpendingWindows(false); err == nil {
		t.Fatal("a failing ledger gave no error")
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
		if component.Key == "diversification" && component.Score != 10 {
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

func TestGivenRecurringItemsWhenComputingBillsThenTheSheetLedgerAndPreferencesAreCombined(t *testing.T) {
	amount := 8_000_000.0
	db := &fakes.Database{
		Accounts: []domain.Account{{Name: "Bank", Type: domain.AccountTypeBank, Balance: 20_000_000}},
		Recurring: domain.RecurringSheet{Exists: true, Items: []domain.RecurringItem{
			{ID: "recurring-2", Name: "Rent", Kind: domain.RecurringBill, Amount: &amount, Frequency: domain.FrequencyMonthly, Day: 20, Match: "Landlord", Status: domain.RecurringActive},
		}},
	}
	bills, err := newAnalytics(db).Bills(false)
	if err != nil {
		t.Fatal(err)
	}
	if !bills.TabExists || len(bills.Items) != 1 || bills.SafeToSpend.BillsDue != 8_000_000 || bills.SafeToSpend.SafeToSpend != 12_000_000 || bills.SafeToSpend.ReadyMoney != 20_000_000 {
		t.Fatalf("bills = %+v", bills.SafeToSpend)
	}
	db.Err = errors.New("sheet down")
	if _, err := newAnalytics(db).Bills(false); err == nil {
		t.Fatal("a read failure must surface")
	}
}

func TestGivenAFixedCategoryWithABudgetWhenComputingBillsThenItsCostIsSetAsideAndAMissingYearIsNoLimit(t *testing.T) {
	db := &fakes.Database{
		Accounts:     []domain.Account{{Name: "Bank", Type: domain.AccountTypeBank, Balance: 20_000_000}},
		Categories:   []domain.Category{{Name: "Utilities", Type: domain.CategoryTypeExpense, Nature: domain.NatureFixed}},
		BudgetByYear: map[int][]domain.BudgetItem{2026: {{Category: "Utilities", MonthlyLimit: 1_000_000}}}, // no 2027 tab
	}
	bills, err := newAnalytics(db).Bills(false)
	if err != nil {
		t.Fatal(err)
	}
	// No income: 12 October to 11 November, so October's and November's utilities.
	safe := bills.SafeToSpend
	if !safe.NatureTagged || safe.FixedCosts != 2_000_000 || len(safe.Fixed) != 2 || safe.SafeToSpend != 18_000_000 {
		t.Fatalf("safe = %+v", safe)
	}
}

func TestGivenAccountsMarkedNotReadyInPreferencesWhenComputingBillsThenSafeToSpendLeavesThemOut(t *testing.T) {
	prefs := domain.ParsePreferences([][]string{{"safeToSpend.notReadyAccounts", `["Broker B","Old Broker"]`}}, 2)
	db := &fakes.Database{
		Accounts: []domain.Account{
			{Name: "Bank A", Type: domain.AccountTypeBank, Balance: 20_000_000},
			{Name: "Broker B", Type: domain.AccountTypeActiveUse, Balance: 30_000_000},
		},
		Preferences: &prefs,
	}
	analytics := newAnalytics(db)
	bills, err := analytics.Bills(false)
	if err != nil {
		t.Fatal(err)
	}
	safe := bills.SafeToSpend
	if safe.ReadyMoney != 20_000_000 || len(safe.ExcludedAccounts) != 1 || safe.ExcludedAccounts[0].Reason != domain.ExcludedNotReady {
		t.Fatalf("safe = %+v", safe)
	}
	if len(safe.Issues) != 1 || safe.Readiness.ETag != prefs.SafeToSpend.ETag {
		t.Fatalf("the unknown name is reported and the etag returned: %+v", safe)
	}
	// Idle cash uses the same ready money: the slow account is never idle.
	idle, err := analytics.IdleCash(context.Background(), false)
	if err != nil || idle.Components.ReadyMoney != 20_000_000 || len(idle.Excluded) != 1 || idle.Excluded[0].Name != "Broker B" {
		t.Fatalf("idle = %+v err %v", idle, err)
	}
}

func TestGivenIdleCashPreferencesWhenCheckingThenTheFloatAndEmergencyAccountsApply(t *testing.T) {
	prefs := domain.ParsePreferences([][]string{
		{"idleCash.dayToDayFloat", "5000000"},
		{"idleCash.emergencyAccounts", `["Rainy Day Bank"]`},
	}, 2)
	db := &fakes.Database{
		Accounts: []domain.Account{
			{Name: "Bank A", Type: domain.AccountTypeBank, Balance: 50_000_000},
			{Name: "Rainy Day Bank", Type: domain.AccountTypeBank, Balance: 30_000_000},
		},
		Preferences: &prefs,
	}
	idle, err := NewAnalyticsService(db, func() time.Time { return fixedNow }, WithBankRates(&fakes.Market{})).IdleCash(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	c := idle.Components
	if c.ReadyMoney != 50_000_000 || c.DayToDayFloat != 5_000_000 || !c.FloatFromPreference || c.EmergencyInAccounts != 30_000_000 || idle.Idle != 45_000_000 {
		t.Fatalf("components = %+v idle = %v", c, idle.Idle)
	}
	if len(idle.Emergency.Accounts) != 1 || idle.Emergency.Settings.ETag != prefs.IdleCash.ETag {
		t.Fatalf("emergency = %+v", idle.Emergency)
	}
}

func TestGivenIdleCashWhenCheckingThenBillsGoalPlanAndBankRatesAreCombined(t *testing.T) {
	db := &fakes.Database{Accounts: []domain.Account{{Name: "Bank", Type: domain.AccountTypeBank, Balance: 50_000_000}}}
	market := &fakes.Market{}
	idle, err := NewAnalyticsService(db, func() time.Time { return fixedNow }, WithBankRates(market)).IdleCash(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	// The best instant-access rate is the 1-month 4.5%, never the 12-month 6.5%.
	if idle.Components.ReadyMoney != 50_000_000 || idle.Idle != 50_000_000 || idle.Best == nil || idle.Best.Rate != 4.5 || idle.Best.Access != domain.DepositOneMonth {
		t.Fatalf("idle = %+v", idle)
	}
	if idle.Verdict != domain.IdleCashWorthMoving || idle.ExtraPerYear != 2_250_000 {
		t.Fatalf("verdict = %s, extra = %v", idle.Verdict, idle.ExtraPerYear)
	}

	market.Err = errors.New("rates offline")
	idle, err = NewAnalyticsService(db, func() time.Time { return fixedNow }, WithBankRates(market)).IdleCash(context.Background(), false)
	if err != nil || idle.Verdict != domain.IdleCashNoRate || idle.RatesIssue != "bank rates unavailable: rates offline" {
		t.Fatalf("a rate failure is reported in the result: %v %+v", err, idle)
	}

	idle, err = newAnalytics(db).IdleCash(context.Background(), false)
	if err != nil || idle.Verdict != domain.IdleCashNoRate || idle.RatesIssue == "" {
		t.Fatalf("without a rate source: %v %+v", err, idle)
	}

	db.Err = errors.New("sheet down")
	if _, err := newAnalytics(db).IdleCash(context.Background(), false); err == nil {
		t.Fatal("a read failure must surface")
	}
}

func TestGivenAMinimumBalanceInPreferencesWhenComputingBillsAndIdleCashThenTheReserveIsKeptBack(t *testing.T) {
	prefs := domain.ParsePreferences([][]string{{"accounts.minimumBalance", `{"Bank A":{"minimum":5000000,"note":"points"},"Old Bank":1}`}}, 2)
	db := &fakes.Database{
		Accounts:    []domain.Account{{Name: "Bank A", Type: domain.AccountTypeBank, Balance: 20_000_000}},
		Preferences: &prefs,
	}
	analytics := newAnalytics(db)
	bills, err := analytics.Bills(false)
	if err != nil {
		t.Fatal(err)
	}
	safe := bills.SafeToSpend
	if safe.ReadyMoney != 15_000_000 || safe.Reserved != 5_000_000 || safe.MinimumBalance.ETag != prefs.AccountMinimums.ETag ||
		len(safe.MinimumBalances) != 1 || len(safe.UnknownMinimum) != 1 {
		t.Fatalf("safe = %+v", safe)
	}
	idle, err := analytics.IdleCash(context.Background(), false)
	if err != nil || idle.Components.ReadyMoney != 15_000_000 || idle.Components.MinimumReserved != 5_000_000 {
		t.Fatalf("idle = %+v err %v", idle.Components, err)
	}
}

func TestGivenALinkedWalletWhenCheckingIdleCashThenProductsAreComparedAndFailuresReported(t *testing.T) {
	prefs := domain.ParsePreferences([][]string{{"idleCash.accountProducts", `{"Wallet":"fake:wallet"}`}}, 2)
	db := &fakes.Database{
		Accounts:    []domain.Account{{Name: "Bank", Type: domain.AccountTypeBank, Balance: 50_000_000}, {Name: "Wallet", Type: domain.AccountTypeActiveUse, Balance: 20_000_000}},
		Preferences: &prefs,
	}
	market := &fakes.Market{}
	idle, err := NewAnalyticsService(db, func() time.Time { return fixedNow }, WithBankRates(market)).IdleCash(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	wallet := idle.Accounts[slices.IndexFunc(idle.Accounts, func(a domain.IdleCashAccount) bool { return a.Name == "Wallet" })]
	if wallet.Rate != 4 || wallet.ProductID != "fake:wallet" || idle.ProductsIssue != "" {
		t.Fatalf("wallet = %+v, issue %q", wallet, idle.ProductsIssue)
	}
	if !slices.ContainsFunc(idle.Options, func(o domain.DepositRate) bool { return o.ProductID == "fake:wallet" }) {
		t.Fatalf("options = %+v", idle.Options)
	}
}

func TestGivenNoDataBeforeThisYearWhenReadingAccountHistoriesThenEveryAccountStartsOnFirstJanuary(t *testing.T) {
	deposit := 100.0
	db := &fakes.Database{
		Accounts:     []domain.Account{{Name: "Bank", Balance: 500}, {Name: "Wallet", Balance: 50}},
		Transactions: []domain.Transaction{{AccountName: "Bank", Date: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC), Deposit: &deposit}},
	}

	histories, err := newAnalytics(db).AccountHistories(366, false)
	if err != nil {
		t.Fatal(err)
	}
	for name, points := range histories {
		if len(points) != 285 || points[0].Date != "2026-01-01" || points[len(points)-1].Date != "2026-10-12" {
			t.Fatalf("%s: %d points from %s", name, len(points), points[0].Date)
		}
	}
	if histories["Bank"][0].Value != 400 {
		t.Fatalf("Bank opens at %v, want 400", histories["Bank"][0].Value)
	}
}
