package httptransport

import (
	"encoding/json"
	"net/http"

	"apps/ai-gateway/internal/service"
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
