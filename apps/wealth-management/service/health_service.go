package service

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"sync"
	"time"
)

const healthProbeTimeout = 5 * time.Second

type healthService struct {
	mu     sync.RWMutex
	checks []port.HealthCheck
}

// HealthRegistry is a HealthService whose checks are added as dependencies
// come up during startup.
type HealthRegistry interface {
	port.HealthService
	Register(check port.HealthCheck)
}

func NewHealthService(checks ...port.HealthCheck) HealthRegistry {
	return &healthService{checks: checks}
}

func (s *healthService) Register(check port.HealthCheck) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checks = append(s.checks, check)
}

// Check probes every dependency concurrently. Any failing or unconfigured
// dependency makes the overall status degraded.
func (s *healthService) Check(ctx context.Context) domain.HealthStatus {
	s.mu.RLock()
	checks := append([]port.HealthCheck(nil), s.checks...)
	s.mu.RUnlock()

	results := make([]domain.DependencyHealth, len(checks))
	var wg sync.WaitGroup
	for i, check := range checks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = probe(ctx, check)
		}()
	}
	wg.Wait()

	status := domain.HealthStatus{Status: domain.HealthOK, Checks: results}
	for _, result := range results {
		if result.Status != domain.HealthOK {
			status.Status = domain.HealthDegraded
		}
	}
	return status
}

func probe(ctx context.Context, check port.HealthCheck) domain.DependencyHealth {
	result := domain.DependencyHealth{Name: check.Name, Status: domain.HealthOK}
	if check.Probe == nil {
		result.Status = domain.HealthUnconfigured
		return result
	}
	ctx, cancel := context.WithTimeout(ctx, healthProbeTimeout)
	defer cancel()
	started := time.Now()
	err := check.Probe(ctx)
	result.LatencyMs = time.Since(started).Milliseconds()
	if err != nil {
		result.Status = domain.HealthDown
		result.Error = err.Error()
	}
	return result
}
