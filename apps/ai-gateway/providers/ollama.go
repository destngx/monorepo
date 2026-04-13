package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"apps/ai-gateway/types"
)

type OllamaProvider struct {
	baseURL string
	client  *http.Client
	ready   bool
}

func NewOllama(baseURL string) *OllamaProvider {
	return &OllamaProvider{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 300 * time.Second},
	}
}

func (o *OllamaProvider) Name() string { return "ollama" }

// Ollama v0.x supports the OpenAI-compatible endpoint at /v1/chat/completions
func (o *OllamaProvider) Chat(ctx context.Context, req types.ChatRequest) (*types.ChatResponse, error) {
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		o.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama error %d: %s", resp.StatusCode, b)
	}

	var result types.ChatResponse
	json.NewDecoder(resp.Body).Decode(&result)
	return &result, nil
}

func (o *OllamaProvider) ChatStream(ctx context.Context, req types.ChatRequest, w io.Writer) (types.Usage, error) {
	req.Stream = true
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		o.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return types.Usage{}, err
	}
	defer resp.Body.Close()

	return streamSSEAndCountTokens(resp.Body, w)
}

// ListModels fetches the locally available models from Ollama.
func (o *OllamaProvider) Embeddings(ctx context.Context, req types.EmbeddingRequest) (*types.EmbeddingResponse, error) {
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		o.baseURL+"/v1/embeddings", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama embeddings error %d: %s", resp.StatusCode, b)
	}

	var result types.EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (o *OllamaProvider) ListModels(ctx context.Context) (*types.ModelsResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, o.baseURL+"/v1/models", nil)
	if err != nil {
		return nil, err
	}

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama models error (is Ollama running?): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama models error %d: %s", resp.StatusCode, b)
	}

	var result types.ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode ollama models: %w", err)
	}
	return &result, nil
}

func (o *OllamaProvider) IsConfigured() bool {
	return true // Ollama is local, always "configured"
}

func (o *OllamaProvider) Ping(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, o.baseURL+"/", nil)
	if err != nil {
		return err
	}

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama service returned status %d", resp.StatusCode)
	}
	return nil
}

func (o *OllamaProvider) Usage(ctx context.Context) (any, error) {
	return nil, fmt.Errorf("usage API not supported for %s", o.Name())
}

func (o *OllamaProvider) IsReady() bool   { return o.ready }
func (o *OllamaProvider) SetReady(r bool) { o.ready = r }
