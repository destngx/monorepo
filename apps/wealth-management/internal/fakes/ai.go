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
}

var _ port.AIService = (*AI)(nil)

func (f *AI) Status(context.Context) domain.AIStatus {
	return domain.AIStatus{Configured: true, MissingKeys: []string{"tasks/loan-review"}}
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
