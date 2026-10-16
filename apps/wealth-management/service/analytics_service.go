package service

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	dashboardHistoryMonths = 12
	maxHistoryDays         = 366
	maxBudgetHistoryMonths = 24
	minCashFlowYear        = 1900
	maxCashFlowYear        = 9999
)

type analyticsService struct {
	db         port.DatabaseService
	now        func() time.Time
	allocation port.AllocationSource
	rates      port.BankRateSource
}

// AnalyticsOption tunes the analytics service.
type AnalyticsOption func(*analyticsService)

// WithAllocation scores health diversification on the engine allocation
// (ROADMAP C4). Without it the allocation is built from the accounts and
// transactions alone, with no fund holdings.
func WithAllocation(source port.AllocationSource) AnalyticsOption {
	return func(s *analyticsService) { s.allocation = source }
}

// WithBankRates gives the idle cash check its comparison rate. Without it
// the check reports that no rate is available.
func WithBankRates(source port.BankRateSource) AnalyticsOption {
	return func(s *analyticsService) { s.rates = source }
}

func NewAnalyticsService(db port.DatabaseService, now func() time.Time, options ...AnalyticsOption) port.AnalyticsService {
	if now == nil {
		now = time.Now
	}
	service := &analyticsService{db: db, now: now}
	for _, option := range options {
		option(service)
	}
	return service
}

func (s *analyticsService) today() time.Time {
	return domain.Today(s.now())
}

func (s *analyticsService) DashboardSummary(forceFresh bool) (domain.DashboardSummary, error) {
	accounts, transactions, err := s.position(forceFresh)
	if err != nil {
		return domain.DashboardSummary{}, err
	}
	categories, err := s.db.ListCategories(forceFresh)
	if err != nil {
		return domain.DashboardSummary{}, err
	}
	today := s.today()
	summary := domain.ComputeDashboardSummary(accounts, transactions, today, dashboardHistoryMonths)
	pace := domain.ComputeExpensePace(transactions, categories, today)
	summary.ExpensePace = &pace
	return summary, nil
}

func (s *analyticsService) BudgetReport(month string, forceFresh bool) (domain.BudgetReport, error) {
	today := s.today()
	target := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	if month != "" {
		parsed, err := domain.ParseMonthKey(month)
		if err != nil {
			return domain.BudgetReport{}, fmt.Errorf("%w: month must be YYYY-MM", domain.ErrValidation)
		}
		target = parsed
	}
	items, err := s.db.ListBudget(target.Year(), forceFresh)
	if err != nil {
		return domain.BudgetReport{}, err
	}
	transactions, err := s.db.ListTransactions(forceFresh)
	if err != nil {
		return domain.BudgetReport{}, err
	}
	return domain.ComputeBudgetReport(items, transactions, target, today), nil
}

// BudgetHistory returns limit and spending per expense category for the
// `months` months ending at `month` (default: the current month). Each month
// reads its own year's tab; an earlier year without a tab has no limits,
// while a missing tab for the target year is NOT_FOUND.
func (s *analyticsService) BudgetHistory(month string, months int, forceFresh bool) (domain.BudgetHistory, error) {
	if months < 1 || months > maxBudgetHistoryMonths {
		return domain.BudgetHistory{}, fmt.Errorf("%w: months must be between 1 and %d", domain.ErrValidation, maxBudgetHistoryMonths)
	}
	today := s.today()
	end := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	if month != "" {
		parsed, err := domain.ParseMonthKey(month)
		if err != nil {
			return domain.BudgetHistory{}, fmt.Errorf("%w: month must be YYYY-MM", domain.ErrValidation)
		}
		end = parsed
	}
	window := domain.BudgetHistoryMonths(end, months)
	itemsByYear := map[int][]domain.BudgetItem{}
	for _, monthStart := range window {
		year := monthStart.Year()
		if _, seen := itemsByYear[year]; seen {
			continue
		}
		items, err := s.db.ListBudget(year, forceFresh)
		if err != nil {
			if year != end.Year() && errors.Is(err, domain.ErrNotFound) {
				itemsByYear[year] = nil
				continue
			}
			return domain.BudgetHistory{}, err
		}
		itemsByYear[year] = items
	}
	transactions, err := s.db.ListTransactions(forceFresh)
	if err != nil {
		return domain.BudgetHistory{}, err
	}
	return domain.ComputeBudgetHistory(itemsByYear, transactions, window, today), nil
}

func (s *analyticsService) YearlyCashFlow(year int, forceFresh bool) (domain.YearlyCashFlow, error) {
	today := s.today()
	if year == 0 {
		year = today.Year()
	}
	if year < minCashFlowYear || year > maxCashFlowYear {
		return domain.YearlyCashFlow{}, fmt.Errorf("%w: year must be between %d and %d", domain.ErrValidation, minCashFlowYear, maxCashFlowYear)
	}
	transactions, err := s.db.ListTransactions(forceFresh)
	if err != nil {
		return domain.YearlyCashFlow{}, err
	}
	categories, err := s.db.ListCategories(forceFresh)
	if err != nil {
		return domain.YearlyCashFlow{}, err
	}
	return domain.ComputeYearlyCashFlow(transactions, categories, year, today), nil
}

func (s *analyticsService) CardReports(forceFresh bool) ([]domain.CardBankReport, error) {
	cards, err := s.db.ListCardConfig(forceFresh)
	if err != nil {
		return nil, err
	}
	accounts, transactions, err := s.position(forceFresh)
	if err != nil {
		return nil, err
	}
	return domain.ComputeCardReports(cards, accounts, transactions, s.today()), nil
}

func (s *analyticsService) CardPeriod(card string, month string, forceFresh bool) (domain.CardPeriodDetail, error) {
	cards, err := s.db.ListCardConfig(forceFresh)
	if err != nil {
		return domain.CardPeriodDetail{}, err
	}
	accounts, transactions, err := s.position(forceFresh)
	if err != nil {
		return domain.CardPeriodDetail{}, err
	}
	return domain.ComputeCardPeriod(cards, accounts, transactions, card, month, s.today())
}

func (s *analyticsService) SuggestCards(query domain.CardPaymentQuery, forceFresh bool) ([]domain.CardSuggestion, error) {
	if err := query.Validate(); err != nil {
		return nil, err
	}
	cards, err := s.db.ListCardConfig(forceFresh)
	if err != nil {
		return nil, err
	}
	accounts, transactions, err := s.position(forceFresh)
	if err != nil {
		return nil, err
	}
	return domain.SuggestCards(cards, accounts, transactions, query, s.today()), nil
}

func (s *analyticsService) ComparePrograms(forceFresh bool) (domain.ProgramComparisonReport, error) {
	cards, err := s.db.ListCardConfig(forceFresh)
	if err != nil {
		return domain.ProgramComparisonReport{}, err
	}
	accounts, transactions, err := s.position(forceFresh)
	if err != nil {
		return domain.ProgramComparisonReport{}, err
	}
	return domain.ComparePrograms(cards, accounts, transactions, s.today()), nil
}

func (s *analyticsService) FinancialHealth(forceFresh bool) (domain.FinancialHealth, error) {
	accounts, transactions, err := s.position(forceFresh)
	if err != nil {
		return domain.FinancialHealth{}, err
	}
	allocation, err := s.allocationFor(accounts, transactions, forceFresh)
	if err != nil {
		return domain.FinancialHealth{}, err
	}
	return domain.ComputeFinancialHealth(accounts, transactions, allocation, s.today()), nil
}

// allocationFor returns the engine allocation; health has no request
// context, so the source runs in the background context.
func (s *analyticsService) allocationFor(accounts []domain.Account, transactions []domain.Transaction, forceFresh bool) (domain.Allocation, error) {
	if s.allocation == nil {
		return domain.ComputeAllocation(domain.AllocationInputs{Accounts: accounts, Transactions: transactions, Now: s.now()}), nil
	}
	allocation, err := s.allocation.Allocation(context.Background(), forceFresh)
	if err != nil {
		return domain.Allocation{}, fmt.Errorf("allocation: %w", err)
	}
	return allocation, nil
}

func (s *analyticsService) MoneyLent(forceFresh bool) (domain.MoneyLent, error) {
	transactions, err := s.db.ListTransactions(forceFresh)
	if err != nil {
		return domain.MoneyLent{}, err
	}
	return domain.ComputeMoneyLent(transactions, s.today()), nil
}

func (s *analyticsService) AccountHistories(days int, forceFresh bool) (map[string][]domain.TrendPoint, error) {
	if days < 1 || days > maxHistoryDays {
		return nil, fmt.Errorf("%w: days must be between 1 and %d", domain.ErrValidation, maxHistoryDays)
	}
	accounts, transactions, err := s.position(forceFresh)
	if err != nil {
		return nil, err
	}
	byAccount := map[string][]domain.Transaction{}
	for _, transaction := range transactions {
		byAccount[transaction.AccountName] = append(byAccount[transaction.AccountName], transaction)
	}
	today := s.today()
	histories := make(map[string][]domain.TrendPoint, len(accounts))
	for _, account := range accounts {
		histories[account.Name] = domain.BackCalculateSeries(account.Balance, byAccount[account.Name], today, days)
	}
	return histories, nil
}

func (s *analyticsService) position(forceFresh bool) ([]domain.Account, []domain.Transaction, error) {
	ledger, err := s.db.ReadLedger(forceFresh)
	if err != nil {
		return nil, nil, err
	}
	return ledger.Accounts, ledger.Transactions, nil
}

// GoalPlan reads the Goal Plan tab, the ledger and the categories and
// computes the plan (D31, D32). Before setup it still returns the pool, so
// the page can show it next to the setup action.
func (s *analyticsService) GoalPlan(forceFresh bool) (domain.GoalPlan, error) {
	sheet, err := s.db.ReadGoalPlan(forceFresh)
	if err != nil {
		return domain.GoalPlan{}, err
	}
	accounts, transactions, err := s.position(forceFresh)
	if err != nil {
		return domain.GoalPlan{}, err
	}
	categories, err := s.db.ListCategories(forceFresh)
	if err != nil {
		return domain.GoalPlan{}, err
	}
	// Goal pool settings come from the Preferences tab (WM-SET-6..9);
	// a missing tab or key reads as the defaults.
	prefs, err := s.db.ListPreferences(forceFresh)
	if err != nil {
		return domain.GoalPlan{}, err
	}
	return domain.ComputeGoalPlan(sheet, accounts, transactions, categories, prefs.GoalPool, s.today()), nil
}

// Bills reads the Recurring tab, the ledger, the card settings and the goal
// plan and lays out the bills (WM-BILL-2..6). Before the tab exists it
// still suggests items and shows the card payments and safe-to-spend.
func (s *analyticsService) Bills(forceFresh bool) (domain.Bills, error) {
	plan, err := s.GoalPlan(forceFresh)
	if err != nil {
		return domain.Bills{}, fmt.Errorf("goal plan: %w", err)
	}
	return s.billsFor(plan, forceFresh)
}

// billsFor lays out the bills with an already computed goal plan.
func (s *analyticsService) billsFor(plan domain.GoalPlan, forceFresh bool) (domain.Bills, error) {
	sheet, err := s.db.ReadRecurring(forceFresh)
	if err != nil {
		return domain.Bills{}, err
	}
	accounts, transactions, err := s.position(forceFresh)
	if err != nil {
		return domain.Bills{}, err
	}
	cards, err := s.db.ListCardConfig(forceFresh)
	if err != nil {
		return domain.Bills{}, err
	}
	return domain.ComputeBills(domain.BillsInput{
		Sheet: sheet, Accounts: accounts, Transactions: transactions, Cards: cards, Goals: &plan, Today: s.today(),
	}), nil
}

// IdleCash combines the bills (unpaid in the next 30 days, goal set-aside),
// the goal plan (spending buffer, emergency fund) and the published bank
// rates (WM-CASH-1..5). Sheet read failures are errors; a missing or
// failing rate source is reported in the result, so the idle amount still
// shows.
func (s *analyticsService) IdleCash(ctx context.Context, forceFresh bool) (domain.IdleCash, error) {
	plan, err := s.GoalPlan(forceFresh)
	if err != nil {
		return domain.IdleCash{}, fmt.Errorf("goal plan: %w", err)
	}
	bills, err := s.billsFor(plan, forceFresh)
	if err != nil {
		return domain.IdleCash{}, fmt.Errorf("bills: %w", err)
	}
	accounts, _, err := s.position(forceFresh)
	if err != nil {
		return domain.IdleCash{}, err
	}
	input := domain.IdleCashInput{Accounts: accounts, Bills: bills, Plan: &plan, Today: s.today()}
	if s.rates == nil {
		input.RatesIssue = "bank rates unavailable: the market data server is not configured"
	} else if rates, err := s.rates.GetBankInterestRate(ctx); err != nil {
		input.RatesIssue = "bank rates unavailable: " + err.Error()
	} else {
		input.Rates = rates.Rates
	}
	return domain.ComputeIdleCash(input), nil
}
