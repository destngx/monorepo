package service

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"fmt"
	"time"
)

const (
	dashboardHistoryMonths = 12
	maxHistoryDays         = 366
)

type analyticsService struct {
	db    port.DatabaseService
	cards domain.CardConfig
	now   func() time.Time
}

func NewAnalyticsService(db port.DatabaseService, cards domain.CardConfig, now func() time.Time) port.AnalyticsService {
	if now == nil {
		now = time.Now
	}
	return &analyticsService{db: db, cards: cards, now: now}
}

func (s *analyticsService) today() time.Time {
	return domain.Today(s.now())
}

func (s *analyticsService) DashboardSummary(forceFresh bool) (domain.DashboardSummary, error) {
	accounts, loans, transactions, err := s.position(forceFresh)
	if err != nil {
		return domain.DashboardSummary{}, err
	}
	return domain.ComputeDashboardSummary(accounts, loans, transactions, s.today(), dashboardHistoryMonths), nil
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

func (s *analyticsService) CardReports(forceFresh bool) ([]domain.CardBankReport, error) {
	accounts, err := s.db.ListAccounts(forceFresh)
	if err != nil {
		return nil, err
	}
	transactions, err := s.db.ListTransactions(forceFresh)
	if err != nil {
		return nil, err
	}
	return domain.ComputeCardReports(s.cards, accounts, transactions, s.today()), nil
}

func (s *analyticsService) FinancialHealth(forceFresh bool) (domain.FinancialHealth, error) {
	accounts, loans, transactions, err := s.position(forceFresh)
	if err != nil {
		return domain.FinancialHealth{}, err
	}
	return domain.ComputeFinancialHealth(accounts, loans, transactions, s.today()), nil
}

func (s *analyticsService) AccountHistories(days int, forceFresh bool) (map[string][]domain.TrendPoint, error) {
	if days < 1 || days > maxHistoryDays {
		return nil, fmt.Errorf("%w: days must be between 1 and %d", domain.ErrValidation, maxHistoryDays)
	}
	accounts, err := s.db.ListAccounts(forceFresh)
	if err != nil {
		return nil, err
	}
	transactions, err := s.db.ListTransactions(forceFresh)
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

func (s *analyticsService) position(forceFresh bool) ([]domain.Account, []domain.Loan, []domain.Transaction, error) {
	accounts, err := s.db.ListAccounts(forceFresh)
	if err != nil {
		return nil, nil, nil, err
	}
	loans, err := s.db.ListLoans(forceFresh)
	if err != nil {
		return nil, nil, nil, err
	}
	transactions, err := s.db.ListTransactions(forceFresh)
	if err != nil {
		return nil, nil, nil, err
	}
	return accounts, loans, transactions, nil
}
