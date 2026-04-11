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

	// ListModels returns the available models for this provider.
	ListModels(context.Context) (*types.ModelsResponse, error)

	// Embeddings generates vector representations for the given input.
	Embeddings(context.Context, types.EmbeddingRequest) (*types.EmbeddingResponse, error)

	// IsConfigured returns true if the provider has all required credentials.
	IsConfigured() bool

	// Ping verifies that the provider endpoint is reachable.
	Ping(ctx context.Context) error

	// IsReady returns true if the provider is both configured and reachable.
	IsReady() bool

	// SetReady manually updates the readiness status.
	SetReady(bool)
}
