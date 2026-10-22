package service

import (
	"context"
	"errors"
	"io"
	"slices"
	"sort"

	"apps/health-watcher/domain"
	"apps/health-watcher/port"
)

// ItemDetail is everything the review page shows for one inbox item: its
// pages with their masks, the reading, and the drafts (HW-READ-4).
type ItemDetail struct {
	Item         domain.InboxItem     `json:"item"`
	Pages        []domain.Page        `json:"pages"`
	Observations []domain.Observation `json:"observations"`
	Findings     []domain.Finding     `json:"findings"`
	AIEnabled    bool                 `json:"aiEnabled"`
}

// Item returns one unfiled item for review.
func (in *Inbox) Item(ctx context.Context, sc domain.Scope, itemID string) (ItemDetail, error) {
	if err := sc.RequireRead(); err != nil {
		return ItemDetail{}, err
	}
	item, err := in.Store.InboxItem(ctx, sc.TenantID, itemID)
	if err != nil {
		return ItemDetail{}, err
	}
	if err := in.withProgress(ctx, sc.TenantID, &item); err != nil {
		return ItemDetail{}, err
	}
	if err := in.refreshStaleMasks(ctx, sc.TenantID, item); err != nil {
		return ItemDetail{}, err
	}
	d := ItemDetail{Item: item, Pages: []domain.Page{}, AIEnabled: in.Pipeline != nil && in.Pipeline.AIEnabled()}
	for _, f := range item.Files {
		pages, err := in.Store.PagesOf(ctx, sc.TenantID, f.ID)
		if err != nil {
			return ItemDetail{}, err
		}
		d.Pages = append(d.Pages, pages...)
	}
	if d.Observations, err = in.Store.ObservationsOfItem(ctx, sc.TenantID, itemID); err != nil {
		return ItemDetail{}, err
	}
	if d.Findings, err = in.Store.FindingsOfItem(ctx, sc.TenantID, itemID); err != nil {
		return ItemDetail{}, err
	}
	return d, nil
}

// refreshStaleMasks masks again the pages of an unfiled item that were
// masked with older rules, so the review page shows what would be sent now.
func (in *Inbox) refreshStaleMasks(ctx context.Context, tenantID string, item domain.InboxItem) error {
	if in.Pipeline == nil || item.Status != domain.InboxNew {
		return nil
	}
	pages, _, err := in.Pipeline.itemPages(ctx, tenantID, item)
	if err != nil {
		return err
	}
	var names []string
	for _, ip := range pages {
		if ip.page.Mask == nil || ip.page.Mask.Version >= domain.MaskRulesVersion {
			continue
		}
		if names == nil {
			people, err := in.Store.People(ctx, tenantID)
			if err != nil {
				return err
			}
			names = peopleNames(people)
		}
		if _, err := in.Pipeline.maskPage(ctx, tenantID, ip, names); err != nil {
			return err
		}
	}
	return nil
}

// newItem returns an unfiled item, or a conflict when it is filed.
func (in *Inbox) newItem(ctx context.Context, sc domain.Scope, itemID string) (domain.InboxItem, error) {
	item, err := in.Store.InboxItem(ctx, sc.TenantID, itemID)
	if err != nil {
		return item, err
	}
	if item.Status != domain.InboxNew {
		return item, domain.Conflict("this item is already %s", item.Status)
	}
	return item, nil
}

func (in *Inbox) requirePipeline() error {
	if in.Pipeline == nil || !in.Pipeline.AIEnabled() {
		return domain.Conflict("AI reading is off: an engine admin turns it on in Engine settings")
	}
	return nil
}

// SendReading sends an item's pages after a person checked the masked
// previews (HW-AI-7). Pages without an image can't be sent at all.
func (in *Inbox) SendReading(ctx context.Context, sc domain.Scope, itemID string) error {
	if err := sc.RequireEdit(); err != nil {
		return err
	}
	if err := in.requirePipeline(); err != nil {
		return err
	}
	item, err := in.newItem(ctx, sc, itemID)
	if err != nil {
		return err
	}
	pages, ready, err := in.Pipeline.itemPages(ctx, sc.TenantID, item)
	if err != nil {
		return err
	}
	if !ready {
		return domain.Conflict("the pages are still being read on the Mac")
	}
	for _, ip := range pages {
		if !ip.page.HasImage {
			return domain.Conflict("page %d has no image; rebuild mac-ocr so it can render pages, then retry", ip.page.Number)
		}
	}
	for _, ip := range pages {
		m := domain.PageMask{Status: domain.MaskChecked}
		if ip.page.Mask != nil {
			m = *ip.page.Mask
			m.Status = domain.MaskChecked
		}
		if _, err := in.Store.SetPageMask(ctx, sc.TenantID, ip.page.ID, m); err != nil {
			return err
		}
	}
	return in.Pipeline.ScheduleReading(ctx, sc.TenantID, itemID, true)
}

// SkipReading turns AI reading off for an item, e.g. a film photo (D5) or
// a paper the user keeps private. Its drafts are dropped.
func (in *Inbox) SkipReading(ctx context.Context, sc domain.Scope, itemID string) error {
	if err := sc.RequireEdit(); err != nil {
		return err
	}
	if _, err := in.newItem(ctx, sc, itemID); err != nil {
		return err
	}
	return in.Store.InTx(ctx, func(r port.Repos) error {
		if err := r.DeleteItemDrafts(ctx, sc.TenantID, itemID); err != nil {
			return err
		}
		return r.SaveReading(ctx, sc.TenantID, domain.Reading{ItemID: itemID, Status: domain.ReadingSkipped,
			Reason: domain.SkipByUser, UpdatedAt: in.Clock()})
	})
}

// RetryReading reads an item again: OCR for files that have no pages yet,
// then the AI reading, after a failure, after Skip, or to replace the
// drafts (HW-READ-6).
func (in *Inbox) RetryReading(ctx context.Context, sc domain.Scope, itemID string) error {
	if err := sc.RequireEdit(); err != nil {
		return err
	}
	if in.Pipeline == nil {
		return domain.Conflict("the background reader is off")
	}
	item, err := in.newItem(ctx, sc, itemID)
	if err != nil {
		return err
	}
	ocr := 0
	for _, f := range item.Files {
		pages, err := in.Store.PagesOf(ctx, sc.TenantID, f.ID)
		if err != nil {
			return err
		}
		if len(pages) > 0 {
			continue
		}
		job := domain.Job{ID: NewID(), TenantID: sc.TenantID, Kind: domain.JobOCR, SubjectID: f.ID}
		if err := in.Store.EnqueueJob(ctx, job, in.Clock()); err != nil {
			return err
		}
		ocr++
	}
	if ocr > 0 {
		in.Pipeline.Wake()
	}
	if !in.Pipeline.AIEnabled() {
		if ocr == 0 {
			return in.requirePipeline()
		}
		return nil
	}
	return in.Pipeline.ScheduleReading(ctx, sc.TenantID, itemID, true)
}

// Unfile moves a filed document's item back to the inbox, so it can be
// read with AI and filed again (HW-READ-8). The document goes; its visit
// and files stay. A document with values or findings is refused, so no
// approved record is lost. An item read while AI was off is sent now.
func (in *Inbox) Unfile(ctx context.Context, sc domain.Scope, documentID string) (string, error) {
	if err := sc.RequireEdit(); err != nil {
		return "", err
	}
	var itemID string
	err := in.Store.InTx(ctx, func(r port.Repos) error {
		if _, err := r.Document(ctx, sc.TenantID, documentID); err != nil {
			return err
		}
		obs, err := r.ObservationsOfDocument(ctx, sc.TenantID, documentID)
		if err != nil {
			return err
		}
		finds, err := r.FindingsOfDocument(ctx, sc.TenantID, documentID)
		if err != nil {
			return err
		}
		if len(obs) > 0 || len(finds) > 0 {
			return domain.Conflict("this document has %d values and %d findings; delete them first", len(obs), len(finds))
		}
		itemID, err = r.UnfileDocument(ctx, sc.TenantID, documentID)
		if errors.Is(err, domain.ErrNotFound) {
			return domain.Conflict("this document was not filed from the inbox")
		}
		if err != nil {
			return err
		}
		return r.DeleteDocument(ctx, sc.TenantID, documentID)
	})
	if err != nil {
		return "", err
	}
	if in.Pipeline == nil {
		return itemID, nil
	}
	return itemID, in.Pipeline.ScheduleReading(ctx, sc.TenantID, itemID, false)
}

// MaxManualBoxes bounds the boxes a person draws on one page.
const MaxManualBoxes = 50

// SetPageBoxes stores the boxes a person drew on a page to black out what
// the rules missed, and remasks the page at once so the preview shows them.
func (in *Inbox) SetPageBoxes(ctx context.Context, sc domain.Scope, itemID, pageID string, boxes []domain.Quad) (domain.Page, error) {
	return in.remask(ctx, sc, itemID, pageID, func(m *domain.PageMask) error {
		if len(boxes) > MaxManualBoxes {
			return domain.Invalid("at most %d boxes per page", MaxManualBoxes)
		}
		for _, b := range boxes {
			if !b.Valid() {
				return domain.Invalid("a box is outside the page")
			}
		}
		m.Manual = boxes
		return nil
	})
}

// SetPageOff turns off the rule boxes with these keys on a page, and turns
// the others back on, when a box hides clinical text. The page is remasked
// at once so the preview and the text show the change.
func (in *Inbox) SetPageOff(ctx context.Context, sc domain.Scope, itemID, pageID string, off []string) (domain.Page, error) {
	return in.remask(ctx, sc, itemID, pageID, func(m *domain.PageMask) error {
		for _, key := range off {
			if !slices.ContainsFunc(m.Auto, func(b domain.MaskBox) bool { return b.Key == key }) {
				return domain.Invalid("no box %q on this page", key)
			}
		}
		for i := range m.Auto {
			m.Auto[i].Off = slices.Contains(off, m.Auto[i].Key)
		}
		return nil
	})
}

// remask changes a page's mask and masks the page again. A change to what
// is sent means a checked page must be checked again.
func (in *Inbox) remask(ctx context.Context, sc domain.Scope, itemID, pageID string, change func(*domain.PageMask) error) (domain.Page, error) {
	if err := sc.RequireEdit(); err != nil {
		return domain.Page{}, err
	}
	if in.Pipeline == nil {
		return domain.Page{}, domain.Conflict("page masks need the background reader")
	}
	item, err := in.newItem(ctx, sc, itemID)
	if err != nil {
		return domain.Page{}, err
	}
	pages, _, err := in.Pipeline.itemPages(ctx, sc.TenantID, item)
	if err != nil {
		return domain.Page{}, err
	}
	people, err := in.Store.People(ctx, sc.TenantID)
	if err != nil {
		return domain.Page{}, err
	}
	for _, ip := range pages {
		if ip.page.ID != pageID {
			continue
		}
		prev := domain.PageMask{}
		if ip.page.Mask != nil {
			prev = *ip.page.Mask
		}
		if err := change(&prev); err != nil {
			return domain.Page{}, err
		}
		if prev.Status == domain.MaskChecked {
			prev.Status = domain.MaskNeedsCheck
		}
		ip.page.Mask = &prev
		if _, err := in.Pipeline.maskPage(ctx, sc.TenantID, ip, peopleNames(people)); err != nil {
			return domain.Page{}, err
		}
		return in.Store.Page(ctx, sc.TenantID, pageID)
	}
	return domain.Page{}, domain.NotFound("page")
}

// OpenPageImage returns a page's rendered image, or the masked image the
// AI provider saw (HW-FIND-3, HW-AI-7).
func (s *Records) OpenPageImage(ctx context.Context, sc domain.Scope, pageID string, masked bool) (io.ReadCloser, error) {
	if err := sc.RequireRead(); err != nil {
		return nil, err
	}
	pg, err := s.Store.Page(ctx, sc.TenantID, pageID)
	if err != nil {
		return nil, err
	}
	sha := pg.ImageSHA
	if masked {
		sha = ""
		if pg.Mask != nil {
			sha = pg.Mask.ImageSHA
		}
	}
	if sha == "" {
		return nil, domain.NotFound("page image")
	}
	return s.Files.Open(ctx, sha)
}

// InboxGroup is a suggested visit in backlog mode: unfiled items read as
// one person, one date and one provider (HW-IMP-6).
type InboxGroup struct {
	PersonID     string              `json:"personId"`
	Date         string              `json:"date"`
	ProviderName string              `json:"providerName"`
	ProviderKind domain.ProviderKind `json:"providerKind"`
	VisitType    domain.VisitType    `json:"visitType"`
	Items        []domain.InboxItem  `json:"items"`
}

// Groups suggests one visit per person, date and provider among the read
// items. Items without a known person or date are left out.
func (in *Inbox) Groups(ctx context.Context, sc domain.Scope) ([]InboxGroup, error) {
	items, err := in.Items(ctx, sc)
	if err != nil {
		return nil, err
	}
	type key struct{ person, date, provider string }
	byKey := map[key]*InboxGroup{}
	var order []key
	for _, item := range items {
		rd := item.Reading
		if rd == nil || rd.Status != domain.ReadingDone || rd.Proposal == nil {
			continue
		}
		pr := rd.Proposal
		person := item.PersonID
		if person == "" {
			person = pr.PersonID
		}
		if person == "" || pr.Date == "" {
			continue
		}
		k := key{person, pr.Date, domain.FoldKey(pr.ProviderName)}
		g, ok := byKey[k]
		if !ok {
			g = &InboxGroup{PersonID: person, Date: pr.Date, ProviderName: pr.ProviderName, ProviderKind: pr.ProviderKind, VisitType: pr.VisitType}
			byKey[k] = g
			order = append(order, k)
		}
		g.Items = append(g.Items, item)
	}
	out := make([]InboxGroup, 0, len(order))
	for _, k := range order {
		out = append(out, *byKey[k])
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Date > out[j].Date })
	return out, nil
}

// GroupDocument is one item's document in a group filing.
type GroupDocument struct {
	ItemID string `json:"itemId"`
	Kind   string `json:"kind"`
	Title  string `json:"title"`
}

// GroupFileInput files several items as documents of one new visit.
type GroupFileInput struct {
	Visit     domain.VisitInput `json:"visit"`
	Documents []GroupDocument   `json:"documents"`
}

// FileGroup files a backlog group in one go: one new visit, one document
// per item, every remaining draft approved (HW-IMP-6).
func (in *Inbox) FileGroup(ctx context.Context, sc domain.Scope, req GroupFileInput) (domain.Visit, error) {
	if err := sc.RequireEdit(); err != nil {
		return domain.Visit{}, err
	}
	if len(req.Documents) == 0 {
		return domain.Visit{}, domain.Invalid("choose at least one item")
	}
	if len(req.Documents) > domain.MaxFilesPerItem {
		return domain.Visit{}, domain.Invalid("at most %d items at once", domain.MaxFilesPerItem)
	}
	type doc struct {
		itemID string
		kind   domain.DocumentKind
		title  string
	}
	docs := make([]doc, len(req.Documents))
	seen := map[string]bool{}
	for i, d := range req.Documents {
		if seen[d.ItemID] {
			return domain.Visit{}, domain.Invalid("an item is listed twice")
		}
		seen[d.ItemID] = true
		kind, title, err := docInput(d.Kind, d.Title)
		if err != nil {
			return domain.Visit{}, err
		}
		docs[i] = doc{d.ItemID, kind, title}
	}
	var visitID string
	err := in.Store.InTx(ctx, func(r port.Repos) error {
		visit, err := in.Records.createVisit(ctx, r, sc, req.Visit)
		if err != nil {
			return err
		}
		visitID = visit.ID
		for _, d := range docs {
			if _, err := in.fileItem(ctx, r, sc, d.itemID, visit, d.kind, d.title); err != nil {
				if errors.Is(err, domain.ErrNotFound) {
					return domain.NotFound("inbox item")
				}
				return err
			}
		}
		return nil
	})
	if err != nil {
		return domain.Visit{}, err
	}
	return in.Records.Visit(ctx, sc, visitID)
}
