package port

import (
	"apps/wealth-management/domain"
	"context"
	"time"
)

// CaptureInboxService is quick capture (§6.3): the phone submits a
// screenshot, the engine reads it in the background into drafts, and the
// user approves or rejects them.
type CaptureInboxService interface {
	// Submit stores a capture and returns at once. A resent submission
	// returns the first capture with duplicate set.
	Submit(ctx context.Context, input domain.QuickCaptureInput) (capture domain.QuickCapture, duplicate bool, err error)
	// Overview lists the latest captures and counts the pending drafts.
	Overview(ctx context.Context) (domain.QuickCaptureOverview, error)
	Retry(ctx context.Context, captureID string) error
	// Discard deletes a capture that produced no drafts, with its material.
	Discard(ctx context.Context, captureID string) error
	Drafts(ctx context.Context) (domain.CaptureDrafts, error)
	Material(ctx context.Context, materialID string) (domain.CaptureMaterial, error)
	// Edit saves a pending draft's row without writing it to the sheet.
	Edit(ctx context.Context, draftID string, edit domain.CaptureDraftEdit) error
	Approve(ctx context.Context, approval domain.CaptureDraftApproval) (domain.TransactionBatchResult, error)
	Reject(ctx context.Context, draftIDs []string) error
}

// CaptureDraftStore persists captures, their materials and drafts in a
// private store; nothing in it reaches the sheet, the cache or the logs.
type CaptureDraftStore interface {
	// AddCapture stores a capture and its image. When a capture with the
	// same hash exists, it is returned with existed set and nothing changes.
	AddCapture(ctx context.Context, capture domain.QuickCapture, hash string, image []byte) (stored domain.QuickCapture, existed bool, err error)
	// ClaimCapture marks the oldest queued capture due at now as
	// processing and returns it with its image (nil when it has none).
	ClaimCapture(ctx context.Context, now time.Time) (capture domain.QuickCapture, image []byte, found bool, err error)
	// RequeueProcessing returns captures left processing by a stopped
	// engine to the queue.
	RequeueProcessing(ctx context.Context) error
	// CompleteCapture saves the outcome of a capture: its new drafts and
	// the earlier drafts it showed again, in one transaction, then drops
	// materials no pending draft needs.
	CompleteCapture(ctx context.Context, capture domain.QuickCapture, created []domain.CaptureDraft, merged []domain.CaptureDraft) error
	// UpdateCapture saves a capture's status, attempts, schedule and message.
	UpdateCapture(ctx context.Context, capture domain.QuickCapture) error
	GetCapture(ctx context.Context, id string) (domain.QuickCapture, error)
	ListCaptures(ctx context.Context, limit int) ([]domain.QuickCapture, error)
	// DeleteCapture removes a capture without drafts and its material.
	DeleteCapture(ctx context.Context, id string) error
	// ListDrafts returns drafts created at or after since, any status,
	// oldest first; a zero since returns every draft.
	ListDrafts(ctx context.Context, status domain.CaptureDraftStatus, since time.Time) ([]domain.CaptureDraft, error)
	GetDrafts(ctx context.Context, ids []string) ([]domain.CaptureDraft, error)
	// EditDraft saves the user's edit of a pending draft; ErrConflict once
	// the draft is decided, ErrNotFound when it does not exist.
	EditDraft(ctx context.Context, id string, edit domain.CaptureDraftEdit) error
	// DecideDrafts applies decisions to pending drafts only, then drops
	// materials no pending draft needs.
	DecideDrafts(ctx context.Context, decisions []domain.CaptureDraftDecision, now time.Time) error
	// Material returns a stored image or note by material id.
	Material(ctx context.Context, materialID string) (domain.CaptureMaterial, error)
	// Materials lists the stored materials of the given captures.
	Materials(ctx context.Context, captureIDs []string) ([]domain.CaptureMaterialEntry, error)
	// Purge deletes decided drafts and finished captures older than before.
	Purge(ctx context.Context, before time.Time) error
}

// CaptureMemoryStore keeps what Smart Capture read for each written row
// (WM-CAP-23) in the private capture store. Memories are never purged.
type CaptureMemoryStore interface {
	// AddMemories stores memories; one whose id is already stored is skipped.
	AddMemories(ctx context.Context, memories []domain.CaptureMemory) error
	// MemoriesByKey returns the memories of the given counterparty keys,
	// oldest first.
	MemoriesByKey(ctx context.Context, keys []string) ([]domain.CaptureMemory, error)
}

// CaptureLearningService learns from rows the Smart Capture page wrote
// (WM-CAP-23); approved drafts are learned from by the inbox itself.
type CaptureLearningService interface {
	LearnWritten(ctx context.Context, rows []domain.CaptureWrittenRow) (domain.CaptureLearnResult, error)
}
