package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"apps/ai-gateway/internal/domain"
)

const (
	baseURL    = "https://api.anthropic.com/v1"
	apiVersion = "2023-06-01"

	headerAPIKey           = "x-api-key"
	headerAnthropicVersion = "anthropic-version"
	headerContentType      = "Content-Type"
	contentTypeJSON        = "application/json"

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

func (p *Provider) Name() string { return "anthropic" }

// ConvertToAnthropicRequest transforms an OpenAI-compatible request into
// Anthropic's Messages API format. Extracts system messages separately.
func (p *Provider) ConvertToAnthropicRequest(req domain.ChatRequest) Request {
	var system string
	messages := make([]Message, 0, len(req.Messages))

	for _, m := range req.Messages {
		if m.Role == roleSystem {
			if system != "" {
				system += "\n\n"
			}
			system += m.Content
			continue
		}

		if m.Role == roleTool {
			// OpenAI tool role -> Anthropic tool_result block in a user message
			messages = append(messages, Message{
				Role: roleUser,
				Content: []Content{{
					Type:    typeToolResult,
					ToolID:  m.ToolCallID,
					Content: m.Content,
				}},
			})
			continue
		}

		if m.Role == roleAssistant && len(m.ToolCalls) > 0 {
			// OpenAI assistant message with tool calls -> Anthropic assistant message with tool_use blocks
			content := make([]Content, 0, len(m.ToolCalls)+1)
			if m.Content != "" {
				content = append(content, Content{Type: typeText, Text: m.Content})
			}
			for _, tc := range m.ToolCalls {
				var input any
				json.Unmarshal([]byte(tc.Function.Arguments), &input)
				content = append(content, Content{
					Type:  typeToolUse,
					ID:    tc.ID,
					Name:  tc.Function.Name,
					Input: input,
				})
			}
			messages = append(messages, Message{
				Role:    roleAssistant,
				Content: content,
			})
			continue
		}

		messages = append(messages, Message{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	maxTokens := 4096
	if req.MaxTokens != nil {
		maxTokens = *req.MaxTokens
	}

	ar := Request{
		Model:       req.Model,
		MaxTokens:   maxTokens,
		System:      system,
		Messages:    messages,
		Temperature: req.Temperature,
		TopP:        req.TopP,
	}

	// Convert tools
	for _, t := range req.Tools {
		ar.Tools = append(ar.Tools, Tool{
			Name:        t.Function.Name,
			Description: t.Function.Description,
			InputSchema: t.Function.Parameters,
		})
	}

	// Convert stop sequences
	if req.Stop != nil {
		switch v := req.Stop.(type) {
		case string:
			ar.StopSeqs = []string{v}
		case []interface{}:
			for _, s := range v {
				if str, ok := s.(string); ok {
					ar.StopSeqs = append(ar.StopSeqs, str)
				}
			}
		}
	}

	return ar
}

func (p *Provider) headers() map[string]string {
	return map[string]string{
		headerAPIKey:           p.apiKey,
		headerAnthropicVersion: apiVersion,
		headerContentType:      contentTypeJSON,
	}
}

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
		if c.Type == typeText {
			content += c.Text
		} else if c.Type == typeToolUse {
			args, _ := json.Marshal(c.Input)
			toolCalls = append(toolCalls, domain.ToolCall{
				ID:   c.ID,
				Type: "function",
				Function: domain.FunctionCall{
					Name:      c.Name,
					Arguments: string(args),
				},
			})
		}
	}

	finishReason := finishReasonStop
	if ar2.StopReason == stopReasonMaxTokens {
		finishReason = finishReasonLength
	} else if ar2.StopReason == stopReasonToolUse {
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

// convertStreamToOpenAI reads Anthropic's SSE stream and re-emits it as
// OpenAI-compatible SSE events so the client sees a uniform format.
func (p *Provider) convertStreamToOpenAI(body io.Reader, w io.Writer) (domain.Usage, error) {
	var usage domain.Usage
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024*64), 1024*64)

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, sseDataPrefix) {
			continue
		}
		payload := strings.TrimPrefix(line, sseDataPrefix)

		var event map[string]interface{}
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			continue
		}

		eventType, _ := event["type"].(string)
		switch eventType {
		case eventMessageStart:
			msg, _ := event["message"].(map[string]interface{})
			if u, ok := msg["usage"].(map[string]interface{}); ok {
				if inputTokens, ok := u["input_tokens"].(float64); ok {
					usage.PromptTokens = int(inputTokens)
				}
			}

		case eventContentBlockStart:
			index, _ := event["index"].(float64)
			block, _ := event["content_block"].(map[string]interface{})
			blockType, _ := block["type"].(string)

			if blockType == typeToolUse {
				id, _ := block["id"].(string)
				name, _ := block["name"].(string)
				chunk := map[string]interface{}{
					"object": objectChatCompletionChunk,
					"choices": []map[string]interface{}{{
						"index": 0,
						"delta": map[string]interface{}{
							"tool_calls": []map[string]interface{}{{
								"index":    int(index),
								"id":       id,
								"type":     "function",
								"function": map[string]string{"name": name},
							}},
						},
					}},
				}
				b, _ := json.Marshal(chunk)
				io.WriteString(w, sseDataPrefix+string(b)+"\n\n")
			}

		case eventContentBlockDelta:
			index, _ := event["index"].(float64)
			delta, _ := event["delta"].(map[string]interface{})
			deltaType, _ := delta["type"].(string)

			chunk := map[string]interface{}{
				"object":  objectChatCompletionChunk,
				"choices": []map[string]interface{}{{"index": 0}},
			}

			if deltaType == typeTextDelta {
				text, _ := delta["text"].(string)
				chunk["choices"].([]map[string]interface{})[0]["delta"] = map[string]string{"content": text}
			} else if deltaType == typeInputJSON {
				partial, _ := delta["partial_json"].(string)
				chunk["choices"].([]map[string]interface{})[0]["delta"] = map[string]interface{}{
					"tool_calls": []map[string]interface{}{{
						"index":    int(index),
						"function": map[string]string{"arguments": partial},
					}},
				}
			}

			b, _ := json.Marshal(chunk)
			if _, err := io.WriteString(w, "data: "+string(b)+"\n\n"); err != nil {
				return usage, err
			}
			if f, ok := w.(interface{ Flush() }); ok {
				f.Flush()
			}

		case eventMessageDelta:
			// Extract usage from the final message_delta event
			u, _ := event["usage"].(map[string]interface{})
			if outputTokens, ok := u["output_tokens"].(float64); ok {
				usage.CompletionTokens = int(outputTokens)
			}

		case eventMessageStop:
			usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
			io.WriteString(w, sseDataPrefix+sseDone+"\n\n")
			if f, ok := w.(interface{ Flush() }); ok {
				f.Flush()
			}
		}
	}

	if usage.TotalTokens == 0 {
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}

	return usage, scanner.Err()
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
