package service

import (
	"apps/wealth-management-engine/domain"
	"apps/wealth-management-engine/port"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type aiService struct {
	client       port.AIClient
	defaultModel string
}

func NewAIService(client port.AIClient, defaultModel string) port.AIService {
	return &aiService{client: client, defaultModel: defaultModel}
}

func (s *aiService) Stream(ctx context.Context, prompt string, model string) (io.ReadCloser, error) {
	if model == "" {
		model = s.defaultModel
	}
	return s.client.StreamCompletion(ctx, prompt, model)
}

func (s *aiService) GenerateStructuredJSON(
	ctx context.Context,
	systemPrompt string,
	userPrompt string,
	assistantHistory string,
	model string,
) (domain.StructuredAIResponse, error) {
	if model == "" {
		model = s.defaultModel
	}

	messages := []domain.RoleMessage{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role:    "assistant",
			Content: assistantHistory,
		},
		{
			Role:    "user",
			Content: userPrompt,
		},
	}

	rawJSON, err := s.client.CompleteJSON(ctx, messages, model)
	if err != nil {
		return domain.StructuredAIResponse{}, err
	}

	var parsed domain.StructuredAIResponse
	if err := json.Unmarshal([]byte(rawJSON), &parsed); err != nil {
		return domain.StructuredAIResponse{}, fmt.Errorf("failed to parse AI JSON response: %w", err)
	}

	return normalizeStructuredAIResponse(parsed), nil
}

func normalizeStructuredAIResponse(parsed domain.StructuredAIResponse) domain.StructuredAIResponse {
	if strings.TrimSpace(parsed.Persona) == "" {
		parsed.Persona = "Financial Assistant"
	}
	if strings.TrimSpace(parsed.Summary) == "" {
		parsed.Summary = "Generated investment briefing is available."
	}
	if len(parsed.Actions) == 0 {
		parsed.Actions = []string{"Review portfolio allocation and risk exposure."}
	}
	if len(parsed.Roles) == 0 {
		parsed.Roles = []string{"system", "assistant", "user"}
	}
	return parsed
}
