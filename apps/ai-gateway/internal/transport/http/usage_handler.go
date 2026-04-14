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
