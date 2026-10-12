package service

import (
	"apps/wealth-management/domain"
	"errors"
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

// ReallocateBudget moves part of one category's monthly limit to another
// (WM-BUD-7). It holds the write lock across both cell writes and reads the
// limits from the sheet, not the cache, so concurrent edits are not lost.
// When the second write fails it restores the first cell and reports both
// outcomes.
func (s *dbService) ReallocateBudget(input domain.BudgetReallocation) (domain.BudgetReallocationResult, error) {
	if err := input.Validate(); err != nil {
		return domain.BudgetReallocationResult{}, err
	}
	month, _ := domain.ParseMonthKey(input.Month)
	year := month.Year()
	sheet := s.budgetSheetName(year)

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	rows, err := s.readBudgetSheet(year)
	if err != nil {
		return domain.BudgetReallocationResult{}, err
	}
	items, layout := parseBudget(rows)
	column, ok := layout.monthColumns[input.Month]
	if !ok {
		return domain.BudgetReallocationResult{}, fmt.Errorf("%w: month %s has no column in %s", domain.ErrNotFound, input.Month, sheet)
	}
	from, err := budgetCell(items, layout, input.From, sheet)
	if err != nil {
		return domain.BudgetReallocationResult{}, err
	}
	to, err := budgetCell(items, layout, input.To, sheet)
	if err != nil {
		return domain.BudgetReallocationResult{}, err
	}
	fromLimit, toLimit := from.item.LimitFor(month), to.item.LimitFor(month)
	if fromLimit < input.Amount {
		return domain.BudgetReallocationResult{}, fmt.Errorf("%w: %s has only %.0f left to move in %s", domain.ErrValidation, from.item.Category, fromLimit, input.Month)
	}

	fromCell := fmt.Sprintf("%s!%s%d", quoteSheetName(sheet), columnLetter(column), from.row)
	toCell := fmt.Sprintf("%s!%s%d", quoteSheetName(sheet), columnLetter(column), to.row)
	original := strings.TrimSpace(cellAt(rowAt(rows, from.row-1), column))
	if err := s.client.UpdateRow(fromCell, []any{fromLimit - input.Amount}); err != nil {
		return domain.BudgetReallocationResult{}, fmt.Errorf("lower %s limit: %w", from.item.Category, err)
	}
	defer s.invalidate(budgetCacheKey(year))
	if err := s.client.UpdateRow(toCell, []any{toLimit + input.Amount}); err != nil {
		moveErr := fmt.Errorf("raise %s limit: %w", to.item.Category, err)
		if restoreErr := s.client.UpdateRow(fromCell, []any{original}); restoreErr != nil {
			return domain.BudgetReallocationResult{}, errors.Join(moveErr,
				fmt.Errorf("restore %s limit at %s to %q failed, fix it by hand: %w", from.item.Category, fromCell, original, restoreErr))
		}
		return domain.BudgetReallocationResult{}, fmt.Errorf("%w; %s limit was restored", moveErr, from.item.Category)
	}
	return domain.BudgetReallocationResult{
		From: domain.BudgetCategoryLimit{Category: from.item.Category, Limit: fromLimit - input.Amount},
		To:   domain.BudgetCategoryLimit{Category: to.item.Category, Limit: toLimit + input.Amount},
	}, nil
}

type budgetCategoryCell struct {
	item domain.BudgetItem
	row  int
}

func budgetCell(items []domain.BudgetItem, layout budgetLayout, category string, sheet string) (budgetCategoryCell, error) {
	key := normalizeKey(category)
	row, ok := layout.categoryRows[key]
	if ok {
		for _, item := range items {
			if normalizeKey(item.Category) == key {
				return budgetCategoryCell{item: item, row: row}, nil
			}
		}
	}
	return budgetCategoryCell{}, fmt.Errorf("%w: category %q is not in %s", domain.ErrNotFound, category, sheet)
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
