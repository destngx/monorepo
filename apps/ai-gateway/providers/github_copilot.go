package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"apps/ai-gateway/types"
	"golang.org/x/sync/singleflight"
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
	verbose     int

	mu             sync.Mutex
	cachedToken    string
	expiresAt      int64
	copilotAPIBase string
	defaultBaseURL string
	tokenURL       string
	tokenGroup     singleflight.Group
}

type githubCopilotTokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
	Endpoints struct {
		API string `json:"api"`
	} `json:"endpoints"`
}

func NewGitHubCopilot(githubToken string, verbose int) *GitHubCopilotProvider {
	return &GitHubCopilotProvider{
		githubToken:    githubToken,
		client:         &http.Client{Timeout: 120 * time.Second},
		defaultBaseURL: githubCopilotBaseURL,
		tokenURL:       githubCopilotTokenURL,
		verbose:        verbose,
	}
}

func (g *GitHubCopilotProvider) Name() string { return "github-copilot" }

func (g *GitHubCopilotProvider) Chat(ctx context.Context, req types.ChatRequest) (*types.ChatResponse, error) {
	start := time.Now()
	g.vlogf(1, "[github-copilot] chat start model=%q", req.Model)

	// GitHub Copilot enforces stream=true on many models (e.g. Anthropic/OpenAI models on its edge).
	// To support sync requests, we force stream=true upstream and assemble the SSE chunks into a single response.
	req.Stream = true

	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		_, err := g.ChatStream(ctx, req, pw)
		if err != nil {
			g.vlogf(1, "[github-copilot] stream fallback error: %v", err)
		}
	}()

	scanner := bufio.NewScanner(pr)
	var contentBuilder strings.Builder
	toolCallsMap := make(map[int]*types.ToolCall)
	var finalUsage types.Usage
	var lastChunk map[string]any

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content   string `json:"content"`
					ToolCalls []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Type     string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *types.Usage `json:"usage"`
		}

		if err := json.Unmarshal([]byte(data), &chunk); err == nil {
			if len(chunk.Choices) > 0 {
				delta := chunk.Choices[0].Delta
				contentBuilder.WriteString(delta.Content)

				for _, tc := range delta.ToolCalls {
					if existing, ok := toolCallsMap[tc.Index]; ok {
						existing.Function.Arguments += tc.Function.Arguments
					} else {
						toolCallsMap[tc.Index] = &types.ToolCall{
							ID:   tc.ID,
							Type: tc.Type,
							Function: types.FunctionCall{
								Name:      tc.Function.Name,
								Arguments: tc.Function.Arguments,
							},
						}
						if toolCallsMap[tc.Index].Type == "" {
							toolCallsMap[tc.Index].Type = "function"
						}
					}
				}
			}
			if chunk.Usage != nil {
				finalUsage = *chunk.Usage
			}
		}

		if len(data) > 0 {
			json.Unmarshal([]byte(data), &lastChunk)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	var result types.ChatResponse
	if lastChunk != nil {
		if id, ok := lastChunk["id"].(string); ok {
			result.ID = id
		}
		if m, ok := lastChunk["model"].(string); ok {
			result.Model = m
		}
	}
	if result.ID == "" {
		result.ID = "chatcmpl-copilot-assembled"
	}
	if result.Model == "" {
		result.Model = req.Model
	}

	result.Usage = finalUsage

	var targetToolCalls []types.ToolCall
	if len(toolCallsMap) > 0 {
		maxIdx := -1
		for k := range toolCallsMap {
			if k > maxIdx {
				maxIdx = k
			}
		}
		for i := 0; i <= maxIdx; i++ {
			if tc, ok := toolCallsMap[i]; ok {
				targetToolCalls = append(targetToolCalls, *tc)
			}
		}
	}

	finishReason := "stop"
	if lastChunk != nil {
		if choices, ok := lastChunk["choices"].([]any); ok && len(choices) > 0 {
			if c, ok := choices[0].(map[string]any); ok {
				if fr, ok := c["finish_reason"].(string); ok && fr != "" {
					finishReason = fr
				}
			}
		}
	}
	if len(targetToolCalls) > 0 {
		finishReason = "tool_calls"
	}

	result.Choices = []types.Choice{
		{
			Message: types.Message{
				Role:      "assistant",
				Content:   contentBuilder.String(),
				ToolCalls: targetToolCalls,
			},
			FinishReason: finishReason,
		},
	}

	g.vlogf(1, "[github-copilot] assembled sync response took=%s", time.Since(start))
	return &result, nil
}

func (g *GitHubCopilotProvider) ChatStream(ctx context.Context, req types.ChatRequest, w io.Writer) (types.Usage, error) {
	start := time.Now()
	g.vlogf(1, "[github-copilot] stream start model=%q", req.Model)

	payloadStart := time.Now()
	httpReq, err := g.newChatRequest(ctx, req, true)
	if err != nil {
		return types.Usage{}, err
	}
	g.vlogf(2, "[github-copilot] stream payload+session build took=%s", time.Since(payloadStart))

	callStart := time.Now()
	resp, err := g.client.Do(httpReq)
	if err != nil {
		return types.Usage{}, err
	}
	defer resp.Body.Close()
	g.vlogf(1, "[github-copilot] upstream stream call took=%s status=%d", time.Since(callStart), resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return types.Usage{}, fmt.Errorf("github copilot error %d: %s", resp.StatusCode, b)
	}

	usage, err := streamSSEAndCountTokens(resp.Body, w)
	g.vlogf(1, "[github-copilot] stream total took=%s", time.Since(start))
	return usage, err
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

func (g *GitHubCopilotProvider) Usage(ctx context.Context) (any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/copilot_internal/user", nil)
	if err != nil {
		return nil, err
	}

	// Copilot uses the primary github token for user identification
	req.Header.Set("Authorization", "token "+g.githubToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", githubCopilotUserAgent)

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github copilot usage error %d: %s", resp.StatusCode, b)
	}

	var usage types.CopilotUsageResponse
	if err := json.NewDecoder(resp.Body).Decode(&usage); err != nil {
		return nil, err
	}

	return usage, nil
}

func (g *GitHubCopilotProvider) IsReady() bool   { return g.ready }
func (g *GitHubCopilotProvider) SetReady(r bool) { g.ready = r }

func (g *GitHubCopilotProvider) getCopilotSession(ctx context.Context) (string, string, error) {
	g.mu.Lock()
	if g.cachedToken != "" && g.expiresAt > time.Now().Unix()+300 {
		token := g.cachedToken
		baseURL := g.copilotAPIBase
		if baseURL == "" {
			baseURL = g.defaultBaseURL
		}
		g.mu.Unlock()
		g.vlogf(2, "[github-copilot] session cache hit token=hit base_url=%q", baseURL)
		return token, baseURL, nil
	}
	g.mu.Unlock()

	sessionStart := time.Now()
	v, err, _ := g.tokenGroup.Do("copilot-token", func() (any, error) {
		g.vlogf(1, "[github-copilot] session cache miss, fetching token")
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, g.tokenURL, nil)
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Authorization", "token "+g.githubToken)
		httpReq.Header.Set("Accept", "application/json")
		httpReq.Header.Set("User-Agent", githubCopilotUserAgent)
		httpReq.Header.Set("Editor-Version", githubCopilotEditorVersion)
		httpReq.Header.Set("Editor-Plugin-Version", githubCopilotPluginVersion)

		resp, err := g.client.Do(httpReq)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("github copilot token error %d: %s", resp.StatusCode, b)
		}

		var tokenResp githubCopilotTokenResponse
		if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
			return nil, fmt.Errorf("failed to decode github copilot token: %w", err)
		}
		if tokenResp.Token == "" {
			return nil, fmt.Errorf("github copilot token response missing token")
		}

		baseURL := g.defaultBaseURL
		if tokenResp.Endpoints.API != "" {
			baseURL = tokenResp.Endpoints.API
		}

		g.mu.Lock()
		g.cachedToken = tokenResp.Token
		g.expiresAt = tokenResp.ExpiresAt
		g.copilotAPIBase = baseURL
		g.mu.Unlock()
		g.vlogf(1, "[github-copilot] token fetch took=%s base_url=%q expires_at=%d", time.Since(sessionStart), baseURL, tokenResp.ExpiresAt)

		return githubCopilotSession{
			token:   tokenResp.Token,
			baseURL: baseURL,
		}, nil
	})
	if err != nil {
		return "", "", err
	}
	session := v.(githubCopilotSession)
	g.vlogf(2, "[github-copilot] session ready base_url=%q", session.baseURL)
	return session.token, session.baseURL, nil
}

func (g *GitHubCopilotProvider) newChatRequest(ctx context.Context, req types.ChatRequest, stream bool) (*http.Request, error) {
	payloadStart := time.Now()
	if stream {
		req.Stream = true
		req.StreamOptions = &types.StreamOptions{IncludeUsage: true}
	}

	payload, err := githubCopilotPayload(req)
	if err != nil {
		return nil, err
	}
	g.vlogf(2, "[github-copilot] payload marshal took=%s", time.Since(payloadStart))

	token, baseURL, err := g.getCopilotSession(ctx)
	if err != nil {
		return nil, err
	}
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

type githubCopilotSession struct {
	token   string
	baseURL string
}

func (g *GitHubCopilotProvider) vlogf(level int, format string, args ...any) {
	if g.verbose < level {
		return
	}
	log.Printf(format, args...)
}

func githubCopilotPayload(req types.ChatRequest) ([]byte, error) {
	payload := copilotChatRequest{
		Model:         req.Model,
		Messages:      req.Messages,
		Stream:        req.Stream,
		StreamOptions: req.StreamOptions,
		Temperature:   req.Temperature,
		TopP:          req.TopP,
		MaxTokens:     req.MaxTokens,
		Stop:          req.Stop,
		N:             req.N,
		Tools:         make([]copilotTool, 0, len(req.Tools)),
		ToolChoice:    req.ToolChoice,
	}

	for _, tool := range req.Tools {
		payload.Tools = append(payload.Tools, copilotTool{
			Type: tool.Type,
			Function: copilotFunctionDefinition{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  sanitizeCopilotParameters(tool.Function.Parameters),
			},
		})
	}

	return json.Marshal(payload)
}

type copilotChatRequest struct {
	Model         string               `json:"model"`
	Messages      []types.Message      `json:"messages"`
	Stream        bool                 `json:"stream"`
	StreamOptions *types.StreamOptions `json:"stream_options,omitempty"`
	Temperature   *float64             `json:"temperature,omitempty"`
	TopP          *float64             `json:"top_p,omitempty"`
	MaxTokens     *int                 `json:"max_tokens,omitempty"`
	Stop          any                  `json:"stop,omitempty"`
	N             *int                 `json:"n,omitempty"`
	Tools         []copilotTool        `json:"tools,omitempty"`
	ToolChoice    any                  `json:"tool_choice,omitempty"`
}

type copilotTool struct {
	Type     string                    `json:"type"`
	Function copilotFunctionDefinition `json:"function"`
}

type copilotFunctionDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

func sanitizeCopilotParameters(v any) any {
	switch node := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(node))
		for key, value := range node {
			if value == nil {
				continue
			}
			out[key] = sanitizeCopilotParameters(value)
		}
		if typeValue, ok := out["type"].(string); !ok || strings.EqualFold(typeValue, "none") {
			if _, hasType := out["type"]; !hasType {
				out["type"] = "object"
			} else if strings.EqualFold(typeValue, "none") {
				out["type"] = "object"
			}
		}
		return out
	case []any:
		out := make([]any, 0, len(node))
		for _, value := range node {
			if value == nil {
				continue
			}
			out = append(out, sanitizeCopilotParameters(value))
		}
		return out
	default:
		return v
	}
}
