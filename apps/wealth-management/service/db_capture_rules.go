package service

import (
	"apps/wealth-management/domain"
	"fmt"
	"strings"
)

// Smart Capture rules (WM-CAP-11): account aliases and payee categories
// confirmed during review, kept in their own tab so they are visible and
// editable in the sheet. The tab is created on the first save.

const (
	captureRulesSheet       = "CaptureRules"
	captureRulesRange       = "CaptureRules!A2:D"
	captureRulesAnchorRange = "CaptureRules!A2:A"
	captureRulesFirstRow    = 2
	cacheKeyCaptureRules    = "capture:rules"
)

var captureRulesHeader = []any{"Kind", "Pattern", "Value", "Updated"}

func (s *dbService) ListCaptureRules(forceFresh bool) ([]domain.CaptureRule, error) {
	return cachedRead(s, cacheKeyCaptureRules, sheetDataTTL, forceFresh, func() ([]domain.CaptureRule, error) {
		exists, err := s.hasSheet(captureRulesSheet)
		if err != nil || !exists {
			return []domain.CaptureRule{}, err
		}
		rows, err := s.client.ReadSheet(captureRulesRange)
		if err != nil {
			return nil, fmt.Errorf("read capture rules: %w", err)
		}
		rules := make([]domain.CaptureRule, 0, len(rows))
		for _, row := range rows {
			if rule, ok := mapCaptureRule(row); ok {
				rules = append(rules, rule)
			}
		}
		return rules, nil
	})
}

func mapCaptureRule(row []string) (domain.CaptureRule, bool) {
	rule := domain.CaptureRule{
		Kind:      domain.CaptureRuleKind(strings.ToLower(strings.TrimSpace(cellAt(row, 0)))),
		Pattern:   strings.TrimSpace(cellAt(row, 1)),
		Value:     strings.TrimSpace(cellAt(row, 2)),
		UpdatedAt: strings.TrimSpace(cellAt(row, 3)),
	}
	return rule, rule.Validate() == nil
}

func (s *dbService) hasSheet(title string) (bool, error) {
	titles, err := s.sheetTitles()
	return titles[title], err
}

// sheetTitles is the set of tab names, read in one request.
func (s *dbService) sheetTitles() (map[string]bool, error) {
	titles, err := s.client.SheetTitles()
	if err != nil {
		return nil, fmt.Errorf("list sheet tabs: %w", err)
	}
	set := make(map[string]bool, len(titles))
	for _, title := range titles {
		set[title] = true
	}
	return set, nil
}

// SaveCaptureRules upserts rules by kind and folded pattern: an existing
// row is rewritten when its value changed, new rules are appended. Within
// one call the last rule for a key wins.
func (s *dbService) SaveCaptureRules(rules []domain.CaptureRule) (domain.CaptureRulesSaveResult, error) {
	if len(rules) == 0 {
		return domain.CaptureRulesSaveResult{}, fmt.Errorf("%w: at least one rule is required", domain.ErrValidation)
	}
	order := []string{}
	byKey := map[string]domain.CaptureRule{}
	for index, rule := range rules {
		rule.Kind = domain.CaptureRuleKind(strings.ToLower(strings.TrimSpace(string(rule.Kind))))
		rule.Pattern, rule.Value = strings.TrimSpace(rule.Pattern), strings.TrimSpace(rule.Value)
		if err := rule.Validate(); err != nil {
			return domain.CaptureRulesSaveResult{}, fmt.Errorf("rule %d: %w", index+1, err)
		}
		if _, seen := byKey[rule.Key()]; !seen {
			order = append(order, rule.Key())
		}
		byKey[rule.Key()] = rule
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	exists, err := s.hasSheet(captureRulesSheet)
	if err != nil {
		return domain.CaptureRulesSaveResult{}, err
	}
	if !exists {
		if err := s.client.CreateSheet(captureRulesSheet, captureRulesHeader); err != nil {
			return domain.CaptureRulesSaveResult{}, fmt.Errorf("create %s tab: %w", captureRulesSheet, err)
		}
	}
	rows := [][]string{}
	if exists {
		if rows, err = s.client.ReadSheet(captureRulesRange); err != nil {
			return domain.CaptureRulesSaveResult{}, fmt.Errorf("read capture rules: %w", err)
		}
	}
	existing := map[string]int{}
	for index, row := range rows {
		if rule, ok := mapCaptureRule(row); ok {
			existing[rule.Key()] = index
		}
	}

	updated := s.now().In(domain.AppLocation).Format("2006-01-02 15:04")
	result := domain.CaptureRulesSaveResult{}
	var appends [][]any
	for _, key := range order {
		rule := byKey[key]
		values := []any{string(rule.Kind), rule.Pattern, rule.Value, updated}
		index, found := existing[key]
		if !found {
			appends = append(appends, values)
			continue
		}
		if current, _ := mapCaptureRule(rows[index]); current.Value == rule.Value {
			result.Unchanged++
			continue
		}
		sheetRow := captureRulesFirstRow + index
		if err := s.client.UpdateRow(fmt.Sprintf("%s!A%d:D%d", captureRulesSheet, sheetRow, sheetRow), values); err != nil {
			s.invalidate(cacheKeyCaptureRules)
			return result, fmt.Errorf("update capture rule: %w", err)
		}
		result.Updated++
	}
	if len(appends) > 0 {
		if _, err := s.client.WriteRowsAtFirstEmptyRow(captureRulesSheet, captureRulesAnchorRange, appends); err != nil {
			s.invalidate(cacheKeyCaptureRules)
			return result, fmt.Errorf("write capture rules: %w", err)
		}
		result.Created = len(appends)
	}
	s.invalidate(cacheKeyCaptureRules)
	return result, nil
}
