package copilot

import (
	"apps/wealth-management-engine/domain"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestGivenTokenNotCachedWhenStreamCompletionThenExchangesTokenAndCallsCopilot(t *testing.T) {
	var tokenHits atomic.Int32
	var chatHits atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/copilot_internal/v2/token":
			tokenHits.Add(1)
			if r.Header.Get("Authorization") != "token gh-test-token" {
				t.Fatalf("unexpected token auth header")
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"token":      "copilot-bearer",
				"expires_at": time.Now().Unix() + 3600,
			})
		case "/chat/completions":
			chatHits.Add(1)
			if r.Header.Get("Authorization") != "Bearer copilot-bearer" {
				t.Fatalf("missing bearer token")
			}
			if r.Header.Get("Editor-Version") == "" || r.Header.Get("Editor-Plugin-Version") == "" {
				t.Fatalf("missing editor headers")
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("invalid body: %v", err)
			}
			if body["model"] != "gpt-4.1" {
				t.Fatalf("expected gpt-4.1 model")
			}
			if body["stream"] != true {
				t.Fatalf("expected stream=true")
			}
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\n"))
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(domain.AIConfig{
		GitHubToken:    "gh-test-token",
		GitHubAPIBase:  server.URL,
		CopilotAPIBase: server.URL,
		DefaultModel:   "gpt-4.1",
	})
	if err != nil {
		t.Fatalf("new client error: %v", err)
	}

	stream, err := client.StreamCompletion(context.Background(), "Say hello", "gpt-4.1")
	if err != nil {
		t.Fatalf("stream completion error: %v", err)
	}
	defer stream.Close()

	body, _ := io.ReadAll(stream)
	if string(body) == "" || !strings.Contains(string(body), "[DONE]") {
		t.Fatalf("expected stream to contain DONE marker")
	}
	if tokenHits.Load() != 1 || chatHits.Load() != 1 {
		t.Fatalf("expected one token and one chat call, got token=%d chat=%d", tokenHits.Load(), chatHits.Load())
	}
}

func TestGivenTokenCachedWhenStreamCompletionThenSkipsTokenExchange(t *testing.T) {
	var tokenHits atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/copilot_internal/v2/token":
			tokenHits.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"token":      "copilot-bearer",
				"expires_at": time.Now().Unix() + 3600,
			})
		case "/chat/completions":
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(domain.AIConfig{
		GitHubToken:    "gh-test-token",
		GitHubAPIBase:  server.URL,
		CopilotAPIBase: server.URL,
		DefaultModel:   "gpt-4.1",
	})
	if err != nil {
		t.Fatalf("new client error: %v", err)
	}

	for i := 0; i < 2; i++ {
		stream, err := client.StreamCompletion(context.Background(), "Ping", "gpt-4.1")
		if err != nil {
			t.Fatalf("stream completion error: %v", err)
		}
		_ = stream.Close()
	}

	if tokenHits.Load() != 1 {
		t.Fatalf("expected one token exchange due to cache, got %d", tokenHits.Load())
	}
}

func TestGivenRoleMessagesWhenCompleteJSONThenReturnsStructuredJSONContent(t *testing.T) {
	var tokenHits atomic.Int32
	var completionHits atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/copilot_internal/v2/token":
			tokenHits.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"token":      "copilot-bearer",
				"expires_at": time.Now().Unix() + 3600,
			})
		case "/chat/completions":
			completionHits.Add(1)
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("invalid request body: %v", err)
			}
			if body["stream"] != false {
				t.Fatalf("expected stream=false for CompleteJSON")
			}
			responseFormat, ok := body["response_format"].(map[string]any)
			if !ok || responseFormat["type"] != "json_object" {
				t.Fatalf("expected json_object response_format")
			}

			_ = json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{
					{
						"message": map[string]any{
							"content": `{"persona":"CFO","summary":"OK","actions":["Hold"],"roles":["system","assistant","user"]}`,
						},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(domain.AIConfig{
		GitHubToken:    "gh-test-token",
		GitHubAPIBase:  server.URL,
		CopilotAPIBase: server.URL,
		DefaultModel:   "gpt-4.1",
	})
	if err != nil {
		t.Fatalf("new client error: %v", err)
	}

	result, err := client.CompleteJSON(context.Background(), []domain.RoleMessage{
		{Role: "system", Content: "json only"},
		{Role: "assistant", Content: "history"},
		{Role: "user", Content: "brief"},
	}, "gpt-4.1")
	if err != nil {
		t.Fatalf("complete json error: %v", err)
	}

	if !strings.Contains(result, `"persona":"CFO"`) {
		t.Fatalf("unexpected complete json result: %s", result)
	}
	if tokenHits.Load() != 1 || completionHits.Load() != 1 {
		t.Fatalf("expected one token and one completion call, got token=%d completion=%d", tokenHits.Load(), completionHits.Load())
	}
}
