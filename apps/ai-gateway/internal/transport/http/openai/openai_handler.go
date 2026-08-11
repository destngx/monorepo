package openai

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"

	"apps/ai-gateway/internal/domain"
	"apps/ai-gateway/internal/providers/shared"
	"apps/ai-gateway/internal/service"
	httptransport "apps/ai-gateway/internal/transport/http"
	"apps/ai-gateway/internal/transport/http/common"
)

const (
	pathChatCompletions = "/v1/chat/completions"
	pathResponses       = "/v1/responses"
	pathModelsV1        = "/v1/models/"
)

// OpenAIHandler is the primary entry point for the AI Gateway's chat completion interface.
type OpenAIHandler struct {
	registry *service.Registry
}

func NewOpenAIHandler(registry *service.Registry) *OpenAIHandler {
	return &OpenAIHandler{registry: registry}
}

// ServeHTTP handles the /v1/chat/completions endpoint.
// @Summary Chat completions
// @Description Entry point for the AI Gateway's chat completion interface (OpenAI compatible).
// @Tags completions
// @Accept json
// @Produce json
// @Param body body domain.ChatRequest true "Chat Request"
// @Success 200 {object} domain.ChatResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 405 {string} string "method not allowed"
// @Failure 502 {object} map[string]interface{}
// @Router /v1/chat/completions [post]
func (h *OpenAIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, common.ErrMsgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	var req domain.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteError(w, r, http.StatusBadRequest, common.ErrMsgInvalidBody+err.Error())
		return
	}

	rid, _ := r.Context().Value(domain.RequestIDKey).(string)

	if h.registry.Config.Verbose >= 1 {
		body, _ := json.MarshalIndent(req, "", "  ")
		slog.Info("FULL OPENAI REQUEST", "rid", rid, "body", string(body))
	}

	provider, targetModel, err := h.registry.ResolveRoute(r, req.Model)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not ready") {
			status = http.StatusNotFound
		}
		common.WriteError(w, r, status, common.ErrMsgRoutingFailed+err.Error())
		return
	}

	r = httptransport.SetLogMapping(r, fmt.Sprintf("%s -> %s", req.Model, targetModel))
	r = httptransport.SetLogProvider(r, provider.Name())
	r = httptransport.SetLogModel(r, req.Model)
	req.Model = targetModel

	if req.Stream {
		h.handleStream(w, r, provider, req, req.Model)
	} else {
		h.handleSync(w, r, provider, req, req.Model)
	}
}

func (h *OpenAIHandler) handleSync(w http.ResponseWriter, r *http.Request, p shared.Provider, req domain.ChatRequest, inputModel string) {
	rid, _ := r.Context().Value(domain.RequestIDKey).(string)
	if h.registry.Config.Verbose >= 2 {
		slog.Debug("Entering OpenAI handleSync", "rid", rid)
	}

	resp, err := p.Chat(r.Context(), req)
	if err != nil {
		if h.registry.Config.Verbose >= 1 {
			slog.Warn("Provider returned error", "rid", rid, "error", err)
		}
		if _, ok := err.(*shared.ErrRateLimitExceeded); ok {
			common.WriteError(w, r, http.StatusTooManyRequests, err.Error())
		} else {
			common.WriteError(w, r, http.StatusBadGateway, err.Error())
		}
		common.SetMetrics(r, p.Name(), req.Model, inputModel, domain.Usage{}, req.Stream, err)
		return
	}

	common.SetMetrics(r, p.Name(), req.Model, inputModel, resp.Usage, req.Stream, nil)

	if h.registry.Config.Verbose >= 1 {
		body, _ := json.MarshalIndent(resp, "", "  ")
		slog.Info("FULL OPENAI RESPONSE", "rid", rid, "body", string(body))
	}

	w.Header().Set(common.HeaderContentType, common.ContentTypeJSON)
	json.NewEncoder(w).Encode(resp)
}

func (h *OpenAIHandler) handleStream(w http.ResponseWriter, r *http.Request, p shared.Provider, req domain.ChatRequest, inputModel string) {
	w.Header().Set(common.HeaderContentType, common.ContentTypeEventStream)
	w.Header().Set(common.HeaderCacheControl, common.ValueNoCache)
	w.Header().Set(common.HeaderConnection, common.ValueKeepAlive)
	w.Header().Set(common.HeaderXAccelBuffering, common.ValueNo)

	if _, ok := w.(http.Flusher); !ok {
		common.WriteError(w, r, http.StatusInternalServerError, common.ErrMsgStreamNotSupp)
		return
	}

	rid, _ := r.Context().Value(domain.RequestIDKey).(string)
	if h.registry.Config.Verbose >= 2 {
		slog.Debug("Entering OpenAI handleStream", "rid", rid)
	}

	var writer io.Writer = w
	if h.registry.Config.Verbose >= 1 {
		writer = &common.StreamLogWriter{W: w, RID: rid}
	}

	usage, err := p.ChatStream(r.Context(), req, writer)
	if err != nil {
		slog.Error("OpenAI stream error", "rid", rid, "error", err)

		status := http.StatusBadGateway
		if _, ok := err.(*shared.ErrRateLimitExceeded); ok {
			status = http.StatusTooManyRequests
		}

		errResp := map[string]interface{}{
			"error":  err.Error(),
			"stack":  string(debug.Stack()),
			"status": status,
		}
		b, _ := json.Marshal(errResp)
		w.Write([]byte(common.SSEDataPrefix + string(b) + "\n\n"))

		common.SetMetrics(r, p.Name(), req.Model, inputModel, domain.Usage{}, req.Stream, err)
		return
	}

	common.SetMetrics(r, p.Name(), req.Model, inputModel, usage, req.Stream, nil)
}

// ResponsesHandler handles the /v1/responses endpoint.
type ResponsesHandler struct {
	registry *service.Registry
}

func NewResponsesHandler(registry *service.Registry) *ResponsesHandler {
	return &ResponsesHandler{registry: registry}
}

// @Summary Responses
// @Description Native Responses API endpoint.
// @Tags responses
// @Accept json
// @Produce json
// @Param body body domain.ResponsesRequest true "Responses Request"
// @Param X-AI-Provider header string false "Provider name override"
// @Success 200 {object} domain.ResponsesResponse
// @Router /v1/responses [post]
func (h *ResponsesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, common.ErrMsgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	var req domain.ResponsesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteError(w, r, http.StatusBadRequest, common.ErrMsgInvalidBody+err.Error())
		return
	}

	provider, targetModel, err := h.registry.ResolveRoute(r, req.Model)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not ready") {
			status = http.StatusNotFound
		}
		common.WriteError(w, r, status, common.ErrMsgRoutingFailed+err.Error())
		return
	}

	r = httptransport.SetLogMapping(r, fmt.Sprintf("%s -> %s", req.Model, targetModel))
	r = httptransport.SetLogModel(r, req.Model)
	req = req.WithModel(targetModel)

	if req.Stream {
		h.handleStream(w, r, provider, req, req.Model)
		return
	}
	h.handleSync(w, r, provider, req, req.Model)
}

func (h *ResponsesHandler) handleSync(w http.ResponseWriter, r *http.Request, p shared.Provider, req domain.ResponsesRequest, inputModel string) {
	resp, err := p.Responses(r.Context(), req)
	if err != nil {
		if _, ok := err.(*shared.ErrRateLimitExceeded); ok {
			common.WriteError(w, r, http.StatusTooManyRequests, err.Error())
		} else {
			common.WriteError(w, r, http.StatusBadGateway, err.Error())
		}
		common.SetMetrics(r, p.Name(), req.Model, inputModel, domain.Usage{}, false, err)
		return
	}

	usage := domain.UsageFromResponsesValue(map[string]any(*resp))
	common.SetMetrics(r, p.Name(), req.Model, inputModel, usage, false, nil)

	w.Header().Set(common.HeaderContentType, common.ContentTypeJSON)
	json.NewEncoder(w).Encode(resp)
}

func (h *ResponsesHandler) handleStream(w http.ResponseWriter, r *http.Request, p shared.Provider, req domain.ResponsesRequest, inputModel string) {
	w.Header().Set(common.HeaderContentType, common.ContentTypeEventStream)
	w.Header().Set(common.HeaderCacheControl, common.ValueNoCache)
	w.Header().Set(common.HeaderConnection, common.ValueKeepAlive)
	w.Header().Set(common.HeaderXAccelBuffering, common.ValueNo)

	if _, ok := w.(http.Flusher); !ok {
		common.WriteError(w, r, http.StatusInternalServerError, common.ErrMsgStreamNotSupp)
		return
	}

	rid, _ := r.Context().Value(domain.RequestIDKey).(string)
	var writer io.Writer = w
	if h.registry.Config.Verbose >= 1 {
		writer = &common.StreamLogWriter{W: w, RID: rid}
	}

	usage, err := p.ResponsesStream(r.Context(), req, writer)
	if err != nil {
		status := http.StatusBadGateway
		if _, ok := err.(*shared.ErrRateLimitExceeded); ok {
			status = http.StatusTooManyRequests
		}
		errResp := map[string]interface{}{
			"error":  err.Error(),
			"stack":  string(debug.Stack()),
			"status": status,
		}
		b, _ := json.Marshal(errResp)
		w.Write([]byte(common.SSEDataPrefix + string(b) + "\n\n"))
		common.SetMetrics(r, p.Name(), req.Model, inputModel, domain.Usage{}, true, err)
		return
	}

	common.SetMetrics(r, p.Name(), req.Model, inputModel, usage, true, nil)
}
