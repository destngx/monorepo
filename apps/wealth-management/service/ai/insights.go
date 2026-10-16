package ai

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"fmt"
	"strings"
)

// HealthCommentary explains the deterministic health score (WM-AI-14, WM-HEA-1).
func (s *Service) HealthCommentary(ctx context.Context, force bool, model string) (domain.Insight, error) {
	snap, err := s.loadSnapshot()
	if err != nil {
		return domain.Insight{}, err
	}
	health, err := s.analytics.FinancialHealth(false)
	if err != nil {
		return domain.Insight{}, fmt.Errorf("load financial health: %w", err)
	}
	dashboard, err := s.analytics.DashboardSummary(false)
	if err != nil {
		return domain.Insight{}, fmt.Errorf("load dashboard: %w", err)
	}
	return s.structuredInsight(ctx, snap, structuredRequest{
		feature: "financial-health",
		taskKey: keyHealthTask,
		taskVars: map[string]string{
			"netWorth":         amount(snap.sheet.NetWorth),
			"totalAssets":      amount(snap.sheet.Assets),
			"totalLiabilities": amount(snap.sheet.Liabilities),
			"moneyLent":        toJSON(s.moneyLentContext(snap.lent)),
			"months":           toJSON(dashboard.CashFlowHistory),
		},
		actionKey: "actions/financial-health",
		facts: map[string]any{
			"score": health.Score, "band": health.Band, "components": health.Components,
			"liquidAssets": health.LiquidAssets, "avgMonthlyIncome": health.AvgMonthlyIncome, "avgMonthlyExpense": health.AvgMonthlyExpense,
			"freedomMonths": health.FreedomMonths, "allocation": health.Allocation,
			"monthlyDebtPayments": health.MonthlyDebtPayments, "moneyLent": snap.lent.MoneyLentFacts,
		},
		model: model,
		force: force,
	})
}

// ChartInsight comments on the data behind a chart (WM-UI-8); markdown.
func (s *Service) ChartInsight(ctx context.Context, request port.ChartInsightRequest) (string, error) {
	if strings.TrimSpace(request.ChartType) == "" || request.Data == nil {
		return "", fmt.Errorf("%w: chartType and data are required", domain.ErrValidation)
	}
	data := toJSON(request.Data)
	if len(data) > maxChartDataBytes {
		return "", fmt.Errorf("%w: chart data is larger than %d bytes", domain.ErrValidation, maxChartDataBytes)
	}
	if err := s.requireConfigured(); err != nil {
		return "", err
	}
	snap, err := s.loadSnapshot()
	if err != nil {
		return "", err
	}
	p, err := s.loadPrompts()
	if err != nil {
		return "", err
	}
	market := firstNonEmpty(request.Title, "General")
	task, err := p.render(keyChartTask, map[string]string{
		"chartType":        request.ChartType,
		"market":           market,
		"timeframe":        firstNonEmpty(request.Timeframe, "N/A"),
		"chartDescription": s.masker.Mask(firstNonEmpty(request.Description, "N/A")),
		"chartData":        s.masker.Mask(data),
	})
	if err != nil {
		return "", err
	}
	action, err := p.render("actions/chart-insight", map[string]string{"chartType": request.ChartType, "market": firstNonEmpty(request.Title, "the current view")})
	if err != nil {
		return "", err
	}
	system, err := s.systemPrompt(p, snap, task)
	if err != nil {
		return "", err
	}
	messages := []domain.AIMessage{{Role: domain.AIRoleSystem, Content: system}, {Role: domain.AIRoleUser, Content: action + "\n\nAnswer in concise markdown."}}
	model := s.models(domain.AIUseCaseInsight, request.Model)[0]
	return cached(ctx, s, "chart-insight", request.Force, []any{model, messages}, func() (string, error) {
		response, err := s.complete(ctx, call{feature: "chart-insight", useCase: domain.AIUseCaseInsight, model: request.Model, prompts: p.used, messages: messages, validate: nonEmpty})
		return strings.TrimSpace(response.Content), err
	})
}
