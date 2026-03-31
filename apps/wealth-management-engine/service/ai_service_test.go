package service

import (
	"apps/wealth-management-engine/domain"
	"context"
	"io"
	"strings"
	"testing"
)

func TestGivenEmptyModelWhenStreamThenUsesDefaultModel(t *testing.T) {
	client := &fakeAIClient{}
	service := NewAIService(client, "gpt-4.1")

	stream, err := service.Stream(context.Background(), "hello", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	_ = stream.Close()

	if client.lastModel != "gpt-4.1" {
		t.Fatalf("expected default model gpt-4.1, got %s", client.lastModel)
	}
}

type fakeAIClient struct {
	lastModel string
	lastMsgs  []domain.RoleMessage
	jsonReply string
}

func (f *fakeAIClient) StreamCompletion(_ context.Context, _ string, model string) (io.ReadCloser, error) {
	f.lastModel = model
	return io.NopCloser(strings.NewReader("data: [DONE]\n\n")), nil
}

func (f *fakeAIClient) CompleteJSON(_ context.Context, messages []domain.RoleMessage, model string) (string, error) {
	f.lastModel = model
	f.lastMsgs = messages
	if strings.TrimSpace(f.jsonReply) != "" {
		return f.jsonReply, nil
	}
	return `{"persona":"Portfolio Strategist","summary":"Stable trend","actions":["Rebalance cash"],"roles":["system","assistant","user"]}`, nil
}

func TestGivenSystemUserAssistantWhenGenerateStructuredJSONThenReturnsParsedStruct(t *testing.T) {
	client := &fakeAIClient{}
	service := NewAIService(client, "gpt-4.1")

	parsed, err := service.GenerateStructuredJSON(
		context.Background(),
		"You are a strict JSON-only financial assistant.",
		"Give me an investment briefing.",
		"Previous answer discussed risk budgeting.",
		"",
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if parsed.Persona != "Portfolio Strategist" {
		t.Fatalf("unexpected persona: %s", parsed.Persona)
	}
	if len(client.lastMsgs) != 3 {
		t.Fatalf("expected 3 role-tagged messages, got %d", len(client.lastMsgs))
	}
	if client.lastMsgs[0].Role != "system" || client.lastMsgs[1].Role != "assistant" || client.lastMsgs[2].Role != "user" {
		t.Fatalf("unexpected role sequence")
	}
	if client.lastModel != "gpt-4.1" {
		t.Fatalf("expected default model gpt-4.1, got %s", client.lastModel)
	}
}

func TestGivenEmptyStructuredFieldsWhenGenerateStructuredJSONThenAppliesFallbackValues(t *testing.T) {
	client := &fakeAIClient{
		jsonReply: `{"persona":"","summary":"","actions":null,"roles":null}`,
	}
	service := NewAIService(client, "gpt-4.1")

	parsed, err := service.GenerateStructuredJSON(
		context.Background(),
		"You are a strict JSON-only financial assistant.",
		"Give me an investment briefing.",
		"Previous answer discussed risk budgeting.",
		"",
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if strings.TrimSpace(parsed.Persona) == "" {
		t.Fatalf("expected non-empty persona fallback")
	}
	if strings.TrimSpace(parsed.Summary) == "" {
		t.Fatalf("expected non-empty summary fallback")
	}
	if len(parsed.Actions) == 0 {
		t.Fatalf("expected non-empty actions fallback")
	}
	if len(parsed.Roles) == 0 {
		t.Fatalf("expected non-empty roles fallback")
	}
}
