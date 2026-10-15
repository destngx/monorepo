package fakes

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
)

// AI is a port.AIService with canned answers. Err fails every use case;
// ChatErr fails a chat after its first text event.
type AI struct {
	Err     error
	ChatErr error
	Reviews []port.ReviewRequest
	Chats   []port.ChatRequest
	// Captures records Smart Capture requests.
	Captures []port.CaptureRequest
	News     []port.NewsRequest
	Tickers  []port.TickerInsightRequest
}

var _ port.AIService = (*AI)(nil)

func (f *AI) Status(context.Context) domain.AIStatus {
	return domain.AIStatus{Configured: true, MissingKeys: []string{"tasks/loan-review"}}
}

func (f *AI) ProviderUsage(context.Context) (domain.AIProviderUsage, error) {
	return domain.AIProviderUsage{Provider: "openai", Windows: []domain.AIUsageWindow{{Name: "5h", UsedPercent: 12}}}, f.Err
}

func (f *AI) ProviderModels(context.Context) ([]string, error) {
	return []string{"gpt-5", "gpt-5-mini"}, f.Err
}

func (f *AI) SuggestCategory(context.Context, string, string) (port.CategorySuggestion, error) {
	return port.CategorySuggestion{Category: "Coffee", Source: "history"}, f.Err
}

func (f *AI) Review(_ context.Context, request port.ReviewRequest) (domain.Insight, error) {
	f.Reviews = append(f.Reviews, request)
	return domain.Insight{Summary: "ok"}, f.Err
}

func (f *AI) HealthCommentary(context.Context, bool, string) (domain.Insight, error) {
	return domain.Insight{Summary: "healthy"}, f.Err
}

func (f *AI) ChartInsight(context.Context, port.ChartInsightRequest) (string, error) {
	return "**flat**", f.Err
}

func (f *AI) ExplainCashback(context.Context, port.CashbackExplainRequest) (string, error) {
	return "**earned**", f.Err
}

func (f *AI) Briefing(context.Context, bool, string) (domain.Briefing, error) {
	return domain.Briefing{Briefing: "hi"}, f.Err
}

func (f *AI) BudgetAdvice(context.Context, bool, string) (domain.BudgetAdvice, error) {
	return domain.BudgetAdvice{Briefing: "advice"}, f.Err
}

func (f *AI) Chat(_ context.Context, request port.ChatRequest, emit func(port.ChatEvent) error) error {
	f.Chats = append(f.Chats, request)
	if err := emit(port.ChatEvent{Type: "tool", Tool: "getBudgetStatus", Status: "running"}); err != nil {
		return err
	}
	if err := emit(port.ChatEvent{Type: "text", Delta: "Hel"}); err != nil {
		return err
	}
	if f.ChatErr != nil {
		return f.ChatErr
	}
	if err := emit(port.ChatEvent{Type: "text", Delta: "lo"}); err != nil {
		return err
	}
	return emit(port.ChatEvent{Type: "done", Model: "m"})
}

// Capture reports one extraction step and returns one suggestion per
// material; CaptureErr fails it after the progress event.
func (f *AI) Capture(_ context.Context, request port.CaptureRequest, progress func(port.CaptureProgress)) (domain.CaptureResult, error) {
	f.Captures = append(f.Captures, request)
	if err := domain.ValidateCaptureMaterials(request.Materials, domain.DefaultCaptureLimits()); err != nil {
		return domain.CaptureResult{}, err
	}
	if progress != nil {
		progress(port.CaptureProgress{Stage: "extract", MaterialID: request.Materials[0].ID, Status: "ok", Events: 1})
	}
	if f.Err != nil {
		return domain.CaptureResult{}, f.Err
	}
	result := domain.CaptureResult{Suggestions: []domain.CaptureSuggestion{}, Unreadable: []domain.CaptureUnreadable{}, Balances: []domain.CaptureBalance{}}
	for index, material := range request.Materials {
		payment := 1000.0
		result.Suggestions = append(result.Suggestions, domain.CaptureSuggestion{
			ID:          "s" + string(rune('1'+index)),
			Selected:    true,
			Transaction: domain.TransactionCreateInput{AccountName: "Cash", Date: "03/10/2026", Payee: "Shop", Category: "Food", Tags: []string{}, Payment: &payment},
			Direction:   domain.CaptureOut,
			Amount:      payment,
			Currency:    "VND",
			Evidence:    []domain.CaptureEvidence{{MaterialID: material.ID}},
		})
	}
	return result, nil
}

func (f *AI) NewsSentiment(_ context.Context, request port.NewsRequest) (domain.NewsReport, error) {
	f.News = append(f.News, request)
	return domain.NewsReport{Topic: request.Topic, Bias: domain.NewsBullish}, f.Err
}

func (f *AI) TickerInsight(_ context.Context, request port.TickerInsightRequest) (domain.Insight, error) {
	f.Tickers = append(f.Tickers, request)
	return domain.Insight{Summary: request.Symbol}, f.Err
}

// InvestmentAnalysis streams one phase and a command, then fails with Err
// or finishes.
func (f *AI) InvestmentAnalysis(_ context.Context, _ port.InvestmentAnalysisRequest, emit func(port.InvestmentEvent) error) error {
	if err := emit(port.InvestmentEvent{Type: "phase", Phase: "think-tank", Status: "ok"}); err != nil {
		return err
	}
	if f.Err != nil {
		return f.Err
	}
	if err := emit(port.InvestmentEvent{Type: "actions", Commands: []domain.InvestmentCommand{{Action: domain.ActionObserve, Target: "VN30"}}}); err != nil {
		return err
	}
	return emit(port.InvestmentEvent{Type: "done"})
}
