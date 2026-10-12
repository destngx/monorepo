package domain

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Canonical business rules (REQUIREMENTS.md §12). The UI, AI and MCP all read
// these results so every surface agrees on the numbers.

// PaymentAmount is the money that left the account (0 when empty).
func (t Transaction) PaymentAmount() float64 {
	if t.Payment == nil {
		return 0
	}
	return *t.Payment
}

// DepositAmount is the money that entered the account (0 when empty).
func (t Transaction) DepositAmount() float64 {
	if t.Deposit == nil {
		return 0
	}
	return *t.Deposit
}

// Usable reports whether the row is safe to feed into calculations.
func (t Transaction) Usable() bool {
	return len(t.Issues) == 0 && !t.Date.IsZero()
}

// HasCategoryType reports whether the transaction category resolved to the given type.
func (t Transaction) HasCategoryType(categoryType CategoryType) bool {
	return t.CategoryType != nil && *t.CategoryType == categoryType
}

// IsRefundCategory matches the sheet's "Refunds/Reimbursements" category, ignoring case and spaces.
func IsRefundCategory(category string) bool {
	return strings.ReplaceAll(strings.ToLower(category), " ", "") == "refunds/reimbursements"
}

var (
	memoMonthDayYear = regexp.MustCompile(`(0?[1-9]|1[0-2])/(0?[1-9]|[12]\d|3[01])/(\d{4}|\d{2})`)
	memoMonthYear    = regexp.MustCompile(`(0?[1-9]|1[0-2])[-/](20\d{2})`)
	memoYearMonth    = regexp.MustCompile(`(20\d{2})[-/](0?[1-9]|1[0-2])`)
)

// EffectiveDate implements WM-RULE-4: a refund deposit is attributed to the
// month named in its memo; every other transaction uses its own date.
func EffectiveDate(t Transaction) time.Time {
	if t.DepositAmount() <= 0 || !IsRefundCategory(t.Category) || t.Memo == nil {
		return t.Date
	}
	memo := strings.TrimSpace(strings.NewReplacer(`'`, "", `"`, "").Replace(*t.Memo))
	if serial, err := strconv.ParseFloat(memo, 64); err == nil && serial > 40000 && serial < 60000 {
		return SerialToDate(serial)
	}
	if match := memoMonthDayYear.FindStringSubmatch(memo); match != nil {
		year := match[3]
		if len(year) == 2 {
			year = "20" + year
		}
		return firstOfMonth(year, match[1])
	}
	if match := memoMonthYear.FindStringSubmatch(memo); match != nil {
		return firstOfMonth(match[2], match[1])
	}
	if match := memoYearMonth.FindStringSubmatch(memo); match != nil {
		return firstOfMonth(match[1], match[2])
	}
	return t.Date
}

func firstOfMonth(year string, month string) time.Time {
	y, _ := strconv.Atoi(year)
	m, _ := strconv.Atoi(month)
	return time.Date(y, time.Month(m), 1, 0, 0, 0, 0, time.UTC)
}

// BalanceSheet is the WM-RULE-1/2 position at a point in time.
type BalanceSheet struct {
	Assets         float64 `json:"assets"`
	Liabilities    float64 `json:"liabilities"`
	AccountsNet    float64 `json:"accountsNet"`
	LoansRemaining float64 `json:"loansRemaining"`
	NetWorth       float64 `json:"netWorth"`
}

// ComputeBalanceSheet: assets are positive balances; liabilities are negative
// balances plus loans still owed this year; net worth = assets - liabilities.
func ComputeBalanceSheet(accounts []Account, loans []Loan) BalanceSheet {
	var sheet BalanceSheet
	for _, account := range accounts {
		if account.Balance >= 0 {
			sheet.Assets += account.Balance
		} else {
			sheet.Liabilities += -account.Balance
		}
		sheet.AccountsNet += account.Balance
	}
	for _, loan := range loans {
		sheet.LoansRemaining += loan.YearlyRemaining
	}
	sheet.Liabilities += sheet.LoansRemaining
	sheet.NetWorth = sheet.AccountsNet - sheet.LoansRemaining
	return sheet
}

// CashFlow is the WM-RULE-3 income statement for one month.
type CashFlow struct {
	Month       string  `json:"month"`
	Income      float64 `json:"income"`
	Expense     float64 `json:"expense"`
	CashFlow    float64 `json:"cashFlow"`
	SavingsRate float64 `json:"savingsRate"`
}

// ComputeCashFlow sums income deposits and expense payments attributed to the
// month (WM-RULE-4). Non-budget categories such as transfers are excluded.
func ComputeCashFlow(transactions []Transaction, month time.Time) CashFlow {
	flow := CashFlow{Month: MonthKey(month)}
	for _, t := range transactions {
		if !t.Usable() || !SameMonth(EffectiveDate(t), month) {
			continue
		}
		switch {
		case t.HasCategoryType(CategoryTypeIncome):
			flow.Income += t.DepositAmount()
		case t.HasCategoryType(CategoryTypeExpense):
			flow.Expense += t.PaymentAmount()
		}
	}
	flow.CashFlow = flow.Income - flow.Expense
	if flow.Income > 0 {
		flow.SavingsRate = round2(flow.CashFlow / flow.Income * 100)
	}
	return flow
}

// MonthlyCashFlows returns one CashFlow per month from first to last, inclusive.
func MonthlyCashFlows(transactions []Transaction, first time.Time, last time.Time) []CashFlow {
	flows := []CashFlow{}
	start := time.Date(first.Year(), first.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(last.Year(), last.Month(), 1, 0, 0, 0, 0, time.UTC)
	for month := start; !month.After(end); month = month.AddDate(0, 1, 0) {
		flows = append(flows, ComputeCashFlow(transactions, month))
	}
	return flows
}

// AverageMonthlyExpense averages expense over the `months` full months before today.
func AverageMonthlyExpense(transactions []Transaction, today time.Time, months int) float64 {
	if months <= 0 {
		return 0
	}
	currentMonth := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	total := 0.0
	for i := 1; i <= months; i++ {
		total += ComputeCashFlow(transactions, currentMonth.AddDate(0, -i, 0)).Expense
	}
	return total / float64(months)
}

// TrendPoint is one day of a back-calculated balance series.
type TrendPoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

// BackCalculateSeries rebuilds end-of-day values for the last `days` days by
// walking backwards from the current value and undoing each day's net flow.
func BackCalculateSeries(current float64, transactions []Transaction, today time.Time, days int) []TrendPoint {
	if days <= 0 {
		return []TrendPoint{}
	}
	start := today.AddDate(0, 0, -(days - 1))
	flowByDay := map[time.Time]float64{}
	for _, t := range transactions {
		if !t.Usable() || t.Date.Before(start) || t.Date.After(today) {
			continue
		}
		flowByDay[t.Date] += t.DepositAmount() - t.PaymentAmount()
	}
	points := make([]TrendPoint, days)
	value := current
	for i := days - 1; i >= 0; i-- {
		day := start.AddDate(0, 0, i)
		points[i] = TrendPoint{Date: day.Format(time.DateOnly), Value: value}
		value -= flowByDay[day]
	}
	return points
}

// MonthNetFlow is total deposits minus total payments dated in the month.
func MonthNetFlow(transactions []Transaction, month time.Time) float64 {
	flow := 0.0
	for _, t := range transactions {
		if t.Usable() && SameMonth(t.Date, month) {
			flow += t.DepositAmount() - t.PaymentAmount()
		}
	}
	return flow
}

// PercentChange returns (current-previous)/|previous|*100, or nil when previous is 0.
func PercentChange(current float64, previous float64) *float64 {
	if previous == 0 {
		return nil
	}
	value := round2((current - previous) / math.Abs(previous) * 100)
	return &value
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
