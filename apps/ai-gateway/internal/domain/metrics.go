package domain

import (
	"time"
)

// MetricsPayload is the mutable struct injected into context by
// the Metrics middleware. Handlers populate it; middleware reads it.
type MetricsPayload struct {
	Provider   string
	Model      string
	InputModel string
	Usage      Usage
	Stream     bool
	Error      string
}

// RequestRecord is the atomic unit of metrics — one per proxied request.
type RequestRecord struct {
	Timestamp  time.Time `json:"timestamp"`
	Route      string    `json:"route"`
	Provider   string    `json:"provider"`
	Model      string    `json:"model"`
	InputModel string    `json:"input_model"`
	Stream     bool      `json:"stream"`
	StatusCode int       `json:"status_code"`
	DurationMs int64     `json:"duration_ms"`
	Usage      Usage     `json:"usage"`
	Error      string    `json:"error,omitempty"`
}

// MetricsSummary is the aggregated view returned by GET /metrics.
type MetricsSummary struct {
	TotalRequests  int64                  `json:"total_requests"`
	TotalErrors    int64                  `json:"total_errors"`
	TotalTokens    int64                  `json:"total_tokens"`
	AvgDurationMs  float64                `json:"avg_duration_ms"`
	UptimeSecs     int64                  `json:"uptime_secs"`
	ByRoute        map[string]*RouteStats `json:"by_route"`
	ByProvider     map[string]*RouteStats `json:"by_provider"`
	ByModel        map[string]*RouteStats `json:"by_model"`
	RecentRequests []RequestSnapshot      `json:"recent_requests"`
	TimeSeries     *TimeSeriesUsage       `json:"time_series,omitempty"`
}

// TimeSeriesUsage holds historical token consumption at different grains.
type TimeSeriesUsage struct {
	Daily   []UsagePoint `json:"daily"`
	Weekly  []UsagePoint `json:"weekly"`
	Monthly []UsagePoint `json:"monthly"`
}

// UsagePoint is a single data point in a time series.
type UsagePoint struct {
	Label  string `json:"label"` // e.g. "2024-04-15"
	Tokens int64  `json:"tokens"`
}

// RouteStats groups metrics for a single dimension value.
// Reusing the same struct for ByRoute, ByProvider, ByModel.
type RouteStats struct {
	Count          int64   `json:"count"`
	Errors         int64   `json:"errors"`
	AvgDurationMs  float64 `json:"avg_duration_ms"`
	P95DurationMs  int64   `json:"p95_duration_ms"`
	TotalPromptTkn int64   `json:"total_prompt_tokens"`
	TotalCompTkn   int64   `json:"total_completion_tokens"`
	TotalTokens    int64   `json:"total_tokens"`
}

// RequestSnapshot is a lightweight view of a recent request for the live feed.
type RequestSnapshot struct {
	Timestamp    string `json:"timestamp"`
	Route        string `json:"route"`
	Provider     string `json:"provider"`
	Model        string `json:"model"`
	StatusCode   int    `json:"status_code"`
	DurationMs   int64  `json:"duration_ms"`
	PromptTokens int    `json:"prompt_tokens"`
	CompTokens   int    `json:"completion_tokens"`
	TotalTokens  int    `json:"total_tokens"`
	Error        string `json:"error,omitempty"`
}

// P/C/T Notation explanation:
// P: Prompt Tokens (Input)
// C: Completion Tokens (Output)
// T: Total Tokens (Sum)
