package xiaomi_mimo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"apps/ai-gateway/internal/domain"
	"apps/ai-gateway/internal/providers/shared"
)

type Provider struct {
	apiKey  string
	baseURL string
	client  *http.Client
	ready   bool
}

func New(apiKey, baseURL string) *Provider {
	if baseURL == "" {
		baseURL = "https://token-plan-sgp.xiaomimimo.com/v1"
	}
	return &Provider{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 300 * time.Second},
	}
}

func (p *Provider) Name() string { return domain.ProviderXiaomiMimo }

type Request struct {
	domain.ChatRequest
	Thinking *ThinkingConfig `json:"thinking,omitempty"`
}

type ThinkingConfig struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens,omitempty"`
}

func (p *Provider) Chat(ctx context.Context, req domain.ChatRequest) (*domain.ChatResponse, error) {
	mimoReq := p.convertToMimoRequest(req)
	body, _ := json.Marshal(mimoReq)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimSuffix(p.baseURL, "/")+"/chat/completions", bytes.NewReader(body))

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
		return nil, fmt.Errorf("xiaomi-mimo error %d: %s", resp.StatusCode, b)
	}

	var result domain.ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (p *Provider) ChatStream(ctx context.Context, req domain.ChatRequest, w io.Writer) (domain.Usage, error) {
	mimoReq := p.convertToMimoRequest(req)
	mimoReq.Stream = true
	if mimoReq.StreamOptions == nil {
		mimoReq.StreamOptions = &domain.StreamOptions{IncludeUsage: true}
	}
	body, _ := json.Marshal(mimoReq)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimSuffix(p.baseURL, "/")+"/chat/completions", bytes.NewReader(body))

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
		return domain.Usage{}, fmt.Errorf("xiaomi-mimo error %d: %s", resp.StatusCode, b)
	}

	return shared.StreamSSEAndCountTokens(resp.Body, w)
}

func (p *Provider) convertToMimoRequest(req domain.ChatRequest) Request {
	// Strip scope prefix for upstream
	req.Model = domain.StripMimoScope(req.Model)

	mimoReq := Request{
		ChatRequest: req,
	}

	// Handle ReasoningEffort -> Thinking mapping
	if req.ReasoningEffort != "" {
		mimoReq.Thinking = &ThinkingConfig{}
		switch req.ReasoningEffort {
		case domain.ReasoningEffortNone:
			mimoReq.Thinking.Type = "disabled"
		case domain.ReasoningEffortLow:
			mimoReq.Thinking.Type = "enabled"
			mimoReq.Thinking.BudgetTokens = 1024
		case domain.ReasoningEffortMedium:
			mimoReq.Thinking.Type = "enabled"
			mimoReq.Thinking.BudgetTokens = 2048
		case domain.ReasoningEffortHigh:
			mimoReq.Thinking.Type = "enabled"
			mimoReq.Thinking.BudgetTokens = 4096
		}
	} else if strings.Contains(req.Model, "pro") {
		// Default thinking for pro models if not specified?
		// User's curl had it disabled by default.
		mimoReq.Thinking = &ThinkingConfig{Type: "disabled"}
	}

	return mimoReq
}

func (p *Provider) headers() map[string]string {
	return map[string]string{
		"Content-Type":  domain.ContentTypeJSON,
		"api-key":       p.apiKey,
		"Authorization": "Bearer " + p.apiKey,
	}
}

func (p *Provider) ListModels(ctx context.Context) (*domain.ModelsResponse, error) {
	baseModels := []string{domain.ModelMimoV25Pro, domain.ModelMimoV25}
	var knownModels []domain.ModelInfo
	for _, bm := range baseModels {
		knownModels = append(knownModels, domain.ModelInfo{ID: bm, Object: "model", OwnedBy: "xiaomi-mimo"})
	}
	return &domain.ModelsResponse{
		Object: "list",
		Data:   knownModels,
	}, nil
}

func (p *Provider) Responses(ctx context.Context, req domain.ResponsesRequest) (*domain.ResponsesResponse, error) {
	return nil, domain.UnsupportedResponsesError(p.Name())
}

func (p *Provider) ResponsesStream(ctx context.Context, req domain.ResponsesRequest, w io.Writer) (domain.Usage, error) {
	return domain.Usage{}, domain.UnsupportedResponsesError(p.Name())
}

func (p *Provider) Embeddings(ctx context.Context, req domain.EmbeddingRequest) (*domain.EmbeddingResponse, error) {
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimSuffix(p.baseURL, "/")+"/embeddings", bytes.NewReader(body))

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
		return nil, fmt.Errorf("xiaomi-mimo embeddings error %d: %s", resp.StatusCode, b)
	}

	var result domain.EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (p *Provider) IsConfigured() bool {
	return p.apiKey != ""
}

func (p *Provider) Ping(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSuffix(p.baseURL, "/")+"/models", nil)
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

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("xiaomi-mimo service returned status %d", resp.StatusCode)
	}
	return nil
}

func (p *Provider) Usage(ctx context.Context) (any, error) {
	return nil, fmt.Errorf("usage API not supported for %s", p.Name())
}

func (p *Provider) IsReady() bool   { return p.ready }
func (p *Provider) SetReady(r bool) { p.ready = r }
