package httptransport

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
)

const (
	pathModelsV1 = "/v1/models/"

	headerContentType     = "Content-Type"
	headerCacheControl    = "Cache-Control"
	headerConnection      = "Connection"
	headerXAccelBuffering = "X-Accel-Buffering"

	contentTypeJSON        = "application/json"
	contentTypeEventStream = "text/event-stream"

	valueNoCache   = "no-cache"
	valueKeepAlive = "keep-alive"
	valueNo        = "no"

	sseDataPrefix = "data: "

	errMsgMethodNotAllowed = "method not allowed"
	errMsgRoutingFailed    = "routing failed: "
	errMsgStreamNotSupp    = "streaming not supported by response writer"
	errMsgInvalidBody      = "invalid request body: "
)

// OpenAIHandler is the primary entry point for the AI Gateway's chat completion interface.
type OpenAIHandler struct {
	registry *service.Registry
}

func NewOpenAIHandler(registry *service.Registry) *OpenAIHandler {
	return &OpenAIHandler{registry: registry}
}

// ServeHTTP handles the /v1/chat/completions endpoint.
func (h *OpenAIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, errMsgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	var req domain.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, errMsgInvalidBody+err.Error())
		return
	}

	rid, _ := r.Context().Value(domain.RequestIDKey).(string)

	if h.registry.Config.Verbose >= 1 {
		slog.Debug("Received OpenAI Request", "rid", rid, "body", req)
	}
	if h.registry.Config.Verbose >= 2 {
		slog.Debug("Finished decoding request", "rid", rid)
	}

	provider, targetModel, err := h.registry.ResolveRoute(r, req.Model)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not ready") {
			status = http.StatusNotFound
		}
		WriteError(w, r, status, errMsgRoutingFailed+err.Error())
		return
	}

	r = SetLogMapping(r, fmt.Sprintf("%s -> %s", req.Model, targetModel))
	req.Model = targetModel

	if req.Stream {
		h.handleStream(w, r, provider, req)
	} else {
		h.handleSync(w, r, provider, req)
	}
}

func (h *OpenAIHandler) handleSync(w http.ResponseWriter, r *http.Request, p shared.Provider, req domain.ChatRequest) {
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
			WriteError(w, r, http.StatusTooManyRequests, err.Error())
		} else {
			WriteError(w, r, http.StatusBadGateway, err.Error())
		}
		return
	}

	if h.registry.Config.Verbose >= 1 {
		slog.Debug("Provider Response", "rid", rid, "response", resp)
	}

	w.Header().Set(headerContentType, contentTypeJSON)
	json.NewEncoder(w).Encode(resp)
}

func (h *OpenAIHandler) handleStream(w http.ResponseWriter, r *http.Request, p shared.Provider, req domain.ChatRequest) {
	w.Header().Set(headerContentType, contentTypeEventStream)
	w.Header().Set(headerCacheControl, valueNoCache)
	w.Header().Set(headerConnection, valueKeepAlive)
	w.Header().Set(headerXAccelBuffering, valueNo)

	if _, ok := w.(http.Flusher); !ok {
		WriteError(w, r, http.StatusInternalServerError, errMsgStreamNotSupp)
		return
	}

	rid, _ := r.Context().Value(domain.RequestIDKey).(string)
	if h.registry.Config.Verbose >= 2 {
		slog.Debug("Entering OpenAI handleStream", "rid", rid)
	}

	var writer io.Writer = w
	if h.registry.Config.Verbose >= 1 {
		writer = &StreamLogWriter{w: w, rid: rid}
	}

	_, err := p.ChatStream(r.Context(), req, writer)
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
		w.Write([]byte(sseDataPrefix + string(b) + "\n\n"))
	}
}

// ModelsHandler exposes the /v1/models endpoint to list available capabilities for a provider.
type ModelsHandler struct {
	registry *service.Registry
}

func NewModelsHandler(registry *service.Registry) *ModelsHandler {
	return &ModelsHandler{registry: registry}
}

func (m *ModelsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, errMsgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	providerName := providerNameFromModelsPath(r.URL.Path)
	if providerName == "" {
		providerName = r.Header.Get(HeaderAIProvider)
	}
	if providerName == "" {
		providerName = "github-copilot"
	}

	provider, err := m.registry.Get(providerName)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if !provider.IsConfigured() {
		WriteError(w, r, http.StatusNotFound, "provider "+providerName+" not configured")
		return
	}

	models, err := provider.ListModels(r.Context())
	if err != nil {
		WriteError(w, r, http.StatusBadGateway, err.Error())
		return
	}

	w.Header().Set(headerContentType, contentTypeJSON)
	json.NewEncoder(w).Encode(models)
}

func providerNameFromModelsPath(path string) string {
	if !strings.HasPrefix(path, pathModelsV1) {
		return ""
	}
	providerName := strings.TrimPrefix(path, pathModelsV1)
	if providerName == "" || strings.Contains(providerName, "/") {
		return ""
	}
	return providerName
}

// EmbeddingsHandler handles the /v1/embeddings endpoint for vector generation.
type EmbeddingsHandler struct {
	registry *service.Registry
}

func NewEmbeddingsHandler(registry *service.Registry) *EmbeddingsHandler {
	return &EmbeddingsHandler{registry: registry}
}

func (e *EmbeddingsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, errMsgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	providerName := r.Header.Get(HeaderAIProvider)
	if providerName == "" {
		providerName = "github"
	}
	provider, err := e.registry.Get(providerName)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if !provider.IsReady() {
		WriteError(w, r, http.StatusNotFound, "provider "+providerName+" not ready")
		return
	}

	var req domain.EmbeddingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	rid, _ := r.Context().Value(domain.RequestIDKey).(string)
	if e.registry.Config.Verbose >= 1 {
		slog.Debug("Received Embeddings Request", "rid", rid, "body", req)
	}
	if e.registry.Config.Verbose >= 2 {
		slog.Debug("Entering Embeddings handler sync", "rid", rid)
	}

	resp, err := provider.Embeddings(r.Context(), req)
	if err != nil {
		if _, ok := err.(*shared.ErrRateLimitExceeded); ok {
			WriteError(w, r, http.StatusTooManyRequests, err.Error())
		} else {
			WriteError(w, r, http.StatusBadGateway, err.Error())
		}
		return
	}

	if e.registry.Config.Verbose >= 1 {
		slog.Debug("Provider Embedding Response", "rid", rid, "response", resp)
	}

	w.Header().Set(headerContentType, contentTypeJSON)
	json.NewEncoder(w).Encode(resp)
}

// WriteError provides a uniform JSON error response including a stack trace for debugging.
func WriteError(w http.ResponseWriter, r *http.Request, code int, msg string) {
	rid, _ := r.Context().Value(domain.RequestIDKey).(string)
	slog.Error("HTTP Error", "rid", rid, "status", code, "message", msg)

	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(code)

	resp := map[string]interface{}{
		"error": msg,
		"stack": string(debug.Stack()),
	}
	json.NewEncoder(w).Encode(resp)
}

// StreamLogWriter wraps an io.Writer to log stream chunks.
type StreamLogWriter struct {
	w   io.Writer
	rid string
}

func (sw *StreamLogWriter) Write(p []byte) (n int, err error) {
	slog.Debug("Stream chunk", "rid", sw.rid, "content", string(p))
	return sw.w.Write(p)
}

func (sw *StreamLogWriter) Flush() {
	if f, ok := sw.w.(http.Flusher); ok {
		f.Flush()
	}
}
