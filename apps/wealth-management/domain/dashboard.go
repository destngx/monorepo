package domain

import (
	"math"
	"sort"
	"strings"
	"time"
)

const (
	dashboardTrendDays   = 30
	dashboardTopAccounts = 5
)

// DashboardSummary backs the dashboard (WM-DASH-1/2/3/5/9).
type DashboardSummary struct {
	AsOf                  string       `json:"asOf"`
	BalanceSheet          BalanceSheet `json:"balanceSheet"`
	NetGrowthThisMonth    float64      `json:"netGrowthThisMonth"`
	NetWorthChangePercent *float64     `json:"netWorthChangePercent"`
	NetWorthTrend         []TrendPoint `json:"netWorthTrend"`
	// ThisMonth is month to date: day 1 to today, rows dated ahead left out.
	ThisMonth CashFlow `json:"thisMonth"`
	// LastMonth is the whole of last month.
	LastMonth CashFlow `json:"lastMonth"`
	// LastMonthToDate is last month's first ComparedDays days, the fair
	// comparison for ThisMonth.
	LastMonthToDate CashFlow `json:"lastMonthToDate"`
	// ComparedDays is today's day of the month, capped at last month's length.
	ComparedDays int `json:"comparedDays"`
	// IncomeChangePercent and ExpenseChangePercent compare ThisMonth with LastMonthToDate.
	IncomeChangePercent  *float64 `json:"incomeChangePercent"`
	ExpenseChangePercent *float64 `json:"expenseChangePercent"`
	// ExpensePace is the month-end expense at the current pace; set by the
	// service, which reads the category natures (ComputeExpensePace).
	ExpensePace     *ExpensePace `json:"expensePace"`
	CashFlowHistory []CashFlow   `json:"cashFlowHistory"`
	TopAccounts     []Account    `json:"topAccounts"`
}

// ComputeDashboardSummary derives the dashboard from sheet data. Month-over-
// month net worth compares to the start of the month, reconstructed by
// undoing this month's net flow; the trend is back-calculated daily. Income
// and expense compare month to date with the same days of last month, so
// mid-month figures are not set against a whole month.
func ComputeDashboardSummary(accounts []Account, loans []Loan, transactions []Transaction, today time.Time, historyMonths int) DashboardSummary {
	sheet := ComputeBalanceSheet(accounts, loans)
	month := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	growth := MonthNetFlow(transactions, today)
	previous := month.AddDate(0, -1, 0)
	thisMonth := CashFlowToDay(transactions, month, today.Day())
	comparedDays := min(today.Day(), DaysInMonth(previous))
	lastMonthToDate := CashFlowToDay(transactions, previous, comparedDays)
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
		LastMonth:             ComputeCashFlow(transactions, previous),
		LastMonthToDate:       lastMonthToDate,
		ComparedDays:          comparedDays,
		IncomeChangePercent:   PercentChange(thisMonth.Income, lastMonthToDate.Income),
		ExpenseChangePercent:  PercentChange(thisMonth.Expense, lastMonthToDate.Expense),
		CashFlowHistory:       MonthlyCashFlows(transactions, month.AddDate(0, -(historyMonths-1), 0), month),
		TopAccounts:           topAccountsByMagnitude(accounts, dashboardTopAccounts),
	}
}

// CashFlowToDay is ComputeCashFlow for the month's first `day` days (capped
// at the month's length), attributing each row by its effective date
// (WM-RULE-4) the same way, so a refund for the month counts from day 1.
func CashFlowToDay(transactions []Transaction, month time.Time, day int) CashFlow {
	month = time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := month.AddDate(0, 0, min(day, DaysInMonth(month)))
	inWindow := make([]Transaction, 0, len(transactions))
	for _, t := range transactions {
		if effective := EffectiveDate(t); SameMonth(effective, month) && effective.Before(end) {
			inWindow = append(inWindow, t)
		}
	}
	return ComputeCashFlow(inWindow, month)
}

// ExpensePaceMinDays is the first day of the month with a pace: before it a
// few days of spending say little about the month.
const ExpensePaceMinDays = 7

// ExpensePace projects this month's expense to the last day (WM-DASH-9).
// Fixed categories (rent, insurance, subscriptions) are paid once, so they
// are added as recorded; only the rest is extrapolated linearly over the
// days elapsed. Untagged categories count as the rest. Rows dated ahead are
// left out.
type ExpensePace struct {
	DaysElapsed int    `json:"daysElapsed"`
	DaysInMonth int    `json:"daysInMonth"`
	MonthEnd    string `json:"monthEnd"`
	MinDays     int    `json:"minDays"`
	// Spent is month-to-date expense: Fixed plus Variable.
	Spent    float64 `json:"spent"`
	Fixed    float64 `json:"fixed"`
	Variable float64 `json:"variable"`
	// Projected is Fixed + Variable / DaysElapsed * DaysInMonth; nil before MinDays.
	Projected        *float64 `json:"projected"`
	LastMonth        string   `json:"lastMonth"`
	LastMonthExpense float64  `json:"lastMonthExpense"`
	// ChangePercent compares Projected with LastMonthExpense; nil without either.
	ChangePercent *float64 `json:"changePercent"`
}

// ComputeExpensePace is the month-end expense pace for today's month.
func ComputeExpensePace(transactions []Transaction, categories []Category, today time.Time) ExpensePace {
	month := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	previous := month.AddDate(0, -1, 0)
	days := DaysInMonth(month)
	fixedNames := map[string]bool{}
	for _, category := range categories {
		if category.Nature == NatureFixed {
			fixedNames[categoryKey(category.Name)] = true
		}
	}
	fixedRows := []Transaction{}
	for _, t := range transactions {
		if fixedNames[categoryKey(t.Category)] {
			fixedRows = append(fixedRows, t)
		}
	}
	spent := CashFlowToDay(transactions, month, today.Day()).Expense
	fixed := CashFlowToDay(fixedRows, month, today.Day()).Expense
	pace := ExpensePace{
		DaysElapsed:      today.Day(),
		DaysInMonth:      days,
		MonthEnd:         month.AddDate(0, 0, days-1).Format(time.DateOnly),
		MinDays:          ExpensePaceMinDays,
		Spent:            spent,
		Fixed:            fixed,
		Variable:         spent - fixed,
		LastMonth:        MonthKey(previous),
		LastMonthExpense: ComputeCashFlow(transactions, previous).Expense,
	}
	if today.Day() >= ExpensePaceMinDays {
		projected := math.Round(fixed + pace.Variable/float64(today.Day())*float64(days))
		pace.Projected = &projected
		pace.ChangePercent = PercentChange(projected, pace.LastMonthExpense)
	}
	return pace
}

// categoryKey matches category names ignoring case and surrounding spaces.
func categoryKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
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
