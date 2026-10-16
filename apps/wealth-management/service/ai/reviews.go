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

	case port.ReviewYear:
		vars, facts, err := s.yearReviewInputs(request.Year)
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

	// closedStat is a closed period: its estimate next to the refund actually paid for it.
	type closedStat struct {
		Start            string  `json:"start"`
		End              string  `json:"end"`
		Spent            float64 `json:"spent"`
		CashbackEstimate float64 `json:"cashbackEstimate"`
		ActualRefund     float64 `json:"actualRefund"`
		Reconciliation   string  `json:"reconciliation"`
		Missed           float64 `json:"missedCashback,omitempty"`
	}
	closedPeriod := func(history domain.CardPeriodHistory) *closedStat {
		return &closedStat{
			Start: history.Start, End: history.End, Spent: history.Spent, CashbackEstimate: history.Cashback.Total,
			ActualRefund: history.Refund, Reconciliation: string(history.Status), Missed: history.Missed,
		}
	}
	// currentStat is the open period: its cashback is only an estimate (the
	// part on confirmed MCCs is the surer share); no refund is paid for it yet.
	type currentStat struct {
		Start               string             `json:"start"`
		End                 string             `json:"end"`
		Spent               float64            `json:"spent"`
		CashbackEstimate    float64            `json:"cashbackEstimate"`
		CashbackOnConfirmed float64            `json:"cashbackEstimateOnConfirmedMcc"`
		UnconfirmedPayments int                `json:"unconfirmedMccPayments,omitempty"`
		CapLeft             float64            `json:"capLeft,omitempty"`
		RuleCapsLeft        map[string]float64 `json:"ruleCapsLeft,omitempty"`
		NextTierSpendNeeded float64            `json:"nextTierSpendNeeded,omitempty"`
	}
	type cardStat struct {
		Card              string       `json:"card"`
		Bank              string       `json:"bank"`
		Program           string       `json:"program"`
		BankLimit         float64      `json:"bankLimit"`
		BankUtilization   float64      `json:"bankUtilizationPercent"`
		ClosedFrom        string       `json:"closedFrom,omitempty"`
		CurrentPeriod     *currentStat `json:"currentPeriod,omitempty"`
		PreviousPeriod    *closedStat  `json:"previousPeriod,omitempty"`
		ClosedPeriod      *closedStat  `json:"closedPeriod,omitempty"`
		StatementEstimate float64      `json:"statementEstimate,omitempty"`
		PayBy             string       `json:"payBy,omitempty"`
		DaysUntilPayBy    int          `json:"daysUntilPayBy,omitempty"`
		Rules             []string     `json:"cashbackRules"`
	}
	stats := []cardStat{}
	accountNames := map[string]bool{}
	for _, bank := range reports {
		for _, card := range bank.Cards {
			if request.Card != "" && !strings.EqualFold(card.AccountName, request.Card) && !strings.EqualFold(card.Name, request.Card) {
				continue
			}
			accountNames[strings.ToLower(card.AccountName)] = true
			stat := cardStat{Card: card.Name, Bank: bank.Name, Program: card.Program, BankLimit: bank.Limit, BankUtilization: bank.Utilization, ClosedFrom: card.ClosedFrom}
			for _, rule := range card.Rules {
				stat.Rules = append(stat.Rules, rule.Describe())
			}
			if !isCurrent {
				// A past month is reported as that period alone.
				for _, history := range card.History {
					if history.Month == domain.MonthKey(month) {
						stat.ClosedPeriod = closedPeriod(history)
					}
				}
				stats = append(stats, stat)
				continue
			}
			// History is newest first; the first period that is not open is the last one a refund is paid for.
			for _, history := range card.History {
				if history.Status != domain.ReconciliationOpen {
					stat.PreviousPeriod = closedPeriod(history)
					break
				}
			}
			// A closed card has no current period, and its spend is paid with the bank's open cards.
			if card.ClosedFrom == "" {
				standing := card.Standing
				current := &currentStat{
					Start: standing.Start, End: standing.End, Spent: standing.Spent,
					CashbackEstimate: standing.Cashback.Total, CashbackOnConfirmed: standing.Cashback.Confirmed,
					UnconfirmedPayments: standing.UnconfirmedPayments, CapLeft: standing.CapLeft,
				}
				// Rule and cap-group caps bound what the card earns even when it has no card-level cap.
				for _, rule := range standing.Rules {
					if rule.Cap > 0 {
						if current.RuleCapsLeft == nil {
							current.RuleCapsLeft = map[string]float64{}
						}
						current.RuleCapsLeft[rule.Name] = rule.CapLeft
					}
				}
				if standing.NextTier != nil {
					current.NextTierSpendNeeded = standing.NextTier.SpendNeeded
				}
				stat.CurrentPeriod = current
				stat.PayBy, stat.DaysUntilPayBy = card.Cycle.PayBy, card.Cycle.DaysUntilPayBy
			}
			if card.Statement != nil {
				stat.StatementEstimate = card.Statement.Amount
			}
			stats = append(stats, stat)
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
