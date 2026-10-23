package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"apps/health-watcher/domain"
	"apps/health-watcher/port"
)

func (r repos) CustomMetrics(ctx context.Context, tenantID string) ([]domain.Metric, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT code, loinc, name, name_vi, category, unit, aliases_json, about_quote, about_title,
			about_url, about_source, about_fetched
		FROM custom_metrics WHERE tenant_id = ? ORDER BY created_at, code`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list custom metrics: %w", err)
	}
	defer rows.Close()
	out := []domain.Metric{}
	for rows.Next() {
		m := domain.Metric{Custom: true}
		var category, aliases string
		var a domain.MetricAbout
		if err := rows.Scan(&m.Code, &m.LOINC, &m.Name, &m.NameVI, &category, &m.Unit, &aliases, &a.Quote, &a.Title, &a.URL,
			&a.Source, &a.Fetched); err != nil {
			return nil, fmt.Errorf("scan custom metric: %w", err)
		}
		m.Category = domain.MetricCategory(category)
		if err := json.Unmarshal([]byte(aliases), &m.Aliases); err != nil {
			return nil, fmt.Errorf("decode custom metric aliases: %w", err)
		}
		if a.Quote != "" {
			m.About = &a
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r repos) CreateCustomMetric(ctx context.Context, tenantID string, m domain.Metric, by string, at time.Time) error {
	aliases, err := json.Marshal(nonNil(m.Aliases))
	if err != nil {
		return fmt.Errorf("encode custom metric aliases: %w", err)
	}
	var a domain.MetricAbout
	if m.About != nil {
		a = *m.About
	}
	res, err := r.q.ExecContext(ctx, `INSERT INTO custom_metrics (tenant_id, code, loinc, name, name_vi, category, unit, aliases_json,
			about_quote, about_title, about_url, about_source, about_fetched, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		tenantID, m.Code, m.LOINC, m.Name, m.NameVI, string(m.Category), m.Unit, string(aliases), a.Quote, a.Title, a.URL, a.Source,
		a.Fetched, nullable(by), formatTime(at))
	return exec(res, err, "custom metric")
}

func (r repos) UnlinkedObservations(ctx context.Context, tenantID string) ([]domain.Observation, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT `+observationColumns+`
		FROM observations o LEFT JOIN pages pg ON pg.id = o.page_id
		WHERE o.tenant_id = ? AND o.status = 'approved' AND o.metric_code = ''
		ORDER BY o.sample_date, o.created_at, o.position, o.id`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list unlinked observations: %w", err)
	}
	defer rows.Close()
	out := []domain.Observation{}
	for rows.Next() {
		o, err := scanObservation(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scan unlinked observation: %w", err)
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (r repos) TenantsWithUnlinked(ctx context.Context) ([]string, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT DISTINCT tenant_id FROM observations WHERE status = 'approved' AND metric_code = ''
		ORDER BY tenant_id`)
	if err != nil {
		return nil, fmt.Errorf("list tenants with unlinked values: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan tenant: %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (r repos) SuggestionKeys(ctx context.Context, tenantID string) (map[string]bool, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT analyte_key FROM metric_suggestions WHERE tenant_id = ?`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list suggestion names: %w", err)
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, fmt.Errorf("scan suggestion name: %w", err)
		}
		out[k] = true
	}
	return out, rows.Err()
}

func (r repos) CreateSuggestion(ctx context.Context, tenantID string, s domain.MetricSuggestion) error {
	proposal := ""
	if s.Proposal != nil {
		raw, err := json.Marshal(s.Proposal)
		if err != nil {
			return fmt.Errorf("encode metric proposal: %w", err)
		}
		proposal = string(raw)
	}
	res, err := r.q.ExecContext(ctx, `INSERT INTO metric_suggestions (id, tenant_id, analyte_key, analyte, unit, kind, metric_code,
			proposal_json, reason, model, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, tenantID, s.AnalyteKey, s.Analyte, s.Unit, string(s.Kind), s.MetricCode, proposal, s.Reason, s.Model, string(s.Status),
		formatTime(s.CreatedAt))
	return exec(res, err, "metric suggestion")
}

const suggestionColumns = `id, analyte_key, analyte, unit, kind, metric_code, proposal_json, reason, model, status, created_at,
	COALESCE(decided_by, ''), decided_at`

func scanSuggestion(scan func(...any) error) (domain.MetricSuggestion, error) {
	var s domain.MetricSuggestion
	var kind, status, proposal, created, decided string
	if err := scan(&s.ID, &s.AnalyteKey, &s.Analyte, &s.Unit, &kind, &s.MetricCode, &proposal, &s.Reason, &s.Model, &status, &created,
		&s.DecidedBy, &decided); err != nil {
		return s, err
	}
	s.Kind, s.Status = domain.SuggestionKind(kind), domain.SuggestionStatus(status)
	if proposal != "" {
		s.Proposal = &domain.MetricProposal{}
		if err := json.Unmarshal([]byte(proposal), s.Proposal); err != nil {
			return s, fmt.Errorf("decode metric proposal: %w", err)
		}
	}
	var err error
	if s.CreatedAt, err = parseTime(created); err != nil {
		return s, err
	}
	s.DecidedAt, err = optionalTime(decided)
	return s, err
}

func (r repos) Suggestions(ctx context.Context, tenantID string, status domain.SuggestionStatus) ([]domain.MetricSuggestion, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT `+suggestionColumns+` FROM metric_suggestions WHERE tenant_id = ? AND status = ?
		ORDER BY created_at, id`, tenantID, string(status))
	if err != nil {
		return nil, fmt.Errorf("list metric suggestions: %w", err)
	}
	defer rows.Close()
	out := []domain.MetricSuggestion{}
	for rows.Next() {
		s, err := scanSuggestion(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scan metric suggestion: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r repos) Suggestion(ctx context.Context, tenantID, id string) (domain.MetricSuggestion, error) {
	s, err := scanSuggestion(r.q.QueryRowContext(ctx, `SELECT `+suggestionColumns+` FROM metric_suggestions WHERE tenant_id = ? AND id = ?`,
		tenantID, id).Scan)
	return s, one(err, "suggestion")
}

func (r repos) DecideSuggestion(ctx context.Context, tenantID, id string, status domain.SuggestionStatus, by string, at time.Time) error {
	res, err := r.q.ExecContext(ctx, `UPDATE metric_suggestions SET status = ?, decided_by = ?, decided_at = ? WHERE tenant_id = ? AND id = ?`,
		string(status), nullable(by), formatTime(at), tenantID, id)
	return affected(res, err, "suggestion")
}

func (r repos) AllAnalyteLinks(ctx context.Context) ([]port.AnalyteLink, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT tenant_id, provider_id, analyte_key, metric_code FROM analyte_links
		ORDER BY tenant_id, provider_id, analyte_key`)
	if err != nil {
		return nil, fmt.Errorf("list analyte links: %w", err)
	}
	defer rows.Close()
	var out []port.AnalyteLink
	for rows.Next() {
		var l port.AnalyteLink
		if err := rows.Scan(&l.TenantID, &l.ProviderID, &l.Key, &l.MetricCode); err != nil {
			return nil, fmt.Errorf("scan analyte link: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r repos) DeleteAnalyteLink(ctx context.Context, tenantID, providerID, analyteKey string) error {
	res, err := r.q.ExecContext(ctx, `DELETE FROM analyte_links WHERE tenant_id = ? AND provider_id = ? AND analyte_key = ?`,
		tenantID, providerID, analyteKey)
	return affected(res, err, "analyte link")
}

func (r repos) LinkedAnalytes(ctx context.Context, tenantID, metricCode string) ([]string, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT DISTINCT analyte FROM observations WHERE tenant_id = ? AND metric_code = ? ORDER BY analyte`,
		tenantID, metricCode)
	if err != nil {
		return nil, fmt.Errorf("list linked names: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err != nil {
			return nil, fmt.Errorf("scan linked name: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
