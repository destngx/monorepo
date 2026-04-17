package openai

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

const (
	baseURL = "https://api.openai.com/v1"

	headerAuthorization = "Authorization"
	headerContentType   = "Content-Type"
	contentTypeJSON     = domain.ContentTypeJSON
	tokenPrefixBearer   = "Bearer "

	pathChatCompletions = "/chat/completions"
	pathModels          = "/models"
	pathEmbeddings      = "/embeddings"
)

type Provider struct {
	apiKey string
	client *http.Client
	ready  bool
}

func New(apiKey string) *Provider {
	return &Provider{
		apiKey: apiKey,
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *Provider) Name() string { return domain.ProviderOpenAI }

func (p *Provider) Chat(ctx context.Context, req domain.ChatRequest) (*domain.ChatResponse, error) {
	req = shared.NormalizeTools(req)
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		baseURL+pathChatCompletions, bytes.NewReader(body))
	httpReq.Header.Set(headerAuthorization, tokenPrefixBearer+p.apiKey)
	httpReq.Header.Set(headerContentType, contentTypeJSON)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai error %d: %s", resp.StatusCode, b)
	}

	var result domain.ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (p *Provider) ChatStream(ctx context.Context, req domain.ChatRequest, w io.Writer) (domain.Usage, error) {
	req.Stream = true
	req.StreamOptions = &domain.StreamOptions{IncludeUsage: true}
	req = shared.NormalizeTools(req)
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		baseURL+pathChatCompletions, bytes.NewReader(body))
	httpReq.Header.Set(headerAuthorization, tokenPrefixBearer+p.apiKey)
	httpReq.Header.Set(headerContentType, contentTypeJSON)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return domain.Usage{}, err
	}
	defer resp.Body.Close()

	return shared.StreamSSEAndCountTokens(resp.Body, w)
}

func (p *Provider) Embeddings(ctx context.Context, req domain.EmbeddingRequest) (*domain.EmbeddingResponse, error) {
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		baseURL+pathEmbeddings, bytes.NewReader(body))
	httpReq.Header.Set(headerAuthorization, tokenPrefixBearer+p.apiKey)
	httpReq.Header.Set(headerContentType, contentTypeJSON)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai embeddings error %d: %s", resp.StatusCode, b)
	}

	var result domain.EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (p *Provider) ListModels(ctx context.Context) (*domain.ModelsResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+pathModels, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set(headerAuthorization, tokenPrefixBearer+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai models error %d: %s", resp.StatusCode, b)
	}

	var result domain.ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode openai models: %w", err)
	}
	return &result, nil
}

func (p *Provider) IsConfigured() bool {
	return p.apiKey != ""
}

func (p *Provider) Ping(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodHead, baseURL+pathModels, nil)
	if err != nil {
		return err
	}
	httpReq.Header.Set(headerAuthorization, tokenPrefixBearer+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("openai service unavailable (status %d)", resp.StatusCode)
	}
	return nil
}

func (p *Provider) Usage(ctx context.Context) (any, error) {
	return nil, fmt.Errorf("usage API not supported for %s", p.Name())
}

func (p *Provider) IsReady() bool   { return p.ready }
func (p *Provider) SetReady(r bool) { p.ready = r }
