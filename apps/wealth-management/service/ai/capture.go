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
	accounts   []domain.Account
	categories []string
	transfer   string
	// lending are the lending-nature categories (WM-CAP-22).
	lending      []string
	transactions []domain.Transaction
	rules        []domain.CaptureRule
	tagOptions   map[string][]string
	cards        []domain.CaptureCard
	// splits and guide are the categories' split tags and definitions (WM-CAT-5).
	splits domain.SplitTagIndex
	guide  string
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
		"accounts": captureAccountList(data.accounts, data.cards),
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
	// A typed note that reads as no transaction ("lend") is context for
	// the other materials' rows.
	var notes []string
	for index, extraction := range extractions {
		material := request.Materials[index]
		materialID := material.ID
		if extraction.err != nil || extraction.value.Unreadable {
			result.Unreadable = append(result.Unreadable, domain.CaptureUnreadable{MaterialID: materialID, Reason: extraction.reason()})
		}
		if extraction.err == nil && material.Kind == domain.CaptureText && len(extraction.value.Events) == 0 {
			notes = append(notes, strings.TrimSpace(material.Text))
		}
		for _, event := range extraction.value.Events {
			items = append(items, domain.NewCaptureItem(event, matchCaptureAccount(event, data)))
		}
	}

	report(port.CaptureProgress{Stage: "match", Status: "running", Events: len(items)})
	items = domain.ConsolidateCaptureItems(items)
	result.Suggestions = domain.BuildCaptureSuggestions(domain.CaptureBuildInput{Items: items, Materials: request.Materials, Accounts: data.accounts, TransferCategory: data.transfer})
	domain.ApplyCaptureLending(result.Suggestions, domain.CaptureLendingInput{Transactions: data.transactions, Categories: data.lending, Notes: notes})
	report(port.CaptureProgress{Stage: "categorize", Status: "running"})
	s.categorizeCapture(ctx, p, result.Suggestions, data, notes)
	domain.ApplyCaptureDuplicates(result.Suggestions, data.transactions, s.today())
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
		if category.Nature == domain.NatureLending {
			data.lending = append(data.lending, category.Name)
		}
	}
	data.transfer, _ = domain.TransferCategory(categories)
	data.splits, data.guide = domain.NewSplitTagIndex(categories), domain.CategoryGuide(categories)
	if s.analytics != nil {
		reports, err := s.analytics.CardReports(false)
		if err != nil {
			return captureData{}, fmt.Errorf("load card programs: %w", err)
		}
		// A card's keyword as a tag files a row on a shared account to that
		// card; a closed card takes no new payments.
		for _, bank := range reports {
			for _, card := range bank.Cards {
				if card.ClosedFrom != "" {
					continue
				}
				data.tagOptions[card.AccountName] = appendUnique(data.tagOptions[card.AccountName], card.Keyword)
				data.cards = append(data.cards, domain.CaptureCard{Name: card.Name, Account: card.AccountName, Keyword: card.Keyword, Last4: card.Last4})
			}
		}
	}
	return data, nil
}

// matchCaptureAccount prefers a held card's last four digits to every
// other match but a confirmed alias: they name the card, not just a bank.
func matchCaptureAccount(event domain.CaptureEvent, data captureData) domain.AccountMatch {
	match := domain.MatchCaptureAccount(event.AccountHint, event.SourceApp, data.accounts, data.rules)
	if match.Source == domain.MatchSourceAlias {
		return match
	}
	if card, ok := domain.MatchCaptureCard(event.AccountHint, data.cards); ok {
		if name, ok := accountNamed(card.Account, data.accounts); ok {
			return domain.AccountMatch{Name: name, Source: domain.MatchSourceDigits}
		}
	}
	return match
}

func accountNamed(name string, accounts []domain.Account) (string, bool) {
	for _, account := range accounts {
		if account.Type != domain.AccountTypeDeprecated && strings.EqualFold(account.Name, name) {
			return account.Name, true
		}
	}
	return "", false
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

// captureAccountList lists the accounts for the extract prompt, each with
// the open cards billing to it and their last four digits, so the model
// can copy a card's "*1234" into the account hint.
func captureAccountList(accounts []domain.Account, cards []domain.CaptureCard) string {
	lines := []string{}
	for _, account := range accounts {
		if account.Type == domain.AccountTypeDeprecated {
			continue
		}
		lines = append(lines, fmt.Sprintf("- %s (%s, %s)", account.Name, account.Type, account.Currency))
		for _, card := range cards {
			if !strings.EqualFold(card.Account, account.Name) {
				continue
			}
			if card.Last4 != "" {
				lines = append(lines, fmt.Sprintf("  - card %s *%s", card.Name, card.Last4))
			} else {
				lines = append(lines, fmt.Sprintf("  - card %s", card.Name))
			}
		}
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
