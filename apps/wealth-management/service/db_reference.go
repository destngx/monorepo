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

const (
	// categoryNatureColumn is Categories column B.
	categoryNatureColumn = "B"
	// categoryDefinitionColumn is Categories column D; the split tags are E.
	categoryDefinitionColumn = "D"
)

// UpdateCategoryNature writes the category's column B (WM-LEND-5). The
// transactions cache goes too: transactions carry their category's nature.
func (s *dbService) UpdateCategoryNature(input domain.CategoryNatureUpdate) (domain.Category, error) {
	if err := input.Validate(); err != nil {
		return domain.Category{}, err
	}
	nature, _ := input.ParsedNature()
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	row, err := s.findCategoryRow(input.Category)
	if err != nil {
		return domain.Category{}, err
	}
	cell := fmt.Sprintf("Categories!%s%d", categoryNatureColumn, row.sheetRow)
	if err := s.client.UpdateRow(cell, []any{string(nature)}); err != nil {
		return domain.Category{}, fmt.Errorf("update category nature: %w", err)
	}
	s.invalidate(cacheKeyCats, "transactions:*")
	category := row.category
	category.Nature = nature
	return category, nil
}

// UpdateCategoryDefinition writes the category's definition and split tags
// to columns D and E (WM-CAT-1, WM-CAT-2).
func (s *dbService) UpdateCategoryDefinition(input domain.CategoryDefinitionUpdate) (domain.Category, error) {
	if err := input.Validate(); err != nil {
		return domain.Category{}, err
	}
	definition, tags := strings.TrimSpace(input.Definition), input.NormalizedSplitTags()
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	row, err := s.findCategoryRow(input.Category)
	if err != nil {
		return domain.Category{}, err
	}
	cell := fmt.Sprintf("Categories!%s%d", categoryDefinitionColumn, row.sheetRow)
	if err := s.client.UpdateRow(cell, []any{definition, strings.Join(tags, ",")}); err != nil {
		return domain.Category{}, fmt.Errorf("update category definition: %w", err)
	}
	s.invalidate(cacheKeyCats)
	category := row.category
	category.Definition, category.SplitTags = definition, tags
	return category, nil
}

// findCategoryRow finds the category's row by exact name (spaces around it
// ignored) in a fresh read; a missing or repeated name is an error.
func (s *dbService) findCategoryRow(name string) (categoryRow, error) {
	name = strings.TrimSpace(name)
	rows, err := s.client.ReadSheet(categoriesSheetRange)
	if err != nil {
		return categoryRow{}, fmt.Errorf("read categories: %w", err)
	}
	var matches []categoryRow
	for _, row := range scanCategoryRows(rows) {
		if row.category.Name == name {
			matches = append(matches, row)
		}
	}
	switch len(matches) {
	case 0:
		return categoryRow{}, fmt.Errorf("%w: category %q is not in the Categories tab", domain.ErrNotFound, name)
	case 1:
		return matches[0], nil
	default:
		return categoryRow{}, fmt.Errorf("%w: category %q appears %d times in the Categories tab; remove the duplicates first", domain.ErrConflict, name, len(matches))
	}
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
