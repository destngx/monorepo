package openai

import (
	"apps/ai-gateway/internal/domain"
	"encoding/json"
)

type oauthRefreshResponse struct {
	AccountID        string `json:"account_id"`
	ChatGPTAccountID string `json:"chatgpt_account_id"`
	IDToken          string `json:"id_token"`
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
}

type codexResponseRequest struct {
	Model           string              `json:"model"`
	Instructions    string              `json:"instructions"`
	Input           []codexInputMessage `json:"input"`
	Stream          bool                `json:"stream"`
	Store           bool                `json:"store"`
	MaxOutputTokens *int                `json:"max_output_tokens,omitempty"`
}

type codexInputMessage struct {
	Role    string              `json:"role"`
	Content []codexInputContent `json:"content"`
}

type codexInputContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type codexStreamEvent struct {
	Type     string          `json:"type"`
	Delta    string          `json:"delta,omitempty"`
	Response *codexResponse  `json:"response,omitempty"`
	Error    json.RawMessage `json:"error,omitempty"`
}

type chatGPTModelsResponse struct {
	Models []chatGPTModel `json:"models"`
}

type chatGPTModel struct {
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type codexResponse struct {
	ID        string      `json:"id"`
	CreatedAt int64       `json:"created_at"`
	Model     string      `json:"model"`
	Usage     *codexUsage `json:"usage,omitempty"`
}

type codexUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

func (u codexUsage) toDomain() domain.Usage {
	return domain.Usage{
		PromptTokens:     u.InputTokens,
		CompletionTokens: u.OutputTokens,
		TotalTokens:      u.TotalTokens,
	}
}

type codexUsageSnapshot struct {
	Provider   string `json:"provider"`
	AuthMode   string `json:"auth_mode"`
	Source     string `json:"source"`
	RateLimits any    `json:"rate_limits"`
	Display    any    `json:"display"`
	Limits     any    `json:"limits"`
}

type codexUsageDisplay struct {
	FiveHour string `json:"5h"`
	Weekly   string `json:"weekly"`
}

type codexLimitDisplay struct {
	LeftPercent int `json:"left_percent"`
}
