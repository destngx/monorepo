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
	providerLimitCooldown        = 5 * time.Hour
	logProviderLimitReached      = "Provider limit reached; enabling fallback"
	logProviderFallbackActive    = "Provider fallback active"
	logProviderFallbackSucceeded = "Provider fallback succeeded"
	logProviderFallbackFailed    = "Provider fallback failed"
	logProviderFallbackSkipped   = "Provider limit reached after stream started; fallback skipped"
)

const (
	operationChat            = "chat"
	operationChatStream      = "chat_stream"
	operationResponses       = "responses"
	operationResponsesStream = "responses_stream"
	operationEmbeddings      = "embeddings"
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
		strings.Contains(message, "quota exceeded") ||
		strings.Contains(message, "server_is_overloaded")
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
	limitedAt    time.Time
	limitedUntil time.Time
	limitError   string
	limitModel   string
}

// failoverCooldown is a snapshot of the limit that put the primary into
// cooldown, carried into every log line so each fallback explains its cause.
type failoverCooldown struct {
	limitedAt    time.Time
	until        time.Time
	triggerError string
	triggerModel string
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
	return withFailover(p.attempt(ctx, operationEmbeddings, req.Model), func(provider Provider) (*domain.EmbeddingResponse, error) {
		return provider.Embeddings(ctx, req)
	})
}
func (p *FailoverProvider) IsReady() bool       { return p.primary.IsReady() }
func (p *FailoverProvider) SetReady(ready bool) { p.primary.SetReady(ready) }

func (p *FailoverProvider) Chat(ctx context.Context, req domain.ChatRequest) (*domain.ChatResponse, error) {
	return withFailover(p.attempt(ctx, operationChat, req.Model), func(provider Provider) (*domain.ChatResponse, error) {
		return provider.Chat(ctx, req)
	})
}

func (p *FailoverProvider) Responses(ctx context.Context, req domain.ResponsesRequest) (*domain.ResponsesResponse, error) {
	return withFailover(p.attempt(ctx, operationResponses, req.Model), func(provider Provider) (*domain.ResponsesResponse, error) {
		return provider.Responses(ctx, req)
	})
}

func (p *FailoverProvider) ChatStream(ctx context.Context, req domain.ChatRequest, w io.Writer) (domain.Usage, error) {
	return streamWithFailover(p.attempt(ctx, operationChatStream, req.Model), func(provider Provider, writer io.Writer) (domain.Usage, error) {
		return provider.ChatStream(ctx, req, writer)
	}, w)
}

func (p *FailoverProvider) ResponsesStream(ctx context.Context, req domain.ResponsesRequest, w io.Writer) (domain.Usage, error) {
	return streamWithFailover(p.attempt(ctx, operationResponsesStream, req.Model), func(provider Provider, writer io.Writer) (domain.Usage, error) {
		return provider.ResponsesStream(ctx, req, writer)
	}, w)
}

// activeCooldown returns the current cooldown and whether the primary is
// still inside it.
func (p *FailoverProvider) activeCooldown() (failoverCooldown, bool) {
	p.state.mu.RLock()
	defer p.state.mu.RUnlock()
	cooldown := failoverCooldown{
		limitedAt:    p.state.limitedAt,
		until:        p.state.limitedUntil,
		triggerError: p.state.limitError,
		triggerModel: p.state.limitModel,
	}
	return cooldown, time.Now().Before(cooldown.until)
}

func (p *FailoverProvider) markLimited(model string, err error) failoverCooldown {
	now := time.Now()
	cooldown := failoverCooldown{
		limitedAt:    now,
		until:        now.Add(providerLimitCooldown),
		triggerError: err.Error(),
		triggerModel: model,
	}
	p.state.mu.Lock()
	defer p.state.mu.Unlock()
	p.state.limitedAt = cooldown.limitedAt
	p.state.limitedUntil = cooldown.until
	p.state.limitError = cooldown.triggerError
	p.state.limitModel = cooldown.triggerModel
	return cooldown
}

// failoverAttempt carries one request's identity through the failover path
// so every log line and the request log metadata describe the same request.
type failoverAttempt struct {
	ctx       context.Context
	provider  *FailoverProvider
	operation string
	model     string
}

func (p *FailoverProvider) attempt(ctx context.Context, operation, model string) failoverAttempt {
	return failoverAttempt{ctx: ctx, provider: p, operation: operation, model: model}
}

func (a failoverAttempt) attrs(extra ...any) []any {
	attrs := []any{
		"request_id", requestIDFromContext(a.ctx),
		"operation", a.operation,
		"primary", a.provider.primary.Name(),
		"fallback", a.provider.fallback.Name(),
		"model", a.model,
	}
	return append(attrs, extra...)
}

// recordFailover attaches the failover outcome to the request log metadata so
// the access log reports the provider that actually served the request.
func (a failoverAttempt) recordFailover(reason string, cooldown failoverCooldown) *domain.FailoverLogMeta {
	failover := &domain.FailoverLogMeta{
		Primary:      a.provider.primary.Name(),
		Fallback:     a.provider.fallback.Name(),
		Reason:       reason,
		TriggerModel: cooldown.triggerModel,
		TriggerError: cooldown.triggerError,
	}
	if meta, ok := a.ctx.Value(domain.LogMetaKey).(*domain.RequestLogMeta); ok && meta != nil {
		meta.Failover = failover
	}
	return failover
}

func (a failoverAttempt) logLimitReached(cooldown failoverCooldown, primaryDuration time.Duration) {
	slog.Warn(logProviderLimitReached, a.attrs(
		"reason", domain.FailoverReasonPrimaryLimit,
		"fallback_model", a.model,
		"trigger_error", cooldown.triggerError,
		"primary_duration", primaryDuration,
		"cooldown", providerLimitCooldown,
		"until", cooldown.until,
	)...)
}

func (a failoverAttempt) logCooldownRouting(cooldown failoverCooldown) {
	slog.Warn(logProviderFallbackActive, a.attrs(
		"reason", domain.FailoverReasonPrimaryCooldown,
		"fallback_model", a.model,
		"trigger_model", cooldown.triggerModel,
		"trigger_error", cooldown.triggerError,
		"limited_at", cooldown.limitedAt,
		"until", cooldown.until,
		"cooldown_remaining", time.Until(cooldown.until).Round(time.Second),
	)...)
}

func (a failoverAttempt) logStreamSkipped(cooldown failoverCooldown, primaryDuration time.Duration, written int) {
	slog.Warn(logProviderFallbackSkipped, a.attrs(
		"reason", domain.FailoverReasonStreamStarted,
		"trigger_error", cooldown.triggerError,
		"bytes_written", written,
		"primary_duration", primaryDuration,
		"cooldown", providerLimitCooldown,
		"until", cooldown.until,
	)...)
}

// runFallback serves the request from the fallback provider, then records and
// logs how the fallback went.
func runFallback[T any](a failoverAttempt, reason string, cooldown failoverCooldown, call func() (T, error)) (T, error) {
	failover := a.recordFailover(reason, cooldown)
	start := time.Now()
	result, err := call()
	attrs := a.attrs(
		"reason", reason,
		"fallback_model", a.model,
		"trigger_error", cooldown.triggerError,
		"fallback_duration", time.Since(start),
	)
	if err != nil {
		failover.FallbackError = err.Error()
		slog.Error(logProviderFallbackFailed, append(attrs, "fallback_error", err.Error())...)
		return result, err
	}
	slog.Info(logProviderFallbackSucceeded, attrs...)
	return result, nil
}

func withFailover[T any](a failoverAttempt, call func(Provider) (T, error)) (T, error) {
	p := a.provider
	fallback := func() (T, error) { return call(p.fallback) }
	if cooldown, active := p.activeCooldown(); active {
		a.logCooldownRouting(cooldown)
		return runFallback(a, domain.FailoverReasonPrimaryCooldown, cooldown, fallback)
	}

	start := time.Now()
	result, err := call(p.primary)
	if !IsProviderLimitError(err) {
		return result, err
	}

	cooldown := p.markLimited(a.model, err)
	a.logLimitReached(cooldown, time.Since(start))
	return runFallback(a, domain.FailoverReasonPrimaryLimit, cooldown, fallback)
}

func streamWithFailover(a failoverAttempt, call func(Provider, io.Writer) (domain.Usage, error), w io.Writer) (domain.Usage, error) {
	p := a.provider
	fallback := func() (domain.Usage, error) { return call(p.fallback, w) }
	if cooldown, active := p.activeCooldown(); active {
		a.logCooldownRouting(cooldown)
		return runFallback(a, domain.FailoverReasonPrimaryCooldown, cooldown, fallback)
	}

	tracked := &trackingWriter{writer: w}
	start := time.Now()
	usage, err := call(p.primary, tracked)
	if !IsProviderLimitError(err) {
		return usage, err
	}

	cooldown := p.markLimited(a.model, err)
	if tracked.written > 0 {
		// Output already reached the client, so this request cannot be
		// replayed; the cooldown still routes the next requests to fallback.
		a.recordFailover(domain.FailoverReasonStreamStarted, cooldown)
		a.logStreamSkipped(cooldown, time.Since(start), tracked.written)
		return usage, err
	}
	a.logLimitReached(cooldown, time.Since(start))
	return runFallback(a, domain.FailoverReasonPrimaryLimit, cooldown, fallback)
}

func requestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(domain.RequestIDKey).(string)
	return id
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
