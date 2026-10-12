package domain

const (
	HealthOK           = "ok"
	HealthDegraded     = "degraded"
	HealthDown         = "down"
	HealthUnconfigured = "unconfigured"
)

// HealthStatus is the overall status plus one entry per dependency (WM-OPS-6).
type HealthStatus struct {
	Status string             `json:"status"`
	Checks []DependencyHealth `json:"checks"`
}

type DependencyHealth struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
	LatencyMs int64  `json:"latencyMs"`
}
