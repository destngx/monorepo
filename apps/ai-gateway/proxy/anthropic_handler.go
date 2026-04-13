package proxy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"apps/ai-gateway/providers"
	"apps/ai-gateway/types"
)

type AnthropicHandler struct {
	registry *Registry
}

func NewAnthropicHandler(registry *Registry) *AnthropicHandler {
	return &AnthropicHandler{registry: registry}
}

func (h *AnthropicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	providerName := r.Header.Get("X-AI-Provider")
	if providerName == "" {
		providerName = "github"
	}
	provider, err := h.registry.Get(providerName)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if !provider.IsReady() {
		writeError(w, r, http.StatusNotFound, "provider "+providerName+" not ready")
		return
	}

	var anthroReq types.AnthropicRequest
	if err := json.NewDecoder(r.Body).Decode(&anthroReq); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid anthropic request body: "+err.Error())
		return
	}

	req := convertFromAnthropicRequest(anthroReq)

	if anthroReq.Stream {
		h.handleStream(w, r, provider, req)
	} else {
		h.handleSync(w, r, provider, req)
	}
}

func (h *AnthropicHandler) handleSync(w http.ResponseWriter, r *http.Request, p providers.Provider, req types.ChatRequest) {
	resp, err := p.Chat(r.Context(), req)
	if err != nil {
		writeError(w, r, http.StatusBadGateway, err.Error())
		return
	}

	anthroResp := convertToAnthropicResponse(resp)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(anthroResp)
}

func (h *AnthropicHandler) handleStream(w http.ResponseWriter, r *http.Request, p providers.Provider, req types.ChatRequest) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		_, err := p.ChatStream(r.Context(), req, pw)
		if err != nil {
			log.Printf("Anthropic proxy stream error: %v", err)
		}
	}()

	convertToAnthropicStream(pr, w)
}

func convertFromAnthropicRequest(ar types.AnthropicRequest) types.ChatRequest {
	req := types.ChatRequest{
		Model:       ar.Model,
		Stream:      ar.Stream,
		Temperature: ar.Temperature,
		TopP:        ar.TopP,
		MaxTokens:   &ar.MaxTokens,
	}

	if ar.System != "" {
		req.Messages = append(req.Messages, types.Message{
			Role:    "system",
			Content: ar.System,
		})
	}

	for _, m := range ar.Messages {
		msg := types.Message{Role: m.Role}
		switch v := m.Content.(type) {
		case string:
			msg.Content = v
		case []any:
			for _, block := range v {
				if b, ok := block.(map[string]any); ok {
					blockType, _ := b["type"].(string)
					switch blockType {
					case "text":
						if text, ok := b["text"].(string); ok {
							if msg.Content != "" {
								msg.Content += "\n"
							}
							msg.Content += text
						}
					case "tool_use":
						id, _ := b["id"].(string)
						name, _ := b["name"].(string)
						input, _ := b["input"]
						inputJSON, _ := json.Marshal(input)
						msg.ToolCalls = append(msg.ToolCalls, types.ToolCall{
							ID:   id,
							Type: "function",
							Function: types.FunctionCall{
								Name:      name,
								Arguments: string(inputJSON),
							},
						})
					case "tool_result":
						// If it's a tool_result, it should actually be its own message in OpenAI
						// But Anthropic puts it in a content block.
						// We'll create a separate message if we encounter this, but this is tricky
						// because we are inside a loop over messages.
						// For now, let's extract tool results.
						toolID, _ := b["tool_use_id"].(string)
						content, _ := b["content"].(string)
						// We'll handle this by adding a special message after this one or before.
						// Implementation detail: OpenAI expects role="tool" for results.
						req.Messages = append(req.Messages, types.Message{
							Role:       "tool",
							ToolCallID: toolID,
							Content:    content,
						})
						continue // skip adding this to the current message
					}
				}
			}
		}
		if msg.Role != "" || msg.Content != "" || len(msg.ToolCalls) > 0 {
			req.Messages = append(req.Messages, msg)
		}
	}

	for _, t := range ar.Tools {
		req.Tools = append(req.Tools, types.Tool{
			Type: "function",
			Function: types.FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		})
	}

	if ar.StopSeqs != nil {
		req.Stop = ar.StopSeqs
	}

	return req
}

func convertToAnthropicResponse(resp *types.ChatResponse) types.AnthropicResponse {
	ar := types.AnthropicResponse{
		ID:    resp.ID,
		Type:  "message",
		Role:  "assistant",
		Model: resp.Model,
		Usage: types.AnthropicUsage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		},
	}

	if len(resp.Choices) > 0 {
		msg := resp.Choices[0].Message
		if msg.Content != "" {
			ar.Content = append(ar.Content, types.AnthropicContent{
				Type: "text",
				Text: msg.Content,
			})
		}
		for _, tc := range msg.ToolCalls {
			var input any
			json.Unmarshal([]byte(tc.Function.Arguments), &input)
			ar.Content = append(ar.Content, types.AnthropicContent{
				Type:  "tool_use",
				ID:    tc.ID,
				Name:  tc.Function.Name,
				Input: input,
			})
		}

		ar.StopReason = "end_turn"
		if resp.Choices[0].FinishReason == "length" {
			ar.StopReason = "max_tokens"
		} else if resp.Choices[0].FinishReason == "tool_calls" {
			ar.StopReason = "tool_use"
		}
	}

	return ar
}

func convertToAnthropicStream(r io.Reader, w io.Writer) {
	scanner := bufio.NewScanner(r)
	flusher, _ := w.(http.Flusher)

	// Send message_start
	// We don't have all info yet, but we can send a partial one
	// Or we wait for the first chunk to get the model name etc.

	first := true

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			sendAnthroEvent(w, "message_stop", map[string]any{})
			break
		}

		var chunk map[string]any
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if first {
			model, _ := chunk["model"].(string)
			id, _ := chunk["id"].(string)
			sendAnthroEvent(w, "message_start", map[string]any{
				"message": map[string]any{
					"id":    id,
					"type":  "message",
					"role":  "assistant",
					"model": model,
					"usage": map[string]int{"input_tokens": 0}, // fallback
				},
			})
			first = false
		}

		choices, ok := chunk["choices"].([]any)
		if !ok || len(choices) == 0 {
			continue
		}
		choice := choices[0].(map[string]any)
		delta, _ := choice["delta"].(map[string]any)

		if content, ok := delta["content"].(string); ok && content != "" {
			sendAnthroEvent(w, "content_block_delta", map[string]any{
				"index": 0,
				"delta": map[string]any{
					"type": "text_delta",
					"text": content,
				},
			})
		}

		if toolCalls, ok := delta["tool_calls"].([]any); ok {
			for _, tc := range toolCalls {
				t := tc.(map[string]any)
				index, _ := t["index"].(float64)

				// Handle both start and delta
				if id, ok := t["id"].(string); ok {
					name, _ := t["function"].(map[string]any)["name"].(string)
					sendAnthroEvent(w, "content_block_start", map[string]any{
						"index": int(index),
						"content_block": map[string]any{
							"type": "tool_use",
							"id":   id,
							"name": name,
						},
					})
				} else if function, ok := t["function"].(map[string]any); ok {
					if args, ok := function["arguments"].(string); ok {
						sendAnthroEvent(w, "content_block_delta", map[string]any{
							"index": int(index),
							"delta": map[string]any{
								"type":         "input_json_delta",
								"partial_json": args,
							},
						})
					}
				}
			}
		}

		if finish, ok := choice["finish_reason"].(string); ok && finish != "" {
			stopReason := "end_turn"
			if finish == "length" {
				stopReason = "max_tokens"
			} else if finish == "tool_calls" {
				stopReason = "tool_use"
			}
			sendAnthroEvent(w, "message_delta", map[string]any{
				"delta": map[string]any{
					"stop_reason": stopReason,
				},
			})
		}

		if flusher != nil {
			flusher.Flush()
		}
	}
}

func sendAnthroEvent(w io.Writer, eventType string, data any) {
	if m, ok := data.(map[string]any); ok {
		m["type"] = eventType
	}
	b, _ := json.Marshal(data)
	payload := string(b)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, payload)
}
