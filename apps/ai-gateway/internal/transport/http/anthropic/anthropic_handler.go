package anthropic

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"apps/ai-gateway/internal/domain"
	"apps/ai-gateway/internal/providers/anthropic"
	"apps/ai-gateway/internal/providers/shared"
	"apps/ai-gateway/internal/service"
	"apps/ai-gateway/internal/transport/http/common"
)

type AnthropicHandler struct {
	registry         *service.Registry
	routeInterceptor AnthropicRouteInterceptor
}

func NewHandler(registry *service.Registry) *AnthropicHandler { return NewAnthropicHandler(registry) }

func NewAnthropicHandler(registry *service.Registry) *AnthropicHandler {
	return &AnthropicHandler{
		registry:         registry,
		routeInterceptor: newAnthropicRouteInterceptor(registry),
	}
}

func (h *AnthropicHandler) SetRouteInterceptor(interceptor AnthropicRouteInterceptor) {
	h.routeInterceptor = interceptor
}

// @Summary Anthropic Messages
// @Description Anthropic-compatible messages endpoint.
// @Tags completions
// @Accept json
// @Produce json
// @Param body body anthropic.Request true "Anthropic Request"
// @Success 200 {object} anthropic.Response
// @Router /v1/messages [post]
func (h *AnthropicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, common.ErrMsgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	var anthroReq anthropic.Request
	if err := json.NewDecoder(r.Body).Decode(&anthroReq); err != nil {
		common.WriteError(w, r, http.StatusBadRequest, errMsgInvalidAnthroBody+err.Error())
		return
	}

	rid, _ := r.Context().Value(domain.RequestIDKey).(string)

	if h.registry.Config.Verbose >= 1 {
		body, _ := json.MarshalIndent(anthroReq, "", "  ")
		slog.Info("FULL ANTHROPIC REQUEST", "rid", rid, "body", string(body))
	}

	provider, targetModel, err := h.registry.ResolveRoute(r, anthroReq.Model)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not ready") {
			status = http.StatusNotFound
		}
		common.WriteError(w, r, status, common.ErrMsgRoutingFailed+err.Error())
		return
	}

	route := AnthropicRoute{
		Provider:        provider,
		Model:           targetModel,
		ReasoningEffort: anthroReq.ReasoningEffort,
	}
	if anthroReq.OutputConfig != nil && anthroReq.OutputConfig.Effort != "" {
		route.ReasoningEffort = anthroReq.OutputConfig.Effort
	}
	// Claude Code omits output_config.effort for models without effort support; keep the gateway default low.
	if route.ReasoningEffort == "" {
		route.ReasoningEffort = domain.ReasoningEffortLow
	}
	if h.routeInterceptor != nil {
		route, err = h.routeInterceptor(r, anthroReq, route)
		if err != nil {
			common.WriteError(w, r, http.StatusBadRequest, common.ErrMsgRoutingFailed+err.Error())
			return
		}
	}

	r = common.SetLogMapping(r, fmt.Sprintf("%s -> %s", anthroReq.Model, route.Model))
	r = common.SetLogProvider(r, route.Provider.Name())
	r = common.SetLogModel(r, anthroReq.Model)
	r = common.SetLogReasoningEffort(r, route.ReasoningEffort)

	if hasUnsupported, _ := detectUnsupportedNativeTools(anthroReq, route.Provider.Name()); hasUnsupported {
		if anthroReq.Stream {
			w.Header().Set(common.HeaderContentType, common.ContentTypeEventStream)
			w.Header().Set(common.HeaderCacheControl, common.ValueNoCache)
			w.Header().Set(common.HeaderConnection, common.ValueKeepAlive)
			w.Header().Set(common.HeaderXAccelBuffering, common.ValueNo)
			w.WriteHeader(http.StatusOK)

			errorResponse := map[string]any{
				"type": "error",
				"error": map[string]any{
					"type":    "unsupported_native_tool",
					"message": "Gateway blocked Anthropic-native tool execution for a non-Anthropic target model. Please use a provider MCP tool such as Tavily or Exa or Brave or Perplexity for web search instead.",
				},
			}
			sendAnthroEvent(w, eventError, errorResponse)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		} else {
			w.Header().Set(common.HeaderContentType, common.ContentTypeJSON)
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"type": "error",
				"error": map[string]any{
					"type":    "unsupported_native_tool",
					"message": "Gateway blocked Anthropic-native tool execution for a non-Anthropic target model. Please use a provider MCP tool such as Tavily or Exa or Brave or Perplexity for web search instead.",
				},
			})
		}
		return
	}

	req := convertFromAnthropicRequest(anthroReq, route.Provider.Name())
	req.Model = route.Model
	req.ReasoningEffort = route.ReasoningEffort
	normalizeRequestForRoute(&req, anthroReq, route)

	if anthroReq.Stream {
		h.handleStream(w, r, route.Provider, req, anthroReq.Model)
	} else {
		h.handleSync(w, r, route.Provider, req, anthroReq.Model, anthroReq.Stream)
	}
}

/* func (h *AnthropicHandler) handleStream(w http.ResponseWriter, r *http.Request, p shared.Provider, req domain.ChatRequest, inputModel string) {
	rid, _ := r.Context().Value(domain.RequestIDKey).(string)
	if h.registry.Config.Verbose >= 2 {
		slog.Debug("Entering Anthropic handleStream", "rid", rid)
	}

	w.Header().Set(common.HeaderContentType, common.ContentTypeEventStream)
	w.Header().Set(common.HeaderCacheControl, common.ValueNoCache)
	w.Header().Set(common.HeaderConnection, common.ValueKeepAlive)
	w.Header().Set(common.HeaderXAccelBuffering, common.ValueNo)

	if _, ok := w.(http.Flusher); !ok {
		common.WriteError(w, r, http.StatusInternalServerError, common.ErrMsgStreamNotSupp)
		return
	}

	pr, pw := io.Pipe()
	errCh := make(chan error, 1)
	go func() {
		defer pw.Close()
		providerWriter := io.Writer(pw)
		if h.registry.Config.Verbose >= 2 {
			providerWriter = &common.RawStreamLogWriter{W: pw, RID: rid}
		}
		usage, err := p.ChatStream(r.Context(), req, providerWriter)
		errCh <- err
		common.SetMetrics(r, p.Name(), req.Model, inputModel, usage, req.Stream, err)
	}()

	var writer io.Writer = w
	if h.registry.Config.Verbose >= 1 {
		writer = &common.StreamLogWriter{w: w, rid: rid}
	}

	eventCount, convertErr := convertToAnthropicStream(pr, writer)
	if convertErr != nil {
		slog.Error("Stream convert error", "rid", rid, "error", convertErr)
		// Once headers/events have been sent, the only useful way to report a
		// failure is an Anthropic SSE error event.  Without this Claude sees a
		// truncated turn and may appear to stop after a tool call.
		sendAnthroEvent(writer, eventError, map[string]any{
			"error": map[string]any{
				"type":    typeAnthroAPIError,
				"message": convertErr.Error(),
			},
		})
		if flusher, ok := writer.(interface{ Flush() }); ok {
			flusher.Flush()
		}
	}

	if err := <-errCh; err != nil {
		slog.Error("Anthropic stream error", "rid", rid, "error", err)
		if eventCount == 0 && convertErr == nil {
			sendAnthroEvent(writer, eventError, map[string]any{
				"error": map[string]any{
					"type":    typeAnthroAPIError,
					"message": err.Error(),
				},
			})
			if flusher, ok := writer.(interface{ Flush() }); ok {
				flusher.Flush()
			}
		}
	}
} */

func convertFromAnthropicRequest(ar anthropic.Request, providerName string) domain.ChatRequest {
	isGemini := providerName == "google" || strings.Contains(providerName, "gemini")

	req := domain.ChatRequest{
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
				switch t {
				case "auto":
					req.ToolChoice = t
				case "any":
					req.ToolChoice = "required"
				case "tool":
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
		var systemParts []domain.ContentPart
		switch v := ar.System.(type) {
		case string:
			systemText = v
		case []any:
			for _, block := range v {
				if b, ok := block.(map[string]any); ok {
					if t, ok := b["type"].(string); ok && t == "text" {
						if txt, ok := b["text"].(string); ok {
							part := domain.ContentPart{Type: "text", Text: txt}
							if _, marked := b["cache_control"].(map[string]any); marked {
								part.PromptCacheBreakpoint = &domain.CacheBreakpoint{Mode: "explicit"}
							}
							systemParts = append(systemParts, part)
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
			req.Messages = append(req.Messages, domain.Message{
				Role: domain.RoleSystem, Content: systemText, Parts: systemParts,
			})
		}
	}

	for _, m := range ar.Messages {
		msg := domain.Message{Role: m.Role}
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
						msg.ToolCalls = append(msg.ToolCalls, domain.ToolCall{
							ID:   id,
							Type: "function",
							Function: &domain.FunctionCall{
								Name:      name,
								Arguments: string(inputJSON),
							},
						})
					case "tool_result":
						toolID, _ := b["tool_use_id"].(string)
						contentStr := ""

						switch c := b["content"].(type) {
						case string:
							contentStr = c
						case []any:
							contentStr = extractToolResultText(c)
						}

						if contentStr == "" {
							contentStr = "..."
						}

						req.Messages = append(req.Messages, domain.Message{
							Role:       "tool",
							ToolCallID: toolID,
							Content:    contentStr,
						})
						continue
					}
				}
			}
		}

		if msg.Content == "" && len(msg.ToolCalls) == 0 && msg.Role != "tool" {
			msg.Content = "..."
		}
		if blocks, ok := m.Content.([]any); ok {
			for _, raw := range blocks {
				block, ok := raw.(map[string]any)
				if !ok || block["type"] != "text" {
					continue
				}
				text, _ := block["text"].(string)
				part := domain.ContentPart{Type: "text", Text: text}
				if _, ok := block["cache_control"].(map[string]any); ok {
					part.PromptCacheBreakpoint = &domain.CacheBreakpoint{Mode: "explicit"}
				}
				msg.Parts = append(msg.Parts, part)
			}
		}

		if msg.Role != "" || msg.Content != "" || len(msg.ToolCalls) > 0 {
			req.Messages = append(req.Messages, msg)
		}
	}

	for _, t := range ar.Tools {
		toolType := t.Type
		if toolType == "" {
			toolType = domain.ToolTypeFunction
		}

		if toolType != domain.ToolTypeFunction {
			continue // Skip native tools, they will be stripped or handled by the proxy
		}

		params := t.InputSchema
		if isGemini {
			params = shared.CleanJSONSchema(params)
		}

		req.Tools = append(req.Tools, domain.Tool{
			Type: domain.ToolTypeFunction,
			Function: &domain.FunctionDefinition{
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

func convertToAnthropicStream(r io.Reader, w io.Writer) (int, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*64), 1024*64)
	flusher, _ := w.(http.Flusher)
	emitted := 0

	var (
		first             = true
		textBlockStarted  = false
		blockIndex        = -1
		activeToolIndex   = -1
		activeToolID      = ""
		activeToolStarted = false
		toolArguments     = make(map[int]string)
		toolArgumentsSent = make(map[int]bool)
		messageDeltaSent  = false
		toolUseSeen       = false
	)

	writeEvent := func(eventType string, data any) {
		sendAnthroEvent(w, eventType, data)
		emitted++
	}

	ensureTextStarted := func() {
		if !textBlockStarted {
			blockIndex++
			writeEvent(eventContentBlockStart, map[string]any{
				"index":         blockIndex,
				"content_block": map[string]any{"type": "text", "text": ""},
			})
			textBlockStarted = true
		}
	}

	ensureTextStopped := func() {
		if textBlockStarted {
			writeEvent(eventContentBlockStop, map[string]any{"index": blockIndex})
			textBlockStarted = false
		}
	}

	emitToolArguments := func() {
		for index, arguments := range toolArguments {
			if toolArgumentsSent[index] || arguments == "" {
				continue
			}
			if normalized, changed := normalizeStreamToolArguments(arguments); changed {
				arguments = normalized
			}
			writeEvent(eventContentBlockDelta, map[string]any{
				"index": index,
				"delta": map[string]any{"type": "input_json_delta", "partial_json": arguments},
			})
			toolArgumentsSent[index] = true
		}
	}

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, common.SSEDataPrefix) {
			continue
		}
		data := strings.TrimPrefix(line, common.SSEDataPrefix)
		if data == "[DONE]" {
			ensureTextStopped()
			emitToolArguments()
			if activeToolIndex != -1 {
				if activeToolStarted {
					writeEvent(eventContentBlockStop, map[string]any{"index": activeToolIndex})
				}
			}
			if !messageDeltaSent {
				stopReason := stopReasonEndTurn
				if toolUseSeen {
					stopReason = "tool_use"
				}
				writeEvent(eventMessageDelta, map[string]any{"delta": map[string]any{"stop_reason": stopReason}, "usage": map[string]any{"output_tokens": 0}})
			}
			writeEvent(eventMessageStop, map[string]any{})
			break
		}

		var chunk map[string]any
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if first {
			model, _ := chunk["model"].(string)
			id, _ := chunk["id"].(string)
			writeEvent(eventMessageStart, map[string]any{
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
		choice, ok := choices[0].(map[string]any)
		if !ok {
			return emitted, fmt.Errorf("invalid OpenAI stream choice: expected object")
		}
		delta, _ := choice["delta"].(map[string]any)

		if content, ok := delta["content"].(string); ok && content != "" {
			ensureTextStarted()
			writeEvent(eventContentBlockDelta, map[string]any{
				"index": blockIndex,
				"delta": map[string]any{
					"type": "text_delta",
					"text": content,
				},
			})
		}

		if toolCalls, ok := delta["tool_calls"].([]any); ok {
			ensureTextStopped()

			for _, tc := range toolCalls {
				t, ok := tc.(map[string]any)
				if !ok {
					return emitted, fmt.Errorf("invalid OpenAI tool call: expected object")
				}
				if id, ok := t["id"].(string); ok && id != activeToolID {
					function, _ := t["function"].(map[string]any)
					name, _ := function["name"].(string)

					if activeToolIndex != -1 && activeToolStarted {
						// Finish the previous block's buffered arguments before closing it.
						// Parallel OpenAI tool calls can arrive in separate chunks; emitting
						// the stop event first causes the client to observe input: {}.
						emitToolArguments()
						writeEvent(eventContentBlockStop, map[string]any{"index": activeToolIndex})
					}

					blockIndex++
					activeToolIndex = blockIndex
					activeToolID = id
					toolUseSeen = true

					if args, ok := function["arguments"].(string); ok {
						toolArguments[activeToolIndex] = args
						if args != "" {
							writeEvent(eventContentBlockStart, map[string]any{
								"index":         activeToolIndex,
								"content_block": map[string]any{"type": "tool_use", "id": id, "name": name, "input": map[string]any{}},
							})
							activeToolStarted = true
						}
					}
				} else if function, ok := t["function"].(map[string]any); ok {
					if args, ok := function["arguments"].(string); ok {
						if args != "" && !activeToolStarted {
							writeEvent(eventContentBlockStart, map[string]any{
								"index":         activeToolIndex,
								"content_block": map[string]any{"type": "tool_use", "id": activeToolID, "name": "", "input": map[string]any{}},
							})
							activeToolStarted = true
						}
						// Buffer tool JSON until the provider signals completion. This
						// lets us remove invalid Claude-Code arguments such as
						// Read.pages="" without having already emitted them.
						if json.Valid([]byte(args)) {
							// Responses providers may send the complete arguments in
							// the final chunk after sending deltas. Avoid duplicating it.
							toolArguments[activeToolIndex] = args
						} else {
							toolArguments[activeToolIndex] += args
						}
					}
				}
			}
		}

		if finish, ok := choice["finish_reason"].(string); ok && finish != "" {
			ensureTextStopped()
			emitToolArguments()

			if activeToolIndex != -1 && activeToolStarted {
				writeEvent(eventContentBlockStop, map[string]any{"index": activeToolIndex})
				activeToolIndex = -1
				activeToolID = ""
				activeToolStarted = false
			}

			stopReason := stopReasonEndTurn
			switch finish {
			case "length":
				stopReason = "max_tokens"
			case "tool_calls":
				stopReason = "tool_use"
			}

			writeEvent(eventMessageDelta, map[string]any{
				"delta": map[string]any{
					"stop_reason": stopReason,
				},
				"usage": map[string]any{"output_tokens": 0},
			})
			messageDeltaSent = true
		}

		if flusher != nil {
			flusher.Flush()
		}
	}

	return emitted, scanner.Err()
}

func (h *AnthropicHandler) writeAnthroStreamResponse(w http.ResponseWriter, resp *domain.ChatResponse) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, _ := w.(http.Flusher)

	// 1. message_start
	sendAnthroEvent(w, eventMessageStart, map[string]any{
		"message": map[string]any{
			"id":    resp.ID,
			"type":  "message",
			"role":  "assistant",
			"model": resp.Model,
			"usage": map[string]int{
				"input_tokens":  resp.Usage.PromptTokens,
				"output_tokens": 0,
			},
		},
	})
	if flusher != nil {
		flusher.Flush()
	}

	if len(resp.Choices) > 0 {
		msg := resp.Choices[0].Message
		blockIdx := 0

		// 2. content_block_start + delta + stop (for text)
		if msg.Content != "" {
			sendAnthroEvent(w, eventContentBlockStart, map[string]any{
				"index":         blockIdx,
				"content_block": map[string]any{"type": "text", "text": ""},
			})
			sendAnthroEvent(w, eventContentBlockDelta, map[string]any{
				"index": blockIdx,
				"delta": map[string]any{"type": "text_delta", "text": msg.Content},
			})
			sendAnthroEvent(w, eventContentBlockStop, map[string]any{"index": blockIdx})
			blockIdx++
		}

		// 3. tool use blocks if any
		for _, tc := range msg.ToolCalls {
			var input any
			json.Unmarshal([]byte(tc.Function.Arguments), &input)

			name := tc.Function.Name
			if name == "" {
				name = tc.Type
			}
			input = normalizeAnthropicToolInput(name, input)

			sendAnthroEvent(w, eventContentBlockStart, map[string]any{
				"index": blockIdx,
				"content_block": map[string]any{
					"type":  "tool_use",
					"id":    tc.ID,
					"name":  name,
					"input": input,
				},
			})
			sendAnthroEvent(w, eventContentBlockStop, map[string]any{"index": blockIdx})
			blockIdx++
		}

		// 4. message_delta (stop_reason)
		stopReason := stopReasonEndTurn
		switch resp.Choices[0].FinishReason {
		case "length":
			stopReason = "max_tokens"
		case "tool_calls":
			stopReason = "tool_use"
		}

		sendAnthroEvent(w, eventMessageDelta, map[string]any{
			"delta": map[string]any{
				"stop_reason": stopReason,
			},
			"usage": map[string]any{
				"output_tokens": resp.Usage.CompletionTokens,
			},
		})
	}

	// 5. message_stop
	sendAnthroEvent(w, eventMessageStop, map[string]any{})
	if flusher != nil {
		flusher.Flush()
	}
}
