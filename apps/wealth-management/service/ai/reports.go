package ai

import (
	"apps/wealth-management/domain"
	"context"
	"fmt"
)

// Briefing is the dashboard intelligence briefing (WM-AI-6, WM-DASH-6).
func (s *Service) Briefing(ctx context.Context, force bool, model string) (domain.Briefing, error) {
	snap, err := s.loadSnapshot()
	if err != nil {
		return domain.Briefing{}, err
	}
	facts := domain.ComputeBriefingFacts(snap.accounts, snap.transactions, snap.today)
	dashboard, err := s.analytics.DashboardSummary(false)
	if err != nil {
		return domain.Briefing{}, fmt.Errorf("load dashboard: %w", err)
	}
	facts = facts.WithMonthComparison(dashboard)
	budget := []map[string]any{}
	for _, line := range snap.budget.Lines {
		if line.CountsAsExpense() && (line.Spent > 0 || line.Limit > 0) {
			budget = append(budget, map[string]any{"category": line.Category, "spent": line.Spent, "limit": line.Limit, "status": line.Status})
		}
	}
	monthStart := snap.today.AddDate(0, 0, 1-snap.today.Day())
	vars := map[string]string{
		"netWorth":         amount(facts.NetWorth),
		"totalAssets":      amount(facts.Assets),
		"totalLiabilities": amount(facts.Liabilities),
		"cashFlow":         amount(facts.CashFlow),
		"savingsRate":      fmt.Sprintf("%.1f", facts.SavingsRate),
		"accountsJson":     toJSON(s.accountContexts(snap.accounts)),
		"transactionsJson": toJSON(s.txContexts(recent(snap.transactions, monthStart.AddDate(0, -1, 0), snap.today, briefingTransactions))),
		"budgetJson":       toJSON(budget),
		"moneyLentJson":    toJSON(s.moneyLentContext(snap.lent)),
	}
	briefing, err := structured(ctx, s, snap, structuredRequest{
		feature:   "intelligence-briefing",
		taskKey:   "tasks/intelligence-briefing",
		taskVars:  vars,
		actionKey: "actions/intelligence-briefing",
		facts:     facts,
		model:     model,
		force:     force,
	}, domain.BriefingFormatInstruction, func(content string) (domain.Briefing, error) {
		var briefing domain.Briefing
		if err := domain.ParseAIJSON(content, &briefing); err != nil {
			return briefing, err
		}
		return briefing, briefing.Normalize()
	})
	if err != nil {
		return domain.Briefing{}, err
	}
	briefing.Facts = facts
	return briefing, nil
}

// BudgetAdvice is the budget advisor (WM-BUD-5): the model suggests limits
// and alerts; projections, averages and velocity are engine facts.
func (s *Service) BudgetAdvice(ctx context.Context, force bool, model string) (domain.BudgetAdvice, error) {
	snap, err := s.loadSnapshot()
	if err != nil {
		return domain.BudgetAdvice{}, err
	}
	items, err := s.db.ListBudget(snap.today.Year(), false)
	if err != nil {
		return domain.BudgetAdvice{}, fmt.Errorf("load budget: %w", err)
	}
	goals, err := s.db.ListGoals(false)
	if err != nil {
		return domain.BudgetAdvice{}, fmt.Errorf("load goals: %w", err)
	}
	facts := domain.ComputeAdvisorFacts(items, snap.accounts, snap.transactions, snap.today)
	monthStart := snap.today.AddDate(0, 0, 1-snap.today.Day())
	vars := map[string]string{
		"date":       snap.today.Format("02/01/2006"),
		"budget":     toJSON(facts.Categories),
		"goals":      toJSON(s.goalContexts(goals)),
		"recentTxns": toJSON(s.txContexts(recent(snap.transactions, monthStart, snap.today, advisorTransactions))),
	}
	return structured(ctx, s, snap, structuredRequest{
		feature:   "budget-advisor",
		taskKey:   keyAdvisorTask,
		taskVars:  vars,
		actionKey: "actions/budget-advisor",
		facts:     facts,
		model:     model,
		force:     force,
	}, domain.AdvisorFormatInstruction, func(content string) (domain.BudgetAdvice, error) {
		var advice domain.BudgetAdvice
		if err := domain.ParseAIJSON(content, &advice); err != nil {
			return advice, err
		}
		return advice, advice.ApplyFacts(facts)
	})
}
