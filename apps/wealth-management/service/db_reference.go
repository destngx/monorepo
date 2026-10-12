package service

import (
	"apps/wealth-management/domain"
	"fmt"
	"slices"
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

func (s *dbService) ListLoans(forceFresh bool) ([]domain.Loan, error) {
	return cachedRead(s, cacheKeyLoans, sheetDataTTL, forceFresh, func() ([]domain.Loan, error) {
		rows, err := s.client.ReadSheet(loansSheetRange)
		if err != nil {
			return nil, fmt.Errorf("read loans: %w", err)
		}
		return parseLoans(rows), nil
	})
}

func parseLoans(rows [][]string) []domain.Loan {
	loans := make([]domain.Loan, 0)
	for _, row := range rows {
		if loan, ok := mapLoan(row); ok {
			loans = append(loans, loan)
		}
	}
	return loans
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
