package github_copilot

import (
	"apps/ai-gateway/internal/domain"
	"apps/ai-gateway/internal/providers/shared"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func (p *Provider) chatResponsesStream(ctx context.Context, req domain.ChatRequest, w io.Writer) (domain.Usage, error) {
	start := time.Now()
	p.vlogf(1, "[github-copilot] responses stream start model=%q", req.Model)

	httpReq, err := p.newResponsesRequest(ctx, req)
	if err != nil {
		return domain.Usage{}, err
	}

	resp, err := p.streamingClient().Do(httpReq)
	if err != nil {
		return domain.Usage{}, err
	}
	defer resp.Body.Close()
	p.vlogf(1, "[github-copilot] upstream responses call took=%s status=%d", time.Since(start), resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return domain.Usage{}, fmt.Errorf("github copilot responses error %d: %s", resp.StatusCode, b)
	}

	return transformResponsesStream(resp.Body, w, req.Model)
}

func transformResponsesStream(body io.Reader, w io.Writer, fallbackModel string) (domain.Usage, error) {
	var usage domain.Usage
	var responseID string
	var created int64
	model := fallbackModel
	var completionTokens int
	toolCallIndexes := make(map[string]int)

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024*64), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, sseDataPrefix) {
			continue
		}
		payload := strings.TrimPrefix(line, sseDataPrefix)
		if payload == sseDone {
			break
		}

		var rawEvent map[string]json.RawMessage
		if err := json.Unmarshal([]byte(payload), &rawEvent); err != nil {
			continue
		}

		var eventType string
		if t, ok := rawEvent["type"]; ok {
			json.Unmarshal(t, &eventType)
		}

		var event responsesStreamEvent
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			// If it fails, maybe Delta is an object. Try to extract it anyway.
			event.Type = eventType
		}

		// Debug logging to a local file
		shared.LogToFile("copilot-sse.log", fmt.Sprintf("Event: %s, Payload: %s", eventType, payload))

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
		case "response.output_item.added", "response.output_item.done":
			if event.Item == nil || event.Item.Type != "function_call" {
				break
			}
			callID := event.Item.CallID
			if callID == "" {
				callID = event.Item.ID
			}
			if callID == "" {
				break
			}
			if _, ok := toolCallIndexes[callID]; !ok {
				toolCallIndexes[callID] = len(toolCallIndexes)
			}
			if event.Item.Name == "" {
				break
			}
			if err := writeToolCallDelta(w, responseID, created, model, toolCallIndexes[callID], callID, event.Item.Name, event.Item.Arguments); err != nil {
				return usage, err
			}
		case "response.function_call_arguments.delta":
			callID := event.CallID
			if callID == "" {
				callID = event.ItemID
			}
			if _, ok := toolCallIndexes[callID]; !ok {
				toolCallIndexes[callID] = len(toolCallIndexes)
			}
			if event.Name == "" {
				break
			}
			if err := writeToolCallDelta(w, responseID, created, model, toolCallIndexes[callID], callID, event.Name, event.Delta); err != nil {
				return usage, err
			}
		case "response.output_text.delta", "response.text.delta", "response.content_part.delta":
			completionTokens += shared.EstimateTokens(event.Delta)
			if responseID == "" {
				responseID = "chatcmpl-copilot-responses"
			}
			if created == 0 {
				created = time.Now().Unix()
			}
			if err := writeChatCompletionDelta(w, responseID, created, model, event.Delta, ""); err != nil {
				return usage, err
			}
		case "response.reasoning_text.delta":
			completionTokens += shared.EstimateTokens(event.Delta)
			if responseID == "" {
				responseID = "chatcmpl-copilot-responses"
			}
			if created == 0 {
				created = time.Now().Unix()
			}
			if err := writeChatCompletionDelta(w, responseID, created, model, "", event.Delta); err != nil {
				return usage, err
			}
		case "response.failed":
			return usage, fmt.Errorf("github copilot responses failed: %s", event.Error)
		default:
			// Capture any other delta-like events just in case
			if event.Delta != "" && strings.Contains(event.Type, "delta") {
				completionTokens += shared.EstimateTokens(event.Delta)
				writeChatCompletionDelta(w, responseID, created, model, event.Delta, "")
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return usage, err
	}

	if usage.TotalTokens == 0 {
		usage.CompletionTokens = completionTokens
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}
	shared.InjectUsageChunk(w, usage)
	return usage, nil
}

func writeToolCallDelta(w io.Writer, id string, created int64, model string, index int, callID, name, arguments string) error {
	if id == "" {
		id = "chatcmpl-copilot-responses"
	}
	if created == 0 {
		created = time.Now().Unix()
	}
	chunk := map[string]interface{}{"id": id, "object": "chat.completion.chunk", "created": created, "model": model, "choices": []map[string]interface{}{{"index": 0, "delta": map[string]interface{}{"tool_calls": []map[string]interface{}{{"index": index, "id": callID, "type": "function", "function": map[string]string{"name": name, "arguments": arguments}}}}, "finish_reason": nil}}}
	b, err := json.Marshal(chunk)
	if err != nil {
		return err
	}
	if _, err = io.WriteString(w, sseDataPrefix+string(b)+"\n\n"); err != nil {
		return err
	}
	if f, ok := w.(interface{ Flush() }); ok {
		f.Flush()
	}
	return nil
}

func writeChatCompletionDelta(w io.Writer, id string, created int64, model string, content string, reasoningContent string) error {
	delta := map[string]string{}
	if content != "" {
		delta["content"] = content
	}
	if reasoningContent != "" {
		delta["reasoning_content"] = reasoningContent
	}

	chunk := map[string]interface{}{
		"id":      id,
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   model,
		"choices": []map[string]interface{}{
			{
				"index":         0,
				"delta":         delta,
				"finish_reason": nil,
			},
		},
	}
	b, err := json.Marshal(chunk)
	if err != nil {
		return err
	}
	if _, err := io.WriteString(w, sseDataPrefix+string(b)+"\n\n"); err != nil {
		return err
	}
	if f, ok := w.(interface{ Flush() }); ok {
		f.Flush()
	}
	return nil
}
