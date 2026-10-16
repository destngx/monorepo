package ai

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestChatRunsToolsAndStreams(t *testing.T) {
	client := &scriptedClient{replies: []reply{
		{toolCalls: []domain.AIToolCall{{ID: "c1", Name: "getRecentTransactions", Arguments: `{"payee":"grab"}`}}},
		{content: "You spent 120.000 ₫ on Grab."},
	}}
	var events []port.ChatEvent
	err := newTestService(t, fixtureDatabase(), client, nil).Chat(context.Background(), port.ChatRequest{
		Messages: []port.ChatMessage{{Role: "user", Content: "How much on Grab?"}},
		Page:     &port.ChatPageContext{Path: "/transactions", Title: "Transactions"},
	}, func(event port.ChatEvent) error {
		events = append(events, event)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	var text strings.Builder
	kinds := []string{}
	for _, event := range events {
		if event.Type == "text" {
			text.WriteString(event.Delta)
			continue
		}
		kinds = append(kinds, event.Type+":"+event.Status)
	}
	if strings.Join(kinds, ",") != "tool:running,tool:ok,done:" || text.String() != "You spent 120.000 ₫ on Grab." {
		t.Fatalf("unexpected events %v %q", kinds, text.String())
	}
	if len(client.requests[0].Tools) != 7 || !strings.Contains(client.requests[0].Messages[0].Content, "/transactions") {
		t.Fatalf("first turn missing tools or page context")
	}
	second := client.requests[1].Messages
	toolResult := second[len(second)-1]
	if toolResult.Role != domain.AIRoleTool || toolResult.ToolCallID != "c1" || !strings.Contains(toolResult.Content, `"total":1`) {
		t.Fatalf("tool result not passed back: %+v", toolResult)
	}

	err = newTestService(t, fixtureDatabase(), client, nil).Chat(context.Background(), port.ChatRequest{Messages: []port.ChatMessage{{Role: "assistant", Content: "hi"}}}, func(port.ChatEvent) error { return nil })
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("chat must end with a user message, got %v", err)
	}
}

func TestChatFallsBackOnlyBeforeStreaming(t *testing.T) {
	client := &scriptedClient{replies: []reply{{err: domain.ErrAIUnavailable}, {content: "Hello there"}}}
	err := newTestService(t, fixtureDatabase(), client, nil).Chat(context.Background(), port.ChatRequest{Messages: []port.ChatMessage{{Role: "user", Content: "hi"}}}, func(port.ChatEvent) error { return nil })
	if err != nil || strings.Join(client.models(), ",") != "primary,fallback" {
		t.Fatalf("expected fallback before streaming: %v %v", err, client.models())
	}
}
