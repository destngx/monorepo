package service

import (
	"context"
	"slices"
	"strings"

	"apps/health-watcher/domain"
	"apps/health-watcher/port"
)

// MetricsOverview is a person's metrics dashboard (HW-MET-4): each metric
// with values, grouped by category in catalog order, then the metrics made
// from names the catalog doesn't know (HW-MET-7), and how many approved
// results those hold.
type MetricsOverview struct {
	Person   domain.Person          `json:"person"`
	Groups   []domain.OverviewGroup `json:"groups"`
	Unlinked int                    `json:"unlinked"`
}

// MetricTrend is one metric of a person over time (HW-MET-3), with what the
// metric is when a trusted source says so (HW-MET-6).
type MetricTrend struct {
	Person domain.Person       `json:"person"`
	Metric domain.Metric       `json:"metric"`
	About  *domain.MetricAbout `json:"about,omitempty"`
	Points []domain.TrendPoint `json:"points"`
}

// personPoints is a person with their approved observations as trend
// points by metric code. BMI is computed from weight and height, and every
// value the catalog doesn't know joins a metric of its printed name, so no
// value is left out (HW-MET-7).
type personPoints struct {
	person domain.Person
	byCode map[string][]domain.TrendPoint
	// printed are the metrics made from printed names, by code.
	printed  map[string]domain.Metric
	unlinked int
}

func (c *Clinical) personPoints(ctx context.Context, sc domain.Scope, personID string) (personPoints, error) {
	if err := sc.RequireRead(); err != nil {
		return personPoints{}, err
	}
	person, err := c.Store.Person(ctx, sc.TenantID, personID)
	if err != nil {
		return personPoints{}, err
	}
	obs, err := c.Store.ObservationsOfPerson(ctx, sc.TenantID, personID)
	if err != nil {
		return personPoints{}, err
	}
	pp := personPoints{person: person, byCode: map[string][]domain.TrendPoint{}, printed: map[string]domain.Metric{}}
	unknown := map[string][]domain.PersonObservation{}
	for _, o := range obs {
		m, ok := c.Catalog.Metric(o.MetricCode)
		if !ok {
			code := domain.PrintedMetricCode(o.Analyte)
			unknown[code] = append(unknown[code], o)
			pp.unlinked++
			continue
		}
		pp.byCode[m.Code] = append(pp.byCode[m.Code], domain.TrendPointOf(m, o))
	}
	for code, os := range unknown {
		m := domain.PrintedMetric(code, os)
		pp.printed[code] = m
		for _, o := range os {
			pp.byCode[code] = append(pp.byCode[code], domain.TrendPointOf(m, o))
		}
	}
	if bmi := domain.ComputeBMI(pp.byCode["weight"], pp.byCode["height"], pp.byCode["bmi"], person.BirthDate); len(bmi) > 0 {
		pp.byCode["bmi"] = append(pp.byCode["bmi"], bmi...)
		domain.SortPoints(pp.byCode["bmi"])
	}
	return pp, nil
}

// PersonMetrics returns a person's metrics overview (HW-MET-4).
func (c *Clinical) PersonMetrics(ctx context.Context, sc domain.Scope, personID string) (MetricsOverview, error) {
	pp, err := c.personPoints(ctx, sc, personID)
	if err != nil {
		return MetricsOverview{}, err
	}
	out := MetricsOverview{Person: pp.person, Groups: []domain.OverviewGroup{}, Unlinked: pp.unlinked}
	printed := make([]domain.Metric, 0, len(pp.printed))
	for _, m := range pp.printed {
		printed = append(printed, m)
	}
	slices.SortFunc(printed, func(a, b domain.Metric) int { return strings.Compare(domain.FoldKey(a.Name), domain.FoldKey(b.Name)) })
	for _, m := range append(slices.Clone(c.Catalog.Metrics()), printed...) {
		item, ok := domain.OverviewOf(m, pp.byCode[m.Code])
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
// first (HW-MET-3); the metric may be one made from a printed name.
func (c *Clinical) PersonMetric(ctx context.Context, sc domain.Scope, personID, code string) (MetricTrend, error) {
	m, ok := c.Catalog.Metric(code)
	pp, err := c.personPoints(ctx, sc, personID)
	if err != nil {
		return MetricTrend{}, err
	}
	if !ok {
		if m, ok = pp.printed[code]; !ok {
			return MetricTrend{}, domain.NotFound("metric")
		}
	}
	person := pp.person
	points := pp.byCode[code]
	if points == nil {
		points = []domain.TrendPoint{}
	}
	trend := MetricTrend{Person: person, Metric: m, Points: points}
	if about, ok := domain.AboutMetric(code); ok {
		trend.About = &about
	}
	return trend, nil
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
