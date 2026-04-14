package httptransport

import (
	"encoding/json"
	"net/http"

	"apps/ai-gateway/internal/domain"
	"apps/ai-gateway/internal/service"
	"github.com/pkoukk/tiktoken-go"
)

type ProviderInfo struct {
	Name         string `json:"name"`
	IsConfigured bool   `json:"configured"`
	IsReady      bool   `json:"ready"`
}

type ProvidersHandler struct {
	registry *service.Registry
}

func NewProvidersHandler(registry *service.Registry) *ProvidersHandler {
	return &ProvidersHandler{registry: registry}
}

// ServeHTTP handles listing registered providers.
// @Summary List providers
// @Description Returns all registered providers with config and ready status
// @Tags providers
// @Produce json
// @Success 200 {array} ProviderInfo
// @Router /providers [get]
func (h *ProvidersHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var infos []ProviderInfo
	for name, p := range h.registry.Providers() {
		infos = append(infos, ProviderInfo{
			Name:         name,
			IsConfigured: p.IsConfigured(),
			IsReady:      p.IsReady(),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(infos)
}

type HealthHandler struct {
	registry *service.Registry
}

func NewHealthHandler(registry *service.Registry) *HealthHandler {
	return &HealthHandler{registry: registry}
}

// ServeHTTP handles health checks.
// @Summary Gateway Health
// @Description Health check and list of registered providers
// @Tags health
// @Produce json
// @Success 200 {object} map[string]interface{} "status: ok, providers: list"
// @Router /health [get]
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"providers": h.registry.List(),
	})
}

type TokenizeHandler struct {
	registry *service.Registry
}

func NewTokenizeHandler(registry *service.Registry) *TokenizeHandler {
	return &TokenizeHandler{registry: registry}
}

// ServeHTTP handles text tokenization.
// @Summary Tokenize text
// @Description Tokenize string text using tiktoken (cl100k_base) and return stats
// @Tags utilities
// @Accept json
// @Produce json
// @Param body body domain.TokenizeRequest true "Text to tokenize"
// @Success 200 {object} domain.TokenizeResponse
// @Router /tokenize [post]
func (h *TokenizeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req domain.TokenizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	tkm, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		http.Error(w, "failed to initialize tokenizer: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tokenIds := tkm.Encode(req.Text, nil, nil)
	tokens := make([]string, 0, len(tokenIds))
	for _, id := range tokenIds {
		tokens = append(tokens, tkm.Decode([]int{id}))
	}

	resp := domain.TokenizeResponse{
		Tokens: tokens,
		Stats: domain.TokenizeStats{
			TokenCount:     len(tokenIds),
			CharacterCount: len([]rune(req.Text)),
			ByteCount:      len(req.Text),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
