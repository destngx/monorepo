package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"slices"
	"sync"
	"time"
	"unicode/utf8"

	"apps/health-watcher/domain"
	"apps/health-watcher/port"
)

// Limits are the upload limits (HW-IMP-2).
type Limits struct {
	MaxFileBytes  int64
	MaxBatchBytes int64
}

// DefaultLimits are the limits when settings leave them unset.
var DefaultLimits = Limits{
	MaxFileBytes:  domain.DefaultMaxFileMB << 20,
	MaxBatchBytes: domain.DefaultMaxBatchMB << 20,
}

// Inbox receives uploads and files them under visits (HW-IMP-1..4), with
// their AI readings as drafts (HW-READ-3, HW-READ-4) and backlog groups
// (HW-IMP-6).
type Inbox struct {
	Deps
	Records *Records
	// Limits may change while the engine runs; see SetLimits.
	Limits Limits
	// Pipeline reads uploads in the background; nil in tests that only file.
	Pipeline *Pipeline

	limitsMu sync.RWMutex
}

// SetLimits changes the upload limits for the next upload (HW-OPS-9).
func (in *Inbox) SetLimits(l Limits) {
	in.limitsMu.Lock()
	in.Limits = l
	in.limitsMu.Unlock()
}

func (in *Inbox) limits() Limits {
	in.limitsMu.RLock()
	defer in.limitsMu.RUnlock()
	return in.Limits
}

// UploadRequest is one batch of files.
type UploadRequest struct {
	Source domain.InboxSource
	Note   string
	Files  []domain.Upload
	// OneItem keeps every new file of the batch in a single item, as pages
	// of one document. Otherwise each new file is its own item.
	OneItem bool
	// PersonID is the backlog person every new item belongs to (HW-IMP-6).
	PersonID string
	// FileIDs are files the same user staged ahead of Send (HW-IMP-11);
	// they join the new items after the uploaded ones.
	FileIDs []string
}

// UploadResponse lists the items created and what happened to each file.
type UploadResponse struct {
	Items []domain.InboxItem    `json:"items"`
	Files []domain.UploadResult `json:"files"`
}

func (in *Inbox) checkBatch(req UploadRequest) error {
	if len(req.Files)+len(req.FileIDs) == 0 {
		return domain.Invalid("no files received")
	}
	if len(req.Files)+len(req.FileIDs) > domain.MaxFilesPerItem {
		return domain.Invalid("at most %d files per upload", domain.MaxFilesPerItem)
	}
	if utf8.RuneCountInString(req.Note) > domain.MaxNoteLength {
		return domain.Invalid("note is longer than %d characters", domain.MaxNoteLength)
	}
	limits := in.limits()
	var total int64
	for _, f := range req.Files {
		size := int64(len(f.Data))
		if size == 0 {
			return domain.Invalid("%s is empty", displayName(f.Name))
		}
		if size > limits.MaxFileBytes {
			return domain.Invalid("%s is larger than %d MB", displayName(f.Name), limits.MaxFileBytes>>20)
		}
		total += size
	}
	if total > limits.MaxBatchBytes {
		return domain.Invalid("the upload is larger than %d MB", limits.MaxBatchBytes>>20)
	}
	return nil
}

func displayName(name string) string {
	if name == "" {
		return "a file"
	}
	return name
}

// cleanName keeps only the base name, so a client path never leaks into storage.
func cleanName(name string) string {
	base := filepath.Base(filepath.Clean("/" + name))
	if base == "/" || base == "." {
		return ""
	}
	if utf8.RuneCountInString(base) > 200 {
		return string([]rune(base)[:200])
	}
	return base
}

// Upload stores a batch as new items. A file the tenant already has is
// reported as a duplicate and left out of the new items (HW-IMP-1).
func (in *Inbox) Upload(ctx context.Context, sc domain.Scope, req UploadRequest) (UploadResponse, error) {
	if err := sc.RequireUpload(); err != nil {
		return UploadResponse{}, err
	}
	if sc.ItemID != "" {
		return UploadResponse{}, domain.Forbidden("start new items: this link adds to one item")
	}
	if err := in.checkBatch(req); err != nil {
		return UploadResponse{}, err
	}
	if req.PersonID != "" {
		if _, err := in.Store.Person(ctx, sc.TenantID, req.PersonID); errors.Is(err, domain.ErrNotFound) {
			return UploadResponse{}, domain.Invalid("unknown person")
		} else if err != nil {
			return UploadResponse{}, err
		}
	}
	var resp UploadResponse
	results, err := in.storeFiles(ctx, sc, req.Files, req.FileIDs, func(r port.Repos, newIDs []string) error {
		if err := r.UnstageFiles(ctx, sc.TenantID, newIDs); err != nil {
			return err
		}
		resp.Items = []domain.InboxItem{}
		groups := [][]string{}
		if req.OneItem && len(newIDs) > 0 {
			groups = append(groups, newIDs)
		} else {
			for _, id := range newIDs {
				groups = append(groups, []string{id})
			}
		}
		for _, ids := range groups {
			item := domain.InboxItem{ID: NewID(), Status: domain.InboxNew, Source: req.Source, Note: req.Note,
				UploadedBy: sc.UserID, PersonID: req.PersonID, CreatedAt: in.Clock()}
			if err := r.CreateInboxItem(ctx, sc.TenantID, item, ids); err != nil {
				return err
			}
			stored, err := r.InboxItem(ctx, sc.TenantID, item.ID)
			if err != nil {
				return err
			}
			resp.Items = append(resp.Items, stored)
		}
		return nil
	})
	if err != nil {
		return UploadResponse{}, err
	}
	resp.Files = results
	return resp, in.readStaged(ctx, sc.TenantID, resp.Items, req.FileIDs)
}

// AddFiles adds files to an unfiled item (HW-IMP-10), after the ones it has, such as
// the X-ray image or a page that came later. The item is read again once
// the new files have pages; a reading turned off by a person stays off.
func (in *Inbox) AddFiles(ctx context.Context, sc domain.Scope, itemID string, uploads []domain.Upload, fileIDs []string) (UploadResponse, error) {
	if err := sc.RequireAddTo(itemID); err != nil {
		return UploadResponse{}, err
	}
	if err := in.checkBatch(UploadRequest{Files: uploads, FileIDs: fileIDs}); err != nil {
		return UploadResponse{}, err
	}
	var resp UploadResponse
	results, err := in.storeFiles(ctx, sc, uploads, fileIDs, func(r port.Repos, newIDs []string) error {
		if err := r.UnstageFiles(ctx, sc.TenantID, newIDs); err != nil {
			return err
		}
		item, err := r.InboxItem(ctx, sc.TenantID, itemID)
		if err != nil {
			return err
		}
		if item.Status != domain.InboxNew {
			return domain.Conflict("this item is already %s", item.Status)
		}
		if len(item.Files)+len(newIDs) > domain.MaxFilesPerItem {
			return domain.Invalid("an item holds at most %d files", domain.MaxFilesPerItem)
		}
		if len(newIDs) > 0 {
			if err := r.AddInboxFiles(ctx, sc.TenantID, itemID, newIDs); err != nil {
				return err
			}
			if err := in.readingWaits(ctx, r, sc.TenantID, itemID); err != nil {
				return err
			}
		}
		stored, err := r.InboxItem(ctx, sc.TenantID, itemID)
		resp.Items = []domain.InboxItem{stored}
		return err
	})
	if err != nil {
		return UploadResponse{}, err
	}
	resp.Files = results
	return resp, in.readStaged(ctx, sc.TenantID, resp.Items, fileIDs)
}

// StagedLifetime is how long a file sent ahead waits for Send.
const StagedLifetime = 24 * time.Hour

// Stage stores files a phone sends ahead, one photo at a time while the
// person takes the next (HW-IMP-11). Their text is read at once; Send then
// puts them in an item by id. A file the user staged before comes back as
// it was, so a page reload can send it again.
func (in *Inbox) Stage(ctx context.Context, sc domain.Scope, uploads []domain.Upload) ([]domain.UploadResult, error) {
	if err := sc.RequireUpload(); err != nil {
		return nil, err
	}
	if err := in.checkBatch(UploadRequest{Files: uploads}); err != nil {
		return nil, err
	}
	if err := in.DropStaged(ctx); err != nil {
		return nil, err
	}
	return in.storeFiles(ctx, sc, uploads, nil, func(r port.Repos, newIDs []string) error {
		for _, id := range newIDs {
			if err := r.StageFile(ctx, sc.TenantID, sc.UserID, id, in.Clock()); err != nil {
				return err
			}
		}
		return nil
	})
}

// DropStaged deletes the files staged longer than StagedLifetime and never
// sent, with their pages.
func (in *Inbox) DropStaged(ctx context.Context) error {
	old, err := in.Store.StagedBefore(ctx, in.Clock().Add(-StagedLifetime))
	if err != nil {
		return err
	}
	for _, tf := range old {
		if err := in.Store.UnstageFiles(ctx, tf.TenantID, []string{tf.FileID}); err != nil {
			return err
		}
		f, err := in.Store.File(ctx, tf.TenantID, tf.FileID)
		if errors.Is(err, domain.ErrNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		if err := releaseFiles(ctx, in.Deps, tf.TenantID, []domain.StoredFile{f}); err != nil {
			return err
		}
	}
	return nil
}

// readStaged starts reading the items that got staged files: their OCR may
// be done already, so no OCR job will start it.
func (in *Inbox) readStaged(ctx context.Context, tenantID string, items []domain.InboxItem, staged []string) error {
	if len(staged) == 0 || in.Pipeline == nil {
		return nil
	}
	for _, item := range items {
		if err := in.Pipeline.ScheduleReading(ctx, tenantID, item.ID, false); err != nil {
			return err
		}
	}
	return nil
}

// readingWaits marks an item's reading as waiting for the OCR of new files,
// unless a person turned it off.
func (in *Inbox) readingWaits(ctx context.Context, r port.Repos, tenantID, itemID string) error {
	rd, err := r.Reading(ctx, tenantID, itemID)
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return nil
	case err != nil:
		return err
	case skippedByUser(rd):
		return nil
	}
	rd.Status, rd.Reason, rd.Error, rd.UpdatedAt = domain.ReadingWaiting, "", "", in.Clock()
	return r.SaveReading(ctx, tenantID, rd)
}

func skippedByUser(rd domain.Reading) bool {
	return rd.Status == domain.ReadingSkipped && rd.Reason == domain.SkipByUser
}

// storeFiles stores a batch of uploads and queues OCR for each new one. A
// file the tenant already has is reported as a duplicate and left out of
// newIDs (HW-IMP-1). place puts the new files in items inside the same
// transaction. File bytes are written first, then every row, so a failure
// leaves no half-stored batch behind.
func (in *Inbox) storeFiles(ctx context.Context, sc domain.Scope, uploads []domain.Upload, staged []string,
	place func(r port.Repos, newIDs []string) error) ([]domain.UploadResult, error) {
	files := make([]domain.StoredFile, len(uploads))
	for i, up := range uploads {
		mime, err := domain.SniffMIME(up.Data)
		if err != nil {
			return nil, domain.Invalid("%s: only PDF, JPEG, PNG, WebP and HEIC files are accepted", displayName(up.Name))
		}
		sum := sha256.Sum256(up.Data)
		files[i] = domain.StoredFile{ID: NewID(), SHA256: hex.EncodeToString(sum[:]), MIME: mime, Size: int64(len(up.Data)),
			OriginalName: cleanName(up.Name), CreatedAt: in.Clock()}
	}
	in.FileLock.Lock()
	defer in.FileLock.Unlock()
	// Put is idempotent and cheap for bytes already on disk, so every file
	// is written; under the lock no release can remove them before commit.
	var written []string
	for i, f := range files {
		if err := in.Files.Put(ctx, f.SHA256, uploads[i].Data); err != nil {
			return nil, errors.Join(err, in.dropOrphans(ctx, written))
		}
		written = append(written, f.SHA256)
	}

	var results []domain.UploadResult
	err := in.Store.InTx(ctx, func(r port.Repos) error {
		results = []domain.UploadResult{}
		var newIDs []string
		seen := map[string]domain.StoredFile{}
		for i, f := range files {
			if first, ok := seen[f.SHA256]; ok {
				results = append(results, domain.UploadResult{Name: cleanName(uploads[i].Name), File: first, Duplicate: true})
				continue
			}
			existing, err := r.FileBySHA(ctx, sc.TenantID, f.SHA256)
			mine := false
			if err == nil {
				if mine, err = r.StagedBy(ctx, sc.TenantID, sc.UserID, existing.ID); err != nil {
					return err
				}
			}
			switch {
			case mine:
				// Sent ahead by this user: not a duplicate, but not stored twice.
				results = append(results, domain.UploadResult{Name: f.OriginalName, File: existing})
				seen[f.SHA256] = existing
				newIDs = append(newIDs, existing.ID)
				continue
			case err == nil:
				results = append(results, domain.UploadResult{Name: f.OriginalName, File: existing, Duplicate: true})
				seen[f.SHA256] = existing
				continue
			case !errors.Is(err, domain.ErrNotFound):
				return err
			}
			if err := r.CreateFile(ctx, sc.TenantID, f); err != nil {
				return err
			}
			// Every new file is read in the background (HW-READ-1).
			if err := r.EnqueueJob(ctx, domain.Job{ID: NewID(), TenantID: sc.TenantID, Kind: domain.JobOCR, SubjectID: f.ID}, in.Clock()); err != nil {
				return err
			}
			seen[f.SHA256] = f
			results = append(results, domain.UploadResult{Name: f.OriginalName, File: f})
			newIDs = append(newIDs, f.ID)
		}
		for _, id := range staged {
			if slices.Contains(newIDs, id) {
				continue
			}
			ok, err := r.StagedBy(ctx, sc.TenantID, sc.UserID, id)
			if err != nil {
				return err
			}
			if !ok {
				return domain.Invalid("a photo sent ahead is gone; it may be over a day old, so take it again")
			}
			newIDs = append(newIDs, id)
		}
		return place(r, newIDs)
	})
	if err != nil {
		return nil, errors.Join(err, in.dropOrphans(ctx, written))
	}
	if in.Pipeline != nil {
		in.Pipeline.Wake()
	}
	return results, nil
}

// dropOrphans deletes written bytes that no tenant references. The caller
// holds FileLock.
func (in *Inbox) dropOrphans(ctx context.Context, shas []string) error {
	return dropUnused(ctx, in.Deps, shas)
}

// Items lists unfiled items, oldest first, each with its OCR progress and
// AI reading.
func (in *Inbox) Items(ctx context.Context, sc domain.Scope) ([]domain.InboxItem, error) {
	if err := sc.RequireRead(); err != nil {
		return nil, err
	}
	items, err := in.Store.InboxItems(ctx, sc.TenantID, domain.InboxNew)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if err := in.withProgress(ctx, sc.TenantID, &items[i]); err != nil {
			return nil, err
		}
	}
	return items, nil
}

func (in *Inbox) withProgress(ctx context.Context, tenantID string, item *domain.InboxItem) error {
	prog, err := in.Store.OCRProgress(ctx, tenantID, item.ID)
	if err != nil {
		return err
	}
	item.OCR = &prog
	rd, err := in.Store.Reading(ctx, tenantID, item.ID)
	switch {
	case err == nil:
		item.Reading = &rd
	case !errors.Is(err, domain.ErrNotFound):
		return err
	}
	return nil
}

// Filed is the result of filing an item.
type Filed struct {
	Visit    domain.Visit    `json:"visit"`
	Document domain.Document `json:"document"`
}

// docInput is the document part of a filing request.
func docInput(kindRaw, titleRaw string) (domain.DocumentKind, string, error) {
	kind, err := domain.ParseDocumentKind(kindRaw)
	if err != nil {
		return "", "", err
	}
	title := domain.CleanText(titleRaw)
	if utf8.RuneCountInString(title) > 120 {
		return "", "", domain.Invalid("title is longer than 120 characters")
	}
	return kind, title, nil
}

// fileItem files one unfiled item as a document under visit and approves
// its drafts (HW-READ-4), inside a transaction.
func (in *Inbox) fileItem(ctx context.Context, r port.Repos, sc domain.Scope, itemID string, visit domain.Visit,
	kind domain.DocumentKind, title string) (string, error) {
	item, err := r.InboxItem(ctx, sc.TenantID, itemID)
	if err != nil {
		return "", err
	}
	if item.Status != domain.InboxNew {
		return "", domain.Conflict("this item is already %s", item.Status)
	}
	fileIDs := make([]string, len(item.Files))
	for i, f := range item.Files {
		fileIDs[i] = f.ID
	}
	doc := domain.Document{ID: NewID(), VisitID: visit.ID, PersonID: visit.PersonID, Kind: kind, Title: title, CreatedAt: in.Clock()}
	if err := r.CreateDocument(ctx, sc.TenantID, doc, fileIDs); err != nil {
		return "", err
	}
	if err := r.ApproveItemDrafts(ctx, sc.TenantID, item.ID, doc.ID, visit.PersonID, visit.Date, sc.UserID, in.Clock()); err != nil {
		return "", err
	}
	if err := r.SetInboxStatus(ctx, sc.TenantID, item.ID, domain.InboxFiled, doc.ID); err != nil {
		return "", err
	}
	// Pages the reading left without a title, such as pages past the
	// reading's limit, get one from a title job (HW-READ-9).
	return doc.ID, in.Pipeline.scheduleTitles(ctx, r, sc.TenantID, doc.ID)
}

// File turns an unfiled item into a document under an existing or new
// visit. Its remaining drafts are approved; rejected ones are dropped.
func (in *Inbox) File(ctx context.Context, sc domain.Scope, itemID string, req domain.FileInput) (Filed, error) {
	if err := sc.RequireEdit(); err != nil {
		return Filed{}, err
	}
	kind, title, err := docInput(req.Kind, req.Title)
	if err != nil {
		return Filed{}, err
	}
	if (req.VisitID == "") == (req.Visit == nil) {
		return Filed{}, domain.Invalid("choose an existing visit or describe a new one")
	}
	var visitID, docID string
	err = in.Store.InTx(ctx, func(r port.Repos) error {
		var visit domain.Visit
		var err error
		if req.VisitID != "" {
			visit, err = r.Visit(ctx, sc.TenantID, req.VisitID)
		} else {
			visit, err = in.Records.createVisit(ctx, r, sc, *req.Visit)
		}
		if err != nil {
			return err
		}
		visitID = visit.ID
		docID, err = in.fileItem(ctx, r, sc, itemID, visit, kind, title)
		return err
	})
	if err != nil {
		return Filed{}, err
	}
	visit, err := in.Records.Visit(ctx, sc, visitID)
	if err != nil {
		return Filed{}, err
	}
	doc, err := in.Store.Document(ctx, sc.TenantID, docID)
	return Filed{Visit: visit, Document: doc}, err
}

// Discard drops an unfiled item, its drafts and the files only it used.
func (in *Inbox) Discard(ctx context.Context, sc domain.Scope, itemID string) error {
	if err := sc.RequireEdit(); err != nil {
		return err
	}
	item, err := in.Store.InboxItem(ctx, sc.TenantID, itemID)
	if err != nil {
		return err
	}
	if item.Status != domain.InboxNew {
		return domain.Conflict("this item is already %s", item.Status)
	}
	err = in.Store.InTx(ctx, func(r port.Repos) error {
		if err := r.SetInboxStatus(ctx, sc.TenantID, itemID, domain.InboxDiscarded, ""); errors.Is(err, domain.ErrNotFound) {
			return domain.Conflict("this item was filed or discarded meanwhile")
		} else if err != nil {
			return err
		}
		return r.DeleteItemDrafts(ctx, sc.TenantID, itemID)
	})
	if err != nil {
		return err
	}
	return releaseFiles(ctx, in.Deps, sc.TenantID, item.Files)
}
