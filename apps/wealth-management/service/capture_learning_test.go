package service

import (
	"apps/wealth-management/adapter/captureinbox"
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

type fakeRuleBook struct {
	rules   []domain.CaptureRule
	saved   [][]domain.CaptureRule
	saveErr error
}

func (f *fakeRuleBook) ListCaptureRules(bool) ([]domain.CaptureRule, error) { return f.rules, nil }
func (f *fakeRuleBook) SaveCaptureRules(rules []domain.CaptureRule) (domain.CaptureRulesSaveResult, error) {
	if f.saveErr != nil {
		return domain.CaptureRulesSaveResult{}, f.saveErr
	}
	f.saved = append(f.saved, rules)
	f.rules = append(f.rules, rules...)
	return domain.CaptureRulesSaveResult{Created: len(rules)}, nil
}

func openMemoryStore(t *testing.T) *captureinbox.Store {
	t.Helper()
	store, err := captureinbox.Open(filepath.Join(t.TempDir(), "capture.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func counterpartyRow(counterparty string, amount float64, date string) domain.CaptureSuggestion {
	suggestion := row("Vietcombank", amount, date+" 08:00", "m1")
	suggestion.Counterparty = counterparty
	suggestion.Transaction.Payee = counterparty
	return suggestion
}

func TestCaptureLearningStoresWrittenRowsAndSavesTheCounterpartyRules(t *testing.T) {
	store := openMemoryStore(t)
	rules := &fakeRuleBook{}
	ids := 0
	learning := NewCaptureLearning(store, rules, CaptureLearningOptions{
		Now:   func() time.Time { return time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC) },
		NewID: func() string { ids++; return fmt.Sprintf("ID%d", ids) },
	})
	written := counterpartyRow("NGUYEN VAN A", 25000, "2026-10-16").Transaction
	written.Payee, written.Category = "Banh mi stall", "Eating out"

	result, err := learning.LearnWritten(context.Background(), []domain.CaptureWrittenRow{
		{Suggestion: counterpartyRow("NGUYEN VAN A", 25000, "2026-10-16"), Transaction: written, TransactionID: "tx1", Note: "breakfast stall by the office"},
		{Suggestion: counterpartyRow("PHAM VAN D", 40000, "2026-10-16"), Transaction: counterpartyRow("PHAM VAN D", 40000, "2026-10-16").Transaction, TransactionID: "tx2"},
	})

	if err != nil || result.Remembered != 2 || result.Rules.Created != 1 {
		t.Fatalf("result = %+v err=%v", result, err)
	}
	if len(rules.saved) != 1 || rules.saved[0][0] != (domain.CaptureRule{Kind: domain.CaptureRuleCounterparty, Pattern: "nguyen van a", Value: "Banh mi stall"}) {
		t.Fatalf("saved rules = %+v, want only the renamed counterparty", rules.saved)
	}
	memories, err := store.MemoriesByKey(context.Background(), []string{"nguyen van a"})
	if err != nil || len(memories) != 1 || memories[0].ID != "mem_id1" || memories[0].Category != "Eating out" || memories[0].TransactionID != "tx1" || memories[0].Note != "breakfast stall by the office" {
		t.Fatalf("memories = %+v err=%v", memories, err)
	}
	if _, err := learning.LearnWritten(context.Background(), []domain.CaptureWrittenRow{{Suggestion: counterpartyRow("X", 1, "2026-10-16"), Transaction: written}}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("a row without its transaction id: err = %v, want validation", err)
	}
}

func newLearningInboxFixture(t *testing.T, rules *fakeRuleBook) *inboxFixture {
	t.Helper()
	fixture := newInboxFixture(t)
	ids := 0
	learning := NewCaptureLearning(fixture.store, rules, CaptureLearningOptions{
		Now:   func() time.Time { return fixture.now },
		NewID: func() string { ids++; return fmt.Sprintf("L%d", ids) },
	})
	draftIDs := 0
	fixture.inbox = NewCaptureInbox(fixture.store, fixture.reader, fixture.ledger, logger.NewTestLogger(t), CaptureInboxOptions{
		Now:     func() time.Time { return fixture.now },
		NewID:   func() string { draftIDs++; return fmt.Sprintf("ID%d", draftIDs) },
		Learner: learning,
	})
	return fixture
}

func TestGivenADraftApprovedUnderAnotherPayeeWhenApprovingThenTheCounterpartyIsLearned(t *testing.T) {
	rules := &fakeRuleBook{}
	f := newLearningInboxFixture(t, rules)
	f.submit(t, inboxPNG, "")
	f.reader.results = []domain.CaptureResult{{Suggestions: []domain.CaptureSuggestion{counterpartyRow("NGUYEN VAN A", 25000, "2026-10-16")}}}
	f.process(t)
	drafts, _ := f.inbox.Drafts(context.Background())
	edited := drafts.Drafts[0].Suggestion.Transaction
	edited.Payee = "Banh mi stall"

	if _, err := f.inbox.Approve(context.Background(), domain.CaptureDraftApproval{BatchID: "b1", Drafts: []domain.CaptureDraftApprovalRow{{ID: drafts.Drafts[0].ID, Transaction: edited, Note: "breakfast"}}}); err != nil {
		t.Fatalf("approve: %v", err)
	}

	if len(rules.saved) != 1 || rules.saved[0][0].Pattern != "nguyen van a" || rules.saved[0][0].Value != "Banh mi stall" {
		t.Fatalf("saved rules = %+v", rules.saved)
	}
	memories, _ := f.store.MemoriesByKey(context.Background(), []string{"nguyen van a"})
	if len(memories) != 1 || memories[0].ID != "mem_"+drafts.Drafts[0].ID || memories[0].TransactionID != "tx1" || memories[0].Source != domain.CaptureMemoryApproved || memories[0].Note != "breakfast" {
		t.Fatalf("memories = %+v, want one keyed by the draft", memories)
	}
}

func TestGivenRulesCannotBeSavedWhenApprovingThenTheApprovalStillSucceeds(t *testing.T) {
	rules := &fakeRuleBook{saveErr: errors.New("sheet down")}
	f := newLearningInboxFixture(t, rules)
	f.submit(t, inboxPNG, "")
	f.reader.results = []domain.CaptureResult{{Suggestions: []domain.CaptureSuggestion{counterpartyRow("NGUYEN VAN A", 25000, "2026-10-16")}}}
	f.process(t)
	drafts, _ := f.inbox.Drafts(context.Background())
	edited := drafts.Drafts[0].Suggestion.Transaction
	edited.Payee = "Banh mi stall"

	result, err := f.inbox.Approve(context.Background(), domain.CaptureDraftApproval{BatchID: "b1", Drafts: []domain.CaptureDraftApprovalRow{{ID: drafts.Drafts[0].ID, Transaction: edited}}})

	if err != nil || result.Created != 1 {
		t.Fatalf("approve = %+v err=%v, want written despite the rules failure", result, err)
	}
}

func TestGivenALikelyDuplicateRejectedWhenRejectingThenTheSheetPayeeIsLearned(t *testing.T) {
	rules := &fakeRuleBook{}
	f := newLearningInboxFixture(t, rules)
	f.submit(t, inboxPNG, "")
	f.reader.results = []domain.CaptureResult{{Suggestions: []domain.CaptureSuggestion{
		counterpartyRow("NGUYEN VAN A", 25000, "2026-10-16"),
		counterpartyRow("TRAN THI B", 70000, "2026-10-16"),
	}}}
	f.process(t)
	// Entered by hand under the user's own name for the stall; nothing
	// matches the second draft.
	f.ledger.transactions = []domain.Transaction{{ID: "tx9", AccountName: "Vietcombank", Payee: "Banh mi stall", Category: "Eating out", Date: time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC), Payment: floatPointer(25000)}}
	drafts, _ := f.inbox.Drafts(context.Background())
	// A note saved on the draft before rejecting it is kept.
	if err := f.inbox.Edit(context.Background(), drafts.Drafts[0].ID, domain.CaptureDraftEdit{Transaction: drafts.Drafts[0].Suggestion.Transaction, Note: "same as the stall row"}); err != nil {
		t.Fatalf("edit: %v", err)
	}

	if err := f.inbox.Reject(context.Background(), []string{drafts.Drafts[0].ID, drafts.Drafts[1].ID}); err != nil {
		t.Fatalf("reject: %v", err)
	}

	if len(rules.saved) != 1 || len(rules.saved[0]) != 1 || rules.saved[0][0].Pattern != "nguyen van a" || rules.saved[0][0].Value != "Banh mi stall" {
		t.Fatalf("saved rules = %+v, want the duplicate's sheet payee only", rules.saved)
	}
	memories, _ := f.store.MemoriesByKey(context.Background(), []string{"nguyen van a", "tran thi b"})
	if len(memories) != 1 || memories[0].Source != domain.CaptureMemoryDuplicate || memories[0].TransactionID != "tx9" || memories[0].Category != "Eating out" || memories[0].Note != "same as the stall row" {
		t.Fatalf("memories = %+v", memories)
	}
}
