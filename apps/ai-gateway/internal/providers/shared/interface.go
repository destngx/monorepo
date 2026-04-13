package shared

import (
	"context"
	"io"

	"apps/ai-gateway/internal/domain"
)

// Provider is the universal contract all AI backends must satisfy.
type Provider interface {
	// Name returns the provider identifier (e.g., "github", "openai").
	Name() string

	// Chat sends a non-streaming request and returns a parsed response.
	Chat(ctx context.Context, req domain.ChatRequest) (*domain.ChatResponse, error)

	// ChatStream sends a streaming request and writes raw SSE bytes to w.
	// It returns the accumulated usage after the stream ends.
	ChatStream(ctx context.Context, req domain.ChatRequest, w io.Writer) (domain.Usage, error)

	// ListModels returns the available models for this provider.
	ListModels(context.Context) (*domain.ModelsResponse, error)

	// Embeddings generates vector representations for the given input.
	Embeddings(context.Context, domain.EmbeddingRequest) (*domain.EmbeddingResponse, error)

	// IsConfigured returns true if the provider has all required credentials.
	IsConfigured() bool

	// Ping verifies that the provider endpoint is reachable.
	Ping(ctx context.Context) error

	// Usage returns provider-specific usage or quota information.
	Usage(ctx context.Context) (any, error)

	// IsReady returns true if the provider is both configured and reachable.
	IsReady() bool
	// SetReady manually updates the readiness status.
	SetReady(bool)
}
