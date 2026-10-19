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

// SuggestCategory proposes a category for a payee (WM-AI-8, WM-TX-5): a
// confirmed payee rule (WM-CAP-11) first, then the payee's own history, then
// the model, constrained to known categories. A split tag of the category
// comes from the payee's history in it, or from the model, which then sees
// the category definitions (WM-CAT-5).
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
	splits := domain.NewSplitTagIndex(categories)
	transactions, err := s.db.ListTransactions(false)
	if err != nil {
		return port.CategorySuggestion{}, fmt.Errorf("load transactions: %w", err)
	}
	rules, err := s.db.ListCaptureRules(false)
	if err != nil {
		return port.CategorySuggestion{}, fmt.Errorf("load capture rules: %w", err)
	}
	fromHistory := func(category, source string) port.CategorySuggestion {
		tag, _ := domain.HistorySplitTag(payee, category, splits.For(category), transactions)
		return port.CategorySuggestion{Category: category, Source: source, SplitTag: tag}
	}
	if category, ok := payeeRuleCategories(rules, names)[domain.FoldText(payee)]; ok {
		return fromHistory(category, domain.CategorySourceRule), nil
	}
	if category, ok := historyCategory(payee, transactions, names); ok {
		return fromHistory(category, domain.CategorySourceHistory), nil
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
	user := action + "\n\nAnswer with the exact category name only."
	if guide := domain.CategoryGuide(categories); guide != "" {
		user = action + "\n\nCategory guide:\n" + s.masker.Mask(guide) + "\n\n" + suggestGuidedFormat
	}
	messages := []domain.AIMessage{
		{Role: domain.AIRoleSystem, Content: task},
		{Role: domain.AIRoleUser, Content: user},
	}
	primary := s.models(domain.AIUseCaseSuggest, model)[0]
	suggestion, err := cached(ctx, s, "suggest-category", false, []any{primary, messages}, func() (port.CategorySuggestion, error) {
		var suggestion port.CategorySuggestion
		_, err := s.complete(ctx, call{
			feature:  "suggest-category",
			useCase:  domain.AIUseCaseSuggest,
			model:    model,
			prompts:  p.used,
			messages: messages,
			validate: func(content string) error {
				answer := suggestAnswer{Category: content}
				if domain.ParseAIJSON(content, &answer) != nil {
					answer = suggestAnswer{Category: content}
				}
				resolved, ok := domain.ResolveCategory(answer.Category, names)
				if !ok {
					return fmt.Errorf("%w: %q is not a known category", domain.ErrAIOutput, strings.TrimSpace(answer.Category))
				}
				suggestion = port.CategorySuggestion{Category: resolved, Source: domain.CategorySourceAI}
				suggestion.SplitTag, _ = splits.Resolve(resolved, answer.SplitTag)
				return nil
			},
		})
		return suggestion, err
	})
	if errors.Is(err, domain.ErrAIOutput) {
		if fallback, ok := domain.ResolveCategory(uncategorized, names); ok {
			return port.CategorySuggestion{Category: fallback, Source: domain.CategorySourceDefault}, nil
		}
	}
	if err != nil {
		return port.CategorySuggestion{}, err
	}
	return suggestion, nil
}

// suggestAnswer is the guided suggestion reply; a bare category name is
// accepted too.
type suggestAnswer struct {
	Category string `json:"category"`
	SplitTag string `json:"splitTag"`
}

const suggestGuidedFormat = `Use the category guide to choose between categories. Respond with ONLY a JSON object: {"category": "<exact category name>", "splitTag": "<one of that category's split tags when the guide says it applies to this payee, else empty>"}`

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
