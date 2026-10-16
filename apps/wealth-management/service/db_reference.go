package service

import (
	"apps/wealth-management/domain"
	"fmt"
	"slices"
	"strings"
)

func (s *dbService) ListCategories(forceFresh bool) ([]domain.Category, error) {
	return cachedRead(s, cacheKeyCats, sheetDataTTL, forceFresh, func() ([]domain.Category, error) {
		rows, err := s.client.ReadSheet(categoriesSheetRange)
		if err != nil {
			return nil, fmt.Errorf("read categories: %w", err)
		}
		return parseCategories(rows), nil
	})
}

// categoryNatureColumn is Categories column B.
const categoryNatureColumn = "B"

// UpdateCategoryNature finds the category's row by exact name (spaces
// around it ignored) in a fresh read and writes column B (WM-LEND-5). The
// transactions cache goes too: transactions carry their category's nature.
func (s *dbService) UpdateCategoryNature(input domain.CategoryNatureUpdate) (domain.Category, error) {
	if err := input.Validate(); err != nil {
		return domain.Category{}, err
	}
	nature, _ := input.ParsedNature()
	name := strings.TrimSpace(input.Category)
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	rows, err := s.client.ReadSheet(categoriesSheetRange)
	if err != nil {
		return domain.Category{}, fmt.Errorf("read categories: %w", err)
	}
	var matches []categoryRow
	for _, row := range scanCategoryRows(rows) {
		if row.category.Name == name {
			matches = append(matches, row)
		}
	}
	switch len(matches) {
	case 0:
		return domain.Category{}, fmt.Errorf("%w: category %q is not in the Categories tab", domain.ErrNotFound, name)
	case 1:
	default:
		return domain.Category{}, fmt.Errorf("%w: category %q appears %d times in the Categories tab; remove the duplicates first", domain.ErrConflict, name, len(matches))
	}
	cell := fmt.Sprintf("Categories!%s%d", categoryNatureColumn, matches[0].sheetRow)
	if err := s.client.UpdateRow(cell, []any{string(nature)}); err != nil {
		return domain.Category{}, fmt.Errorf("update category nature: %w", err)
	}
	s.invalidate(cacheKeyCats, "transactions:*")
	category := matches[0].category
	category.Nature = nature
	return category, nil
}

func (s *dbService) ListGoals(forceFresh bool) ([]domain.Goal, error) {
	return cachedRead(s, cacheKeyGoals, sheetDataTTL, forceFresh, func() ([]domain.Goal, error) {
		rows, err := s.client.ReadSheet(goalsSheetRange)
		if err != nil {
			return nil, fmt.Errorf("read goals: %w", err)
		}
		goals := make([]domain.Goal, 0)
		for i, row := range rows {
			if goal, ok := mapGoal(row, i); ok {
				goals = append(goals, goal)
			}
		}
		return goals, nil
	})
}

func (s *dbService) ListTags(forceFresh bool) ([]string, error) {
	transactions, err := s.ListTransactions(forceFresh)
	if err != nil {
		return nil, err
	}
	set := map[string]struct{}{}
	for _, transaction := range transactions {
		for _, tag := range transaction.Tags {
			if normalized := normalizeKey(tag); normalized != "" {
				set[normalized] = struct{}{}
			}
		}
	}
	tags := make([]string, 0, len(set))
	for tag := range set {
		tags = append(tags, tag)
	}
	slices.Sort(tags)
	return tags, nil
}

func (s *dbService) GetInvestmentAssets(forceFresh bool) (domain.InvestmentAssets, error) {
	return cachedRead(s, cacheKeyInvest, sheetDataTTL, forceFresh, func() (domain.InvestmentAssets, error) {
		tabs, err := s.readRanges(cryptoSheetRange, fundsSheetRange)
		if err != nil {
			return domain.InvestmentAssets{}, err
		}
		return domain.InvestmentAssets{
			Crypto: parseHoldings(tabs[0], cryptoSection),
			Funds:  parseHoldings(tabs[1], fundSection),
		}, nil
	})
}
