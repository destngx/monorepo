package openai

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"apps/ai-gateway/internal/domain"
)

func TestResponsesRequestFromChatPreservesInstructionsAndCacheOptions(t *testing.T) {
	req := domain.ChatRequest{
		Model: "gpt-5.6-terra",
		Messages: []domain.Message{
			{Role: domain.RoleSystem, Content: "stable instructions"},
			{Role: domain.RoleUser, Parts: []domain.ContentPart{{Type: "text", Text: "question", PromptCacheBreakpoint: &domain.CacheBreakpoint{Mode: "explicit"}}}},
		},
		PromptCacheKey:     "cc:test",
		PromptCacheOptions: &domain.PromptCacheOptions{Mode: "explicit"},
	}

	payload := responsesRequestFromChat(req, false).CloneBody()
	if payload["instructions"] != "stable instructions" {
		t.Fatalf("instructions not preserved: %#v", payload["instructions"])
	}
	if payload["prompt_cache_key"] != "cc:test" {
		t.Fatalf("cache key not preserved: %#v", payload["prompt_cache_key"])
	}
	input, ok := payload["input"].([]any)
	if !ok || len(input) != 1 {
		t.Fatalf("unexpected input: %#v", payload["input"])
	}
	encoded, err := json.Marshal(input[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) == "" {
		t.Fatal("empty input payload")
	}
}

func TestResponsesToolsPreservesHostedWebSearch(t *testing.T) {
	tools := responsesToolsFromChat([]domain.Tool{{Type: domain.ToolTypeWebSearch}})
	if len(tools) != 1 {
		t.Fatalf("expected one hosted tool, got %#v", tools)
	}
	tool, ok := tools[0].(map[string]any)
	if !ok || tool["type"] != domain.ToolTypeWebSearch {
		t.Fatalf("unexpected hosted web-search tool: %#v", tools[0])
	}
}

func TestResponsesRequestFromChatIncludesHostedWebSearchSources(t *testing.T) {
	payload := responsesRequestFromChat(domain.ChatRequest{
		Model: "gpt-5.6-luna",
		Tools: []domain.Tool{{Type: domain.ToolTypeWebSearch}},
	}, true).CloneBody()

	include, ok := payload[responsesFieldInclude].([]string)
	if !ok || len(include) != 1 || include[0] != responsesIncludeWebSearchSources {
		t.Fatalf("expected web-search sources include, got %#v", payload[responsesFieldInclude])
	}
}

func TestProxyResponsesAsChatSSEEmitsHostedWebSearchToolCall(t *testing.T) {
	body := "data: {\"type\":\"response.web_search_call.searching\",\"item_id\":\"ws_1\",\"action\":{\"query\":\"Node.js LTS\"}}\n\n" +
		"data: {\"type\":\"response.output_item.done\",\"item\":{\"id\":\"ws_1\",\"type\":\"web_search_call\",\"action\":{\"queries\":[\"Node.js LTS\"],\"sources\":[{\"type\":\"url\",\"url\":\"https://nodejs.org/en/about/previous-releases\",\"title\":\"Node.js releases\"}]}}}\n\n" +
		"data: {\"type\":\"response.output_text.delta\",\"delta\":\"Node.js 22 is LTS.\"}\n\n" +
		"data: {\"type\":\"response.completed\",\"response\":{\"usage\":{\"total_tokens\":1}}}\n\n"
	var output bytes.Buffer
	if _, err := proxyResponsesAsChatSSE(bytes.NewBufferString(body), &output); err != nil {
		t.Fatalf("proxy failed: %v", err)
	}
	if !strings.Contains(output.String(), `"web_search":{"id":"ws_1","query":"Node.js LTS"}`) {
		t.Fatalf("expected hosted web-search tool call, got %s", output.String())
	}
	if !strings.Contains(output.String(), `"web_search_result":{"id":"ws_1","sources":[{"title":"Node.js releases","type":"url","url":"https://nodejs.org/en/about/previous-releases"}]}`) {
		t.Fatalf("expected hosted web-search sources, got %s", output.String())
	}
}

func TestChatResponseFromResponsesParsesOutputAndCacheUsage(t *testing.T) {
	raw := map[string]any{
		"id":          "resp_123",
		"output_text": "answer",
		"usage": map[string]any{
			"input_tokens": float64(100), "output_tokens": float64(20), "total_tokens": float64(120),
			"input_tokens_details": map[string]any{"cached_tokens": float64(80), "cache_write_tokens": float64(10)},
		},
	}
	resp := chatResponseFromResponses(raw, "gpt-5.6-terra")
	if resp.Choices[0].Message.Content != "answer" {
		t.Fatalf("unexpected content: %#v", resp.Choices[0].Message.Content)
	}
	if resp.Usage.PromptTokensDetails == nil || resp.Usage.PromptTokensDetails.CachedTokens != 80 || resp.Usage.PromptTokensDetails.CacheWriteTokens != 10 {
		t.Fatalf("unexpected usage: %#v", resp.Usage)
	}
}

func TestResponsesConversionPreservesFunctionCallsAndResults(t *testing.T) {
	req := domain.ChatRequest{Model: "gpt-5.6-terra", Messages: []domain.Message{
		{Role: domain.RoleAssistant, ToolCalls: []domain.ToolCall{{ID: "call_1", Type: domain.ToolTypeFunction, Function: &domain.FunctionCall{Name: "lookup", Arguments: `{"q":"x"}`}}}},
		{Role: domain.RoleTool, ToolCallID: "call_1", Content: "result"},
	}, Tools: []domain.Tool{{Type: domain.ToolTypeFunction, Function: &domain.FunctionDefinition{Name: "lookup", Description: "look up", Parameters: map[string]any{"type": "object"}}}},
	}
	payload := responsesRequestFromChat(req, false).CloneBody()
	input := payload["input"].([]any)
	call := input[0].(map[string]any)
	if call["type"] != "function_call" || call["call_id"] != "call_1" {
		t.Fatalf("unexpected function call: %#v", call)
	}
	result := input[1].(map[string]any)
	if result["type"] != "function_call_output" || result["output"] != "result" {
		t.Fatalf("unexpected function result: %#v", result)
	}
	tools := payload["tools"].([]any)
	tool := tools[0].(map[string]any)
	if tool["name"] != "lookup" || tool["type"] != "function" {
		t.Fatalf("unexpected tool: %#v", tool)
	}
}

func TestChatResponseFromResponsesParsesFunctionCall(t *testing.T) {
	resp := chatResponseFromResponses(map[string]any{
		"id": "resp_1", "output": []any{map[string]any{"type": "function_call", "call_id": "call_1", "name": "lookup", "arguments": `{"q":"x"}`}},
	}, "gpt-5.6-terra")
	if len(resp.Choices[0].Message.ToolCalls) != 1 {
		t.Fatalf("expected one tool call: %#v", resp)
	}
	call := resp.Choices[0].Message.ToolCalls[0]
	if call.ID != "call_1" || call.Function == nil || call.Function.Name != "lookup" {
		t.Fatalf("unexpected tool call: %#v", call)
	}
}

func TestCodexRequestPreservesPromptCacheConfiguration(t *testing.T) {
	req := domain.ChatRequest{Model: "gpt-5.6-luna", PromptCacheKey: "cc:test", PromptCacheOptions: &domain.PromptCacheOptions{Mode: "explicit"}, Messages: []domain.Message{{Role: domain.RoleUser, Parts: []domain.ContentPart{{Type: "text", Text: "stable", PromptCacheBreakpoint: &domain.CacheBreakpoint{Mode: "explicit"}}}}}}
	payload := toCodexResponseRequest(req)
	first, ok := payload.Input[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected Codex input: %#v", payload.Input[0])
	}
	content, ok := first["content"].([]map[string]any)
	if !ok || len(content) == 0 || content[0]["type"] != "input_text" || content[0]["text"] != "stable" {
		t.Fatalf("unexpected Codex content: %#v", payload.Input[0])
	}
}
