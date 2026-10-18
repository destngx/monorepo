package service

import (
	"apps/wealth-management/adapter/captureinbox"
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var inboxPNG = append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 32)...)

type fakeCaptureReader struct {
	results  []domain.CaptureResult
	errs     []error
	failures []string
	requests []port.CaptureRequest
}

func (f *fakeCaptureReader) Capture(_ context.Context, request port.CaptureRequest, progress func(port.CaptureProgress)) (domain.CaptureResult, error) {
	call := len(f.requests)
	f.requests = append(f.requests, request)
	if call < len(f.failures) && f.failures[call] != "" {
		progress(port.CaptureProgress{Stage: "extract", MaterialID: request.Materials[0].ID, Status: "error", Message: f.failures[call]})
	}
	if call < len(f.errs) && f.errs[call] != nil {
		return domain.CaptureResult{}, f.errs[call]
	}
	if call < len(f.results) {
		return f.results[call], nil
	}
	return domain.CaptureResult{Suggestions: []domain.CaptureSuggestion{}}, nil
}

type fakeCaptureLedger struct {
	accounts     []domain.Account
	transactions []domain.Transaction
	batches      []domain.TransactionBatchInput
	batchErr     error
	// written replays a batch id like the sheet ledger does (WM-TX-10).
	written map[string]domain.TransactionBatchResult
}

func (f *fakeCaptureLedger) ListAccounts(bool) ([]domain.Account, error) { return f.accounts, nil }
func (f *fakeCaptureLedger) ListTransactions(bool) ([]domain.Transaction, error) {
	return f.transactions, nil
}
func (f *fakeCaptureLedger) CreateTransactions(input domain.TransactionBatchInput) (domain.TransactionBatchResult, error) {
	if f.batchErr != nil {
		return domain.TransactionBatchResult{}, f.batchErr
	}
	if result, ok := f.written[input.BatchID]; ok {
		result.Replayed = true
		return result, nil
	}
	f.batches = append(f.batches, input)
	result := domain.TransactionBatchResult{BatchID: input.BatchID}
	for index := range input.Transactions {
		result.Rows = append(result.Rows, domain.TransactionBatchRow{Index: index, ID: fmt.Sprintf("tx%d", index+1)})
		result.Created++
	}
	if f.written == nil {
		f.written = map[string]domain.TransactionBatchResult{}
	}
	f.written[input.BatchID] = result
	return result, nil
}

type inboxFixture struct {
	inbox  *CaptureInbox
	store  *captureinbox.Store
	reader *fakeCaptureReader
	ledger *fakeCaptureLedger
	now    time.Time
}

func newInboxFixture(t *testing.T) *inboxFixture {
	t.Helper()
	store, err := captureinbox.Open(filepath.Join(t.TempDir(), "capture.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	fixture := &inboxFixture{store: store, reader: &fakeCaptureReader{}, ledger: &fakeCaptureLedger{}, now: time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)}
	ids := 0
	fixture.inbox = NewCaptureInbox(store, fixture.reader, fixture.ledger, logger.NewTestLogger(t), CaptureInboxOptions{
		Now:   func() time.Time { return fixture.now },
		NewID: func() string { ids++; return fmt.Sprintf("ID%d", ids) },
	})
	return fixture
}

func (f *inboxFixture) submit(t *testing.T, image []byte, note string) domain.QuickCapture {
	t.Helper()
	capture, _, err := f.inbox.Submit(context.Background(), domain.QuickCaptureInput{Image: image, Note: note})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	return capture
}

func (f *inboxFixture) process(t *testing.T) {
	t.Helper()
	processed, err := f.inbox.ProcessNext(context.Background())
	if err != nil || !processed {
		t.Fatalf("process: processed=%v err=%v", processed, err)
	}
}

func row(account string, amount float64, occurredAt string, materialID string) domain.CaptureSuggestion {
	date, _ := time.Parse("2006-01-02", occurredAt[:10])
	return domain.CaptureSuggestion{
		ID: "s", Selected: true, Direction: domain.CaptureOut, Amount: amount, OccurredAt: occurredAt,
		Transaction: domain.TransactionCreateInput{AccountName: account, Date: date.Format(domain.SheetDateLayout), Payment: &amount},
		Evidence:    []domain.CaptureEvidence{{MaterialID: materialID}},
	}
}

func TestQuickCaptureResendReturnsTheFirstCapture(t *testing.T) {
	f := newInboxFixture(t)

	first := f.submit(t, inboxPNG, "lunch")
	again, duplicate, err := f.inbox.Submit(context.Background(), domain.QuickCaptureInput{Image: inboxPNG, Note: " lunch "})

	if err != nil || !duplicate || again.ID != first.ID || first.Status != domain.QuickCaptureQueued {
		t.Fatalf("resend = %+v duplicate=%v err=%v, want the first queued capture", again, duplicate, err)
	}
}

func TestQuickCaptureRejectsInvalidMaterialBeforeStoring(t *testing.T) {
	f := newInboxFixture(t)

	_, _, err := f.inbox.Submit(context.Background(), domain.QuickCaptureInput{Image: []byte("plain text")})

	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("err = %v, want validation", err)
	}
	if overview, _ := f.inbox.Overview(context.Background()); len(overview.Captures) != 0 {
		t.Fatalf("captures = %d, want none stored", len(overview.Captures))
	}
}

func TestQuickCaptureOverlappingScreenshotsMakeOneDraftPerRow(t *testing.T) {
	// Given two screenshots of one history screen that share a row
	f := newInboxFixture(t)
	f.ledger.accounts = []domain.Account{{Name: "Sacombank", Balance: 900}}
	first := f.submit(t, inboxPNG, "")
	second := f.submit(t, append([]byte{}, append(inboxPNG, 1)...), "team lunch")
	f.reader.results = []domain.CaptureResult{
		{Suggestions: []domain.CaptureSuggestion{row("Sacombank", 50000, "2026-10-16 08:15", first.ID+"-image")}, Balances: []domain.CaptureBalance{{Account: "Sacombank", BalanceAfter: 1000, AsOf: "2026-10-16 08:15"}}},
		{Suggestions: []domain.CaptureSuggestion{
			row("Sacombank", 50000, "2026-10-16 08:15", second.ID+"-image"),
			row("Sacombank", 75000, "2026-10-16 12:00", second.ID+"-image"),
		}},
	}

	// When both are processed
	f.process(t)
	f.process(t)

	// Then the shared row is one draft with evidence from both, the note
	// reaches every row of its capture, and the note went to the reader.
	drafts, err := f.inbox.Drafts(context.Background())
	if err != nil {
		t.Fatalf("drafts: %v", err)
	}
	if len(drafts.Drafts) != 2 {
		t.Fatalf("drafts = %d, want 2", len(drafts.Drafts))
	}
	shared := drafts.Drafts[0]
	if len(shared.CaptureIDs) != 2 || len(shared.Suggestion.Evidence) != 2 || *shared.Suggestion.Transaction.Memo != "team lunch" {
		t.Fatalf("shared draft = %+v, want both captures and the note", shared)
	}
	if got := f.reader.requests[1].Materials; len(got) != 2 || got[1].Text != "team lunch" {
		t.Fatalf("second request materials = %+v, want the image and the note", got)
	}
	if len(drafts.Materials) != 3 {
		t.Fatalf("materials = %+v, want two images and one note", drafts.Materials)
	}
	if len(drafts.Balances) != 1 || drafts.Balances[0].SheetBalance != 900 {
		t.Fatalf("balances = %+v, want the screenshot balance against the sheet", drafts.Balances)
	}
	overview, _ := f.inbox.Overview(context.Background())
	captures := overview.Captures
	if captures[0].NewDrafts != 1 || captures[1].NewDrafts != 1 || captures[0].Status != domain.QuickCaptureDone || overview.PendingDrafts != 2 {
		t.Fatalf("overview = %+v, want one new draft each and two pending", overview)
	}
}

func TestQuickCaptureDoesNotOfferRejectedRowsAgain(t *testing.T) {
	f := newInboxFixture(t)
	first := f.submit(t, inboxPNG, "")
	second := f.submit(t, append([]byte{}, append(inboxPNG, 1)...), "")
	f.reader.results = []domain.CaptureResult{
		{Suggestions: []domain.CaptureSuggestion{row("Sacombank", 50000, "2026-10-16 08:15", first.ID+"-image")}},
		{Suggestions: []domain.CaptureSuggestion{row("Sacombank", 50000, "2026-10-16 08:15", second.ID+"-image")}},
	}
	f.process(t)
	drafts, _ := f.inbox.Drafts(context.Background())
	if err := f.inbox.Reject(context.Background(), []string{drafts.Drafts[0].ID}); err != nil {
		t.Fatalf("reject: %v", err)
	}

	f.process(t)

	drafts, _ = f.inbox.Drafts(context.Background())
	overview, _ := f.inbox.Overview(context.Background())
	captures := overview.Captures
	if len(drafts.Drafts) != 0 || !strings.Contains(captures[0].Message, "Already decided") {
		t.Fatalf("drafts = %d, message %q, want none and an explanation", len(drafts.Drafts), captures[0].Message)
	}
	if _, err := f.inbox.Material(context.Background(), first.ID+"-image"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("material after reject: err = %v, want deleted", err)
	}
}

func TestQuickCaptureNothingFoundDeletesTheMaterial(t *testing.T) {
	f := newInboxFixture(t)
	capture := f.submit(t, inboxPNG, "")
	f.reader.results = []domain.CaptureResult{{Suggestions: []domain.CaptureSuggestion{}, Unreadable: []domain.CaptureUnreadable{{MaterialID: capture.ID + "-image", Reason: "Not a banking screen."}}}}

	f.process(t)

	stored, _ := f.store.GetCapture(context.Background(), capture.ID)
	if stored.Status != domain.QuickCaptureDone || stored.Message != "Nothing found: Not a banking screen." {
		t.Fatalf("capture = %+v, want done with the reason", stored)
	}
	if _, err := f.inbox.Material(context.Background(), capture.ID+"-image"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("material: err = %v, want deleted at once", err)
	}
}

func TestQuickCaptureRetriesProviderFailuresWithBackoff(t *testing.T) {
	f := newInboxFixture(t)
	capture := f.submit(t, inboxPNG, "")
	f.reader.failures = []string{"The provider is busy."}
	f.reader.errs = []error{nil, errors.New("sheet unavailable")}

	f.process(t)
	stored, _ := f.store.GetCapture(context.Background(), capture.ID)
	if stored.Status != domain.QuickCaptureQueued || stored.Attempts != 1 || stored.Message != "The provider is busy." || !stored.NextAttemptAt.Equal(f.now.Add(time.Minute)) {
		t.Fatalf("after a failed extraction = %+v, want queued for a retry in 1m", stored)
	}
	if processed, _ := f.inbox.ProcessNext(context.Background()); processed {
		t.Fatal("before the retry is due: want nothing processed")
	}

	f.now = f.now.Add(time.Minute)
	f.process(t)
	stored, _ = f.store.GetCapture(context.Background(), capture.ID)
	if stored.Attempts != 2 || !stored.NextAttemptAt.Equal(f.now.Add(5*time.Minute)) {
		t.Fatalf("after a second failure = %+v, want a retry in 5m", stored)
	}
}

func TestQuickCaptureFailsFinallyAndRetriesOnRequest(t *testing.T) {
	f := newInboxFixture(t)
	capture := f.submit(t, inboxPNG, "")
	f.reader.errs = []error{fmt.Errorf("%w: AI is not configured", domain.ErrValidation)}

	f.process(t)
	stored, _ := f.store.GetCapture(context.Background(), capture.ID)
	if stored.Status != domain.QuickCaptureFailed {
		t.Fatalf("capture = %+v, want failed without retries", stored)
	}

	if err := f.inbox.Retry(context.Background(), capture.ID); err != nil {
		t.Fatalf("retry: %v", err)
	}
	f.process(t)
	stored, _ = f.store.GetCapture(context.Background(), capture.ID)
	if stored.Status != domain.QuickCaptureDone {
		t.Fatalf("after retry = %+v, want done", stored)
	}
	if err := f.inbox.Retry(context.Background(), capture.ID); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("retry a done capture: err = %v, want conflict", err)
	}
}

func TestQuickCaptureApproveWritesTheEditedRowsOnce(t *testing.T) {
	f := newInboxFixture(t)
	capture := f.submit(t, inboxPNG, "")
	f.reader.results = []domain.CaptureResult{{Suggestions: []domain.CaptureSuggestion{
		row("Sacombank", 50000, "2026-10-16 08:15", capture.ID+"-image"),
		row("Sacombank", 75000, "2026-10-16 12:00", capture.ID+"-image"),
	}}}
	f.process(t)
	drafts, _ := f.inbox.Drafts(context.Background())
	edited := drafts.Drafts[0].Suggestion.Transaction
	edited.Payee = "Pho 24"

	result, err := f.inbox.Approve(context.Background(), domain.CaptureDraftApproval{BatchID: "b1", Drafts: []domain.CaptureDraftApprovalRow{{ID: drafts.Drafts[0].ID, Transaction: edited}}})

	if err != nil || result.Created != 1 || f.ledger.batches[0].Transactions[0].Payee != "Pho 24" {
		t.Fatalf("approve = %+v err=%v, want the edited row written", result, err)
	}
	approved, _ := f.store.GetDrafts(context.Background(), []string{drafts.Drafts[0].ID})
	if approved[0].Status != domain.CaptureDraftApproved || approved[0].TransactionID != "tx1" {
		t.Fatalf("draft = %+v, want approved as tx1", approved[0])
	}
	if _, err := f.inbox.Approve(context.Background(), domain.CaptureDraftApproval{BatchID: "b2", Drafts: []domain.CaptureDraftApprovalRow{{ID: drafts.Drafts[0].ID, Transaction: edited}}}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("approve twice: err = %v, want conflict", err)
	}
	// A retry whose first answer was lost replays the same batch.
	replay, err := f.inbox.Approve(context.Background(), domain.CaptureDraftApproval{BatchID: "b1", Drafts: []domain.CaptureDraftApprovalRow{{ID: drafts.Drafts[0].ID, Transaction: edited}}})
	if err != nil || !replay.Replayed || replay.Rows[0].ID != "tx1" {
		t.Fatalf("replay = %+v err=%v, want the first result replayed", replay, err)
	}
	if len(f.ledger.batches) != 1 {
		t.Fatalf("batches = %d, want one write", len(f.ledger.batches))
	}
	// The other draft still needs the screenshot.
	if _, err := f.inbox.Material(context.Background(), capture.ID+"-image"); err != nil {
		t.Fatalf("material with a pending draft: %v", err)
	}
}

func TestQuickCaptureDraftsRerunLedgerDedup(t *testing.T) {
	f := newInboxFixture(t)
	capture := f.submit(t, inboxPNG, "")
	f.reader.results = []domain.CaptureResult{{Suggestions: []domain.CaptureSuggestion{row("Sacombank", 50000, "2026-10-16 08:15", capture.ID+"-image")}}}
	f.process(t)

	// Entered by hand after the capture.
	f.ledger.transactions = []domain.Transaction{{ID: "tx9", AccountName: "Sacombank", Date: time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC), Payment: floatPointer(50000)}}
	drafts, err := f.inbox.Drafts(context.Background())

	if err != nil || drafts.Drafts[0].Suggestion.Duplicate == nil || drafts.Drafts[0].Suggestion.Selected {
		t.Fatalf("draft = %+v err=%v, want marked likely recorded and unselected", drafts.Drafts[0].Suggestion, err)
	}
}

func TestGivenTheSheetFilledInPastADraftWhenListingDraftsThenItIsPossiblyRecorded(t *testing.T) {
	f := newInboxFixture(t)
	capture := f.submit(t, inboxPNG, "")
	f.reader.results = []domain.CaptureResult{{Suggestions: []domain.CaptureSuggestion{
		row("Sacombank", 50000, "2026-10-14 08:15", capture.ID+"-image"),
		row("Sacombank", 70000, "2026-10-16 08:15", capture.ID+"-image"),
	}}}
	f.process(t)

	// Other rows entered by hand after the capture, up to the 15th.
	f.ledger.transactions = []domain.Transaction{{ID: "tx9", AccountName: "Sacombank", Date: time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC), Payment: floatPointer(12000)}}
	drafts, err := f.inbox.Drafts(context.Background())

	if err != nil || len(drafts.Drafts) != 2 {
		t.Fatalf("drafts = %+v err=%v", drafts.Drafts, err)
	}
	byDate := map[string]domain.CaptureSuggestion{}
	for _, draft := range drafts.Drafts {
		byDate[draft.Suggestion.Transaction.Date] = draft.Suggestion
	}
	if older := byDate["14/10/2026"]; older.RecordedThrough != "15/10/2026" || older.Selected {
		t.Fatalf("older draft = %+v, want possibly recorded and unselected", older)
	}
	if newer := byDate["16/10/2026"]; newer.RecordedThrough != "" || !newer.Selected {
		t.Fatalf("newer draft = %+v, want new and selected", newer)
	}
}

func TestQuickCaptureDiscardRefusesCapturesWithDrafts(t *testing.T) {
	f := newInboxFixture(t)
	capture := f.submit(t, inboxPNG, "")
	f.reader.results = []domain.CaptureResult{{Suggestions: []domain.CaptureSuggestion{row("Sacombank", 50000, "2026-10-16 08:15", capture.ID+"-image")}}}
	f.process(t)

	if err := f.inbox.Discard(context.Background(), capture.ID); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("discard: err = %v, want conflict", err)
	}
}

func TestQuickCaptureRunProcessesSubmissionsUntilStopped(t *testing.T) {
	f := newInboxFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		f.inbox.Run(ctx)
		close(done)
	}()

	capture := f.submit(t, inboxPNG, "")
	deadline := time.Now().Add(5 * time.Second)
	for {
		stored, err := f.store.GetCapture(context.Background(), capture.ID)
		if err == nil && stored.Status == domain.QuickCaptureDone {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("capture = %+v err=%v, want done by the worker", stored, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not stop")
	}
}

func floatPointer(value float64) *float64 { return &value }

func TestQuickCaptureEditSavesTheRowWithoutWritingIt(t *testing.T) {
	f := newInboxFixture(t)
	capture := f.submit(t, inboxPNG, "")
	f.reader.results = []domain.CaptureResult{{Suggestions: []domain.CaptureSuggestion{row("Sacombank", 50000, "2026-10-16 08:15", capture.ID+"-image")}}}
	f.process(t)
	drafts, _ := f.inbox.Drafts(context.Background())
	id := drafts.Drafts[0].ID
	edited := drafts.Drafts[0].Suggestion.Transaction
	edited.Payee, edited.Category = "Pho 24", "Food"

	if err := f.inbox.Edit(context.Background(), id, domain.CaptureDraftEdit{Transaction: edited, CategorySource: domain.CategorySourceAI}); err != nil {
		t.Fatalf("edit: %v", err)
	}
	reloaded, _ := f.inbox.Drafts(context.Background())
	if got := reloaded.Drafts[0].Edit; got == nil || got.Transaction.Payee != "Pho 24" || got.CategorySource != domain.CategorySourceAI {
		t.Fatalf("edit = %+v, want the saved row", got)
	}
	if len(f.ledger.batches) != 0 {
		t.Fatalf("batches = %d, want nothing written", len(f.ledger.batches))
	}
	if err := f.inbox.Edit(context.Background(), id, domain.CaptureDraftEdit{Transaction: edited, CategorySource: "guess"}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("unknown source: err = %v, want validation", err)
	}
	if err := f.inbox.Reject(context.Background(), []string{id}); err != nil {
		t.Fatalf("reject: %v", err)
	}
	if err := f.inbox.Edit(context.Background(), id, domain.CaptureDraftEdit{Transaction: edited}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("edit rejected draft: err = %v, want conflict", err)
	}
}
