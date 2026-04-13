package providers

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

	"apps/ai-gateway/types"
)

const (
	anthropicBaseURL    = "https://api.anthropic.com/v1"
	anthropicAPIVersion = "2023-06-01"

	HeaderAPIKey           = "x-api-key"
	HeaderAnthropicVersion = "anthropic-version"

	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"

	TypeMessage    = "message"
	TypeText       = "text"
	TypeToolUse    = "tool_use"
	TypeToolResult = "tool_result"
	TypeTextDelta  = "text_delta"
	TypeInputJSON  = "input_json_delta"

	StopReasonMaxTokens = "max_tokens"
	StopReasonToolUse   = "tool_use"

	FinishReasonLength    = "length"
	FinishReasonToolCalls = "tool_calls"
	FinishReasonStop      = "stop"

	EventMessageStart      = "message_start"
	EventMessageDelta      = "message_delta"
	EventMessageStop       = "message_stop"
	EventContentBlockStart = "content_block_start"
	EventContentBlockDelta = "content_block_delta"

	PathMessages = "/messages"

	ObjectChatCompletion      = "chat.completion"
	ObjectChatCompletionChunk = "chat.completion.chunk"
)

type AnthropicProvider struct {
	apiKey string
	client *http.Client
	ready  bool
}

func NewAnthropic(apiKey string) *AnthropicProvider {
	return &AnthropicProvider{
		apiKey: apiKey,
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

func (a *AnthropicProvider) Name() string { return "anthropic" }

// ConvertToAnthropicRequest transforms an OpenAI-compatible request into
// Anthropic's Messages API format. Extracts system messages separately.
func ConvertToAnthropicRequest(req types.ChatRequest) types.AnthropicRequest {
	var system string
	messages := make([]types.AnthropicMessage, 0, len(req.Messages))

	for _, m := range req.Messages {
		if m.Role == RoleSystem {
			if system != "" {
				system += "\n\n"
			}
			system += m.Content
			continue
		}

		if m.Role == RoleTool {
			// OpenAI tool role -> Anthropic tool_result block in a user message
			messages = append(messages, types.AnthropicMessage{
				Role: RoleUser,
				Content: []types.AnthropicContent{{
					Type:    TypeToolResult,
					ToolID:  m.ToolCallID,
					Content: m.Content,
				}},
			})
			continue
		}

		if m.Role == RoleAssistant && len(m.ToolCalls) > 0 {
			// OpenAI assistant message with tool calls -> Anthropic assistant message with tool_use blocks
			content := make([]types.AnthropicContent, 0, len(m.ToolCalls)+1)
			if m.Content != "" {
				content = append(content, types.AnthropicContent{Type: TypeText, Text: m.Content})
			}
			for _, tc := range m.ToolCalls {
				var input any
				json.Unmarshal([]byte(tc.Function.Arguments), &input)
				content = append(content, types.AnthropicContent{
					Type:  TypeToolUse,
					ID:    tc.ID,
					Name:  tc.Function.Name,
					Input: input,
				})
			}
			messages = append(messages, types.AnthropicMessage{
				Role:    RoleAssistant,
				Content: content,
			})
			continue
		}

		messages = append(messages, types.AnthropicMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	maxTokens := 4096
	if req.MaxTokens != nil {
		maxTokens = *req.MaxTokens
	}

	ar := types.AnthropicRequest{
		Model:       req.Model,
		MaxTokens:   maxTokens,
		System:      system,
		Messages:    messages,
		Temperature: req.Temperature,
		TopP:        req.TopP,
	}

	// Convert tools
	for _, t := range req.Tools {
		ar.Tools = append(ar.Tools, types.AnthropicTool{
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

func (a *AnthropicProvider) headers() map[string]string {
	return map[string]string{
		HeaderAPIKey:           a.apiKey,
		HeaderAnthropicVersion: anthropicAPIVersion,
		HeaderContentType:      ContentTypeJSON,
	}
}

func (a *AnthropicProvider) Chat(ctx context.Context, req types.ChatRequest) (*types.ChatResponse, error) {
	ar := ConvertToAnthropicRequest(req)
	ar.Stream = false

	body, _ := json.Marshal(ar)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		anthropicBaseURL+PathMessages, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	for k, v := range a.headers() {
		httpReq.Header.Set(k, v)
	}

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("anthropic error %d: %s", resp.StatusCode, b)
	}

	var ar2 types.AnthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar2); err != nil {
		return nil, err
	}

	// Convert Anthropic response back to OpenAI format
	var content string
	var toolCalls []types.ToolCall
	for _, c := range ar2.Content {
		if c.Type == TypeText {
			content += c.Text
		} else if c.Type == TypeToolUse {
			args, _ := json.Marshal(c.Input)
			toolCalls = append(toolCalls, types.ToolCall{
				ID:   c.ID,
				Type: "function",
				Function: types.FunctionCall{
					Name:      c.Name,
					Arguments: string(args),
				},
			})
		}
	}

	finishReason := FinishReasonStop
	if ar2.StopReason == StopReasonMaxTokens {
		finishReason = FinishReasonLength
	} else if ar2.StopReason == StopReasonToolUse {
		finishReason = FinishReasonToolCalls
	}

	return &types.ChatResponse{
		ID:     ar2.ID,
		Object: ObjectChatCompletion,
		Model:  ar2.Model,
		Choices: []types.Choice{{
			Index: 0,
			Message: types.Message{
				Role:      RoleAssistant,
				Content:   content,
				ToolCalls: toolCalls,
			},
			FinishReason: finishReason,
		}},
		Usage: types.Usage{
			PromptTokens:     ar2.Usage.InputTokens,
			CompletionTokens: ar2.Usage.OutputTokens,
			TotalTokens:      ar2.Usage.InputTokens + ar2.Usage.OutputTokens,
		},
	}, nil
}

func (a *AnthropicProvider) ChatStream(ctx context.Context, req types.ChatRequest, w io.Writer) (types.Usage, error) {
	ar := ConvertToAnthropicRequest(req)
	ar.Stream = true

	body, _ := json.Marshal(ar)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		anthropicBaseURL+PathMessages, bytes.NewReader(body))
	if err != nil {
		return types.Usage{}, err
	}
	for k, v := range a.headers() {
		httpReq.Header.Set(k, v)
	}

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return types.Usage{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return types.Usage{}, fmt.Errorf("anthropic error %d: %s", resp.StatusCode, b)
	}

	// Convert Anthropic SSE stream to OpenAI-compatible SSE format
	return a.convertStreamToOpenAI(resp.Body, w)
}

// convertStreamToOpenAI reads Anthropic's SSE stream and re-emits it as
// OpenAI-compatible SSE events so the client sees a uniform format.
func (a *AnthropicProvider) convertStreamToOpenAI(body io.Reader, w io.Writer) (types.Usage, error) {
	var usage types.Usage
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024*64), 1024*64)

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, SSEDataPrefix) {
			continue
		}
		payload := strings.TrimPrefix(line, SSEDataPrefix)

		var event map[string]interface{}
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			continue
		}

		eventType, _ := event["type"].(string)
		switch eventType {
		case EventContentBlockStart:
			index, _ := event["index"].(float64)
			block, _ := event["content_block"].(map[string]interface{})
			blockType, _ := block["type"].(string)

			if blockType == TypeToolUse {
				id, _ := block["id"].(string)
				name, _ := block["name"].(string)
				chunk := map[string]interface{}{
					"object": ObjectChatCompletionChunk,
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
				io.WriteString(w, SSEDataPrefix+string(b)+"\n\n")
			}

		case EventContentBlockDelta:
			index, _ := event["index"].(float64)
			delta, _ := event["delta"].(map[string]interface{})
			deltaType, _ := delta["type"].(string)

			chunk := map[string]interface{}{
				"object":  ObjectChatCompletionChunk,
				"choices": []map[string]interface{}{{"index": 0}},
			}

			if deltaType == TypeTextDelta {
				text, _ := delta["text"].(string)
				chunk["choices"].([]map[string]interface{})[0]["delta"] = map[string]string{"content": text}
			} else if deltaType == TypeInputJSON {
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

		case EventMessageDelta:
			// Extract usage from the final message_delta event
			u, _ := event["usage"].(map[string]interface{})
			if outputTokens, ok := u["output_tokens"].(float64); ok {
				usage.CompletionTokens = int(outputTokens)
			}

		case EventMessageStart:
			msg, _ := event["message"].(map[string]interface{})
			if u, ok := msg["usage"].(map[string]interface{}); ok {
				if inputTokens, ok := u["input_tokens"].(float64); ok {
					usage.PromptTokens = int(inputTokens)
				}
			}

		case EventMessageStop:
			usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
			io.WriteString(w, SSEDataPrefix+SSEDone+"\n\n")
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

// ListModels returns a curated list of known Anthropic models.
// Anthropic does not provide a public model listing API.
func (a *AnthropicProvider) Embeddings(ctx context.Context, req types.EmbeddingRequest) (*types.EmbeddingResponse, error) {
	return nil, fmt.Errorf("embeddings not supported by anthropic")
}

func (a *AnthropicProvider) ListModels(ctx context.Context) (*types.ModelsResponse, error) {
	knownModels := []types.ModelInfo{
		{ID: "claude-sonnet-4-20250514", Object: "model", OwnedBy: "anthropic"},
		{ID: "claude-haiku-4-20250514", Object: "model", OwnedBy: "anthropic"},
		{ID: "claude-3-7-sonnet-20250219", Object: "model", OwnedBy: "anthropic"},
		{ID: "claude-3-5-sonnet-20241022", Object: "model", OwnedBy: "anthropic"},
		{ID: "claude-3-5-haiku-20241022", Object: "model", OwnedBy: "anthropic"},
		{ID: "claude-3-opus-20240229", Object: "model", OwnedBy: "anthropic"},
	}
	return &types.ModelsResponse{
		Object: "list",
		Data:   knownModels,
	}, nil
}

func (a *AnthropicProvider) IsConfigured() bool {
	return a.apiKey != ""
}

func (a *AnthropicProvider) Ping(ctx context.Context) error {
	// Anthropic doesn't have a simple GET ping; try HEAD on messages endpoint
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodHead, anthropicBaseURL+"/messages", nil)
	if err != nil {
		return err
	}
	for k, v := range a.headers() {
		httpReq.Header.Set(k, v)
	}

	resp, err := a.client.Do(httpReq)
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

func (a *AnthropicProvider) Usage(ctx context.Context) (any, error) {
	return nil, fmt.Errorf("usage API not supported for %s", a.Name())
}

func (a *AnthropicProvider) IsReady() bool   { return a.ready }
func (a *AnthropicProvider) SetReady(r bool) { a.ready = r }
