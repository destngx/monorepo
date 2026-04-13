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

const openaiBaseURL = "https://api.openai.com/v1"

type OpenAIProvider struct {
	apiKey string
	client *http.Client
	ready  bool
}

func NewOpenAI(apiKey string) *OpenAIProvider {
	return &OpenAIProvider{
		apiKey: apiKey,
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

func (o *OpenAIProvider) Name() string { return "openai" }

func (o *OpenAIProvider) Chat(ctx context.Context, req types.ChatRequest) (*types.ChatResponse, error) {
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		openaiBaseURL+"/chat/completions", bytes.NewReader(body))
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai error %d: %s", resp.StatusCode, b)
	}

	var result types.ChatResponse
	json.NewDecoder(resp.Body).Decode(&result)
	return &result, nil
}

func (o *OpenAIProvider) ChatStream(ctx context.Context, req types.ChatRequest, w io.Writer) (types.Usage, error) {
	req.Stream = true
	req.StreamOptions = &types.StreamOptions{IncludeUsage: true}
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		openaiBaseURL+"/chat/completions", bytes.NewReader(body))
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return types.Usage{}, err
	}
	defer resp.Body.Close()

	return streamSSEAndCountTokens(resp.Body, w)
}

// ListModels fetches the model list from OpenAI API.
func (o *OpenAIProvider) Embeddings(ctx context.Context, req types.EmbeddingRequest) (*types.EmbeddingResponse, error) {
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		openaiBaseURL+"/embeddings", bytes.NewReader(body))
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai embeddings error %d: %s", resp.StatusCode, b)
	}

	var result types.EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (o *OpenAIProvider) ListModels(ctx context.Context) (*types.ModelsResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, openaiBaseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai models error %d: %s", resp.StatusCode, b)
	}

	var result types.ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode openai models: %w", err)
	}
	return &result, nil
}

func (o *OpenAIProvider) IsConfigured() bool {
	return o.apiKey != ""
}

func (o *OpenAIProvider) Ping(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodHead, openaiBaseURL+"/models", nil)
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("openai service unavailable (status %d)", resp.StatusCode)
	}
	return nil
}

func (o *OpenAIProvider) Usage(ctx context.Context) (any, error) {
	return nil, fmt.Errorf("usage API not supported for %s", o.Name())
}

func (o *OpenAIProvider) IsReady() bool   { return o.ready }
func (o *OpenAIProvider) SetReady(r bool) { o.ready = r }
