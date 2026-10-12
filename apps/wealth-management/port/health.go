package port

import "apps/wealth-management/domain"

type HealthService interface {
	Check() domain.HealthStatus
}
