// Package port declares the interfaces services depend on.
package port

import (
	"context"
	"io"
	"time"

	"apps/health-watcher/domain"
)

// Store is the database. Reads run directly; writes that must succeed or
// fail together run in InTx.
type Store interface {
	Repos
	InTx(ctx context.Context, fn func(Repos) error) error
}

// Repos is every repository. Each method that reads or writes tenant data
// takes the tenant id and never matches rows of another tenant (HW-TEN-6).
// Lookups that find nothing return domain.ErrNotFound.
type Repos interface {
	UserRepo
	SessionRepo
	TenantRepo
	InviteRepo
	UploadTokenRepo
	PersonRepo
	ProviderRepo
	VisitRepo
	DocumentRepo
	FileRepo
	InboxRepo
	PageRepo
	StagedRepo
	JobRepo
	ReadingRepo
	ClinicalRepo
	AnalyteLinkRepo
	SettingsRepo
	SearchRepo
	MetaRepo
	MetricRepo
}

// MetricRepo stores a tenant's custom metrics and the metric suggestions
// for printed names no metric knows (HW-MET-8).
type MetricRepo interface {
	CustomMetrics(ctx context.Context, tenantID string) ([]domain.Metric, error)
	CreateCustomMetric(ctx context.Context, tenantID string, m domain.Metric, by string, at time.Time) error
	// UnlinkedObservations lists a tenant's approved observations linked to
	// no metric, oldest first.
	UnlinkedObservations(ctx context.Context, tenantID string) ([]domain.Observation, error)
	// SuggestionKeys lists the folded names that have a suggestion, in any status.
	SuggestionKeys(ctx context.Context, tenantID string) (map[string]bool, error)
	CreateSuggestion(ctx context.Context, tenantID string, s domain.MetricSuggestion) error
	// Suggestions lists a tenant's suggestions in a status, oldest first.
	Suggestions(ctx context.Context, tenantID string, status domain.SuggestionStatus) ([]domain.MetricSuggestion, error)
	Suggestion(ctx context.Context, tenantID, id string) (domain.MetricSuggestion, error)
	DecideSuggestion(ctx context.Context, tenantID, id string, status domain.SuggestionStatus, by string, at time.Time) error
	// TenantsWithUnlinked lists the tenants with an approved value linked to no metric.
	TenantsWithUnlinked(ctx context.Context) ([]string, error)
	// AllAnalyteLinks lists every remembered analyte link of every tenant.
	AllAnalyteLinks(ctx context.Context) ([]AnalyteLink, error)
	DeleteAnalyteLink(ctx context.Context, tenantID, providerID, analyteKey string) error
	// LinkedAnalytes lists the distinct printed names of the values linked
	// to a metric.
	LinkedAnalytes(ctx context.Context, tenantID, metricCode string) ([]string, error)
}

// AnalyteLink is a remembered link from a printed name, for a provider or
// for every provider, to a metric (HW-READ-5, HW-MET-8).
type AnalyteLink struct {
	TenantID   string
	ProviderID string
	Key        string
	MetricCode string
}

// LOINC looks up LOINC codes and their descriptions by code; only the code
// leaves the Mac (HW-MET-8).
type LOINC interface {
	// Name returns the code's long common name, false when the code is unknown.
	Name(ctx context.Context, code string) (string, bool, error)
	// About returns a word-for-word description of the test, false when the
	// source has none (HW-MET-6).
	About(ctx context.Context, code string) (domain.MetricAbout, bool, error)
}

// MetaRepo keeps engine bookkeeping values, such as the catalog version the
// stored links were made with.
type MetaRepo interface {
	// Meta returns a value, "" when it was never set.
	Meta(ctx context.Context, key string) (string, error)
	SetMeta(ctx context.Context, key, value string) error
}

// SettingsRepo stores the engine settings as key and value (HW-OPS-9).
type SettingsRepo interface {
	// Settings returns every stored value and when they last changed.
	Settings(ctx context.Context) (map[string]string, domain.SettingsStamp, error)
	// SaveSettings writes each value, stamped with the time and the user.
	SaveSettings(ctx context.Context, values map[string]string, at time.Time, by string) error
}

type UserRepo interface {
	CountUsers(ctx context.Context) (int, error)
	CreateUser(ctx context.Context, u domain.User, passwordHash string) error
	UserByEmail(ctx context.Context, email string) (domain.User, string, error)
	UserByID(ctx context.Context, id string) (domain.User, string, error)
	SetPasswordHash(ctx context.Context, userID, passwordHash string) error
	// ListUsers lists every account of the engine, oldest first.
	ListUsers(ctx context.Context) ([]domain.User, error)
	CountAdmins(ctx context.Context) (int, error)
	SetUserAdmin(ctx context.Context, userID string, admin bool) error
	RecordLoginFailure(ctx context.Context, email string, at time.Time) error
	CountLoginFailures(ctx context.Context, email string, since time.Time) (int, error)
	ClearLoginFailures(ctx context.Context, email string) error
}

type SessionRepo interface {
	CreateSession(ctx context.Context, idHash string, s domain.Session, at time.Time) error
	SessionByHash(ctx context.Context, idHash string) (domain.Session, error)
	ExtendSession(ctx context.Context, idHash string, expiresAt time.Time) error
	DeleteSession(ctx context.Context, idHash string) error
	DeleteUserSessions(ctx context.Context, userID string) error
}

type TenantRepo interface {
	CreateTenant(ctx context.Context, t domain.Tenant) error
	Tenant(ctx context.Context, id string) (domain.Tenant, error)
	RenameTenant(ctx context.Context, tenantID, name string) error
	AddMember(ctx context.Context, tenantID, userID string, role domain.Role, at time.Time) error
	MembershipsOf(ctx context.Context, userID string) ([]domain.Membership, error)
	RoleOf(ctx context.Context, tenantID, userID string) (domain.Role, error)
	Members(ctx context.Context, tenantID string) ([]domain.Member, error)
	SetRole(ctx context.Context, tenantID, userID string, role domain.Role) error
	RemoveMember(ctx context.Context, tenantID, userID string) error
	CountOwners(ctx context.Context, tenantID string) (int, error)
}

type InviteRepo interface {
	CreateInvite(ctx context.Context, inv domain.Invite, tokenHash string) error
	InviteByHash(ctx context.Context, tokenHash string) (domain.Invite, error)
	// UseInvite marks a pending invite used; it returns domain.ErrNotFound
	// when the invite is gone or already used.
	UseInvite(ctx context.Context, id, userID string, at time.Time) error
	PendingInvites(ctx context.Context, tenantID string, now time.Time) ([]domain.Invite, error)
	DeleteInvite(ctx context.Context, tenantID, id string) error
}

type UploadTokenRepo interface {
	CreateUploadToken(ctx context.Context, tok domain.UploadToken, tokenHash string) error
	UploadTokenByHash(ctx context.Context, tokenHash string) (domain.UploadToken, error)
	TouchUploadToken(ctx context.Context, id string, at time.Time) error
	// UploadTokens lists the user's lasting tokens; capture links are left out.
	UploadTokens(ctx context.Context, tenantID, userID string) ([]domain.UploadToken, error)
	DeleteExpiredUploadTokens(ctx context.Context, now time.Time) error
	DeleteUploadToken(ctx context.Context, tenantID, userID, id string) error
	DeleteUserUploadTokens(ctx context.Context, tenantID, userID string) error
}

// QRCoder draws a QR code as a PNG.
type QRCoder interface {
	PNG(text string) ([]byte, error)
}

type PersonRepo interface {
	CreatePerson(ctx context.Context, tenantID string, p domain.Person) error
	UpdatePerson(ctx context.Context, tenantID, id string, in domain.PersonInput) error
	Person(ctx context.Context, tenantID, id string) (domain.Person, error)
	People(ctx context.Context, tenantID string) ([]domain.Person, error)
	DeletePerson(ctx context.Context, tenantID, id string) error
	// UnlinkUser clears the user link of the tenant's people linked to userID.
	UnlinkUser(ctx context.Context, tenantID, userID string) error
}

type ProviderRepo interface {
	ProviderByKey(ctx context.Context, tenantID, key string) (domain.Provider, error)
	CreateProvider(ctx context.Context, tenantID string, p domain.Provider, key string) error
	Providers(ctx context.Context, tenantID string) ([]domain.Provider, error)
}

// VisitFilter narrows a person's timeline (HW-FIND-1). Empty fields match all.
type VisitFilter struct {
	Type       string
	ProviderID string
	From, To   string
}

type VisitRepo interface {
	CreateVisit(ctx context.Context, tenantID string, v domain.Visit) error
	UpdateVisit(ctx context.Context, tenantID string, v domain.Visit) error
	Visit(ctx context.Context, tenantID, id string) (domain.Visit, error)
	VisitsOf(ctx context.Context, tenantID, personID string, f VisitFilter) ([]domain.Visit, error)
	DeleteVisit(ctx context.Context, tenantID, id string) error
}

type DocumentRepo interface {
	CreateDocument(ctx context.Context, tenantID string, d domain.Document, fileIDs []string) error
	Document(ctx context.Context, tenantID, id string) (domain.Document, error)
	DocumentsOf(ctx context.Context, tenantID, visitID string) ([]domain.Document, error)
	UpdateDocument(ctx context.Context, tenantID string, d domain.Document) error
	DeleteDocument(ctx context.Context, tenantID, id string) error
}

type FileRepo interface {
	CreateFile(ctx context.Context, tenantID string, f domain.StoredFile) error
	FileBySHA(ctx context.Context, tenantID, sha string) (domain.StoredFile, error)
	File(ctx context.Context, tenantID, id string) (domain.StoredFile, error)
	// FileInUse reports whether a document or an unfiled inbox item uses the file.
	FileInUse(ctx context.Context, tenantID, id string) (bool, error)
	DeleteFile(ctx context.Context, tenantID, id string) error
	// SHAInUse reports whether any tenant still has a file, a page image or
	// a masked page image with this hash.
	SHAInUse(ctx context.Context, sha string) (bool, error)
}

type InboxRepo interface {
	// CreateInboxItem stores an item with its files and the item's PersonID.
	CreateInboxItem(ctx context.Context, tenantID string, item domain.InboxItem, fileIDs []string) error
	// AddInboxFiles adds files to an item, after the ones it has.
	AddInboxFiles(ctx context.Context, tenantID, itemID string, fileIDs []string) error
	InboxItem(ctx context.Context, tenantID, id string) (domain.InboxItem, error)
	InboxItems(ctx context.Context, tenantID string, status domain.InboxStatus) ([]domain.InboxItem, error)
	// SetInboxStatus moves an unfiled item to status; an item that is no
	// longer new is domain.ErrNotFound, so two moves can't both win.
	SetInboxStatus(ctx context.Context, tenantID, id string, status domain.InboxStatus, documentID string) error
	// UnfileDocument moves the item filed as a document back to new and
	// returns its ID; a document no item was filed as is domain.ErrNotFound.
	UnfileDocument(ctx context.Context, tenantID, documentID string) (string, error)
	// NewItemsWithFile lists the unfiled items holding a file.
	NewItemsWithFile(ctx context.Context, tenantID, fileID string) ([]string, error)
	// OCRProgress counts the item's files with pages and with failed OCR jobs.
	OCRProgress(ctx context.Context, tenantID, itemID string) (domain.OCRProgress, error)
}

// TenantFile names a file of a tenant.
type TenantFile struct {
	TenantID string
	FileID   string
}

// TenantObservation is an observation with its tenant and the provider of
// its visit, "" when the visit has none.
type TenantObservation struct {
	TenantID   string
	ProviderID string
	domain.Observation
}

// TenantDocument names a document of a tenant.
type TenantDocument struct {
	TenantID   string
	DocumentID string
}

// PageRepo stores the pages read from files (HW-READ-1, HW-READ-2, HW-AI-7).
type PageRepo interface {
	// SavePages replaces a file's pages, keeping the id of each page number
	// that existed before, and reindexes their text for search. It returns
	// the image hashes the file's pages no longer use.
	SavePages(ctx context.Context, tenantID, fileID string, pages []domain.Page) ([]string, error)
	PagesOf(ctx context.Context, tenantID, fileID string) ([]domain.Page, error)
	Page(ctx context.Context, tenantID, id string) (domain.Page, error)
	// SetPageMask stores a page's mask and returns the masked image hash it replaced.
	SetPageMask(ctx context.Context, tenantID, pageID string, m domain.PageMask) (string, error)
	// SetPageTitle names a page (HW-READ-9).
	SetPageTitle(ctx context.Context, tenantID, pageID, title string) error
	// UntitledDocuments lists the documents with a page that has no title
	// and no title job yet, oldest first.
	UntitledDocuments(ctx context.Context) ([]TenantDocument, error)
	// PageImageSHAs lists the page and masked image hashes of a file's pages.
	PageImageSHAs(ctx context.Context, tenantID, fileID string) ([]string, error)
	// FilesWithoutPages lists files with no pages and no OCR job, oldest first.
	FilesWithoutPages(ctx context.Context) ([]TenantFile, error)
}

// StagedRepo holds the files a phone sent ahead of Send (HW-IMP-11).
type StagedRepo interface {
	// StageFile marks a file as sent ahead by a user; staging it again is a no-op.
	StageFile(ctx context.Context, tenantID, userID, fileID string, at time.Time) error
	// StagedBy reports whether the user staged the file and has not sent it yet.
	StagedBy(ctx context.Context, tenantID, userID, fileID string) (bool, error)
	// UnstageFiles drops the staging rows of files that went into an item.
	UnstageFiles(ctx context.Context, tenantID string, fileIDs []string) error
	// StagedBefore lists the files staged before a time and never sent.
	StagedBefore(ctx context.Context, before time.Time) ([]TenantFile, error)
}

// JobRepo is the background work queue (§11 Workers, HW-READ-6).
type JobRepo interface {
	// EnqueueJob adds a job, or queues a done or failed one again from
	// scratch. A queued or running job is left as it is.
	EnqueueJob(ctx context.Context, job domain.Job, at time.Time) error
	// ClaimJob marks the next due queued job running and returns it, or
	// domain.ErrNotFound when none is due.
	ClaimJob(ctx context.Context, now time.Time) (domain.Job, error)
	CompleteJob(ctx context.Context, id string, at time.Time) error
	// RetryJob queues a job again after a failed attempt.
	RetryJob(ctx context.Context, id string, attempts int, next time.Time, lastErr string, at time.Time) error
	FailJob(ctx context.Context, id string, attempts int, lastErr string, at time.Time) error
	JobFor(ctx context.Context, tenantID string, kind domain.JobKind, subjectID string) (domain.Job, error)
	// RequeueRunningJobs queues jobs left running by a stopped engine.
	RequeueRunningJobs(ctx context.Context, at time.Time) error
}

// ReadingRepo stores the AI reading of each inbox item (HW-READ-3).
type ReadingRepo interface {
	SaveReading(ctx context.Context, tenantID string, r domain.Reading) error
	// Reading returns an item's reading with attempts and next attempt
	// from its job; an item never read is domain.ErrNotFound.
	Reading(ctx context.Context, tenantID, itemID string) (domain.Reading, error)
	// OffReadings lists unfiled items whose reading is off, oldest first.
	OffReadings(ctx context.Context) ([]TenantItem, error)
}

// TenantItem is an inbox item and its tenant.
type TenantItem struct {
	TenantID string
	ItemID   string
}

// ClinicalRepo stores observations and findings with their audit log
// (HW-REC-3, HW-REC-4, HW-REC-6, HW-READ-4).
type ClinicalRepo interface {
	CreateObservation(ctx context.Context, tenantID string, o domain.Observation) error
	UpdateObservation(ctx context.Context, tenantID string, o domain.Observation) error
	Observation(ctx context.Context, tenantID, id string) (domain.Observation, error)
	ObservationsOfItem(ctx context.Context, tenantID, itemID string) ([]domain.Observation, error)
	ObservationsOfDocument(ctx context.Context, tenantID, documentID string) ([]domain.Observation, error)
	DeleteObservation(ctx context.Context, tenantID, id string) error

	CreateFinding(ctx context.Context, tenantID string, f domain.Finding) error
	UpdateFinding(ctx context.Context, tenantID string, f domain.Finding) error
	Finding(ctx context.Context, tenantID, id string) (domain.Finding, error)
	FindingsOfItem(ctx context.Context, tenantID, itemID string) ([]domain.Finding, error)
	FindingsOfDocument(ctx context.Context, tenantID, documentID string) ([]domain.Finding, error)
	DeleteFinding(ctx context.Context, tenantID, id string) error

	// DeleteItemDrafts removes an item's unapproved observations and findings.
	DeleteItemDrafts(ctx context.Context, tenantID, itemID string) error
	// ApproveItemDrafts approves an item's drafts under a document and
	// person; a draft with no sample date gets sampleDate, and userID is
	// recorded as the approver. Rejected rows are deleted.
	ApproveItemDrafts(ctx context.Context, tenantID, itemID, documentID, personID, sampleDate, userID string, at time.Time) error
	// ConfirmApprovals records userID as the approver of a document's
	// approved records that have none (records from before D9) and
	// returns how many it changed.
	ConfirmApprovals(ctx context.Context, tenantID, documentID, userID string, at time.Time) (int, error)
	// ObservationsOfPerson lists a person's approved observations, oldest
	// sample first, with the visit and place each came from (HW-MET-3).
	ObservationsOfPerson(ctx context.Context, tenantID, personID string) ([]domain.PersonObservation, error)
	// CountTypedObservations counts a person's approved observations typed
	// in without a document (HW-MET-2).
	CountTypedObservations(ctx context.Context, tenantID, personID string) (int, error)
	// UneditedReadObservations lists every tenant's approved observations
	// that no person changed: the reading's own confidence, since any edit
	// saves 100, and no audit entry made by a user. The engine's own
	// relinks don't count. Each comes with its visit's provider.
	UneditedReadObservations(ctx context.Context) ([]TenantObservation, error)
	AddAudit(ctx context.Context, tenantID string, e domain.AuditEntry) error
	AuditOf(ctx context.Context, tenantID string, entity domain.AuditEntity, entityID string) ([]domain.AuditEntry, error)
}

// AnalyteLinkRepo remembers analyte names linked to the catalog by hand,
// per provider (HW-READ-5).
type AnalyteLinkRepo interface {
	SetAnalyteLink(ctx context.Context, tenantID, providerID, analyteKey, metricCode string) error
	AnalyteLinks(ctx context.Context, tenantID, providerID string) (map[string]string, error)
}

// SearchRow is one index row that matched, with where it lives.
type SearchRow struct {
	Hit  domain.SearchHit
	Body string
}

// SearchRepo runs full-text queries (HW-FIND-2). Repositories keep the
// index current as pages, visits and approved findings change.
type SearchRepo interface {
	Search(ctx context.Context, tenantID, expr string, limit int) ([]SearchRow, error)
	// RebuildSearchIndex rebuilds the index when its version differs.
	RebuildSearchIndex(ctx context.Context, version string) (bool, error)
}

// OCRPage is one page as the OCR tool read it. Quads are 0-1 from the
// top-left corner of the rendered image.
type OCRPage struct {
	Number     int
	Method     domain.PageMethod
	Text       string
	Confidence float64
	Lines      []domain.Line
	Barcodes   []domain.Quad
	// BarcodesChecked is false when the tool could not look for barcodes.
	BarcodesChecked bool
	// ImagePath is the rendered JPEG, "" when the tool can't render pages.
	ImagePath     string
	Width, Height int
}

// OCR reads a file's text on the Mac and renders its pages (HW-READ-1, D4).
type OCR interface {
	// Read runs the tool on one file, writing page images under workDir.
	Read(ctx context.Context, path, workDir string) ([]OCRPage, error)
}

// ImageMasker blacks out boxes on a JPEG (HW-AI-7).
type ImageMasker interface {
	Mask(jpeg []byte, boxes []domain.Quad) ([]byte, error)
}

// ReadPage is one masked page sent to the AI provider.
type ReadPage struct {
	Number int
	JPEG   []byte
	Text   string
}

// ReadResult is the provider's reading.
type ReadResult struct {
	Model  string
	Output domain.ReadingOutput
}

// TitleResult is the provider's page titles.
type TitleResult struct {
	Model  string
	Output domain.PageTitlesOutput
}

// Reader sends masked pages to the AI provider (HW-READ-3, HW-AI-1).
type Reader interface {
	Read(ctx context.Context, pages []ReadPage) (ReadResult, error)
	// TitlePages names pages from their masked text alone; JPEG is not sent
	// (HW-READ-9).
	TitlePages(ctx context.Context, pages []ReadPage) (TitleResult, error)
	// SuggestMetrics asks what printed test names are, given the catalog;
	// only names and units are sent (HW-MET-8).
	SuggestMetrics(ctx context.Context, names []domain.UnknownName, catalog []domain.Metric) (SuggestResult, error)
}

// SuggestResult is the provider's metric suggestions.
type SuggestResult struct {
	Model  string
	Output domain.SuggestOutput
}

// FileStore keeps file bytes by SHA-256 (HW-OPS-2). Put is idempotent.
type FileStore interface {
	Put(ctx context.Context, sha string, data []byte) error
	// Path returns where the bytes are on disk, for tools that read files.
	Path(sha string) (string, error)
	Open(ctx context.Context, sha string) (io.ReadCloser, error)
	Delete(ctx context.Context, sha string) error
}

// PasswordHasher hashes and checks passwords (HW-AUTH-1).
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, encoded string) (bool, error)
}

// Clock returns the current time.
type Clock func() time.Time
