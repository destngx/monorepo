package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"apps/health-watcher/domain"
)

func nullFloat(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

func floatPtr(v sql.NullFloat64) *float64 {
	if !v.Valid {
		return nil
	}
	f := v.Float64
	return &f
}

const observationSelect = `SELECT o.id, o.status, COALESCE(o.item_id, ''), COALESCE(o.document_id, ''), COALESCE(o.person_id, ''),
	COALESCE(o.page_id, ''), COALESCE(pg.number, 0), o.analyte, o.metric_code, o.value, o.value_num, o.unit, o.ref_text, o.ref_low,
	o.ref_high, o.flag, o.sample_date, o.confidence, o.created_at, o.updated_at
	FROM observations o LEFT JOIN pages pg ON pg.id = o.page_id`

func scanObservation(scan func(...any) error) (domain.Observation, error) {
	var o domain.Observation
	var status, created, updated string
	var num, low, high sql.NullFloat64
	if err := scan(&o.ID, &status, &o.ItemID, &o.DocumentID, &o.PersonID, &o.PageID, &o.PageNumber, &o.Analyte, &o.MetricCode,
		&o.Value, &num, &o.Unit, &o.RefText, &low, &high, &o.Flag, &o.SampleDate, &o.Confidence, &created, &updated); err != nil {
		return o, err
	}
	o.Status = domain.RecordStatus(status)
	o.ValueNum, o.RefLow, o.RefHigh = floatPtr(num), floatPtr(low), floatPtr(high)
	var err error
	if o.CreatedAt, err = parseTime(created); err != nil {
		return o, err
	}
	o.UpdatedAt, err = parseTime(updated)
	return o, err
}

func (r repos) observations(ctx context.Context, where string, args ...any) ([]domain.Observation, error) {
	rows, err := r.q.QueryContext(ctx, observationSelect+` WHERE `+where+` ORDER BY o.position, o.created_at, o.id`, args...)
	if err != nil {
		return nil, fmt.Errorf("list observations: %w", err)
	}
	defer rows.Close()
	out := []domain.Observation{}
	for rows.Next() {
		o, err := scanObservation(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scan observation: %w", err)
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (r repos) CreateObservation(ctx context.Context, tenantID string, o domain.Observation) error {
	res, err := r.q.ExecContext(ctx, `INSERT INTO observations (id, tenant_id, status, item_id, document_id, person_id, page_id, position,
			analyte, metric_code, value, value_num, unit, ref_text, ref_low, ref_high, flag, sample_date, confidence, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, (SELECT COUNT(*) FROM observations WHERE tenant_id = ? AND (item_id = ? OR document_id = ?)),
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		o.ID, tenantID, string(o.Status), nullable(o.ItemID), nullable(o.DocumentID), nullable(o.PersonID), nullable(o.PageID),
		tenantID, o.ItemID, o.DocumentID,
		o.Analyte, o.MetricCode, o.Value, nullFloat(o.ValueNum), o.Unit, o.RefText, nullFloat(o.RefLow), nullFloat(o.RefHigh), o.Flag,
		o.SampleDate, o.Confidence, formatTime(o.CreatedAt), formatTime(o.UpdatedAt))
	return exec(res, err, "observation")
}

func (r repos) UpdateObservation(ctx context.Context, tenantID string, o domain.Observation) error {
	res, err := r.q.ExecContext(ctx, `UPDATE observations SET status = ?, page_id = ?, analyte = ?, metric_code = ?, value = ?, value_num = ?,
			unit = ?, ref_text = ?, ref_low = ?, ref_high = ?, flag = ?, sample_date = ?, confidence = ?, updated_at = ?
		WHERE tenant_id = ? AND id = ?`,
		string(o.Status), nullable(o.PageID), o.Analyte, o.MetricCode, o.Value, nullFloat(o.ValueNum), o.Unit, o.RefText,
		nullFloat(o.RefLow), nullFloat(o.RefHigh), o.Flag, o.SampleDate, o.Confidence, formatTime(o.UpdatedAt), tenantID, o.ID)
	return affected(res, err, "observation")
}

func (r repos) Observation(ctx context.Context, tenantID, id string) (domain.Observation, error) {
	o, err := scanObservation(r.q.QueryRowContext(ctx, observationSelect+` WHERE o.tenant_id = ? AND o.id = ?`, tenantID, id).Scan)
	return o, one(err, "observation")
}

func (r repos) ObservationsOfItem(ctx context.Context, tenantID, itemID string) ([]domain.Observation, error) {
	return r.observations(ctx, `o.tenant_id = ? AND o.item_id = ? AND o.status <> 'approved'`, tenantID, itemID)
}

func (r repos) ObservationsOfDocument(ctx context.Context, tenantID, documentID string) ([]domain.Observation, error) {
	return r.observations(ctx, `o.tenant_id = ? AND o.document_id = ? AND o.status = 'approved'`, tenantID, documentID)
}

func (r repos) DeleteObservation(ctx context.Context, tenantID, id string) error {
	res, err := r.q.ExecContext(ctx, `DELETE FROM observations WHERE tenant_id = ? AND id = ?`, tenantID, id)
	return affected(res, err, "observation")
}

const findingSelect = `SELECT f.id, f.status, COALESCE(f.item_id, ''), COALESCE(f.document_id, ''), COALESCE(f.person_id, ''),
	COALESCE(f.page_id, ''), COALESCE(pg.number, 0), f.kind, f.section, f.text, f.icd10, f.drug, f.dose, f.frequency, f.duration, f.confidence,
	f.created_at, f.updated_at
	FROM findings f LEFT JOIN pages pg ON pg.id = f.page_id`

func scanFinding(scan func(...any) error) (domain.Finding, error) {
	var f domain.Finding
	var status, kind, created, updated string
	if err := scan(&f.ID, &status, &f.ItemID, &f.DocumentID, &f.PersonID, &f.PageID, &f.PageNumber, &kind, &f.Section, &f.Text, &f.ICD10, &f.Drug,
		&f.Dose, &f.Frequency, &f.Duration, &f.Confidence, &created, &updated); err != nil {
		return f, err
	}
	f.Status, f.Kind = domain.RecordStatus(status), domain.FindingKind(kind)
	var err error
	if f.CreatedAt, err = parseTime(created); err != nil {
		return f, err
	}
	f.UpdatedAt, err = parseTime(updated)
	return f, err
}

func (r repos) findings(ctx context.Context, where string, args ...any) ([]domain.Finding, error) {
	rows, err := r.q.QueryContext(ctx, findingSelect+` WHERE `+where+` ORDER BY f.position, f.created_at, f.id`, args...)
	if err != nil {
		return nil, fmt.Errorf("list findings: %w", err)
	}
	defer rows.Close()
	out := []domain.Finding{}
	for rows.Next() {
		f, err := scanFinding(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scan finding: %w", err)
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r repos) CreateFinding(ctx context.Context, tenantID string, f domain.Finding) error {
	res, err := r.q.ExecContext(ctx, `INSERT INTO findings (id, tenant_id, status, item_id, document_id, person_id, page_id, position, kind,
			section, text, icd10, drug, dose, frequency, duration, confidence, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, (SELECT COUNT(*) FROM findings WHERE tenant_id = ? AND (item_id = ? OR document_id = ?)),
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		f.ID, tenantID, string(f.Status), nullable(f.ItemID), nullable(f.DocumentID), nullable(f.PersonID), nullable(f.PageID),
		tenantID, f.ItemID, f.DocumentID,
		string(f.Kind), f.Section, f.Text, f.ICD10, f.Drug, f.Dose, f.Frequency, f.Duration, f.Confidence, formatTime(f.CreatedAt), formatTime(f.UpdatedAt))
	if err := exec(res, err, "finding"); err != nil {
		return err
	}
	return r.indexFinding(ctx, tenantID, f)
}

func (r repos) UpdateFinding(ctx context.Context, tenantID string, f domain.Finding) error {
	res, err := r.q.ExecContext(ctx, `UPDATE findings SET status = ?, page_id = ?, kind = ?, section = ?, text = ?, icd10 = ?, drug = ?, dose = ?,
			frequency = ?, duration = ?, confidence = ?, updated_at = ?
		WHERE tenant_id = ? AND id = ?`,
		string(f.Status), nullable(f.PageID), string(f.Kind), f.Section, f.Text, f.ICD10, f.Drug, f.Dose, f.Frequency, f.Duration, f.Confidence,
		formatTime(f.UpdatedAt), tenantID, f.ID)
	if err := affected(res, err, "finding"); err != nil {
		return err
	}
	return r.indexFinding(ctx, tenantID, f)
}

func (r repos) Finding(ctx context.Context, tenantID, id string) (domain.Finding, error) {
	f, err := scanFinding(r.q.QueryRowContext(ctx, findingSelect+` WHERE f.tenant_id = ? AND f.id = ?`, tenantID, id).Scan)
	return f, one(err, "finding")
}

func (r repos) FindingsOfItem(ctx context.Context, tenantID, itemID string) ([]domain.Finding, error) {
	return r.findings(ctx, `f.tenant_id = ? AND f.item_id = ? AND f.status <> 'approved'`, tenantID, itemID)
}

func (r repos) FindingsOfDocument(ctx context.Context, tenantID, documentID string) ([]domain.Finding, error) {
	return r.findings(ctx, `f.tenant_id = ? AND f.document_id = ? AND f.status = 'approved'`, tenantID, documentID)
}

func (r repos) DeleteFinding(ctx context.Context, tenantID, id string) error {
	res, err := r.q.ExecContext(ctx, `DELETE FROM findings WHERE tenant_id = ? AND id = ?`, tenantID, id)
	if err := affected(res, err, "finding"); err != nil {
		return err
	}
	return r.unindex(ctx, domain.SearchFinding, id)
}

func (r repos) DeleteItemDrafts(ctx context.Context, tenantID, itemID string) error {
	for _, table := range []string{"observations", "findings"} {
		if _, err := r.q.ExecContext(ctx, `DELETE FROM `+table+` WHERE tenant_id = ? AND item_id = ? AND status <> 'approved'`,
			tenantID, itemID); err != nil {
			return fmt.Errorf("delete drafts: %w", err)
		}
	}
	return nil
}

func (r repos) ApproveItemDrafts(ctx context.Context, tenantID, itemID, documentID, personID, sampleDate string, at time.Time) error {
	if _, err := r.q.ExecContext(ctx, `UPDATE observations SET status = 'approved', document_id = ?, person_id = ?,
			sample_date = CASE WHEN sample_date = '' THEN ? ELSE sample_date END, updated_at = ?
		WHERE tenant_id = ? AND item_id = ? AND status = 'draft'`, documentID, personID, sampleDate, formatTime(at), tenantID, itemID); err != nil {
		return fmt.Errorf("approve observations: %w", err)
	}
	if _, err := r.q.ExecContext(ctx, `UPDATE findings SET status = 'approved', document_id = ?, person_id = ?, updated_at = ?
		WHERE tenant_id = ? AND item_id = ? AND status = 'draft'`, documentID, personID, formatTime(at), tenantID, itemID); err != nil {
		return fmt.Errorf("approve findings: %w", err)
	}
	if err := r.DeleteItemDrafts(ctx, tenantID, itemID); err != nil {
		return err
	}
	approved, err := r.FindingsOfDocument(ctx, tenantID, documentID)
	if err != nil {
		return err
	}
	for _, f := range approved {
		if err := r.indexFinding(ctx, tenantID, f); err != nil {
			return err
		}
	}
	return nil
}

func (r repos) AddAudit(ctx context.Context, tenantID string, e domain.AuditEntry) error {
	res, err := r.q.ExecContext(ctx, `INSERT INTO audit_log (id, tenant_id, entity, entity_id, action, user_id, at, old_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, e.ID, tenantID, string(e.Entity), e.EntityID, e.Action, nullable(e.UserID), formatTime(e.At), e.OldJSON)
	return exec(res, err, "audit entry")
}

func (r repos) AuditOf(ctx context.Context, tenantID string, entity domain.AuditEntity, entityID string) ([]domain.AuditEntry, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT id, entity, entity_id, action, COALESCE(user_id, ''), at, old_json FROM audit_log
		WHERE tenant_id = ? AND entity = ? AND entity_id = ? ORDER BY at, id`, tenantID, string(entity), entityID)
	if err != nil {
		return nil, fmt.Errorf("list audit: %w", err)
	}
	defer rows.Close()
	out := []domain.AuditEntry{}
	for rows.Next() {
		var e domain.AuditEntry
		var ent, at string
		if err := rows.Scan(&e.ID, &ent, &e.EntityID, &e.Action, &e.UserID, &at, &e.OldJSON); err != nil {
			return nil, fmt.Errorf("scan audit: %w", err)
		}
		e.Entity = domain.AuditEntity(ent)
		if e.At, err = parseTime(at); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r repos) SetAnalyteLink(ctx context.Context, tenantID, providerID, analyteKey, metricCode string) error {
	res, err := r.q.ExecContext(ctx, `INSERT INTO analyte_links (tenant_id, provider_id, analyte_key, metric_code) VALUES (?, ?, ?, ?)
		ON CONFLICT (tenant_id, provider_id, analyte_key) DO UPDATE SET metric_code = excluded.metric_code`,
		tenantID, providerID, analyteKey, metricCode)
	return exec(res, err, "analyte link")
}

func (r repos) AnalyteLinks(ctx context.Context, tenantID, providerID string) (map[string]string, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT analyte_key, metric_code FROM analyte_links WHERE tenant_id = ? AND provider_id = ?`,
		tenantID, providerID)
	if err != nil {
		return nil, fmt.Errorf("list analyte links: %w", err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, fmt.Errorf("scan analyte link: %w", err)
		}
		out[k] = v
	}
	return out, rows.Err()
}
