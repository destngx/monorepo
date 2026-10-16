package domain

import (
	"strings"
	"testing"
	"time"
)

func draftSuggestion(account string, direction CaptureDirection, amount float64, occurredAt string, reference string) CaptureSuggestion {
	date, _ := time.Parse("2006-01-02", occurredAt[:10])
	suggestion := CaptureSuggestion{
		Transaction: TransactionCreateInput{AccountName: account, Date: date.Format(SheetDateLayout)},
		Direction:   direction,
		Amount:      amount,
		OccurredAt:  occurredAt,
	}
	if reference != "" {
		suggestion.Transaction.ReferenceNumber = stringPointer(reference)
	}
	return suggestion
}

func draftOf(id string, status CaptureDraftStatus, suggestion CaptureSuggestion) CaptureDraft {
	return CaptureDraft{ID: id, Status: status, Suggestion: suggestion}
}

func TestMatchCaptureDraftsLinksTheSameRowReadAgain(t *testing.T) {
	drafts := []CaptureDraft{
		draftOf("d1", CaptureDraftPending, draftSuggestion("Techcombank", CaptureOut, 50000, "2026-10-16 08:15", "")),
		draftOf("d2", CaptureDraftPending, draftSuggestion("Techcombank", CaptureOut, 120000, "2026-10-16 12:30", "")),
	}
	suggestions := []CaptureSuggestion{
		draftSuggestion("Techcombank", CaptureOut, 120000, "2026-10-16 12:30", ""),
		draftSuggestion("Techcombank", CaptureOut, 75000, "2026-10-16 18:00", ""),
	}

	got := MatchCaptureDrafts(suggestions, drafts)

	if got[0] != 1 || got[1] != -1 {
		t.Fatalf("matches = %v, want [1 -1]", got)
	}
}

func TestMatchCaptureDraftsIsOneToOneForIdenticalRows(t *testing.T) {
	// Two coffees of the same price on the same day, no time shown: a second
	// screenshot of the list must link each to its own draft, not both to one.
	drafts := []CaptureDraft{
		draftOf("d1", CaptureDraftPending, draftSuggestion("ZaloPay", CaptureOut, 45000, "2026-10-16", "")),
		draftOf("d2", CaptureDraftPending, draftSuggestion("ZaloPay", CaptureOut, 45000, "2026-10-16", "")),
	}
	suggestions := []CaptureSuggestion{
		draftSuggestion("ZaloPay", CaptureOut, 45000, "2026-10-16", ""),
		draftSuggestion("ZaloPay", CaptureOut, 45000, "2026-10-16", ""),
		draftSuggestion("ZaloPay", CaptureOut, 45000, "2026-10-16", ""),
	}

	got := MatchCaptureDrafts(suggestions, drafts)

	if got[0] != 0 || got[1] != 1 || got[2] != -1 {
		t.Fatalf("matches = %v, want [0 1 -1]", got)
	}
}

func TestMatchCaptureDraftsRequiresEqualTimesAndReferencesWhenBothKnown(t *testing.T) {
	drafts := []CaptureDraft{
		draftOf("d1", CaptureDraftPending, draftSuggestion("Sacombank", CaptureOut, 50000, "2026-10-16 08:15", "FT1")),
	}
	cases := map[string]CaptureSuggestion{
		"other time":      draftSuggestion("Sacombank", CaptureOut, 50000, "2026-10-16 09:15", ""),
		"other reference": draftSuggestion("Sacombank", CaptureOut, 50000, "2026-10-16 08:15", "FT2"),
		"other day":       draftSuggestion("Sacombank", CaptureOut, 50000, "2026-10-17", ""),
		"other direction": draftSuggestion("Sacombank", CaptureIn, 50000, "2026-10-16 08:15", ""),
		"other account":   draftSuggestion("Techcombank", CaptureOut, 50000, "2026-10-16 08:15", ""),
		"other amount":    draftSuggestion("Sacombank", CaptureOut, 50001, "2026-10-16 08:15", ""),
	}
	for name, suggestion := range cases {
		if got := MatchCaptureDrafts([]CaptureSuggestion{suggestion}, drafts); got[0] != -1 {
			t.Errorf("%s: matched draft %d, want no match", name, got[0])
		}
	}

	dateOnly := draftSuggestion("Sacombank", CaptureOut, 50000, "2026-10-16", "")
	if got := MatchCaptureDrafts([]CaptureSuggestion{dateOnly}, drafts); got[0] != 0 {
		t.Errorf("date-only read of the same row: matches = %v, want [0]", got)
	}
}

func TestMatchCaptureDraftsPrefersReferenceAndTimeOverDateOnly(t *testing.T) {
	drafts := []CaptureDraft{
		draftOf("d1", CaptureDraftPending, draftSuggestion("Sacombank", CaptureOut, 50000, "2026-10-16", "")),
		draftOf("d2", CaptureDraftPending, draftSuggestion("Sacombank", CaptureOut, 50000, "2026-10-16 08:15", "FT9")),
	}
	suggestions := []CaptureSuggestion{
		draftSuggestion("Sacombank", CaptureOut, 50000, "2026-10-16", ""),
		draftSuggestion("Sacombank", CaptureOut, 50000, "2026-10-16 08:15", "FT9"),
	}

	got := MatchCaptureDrafts(suggestions, drafts)

	if got[0] != 0 || got[1] != 1 {
		t.Fatalf("matches = %v, want [0 1]", got)
	}
}

func TestMatchCaptureDraftsUsesTheAccountHintWhenNoAccountMatched(t *testing.T) {
	draft := draftSuggestion("", CaptureOut, 50000, "2026-10-16", "")
	draft.AccountHint = "MoMo *1234"
	again := draftSuggestion("", CaptureOut, 50000, "2026-10-16", "")
	again.AccountHint = "momo *1234"
	other := draftSuggestion("", CaptureOut, 50000, "2026-10-16", "")
	other.AccountHint = "ShopeePay"

	got := MatchCaptureDrafts([]CaptureSuggestion{other, again}, []CaptureDraft{draftOf("d1", CaptureDraftRejected, draft)})

	if got[0] != -1 || got[1] != 0 {
		t.Fatalf("matches = %v, want [-1 0]", got)
	}
}

func TestMergeCaptureDraftEvidenceAddsNewSourcesOnly(t *testing.T) {
	draft := CaptureDraft{
		ID:         "d1",
		CaptureIDs: []string{"c1"},
		Suggestion: CaptureSuggestion{Evidence: []CaptureEvidence{{MaterialID: "c1-image", Quote: "-50.000"}}},
	}
	again := CaptureSuggestion{Evidence: []CaptureEvidence{{MaterialID: "c2-image", Quote: "-50,000 VND"}}}

	MergeCaptureDraft(&draft, again, "c2", "team lunch")
	MergeCaptureDraft(&draft, again, "c2", "team lunch")

	if len(draft.Suggestion.Evidence) != 2 || draft.Suggestion.Evidence[1].MaterialID != "c2-image" {
		t.Fatalf("evidence = %+v, want c1-image then c2-image", draft.Suggestion.Evidence)
	}
	if strings.Join(draft.CaptureIDs, ",") != "c1,c2" {
		t.Fatalf("captureIds = %v, want [c1 c2]", draft.CaptureIDs)
	}
	if draft.Suggestion.Transaction.Memo == nil || *draft.Suggestion.Transaction.Memo != "team lunch" {
		t.Fatalf("memo = %v, want the later note added once", draft.Suggestion.Transaction.Memo)
	}
}

func TestNewCaptureDraftsScopesIdentifiersToTheCapture(t *testing.T) {
	suggestions := []CaptureSuggestion{
		{ID: "s1", TransferGroup: "t1", Transaction: TransactionCreateInput{Memo: stringPointer("[capture] Sacombank screenshot")}},
		{ID: "s2", TransferGroup: "t1"},
	}
	ids := []string{"d_a", "d_b"}
	now := time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)

	drafts := NewCaptureDrafts(suggestions, "c_7", "lunch with team", now, func() string {
		id := ids[0]
		ids = ids[1:]
		return id
	})

	if drafts[0].ID != "d_a" || drafts[0].Suggestion.ID != "d_a" || drafts[1].Suggestion.ID != "d_b" {
		t.Fatalf("ids = %s/%s, %s, want draft ids on the suggestions", drafts[0].ID, drafts[0].Suggestion.ID, drafts[1].Suggestion.ID)
	}
	if drafts[0].Suggestion.TransferGroup != "c_7-t1" || drafts[1].Suggestion.TransferGroup != "c_7-t1" {
		t.Fatalf("transfer groups = %q, %q, want them scoped to the capture", drafts[0].Suggestion.TransferGroup, drafts[1].Suggestion.TransferGroup)
	}
	if got := *drafts[0].Suggestion.Transaction.Memo; got != "[capture] Sacombank screenshot - lunch with team" {
		t.Fatalf("memo = %q, want the note appended", got)
	}
	if got := *drafts[1].Suggestion.Transaction.Memo; got != "lunch with team" {
		t.Fatalf("memo without a capture memo = %q, want the note", got)
	}
	if drafts[0].Status != CaptureDraftPending || !drafts[0].CreatedAt.Equal(now) || drafts[0].CaptureIDs[0] != "c_7" {
		t.Fatalf("draft = %+v, want pending, created now, from c_7", drafts[0])
	}
}

func TestQuickCaptureMaterials(t *testing.T) {
	png := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 64)...)
	limits := DefaultCaptureLimits()

	materials, err := QuickCaptureMaterials("c1", QuickCaptureInput{Image: png, Note: "  cash 50k  "}, limits)
	if err != nil {
		t.Fatalf("materials: %v", err)
	}
	if len(materials) != 2 || materials[0].ID != "c1-image" || materials[0].MediaType != "image/png" || materials[1].ID != "c1-note" || materials[1].Text != "cash 50k" {
		t.Fatalf("materials = %+v, want image then trimmed note", materials)
	}

	if _, err := QuickCaptureMaterials("c1", QuickCaptureInput{Note: " "}, limits); err == nil {
		t.Fatal("empty capture: want a validation error")
	}
	if _, err := QuickCaptureMaterials("c1", QuickCaptureInput{Image: []byte("not an image")}, limits); err == nil || !strings.Contains(err.Error(), "media type") {
		t.Fatalf("text as image: err = %v, want a media type error", err)
	}
	if _, err := QuickCaptureMaterials("c1", QuickCaptureInput{Note: strings.Repeat("x", QuickCaptureMaxNoteBytes+1)}, limits); err == nil {
		t.Fatal("long note: want a validation error")
	}
}

func TestQuickCaptureHashIgnoresNoteWhitespace(t *testing.T) {
	a := QuickCaptureInput{Image: []byte("img"), Note: "lunch"}.Hash()
	b := QuickCaptureInput{Image: []byte("img"), Note: " lunch "}.Hash()
	c := QuickCaptureInput{Image: []byte("img"), Note: "dinner"}.Hash()
	if a != b || a == c {
		t.Fatalf("hashes: same=%v different=%v, want equal for whitespace and different for another note", a == b, a != c)
	}
}

func TestQuickCaptureRetryDelay(t *testing.T) {
	if delay, ok := QuickCaptureRetryDelay(1); !ok || delay != time.Minute {
		t.Fatalf("first retry = %v %v, want 1m", delay, ok)
	}
	if _, ok := QuickCaptureRetryDelay(len(quickCaptureBackoff) + 1); ok {
		t.Fatal("after the last backoff step: want no more retries")
	}
}

func TestLatestCaptureBalancesKeepsTheNewestPerAccount(t *testing.T) {
	captures := []QuickCapture{
		{Balances: []CaptureBalance{{Account: "Techcombank", BalanceAfter: 100, AsOf: "2026-10-15 09:00"}}},
		{Balances: []CaptureBalance{{Account: "Techcombank", BalanceAfter: 80, AsOf: "2026-10-16 09:00"}, {Account: "ZaloPay", BalanceAfter: 5, AsOf: "2026-10-16"}}},
	}
	accounts := []Account{{Name: "Techcombank", Balance: 90}, {Name: "ZaloPay", Balance: 5}}

	got := LatestCaptureBalances(captures, accounts)

	if len(got) != 2 || got[0].Account != "Techcombank" || got[0].BalanceAfter != 80 || got[0].SheetBalance != 90 {
		t.Fatalf("balances = %+v, want the 16/10 Techcombank balance against the current sheet balance", got)
	}
}
