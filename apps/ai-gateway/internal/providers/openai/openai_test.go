package openai

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"apps/ai-gateway/config"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestUsageFetchesOpenAIOrganizationUsage(t *testing.T) {
	var captured *http.Request
	p := New("test-admin-key", nil)
	p.client = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			captured = r
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body: io.NopCloser(strings.NewReader(`{
					"object": "page",
					"data": [
						{
							"object": "bucket",
							"results": [
								{
									"object": "organization.usage.completions.result",
									"input_tokens": 10,
									"output_tokens": 5,
									"num_model_requests": 1
								}
							]
						}
					]
				}`)),
			}, nil
		}),
	}

	usage, err := p.Usage(context.Background())
	if err != nil {
		t.Fatalf("usage error: %v", err)
	}
	if usage == nil {
		t.Fatalf("expected usage response")
	}
	if captured == nil {
		t.Fatalf("expected request to be captured")
	}
	if captured.URL.Path != "/v1"+pathUsageCompletions {
		t.Fatalf("expected path %s, got %s", "/v1"+pathUsageCompletions, captured.URL.Path)
	}
	if captured.URL.Query().Get("start_time") == "" {
		t.Fatalf("expected start_time query")
	}
	if captured.URL.Query().Get("bucket_width") != "1d" {
		t.Fatalf("expected bucket_width=1d, got %s", captured.URL.Query().Get("bucket_width"))
	}
	if captured.URL.Query().Get("limit") != "7" {
		t.Fatalf("expected limit=7, got %s", captured.URL.Query().Get("limit"))
	}
	if captured.Header.Get(headerAuthorization) != tokenPrefixBearer+"test-admin-key" {
		t.Fatalf("expected bearer authorization header")
	}
}

func TestUsageReturnsCodexSnapshotForOAuth(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sessionDir := filepath.Join(home, ".codex", "sessions", "2026", "04", "23")
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		t.Fatalf("mkdir session dir: %v", err)
	}
	sessionPath := filepath.Join(sessionDir, "rollout.jsonl")
	content := strings.Join([]string{
		`{"timestamp":"2026-04-23T01:00:00Z","type":"event_msg","payload":{"type":"token_count","info":null,"rate_limits":{"primary":{"used_percent":10,"window_minutes":300},"secondary":{"used_percent":2,"window_minutes":10080},"plan_type":"plus"}}}`,
		`{"timestamp":"2026-04-23T02:00:00Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"total_tokens":123}},"rate_limits":{"primary":{"used_percent":12,"window_minutes":300,"resets_at":1776932940},"secondary":{"used_percent":14,"window_minutes":10080,"resets_at":1777445040},"plan_type":"plus"}}}`,
	}, "\n")
	if err := os.WriteFile(sessionPath, []byte(content), 0600); err != nil {
		t.Fatalf("write session: %v", err)
	}

	p := New("", &config.OpenAIOAuth{AccessToken: "oauth-token"})

	usage, err := p.Usage(context.Background())
	if err != nil {
		t.Fatalf("usage error: %v", err)
	}
	snapshot, ok := usage.(codexUsageSnapshot)
	if !ok {
		t.Fatalf("expected codex usage snapshot, got %T", usage)
	}
	if snapshot.Provider != "openai" || snapshot.AuthMode != "oauth" || snapshot.Source != "codex_session" {
		t.Fatalf("unexpected snapshot metadata: %+v", snapshot)
	}
	limits, ok := snapshot.RateLimits.(map[string]any)
	if !ok {
		t.Fatalf("expected rate limits map")
	}
	primary, ok := limits["primary"].(map[string]any)
	if !ok {
		t.Fatalf("expected primary limit")
	}
	if primary["used_percent"] != float64(12) {
		t.Fatalf("expected latest primary used percent, got %v", primary["used_percent"])
	}
	display, ok := snapshot.Display.(codexUsageDisplay)
	if !ok {
		t.Fatalf("expected codex display, got %T", snapshot.Display)
	}
	if !strings.Contains(display.FiveHour, "88% left") {
		t.Fatalf("expected 5h left display, got %q", display.FiveHour)
	}
	if !strings.Contains(display.Weekly, "86% left") {
		t.Fatalf("expected weekly left display, got %q", display.Weekly)
	}
	limitsDisplay, ok := snapshot.Limits.(map[string]codexLimitDisplay)
	if !ok {
		t.Fatalf("expected codex limit display map, got %T", snapshot.Limits)
	}
	if limitsDisplay["5h"].LeftPercent != 88 {
		t.Fatalf("expected 5h left percent 88, got %v", limitsDisplay["5h"].LeftPercent)
	}
	if limitsDisplay["weekly"].LeftPercent != 86 {
		t.Fatalf("expected weekly left percent 86, got %v", limitsDisplay["weekly"].LeftPercent)
	}
}
