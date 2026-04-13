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

	// 1. Request Decoding: Standardize the incoming Anthropic-style payload.
	// We do this first so we can use the 'model' field for smart routing.
	var anthroReq types.AnthropicRequest
	if err := json.NewDecoder(r.Body).Decode(&anthroReq); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid anthropic request body: "+err.Error())
		return
	}

	rid, _ := r.Context().Value(requestIDKey).(string)

	if h.registry.Config.Verbose >= 1 {
		reqBytes, _ := json.Marshal(anthroReq)
		log.Printf("[ID:%s] [VERBOSE 1] Received Anthropic Request: %s", rid, string(reqBytes))
	}
	if h.registry.Config.Verbose >= 2 {
		log.Printf("[ID:%s] [VERBOSE 2] Finished decoding Anthropic request", rid)
	}

	// 2. Smart Routing: Determine which backend provider and model to use.
	provider, targetModel, err := h.registry.ResolveRoute(r, anthroReq.Model)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not ready") {
			status = http.StatusNotFound
		}
		writeError(w, r, status, "routing failed: "+err.Error())
		return
	}

	// Improvement 5: Rich Mapping Logs
	r = SetLogMapping(r, fmt.Sprintf("%s -> %s", anthroReq.Model, targetModel))

	// 3. Translation: Convert Anthropic format to internal OpenAI format.
	req := convertFromAnthropicRequest(anthroReq, provider.Name())
	req.Model = targetModel

	if anthroReq.Stream {
		h.handleStream(w, r, provider, req)
	} else {
		h.handleSync(w, r, provider, req)
	}
}

func (h *AnthropicHandler) handleSync(w http.ResponseWriter, r *http.Request, p providers.Provider, req types.ChatRequest) {
	rid, _ := r.Context().Value(requestIDKey).(string)
	if h.registry.Config.Verbose >= 2 {
		log.Printf("[ID:%s] [VERBOSE 2] Entering Anthropic handleSync", rid)
	}

	resp, err := p.Chat(r.Context(), req)
	if err != nil {
		if h.registry.Config.Verbose >= 1 {
			log.Printf("[ID:%s] [VERBOSE 1] Provider returned error: %v", rid, err)
		}
		writeError(w, r, http.StatusBadGateway, err.Error())
		return
	}

	anthroResp := convertToAnthropicResponse(resp)

	if h.registry.Config.Verbose >= 1 {
		respBytes, _ := json.Marshal(anthroResp)
		log.Printf("[ID:%s] [VERBOSE 1] Provider Response (Anthropic format): %s", rid, string(respBytes))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(anthroResp)
}

func (h *AnthropicHandler) handleStream(w http.ResponseWriter, r *http.Request, p providers.Provider, req types.ChatRequest) {
	rid, _ := r.Context().Value(requestIDKey).(string)
	if h.registry.Config.Verbose >= 2 {
		log.Printf("[ID:%s] [VERBOSE 2] Entering Anthropic handleStream", rid)
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	if _, ok := w.(http.Flusher); !ok {
		writeError(w, r, http.StatusInternalServerError, "streaming not supported by response writer")
		return
	}

	pr, pw := io.Pipe()
	errCh := make(chan error, 1)
	go func() {
		defer pw.Close()
		_, err := p.ChatStream(r.Context(), req, pw)
		errCh <- err
	}()

	var writer io.Writer = w
	if h.registry.Config.Verbose >= 1 {
		writer = &StreamLogWriter{w: w, rid: rid}
	}

	eventCount, convertErr := convertToAnthropicStream(pr, writer)
	if convertErr != nil {
		log.Printf("[ID:%s] STREAM CONVERT ERROR: %v", rid, convertErr)
	}

	if err := <-errCh; err != nil {
		log.Printf("[ID:%s] STREAM ERROR: %v", rid, err)
		if eventCount == 0 {
			sendAnthroEvent(writer, "error", map[string]any{
				"error": map[string]any{
					"type":    "api_error",
					"message": err.Error(),
				},
			})
			if flusher, ok := writer.(interface{ Flush() }); ok {
				flusher.Flush()
			}
		}
	}
}

func convertFromAnthropicRequest(ar types.AnthropicRequest, providerName string) types.ChatRequest {
	isGemini := providerName == "google" || strings.Contains(providerName, "gemini")

	req := types.ChatRequest{
		Model:       ar.Model,
		Stream:      ar.Stream,
		Temperature: ar.Temperature,
		TopP:        ar.TopP,
	}
	if ar.MaxTokens > 0 {
		req.MaxTokens = &ar.MaxTokens
	}

	if ar.ToolChoice != nil {
		if m, ok := ar.ToolChoice.(map[string]any); ok {
			if t, ok := m["type"].(string); ok {
				if t == "auto" {
					req.ToolChoice = t
				} else if t == "any" {
					req.ToolChoice = "required"
				} else if t == "tool" {
					name, _ := m["name"].(string)
					req.ToolChoice = map[string]any{
						"type": "function",
						"function": map[string]string{
							"name": name,
						},
					}
				}
			}
		} else if s, ok := ar.ToolChoice.(string); ok {
			if s == "any" {
				req.ToolChoice = "required"
			} else {
				req.ToolChoice = s
			}
		}
	}

	if ar.System != nil {
		systemText := ""
		switch v := ar.System.(type) {
		case string:
			systemText = v
		case []any:
			for _, block := range v {
				if b, ok := block.(map[string]any); ok {
					if t, ok := b["type"].(string); ok && t == "text" {
						if txt, ok := b["text"].(string); ok {
							if systemText != "" {
								systemText += "\n"
							}
							systemText += txt
						}
					}
				}
			}
		}

		if systemText != "" {
			req.Messages = append(req.Messages, types.Message{
				Role:    "system",
				Content: systemText,
			})
		}
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
						toolID, _ := b["tool_use_id"].(string)
						contentStr := ""

						// Improvement 3: Robust tool result content handling
						switch c := b["content"].(type) {
						case string:
							contentStr = c
						case []any:
							for _, innerBlock := range c {
								if ib, ok := innerBlock.(map[string]any); ok {
									if it, ok := ib["type"].(string); ok && it == "text" {
										if itxt, ok := ib["text"].(string); ok {
											contentStr += itxt
										}
									}
								}
							}
						}

						if contentStr == "" {
							contentStr = "..." // Ensure no empty content
						}

						req.Messages = append(req.Messages, types.Message{
							Role:       "tool",
							ToolCallID: toolID,
							Content:    contentStr,
						})
						continue
					}
				}
			}
		}

		// Improvement 3: Ensure content is never empty for OpenAI/Copilot
		if msg.Content == "" && len(msg.ToolCalls) == 0 && msg.Role != "tool" {
			msg.Content = "..."
		}

		if msg.Role != "" || msg.Content != "" || len(msg.ToolCalls) > 0 {
			req.Messages = append(req.Messages, msg)
		}
	}

	for _, t := range ar.Tools {
		params := t.InputSchema
		if isGemini {
			params = CleanJSONSchema(params)
		}

		req.Tools = append(req.Tools, types.Tool{
			Type: "function",
			Function: types.FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
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
		ID:      resp.ID,
		Type:    "message",
		Role:    "assistant",
		Model:   resp.Model,
		Content: []types.AnthropicContent{},
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

func convertToAnthropicStream(r io.Reader, w io.Writer) (int, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*64), 1024*64)
	flusher, _ := w.(http.Flusher)
	emitted := 0

	var (
		first            = true
		textBlockStarted = false
		blockIndex       = -1
		activeToolIndex  = -1
	)

	writeEvent := func(eventType string, data any) {
		sendAnthroEvent(w, eventType, data)
		emitted++
	}

	// Helper to ensure text block is open/closed
	ensureTextStarted := func() {
		if !textBlockStarted {
			blockIndex++
			writeEvent("content_block_start", map[string]any{
				"index":         blockIndex,
				"content_block": map[string]any{"type": "text", "text": ""},
			})
			textBlockStarted = true
		}
	}

	ensureTextStopped := func() {
		if textBlockStarted {
			writeEvent("content_block_stop", map[string]any{"index": blockIndex})
			textBlockStarted = false
		}
	}

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			ensureTextStopped()
			writeEvent("message_stop", map[string]any{})
			break
		}

		var chunk map[string]any
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if first {
			model, _ := chunk["model"].(string)
			id, _ := chunk["id"].(string)
			writeEvent("message_start", map[string]any{
				"message": map[string]any{
					"id":            id,
					"type":          "message",
					"role":          "assistant",
					"model":         model,
					"content":       []any{},
					"stop_reason":   nil,
					"stop_sequence": nil,
					"usage":         map[string]int{"input_tokens": 0},
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

		// 1. Handle Text Content
		if content, ok := delta["content"].(string); ok && content != "" {
			ensureTextStarted()
			writeEvent("content_block_delta", map[string]any{
				"index": blockIndex,
				"delta": map[string]any{
					"type": "text_delta",
					"text": content,
				},
			})
		}

		// 2. Handle Tool Calls
		if toolCalls, ok := delta["tool_calls"].([]any); ok {
			ensureTextStopped() // text must stop before tool use begins

			for _, tc := range toolCalls {
				t := tc.(map[string]any)
				// New tool call?
				if id, ok := t["id"].(string); ok {
					name, _ := t["function"].(map[string]any)["name"].(string)

					if activeToolIndex != -1 {
						writeEvent("content_block_stop", map[string]any{"index": activeToolIndex})
					}

					blockIndex++
					activeToolIndex = blockIndex

					writeEvent("content_block_start", map[string]any{
						"index": activeToolIndex,
						"content_block": map[string]any{
							"type":  "tool_use",
							"id":    id,
							"name":  name,
							"input": map[string]any{},
						},
					})
				} else if function, ok := t["function"].(map[string]any); ok {
					if args, ok := function["arguments"].(string); ok {
						writeEvent("content_block_delta", map[string]any{
							"index": activeToolIndex,
							"delta": map[string]any{
								"type":         "input_json_delta",
								"partial_json": args,
							},
						})
					}
				}
			}
		}

		// 3. Handle Finish Reason
		if finish, ok := choice["finish_reason"].(string); ok && finish != "" {
			ensureTextStopped()

			if activeToolIndex != -1 {
				writeEvent("content_block_stop", map[string]any{"index": activeToolIndex})
				activeToolIndex = -1
			}

			stopReason := "end_turn"
			if finish == "length" {
				stopReason = "max_tokens"
			} else if finish == "tool_calls" {
				stopReason = "tool_use"
			}

			writeEvent("message_delta", map[string]any{
				"delta": map[string]any{
					"stop_reason": stopReason,
				},
				"usage": map[string]any{"output_tokens": 0},
			})
		}

		if flusher != nil {
			flusher.Flush()
		}
	}

	return emitted, scanner.Err()
}

func sendAnthroEvent(w io.Writer, eventType string, data any) {
	if m, ok := data.(map[string]any); ok {
		m["type"] = eventType
	}
	b, _ := json.Marshal(data)
	payload := string(b)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, payload)
}
