package domain

import "time"

// AIUsage is the AI provider's quota as ai-gateway reports it at
// GET /v1/usage (HW-OPS-10). A ChatGPT (Codex) account reports rate-limit
// windows; an API-key account reports token totals over recent days.
type AIUsage struct {
	Provider string          `json:"provider"`
	AuthMode string          `json:"authMode,omitempty"`
	Source   string          `json:"source,omitempty"`
	Windows  []AIUsageWindow `json:"windows"`
	Totals   *AIUsageTotals  `json:"totals,omitempty"`
}

// AIUsageWindow is one rate-limit window, "5h" or "weekly".
type AIUsageWindow struct {
	Name        string     `json:"name"`
	UsedPercent float64    `json:"usedPercent"`
	ResetsAt    *time.Time `json:"resetsAt,omitempty"`
}

// AIUsageTotals sums the provider's usage since Since.
type AIUsageTotals struct {
	Since        *time.Time `json:"since,omitempty"`
	InputTokens  int64      `json:"inputTokens"`
	OutputTokens int64      `json:"outputTokens"`
	Requests     int64      `json:"requests"`
}
