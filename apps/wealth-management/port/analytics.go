package port

import (
	"apps/wealth-management/domain"
	"context"
)

// AnalyticsService derives reports from sheet data using the domain rules.
type AnalyticsService interface {
	DashboardSummary(forceFresh bool) (domain.DashboardSummary, error)
	// BudgetReport covers a "YYYY-MM" month; empty means the current month.
	BudgetReport(month string, forceFresh bool) (domain.BudgetReport, error)
	// BudgetHistory compares limit and spend for `months` months ending at
	// month ("YYYY-MM", empty = current), oldest first (WM-BUD-7).
	BudgetHistory(month string, months int, forceFresh bool) (domain.BudgetHistory, error)
	// YearlyCashFlow gives income and expense per month and category for a
	// calendar year (0 = the current year) (WM-DASH-8).
	YearlyCashFlow(year int, forceFresh bool) (domain.YearlyCashFlow, error)
	CardReports(forceFresh bool) ([]domain.CardBankReport, error)
	// CardPeriod lays out one card's period starting in month ("YYYY-MM",
	// empty = current) payment by payment (WM-CARD-17).
	CardPeriod(card string, month string, forceFresh bool) (domain.CardPeriodDetail, error)
	// SuggestCards ranks held cards for a payment by the cashback it would add (WM-CARD-13).
	SuggestCards(query domain.CardPaymentQuery, forceFresh bool) ([]domain.CardSuggestion, error)
	// ComparePrograms replays the last year of card spend under every programme (WM-CARD-14).
	ComparePrograms(forceFresh bool) (domain.ProgramComparisonReport, error)
	FinancialHealth(forceFresh bool) (domain.FinancialHealth, error)
	// GoalPlan fills the Goal Plan goals from the pooled savings balance by
	// priority and projects them from monthly savings (D31, D32).
	GoalPlan(forceFresh bool) (domain.GoalPlan, error)
	// MoneyLent is the per-person ledger of lending transactions (WM-LEND-1, D30).
	MoneyLent(forceFresh bool) (domain.MoneyLent, error)
	// Bills matches the Recurring items against the ledger and returns
	// them with suggestions, the 30-day calendar (card payments included)
	// and safe-to-spend until the next income (WM-BILL-2..6).
	Bills(forceFresh bool) (domain.Bills, error)
	// IdleCash compares ready money beyond the day-to-day float and what
	// must stay liquid with the best instant-access deposit rate, no-term
	// or 1-month (ROADMAP R6, WM-CASH-1..5, D34). A missing
	// or failing rate source is reported in the result, not as an error.
	IdleCash(ctx context.Context, forceFresh bool) (domain.IdleCash, error)
	// AccountHistories back-calculates daily balances per account (WM-ACC-3).
	AccountHistories(days int, forceFresh bool) (map[string][]domain.TrendPoint, error)
}

// BankRateSource gives the published bank savings rates (the one
// bank-rates list, ROADMAP C5).
type BankRateSource interface {
	GetBankInterestRate(ctx context.Context) (domain.BankRates, error)
}
