package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"apps/health-watcher/domain"
)

const pictureColumns = `pc.id, pc.page_id, pc.position, pc.kind, pc.caption, pc.box_json, pc.width, pc.height, pc.image_sha, pc.created_at`

func scanPicture(scan func(...any) error) (domain.Picture, error) {
	var p domain.Picture
	var kind, box, created string
	if err := scan(&p.ID, &p.PageID, &p.Position, &kind, &p.Caption, &box, &p.Width, &p.Height, &p.ImageSHA, &created); err != nil {
		return p, err
	}
	p.Kind = domain.PictureKind(kind)
	if err := json.Unmarshal([]byte(box), &p.Box); err != nil {
		return p, fmt.Errorf("decode picture box: %w", err)
	}
	var err error
	p.CreatedAt, err = parseTime(created)
	return p, err
}

func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?, ", n), ", ")
}

func (r repos) SetPagePictures(ctx context.Context, tenantID string, pageIDs []string, pics []domain.Picture) ([]string, error) {
	if len(pageIDs) == 0 {
		return nil, nil
	}
	args := []any{tenantID}
	for _, id := range pageIDs {
		args = append(args, id)
	}
	where := `tenant_id = ? AND page_id IN (` + placeholders(len(pageIDs)) + `)`
	rows, err := r.q.QueryContext(ctx, `SELECT image_sha FROM pictures WHERE `+where, args...)
	if err != nil {
		return nil, fmt.Errorf("list pictures: %w", err)
	}
	var old []string
	for rows.Next() {
		var sha string
		if err := rows.Scan(&sha); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan picture: %w", err)
		}
		old = append(old, sha)
	}
	if err := errors.Join(rows.Err(), rows.Close()); err != nil {
		return nil, fmt.Errorf("list pictures: %w", err)
	}
	if _, err := r.q.ExecContext(ctx, `DELETE FROM pictures WHERE `+where, args...); err != nil {
		return nil, fmt.Errorf("delete pictures: %w", err)
	}
	for _, p := range pics {
		box, err := jsonText(p.Box)
		if err != nil {
			return nil, err
		}
		res, err := r.q.ExecContext(ctx, `INSERT INTO pictures (id, tenant_id, page_id, position, kind, caption, box_json, width, height, image_sha, created_at)
			SELECT ?, ?, id, ?, ?, ?, ?, ?, ?, ?, ? FROM pages WHERE tenant_id = ? AND id = ?`,
			p.ID, tenantID, p.Position, string(p.Kind), p.Caption, box, p.Width, p.Height, p.ImageSHA, formatTime(p.CreatedAt), tenantID, p.PageID)
		if err := affected(res, err, "page"); err != nil {
			return nil, err
		}
	}
	return old, nil
}

func (r repos) PicturesOf(ctx context.Context, tenantID string, pageIDs []string) ([]domain.Picture, error) {
	out := []domain.Picture{}
	if len(pageIDs) == 0 {
		return out, nil
	}
	args := []any{tenantID}
	for _, id := range pageIDs {
		args = append(args, id)
	}
	rows, err := r.q.QueryContext(ctx, `SELECT `+pictureColumns+` FROM pictures pc JOIN pages p ON p.id = pc.page_id
		WHERE pc.tenant_id = ? AND p.removed_at IS NULL AND pc.page_id IN (`+placeholders(len(pageIDs))+`)
		ORDER BY pc.page_id, pc.position`, args...)
	if err != nil {
		return nil, fmt.Errorf("list pictures: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		p, err := scanPicture(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scan picture: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r repos) Picture(ctx context.Context, tenantID, id string) (domain.Picture, error) {
	p, err := scanPicture(r.q.QueryRowContext(ctx, `SELECT `+pictureColumns+` FROM pictures pc JOIN pages p ON p.id = pc.page_id
		WHERE pc.tenant_id = ? AND pc.id = ? AND p.removed_at IS NULL`, tenantID, id).Scan)
	if err != nil {
		return p, one(err, "picture")
	}
	return p, nil
}
