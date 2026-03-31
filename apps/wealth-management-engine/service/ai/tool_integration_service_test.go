package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestGivenUserPromptWhenRunConversationThenReturnsMultiTurnToolFlow(t *testing.T) {
	service := NewToolIntegrationService(
		&fakeBalanceTool{value: 12345.67},
		&fakeMarketSkill{summary: "Momentum is positive with moderate volatility."},
	)

	conversation, err := service.RunConversation(context.Background(), "Check cash balance and market outlook for BTC")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(conversation.Turns) < 6 {
		t.Fatalf("expected at least 6 turns, got %d", len(conversation.Turns))
	}

	firstToolCall := conversation.Turns[1].ToolCallJSON
	if !json.Valid([]byte(firstToolCall)) {
		t.Fatalf("expected valid tool call json, got %s", firstToolCall)
	}
	if !strings.Contains(firstToolCall, "\"GetBalance\"") {
		t.Fatalf("expected GetBalance tool call, got %s", firstToolCall)
	}

	secondToolCall := conversation.Turns[3].ToolCallJSON
	if !json.Valid([]byte(secondToolCall)) {
		t.Fatalf("expected valid tool call json, got %s", secondToolCall)
	}
	if !strings.Contains(secondToolCall, "\"MarketAnalysis\"") {
		t.Fatalf("expected MarketAnalysis tool call, got %s", secondToolCall)
	}

	finalMessage := conversation.Turns[len(conversation.Turns)-1].Content
	if !strings.Contains(finalMessage, "Market analysis for BTC") {
		t.Fatalf("expected synthesized assistant response, got %s", finalMessage)
	}
}

type fakeBalanceTool struct {
	value float64
}

func (f *fakeBalanceTool) GetBalance(_ context.Context, _ string) (float64, error) {
	return f.value, nil
}

type fakeMarketSkill struct {
	summary string
}

func (f *fakeMarketSkill) Analyze(_ context.Context, _ string) (string, error) {
	return f.summary, nil
}
