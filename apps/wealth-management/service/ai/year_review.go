package ai

import (
	"apps/wealth-management/domain"
	"fmt"
	"strconv"
)

const (
	yearReviewExpenseCategories = 8
	yearReviewIncomeCategories  = 5
	otherCategory               = "Other"
)

// yearReviewFacts is the yearly cash flow trimmed to what a review needs
// (WM-DASH-8): the long tail of categories folds into "Other".
type yearReviewFacts struct {
	Year           int                     `json:"year"`
	MonthsElapsed  int                     `json:"monthsElapsed"`
	ComparedWith   string                  `json:"comparedWith"`
	Total          domain.CashFlow         `json:"total"`
	MonthlyAverage domain.CashFlow         `json:"monthlyAverage"`
	PreviousYear   domain.CashFlow         `json:"previousYearSameMonths"`
	IncomeChange   *float64                `json:"incomeChangePercent"`
	ExpenseChange  *float64                `json:"expenseChangePercent"`
	Verdict        domain.YearVerdict      `json:"verdict"`
	SavingsTarget  domain.SavingsTarget    `json:"savingsTargetPercent"`
	Projection     *domain.YearProjection  `json:"yearEndProjection,omitempty"`
	Months         []domain.CashFlow       `json:"months"`
	Spikes         []domain.ExpenseSpike   `json:"unusualMonths"`
	Expense        []domain.CategoryMonths `json:"expenseByCategory"`
	Income         []domain.CategoryMonths `json:"incomeByCategory"`
	ExpenseSplit   expenseSplitFacts       `json:"needsVsWants"`
}

// expenseSplitFacts is the needs/wants split as year totals, with the room to save.
type expenseSplitFacts struct {
	Fixed                 float64               `json:"fixed"`
	Essential             float64               `json:"essential"`
	Flexible              float64               `json:"flexible"`
	Untagged              float64               `json:"untagged"`
	NeedsShareOfIncome    *float64              `json:"needsShareOfIncomePercent"`
	NeedsShareComfortable float64               `json:"needsShareComfortablePercent"`
	WantsShareOfIncome    *float64              `json:"wantsShareOfIncomePercent"`
	SavingsShareOfIncome  *float64              `json:"savingsShareOfIncomePercent"`
	SavingsShareTarget    float64               `json:"savingsShareTargetPercent"`
	UntaggedCategories    []string              `json:"untaggedCategories"`
	Room                  []domain.CategoryRoom `json:"roomToSave"`
	RoomPerMonth          float64               `json:"roomToSavePerMonth"`
	RoomPerYear           float64               `json:"roomToSavePerYear"`
}

func (s *Service) yearReviewInputs(year int) (map[string]string, yearReviewFacts, error) {
	report, err := s.analytics.YearlyCashFlow(year, false)
	if err != nil {
		return nil, yearReviewFacts{}, fmt.Errorf("load yearly cash flow: %w", err)
	}
	if report.MonthsElapsed == 0 {
		return nil, yearReviewFacts{}, fmt.Errorf("%w: %d has not started", domain.ErrValidation, report.Year)
	}
	split := report.ExpenseSplit
	facts := yearReviewFacts{
		Year: report.Year, MonthsElapsed: report.MonthsElapsed, ComparedWith: "the same months of " + strconv.Itoa(report.Year-1),
		Total: report.Total, MonthlyAverage: report.MonthlyAverage, PreviousYear: report.PreviousYear,
		IncomeChange: report.IncomeChangePercent, ExpenseChange: report.ExpenseChangePercent,
		Verdict: report.Verdict, SavingsTarget: report.SavingsTarget, Projection: report.Projection,
		Months: report.Months[:report.MonthsElapsed], Spikes: report.Spikes,
		Expense: topCategories(report.Expense, yearReviewExpenseCategories, report.MonthsElapsed),
		Income:  topCategories(report.Income, yearReviewIncomeCategories, report.MonthsElapsed),
		ExpenseSplit: expenseSplitFacts{
			Fixed: split.Fixed.Total, Essential: split.Essential.Total, Flexible: split.Flexible.Total, Untagged: split.Untagged.Total,
			NeedsShareOfIncome: split.NeedsShareOfIncome, NeedsShareComfortable: split.NeedsShareComfortable,
			WantsShareOfIncome: split.WantsShareOfIncome, SavingsShareOfIncome: split.SavingsShareOfIncome, SavingsShareTarget: split.SavingsShareTarget,
			UntaggedCategories: split.UntaggedCategories, Room: split.Room, RoomPerMonth: split.RoomPerMonth, RoomPerYear: split.RoomPerYear,
		},
	}
	vars := map[string]string{
		"year":          strconv.Itoa(report.Year),
		"monthsElapsed": strconv.Itoa(report.MonthsElapsed),
		"verdict":       report.Verdict.Status,
		"savingsRate":   fmt.Sprintf("%.1f", report.Total.SavingsRate),
		"totalIncome":   amount(report.Total.Income),
		"totalExpense":  amount(report.Total.Expense),
		"yearJson":      toJSON(facts),
	}
	return vars, facts, nil
}

// topCategories keeps the largest `limit` rows over the elapsed months and folds the rest into "Other".
func topCategories(rows []domain.CategoryMonths, limit int, months int) []domain.CategoryMonths {
	trimmed := make([]domain.CategoryMonths, 0, min(len(rows), limit+1))
	var other *domain.CategoryMonths
	for i, row := range rows {
		row.Values = row.Values[:months]
		if i < limit {
			trimmed = append(trimmed, row)
			continue
		}
		if other == nil {
			other = &domain.CategoryMonths{Category: otherCategory, Values: make([]float64, months)}
		}
		for month, value := range row.Values {
			other.Values[month] += value
		}
		other.Total += row.Total
	}
	if other != nil {
		trimmed = append(trimmed, *other)
	}
	return trimmed
}
