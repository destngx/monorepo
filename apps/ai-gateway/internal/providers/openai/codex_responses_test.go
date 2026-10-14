package openai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"apps/ai-gateway/config"
	"apps/ai-gateway/internal/domain"
)

func newCodexTestProvider(t *testing.T, sse string, captured *map[string]any) *Provider {
	t.Helper()
	t.Setenv("OPENAI_OAUTH_PATH", "/nonexistent/openai-auth.json")
	t.Setenv("CODEX_AUTH_PATH", "/nonexistent/codex-auth.json")
	t.Setenv("HOME", t.TempDir())
	p := New("", &config.OpenAIOAuth{AccessToken: "oauth-token"})
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/backend-api/codex/responses" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, captured)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{headerContentType: []string{"text/event-stream"}},
			Body:       io.NopCloser(strings.NewReader(sse)),
		}, nil
	})
	p.client = &http.Client{Transport: transport}
	p.streamClient = &http.Client{Transport: transport}
	return p
}

func sseData(events ...string) string {
	var b strings.Builder
	for _, event := range events {
		b.WriteString("data: " + event + "\n\n")
	}
	return b.String()
}

// The ChatGPT Codex backend rejects stream=false, so a non-streaming Responses
// call is streamed upstream and the final response is assembled.
func TestCodexResponsesStreamsUpstreamAndAssemblesTheResponse(t *testing.T) {
	var body map[string]any
	p := newCodexTestProvider(t, sseData(
		`{"type":"response.created","response":{"id":"resp_1","status":"in_progress","output":[]}}`,
		`{"type":"response.output_item.done","output_index":1,"item":{"type":"function_call","call_id":"c1","name":"lookup","arguments":"{}"}}`,
		`{"type":"response.output_text.delta","output_index":0,"delta":"Hi"}`,
		`{"type":"response.output_item.done","output_index":0,"item":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Hi"}]}}`,
		`{"type":"response.completed","response":{"id":"resp_1","model":"gpt-6-luna","status":"completed","output":[],"usage":{"input_tokens":10,"output_tokens":2,"total_tokens":12}}}`,
	), &body)

	resp, err := p.Responses(context.Background(), domain.ResponsesRequest{Model: "gpt-6-luna", Body: map[string]any{"input": "hi"}})
	if err != nil {
		t.Fatalf("responses error: %v", err)
	}
	if body["stream"] != true {
		t.Fatalf("expected the upstream request to stream, got %v", body["stream"])
	}
	result := map[string]any(*resp)
	if result["id"] != "resp_1" || result["status"] != "completed" {
		t.Fatalf("unexpected response %v", result)
	}
	output, _ := result["output"].([]any)
	if len(output) != 2 {
		t.Fatalf("expected output items from the stream in index order, got %v", result["output"])
	}
	if output[0].(map[string]any)["type"] != "message" || output[1].(map[string]any)["type"] != "function_call" {
		t.Fatalf("output items out of order: %v", output)
	}
	if usage := domain.UsageFromResponsesValue(result); usage.TotalTokens != 12 {
		t.Fatalf("usage not preserved: %+v", usage)
	}
}

func TestCodexResponsesKeepsTheCompletedOutputWhenPresent(t *testing.T) {
	var body map[string]any
	p := newCodexTestProvider(t, sseData(
		`{"type":"response.output_item.done","output_index":0,"item":{"type":"message","content":[{"type":"output_text","text":"stale"}]}}`,
		`{"type":"response.completed","response":{"id":"resp_2","status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"final"}]}]}}`,
	), &body)

	resp, err := p.Responses(context.Background(), domain.ResponsesRequest{Model: "m"})
	if err != nil {
		t.Fatalf("responses error: %v", err)
	}
	output := map[string]any(*resp)["output"].([]any)
	text := output[0].(map[string]any)["content"].([]any)[0].(map[string]any)["text"]
	if len(output) != 1 || text != "final" {
		t.Fatalf("expected the completed output, got %v", output)
	}
}

func TestCodexResponsesReportsStreamFailures(t *testing.T) {
	cases := map[string]string{
		"failed":     sseData(`{"type":"response.failed","response":{"status":"failed","error":{"code":"server_error","message":"upstream broke"}}}`),
		"error":      sseData(`{"type":"error","message":"rate limited"}`),
		"incomplete": sseData(`{"type":"response.output_text.delta","delta":"Hi"}`),
	}
	want := map[string]string{"failed": "upstream broke", "error": "rate limited", "incomplete": "ended before"}
	for name, sse := range cases {
		t.Run(name, func(t *testing.T) {
			var body map[string]any
			p := newCodexTestProvider(t, sse, &body)
			_, err := p.Responses(context.Background(), domain.ResponsesRequest{Model: "m"})
			if err == nil || !strings.Contains(err.Error(), want[name]) {
				t.Fatalf("expected error containing %q, got %v", want[name], err)
			}
		})
	}
}
