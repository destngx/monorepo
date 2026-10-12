package domain

import (
	"math"
	"sort"
	"time"
)

const (
	dashboardTrendDays   = 30
	dashboardTopAccounts = 5
)

// DashboardSummary backs the dashboard (WM-DASH-1/2/3/5).
type DashboardSummary struct {
	AsOf                  string       `json:"asOf"`
	BalanceSheet          BalanceSheet `json:"balanceSheet"`
	NetGrowthThisMonth    float64      `json:"netGrowthThisMonth"`
	NetWorthChangePercent *float64     `json:"netWorthChangePercent"`
	NetWorthTrend         []TrendPoint `json:"netWorthTrend"`
	ThisMonth             CashFlow     `json:"thisMonth"`
	LastMonth             CashFlow     `json:"lastMonth"`
	IncomeChangePercent   *float64     `json:"incomeChangePercent"`
	ExpenseChangePercent  *float64     `json:"expenseChangePercent"`
	CashFlowHistory       []CashFlow   `json:"cashFlowHistory"`
	TopAccounts           []Account    `json:"topAccounts"`
}

// ComputeDashboardSummary derives the dashboard from sheet data. Month-over-
// month net worth compares to the start of the month, reconstructed by
// undoing this month's net flow; the trend is back-calculated daily.
func ComputeDashboardSummary(accounts []Account, loans []Loan, transactions []Transaction, today time.Time, historyMonths int) DashboardSummary {
	sheet := ComputeBalanceSheet(accounts, loans)
	month := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	growth := MonthNetFlow(transactions, today)
	thisMonth := ComputeCashFlow(transactions, month)
	lastMonth := ComputeCashFlow(transactions, month.AddDate(0, -1, 0))
	if historyMonths < 1 {
		historyMonths = 1
	}
	return DashboardSummary{
		AsOf:                  today.Format(time.DateOnly),
		BalanceSheet:          sheet,
		NetGrowthThisMonth:    growth,
		NetWorthChangePercent: PercentChange(sheet.NetWorth, sheet.NetWorth-growth),
		NetWorthTrend:         BackCalculateSeries(sheet.NetWorth, transactions, today, dashboardTrendDays),
		ThisMonth:             thisMonth,
		LastMonth:             lastMonth,
		IncomeChangePercent:   PercentChange(thisMonth.Income, lastMonth.Income),
		ExpenseChangePercent:  PercentChange(thisMonth.Expense, lastMonth.Expense),
		CashFlowHistory:       MonthlyCashFlows(transactions, month.AddDate(0, -(historyMonths-1), 0), month),
		TopAccounts:           topAccountsByMagnitude(accounts, dashboardTopAccounts),
	}
}

// topAccountsByMagnitude ranks non-archived accounts by |balance|.
func topAccountsByMagnitude(accounts []Account, limit int) []Account {
	ranked := make([]Account, 0, len(accounts))
	for _, account := range accounts {
		if account.Type != AccountTypeDeprecated {
			ranked = append(ranked, account)
		}
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		return math.Abs(ranked[i].Balance) > math.Abs(ranked[j].Balance)
	})
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	return ranked
}
