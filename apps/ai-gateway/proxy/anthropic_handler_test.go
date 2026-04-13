package proxy

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"apps/ai-gateway/config"
	"apps/ai-gateway/providers"
	"apps/ai-gateway/types"
)

func TestConvertFromAnthropicRequest(t *testing.T) {
	anthroReq := types.AnthropicRequest{
		Model:     "claude-3-opus",
		MaxTokens: 1024,
		System:    "You are a test assistant.",
		Messages: []types.AnthropicMessage{
			{Role: "user", Content: "Hello!"},
			{Role: "assistant", Content: []any{
				map[string]any{"type": "text", "text": "I can help."},
				map[string]any{"type": "tool_use", "id": "t1", "name": "get_weather", "input": map[string]any{"city": "Paris"}},
			}},
		},
	}

	req := convertFromAnthropicRequest(anthroReq, "anthropic")

	if req.Model != "claude-3-opus" {
		t.Errorf("expected model claude-3-opus, got %s", req.Model)
	}
	if *req.MaxTokens != 1024 {
		t.Errorf("expected max_tokens 1024, got %d", *req.MaxTokens)
	}

	// Should have system message + user message + assistant message
	if len(req.Messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(req.Messages))
	}

	if req.Messages[0].Role != "system" || req.Messages[0].Content != "You are a test assistant." {
		t.Errorf("system message mismatch")
	}
	if req.Messages[2].Role != "assistant" || len(req.Messages[2].ToolCalls) != 1 {
		t.Errorf("assistant tool call mismatch")
	}
}

func TestConvertFromAnthropicRequest_WithArraySystem(t *testing.T) {
	anthroReq := types.AnthropicRequest{
		Model:     "claude-3-opus",
		MaxTokens: 1024,
		System: []any{
			map[string]any{"type": "text", "text": "Part 1."},
			map[string]any{"type": "text", "text": "Part 2."},
		},
		Messages: []types.AnthropicMessage{
			{Role: "user", Content: "Hello!"},
		},
	}

	req := convertFromAnthropicRequest(anthroReq, "anthropic")

	// Should have system message + user message
	if len(req.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(req.Messages))
	}

	if req.Messages[0].Role != "system" {
		t.Errorf("expected role system, got %s", req.Messages[0].Role)
	}

	expectedContent := "Part 1.\nPart 2."
	if req.Messages[0].Content != expectedContent {
		t.Errorf("expected content %q, got %q", expectedContent, req.Messages[0].Content)
	}
}

func TestConvertToAnthropicResponse(t *testing.T) {
	openaiResp := &types.ChatResponse{
		ID:    "res_123",
		Model: "gpt-4",
		Choices: []types.Choice{
			{
				Message: types.Message{
					Role:    "assistant",
					Content: "Hello from OpenAI!",
					ToolCalls: []types.ToolCall{
						{
							ID:   "tc_1",
							Type: "function",
							Function: types.FunctionCall{
								Name:      "test_tool",
								Arguments: `{"arg1": "val1"}`,
							},
						},
					},
				},
				FinishReason: "tool_calls",
			},
		},
		Usage: types.Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}

	anthroResp := convertToAnthropicResponse(openaiResp)

	if anthroResp.ID != "res_123" {
		t.Errorf("id mismatch")
	}
	if len(anthroResp.Content) != 2 {
		t.Errorf("expected 2 content blocks, got %d", len(anthroResp.Content))
	}
	if anthroResp.Content[0].Type != "text" || anthroResp.Content[0].Text != "Hello from OpenAI!" {
		t.Errorf("content block 0 mismatch")
	}
	if anthroResp.Content[1].Type != "tool_use" || anthroResp.Content[1].Name != "test_tool" {
		t.Errorf("content block 1 mismatch")
	}
	if anthroResp.StopReason != "tool_use" {
		t.Errorf("stop reason mismatch")
	}
}

func TestAnthropicHandler_Routing(t *testing.T) {
	// Simple test to ensure handler doesn't panic and routes correctly
	// Note: Fully testing SSE would require more setup

	registry := &Registry{
		providers: make(map[string]providers.Provider),
		Config:    &config.Config{},
		Mapper:    NewModelMapper(),
	}
	// We'd need to register a mock provider here for a full integration test
	handler := NewAnthropicHandler(registry)

	reqBody, _ := json.Marshal(types.AnthropicRequest{
		Model:     "test",
		MaxTokens: 10,
		Messages:  []types.AnthropicMessage{{Role: "user", Content: "hi"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(reqBody))
	req.Header.Set("X-AI-Provider", "non-existent")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unknown provider, got %d", w.Code)
	}
}
