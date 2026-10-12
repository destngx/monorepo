package port

import (
	"apps/wealth-management/domain"
	"context"
)

type HealthService interface {
	Check(ctx context.Context) domain.HealthStatus
}

// HealthCheck probes one dependency. A nil Probe reports "unconfigured".
type HealthCheck struct {
	Name  string
	Probe func(ctx context.Context) error
}
