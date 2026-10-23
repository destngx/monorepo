package service

import (
	"context"

	"apps/health-watcher/domain"
	"apps/health-watcher/port"
)

// MetricsOverview is a person's metrics dashboard (HW-MET-4): each metric
// with values, grouped by category in catalog order, and how many approved
// results are not linked to the catalog yet.
type MetricsOverview struct {
	Person   domain.Person          `json:"person"`
	Groups   []domain.OverviewGroup `json:"groups"`
	Unlinked int                    `json:"unlinked"`
}

// MetricTrend is one metric of a person over time (HW-MET-3).
type MetricTrend struct {
	Person domain.Person       `json:"person"`
	Metric domain.Metric       `json:"metric"`
	Points []domain.TrendPoint `json:"points"`
}

// personPoints loads a person and their approved observations as trend
// points by metric code, with BMI computed from weight and height.
func (c *Clinical) personPoints(ctx context.Context, sc domain.Scope, personID string) (domain.Person, map[string][]domain.TrendPoint, int, error) {
	if err := sc.RequireRead(); err != nil {
		return domain.Person{}, nil, 0, err
	}
	person, err := c.Store.Person(ctx, sc.TenantID, personID)
	if err != nil {
		return person, nil, 0, err
	}
	obs, err := c.Store.ObservationsOfPerson(ctx, sc.TenantID, personID)
	if err != nil {
		return person, nil, 0, err
	}
	byCode := map[string][]domain.TrendPoint{}
	unlinked := 0
	for _, o := range obs {
		m, ok := c.Catalog.Metric(o.MetricCode)
		if !ok {
			unlinked++
			continue
		}
		byCode[m.Code] = append(byCode[m.Code], domain.TrendPointOf(m, o))
	}
	if bmi := domain.ComputeBMI(byCode["weight"], byCode["height"], byCode["bmi"], person.BirthDate); len(bmi) > 0 {
		byCode["bmi"] = append(byCode["bmi"], bmi...)
		domain.SortPoints(byCode["bmi"])
	}
	return person, byCode, unlinked, nil
}

// PersonMetrics returns a person's metrics overview (HW-MET-4).
func (c *Clinical) PersonMetrics(ctx context.Context, sc domain.Scope, personID string) (MetricsOverview, error) {
	person, byCode, unlinked, err := c.personPoints(ctx, sc, personID)
	if err != nil {
		return MetricsOverview{}, err
	}
	out := MetricsOverview{Person: person, Groups: []domain.OverviewGroup{}, Unlinked: unlinked}
	for _, m := range c.Catalog.Metrics() {
		item, ok := domain.OverviewOf(m, byCode[m.Code])
		if !ok {
			continue
		}
		if n := len(out.Groups); n > 0 && out.Groups[n-1].Category == m.Category {
			out.Groups[n-1].Items = append(out.Groups[n-1].Items, item)
		} else {
			out.Groups = append(out.Groups, domain.OverviewGroup{Category: m.Category, Items: []domain.OverviewItem{item}})
		}
	}
	return out, nil
}

// PersonMetric returns every value of one metric for a person, oldest
// first (HW-MET-3).
func (c *Clinical) PersonMetric(ctx context.Context, sc domain.Scope, personID, code string) (MetricTrend, error) {
	m, ok := c.Catalog.Metric(code)
	if !ok {
		return MetricTrend{}, domain.NotFound("metric")
	}
	person, byCode, _, err := c.personPoints(ctx, sc, personID)
	if err != nil {
		return MetricTrend{}, err
	}
	points := byCode[code]
	if points == nil {
		points = []domain.TrendPoint{}
	}
	return MetricTrend{Person: person, Metric: m, Points: points}, nil
}

// AddMeasurements stores vitals typed in for a person without a paper; they
// are approved at once and edited or deleted like any observation (HW-MET-2).
func (c *Clinical) AddMeasurements(ctx context.Context, sc domain.Scope, personID string, input domain.MeasurementInput) ([]domain.Observation, error) {
	if err := sc.RequireEdit(); err != nil {
		return nil, err
	}
	in, err := input.Validate(c.today(), c.Catalog)
	if err != nil {
		return nil, err
	}
	now := c.Clock()
	ids := make([]string, 0, len(in.Values))
	err = c.Store.InTx(ctx, func(r port.Repos) error {
		if _, err := r.Person(ctx, sc.TenantID, personID); err != nil {
			return err
		}
		for _, v := range in.Values {
			m, _ := c.Catalog.Metric(v.MetricCode)
			o := domain.Observation{ID: NewID(), Status: domain.StatusApproved, PersonID: personID, Analyte: m.Name, MetricCode: m.Code,
				Value: v.Value, ValueNum: domain.ParseNumber(v.Value), Unit: v.Unit, SampleDate: in.Date, Confidence: 100,
				Provenance: manualProvenance(sc, now), CreatedAt: now, UpdatedAt: now}
			if err := r.CreateObservation(ctx, sc.TenantID, o); err != nil {
				return err
			}
			ids = append(ids, o.ID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Observation, 0, len(ids))
	for _, id := range ids {
		o, err := c.Store.Observation(ctx, sc.TenantID, id)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, nil
}
