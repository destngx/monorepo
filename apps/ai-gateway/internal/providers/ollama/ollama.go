package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"apps/ai-gateway/internal/domain"
	"apps/ai-gateway/internal/providers/shared"
)

type Provider struct {
	baseURL string
	client  *http.Client
	ready   bool
}

func New(baseURL string) *Provider {
	return &Provider{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 300 * time.Second},
	}
}

func (p *Provider) Name() string { return "ollama" }

// Ollama v0.x supports the OpenAI-compatible endpoint at /v1/chat/completions
func (p *Provider) Chat(ctx context.Context, req domain.ChatRequest) (*domain.ChatResponse, error) {
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		p.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama error %d: %s", resp.StatusCode, b)
	}

	var result domain.ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (p *Provider) ChatStream(ctx context.Context, req domain.ChatRequest, w io.Writer) (domain.Usage, error) {
	req.Stream = true
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		p.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return domain.Usage{}, err
	}
	defer resp.Body.Close()

	return shared.StreamSSEAndCountTokens(resp.Body, w)
}

// ListModels fetches the locally available models from Ollama.
func (p *Provider) Embeddings(ctx context.Context, req domain.EmbeddingRequest) (*domain.EmbeddingResponse, error) {
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		p.baseURL+"/v1/embeddings", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama embeddings error %d: %s", resp.StatusCode, b)
	}

	var result domain.EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (p *Provider) ListModels(ctx context.Context) (*domain.ModelsResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/v1/models", nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama models error (is Ollama running?): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama models error %d: %s", resp.StatusCode, b)
	}

	var result domain.ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode ollama models: %w", err)
	}
	return &result, nil
}

func (p *Provider) IsConfigured() bool {
	return true // Ollama is local, always "configured"
}

func (p *Provider) Ping(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/", nil)
	if err != nil {
		return err
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama service returned status %d", resp.StatusCode)
	}
	return nil
}

func (p *Provider) Usage(ctx context.Context) (any, error) {
	return nil, fmt.Errorf("usage API not supported for %s", p.Name())
}

func (p *Provider) IsReady() bool   { return p.ready }
func (p *Provider) SetReady(r bool) { p.ready = r }
