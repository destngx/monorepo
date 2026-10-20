package ai

import (
	"apps/wealth-management/domain"
	"context"
	"fmt"
	"sort"
	"strings"
)

// categorizeCapture fills category, card tag and split tag per suggestion:
// confirmed payee rules, then the payee's history, then one batched model
// call constrained to known categories, the card's rule tags and the
// category's split tags, guided by the category definitions (WM-CAP-5,
// WM-CAT-5). A category's split tag comes from the payee's history in it
// when there is any; otherwise the model picks it. Whatever is left is
// Uncategorized and flagged for review. Notes are the user's typed notes
// that read as no transaction, shown to the model.
func (s *Service) categorizeCapture(ctx context.Context, p *prompts, suggestions []domain.CaptureSuggestion, data captureData, notes []string) {
	payeeRules := payeeRuleCategories(data.rules, data.categories)
	var pending []int
	// splitOpen marks rows whose category has split tags but no tagged
	// history to decide one; the model picks it.
	splitOpen := map[int]bool{}
	for index := range suggestions {
		suggestion := &suggestions[index]
		payee := suggestion.Transaction.Payee
		suggestion.TagOptions = data.tagOptions[suggestion.Transaction.AccountName]
		// A category the user's note asked for outranks rules and history;
		// one that names no category exactly is left to the model.
		if suggestion.Transaction.Category == "" && suggestion.CategoryHint != "" {
			if category, ok := domain.ResolveCategory(suggestion.CategoryHint, data.categories); ok {
				suggestion.Transaction.Category, suggestion.CategorySource = category, domain.CategorySourceManual
			}
		} else if suggestion.Transaction.Category == "" {
			if category, ok := payeeRules[domain.FoldText(payee)]; ok {
				suggestion.Transaction.Category, suggestion.CategorySource = category, domain.CategorySourceRule
			} else if category, ok := historyCategory(payee, data.transactions, data.categories); ok {
				suggestion.Transaction.Category, suggestion.CategorySource = category, domain.CategorySourceHistory
			}
		}
		// The card's own digits on the material outrank the payee's history;
		// an account with one open card can only have been paid with it.
		if card, ok := domain.MatchCaptureCard(suggestion.AccountHint, data.cards); ok && strings.EqualFold(card.Account, suggestion.Transaction.AccountName) {
			suggestion.Transaction.Tags = []string{card.Keyword}
		} else if len(suggestion.TagOptions) == 1 {
			suggestion.Transaction.Tags = []string{suggestion.TagOptions[0]}
		} else if tag, ok := historyTag(payee, suggestion.Transaction.AccountName, suggestion.TagOptions, data.transactions); ok {
			suggestion.Transaction.Tags = []string{tag}
		}
		if category := suggestion.Transaction.Category; category != "" {
			if tags := data.splits.For(category); len(tags) > 0 {
				tag, decided := domain.HistorySplitTag(payee, category, tags, data.transactions)
				addTag(&suggestion.Transaction, tag)
				splitOpen[index] = !decided
			}
		}
		if suggestion.Transaction.Category == "" || !hasCardTag(*suggestion) || splitOpen[index] {
			pending = append(pending, index)
		}
	}
	if len(pending) > 0 {
		if err := s.categorizeWithModel(ctx, p, suggestions, pending, data, splitOpen, notes); err != nil {
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

// hasCardTag reports whether the row has its card's rule tag, or needs none.
func hasCardTag(suggestion domain.CaptureSuggestion) bool {
	if len(suggestion.TagOptions) == 0 {
		return true
	}
	for _, tag := range suggestion.Transaction.Tags {
		for _, option := range suggestion.TagOptions {
			if strings.EqualFold(strings.TrimSpace(tag), option) {
				return true
			}
		}
	}
	return false
}

// addTag appends tag unless it is empty or already there.
func addTag(transaction *domain.TransactionCreateInput, tag string) {
	if tag == "" {
		return
	}
	for _, existing := range transaction.Tags {
		if strings.EqualFold(strings.TrimSpace(existing), tag) {
			return
		}
	}
	transaction.Tags = append(transaction.Tags, tag)
}

// payeeRuleCategories maps each confirmed payee rule, folded, to its known
// category; rules naming a category that no longer exists are ignored.
func payeeRuleCategories(rules []domain.CaptureRule, categories []string) map[string]string {
	payeeRules := map[string]string{}
	for _, rule := range rules {
		if rule.Kind != domain.CaptureRulePayee {
			continue
		}
		if category, ok := domain.ResolveCategory(rule.Value, categories); ok {
			payeeRules[domain.FoldText(rule.Pattern)] = category
		}
	}
	return payeeRules
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
	Hint        string   `json:"hint,omitempty"`
	Direction   string   `json:"direction"`
	Amount      float64  `json:"amount"`
	Currency    string   `json:"currency"`
	Account     string   `json:"account,omitempty"`
	TagOptions  []string `json:"tagOptions,omitempty"`
	// Category is set when it is already decided and only the tags are open.
	Category string `json:"category,omitempty"`
	// Notes are the user's notes about earlier rows with this counterparty
	// (WM-CAP-24).
	Notes []string `json:"notes,omitempty"`
}

type categorizeReply struct {
	Items []struct {
		ID       string `json:"id"`
		Category string `json:"category"`
		Tag      string `json:"tag"`
		SplitTag string `json:"splitTag"`
	} `json:"items"`
}

const captureCategorizeFormat = `IMPORTANT: Respond with ONLY a valid JSON object (no markdown, no backticks, no explanation) in this exact format:
{"items": [{"id": "s1", "category": "<exact name from the category list>", "tag": "<one of the item's tagOptions, or empty>", "splitTag": "<one of the chosen category's split tags, or empty>"}]}

Rules:
- One entry per input item, same id.
- category must be copied exactly from the category list; use "" when none fits.
- An item's hint is the category the user asked for in their own words: pick the category it names or means ("transit" = a transport category), whatever the payee suggests.
- The user's note, when given, was sent with the materials: let what it says the money was for guide the category of the items it fits.
- An item's notes are what the user wrote about earlier payments to or from the same counterparty: use them to tell what this payment is likely for.
- tag must be copied exactly from that item's tagOptions (the card the payment was made with, when the item shows which); use "" when it does not or the item has no tagOptions.
- An item that already has a category keeps it: copy it back and only choose tag and splitTag.
- When a category guide is given, choose between categories by their definitions, and set splitTag to one of the chosen category's split tags when its definition says that tag applies to the item; otherwise "".`

func (s *Service) categorizeWithModel(ctx context.Context, p *prompts, suggestions []domain.CaptureSuggestion, pending []int, data captureData, splitOpen map[int]bool, notes []string) error {
	categories := data.categories
	task, ok := p.lookup(keyCaptureCategorize, map[string]string{"categories": strings.Join(categories, ", ")})
	if !ok {
		return fmt.Errorf("%w: %s", domain.ErrPromptMissing, keyCaptureCategorize)
	}
	items := make([]categorizeItem, 0, len(pending))
	byID := map[string]int{}
	for _, index := range pending {
		suggestion := suggestions[index]
		var notes []string
		for _, note := range suggestion.PastNotes {
			notes = append(notes, s.masker.Mask(note))
		}
		description := ""
		if suggestion.Transaction.Memo != nil {
			description = strings.TrimSpace(strings.TrimPrefix(*suggestion.Transaction.Memo, domain.CaptureMemoPrefix))
		}
		items = append(items, categorizeItem{
			ID: suggestion.ID, Payee: s.masker.Mask(suggestion.Transaction.Payee), Description: s.masker.Mask(description), Hint: s.masker.Mask(suggestion.CategoryHint),
			Direction: string(suggestion.Direction), Amount: suggestion.Amount, Currency: suggestion.Currency,
			Account: suggestion.Transaction.AccountName, TagOptions: suggestion.TagOptions, Category: suggestion.Transaction.Category,
			Notes: notes,
		})
		byID[suggestion.ID] = index
	}
	sort.Slice(items, func(a, b int) bool { return byID[items[a].ID] < byID[items[b].ID] })
	user := "Categories: " + strings.Join(categories, ", ") + "\n\n"
	if data.guide != "" {
		user += "Category guide:\n" + s.masker.Mask(data.guide) + "\n\n"
	}
	user += "Items:\n" + toJSON(items) + "\n\n"
	if note := strings.TrimSpace(strings.Join(notes, "\n")); note != "" {
		user += "The user's note:\n" + s.masker.Mask(note) + "\n\n"
	}
	user += captureCategorizeFormat
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
		splitDecided := suggestion.Transaction.Category != "" && !splitOpen[index]
		if suggestion.Transaction.Category == "" {
			if category, ok := domain.ResolveCategory(answer.Category, categories); ok {
				suggestion.Transaction.Category, suggestion.CategorySource = category, domain.CategorySourceAI
			}
		}
		if !hasCardTag(*suggestion) {
			for _, option := range suggestion.TagOptions {
				if strings.EqualFold(strings.TrimSpace(answer.Tag), option) {
					suggestion.Transaction.Tags = append([]string{option}, suggestion.Transaction.Tags...)
				}
			}
		}
		if !splitDecided {
			if tag, ok := data.splits.Resolve(suggestion.Transaction.Category, answer.SplitTag); ok {
				addTag(&suggestion.Transaction, tag)
			}
		}
	}
	return nil
}
