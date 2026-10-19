package domain

import (
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxCategoryDefinitionRunes = 500
	maxSplitTagRunes           = 30
	maxSplitTags               = 5
)

// ParseSplitTags reads a comma-separated split tags cell: trimmed, lower
// case, blanks and repeats dropped, first occurrence order kept.
func ParseSplitTags(cell string) []string {
	return normalizeSplitTags(strings.Split(cell, ","))
}

func normalizeSplitTags(values []string) []string {
	var tags []string
	for _, value := range values {
		tag := strings.ToLower(strings.TrimSpace(value))
		if tag != "" && !slices.Contains(tags, tag) {
			tags = append(tags, tag)
		}
	}
	return tags
}

// HasSplitTag reports whether the category lists tag as a split tag
// (case-insensitive).
func (c Category) HasSplitTag(tag string) bool {
	return slices.Contains(c.SplitTags, strings.ToLower(strings.TrimSpace(tag)))
}

// CategoryDefinitionUpdate sets a category's definition (column D) and split
// tags (column E) (WM-CAT-1, WM-CAT-2). Empty values clear them. Category is
// matched by its exact name.
type CategoryDefinitionUpdate struct {
	Category   string   `json:"category"`
	Definition string   `json:"definition"`
	SplitTags  []string `json:"splitTags"`
}

func (input CategoryDefinitionUpdate) Validate() error {
	var problems []string
	if strings.TrimSpace(input.Category) == "" {
		problems = append(problems, "category is required")
	}
	if utf8.RuneCountInString(strings.TrimSpace(input.Definition)) > maxCategoryDefinitionRunes {
		problems = append(problems, fmt.Sprintf("definition must be at most %d characters", maxCategoryDefinitionRunes))
	}
	for _, tag := range input.SplitTags {
		if strings.Contains(tag, ",") {
			problems = append(problems, fmt.Sprintf("split tag %q must not contain a comma", tag))
		}
		if utf8.RuneCountInString(strings.TrimSpace(tag)) > maxSplitTagRunes {
			problems = append(problems, fmt.Sprintf("split tag %q must be at most %d characters", tag, maxSplitTagRunes))
		}
	}
	if len(input.NormalizedSplitTags()) > maxSplitTags {
		problems = append(problems, fmt.Sprintf("at most %d split tags", maxSplitTags))
	}
	return validationError(problems)
}

// NormalizedSplitTags are the split tags to write: trimmed, lower case,
// blanks and repeats dropped.
func (input CategoryDefinitionUpdate) NormalizedSplitTags() []string {
	return normalizeSplitTags(input.SplitTags)
}

// SplitSpend is the share of a category's month spent on transactions
// carrying one of its split tags (WM-CAT-3).
type SplitSpend struct {
	Tag            string  `json:"tag"`
	Spent          float64 `json:"spent"`
	PercentOfSpent float64 `json:"percentOfSpent"`
}

// ApplySplits fills each expense line's Splits from the category's split
// tags: Σ(payment - deposit) of the month's transactions in the category
// that carry the tag, never below zero. A transaction with two split tags
// counts in both.
func ApplySplits(report *BudgetReport, categories []Category, transactions []Transaction) {
	month, err := ParseMonthKey(report.Month)
	if err != nil {
		return
	}
	splits := NewSplitTagIndex(categories)
	for i := range report.Lines {
		line := &report.Lines[i]
		tags := splits.For(line.Category)
		if !line.CountsAsExpense() || len(tags) == 0 {
			continue
		}
		spent := splitSpendByMonth(transactions, line.Category, tags, month, month)
		line.Splits = make([]SplitSpend, 0, len(tags))
		for _, tag := range tags {
			amount := clampZero(spent[tag][0])
			line.Splits = append(line.Splits, SplitSpend{Tag: tag, Spent: amount, PercentOfSpent: percentOf(amount, line.Spent)})
		}
	}
}

// SplitTagIndex finds a category's split tags by name, ignoring case and
// surrounding spaces.
type SplitTagIndex map[string][]string

// NewSplitTagIndex indexes the categories that have split tags.
func NewSplitTagIndex(categories []Category) SplitTagIndex {
	index := SplitTagIndex{}
	for _, category := range categories {
		if len(category.SplitTags) > 0 {
			index[normalizeCategory(category.Name)] = category.SplitTags
		}
	}
	return index
}

// For returns the category's split tags; nil when it has none.
func (index SplitTagIndex) For(category string) []string {
	return index[normalizeCategory(category)]
}

// Resolve returns the split tag of category that matches tag (case and
// spaces ignored); false when the category has no such split tag.
func (index SplitTagIndex) Resolve(category, tag string) (string, bool) {
	tag = strings.ToLower(strings.TrimSpace(tag))
	if tag != "" && slices.Contains(index.For(category), tag) {
		return tag, true
	}
	return "", false
}

// HistorySplitTag reads the payee's earlier transactions in the category
// (WM-CAT-5). History decides only once one of them carries a split tag:
// rows from before the tags existed say nothing. decided is false until
// then; tag is the split tag on at least half of the payee's rows in the
// category, the most used first, or "" when none is that common.
func HistorySplitTag(payee, category string, tags []string, transactions []Transaction) (tag string, decided bool) {
	payee, key := strings.TrimSpace(payee), normalizeCategory(category)
	total, tagged := 0, 0
	counts := map[string]int{}
	for _, t := range transactions {
		if normalizeCategory(t.Category) != key || !strings.EqualFold(strings.TrimSpace(t.Payee), payee) {
			continue
		}
		total++
		carries := false
		for _, candidate := range tags {
			if slices.ContainsFunc(t.Tags, func(value string) bool { return strings.EqualFold(strings.TrimSpace(value), candidate) }) {
				counts[candidate]++
				carries = true
			}
		}
		if carries {
			tagged++
		}
	}
	best := 0
	for _, candidate := range tags {
		if counts[candidate] > best && counts[candidate]*2 >= total {
			tag, best = candidate, counts[candidate]
		}
	}
	return tag, tagged > 0
}

// splitSpendByMonth returns, per split tag, the category's net tagged spend
// in each month from first to last (inclusive), oldest first.
func splitSpendByMonth(transactions []Transaction, category string, tags []string, first, last time.Time) map[string][]float64 {
	months := monthsBetween(first, last) + 1
	spent := map[string][]float64{}
	for _, tag := range tags {
		spent[tag] = make([]float64, months)
	}
	key := normalizeCategory(category)
	for _, t := range transactions {
		if !t.Usable() || normalizeCategory(t.Category) != key {
			continue
		}
		effective := EffectiveDate(t)
		index := monthsBetween(first, effective)
		if index < 0 || index >= months {
			continue
		}
		for _, tag := range tags {
			if slices.ContainsFunc(t.Tags, func(value string) bool { return strings.EqualFold(strings.TrimSpace(value), tag) }) {
				spent[tag][index] += t.PaymentAmount() - t.DepositAmount()
			}
		}
	}
	return spent
}

// CategoryGuide is the category definitions and split tags for an AI prompt,
// one line per category that has either; empty when none has.
func CategoryGuide(categories []Category) string {
	lines := []string{}
	for _, category := range categories {
		definition := strings.TrimSpace(category.Definition)
		if definition == "" && len(category.SplitTags) == 0 {
			continue
		}
		line := "- " + category.Name + ":"
		if definition != "" {
			line += " " + definition
		}
		if len(category.SplitTags) > 0 {
			line += " Split tags: " + strings.Join(category.SplitTags, ", ") + "."
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
