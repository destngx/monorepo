package openai

import (
	"apps/ai-gateway/internal/domain"
	"apps/ai-gateway/internal/providers/shared"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

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

	content, toolCalls, usage, responseID, created, model, err := parseCodexStream(resp.Body, nil)
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
					Role:      domain.RoleAssistant,
					Content:   content,
					ToolCalls: toolCalls,
				},
				FinishReason: codexFinishReason(toolCalls),
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

	_, _, usage, _, _, _, err := parseCodexStream(resp.Body, w)
	return usage, err
}

func (p *Provider) doCodexRequest(ctx context.Context, req domain.ChatRequest) (*http.Response, error) {
	slog.Debug("OpenAI upstream request", "method", http.MethodPost, "path", pathCodexResponses)
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
	instructionText, input := responsesInputFromMessages(req.Messages)
	input = codexResponsesInput(input)
	instructions := []string{}
	if instructionText != "" {
		instructions = append(instructions, instructionText)
	}

	if len(instructions) == 0 {
		instructions = append(instructions, "You are a helpful assistant.")
	}
	if len(input) == 0 {
		input = append(input, map[string]any{
			"role":    domain.RoleUser,
			"content": []map[string]string{{"type": "input_text", "text": ""}},
		})
	}

	maxOutputTokens := req.MaxCompletionTokens
	if maxOutputTokens == nil {
		maxOutputTokens = req.MaxTokens
	}

	out := codexResponseRequest{
		Model:        req.Model,
		Instructions: strings.Join(instructions, "\n\n"),
		Input:        input,
		Stream:       true,
		Store:        false,
	}
	if len(req.Tools) > 0 {
		out.Tools = responsesToolsFromChat(req.Tools)
	}
	if req.ToolChoice != nil {
		out.ToolChoice = req.ToolChoice
	}
	return out
}

func codexResponsesInput(input []any) []any {
	for _, raw := range input {
		message, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		parts, ok := message["content"].([]domain.ContentPart)
		if !ok {
			continue
		}
		converted := make([]map[string]any, 0, len(parts))
		contentType := "input_text"
		if role, _ := message["role"].(string); role == domain.RoleAssistant {
			contentType = "output_text"
		}
		for _, part := range parts {
			typeName := part.Type
			if typeName == "text" {
				typeName = contentType
			}
			converted = append(converted, map[string]any{"type": typeName, "text": part.Text})
		}
		message["content"] = converted
	}
	return input
}

func codexContentFromMessage(msg domain.Message) codexInputContent {
	content := codexInputContent{Type: "input_text", Text: msg.Content}
	if len(msg.Parts) > 0 {
		content.Text = msg.Parts[0].Text
	}
	return content
}

func parseCodexStream(body io.Reader, w io.Writer) (string, []domain.ToolCall, domain.Usage, string, int64, string, error) {
	var content strings.Builder
	toolCalls := make([]domain.ToolCall, 0)
	callIndexes := make(map[string]int)
	callNames := make(map[string]string)
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
					return content.String(), toolCalls, usage, responseID, created, model, err
				}
			}
		case "response.output_item.added", "response.output_item.done":
			item := event.Item
			if item == nil || item.Type != "function_call" {
				continue
			}
			if item.CallID == "" {
				item.CallID = item.ID
			}
			if item.Name == "" {
				item.Name = event.Name
			}
			if index, ok := callIndexes[item.CallID]; ok {
				toolCalls[index].ID = item.CallID
				if toolCalls[index].Function == nil {
					toolCalls[index].Function = &domain.FunctionCall{}
				}
				toolCalls[index].Function.Name = item.Name
				if item.Arguments != "" {
					toolCalls[index].Function.Arguments = item.Arguments
				}
			} else {
				callIndexes[item.CallID] = len(toolCalls)
				if item.ID != "" {
					callIndexes[item.ID] = len(toolCalls)
				}
				callNames[item.CallID] = item.Name
				toolCalls = append(toolCalls, domain.ToolCall{ID: item.CallID, Type: domain.ToolTypeFunction, Function: &domain.FunctionCall{Name: item.Name, Arguments: item.Arguments}})
			}
		case "response.function_call_arguments.delta", "response.function_call_arguments.done":
			callID := event.CallID
			if callID == "" {
				callID = event.ItemID
			}
			index, ok := callIndexes[callID]
			if !ok {
				index = len(toolCalls)
				callIndexes[callID] = index
				toolCalls = append(toolCalls, domain.ToolCall{ID: callID, Type: domain.ToolTypeFunction, Function: &domain.FunctionCall{Name: callNames[callID]}})
			}
			call := &toolCalls[index]
			if call.Function == nil {
				call.Function = &domain.FunctionCall{}
			}
			if event.Type == "response.function_call_arguments.done" && event.Arguments != "" {
				call.Function.Arguments = event.Arguments
			} else {
				call.Function.Arguments += event.Delta
			}
			if call.Function.Name == "" {
				call.Function.Name = event.Name
			}
			if w != nil {
				arguments := event.Delta
				if event.Type == "response.function_call_arguments.done" {
					arguments = event.Arguments
				}
				if err := writeOpenAIToolDelta(w, responseID, created, model, index, callID, call.Function.Name, arguments); err != nil {
					return content.String(), toolCalls, usage, responseID, created, model, err
				}
			}
		case "response.failed":
			return content.String(), toolCalls, usage, responseID, created, model, fmt.Errorf("openai codex response failed: %s", event.Error)
		}
	}
	if err := scanner.Err(); err != nil {
		return content.String(), toolCalls, usage, responseID, created, model, err
	}

	if usage.TotalTokens == 0 {
		usage.CompletionTokens = shared.EstimateTokens(content.String())
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}
	if w != nil {
		if len(toolCalls) > 0 {
			if err := writeOpenAIToolFinish(w, responseID, created, model); err != nil {
				return content.String(), toolCalls, usage, responseID, created, model, err
			}
		}
		shared.InjectUsageChunk(w, usage)
	}

	return content.String(), toolCalls, usage, responseID, created, model, nil
}

func codexFinishReason(calls []domain.ToolCall) string {
	if len(calls) > 0 {
		return "tool_calls"
	}
	return "stop"
}

func writeOpenAIToolDelta(w io.Writer, id string, created int64, model string, index int, callID, name, arguments string) error {
	function := map[string]string{"arguments": arguments}
	if name != "" {
		function["name"] = name
	}
	chunk := map[string]any{"id": id, "object": "chat.completion.chunk", "created": created, "model": model, "choices": []any{map[string]any{"index": 0, "delta": map[string]any{"tool_calls": []any{map[string]any{"index": index, "id": callID, "type": "function", "function": function}}}, "finish_reason": nil}}}
	return writeSSEMap(w, chunk)
}

func writeOpenAIToolFinish(w io.Writer, id string, created int64, model string) error {
	chunk := map[string]any{
		"id": id, "object": "chat.completion.chunk", "created": created, "model": model,
		"choices": []any{map[string]any{
			"index": 0, "delta": map[string]any{}, "finish_reason": "tool_calls",
		}},
	}
	return writeSSEMap(w, chunk)
}

func writeSSEMap(w io.Writer, chunk map[string]any) error {
	b, err := json.Marshal(chunk)
	if err != nil {
		return err
	}
	if _, err = io.WriteString(w, "data: "+string(b)+"\n\n"); err != nil {
		return err
	}
	if f, ok := w.(interface{ Flush() }); ok {
		f.Flush()
	}
	return nil
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
