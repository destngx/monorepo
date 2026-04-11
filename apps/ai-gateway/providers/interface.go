package providers

import (
	"context"
	"io"

	"apps/ai-gateway/types"
)

// Provider is the universal contract all AI backends must satisfy.
type Provider interface {
	// Name returns the provider identifier (e.g., "github", "openai").
	Name() string

	// Chat sends a non-streaming request and returns a parsed response.
	Chat(ctx context.Context, req types.ChatRequest) (*types.ChatResponse, error)

	// ChatStream sends a streaming request and writes raw SSE bytes to w.
	// It returns the accumulated usage after the stream ends.
	ChatStream(ctx context.Context, req types.ChatRequest, w io.Writer) (types.Usage, error)

	// ListModels returns the available models from the provider.
	ListModels(ctx context.Context) (*types.ModelsResponse, error)
}
