package httptransport

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"apps/ai-gateway/internal/domain"
	"apps/ai-gateway/internal/providers/shared"
	"apps/ai-gateway/internal/service"
	"apps/ai-gateway/internal/transport/http/common"
)

const pathModelsV1 = "/v1/models/"

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
		http.Error(w, common.ErrMsgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	providerName := providerNameFromModelsPath(r.URL.Path)
	if providerName == "" {
		providerName = r.Header.Get(common.HeaderAIProvider)
	}
	if providerName == "" {
		providerName = "github-copilot"
	}

	provider, err := m.registry.Get(providerName)
	if err != nil {
		common.WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if !provider.IsConfigured() {
		common.WriteError(w, r, http.StatusNotFound, "provider "+providerName+" not configured")
		return
	}

	models, err := provider.ListModels(r.Context())
	if err != nil {
		common.WriteError(w, r, http.StatusBadGateway, err.Error())
		return
	}

	w.Header().Set(common.HeaderContentType, common.ContentTypeJSON)
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
		http.Error(w, common.ErrMsgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	providerName := r.Header.Get(common.HeaderAIProvider)
	if providerName == "" {
		providerName = domain.ProviderOpenAI
	}
	provider, err := e.registry.Get(providerName)
	if err != nil {
		common.WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if !provider.IsReady() {
		common.WriteError(w, r, http.StatusNotFound, "provider "+providerName+" not ready")
		return
	}

	var req domain.EmbeddingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteError(w, r, http.StatusBadRequest, "invalid request body: "+err.Error())
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
		common.SetMetrics(r, provider.Name(), req.Model, req.Model, domain.Usage{}, false, err)
		if _, ok := err.(*shared.ErrRateLimitExceeded); ok {
			common.WriteError(w, r, http.StatusTooManyRequests, err.Error())
		} else {
			common.WriteError(w, r, http.StatusBadGateway, err.Error())
		}
		return
	}

	if e.registry.Config.Verbose >= 1 {
		slog.Debug("Provider Embedding Response", "rid", rid, "response", resp)
	}

	common.SetMetrics(r, provider.Name(), req.Model, req.Model, resp.Usage, false, nil)

	w.Header().Set(common.HeaderContentType, common.ContentTypeJSON)
	json.NewEncoder(w).Encode(resp)
}
