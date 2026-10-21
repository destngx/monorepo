package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"apps/health-watcher/domain"
)

const jobColumns = `id, tenant_id, kind, subject_id, status, attempts, next_attempt_at, last_error`

func scanJob(scan func(...any) error) (domain.Job, error) {
	var j domain.Job
	var kind, status, next string
	if err := scan(&j.ID, &j.TenantID, &kind, &j.SubjectID, &status, &j.Attempts, &next, &j.LastError); err != nil {
		return j, err
	}
	j.Kind, j.Status = domain.JobKind(kind), domain.JobStatus(status)
	var err error
	j.NextAttemptAt, err = parseTime(next)
	return j, err
}

func (r repos) EnqueueJob(ctx context.Context, j domain.Job, at time.Time) error {
	res, err := r.q.ExecContext(ctx, `INSERT INTO jobs (id, tenant_id, kind, subject_id, status, attempts, next_attempt_at, last_error,
			created_at, updated_at)
		VALUES (?, ?, ?, ?, 'queued', 0, ?, '', ?, ?)
		ON CONFLICT (kind, subject_id) DO UPDATE SET status = 'queued', attempts = 0, next_attempt_at = excluded.next_attempt_at,
			last_error = '', updated_at = excluded.updated_at
		WHERE jobs.status IN ('done', 'failed')`,
		j.ID, j.TenantID, string(j.Kind), j.SubjectID, formatTime(at), formatTime(at), formatTime(at))
	return exec(res, err, "job")
}

func (r repos) ClaimJob(ctx context.Context, now time.Time) (domain.Job, error) {
	j, err := scanJob(r.q.QueryRowContext(ctx, `UPDATE jobs SET status = 'running', updated_at = ?
		WHERE id = (SELECT id FROM jobs WHERE status = 'queued' AND next_attempt_at <= ? ORDER BY next_attempt_at, created_at LIMIT 1)
		RETURNING `+jobColumns, formatTime(now), formatTime(now)).Scan)
	return j, one(err, "job")
}

func (r repos) CompleteJob(ctx context.Context, id string, at time.Time) error {
	res, err := r.q.ExecContext(ctx, `UPDATE jobs SET status = 'done', last_error = '', updated_at = ? WHERE id = ?`, formatTime(at), id)
	return affected(res, err, "job")
}

func (r repos) RetryJob(ctx context.Context, id string, attempts int, next time.Time, lastErr string, at time.Time) error {
	res, err := r.q.ExecContext(ctx, `UPDATE jobs SET status = 'queued', attempts = ?, next_attempt_at = ?, last_error = ?, updated_at = ?
		WHERE id = ?`, attempts, formatTime(next), lastErr, formatTime(at), id)
	return affected(res, err, "job")
}

func (r repos) FailJob(ctx context.Context, id string, attempts int, lastErr string, at time.Time) error {
	res, err := r.q.ExecContext(ctx, `UPDATE jobs SET status = 'failed', attempts = ?, last_error = ?, updated_at = ? WHERE id = ?`,
		attempts, lastErr, formatTime(at), id)
	return affected(res, err, "job")
}

func (r repos) JobFor(ctx context.Context, tenantID string, kind domain.JobKind, subjectID string) (domain.Job, error) {
	j, err := scanJob(r.q.QueryRowContext(ctx, `SELECT `+jobColumns+` FROM jobs WHERE tenant_id = ? AND kind = ? AND subject_id = ?`,
		tenantID, string(kind), subjectID).Scan)
	return j, one(err, "job")
}

func (r repos) RequeueRunningJobs(ctx context.Context, at time.Time) error {
	if _, err := r.q.ExecContext(ctx, `UPDATE jobs SET status = 'queued', updated_at = ? WHERE status = 'running'`, formatTime(at)); err != nil {
		return fmt.Errorf("requeue running jobs: %w", err)
	}
	return nil
}

func (r repos) SaveReading(ctx context.Context, tenantID string, rd domain.Reading) error {
	proposal := ""
	if rd.Proposal != nil {
		raw, err := json.Marshal(rd.Proposal)
		if err != nil {
			return fmt.Errorf("encode proposal: %w", err)
		}
		proposal = string(raw)
	}
	res, err := r.q.ExecContext(ctx, `INSERT INTO readings (item_id, tenant_id, status, reason, model, proposal_json, pages_sent, error, updated_at)
		SELECT id, tenant_id, ?, ?, ?, ?, ?, ?, ? FROM inbox_items WHERE tenant_id = ? AND id = ?
		ON CONFLICT (item_id) DO UPDATE SET status = excluded.status, reason = excluded.reason, model = excluded.model,
			proposal_json = excluded.proposal_json, pages_sent = excluded.pages_sent, error = excluded.error, updated_at = excluded.updated_at`,
		string(rd.Status), rd.Reason, rd.Model, proposal, rd.PagesSent, rd.Error, formatTime(rd.UpdatedAt), tenantID, rd.ItemID)
	return affected(res, err, "inbox item")
}

func (r repos) Reading(ctx context.Context, tenantID, itemID string) (domain.Reading, error) {
	var rd domain.Reading
	var status, proposal, updated string
	var jobStatus, next sql.NullString
	var attempts sql.NullInt64
	err := r.q.QueryRowContext(ctx, `SELECT r.item_id, r.status, r.reason, r.model, r.proposal_json, r.pages_sent, r.error, r.updated_at,
			j.status, j.attempts, j.next_attempt_at
		FROM readings r LEFT JOIN jobs j ON j.kind = 'read' AND j.subject_id = r.item_id
		WHERE r.tenant_id = ? AND r.item_id = ?`, tenantID, itemID).
		Scan(&rd.ItemID, &status, &rd.Reason, &rd.Model, &proposal, &rd.PagesSent, &rd.Error, &updated, &jobStatus, &attempts, &next)
	if err := one(err, "reading"); err != nil {
		return rd, err
	}
	rd.Status = domain.ReadingStatus(status)
	if proposal != "" {
		rd.Proposal = &domain.Proposal{}
		if err := json.Unmarshal([]byte(proposal), rd.Proposal); err != nil {
			return rd, fmt.Errorf("decode proposal: %w", err)
		}
	}
	if rd.UpdatedAt, err = parseTime(updated); err != nil {
		return rd, err
	}
	rd.Attempts = int(attempts.Int64)
	if jobStatus.String == string(domain.JobQueued) && rd.Attempts > 0 && next.Valid {
		t, err := parseTime(next.String)
		if err != nil {
			return rd, err
		}
		rd.NextAttemptAt = &t
	}
	return rd, nil
}
