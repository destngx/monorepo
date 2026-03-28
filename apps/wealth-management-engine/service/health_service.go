package service

import (
	"apps/wealth-management-engine/domain"
	"apps/wealth-management-engine/port"
)

type healthService struct{}

func NewHealthService() port.HealthService {
	return &healthService{}
}

func (s *healthService) Check() domain.HealthStatus {
	return domain.HealthStatus{Status: "OK"}
}
