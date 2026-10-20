package domain

import (
	"strings"
	"testing"
	"time"
)

func TestCounterpartyKeyKeepsTheNameAndDropsCodesAndGenericWords(t *testing.T) {
	tests := map[string]string{
		"NGUYEN VAN A":                 "nguyen van a",
		"Nguyễn Văn A":                 "nguyen van a",
		"CT NGUYEN VAN A 0123456789":   "nguyen van a",
		"Chuyển tiền đến NGUYEN VAN A": "nguyen van a",
		"Chuyển tiền/thanh toán":       "",
		"  ":                           "",
	}
	for text, want := range tests {
		if got := CounterpartyKey(text); got != want {
			t.Errorf("CounterpartyKey(%q) = %q, want %q", text, got, want)
		}
	}
}

func TestCounterpartyRulesAreCaptureRules(t *testing.T) {
	rule := CaptureRule{Kind: CaptureRuleCounterparty, Pattern: "Nguyễn Văn A 0123", Value: "Banh mi stall"}
	if err := rule.Validate(); err != nil {
		t.Fatal(err)
	}
	if rule.Key() != (CaptureRule{Kind: CaptureRuleCounterparty, Pattern: "NGUYEN VAN A"}).Key() {
		t.Fatal("counterparty rules are keyed by the counterparty key")
	}
	if err := (CaptureRule{Kind: CaptureRuleCounterparty, Pattern: "chuyen tien 123", Value: "x"}).Validate(); err == nil {
		t.Fatal("a pattern with no name words is not a rule")
	}
}

func learnedSuggestion(counterparty, payee string) CaptureSuggestion {
	return CaptureSuggestion{
		Counterparty: counterparty, Direction: CaptureOut, Amount: 25000, SourceApp: "Vietcombank",
		Evidence:    []CaptureEvidence{{MaterialID: "m1", Quote: "-25,000 " + counterparty}},
		Transaction: TransactionCreateInput{AccountName: "Vietcombank", Date: "12/10/2026", Payee: payee, ReferenceNumber: note("FT1")},
	}
}

func TestNewCaptureMemoryKeepsWhatWasReadAndWhatWasWritten(t *testing.T) {
	at := time.Date(2026, 10, 12, 8, 0, 0, 0, time.UTC)
	written := TransactionCreateInput{AccountName: "Vietcombank", Date: "12/10/2026", Payee: "Banh mi stall", Category: "Eating out", Tags: []string{"breakfast"}}

	memory, ok := NewCaptureMemory(learnedSuggestion("NGUYEN VAN A", "NGUYEN VAN A"), written, "  near the office, breakfast  ", "row-7", CaptureMemoryApproved, at, "mem1")

	if !ok || memory.ID != "mem1" || memory.CounterpartyKey != "nguyen van a" || memory.Counterparty != "NGUYEN VAN A" || memory.Payee != "Banh mi stall" ||
		memory.Category != "Eating out" || memory.TransactionID != "row-7" || memory.Source != CaptureMemoryApproved || memory.Reference != "FT1" ||
		memory.Quote != "-25,000 NGUYEN VAN A" || memory.SourceApp != "Vietcombank" || memory.Amount != 25000 || !memory.CreatedAt.Equal(at) ||
		memory.Note != "near the office, breakfast" {
		t.Fatalf("memory = %+v", memory)
	}
	transfer := learnedSuggestion("NGUYEN VAN A", "Ví MoMo")
	transfer.TransferGroup = "t1"
	if _, ok := NewCaptureMemory(transfer, written, "", "row-8", CaptureMemoryApproved, at, "mem2"); ok {
		t.Fatal("own-account transfers teach nothing")
	}
	fee := learnedSuggestion("", "Vietcombank fee")
	fee.Fee = true
	if _, ok := NewCaptureMemory(fee, written, "", "row-9", CaptureMemoryApproved, at, "mem3"); ok {
		t.Fatal("rows with no counterparty, like fee rows, teach nothing")
	}
	// Drafts saved before the counterparty was kept fall back to the payee
	// the engine read.
	old := learnedSuggestion("", "NGUYEN VAN B")
	if memory, ok := NewCaptureMemory(old, written, "", "row-10", CaptureMemoryApproved, at, "mem4"); !ok || memory.CounterpartyKey != "nguyen van b" {
		t.Fatalf("old draft memory = %+v ok=%v", memory, ok)
	}
}

func TestCounterpartyRuleUpdates(t *testing.T) {
	memory := func(counterparty, payee string) CaptureMemory {
		return CaptureMemory{Counterparty: counterparty, CounterpartyKey: CounterpartyKey(counterparty), Payee: payee}
	}
	existing := []CaptureRule{
		{Kind: CaptureRuleCounterparty, Pattern: "tran thi b", Value: "Fruit stall"},
		{Kind: CaptureRuleCounterparty, Pattern: "le van c", Value: "Laundry"},
		{Kind: CaptureRulePayee, Pattern: "nguyen van a", Value: "Eating out"},
	}

	updates := CounterpartyRuleUpdates([]CaptureMemory{
		memory("NGUYEN VAN A", "Banh mi"),       // renamed: learned
		memory("NGUYEN VAN A", "Banh mi stall"), // renamed again: the latest wins
		memory("PHAM VAN D", "Pham Van D"),      // kept the bank's name: nothing to learn
		memory("TRAN THI B", "TRAN THI B"),      // set back to the bank's name: the rule follows
		memory("LE VAN C", "Laundry"),           // same as the rule: unchanged
	}, existing)

	got := map[string]string{}
	for _, rule := range updates {
		if rule.Kind != CaptureRuleCounterparty {
			t.Fatalf("unexpected rule kind: %+v", rule)
		}
		got[rule.Pattern] = rule.Value
	}
	if len(got) != 2 || got["nguyen van a"] != "Banh mi stall" || got["tran thi b"] != "TRAN THI B" {
		t.Fatalf("updates = %+v", updates)
	}
}

func TestApplyLearnedPayees(t *testing.T) {
	rules := []CaptureRule{
		{Kind: CaptureRuleCounterparty, Pattern: "nguyen van a", Value: "Banh mi stall"},
		{Kind: CaptureRuleCounterparty, Pattern: "tran thi b", Value: "Fruit stall"},
		{Kind: CaptureRulePayee, Pattern: "Le Van C", Value: "Laundry"},
	}
	memories := []CaptureMemory{
		{CounterpartyKey: "nguyen van a", Payee: "Banh mi stall"},
		{CounterpartyKey: "nguyen van a", Payee: "banh mi stall"},
		{CounterpartyKey: "tran thi b", Payee: "Fruit stall"},
		{CounterpartyKey: "tran thi b", Payee: "Lent to B"},
		{CounterpartyKey: "tran thi b", Payee: "Lent to B"},
	}
	transfer := learnedSuggestion("NGUYEN VAN A", "Ví MoMo")
	transfer.TransferGroup = "t1"
	suggestions := []CaptureSuggestion{
		learnedSuggestion("NGUYỄN VĂN A", "NGUYỄN VĂN A"),
		learnedSuggestion("TRAN THI B", "TRAN THI B"),
		learnedSuggestion("LE VAN C", "LE VAN C"),
		transfer,
	}
	suggestions[0].LowConfidence = []string{}
	suggestions[1].LowConfidence = []string{}

	ApplyLearnedPayees(suggestions, rules, memories)

	shop := suggestions[0]
	if shop.Transaction.Payee != "Banh mi stall" || shop.LearnedPayee == nil || shop.LearnedPayee.Counterparty != "NGUYỄN VĂN A" || shop.LearnedPayee.Count != 2 ||
		len(shop.LearnedPayee.Others) != 0 || len(shop.LowConfidence) != 0 {
		t.Fatalf("learned payee: %+v %+v", shop.Transaction, shop.LearnedPayee)
	}
	mixed := suggestions[1]
	if mixed.Transaction.Payee != "Fruit stall" || mixed.LearnedPayee.Count != 1 || strings.Join(mixed.LearnedPayee.Others, ",") != "Lent to B" ||
		strings.Join(mixed.LowConfidence, ",") != "payee" {
		t.Fatalf("a counterparty saved under other payees too is flagged: %+v %+v", mixed.LearnedPayee, mixed.LowConfidence)
	}
	if suggestions[2].Transaction.Payee != "LE VAN C" || suggestions[2].LearnedPayee != nil {
		t.Fatalf("payee category rules do not rename: %+v", suggestions[2])
	}
	if suggestions[3].Transaction.Payee != "Ví MoMo" || suggestions[3].LearnedPayee != nil {
		t.Fatalf("transfers keep the account as payee: %+v", suggestions[3])
	}
}

func TestGivenALearnedPayeeWhenCheckingDuplicatesThenAHandEnteredRowUnderThatNameMatches(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2026, 10, d, 0, 0, 0, 0, time.UTC) }
	ledger := []Transaction{{ID: "r1", AccountName: "Vietcombank", Payee: "Banh mi stall", Date: day(8), Payment: amountPointer(25000)}}
	suggestion := learnedSuggestion("NGUYEN VAN A", "NGUYEN VAN A")
	suggestion.Transaction.ReferenceNumber = nil
	suggestion.Selected = true
	suggestions := []CaptureSuggestion{suggestion}

	ApplyLearnedPayees(suggestions, []CaptureRule{{Kind: CaptureRuleCounterparty, Pattern: "nguyen van a", Value: "Banh mi stall"}}, nil)
	ApplyCaptureDuplicates(suggestions, ledger, day(12))

	// Four days apart: 70 on dates alone, 99 with the learned payee.
	if suggestions[0].Duplicate == nil || suggestions[0].Duplicate.Score != 99 || suggestions[0].Selected {
		t.Fatalf("duplicate = %+v", suggestions[0].Duplicate)
	}
}

func TestApplyPastNotesShowsTheLatestNotesForTheCounterparty(t *testing.T) {
	memories := []CaptureMemory{
		{CounterpartyKey: "nguyen van a", Note: "banh mi, breakfast"},
		{CounterpartyKey: "nguyen van a"},
		{CounterpartyKey: "nguyen van a", Note: "Banh mi, breakfast "},
		{CounterpartyKey: "nguyen van a", Note: "coffee too"},
		{CounterpartyKey: "nguyen van a", Note: "tet bonus for the owner"},
		{CounterpartyKey: "nguyen van a", Note: "lunch"},
		{CounterpartyKey: "tran thi b", Note: "fruit"},
	}
	transfer := learnedSuggestion("NGUYEN VAN A", "Ví MoMo")
	transfer.TransferGroup = "t1"
	suggestions := []CaptureSuggestion{learnedSuggestion("Nguyễn Văn A", "Nguyễn Văn A"), learnedSuggestion("LE VAN C", "LE VAN C"), transfer}

	ApplyPastNotes(suggestions, memories)

	if got := strings.Join(suggestions[0].PastNotes, " | "); got != "lunch | tet bonus for the owner | coffee too" {
		t.Fatalf("past notes = %q, want the 3 newest, distinct", got)
	}
	if suggestions[1].PastNotes != nil || suggestions[2].PastNotes != nil {
		t.Fatalf("no notes for other counterparties or transfers: %+v %+v", suggestions[1].PastNotes, suggestions[2].PastNotes)
	}
}

func TestCaptureNotesAreBounded(t *testing.T) {
	long := strings.Repeat("ă", CaptureNoteMaxLength+1)
	if err := (CaptureDraftEdit{Note: long}).Validate(); err == nil {
		t.Fatal("a draft note over the limit is refused")
	}
	if err := (CaptureDraftEdit{Note: strings.Repeat("ă", CaptureNoteMaxLength)}).Validate(); err != nil {
		t.Fatalf("a note at the limit is fine: %v", err)
	}
	approval := CaptureDraftApproval{BatchID: "b1", Drafts: []CaptureDraftApprovalRow{{ID: "d1", Note: long}}}
	if err := approval.Validate(); err == nil {
		t.Fatal("an approval note over the limit is refused")
	}
	if err := ValidateCaptureWrittenRows([]CaptureWrittenRow{{TransactionID: "tx1", Note: long}}); err == nil {
		t.Fatal("a written row note over the limit is refused")
	}
}
