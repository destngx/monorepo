package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func (p *Provider) usageAPI(ctx context.Context) (any, error) {
	startTime := time.Now().AddDate(0, 0, -7).Unix()
	path := fmt.Sprintf("%s?start_time=%d&bucket_width=1d&limit=7", pathUsageCompletions, startTime)

	resp, err := p.doOpenAIRequest(ctx, http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai usage error %d: %s", resp.StatusCode, b)
	}

	var result any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func (p *Provider) usageCodex(ctx context.Context) (any, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, chatGPTURL+pathCodexUsage, nil)
	if err != nil {
		return nil, err
	}

	if err := p.setAuthHeaders(httpReq); err != nil {
		return nil, err
	}
	httpReq.Header.Set(headerAccept, contentTypeJSON)
	httpReq.Header.Set(headerOriginator, codexOriginator)
	httpReq.Header.Set(headerUserAgent, codexUserAgent)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai codex usage error %d: %s", resp.StatusCode, b)
	}

	var raw struct {
		RateLimit struct {
			Primary struct {
				UsedPercent int `json:"used_percent"`
			} `json:"primary_window"`
			Secondary struct {
				UsedPercent int `json:"used_percent"`
			} `json:"secondary_window"`
		} `json:"rate_limit"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	snapshot := codexUsageSnapshot{
		Provider: "openai",
		AuthMode: "oauth",
		Source:   "codex_endpoint",
		RateLimits: map[string]any{
			"primary": map[string]any{
				"used_percent": float64(raw.RateLimit.Primary.UsedPercent),
			},
			"secondary": map[string]any{
				"used_percent": float64(raw.RateLimit.Secondary.UsedPercent),
			},
		},
		Display: codexUsageDisplay{
			FiveHour: fmt.Sprintf("%d%% used (%d%% left)", raw.RateLimit.Primary.UsedPercent, 100-raw.RateLimit.Primary.UsedPercent),
			Weekly:   fmt.Sprintf("%d%% used (%d%% left)", raw.RateLimit.Secondary.UsedPercent, 100-raw.RateLimit.Secondary.UsedPercent),
		},
		Limits: map[string]codexLimitDisplay{
			"5h":     codexLimitDisplay{LeftPercent: 100 - raw.RateLimit.Primary.UsedPercent},
			"weekly": codexLimitDisplay{LeftPercent: 100 - raw.RateLimit.Secondary.UsedPercent},
		},
	}

	return snapshot, nil
}
