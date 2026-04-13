package github_copilot

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

	"apps/ai-gateway/internal/domain"
	"apps/ai-gateway/internal/providers/shared"
	"golang.org/x/sync/singleflight"
)

const (
	baseURL       = "https://api.githubcopilot.com"
	tokenURL      = "https://api.github.com/copilot_internal/v2/token"
	userURL       = "https://api.github.com/copilot_internal/user"
	editorVersion = "vscode/1.80.0"
	pluginVersion = "copilot-chat/0.1.0"
	userAgent     = "curl/8.7.1"

	// Model constants
	ModelGPT41         = "gpt-4.1"
	ModelGPT4o         = "gpt-4o"
	ModelGrokCodeFast1 = "grok-code-fast-1"
	ModelGemini3Pro    = "gemini-3-pro-preview"
	ModelClaudeHaiku45 = "claude-haiku-4.5"

	logPrefix = "[github-copilot] "

	headerAuthorization   = "Authorization"
	headerAccept          = "Accept"
	headerUserAgent       = "User-Agent"
	headerContentType     = "Content-Type"
	headerEditorVersion   = "Editor-Version"
	headerEditorPluginVer = "Editor-Plugin-Version"

	contentTypeJSON   = "application/json"
	tokenPrefixBearer = "Bearer "
	tokenPrefixToken  = "token "

	sseDataPrefix = "data: "
	sseDone       = "[DONE]"

	pathChatCompletions = "/chat/completions"
)

type Provider struct {
	githubToken string
	client      *http.Client
	ready       bool
	verbose     int

	mu             sync.Mutex
	cachedToken    string
	expiresAt      int64
	copilotAPIBase string
	tokenGroup     singleflight.Group
}

type tokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
	Endpoints struct {
		API string `json:"api"`
	} `json:"endpoints"`
}

func New(githubToken string, verbose int) *Provider {
	return &Provider{
		githubToken: githubToken,
		client:      &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *Provider) Name() string { return "github-copilot" }

func (p *Provider) Chat(ctx context.Context, req domain.ChatRequest) (*domain.ChatResponse, error) {
	start := time.Now()
	p.vlogf(1, "[github-copilot] chat start model=%q", req.Model)

	// GitHub Copilot enforces stream=true on many models (e.g. Anthropic/OpenAI models on its edge).
	// To support sync requests, we force stream=true upstream and assemble the SSE chunks into a single response.
	req.Stream = true

	pr, pw := io.Pipe()
	errCh := make(chan error, 1)
	go func() {
		defer pw.Close()
		_, err := p.ChatStream(ctx, req, pw)
		if err != nil {
			p.vlogf(1, "[github-copilot] stream fallback error: %v", err)
		}
		errCh <- err
	}()

	scanner := bufio.NewScanner(pr)
	scanner.Buffer(make([]byte, 1024*64), 1024*64)
	var contentBuilder strings.Builder
	var rawBodyBuilder strings.Builder
	toolCallsMap := make(map[int]*domain.ToolCall)
	var finalUsage domain.Usage
	var lastChunk map[string]any

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			if strings.TrimSpace(line) != "" {
				rawBodyBuilder.WriteString(line)
			}
			continue
		}
		data := strings.TrimPrefix(line, sseDataPrefix)
		if data == sseDone {
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
			Usage *domain.Usage `json:"usage"`
		}

		if err := json.Unmarshal([]byte(data), &chunk); err == nil {
			if len(chunk.Choices) > 0 {
				delta := chunk.Choices[0].Delta
				contentBuilder.WriteString(delta.Content)

				for _, tc := range delta.ToolCalls {
					if existing, ok := toolCallsMap[tc.Index]; ok {
						existing.Function.Arguments += tc.Function.Arguments
					} else {
						toolCallsMap[tc.Index] = &domain.ToolCall{
							ID:   tc.ID,
							Type: tc.Type,
							Function: domain.FunctionCall{
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
		var directResp domain.ChatResponse
		if err := json.Unmarshal([]byte(rawBodyBuilder.String()), &directResp); err == nil && len(directResp.Choices) > 0 {
			p.vlogf(1, "[github-copilot] assembled sync response took=%s", time.Since(start))
			return &directResp, streamErr
		}
	}

	if streamErr != nil {
		return nil, streamErr
	}

	var result domain.ChatResponse
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

	var targetToolCalls []domain.ToolCall
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

	result.Choices = []domain.Choice{
		{
			Message: domain.Message{
				Role:      "assistant",
				Content:   contentBuilder.String(),
				ToolCalls: targetToolCalls,
			},
			FinishReason: finishReason,
		},
	}

	p.vlogf(1, "[github-copilot] assembled sync response took=%s", time.Since(start))
	return &result, nil
}

func (p *Provider) ChatStream(ctx context.Context, req domain.ChatRequest, w io.Writer) (domain.Usage, error) {
	start := time.Now()
	p.vlogf(1, "[github-copilot] stream start model=%q", req.Model)

	payloadStart := time.Now()
	httpReq, err := p.newChatRequest(ctx, req, true)
	if err != nil {
		return domain.Usage{}, err
	}
	p.vlogf(2, "[github-copilot] stream payload+session build took=%s", time.Since(payloadStart))

	callStart := time.Now()
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return domain.Usage{}, err
	}
	defer resp.Body.Close()
	p.vlogf(1, "[github-copilot] upstream stream call took=%s status=%d", time.Since(callStart), resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return domain.Usage{}, fmt.Errorf("github copilot error %d: %s", resp.StatusCode, b)
	}

	usage, err := shared.StreamSSEAndCountTokens(resp.Body, w)
	p.vlogf(1, "[github-copilot] stream total took=%s", time.Since(start))
	return usage, err
}

func (p *Provider) Embeddings(ctx context.Context, req domain.EmbeddingRequest) (*domain.EmbeddingResponse, error) {
	return nil, fmt.Errorf("github copilot does not support embeddings")
}

func (p *Provider) ListModels(ctx context.Context) (*domain.ModelsResponse, error) {
	return &domain.ModelsResponse{
		Object: "list",
		Data: []domain.ModelInfo{
			{ID: ModelGPT41, Object: "model", OwnedBy: "github-copilot"},
			{ID: ModelGPT4o, Object: "model", OwnedBy: "github-copilot"},
			{ID: ModelGrokCodeFast1, Object: "model", OwnedBy: "github-copilot"},
			{ID: ModelGemini3Pro, Object: "model", OwnedBy: "github-copilot"},
			{ID: ModelClaudeHaiku45, Object: "model", OwnedBy: "github-copilot"},
		},
	}, nil
}

func (p *Provider) IsConfigured() bool {
	return p.githubToken != ""
}

func (p *Provider) Ping(ctx context.Context) error {
	if !p.IsConfigured() {
		return fmt.Errorf("github copilot not configured")
	}
	return nil
}

func (p *Provider) Usage(ctx context.Context) (any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set(headerAuthorization, tokenPrefixToken+p.githubToken)
	req.Header.Set(headerAccept, contentTypeJSON)
	req.Header.Set(headerUserAgent, userAgent)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github copilot usage error %d: %s", resp.StatusCode, b)
	}

	var usage UsageResponse
	if err := json.NewDecoder(resp.Body).Decode(&usage); err != nil {
		return nil, err
	}

	return usage, nil
}

func (p *Provider) IsReady() bool   { return p.ready }
func (p *Provider) SetReady(r bool) { p.ready = r }

func (p *Provider) getCopilotSession(ctx context.Context) (string, string, error) {
	p.mu.Lock()
	if p.cachedToken != "" && p.expiresAt > time.Now().Unix()+300 {
		token := p.cachedToken
		apiBase := p.copilotAPIBase
		if apiBase == "" {
			apiBase = baseURL
		}
		p.mu.Unlock()
		p.vlogf(2, "[github-copilot] session cache hit token=hit base_url=%q", apiBase)
		return token, apiBase, nil
	}
	p.mu.Unlock()

	sessionStart := time.Now()
	v, err, _ := p.tokenGroup.Do("copilot-token", func() (any, error) {
		p.vlogf(1, "[github-copilot] session cache miss, fetching token")
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL, nil)
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set(headerAuthorization, tokenPrefixToken+p.githubToken)
		httpReq.Header.Set(headerAccept, contentTypeJSON)
		httpReq.Header.Set(headerUserAgent, userAgent)
		httpReq.Header.Set(headerEditorVersion, editorVersion)
		httpReq.Header.Set(headerEditorPluginVer, pluginVersion)

		resp, err := p.client.Do(httpReq)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("github copilot token error %d: %s", resp.StatusCode, b)
		}

		var tr tokenResponse
		if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
			return nil, fmt.Errorf("failed to decode github copilot token: %w", err)
		}
		if tr.Token == "" {
			return nil, fmt.Errorf("github copilot token response missing token")
		}

		apiBase := baseURL
		if tr.Endpoints.API != "" {
			apiBase = tr.Endpoints.API
		}

		p.mu.Lock()
		p.cachedToken = tr.Token
		p.expiresAt = tr.ExpiresAt
		p.copilotAPIBase = apiBase
		p.mu.Unlock()
		p.vlogf(1, "[github-copilot] token fetch took=%s base_url=%q expires_at=%d", time.Since(sessionStart), apiBase, tr.ExpiresAt)

		return sessionData{
			token:   tr.Token,
			baseURL: apiBase,
		}, nil
	})
	if err != nil {
		return "", "", err
	}
	session := v.(sessionData)
	p.vlogf(2, "[github-copilot] session ready base_url=%q", session.baseURL)
	return session.token, session.baseURL, nil
}

type sessionData struct {
	token   string
	baseURL string
}

func (p *Provider) newChatRequest(ctx context.Context, req domain.ChatRequest, stream bool) (*http.Request, error) {
	payloadStart := time.Now()
	if stream {
		req.Stream = true
		req.StreamOptions = &domain.StreamOptions{IncludeUsage: true}
	}

	payload, err := p.buildPayload(req)
	if err != nil {
		return nil, err
	}
	p.vlogf(2, "[github-copilot] payload marshal took=%s", time.Since(payloadStart))

	token, apiBase, err := p.getCopilotSession(ctx)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiBase+pathChatCompletions, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set(headerAuthorization, tokenPrefixBearer+token)
	httpReq.Header.Set(headerContentType, contentTypeJSON)
	httpReq.Header.Set(headerEditorVersion, editorVersion)
	httpReq.Header.Set(headerEditorPluginVer, pluginVersion)
	httpReq.Header.Set(headerUserAgent, userAgent)

	return httpReq, nil
}

func (p *Provider) vlogf(level int, format string, args ...any) {
	if p.verbose < level {
		return
	}
	log.Printf(format, args...)
}

func (p *Provider) buildPayload(req domain.ChatRequest) ([]byte, error) {
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
				Parameters:  sanitizeParameters(tool.Function.Parameters),
			},
		})
	}

	return json.Marshal(payload)
}

type copilotChatRequest struct {
	Model         string                `json:"model"`
	Messages      []domain.Message      `json:"messages"`
	Stream        bool                  `json:"stream"`
	StreamOptions *domain.StreamOptions `json:"stream_options,omitempty"`
	Temperature   *float64              `json:"temperature,omitempty"`
	TopP          *float64              `json:"top_p,omitempty"`
	MaxTokens     *int                  `json:"max_tokens,omitempty"`
	Stop          any                   `json:"stop,omitempty"`
	N             *int                  `json:"n,omitempty"`
	Tools         []copilotTool         `json:"tools,omitempty"`
	ToolChoice    any                   `json:"tool_choice,omitempty"`
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

func sanitizeParameters(v any) any {
	return sanitizeNode(v, true)
}

func sanitizeNode(v any, normalizeSelf bool) any {
	switch node := v.(type) {
	case map[string]any:
		if !normalizeSelf {
			out := make(map[string]any, len(node))
			for key, value := range node {
				if value == nil {
					continue
				}
				out[key] = sanitizeNode(value, true)
			}
			return out
		}

		out := make(map[string]any, len(node))
		for key, value := range node {
			if value == nil {
				continue
			}
			if shouldDropKey(key) {
				continue
			}
			out[key] = sanitizeNode(value, !isSchemaContainerKey(key))
		}
		normalizeSchemaMap(out)
		return out
	case []any:
		out := make([]any, 0, len(node))
		for _, value := range node {
			if value == nil {
				continue
			}
			out = append(out, sanitizeNode(value, true))
		}
		return out
	default:
		return v
	}
}

func shouldDropKey(key string) bool {
	switch key {
	case "$schema", "$id", "default":
		return true
	default:
		return false
	}
}

func isSchemaContainerKey(key string) bool {
	switch key {
	case "properties", "patternProperties", "$defs", "definitions", "dependentSchemas":
		return true
	default:
		return false
	}
}

func normalizeSchemaMap(schema map[string]any) {
	if typeValue, ok := schema["type"].(string); ok {
		if strings.EqualFold(typeValue, "none") {
			schema["type"] = "object"
		}
		return
	}

	switch {
	case hasKey(schema, "properties"),
		hasKey(schema, "required"),
		hasKey(schema, "additionalProperties"),
		hasKey(schema, "propertyNames"):
		schema["type"] = "object"
	case hasKey(schema, "items"):
		schema["type"] = "array"
	}
}

func hasKey(schema map[string]any, key string) bool {
	_, ok := schema[key]
	return ok
}
