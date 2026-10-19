package domain

import (
	"slices"
	"strings"
	"testing"
	"time"
)

func lendingRow(payee string, category string, day int) Transaction {
	return Transaction{
		AccountName: "Vietcombank", Payee: payee, Category: category, CategoryNature: NatureLending,
		Date: time.Date(2026, 9, day, 0, 0, 0, 0, time.UTC), Payment: money(1_000_000),
	}
}

func lendingLedger() []Transaction {
	return []Transaction{
		lendingRow("Lê Thử Nghiệm", "Cho vay", 1),
		lendingRow("Nguyễn Văn A", "Cho vay", 2),
		lendingRow("Trần Thị C", "Cho vay", 3),
		{AccountName: "Vietcombank", Payee: "Highlands Coffee", Category: "Coffee", Date: time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC), Payment: money(45_000)},
	}
}

func lendingSuggestion(payee string) CaptureSuggestion {
	memo := CaptureMemoPrefix + " Vietcombank screenshot"
	return CaptureSuggestion{
		Transaction:   TransactionCreateInput{AccountName: "Vietcombank", Payee: payee, Memo: &memo, Payment: money(2_000_000)},
		Direction:     CaptureOut,
		Amount:        2_000_000,
		LowConfidence: []string{},
	}
}

func TestHasLendingIntent(t *testing.T) {
	for text, want := range map[string]bool{
		"lend":                    true,
		"Lent to a friend":        true,
		"cho bạn vay":             false,
		"cho vay":                 true,
		"Cho mượn 2tr":            true,
		"trả nợ":                  true,
		"muốn ăn phở":             false,
		"váy mới":                 false,
		"blender":                 false,
		"repayment from a friend": true,
		"":                        false,
	} {
		if got := HasLendingIntent(text); got != want {
			t.Errorf("HasLendingIntent(%q) = %v, want %v", text, got, want)
		}
	}
}

func TestApplyCaptureLendingRenamesTheBankSpellingOfALendingPayee(t *testing.T) {
	suggestions := []CaptureSuggestion{lendingSuggestion("LE THU NGHIEM")}

	ApplyCaptureLending(suggestions, CaptureLendingInput{Transactions: lendingLedger(), Categories: []string{"Cho vay"}})

	row := suggestions[0]
	if row.Transaction.Payee != "Lê Thử Nghiệm" {
		t.Fatalf("payee = %q, want the ledger spelling", row.Transaction.Payee)
	}
	// Without a lending note the payee's history picks the category.
	if row.Transaction.Category != "" || slices.Contains(row.LowConfidence, CaptureFieldPayee) {
		t.Fatalf("an exact match alone must not categorise or flag: %+v", row)
	}
	if !strings.Contains(strings.Join(row.Notes, " "), "matched “Lê Thử Nghiệm” in money lent") {
		t.Fatalf("the rename must be explained: %v", row.Notes)
	}
}

func TestApplyCaptureLendingFilesALendNoteUnderLending(t *testing.T) {
	suggestions := []CaptureSuggestion{lendingSuggestion("LE THU NGHIEM")}

	ApplyCaptureLending(suggestions, CaptureLendingInput{Transactions: lendingLedger(), Categories: []string{"Lending", "Cho vay"}, Notes: []string{"lend"}})

	row := suggestions[0]
	if row.Transaction.Payee != "Lê Thử Nghiệm" || row.Transaction.Category != "Cho vay" || row.CategorySource != CategorySourceManual {
		t.Fatalf("expected Lê Thử Nghiệm under the person's lending category, got %q / %q from %q", row.Transaction.Payee, row.Transaction.Category, row.CategorySource)
	}
	if len(row.LowConfidence) != 0 {
		t.Fatalf("a matched person with a lend note is confident: %v", row.LowConfidence)
	}
}

func TestApplyCaptureLendingNearNames(t *testing.T) {
	for name, test := range map[string]struct {
		payee    string
		notes    []string
		want     string
		wantLow  bool
		category string
	}{
		"cut-off name is offered and flagged":       {payee: "LE THU NGHIE", want: "Lê Thử Nghiệm", wantLow: true},
		"typo is offered and flagged":               {payee: "LE THU NGHEIM", want: "Lê Thử Nghiệm", wantLow: true},
		"name inside the transfer text":             {payee: "MBVCB CHUYEN TIEN LE THU NGHIEM", want: "Lê Thử Nghiệm", wantLow: true},
		"word order differs":                        {payee: "NGHIEM LE THU", want: "Lê Thử Nghiệm"},
		"lend note confirms a near name":            {payee: "LE THU NGHIE", notes: []string{"lend"}, want: "Lê Thử Nghiệm", category: "Cho vay"},
		"unrelated payee is left alone":             {payee: "Highlands Coffee", want: "Highlands Coffee"},
		"short names are not near matched":          {payee: "AN", want: "AN"},
		"a payee near two people is left alone":     {payee: "NGUYEN VAN C", want: "NGUYEN VAN C"},
		"lend note without a match flags the payee": {payee: "PHAM VAN D", notes: []string{"lend"}, want: "PHAM VAN D", wantLow: true, category: "Cho vay"},
	} {
		t.Run(name, func(t *testing.T) {
			ledger := append(lendingLedger(), lendingRow("Nguyễn Văn B", "Cho vay", 5))
			suggestions := []CaptureSuggestion{lendingSuggestion(test.payee)}

			ApplyCaptureLending(suggestions, CaptureLendingInput{Transactions: ledger, Categories: []string{"Cho vay"}, Notes: test.notes})

			row := suggestions[0]
			if row.Transaction.Payee != test.want {
				t.Fatalf("payee = %q, want %q", row.Transaction.Payee, test.want)
			}
			if got := slices.Contains(row.LowConfidence, CaptureFieldPayee); got != test.wantLow {
				t.Fatalf("payee flagged = %v, want %v (notes %v)", got, test.wantLow, row.Notes)
			}
			if row.Transaction.Category != test.category {
				t.Fatalf("category = %q, want %q", row.Transaction.Category, test.category)
			}
		})
	}
}

func TestApplyCaptureLendingNoteWithSeveralRowsOnlyFilesMatchedPeople(t *testing.T) {
	suggestions := []CaptureSuggestion{lendingSuggestion("Highlands Coffee"), lendingSuggestion("LE THU NGHIEM")}

	ApplyCaptureLending(suggestions, CaptureLendingInput{Transactions: lendingLedger(), Categories: []string{"Cho vay"}, Notes: []string{"lend"}})

	if suggestions[0].Transaction.Category != "" {
		t.Fatalf("a note must not file an unmatched row among several: %+v", suggestions[0].Transaction)
	}
	if suggestions[1].Transaction.Category != "Cho vay" {
		t.Fatalf("the matched person is lending: %+v", suggestions[1].Transaction)
	}
}

func TestApplyCaptureLendingLeavesTransfersFeesAndCategorisedRows(t *testing.T) {
	transfer := lendingSuggestion("LE THU NGHIEM")
	transfer.TransferGroup = "t1"
	fee := lendingSuggestion("Vietcombank fee")
	fee.Fee = true
	categorised := lendingSuggestion("LE THU NGHIEM")
	categorised.Transaction.Category = "Food"
	suggestions := []CaptureSuggestion{transfer, fee, categorised}

	ApplyCaptureLending(suggestions, CaptureLendingInput{Transactions: lendingLedger(), Categories: []string{"Cho vay"}, Notes: []string{"lend"}})

	if suggestions[0].Transaction.Payee != "LE THU NGHIEM" || suggestions[0].Transaction.Category != "" {
		t.Fatalf("own transfers are untouched: %+v", suggestions[0].Transaction)
	}
	if suggestions[1].Transaction.Category != "" {
		t.Fatalf("fee rows are untouched: %+v", suggestions[1].Transaction)
	}
	if suggestions[2].Transaction.Category != "Food" {
		t.Fatalf("a category already set is kept: %+v", suggestions[2].Transaction)
	}
}

func TestApplyCaptureLendingReadsTheTransferContent(t *testing.T) {
	suggestion := lendingSuggestion("PHAM VAN D")
	memo := *suggestion.Transaction.Memo + " - cho vay tien"
	suggestion.Transaction.Memo = &memo
	suggestions := []CaptureSuggestion{suggestion}

	ApplyCaptureLending(suggestions, CaptureLendingInput{Transactions: lendingLedger(), Categories: []string{"Cho vay"}})

	if suggestions[0].Transaction.Category != "Cho vay" {
		t.Fatalf("transfer content saying cho vay is lending: %+v", suggestions[0].Transaction)
	}
}

func TestApplyCaptureLendingWithoutALendingCategoryOnlyRenames(t *testing.T) {
	suggestions := []CaptureSuggestion{lendingSuggestion("LE THU NGHIEM")}

	ApplyCaptureLending(suggestions, CaptureLendingInput{Transactions: lendingLedger(), Notes: []string{"lend"}})

	if suggestions[0].Transaction.Payee != "Lê Thử Nghiệm" || suggestions[0].Transaction.Category != "" {
		t.Fatalf("expected a rename only: %+v", suggestions[0].Transaction)
	}
}
