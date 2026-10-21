package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"apps/health-watcher/domain"
)

func (r repos) Settings(ctx context.Context) (map[string]string, domain.SettingsStamp, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT s.key, s.value, s.updated_at, COALESCE(s.updated_by, ''), COALESCE(u.display_name, '')
		FROM settings s LEFT JOIN users u ON u.id = s.updated_by ORDER BY s.updated_at`)
	if err != nil {
		return nil, domain.SettingsStamp{}, fmt.Errorf("read settings: %w", err)
	}
	defer rows.Close()
	values := map[string]string{}
	var stamp domain.SettingsStamp
	for rows.Next() {
		var k, v, at string
		if err := rows.Scan(&k, &v, &at, &stamp.UpdatedBy, &stamp.UpdatedByName); err != nil {
			return nil, stamp, fmt.Errorf("scan setting: %w", err)
		}
		values[k] = v
		t, err := parseTime(at)
		if err != nil {
			return nil, stamp, err
		}
		stamp.UpdatedAt = &t
	}
	if err := rows.Err(); err != nil {
		return nil, stamp, fmt.Errorf("read settings: %w", err)
	}
	return values, stamp, nil
}

func (r repos) SaveSettings(ctx context.Context, values map[string]string, at time.Time, by string) error {
	for k, v := range values {
		res, err := r.q.ExecContext(ctx, `INSERT INTO settings (key, value, updated_at, updated_by) VALUES (?, ?, ?, ?)
			ON CONFLICT (key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at, updated_by = excluded.updated_by`,
			k, v, formatTime(at), sql.NullString{String: by, Valid: by != ""})
		if err := exec(res, err, "setting"); err != nil {
			return err
		}
	}
	return nil
}
