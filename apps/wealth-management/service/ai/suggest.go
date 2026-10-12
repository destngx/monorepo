package ai

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"errors"
	"fmt"
	"strings"
)

const (
	uncategorized         = "Uncategorized"
	minSuggestPayeeLength = 3
)

// SuggestCategory proposes a category for a payee (WM-AI-8, WM-TX-5): the
// payee's own history first, then the model, constrained to known categories.
func (s *Service) SuggestCategory(ctx context.Context, payee string, model string) (port.CategorySuggestion, error) {
	payee = strings.TrimSpace(payee)
	if len([]rune(payee)) < minSuggestPayeeLength {
		return port.CategorySuggestion{}, fmt.Errorf("%w: payee needs at least %d characters", domain.ErrValidation, minSuggestPayeeLength)
	}
	categories, err := s.db.ListCategories(false)
	if err != nil {
		return port.CategorySuggestion{}, fmt.Errorf("load categories: %w", err)
	}
	names := make([]string, 0, len(categories))
	for _, category := range categories {
		names = append(names, category.Name)
	}
	transactions, err := s.db.ListTransactions(false)
	if err != nil {
		return port.CategorySuggestion{}, fmt.Errorf("load transactions: %w", err)
	}
	if category, ok := historyCategory(payee, transactions, names); ok {
		return port.CategorySuggestion{Category: category, Source: "history"}, nil
	}

	if err := s.requireConfigured(); err != nil {
		return port.CategorySuggestion{}, err
	}
	p, err := s.loadPrompts()
	if err != nil {
		return port.CategorySuggestion{}, err
	}
	masked := s.masker.Mask(payee)
	task, err := p.render("tasks/suggest-category", map[string]string{"payee": masked, "categories": strings.Join(names, ", ")})
	if err != nil {
		return port.CategorySuggestion{}, err
	}
	action, err := p.render("actions/suggest-category", map[string]string{"payee": masked})
	if err != nil {
		return port.CategorySuggestion{}, err
	}
	messages := []domain.AIMessage{
		{Role: domain.AIRoleSystem, Content: task},
		{Role: domain.AIRoleUser, Content: action + "\n\nAnswer with the exact category name only."},
	}
	primary := s.models(domain.AIUseCaseSuggest, model)[0]
	category, err := cached(ctx, s, "suggest-category", false, []any{primary, messages}, func() (string, error) {
		var category string
		_, err := s.complete(ctx, call{
			feature:  "suggest-category",
			useCase:  domain.AIUseCaseSuggest,
			model:    model,
			prompts:  p.used,
			messages: messages,
			validate: func(content string) error {
				resolved, ok := domain.ResolveCategory(content, names)
				if !ok {
					return fmt.Errorf("%w: %q is not a known category", domain.ErrAIOutput, strings.TrimSpace(content))
				}
				category = resolved
				return nil
			},
		})
		return category, err
	})
	if errors.Is(err, domain.ErrAIOutput) {
		if fallback, ok := domain.ResolveCategory(uncategorized, names); ok {
			return port.CategorySuggestion{Category: fallback, Source: "default"}, nil
		}
	}
	if err != nil {
		return port.CategorySuggestion{}, err
	}
	return port.CategorySuggestion{Category: category, Source: "ai"}, nil
}

// historyCategory is the most frequent known category used with the same payee.
func historyCategory(payee string, transactions []domain.Transaction, names []string) (string, bool) {
	counts := map[string]int{}
	best, bestCount := "", 0
	for _, t := range transactions {
		if !strings.EqualFold(strings.TrimSpace(t.Payee), payee) {
			continue
		}
		category, ok := domain.ResolveCategory(t.Category, names)
		if !ok || strings.EqualFold(category, uncategorized) {
			continue
		}
		counts[category]++
		if counts[category] > bestCount {
			best, bestCount = category, counts[category]
		}
	}
	return best, bestCount > 0
}
