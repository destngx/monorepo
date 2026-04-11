package proxy

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"apps/ai-gateway/types"
)

type Handler struct {
	registry *Registry
}

func NewHandler(registry *Registry) *Handler {
	return &Handler{registry: registry}
}

// ServeHTTP routes /v1/chat/completions — fully OpenAI-compatible.
//
// Required client header:
//
//	X-AI-Provider: github | openai | anthropic | ollama
//
// The model name in the request body is passed directly to the provider.
// Examples:
//
//	GitHub:    "model": "openai/gpt-4.1"
//	OpenAI:   "model": "gpt-4o"
//	Anthropic: "model": "claude-3-5-sonnet-20241022"
//	Ollama:   "model": "llama3.2"
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Resolve provider from header
	providerName := r.Header.Get("X-AI-Provider")
	if providerName == "" {
		providerName = "github" // sensible default
	}
	provider, err := h.registry.Get(providerName)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if !provider.IsReady() {
		writeError(w, http.StatusNotFound, "provider "+providerName+" not ready (token or ping failed)")
		return
	}

	// 2. Decode request
	var req types.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	// 3. Route: streaming vs non-streaming
	if req.Stream {
		h.handleStream(w, r, provider, req)
	} else {
		h.handleSync(w, r, provider, req)
	}
}

func (h *Handler) handleSync(w http.ResponseWriter, r *http.Request, p interface {
	Chat(context.Context, types.ChatRequest) (*types.ChatResponse, error)
}, req types.ChatRequest) {
	resp, err := p.Chat(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) handleStream(w http.ResponseWriter, r *http.Request, p interface {
	ChatStream(context.Context, types.ChatRequest, io.Writer) (types.Usage, error)
}, req types.ChatRequest) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}
	_ = flusher

	_, err := p.ChatStream(r.Context(), req, w)
	if err != nil {
		// Stream already started — cannot change status code; write error as SSE
		w.Write([]byte("data: {\"error\": \"" + err.Error() + "\"}\n\n"))
	}
}

// ModelsHandler handles GET /v1/models — proxies to the correct provider.
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
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if !provider.IsConfigured() {
		writeError(w, http.StatusNotFound, "provider "+providerName+" not configured")
		return
	}

	models, err := provider.ListModels(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
}

// EmbeddingsHandler handles POST /v1/embeddings.
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
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if !provider.IsReady() {
		writeError(w, http.StatusNotFound, "provider "+providerName+" not ready")
		return
	}

	var req types.EmbeddingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	resp, err := provider.Embeddings(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
