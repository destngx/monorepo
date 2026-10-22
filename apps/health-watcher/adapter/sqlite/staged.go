package sqlite

import (
	"context"
	"fmt"
	"time"

	"apps/health-watcher/port"
)

func (r repos) StageFile(ctx context.Context, tenantID, userID, fileID string, at time.Time) error {
	if _, err := r.q.ExecContext(ctx, `INSERT INTO staged_files (file_id, tenant_id, user_id, created_at)
		SELECT id, tenant_id, ?, ? FROM files WHERE tenant_id = ? AND id = ?
		ON CONFLICT (file_id) DO NOTHING`, userID, formatTime(at), tenantID, fileID); err != nil {
		return fmt.Errorf("stage file: %w", err)
	}
	return nil
}

func (r repos) StagedBy(ctx context.Context, tenantID, userID, fileID string) (bool, error) {
	var staged bool
	err := r.q.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM staged_files WHERE tenant_id = ? AND user_id = ? AND file_id = ?)`,
		tenantID, userID, fileID).Scan(&staged)
	if err != nil {
		return false, fmt.Errorf("check staged file: %w", err)
	}
	return staged, nil
}

func (r repos) UnstageFiles(ctx context.Context, tenantID string, fileIDs []string) error {
	for _, id := range fileIDs {
		if _, err := r.q.ExecContext(ctx, `DELETE FROM staged_files WHERE tenant_id = ? AND file_id = ?`, tenantID, id); err != nil {
			return fmt.Errorf("unstage file: %w", err)
		}
	}
	return nil
}

func (r repos) StagedBefore(ctx context.Context, before time.Time) ([]port.TenantFile, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT tenant_id, file_id FROM staged_files WHERE created_at < ? ORDER BY created_at`,
		formatTime(before))
	if err != nil {
		return nil, fmt.Errorf("list staged files: %w", err)
	}
	defer rows.Close()
	var out []port.TenantFile
	for rows.Next() {
		var tf port.TenantFile
		if err := rows.Scan(&tf.TenantID, &tf.FileID); err != nil {
			return nil, fmt.Errorf("scan staged file: %w", err)
		}
		out = append(out, tf)
	}
	return out, rows.Err()
}
