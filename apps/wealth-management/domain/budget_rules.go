package domain

import (
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
}

// BudgetReport is the month overview (WM-BUD-1, WM-BUD-3). Totals cover expense categories only.
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
	for _, t := range transactions {
		if !t.Usable() {
			continue
		}
		key := normalizeCategory(t.Category)
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
		line.Note = item.Note
		line.YearlyLimit = item.YearlyLimit
		line.YearlySpent = clampZero(yearSpend[key])
		line.YearlyRemaining = clampZero(item.YearlyLimit - line.YearlySpent)
		if line.Limit > 0 || line.Spent > 0 {
			report.Lines = append(report.Lines, line)
		}
	}
	for key, spent := range monthSpend {
		if budgeted[key] || categoryTypes[key] != CategoryTypeExpense || spent <= 0 {
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

	for _, line := range report.Lines {
		if line.CategoryType != CategoryTypeExpense {
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
