package ai

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
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

type extractionOutcome struct {
	value domain.CaptureExtraction
	err   error
}

// reason is what the review screen shows for a material without events.
func (o extractionOutcome) reason() string {
	switch {
	case o.err == nil:
		return firstNonBlank(o.value.Reason, "Nothing readable.")
	case errors.Is(o.err, domain.ErrAIOutput):
		return "The model's answer could not be read. Try again or use another model."
	case IsUserFacing(o.err):
		return o.err.Error()
	default:
		return "Extraction failed. Try again."
	}
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// extractAll reads every material, a few at a time. A material that fails
// is reported as unreadable; when all fail for a reason other than the
// model's output, the first error is returned instead.
func (s *Service) extractAll(ctx context.Context, request port.CaptureRequest, system string, promptKeys []string, report func(port.CaptureProgress)) ([]extractionOutcome, error) {
	outcomes := make([]extractionOutcome, len(request.Materials))
	slots := make(chan struct{}, captureConcurrency)
	var wg sync.WaitGroup
	for index, material := range request.Materials {
		wg.Add(1)
		go func() {
			defer wg.Done()
			slots <- struct{}{}
			defer func() { <-slots }()
			report(port.CaptureProgress{Stage: "extract", MaterialID: material.ID, Status: "running"})
			value, err := s.extract(ctx, material, system, promptKeys, request.Model)
			outcomes[index] = extractionOutcome{value: value, err: err}
			event := port.CaptureProgress{Stage: "extract", MaterialID: material.ID, Status: "ok", Events: len(value.Events)}
			switch {
			case err != nil:
				event.Status, event.Message = "error", outcomes[index].reason()
				s.log.LogError(ctx, "capture extraction failed", err, slog.String("material_kind", string(material.Kind)))
			case value.Unreadable:
				event.Status, event.Message = "unreadable", outcomes[index].reason()
			}
			report(event)
		}()
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var firstErr error
	for _, outcome := range outcomes {
		if outcome.err == nil || errors.Is(outcome.err, domain.ErrAIOutput) {
			return outcomes, nil
		}
		if firstErr == nil {
			firstErr = outcome.err
		}
	}
	return nil, firstErr
}

func (s *Service) extract(ctx context.Context, material domain.CaptureMaterial, system string, promptKeys []string, model string) (domain.CaptureExtraction, error) {
	user := fmt.Sprintf("Material %s (%s", material.ID, material.Label())
	if name := strings.TrimSpace(material.Name); name != "" {
		user += ", " + s.masker.Mask(name)
	}
	user += ").\n"
	message := domain.AIMessage{Role: domain.AIRoleUser}
	if material.Kind == domain.CaptureImage {
		message.Images = []domain.AIImage{{MediaType: material.MediaType, Data: material.Data}}
		user += "Read every money movement shown in the image."
	} else {
		user += "Read every money movement in this text:\n```\n" + s.masker.Mask(material.Text) + "\n```"
	}
	message.Content = user + "\n\n" + domain.CaptureFormatInstruction
	var extraction domain.CaptureExtraction
	_, err := s.complete(ctx, call{
		feature:  "capture-extract",
		useCase:  domain.AIUseCaseVision,
		model:    model,
		prompts:  promptKeys,
		messages: []domain.AIMessage{{Role: domain.AIRoleSystem, Content: system}, message},
		json:     true,
		validate: func(content string) error {
			var err error
			extraction, err = domain.ParseCaptureExtraction(content, material.ID)
			return err
		},
	})
	return extraction, err
}

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
		if tag, ok := historyTag(payee, suggestion.Transaction.AccountName, suggestion.TagOptions, data.transactions); ok {
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
- tag must be copied exactly from that item's tagOptions (the card's cashback rule that the merchant falls under); use "" when the item has no tagOptions.`

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
