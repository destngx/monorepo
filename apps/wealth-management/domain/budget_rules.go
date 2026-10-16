package domain

import (
	"math"
	"sort"
	"strings"
	"time"
)

// BudgetStatus is the WM-RULE-6 colour band (decision D4: 70 / 90).
type BudgetStatus string

const (
	BudgetStatusHealthy BudgetStatus = "healthy"
	BudgetStatusWarning BudgetStatus = "warning"
	BudgetStatusUrgent  BudgetStatus = "urgent"
)

const (
	budgetWarningPercent = 70
	budgetUrgentPercent  = 90
)

// BudgetStatusFor maps percent used to a status band.
func BudgetStatusFor(percentUsed float64) BudgetStatus {
	switch {
	case percentUsed >= budgetUrgentPercent:
		return BudgetStatusUrgent
	case percentUsed >= budgetWarningPercent:
		return BudgetStatusWarning
	default:
		return BudgetStatusHealthy
	}
}

// BudgetLine is one category's actuals for a month (WM-RULE-5).
type BudgetLine struct {
	Category        string       `json:"category"`
	CategoryType    CategoryType `json:"categoryType"`
	Note            string       `json:"note,omitempty"`
	Limit           float64      `json:"limit"`
	Spent           float64      `json:"spent"`
	Remaining       float64      `json:"remaining"`
	PercentUsed     float64      `json:"percentUsed"`
	Status          BudgetStatus `json:"status"`
	YearlyLimit     float64      `json:"yearlyLimit"`
	YearlySpent     float64      `json:"yearlySpent"`
	YearlyRemaining float64      `json:"yearlyRemaining"`
	// Unbudgeted marks expense spending in a category that has no budget row.
	Unbudgeted bool `json:"unbudgeted,omitempty"`
	// Lending marks a lending category (D30): shown, but neither expense nor
	// part of the totals (WM-RULE-3).
	Lending bool `json:"lending,omitempty"`
	// BurnOutDate ("YYYY-MM-DD") is the day the limit runs out at the
	// month's daily burn so far (WM-BUD-7); null when it will not run out
	// this month, the month is not the current one, or there is no limit.
	BurnOutDate *string `json:"burnOutDate"`
}

// CountsAsExpense reports whether the line is expense for totals, burn and
// history: an expense category that is not lending.
func (line BudgetLine) CountsAsExpense() bool {
	return line.CategoryType == CategoryTypeExpense && !line.Lending
}

// BudgetReport is the month overview (WM-BUD-1, WM-BUD-3). Totals cover
// expense categories only, lending categories excluded.
type BudgetReport struct {
	Month          string       `json:"month"`
	DaysInMonth    int          `json:"daysInMonth"`
	DaysElapsed    int          `json:"daysElapsed"`
	DaysRemaining  int          `json:"daysRemaining"`
	TotalLimit     float64      `json:"totalLimit"`
	TotalSpent     float64      `json:"totalSpent"`
	TotalRemaining float64      `json:"totalRemaining"`
	PercentUsed    float64      `json:"percentUsed"`
	DailyBurn      float64      `json:"dailyBurn"`
	ProjectedSpend float64      `json:"projectedSpend"`
	Status         BudgetStatus `json:"status"`
	Lines          []BudgetLine `json:"lines"`
}

// LimitFor returns the month's limit: the explicit monthly cell, else yearly/12.
func (b BudgetItem) LimitFor(month time.Time) float64 {
	if limit, ok := b.MonthlyLimits[MonthKey(month)]; ok {
		return limit
	}
	return b.MonthlyLimit
}

// ComputeBudgetReport applies WM-RULE-5: spent = max(0, Σ(payment - deposit))
// per category over transactions attributed to the month (WM-RULE-4).
func ComputeBudgetReport(items []BudgetItem, transactions []Transaction, month time.Time, today time.Time) BudgetReport {
	monthSpend := map[string]float64{}
	yearSpend := map[string]float64{}
	categoryNames := map[string]string{}
	categoryTypes := map[string]CategoryType{}
	lending := map[string]bool{}
	for _, t := range transactions {
		if !t.Usable() {
			continue
		}
		key := normalizeCategory(t.Category)
		if t.IsLending() {
			lending[key] = true
		}
		effective := EffectiveDate(t)
		net := t.PaymentAmount() - t.DepositAmount()
		if effective.Year() == month.Year() {
			yearSpend[key] += net
		}
		if SameMonth(effective, month) {
			monthSpend[key] += net
			categoryNames[key] = t.Category
			if t.CategoryType != nil {
				categoryTypes[key] = *t.CategoryType
			}
		}
	}

	report := BudgetReport{Month: MonthKey(month), DaysInMonth: DaysInMonth(month), Lines: []BudgetLine{}}
	report.DaysElapsed = daysElapsed(month, today, report.DaysInMonth)
	report.DaysRemaining = report.DaysInMonth - report.DaysElapsed

	budgeted := map[string]bool{}
	for _, item := range items {
		key := normalizeCategory(item.Category)
		budgeted[key] = true
		categoryType := CategoryTypeExpense
		if item.CategoryType != nil {
			categoryType = *item.CategoryType
		}
		line := newBudgetLine(item.Category, categoryType, item.LimitFor(month), monthSpend[key])
		line.Lending = lending[key]
		line.Note = item.Note
		line.YearlyLimit = item.YearlyLimit
		line.YearlySpent = clampZero(yearSpend[key])
		line.YearlyRemaining = clampZero(item.YearlyLimit - line.YearlySpent)
		if line.Limit > 0 || line.Spent > 0 {
			report.Lines = append(report.Lines, line)
		}
	}
	for key, spent := range monthSpend {
		if budgeted[key] || lending[key] || categoryTypes[key] != CategoryTypeExpense || spent <= 0 {
			continue
		}
		line := newBudgetLine(categoryNames[key], CategoryTypeExpense, 0, spent)
		line.YearlySpent = clampZero(yearSpend[key])
		line.Unbudgeted = true
		report.Lines = append(report.Lines, line)
	}
	sort.SliceStable(report.Lines, func(i, j int) bool {
		return report.Lines[i].PercentUsed > report.Lines[j].PercentUsed
	})
	if SameMonth(month, today) {
		for i := range report.Lines {
			report.Lines[i].BurnOutDate = burnOutDate(report.Lines[i], today)
		}
	}

	for _, line := range report.Lines {
		if !line.CountsAsExpense() {
			continue
		}
		report.TotalLimit += line.Limit
		report.TotalSpent += line.Spent
	}
	report.TotalRemaining = clampZero(report.TotalLimit - report.TotalSpent)
	report.PercentUsed = percentOf(report.TotalSpent, report.TotalLimit)
	report.Status = BudgetStatusFor(report.PercentUsed)
	if report.DaysElapsed > 0 {
		report.DailyBurn = round2(report.TotalSpent / float64(report.DaysElapsed))
		report.ProjectedSpend = round2(report.DailyBurn * float64(report.DaysInMonth))
	}
	return report
}

func newBudgetLine(category string, categoryType CategoryType, limit float64, net float64) BudgetLine {
	spent := clampZero(net)
	percent := percentOf(spent, limit)
	return BudgetLine{
		Category:     category,
		CategoryType: categoryType,
		Limit:        limit,
		Spent:        spent,
		Remaining:    clampZero(limit - spent),
		PercentUsed:  percent,
		Status:       BudgetStatusFor(percent),
	}
}

// burnOutDate projects an expense line's daily burn (spent / days so far,
// today included) forward; the limit runs out on the day the cumulative
// spend reaches it. An exhausted limit has run out by today.
func burnOutDate(line BudgetLine, today time.Time) *string {
	if !line.CountsAsExpense() || line.Limit <= 0 || line.Spent <= 0 {
		return nil
	}
	day := today
	if line.Spent < line.Limit {
		dailyBurn := line.Spent / float64(today.Day())
		day = today.AddDate(0, 0, int(math.Ceil((line.Limit-line.Spent)/dailyBurn)))
		if !SameMonth(day, today) {
			return nil
		}
	}
	value := day.Format(time.DateOnly)
	return &value
}

// BudgetHistoryValue is one month of a category in the budget history.
type BudgetHistoryValue struct {
	Month string  `json:"month"`
	Limit float64 `json:"limit"`
	Spent float64 `json:"spent"`
}

// BudgetHistoryRow is one expense category across the history months.
type BudgetHistoryRow struct {
	Category string               `json:"category"`
	Values   []BudgetHistoryValue `json:"values"`
}

// BudgetHistory is the month-by-month comparison ("The Wall", WM-BUD-7).
type BudgetHistory struct {
	Months []string           `json:"months"`
	Rows   []BudgetHistoryRow `json:"rows"`
}

// BudgetHistoryMonths returns `count` month starts ending at end, oldest first.
func BudgetHistoryMonths(end time.Time, count int) []time.Time {
	last := time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, time.UTC)
	months := make([]time.Time, 0, count)
	for i := count - 1; i >= 0; i-- {
		months = append(months, last.AddDate(0, -i, 0))
	}
	return months
}

// ComputeBudgetHistory applies the WM-RULE-5 report to each month and keeps
// expense lines. itemsByYear holds each year's budget tab; a year without one
// contributes spend but no limits. Rows are ordered by total spend.
func ComputeBudgetHistory(itemsByYear map[int][]BudgetItem, transactions []Transaction, months []time.Time, today time.Time) BudgetHistory {
	history := BudgetHistory{Months: make([]string, len(months)), Rows: []BudgetHistoryRow{}}
	rowIndex := map[string]int{}
	totals := map[string]float64{}
	for m, month := range months {
		history.Months[m] = MonthKey(month)
		report := ComputeBudgetReport(itemsByYear[month.Year()], transactions, month, today)
		for _, line := range report.Lines {
			if !line.CountsAsExpense() {
				continue
			}
			key := normalizeCategory(line.Category)
			index, ok := rowIndex[key]
			if !ok {
				index = len(history.Rows)
				rowIndex[key] = index
				values := make([]BudgetHistoryValue, len(months))
				for i, other := range months {
					values[i] = BudgetHistoryValue{Month: MonthKey(other)}
				}
				history.Rows = append(history.Rows, BudgetHistoryRow{Category: line.Category, Values: values})
			}
			history.Rows[index].Values[m].Limit = line.Limit
			history.Rows[index].Values[m].Spent = line.Spent
			totals[key] += line.Spent
		}
	}
	sort.SliceStable(history.Rows, func(i, j int) bool {
		return totals[normalizeCategory(history.Rows[i].Category)] > totals[normalizeCategory(history.Rows[j].Category)]
	})
	return history
}

// daysElapsed counts days of the month that have started: all of a past month,
// none of a future month, today's day number for the current month.
func daysElapsed(month time.Time, today time.Time, daysInMonth int) int {
	switch {
	case SameMonth(month, today):
		return today.Day()
	case month.Before(today):
		return daysInMonth
	default:
		return 0
	}
}

func percentOf(value float64, total float64) float64 {
	if total <= 0 {
		if value > 0 {
			return 100
		}
		return 0
	}
	return round2(value / total * 100)
}

func clampZero(value float64) float64 {
	if value < 0 {
		return 0
	}
	return value
}

func normalizeCategory(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
