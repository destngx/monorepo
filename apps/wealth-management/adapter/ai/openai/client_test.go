package openai

import (
	"apps/wealth-management/domain"
	"bytes"
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
	return NewClient(domain.AIConfig{BaseURL: server.URL + "/v1/", APIKey: "key"}, server.Client())
}

func streamEvents(w http.ResponseWriter, events ...string) {
	w.Header().Set("Content-Type", "text/event-stream")
	for _, event := range events {
		var typed struct {
			Type string `json:"type"`
		}
		_ = json.Unmarshal([]byte(event), &typed)
		var compact bytes.Buffer
		_ = json.Compact(&compact, []byte(event))
		_, _ = io.WriteString(w, "event: "+typed.Type+"\ndata: "+compact.String()+"\n\n")
	}
}

func TestCompleteSendsResponsesRequestAndDecodesToolCalls(t *testing.T) {
	var body map[string]any
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" || r.Header.Get("Authorization") != "Bearer key" || r.Header.Get(providerHeader) != "openai" {
			t.Errorf("unexpected request %s %v", r.URL.Path, r.Header)
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		_, _ = io.WriteString(w, `{"model":"m-1","status":"completed","output":[
			{"type":"reasoning","summary":[]},
			{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Checking"}]},
			{"type":"function_call","call_id":"c1","name":"getBudgetStatus","arguments":"{}"}
		],"usage":{"input_tokens":10,"output_tokens":2}}`)
	})

	response, err := client.Complete(context.Background(), domain.AIRequest{
		Model:     "m",
		JSON:      true,
		MaxTokens: 300,
		Messages: []domain.AIMessage{
			{Role: domain.AIRoleSystem, Content: "sys"},
			{Role: domain.AIRoleUser, Content: "look", Images: []domain.AIImage{{MediaType: "image/png", Data: []byte{1, 2}}}},
			{Role: domain.AIRoleAssistant, ToolCalls: []domain.AIToolCall{{ID: "c0", Name: "webSearch", Arguments: `{"q":"vn"}`}}},
			{Role: domain.AIRoleTool, ToolCallID: "c0", Content: ""},
		},
		Tools: []domain.AITool{{Name: "getBudgetStatus", Description: "budget", Parameters: map[string]any{"type": "object"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Model != "m-1" || response.Content != "Checking" || response.FinishReason != "tool_calls" {
		t.Fatalf("unexpected response %+v", response)
	}
	if len(response.ToolCalls) != 1 || response.ToolCalls[0] != (domain.AIToolCall{ID: "c1", Name: "getBudgetStatus", Arguments: "{}"}) {
		t.Fatalf("unexpected tool calls %+v", response.ToolCalls)
	}
	if response.Usage != (domain.AIUsage{PromptTokens: 10, CompletionTokens: 2}) {
		t.Fatalf("unexpected usage %+v", response.Usage)
	}

	if body["text"].(map[string]any)["format"].(map[string]any)["type"] != "json_object" {
		t.Fatalf("json mode not requested: %v", body["text"])
	}
	if body["max_output_tokens"] != float64(300) || body["store"] != false || body["messages"] != nil {
		t.Fatalf("unexpected request fields %v", body)
	}
	tool := body["tools"].([]any)[0].(map[string]any)
	if tool["type"] != "function" || tool["name"] != "getBudgetStatus" || tool["strict"] != false || tool["parameters"] == nil {
		t.Fatalf("unexpected tool %v", tool)
	}

	input := body["input"].([]any)
	if len(input) != 4 {
		t.Fatalf("unexpected input %v", input)
	}
	system := input[0].(map[string]any)
	if system["role"] != "system" || system["content"].([]any)[0].(map[string]any)["text"] != "sys" {
		t.Fatalf("unexpected system item %v", system)
	}
	parts := input[1].(map[string]any)["content"].([]any)
	if parts[0].(map[string]any)["type"] != "input_text" || parts[1].(map[string]any)["type"] != "input_image" {
		t.Fatalf("unexpected user parts %v", parts)
	}
	if image := parts[1].(map[string]any)["image_url"].(string); image != "data:image/png;base64,AQI=" {
		t.Fatalf("image not sent as data URL: %s", image)
	}
	call := input[2].(map[string]any)
	if call["type"] != "function_call" || call["call_id"] != "c0" || call["name"] != "webSearch" || call["arguments"] != `{"q":"vn"}` {
		t.Fatalf("unexpected function call item %v", call)
	}
	output := input[3].(map[string]any)
	if output["type"] != "function_call_output" || output["call_id"] != "c0" || output["output"] != "" {
		t.Fatalf("unexpected function call output item %v", output)
	}
}

func TestCompleteSendsAssistantTextAsOutputText(t *testing.T) {
	var body map[string]any
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		_, _ = io.WriteString(w, `{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"ok"}]}]}`)
	})

	response, err := client.Complete(context.Background(), domain.AIRequest{Model: "m", Messages: []domain.AIMessage{
		{Role: domain.AIRoleUser, Content: "hi"},
		{Role: domain.AIRoleAssistant, Content: "hello"},
		{Role: domain.AIRoleUser, Content: "again"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if response.Model != "m" || response.Content != "ok" || response.FinishReason != "stop" {
		t.Fatalf("unexpected response %+v", response)
	}
	assistant := body["input"].([]any)[1].(map[string]any)
	if assistant["role"] != "assistant" || assistant["content"].([]any)[0].(map[string]any)["type"] != "output_text" {
		t.Fatalf("unexpected assistant item %v", assistant)
	}
	if _, ok := body["text"]; ok {
		t.Fatalf("text format sent without JSON mode: %v", body["text"])
	}
}

func TestCompleteMapsIncompleteToLength(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},"output":[{"type":"message","content":[{"type":"output_text","text":"cut"}]}]}`)
	})
	response, err := client.Complete(context.Background(), domain.AIRequest{Model: "m"})
	if err != nil {
		t.Fatal(err)
	}
	if response.Content != "cut" || response.FinishReason != "length" {
		t.Fatalf("unexpected response %+v", response)
	}
}

func TestCompleteFailedResponseIsUnavailable(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"status":"failed","error":{"code":"server_error","message":"overloaded"},"output":[]}`)
	})
	_, err := client.Complete(context.Background(), domain.AIRequest{Model: "m"})
	if !errors.Is(err, domain.ErrAIUnavailable) || !strings.Contains(err.Error(), "overloaded") {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestStreamAssemblesContentAndToolCalls(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		if r.URL.Path != "/v1/responses" || body["stream"] != true || r.Header.Get("Accept") != "text/event-stream" {
			t.Errorf("unexpected stream request %s %v", r.URL.Path, body)
		}
		streamEvents(w,
			`{"type":"response.created","response":{"model":"m-2","status":"in_progress","output":[]}}`,
			`{"type":"response.output_item.added","output_index":0,"item":{"type":"message","role":"assistant","content":[]}}`,
			`{"type":"response.output_text.delta","output_index":0,"content_index":0,"delta":"Hel"}`,
			`{"type":"response.output_text.delta","output_index":0,"content_index":0,"delta":"lo"}`,
			`{"type":"response.output_item.added","output_index":1,"item":{"type":"function_call","call_id":"c1","name":"webSearch","arguments":""}}`,
			`{"type":"response.function_call_arguments.delta","output_index":1,"delta":"{\"q"}`,
			`{"type":"response.function_call_arguments.delta","output_index":1,"delta":"\":\"vn\"}"}`,
			`{"type":"response.output_item.added","output_index":2,"item":{"type":"function_call","call_id":"c2","name":"getBudgetStatus","arguments":""}}`,
			`{"type":"response.output_item.done","output_index":2,"item":{"type":"function_call","call_id":"c2","name":"getBudgetStatus","arguments":"{}"}}`,
			`{"type":"response.completed","response":{"model":"m-2","status":"completed","output":[],"usage":{"input_tokens":7,"output_tokens":3}}}`,
		)
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
	want := []domain.AIToolCall{{ID: "c1", Name: "webSearch", Arguments: `{"q":"vn"}`}, {ID: "c2", Name: "getBudgetStatus", Arguments: "{}"}}
	if len(response.ToolCalls) != 2 || response.ToolCalls[0] != want[0] || response.ToolCalls[1] != want[1] {
		t.Fatalf("unexpected tool calls %+v", response.ToolCalls)
	}
	if response.Model != "m-2" || response.FinishReason != "tool_calls" || response.Usage != (domain.AIUsage{PromptTokens: 7, CompletionTokens: 3}) {
		t.Fatalf("unexpected response %+v", response)
	}
}

func TestStreamFallsBackToCompletedOutput(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		streamEvents(w,
			`{"type":"response.completed","response":{"status":"completed","output":[
				{"type":"message","content":[{"type":"output_text","text":"All at once"}]},
				{"type":"function_call","call_id":"c1","name":"webSearch","arguments":"{}"}
			]}}`,
		)
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	})

	var deltas []string
	response, err := client.Stream(context.Background(), domain.AIRequest{Model: "m"}, func(delta string) error {
		deltas = append(deltas, delta)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Content != "All at once" || strings.Join(deltas, "|") != "All at once" {
		t.Fatalf("unexpected content %q %v", response.Content, deltas)
	}
	if len(response.ToolCalls) != 1 || response.ToolCalls[0].ID != "c1" || response.Model != "m" {
		t.Fatalf("unexpected response %+v", response)
	}
}

func TestStreamFailuresAreUnavailable(t *testing.T) {
	cases := map[string][]string{
		"failed response": {`{"type":"response.failed","response":{"status":"failed","error":{"code":"server_error","message":"overloaded"}}}`},
		"gateway failure": {`{"type":"response.failed","error":"overloaded","status":502}`},
		"error event":     {`{"type":"error","code":"server_error","message":"overloaded"}`},
		"no terminal":     {`{"type":"response.output_text.delta","output_index":0,"delta":"overloaded"}`},
	}
	for name, events := range cases {
		t.Run(name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				streamEvents(w, events...)
			})
			_, err := client.Stream(context.Background(), domain.AIRequest{Model: "m"}, func(string) error { return nil })
			if !errors.Is(err, domain.ErrAIUnavailable) {
				t.Fatalf("unexpected error %v", err)
			}
			if name != "no terminal" && !strings.Contains(err.Error(), "overloaded") {
				t.Fatalf("provider message lost: %v", err)
			}
		})
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
