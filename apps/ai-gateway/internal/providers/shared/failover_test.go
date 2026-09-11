package shared

import (
	"context"
	"errors"
	"io"
	"strings"
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
