package port

import (
	"apps/wealth-management/domain"
	"context"
	"fmt"
	"strings"
)

// AIClient is a chat model provider (§10.1). Errors wrap domain.ErrAIUnavailable.
type AIClient interface {
	Complete(ctx context.Context, request domain.AIRequest) (domain.AIResponse, error)
	// Stream calls onDelta with each content fragment as it arrives and
	// returns the assembled response, including any tool calls.
	Stream(ctx context.Context, request domain.AIRequest, onDelta func(string) error) (domain.AIResponse, error)
	// Ping checks the provider answers (WM-OPS-6).
	Ping(ctx context.Context) error
}

// ReviewKind names a structured review (WM-AI-5).
type ReviewKind string

const (
	ReviewAccounts     ReviewKind = "account-review"
	ReviewBudget       ReviewKind = "budget-review"
	ReviewLoans        ReviewKind = "loan-review"
	ReviewTransactions ReviewKind = "transaction-review"
	ReviewCreditCard   ReviewKind = "credit-summary"
)

type ReviewRequest struct {
	Kind ReviewKind `json:"kind"`
	// Month "YYYY-MM" for budget and credit-card reviews; empty is the current month.
	Month string `json:"month,omitempty"`
	// Card limits a credit-card summary to one card account; empty is all cards.
	Card  string `json:"card,omitempty"`
	Force bool   `json:"force,omitempty"`
	Model string `json:"model,omitempty"`
}

type ChartInsightRequest struct {
	ChartType   string `json:"chartType"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Timeframe   string `json:"timeframe,omitempty"`
	// Data is the chart's series as the UI rendered it.
	Data  any    `json:"data"`
	Force bool   `json:"force,omitempty"`
	Model string `json:"model,omitempty"`
}

type CategorySuggestion struct {
	Category string `json:"category"`
	// Source is "history" (same payee before), "ai", or "default" when the
	// model named no known category.
	Source string `json:"source"`
}

// ChatMessage is the transport shape of a chat turn; the UI keeps history.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Messages []ChatMessage `json:"messages"`
	// Page is the pathname and a compact summary of what the user sees (WM-AI-11).
	Page  *ChatPageContext `json:"page,omitempty"`
	Model string           `json:"model,omitempty"`
}

type ChatPageContext struct {
	Path    string `json:"path"`
	Title   string `json:"title,omitempty"`
	Summary string `json:"summary,omitempty"`
}

// ChatEvent is one NDJSON line of a chat stream.
type ChatEvent struct {
	Type    string `json:"type"` // "text", "tool", "done", "error"
	Delta   string `json:"delta,omitempty"`
	Tool    string `json:"tool,omitempty"`
	Status  string `json:"status,omitempty"` // tool: "running", "ok", "error"
	Model   string `json:"model,omitempty"`
	Message string `json:"message,omitempty"`
}

// AIService implements the AI use cases on top of sheet data (§10).
type AIService interface {
	Status(ctx context.Context) domain.AIStatus
	SuggestCategory(ctx context.Context, payee string, model string) (CategorySuggestion, error)
	Review(ctx context.Context, request ReviewRequest) (domain.Insight, error)
	HealthCommentary(ctx context.Context, force bool, model string) (domain.Insight, error)
	ChartInsight(ctx context.Context, request ChartInsightRequest) (string, error)
	Briefing(ctx context.Context, force bool, model string) (domain.Briefing, error)
	BudgetAdvice(ctx context.Context, force bool, model string) (domain.BudgetAdvice, error)
	Chat(ctx context.Context, request ChatRequest, emit func(ChatEvent) error) error
}

// Validate checks the conversation ends with a user turn and has only user
// and assistant turns.
func (r ChatRequest) Validate() error {
	if len(r.Messages) == 0 {
		return fmt.Errorf("%w: messages are required", domain.ErrValidation)
	}
	for _, message := range r.Messages {
		if message.Role != string(domain.AIRoleUser) && message.Role != string(domain.AIRoleAssistant) {
			return fmt.Errorf("%w: unknown message role %q", domain.ErrValidation, message.Role)
		}
	}
	last := r.Messages[len(r.Messages)-1]
	if last.Role != string(domain.AIRoleUser) || strings.TrimSpace(last.Content) == "" {
		return fmt.Errorf("%w: the last message must be a non-empty user message", domain.ErrValidation)
	}
	return nil
}
