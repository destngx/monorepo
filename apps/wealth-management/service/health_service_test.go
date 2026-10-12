package service

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"errors"
	"testing"
)

func TestHealthReportsEachDependency(t *testing.T) {
	ok := func(context.Context) error { return nil }
	cases := []struct {
		name   string
		checks []port.HealthCheck
		want   string
	}{
		{"all up", []port.HealthCheck{{Name: "sheets", Probe: ok}, {Name: "cache", Probe: ok}}, domain.HealthOK},
		{"one down", []port.HealthCheck{{Name: "sheets", Probe: ok}, {Name: "cache", Probe: func(context.Context) error { return errors.New("timeout") }}}, domain.HealthDegraded},
		{"unconfigured", []port.HealthCheck{{Name: "sheets"}}, domain.HealthDegraded},
		{"no dependencies", nil, domain.HealthOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status := NewHealthService(tc.checks...).Check(context.Background())
			if status.Status != tc.want || len(status.Checks) != len(tc.checks) {
				t.Fatalf("status = %+v", status)
			}
		})
	}
}

func TestHealthRegisterAddsChecks(t *testing.T) {
	registry := NewHealthService()
	registry.Register(port.HealthCheck{Name: "cache", Probe: func(context.Context) error { return errors.New("down") }})

	status := registry.Check(context.Background())
	if status.Checks[0].Status != domain.HealthDown || status.Checks[0].Error != "down" {
		t.Fatalf("status = %+v", status)
	}
}
