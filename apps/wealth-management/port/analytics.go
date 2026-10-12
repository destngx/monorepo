package port

import "apps/wealth-management/domain"

// AnalyticsService derives reports from sheet data using the domain rules.
type AnalyticsService interface {
	DashboardSummary(forceFresh bool) (domain.DashboardSummary, error)
	// BudgetReport covers a "YYYY-MM" month; empty means the current month.
	BudgetReport(month string, forceFresh bool) (domain.BudgetReport, error)
	CardReports(forceFresh bool) ([]domain.CardBankReport, error)
	FinancialHealth(forceFresh bool) (domain.FinancialHealth, error)
	// AccountHistories back-calculates daily balances per account (WM-ACC-3).
	AccountHistories(days int, forceFresh bool) (map[string][]domain.TrendPoint, error)
}
