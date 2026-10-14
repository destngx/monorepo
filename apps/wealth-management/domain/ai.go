package domain

import (
	"errors"
	"time"
)

// AI sentinel errors. Adapters map them to transport status codes.
var (
	// ErrAIUnavailable: AI is not configured or the provider failed (503).
	ErrAIUnavailable = errors.New("ai unavailable")
	// ErrAIOutput: the model answered but not in the required shape (502).
	ErrAIOutput = errors.New("ai returned invalid output")
	// ErrPromptMissing: a required prompt is not in the Prompts sheet (WM-AI-1).
	ErrPromptMissing = errors.New("prompt missing from the Prompts sheet")
)

// AIUseCase selects the model for a family of features (WM-AI-4).
type AIUseCase string

const (
	// AIUseCaseInsight covers reviews, briefing, budget advisor, chart and health commentary.
	AIUseCaseInsight AIUseCase = "insight"
	// AIUseCaseSuggest is category suggestion: frequent, small, latency sensitive.
	AIUseCaseSuggest AIUseCase = "suggest"
	// AIUseCaseVision reads images (Smart Capture).
	AIUseCaseVision AIUseCase = "vision"
	// AIUseCaseChat is the tool-using chat.
	AIUseCaseChat AIUseCase = "chat"
)

var AIUseCases = []AIUseCase{AIUseCaseInsight, AIUseCaseSuggest, AIUseCaseVision, AIUseCaseChat}

func (u AIUseCase) Valid() bool {
	for _, known := range AIUseCases {
		if u == known {
			return true
		}
	}
	return false
}

// AIProvider is the only model provider the app uses. It is sent as
// X-AI-Provider so ai-gateway routes every AI call to OpenAI.
const AIProvider = "openai"

// AIConfig is the model configuration (D10). The endpoint speaks the OpenAI
// Responses API (POST /responses): ai-gateway or the official OpenAI API.
type AIConfig struct {
	BaseURL string
	APIKey  string
	// Models per use case; every use case has a model once loaded.
	Models map[AIUseCase]string
	// FallbackModel is retried once when the primary model fails (WM-AI-15).
	FallbackModel string
	Timeout       time.Duration
	// PIINames are names masked in text context before it reaches a model (WM-AI-13).
	PIINames []string
}

func (c AIConfig) Configured() bool {
	return c.BaseURL != "" && c.Models[AIUseCaseInsight] != ""
}

type AIRole string

const (
	AIRoleSystem    AIRole = "system"
	AIRoleUser      AIRole = "user"
	AIRoleAssistant AIRole = "assistant"
	AIRoleTool      AIRole = "tool"
)

type AIImage struct {
	MediaType string
	Data      []byte
}

type AIToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type AIMessage struct {
	Role    AIRole
	Content string
	Images  []AIImage
	// ToolCalls are requested by an assistant message.
	ToolCalls []AIToolCall
	// ToolCallID links a tool message to the call it answers.
	ToolCallID string
}

// AITool describes a function the model may call; Parameters is a JSON schema.
type AITool struct {
	Name        string
	Description string
	Parameters  map[string]any
}

type AIRequest struct {
	Model    string
	Messages []AIMessage
	Tools    []AITool
	// JSON asks the provider for a JSON object; callers still validate it.
	JSON        bool
	Temperature *float64
	MaxTokens   int
}

type AIUsage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
}

type AIResponse struct {
	Model        string
	Content      string
	ToolCalls    []AIToolCall
	FinishReason string
	Usage        AIUsage
}

// AIContent is the prompt registry and knowledge base loaded from the sheet.
type AIContent struct {
	Prompts   map[string]string `json:"prompts"`
	Knowledge map[string]string `json:"knowledge"`
}

// AIStatus reports AI configuration for Settings (WM-AI-4); no secrets.
type AIStatus struct {
	Configured    bool                 `json:"configured"`
	Provider      string               `json:"provider,omitempty"`
	Endpoint      string               `json:"endpoint,omitempty"`
	Models        map[AIUseCase]string `json:"models"`
	FallbackModel string               `json:"fallbackModel,omitempty"`
	Prompts       int                  `json:"prompts"`
	MissingKeys   []string             `json:"missingPrompts"`
	PromptsError  string               `json:"promptsError,omitempty"`
	// Capture bounds a Smart Capture batch, for the UI (WM-CAP-1).
	Capture CaptureLimits `json:"capture"`
}

// AIProviderUsage is the provider's quota as reported by the ai-gateway
// GET /v1/usage (WM-SET-4). OAuth (codex) accounts report rate-limit
// windows; API-key accounts report token totals over recent days.
type AIProviderUsage struct {
	Provider string          `json:"provider"`
	AuthMode string          `json:"authMode,omitempty"`
	Source   string          `json:"source,omitempty"`
	Windows  []AIUsageWindow `json:"windows"`
	Totals   *AIUsageTotals  `json:"totals,omitempty"`
}

// AIUsageWindow is one rate-limit window, such as "5h" or "weekly".
type AIUsageWindow struct {
	Name        string     `json:"name"`
	UsedPercent float64    `json:"usedPercent"`
	ResetsAt    *time.Time `json:"resetsAt,omitempty"`
}

// AIUsageTotals sums the provider's usage buckets since Since.
type AIUsageTotals struct {
	Since        *time.Time `json:"since,omitempty"`
	InputTokens  int64      `json:"inputTokens"`
	OutputTokens int64      `json:"outputTokens"`
	Requests     int64      `json:"requests"`
}
