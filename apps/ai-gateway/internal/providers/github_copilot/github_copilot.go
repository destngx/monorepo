package github_copilot

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"apps/ai-gateway/internal/domain"
	"apps/ai-gateway/internal/providers/shared"

	"golang.org/x/sync/singleflight"
)

const (
	baseURL                    = "https://api.githubcopilot.com"
	tokenURL                   = "https://api.github.com/copilot_internal/v2/token"
	userURL                    = "https://api.github.com/copilot_internal/user"
	defaultEditorVersion       = "vscode/1.80.0"
	defaultEditorPluginVersion = "copilot-chat/0.1.0"
	defaultIntegrationID       = "vscode-chat"
	defaultUserAgent           = "GitHubCopilotChat/0.1.0"

	// Model constants
	ModelGPT5Mini      = "gpt-5-mini"
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
	headerIntegrationID   = "Copilot-Integration-Id"

	contentTypeJSON   = domain.ContentTypeJSON
	tokenPrefixBearer = "Bearer "
	tokenPrefixToken  = "token "

	sseDataPrefix = "data: "
	sseDone       = "[DONE]"

	pathChatCompletions = "/chat/completions"
	pathResponses       = "/responses"
	pathModels          = "/models"

	// GitHub Copilot API limits (https://github.com/github/copilot-cli/issues/509)
	maxToolCallsPerMessage = 128
)

type Provider struct {
	githubToken string
	accountType string
	client      *http.Client
	ready       bool
	verbose     int
	headers     ClientHeaders

	mu             sync.Mutex
	cachedToken    string
	expiresAt      int64
	copilotAPIBase string
	tokenGroup     singleflight.Group
}

type ClientHeaders struct {
	EditorVersion       string
	EditorPluginVersion string
	IntegrationID       string
	UserAgent           string
}

func (p *Provider) Name() string { return domain.ProviderGitHubCopilot }

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
	var reasoningContentBuilder strings.Builder
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
					Content          string `json:"content"`
					ReasoningContent string `json:"reasoning_content"`
					ToolCalls        []struct {
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
				reasoningContentBuilder.WriteString(delta.ReasoningContent)

				for _, tc := range delta.ToolCalls {
					if existing, ok := toolCallsMap[tc.Index]; ok {
						existing.Function.Arguments += tc.Function.Arguments
					} else {
						toolCallsMap[tc.Index] = &domain.ToolCall{
							ID:   tc.ID,
							Type: tc.Type,
							Function: &domain.FunctionCall{
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
				Role:             "assistant",
				Content:          contentBuilder.String(),
				ReasoningContent: reasoningContentBuilder.String(),
				ToolCalls:        targetToolCalls,
			},
			FinishReason: finishReason,
		},
	}

	p.vlogf(1, "[github-copilot] assembled sync response took=%s", time.Since(start))
	return &result, nil
}

func (p *Provider) ChatStream(ctx context.Context, req domain.ChatRequest, w io.Writer) (domain.Usage, error) {
	if isResponsesModel(req.Model) {
		return p.chatResponsesStream(ctx, req, w)
	}

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
