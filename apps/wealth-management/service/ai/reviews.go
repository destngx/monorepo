package ai

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	budgetReviewTransactions = 30
	cardSummaryTransactions  = 100
	weeklyReviewDays         = 7
	briefingTransactions     = 10
	advisorTransactions      = 50
	maxChartDataBytes        = 12_000
)

// structuredRequest is a prompt pair answered with a structured insight.
type structuredRequest struct {
	feature    string
	taskKey    string
	taskVars   map[string]string
	actionKey  string
	actionVars map[string]string
	facts      any
	model      string
	force      bool
}

func (s *Service) structuredInsight(ctx context.Context, snap snapshot, request structuredRequest) (domain.Insight, error) {
	return structured(ctx, s, snap, request, domain.InsightFormatInstruction, func(content string) (domain.Insight, error) {
		var insight domain.Insight
		if err := domain.ParseAIJSON(content, &insight); err != nil {
			return insight, err
		}
		return insight, insight.Normalize()
	})
}

// structured renders task and action prompts, appends the engine facts and
// the output contract, and parses the JSON reply (WM-AI-3).
func structured[T any](ctx context.Context, s *Service, snap snapshot, request structuredRequest, format string, parse func(content string) (T, error)) (T, error) {
	var zero T
	if err := s.requireConfigured(); err != nil {
		return zero, err
	}
	p, err := s.loadPrompts()
	if err != nil {
		return zero, err
	}
	task, err := p.render(request.taskKey, request.taskVars)
	if err != nil {
		return zero, err
	}
	action, err := p.render(request.actionKey, request.actionVars)
	if err != nil {
		return zero, err
	}
	system, err := s.systemPrompt(p, snap, task)
	if err != nil {
		return zero, err
	}
	user := action
	if request.facts != nil {
		user += factsBlock(request.facts)
	}
	user += "\n\n" + format
	messages := []domain.AIMessage{{Role: domain.AIRoleSystem, Content: system}, {Role: domain.AIRoleUser, Content: user}}
	model := s.models(domain.AIUseCaseInsight, request.model)[0]

	return cached(ctx, s, request.feature, request.force, []any{model, messages}, func() (T, error) {
		var result T
		_, err := s.complete(ctx, call{
			feature:  request.feature,
			useCase:  domain.AIUseCaseInsight,
			model:    request.model,
			prompts:  p.used,
			messages: messages,
			json:     true,
			validate: func(content string) error {
				var err error
				result, err = parse(content)
				return err
			},
		})
		return result, err
	})
}

// Review runs one of the structured reviews (WM-AI-5).
func (s *Service) Review(ctx context.Context, request port.ReviewRequest) (domain.Insight, error) {
	snap, err := s.loadSnapshot()
	if err != nil {
		return domain.Insight{}, err
	}
	structured := structuredRequest{feature: string(request.Kind), taskKey: "tasks/" + string(request.Kind), actionKey: "actions/" + string(request.Kind), model: request.Model, force: request.Force}

	switch request.Kind {
	case port.ReviewAccounts:
		structured.taskVars = map[string]string{
			"totalAssets":      amount(snap.sheet.Assets),
			"totalLiabilities": amount(snap.sheet.Liabilities),
			"totalNetWorth":    amount(snap.sheet.NetWorth),
			"accountsJson":     toJSON(s.accountContexts(snap.accounts)),
		}
		structured.facts = snap.sheet

	case port.ReviewBudget:
		report, err := s.analytics.BudgetReport(request.Month, false)
		if err != nil {
			return domain.Insight{}, fmt.Errorf("load budget report: %w", err)
		}
		month, _ := domain.ParseMonthKey(report.Month)
		monthEnd := month.AddDate(0, 1, -1)
		active := []map[string]any{}
		for _, line := range report.Lines {
			if line.CategoryType == domain.CategoryTypeExpense && (line.Spent > 0 || line.Limit > 0) {
				active = append(active, map[string]any{"category": line.Category, "spent": line.Spent, "limit": line.Limit, "remaining": line.Remaining, "percentUsed": line.PercentUsed, "status": line.Status})
			}
		}
		date := month.Format("01/2006")
		structured.taskVars = map[string]string{
			"view":                   "month",
			"date":                   date,
			"totalSpent":             amount(report.TotalSpent),
			"totalLimit":             amount(report.TotalLimit),
			"usagePercentage":        fmt.Sprintf("%.0f", report.PercentUsed),
			"activeBudgetsJson":      toJSON(active),
			"recentTransactionsJson": toJSON(s.txContexts(recent(snap.transactions, month, monthEnd, budgetReviewTransactions))),
		}
		structured.actionVars = map[string]string{"view": "month", "date": date}
		structured.facts = map[string]any{"month": report.Month, "daysElapsed": report.DaysElapsed, "daysInMonth": report.DaysInMonth, "dailyBurn": report.DailyBurn, "projectedSpend": report.ProjectedSpend, "status": report.Status}

	case port.ReviewLoans:
		structured.taskVars = map[string]string{
			"totalDebt": amount(snap.sheet.LoansRemaining),
			"loansJson": toJSON(s.loanContexts(snap.loans)),
		}

	case port.ReviewTransactions:
		from := snap.today.AddDate(0, 0, -(weeklyReviewDays - 1))
		week := recent(snap.transactions, from, snap.today, 0)
		structured.taskVars = map[string]string{"transactionsJson": toJSON(s.txContexts(week))}
		structured.facts = weekTotals(week, from, snap.today)

	case port.ReviewCreditCard:
		vars, facts, err := s.cardSummaryInputs(snap, request)
		if err != nil {
			return domain.Insight{}, err
		}
		structured.taskVars, structured.actionVars, structured.facts = vars, vars, facts

	default:
		return domain.Insight{}, fmt.Errorf("%w: unknown review %q", domain.ErrValidation, request.Kind)
	}
	return s.structuredInsight(ctx, snap, structured)
}

func weekTotals(transactions []domain.Transaction, from time.Time, to time.Time) map[string]any {
	var spent, received float64
	byCategory := map[string]float64{}
	for _, t := range transactions {
		if t.HasCategoryType(domain.CategoryTypeExpense) {
			spent += t.PaymentAmount()
			byCategory[t.Category] += t.PaymentAmount()
		}
		if t.HasCategoryType(domain.CategoryTypeIncome) {
			received += t.DepositAmount()
		}
	}
	return map[string]any{"from": from.Format(time.DateOnly), "to": to.Format(time.DateOnly), "count": len(transactions), "expense": spent, "income": received, "expenseByCategory": byCategory}
}

// cardSummaryInputs builds the credit-card summary for a month (WM-CARD-5):
// the current cycle when no month or the current month is asked, otherwise
// that calendar month's card spending.
func (s *Service) cardSummaryInputs(snap snapshot, request port.ReviewRequest) (map[string]string, any, error) {
	reports, err := s.analytics.CardReports(false)
	if err != nil {
		return nil, nil, fmt.Errorf("load card reports: %w", err)
	}
	month := time.Date(snap.today.Year(), snap.today.Month(), 1, 0, 0, 0, 0, time.UTC)
	if request.Month != "" {
		if month, err = domain.ParseMonthKey(request.Month); err != nil {
			return nil, nil, fmt.Errorf("%w: %v", domain.ErrValidation, err)
		}
	}
	isCurrent := domain.SameMonth(month, snap.today)
	monthLabel := "for the current month"
	if !isCurrent {
		monthLabel = "for the month of " + month.Format("01/2006")
	}

	type cardStat struct {
		Card              string   `json:"card"`
		Bank              string   `json:"bank"`
		Limit             float64  `json:"limit"`
		Utilization       float64  `json:"utilizationPercent"`
		Spent             float64  `json:"spent"`
		EstimatedCashback float64  `json:"estimatedCashback"`
		ActualRefund      float64  `json:"actualRefund,omitempty"`
		Missed            float64  `json:"missedCashback,omitempty"`
		DaysUntilDue      int      `json:"daysUntilDue,omitempty"`
		Rules             []string `json:"cashbackRules"`
	}
	stats := []cardStat{}
	accountNames := map[string]bool{}
	for _, bank := range reports {
		for _, group := range bank.Accounts {
			for _, card := range group.Cards {
				if request.Card != "" && !strings.EqualFold(card.AccountName, request.Card) && !strings.EqualFold(card.Name, request.Card) {
					continue
				}
				accountNames[strings.ToLower(card.AccountName)] = true
				stat := cardStat{Card: card.Name, Bank: bank.Name, Limit: card.Limit, Utilization: card.Utilization, Spent: card.SpentThisCycle, EstimatedCashback: card.EstimatedCashback, DaysUntilDue: card.Cycle.DaysUntilDue}
				for _, rule := range card.Rules {
					stat.Rules = append(stat.Rules, fmt.Sprintf("%s: %.2f%% on tag %q", rule.Name, rule.Rate, rule.Tag))
				}
				if !isCurrent {
					stat.Spent, stat.EstimatedCashback, stat.DaysUntilDue = 0, 0, 0
					for _, history := range card.History {
						if history.Month == domain.MonthKey(month) {
							stat.Spent, stat.EstimatedCashback, stat.ActualRefund, stat.Missed = history.Spent, history.PotentialCashback, history.ActualRefund, history.Missed
						}
					}
				}
				stats = append(stats, stat)
			}
		}
	}
	if len(stats) == 0 {
		return nil, nil, fmt.Errorf("%w: no credit card matches %q", domain.ErrNotFound, request.Card)
	}

	cardTransactions := []domain.Transaction{}
	for _, t := range recent(snap.transactions, month, month.AddDate(0, 1, -1), 0) {
		if accountNames[strings.ToLower(t.AccountName)] {
			cardTransactions = append(cardTransactions, t)
		}
	}
	if len(cardTransactions) > cardSummaryTransactions {
		cardTransactions = cardTransactions[:cardSummaryTransactions]
	}
	vars := map[string]string{
		"monthLabel":             monthLabel,
		"cardStatsJson":          toJSON(stats),
		"recentTransactionsJson": toJSON(s.txContexts(cardTransactions)),
	}
	return vars, stats, nil
}

func nonEmpty(content string) error {
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("%w: empty reply", domain.ErrAIOutput)
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func sortNewestFirst(transactions []domain.Transaction) {
	sort.SliceStable(transactions, func(a, b int) bool { return transactions[a].Date.After(transactions[b].Date) })
}
