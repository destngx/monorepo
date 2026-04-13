package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime/debug"
	"strings"

	"apps/ai-gateway/providers"
	"apps/ai-gateway/types"
)

const (
	LogFormatOpenAIRequest         = "[ID:%s] [VERBOSE 1] Received OpenAI Request: %s"
	LogMsgOpenAIDecodingFinished   = "[ID:%s] [VERBOSE 2] Finished decoding request"
	LogMsgOpenAIEnteringSync       = "[ID:%s] [VERBOSE 2] Entering handleSync"
	LogFormatOpenAIProviderError   = "[ID:%s] [VERBOSE 1] Provider returned error: %v"
	LogMsgOpenAIEnteringStream     = "[ID:%s] [VERBOSE 2] Entering handleStream"
	LogFormatOpenAIStreamError     = "[ID:%s] STREAM ERROR: %v"
	LogFormatEmbeddingsRequest     = "[ID:%s] [VERBOSE 1] Received Embeddings Request: %s"
	LogMsgEnteringEmbeddings       = "[ID:%s] [VERBOSE 2] Entering Embeddings handler sync"
	LogFormatProviderEmbedResponse = "[ID:%s] [VERBOSE 1] Provider Embeddings Response: %s"

	PathModelsV1 = "/v1/models/"
)

// OpenAIHandler is the primary entry point for the AI Gateway's chat completion interface.
type OpenAIHandler struct {
	registry *Registry
}

func NewOpenAIHandler(registry *Registry) *OpenAIHandler {
	return &OpenAIHandler{registry: registry}
}

// ServeHTTP handles the /v1/chat/completions endpoint.
// It is fully OpenAI-compatible and serves as a universal interface for multiple AI providers.
//
// Protocol Selection:
// Clients must specify the target backend using the 'X-AI-Provider' header.
// Supported values: "github" (Copilot, default), "github-models", "openai", "anthropic", "ollama".
func (h *OpenAIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, ErrMsgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	// 1. Request Decoding: Standardize the incoming OpenAI-style payload.
	// We do this first so we can use the 'model' field for smart routing.
	var req types.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrMsgInvalidBody+err.Error())
		return
	}

	rid, _ := r.Context().Value(requestIDKey).(string)

	if h.registry.Config.Verbose >= 1 {
		reqBytes, _ := json.Marshal(req)
		log.Printf(LogFormatOpenAIRequest, rid, string(reqBytes))
	}
	if h.registry.Config.Verbose >= 2 {
		log.Printf(LogMsgOpenAIDecodingFinished, rid)
	}

	// 2. Smart Routing: Determine which backend provider and model to use.
	provider, targetModel, err := h.registry.ResolveRoute(r, req.Model)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not ready") {
			status = http.StatusNotFound
		}
		writeError(w, r, status, ErrMsgRoutingFailed+err.Error())
		return
	}

	// Improvement 5: Rich Mapping Logs
	r = SetLogMapping(r, fmt.Sprintf("%s -> %s", req.Model, targetModel))

	req.Model = targetModel

	// 4. Mode Routing: Branch into either Synchronous (JSON) or Streaming (SSE) response modes.
	if req.Stream {
		h.handleStream(w, r, provider, req)
	} else {
		h.handleSync(w, r, provider, req)
	}
}

// handleSync manages standard Request-Response interactions.
// It waits for the full upstream response before returning a single JSON object.
func (h *OpenAIHandler) handleSync(w http.ResponseWriter, r *http.Request, p providers.Provider, req types.ChatRequest) {
	rid, _ := r.Context().Value(requestIDKey).(string)
	if h.registry.Config.Verbose >= 2 {
		log.Printf(LogMsgOpenAIEnteringSync, rid)
	}

	resp, err := p.Chat(r.Context(), req)
	if err != nil {
		if h.registry.Config.Verbose >= 1 {
			log.Printf(LogFormatOpenAIProviderError, rid, err)
		}
		if _, ok := err.(*providers.ErrRateLimitExceeded); ok {
			writeError(w, r, http.StatusTooManyRequests, err.Error())
		} else {
			writeError(w, r, http.StatusBadGateway, err.Error())
		}
		return
	}

	if h.registry.Config.Verbose >= 1 {
		respBytes, _ := json.Marshal(resp)
		log.Printf("[ID:%s] [VERBOSE 1] Provider Response: %s", rid, string(respBytes))
	}

	w.Header().Set(HeaderContentType, ContentTypeJSON)
	json.NewEncoder(w).Encode(resp)
}

// handleStream manages long-lived Server-Sent Event (SSE) connections.
// It proxies provider chunks to the client in real-time.
func (h *OpenAIHandler) handleStream(w http.ResponseWriter, r *http.Request, p interface {
	ChatStream(context.Context, types.ChatRequest, io.Writer) (types.Usage, error)
}, req types.ChatRequest) {
	w.Header().Set(HeaderContentType, ContentTypeEventStream)
	w.Header().Set(HeaderCacheControl, ValueNoCache)
	w.Header().Set(HeaderConnection, ValueKeepAlive)
	w.Header().Set(HeaderXAccelBuffering, ValueNo)

	if _, ok := w.(http.Flusher); !ok {
		writeError(w, r, http.StatusInternalServerError, ErrMsgStreamNotSupp)
		return
	}

	rid, _ := r.Context().Value(requestIDKey).(string)
	if h.registry.Config.Verbose >= 2 {
		log.Printf(LogMsgOpenAIEnteringStream, rid)
	}

	var writer io.Writer = w
	if h.registry.Config.Verbose >= 1 {
		writer = &StreamLogWriter{w: w, rid: rid}
	}

	_, err := p.ChatStream(r.Context(), req, writer)
	if err != nil {
		log.Printf(LogFormatOpenAIStreamError, rid, err)

		status := http.StatusBadGateway
		if _, ok := err.(*providers.ErrRateLimitExceeded); ok {
			status = http.StatusTooManyRequests
		}

		// If the stream has already started, we must communicate the error via an SSE data block.
		errResp := map[string]interface{}{
			"error":  err.Error(),
			"stack":  string(debug.Stack()),
			"status": status,
		}
		b, _ := json.Marshal(errResp)
		w.Write([]byte(SSEDataPrefix + string(b) + "\n\n"))
	}
}

// ModelsHandler exposes the /v1/models endpoint to list available capabilities for a provider.
type ModelsHandler struct {
	registry *Registry
}

func NewModelsHandler(registry *Registry) *ModelsHandler {
	return &ModelsHandler{registry: registry}
}

func (m *ModelsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, ErrMsgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	providerName := providerNameFromModelsPath(r.URL.Path)
	if providerName == "" {
		providerName = r.Header.Get(HeaderAIProvider)
	}
	if providerName == "" {
		providerName = ProviderGitHubCopilot
	}

	provider, err := m.registry.Get(providerName)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if !provider.IsConfigured() {
		writeError(w, r, http.StatusNotFound, "provider "+providerName+" not configured")
		return
	}

	models, err := provider.ListModels(r.Context())
	if err != nil {
		writeError(w, r, http.StatusBadGateway, err.Error())
		return
	}

	w.Header().Set(HeaderContentType, ContentTypeJSON)
	json.NewEncoder(w).Encode(models)
}

func providerNameFromModelsPath(path string) string {
	if !strings.HasPrefix(path, PathModelsV1) {
		return ""
	}
	providerName := strings.TrimPrefix(path, PathModelsV1)
	if providerName == "" || strings.Contains(providerName, "/") {
		return ""
	}
	return providerName
}

// EmbeddingsHandler handles the /v1/embeddings endpoint for vector generation.
type EmbeddingsHandler struct {
	registry *Registry
}

func NewEmbeddingsHandler(registry *Registry) *EmbeddingsHandler {
	return &EmbeddingsHandler{registry: registry}
}

func (e *EmbeddingsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, ErrMsgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	providerName := r.Header.Get(HeaderAIProvider)
	if providerName == "" {
		providerName = ProviderGitHub
	}
	provider, err := e.registry.Get(providerName)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if !provider.IsReady() {
		writeError(w, r, http.StatusNotFound, "provider "+providerName+" not ready")
		return
	}

	var req types.EmbeddingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	rid, _ := r.Context().Value(requestIDKey).(string)
	if e.registry.Config.Verbose >= 1 {
		reqBytes, _ := json.Marshal(req)
		log.Printf(LogFormatEmbeddingsRequest, rid, string(reqBytes))
	}
	if e.registry.Config.Verbose >= 2 {
		log.Printf(LogMsgEnteringEmbeddings, rid)
	}

	resp, err := provider.Embeddings(r.Context(), req)
	if err != nil {
		if _, ok := err.(*providers.ErrRateLimitExceeded); ok {
			writeError(w, r, http.StatusTooManyRequests, err.Error())
		} else {
			writeError(w, r, http.StatusBadGateway, err.Error())
		}
		return
	}

	if e.registry.Config.Verbose >= 1 {
		respBytes, _ := json.Marshal(resp)
		log.Printf(LogFormatProviderEmbedResponse, rid, string(respBytes))
	}

	w.Header().Set(HeaderContentType, ContentTypeJSON)
	json.NewEncoder(w).Encode(resp)
}

// writeError provides a uniform JSON error response including a stack trace for debugging.
func writeError(w http.ResponseWriter, r *http.Request, code int, msg string) {
	rid, _ := r.Context().Value(requestIDKey).(string)
	log.Printf(LogFormatError, rid, code, msg)

	w.Header().Set(HeaderContentType, ContentTypeJSON)
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
	log.Printf(LogFormatStreamChunk, sw.rid, string(p))
	return sw.w.Write(p)
}

func (sw *StreamLogWriter) Flush() {
	if f, ok := sw.w.(http.Flusher); ok {
		f.Flush()
	}
}
