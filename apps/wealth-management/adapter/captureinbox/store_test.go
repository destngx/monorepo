package captureinbox

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/internal/sqlitefile"
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

func TestEditDraftKeepsTheSuggestionAndOnlyChangesPendingDrafts(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)
	mustAdd(t, store, queued("c1", now, ""), "h1", png)
	if err := store.CompleteCapture(ctx, domain.QuickCapture{ID: "c1", Status: domain.QuickCaptureDone}, []domain.CaptureDraft{pendingDraft("d1", "c1", now), pendingDraft("d2", "c1", now)}, nil); err != nil {
		t.Fatalf("complete: %v", err)
	}
	edit := domain.CaptureDraftEdit{Transaction: domain.TransactionCreateInput{AccountName: "Sacombank", Payee: "Pho 24", Category: "Food"}, CategorySource: domain.CategorySourceManual}

	if err := store.EditDraft(ctx, "d1", edit); err != nil {
		t.Fatalf("edit: %v", err)
	}
	drafts, err := store.GetDrafts(ctx, []string{"d1"})
	if err != nil || drafts[0].Edit == nil || drafts[0].Edit.Transaction.Payee != "Pho 24" || drafts[0].Edit.CategorySource != domain.CategorySourceManual {
		t.Fatalf("draft = %+v err=%v, want the edit saved", drafts, err)
	}
	if drafts[0].Suggestion.Transaction.Payee != "" {
		t.Fatalf("suggestion = %+v, want what the engine read kept", drafts[0].Suggestion.Transaction)
	}

	if err := store.DecideDrafts(ctx, []domain.CaptureDraftDecision{{ID: "d2", Status: domain.CaptureDraftRejected}}, now); err != nil {
		t.Fatalf("reject: %v", err)
	}
	if err := store.EditDraft(ctx, "d2", edit); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("edit decided draft: err = %v, want conflict", err)
	}
	if err := store.EditDraft(ctx, "missing", edit); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("edit missing draft: err = %v, want not found", err)
	}
}

func TestOpenAddsTheEditColumnToAnOlderStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "capture.db")
	old, err := sqlitefile.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := old.Exec(`CREATE TABLE capture_drafts (id TEXT PRIMARY KEY, status TEXT NOT NULL, created_at INTEGER NOT NULL, decided_at INTEGER NOT NULL DEFAULT 0, suggestion TEXT NOT NULL, transaction_id TEXT NOT NULL DEFAULT '', batch_id TEXT NOT NULL DEFAULT '', approved TEXT NOT NULL DEFAULT '');
		INSERT INTO capture_drafts (id, status, created_at, suggestion) VALUES ('d1', 'pending', 1, '{"id":"d1"}')`); err != nil {
		t.Fatalf("old schema: %v", err)
	}
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}

	store, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer store.Close()
	if err := store.EditDraft(context.Background(), "d1", domain.CaptureDraftEdit{Transaction: domain.TransactionCreateInput{Payee: "Grab"}}); err != nil {
		t.Fatalf("edit after migration: %v", err)
	}
	drafts, err := store.GetDrafts(context.Background(), []string{"d1"})
	if err != nil || drafts[0].Edit == nil || drafts[0].Edit.Transaction.Payee != "Grab" {
		t.Fatalf("draft = %+v err=%v, want the edit read back", drafts, err)
	}
}

func TestMemoriesAreKeptByCounterpartyKeyAndAddedOnce(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	at := time.Date(2026, 10, 12, 8, 0, 0, 0, time.UTC)
	memory := func(id, key, payee string, offset time.Duration) domain.CaptureMemory {
		return domain.CaptureMemory{ID: id, CreatedAt: at.Add(offset), Source: domain.CaptureMemoryApproved, Counterparty: key, CounterpartyKey: key, Payee: payee, Tags: []string{"breakfast"}}
	}
	if err := store.AddMemories(ctx, []domain.CaptureMemory{memory("m2", "nguyen van a", "Banh mi stall", time.Hour), memory("m1", "nguyen van a", "Banh mi", 0), memory("m3", "tran thi b", "Fruit stall", 0)}); err != nil {
		t.Fatalf("add: %v", err)
	}
	// A replayed approval stores nothing twice.
	if err := store.AddMemories(ctx, []domain.CaptureMemory{memory("m1", "nguyen van a", "Changed", 0)}); err != nil {
		t.Fatalf("add again: %v", err)
	}

	got, err := store.MemoriesByKey(ctx, []string{"nguyen van a", "nobody"})
	if err != nil || len(got) != 2 || got[0].ID != "m1" || got[0].Payee != "Banh mi" || got[1].ID != "m2" || got[1].Tags[0] != "breakfast" || !got[1].CreatedAt.Equal(at.Add(time.Hour)) {
		t.Fatalf("memories = %+v err=%v, want m1 then m2 unchanged", got, err)
	}
	if none, err := store.MemoriesByKey(ctx, nil); err != nil || len(none) != 0 {
		t.Fatalf("no keys = %+v err=%v", none, err)
	}
	// Purging old drafts keeps the memory.
	if err := store.Purge(ctx, at.Add(48*time.Hour)); err != nil {
		t.Fatalf("purge: %v", err)
	}
	if kept, _ := store.MemoriesByKey(ctx, []string{"tran thi b"}); len(kept) != 1 {
		t.Fatalf("memories after purge = %+v", kept)
	}
}

func TestGivenADoneCaptureWithPendingDraftsWhenReadAgainThenItsDraftsGoAndItIsQueuedWithItsImage(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)
	mustAdd(t, store, queued("c1", now, "lunch"), "h1", png)
	mustAdd(t, store, queued("c2", now, ""), "h2", png)
	done := domain.QuickCapture{ID: "c1", Status: domain.QuickCaptureDone, NewDrafts: 2, Message: "2 found", Balances: []domain.CaptureBalance{{Account: "Sacombank", BalanceAfter: 10}}}
	if err := store.CompleteCapture(ctx, done, []domain.CaptureDraft{pendingDraft("d1", "c1", now), pendingDraft("d2", "c1", now)}, nil); err != nil {
		t.Fatalf("complete c1: %v", err)
	}
	if err := store.CompleteCapture(ctx, domain.QuickCapture{ID: "c2", Status: domain.QuickCaptureDone}, []domain.CaptureDraft{pendingDraft("d3", "c2", now)}, nil); err != nil {
		t.Fatalf("complete c2: %v", err)
	}
	if capture, _ := store.GetCapture(ctx, "c1"); !capture.ReadAgain {
		t.Fatalf("a done capture with only pending drafts can be read again: %+v", capture)
	}

	if err := store.ReadAgain(ctx, "c1"); err != nil {
		t.Fatalf("read again: %v", err)
	}

	capture, err := store.GetCapture(ctx, "c1")
	if err != nil || capture.Status != domain.QuickCaptureQueued || capture.Attempts != 0 || capture.NewDrafts != 0 || capture.Message != "" || len(capture.Balances) != 0 || capture.ReadAgain {
		t.Fatalf("capture = %+v err=%v, want queued afresh", capture, err)
	}
	drafts, _ := store.ListDrafts(ctx, "", time.Time{})
	if len(drafts) != 1 || drafts[0].ID != "d3" {
		t.Fatalf("drafts = %+v, want only the other capture's", drafts)
	}
	claimed, image, found, err := store.ClaimCapture(ctx, now.Add(time.Minute))
	if err != nil || !found || claimed.ID != "c1" || len(image) == 0 || claimed.Note != "lunch" {
		t.Fatalf("claim = %+v found=%v err=%v, want c1 with its image and note", claimed, found, err)
	}
}

func TestGivenADecidedDraftWhenReadAgainThenItIsRefused(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)
	mustAdd(t, store, queued("c1", now, ""), "h1", png)
	mustAdd(t, store, queued("c2", now, ""), "h2", png)
	if err := store.CompleteCapture(ctx, domain.QuickCapture{ID: "c1", Status: domain.QuickCaptureDone}, []domain.CaptureDraft{pendingDraft("d1", "c1", now), pendingDraft("d2", "c1", now)}, nil); err != nil {
		t.Fatalf("complete: %v", err)
	}
	if err := store.DecideDrafts(ctx, []domain.CaptureDraftDecision{{ID: "d1", Status: domain.CaptureDraftRejected}}, now); err != nil {
		t.Fatalf("reject: %v", err)
	}

	if capture, _ := store.GetCapture(ctx, "c1"); capture.ReadAgain {
		t.Fatal("a capture with a decided draft cannot be read again")
	}
	for _, id := range []string{"c1", "c2"} {
		if err := store.ReadAgain(ctx, id); !errors.Is(err, domain.ErrConflict) {
			t.Fatalf("%s: err = %v, want conflict (decided draft, or still queued)", id, err)
		}
	}
	if err := store.ReadAgain(ctx, "missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing: err = %v, want not found", err)
	}
	if drafts, _ := store.ListDrafts(ctx, domain.CaptureDraftPending, time.Time{}); len(drafts) != 1 {
		t.Fatalf("a refused read again changes nothing: %+v", drafts)
	}
}
