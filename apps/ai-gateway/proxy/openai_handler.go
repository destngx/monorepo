package proxy

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"runtime/debug"

	"apps/ai-gateway/providers"
	"apps/ai-gateway/types"
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
// Supported values: "github" (default), "openai", "anthropic", "ollama".
func (h *OpenAIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Request Decoding: Standardize the incoming OpenAI-style payload.
	// We do this first so we can use the 'model' field for smart routing.
	var req types.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	// 2. Smart Routing: Determine which backend provider and model to use.
	provider, targetModel, err := h.registry.ResolveRoute(r, req.Model)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "routing failed: "+err.Error())
		return
	}
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
func (h *OpenAIHandler) handleSync(w http.ResponseWriter, r *http.Request, p interface {
	Chat(context.Context, types.ChatRequest) (*types.ChatResponse, error)
}, req types.ChatRequest) {
	resp, err := p.Chat(r.Context(), req)
	if err != nil {
		if _, ok := err.(*providers.ErrRateLimitExceeded); ok {
			writeError(w, r, http.StatusTooManyRequests, err.Error())
		} else {
			writeError(w, r, http.StatusBadGateway, err.Error())
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleStream manages long-lived Server-Sent Event (SSE) connections.
// It proxies provider chunks to the client in real-time.
func (h *OpenAIHandler) handleStream(w http.ResponseWriter, r *http.Request, p interface {
	ChatStream(context.Context, types.ChatRequest, io.Writer) (types.Usage, error)
}, req types.ChatRequest) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	if _, ok := w.(http.Flusher); !ok {
		writeError(w, r, http.StatusInternalServerError, "streaming not supported by response writer")
		return
	}

	_, err := p.ChatStream(r.Context(), req, w)
	if err != nil {
		rid, _ := r.Context().Value(requestIDKey).(string)
		log.Printf("[ID:%s] STREAM ERROR: %v", rid, err)

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
		w.Write([]byte("data: " + string(b) + "\n\n"))
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
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	providerName := r.Header.Get("X-AI-Provider")
	if providerName == "" {
		providerName = "github"
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
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
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	providerName := r.Header.Get("X-AI-Provider")
	if providerName == "" {
		providerName = "github"
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

	resp, err := provider.Embeddings(r.Context(), req)
	if err != nil {
		if _, ok := err.(*providers.ErrRateLimitExceeded); ok {
			writeError(w, r, http.StatusTooManyRequests, err.Error())
		} else {
			writeError(w, r, http.StatusBadGateway, err.Error())
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// writeError provides a uniform JSON error response including a stack trace for debugging.
func writeError(w http.ResponseWriter, r *http.Request, code int, msg string) {
	rid, _ := r.Context().Value(requestIDKey).(string)
	log.Printf("[ID:%s] ERROR: code=%d msg=%s", rid, code, msg)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	resp := map[string]interface{}{
		"error": msg,
		"stack": string(debug.Stack()),
	}
	json.NewEncoder(w).Encode(resp)
}
