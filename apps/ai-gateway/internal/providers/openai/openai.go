package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"apps/ai-gateway/config"
	"apps/ai-gateway/internal/domain"
	"apps/ai-gateway/internal/providers/shared"
)

const (
	baseURL    = "https://api.openai.com/v1"
	chatGPTURL = "https://chatgpt.com/backend-api"

	headerAccept        = "Accept"
	headerAuthorization = "Authorization"
	headerContentType   = "Content-Type"
	headerChatGPTAcctID = "chatgpt-account-id"
	headerOpenAIBeta    = "openai-beta"
	headerOriginator    = "originator"
	headerSessionID     = "session_id"
	headerUserAgent     = "User-Agent"
	headerVersion       = "version"
	contentTypeJSON     = domain.ContentTypeJSON
	tokenPrefixBearer   = "Bearer "
	authURL             = "https://auth.openai.com/oauth/token"

	pathChatCompletions  = "/chat/completions"
	pathResponses        = "/responses"
	pathCodexResponses   = "/codex/responses"
	pathCodexUsage       = "/wham/usage"
	pathModels           = "/models"
	pathEmbeddings       = "/embeddings"
	pathUsageCompletions = "/organization/usage/completions"

	envOpenAIOAuthClientID = "OPENAI_OAUTH_CLIENT_ID"
	envOpenAICodexVersion  = "OPENAI_CODEX_VERSION"

	authProbeModel             = "__ai_gateway_auth_probe__"
	codexDefaultVersion        = "0.122.0"
	codexOriginator            = "codex_cli_rs"
	codexResponsesExperimental = "responses=experimental"
	codexUserAgent             = "codex-cli"
	openAIInsufficientQuotaMsg = "openai upstream reports insufficient quota for api.openai.com; Codex/ChatGPT plan quota is separate from OpenAI API billing quota"
)

type Provider struct {
	apiKey string
	oauth  *config.OpenAIOAuth
	source string
	mu     sync.RWMutex
	client *http.Client
	ready  bool
}

func New(apiKey string, oauth *config.OpenAIOAuth) *Provider {
	return &Provider{
		apiKey: apiKey,
		oauth:  oauth,
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *Provider) Name() string { return domain.ProviderOpenAI }

func (p *Provider) Chat(ctx context.Context, req domain.ChatRequest) (*domain.ChatResponse, error) {
	if p.useCodex() {
		return p.chatCodex(ctx, req)
	}

	body, _ := json.Marshal(req)
	resp, err := p.doOpenAIRequest(ctx, http.MethodPost, pathChatCompletions, body, contentTypeJSON)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		if isInsufficientQuota(b) {
			return nil, fmt.Errorf("%s: openai error %d: %s", openAIInsufficientQuotaMsg, resp.StatusCode, b)
		}
		return nil, fmt.Errorf("openai error %d: %s", resp.StatusCode, b)
	}

	var result domain.ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (p *Provider) ChatStream(ctx context.Context, req domain.ChatRequest, w io.Writer) (domain.Usage, error) {
	if p.useCodex() {
		return p.chatCodexStream(ctx, req, w)
	}

	req.Stream = true
	req.StreamOptions = &domain.StreamOptions{IncludeUsage: true}
	body, _ := json.Marshal(req)
	resp, err := p.doOpenAIRequest(ctx, http.MethodPost, pathChatCompletions, body, contentTypeJSON)
	if err != nil {
		return domain.Usage{}, err
	}
	defer resp.Body.Close()

	return shared.StreamSSEAndCountTokens(resp.Body, w)
}

func (p *Provider) Responses(ctx context.Context, req domain.ResponsesRequest) (*domain.ResponsesResponse, error) {
	req = req.WithStream(false)
	resp, err := p.doResponsesRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai responses error %d: %s", resp.StatusCode, b)
	}

	var result domain.ResponsesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode openai responses: %w", err)
	}
	return &result, nil
}

func (p *Provider) ResponsesStream(ctx context.Context, req domain.ResponsesRequest, w io.Writer) (domain.Usage, error) {
	req = req.WithStream(true)
	resp, err := p.doResponsesRequest(ctx, req)
	if err != nil {
		return domain.Usage{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return domain.Usage{}, fmt.Errorf("openai responses stream error %d: %s", resp.StatusCode, b)
	}

	return shared.StreamResponsesSSEAndCountUsage(resp.Body, w)
}

func (p *Provider) Embeddings(ctx context.Context, req domain.EmbeddingRequest) (*domain.EmbeddingResponse, error) {
	body, _ := json.Marshal(req)
	resp, err := p.doOpenAIRequest(ctx, http.MethodPost, pathEmbeddings, body, contentTypeJSON)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		if isInsufficientQuota(b) {
			return nil, fmt.Errorf("%s: openai embeddings error %d: %s", openAIInsufficientQuotaMsg, resp.StatusCode, b)
		}
		return nil, fmt.Errorf("openai embeddings error %d: %s", resp.StatusCode, b)
	}

	var result domain.EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (p *Provider) ListModels(ctx context.Context) (*domain.ModelsResponse, error) {
	if p.useCodex() {
		models, err := p.listCodexModels(ctx)
		if err == nil {
			return models, nil
		}
		return staticModels(), nil
	}

	resp, err := p.doOpenAIRequest(ctx, http.MethodGet, pathModels, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		if p.apiKey == "" && resp.StatusCode == http.StatusForbidden {
			return staticModels(), nil
		}
		return nil, fmt.Errorf("openai models error %d: %s", resp.StatusCode, b)
	}

	var result domain.ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode openai models: %w", err)
	}
	return &result, nil
}

func (p *Provider) listCodexModels(ctx context.Context) (*domain.ModelsResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, chatGPTURL+pathModels, nil)
	if err != nil {
		return nil, err
	}
	if err := p.setAuthHeaders(httpReq); err != nil {
		return nil, err
	}
	httpReq.Header.Set(headerAccept, contentTypeJSON)
	httpReq.Header.Set(headerOriginator, codexOriginator)
	httpReq.Header.Set(headerSessionID, newCodexSessionID())
	httpReq.Header.Set(headerUserAgent, "")
	httpReq.Header.Set(headerVersion, getEnv(envOpenAICodexVersion, codexDefaultVersion))

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai codex models error %d: %s", resp.StatusCode, b)
	}

	var result chatGPTModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode openai codex models: %w", err)
	}

	models := make([]domain.ModelInfo, 0, len(result.Models))
	seen := make(map[string]struct{}, len(result.Models))
	for _, model := range result.Models {
		if model.Slug == "" {
			continue
		}
		if _, ok := seen[model.Slug]; ok {
			continue
		}
		seen[model.Slug] = struct{}{}
		models = append(models, domain.ModelInfo{
			ID:      model.Slug,
			Object:  "model",
			OwnedBy: "openai",
		})
	}

	return &domain.ModelsResponse{Object: "list", Data: models}, nil
}

func staticModels() *domain.ModelsResponse {
	return &domain.ModelsResponse{
		Object: "list",
		Data: []domain.ModelInfo{
			{ID: "gpt-5.4", Object: "model", OwnedBy: "openai"},
			{ID: "gpt-5.4-mini", Object: "model", OwnedBy: "openai"},
			{ID: domain.ModelGPT5Mini, Object: "model", OwnedBy: "openai"},
			{ID: "gpt-5.1", Object: "model", OwnedBy: "openai"},
			{ID: domain.ModelGPT41, Object: "model", OwnedBy: "openai"},
			{ID: "gpt-4o", Object: "model", OwnedBy: "openai"},
			{ID: "gpt-4o-mini", Object: "model", OwnedBy: "openai"},
			{ID: domain.ModelEmbeddingDefault, Object: "model", OwnedBy: "openai"},
		},
	}
}

func (p *Provider) useCodex() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.oauth != nil && (p.oauth.AccessToken != "" || p.oauth.RefreshToken != "" || p.oauth.IDToken != "")
}

func (p *Provider) IsConfigured() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.apiKey != "" || (p.oauth != nil && (p.oauth.AccessToken != "" || p.oauth.RefreshToken != "" || p.oauth.IDToken != ""))
}

func (p *Provider) RefreshReady(ctx context.Context) bool {
	if p.useCodex() {
		if p.refreshOAuth() == nil {
			return false
		}
		if ctx == nil {
			ctx = context.Background()
		}
		return p.Ping(ctx) == nil
	}

	if p.apiKey != "" {
		p.mu.Lock()
		p.source = "OPENAI_API_KEY"
		p.mu.Unlock()
		return true
	}

	return false
}

func (p *Provider) ReadySummary() string {
	if p.useCodex() {
		p.mu.RLock()
		source := p.source
		p.mu.RUnlock()
		if source != "" {
			return "oauth_access_token (" + source + ")"
		}
		return p.authMode()
	}

	p.mu.RLock()
	apiKey := p.apiKey
	p.mu.RUnlock()
	if apiKey != "" {
		return "api_key (OPENAI_API_KEY)"
	}
	return p.authMode()
}

func (p *Provider) Ping(ctx context.Context) error {
	if !p.IsConfigured() {
		return fmt.Errorf("openai not configured")
	}

	if p.useCodex() {
		// For OAuth, we need to try an actual request
		body, _ := json.Marshal(domain.ChatRequest{
			Model:    authProbeModel,
			Messages: []domain.Message{{Role: domain.RoleUser, Content: "ping"}},
		})
		resp, err := p.doOpenAIRequest(ctx, http.MethodPost, pathChatCompletions, body, contentTypeJSON)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return fmt.Errorf("openai auth failed: %d", resp.StatusCode)
		}

		return nil
	}

	return nil
}

func (p *Provider) Usage(ctx context.Context) (any, error) {
	if p.useCodex() {
		return p.usageCodex(ctx)
	}
	return p.usageAPI(ctx)
}

func (p *Provider) IsReady() bool   { return p.ready }
func (p *Provider) SetReady(r bool) { p.ready = r }

func isInsufficientQuota(body []byte) bool {
	return strings.Contains(string(body), "insufficient_quota")
}
