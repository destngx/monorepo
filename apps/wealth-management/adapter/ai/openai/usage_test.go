package openai

import (
	"apps/wealth-management/domain"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestUsageReadsCodexRateLimitWindows(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/usage" || r.Header.Get("Authorization") != "Bearer key" || r.Header.Get(providerHeader) != "openai" {
			t.Errorf("unexpected request %s %s %v", r.Method, r.URL.Path, r.Header)
		}
		_, _ = io.WriteString(w, `{"provider":"openai","auth_mode":"oauth","source":"codex_endpoint",
			"rate_limits":{"primary":{"used_percent":12},"secondary":{"used_percent":40.5}},
			"display":{"5h":"12% used (88% left)","5h_reset_at":"2026-10-13 15:04 UTC","weekly":"40% used (60% left)","weekly_reset_at":""},
			"limits":{"5h":{"left_percent":88},"weekly":{"left_percent":60}}}`)
	})

	usage, err := client.Usage(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if usage.Provider != "openai" || usage.AuthMode != "oauth" || usage.Source != "codex_endpoint" || usage.Totals != nil {
		t.Fatalf("unexpected usage %+v", usage)
	}
	if len(usage.Windows) != 2 {
		t.Fatalf("want 2 windows, got %+v", usage.Windows)
	}
	fiveHour, weekly := usage.Windows[0], usage.Windows[1]
	want := time.Date(2026, 10, 13, 15, 4, 0, 0, time.UTC)
	if fiveHour.Name != "5h" || fiveHour.UsedPercent != 12 || fiveHour.ResetsAt == nil || !fiveHour.ResetsAt.Equal(want) {
		t.Fatalf("unexpected 5h window %+v", fiveHour)
	}
	if weekly.Name != "weekly" || weekly.UsedPercent != 40.5 || weekly.ResetsAt != nil {
		t.Fatalf("unexpected weekly window %+v", weekly)
	}
}

func TestUsageSendsOpenAIProviderEvenWhenConfiguredOtherwise(t *testing.T) {
	var provider string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		provider = r.Header.Get(providerHeader)
		_, _ = io.WriteString(w, `{"rate_limits":{}}`)
	})
	client.provider = "github-copilot"

	usage, err := client.Usage(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if provider != "openai" || usage.Provider != "openai" || len(usage.Windows) != 0 {
		t.Fatalf("provider %q, usage %+v", provider, usage)
	}
}

func TestUsageSumsAPIKeyUsageBuckets(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"object":"page","data":[
			{"object":"bucket","start_time":1760313600,"results":[{"input_tokens":100,"output_tokens":20,"num_model_requests":3}]},
			{"object":"bucket","start_time":1760227200,"results":[{"input_tokens":50,"output_tokens":5,"num_model_requests":1},{"input_tokens":1,"output_tokens":1,"num_model_requests":1}]},
			{"object":"bucket","start_time":1760400000,"results":[]}]}`)
	})

	usage, err := client.Usage(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if usage.Provider != "openai" || usage.AuthMode != "api_key" || len(usage.Windows) != 0 || usage.Totals == nil {
		t.Fatalf("unexpected usage %+v", usage)
	}
	totals := *usage.Totals
	if totals.InputTokens != 151 || totals.OutputTokens != 26 || totals.Requests != 5 {
		t.Fatalf("unexpected totals %+v", totals)
	}
	if totals.Since == nil || !totals.Since.Equal(time.Unix(1760227200, 0)) {
		t.Fatalf("since = %v, want the earliest bucket", totals.Since)
	}
}

func TestUsageFailuresAreUnavailable(t *testing.T) {
	cases := map[string]http.HandlerFunc{
		"not configured": func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, `{"error":{"message":"provider openai not configured"}}`, http.StatusNotFound)
		},
		"upstream failed": func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "bad gateway", http.StatusBadGateway)
		},
		"not json": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, "<html>")
		},
		"unknown shape": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"hello":"world"}`)
		},
	}
	for name, handler := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := newTestClient(t, handler).Usage(context.Background())
			if !errors.Is(err, domain.ErrAIUnavailable) {
				t.Fatalf("want ErrAIUnavailable, got %v", err)
			}
		})
	}
}
