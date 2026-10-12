package service

import (
	"apps/wealth-management/domain"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// budgetSheetName resolves the tab for a year from the template (WM-DATA-2).
func (s *dbService) budgetSheetName(year int) string {
	return strings.ReplaceAll(s.budgetTemplate, "{year}", strconv.Itoa(year))
}

func (s *dbService) resolveBudgetYear(year int) int {
	if year <= 0 {
		return s.today().Year()
	}
	return year
}

func budgetCacheKey(year int) string {
	return fmt.Sprintf("budget:%d", year)
}

// ListBudget reads the budget tab of the year (0 = current year).
func (s *dbService) ListBudget(year int, forceFresh bool) ([]domain.BudgetItem, error) {
	year = s.resolveBudgetYear(year)
	return cachedRead(s, budgetCacheKey(year), sheetDataTTL, forceFresh, func() ([]domain.BudgetItem, error) {
		rows, err := s.readBudgetSheet(year)
		if err != nil {
			return nil, err
		}
		items, _ := parseBudget(rows)
		return items, nil
	})
}

// UpdateBudgetLimit writes a monthly limit to the month's column, or the
// yearly limit to column C (WM-BUD-6).
func (s *dbService) UpdateBudgetLimit(input domain.BudgetLimitUpdate) error {
	if err := input.Validate(); err != nil {
		return err
	}
	year := s.resolveBudgetYear(input.Year)
	if input.Month != "" {
		month, _ := domain.ParseMonthKey(input.Month)
		year = month.Year()
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	rows, err := s.readBudgetSheet(year)
	if err != nil {
		return err
	}
	_, layout := parseBudget(rows)
	row, ok := layout.categoryRows[normalizeKey(input.Category)]
	if !ok {
		return fmt.Errorf("%w: category %q is not in %s", domain.ErrNotFound, input.Category, s.budgetSheetName(year))
	}
	column := budgetYearlyColumn
	if input.Month != "" {
		column, ok = layout.monthColumns[input.Month]
		if !ok {
			return fmt.Errorf("%w: month %s has no column in %s", domain.ErrNotFound, input.Month, s.budgetSheetName(year))
		}
	}
	cell := fmt.Sprintf("%s!%s%d", quoteSheetName(s.budgetSheetName(year)), columnLetter(column), row)
	if err := s.client.UpdateRow(cell, []any{input.Limit}); err != nil {
		return fmt.Errorf("update budget limit: %w", err)
	}
	s.invalidate(budgetCacheKey(year))
	return nil
}

// readBudgetSheet reads the year's tab; a missing tab is a clear ErrNotFound.
func (s *dbService) readBudgetSheet(year int) ([][]string, error) {
	name := s.budgetSheetName(year)
	rows, err := s.client.ReadSheet(quoteSheetName(name) + "!" + budgetSheetColumns)
	if err == nil {
		return rows, nil
	}
	titles, titlesErr := s.client.SheetTitles()
	if titlesErr == nil && !slices.Contains(titles, name) {
		return nil, fmt.Errorf("%w: budget tab %q does not exist; create it or set BUDGET_SHEET_TEMPLATE", domain.ErrNotFound, name)
	}
	return nil, fmt.Errorf("read budget tab %q: %w", name, err)
}

func quoteSheetName(name string) string {
	return "'" + strings.ReplaceAll(name, "'", "''") + "'"
}
