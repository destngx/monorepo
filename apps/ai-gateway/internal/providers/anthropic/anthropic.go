package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"apps/ai-gateway/internal/domain"
)

const (
	baseURL    = "https://api.anthropic.com/v1"
	apiVersion = "2023-06-01"

	headerAPIKey           = "x-api-key"
	headerAnthropicVersion = "anthropic-version"
	headerContentType      = "Content-Type"
	contentTypeJSON        = domain.ContentTypeJSON

	roleSystem    = "system"
	roleUser      = "user"
	roleAssistant = "assistant"
	roleTool      = "tool"

	typeMessage    = "message"
	typeText       = "text"
	typeToolUse    = "tool_use"
	typeToolResult = "tool_result"
	typeTextDelta  = "text_delta"
	typeInputJSON  = "input_json_delta"

	stopReasonMaxTokens = "max_tokens"
	stopReasonToolUse   = "tool_use"

	finishReasonLength    = "length"
	finishReasonToolCalls = "tool_calls"
	finishReasonStop      = "stop"

	eventMessageStart      = "message_start"
	eventMessageDelta      = "message_delta"
	eventMessageStop       = "message_stop"
	eventContentBlockStart = "content_block_start"
	eventContentBlockDelta = "content_block_delta"

	pathMessages = "/messages"

	objectChatCompletion      = "chat.completion"
	objectChatCompletionChunk = "chat.completion.chunk"

	sseDataPrefix = "data: "
	sseDone       = "[DONE]"
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

func (p *Provider) Name() string { return domain.ProviderAnthropic }

func (p *Provider) Chat(ctx context.Context, req domain.ChatRequest) (*domain.ChatResponse, error) {
	ar := p.ConvertToAnthropicRequest(req)
	ar.Stream = false

	body, _ := json.Marshal(ar)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		baseURL+pathMessages, bytes.NewReader(body))
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
		return nil, fmt.Errorf("anthropic error %d: %s", resp.StatusCode, b)
	}

	var ar2 Response
	if err := json.NewDecoder(resp.Body).Decode(&ar2); err != nil {
		return nil, err
	}

	// Convert Anthropic response back to OpenAI format
	var content string
	var toolCalls []domain.ToolCall
	for _, c := range ar2.Content {
		switch c.Type {
		case typeText:
			content += c.Text
		case typeToolUse:
			args, _ := json.Marshal(c.Input)
			toolCalls = append(toolCalls, domain.ToolCall{
				ID:   c.ID,
				Type: "function",
				Function: &domain.FunctionCall{
					Name:      c.Name,
					Arguments: string(args),
				},
			})
		}
	}

	finishReason := finishReasonStop
	switch ar2.StopReason {
	case stopReasonMaxTokens:
		finishReason = finishReasonLength
	case stopReasonToolUse:
		finishReason = finishReasonToolCalls
	}

	return &domain.ChatResponse{
		ID:     ar2.ID,
		Object: objectChatCompletion,
		Model:  ar2.Model,
		Choices: []domain.Choice{{
			Index: 0,
			Message: domain.Message{
				Role:      roleAssistant,
				Content:   content,
				ToolCalls: toolCalls,
			},
			FinishReason: finishReason,
		}},
		Usage: domain.Usage{
			PromptTokens:     ar2.Usage.InputTokens,
			CompletionTokens: ar2.Usage.OutputTokens,
			TotalTokens:      ar2.Usage.InputTokens + ar2.Usage.OutputTokens,
		},
	}, nil
}

func (p *Provider) ChatStream(ctx context.Context, req domain.ChatRequest, w io.Writer) (domain.Usage, error) {
	ar := p.ConvertToAnthropicRequest(req)
	ar.Stream = true

	body, _ := json.Marshal(ar)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		baseURL+pathMessages, bytes.NewReader(body))
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
		return domain.Usage{}, fmt.Errorf("anthropic error %d: %s", resp.StatusCode, b)
	}

	// Convert Anthropic SSE stream to OpenAI-compatible SSE format
	return p.convertStreamToOpenAI(resp.Body, w)
}

func (p *Provider) Responses(ctx context.Context, req domain.ResponsesRequest) (*domain.ResponsesResponse, error) {
	return nil, domain.UnsupportedResponsesError(p.Name())
}

func (p *Provider) ResponsesStream(ctx context.Context, req domain.ResponsesRequest, w io.Writer) (domain.Usage, error) {
	return domain.Usage{}, domain.UnsupportedResponsesError(p.Name())
}

func (p *Provider) Embeddings(ctx context.Context, req domain.EmbeddingRequest) (*domain.EmbeddingResponse, error) {
	return nil, fmt.Errorf("embeddings not supported by anthropic")
}

func (p *Provider) ListModels(ctx context.Context) (*domain.ModelsResponse, error) {
	knownModels := []domain.ModelInfo{
		{ID: "claude-sonnet-4-20250514", Object: "model", OwnedBy: "anthropic"},
		{ID: "claude-haiku-4-20250514", Object: "model", OwnedBy: "anthropic"},
		{ID: "claude-3-7-sonnet-20250219", Object: "model", OwnedBy: "anthropic"},
		{ID: "claude-3-5-sonnet-20241022", Object: "model", OwnedBy: "anthropic"},
		{ID: "claude-3-5-haiku-20241022", Object: "model", OwnedBy: "anthropic"},
		{ID: "claude-3-opus-20240229", Object: "model", OwnedBy: "anthropic"},
	}
	return &domain.ModelsResponse{
		Object: "list",
		Data:   knownModels,
	}, nil
}

func (p *Provider) IsConfigured() bool {
	return p.apiKey != ""
}

func (p *Provider) Ping(ctx context.Context) error {
	// Anthropic doesn't have a simple GET ping; try HEAD on messages endpoint
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodHead, baseURL+pathMessages, nil)
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

	// 405 Method Not Allowed is acceptable for HEAD on a POST-only endpoint,
	// it proves the server is reachable and understands the URL.
	if resp.StatusCode >= 500 {
		return fmt.Errorf("anthropic service unavailable (status %d)", resp.StatusCode)
	}
	return nil
}

func (p *Provider) Usage(ctx context.Context) (any, error) {
	return nil, fmt.Errorf("usage API not supported for %s", p.Name())
}

func (p *Provider) IsReady() bool   { return p.ready }
func (p *Provider) SetReady(r bool) { p.ready = r }
