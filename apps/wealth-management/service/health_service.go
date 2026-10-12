package service

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
)

type healthService struct{}

func NewHealthService() port.HealthService {
	return &healthService{}
}

func (s *healthService) Check() domain.HealthStatus {
	return domain.HealthStatus{Status: "OK"}
}
