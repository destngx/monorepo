package domain

import (
	"strings"
	"testing"
	"time"
)

func dedupDay(d int) time.Time { return time.Date(2026, 10, d, 0, 0, 0, 0, time.UTC) }

func TestGivenReferencesOfDifferentKindsWhenFindingDuplicatesThenTheyDoNotConflict(t *testing.T) {
	// A bank's own number and the transfer network's code name one transfer twice.
	ledger := []Transaction{{ID: "r1", AccountName: "Techcombank", Payee: "Tran Thi B", Date: dedupDay(17), Deposit: amountPointer(1200000), ReferenceNumber: note(`FT26000000000001\BNK`)}}
	candidate := CaptureCandidate{Account: "Techcombank", Direction: CaptureIn, Amount: 1200000, Date: dedupDay(17), Reference: "MS00T11223344556677", Text: "TRAN T..."}

	got := FindCaptureDuplicates([]CaptureCandidate{candidate}, ledger)

	if match, ok := got[0]; !ok || match.Score != 99 {
		t.Fatalf("expected a same-day text match: %+v", got)
	}
}

func TestGivenReferencesOfOneKindWhenTheyDifferThenTheRowsAreDifferentTransactions(t *testing.T) {
	ledger := []Transaction{{ID: "r1", AccountName: "Techcombank", Date: dedupDay(17), Deposit: amountPointer(50000), ReferenceNumber: note("FT26000000000001")}}
	candidate := CaptureCandidate{Account: "Techcombank", Direction: CaptureIn, Amount: 50000, Date: dedupDay(17), Reference: "FT26000000000999"}

	if got := FindCaptureDuplicates([]CaptureCandidate{candidate}, ledger); len(got) != 0 {
		t.Fatalf("two bank numbers that differ are two transfers: %+v", got)
	}
}

func TestGivenTheReferenceInTheRowMemoWhenFindingDuplicatesThenItMatchesEvenOneCharacterOff(t *testing.T) {
	memo := "[capture] Techcombank screenshot - MS00T1122334456677,TRAN THI B"
	ledger := []Transaction{{ID: "r1", AccountName: "Techcombank", Payee: "Someone else", Date: dedupDay(17), Deposit: amountPointer(1200000), ReferenceNumber: note(`FT26000000000001\BNK`), Memo: &memo}}
	candidate := CaptureCandidate{Account: "Techcombank", Direction: CaptureIn, Amount: 1200000, Date: dedupDay(17), Reference: "MS00T11223344556677"}

	got := FindCaptureDuplicates([]CaptureCandidate{candidate}, ledger)

	if match, ok := got[0]; !ok || match.Score != 100 {
		t.Fatalf("a reference read one digit off in the memo is the same transfer: %+v", got)
	}
	short := CaptureCandidate{Account: "Techcombank", Direction: CaptureIn, Amount: 1200000, Date: dedupDay(17), Reference: "FT2"}
	shortRow := ledger[0]
	shortRow.ReferenceNumber = note("FT3")
	if got := FindCaptureDuplicates([]CaptureCandidate{short}, []Transaction{shortRow}); len(got) != 0 {
		t.Fatalf("short references must match exactly: %+v", got)
	}
}

func TestGivenADraftWithNoAccountWhenCheckingDuplicatesThenItMatchesAnyAccountAndTakesItsAccount(t *testing.T) {
	memo := "[capture] Techcombank screenshot - MS00T1122334456677,TRAN THI B"
	ledger := []Transaction{
		{ID: "r1", AccountName: "Techcombank", Payee: "Tran Thi B", Date: dedupDay(17), Deposit: amountPointer(1200000), ReferenceNumber: note(`FT26000000000001\BNK`), Memo: &memo},
		{ID: "r2", AccountName: "Ví MoMo", Payee: "Gym membership", Date: dedupDay(15), Payment: amountPointer(250000)},
	}
	reference := "MS00T11223344556677"
	suggestions := []CaptureSuggestion{
		{Direction: CaptureIn, Amount: 1200000, LowConfidence: []string{"account"}, Transaction: TransactionCreateInput{Date: "17/10/2026", Payee: "TRAN T...", ReferenceNumber: &reference}},
		{Direction: CaptureOut, Amount: 250000, LowConfidence: []string{"account"}, Transaction: TransactionCreateInput{Date: "17/10/2026", Payee: "Transfer to D..."}},
	}

	ApplyCaptureDuplicates(suggestions, ledger, dedupDay(20))

	filled := suggestions[0]
	if filled.Duplicate == nil || filled.Duplicate.TransactionID != "r1" || filled.Duplicate.Score != 100 || filled.Duplicate.Account != "Techcombank" {
		t.Fatalf("expected the memo reference to match r1: %+v", filled.Duplicate)
	}
	if filled.Transaction.AccountName != "Techcombank" || filled.AccountSource != MatchSourceLedger || len(filled.LowConfidence) != 0 {
		t.Fatalf("a likely match fills the account: %+v", filled)
	}
	// 90 - 5*2 days - 10 for the unknown account.
	possible := suggestions[1]
	if possible.Duplicate == nil || possible.Duplicate.Score != 70 || possible.Duplicate.Account != "Ví MoMo" {
		t.Fatalf("expected a possible match on Ví MoMo: %+v", possible.Duplicate)
	}
	if possible.Transaction.AccountName != "Ví MoMo" || possible.AccountSource != MatchSourceLedger || len(possible.LowConfidence) != 1 {
		t.Fatalf("a possible match fills the account, still highlighted to check: %+v", possible)
	}
}

func TestGivenARowWithTheSameReferenceButAnotherAmountWhenCheckingDuplicatesThenItIsShown(t *testing.T) {
	ledger := []Transaction{{ID: "r9", AccountName: "Techcombank", Payee: "Tran Thi B", Date: dedupDay(19), Deposit: amountPointer(1500000), ReferenceNumber: note("MS00T11223344556677")}}
	reference := "MS00T11223344556677"
	suggestions := []CaptureSuggestion{{Confidence: 0.95, Direction: CaptureIn, Amount: 1200000, AccountSource: MatchSourceAlias, LowConfidence: []string{},
		Transaction: TransactionCreateInput{AccountName: "Techcombank", Date: "17/10/2026", Payee: "TRAN T...", ReferenceNumber: &reference}}}

	ApplyCaptureDuplicates(suggestions, ledger, dedupDay(20))
	ApplyCaptureConfidence(suggestions)

	got := suggestions[0]
	if got.Duplicate != nil || got.SameReference == nil || got.SameReference.TransactionID != "r9" || got.SameReference.Account != "Techcombank" {
		t.Fatalf("expected r9 as the same reference: %+v", got)
	}
	if got.Overall != 75 || !strings.Contains(reasonTexts(got), "Same reference as r9 (Tran Thi B, 19/10/2026) with another amount or date") {
		t.Fatalf("overall=%d reasons=%q", got.Overall, reasonTexts(got))
	}
}
