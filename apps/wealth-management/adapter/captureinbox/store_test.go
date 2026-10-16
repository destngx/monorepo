package captureinbox

import (
	"apps/wealth-management/domain"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var png = append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 32)...)

func openStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "private", "capture.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func queued(id string, at time.Time, note string) domain.QuickCapture {
	return domain.QuickCapture{ID: id, CreatedAt: at, Status: domain.QuickCaptureQueued, Note: note}
}

func pendingDraft(id, captureID string, at time.Time) domain.CaptureDraft {
	return domain.CaptureDraft{
		ID: id, Status: domain.CaptureDraftPending, CreatedAt: at, CaptureIDs: []string{captureID},
		Suggestion: domain.CaptureSuggestion{ID: id, Amount: 50000, Direction: domain.CaptureOut, Transaction: domain.TransactionCreateInput{AccountName: "Sacombank"}},
	}
}

func TestOpenKeepsTheFilePrivate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private", "capture.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer store.Close()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("mode = %o, want 600", mode)
	}
}

func TestResolvePath(t *testing.T) {
	settings := filepath.Join("cfg", "wealth-management", "settings.db")
	if got := ResolvePath("", settings); got != filepath.Join("cfg", "wealth-management", "capture.db") {
		t.Fatalf("default = %q, want capture.db next to settings.db", got)
	}
	if got := ResolvePath(" /x/c.db ", settings); got != "/x/c.db" {
		t.Fatalf("override = %q, want /x/c.db", got)
	}
}

func TestAddCaptureReturnsTheFirstCaptureForTheSameHash(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)

	first, existed, err := store.AddCapture(ctx, queued("c1", now, "lunch"), "h1", png)
	if err != nil || existed || !first.HasImage {
		t.Fatalf("first add = %+v existed=%v err=%v, want new capture with image", first, existed, err)
	}
	again, existed, err := store.AddCapture(ctx, queued("c2", now.Add(time.Minute), "lunch"), "h1", png)
	if err != nil || !existed || again.ID != "c1" {
		t.Fatalf("resend = %+v existed=%v err=%v, want c1 returned", again, existed, err)
	}
	captures, err := store.ListCaptures(ctx, 10)
	if err != nil || len(captures) != 1 {
		t.Fatalf("captures = %d err=%v, want 1", len(captures), err)
	}
}

func TestClaimCaptureTakesTheOldestDueAndRequeueReturnsIt(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)
	later := now.Add(time.Hour)
	scheduled := queued("c0", now.Add(-time.Hour), "")
	scheduled.NextAttemptAt = &later
	mustAdd(t, store, scheduled, "h0", png)
	mustAdd(t, store, queued("c1", now, "note only"), "h1", nil)
	mustAdd(t, store, queued("c2", now.Add(time.Second), ""), "h2", png)

	capture, image, found, err := store.ClaimCapture(ctx, now.Add(time.Minute))
	if err != nil || !found || capture.ID != "c1" || image != nil || capture.Status != domain.QuickCaptureProcessing {
		t.Fatalf("claim = %s %v found=%v err=%v, want c1 processing without image", capture.ID, capture.Status, found, err)
	}
	capture, image, _, _ = store.ClaimCapture(ctx, now.Add(time.Minute))
	if capture.ID != "c2" || len(image) != len(png) {
		t.Fatalf("second claim = %s (%d bytes), want c2 with its image", capture.ID, len(image))
	}
	if _, _, found, _ := store.ClaimCapture(ctx, now.Add(time.Minute)); found {
		t.Fatal("third claim: c0 is not due yet, want nothing")
	}

	if err := store.RequeueProcessing(ctx); err != nil {
		t.Fatalf("requeue: %v", err)
	}
	if capture, _, found, _ := store.ClaimCapture(ctx, now.Add(time.Minute)); !found || capture.ID != "c1" {
		t.Fatalf("after requeue claim = %s found=%v, want c1 again", capture.ID, found)
	}
}

func TestCompleteCaptureKeepsMaterialsWhilePendingDraftsNeedThem(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)
	mustAdd(t, store, queued("c1", now, "lunch"), "h1", png)
	mustAdd(t, store, queued("c2", now, ""), "h2", png)

	c1 := domain.QuickCapture{ID: "c1", Status: domain.QuickCaptureDone, NewDrafts: 1, Balances: []domain.CaptureBalance{{Account: "Sacombank", BalanceAfter: 10}}}
	if err := store.CompleteCapture(ctx, c1, []domain.CaptureDraft{pendingDraft("d1", "c1", now)}, nil); err != nil {
		t.Fatalf("complete c1: %v", err)
	}
	// c2 showed the same row again: it merges into d1 and creates nothing.
	merged := pendingDraft("d1", "c1", now)
	merged.CaptureIDs = []string{"c1", "c2"}
	merged.Suggestion.Evidence = []domain.CaptureEvidence{{MaterialID: "c2-image"}}
	if err := store.CompleteCapture(ctx, domain.QuickCapture{ID: "c2", Status: domain.QuickCaptureDone}, nil, []domain.CaptureDraft{merged}); err != nil {
		t.Fatalf("complete c2: %v", err)
	}

	drafts, err := store.ListDrafts(ctx, domain.CaptureDraftPending, time.Time{})
	if err != nil || len(drafts) != 1 || len(drafts[0].CaptureIDs) != 2 || len(drafts[0].Suggestion.Evidence) != 1 {
		t.Fatalf("drafts = %+v err=%v, want d1 from c1 and c2", drafts, err)
	}
	entries, err := store.Materials(ctx, []string{"c1", "c2"})
	if err != nil || len(entries) != 3 {
		t.Fatalf("materials = %+v err=%v, want c1 image and note, c2 image", entries, err)
	}
	note, err := store.Material(ctx, "c1-note")
	if err != nil || note.Text != "lunch" {
		t.Fatalf("note = %+v err=%v", note, err)
	}
	capture, _ := store.GetCapture(ctx, "c1")
	if len(capture.Balances) != 1 || capture.NewDrafts != 1 {
		t.Fatalf("capture = %+v, want balances and new draft count saved", capture)
	}

	if err := store.DecideDrafts(ctx, []domain.CaptureDraftDecision{{ID: "d1", Status: domain.CaptureDraftRejected}}, now); err != nil {
		t.Fatalf("reject: %v", err)
	}
	if entries, _ := store.Materials(ctx, []string{"c1", "c2"}); len(entries) != 0 {
		t.Fatalf("materials after reject = %+v, want none", entries)
	}
	if _, err := store.Material(ctx, "c1-image"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("image after reject: err = %v, want not found", err)
	}
	if _, err := store.Material(ctx, "c1-note"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("note after reject: err = %v, want not found", err)
	}
}

func TestDecideDraftsOnlyChangesPendingDrafts(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)
	mustAdd(t, store, queued("c1", now, ""), "h1", png)
	if err := store.CompleteCapture(ctx, domain.QuickCapture{ID: "c1", Status: domain.QuickCaptureDone}, []domain.CaptureDraft{pendingDraft("d1", "c1", now)}, nil); err != nil {
		t.Fatalf("complete: %v", err)
	}
	input := domain.TransactionCreateInput{AccountName: "Sacombank"}
	if err := store.DecideDrafts(ctx, []domain.CaptureDraftDecision{{ID: "d1", Status: domain.CaptureDraftApproved, TransactionID: "tx1", Transaction: &input}}, now); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if err := store.DecideDrafts(ctx, []domain.CaptureDraftDecision{{ID: "d1", Status: domain.CaptureDraftRejected}}, now.Add(time.Hour)); err != nil {
		t.Fatalf("reject: %v", err)
	}

	drafts, err := store.GetDrafts(ctx, []string{"d1"})
	if err != nil || len(drafts) != 1 || drafts[0].Status != domain.CaptureDraftApproved || drafts[0].TransactionID != "tx1" || drafts[0].DecidedAt == nil || !drafts[0].DecidedAt.Equal(now) {
		t.Fatalf("draft = %+v err=%v, want approved as tx1 at the first decision", drafts, err)
	}
}

func TestDeleteCaptureRefusesCapturesWithDrafts(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)
	mustAdd(t, store, queued("c1", now, ""), "h1", png)
	mustAdd(t, store, queued("c2", now, ""), "h2", png)
	if err := store.CompleteCapture(ctx, domain.QuickCapture{ID: "c1", Status: domain.QuickCaptureDone}, []domain.CaptureDraft{pendingDraft("d1", "c1", now)}, nil); err != nil {
		t.Fatalf("complete: %v", err)
	}

	if err := store.DeleteCapture(ctx, "c1"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("delete c1: err = %v, want conflict", err)
	}
	if err := store.DeleteCapture(ctx, "c2"); err != nil {
		t.Fatalf("delete c2: %v", err)
	}
	if _, err := store.GetCapture(ctx, "c2"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("get c2: err = %v, want not found", err)
	}
	if err := store.DeleteCapture(ctx, "missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("delete missing: err = %v, want not found", err)
	}
}

func TestPurgeDropsOldDecidedDraftsAndFinishedCaptures(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	old := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	now := time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)
	mustAdd(t, store, queued("c_old", old, ""), "h1", png)
	mustAdd(t, store, queued("c_pending", old, ""), "h2", png)
	mustAdd(t, store, queued("c_empty", old, ""), "h3", png)
	mustAdd(t, store, queued("c_failed", old, ""), "h4", png)
	done := func(id string) domain.QuickCapture {
		return domain.QuickCapture{ID: id, Status: domain.QuickCaptureDone}
	}
	if err := store.CompleteCapture(ctx, done("c_old"), []domain.CaptureDraft{pendingDraft("d_old", "c_old", old)}, nil); err != nil {
		t.Fatal(err)
	}
	if err := store.CompleteCapture(ctx, done("c_pending"), []domain.CaptureDraft{pendingDraft("d_pending", "c_pending", old)}, nil); err != nil {
		t.Fatal(err)
	}
	if err := store.CompleteCapture(ctx, done("c_empty"), nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateCapture(ctx, domain.QuickCapture{ID: "c_failed", Status: domain.QuickCaptureFailed, Message: "unreadable"}); err != nil {
		t.Fatal(err)
	}
	if err := store.DecideDrafts(ctx, []domain.CaptureDraftDecision{{ID: "d_old", Status: domain.CaptureDraftRejected}}, old); err != nil {
		t.Fatal(err)
	}

	if err := store.Purge(ctx, now.Add(-domain.CaptureDraftRetention)); err != nil {
		t.Fatalf("purge: %v", err)
	}

	drafts, _ := store.ListDrafts(ctx, "", time.Time{})
	if len(drafts) != 1 || drafts[0].ID != "d_pending" {
		t.Fatalf("drafts = %+v, want only the pending one kept", drafts)
	}
	captures, _ := store.ListCaptures(ctx, 10)
	ids := map[string]bool{}
	for _, capture := range captures {
		ids[capture.ID] = true
	}
	if len(ids) != 2 || !ids["c_pending"] || !ids["c_failed"] {
		t.Fatalf("captures = %v, want c_pending and c_failed kept", ids)
	}
}

func mustAdd(t *testing.T, store *Store, capture domain.QuickCapture, hash string, image []byte) {
	t.Helper()
	if _, _, err := store.AddCapture(context.Background(), capture, hash, image); err != nil {
		t.Fatalf("add %s: %v", capture.ID, err)
	}
}
