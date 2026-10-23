package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"apps/health-watcher/domain"
	"apps/health-watcher/port"
)

func nullFloat(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

// version stores a record version; 0 means a new record, version 1.
func version(v int) int {
	if v < 1 {
		return 1
	}
	return v
}

func floatPtr(v sql.NullFloat64) *float64 {
	if !v.Valid {
		return nil
	}
	f := v.Float64
	return &f
}

// observationColumns are the columns scanObservation reads, in order.
const observationColumns = `o.id, o.status, COALESCE(o.item_id, ''), COALESCE(o.document_id, ''), COALESCE(o.person_id, ''),
	COALESCE(o.page_id, ''), COALESCE(pg.number, 0), o.analyte, o.metric_code, o.value, o.value_num, o.unit, o.ref_text, o.ref_low,
	o.ref_high, o.flag, o.sample_date, o.confidence, o.version, o.source, o.source_detail, COALESCE(o.approved_by, ''), o.approved_at,
	o.created_at, o.updated_at`

const observationSelect = `SELECT ` + observationColumns + ` FROM observations o LEFT JOIN pages pg ON pg.id = o.page_id`

func scanObservation(scan func(...any) error) (domain.Observation, error) {
	var o domain.Observation
	var status, source, approvedAt, created, updated string
	var num, low, high sql.NullFloat64
	if err := scan(&o.ID, &status, &o.ItemID, &o.DocumentID, &o.PersonID, &o.PageID, &o.PageNumber, &o.Analyte, &o.MetricCode,
		&o.Value, &num, &o.Unit, &o.RefText, &low, &high, &o.Flag, &o.SampleDate, &o.Confidence, &o.Version, &source, &o.SourceDetail,
		&o.ApprovedBy, &approvedAt, &created, &updated); err != nil {
		return o, err
	}
	o.Status, o.Source = domain.RecordStatus(status), domain.RecordSource(source)
	o.ValueNum, o.RefLow, o.RefHigh = floatPtr(num), floatPtr(low), floatPtr(high)
	var err error
	if o.ApprovedAt, err = optionalTime(approvedAt); err != nil {
		return o, err
	}
	o.ExportGaps = domain.RecordExportGaps(o.Status, o.Provenance)
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
			analyte, metric_code, value, value_num, unit, ref_text, ref_low, ref_high, flag, sample_date, confidence,
			version, source, source_detail, approved_by, approved_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, (SELECT COUNT(*) FROM observations WHERE tenant_id = ? AND (item_id = ? OR document_id = ?)),
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		o.ID, tenantID, string(o.Status), nullable(o.ItemID), nullable(o.DocumentID), nullable(o.PersonID), nullable(o.PageID),
		tenantID, o.ItemID, o.DocumentID,
		o.Analyte, o.MetricCode, o.Value, nullFloat(o.ValueNum), o.Unit, o.RefText, nullFloat(o.RefLow), nullFloat(o.RefHigh), o.Flag,
		o.SampleDate, o.Confidence, version(o.Version), string(o.Source), o.SourceDetail, nullable(o.ApprovedBy), formatOptionalTime(o.ApprovedAt),
		formatTime(o.CreatedAt), formatTime(o.UpdatedAt))
	return exec(res, err, "observation")
}

func (r repos) UpdateObservation(ctx context.Context, tenantID string, o domain.Observation) error {
	res, err := r.q.ExecContext(ctx, `UPDATE observations SET status = ?, page_id = ?, analyte = ?, metric_code = ?, value = ?, value_num = ?,
			unit = ?, ref_text = ?, ref_low = ?, ref_high = ?, flag = ?, sample_date = ?, confidence = ?, version = ?, updated_at = ?
		WHERE tenant_id = ? AND id = ?`,
		string(o.Status), nullable(o.PageID), o.Analyte, o.MetricCode, o.Value, nullFloat(o.ValueNum), o.Unit, o.RefText,
		nullFloat(o.RefLow), nullFloat(o.RefHigh), o.Flag, o.SampleDate, o.Confidence, version(o.Version), formatTime(o.UpdatedAt), tenantID, o.ID)
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
	f.version, f.source, f.source_detail, COALESCE(f.approved_by, ''), f.approved_at, f.created_at, f.updated_at
	FROM findings f LEFT JOIN pages pg ON pg.id = f.page_id`

func scanFinding(scan func(...any) error) (domain.Finding, error) {
	var f domain.Finding
	var status, kind, source, approvedAt, created, updated string
	if err := scan(&f.ID, &status, &f.ItemID, &f.DocumentID, &f.PersonID, &f.PageID, &f.PageNumber, &kind, &f.Section, &f.Text, &f.ICD10, &f.Drug,
		&f.Dose, &f.Frequency, &f.Duration, &f.Confidence, &f.Version, &source, &f.SourceDetail, &f.ApprovedBy, &approvedAt,
		&created, &updated); err != nil {
		return f, err
	}
	f.Status, f.Kind, f.Source = domain.RecordStatus(status), domain.FindingKind(kind), domain.RecordSource(source)
	var err error
	if f.ApprovedAt, err = optionalTime(approvedAt); err != nil {
		return f, err
	}
	f.ExportGaps = domain.RecordExportGaps(f.Status, f.Provenance)
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
			section, text, icd10, drug, dose, frequency, duration, confidence, version, source, source_detail, approved_by, approved_at,
			created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, (SELECT COUNT(*) FROM findings WHERE tenant_id = ? AND (item_id = ? OR document_id = ?)),
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		f.ID, tenantID, string(f.Status), nullable(f.ItemID), nullable(f.DocumentID), nullable(f.PersonID), nullable(f.PageID),
		tenantID, f.ItemID, f.DocumentID,
		string(f.Kind), f.Section, f.Text, f.ICD10, f.Drug, f.Dose, f.Frequency, f.Duration, f.Confidence,
		version(f.Version), string(f.Source), f.SourceDetail, nullable(f.ApprovedBy), formatOptionalTime(f.ApprovedAt),
		formatTime(f.CreatedAt), formatTime(f.UpdatedAt))
	if err := exec(res, err, "finding"); err != nil {
		return err
	}
	return r.indexFinding(ctx, tenantID, f)
}

func (r repos) UpdateFinding(ctx context.Context, tenantID string, f domain.Finding) error {
	res, err := r.q.ExecContext(ctx, `UPDATE findings SET status = ?, page_id = ?, kind = ?, section = ?, text = ?, icd10 = ?, drug = ?, dose = ?,
			frequency = ?, duration = ?, confidence = ?, version = ?, updated_at = ?
		WHERE tenant_id = ? AND id = ?`,
		string(f.Status), nullable(f.PageID), string(f.Kind), f.Section, f.Text, f.ICD10, f.Drug, f.Dose, f.Frequency, f.Duration, f.Confidence,
		version(f.Version), formatTime(f.UpdatedAt), tenantID, f.ID)
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

func (r repos) ApproveItemDrafts(ctx context.Context, tenantID, itemID, documentID, personID, sampleDate, userID string, at time.Time) error {
	if _, err := r.q.ExecContext(ctx, `UPDATE observations SET status = 'approved', document_id = ?, person_id = ?,
			sample_date = CASE WHEN sample_date = '' THEN ? ELSE sample_date END, approved_by = ?, approved_at = ?, updated_at = ?
		WHERE tenant_id = ? AND item_id = ? AND status = 'draft'`, documentID, personID, sampleDate, nullable(userID), formatTime(at),
		formatTime(at), tenantID, itemID); err != nil {
		return fmt.Errorf("approve observations: %w", err)
	}
	if _, err := r.q.ExecContext(ctx, `UPDATE findings SET status = 'approved', document_id = ?, person_id = ?, approved_by = ?, approved_at = ?,
			updated_at = ?
		WHERE tenant_id = ? AND item_id = ? AND status = 'draft'`, documentID, personID, nullable(userID), formatTime(at), formatTime(at),
		tenantID, itemID); err != nil {
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

func (r repos) ConfirmApprovals(ctx context.Context, tenantID, documentID, userID string, at time.Time) (int, error) {
	total := 0
	for _, table := range []string{"observations", "findings"} {
		res, err := r.q.ExecContext(ctx, `UPDATE `+table+` SET approved_by = ?, approved_at = ?
			WHERE tenant_id = ? AND document_id = ? AND status = 'approved' AND approved_by IS NULL`,
			userID, formatTime(at), tenantID, documentID)
		if err != nil {
			return 0, fmt.Errorf("confirm approvals: %w", err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("confirm approvals: %w", err)
		}
		total += int(n)
	}
	return total, nil
}

func (r repos) AddAudit(ctx context.Context, tenantID string, e domain.AuditEntry) error {
	res, err := r.q.ExecContext(ctx, `INSERT INTO audit_log (id, tenant_id, entity, entity_id, action, version, user_id, at, old_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, e.ID, tenantID, string(e.Entity), e.EntityID, e.Action, version(e.Version), nullable(e.UserID),
		formatTime(e.At), e.OldJSON)
	return exec(res, err, "audit entry")
}

func (r repos) AuditOf(ctx context.Context, tenantID string, entity domain.AuditEntity, entityID string) ([]domain.AuditEntry, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT id, entity, entity_id, action, version, COALESCE(user_id, ''), at, old_json FROM audit_log
		WHERE tenant_id = ? AND entity = ? AND entity_id = ? ORDER BY at, id`, tenantID, string(entity), entityID)
	if err != nil {
		return nil, fmt.Errorf("list audit: %w", err)
	}
	defer rows.Close()
	out := []domain.AuditEntry{}
	for rows.Next() {
		var e domain.AuditEntry
		var ent, at string
		if err := rows.Scan(&e.ID, &ent, &e.EntityID, &e.Action, &e.Version, &e.UserID, &at, &e.OldJSON); err != nil {
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

func (r repos) ObservationsOfPerson(ctx context.Context, tenantID, personID string) ([]domain.PersonObservation, error) {
	// A page's number in its document counts the pages of the files
	// before its own, in the document's file order.
	rows, err := r.q.QueryContext(ctx, `SELECT `+observationColumns+`, COALESCE(d.visit_id, ''), COALESCE(pr.name, ''),
			COALESCE((SELECT COUNT(*) FROM document_files df JOIN document_files own ON own.document_id = df.document_id
				JOIN pages p2 ON p2.file_id = df.file_id
				WHERE df.document_id = o.document_id AND own.file_id = pg.file_id
					AND (df.position < own.position OR (df.position = own.position AND p2.number <= pg.number))), 0),
			COALESCE(pg.title, '')
		FROM observations o LEFT JOIN pages pg ON pg.id = o.page_id
			LEFT JOIN documents d ON d.id = o.document_id
			LEFT JOIN visits v ON v.id = d.visit_id
			LEFT JOIN providers pr ON pr.id = v.provider_id
		WHERE o.tenant_id = ? AND o.person_id = ? AND o.status = 'approved'
		ORDER BY o.sample_date, o.created_at, o.position, o.id`, tenantID, personID)
	if err != nil {
		return nil, fmt.Errorf("list person observations: %w", err)
	}
	defer rows.Close()
	out := []domain.PersonObservation{}
	for rows.Next() {
		var po domain.PersonObservation
		o, err := scanObservation(func(dest ...any) error {
			return rows.Scan(append(dest, &po.VisitID, &po.Provider, &po.PageNumber, &po.PageTitle)...)
		})
		if err != nil {
			return nil, fmt.Errorf("scan person observation: %w", err)
		}
		po.Observation = o
		out = append(out, po)
	}
	return out, rows.Err()
}

func (r repos) UneditedReadObservations(ctx context.Context) ([]port.TenantObservation, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT `+observationColumns+`, o.tenant_id, COALESCE(v.provider_id, '')
		FROM observations o LEFT JOIN pages pg ON pg.id = o.page_id
			JOIN documents d ON d.id = o.document_id
			JOIN visits v ON v.id = d.visit_id
		WHERE o.status = 'approved' AND o.source = 'reading' AND o.version = 1 AND o.confidence < 100
		ORDER BY o.tenant_id, o.created_at, o.id`)
	if err != nil {
		return nil, fmt.Errorf("list unedited observations: %w", err)
	}
	defer rows.Close()
	out := []port.TenantObservation{}
	for rows.Next() {
		var to port.TenantObservation
		o, err := scanObservation(func(dest ...any) error { return rows.Scan(append(dest, &to.TenantID, &to.ProviderID)...) })
		if err != nil {
			return nil, fmt.Errorf("scan unedited observation: %w", err)
		}
		to.Observation = o
		out = append(out, to)
	}
	return out, rows.Err()
}

func (r repos) CountTypedObservations(ctx context.Context, tenantID, personID string) (int, error) {
	var n int
	err := r.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM observations
		WHERE tenant_id = ? AND person_id = ? AND status = 'approved' AND document_id IS NULL`, tenantID, personID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count typed observations: %w", err)
	}
	return n, nil
}
