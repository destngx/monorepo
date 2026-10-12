package shared

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"apps/ai-gateway/internal/domain"
)

type failoverTestProvider struct {
	name          string
	responses     int
	responseErr   error
	streamErr     error
	streamPayload string
}

func (p *failoverTestProvider) Name() string { return p.name }
func (p *failoverTestProvider) Chat(context.Context, domain.ChatRequest) (*domain.ChatResponse, error) {
	return nil, p.responseErr
}
func (p *failoverTestProvider) ChatStream(_ context.Context, _ domain.ChatRequest, w io.Writer) (domain.Usage, error) {
	if p.streamPayload != "" {
		_, _ = io.WriteString(w, p.streamPayload)
	}
	return domain.Usage{}, p.streamErr
}
func (p *failoverTestProvider) Responses(context.Context, domain.ResponsesRequest) (*domain.ResponsesResponse, error) {
	p.responses++
	if p.responseErr != nil {
		return nil, p.responseErr
	}
	result := domain.ResponsesResponse{"provider": p.name}
	return &result, nil
}
func (p *failoverTestProvider) ResponsesStream(_ context.Context, _ domain.ResponsesRequest, w io.Writer) (domain.Usage, error) {
	if p.streamPayload != "" {
		_, _ = io.WriteString(w, p.streamPayload)
	}
	return domain.Usage{}, p.streamErr
}
func (p *failoverTestProvider) ListModels(context.Context) (*domain.ModelsResponse, error) {
	return nil, nil
}
func (p *failoverTestProvider) Embeddings(context.Context, domain.EmbeddingRequest) (*domain.EmbeddingResponse, error) {
	return nil, p.responseErr
}
func (p *failoverTestProvider) IsConfigured() bool                 { return true }
func (p *failoverTestProvider) Ping(context.Context) error         { return nil }
func (p *failoverTestProvider) Usage(context.Context) (any, error) { return nil, nil }
func (p *failoverTestProvider) IsReady() bool                      { return true }
func (p *failoverTestProvider) SetReady(bool)                      {}

func TestFailoverProviderUsesFallbackDuringCooldown(t *testing.T) {
	primary := &failoverTestProvider{name: "openai", responseErr: errors.New("openai codex error 429: usage_limit_reached")}
	fallback := &failoverTestProvider{name: "github-copilot"}
	provider := NewFailoverProvider(primary, fallback)

	response, err := provider.Responses(context.Background(), domain.ResponsesRequest{})
	if err != nil {
		t.Fatalf("expected fallback success, got %v", err)
	}
	if (*response)["provider"] != "github-copilot" {
		t.Fatalf("expected fallback response, got %#v", response)
	}

	primary.responseErr = errors.New("primary should not be called during cooldown")
	response, err = provider.Responses(context.Background(), domain.ResponsesRequest{})
	if err != nil || (*response)["provider"] != "github-copilot" {
		t.Fatalf("expected cooldown fallback, response=%#v err=%v", response, err)
	}
	if primary.responses != 1 {
		t.Fatalf("expected one primary attempt, got %d", primary.responses)
	}
}

func TestFailoverProviderRetriesPrimaryAfterCooldown(t *testing.T) {
	primary := &failoverTestProvider{name: "openai", responseErr: errors.New("429 usage limit")}
	fallback := &failoverTestProvider{name: "github-copilot"}
	provider := NewFailoverProvider(primary, fallback).(*FailoverProvider)

	_, _ = provider.Responses(context.Background(), domain.ResponsesRequest{})
	provider.state.mu.Lock()
	provider.state.limitedUntil = time.Now().Add(-time.Second)
	provider.state.mu.Unlock()

	primary.responseErr = nil
	response, err := provider.Responses(context.Background(), domain.ResponsesRequest{})
	if err != nil || (*response)["provider"] != "openai" {
		t.Fatalf("expected primary after cooldown, response=%#v err=%v", response, err)
	}
}

func TestFailoverProviderDoesNotFailoverNonLimitErrors(t *testing.T) {
	primary := &failoverTestProvider{name: "openai", responseErr: errors.New("invalid request")}
	fallback := &failoverTestProvider{name: "github-copilot"}
	provider := NewFailoverProvider(primary, fallback)

	_, err := provider.Responses(context.Background(), domain.ResponsesRequest{})
	if err == nil || fallback.responses != 0 {
		t.Fatalf("expected original error without fallback, err=%v fallback calls=%d", err, fallback.responses)
	}
}

func TestFailoverProviderDoesNotSwitchAfterStreamOutput(t *testing.T) {
	primary := &failoverTestProvider{
		name:          "openai",
		streamErr:     errors.New("429 usage_limit_reached"),
		streamPayload: "partial",
	}
	fallback := &failoverTestProvider{name: "github-copilot", streamPayload: "fallback"}
	provider := NewFailoverProvider(primary, fallback)
	var output strings.Builder

	_, err := provider.ResponsesStream(context.Background(), domain.ResponsesRequest{}, &output)
	if err == nil || output.String() != "partial" {
		t.Fatalf("expected partial primary stream and error, output=%q err=%v", output.String(), err)
	}
}

type capturedLog struct {
	level   slog.Level
	message string
	attrs   map[string]any
}

type captureHandler struct {
	mu   *sync.Mutex
	logs *[]capturedLog
}

func (h captureHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h captureHandler) WithAttrs([]slog.Attr) slog.Handler       { return h }
func (h captureHandler) WithGroup(string) slog.Handler            { return h }
func (h captureHandler) Handle(_ context.Context, record slog.Record) error {
	entry := capturedLog{level: record.Level, message: record.Message, attrs: map[string]any{}}
	record.Attrs(func(attr slog.Attr) bool {
		entry.attrs[attr.Key] = attr.Value.Any()
		return true
	})
	h.mu.Lock()
	defer h.mu.Unlock()
	*h.logs = append(*h.logs, entry)
	return nil
}

func captureLogs(t *testing.T) func() []capturedLog {
	t.Helper()
	var mu sync.Mutex
	var logs []capturedLog
	previous := slog.Default()
	slog.SetDefault(slog.New(captureHandler{mu: &mu, logs: &logs}))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return func() []capturedLog {
		mu.Lock()
		defer mu.Unlock()
		return append([]capturedLog(nil), logs...)
	}
}

func findLog(t *testing.T, logs []capturedLog, message string) capturedLog {
	t.Helper()
	for _, entry := range logs {
		if entry.message == message {
			return entry
		}
	}
	t.Fatalf("log %q not found in %#v", message, logs)
	return capturedLog{}
}

func failoverTestContext() (context.Context, *domain.RequestLogMeta) {
	meta := &domain.RequestLogMeta{Provider: "openai"}
	ctx := context.WithValue(context.Background(), domain.RequestIDKey, "rid-1")
	return context.WithValue(ctx, domain.LogMetaKey, meta), meta
}

func TestFailoverLogsWhyAndHowOnPrimaryLimit(t *testing.T) {
	logs := captureLogs(t)
	primary := &failoverTestProvider{name: "openai", responseErr: errors.New("openai codex error 429: usage_limit_reached")}
	fallback := &failoverTestProvider{name: "github-copilot"}
	provider := NewFailoverProvider(primary, fallback)
	ctx, meta := failoverTestContext()

	_, err := provider.Responses(ctx, domain.ResponsesRequest{Model: "gpt-5"})
	if err != nil {
		t.Fatalf("expected fallback success, got %v", err)
	}

	triggered := findLog(t, logs(), logProviderLimitReached)
	if triggered.level != slog.LevelWarn {
		t.Fatalf("expected warn level, got %v", triggered.level)
	}
	for key, want := range map[string]any{
		"request_id":     "rid-1",
		"operation":      "responses",
		"primary":        "openai",
		"fallback":       "github-copilot",
		"model":          "gpt-5",
		"fallback_model": "gpt-5",
		"trigger_error":  "openai codex error 429: usage_limit_reached",
		"cooldown":       providerLimitCooldown,
	} {
		if triggered.attrs[key] != want {
			t.Fatalf("limit log %s = %#v, want %#v", key, triggered.attrs[key], want)
		}
	}
	for _, key := range []string{"until", "primary_duration"} {
		if _, ok := triggered.attrs[key]; !ok {
			t.Fatalf("limit log missing %s: %#v", key, triggered.attrs)
		}
	}

	succeeded := findLog(t, logs(), logProviderFallbackSucceeded)
	if succeeded.attrs["reason"] != domain.FailoverReasonPrimaryLimit || succeeded.attrs["fallback"] != "github-copilot" {
		t.Fatalf("unexpected fallback result log: %#v", succeeded.attrs)
	}
	if _, ok := succeeded.attrs["fallback_duration"]; !ok {
		t.Fatalf("fallback result log missing fallback_duration: %#v", succeeded.attrs)
	}

	want := domain.FailoverLogMeta{
		Primary:      "openai",
		Fallback:     "github-copilot",
		Reason:       domain.FailoverReasonPrimaryLimit,
		TriggerModel: "gpt-5",
		TriggerError: "openai codex error 429: usage_limit_reached",
	}
	if meta.Failover == nil || *meta.Failover != want {
		t.Fatalf("request failover meta = %#v, want %#v", meta.Failover, want)
	}
	if meta.ServingProvider() != "github-copilot" {
		t.Fatalf("expected serving provider github-copilot, got %q", meta.ServingProvider())
	}
}

func TestFailoverLogsCooldownRouting(t *testing.T) {
	logs := captureLogs(t)
	primary := &failoverTestProvider{name: "openai", responseErr: errors.New("429 usage_limit_reached")}
	fallback := &failoverTestProvider{name: "github-copilot"}
	provider := NewFailoverProvider(primary, fallback)
	_, _ = provider.Responses(context.Background(), domain.ResponsesRequest{Model: "gpt-5"})

	ctx, meta := failoverTestContext()
	fallback.responseErr = nil
	_, _ = provider.Chat(ctx, domain.ChatRequest{Model: "gpt-4.1"})

	var routed capturedLog
	for _, entry := range logs() {
		if entry.message == logProviderFallbackActive && entry.attrs["operation"] == "chat" {
			routed = entry
		}
	}
	if routed.message == "" {
		t.Fatalf("cooldown routing log not found: %#v", logs())
	}
	for key, want := range map[string]any{
		"reason":        domain.FailoverReasonPrimaryCooldown,
		"model":         "gpt-4.1",
		"trigger_model": "gpt-5",
		"trigger_error": "429 usage_limit_reached",
	} {
		if routed.attrs[key] != want {
			t.Fatalf("cooldown log %s = %#v, want %#v", key, routed.attrs[key], want)
		}
	}
	if _, ok := routed.attrs["cooldown_remaining"]; !ok {
		t.Fatalf("cooldown log missing cooldown_remaining: %#v", routed.attrs)
	}
	if meta.Failover == nil || meta.Failover.Reason != domain.FailoverReasonPrimaryCooldown || meta.Failover.TriggerError != "429 usage_limit_reached" {
		t.Fatalf("unexpected request failover meta: %#v", meta.Failover)
	}
}

func TestFailoverRecordsFallbackError(t *testing.T) {
	logs := captureLogs(t)
	primary := &failoverTestProvider{name: "openai", responseErr: errors.New("status 429 rate limit")}
	fallback := &failoverTestProvider{name: "github-copilot", responseErr: errors.New("copilot unavailable")}
	provider := NewFailoverProvider(primary, fallback)
	ctx, meta := failoverTestContext()

	if _, err := provider.Responses(ctx, domain.ResponsesRequest{}); err == nil {
		t.Fatal("expected fallback error")
	}

	failed := findLog(t, logs(), logProviderFallbackFailed)
	if failed.level != slog.LevelError || failed.attrs["fallback_error"] != "copilot unavailable" || failed.attrs["trigger_error"] != "status 429 rate limit" {
		t.Fatalf("unexpected fallback failure log: %#v", failed)
	}
	if meta.Failover == nil || meta.Failover.FallbackError != "copilot unavailable" {
		t.Fatalf("expected fallback error in request meta, got %#v", meta.Failover)
	}
}

func TestFailoverLogsLimitAfterStreamStarted(t *testing.T) {
	logs := captureLogs(t)
	primary := &failoverTestProvider{name: "openai", streamErr: errors.New("429 usage_limit_reached"), streamPayload: "partial"}
	fallback := &failoverTestProvider{name: "github-copilot", streamPayload: "fallback"}
	provider := NewFailoverProvider(primary, fallback)
	ctx, meta := failoverTestContext()
	var output strings.Builder

	_, err := provider.ResponsesStream(ctx, domain.ResponsesRequest{}, &output)
	if err == nil || output.String() != "partial" {
		t.Fatalf("expected partial primary stream and error, output=%q err=%v", output.String(), err)
	}

	skipped := findLog(t, logs(), logProviderFallbackSkipped)
	if skipped.attrs["operation"] != "responses_stream" || skipped.attrs["bytes_written"] != int64(len("partial")) {
		t.Fatalf("unexpected stream skip log: %#v", skipped.attrs)
	}
	if meta.Failover == nil || meta.Failover.Reason != domain.FailoverReasonStreamStarted {
		t.Fatalf("expected stream_started meta, got %#v", meta.Failover)
	}
	if meta.ServingProvider() != "openai" {
		t.Fatalf("expected primary to remain the serving provider, got %q", meta.ServingProvider())
	}

	output.Reset()
	if _, err := provider.ResponsesStream(context.Background(), domain.ResponsesRequest{}, &output); err != nil || output.String() != "fallback" {
		t.Fatalf("expected next request to use fallback during cooldown, output=%q err=%v", output.String(), err)
	}
}
