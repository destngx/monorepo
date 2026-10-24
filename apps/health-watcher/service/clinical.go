package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"
	"unicode/utf8"

	"apps/health-watcher/domain"
	"apps/health-watcher/port"
)

// Clinical manages observations and findings: drafts from AI readings,
// approved records with their audit log, manual entry, metric links, the
// document viewer and search (HW-REC-3..6, HW-READ-4, HW-READ-5,
// HW-READ-7, HW-FIND-2, HW-FIND-3, HW-MET-1).
type Clinical struct {
	Deps
	Catalog *domain.Catalog
}

// audit records the old value of an approved record before it changes.
func (c *Clinical) audit(ctx context.Context, r port.Repos, sc domain.Scope, entity domain.AuditEntity, id, action string, version int,
	old any) error {
	return addAudit(ctx, r, c.Clock, sc, entity, id, action, version, old)
}

// addAudit records old, a record at version, before it changes.
func addAudit(ctx context.Context, r port.Repos, clock func() time.Time, sc domain.Scope, entity domain.AuditEntity, id, action string,
	version int, old any) error {
	raw, err := json.Marshal(old)
	if err != nil {
		return fmt.Errorf("encode audit entry: %w", err)
	}
	return r.AddAudit(ctx, sc.TenantID, domain.AuditEntry{ID: NewID(), Entity: entity, EntityID: id, Action: action, Version: version,
		UserID: sc.UserID, At: clock(), OldJSON: string(raw)})
}

// ObservationPatch changes an observation. Status moves a draft between
// draft and rejected; approved records change only by editing.
type ObservationPatch struct {
	domain.ObservationInput
	Status domain.RecordStatus `json:"status"`
}

func (c *Clinical) checkPage(ctx context.Context, r port.Repos, sc domain.Scope, pageID string) error {
	if pageID == "" {
		return nil
	}
	_, err := r.Page(ctx, sc.TenantID, pageID)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.Invalid("unknown page")
	}
	return err
}

func setStatus(current domain.RecordStatus, want domain.RecordStatus) (domain.RecordStatus, error) {
	switch {
	case want == "" || want == current:
		return current, nil
	case current == domain.StatusApproved:
		return "", domain.Invalid("an approved record can be edited or deleted, not moved back to a draft")
	case want == domain.StatusDraft || want == domain.StatusRejected:
		return want, nil
	}
	return "", domain.Invalid("status must be draft or rejected")
}

// UpdateObservation edits a draft or an approved observation. A person's
// edit sets the confidence to 100; editing an approved one is audited.
func (c *Clinical) UpdateObservation(ctx context.Context, sc domain.Scope, id string, patch ObservationPatch) (domain.Observation, error) {
	if err := sc.RequireEdit(); err != nil {
		return domain.Observation{}, err
	}
	catalog, err := c.catalogFor(ctx, sc.TenantID)
	if err != nil {
		return domain.Observation{}, err
	}
	in, err := patch.Validate(c.today(), catalog)
	if err != nil {
		return domain.Observation{}, err
	}
	err = c.Store.InTx(ctx, func(r port.Repos) error {
		old, err := r.Observation(ctx, sc.TenantID, id)
		if err != nil {
			return err
		}
		status, err := setStatus(old.Status, patch.Status)
		if err != nil {
			return err
		}
		if err := c.checkPage(ctx, r, sc, in.PageID); err != nil {
			return err
		}
		next := observationFrom(in, c.Clock())
		next.ID, next.Status, next.ItemID, next.DocumentID, next.PersonID = old.ID, status, old.ItemID, old.DocumentID, old.PersonID
		next.CreatedAt, next.Confidence, next.Provenance = old.CreatedAt, 100, nextVersion(old.Provenance, old.Status)
		if in.PageID == "" {
			next.PageID = old.PageID
		}
		if in.SampleDate == "" {
			next.SampleDate = old.SampleDate
		}
		if old.Status == domain.StatusApproved {
			if err := c.audit(ctx, r, sc, domain.AuditObservation, id, "update", old.Version, old); err != nil {
				return err
			}
		}
		return r.UpdateObservation(ctx, sc.TenantID, next)
	})
	if err != nil {
		return domain.Observation{}, err
	}
	return c.Store.Observation(ctx, sc.TenantID, id)
}

// DeleteObservation deletes an observation; deleting an approved one is audited.
func (c *Clinical) DeleteObservation(ctx context.Context, sc domain.Scope, id string) error {
	if err := sc.RequireEdit(); err != nil {
		return err
	}
	return c.Store.InTx(ctx, func(r port.Repos) error {
		old, err := r.Observation(ctx, sc.TenantID, id)
		if err != nil {
			return err
		}
		if old.Status == domain.StatusApproved {
			if err := c.audit(ctx, r, sc, domain.AuditObservation, id, "delete", old.Version, old); err != nil {
				return err
			}
		}
		return r.DeleteObservation(ctx, sc.TenantID, id)
	})
}

// documentOwner returns a document's person and visit date.
func documentOwner(ctx context.Context, r port.Repos, tenantID, documentID string) (domain.Document, domain.Visit, error) {
	doc, err := r.Document(ctx, tenantID, documentID)
	if err != nil {
		return doc, domain.Visit{}, err
	}
	visit, err := r.Visit(ctx, tenantID, doc.VisitID)
	return doc, visit, err
}

// AddObservation types an observation into a filed document by hand; it
// is approved at once (HW-READ-7).
func (c *Clinical) AddObservation(ctx context.Context, sc domain.Scope, documentID string, input domain.ObservationInput) (domain.Observation, error) {
	if err := sc.RequireEdit(); err != nil {
		return domain.Observation{}, err
	}
	catalog, err := c.catalogFor(ctx, sc.TenantID)
	if err != nil {
		return domain.Observation{}, err
	}
	in, err := input.Validate(c.today(), catalog)
	if err != nil {
		return domain.Observation{}, err
	}
	if in.MetricCode == "" {
		if m, ok := catalog.Match(in.Analyte, in.Unit, in.Value); ok {
			in.MetricCode = m.Code
		}
	}
	o := observationFrom(in, c.Clock())
	err = c.Store.InTx(ctx, func(r port.Repos) error {
		doc, visit, err := documentOwner(ctx, r, sc.TenantID, documentID)
		if err != nil {
			return err
		}
		if err := c.checkPage(ctx, r, sc, in.PageID); err != nil {
			return err
		}
		o.ID, o.Status, o.DocumentID, o.PersonID, o.Confidence = NewID(), domain.StatusApproved, doc.ID, doc.PersonID, 100
		o.Provenance = manualProvenance(sc, o.CreatedAt)
		if o.SampleDate == "" {
			o.SampleDate = visit.Date
		}
		return r.CreateObservation(ctx, sc.TenantID, o)
	})
	if err != nil {
		return domain.Observation{}, err
	}
	return c.Store.Observation(ctx, sc.TenantID, o.ID)
}

// FindingPatch changes a finding; Status works as for observations.
type FindingPatch struct {
	domain.FindingInput
	Status domain.RecordStatus `json:"status"`
}

// UpdateFinding edits a draft or an approved finding; approved edits are audited.
func (c *Clinical) UpdateFinding(ctx context.Context, sc domain.Scope, id string, patch FindingPatch) (domain.Finding, error) {
	if err := sc.RequireEdit(); err != nil {
		return domain.Finding{}, err
	}
	in, err := patch.Validate()
	if err != nil {
		return domain.Finding{}, err
	}
	err = c.Store.InTx(ctx, func(r port.Repos) error {
		old, err := r.Finding(ctx, sc.TenantID, id)
		if err != nil {
			return err
		}
		status, err := setStatus(old.Status, patch.Status)
		if err != nil {
			return err
		}
		if err := c.checkPage(ctx, r, sc, in.PageID); err != nil {
			return err
		}
		next := findingFrom(in, c.Clock())
		next.ID, next.Status, next.ItemID, next.DocumentID, next.PersonID = old.ID, status, old.ItemID, old.DocumentID, old.PersonID
		next.CreatedAt, next.Confidence, next.Provenance = old.CreatedAt, 100, nextVersion(old.Provenance, old.Status)
		if in.PageID == "" {
			next.PageID = old.PageID
		}
		if old.Status == domain.StatusApproved {
			if err := c.audit(ctx, r, sc, domain.AuditFinding, id, "update", old.Version, old); err != nil {
				return err
			}
		}
		return r.UpdateFinding(ctx, sc.TenantID, next)
	})
	if err != nil {
		return domain.Finding{}, err
	}
	return c.Store.Finding(ctx, sc.TenantID, id)
}

// DeleteFinding deletes a finding; deleting an approved one is audited.
func (c *Clinical) DeleteFinding(ctx context.Context, sc domain.Scope, id string) error {
	if err := sc.RequireEdit(); err != nil {
		return err
	}
	return c.Store.InTx(ctx, func(r port.Repos) error {
		old, err := r.Finding(ctx, sc.TenantID, id)
		if err != nil {
			return err
		}
		if old.Status == domain.StatusApproved {
			if err := c.audit(ctx, r, sc, domain.AuditFinding, id, "delete", old.Version, old); err != nil {
				return err
			}
		}
		return r.DeleteFinding(ctx, sc.TenantID, id)
	})
}

// AddFinding types a finding into a filed document by hand (HW-READ-7).
func (c *Clinical) AddFinding(ctx context.Context, sc domain.Scope, documentID string, input domain.FindingInput) (domain.Finding, error) {
	if err := sc.RequireEdit(); err != nil {
		return domain.Finding{}, err
	}
	in, err := input.Validate()
	if err != nil {
		return domain.Finding{}, err
	}
	f := findingFrom(in, c.Clock())
	err = c.Store.InTx(ctx, func(r port.Repos) error {
		doc, _, err := documentOwner(ctx, r, sc.TenantID, documentID)
		if err != nil {
			return err
		}
		if err := c.checkPage(ctx, r, sc, in.PageID); err != nil {
			return err
		}
		f.ID, f.Status, f.DocumentID, f.PersonID, f.Confidence = NewID(), domain.StatusApproved, doc.ID, doc.PersonID, 100
		f.Provenance = manualProvenance(sc, f.CreatedAt)
		return r.CreateFinding(ctx, sc.TenantID, f)
	})
	if err != nil {
		return domain.Finding{}, err
	}
	return c.Store.Finding(ctx, sc.TenantID, f.ID)
}

// observationProvider returns the provider whose papers an observation
// came from: its visit's provider once filed, else the reading's proposal.
func observationProvider(ctx context.Context, r port.Repos, tenantID string, o domain.Observation) (string, error) {
	if o.DocumentID != "" {
		_, visit, err := documentOwner(ctx, r, tenantID, o.DocumentID)
		if err != nil {
			return "", err
		}
		if visit.Provider != nil {
			return visit.Provider.ID, nil
		}
		return "", nil
	}
	if o.ItemID == "" {
		return "", nil
	}
	rd, err := r.Reading(ctx, tenantID, o.ItemID)
	if errors.Is(err, domain.ErrNotFound) || (err == nil && (rd.Proposal == nil || rd.Proposal.ProviderName == "")) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	pr, err := r.ProviderByKey(ctx, tenantID, domain.FoldKey(rd.Proposal.ProviderName))
	if errors.Is(err, domain.ErrNotFound) {
		return "", nil
	}
	return pr.ID, err
}

// LinkMetric links an observation to a catalog entry, or unlinks it with
// "", and remembers the link for that analyte name and provider so later
// readings use it (HW-READ-5).
func (c *Clinical) LinkMetric(ctx context.Context, sc domain.Scope, id, code string) (domain.Observation, error) {
	if err := sc.RequireEdit(); err != nil {
		return domain.Observation{}, err
	}
	if code != "" {
		catalog, err := c.catalogFor(ctx, sc.TenantID)
		if err != nil {
			return domain.Observation{}, err
		}
		if _, ok := catalog.Metric(code); !ok {
			return domain.Observation{}, domain.Invalid("unknown metric %q", code)
		}
	}
	err := c.Store.InTx(ctx, func(r port.Repos) error {
		o, err := r.Observation(ctx, sc.TenantID, id)
		if err != nil {
			return err
		}
		if o.Status == domain.StatusApproved {
			if err := c.audit(ctx, r, sc, domain.AuditObservation, id, "update", o.Version, o); err != nil {
				return err
			}
		}
		o.Provenance = nextVersion(o.Provenance, o.Status)
		providerID, err := observationProvider(ctx, r, sc.TenantID, o)
		if err != nil {
			return err
		}
		if code != "" {
			if err := r.SetAnalyteLink(ctx, sc.TenantID, providerID, domain.AnalyteKey(o.Analyte), code); err != nil {
				return err
			}
		}
		o.MetricCode, o.UpdatedAt = code, c.Clock()
		return r.UpdateObservation(ctx, sc.TenantID, o)
	})
	if err != nil {
		return domain.Observation{}, err
	}
	return c.Store.Observation(ctx, sc.TenantID, id)
}

// History returns the audit log of an observation or finding (HW-REC-6).
func (c *Clinical) History(ctx context.Context, sc domain.Scope, entity domain.AuditEntity, id string) ([]domain.AuditEntry, error) {
	if err := sc.RequireRead(); err != nil {
		return nil, err
	}
	return c.Store.AuditOf(ctx, sc.TenantID, entity, id)
}

// Metrics returns the catalog with the tenant's custom metrics (HW-MET-1,
// HW-MET-8).
func (c *Clinical) Metrics(ctx context.Context, sc domain.Scope) ([]domain.Metric, error) {
	if err := sc.RequireRead(); err != nil {
		return nil, err
	}
	catalog, err := c.catalogFor(ctx, sc.TenantID)
	if err != nil {
		return nil, err
	}
	return catalog.Metrics(), nil
}

// DocumentDetail is the document viewer's data (HW-FIND-3).
type DocumentDetail struct {
	Document     domain.Document      `json:"document"`
	Visit        domain.Visit         `json:"visit"`
	Pages        []domain.Page        `json:"pages"`
	Observations []domain.Observation `json:"observations"`
	Findings     []domain.Finding     `json:"findings"`
	// Pictures are the pictures cut out of the pages (HW-READ-11).
	Pictures []domain.Picture `json:"pictures"`
}

// Document returns a filed document with its pages, observations and findings.
func (c *Clinical) Document(ctx context.Context, sc domain.Scope, id string) (DocumentDetail, error) {
	if err := sc.RequireRead(); err != nil {
		return DocumentDetail{}, err
	}
	doc, err := c.Store.Document(ctx, sc.TenantID, id)
	if err != nil {
		return DocumentDetail{}, err
	}
	d := DocumentDetail{Document: doc, Pages: []domain.Page{}}
	if d.Visit, err = c.Store.Visit(ctx, sc.TenantID, doc.VisitID); err != nil {
		return DocumentDetail{}, err
	}
	for _, f := range doc.Files {
		pages, err := c.Store.PagesOf(ctx, sc.TenantID, f.ID)
		if err != nil {
			return DocumentDetail{}, err
		}
		d.Pages = append(d.Pages, pages...)
	}
	if d.Observations, err = c.Store.ObservationsOfDocument(ctx, sc.TenantID, id); err != nil {
		return DocumentDetail{}, err
	}
	if d.Findings, err = c.Store.FindingsOfDocument(ctx, sc.TenantID, id); err != nil {
		return DocumentDetail{}, err
	}
	if d.Pictures, err = c.Store.PicturesOf(ctx, sc.TenantID, pageIDs(d.Pages)); err != nil {
		return DocumentDetail{}, err
	}
	return d, nil
}

// RemoveDocumentPage takes one page out of a filed document, with the
// values and findings read from it, each approved one audited (HW-FIND-3,
// HW-REC-6). The file's other pages stay. A file left with no pages leaves
// the document, and goes when nothing else uses it. The document's last
// page is refused: deleting the document removes it.
func (c *Clinical) RemoveDocumentPage(ctx context.Context, sc domain.Scope, documentID, pageID string) (DocumentDetail, error) {
	if err := sc.RequireEdit(); err != nil {
		return DocumentDetail{}, err
	}
	var released []domain.StoredFile
	var pictures []string
	err := c.Store.InTx(ctx, func(r port.Repos) error {
		doc, err := r.Document(ctx, sc.TenantID, documentID)
		if err != nil {
			return err
		}
		pg, err := r.Page(ctx, sc.TenantID, pageID)
		if err != nil {
			return err
		}
		i := slices.IndexFunc(doc.Files, func(f domain.StoredFile) bool { return f.ID == pg.FileID })
		if i < 0 {
			return domain.NotFound("page")
		}
		shown, ofFile := 0, 0
		for _, f := range doc.Files {
			pages, err := r.PagesOf(ctx, sc.TenantID, f.ID)
			if err != nil {
				return err
			}
			shown += len(pages)
			if f.ID == pg.FileID {
				ofFile = len(pages)
			}
		}
		if shown <= 1 {
			return domain.Conflict("this is the document's only page; delete the document instead")
		}
		if err := c.deletePageRecords(ctx, r, sc, documentID, pageID); err != nil {
			return err
		}
		if err := r.RemovePage(ctx, sc.TenantID, pageID, c.Clock()); err != nil {
			return err
		}
		if pictures, err = r.SetPagePictures(ctx, sc.TenantID, []string{pageID}, nil); err != nil {
			return err
		}
		if ofFile > 1 {
			return nil
		}
		released = append(released, doc.Files[i])
		return r.UnlinkDocumentFile(ctx, sc.TenantID, documentID, pg.FileID)
	})
	if err != nil {
		return DocumentDetail{}, err
	}
	if err := releaseFiles(ctx, c.Deps, sc.TenantID, released); err != nil {
		return DocumentDetail{}, err
	}
	if err := releaseImages(ctx, c.Deps, pictures); err != nil {
		return DocumentDetail{}, err
	}
	return c.Document(ctx, sc, documentID)
}

// deletePageRecords deletes a document's values and findings read from a
// page, auditing each approved one.
func (c *Clinical) deletePageRecords(ctx context.Context, r port.Repos, sc domain.Scope, documentID, pageID string) error {
	obs, err := r.ObservationsOfDocument(ctx, sc.TenantID, documentID)
	if err != nil {
		return err
	}
	for _, o := range obs {
		if o.PageID != pageID {
			continue
		}
		if o.Status == domain.StatusApproved {
			if err := c.audit(ctx, r, sc, domain.AuditObservation, o.ID, "delete", o.Version, o); err != nil {
				return err
			}
		}
		if err := r.DeleteObservation(ctx, sc.TenantID, o.ID); err != nil {
			return err
		}
	}
	finds, err := r.FindingsOfDocument(ctx, sc.TenantID, documentID)
	if err != nil {
		return err
	}
	for _, f := range finds {
		if f.PageID != pageID {
			continue
		}
		if f.Status == domain.StatusApproved {
			if err := c.audit(ctx, r, sc, domain.AuditFinding, f.ID, "delete", f.Version, f); err != nil {
				return err
			}
		}
		if err := r.DeleteFinding(ctx, sc.TenantID, f.ID); err != nil {
			return err
		}
	}
	return nil
}

// DocumentInput changes a document's kind, title and printed date
// (HW-READ-7). A nil Date keeps the current one; "" clears it.
type DocumentInput struct {
	Kind  string  `json:"kind"`
	Title string  `json:"title"`
	Date  *string `json:"date"`
}

// UpdateDocument changes a filed document's kind and title.
func (c *Clinical) UpdateDocument(ctx context.Context, sc domain.Scope, id string, in DocumentInput) (DocumentDetail, error) {
	if err := sc.RequireEdit(); err != nil {
		return DocumentDetail{}, err
	}
	kind, title, err := docInput(in.Kind, in.Title)
	if err != nil {
		return DocumentDetail{}, err
	}
	date := ""
	if in.Date != nil && *in.Date != "" {
		if date, err = domain.Date(*in.Date); err != nil {
			return DocumentDetail{}, domain.Invalid("document date must be YYYY-MM-DD")
		}
		if date > c.today() {
			return DocumentDetail{}, domain.Invalid("document date is in the future")
		}
	}
	err = c.Store.InTx(ctx, func(r port.Repos) error {
		doc, err := r.Document(ctx, sc.TenantID, id)
		if err != nil {
			return err
		}
		doc.Kind, doc.Title = kind, title
		if in.Date != nil {
			doc.Date = date
		}
		return r.UpdateDocument(ctx, sc.TenantID, doc)
	})
	if err != nil {
		return DocumentDetail{}, err
	}
	return c.Document(ctx, sc, id)
}

// ConfirmApprovals records the caller as the approver of a document's
// approved records that have none, the records approved before D9, so
// they can be exported (GapApprover). The values stay as they are, so no
// version is added.
func (c *Clinical) ConfirmApprovals(ctx context.Context, sc domain.Scope, documentID string) (DocumentDetail, error) {
	if err := sc.RequireEdit(); err != nil {
		return DocumentDetail{}, err
	}
	err := c.Store.InTx(ctx, func(r port.Repos) error {
		if _, err := r.Document(ctx, sc.TenantID, documentID); err != nil {
			return err
		}
		_, err := r.ConfirmApprovals(ctx, sc.TenantID, documentID, sc.UserID, c.Clock())
		return err
	})
	if err != nil {
		return DocumentDetail{}, err
	}
	return c.Document(ctx, sc, documentID)
}

// auditDocumentRecords audits every approved record of a document before
// the document and its records are deleted (HW-REC-6).
func auditDocumentRecords(ctx context.Context, r port.Repos, clock func() time.Time, sc domain.Scope, documentID string) error {
	obs, err := r.ObservationsOfDocument(ctx, sc.TenantID, documentID)
	if err != nil {
		return err
	}
	for _, o := range obs {
		if err := addAudit(ctx, r, clock, sc, domain.AuditObservation, o.ID, "delete", o.Version, o); err != nil {
			return err
		}
	}
	finds, err := r.FindingsOfDocument(ctx, sc.TenantID, documentID)
	if err != nil {
		return err
	}
	for _, f := range finds {
		if err := addAudit(ctx, r, clock, sc, domain.AuditFinding, f.ID, "delete", f.Version, f); err != nil {
			return err
		}
	}
	return nil
}

// MaxSearchResults bounds one search.
const MaxSearchResults = 50

// SearchResult is the answer to a search.
type SearchResult struct {
	Query string             `json:"query"`
	Hits  []domain.SearchHit `json:"hits"`
}

// Search finds pages, findings and visits by their words, ignoring case
// and accents (HW-FIND-2).
func (c *Clinical) Search(ctx context.Context, sc domain.Scope, q string) (SearchResult, error) {
	if err := sc.RequireRead(); err != nil {
		return SearchResult{}, err
	}
	if utf8.RuneCountInString(q) > 200 {
		return SearchResult{}, domain.Invalid("the search is longer than 200 characters")
	}
	expr, terms, err := domain.SearchQuery(q)
	if err != nil {
		return SearchResult{}, err
	}
	rows, err := c.Store.Search(ctx, sc.TenantID, expr, MaxSearchResults)
	if err != nil {
		return SearchResult{}, err
	}
	res := SearchResult{Query: q, Hits: make([]domain.SearchHit, len(rows))}
	for i, row := range rows {
		hit := row.Hit
		hit.Snippet = domain.Snippet(row.Body, terms, 80)
		res.Hits[i] = hit
	}
	return res, nil
}
