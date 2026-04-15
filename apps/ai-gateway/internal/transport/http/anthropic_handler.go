package httptransport

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
)

const (
	typeAnthroAPIError = "api_error"

	stopReasonEndTurn = "end_turn"

	eventMessageStart      = "message_start"
	eventMessageDelta      = "message_delta"
	eventMessageStop       = "message_stop"
	eventContentBlockStart = "content_block_start"
	eventContentBlockDelta = "content_block_delta"
	eventContentBlockStop  = "content_block_stop"
	eventError             = "error"

	logFormatAnthroRequest       = "[ID:%s] [VERBOSE 1] Received Anthropic Request: %s"
	logMsgAnthroFinished         = "[ID:%s] [VERBOSE 2] Finished decoding Anthropic request"
	logMsgAnthroEnteringSync     = "[ID:%s] [VERBOSE 2] Entering Anthropic handleSync"
	logMsgAnthroEnteringStream   = "[ID:%s] [VERBOSE 2] Entering Anthropic handleStream"
	logFormatAnthroProviderError = "[ID:%s] [VERBOSE 1] Provider returned error: %v"
	logFormatAnthroResponse      = "[ID:%s] [VERBOSE 1] Provider Response (Anthropic format): %s"
	logFormatAnthroStreamError   = "[ID:%s] STREAM ERROR: %v"
	logFormatStreamConvErr       = "[ID:%s] STREAM CONVERT ERROR: %v"

	errMsgInvalidAnthroBody = "invalid anthropic request body: "
)

type AnthropicHandler struct {
	registry *service.Registry
}

func NewAnthropicHandler(registry *service.Registry) *AnthropicHandler {
	return &AnthropicHandler{registry: registry}
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
		http.Error(w, errMsgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	var anthroReq anthropic.Request
	if err := json.NewDecoder(r.Body).Decode(&anthroReq); err != nil {
		WriteError(w, r, http.StatusBadRequest, errMsgInvalidAnthroBody+err.Error())
		return
	}

	rid, _ := r.Context().Value(domain.RequestIDKey).(string)

	if h.registry.Config.Verbose >= 1 {
		slog.Debug("Received Anthropic Request", "rid", rid, "body", anthroReq)
	}
	if h.registry.Config.Verbose >= 2 {
		slog.Debug("Finished decoding Anthropic request", "rid", rid)
	}

	provider, targetModel, err := h.registry.ResolveRoute(r, anthroReq.Model)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not ready") {
			status = http.StatusNotFound
		}
		WriteError(w, r, status, errMsgRoutingFailed+err.Error())
		return
	}

	r = SetLogMapping(r, fmt.Sprintf("%s -> %s", anthroReq.Model, targetModel))
	r = SetLogModel(r, anthroReq.Model)

	req := convertFromAnthropicRequest(anthroReq, provider.Name())
	req.Model = targetModel

	if anthroReq.Stream {
		h.handleStream(w, r, provider, req, anthroReq.Model)
	} else {
		h.handleSync(w, r, provider, req, anthroReq.Model)
	}
}

func (h *AnthropicHandler) handleSync(w http.ResponseWriter, r *http.Request, p shared.Provider, req domain.ChatRequest, inputModel string) {
	rid, _ := r.Context().Value(domain.RequestIDKey).(string)
	if h.registry.Config.Verbose >= 2 {
		slog.Debug("Entering Anthropic handleSync", "rid", rid)
	}

	resp, err := p.Chat(r.Context(), req)
	if err != nil {
		if h.registry.Config.Verbose >= 1 {
			slog.Warn("Provider returned error", "rid", rid, "error", err)
		}
		WriteError(w, r, http.StatusBadGateway, err.Error())
		setMetrics(r, p.Name(), req.Model, inputModel, domain.Usage{}, req.Stream, err)
		return
	}

	setMetrics(r, p.Name(), req.Model, inputModel, resp.Usage, req.Stream, nil)

	anthroResp := convertToAnthropicResponse(resp)

	if h.registry.Config.Verbose >= 1 {
		slog.Debug("Provider Response (Anthropic format)", "rid", rid, "response", anthroResp)
	}

	w.Header().Set(headerContentType, contentTypeJSON)
	json.NewEncoder(w).Encode(anthroResp)
}

func (h *AnthropicHandler) handleStream(w http.ResponseWriter, r *http.Request, p shared.Provider, req domain.ChatRequest, inputModel string) {
	rid, _ := r.Context().Value(domain.RequestIDKey).(string)
	if h.registry.Config.Verbose >= 2 {
		slog.Debug("Entering Anthropic handleStream", "rid", rid)
	}

	w.Header().Set(headerContentType, contentTypeEventStream)
	w.Header().Set(headerCacheControl, valueNoCache)
	w.Header().Set(headerConnection, valueKeepAlive)
	w.Header().Set(headerXAccelBuffering, valueNo)

	if _, ok := w.(http.Flusher); !ok {
		WriteError(w, r, http.StatusInternalServerError, errMsgStreamNotSupp)
		return
	}

	pr, pw := io.Pipe()
	errCh := make(chan error, 1)
	go func() {
		defer pw.Close()
		usage, err := p.ChatStream(r.Context(), req, pw)
		errCh <- err
		setMetrics(r, p.Name(), req.Model, inputModel, usage, req.Stream, err)
	}()

	var writer io.Writer = w
	if h.registry.Config.Verbose >= 1 {
		writer = &StreamLogWriter{w: w, rid: rid}
	}

	eventCount, convertErr := convertToAnthropicStream(pr, writer)
	if convertErr != nil {
		slog.Error("Stream convert error", "rid", rid, "error", convertErr)
	}

	if err := <-errCh; err != nil {
		slog.Error("Anthropic stream error", "rid", rid, "error", err)
		if eventCount == 0 {
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
}

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
			req.Messages = append(req.Messages, domain.Message{
				Role:    domain.RoleSystem,
				Content: systemText,
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
							Function: domain.FunctionCall{
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

		if msg.Role != "" || msg.Content != "" || len(msg.ToolCalls) > 0 {
			req.Messages = append(req.Messages, msg)
		}
	}

	for _, t := range ar.Tools {
		params := t.InputSchema
		if isGemini {
			params = shared.CleanJSONSchema(params)
		}

		req.Tools = append(req.Tools, domain.Tool{
			Type: "function",
			Function: domain.FunctionDefinition{
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

func convertToAnthropicResponse(resp *domain.ChatResponse) anthropic.Response {
	ar := anthropic.Response{
		ID:      resp.ID,
		Type:    "message",
		Role:    "assistant",
		Model:   resp.Model,
		Content: []anthropic.Content{},
		Usage: anthropic.Usage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		},
	}

	if len(resp.Choices) > 0 {
		msg := resp.Choices[0].Message
		if msg.Content != "" {
			ar.Content = append(ar.Content, anthropic.Content{
				Type: "text",
				Text: msg.Content,
			})
		}
		for _, tc := range msg.ToolCalls {
			var input any
			json.Unmarshal([]byte(tc.Function.Arguments), &input)
			ar.Content = append(ar.Content, anthropic.Content{
				Type:  "tool_use",
				ID:    tc.ID,
				Name:  tc.Function.Name,
				Input: input,
			})
		}

		ar.StopReason = stopReasonEndTurn
		switch resp.Choices[0].FinishReason {
		case "length":
			ar.StopReason = "max_tokens"
		case "tool_calls":
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

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, sseDataPrefix) {
			continue
		}
		data := strings.TrimPrefix(line, sseDataPrefix)
		if data == "[DONE]" {
			ensureTextStopped()
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
		choice := choices[0].(map[string]any)
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
				t := tc.(map[string]any)
				if id, ok := t["id"].(string); ok {
					name, _ := t["function"].(map[string]any)["name"].(string)

					if activeToolIndex != -1 {
						writeEvent(eventContentBlockStop, map[string]any{"index": activeToolIndex})
					}

					blockIndex++
					activeToolIndex = blockIndex

					writeEvent(eventContentBlockStart, map[string]any{
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
						writeEvent(eventContentBlockDelta, map[string]any{
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

		if finish, ok := choice["finish_reason"].(string); ok && finish != "" {
			ensureTextStopped()

			if activeToolIndex != -1 {
				writeEvent(eventContentBlockStop, map[string]any{"index": activeToolIndex})
				activeToolIndex = -1
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
