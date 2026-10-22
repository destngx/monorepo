package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"apps/health-watcher/domain"
)

const (
	// usageTimeout bounds a quota read: it is a status line, not a reading.
	usageTimeout = 15 * time.Second
	maxUsageBody = 1 << 20
	// usageResetLayout is how ai-gateway writes Codex reset times.
	usageResetLayout = "2006-01-02 15:04 UTC"
)

// wireUsage is the ai-gateway GET /v1/usage body for OpenAI: a Codex
// rate-limit snapshot, or the organization usage page for an API key.
type wireUsage struct {
	Provider   string `json:"provider"`
	AuthMode   string `json:"auth_mode"`
	Source     string `json:"source"`
	RateLimits *struct {
		Primary   *wireWindow `json:"primary"`
		Secondary *wireWindow `json:"secondary"`
	} `json:"rate_limits"`
	Display struct {
		FiveHourReset string `json:"5h_reset_at"`
		WeeklyReset   string `json:"weekly_reset_at"`
	} `json:"display"`
	Object string       `json:"object"`
	Data   []wireBucket `json:"data"`
}

type wireWindow struct {
	UsedPercent float64 `json:"used_percent"`
}

type wireBucket struct {
	StartTime int64 `json:"start_time"`
	Results   []struct {
		InputTokens      int64 `json:"input_tokens"`
		OutputTokens     int64 `json:"output_tokens"`
		NumModelRequests int64 `json:"num_model_requests"`
	} `json:"results"`
}

// errUnexpectedUsage is an answer that is neither usage shape.
var errUnexpectedUsage = errors.New("unrecognized usage response")

// Usage reads the provider's quota from ai-gateway. The error is safe to
// show in a browser, like Ping's.
func (r *Reader) Usage(ctx context.Context) (domain.AIUsage, error) {
	u, err := r.usage(ctx)
	if err != nil {
		return domain.AIUsage{}, publicError(err)
	}
	return u, nil
}

func (r *Reader) usage(ctx context.Context) (domain.AIUsage, error) {
	ctx, cancel := context.WithTimeout(ctx, usageTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.cfg.BaseURL+"/usage", nil)
	if err != nil {
		return domain.AIUsage{}, fmt.Errorf("build usage request: %w", err)
	}
	r.authorize(req)
	req.Header.Set("Accept", "application/json")
	resp, err := r.http.Do(req)
	if err != nil {
		return domain.AIUsage{}, &unreachableError{err: err}
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, maxUsageBody))
		return domain.AIUsage{}, &statusError{code: resp.StatusCode, message: errorMessage(raw)}
	}
	var decoded wireUsage
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxUsageBody)).Decode(&decoded); err != nil {
		return domain.AIUsage{}, fmt.Errorf("decode usage response: %w", err)
	}
	return decoded.toDomain()
}

func (w wireUsage) toDomain() (domain.AIUsage, error) {
	u := domain.AIUsage{Provider: w.Provider, AuthMode: w.AuthMode, Source: w.Source, Windows: []domain.AIUsageWindow{}}
	if u.Provider == "" {
		u.Provider = gatewayProvider
	}
	switch {
	case w.RateLimits != nil:
		if p := w.RateLimits.Primary; p != nil {
			u.Windows = append(u.Windows, domain.AIUsageWindow{Name: "5h", UsedPercent: p.UsedPercent, ResetsAt: parseReset(w.Display.FiveHourReset)})
		}
		if s := w.RateLimits.Secondary; s != nil {
			u.Windows = append(u.Windows, domain.AIUsageWindow{Name: "weekly", UsedPercent: s.UsedPercent, ResetsAt: parseReset(w.Display.WeeklyReset)})
		}
	case w.Object == "page":
		if u.AuthMode == "" {
			u.AuthMode = "api_key"
		}
		u.Totals = sumBuckets(w.Data)
	default:
		return domain.AIUsage{}, errUnexpectedUsage
	}
	return u, nil
}

func sumBuckets(buckets []wireBucket) *domain.AIUsageTotals {
	totals := &domain.AIUsageTotals{}
	var earliest int64
	for _, b := range buckets {
		if b.StartTime > 0 && (earliest == 0 || b.StartTime < earliest) {
			earliest = b.StartTime
		}
		for _, res := range b.Results {
			totals.InputTokens += res.InputTokens
			totals.OutputTokens += res.OutputTokens
			totals.Requests += res.NumModelRequests
		}
	}
	if earliest > 0 {
		since := time.Unix(earliest, 0).UTC()
		totals.Since = &since
	}
	return totals
}

// parseReset is nil for a missing or unreadable reset time.
func parseReset(value string) *time.Time {
	t, err := time.Parse(usageResetLayout, value)
	if err != nil {
		return nil
	}
	return &t
}
