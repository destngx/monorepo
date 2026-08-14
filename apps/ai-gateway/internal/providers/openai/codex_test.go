package openai

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"apps/ai-gateway/internal/domain"
)

func TestCodexRequestIncludesHostedWebSearchSources(t *testing.T) {
	payload := toCodexResponseRequest(domain.ChatRequest{
		Model: "gpt-5.6-luna",
		Tools: []domain.Tool{{Type: domain.ToolTypeWebSearch}},
	})

	if len(payload.Include) != 1 || payload.Include[0] != responsesIncludeWebSearchSources {
		t.Fatalf("expected web-search sources include, got %#v", payload.Include)
	}
}

func TestParseCodexStreamEmitsIncludedWebSearchSources(t *testing.T) {
	input := "data: {\"type\":\"response.web_search_call.searching\",\"item_id\":\"ws_1\",\"action\":{\"query\":\"Node.js LTS\"}}\n\n" +
		"data: {\"type\":\"response.output_item.done\",\"item\":{\"id\":\"ws_1\",\"type\":\"web_search_call\",\"action\":{\"type\":\"search\",\"queries\":[\"Node.js LTS\"],\"sources\":[{\"type\":\"url\",\"url\":\"https://nodejs.org/en/about/previous-releases\",\"title\":\"Node.js releases\"}]}}}\n\n"
	var output bytes.Buffer

	if _, _, _, _, _, _, err := parseCodexStream(strings.NewReader(input), &output); err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if !strings.Contains(output.String(), `"web_search_result":{"id":"ws_1","sources":[{"title":"Node.js releases","type":"url","url":"https://nodejs.org/en/about/previous-releases"}]}`) {
		t.Fatalf("expected included sources in stream, got %s", output.String())
	}
}

func TestLimitWebSearchSourcesCapsAtEight(t *testing.T) {
	sources := make([]any, 10)
	limited, ok := limitWebSearchSources(sources).([]any)
	if !ok || len(limited) != 8 {
		t.Fatalf("expected 8 sources, got %#v", limited)
	}
}

func TestToCodexResponseRequestOmitsMaxOutputTokens(t *testing.T) {
	req := domain.ChatRequest{
		Model: "gpt-5.4-mini",
		Messages: []domain.Message{
			{Role: domain.RoleSystem, Content: "system instructions"},
			{Role: domain.RoleUser, Content: "hello"},
		},
		MaxTokens:           ptrInt(123),
		MaxCompletionTokens: ptrInt(456),
	}

	payload := toCodexResponseRequest(req)
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if strings.Contains(string(body), "max_output_tokens") {
		t.Fatalf("expected codex payload to omit max_output_tokens, got %s", string(body))
	}
}

func TestCodexCompatibleResponsesRequestOmitsUnsupportedMaxOutputTokens(t *testing.T) {
	req := domain.ResponsesRequest{
		Model:  "gpt-5.6-luna",
		Stream: true,
		Body: map[string]any{
			"model":             "gpt-5.6-luna",
			"stream":            true,
			"store":             true,
			"max_output_tokens": 8000,
			"input":             []any{},
		},
	}

	payload := codexCompatibleResponsesRequest(req).CloneBody()
	if _, exists := payload["max_output_tokens"]; exists {
		t.Fatalf("expected Codex Responses payload to omit max_output_tokens: %#v", payload)
	}
	if payload["store"] != false {
		t.Fatalf("expected Codex Responses payload to force store=false: %#v", payload)
	}
	if _, exists := req.Body["max_output_tokens"]; !exists {
		t.Fatal("expected request normalization not to mutate the original body")
	}
}

func ptrInt(v int) *int {
	return &v
}
