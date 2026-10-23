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
	// Usage reports the provider's quota (WM-SET-4).
	Usage(ctx context.Context) (domain.AIProviderUsage, error)
	// Models lists the model ids the provider offers.
	Models(ctx context.Context) ([]string, error)
}

// ReviewKind names a structured review (WM-AI-5).
type ReviewKind string

const (
	ReviewAccounts     ReviewKind = "account-review"
	ReviewTransactions ReviewKind = "transaction-review"
	ReviewCreditCard   ReviewKind = "credit-summary"
	ReviewYear         ReviewKind = "year-review"
)

type ReviewRequest struct {
	Kind ReviewKind `json:"kind"`
	// Month "YYYY-MM" for credit-card reviews; empty is the current month.
	Month string `json:"month,omitempty"`
	// Card limits a credit-card summary to one card account; empty is all cards.
	Card string `json:"card,omitempty"`
	// Year for the yearly review; 0 is the current year.
	Year  int    `json:"year,omitempty"`
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

// CashbackExplainRequest asks why a card's period earned its cashback
// estimate (WM-CARD-17).
type CashbackExplainRequest struct {
	Card string `json:"card"`
	// Month is the "YYYY-MM" the period starts in; empty is the current one.
	Month string `json:"month,omitempty"`
	Force bool   `json:"force,omitempty"`
	Model string `json:"model,omitempty"`
}

type CategorySuggestion struct {
	Category string `json:"category"`
	// Source is "rule" (a confirmed CaptureRules payee rule), "history"
	// (same payee before), "ai", or "default" when the model named no known
	// category.
	Source string `json:"source"`
	// SplitTag is one of the category's split tags when it fits the payee
	// (WM-CAT-5): from the payee's history or the model; empty otherwise.
	SplitTag string `json:"splitTag,omitempty"`
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
	// ProviderUsage reports the AI provider's quota (WM-SET-4).
	ProviderUsage(ctx context.Context) (domain.AIProviderUsage, error)
	// ProviderModels lists the models the AI endpoint offers, for Settings.
	ProviderModels(ctx context.Context) ([]string, error)
	SuggestCategory(ctx context.Context, payee string, model string) (CategorySuggestion, error)
	Review(ctx context.Context, request ReviewRequest) (domain.Insight, error)
	HealthCommentary(ctx context.Context, force bool, model string) (domain.Insight, error)
	ChartInsight(ctx context.Context, request ChartInsightRequest) (string, error)
	// ExplainCashback explains a card period's cashback estimate; markdown.
	ExplainCashback(ctx context.Context, request CashbackExplainRequest) (string, error)
	Briefing(ctx context.Context, force bool, model string) (domain.Briefing, error)
	BudgetAdvice(ctx context.Context, force bool, model string) (domain.BudgetAdvice, error)
	Chat(ctx context.Context, request ChatRequest, emit func(ChatEvent) error) error
	// Capture reads materials into suggested transactions without writing
	// anything (§6.2). progress may be nil; calls to it are serialised.
	Capture(ctx context.Context, request CaptureRequest, progress func(CaptureProgress)) (domain.CaptureResult, error)
	// Fill reads a sentence or images into a form's fields without writing
	// anything (§10.5).
	Fill(ctx context.Context, request domain.FillRequest) (domain.FillResult, error)
	// NewsSentiment searches a topic and classifies each article (WM-INV-17).
	NewsSentiment(ctx context.Context, request NewsRequest) (domain.NewsReport, error)
	// TickerInsight comments on the deterministic analysis of a symbol (WM-AI-14).
	TickerInsight(ctx context.Context, request TickerInsightRequest) (domain.Insight, error)
	// InvestmentAnalysis streams the think tank (WM-AI-9, WM-INV-18).
	InvestmentAnalysis(ctx context.Context, request InvestmentAnalysisRequest, emit func(InvestmentEvent) error) error
}

type NewsRequest struct {
	Topic string `json:"topic"`
	// Force skips the search and result caches.
	Force bool   `json:"force,omitempty"`
	Model string `json:"model,omitempty"`
}

type TickerInsightRequest struct {
	Symbol string `json:"symbol"`
	// Timeframe is 1h, 4h, 1d or 1w; empty is 1d.
	Timeframe string `json:"timeframe,omitempty"`
	Force     bool   `json:"force,omitempty"`
	Model     string `json:"model,omitempty"`
}

type InvestmentAnalysisRequest struct {
	// News are scanner reports the user already ran; they are passed to the
	// think tank as pre-analysed context.
	News []domain.NewsReport `json:"news,omitempty"`
	// Symbols are what to analyse (ROADMAP M5, WM-INV-29); empty means the
	// user's current holdings.
	Symbols []string `json:"symbols,omitempty"`
	Model   string   `json:"model,omitempty"`
}

// InvestmentEvent is one NDJSON line of the think-tank stream. Phases are
// "context", "think-tank", "synthesis" and "actions".
type InvestmentEvent struct {
	Type     string                     `json:"type"` // "phase", "message", "actions", "context", "done", "error"
	Phase    string                     `json:"phase,omitempty"`
	Status   string                     `json:"status,omitempty"` // phase: "running", "ok", "fallback", "error"
	Name     string                     `json:"name,omitempty"`
	Content  string                     `json:"content,omitempty"`
	Commands []domain.InvestmentCommand `json:"commands,omitempty"`
	// Context discloses the raw inputs the models saw.
	Context any    `json:"context,omitempty"`
	Model   string `json:"model,omitempty"`
	Message string `json:"message,omitempty"`
	// Recorded is what the actions phase wrote to the recommendations list
	// (WM-REC-8); its Error says when the list could not be written.
	Recorded *domain.RecommendationRecord `json:"recorded,omitempty"`
}

// CaptureRequest is one Smart Capture batch (WM-CAP-1).
type CaptureRequest struct {
	Materials []domain.CaptureMaterial `json:"materials"`
	// Model overrides the vision model for extraction.
	Model string `json:"model,omitempty"`
}

// CaptureProgress reports a pipeline step. Stage is "extract" (per
// material, Status "running", "ok", "unreadable" or "error"), "categorize"
// or "match".
type CaptureProgress struct {
	Stage      string `json:"stage"`
	MaterialID string `json:"materialId,omitempty"`
	Status     string `json:"status,omitempty"`
	Events     int    `json:"events,omitempty"`
	Message    string `json:"message,omitempty"`
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
