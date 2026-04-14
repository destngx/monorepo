package httptransport

import (
	"encoding/json"
	"net/http"

	"apps/ai-gateway/internal/domain"
	"apps/ai-gateway/internal/service"
)

// UsageHandler handles the /v1/usage endpoint.
type UsageHandler struct {
	registry *service.Registry
}

func NewUsageHandler(registry *service.Registry) *UsageHandler {
	return &UsageHandler{registry: registry}
}

// @Summary Usage
// @Description Get usage statistics for a specific provider.
// @Tags usage
// @Produce json
// @Param X-AI-Provider header string false "Provider name override"
// @Success 200 {object} domain.Usage
// @Router /v1/usage [get]
func (h *UsageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	providerName := r.Header.Get(domain.HeaderAIProvider)
	if providerName == "" {
		providerName = domain.ProviderGitHubCopilot
	}

	provider, err := h.registry.Get(providerName)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if !provider.IsConfigured() {
		WriteError(w, r, http.StatusNotFound, "provider "+providerName+" not configured")
		return
	}

	usage, err := provider.Usage(r.Context())
	if err != nil {
		WriteError(w, r, http.StatusBadGateway, err.Error())
		return
	}

	w.Header().Set("Content-Type", domain.ContentTypeJSON)
	json.NewEncoder(w).Encode(usage)
}
