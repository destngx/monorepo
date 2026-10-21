package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"apps/health-watcher/domain"
	"apps/health-watcher/port"
)

const pageColumns = `id, file_id, number, method, text, confidence, width, height, image_sha, lines_json, barcodes_json,
	mask_status, mask_reasons_json, mask_boxes_json, mask_manual_json, mask_lines, mask_text, mask_image_sha`

func scanPage(scan func(...any) error) (domain.Page, error) {
	var p domain.Page
	var method, linesJSON, maskStatus, reasonsJSON, boxesJSON, manualJSON, maskText, maskSHA string
	var barcodesJSON sql.NullString
	var maskLines int
	if err := scan(&p.ID, &p.FileID, &p.Number, &method, &p.Text, &p.Confidence, &p.Width, &p.Height, &p.ImageSHA, &linesJSON,
		&barcodesJSON, &maskStatus, &reasonsJSON, &boxesJSON, &manualJSON, &maskLines, &maskText, &maskSHA); err != nil {
		return p, err
	}
	p.Method = domain.PageMethod(method)
	p.HasImage = p.ImageSHA != ""
	if err := json.Unmarshal([]byte(linesJSON), &p.Lines); err != nil {
		return p, fmt.Errorf("decode page lines: %w", err)
	}
	if barcodesJSON.Valid {
		p.BarcodesChecked = true
		if err := json.Unmarshal([]byte(barcodesJSON.String), &p.Barcodes); err != nil {
			return p, fmt.Errorf("decode page barcodes: %w", err)
		}
	}
	if maskStatus != "" {
		m := &domain.PageMask{Status: domain.MaskStatus(maskStatus), MaskedLines: maskLines, Text: maskText, ImageSHA: maskSHA}
		for _, d := range []struct {
			raw string
			out any
		}{{reasonsJSON, &m.Reasons}, {boxesJSON, &m.Boxes}, {manualJSON, &m.Manual}} {
			if err := json.Unmarshal([]byte(d.raw), d.out); err != nil {
				return p, fmt.Errorf("decode page mask: %w", err)
			}
		}
		p.Mask = m
	}
	return p, nil
}

func jsonText(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("encode page data: %w", err)
	}
	return string(raw), nil
}

func (r repos) PagesOf(ctx context.Context, tenantID, fileID string) ([]domain.Page, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT `+pageColumns+` FROM pages WHERE tenant_id = ? AND file_id = ? ORDER BY number`, tenantID, fileID)
	if err != nil {
		return nil, fmt.Errorf("list pages: %w", err)
	}
	defer rows.Close()
	out := []domain.Page{}
	for rows.Next() {
		p, err := scanPage(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scan page: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r repos) Page(ctx context.Context, tenantID, id string) (domain.Page, error) {
	p, err := scanPage(r.q.QueryRowContext(ctx, `SELECT `+pageColumns+` FROM pages WHERE tenant_id = ? AND id = ?`, tenantID, id).Scan)
	return p, one(err, "page")
}

func (r repos) SavePages(ctx context.Context, tenantID, fileID string, pages []domain.Page) ([]string, error) {
	if _, err := r.File(ctx, tenantID, fileID); err != nil {
		return nil, err
	}
	old, err := r.PagesOf(ctx, tenantID, fileID)
	if err != nil {
		return nil, err
	}
	oldByNumber := map[int]domain.Page{}
	for _, p := range old {
		oldByNumber[p.Number] = p
	}
	kept := map[string]bool{}
	for _, p := range pages {
		if prev, ok := oldByNumber[p.Number]; ok {
			p.ID = prev.ID
		}
		lines, err := jsonText(nonNil(p.Lines))
		if err != nil {
			return nil, err
		}
		var barcodes any
		if p.BarcodesChecked {
			if barcodes, err = jsonText(nonNil(p.Barcodes)); err != nil {
				return nil, err
			}
		}
		// A new reading invalidates the old mask.
		res, err := r.q.ExecContext(ctx, `INSERT INTO pages (id, tenant_id, file_id, number, method, text, confidence, width, height,
				image_sha, lines_json, barcodes_json, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (file_id, number) DO UPDATE SET method = excluded.method, text = excluded.text,
				confidence = excluded.confidence, width = excluded.width, height = excluded.height, image_sha = excluded.image_sha,
				lines_json = excluded.lines_json, barcodes_json = excluded.barcodes_json, mask_status = '', mask_reasons_json = '[]',
				mask_boxes_json = '[]', mask_manual_json = '[]', mask_lines = 0, mask_text = '', mask_image_sha = ''`,
			p.ID, tenantID, fileID, p.Number, string(p.Method), p.Text, p.Confidence, p.Width, p.Height, p.ImageSHA, lines, barcodes,
			formatTime(p.CreatedAt))
		if err := exec(res, err, "page"); err != nil {
			return nil, err
		}
		kept[p.ImageSHA] = true
		if err := r.index(ctx, tenantID, domain.SearchPage, p.ID, p.Text); err != nil {
			return nil, err
		}
	}
	var released []string
	for _, p := range old {
		if p.Number > len(pages) {
			if err := r.unindex(ctx, domain.SearchPage, p.ID); err != nil {
				return nil, err
			}
		}
		if p.ImageSHA != "" && !kept[p.ImageSHA] {
			released = append(released, p.ImageSHA)
		}
		if p.Mask != nil && p.Mask.ImageSHA != "" {
			released = append(released, p.Mask.ImageSHA)
		}
	}
	if _, err := r.q.ExecContext(ctx, `DELETE FROM pages WHERE tenant_id = ? AND file_id = ? AND number > ?`, tenantID, fileID, len(pages)); err != nil {
		return nil, fmt.Errorf("delete extra pages: %w", err)
	}
	return released, nil
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func (r repos) SetPageMask(ctx context.Context, tenantID, pageID string, m domain.PageMask) (string, error) {
	var prev string
	if err := r.q.QueryRowContext(ctx, `SELECT mask_image_sha FROM pages WHERE tenant_id = ? AND id = ?`, tenantID, pageID).Scan(&prev); err != nil {
		return "", one(err, "page")
	}
	reasons, err := jsonText(nonNil(m.Reasons))
	if err != nil {
		return "", err
	}
	boxes, err := jsonText(nonNil(m.Boxes))
	if err != nil {
		return "", err
	}
	manual, err := jsonText(nonNil(m.Manual))
	if err != nil {
		return "", err
	}
	res, err := r.q.ExecContext(ctx, `UPDATE pages SET mask_status = ?, mask_reasons_json = ?, mask_boxes_json = ?, mask_manual_json = ?,
		mask_lines = ?, mask_text = ?, mask_image_sha = ? WHERE tenant_id = ? AND id = ?`,
		string(m.Status), reasons, boxes, manual, m.MaskedLines, m.Text, m.ImageSHA, tenantID, pageID)
	if err := affected(res, err, "page"); err != nil {
		return "", err
	}
	if prev == m.ImageSHA {
		return "", nil
	}
	return prev, nil
}

func (r repos) PageImageSHAs(ctx context.Context, tenantID, fileID string) ([]string, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT image_sha, mask_image_sha FROM pages WHERE tenant_id = ? AND file_id = ?`, tenantID, fileID)
	if err != nil {
		return nil, fmt.Errorf("list page images: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var img, mask string
		if err := rows.Scan(&img, &mask); err != nil {
			return nil, fmt.Errorf("scan page images: %w", err)
		}
		for _, s := range []string{img, mask} {
			if s != "" {
				out = append(out, s)
			}
		}
	}
	return out, rows.Err()
}

func (r repos) FilesWithoutPages(ctx context.Context) ([]port.TenantFile, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT f.tenant_id, f.id FROM files f
		WHERE NOT EXISTS (SELECT 1 FROM pages p WHERE p.file_id = f.id)
		AND NOT EXISTS (SELECT 1 FROM jobs j WHERE j.kind = 'ocr' AND j.subject_id = f.id)
		ORDER BY f.created_at`)
	if err != nil {
		return nil, fmt.Errorf("list files without pages: %w", err)
	}
	defer rows.Close()
	var out []port.TenantFile
	for rows.Next() {
		var tf port.TenantFile
		if err := rows.Scan(&tf.TenantID, &tf.FileID); err != nil {
			return nil, fmt.Errorf("scan file: %w", err)
		}
		out = append(out, tf)
	}
	return out, rows.Err()
}
