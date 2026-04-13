package types

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

type CopilotUsageResponse struct {
	AccessTypeSKU       string         `json:"access_type_sku"`
	AnalyticsTrackingID string         `json:"analytics_tracking_id"`
	AssignedDate        string         `json:"assigned_date"`
	CanSignupForLimited bool           `json:"can_signup_for_limited"`
	ChatEnabled         bool           `json:"chat_enabled"`
	CopilotPlan         string         `json:"copilot_plan"`
	QuotaResetDate      string         `json:"quota_reset_date"`
	QuotaSnapshots      QuotaSnapshots `json:"quota_snapshots"`
}
