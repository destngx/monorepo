package openai

import (
	"encoding/json"
	"strings"
	"testing"

	"apps/ai-gateway/internal/domain"
)

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
