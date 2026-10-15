package ai

import (
	"apps/wealth-management/domain"
	"context"
	"fmt"
	"sort"
	"strings"
)

// categorizeCapture fills category and card tag per suggestion: confirmed
// payee rules, then the payee's history, then one batched model call
// constrained to known categories and the card's rule tags (WM-CAP-5).
// Whatever is left is Uncategorized and flagged for review.
func (s *Service) categorizeCapture(ctx context.Context, p *prompts, suggestions []domain.CaptureSuggestion, data captureData) {
	payeeRules := map[string]string{}
	for _, rule := range data.rules {
		if rule.Kind != domain.CaptureRulePayee {
			continue
		}
		if category, ok := domain.ResolveCategory(rule.Value, data.categories); ok {
			payeeRules[domain.FoldText(rule.Pattern)] = category
		}
	}
	var pending []int
	for index := range suggestions {
		suggestion := &suggestions[index]
		payee := suggestion.Transaction.Payee
		suggestion.TagOptions = data.tagOptions[suggestion.Transaction.AccountName]
		if suggestion.Transaction.Category == "" {
			if category, ok := payeeRules[domain.FoldText(payee)]; ok {
				suggestion.Transaction.Category, suggestion.CategorySource = category, domain.CategorySourceRule
			} else if category, ok := historyCategory(payee, data.transactions, data.categories); ok {
				suggestion.Transaction.Category, suggestion.CategorySource = category, domain.CategorySourceHistory
			}
		}
		// The card's own digits on the material outrank the payee's history.
		if card, ok := domain.MatchCaptureCard(suggestion.AccountHint, data.cards); ok && strings.EqualFold(card.Account, suggestion.Transaction.AccountName) {
			suggestion.Transaction.Tags = []string{card.Keyword}
		} else if tag, ok := historyTag(payee, suggestion.Transaction.AccountName, suggestion.TagOptions, data.transactions); ok {
			suggestion.Transaction.Tags = []string{tag}
		}
		if suggestion.Transaction.Category == "" || (len(suggestion.TagOptions) > 0 && len(suggestion.Transaction.Tags) == 0) {
			pending = append(pending, index)
		}
	}
	if len(pending) > 0 {
		if err := s.categorizeWithModel(ctx, p, suggestions, pending, data.categories); err != nil {
			s.log.LogError(ctx, "capture categorisation failed; rows left for review", err)
			for _, index := range pending {
				if suggestions[index].Transaction.Category == "" {
					suggestions[index].Notes = append(suggestions[index].Notes, "Automatic categorisation failed; pick a category.")
				}
			}
		}
	}
	fallback, hasFallback := domain.ResolveCategory(uncategorized, data.categories)
	for index := range suggestions {
		suggestion := &suggestions[index]
		if suggestion.Transaction.Category == "" {
			if hasFallback {
				suggestion.Transaction.Category, suggestion.CategorySource = fallback, domain.CategorySourceDefault
			}
			suggestion.LowConfidence = appendUnique(suggestion.LowConfidence, "category")
		}
	}
}

// historyTag is the card rule tag most used with this payee on the account.
func historyTag(payee string, account string, options []string, transactions []domain.Transaction) (string, bool) {
	if len(options) == 0 {
		return "", false
	}
	counts := map[string]int{}
	best, bestCount := "", 0
	for _, t := range transactions {
		if t.AccountName != account || !strings.EqualFold(strings.TrimSpace(t.Payee), payee) {
			continue
		}
		for _, tag := range t.Tags {
			for _, option := range options {
				if strings.EqualFold(strings.TrimSpace(tag), option) {
					counts[option]++
					if counts[option] > bestCount {
						best, bestCount = option, counts[option]
					}
				}
			}
		}
	}
	return best, bestCount > 0
}

type categorizeItem struct {
	ID          string   `json:"id"`
	Payee       string   `json:"payee"`
	Description string   `json:"description,omitempty"`
	Direction   string   `json:"direction"`
	Amount      float64  `json:"amount"`
	Currency    string   `json:"currency"`
	Account     string   `json:"account,omitempty"`
	TagOptions  []string `json:"tagOptions,omitempty"`
}

type categorizeReply struct {
	Items []struct {
		ID       string `json:"id"`
		Category string `json:"category"`
		Tag      string `json:"tag"`
	} `json:"items"`
}

const captureCategorizeFormat = `IMPORTANT: Respond with ONLY a valid JSON object (no markdown, no backticks, no explanation) in this exact format:
{"items": [{"id": "s1", "category": "<exact name from the category list>", "tag": "<one of the item's tagOptions, or empty>"}]}

Rules:
- One entry per input item, same id.
- category must be copied exactly from the category list; use "" when none fits.
- tag must be copied exactly from that item's tagOptions (the card the payment was made with, when the item shows which); use "" when it does not or the item has no tagOptions.`

func (s *Service) categorizeWithModel(ctx context.Context, p *prompts, suggestions []domain.CaptureSuggestion, pending []int, categories []string) error {
	task, ok := p.lookup(keyCaptureCategorize, map[string]string{"categories": strings.Join(categories, ", ")})
	if !ok {
		return fmt.Errorf("%w: %s", domain.ErrPromptMissing, keyCaptureCategorize)
	}
	items := make([]categorizeItem, 0, len(pending))
	byID := map[string]int{}
	for _, index := range pending {
		suggestion := suggestions[index]
		description := ""
		if suggestion.Transaction.Memo != nil {
			description = strings.TrimSpace(strings.TrimPrefix(*suggestion.Transaction.Memo, domain.CaptureMemoPrefix))
		}
		items = append(items, categorizeItem{
			ID: suggestion.ID, Payee: s.masker.Mask(suggestion.Transaction.Payee), Description: s.masker.Mask(description),
			Direction: string(suggestion.Direction), Amount: suggestion.Amount, Currency: suggestion.Currency,
			Account: suggestion.Transaction.AccountName, TagOptions: suggestion.TagOptions,
		})
		byID[suggestion.ID] = index
	}
	sort.Slice(items, func(a, b int) bool { return byID[items[a].ID] < byID[items[b].ID] })
	user := "Categories: " + strings.Join(categories, ", ") + "\n\nItems:\n" + toJSON(items) + "\n\n" + captureCategorizeFormat
	var reply categorizeReply
	_, err := s.complete(ctx, call{
		feature:  "capture-categorize",
		useCase:  domain.AIUseCaseSuggest,
		prompts:  append([]string(nil), p.used...),
		messages: []domain.AIMessage{{Role: domain.AIRoleSystem, Content: task}, {Role: domain.AIRoleUser, Content: user}},
		json:     true,
		validate: func(content string) error {
			reply = categorizeReply{}
			return domain.ParseAIJSON(content, &reply)
		},
	})
	if err != nil {
		return err
	}
	for _, answer := range reply.Items {
		index, ok := byID[answer.ID]
		if !ok {
			continue
		}
		suggestion := &suggestions[index]
		if suggestion.Transaction.Category == "" {
			if category, ok := domain.ResolveCategory(answer.Category, categories); ok {
				suggestion.Transaction.Category, suggestion.CategorySource = category, domain.CategorySourceAI
			}
		}
		if len(suggestion.Transaction.Tags) == 0 {
			for _, option := range suggestion.TagOptions {
				if strings.EqualFold(strings.TrimSpace(answer.Tag), option) {
					suggestion.Transaction.Tags = []string{option}
				}
			}
		}
	}
	return nil
}
