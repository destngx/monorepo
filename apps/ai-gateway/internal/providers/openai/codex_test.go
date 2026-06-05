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

func ptrInt(v int) *int {
	return &v
}
