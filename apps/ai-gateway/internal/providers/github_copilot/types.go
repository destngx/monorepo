package github_copilot

import (
	"apps/ai-gateway/internal/domain"
	"encoding/json"
)

type QuotaDetail struct {
	Entitlement      float64 `json:"entitlement"`
	OverageCount     int     `json:"overage_count"`
	OveragePermitted bool    `json:"overage_permitted"`
	PercentRemaining float64 `json:"percent_remaining"`
	QuotaID          string  `json:"quota_id"`
	QuotaRemaining   float64 `json:"quota_remaining"`
	Remaining        float64 `json:"remaining"`
	Unlimited        bool    `json:"unlimited"`
}

type QuotaSnapshots struct {
	Chat                *QuotaDetail `json:"chat,omitempty"`
	Completions         *QuotaDetail `json:"completions,omitempty"`
	PremiumInteractions *QuotaDetail `json:"premium_interactions,omitempty"`
}

type UsageResponse struct {
	AccessTypeSKU       string         `json:"access_type_sku"`
	AnalyticsTrackingID string         `json:"analytics_tracking_id"`
	AssignedDate        string         `json:"assigned_date"`
	CanSignupForLimited bool           `json:"can_signup_for_limited"`
	ChatEnabled         bool           `json:"chat_enabled"`
	CopilotPlan         string         `json:"copilot_plan"`
	QuotaResetDate      string         `json:"quota_reset_date"`
	QuotaSnapshots      QuotaSnapshots `json:"quota_snapshots"`
}

type tokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
	Endpoints struct {
		API string `json:"api"`
	} `json:"endpoints"`
}

type responsesRequest struct {
	Model           string                 `json:"model"`
	Instructions    string                 `json:"instructions,omitempty"`
	Input           []responsesInputItem   `json:"input"`
	Stream          bool                   `json:"stream"`
	Temperature     *float64               `json:"temperature,omitempty"`
	TopP            *float64               `json:"top_p,omitempty"`
	Tools           []copilotTool          `json:"tools,omitempty"`
	Reasoning       *responsesReasoning    `json:"reasoning,omitempty"`
	MaxOutputTokens *int                   `json:"max_output_tokens,omitempty"`
	ToolChoice      any                    `json:"tool_choice,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

type responsesInputItem struct {
	Role    string                  `json:"role"`
	Content []responsesInputContent `json:"content"`
}

type responsesInputContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type responsesReasoning struct {
	Effort string `json:"effort,omitempty"`
}

type responsesStreamEvent struct {
	Type     string             `json:"type"`
	Delta    string             `json:"delta,omitempty"`
	Response *responsesEnvelope `json:"response,omitempty"`
	Error    json.RawMessage    `json:"error,omitempty"`
}

type responsesEnvelope struct {
	ID        string          `json:"id"`
	CreatedAt int64           `json:"created_at"`
	Model     string          `json:"model"`
	Usage     *responsesUsage `json:"usage,omitempty"`
}

type responsesUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

func (u responsesUsage) toDomain() domain.Usage {
	return domain.Usage{
		PromptTokens:     u.InputTokens,
		CompletionTokens: u.OutputTokens,
		TotalTokens:      u.TotalTokens,
	}
}

type copilotChatRequest struct {
	Model               string                `json:"model"`
	Messages            []domain.Message      `json:"messages"`
	Stream              bool                  `json:"stream"`
	StreamOptions       *domain.StreamOptions `json:"stream_options,omitempty"`
	Temperature         *float64              `json:"temperature,omitempty"`
	TopP                *float64              `json:"top_p,omitempty"`
	MaxTokens           *int                  `json:"max_tokens,omitempty"`
	Stop                any                   `json:"stop,omitempty"`
	N                   *int                  `json:"n,omitempty"`
	Tools               []copilotTool         `json:"tools,omitempty"`
	ToolChoice          any                   `json:"tool_choice,omitempty"`
	ReasoningEffort     string                `json:"reasoning_effort,omitempty"`
	MaxCompletionTokens *int                  `json:"max_completion_tokens,omitempty"`
}

type copilotTool struct {
	Type     string                    `json:"type"`
	Function copilotFunctionDefinition `json:"function"`
}

type copilotFunctionDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

type sessionData struct {
	token   string
	baseURL string
}
