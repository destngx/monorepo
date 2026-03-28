package port

import "apps/wealth-management-engine/domain"

type HealthService interface {
	Check() domain.HealthStatus
}
