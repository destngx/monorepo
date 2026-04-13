package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"apps/ai-gateway/types"
)

func TestGitHubCopilotPayloadFixesToolSchema(t *testing.T) {
	req := types.ChatRequest{
		Model: "claude-haiku-4.5",
		Messages: []types.Message{
			{Role: "user", Content: "hello"},
		},
		Tools: []types.Tool{
			{
				Type: "function",
				Function: types.FunctionDefinition{
					Name:       "lookup",
					Parameters: map[string]any{"properties": map[string]any{}},
				},
			},
			{
				Type: "function",
				Function: types.FunctionDefinition{
					Name:       "lookup2",
					Parameters: map[string]any{"type": "None"},
				},
			},
		},
	}

	payload, err := githubCopilotPayload(req)
	if err != nil {
		t.Fatalf("githubCopilotPayload() error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	tools := decoded["tools"].([]any)
	firstType := tools[0].(map[string]any)["function"].(map[string]any)["parameters"].(map[string]any)["type"]
	secondType := tools[1].(map[string]any)["function"].(map[string]any)["parameters"].(map[string]any)["type"]

	if firstType != "object" {
		t.Fatalf("expected missing type to become object, got %v", firstType)
	}
	if secondType != "object" {
		t.Fatalf("expected None type to become object, got %v", secondType)
	}
}

func TestGitHubCopilotProviderChatUsesTokenExchange(t *testing.T) {
	var authHeaders []string
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/copilot_internal/v2/token", func(w http.ResponseWriter, r *http.Request) {
		authHeaders = append(authHeaders, r.Header.Get("Authorization"))
		if r.Header.Get("User-Agent") != githubCopilotUserAgent {
			t.Fatalf("expected token exchange User-Agent %q, got %q", githubCopilotUserAgent, r.Header.Get("User-Agent"))
		}
		if r.Header.Get("Editor-Version") != githubCopilotEditorVersion {
			t.Fatalf("expected token exchange Editor-Version %q, got %q", githubCopilotEditorVersion, r.Header.Get("Editor-Version"))
		}
		if r.Header.Get("Editor-Plugin-Version") != githubCopilotPluginVersion {
			t.Fatalf("expected token exchange Editor-Plugin-Version %q, got %q", githubCopilotPluginVersion, r.Header.Get("Editor-Plugin-Version"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"token":"copilot-token","expires_at":4102444800}`))
	})

	mux.HandleFunc("/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		authHeaders = append(authHeaders, r.Header.Get("Authorization"))
		if r.Header.Get("Editor-Version") != githubCopilotEditorVersion {
			t.Fatalf("expected chat Editor-Version %q, got %q", githubCopilotEditorVersion, r.Header.Get("Editor-Version"))
		}
		if r.Header.Get("Editor-Plugin-Version") != githubCopilotPluginVersion {
			t.Fatalf("expected chat Editor-Plugin-Version %q, got %q", githubCopilotPluginVersion, r.Header.Get("Editor-Plugin-Version"))
		}
		if r.Header.Get("User-Agent") != githubCopilotUserAgent {
			t.Fatalf("expected chat User-Agent %q, got %q", githubCopilotUserAgent, r.Header.Get("User-Agent"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"copilot-id","object":"chat.completion","model":"claude-haiku-4.5","choices":[{"index":0,"message":{"role":"assistant","content":"hi"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
	})

	provider := NewGitHubCopilot("gh-token")
	provider.client = server.Client()
	provider.defaultBaseURL = server.URL
	provider.copilotAPIBase = server.URL
	provider.tokenURL = server.URL + "/copilot_internal/v2/token"

	resp, err := provider.Chat(context.Background(), types.ChatRequest{
		Model:    "claude-haiku-4.5",
		Messages: []types.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if resp.ID != "copilot-id" {
		t.Fatalf("expected response id copilot-id, got %q", resp.ID)
	}
	if len(authHeaders) != 2 {
		t.Fatalf("expected 2 auth headers to be captured, got %d", len(authHeaders))
	}
	if authHeaders[0] != "token gh-token" {
		t.Fatalf("expected token exchange auth header, got %q", authHeaders[0])
	}
	if authHeaders[1] != "Bearer copilot-token" {
		t.Fatalf("expected copilot bearer auth header, got %q", authHeaders[1])
	}
}

func TestGitHubCopilotProviderCachesToken(t *testing.T) {
	var tokenCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/copilot_internal/v2/token"):
			tokenCalls++
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"token":"copilot-token","expires_at":4102444800}`))
		case strings.HasSuffix(r.URL.Path, "/chat/completions"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"id":"copilot-id","object":"chat.completion","model":"gpt-4.1","choices":[{"index":0,"message":{"role":"assistant","content":"hi"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	provider := NewGitHubCopilot("gh-token")
	provider.client = server.Client()
	provider.defaultBaseURL = server.URL
	provider.copilotAPIBase = server.URL
	provider.tokenURL = server.URL + "/copilot_internal/v2/token"
	provider.cachedToken = "stale"
	provider.expiresAt = time.Now().Unix() - 1

	req := types.ChatRequest{
		Model:    "gpt-4.1",
		Messages: []types.Message{{Role: "user", Content: "hello"}},
	}
	if _, err := provider.Chat(context.Background(), req); err != nil {
		t.Fatalf("first Chat() error = %v", err)
	}
	if _, err := provider.Chat(context.Background(), req); err != nil {
		t.Fatalf("second Chat() error = %v", err)
	}

	if tokenCalls != 1 {
		t.Fatalf("expected token exchange to be called once, got %d", tokenCalls)
	}
}
