package openai

import (
	"apps/wealth-management/domain"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return NewClient(domain.AIConfig{BaseURL: server.URL + "/v1/", APIKey: "key", Provider: "openai"}, server.Client())
}

func TestCompleteSendsOpenAIRequestAndDecodesToolCalls(t *testing.T) {
	var body map[string]any
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer key" || r.Header.Get(providerHeader) != "openai" {
			t.Errorf("unexpected request %s %v", r.URL.Path, r.Header)
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		_, _ = io.WriteString(w, `{"model":"m-1","choices":[{"message":{"content":null,"tool_calls":[{"id":"c1","type":"function","function":{"name":"getBudgetStatus","arguments":"{}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":10,"completion_tokens":2}}`)
	})

	response, err := client.Complete(context.Background(), domain.AIRequest{
		Model: "m",
		JSON:  true,
		Messages: []domain.AIMessage{
			{Role: domain.AIRoleSystem, Content: "sys"},
			{Role: domain.AIRoleUser, Content: "look", Images: []domain.AIImage{{MediaType: "image/png", Data: []byte{1, 2}}}},
		},
		Tools: []domain.AITool{{Name: "getBudgetStatus", Parameters: map[string]any{"type": "object"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Model != "m-1" || len(response.ToolCalls) != 1 || response.ToolCalls[0].Name != "getBudgetStatus" || response.Usage.PromptTokens != 10 {
		t.Fatalf("unexpected response %+v", response)
	}
	if body["response_format"].(map[string]any)["type"] != "json_object" {
		t.Fatalf("json mode not requested: %v", body["response_format"])
	}
	parts := body["messages"].([]any)[1].(map[string]any)["content"].([]any)
	image := parts[1].(map[string]any)["image_url"].(map[string]any)["url"].(string)
	if !strings.HasPrefix(image, "data:image/png;base64,AQI=") {
		t.Fatalf("image not sent as data URL: %s", image)
	}
}

func TestStreamAssemblesContentAndToolCalls(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		for _, chunk := range []string{
			`{"choices":[{"delta":{"content":"Hel"}}]}`,
			`{"choices":[{"delta":{"content":"lo"}}]}`,
			`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"c1","function":{"name":"webSearch","arguments":"{\"q"}}]}}]}`,
			`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\":\"vn\"}"}}]},"finish_reason":"tool_calls"}]}`,
			`[DONE]`,
		} {
			_, _ = io.WriteString(w, "data: "+chunk+"\n\n")
		}
	})

	var deltas []string
	response, err := client.Stream(context.Background(), domain.AIRequest{Model: "m"}, func(delta string) error {
		deltas = append(deltas, delta)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Content != "Hello" || strings.Join(deltas, "|") != "Hel|lo" {
		t.Fatalf("unexpected content %q %v", response.Content, deltas)
	}
	if len(response.ToolCalls) != 1 || response.ToolCalls[0].Arguments != `{"q":"vn"}` || response.FinishReason != "tool_calls" {
		t.Fatalf("unexpected tool calls %+v", response)
	}
}

func TestProviderErrorsAreUnavailable(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":{"message":"model not found"}}`)
	})
	_, err := client.Complete(context.Background(), domain.AIRequest{Model: "nope"})
	if !errors.Is(err, domain.ErrAIUnavailable) || !strings.Contains(err.Error(), "model not found") {
		t.Fatalf("unexpected error %v", err)
	}
}
