package shared

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	"apps/ai-gateway/internal/domain"
)

const (
	providerLimitCooldown = 5 * time.Hour
	logProviderFailover   = "Provider fallback active"
	logProviderFailoverOn = "Provider limit reached; enabling fallback"
)

// IsProviderLimitError identifies errors that mean the provider's plan or
// request quota is unavailable. It intentionally excludes authentication,
// validation, and general upstream failures.
func IsProviderLimitError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, " 429") ||
		strings.Contains(message, "status 429") ||
		strings.Contains(message, "usage_limit_reached") ||
		strings.Contains(message, "usage limit has been reached") ||
		strings.Contains(message, "rate limit") ||
		strings.Contains(message, "quota exceeded")
}

// FailoverProvider retries a provider-limit failure against one fallback
// provider. Once the primary is limited, subsequent requests use the
// fallback for providerLimitCooldown before probing the primary again.
type FailoverProvider struct {
	primary  Provider
	fallback Provider
	state    *failoverState
}

type failoverState struct {
	mu           sync.RWMutex
	limitedUntil time.Time
}

func NewFailoverProvider(primary, fallback Provider) Provider {
	if primary == nil || fallback == nil || primary.Name() == fallback.Name() {
		return primary
	}
	return &FailoverProvider{
		primary:  primary,
		fallback: fallback,
		state:    &failoverState{},
	}
}

func (p *FailoverProvider) Name() string                           { return p.primary.Name() }
func (p *FailoverProvider) IsConfigured() bool                     { return p.primary.IsConfigured() }
func (p *FailoverProvider) Ping(ctx context.Context) error         { return p.primary.Ping(ctx) }
func (p *FailoverProvider) Usage(ctx context.Context) (any, error) { return p.primary.Usage(ctx) }
func (p *FailoverProvider) ListModels(ctx context.Context) (*domain.ModelsResponse, error) {
	return p.primary.ListModels(ctx)
}
func (p *FailoverProvider) Embeddings(ctx context.Context, req domain.EmbeddingRequest) (*domain.EmbeddingResponse, error) {
	return withFailover(p, req.Model, func(provider Provider) (*domain.EmbeddingResponse, error) {
		return provider.Embeddings(ctx, req)
	})
}
func (p *FailoverProvider) IsReady() bool       { return p.primary.IsReady() }
func (p *FailoverProvider) SetReady(ready bool) { p.primary.SetReady(ready) }

func (p *FailoverProvider) Chat(ctx context.Context, req domain.ChatRequest) (*domain.ChatResponse, error) {
	return withFailover(p, req.Model, func(provider Provider) (*domain.ChatResponse, error) {
		return provider.Chat(ctx, req)
	})
}

func (p *FailoverProvider) Responses(ctx context.Context, req domain.ResponsesRequest) (*domain.ResponsesResponse, error) {
	return withFailover(p, req.Model, func(provider Provider) (*domain.ResponsesResponse, error) {
		return provider.Responses(ctx, req)
	})
}

func (p *FailoverProvider) ChatStream(ctx context.Context, req domain.ChatRequest, w io.Writer) (domain.Usage, error) {
	return streamWithFailover(p, req.Model, func(provider Provider, writer io.Writer) (domain.Usage, error) {
		return provider.ChatStream(ctx, req, writer)
	}, w)
}

func (p *FailoverProvider) ResponsesStream(ctx context.Context, req domain.ResponsesRequest, w io.Writer) (domain.Usage, error) {
	return streamWithFailover(p, req.Model, func(provider Provider, writer io.Writer) (domain.Usage, error) {
		return provider.ResponsesStream(ctx, req, writer)
	}, w)
}

func (p *FailoverProvider) primaryAvailable() bool {
	p.state.mu.RLock()
	defer p.state.mu.RUnlock()
	return time.Now().After(p.state.limitedUntil)
}

func (p *FailoverProvider) cooldownUntil() time.Time {
	p.state.mu.RLock()
	defer p.state.mu.RUnlock()
	return p.state.limitedUntil
}

func (p *FailoverProvider) markLimited(model string, err error) {
	p.state.mu.Lock()
	p.state.limitedUntil = time.Now().Add(providerLimitCooldown)
	until := p.state.limitedUntil
	p.state.mu.Unlock()

	slog.Warn(logProviderFailoverOn,
		"primary", p.primary.Name(),
		"fallback", p.fallback.Name(),
		"model", model,
		"fallback_model", model,
		"until", until,
		"error", err,
	)
}

func (p *FailoverProvider) logFallback(model string) {
	slog.Warn(logProviderFailover,
		"primary", p.primary.Name(),
		"fallback", p.fallback.Name(),
		"model", model,
		"fallback_model", model,
		"until", p.cooldownUntil(),
	)
}

func withFailover[T any](p *FailoverProvider, model string, call func(Provider) (T, error)) (T, error) {
	provider := p.primary
	if !p.primaryAvailable() {
		provider = p.fallback
	}

	if provider == p.fallback {
		p.logFallback(model)
	}
	result, err := call(provider)
	if err == nil || provider != p.primary || !IsProviderLimitError(err) {
		return result, err
	}

	p.markLimited(model, err)
	p.logFallback(model)
	return call(p.fallback)
}

func streamWithFailover(
	p *FailoverProvider,
	model string,
	call func(Provider, io.Writer) (domain.Usage, error),
	w io.Writer,
) (domain.Usage, error) {
	provider := p.primary
	if !p.primaryAvailable() {
		provider = p.fallback
	}

	tracked := &trackingWriter{writer: w}
	if provider == p.fallback {
		p.logFallback(model)
	}
	usage, err := call(provider, tracked)
	if err == nil || provider != p.primary || !IsProviderLimitError(err) || tracked.written > 0 {
		return usage, err
	}

	p.markLimited(model, err)
	p.logFallback(model)
	return call(p.fallback, w)
}

type trackingWriter struct {
	writer  io.Writer
	written int
}

func (w *trackingWriter) Write(data []byte) (int, error) {
	n, err := w.writer.Write(data)
	w.written += n
	return n, err
}
