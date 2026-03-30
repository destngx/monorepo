package port

import (
	"apps/wealth-management-engine/domain"
	"context"
	"io"
)

type AIService interface {
	Stream(ctx context.Context, prompt string, model string) (io.ReadCloser, error)
	GenerateStructuredJSON(
		ctx context.Context,
		systemPrompt string,
		userPrompt string,
		assistantHistory string,
		model string,
	) (domain.StructuredAIResponse, error)
}

// AIClient is provider-agnostic so adapters like Copilot/OpenAI/Anthropic/Gemini
// can be swapped without changing service or handler layers.
type AIClient interface {
	StreamCompletion(ctx context.Context, prompt string, model string) (io.ReadCloser, error)
	CompleteJSON(ctx context.Context, messages []domain.RoleMessage, model string) (string, error)
}
