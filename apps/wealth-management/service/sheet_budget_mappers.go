package service

import (
	"apps/wealth-management/domain"
	"sort"
	"strings"
)

func parseCategories(rows [][]string) []domain.Category {
	categories := make([]domain.Category, 0)
	seen := map[string]struct{}{}
	for _, row := range scanCategoryRows(rows) {
		key := normalizeKey(row.category.Name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		categories = append(categories, row.category)
	}
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Name < categories[j].Name
	})
	return categories
}

// categoryRow is a category cell of the Categories tab and its 1-based row.
type categoryRow struct {
	category domain.Category
	sheetRow int
}

// scanCategoryRows returns every category row under a section header, in
// sheet order and with duplicates kept. rows must start at row 1.
func scanCategoryRows(rows [][]string) []categoryRow {
	found := []categoryRow{}
	var currentGroup *domain.CategoryType
	for i, row := range rows {
		cell := strings.TrimSpace(cellAt(row, 0))
		if cell == "" {
			continue
		}
		lower := strings.ToLower(cell)
		switch {
		case strings.Contains(lower, "income categories"):
			group := domain.CategoryTypeIncome
			currentGroup = &group
			continue
		case strings.Contains(lower, "expense categories"):
			group := domain.CategoryTypeExpense
			currentGroup = &group
			continue
		case strings.Contains(lower, "non-budget categories"):
			group := domain.CategoryTypeNonBudget
			currentGroup = &group
			continue
		case lower == "categories":
			continue
		}
		if currentGroup == nil {
			continue
		}
		// Column C is the user's own note and is not read.
		category := domain.Category{
			Name: cell, Type: *currentGroup, Nature: domain.ParseCategoryNature(cellAt(row, 1)),
			Definition: strings.TrimSpace(cellAt(row, 3)), SplitTags: domain.ParseSplitTags(cellAt(row, 4)),
		}
		found = append(found, categoryRow{category: category, sheetRow: i + 1})
	}
	return found
}

// budgetLayout locates the cells of a budget tab, for reads and limit edits.
type budgetLayout struct {
	monthColumns  map[string]int // "YYYY-MM" -> column index
	categoryRows  map[string]int // normalised category -> 1-based sheet row
	categoryNames map[string]string
}

func parseBudget(rows [][]string) ([]domain.BudgetItem, budgetLayout) {
	layout := budgetLayout{monthColumns: map[string]int{}, categoryRows: map[string]int{}, categoryNames: map[string]string{}}
	headerRow := rowAt(rows, budgetHeaderRowIndex)
	for c := budgetFirstMonthColumn; c < len(headerRow); c++ {
		if serial := parseNumber(headerRow[c]); serial > 1000 {
			layout.monthColumns[domain.MonthKey(domain.SerialToDate(serial))] = c
		}
	}
	items := make([]domain.BudgetItem, 0)
	currentType := domain.CategoryTypeNonBudget
	for r := budgetFirstDataRowIndex; r < len(rows); r++ {
		row := rows[r]
		category := strings.TrimSpace(cellAt(row, 0))
		if category == "" {
			continue
		}
		switch strings.ToLower(category) {
		case "income categories":
			currentType = domain.CategoryTypeIncome
			continue
		case "expense categories":
			currentType = domain.CategoryTypeExpense
			continue
		case "transfer categories":
			currentType = domain.CategoryTypeNonBudget
			continue
		}
		if isJunkBudgetCategory(category) {
			continue
		}
		layout.categoryRows[normalizeKey(category)] = r + 1
		layout.categoryNames[normalizeKey(category)] = category

		yearlyLimit := parseNumber(cellAt(row, budgetYearlyColumn))
		monthlyLimits := map[string]float64{}
		monthlyLimitSum := 0.0
		for month, col := range layout.monthColumns {
			value := parseNumber(cellAt(row, col))
			if value == 0 {
				continue
			}
			monthlyLimits[month] = value
			monthlyLimitSum += value
		}
		if yearlyLimit == 0 {
			yearlyLimit = monthlyLimitSum
		}
		monthlyLimit := 0.0
		if yearlyLimit > 0 {
			monthlyLimit = float64(int(yearlyLimit / 12.0))
		}
		categoryType := currentType
		items = append(items, domain.BudgetItem{
			Category:         category,
			CategoryType:     &categoryType,
			MonthlyLimit:     monthlyLimit,
			YearlyLimit:      yearlyLimit,
			MonthlyLimits:    monthlyLimits,
			MonthlyRemaining: monthlyLimit,
			YearlyRemaining:  yearlyLimit,
			Note:             strings.TrimSpace(cellAt(row, 1)),
		})
	}
	return items, layout
}

func isJunkBudgetCategory(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	if lower == "" {
		return true
	}
	switch lower {
	case "help", "income categories", "expense categories", "transfer categories":
		return true
	}
	for _, prefix := range []string{"total ", "net ", "projected", "[", "starting balance"} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return onlyDigits(lower)
}

func buildCategoryIndex(categories []domain.Category) map[string]domain.Category {
	index := make(map[string]domain.Category, len(categories))
	for _, category := range categories {
		index[normalizeKey(category.Name)] = category
	}
	return index
}
