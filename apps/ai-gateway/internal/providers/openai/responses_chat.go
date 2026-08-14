package openai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"apps/ai-gateway/internal/domain"
)

const (
	hostedWebSearchToolName          = "web_search"
	responsesFieldInclude            = "include"
	responsesIncludeWebSearchSources = "web_search_call.action.sources"
	webSearchResultErrorUnavailable  = "unavailable"
	maxWebSearchSources              = 8
)

// chatViaResponses adapts the internal chat contract to the Responses API while
// retaining the ChatResponse shape expected by the Anthropic transport adapter.
func (p *Provider) chatViaResponses(ctx context.Context, req domain.ChatRequest) (*domain.ChatResponse, error) {
	responsesReq := responsesRequestFromChat(req, false)
	resp, err := p.doResponsesRequest(ctx, responsesReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai responses error %d: %s", resp.StatusCode, body)
	}
	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	return chatResponseFromResponses(raw, req.Model), nil
}

func (p *Provider) chatStreamViaResponses(ctx context.Context, req domain.ChatRequest, w io.Writer) (domain.Usage, error) {
	responsesReq := responsesRequestFromChat(req, true)
	resp, err := p.doResponsesRequest(ctx, responsesReq)
	if err != nil {
		return domain.Usage{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return domain.Usage{}, fmt.Errorf("openai responses stream error %d: %s", resp.StatusCode, b)
	}
	return proxyResponsesAsChatSSE(resp.Body, w)
}

func responsesRequestFromChat(req domain.ChatRequest, stream bool) domain.ResponsesRequest {
	instructions, input := responsesInputFromMessages(req.Messages)
	body := map[string]any{
		"model":  req.Model,
		"stream": stream,
		"input":  input,
	}
	if instructions != "" {
		body["instructions"] = instructions
	}
	if req.PromptCacheKey != "" {
		body["prompt_cache_key"] = req.PromptCacheKey
	}
	if req.PromptCacheOptions != nil {
		body["prompt_cache_options"] = req.PromptCacheOptions
	}
	if req.ReasoningEffort != "" {
		body["reasoning"] = map[string]any{"effort": req.ReasoningEffort}
	}
	if len(req.Tools) > 0 {
		body["tools"] = responsesToolsFromChat(req.Tools)
	}
	if hasWebSearchTool(req.Tools) {
		body[responsesFieldInclude] = []string{responsesIncludeWebSearchSources}
	}
	return domain.ResponsesRequest{Model: req.Model, Stream: stream, Body: body}
}

func responsesInputFromMessages(messages []domain.Message) (string, []any) {
	input := make([]any, 0, len(messages))
	instructions := make([]string, 0)
	for _, message := range messages {
		if message.Role == domain.RoleSystem {
			if message.Content != "" {
				instructions = append(instructions, message.Content)
			}
			continue
		}
		if message.Role == domain.RoleAssistant && len(message.ToolCalls) > 0 {
			for _, call := range message.ToolCalls {
				if call.Function == nil {
					continue
				}
				input = append(input, map[string]any{"type": "function_call", "call_id": call.ID, "name": call.Function.Name, "arguments": call.Function.Arguments})
			}
			if message.Content == "" {
				continue
			}
		}
		if message.Role == domain.RoleTool {
			input = append(input, map[string]any{"type": "function_call_output", "call_id": message.ToolCallID, "output": message.Content})
			continue
		}
		content := any(message.Content)
		if len(message.Parts) > 0 {
			content = message.Parts
		}
		input = append(input, map[string]any{"role": message.Role, "content": content})
	}
	return strings.Join(instructions, "\n"), input
}

func responsesToolsFromChat(tools []domain.Tool) []any {
	result := make([]any, 0, len(tools))
	for _, tool := range tools {
		if tool.Type == domain.ToolTypeWebSearch {
			result = append(result, map[string]any{"type": domain.ToolTypeWebSearch})
			continue
		}
		if tool.Function == nil {
			continue
		}
		result = append(result, map[string]any{"type": "function", "name": tool.Function.Name, "description": tool.Function.Description, "parameters": tool.Function.Parameters})
	}
	return result
}

func chatResponseFromResponses(raw map[string]any, model string) *domain.ChatResponse {
	content, _ := raw["output_text"].(string)
	if content == "" {
		if output, ok := raw["output"].([]any); ok {
			for _, item := range output {
				obj, _ := item.(map[string]any)
				parts, _ := obj["content"].([]any)
				for _, part := range parts {
					p, _ := part.(map[string]any)
					if text, ok := p["text"].(string); ok {
						content += text
					}
				}
			}
		}
	}
	toolCalls := make([]domain.ToolCall, 0)
	if output, ok := raw["output"].([]any); ok {
		for _, item := range output {
			obj, _ := item.(map[string]any)
			if obj["type"] != "function_call" {
				continue
			}
			callID, _ := obj["call_id"].(string)
			name, _ := obj["name"].(string)
			args, _ := obj["arguments"].(string)
			toolCalls = append(toolCalls, domain.ToolCall{ID: callID, Type: domain.ToolTypeFunction, Function: &domain.FunctionCall{Name: name, Arguments: args}})
		}
	}
	usage := domain.UsageFromResponsesValue(raw)
	id, _ := raw["id"].(string)
	created := int64(numberFromAny(raw["created_at"]))
	return &domain.ChatResponse{ID: id, Object: "chat.completion", Created: created, Model: model, Choices: []domain.Choice{{Index: 0, Message: domain.Message{Role: domain.RoleAssistant, Content: content, ToolCalls: toolCalls}, FinishReason: finishReason(toolCalls)}}, Usage: usage}
}

func finishReason(calls []domain.ToolCall) string {
	if len(calls) > 0 {
		return "tool_calls"
	}
	return "stop"
}

func proxyResponsesAsChatSSE(body io.Reader, w io.Writer) (domain.Usage, error) {
	var usage domain.Usage
	webSearchEmitted := make(map[string]bool)
	webSearchResultEmitted := make(map[string]bool)
	pendingWebSearchResults := make(map[string]any)
	webSearchOrder := make([]string, 0)
	emitSearch := func(callID, query string) error {
		if callID == "" || query == "" || webSearchEmitted[callID] {
			return nil
		}
		webSearchEmitted[callID] = true
		webSearchOrder = append(webSearchOrder, callID)
		if err := writeOpenAIWebSearch(w, callID, query); err != nil {
			return err
		}
		if sources, ok := pendingWebSearchResults[callID]; ok {
			if err := writeOpenAIWebSearchResult(w, callID, sources); err != nil {
				return err
			}
			webSearchResultEmitted[callID] = true
			delete(pendingWebSearchResults, callID)
		}
		return nil
	}
	emitResult := func(callID string, sources any) error {
		if callID == "" || webSearchResultEmitted[callID] {
			return nil
		}
		if webSearchEmitted[callID] {
			if err := writeOpenAIWebSearchResult(w, callID, sources); err != nil {
				return err
			}
			webSearchResultEmitted[callID] = true
			return nil
		}
		pendingWebSearchResults[callID] = sources
		return nil
	}
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		var event map[string]any
		if json.Unmarshal([]byte(payload), &event) != nil {
			continue
		}
		typeName, _ := event["type"].(string)
		switch typeName {
		case "response.web_search_call.searching", "response.web_search_call.in_progress":
			callID, _ := event["item_id"].(string)
			action, _ := event["action"].(map[string]any)
			if err := emitSearch(callID, webSearchQuery(action)); err != nil {
				return usage, err
			}
		case "response.output_item.added", "response.output_item.done":
			item, _ := event["item"].(map[string]any)
			itemType, _ := item["type"].(string)
			if itemType != "web_search_call" {
				break
			}
			callID, _ := item["id"].(string)
			action, _ := item["action"].(map[string]any)
			if err := emitSearch(callID, webSearchQuery(action)); err != nil {
				return usage, err
			}
			if sources, ok := webSearchSources(action); ok {
				if err := emitResult(callID, sources); err != nil {
					return usage, err
				}
			}
		case "response.web_search_call.completed":
			callID, _ := event["item_id"].(string)
			action, _ := event["action"].(map[string]any)
			if sources, ok := webSearchSources(action); ok {
				if err := emitResult(callID, sources); err != nil {
					return usage, err
				}
			}
		case "response.output_text.delta":
			delta, _ := event["delta"].(string)
			chunk := map[string]any{"object": "chat.completion.chunk", "choices": []any{map[string]any{"index": 0, "delta": map[string]any{"role": "assistant", "content": delta}}}}
			if err := writeSSEJSON(w, chunk); err != nil {
				return usage, err
			}
		case "response.function_call_arguments.delta":
			delta, _ := event["delta"].(string)
			callID, _ := event["call_id"].(string)
			chunk := map[string]any{
				"object": "chat.completion.chunk",
				"choices": []any{map[string]any{
					"index": 0,
					"delta": map[string]any{
						"tool_calls": []any{map[string]any{
							"index": 0, "id": callID, "type": "function",
							"function": map[string]any{"arguments": delta},
						}},
					},
				}},
			}
			if err := writeSSEJSON(w, chunk); err != nil {
				return usage, err
			}
		case "response.completed":
			if response, ok := event["response"].(map[string]any); ok {
				usage = domain.UsageFromResponsesValue(response)
				if output, ok := response["output"].([]any); ok {
					for _, rawItem := range output {
						item, _ := rawItem.(map[string]any)
						itemType, _ := item["type"].(string)
						if itemType != "web_search_call" {
							continue
						}
						callID, _ := item["id"].(string)
						action, _ := item["action"].(map[string]any)
						if sources, found := webSearchSources(action); found {
							if err := emitResult(callID, sources); err != nil {
								return usage, err
							}
						}
					}
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return usage, err
	}
	for _, callID := range webSearchOrder {
		if webSearchResultEmitted[callID] {
			continue
		}
		if err := writeOpenAIWebSearchError(w, callID, webSearchResultErrorUnavailable); err != nil {
			return usage, err
		}
	}
	if usage.TotalTokens > 0 {
		if err := writeSSEJSON(w, map[string]any{"object": "chat.completion.chunk", "choices": []any{}, "usage": usage}); err != nil {
			return usage, err
		}
	}
	_, _ = io.WriteString(w, "data: [DONE]\n\n")
	if f, ok := w.(interface{ Flush() }); ok {
		f.Flush()
	}
	return usage, nil
}

func webSearchQuery(action map[string]any) string {
	query, _ := action["query"].(string)
	if query != "" {
		return query
	}
	if queries, ok := action["queries"].([]any); ok && len(queries) > 0 {
		query, _ = queries[0].(string)
	}
	return query
}

func writeSSEJSON(w io.Writer, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", b)
	if f, ok := w.(interface{ Flush() }); ok {
		f.Flush()
	}
	return err
}
func numberFromAny(value any) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	default:
		return 0
	}
}
