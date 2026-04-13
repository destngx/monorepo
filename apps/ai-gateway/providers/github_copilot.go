package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"apps/ai-gateway/types"
)

const githubCopilotBaseURL = "https://api.githubcopilot.com"
const githubCopilotTokenURL = "https://api.github.com/copilot_internal/v2/token"
const githubCopilotEditorVersion = "vscode/1.80.0"
const githubCopilotPluginVersion = "copilot-chat/0.1.0"
const githubCopilotUserAgent = "curl/8.7.1"

type GitHubCopilotProvider struct {
	githubToken string
	client      *http.Client
	ready       bool

	mu             sync.Mutex
	cachedToken    string
	expiresAt      int64
	copilotAPIBase string
	defaultBaseURL string
	tokenURL       string
}

type githubCopilotTokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
	Endpoints struct {
		API string `json:"api"`
	} `json:"endpoints"`
}

func NewGitHubCopilot(githubToken string) *GitHubCopilotProvider {
	return &GitHubCopilotProvider{
		githubToken:    githubToken,
		client:         &http.Client{Timeout: 120 * time.Second},
		defaultBaseURL: githubCopilotBaseURL,
		tokenURL:       githubCopilotTokenURL,
	}
}

func (g *GitHubCopilotProvider) Name() string { return "github-copilot" }

func (g *GitHubCopilotProvider) Chat(ctx context.Context, req types.ChatRequest) (*types.ChatResponse, error) {
	httpReq, err := g.newChatRequest(ctx, req, false)
	if err != nil {
		return nil, err
	}

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github copilot error %d: %s", resp.StatusCode, b)
	}

	var result types.ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (g *GitHubCopilotProvider) ChatStream(ctx context.Context, req types.ChatRequest, w io.Writer) (types.Usage, error) {
	httpReq, err := g.newChatRequest(ctx, req, true)
	if err != nil {
		return types.Usage{}, err
	}

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return types.Usage{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return types.Usage{}, fmt.Errorf("github copilot error %d: %s", resp.StatusCode, b)
	}

	return streamSSEAndCountTokens(resp.Body, w)
}

func (g *GitHubCopilotProvider) Embeddings(ctx context.Context, req types.EmbeddingRequest) (*types.EmbeddingResponse, error) {
	return nil, fmt.Errorf("github copilot does not support embeddings")
}

func (g *GitHubCopilotProvider) ListModels(ctx context.Context) (*types.ModelsResponse, error) {
	// Copilot does not expose a general-purpose models catalog endpoint for this gateway.
	// Return the curated aliases we support for compatibility with the legacy frontend.
	return &types.ModelsResponse{
		Object: "list",
		Data: []types.ModelInfo{
			{ID: "gpt-4.1", Object: "model", OwnedBy: "github-copilot"},
			{ID: "gpt-4o", Object: "model", OwnedBy: "github-copilot"},
			{ID: "grok-code-fast-1", Object: "model", OwnedBy: "github-copilot"},
			{ID: "gemini-3-pro-preview", Object: "model", OwnedBy: "github-copilot"},
			{ID: "claude-haiku-4.5", Object: "model", OwnedBy: "github-copilot"},
		},
	}, nil
}

func (g *GitHubCopilotProvider) IsConfigured() bool {
	return g.githubToken != ""
}

func (g *GitHubCopilotProvider) Ping(ctx context.Context) error {
	if !g.IsConfigured() {
		return fmt.Errorf("github copilot not configured")
	}
	return nil
}

func (g *GitHubCopilotProvider) IsReady() bool   { return g.ready }
func (g *GitHubCopilotProvider) SetReady(r bool) { g.ready = r }

func (g *GitHubCopilotProvider) getCopilotToken(ctx context.Context) (string, error) {
	g.mu.Lock()
	if g.cachedToken != "" && g.expiresAt > time.Now().Unix()+300 {
		token := g.cachedToken
		g.mu.Unlock()
		return token, nil
	}
	g.mu.Unlock()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, g.tokenURL, nil)
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Authorization", "token "+g.githubToken)
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", githubCopilotUserAgent)
	httpReq.Header.Set("Editor-Version", githubCopilotEditorVersion)
	httpReq.Header.Set("Editor-Plugin-Version", githubCopilotPluginVersion)

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("github copilot token error %d: %s", resp.StatusCode, b)
	}

	var tokenResp githubCopilotTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode github copilot token: %w", err)
	}
	if tokenResp.Token == "" {
		return "", fmt.Errorf("github copilot token response missing token")
	}

	g.mu.Lock()
	g.cachedToken = tokenResp.Token
	g.expiresAt = tokenResp.ExpiresAt
	if tokenResp.Endpoints.API != "" {
		g.copilotAPIBase = tokenResp.Endpoints.API
	} else {
		g.copilotAPIBase = g.defaultBaseURL
	}
	g.mu.Unlock()

	return tokenResp.Token, nil
}

func (g *GitHubCopilotProvider) newChatRequest(ctx context.Context, req types.ChatRequest, stream bool) (*http.Request, error) {
	if stream {
		req.Stream = true
		req.StreamOptions = &types.StreamOptions{IncludeUsage: true}
	}

	payload, err := githubCopilotPayload(req)
	if err != nil {
		return nil, err
	}

	token, err := g.getCopilotToken(ctx)
	if err != nil {
		return nil, err
	}

	baseURL := g.apiBaseURL()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Editor-Version", githubCopilotEditorVersion)
	httpReq.Header.Set("Editor-Plugin-Version", githubCopilotPluginVersion)
	httpReq.Header.Set("User-Agent", githubCopilotUserAgent)

	return httpReq, nil
}

func (g *GitHubCopilotProvider) apiBaseURL() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.copilotAPIBase != "" {
		return g.copilotAPIBase
	}
	return g.defaultBaseURL
}

func githubCopilotPayload(req types.ChatRequest) ([]byte, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	pruneNilValues(payload)
	fixCopilotToolSchema(payload)

	return json.Marshal(payload)
}

func pruneNilValues(v any) any {
	switch node := v.(type) {
	case map[string]any:
		for key, value := range node {
			if value == nil {
				delete(node, key)
				continue
			}
			node[key] = pruneNilValues(value)
		}
		return node
	case []any:
		for i, value := range node {
			node[i] = pruneNilValues(value)
		}
		return node
	default:
		return v
	}
}

func fixCopilotToolSchema(payload map[string]any) {
	tools, ok := payload["tools"].([]any)
	if !ok {
		return
	}

	for _, rawTool := range tools {
		tool, ok := rawTool.(map[string]any)
		if !ok {
			continue
		}
		fn, ok := tool["function"].(map[string]any)
		if !ok {
			continue
		}
		params, ok := fn["parameters"].(map[string]any)
		if !ok {
			continue
		}
		typeValue, hasType := params["type"]
		if !hasType {
			params["type"] = "object"
			continue
		}
		typeString, ok := typeValue.(string)
		if ok && strings.EqualFold(typeString, "none") {
			params["type"] = "object"
		}
	}
}
