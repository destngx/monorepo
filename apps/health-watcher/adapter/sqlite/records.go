package sqlite

import (
	"context"
	"fmt"
	"strings"

	"apps/health-watcher/domain"
	"apps/health-watcher/port"
)

func (r repos) CreatePerson(ctx context.Context, tenantID string, p domain.Person) error {
	res, err := r.q.ExecContext(ctx, `INSERT INTO people (id, tenant_id, display_name, birth_date, sex, user_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, p.ID, tenantID, p.DisplayName, p.BirthDate, string(p.Sex), nullable(p.UserID), formatTime(p.CreatedAt))
	return exec(res, err, "person")
}

func (r repos) UpdatePerson(ctx context.Context, tenantID, id string, in domain.PersonInput) error {
	res, err := r.q.ExecContext(ctx, `UPDATE people SET display_name = ?, birth_date = ?, sex = ?, user_id = ?
		WHERE tenant_id = ? AND id = ?`, in.DisplayName, in.BirthDate, in.Sex, nullable(in.UserID), tenantID, id)
	return affected(res, err, "person")
}

const personSelect = `SELECT p.id, p.display_name, p.birth_date, p.sex, COALESCE(p.user_id, ''), p.created_at,
	COALESCE((SELECT MAX(v.date) FROM visits v WHERE v.person_id = p.id), ''),
	(SELECT COUNT(*) FROM visits v WHERE v.person_id = p.id)
	FROM people p`

func scanPerson(scan func(...any) error) (domain.Person, error) {
	var p domain.Person
	var sex, created string
	if err := scan(&p.ID, &p.DisplayName, &p.BirthDate, &sex, &p.UserID, &created, &p.LastVisitOn, &p.VisitCount); err != nil {
		return p, err
	}
	p.Sex = domain.Sex(sex)
	var err error
	p.CreatedAt, err = parseTime(created)
	return p, err
}

func (r repos) Person(ctx context.Context, tenantID, id string) (domain.Person, error) {
	p, err := scanPerson(r.q.QueryRowContext(ctx, personSelect+` WHERE p.tenant_id = ? AND p.id = ?`, tenantID, id).Scan)
	return p, one(err, "person")
}

func (r repos) People(ctx context.Context, tenantID string) ([]domain.Person, error) {
	rows, err := r.q.QueryContext(ctx, personSelect+` WHERE p.tenant_id = ? ORDER BY p.created_at, p.id`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list people: %w", err)
	}
	defer rows.Close()
	out := []domain.Person{}
	for rows.Next() {
		p, err := scanPerson(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scan person: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r repos) DeletePerson(ctx context.Context, tenantID, id string) error {
	res, err := r.q.ExecContext(ctx, `DELETE FROM people WHERE tenant_id = ? AND id = ?`, tenantID, id)
	return affected(res, err, "person")
}

func (r repos) UnlinkUser(ctx context.Context, tenantID, userID string) error {
	res, err := r.q.ExecContext(ctx, `UPDATE people SET user_id = NULL WHERE tenant_id = ? AND user_id = ?`, tenantID, userID)
	return exec(res, err, "person")
}

func (r repos) ProviderByKey(ctx context.Context, tenantID, key string) (domain.Provider, error) {
	var p domain.Provider
	var kind string
	err := r.q.QueryRowContext(ctx, `SELECT id, name, kind FROM providers WHERE tenant_id = ? AND name_key = ?`, tenantID, key).
		Scan(&p.ID, &p.Name, &kind)
	p.Kind = domain.ProviderKind(kind)
	return p, one(err, "provider")
}

func (r repos) CreateProvider(ctx context.Context, tenantID string, p domain.Provider, key string) error {
	res, err := r.q.ExecContext(ctx, `INSERT INTO providers (id, tenant_id, name, name_key, kind) VALUES (?, ?, ?, ?, ?)`,
		p.ID, tenantID, p.Name, key, string(p.Kind))
	return exec(res, err, "provider")
}

func (r repos) Providers(ctx context.Context, tenantID string) ([]domain.Provider, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT id, name, kind FROM providers WHERE tenant_id = ? ORDER BY name_key`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}
	defer rows.Close()
	out := []domain.Provider{}
	for rows.Next() {
		var p domain.Provider
		var kind string
		if err := rows.Scan(&p.ID, &p.Name, &kind); err != nil {
			return nil, fmt.Errorf("scan provider: %w", err)
		}
		p.Kind = domain.ProviderKind(kind)
		out = append(out, p)
	}
	return out, rows.Err()
}

func providerID(v domain.Visit) any {
	if v.Provider == nil {
		return nil
	}
	return v.Provider.ID
}

func (r repos) CreateVisit(ctx context.Context, tenantID string, v domain.Visit) error {
	res, err := r.q.ExecContext(ctx, `INSERT INTO visits (id, tenant_id, person_id, provider_id, date, end_date, type, title, note, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, v.ID, tenantID, v.PersonID, providerID(v), v.Date, v.EndDate, string(v.Type), v.Title, v.Note,
		formatTime(v.CreatedAt))
	if err := exec(res, err, "visit"); err != nil {
		return err
	}
	return r.indexVisit(ctx, tenantID, v.ID)
}

func (r repos) UpdateVisit(ctx context.Context, tenantID string, v domain.Visit) error {
	res, err := r.q.ExecContext(ctx, `UPDATE visits SET person_id = ?, provider_id = ?, date = ?, end_date = ?, type = ?, title = ?, note = ?
		WHERE tenant_id = ? AND id = ?`, v.PersonID, providerID(v), v.Date, v.EndDate, string(v.Type), v.Title, v.Note, tenantID, v.ID)
	if err := affected(res, err, "visit"); err != nil {
		return err
	}
	return r.indexVisit(ctx, tenantID, v.ID)
}

const visitSelect = `SELECT v.id, v.person_id, v.date, v.end_date, v.type, v.title, v.note, v.created_at,
	COALESCE(pr.id, ''), COALESCE(pr.name, ''), COALESCE(pr.kind, ''),
	(SELECT COUNT(*) FROM documents d WHERE d.visit_id = v.id),
	(SELECT COUNT(*) FROM observations o JOIN documents d ON d.id = o.document_id
		WHERE d.visit_id = v.id AND o.status = 'approved' AND o.flag <> '')
	FROM visits v LEFT JOIN providers pr ON pr.id = v.provider_id`

func scanVisit(scan func(...any) error) (domain.Visit, error) {
	var v domain.Visit
	var typ, created, prID, prName, prKind string
	if err := scan(&v.ID, &v.PersonID, &v.Date, &v.EndDate, &typ, &v.Title, &v.Note, &created, &prID, &prName, &prKind, &v.DocumentCount,
		&v.FlaggedCount); err != nil {
		return v, err
	}
	v.Type = domain.VisitType(typ)
	if prID != "" {
		v.Provider = &domain.Provider{ID: prID, Name: prName, Kind: domain.ProviderKind(prKind)}
	}
	v.ExportGaps = domain.VisitExportGaps(v)
	var err error
	v.CreatedAt, err = parseTime(created)
	return v, err
}

func (r repos) Visit(ctx context.Context, tenantID, id string) (domain.Visit, error) {
	v, err := scanVisit(r.q.QueryRowContext(ctx, visitSelect+` WHERE v.tenant_id = ? AND v.id = ?`, tenantID, id).Scan)
	return v, one(err, "visit")
}

func (r repos) VisitsOf(ctx context.Context, tenantID, personID string, f port.VisitFilter) ([]domain.Visit, error) {
	where := []string{"v.tenant_id = ?", "v.person_id = ?"}
	args := []any{tenantID, personID}
	add := func(cond, value string) {
		if value != "" {
			where = append(where, cond)
			args = append(args, value)
		}
	}
	add("v.type = ?", f.Type)
	add("v.provider_id = ?", f.ProviderID)
	add("v.date >= ?", f.From)
	add("v.date <= ?", f.To)
	rows, err := r.q.QueryContext(ctx, visitSelect+` WHERE `+strings.Join(where, " AND ")+` ORDER BY v.date DESC, v.created_at DESC`, args...)
	if err != nil {
		return nil, fmt.Errorf("list visits: %w", err)
	}
	defer rows.Close()
	out := []domain.Visit{}
	for rows.Next() {
		v, err := scanVisit(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scan visit: %w", err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r repos) DeleteVisit(ctx context.Context, tenantID, id string) error {
	res, err := r.q.ExecContext(ctx, `DELETE FROM visits WHERE tenant_id = ? AND id = ?`, tenantID, id)
	if err := affected(res, err, "visit"); err != nil {
		return err
	}
	return r.unindex(ctx, domain.SearchVisit, id)
}

func (r repos) CreateDocument(ctx context.Context, tenantID string, d domain.Document, fileIDs []string) error {
	res, err := r.q.ExecContext(ctx, `INSERT INTO documents (id, tenant_id, visit_id, person_id, kind, title, date, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, d.ID, tenantID, d.VisitID, d.PersonID, string(d.Kind), d.Title, d.Date, formatTime(d.CreatedAt))
	if err := exec(res, err, "document"); err != nil {
		return err
	}
	for i, fileID := range fileIDs {
		res, err := r.q.ExecContext(ctx, `INSERT INTO document_files (document_id, position, file_id)
			SELECT ?, ?, id FROM files WHERE tenant_id = ? AND id = ?`, d.ID, i, tenantID, fileID)
		if err := affected(res, err, "file"); err != nil {
			return err
		}
	}
	return nil
}

const fileColumns = `f.id, f.sha256, f.mime, f.size, f.original_name, f.created_at`

func scanFile(scan func(...any) error) (domain.StoredFile, error) {
	var f domain.StoredFile
	var created string
	if err := scan(&f.ID, &f.SHA256, &f.MIME, &f.Size, &f.OriginalName, &created); err != nil {
		return f, err
	}
	var err error
	f.CreatedAt, err = parseTime(created)
	return f, err
}

func (r repos) filesOf(ctx context.Context, query string, id string) ([]domain.StoredFile, error) {
	rows, err := r.q.QueryContext(ctx, query, id)
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}
	defer rows.Close()
	out := []domain.StoredFile{}
	for rows.Next() {
		f, err := scanFile(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scan file: %w", err)
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r repos) documentFiles(ctx context.Context, documentID string) ([]domain.StoredFile, error) {
	return r.filesOf(ctx, `SELECT `+fileColumns+` FROM document_files df JOIN files f ON f.id = df.file_id
		WHERE df.document_id = ? ORDER BY df.position`, documentID)
}

const documentSelect = `SELECT id, visit_id, person_id, kind, title, date, created_at FROM documents`

func scanDocument(scan func(...any) error) (domain.Document, error) {
	var d domain.Document
	var kind, created string
	if err := scan(&d.ID, &d.VisitID, &d.PersonID, &kind, &d.Title, &d.Date, &created); err != nil {
		return d, err
	}
	d.Kind = domain.DocumentKind(kind)
	var err error
	d.CreatedAt, err = parseTime(created)
	return d, err
}

func (r repos) Document(ctx context.Context, tenantID, id string) (domain.Document, error) {
	d, err := scanDocument(r.q.QueryRowContext(ctx, documentSelect+` WHERE tenant_id = ? AND id = ?`, tenantID, id).Scan)
	if err := one(err, "document"); err != nil {
		return d, err
	}
	d.Files, err = r.documentFiles(ctx, d.ID)
	return d, err
}

func (r repos) DocumentsOf(ctx context.Context, tenantID, visitID string) ([]domain.Document, error) {
	rows, err := r.q.QueryContext(ctx, documentSelect+` WHERE tenant_id = ? AND visit_id = ? ORDER BY created_at, id`, tenantID, visitID)
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	var out []domain.Document
	for rows.Next() {
		d, err := scanDocument(rows.Scan)
		if err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan document: %w", err)
		}
		out = append(out, d)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	// Files are read after the rows close: the pool has a single connection.
	for i := range out {
		if out[i].Files, err = r.documentFiles(ctx, out[i].ID); err != nil {
			return nil, err
		}
	}
	if out == nil {
		out = []domain.Document{}
	}
	return out, nil
}

func (r repos) UpdateDocument(ctx context.Context, tenantID string, d domain.Document) error {
	res, err := r.q.ExecContext(ctx, `UPDATE documents SET kind = ?, title = ?, date = ? WHERE tenant_id = ? AND id = ?`,
		string(d.Kind), d.Title, d.Date, tenantID, d.ID)
	return affected(res, err, "document")
}

func (r repos) DeleteDocument(ctx context.Context, tenantID, id string) error {
	// The document's findings cascade away; their search rows go first.
	if _, err := r.q.ExecContext(ctx, `DELETE FROM search_index WHERE kind = ? AND ref_id IN (
			SELECT id FROM findings WHERE tenant_id = ? AND document_id = ?)`, string(domain.SearchFinding), tenantID, id); err != nil {
		return fmt.Errorf("unindex findings: %w", err)
	}
	res, err := r.q.ExecContext(ctx, `DELETE FROM documents WHERE tenant_id = ? AND id = ?`, tenantID, id)
	return affected(res, err, "document")
}

func (r repos) UnlinkDocumentFile(ctx context.Context, tenantID, documentID, fileID string) error {
	res, err := r.q.ExecContext(ctx, `DELETE FROM document_files WHERE file_id = ?
		AND document_id IN (SELECT id FROM documents WHERE tenant_id = ? AND id = ?)`, fileID, tenantID, documentID)
	if err := affected(res, err, "document file"); err != nil {
		return err
	}
	// The item it was filed from drops the file too, so going back to the
	// inbox doesn't bring it back.
	if _, err := r.q.ExecContext(ctx, `DELETE FROM inbox_files WHERE file_id = ?
		AND item_id IN (SELECT id FROM inbox_items WHERE tenant_id = ? AND document_id = ?)`, fileID, tenantID, documentID); err != nil {
		return fmt.Errorf("unlink item file: %w", err)
	}
	return nil
}

func (r repos) CreateFile(ctx context.Context, tenantID string, f domain.StoredFile) error {
	res, err := r.q.ExecContext(ctx, `INSERT INTO files (id, tenant_id, sha256, mime, size, original_name, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, f.ID, tenantID, f.SHA256, f.MIME, f.Size, f.OriginalName, formatTime(f.CreatedAt))
	return exec(res, err, "file")
}

func (r repos) FileBySHA(ctx context.Context, tenantID, sha string) (domain.StoredFile, error) {
	f, err := scanFile(r.q.QueryRowContext(ctx, `SELECT `+fileColumns+` FROM files f WHERE f.tenant_id = ? AND f.sha256 = ?`, tenantID, sha).Scan)
	return f, one(err, "file")
}

func (r repos) File(ctx context.Context, tenantID, id string) (domain.StoredFile, error) {
	f, err := scanFile(r.q.QueryRowContext(ctx, `SELECT `+fileColumns+` FROM files f WHERE f.tenant_id = ? AND f.id = ?`, tenantID, id).Scan)
	return f, one(err, "file")
}

func (r repos) FileInUse(ctx context.Context, tenantID, id string) (bool, error) {
	var used bool
	err := r.q.QueryRowContext(ctx, `SELECT EXISTS (
			SELECT 1 FROM document_files df JOIN documents d ON d.id = df.document_id WHERE d.tenant_id = ? AND df.file_id = ?
		) OR EXISTS (
			SELECT 1 FROM inbox_files inf JOIN inbox_items i ON i.id = inf.item_id
			WHERE i.tenant_id = ? AND inf.file_id = ? AND i.status = 'new'
		) OR EXISTS (
			SELECT 1 FROM staged_files WHERE tenant_id = ? AND file_id = ?
		)`, tenantID, id, tenantID, id, tenantID, id).Scan(&used)
	if err != nil {
		return false, fmt.Errorf("check file use: %w", err)
	}
	return used, nil
}

func (r repos) DeleteFile(ctx context.Context, tenantID, id string) error {
	// Pages go with the file; their search rows must go first.
	if _, err := r.q.ExecContext(ctx, `DELETE FROM search_index WHERE kind = 'page' AND ref_id IN (
		SELECT id FROM pages WHERE tenant_id = ? AND file_id = ?)`, tenantID, id); err != nil {
		return fmt.Errorf("unindex pages: %w", err)
	}
	// Filed or discarded inbox items keep no hold on their files.
	if _, err := r.q.ExecContext(ctx, `DELETE FROM inbox_files WHERE file_id = ?
		AND item_id IN (SELECT id FROM inbox_items WHERE tenant_id = ? AND status <> 'new')`, id, tenantID); err != nil {
		return fmt.Errorf("release file: %w", err)
	}
	res, err := r.q.ExecContext(ctx, `DELETE FROM files WHERE tenant_id = ? AND id = ?`, tenantID, id)
	return affected(res, err, "file")
}

func (r repos) SHAInUse(ctx context.Context, sha string) (bool, error) {
	var used bool
	if err := r.q.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM files WHERE sha256 = ?)
		OR EXISTS (SELECT 1 FROM pages WHERE image_sha = ?) OR EXISTS (SELECT 1 FROM pages WHERE mask_image_sha = ?)`,
		sha, sha, sha).Scan(&used); err != nil {
		return false, fmt.Errorf("check file hash use: %w", err)
	}
	return used, nil
}

func (r repos) CreateInboxItem(ctx context.Context, tenantID string, item domain.InboxItem, fileIDs []string) error {
	res, err := r.q.ExecContext(ctx, `INSERT INTO inbox_items (id, tenant_id, status, source, note, uploaded_by, person_id, created_at)
		SELECT ?, ?, ?, ?, ?, ?, (SELECT id FROM people WHERE tenant_id = ? AND id = ?), ?`, item.ID, tenantID, string(item.Status),
		string(item.Source), item.Note, item.UploadedBy, tenantID, item.PersonID, formatTime(item.CreatedAt))
	if err := exec(res, err, "inbox item"); err != nil {
		return err
	}
	for i, fileID := range fileIDs {
		res, err := r.q.ExecContext(ctx, `INSERT INTO inbox_files (item_id, position, file_id)
			SELECT ?, ?, id FROM files WHERE tenant_id = ? AND id = ?`, item.ID, i, tenantID, fileID)
		if err := affected(res, err, "file"); err != nil {
			return err
		}
	}
	return nil
}

func (r repos) AddInboxFiles(ctx context.Context, tenantID, itemID string, fileIDs []string) error {
	for _, fileID := range fileIDs {
		res, err := r.q.ExecContext(ctx, `INSERT INTO inbox_files (item_id, position, file_id)
			SELECT i.id, (SELECT COALESCE(MAX(position), -1) + 1 FROM inbox_files WHERE item_id = i.id), f.id
			FROM inbox_items i JOIN files f ON f.tenant_id = i.tenant_id AND f.id = ?
			WHERE i.tenant_id = ? AND i.id = ?`, fileID, tenantID, itemID)
		if err := affected(res, err, "file"); err != nil {
			return err
		}
	}
	return nil
}

const inboxSelect = `SELECT id, status, source, note, uploaded_by, COALESCE(document_id, ''), COALESCE(person_id, ''), created_at
	FROM inbox_items`

func scanInboxItem(scan func(...any) error) (domain.InboxItem, error) {
	var item domain.InboxItem
	var status, source, created string
	if err := scan(&item.ID, &status, &source, &item.Note, &item.UploadedBy, &item.DocumentID, &item.PersonID, &created); err != nil {
		return item, err
	}
	item.Status = domain.InboxStatus(status)
	item.Source = domain.InboxSource(source)
	var err error
	item.CreatedAt, err = parseTime(created)
	return item, err
}

func (r repos) inboxFiles(ctx context.Context, itemID string) ([]domain.StoredFile, error) {
	return r.filesOf(ctx, `SELECT `+fileColumns+` FROM inbox_files inf JOIN files f ON f.id = inf.file_id
		WHERE inf.item_id = ? ORDER BY inf.position`, itemID)
}

func (r repos) InboxItem(ctx context.Context, tenantID, id string) (domain.InboxItem, error) {
	item, err := scanInboxItem(r.q.QueryRowContext(ctx, inboxSelect+` WHERE tenant_id = ? AND id = ?`, tenantID, id).Scan)
	if err := one(err, "inbox item"); err != nil {
		return item, err
	}
	item.Files, err = r.inboxFiles(ctx, item.ID)
	return item, err
}

func (r repos) InboxItems(ctx context.Context, tenantID string, status domain.InboxStatus) ([]domain.InboxItem, error) {
	rows, err := r.q.QueryContext(ctx, inboxSelect+` WHERE tenant_id = ? AND status = ? ORDER BY created_at, id`, tenantID, string(status))
	if err != nil {
		return nil, fmt.Errorf("list inbox: %w", err)
	}
	var out []domain.InboxItem
	for rows.Next() {
		item, err := scanInboxItem(rows.Scan)
		if err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan inbox item: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("list inbox: %w", err)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list inbox: %w", err)
	}
	for i := range out {
		if out[i].Files, err = r.inboxFiles(ctx, out[i].ID); err != nil {
			return nil, err
		}
	}
	if out == nil {
		out = []domain.InboxItem{}
	}
	return out, nil
}

func (r repos) SetInboxStatus(ctx context.Context, tenantID, id string, status domain.InboxStatus, documentID string) error {
	res, err := r.q.ExecContext(ctx, `UPDATE inbox_items SET status = ?, document_id = ? WHERE tenant_id = ? AND id = ? AND status = 'new'`,
		string(status), nullable(documentID), tenantID, id)
	return affected(res, err, "inbox item")
}

func (r repos) UnfileDocument(ctx context.Context, tenantID, documentID string) (string, error) {
	var id string
	err := r.q.QueryRowContext(ctx, `UPDATE inbox_items SET status = 'new', document_id = NULL
		WHERE tenant_id = ? AND document_id = ? AND status = 'filed' RETURNING id`, tenantID, documentID).Scan(&id)
	return id, one(err, "inbox item")
}

func (r repos) NewItemsWithFile(ctx context.Context, tenantID, fileID string) ([]string, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT i.id FROM inbox_items i JOIN inbox_files inf ON inf.item_id = i.id
		WHERE i.tenant_id = ? AND inf.file_id = ? AND i.status = 'new' ORDER BY i.created_at`, tenantID, fileID)
	if err != nil {
		return nil, fmt.Errorf("list items of file: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan item id: %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (r repos) OCRProgress(ctx context.Context, tenantID, itemID string) (domain.OCRProgress, error) {
	var p domain.OCRProgress
	err := r.q.QueryRowContext(ctx, `SELECT COUNT(*),
			COALESCE(SUM(EXISTS (SELECT 1 FROM pages pg WHERE pg.file_id = inf.file_id)), 0),
			COALESCE(SUM(EXISTS (SELECT 1 FROM jobs j WHERE j.kind = 'ocr' AND j.subject_id = inf.file_id AND j.status = 'failed')), 0),
			COALESCE(MAX((SELECT j.last_error FROM jobs j WHERE j.kind = 'ocr' AND j.subject_id = inf.file_id AND j.status IN ('queued', 'failed'))), '')
		FROM inbox_files inf JOIN inbox_items i ON i.id = inf.item_id WHERE i.tenant_id = ? AND inf.item_id = ?`,
		tenantID, itemID).Scan(&p.Total, &p.Done, &p.Failed, &p.Error)
	if err != nil {
		return p, fmt.Errorf("read OCR progress: %w", err)
	}
	return p, nil
}
