package proxy

import (
	"encoding/json"
	"net/http"
)

// UsageHandler handles the /v1/usage endpoint.
type UsageHandler struct {
	registry *Registry
}

func NewUsageHandler(registry *Registry) *UsageHandler {
	return &UsageHandler{registry: registry}
}

func (h *UsageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	providerName := r.Header.Get("X-AI-Provider")
	if providerName == "" {
		providerName = "github-copilot"
	}
	if providerName == "github" {
		providerName = "github-copilot"
	}

	provider, err := h.registry.Get(providerName)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if !provider.IsConfigured() {
		writeError(w, r, http.StatusNotFound, "provider "+providerName+" not configured")
		return
	}

	usage, err := provider.Usage(r.Context())
	if err != nil {
		writeError(w, r, http.StatusBadGateway, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(usage)
}
