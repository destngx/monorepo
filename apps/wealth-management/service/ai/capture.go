package ai

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"fmt"
	"strings"
	"sync"
)

// Smart Capture (§6.2): the model reads each material into events and
// picks categories it cannot get from rules or history; everything in
// between is the deterministic domain pipeline. Nothing is cached or
// written, and material content never reaches the logs (WM-CAP-10).
//
// Consolidation across materials is deterministic (domain.ConsolidateCaptureItems),
// so the tasks/capture-consolidate prompt of WM-CAP-13 is not used.

const (
	keyCaptureExtract    = "tasks/capture-extract"
	keyCaptureCategorize = "tasks/capture-categorize"
	captureConcurrency   = 3
)

// capturePrompts are reported by Status alongside RequiredPrompts.
var capturePrompts = []string{keyCaptureExtract, keyCaptureCategorize}

type captureData struct {
	accounts     []domain.Account
	categories   []string
	transfer     string
	transactions []domain.Transaction
	rules        []domain.CaptureRule
	tagOptions   map[string][]string
}

func (s *Service) Capture(ctx context.Context, request port.CaptureRequest, progress func(port.CaptureProgress)) (domain.CaptureResult, error) {
	if err := domain.ValidateCaptureMaterials(request.Materials, s.captureLimits); err != nil {
		return domain.CaptureResult{}, err
	}
	if err := s.requireConfigured(); err != nil {
		return domain.CaptureResult{}, err
	}
	report := serialised(progress)
	data, err := s.loadCaptureData()
	if err != nil {
		return domain.CaptureResult{}, err
	}
	p, err := s.loadPrompts()
	if err != nil {
		return domain.CaptureResult{}, err
	}
	system, err := p.render(keyCaptureExtract, map[string]string{
		"today":    s.today().Format("2006-01-02"),
		"accounts": captureAccountList(data.accounts),
		"aliases":  captureAliasList(data.rules),
	})
	if err != nil {
		return domain.CaptureResult{}, err
	}

	extractions, err := s.extractAll(ctx, request, system, p.used, report)
	if err != nil {
		return domain.CaptureResult{}, err
	}
	result := domain.CaptureResult{Suggestions: []domain.CaptureSuggestion{}, Unreadable: []domain.CaptureUnreadable{}}
	var items []domain.CaptureItem
	for index, extraction := range extractions {
		materialID := request.Materials[index].ID
		if extraction.err != nil || extraction.value.Unreadable {
			result.Unreadable = append(result.Unreadable, domain.CaptureUnreadable{MaterialID: materialID, Reason: extraction.reason()})
		}
		for _, event := range extraction.value.Events {
			items = append(items, domain.NewCaptureItem(event, domain.MatchCaptureAccount(event.AccountHint, event.SourceApp, data.accounts, data.rules)))
		}
	}

	report(port.CaptureProgress{Stage: "match", Status: "running", Events: len(items)})
	items = domain.ConsolidateCaptureItems(items)
	result.Suggestions = domain.BuildCaptureSuggestions(domain.CaptureBuildInput{Items: items, Materials: request.Materials, Accounts: data.accounts, TransferCategory: data.transfer})
	report(port.CaptureProgress{Stage: "categorize", Status: "running"})
	s.categorizeCapture(ctx, p, result.Suggestions, data)
	domain.ApplyCaptureDuplicates(result.Suggestions, data.transactions)
	result.Balances = domain.CaptureBalances(items, data.accounts)
	return result, nil
}

// serialised wraps a progress callback so concurrent extractions can call it.
func serialised(progress func(port.CaptureProgress)) func(port.CaptureProgress) {
	var mu sync.Mutex
	return func(event port.CaptureProgress) {
		if progress == nil {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		progress(event)
	}
}

func (s *Service) loadCaptureData() (captureData, error) {
	accounts, err := s.db.ListAccounts(false)
	if err != nil {
		return captureData{}, fmt.Errorf("load accounts: %w", err)
	}
	categories, err := s.db.ListCategories(false)
	if err != nil {
		return captureData{}, fmt.Errorf("load categories: %w", err)
	}
	transactions, err := s.db.ListTransactions(false)
	if err != nil {
		return captureData{}, fmt.Errorf("load transactions: %w", err)
	}
	rules, err := s.db.ListCaptureRules(false)
	if err != nil {
		return captureData{}, fmt.Errorf("load capture rules: %w", err)
	}
	data := captureData{accounts: accounts, transactions: transactions, rules: rules, tagOptions: map[string][]string{}}
	for _, category := range categories {
		data.categories = append(data.categories, category.Name)
	}
	data.transfer, _ = domain.TransferCategory(categories)
	if s.analytics != nil {
		reports, err := s.analytics.CardReports(false)
		if err != nil {
			return captureData{}, fmt.Errorf("load card programs: %w", err)
		}
		for _, bank := range reports {
			for _, group := range bank.Accounts {
				for _, card := range group.Cards {
					for _, rule := range card.Rules {
						data.tagOptions[card.AccountName] = appendUnique(data.tagOptions[card.AccountName], rule.Tag)
					}
				}
			}
		}
	}
	return data, nil
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func captureAccountList(accounts []domain.Account) string {
	lines := []string{}
	for _, account := range accounts {
		if account.Type == domain.AccountTypeDeprecated {
			continue
		}
		lines = append(lines, fmt.Sprintf("- %s (%s, %s)", account.Name, account.Type, account.Currency))
	}
	return strings.Join(lines, "\n")
}

func captureAliasList(rules []domain.CaptureRule) string {
	lines := []string{}
	for _, rule := range rules {
		if rule.Kind == domain.CaptureRuleAlias {
			lines = append(lines, fmt.Sprintf("- %q is %s", rule.Pattern, rule.Value))
		}
	}
	if len(lines) == 0 {
		return "(none yet)"
	}
	return strings.Join(lines, "\n")
}
