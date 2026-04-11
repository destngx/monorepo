package providers

import (
	"context"
	"fmt"
	"io"

	"apps/ai-gateway/types"
	"golang.org/x/time/rate"
)

// ErrRateLimitExceeded is returned when the local provider-level rate limit is hit.
type ErrRateLimitExceeded struct {
	Provider string
}

func (e *ErrRateLimitExceeded) Error() string {
	return fmt.Sprintf("gateway rate limit exceeded for provider %q", e.Provider)
}

// RateLimitedProvider wraps any Provider and enforces a requests-per-minute (RPM) limit.
type RateLimitedProvider struct {
	Provider
	limiter *rate.Limiter
}

// NewRateLimitedProvider creates a new decorator with the specified RPM and Burst limits.
func NewRateLimitedProvider(p Provider, rpm, burst int) Provider {
	if rpm <= 0 {
		return p // No limit
	}
	// Per minute = rpm / 60 requests per second
	limit := rate.Limit(float64(rpm) / 60.0)

	// Ensure burst is at least 1
	if burst < 1 {
		burst = 1
	}

	return &RateLimitedProvider{
		Provider: p,
		limiter:  rate.NewLimiter(limit, burst),
	}
}

func (r *RateLimitedProvider) Chat(ctx context.Context, req types.ChatRequest) (*types.ChatResponse, error) {
	if !r.limiter.Allow() {
		return nil, &ErrRateLimitExceeded{Provider: r.Name()}
	}
	return r.Provider.Chat(ctx, req)
}

func (r *RateLimitedProvider) ChatStream(ctx context.Context, req types.ChatRequest, w io.Writer) (types.Usage, error) {
	if !r.limiter.Allow() {
		return types.Usage{}, &ErrRateLimitExceeded{Provider: r.Name()}
	}
	return r.Provider.ChatStream(ctx, req, w)
}

func (r *RateLimitedProvider) Embeddings(ctx context.Context, req types.EmbeddingRequest) (*types.EmbeddingResponse, error) {
	if !r.limiter.Allow() {
		return nil, &ErrRateLimitExceeded{Provider: r.Name()}
	}
	return r.Provider.Embeddings(ctx, req)
}
