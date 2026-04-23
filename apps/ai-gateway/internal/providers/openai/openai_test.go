package openai

import (
	"context"
	"io"
	"net/http"
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
	var captured *http.Request
	t.Setenv("OPENAI_OAUTH_PATH", "/nonexistent/openai-auth.json")
	t.Setenv("CODEX_AUTH_PATH", "/nonexistent/codex-auth.json")
	t.Setenv("HOME", t.TempDir())

	p := New("", &config.OpenAIOAuth{AccessToken: "oauth-token"})
	p.client = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			captured = r
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body: io.NopCloser(strings.NewReader(`{
					"rate_limit": {
						"primary_window": {"used_percent": 12, "limit_window_seconds": 18000, "reset_at": 1776932940},
						"secondary_window": {"used_percent": 14, "limit_window_seconds": 604800, "reset_at": 1777445040}
					},
					"plan_type": "plus"
				}`)),
			}, nil
		}),
	}

	usage, err := p.Usage(context.Background())
	if err != nil {
		t.Fatalf("usage error: %v", err)
	}
	if captured == nil {
		t.Fatalf("expected request to be captured")
	}
	if captured.URL.Path != "/backend-api/wham/usage" {
		t.Fatalf("expected path %s, got %s", "/backend-api/wham/usage", captured.URL.Path)
	}
	if captured.Header.Get(headerAuthorization) != tokenPrefixBearer+"oauth-token" {
		t.Fatalf("expected oauth bearer authorization header")
	}
	if captured.Header.Get(headerUserAgent) != codexUserAgent {
		t.Fatalf("expected codex user agent header")
	}
	if captured.Header.Get(headerOriginator) != codexOriginator {
		t.Fatalf("expected codex originator header")
	}
	snapshot, ok := usage.(codexUsageSnapshot)
	if !ok {
		t.Fatalf("expected codex usage snapshot, got %T", usage)
	}
	if snapshot.Provider != "openai" || snapshot.AuthMode != "oauth" || snapshot.Source != "codex_endpoint" {
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
