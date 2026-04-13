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

const (
	githubCopilotBaseURL       = "https://api.githubcopilot.com"
	githubCopilotTokenURL      = "https://api.github.com/copilot_internal/v2/token"
	githubCopilotUserURL       = "https://api.github.com/copilot_internal/user"
	githubCopilotEditorVersion = "vscode/1.80.0"
	githubCopilotPluginVersion = "copilot-chat/0.1.0"
	githubCopilotUserAgent     = "curl/8.7.1"

	ProviderGitHubCopilot = "github-copilot"

	ModelGPT41         = "gpt-4.1"
	ModelGPT4o         = "gpt-4o"
	ModelGrokCodeFast1 = "grok-code-fast-1"
	ModelGemini3Pro    = "gemini-3-pro-preview"
	ModelClaudeHaiku45 = "claude-haiku-4.5"

	LogPrefix = "[github-copilot] "

	LogFormatChatStart      = LogPrefix + "chat start model=%q"
	LogFormatStreamFallback = LogPrefix + "stream fallback error: %v"
	LogFormatSyncAssembled  = LogPrefix + "assembled sync response took=%s"
	LogFormatStreamStart    = LogPrefix + "stream start model=%q"
	LogFormatPayloadBuild   = LogPrefix + "stream payload+session build took=%s"
	LogFormatUpstreamCall   = LogPrefix + "upstream stream call took=%s status=%d"
	LogFormatStreamTotal    = LogPrefix + "stream total took=%s"
	LogFormatCacheHit       = LogPrefix + "session cache hit token=hit base_url=%q"
	LogFormatCacheMiss      = LogPrefix + "session cache miss, fetching token"
	LogFormatTokenFetch     = LogPrefix + "token fetch took=%s base_url=%q expires_at=%d"
	LogFormatSessionReady   = LogPrefix + "session ready base_url=%q"
	LogFormatPayloadMarshal = LogPrefix + "payload marshal took=%s"

	ErrNotConfigured        = "github copilot not configured"
	ErrNotSupportedEmbed    = "github copilot does not support embeddings"
	ErrMissingTokenResponse = "github copilot token response missing token"
)

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

func (g *GitHubCopilotProvider) Name() string { return ProviderGitHubCopilot }

func (g *GitHubCopilotProvider) Chat(ctx context.Context, req types.ChatRequest) (*types.ChatResponse, error) {
	start := time.Now()
	g.vlogf(1, LogFormatChatStart, req.Model)

	// GitHub Copilot enforces stream=true on many models (e.g. Anthropic/OpenAI models on its edge).
	// To support sync requests, we force stream=true upstream and assemble the SSE chunks into a single response.
	req.Stream = true

	pr, pw := io.Pipe()
	errCh := make(chan error, 1)
	go func() {
		defer pw.Close()
		_, err := g.ChatStream(ctx, req, pw)
		if err != nil {
			g.vlogf(1, LogFormatStreamFallback, err)
		}
		errCh <- err
	}()

	scanner := bufio.NewScanner(pr)
	scanner.Buffer(make([]byte, 1024*64), 1024*64)
	var contentBuilder strings.Builder
	var rawBodyBuilder strings.Builder
	toolCallsMap := make(map[int]*types.ToolCall)
	var finalUsage types.Usage
	var lastChunk map[string]any

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			if strings.TrimSpace(line) != "" {
				rawBodyBuilder.WriteString(line)
			}
			continue
		}
		data := strings.TrimPrefix(line, SSEDataPrefix)
		if data == SSEDone {
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

	streamErr := <-errCh

	if rawBodyBuilder.Len() > 0 && contentBuilder.Len() == 0 && len(toolCallsMap) == 0 {
		var directResp types.ChatResponse
		if err := json.Unmarshal([]byte(rawBodyBuilder.String()), &directResp); err == nil && len(directResp.Choices) > 0 {
			g.vlogf(1, LogFormatSyncAssembled, time.Since(start))
			return &directResp, streamErr
		}
	}

	if streamErr != nil {
		return nil, streamErr
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

	g.vlogf(1, LogFormatSyncAssembled, time.Since(start))
	return &result, nil
}

func (g *GitHubCopilotProvider) ChatStream(ctx context.Context, req types.ChatRequest, w io.Writer) (types.Usage, error) {
	start := time.Now()
	g.vlogf(1, LogFormatStreamStart, req.Model)

	payloadStart := time.Now()
	httpReq, err := g.newChatRequest(ctx, req, true)
	if err != nil {
		return types.Usage{}, err
	}
	g.vlogf(2, LogFormatPayloadBuild, time.Since(payloadStart))

	callStart := time.Now()
	resp, err := g.client.Do(httpReq)
	if err != nil {
		return types.Usage{}, err
	}
	defer resp.Body.Close()
	g.vlogf(1, LogFormatUpstreamCall, time.Since(callStart), resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return types.Usage{}, fmt.Errorf("github copilot error %d: %s", resp.StatusCode, b)
	}

	usage, err := streamSSEAndCountTokens(resp.Body, w)
	g.vlogf(1, LogFormatStreamTotal, time.Since(start))
	return usage, err
}

func (g *GitHubCopilotProvider) Embeddings(ctx context.Context, req types.EmbeddingRequest) (*types.EmbeddingResponse, error) {
	return nil, fmt.Errorf(ErrNotSupportedEmbed)
}

func (g *GitHubCopilotProvider) ListModels(ctx context.Context) (*types.ModelsResponse, error) {
	// Copilot does not expose a general-purpose models catalog endpoint for this gateway.
	// Return the curated aliases we support for compatibility with the legacy frontend.
	return &types.ModelsResponse{
		Object: "list",
		Data: []types.ModelInfo{
			{ID: ModelGPT41, Object: "model", OwnedBy: ProviderGitHubCopilot},
			{ID: ModelGPT4o, Object: "model", OwnedBy: ProviderGitHubCopilot},
			{ID: ModelGrokCodeFast1, Object: "model", OwnedBy: ProviderGitHubCopilot},
			{ID: ModelGemini3Pro, Object: "model", OwnedBy: ProviderGitHubCopilot},
			{ID: ModelClaudeHaiku45, Object: "model", OwnedBy: ProviderGitHubCopilot},
		},
	}, nil
}

func (g *GitHubCopilotProvider) IsConfigured() bool {
	return g.githubToken != ""
}

func (g *GitHubCopilotProvider) Ping(ctx context.Context) error {
	if !g.IsConfigured() {
		return fmt.Errorf(ErrNotConfigured)
	}
	return nil
}

func (g *GitHubCopilotProvider) Usage(ctx context.Context) (any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubCopilotUserURL, nil)
	if err != nil {
		return nil, err
	}

	// Copilot uses the primary github token for user identification
	req.Header.Set(HeaderAuthorization, TokenPrefixToken+g.githubToken)
	req.Header.Set(HeaderAccept, ContentTypeJSON)
	req.Header.Set(HeaderUserAgent, githubCopilotUserAgent)

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
		g.vlogf(2, LogFormatCacheHit, baseURL)
		return token, baseURL, nil
	}
	g.mu.Unlock()

	sessionStart := time.Now()
	v, err, _ := g.tokenGroup.Do("copilot-token", func() (any, error) {
		g.vlogf(1, LogFormatCacheMiss)
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, g.tokenURL, nil)
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set(HeaderAuthorization, TokenPrefixToken+g.githubToken)
		httpReq.Header.Set(HeaderAccept, ContentTypeJSON)
		httpReq.Header.Set(HeaderUserAgent, githubCopilotUserAgent)
		httpReq.Header.Set(HeaderEditorVersion, githubCopilotEditorVersion)
		httpReq.Header.Set(HeaderEditorPluginVer, githubCopilotPluginVersion)

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
			return nil, fmt.Errorf(ErrMissingTokenResponse)
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
		g.vlogf(1, LogFormatTokenFetch, time.Since(sessionStart), baseURL, tokenResp.ExpiresAt)

		return githubCopilotSession{
			token:   tokenResp.Token,
			baseURL: baseURL,
		}, nil
	})
	if err != nil {
		return "", "", err
	}
	session := v.(githubCopilotSession)
	g.vlogf(2, LogFormatSessionReady, session.baseURL)
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
	g.vlogf(2, LogFormatPayloadMarshal, time.Since(payloadStart))

	token, baseURL, err := g.getCopilotSession(ctx)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+PathChatCompletions, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set(HeaderAuthorization, TokenPrefixBearer+token)
	httpReq.Header.Set(HeaderContentType, ContentTypeJSON)
	httpReq.Header.Set(HeaderEditorVersion, githubCopilotEditorVersion)
	httpReq.Header.Set(HeaderEditorPluginVer, githubCopilotPluginVersion)
	httpReq.Header.Set(HeaderUserAgent, githubCopilotUserAgent)

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
	return sanitizeCopilotParametersNode(v, true)
}

func sanitizeCopilotParametersNode(v any, normalizeSelf bool) any {
	switch node := v.(type) {
	case map[string]any:
		if !normalizeSelf {
			out := make(map[string]any, len(node))
			for key, value := range node {
				if value == nil {
					continue
				}
				out[key] = sanitizeCopilotParametersNode(value, true)
			}
			return out
		}

		out := make(map[string]any, len(node))
		for key, value := range node {
			if value == nil {
				continue
			}
			if shouldDropCopilotSchemaKey(key) {
				continue
			}
			out[key] = sanitizeCopilotParametersNode(value, !isCopilotSchemaContainerKey(key))
		}
		normalizeCopilotSchemaMap(out)
		return out
	case []any:
		out := make([]any, 0, len(node))
		for _, value := range node {
			if value == nil {
				continue
			}
			out = append(out, sanitizeCopilotParametersNode(value, true))
		}
		return out
	default:
		return v
	}
}

func shouldDropCopilotSchemaKey(key string) bool {
	switch key {
	case "$schema", "$id", "default":
		return true
	default:
		return false
	}
}

func isCopilotSchemaContainerKey(key string) bool {
	switch key {
	case "properties", "patternProperties", "$defs", "definitions", "dependentSchemas":
		return true
	default:
		return false
	}
}

func normalizeCopilotSchemaMap(schema map[string]any) {
	if typeValue, ok := schema["type"].(string); ok {
		if strings.EqualFold(typeValue, "none") {
			schema["type"] = "object"
		}
		return
	}

	switch {
	case hasSchemaKey(schema, "properties"),
		hasSchemaKey(schema, "required"),
		hasSchemaKey(schema, "additionalProperties"),
		hasSchemaKey(schema, "propertyNames"):
		schema["type"] = "object"
	case hasSchemaKey(schema, "items"):
		schema["type"] = "array"
	}
}

func hasSchemaKey(schema map[string]any, key string) bool {
	_, ok := schema[key]
	return ok
}
