package openai

import (
	"apps/wealth-management/domain"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// usageProvider is fixed: the app only uses OpenAI through the ai-gateway.
	usageProvider = "openai"
	maxUsageBody  = 1 << 20
	// usageResetLayout is how the ai-gateway formats codex reset times.
	usageResetLayout = "2006-01-02 15:04 UTC"
)

// wireQuota is the ai-gateway GET /v1/usage body for OpenAI: a codex
// rate-limit snapshot (OAuth) or the OpenAI organization usage page (API key).
type wireQuota struct {
	Provider   string `json:"provider"`
	AuthMode   string `json:"auth_mode"`
	Source     string `json:"source"`
	RateLimits *struct {
		Primary   *wireQuotaWindow `json:"primary"`
		Secondary *wireQuotaWindow `json:"secondary"`
	} `json:"rate_limits"`
	Display struct {
		FiveHourReset string `json:"5h_reset_at"`
		WeeklyReset   string `json:"weekly_reset_at"`
	} `json:"display"`
	Object string            `json:"object"`
	Data   []wireQuotaBucket `json:"data"`
}

type wireQuotaWindow struct {
	UsedPercent float64 `json:"used_percent"`
}

type wireQuotaBucket struct {
	StartTime int64 `json:"start_time"`
	Results   []struct {
		InputTokens      int64 `json:"input_tokens"`
		OutputTokens     int64 `json:"output_tokens"`
		NumModelRequests int64 `json:"num_model_requests"`
	} `json:"results"`
}

// Usage reads the provider's quota from the ai-gateway GET {base}/usage.
func (c *Client) Usage(ctx context.Context) (domain.AIProviderUsage, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/usage", nil)
	if err != nil {
		return domain.AIProviderUsage{}, fmt.Errorf("%w: %v", domain.ErrAIUnavailable, err)
	}
	c.authorize(request)
	request.Header.Set(providerHeader, usageProvider)
	request.Header.Set("Accept", "application/json")
	response, err := c.http.Do(request)
	if err != nil {
		return domain.AIProviderUsage{}, fmt.Errorf("%w: %v", domain.ErrAIUnavailable, err)
	}
	defer response.Body.Close()
	if response.StatusCode/100 != 2 {
		return domain.AIProviderUsage{}, providerError(response)
	}
	var decoded wireQuota
	if err := json.NewDecoder(io.LimitReader(response.Body, maxUsageBody)).Decode(&decoded); err != nil {
		return domain.AIProviderUsage{}, fmt.Errorf("%w: decode usage response: %v", domain.ErrAIUnavailable, err)
	}
	return decoded.toDomain()
}

func (u wireQuota) toDomain() (domain.AIProviderUsage, error) {
	usage := domain.AIProviderUsage{
		Provider: firstNonEmpty(u.Provider, usageProvider),
		AuthMode: u.AuthMode,
		Source:   u.Source,
		Windows:  []domain.AIUsageWindow{},
	}
	switch {
	case u.RateLimits != nil:
		if u.RateLimits.Primary != nil {
			usage.Windows = append(usage.Windows, domain.AIUsageWindow{Name: "5h", UsedPercent: u.RateLimits.Primary.UsedPercent, ResetsAt: parseUsageReset(u.Display.FiveHourReset)})
		}
		if u.RateLimits.Secondary != nil {
			usage.Windows = append(usage.Windows, domain.AIUsageWindow{Name: "weekly", UsedPercent: u.RateLimits.Secondary.UsedPercent, ResetsAt: parseUsageReset(u.Display.WeeklyReset)})
		}
	case u.Object == "page":
		if usage.AuthMode == "" {
			usage.AuthMode = "api_key"
		}
		usage.Totals = sumUsageBuckets(u.Data)
	default:
		return domain.AIProviderUsage{}, fmt.Errorf("%w: unrecognized usage response", domain.ErrAIUnavailable)
	}
	return usage, nil
}

func sumUsageBuckets(buckets []wireQuotaBucket) *domain.AIUsageTotals {
	totals := &domain.AIUsageTotals{}
	var earliest int64
	for _, bucket := range buckets {
		if bucket.StartTime > 0 && (earliest == 0 || bucket.StartTime < earliest) {
			earliest = bucket.StartTime
		}
		for _, result := range bucket.Results {
			totals.InputTokens += result.InputTokens
			totals.OutputTokens += result.OutputTokens
			totals.Requests += result.NumModelRequests
		}
	}
	if earliest > 0 {
		since := time.Unix(earliest, 0).UTC()
		totals.Since = &since
	}
	return totals
}

// parseUsageReset returns nil when the gateway sent no (or an unreadable) reset time.
func parseUsageReset(value string) *time.Time {
	if value == "" {
		return nil
	}
	parsed, err := time.Parse(usageResetLayout, value)
	if err != nil {
		return nil
	}
	return &parsed
}
