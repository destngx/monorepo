package ai

import (
	"apps/wealth-management/domain"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

// AI form filling (§10.5, D47): the model reads the request into the
// form's fields; domain.ParseFillAnswer checks every value and matches the
// names against the sheet. Nothing is cached or written, and the request
// never reaches the logs (WM-FILL-8).

const (
	keyFormFill = "tasks/form-fill"
	// fillPayeeLimit bounds the payees offered, most used first.
	fillPayeeLimit = 200
)

func (s *Service) Fill(ctx context.Context, request domain.FillRequest) (domain.FillResult, error) {
	form, err := request.Validate(s.captureLimits)
	if err != nil {
		return domain.FillResult{}, err
	}
	if err := s.requireConfigured(); err != nil {
		return domain.FillResult{}, err
	}
	known, err := s.fillContext(ctx, form, request.Current)
	if err != nil {
		return domain.FillResult{}, err
	}
	p, err := s.loadPrompts()
	if err != nil {
		return domain.FillResult{}, err
	}
	guide, _ := p.lookup(keyFormFill, map[string]string{"today": known.Today.Format("2006-01-02")})
	current := s.maskFillCurrent(request.Current)
	system := domain.FillInstruction(form, known, current, guide)

	message := domain.AIMessage{Role: domain.AIRoleUser}
	if text := strings.TrimSpace(request.Text); text != "" {
		message.Content = "The request:\n```\n" + s.masker.Mask(text) + "\n```"
	} else {
		message.Content = "Fill the form from the images."
	}
	// OpenAI's JSON mode needs "JSON" in an input message; the system prompt
	// goes out as instructions, which it does not read.
	message.Content += "\n\nReply with the JSON object described in the instructions."
	for _, image := range request.Images {
		message.Images = append(message.Images, domain.AIImage{MediaType: image.MediaType, Data: image.Data})
	}
	useCase := domain.AIUseCaseSuggest
	if len(message.Images) > 0 {
		useCase = domain.AIUseCaseVision
	}
	var result domain.FillResult
	_, err = s.complete(ctx, call{
		feature:  "form-fill:" + form.Key,
		useCase:  useCase,
		model:    request.Model,
		prompts:  p.used,
		messages: []domain.AIMessage{{Role: domain.AIRoleSystem, Content: system}, message},
		json:     true,
		validate: func(content string) error {
			var err error
			result, err = domain.ParseFillAnswer(content, form, known, request.Current)
			return err
		},
	})
	if err != nil {
		return domain.FillResult{}, err
	}
	return result, nil
}

// maskFillCurrent masks the text the form holds, as the request is masked.
func (s *Service) maskFillCurrent(current map[string]any) map[string]any {
	if len(current) == 0 {
		return nil
	}
	masked := make(map[string]any, len(current))
	for key, value := range current {
		if text, ok := value.(string); ok {
			value = s.masker.Mask(text)
		}
		masked[key] = value
	}
	return masked
}

// fillContext reads the lists the form's fields match against and the
// facts the model needs for the form.
func (s *Service) fillContext(ctx context.Context, form domain.FillForm, current map[string]any) (domain.FillContext, error) {
	known := domain.FillContext{Today: s.today(), Lists: map[domain.FillList][]string{}}
	var facts []string
	needs := map[domain.FillList]bool{}
	for _, list := range form.Lists() {
		needs[list] = true
	}

	if needs[domain.FillAccounts] {
		accounts, err := s.db.ListAccounts(false)
		if err != nil {
			return known, fmt.Errorf("read accounts: %w", err)
		}
		for _, account := range accounts {
			if account.Type == domain.AccountTypeDeprecated || strings.HasPrefix(account.Name, domain.DeletedAccountPrefix) {
				continue
			}
			known.Lists[domain.FillAccounts] = append(known.Lists[domain.FillAccounts], account.Name)
		}
	}
	if needs[domain.FillCategories] {
		categories, err := s.db.ListCategories(false)
		if err != nil {
			return known, fmt.Errorf("read categories: %w", err)
		}
		for _, category := range categories {
			known.Lists[domain.FillCategories] = append(known.Lists[domain.FillCategories], category.Name)
		}
	}
	if needs[domain.FillTags] {
		tags, err := s.db.ListTags(false)
		if err != nil {
			return known, fmt.Errorf("read tags: %w", err)
		}
		known.Lists[domain.FillTags] = tags
	}
	if needs[domain.FillPayees] {
		transactions, err := s.db.ListTransactions(false)
		if err != nil {
			return known, fmt.Errorf("read transactions: %w", err)
		}
		known.Lists[domain.FillPayees] = frequentPayees(transactions, fillPayeeLimit)
	}
	if needs[domain.FillCards] || needs[domain.FillPrograms] || needs[domain.FillBanks] {
		config, err := s.db.ListCardConfig(false)
		if err != nil {
			return known, fmt.Errorf("read card settings: %w", err)
		}
		for _, card := range config.Cards {
			known.Lists[domain.FillCards] = append(known.Lists[domain.FillCards], card.Name)
		}
		for _, program := range config.Programs {
			known.Lists[domain.FillPrograms] = append(known.Lists[domain.FillPrograms], program.Name)
		}
		for _, bank := range config.Banks {
			known.Lists[domain.FillBanks] = append(known.Lists[domain.FillBanks], bank.Name)
		}
		facts = append(facts, cardFacts(form.Key, config)...)
	}
	if needs[domain.FillFunds] || needs[domain.FillPlatforms] {
		assets, err := s.db.GetInvestmentAssets(false)
		if err != nil {
			return known, fmt.Errorf("read holdings: %w", err)
		}
		facts = append(facts, holdingFacts(form.Key, assets, known.Lists)...)
	}
	if needs[domain.FillProducts] {
		facts = append(facts, s.productFacts(ctx, known.Lists)...)
	}

	switch form.Key {
	case "goal":
		plan, err := s.db.ReadGoalPlan(false)
		if err != nil {
			return known, fmt.Errorf("read goals: %w", err)
		}
		facts = append(facts, goalFacts(plan.Goals)...)
	case "budget-limits", "budget-move":
		// The page's month, when the form sends one, else this month.
		month := known.Today
		if text, ok := current["month"].(string); ok {
			if parsed, err := time.Parse("2006-01", text); err == nil {
				month = parsed
			}
		}
		items, err := s.db.ListBudget(month.Year(), false)
		if err != nil {
			return known, fmt.Errorf("read the budget: %w", err)
		}
		facts = append(facts, budgetFacts(items, month)...)
	case "recurring":
		sheet, err := s.db.ReadRecurring(false)
		if err != nil {
			return known, fmt.Errorf("read recurring items: %w", err)
		}
		if len(sheet.Items) > 0 {
			names := make([]string, len(sheet.Items))
			for index, item := range sheet.Items {
				names[index] = item.Name
			}
			facts = append(facts, "Recurring items already saved: "+strings.Join(names, ", ")+".")
		}
	}
	known.Facts = strings.Join(facts, "\n")
	return known, nil
}

// frequentPayees lists payees most used first, then by name.
func frequentPayees(transactions []domain.Transaction, limit int) []string {
	counts := map[string]int{}
	for _, transaction := range transactions {
		if payee := strings.TrimSpace(transaction.Payee); payee != "" {
			counts[payee]++
		}
	}
	payees := make([]string, 0, len(counts))
	for payee := range counts {
		payees = append(payees, payee)
	}
	sort.Slice(payees, func(i, j int) bool {
		if counts[payees[i]] != counts[payees[j]] {
			return counts[payees[i]] > counts[payees[j]]
		}
		return payees[i] < payees[j]
	})
	if len(payees) > limit {
		payees = payees[:limit]
	}
	return payees
}

func goalFacts(goals []domain.PlannedGoal) []string {
	if len(goals) == 0 {
		return []string{"There are no goals yet."}
	}
	sorted := slices.Clone(goals)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Priority < sorted[j].Priority })
	lines := []string{"Goals by priority:"}
	for _, goal := range sorted {
		line := fmt.Sprintf("- priority %d: %s (%s", goal.Priority, goal.Name, goal.Status)
		if goal.Target != nil {
			line += ", target " + strconv.FormatFloat(*goal.Target, 'f', -1, 64)
		}
		if goal.TargetDate != "" {
			line += ", by " + goal.TargetDate
		}
		if domain.IsEmergencyFund(goal.Name, goal.Note) {
			line += ", the emergency fund"
		}
		if goal.UseSlowMoney {
			line += ", can use slow money"
		}
		lines = append(lines, line+")")
	}
	return lines
}

func budgetFacts(items []domain.BudgetItem, month time.Time) []string {
	lines := []string{"The limits for " + month.Format("2006-01") + ":"}
	for _, item := range items {
		lines = append(lines, fmt.Sprintf("- %s: %s", item.Category, strconv.FormatFloat(item.LimitFor(month), 'f', -1, 64)))
	}
	return lines
}

func cardFacts(form string, config domain.CardConfig) []string {
	var lines []string
	switch form {
	case "card":
		for _, card := range config.Cards {
			lines = append(lines, fmt.Sprintf("- card %s: programme %s, statement day %d, due day %d", card.Name, card.Program, card.StatementDay, card.DueDay))
		}
	case "cashback-program":
		for _, program := range config.Programs {
			line := fmt.Sprintf("- programme %s of %s", program.Name, program.Bank)
			if n := len(program.Versions); n > 0 {
				line += ", latest terms from " + program.Versions[n-1].From.Format("2006-01-02")
			}
			lines = append(lines, line)
		}
	}
	return lines
}

func holdingFacts(form string, assets domain.InvestmentAssets, lists map[domain.FillList][]string) []string {
	var lines []string
	switch form {
	case "fund-trade":
		for _, holding := range assets.Funds.Items {
			if holding.Symbol == "" || slices.Contains(lists[domain.FillFunds], holding.Symbol) {
				continue
			}
			lists[domain.FillFunds] = append(lists[domain.FillFunds], holding.Symbol)
			if holding.Quantity != nil {
				lines = append(lines, fmt.Sprintf("- %s: %s units held", holding.Symbol, strconv.FormatFloat(*holding.Quantity, 'f', -1, 64)))
			}
		}
	case "crypto-holdings":
		for _, holding := range assets.Crypto.Items {
			if holding.Platform != "" && !slices.Contains(lists[domain.FillPlatforms], holding.Platform) {
				lists[domain.FillPlatforms] = append(lists[domain.FillPlatforms], holding.Platform)
			}
			if holding.Symbol != "" && holding.Quantity != nil {
				lines = append(lines, fmt.Sprintf("- %s on %s: %s held", holding.Symbol, holding.Platform, strconv.FormatFloat(*holding.Quantity, 'f', -1, 64)))
			}
		}
	}
	return lines
}

// productFacts lists the flexible savings products (D41). They come from
// the market data server; without it the product field stays blank.
func (s *Service) productFacts(ctx context.Context, lists map[domain.FillList][]string) []string {
	if s.rates == nil {
		return []string{"The savings product list is not available."}
	}
	savings, err := s.rates.GetFlexibleSavings(ctx)
	if err != nil {
		s.log.LogError(ctx, "form fill: savings products unavailable", err, slog.String("feature", "form-fill"))
		return []string{"The savings product list is not available."}
	}
	lines := []string{"Savings products (id: name, rate):"}
	for _, product := range savings.Products {
		lists[domain.FillProducts] = append(lists[domain.FillProducts], product.ID)
		lines = append(lines, fmt.Sprintf("- %s: %s %s, %s%%", product.ID, product.Provider, product.Product, strconv.FormatFloat(product.Rate, 'f', -1, 64)))
	}
	return lines
}
