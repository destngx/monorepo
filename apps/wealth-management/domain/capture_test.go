package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func captureEvent(material string, at string, amount float64, direction CaptureDirection) CaptureEvent {
	event := CaptureEvent{MaterialID: material, OccurredAt: at, Amount: amount, Direction: direction, Currency: "VND", Confidence: 0.9}
	if !event.normalize() {
		panic("bad test event " + at)
	}
	return event
}

func TestValidateCaptureMaterials(t *testing.T) {
	limits := CaptureLimits{MaxItems: 2, MaxItemBytes: 10, MaxBatchBytes: 15}
	ok := []CaptureMaterial{{ID: "a", Kind: CaptureText, Text: "GD -50k"}, {ID: "b", Kind: CaptureImage, MediaType: "image/png", Data: []byte("png")}}
	if err := ValidateCaptureMaterials(ok, limits); err != nil {
		t.Fatalf("valid batch rejected: %v", err)
	}
	for name, materials := range map[string][]CaptureMaterial{
		"empty":         nil,
		"too many":      {ok[0], ok[1], {ID: "c", Kind: CaptureText, Text: "x"}},
		"duplicate id":  {ok[0], {ID: "a", Kind: CaptureText, Text: "x"}},
		"heic":          {{ID: "a", Kind: CaptureImage, MediaType: "image/heic", Data: []byte("x")}},
		"blank text":    {{ID: "a", Kind: CaptureText, Text: "  "}},
		"item too big":  {{ID: "a", Kind: CaptureText, Text: "01234567890"}},
		"batch too big": {{ID: "a", Kind: CaptureText, Text: "0123456789"}, {ID: "b", Kind: CaptureText, Text: "012345"}},
		"unknown kind":  {{ID: "a", Kind: "pdf", Text: "x"}},
	} {
		if err := ValidateCaptureMaterials(materials, limits); !errors.Is(err, ErrValidation) {
			t.Errorf("%s: expected validation error, got %v", name, err)
		}
	}
}

func TestParseCaptureExtractionNormalisesEvents(t *testing.T) {
	reply := "```json\n" + `{"events":[
		{"occurredAt":"2026-10-03 14:22","amount":-120000,"currency":"đ","direction":"","counterparty":" Highlands ","confidence":0},
		{"occurredAt":"03/10/2026","amount":1250000,"currency":"vnd","direction":"credit","balanceAfter":3456789,"fee":0,"confidence":1.4},
		{"occurredAt":"yesterday","amount":5,"direction":"out"},
		{"occurredAt":"2026-10-03","amount":0,"direction":"out"},
		{"occurredAt":"2026-10-03","amount":10,"direction":"sideways"}
	],"unreadable":false,"reason":""}` + "\n```"
	extraction, err := ParseCaptureExtraction(reply, "m1")
	if err != nil {
		t.Fatal(err)
	}
	if extraction.Dropped != 3 || len(extraction.Events) != 2 {
		t.Fatalf("expected 2 events and 3 dropped, got %+v", extraction)
	}
	out, in := extraction.Events[0], extraction.Events[1]
	if out.Direction != CaptureOut || out.Amount != 120000 || out.Currency != "VND" || out.Counterparty != "Highlands" || !out.HasTime || out.At.Hour() != 14 || out.Confidence != 0.5 || out.MaterialID != "m1" {
		t.Fatalf("out event: %+v", out)
	}
	if in.Direction != CaptureIn || in.HasTime || in.Date() != time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC) || in.Fee != nil || in.BalanceAfter == nil || in.Confidence != 1 {
		t.Fatalf("in event: %+v", in)
	}
}

func TestParseCaptureExtractionReportsUnreadable(t *testing.T) {
	extraction, err := ParseCaptureExtraction(`{"events":[{"occurredAt":"?","amount":1,"direction":"out"}]}`, "m1")
	if err != nil || !extraction.Unreadable || extraction.Reason == "" {
		t.Fatalf("all-dropped material must be unreadable: %+v %v", extraction, err)
	}
	if _, err := ParseCaptureExtraction("Here you go: {}", "m1"); !errors.Is(err, ErrAIOutput) {
		t.Fatalf("prose must be rejected, got %v", err)
	}
}

func TestFoldText(t *testing.T) {
	for input, want := range map[string]string{"Ví MoMo": "vi momo", "Đức  Anh!": "duc anh", "Sacombank *1234": "sacombank 1234", "TIỀN": "tien"} {
		if got := FoldText(input); got != want {
			t.Errorf("FoldText(%q) = %q, want %q", input, got, want)
		}
	}
}

func note(text string) *string { return &text }

var captureAccounts = []Account{
	{Name: "Credit Card Sacombank", Type: AccountTypeNegativeActive, Currency: CurrencyVND},
	{Name: "Sacombank Saving", Type: AccountTypeBank, Currency: CurrencyVND},
	{Name: "Ví MoMo", Type: AccountTypeActiveUse, Currency: CurrencyVND},
	{Name: "Techcombank", Type: AccountTypeBank, Currency: CurrencyVND, Note: note("acc 1903 4567 8899")},
	{Name: "Vietcombank", Type: AccountTypeBank, Currency: CurrencyVND},
	{Name: "Binance", Type: AccountTypeCrypto, Currency: CurrencyUSD},
	{Name: "Old MoMo", Type: AccountTypeDeprecated},
}

func TestMatchCaptureAccount(t *testing.T) {
	rules := []CaptureRule{
		{Kind: CaptureRuleAlias, Pattern: "Sacombank 1234", Value: "Credit Card Sacombank"},
		{Kind: CaptureRuleAlias, Pattern: "VCB", Value: "Vietcombank"},
		{Kind: CaptureRuleAlias, Pattern: "Gone", Value: "Closed Account"},
	}
	for _, tc := range []struct {
		hint, app  string
		want       string
		source     string
		candidates int
	}{
		{"Sacombank *1234", "", "Credit Card Sacombank", MatchSourceAlias, 0},
		{"", "VCB Digibank", "Vietcombank", MatchSourceAlias, 0},
		{"TK ****8899", "", "Techcombank", MatchSourceDigits, 0},
		{"vi momo", "MoMo", "Ví MoMo", MatchSourceName, 0},
		{"Thẻ tín dụng Sacombank", "", "Credit Card Sacombank", MatchSourceName, 0},
		{"Sacombank", "", "Sacombank Saving", MatchSourceName, 0},
		{"Gone", "", "", "", 0},
		{"Shopee Pay", "", "", "", 0},
	} {
		got := MatchCaptureAccount(tc.hint, tc.app, captureAccounts, rules)
		if got.Name != tc.want || got.Source != tc.source || len(got.Candidates) != tc.candidates {
			t.Errorf("hint %q app %q: got %+v, want %q via %q", tc.hint, tc.app, got, tc.want, tc.source)
		}
	}
	accounts := append([]Account{}, captureAccounts...)
	accounts = append(accounts, Account{Name: "MoMo Savings", Type: AccountTypeActiveUse})
	if got := MatchCaptureAccount("MoMo", "", accounts, nil); got.Name != "" || len(got.Candidates) != 2 {
		t.Errorf("ambiguous hint must ask the user, got %+v", got)
	}
}

func TestConsolidateCaptureItemsMergesAcrossMaterialsOnly(t *testing.T) {
	sms := captureEvent("sms", "2026-10-03 14:22", 500000, CaptureOut)
	sms.Reference = "FT123"
	sms.Confidence = 0.6
	screenshot := captureEvent("shot", "2026-10-03 14:25", 500000, CaptureOut)
	screenshot.Counterparty = "Nguyen Van A"
	balance := 1000.0
	screenshot.BalanceAfter = &balance
	sameShotTwin := captureEvent("shot", "2026-10-03 14:25", 500000, CaptureOut)
	later := captureEvent("sms", "2026-10-03 16:00", 500000, CaptureOut)
	otherRef := captureEvent("other", "2026-10-03 14:23", 500000, CaptureOut)
	otherRef.Reference = "FT999"
	account := AccountMatch{Name: "Vietcombank", Source: MatchSourceAlias}

	items := ConsolidateCaptureItems([]CaptureItem{
		NewCaptureItem(sms, account), NewCaptureItem(screenshot, AccountMatch{}), NewCaptureItem(sameShotTwin, account),
		NewCaptureItem(later, account), NewCaptureItem(otherRef, account),
	})
	// sms+screenshot merge; the screenshot's twin cannot join them (same
	// material) but merges with "other"; the 16:00 event is too late.
	if len(items) != 3 || len(items[1].Evidence) != 2 || len(items[2].Evidence) != 1 {
		t.Fatalf("unexpected consolidation: %+v", items)
	}
	merged := items[0]
	if len(merged.Evidence) != 2 || merged.Event.Counterparty != "Nguyen Van A" || merged.Event.Reference != "FT123" || merged.Event.BalanceAfter == nil || merged.Account.Name != "Vietcombank" {
		t.Fatalf("merge should keep the confident event and fill blanks: %+v", merged)
	}
}

func TestPairCaptureTransfers(t *testing.T) {
	out := NewCaptureItem(captureEvent("a", "2026-10-03 09:00", 2000000, CaptureOut), AccountMatch{Name: "Vietcombank"})
	in := NewCaptureItem(captureEvent("b", "2026-10-03 09:01", 2000000, CaptureIn), AccountMatch{Name: "Ví MoMo"})
	sameAccount := NewCaptureItem(captureEvent("c", "2026-10-03 09:02", 2000000, CaptureIn), AccountMatch{Name: "Vietcombank"})
	tooLate := NewCaptureItem(captureEvent("d", "2026-10-03 12:00", 2000000, CaptureIn), AccountMatch{Name: "Techcombank"})
	unmatched := NewCaptureItem(captureEvent("e", "2026-10-03 09:00", 2000000, CaptureIn), AccountMatch{})
	pairs := PairCaptureTransfers([]CaptureItem{out, sameAccount, tooLate, unmatched, in})
	if len(pairs) != 1 || pairs[0] != [2]int{0, 4} {
		t.Fatalf("pairs: %v", pairs)
	}
}

func amountPointer(value float64) *float64 { return &value }

func TestFindCaptureDuplicates(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2026, 10, d, 0, 0, 0, 0, time.UTC) }
	ledger := []Transaction{
		{ID: "r1", AccountName: "Vietcombank", Date: day(2), Payment: amountPointer(50000)},
		{ID: "r2", AccountName: "Vietcombank", Date: day(3), Payment: amountPointer(50000), ReferenceNumber: note("FT1")},
		{ID: "r3", AccountName: "Vietcombank", Date: day(9), Payment: amountPointer(70000)},
		{ID: "r4", AccountName: "Vietcombank", Date: day(3), Deposit: amountPointer(90000), ReferenceNumber: note("FT7")},
	}
	candidates := []CaptureCandidate{
		{Account: "Vietcombank", Direction: CaptureOut, Amount: 50000, Date: day(3), Reference: "FT1"},
		{Account: "Vietcombank", Direction: CaptureOut, Amount: 50000, Date: day(3)},
		{Account: "Vietcombank", Direction: CaptureOut, Amount: 50000, Date: day(3)},
		{Account: "Vietcombank", Direction: CaptureOut, Amount: 70000, Date: day(5)},
		{Account: "Vietcombank", Direction: CaptureIn, Amount: 90000, Date: day(3), Reference: "FT8"},
		{Account: "Techcombank", Direction: CaptureOut, Amount: 50000, Date: day(3)},
	}
	got := FindCaptureDuplicates(candidates, ledger)
	if len(got) != 2 || got[0].ID != "r2" || got[1].ID != "r1" {
		t.Fatalf("expected the reference match then the remaining row, each used once: %+v", got)
	}
}

func TestBuildCaptureSuggestions(t *testing.T) {
	materials := []CaptureMaterial{{ID: "a", Kind: CaptureImage}, {ID: "b", Kind: CaptureText}}
	out := captureEvent("a", "2026-10-03 09:00", 2000000, CaptureOut)
	out.SourceApp = "Vietcombank"
	out.Description = "chuyen tien"
	out.Reference = "FT1"
	fee := 1100.0
	out.Fee = &fee
	in := captureEvent("b", "2026-10-03 09:01", 2000000, CaptureIn)
	coffee := captureEvent("b", "2026-10-03", 45, CaptureOut)
	coffee.Counterparty = "Highlands"
	coffee.Currency = "VND"
	coffee.Confidence = 0.4

	suggestions := BuildCaptureSuggestions(CaptureBuildInput{
		Items: []CaptureItem{
			NewCaptureItem(out, AccountMatch{Name: "Vietcombank", Source: MatchSourceAlias}),
			NewCaptureItem(in, AccountMatch{Name: "Ví MoMo", Source: MatchSourceAlias}),
			NewCaptureItem(coffee, AccountMatch{Name: "Binance", Source: MatchSourceName}),
		},
		Materials:        materials,
		Accounts:         captureAccounts,
		TransferCategory: "Transfer",
	})
	if len(suggestions) != 4 {
		t.Fatalf("expected transfer pair, fee row and coffee: %+v", suggestions)
	}
	transferOut, feeRow, transferIn, coffeeRow := suggestions[0], suggestions[1], suggestions[2], suggestions[3]
	if transferOut.TransferGroup != "t1" || transferIn.TransferGroup != "t1" || transferOut.Transaction.Category != "Transfer" || transferOut.CategorySource != CategorySourceTransfer {
		t.Fatalf("transfer pair: %+v %+v", transferOut, transferIn)
	}
	if transferOut.Transaction.Payee != "Ví MoMo" || transferIn.Transaction.Payee != "Vietcombank" || *transferOut.Transaction.Payment != 2000000 || *transferIn.Transaction.Deposit != 2000000 {
		t.Fatalf("transfer rows: %+v %+v", transferOut.Transaction, transferIn.Transaction)
	}
	if *transferOut.Transaction.Memo != "[capture] Vietcombank screenshot - chuyen tien" || *transferOut.Transaction.ReferenceNumber != "FT1" || transferOut.Transaction.Date != "03/10/2026" || transferOut.OccurredAt != "2026-10-03 09:00" {
		t.Fatalf("transfer out details: %+v", transferOut)
	}
	if feeRow.Transaction.Payee != "Vietcombank fee" || *feeRow.Transaction.Payment != 1100 || feeRow.TransferGroup != "" || feeRow.Transaction.Category != "" {
		t.Fatalf("fee row: %+v", feeRow)
	}
	if coffeeRow.Transaction.Payee != "Highlands" || strings.Join(coffeeRow.LowConfidence, ",") != "amount,date,account" || len(coffeeRow.Notes) != 2 {
		t.Fatalf("coffee row should flag confidence, name match and currency: %+v", coffeeRow)
	}
	for index, suggestion := range suggestions {
		if suggestion.ID == "" || !suggestion.Selected || (index > 0 && suggestion.ID == suggestions[index-1].ID) {
			t.Fatalf("ids and selection: %+v", suggestions)
		}
	}

	ApplyCaptureDuplicates(suggestions, []Transaction{{ID: "r9", AccountName: "Ví MoMo", Payee: "VCB", Date: time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC), Deposit: amountPointer(2000000)}})
	if suggestions[2].Selected || suggestions[2].Duplicate == nil || suggestions[2].Duplicate.TransactionID != "r9" || suggestions[2].Duplicate.Date != "04/10/2026" || !suggestions[0].Selected {
		t.Fatalf("duplicate marking: %+v", suggestions[2])
	}
}

func TestCaptureBalancesUsesLatestPerAccountAndSkipsCards(t *testing.T) {
	first := captureEvent("a", "2026-10-03 09:00", 10, CaptureOut)
	first.BalanceAfter = amountPointer(100)
	second := captureEvent("b", "2026-10-03 10:00", 10, CaptureOut)
	second.BalanceAfter = amountPointer(90)
	card := captureEvent("c", "2026-10-03 10:00", 10, CaptureOut)
	card.BalanceAfter = amountPointer(5000)
	accounts := []Account{{Name: "Vietcombank", Balance: 120, Type: AccountTypeBank}, {Name: "Credit Card Sacombank", Type: AccountTypeNegativeActive}}
	balances := CaptureBalances([]CaptureItem{
		NewCaptureItem(second, AccountMatch{Name: "Vietcombank"}),
		NewCaptureItem(first, AccountMatch{Name: "Vietcombank"}),
		NewCaptureItem(card, AccountMatch{Name: "Credit Card Sacombank"}),
	}, accounts)
	if len(balances) != 1 || balances[0].BalanceAfter != 90 || balances[0].SheetBalance != 120 || balances[0].AsOf != "2026-10-03 10:00" || balances[0].MaterialID != "b" {
		t.Fatalf("balances: %+v", balances)
	}
}

func TestTransferCategory(t *testing.T) {
	categories := []Category{{Name: "Transfer fee", Type: CategoryTypeExpense}, {Name: "Chuyển khoản", Type: CategoryTypeNonBudget}}
	if name, ok := TransferCategory(categories); !ok || name != "Chuyển khoản" {
		t.Fatalf("got %q %v", name, ok)
	}
	if _, ok := TransferCategory(categories[:1]); ok {
		t.Fatal("expense categories are never transfers")
	}
}

func TestCaptureRuleValidateAndKey(t *testing.T) {
	if err := (CaptureRule{Kind: CaptureRulePayee, Pattern: " Highlands ", Value: "Food"}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (CaptureRule{Kind: "x", Pattern: "!!", Value: ""}).Validate(); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if (CaptureRule{Kind: CaptureRuleAlias, Pattern: "Ví MoMo"}).Key() != (CaptureRule{Kind: CaptureRuleAlias, Pattern: "vi  momo"}).Key() {
		t.Fatal("keys must fold the pattern")
	}
}
