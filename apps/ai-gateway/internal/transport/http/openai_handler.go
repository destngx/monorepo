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
	pathChatCompletions = "/v1/chat/completions"
	pathModelsV1        = "/v1/models/"

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
		body, _ := json.MarshalIndent(req, "", "  ")
		slog.Info("FULL OPENAI REQUEST", "rid", rid, "body", string(body))
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
	r = SetLogModel(r, req.Model)
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
			WriteError(w, r, http.StatusTooManyRequests, err.Error())
		} else {
			WriteError(w, r, http.StatusBadGateway, err.Error())
		}
		setMetrics(r, p.Name(), req.Model, inputModel, domain.Usage{}, req.Stream, err)
		return
	}

	setMetrics(r, p.Name(), req.Model, inputModel, resp.Usage, req.Stream, nil)

	if h.registry.Config.Verbose >= 1 {
		body, _ := json.MarshalIndent(resp, "", "  ")
		slog.Info("FULL OPENAI RESPONSE", "rid", rid, "body", string(body))
	}

	w.Header().Set(headerContentType, contentTypeJSON)
	json.NewEncoder(w).Encode(resp)
}

func (h *OpenAIHandler) handleStream(w http.ResponseWriter, r *http.Request, p shared.Provider, req domain.ChatRequest, inputModel string) {
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
		w.Write([]byte(sseDataPrefix + string(b) + "\n\n"))

		setMetrics(r, p.Name(), req.Model, inputModel, domain.Usage{}, req.Stream, err)
		return
	}

	setMetrics(r, p.Name(), req.Model, inputModel, usage, req.Stream, nil)
}

// ModelsHandler exposes the /v1/models endpoint to list available capabilities for a provider.
type ModelsHandler struct {
	registry *service.Registry
}

func NewModelsHandler(registry *service.Registry) *ModelsHandler {
	return &ModelsHandler{registry: registry}
}

// @Summary List models
// @Description List available capabilities/models for a provider.
// @Tags models
// @Produce json
// @Param provider path string false "Provider name (e.g., openai, anthropic)"
// @Param X-AI-Provider header string false "Provider name if not in path"
// @Success 200 {array} string
// @Router /v1/models/{provider} [get]
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

// @Summary Embeddings
// @Description Vector generation endpoint.
// @Tags embeddings
// @Accept json
// @Produce json
// @Param body body domain.EmbeddingRequest true "Embedding Request"
// @Param X-AI-Provider header string false "Provider name override"
// @Success 200 {object} domain.EmbeddingResponse
// @Router /v1/embeddings [post]
func (e *EmbeddingsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, errMsgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	providerName := r.Header.Get(HeaderAIProvider)
	if providerName == "" {
		providerName = domain.ProviderGitHubModels
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
		setMetrics(r, provider.Name(), req.Model, req.Model, domain.Usage{}, false, err)
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

	setMetrics(r, provider.Name(), req.Model, req.Model, resp.Usage, false, nil)

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

func setMetrics(r *http.Request, provider, model, inputModel string, usage domain.Usage, stream bool, err error) {
	payload, ok := r.Context().Value(domain.MetricsPayloadKey).(*domain.MetricsPayload)
	if !ok || payload == nil {
		return
	}
	payload.Provider = provider
	payload.Model = model
	payload.InputModel = inputModel
	payload.Usage = usage
	payload.Stream = stream
	if err != nil {
		payload.Error = err.Error()
	}
}
