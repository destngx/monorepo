package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
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
	pathCodexResponses   = "/codex/responses"
	pathModels           = "/models"
	pathEmbeddings       = "/embeddings"
	pathUsageCompletions = "/organization/usage/completions"

	envOpenAIOAuthClientID = "OPENAI_OAUTH_CLIENT_ID"
	envOpenAICodexVersion  = "OPENAI_CODEX_VERSION"

	authProbeModel             = "__ai_gateway_auth_probe__"
	codexDefaultVersion        = "0.122.0"
	codexOriginator            = "codex_cli_rs"
	codexResponsesExperimental = "responses=experimental"
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

func (p *Provider) authMode() string {
	if p.apiKey != "" {
		return "api_key"
	}
	return "oauth"
}

func (p *Provider) refreshOAuth() *config.OpenAIOAuth {
	if oauth, source := config.ReloadOpenAIOAuthSource(); oauth != nil {
		p.mu.Lock()
		p.oauth = oauth
		p.source = source
		p.mu.Unlock()
		return oauth
	}

	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.oauth
}

type oauthRefreshResponse struct {
	AccountID        string `json:"account_id"`
	ChatGPTAccountID string `json:"chatgpt_account_id"`
	IDToken          string `json:"id_token"`
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
}

type codexResponseRequest struct {
	Model           string              `json:"model"`
	Instructions    string              `json:"instructions"`
	Input           []codexInputMessage `json:"input"`
	Stream          bool                `json:"stream"`
	Store           bool                `json:"store"`
	MaxOutputTokens *int                `json:"max_output_tokens,omitempty"`
}

type codexInputMessage struct {
	Role    string              `json:"role"`
	Content []codexInputContent `json:"content"`
}

type codexInputContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type codexStreamEvent struct {
	Type     string          `json:"type"`
	Delta    string          `json:"delta,omitempty"`
	Response *codexResponse  `json:"response,omitempty"`
	Error    json.RawMessage `json:"error,omitempty"`
}

type chatGPTModelsResponse struct {
	Models []chatGPTModel `json:"models"`
}

type chatGPTModel struct {
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type codexResponse struct {
	ID        string      `json:"id"`
	CreatedAt int64       `json:"created_at"`
	Model     string      `json:"model"`
	Usage     *codexUsage `json:"usage,omitempty"`
}

type codexUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

func (u codexUsage) toDomain() domain.Usage {
	return domain.Usage{
		PromptTokens:     u.InputTokens,
		CompletionTokens: u.OutputTokens,
		TotalTokens:      u.TotalTokens,
	}
}

func (p *Provider) refreshAccessToken(ctx context.Context, force bool) (*config.OpenAIOAuth, error) {
	oauth := p.refreshOAuth()
	if oauth == nil || oauth.RefreshToken == "" {
		return oauth, nil
	}
	if !force && oauth.AccessToken != "" {
		return oauth, nil
	}

	clientID := os.Getenv(envOpenAIOAuthClientID)
	if clientID == "" {
		return oauth, nil
	}

	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", oauth.RefreshToken)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, authURL, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return oauth, err
	}
	httpReq.Header.Set(headerContentType, "application/x-www-form-urlencoded")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return oauth, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return oauth, fmt.Errorf("openai oauth refresh error %d: %s", resp.StatusCode, body)
	}

	var refreshed oauthRefreshResponse
	if err := json.NewDecoder(resp.Body).Decode(&refreshed); err != nil {
		return oauth, fmt.Errorf("failed to decode openai oauth refresh: %w", err)
	}
	if refreshed.AccessToken == "" {
		return oauth, fmt.Errorf("openai oauth refresh did not return access_token")
	}

	next := *oauth
	if refreshed.AccountID != "" {
		next.AccountID = refreshed.AccountID
	}
	if refreshed.ChatGPTAccountID != "" {
		next.AccountID = refreshed.ChatGPTAccountID
	}
	if refreshed.IDToken != "" {
		next.IDToken = refreshed.IDToken
	}
	next.AccessToken = refreshed.AccessToken
	if refreshed.RefreshToken != "" {
		next.RefreshToken = refreshed.RefreshToken
	}

	p.mu.Lock()
	p.oauth = &next
	source := p.source
	p.mu.Unlock()

	if source != "" {
		_ = config.SaveOpenAIOAuthSource(source, &next)
	}

	return &next, nil
}

func New(apiKey string, oauth *config.OpenAIOAuth) *Provider {
	return &Provider{
		apiKey: apiKey,
		oauth:  oauth,
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *Provider) setAuthHeaders(httpReq *http.Request) error {
	if p.apiKey != "" {
		httpReq.Header.Set(headerAuthorization, tokenPrefixBearer+p.apiKey)
		return nil
	}

	oauth, err := p.refreshAccessToken(httpReq.Context(), false)
	if err != nil {
		return err
	}
	if oauth != nil {
		if oauth.AccessToken != "" {
			httpReq.Header.Set(headerAuthorization, tokenPrefixBearer+oauth.AccessToken)
		} else if oauth.IDToken != "" {
			httpReq.Header.Set(headerAuthorization, tokenPrefixBearer+oauth.IDToken)
		}
		if oauth.AccountID != "" {
			httpReq.Header.Set(headerChatGPTAcctID, oauth.AccountID)
		}
	}
	return nil
}

func (p *Provider) doOpenAIRequest(ctx context.Context, method, path string, body []byte, contentType string) (*http.Response, error) {
	resp, err := p.doOpenAIRequestOnce(ctx, method, path, body, contentType)
	if err != nil || resp.StatusCode != http.StatusUnauthorized || p.apiKey != "" {
		return resp, err
	}

	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	if _, err := p.refreshAccessToken(ctx, true); err != nil {
		return nil, err
	}
	return p.doOpenAIRequestOnce(ctx, method, path, body, contentType)
}

func (p *Provider) doOpenAIRequestOnce(ctx context.Context, method, path string, body []byte, contentType string) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, baseURL+path, reader)
	if err != nil {
		return nil, err
	}
	if err := p.setAuthHeaders(httpReq); err != nil {
		return nil, err
	}
	if contentType != "" {
		httpReq.Header.Set(headerContentType, contentType)
	}
	return p.client.Do(httpReq)
}

func (p *Provider) Name() string { return domain.ProviderOpenAI }

func (p *Provider) Chat(ctx context.Context, req domain.ChatRequest) (*domain.ChatResponse, error) {
	if p.apiKey == "" {
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
	if p.apiKey == "" {
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

func (p *Provider) chatCodex(ctx context.Context, req domain.ChatRequest) (*domain.ChatResponse, error) {
	resp, err := p.doCodexRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai codex error %d: %s", resp.StatusCode, b)
	}

	content, usage, responseID, created, model, err := parseCodexStream(resp.Body, nil)
	if err != nil {
		return nil, err
	}
	if responseID == "" {
		responseID = "chatcmpl-codex"
	}
	if created == 0 {
		created = time.Now().Unix()
	}
	if model == "" {
		model = req.Model
	}

	return &domain.ChatResponse{
		ID:      responseID,
		Object:  "chat.completion",
		Created: created,
		Model:   model,
		Choices: []domain.Choice{
			{
				Index: 0,
				Message: domain.Message{
					Role:    domain.RoleAssistant,
					Content: content,
				},
				FinishReason: "stop",
			},
		},
		Usage: usage,
	}, nil
}

func (p *Provider) chatCodexStream(ctx context.Context, req domain.ChatRequest, w io.Writer) (domain.Usage, error) {
	resp, err := p.doCodexRequest(ctx, req)
	if err != nil {
		return domain.Usage{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return domain.Usage{}, fmt.Errorf("openai codex stream error %d: %s", resp.StatusCode, b)
	}

	_, usage, _, _, _, err := parseCodexStream(resp.Body, w)
	return usage, err
}

func (p *Provider) doCodexRequest(ctx context.Context, req domain.ChatRequest) (*http.Response, error) {
	body, err := json.Marshal(toCodexResponseRequest(req))
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, chatGPTURL+pathCodexResponses, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if err := p.setAuthHeaders(httpReq); err != nil {
		return nil, err
	}
	httpReq.Header.Set(headerContentType, contentTypeJSON)
	httpReq.Header.Set(headerOpenAIBeta, codexResponsesExperimental)
	httpReq.Header.Set(headerOriginator, codexOriginator)
	httpReq.Header.Set(headerSessionID, newCodexSessionID())
	httpReq.Header.Set(headerUserAgent, "")
	httpReq.Header.Set(headerVersion, getEnv(envOpenAICodexVersion, codexDefaultVersion))

	return p.client.Do(httpReq)
}

func toCodexResponseRequest(req domain.ChatRequest) codexResponseRequest {
	instructions := make([]string, 0)
	input := make([]codexInputMessage, 0, len(req.Messages))

	for _, msg := range req.Messages {
		if msg.Role == domain.RoleSystem {
			if msg.Content != "" {
				instructions = append(instructions, msg.Content)
			}
			continue
		}

		role := msg.Role
		if role == domain.RoleAssistant || role == domain.RoleTool {
			role = domain.RoleUser
		}
		if role == "" {
			role = domain.RoleUser
		}
		input = append(input, codexInputMessage{
			Role: role,
			Content: []codexInputContent{
				{Type: "input_text", Text: msg.Content},
			},
		})
	}

	if len(instructions) == 0 {
		instructions = append(instructions, "You are a helpful assistant.")
	}
	if len(input) == 0 {
		input = append(input, codexInputMessage{
			Role: domain.RoleUser,
			Content: []codexInputContent{
				{Type: "input_text", Text: ""},
			},
		})
	}

	maxOutputTokens := req.MaxCompletionTokens
	if maxOutputTokens == nil {
		maxOutputTokens = req.MaxTokens
	}

	out := codexResponseRequest{
		Model:           req.Model,
		Instructions:    strings.Join(instructions, "\n\n"),
		Input:           input,
		Stream:          true,
		Store:           false,
		MaxOutputTokens: maxOutputTokens,
	}
	return out
}

func parseCodexStream(body io.Reader, w io.Writer) (string, domain.Usage, string, int64, string, error) {
	var content strings.Builder
	var usage domain.Usage
	var responseID string
	var created int64
	var model string

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024*64), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		payload := strings.TrimPrefix(line, "data: ")
		var event codexStreamEvent
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			continue
		}

		if event.Response != nil {
			if event.Response.ID != "" {
				responseID = event.Response.ID
			}
			if event.Response.CreatedAt != 0 {
				created = event.Response.CreatedAt
			}
			if event.Response.Model != "" {
				model = event.Response.Model
			}
			if event.Response.Usage != nil {
				usage = event.Response.Usage.toDomain()
			}
		}

		switch event.Type {
		case "response.output_text.delta":
			content.WriteString(event.Delta)
			if w != nil {
				if responseID == "" {
					responseID = "chatcmpl-codex"
				}
				if created == 0 {
					created = time.Now().Unix()
				}
				if err := writeOpenAIStreamDelta(w, responseID, created, model, event.Delta); err != nil {
					return content.String(), usage, responseID, created, model, err
				}
			}
		case "response.failed":
			return content.String(), usage, responseID, created, model, fmt.Errorf("openai codex response failed: %s", event.Error)
		}
	}
	if err := scanner.Err(); err != nil {
		return content.String(), usage, responseID, created, model, err
	}

	if usage.TotalTokens == 0 {
		usage.CompletionTokens = shared.EstimateTokens(content.String())
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}
	if w != nil {
		shared.InjectUsageChunk(w, usage)
	}

	return content.String(), usage, responseID, created, model, nil
}

func writeOpenAIStreamDelta(w io.Writer, id string, created int64, model string, delta string) error {
	chunk := map[string]interface{}{
		"id":      id,
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   model,
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"delta": map[string]string{
					"content": delta,
				},
				"finish_reason": nil,
			},
		},
	}
	b, err := json.Marshal(chunk)
	if err != nil {
		return err
	}
	if _, err := io.WriteString(w, "data: "+string(b)+"\n\n"); err != nil {
		return err
	}
	if f, ok := w.(interface{ Flush() }); ok {
		f.Flush()
	}
	return nil
}

func newCodexSessionID() string {
	return fmt.Sprintf("ai-gateway-%d", time.Now().UnixNano())
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
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
	if p.apiKey == "" {
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

func (p *Provider) IsConfigured() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.apiKey != "" || (p.oauth != nil && (p.oauth.AccessToken != "" || p.oauth.RefreshToken != "" || p.oauth.IDToken != ""))
}

func (p *Provider) RefreshReady(ctx context.Context) bool {
	if p.apiKey != "" {
		p.mu.Lock()
		p.source = "OPENAI_API_KEY"
		p.mu.Unlock()
		return true
	}

	if p.refreshOAuth() == nil {
		return false
	}

	if ctx == nil {
		ctx = context.Background()
	}
	return p.Ping(ctx) == nil
}

func (p *Provider) ReadySummary() string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.apiKey != "" {
		return "api_key (OPENAI_API_KEY)"
	}
	if p.source != "" {
		return "oauth_access_token (" + p.source + ")"
	}
	return p.authMode()
}

func (p *Provider) Ping(ctx context.Context) error {
	if p.apiKey == "" {
		return p.pingOAuth(ctx)
	}

	resp, err := p.doOpenAIRequest(ctx, http.MethodHead, pathModels, nil, "")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(resp.Body)
		if isInsufficientQuota(body) {
			return fmt.Errorf("%s: openai ping failed (status %d): %s", openAIInsufficientQuotaMsg, resp.StatusCode, body)
		}
		return fmt.Errorf("openai ping failed (status %d): %s", resp.StatusCode, body)
	}
	return nil
}

func (p *Provider) pingOAuth(ctx context.Context) error {
	req := domain.ChatRequest{
		Model: authProbeModel,
		Messages: []domain.Message{
			{Role: domain.RoleUser, Content: "ping"},
		},
		MaxCompletionTokens: intPtr(1),
	}
	body, _ := json.Marshal(req)

	resp, err := p.doOpenAIRequest(ctx, http.MethodPost, pathChatCompletions, body, contentTypeJSON)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusTooManyRequests {
		io.Copy(io.Discard, resp.Body)
		return nil
	}

	b, _ := io.ReadAll(resp.Body)
	if isInsufficientQuota(b) {
		return fmt.Errorf("%s: openai oauth ping failed (status %d): %s", openAIInsufficientQuotaMsg, resp.StatusCode, b)
	}
	return fmt.Errorf("openai oauth ping failed (status %d): %s", resp.StatusCode, b)
}

func intPtr(value int) *int {
	return &value
}

func isInsufficientQuota(body []byte) bool {
	return bytes.Contains(body, []byte(`"code": "insufficient_quota"`)) ||
		bytes.Contains(body, []byte(`"code":"insufficient_quota"`))
}

func (p *Provider) Usage(ctx context.Context) (any, error) {
	if p.apiKey == "" {
		return loadLatestCodexUsageSnapshot()
	}

	values := url.Values{}
	values.Set("start_time", strconv.FormatInt(time.Now().AddDate(0, 0, -7).Unix(), 10))
	values.Set("bucket_width", "1d")
	values.Set("limit", "7")

	resp, err := p.doOpenAIRequest(ctx, http.MethodGet, pathUsageCompletions+"?"+values.Encode(), nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai usage error %d: %s", resp.StatusCode, b)
	}

	var usage any
	if err := json.NewDecoder(resp.Body).Decode(&usage); err != nil {
		return nil, fmt.Errorf("failed to decode openai usage: %w", err)
	}
	return usage, nil
}

type codexUsageSnapshot struct {
	Provider   string `json:"provider"`
	AuthMode   string `json:"auth_mode"`
	Source     string `json:"source"`
	UpdatedAt  string `json:"updated_at"`
	Display    any    `json:"display,omitempty"`
	Limits     any    `json:"limits,omitempty"`
	RateLimits any    `json:"rate_limits"`
	TokenUsage any    `json:"token_usage,omitempty"`
}

type codexUsageDisplay struct {
	FiveHour string `json:"5h_limit,omitempty"`
	Weekly   string `json:"weekly_limit,omitempty"`
}

type codexLimitDisplay struct {
	Label         string  `json:"label"`
	WindowMinutes int     `json:"window_minutes"`
	UsedPercent   float64 `json:"used_percent"`
	LeftPercent   float64 `json:"left_percent"`
	ResetsAt      int64   `json:"resets_at,omitempty"`
	ResetText     string  `json:"reset_text,omitempty"`
	Text          string  `json:"text"`
}

func loadLatestCodexUsageSnapshot() (codexUsageSnapshot, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return codexUsageSnapshot{}, err
	}

	sessionsRoot := filepath.Join(home, ".codex", "sessions")
	var latest codexUsageSnapshot
	var latestAt time.Time

	err = filepath.WalkDir(sessionsRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".jsonl") {
			return nil
		}

		snapshot, ts, err := latestCodexUsageSnapshotInFile(path)
		if err != nil || ts.IsZero() || (!latestAt.IsZero() && !ts.After(latestAt)) {
			return nil
		}

		latest = snapshot
		latestAt = ts
		return nil
	})
	if err != nil {
		return codexUsageSnapshot{}, err
	}
	if latestAt.IsZero() {
		return codexUsageSnapshot{}, fmt.Errorf("codex usage snapshot not found in %s", sessionsRoot)
	}

	return latest, nil
}

func latestCodexUsageSnapshotInFile(path string) (codexUsageSnapshot, time.Time, error) {
	file, err := os.Open(path)
	if err != nil {
		return codexUsageSnapshot{}, time.Time{}, err
	}
	defer file.Close()

	var latest codexUsageSnapshot
	var latestAt time.Time
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var event struct {
			Timestamp string `json:"timestamp"`
			Type      string `json:"type"`
			Payload   struct {
				Type       string `json:"type"`
				Info       any    `json:"info,omitempty"`
				RateLimits any    `json:"rate_limits,omitempty"`
			} `json:"payload"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			continue
		}
		if event.Type != "event_msg" || event.Payload.Type != "token_count" || event.Payload.RateLimits == nil {
			continue
		}

		ts, err := time.Parse(time.RFC3339Nano, event.Timestamp)
		if err != nil || (!latestAt.IsZero() && !ts.After(latestAt)) {
			continue
		}

		latestAt = ts
		display, limits := formatCodexUsageLimits(event.Payload.RateLimits)
		latest = codexUsageSnapshot{
			Provider:   domain.ProviderOpenAI,
			AuthMode:   "oauth",
			Source:     "codex_session",
			UpdatedAt:  event.Timestamp,
			Display:    display,
			Limits:     limits,
			RateLimits: event.Payload.RateLimits,
			TokenUsage: event.Payload.Info,
		}
	}
	if err := scanner.Err(); err != nil {
		return codexUsageSnapshot{}, time.Time{}, err
	}
	return latest, latestAt, nil
}

func formatCodexUsageLimits(rateLimits any) (codexUsageDisplay, map[string]codexLimitDisplay) {
	limits := make(map[string]codexLimitDisplay)
	limitMap, ok := rateLimits.(map[string]any)
	if !ok {
		return codexUsageDisplay{}, limits
	}

	if primary, ok := limitMap["primary"].(map[string]any); ok {
		limits["5h"] = formatCodexLimit("5h limit", primary)
	}
	if secondary, ok := limitMap["secondary"].(map[string]any); ok {
		limits["weekly"] = formatCodexLimit("Weekly limit", secondary)
	}

	display := codexUsageDisplay{}
	if limit, ok := limits["5h"]; ok {
		display.FiveHour = limit.Text
	}
	if limit, ok := limits["weekly"]; ok {
		display.Weekly = limit.Text
	}
	return display, limits
}

func formatCodexLimit(label string, limit map[string]any) codexLimitDisplay {
	usedPercent := numberFromMap(limit, "used_percent")
	leftPercent := clampPercent(100 - usedPercent)
	resetAt := int64(numberFromMap(limit, "resets_at"))
	windowMinutes := int(numberFromMap(limit, "window_minutes"))
	resetText := formatCodexResetText(resetAt, windowMinutes)

	text := fmt.Sprintf("%s: %.0f%% left", label, leftPercent)
	if resetText != "" {
		text += " (resets " + resetText + ")"
	}

	return codexLimitDisplay{
		Label:         label,
		WindowMinutes: windowMinutes,
		UsedPercent:   usedPercent,
		LeftPercent:   leftPercent,
		ResetsAt:      resetAt,
		ResetText:     resetText,
		Text:          text,
	}
}

func numberFromMap(values map[string]any, key string) float64 {
	switch value := values[key].(type) {
	case float64:
		return value
	case int:
		return float64(value)
	case int64:
		return float64(value)
	case json.Number:
		n, _ := value.Float64()
		return n
	default:
		return 0
	}
}

func clampPercent(value float64) float64 {
	return math.Max(0, math.Min(100, value))
}

func formatCodexResetText(resetAt int64, windowMinutes int) string {
	if resetAt == 0 {
		return ""
	}

	reset := time.Unix(resetAt, 0).Local()
	if windowMinutes >= 10080 {
		return reset.Format("15:04 on 2 Jan")
	}
	return reset.Format("15:04")
}

func (p *Provider) IsReady() bool   { return p.ready }
func (p *Provider) SetReady(r bool) { p.ready = r }
