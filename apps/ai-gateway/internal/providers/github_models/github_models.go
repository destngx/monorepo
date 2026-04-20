package github_models

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
	baseURL    = "https://models.github.ai/inference"
	catalogURL = "https://models.github.ai/catalog/models"

	headerAuthorization    = "Authorization"
	headerAccept           = "Accept"
	headerGitHubApiVersion = "X-GitHub-Api-Version"
	headerContentType      = "Content-Type"

	contentTypeJSON   = domain.ContentTypeJSON
	tokenPrefixBearer = "Bearer "
)

type Provider struct {
	token  string
	client *http.Client
	ready  bool
}

func New(token string) *Provider {
	return &Provider{
		token:  token,
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *Provider) Name() string { return domain.ProviderGitHubModels }

func (p *Provider) headers() map[string]string {
	return map[string]string{
		headerAuthorization:    tokenPrefixBearer + p.token,
		headerAccept:           "application/vnd.github+json",
		headerGitHubApiVersion: "2026-03-10",
		headerContentType:      contentTypeJSON,
	}
}

func (p *Provider) Chat(ctx context.Context, req domain.ChatRequest) (*domain.ChatResponse, error) {
	req = shared.CropRequest(req, 5000)

	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	for k, v := range p.headers() {
		httpReq.Header.Set(k, v)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github models error %d: %s", resp.StatusCode, b)
	}

	var result domain.ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (p *Provider) ChatStream(ctx context.Context, req domain.ChatRequest, w io.Writer) (domain.Usage, error) {
	req = shared.CropRequest(req, 8000)

	req.Stream = true
	req.StreamOptions = &domain.StreamOptions{IncludeUsage: true}
	body, _ := json.Marshal(req)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return domain.Usage{}, err
	}
	for k, v := range p.headers() {
		httpReq.Header.Set(k, v)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return domain.Usage{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return domain.Usage{}, fmt.Errorf("github models error %d: %s", resp.StatusCode, b)
	}

	return shared.StreamSSEAndCountTokens(resp.Body, w)
}

func (p *Provider) Embeddings(ctx context.Context, req domain.EmbeddingRequest) (*domain.EmbeddingResponse, error) {
	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	for k, v := range p.headers() {
		httpReq.Header.Set(k, v)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github embeddings error %d: %s", resp.StatusCode, b)
	}

	var result domain.EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (p *Provider) ListModels(ctx context.Context) (*domain.ModelsResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, catalogURL, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range p.headers() {
		httpReq.Header.Set(k, v)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github models catalog error %d: %s", resp.StatusCode, b)
	}

	var catalog []struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Publisher string `json:"publisher"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&catalog); err != nil {
		return nil, fmt.Errorf("failed to decode github catalog: %w", err)
	}

	models := make([]domain.ModelInfo, 0, len(catalog))
	for _, m := range catalog {
		models = append(models, domain.ModelInfo{
			ID:      m.ID,
			Object:  "model",
			OwnedBy: m.Publisher,
		})
	}

	return &domain.ModelsResponse{
		Object: "list",
		Data:   models,
	}, nil
}

func (p *Provider) IsConfigured() bool {
	return p.token != ""
}

func (p *Provider) Ping(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodHead, baseURL, nil)
	if err != nil {
		return err
	}
	for k, v := range p.headers() {
		httpReq.Header.Set(k, v)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("github models service unavailable (status %d)", resp.StatusCode)
	}
	return nil
}

func (p *Provider) Usage(ctx context.Context) (any, error) {
	return nil, fmt.Errorf("usage API not supported for %s", p.Name())
}

func (p *Provider) IsReady() bool   { return p.ready }
func (p *Provider) SetReady(r bool) { p.ready = r }
