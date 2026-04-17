package httptransport

import (
	"apps/ai-gateway/internal/service"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
)

type MetricsHandler struct {
	collector *service.MetricsCollector
}

func NewMetricsHandler(collector *service.MetricsCollector) *MetricsHandler {
	return &MetricsHandler{collector: collector}
}

// ServeHTTP handles the /metrics endpoint.
// @Summary Get gateway metrics
// @Description Returns an aggregated summary of gateway usage, including requests, errors, tokens, and latency by provider/model.
// @Tags monitoring
// @Produce json
// @Success 200 {object} domain.MetricsSummary
// @Router /metrics [get]
func (h *MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	summary := h.collector.Summary()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

type MetricsResetHandler struct {
	collector *service.MetricsCollector
}

func NewMetricsResetHandler(collector *service.MetricsCollector) *MetricsResetHandler {
	return &MetricsResetHandler{collector: collector}
}

// ServeHTTP handles the /metrics/reset endpoint.
// @Summary Reset gateway metrics
// @Description Clears all in-memory metrics counters and buffers.
// @Tags monitoring
// @Produce json
// @Success 200 {object} map[string]string
// @Router /metrics/reset [post]
func (h *MetricsResetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	h.collector.Reset()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

type DashboardHandler struct{}

func NewDashboardHandler() *DashboardHandler {
	return &DashboardHandler{}
}

// ServeHTTP handles the /metrics/dashboard endpoint.
// @Summary Metrics dashboard
// @Description Returns the self-contained HTML/JS monitoring dashboard.
// @Tags monitoring
// @Produce html
// @Success 200 {string} string "HTML Dashboard"
// @Router /metrics/dashboard [get]
func (h *DashboardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, dashboardHTML)
}

//go:embed dashboard.html
var dashboardHTML string
