package copilot

import (
	"apps/wealth-management-engine/domain"
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestGivenDirectBearerTokenWhenStreamCompletionThenSkipsTokenExchange(t *testing.T) {
	var tokenHits atomic.Int32
	var chatHits atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/copilot_internal/v2/token":
			tokenHits.Add(1)
			http.NotFound(w, r)
		case "/chat/completions":
			chatHits.Add(1)
			if r.Header.Get("Authorization") != "Bearer direct-bearer-token" {
				t.Fatalf("expected direct bearer token auth header")
			}
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewCopilotClient(domain.AIConfig{
		CopilotBearerToken: "direct-bearer-token",
		GitHubAPIBase:      server.URL,
		CopilotAPIBase:     server.URL,
		DefaultModel:       "gpt-4.1",
	})
	if err != nil {
		t.Fatalf("new client error: %v", err)
	}

	stream, err := client.StreamCompletion(context.Background(), "Ping", "gpt-4.1")
	if err != nil {
		t.Fatalf("stream completion error: %v", err)
	}
	_ = stream.Close()

	if tokenHits.Load() != 0 {
		t.Fatalf("expected token exchange to be skipped when direct bearer is configured")
	}
	if chatHits.Load() != 1 {
		t.Fatalf("expected one chat completion request, got %d", chatHits.Load())
	}
}
