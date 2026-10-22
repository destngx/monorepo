package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"apps/health-watcher/domain"
	"apps/health-watcher/port"
)

// index writes one search row: the original text for snippets and its
// folded form for matching (HW-FIND-2). Empty text is not indexed.
func (r repos) index(ctx context.Context, tenantID string, kind domain.SearchKind, refID, body string) error {
	if err := r.unindex(ctx, kind, refID); err != nil {
		return err
	}
	if strings.TrimSpace(body) == "" {
		return nil
	}
	if _, err := r.q.ExecContext(ctx, `INSERT INTO search_index (tenant_id, kind, ref_id, body, folded) VALUES (?, ?, ?, ?, ?)`,
		tenantID, string(kind), refID, body, domain.FoldSearchText(body)); err != nil {
		return fmt.Errorf("index %s: %w", kind, err)
	}
	return nil
}

func (r repos) unindex(ctx context.Context, kind domain.SearchKind, refID string) error {
	if _, err := r.q.ExecContext(ctx, `DELETE FROM search_index WHERE kind = ? AND ref_id = ?`, string(kind), refID); err != nil {
		return fmt.Errorf("unindex %s: %w", kind, err)
	}
	return nil
}

// visitText is what a visit is found by: its title, note and provider name.
func visitText(v domain.Visit) string {
	parts := []string{v.Title, v.Note}
	if v.Provider != nil {
		parts = append(parts, v.Provider.Name)
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func (r repos) indexVisit(ctx context.Context, tenantID, visitID string) error {
	v, err := r.Visit(ctx, tenantID, visitID)
	if err != nil {
		return err
	}
	return r.index(ctx, tenantID, domain.SearchVisit, v.ID, visitText(v))
}

// findingText is what a finding is found by.
func findingText(f domain.Finding) string {
	return strings.TrimSpace(strings.Join([]string{f.Text, f.Drug, f.Dose, f.ICD10, f.Section}, "\n"))
}

func (r repos) indexFinding(ctx context.Context, tenantID string, f domain.Finding) error {
	if f.Status != domain.StatusApproved {
		return r.unindex(ctx, domain.SearchFinding, f.ID)
	}
	return r.index(ctx, tenantID, domain.SearchFinding, f.ID, findingText(f))
}

// Search returns the best matches of an FTS5 expression in a tenant, each
// with where it lives. Rows whose record is gone or whose file belongs to
// no document and no unfiled inbox item are skipped.
func (r repos) Search(ctx context.Context, tenantID, expr string, limit int) ([]port.SearchRow, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT kind, ref_id, body FROM search_index
		WHERE search_index MATCH ? AND tenant_id = ? ORDER BY rank LIMIT ?`, expr, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	type raw struct{ kind, ref, body string }
	var found []raw
	for rows.Next() {
		var x raw
		if err := rows.Scan(&x.kind, &x.ref, &x.body); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan search row: %w", err)
		}
		found = append(found, x)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	out := []port.SearchRow{}
	for _, x := range found {
		hit, ok, err := r.searchContext(ctx, tenantID, domain.SearchKind(x.kind), x.ref)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, port.SearchRow{Hit: hit, Body: x.body})
		}
	}
	return out, nil
}

// visitContext fills the person, visit and provider of a hit.
const visitContextSelect = `SELECT v.id, v.date, v.title, v.type, p.id, p.display_name, COALESCE(pr.name, '')
	FROM visits v JOIN people p ON p.id = v.person_id LEFT JOIN providers pr ON pr.id = v.provider_id
	WHERE v.tenant_id = ? AND v.id = ?`

func (r repos) fillVisit(ctx context.Context, tenantID, visitID string, hit *domain.SearchHit) (bool, error) {
	var typ string
	err := r.q.QueryRowContext(ctx, visitContextSelect, tenantID, visitID).
		Scan(&hit.VisitID, &hit.Date, &hit.VisitTitle, &typ, &hit.PersonID, &hit.PersonName, &hit.ProviderName)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read search hit visit: %w", err)
	}
	if hit.VisitTitle == "" {
		hit.VisitTitle = typ
	}
	return true, nil
}

func (r repos) fillDocument(ctx context.Context, tenantID, documentID string, hit *domain.SearchHit) (bool, error) {
	var kind, visitID string
	err := r.q.QueryRowContext(ctx, `SELECT id, kind, title, visit_id FROM documents WHERE tenant_id = ? AND id = ?`, tenantID, documentID).
		Scan(&hit.DocumentID, &kind, &hit.DocumentTitle, &visitID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read search hit document: %w", err)
	}
	hit.DocumentKind = domain.DocumentKind(kind)
	return r.fillVisit(ctx, tenantID, visitID, hit)
}

func (r repos) searchContext(ctx context.Context, tenantID string, kind domain.SearchKind, ref string) (domain.SearchHit, bool, error) {
	hit := domain.SearchHit{Kind: kind}
	switch kind {
	case domain.SearchVisit:
		ok, err := r.fillVisit(ctx, tenantID, ref, &hit)
		return hit, ok, err
	case domain.SearchFinding:
		var docID string
		err := r.q.QueryRowContext(ctx, `SELECT COALESCE(document_id, '') FROM findings WHERE tenant_id = ? AND id = ? AND status = 'approved'`,
			tenantID, ref).Scan(&docID)
		if errors.Is(err, sql.ErrNoRows) || docID == "" {
			return hit, false, nil
		}
		if err != nil {
			return hit, false, fmt.Errorf("read search hit finding: %w", err)
		}
		ok, err := r.fillDocument(ctx, tenantID, docID, &hit)
		return hit, ok, err
	case domain.SearchPage:
		var fileID string
		err := r.q.QueryRowContext(ctx, `SELECT id, number, file_id FROM pages WHERE tenant_id = ? AND id = ?`, tenantID, ref).
			Scan(&hit.PageID, &hit.PageNumber, &fileID)
		if errors.Is(err, sql.ErrNoRows) {
			return hit, false, nil
		}
		if err != nil {
			return hit, false, fmt.Errorf("read search hit page: %w", err)
		}
		var docID string
		err = r.q.QueryRowContext(ctx, `SELECT d.id FROM document_files df JOIN documents d ON d.id = df.document_id
			WHERE d.tenant_id = ? AND df.file_id = ? ORDER BY d.created_at LIMIT 1`, tenantID, fileID).Scan(&docID)
		switch {
		case err == nil:
			ok, err := r.fillDocument(ctx, tenantID, docID, &hit)
			return hit, ok, err
		case !errors.Is(err, sql.ErrNoRows):
			return hit, false, fmt.Errorf("read search hit document: %w", err)
		}
		var personID string
		err = r.q.QueryRowContext(ctx, `SELECT i.id, COALESCE(i.person_id, '') FROM inbox_files inf JOIN inbox_items i ON i.id = inf.item_id
			WHERE i.tenant_id = ? AND inf.file_id = ? AND i.status = 'new' LIMIT 1`, tenantID, fileID).Scan(&hit.ItemID, &personID)
		if errors.Is(err, sql.ErrNoRows) {
			return hit, false, nil
		}
		if err != nil {
			return hit, false, fmt.Errorf("read search hit item: %w", err)
		}
		if personID != "" {
			if p, err := r.Person(ctx, tenantID, personID); err == nil {
				hit.PersonID, hit.PersonName = p.ID, p.DisplayName
			}
		}
		return hit, true, nil
	}
	return hit, false, nil
}

// RebuildSearchIndex refills the index from pages, visits and approved
// findings when meta.search_version differs from version.
func (r repos) RebuildSearchIndex(ctx context.Context, version string) (bool, error) {
	var current string
	err := r.q.QueryRowContext(ctx, `SELECT value FROM meta WHERE key = 'search_version'`).Scan(&current)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, fmt.Errorf("read search version: %w", err)
	}
	if current == version {
		return false, nil
	}
	type entry struct {
		tenant, kind, ref, body string
	}
	var entries []entry
	// collect reads one row per record; text builds its body from the columns
	// after the tenant and id, the same way the live indexing does.
	collect := func(kind domain.SearchKind, query string, text func(cols []string) string) error {
		rows, err := r.q.QueryContext(ctx, query)
		if err != nil {
			return fmt.Errorf("rebuild search: %w", err)
		}
		defer rows.Close()
		cols, err := rows.Columns()
		if err != nil {
			return fmt.Errorf("rebuild search: %w", err)
		}
		vals := make([]string, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		for rows.Next() {
			if err := rows.Scan(ptrs...); err != nil {
				return fmt.Errorf("rebuild search: %w", err)
			}
			entries = append(entries, entry{tenant: vals[0], kind: string(kind), ref: vals[1], body: text(vals[2:])})
		}
		return rows.Err()
	}
	if err := collect(domain.SearchPage, `SELECT tenant_id, id, text FROM pages`, func(c []string) string { return c[0] }); err != nil {
		return false, err
	}
	if err := collect(domain.SearchVisit, `SELECT v.tenant_id, v.id, v.title, v.note, COALESCE(pr.name, '')
		FROM visits v LEFT JOIN providers pr ON pr.id = v.provider_id`, func(c []string) string {
		v := domain.Visit{Title: c[0], Note: c[1]}
		if c[2] != "" {
			v.Provider = &domain.Provider{Name: c[2]}
		}
		return visitText(v)
	}); err != nil {
		return false, err
	}
	if err := collect(domain.SearchFinding, `SELECT tenant_id, id, text, drug, dose, icd10, section FROM findings WHERE status = 'approved'`,
		func(c []string) string {
			return findingText(domain.Finding{Text: c[0], Drug: c[1], Dose: c[2], ICD10: c[3], Section: c[4]})
		}); err != nil {
		return false, err
	}
	if _, err := r.q.ExecContext(ctx, `DELETE FROM search_index`); err != nil {
		return false, fmt.Errorf("clear search index: %w", err)
	}
	for _, e := range entries {
		if strings.TrimSpace(e.body) == "" {
			continue
		}
		if _, err := r.q.ExecContext(ctx, `INSERT INTO search_index (tenant_id, kind, ref_id, body, folded) VALUES (?, ?, ?, ?, ?)`,
			e.tenant, e.kind, e.ref, e.body, domain.FoldSearchText(e.body)); err != nil {
			return false, fmt.Errorf("rebuild search: %w", err)
		}
	}
	if _, err := r.q.ExecContext(ctx, `INSERT INTO meta (key, value) VALUES ('search_version', ?)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`, version); err != nil {
		return false, fmt.Errorf("write search version: %w", err)
	}
	return true, nil
}
