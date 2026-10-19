package ai

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/internal/fakes"
	"apps/wealth-management/port"
	"apps/wealth-management/service"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

func captureDatabase() *fakes.Database {
	db := fixtureDatabase()
	db.Accounts = []domain.Account{
		{Name: "Vietcombank", Type: domain.AccountTypeBank, Currency: domain.CurrencyVND, Balance: 7_045_000},
		{Name: "Ví MoMo", Type: domain.AccountTypeActiveUse, Currency: domain.CurrencyVND},
		{Name: "Credit Card Acme", Type: domain.AccountTypeNegativeActive, Currency: domain.CurrencyVND},
	}
	db.Categories = append(db.Categories, domain.Category{Name: "Shopping", Type: domain.CategoryTypeExpense}, domain.Category{Name: "Transfer", Type: domain.CategoryTypeNonBudget})
	db.Transactions = append(db.Transactions, domain.Transaction{
		ID: "r7", AccountName: "Vietcombank", Payee: "Highlands Coffee", Category: "Coffee",
		Date: time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC), Payment: money(45_000),
	})
	db.CaptureRules = []domain.CaptureRule{{Kind: domain.CaptureRuleAlias, Pattern: "VCB", Value: "Vietcombank"}}
	db.Cards = captureCards
	db.AIContent.Prompts[keyCaptureExtract] = "Today is {{today}}. Accounts:\n{{accounts}}\nAliases:\n{{aliases}}"
	db.AIContent.Prompts[keyCaptureCategorize] = "Categorise into: {{categories}}"
	return db
}

// captureCards bills two open cards to the shared account: their keywords
// are the tags offered on its rows; the closed card's is not.
var captureCardClosed = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

var captureCards = domain.CardConfig{
	Programs: []domain.Program{
		{Name: "Gold", Bank: "Acme", Versions: []domain.ProgramVersion{{Period: domain.CapPeriodCycle, Rules: []domain.CashbackRule{}}}},
		{Name: "Silver", Bank: "Acme", Versions: []domain.ProgramVersion{{Period: domain.CapPeriodMonth, Rules: []domain.CashbackRule{}}}},
	},
	Banks: []domain.Bank{{Name: "Acme", SharedAccount: "Credit Card Acme"}},
	Cards: []domain.Card{
		{Name: "Gold Card", Program: "Gold", Keyword: "gold", StatementDay: 5, DueDay: 30, Last4: "0123"},
		{Name: "Silver Card", Program: "Silver", Keyword: "silver", StatementDay: 5, DueDay: 30, Default: true},
		{Name: "Old Card", Program: "Silver", Keyword: "old", StatementDay: 5, DueDay: 30, ClosedFrom: &captureCardClosed, Last4: "7777"},
	},
	Merchants: []domain.MerchantMCC{},
}

func newCaptureService(t *testing.T, db *fakes.Database, client *scriptedClient) *Service {
	t.Helper()
	now := func() time.Time { return testToday }
	analytics := service.NewAnalyticsService(db, now)
	return New(client, db, analytics, nil, testConfig(), logger.NewTestLogger(t), Options{Now: now})
}

const (
	smsReply = `{"events":[
		{"occurredAt":"2026-10-03 09:00","amount":2000000,"currency":"VND","direction":"out","description":"chuyen tien momo","sourceApp":"Vietcombank","accountHint":"VCB *1234","balanceAfter":5000000,"confidence":0.95,"quote":"-2,000,000VND"},
		{"occurredAt":"2026-10-03 12:30","amount":45000,"currency":"VND","direction":"out","counterparty":"Highlands Coffee","sourceApp":"Vietcombank","accountHint":"VCB *1234","confidence":0.9,"quote":"-45,000VND"}
	]}`
	momoReply = `{"events":[{"occurredAt":"2026-10-03 09:01","amount":2000000,"currency":"VND","direction":"in","counterparty":"Vietcombank","sourceApp":"MoMo","accountHint":"Ví MoMo","confidence":0.9,"quote":"+2.000.000đ"}]}`
	cardReply = `{"events":[{"occurredAt":"2026-10-04","amount":300000,"currency":"VND","direction":"out","counterparty":"Shopee","sourceApp":"Acme","accountHint":"Thẻ tín dụng Acme","confidence":0.8,"quote":"Shopee 300,000"}]}`
)

func captureRoute(t *testing.T, categorize string) func(domain.AIRequest) reply {
	return func(request domain.AIRequest) reply {
		user := request.Messages[len(request.Messages)-1].Content
		switch {
		case strings.Contains(user, "Items:"):
			return reply{content: categorize}
		case strings.HasPrefix(user, "Material sms "):
			return reply{content: smsReply}
		case strings.HasPrefix(user, "Material momo "):
			return reply{content: momoReply}
		case strings.HasPrefix(user, "Material card "):
			return reply{content: cardReply}
		case strings.HasPrefix(user, "Material blur "):
			return reply{content: `{"events":[],"unreadable":true,"reason":"Image too blurry."}`}
		case strings.HasPrefix(user, "Material junk "):
			return reply{content: "I think it is a coffee."}
		}
		t.Errorf("unexpected request: %q", user)
		return reply{err: errors.New("unexpected")}
	}
}

func captureMaterials() []domain.CaptureMaterial {
	png := []byte("\x89PNG")
	return []domain.CaptureMaterial{
		{ID: "sms", Kind: domain.CaptureText, Text: "VCB: TK 0011001234 cua Nguyen Van A -2,000,000VND luc 03/10 09:00. SD 5,000,000VND"},
		{ID: "momo", Kind: domain.CaptureImage, MediaType: "image/png", Data: png, Name: "momo.png"},
		{ID: "card", Kind: domain.CaptureImage, MediaType: "image/png", Data: png},
		{ID: "blur", Kind: domain.CaptureImage, MediaType: "image/jpeg", Data: png},
		{ID: "junk", Kind: domain.CaptureText, Text: "coffee?"},
	}
}

func TestCapturePipeline(t *testing.T) {
	client := &scriptedClient{}
	client.route = captureRoute(t, `{"items":[{"id":"s4","category":"shopping","tag":"gold"},{"id":"s9","category":"Food"}]}`)
	svc := newCaptureService(t, captureDatabase(), client)

	var mu sync.Mutex
	var progress []port.CaptureProgress
	result, err := svc.Capture(context.Background(), port.CaptureRequest{Materials: captureMaterials()}, func(event port.CaptureProgress) {
		mu.Lock()
		defer mu.Unlock()
		progress = append(progress, event)
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Suggestions) != 4 {
		t.Fatalf("expected transfer out, coffee, transfer in and card purchase: %+v", result.Suggestions)
	}
	out, coffee, in, card := result.Suggestions[0], result.Suggestions[1], result.Suggestions[2], result.Suggestions[3]
	if out.TransferGroup == "" || out.TransferGroup != in.TransferGroup || out.Transaction.Category != "Transfer" || in.Transaction.AccountName != "Ví MoMo" || out.Transaction.AccountName != "Vietcombank" || out.AccountSource != domain.MatchSourceAlias {
		t.Fatalf("transfer pair: %+v / %+v", out, in)
	}
	if coffee.Transaction.Category != "Coffee" || coffee.CategorySource != domain.CategorySourceHistory || coffee.Selected || coffee.Duplicate == nil || coffee.Duplicate.TransactionID != "r7" {
		t.Fatalf("coffee should come from history and be flagged as recorded: %+v", coffee)
	}
	if card.Transaction.AccountName != "Credit Card Acme" || card.Transaction.Category != "Shopping" || card.CategorySource != domain.CategorySourceAI || len(card.Transaction.Tags) != 1 || card.Transaction.Tags[0] != "gold" || len(card.TagOptions) != 2 {
		t.Fatalf("card purchase: %+v", card)
	}
	if len(result.Unreadable) != 2 || result.Unreadable[0].MaterialID != "blur" || result.Unreadable[0].Reason != "Image too blurry." || result.Unreadable[1].MaterialID != "junk" {
		t.Fatalf("unreadable: %+v", result.Unreadable)
	}
	if len(result.Balances) != 1 || result.Balances[0].Account != "Vietcombank" || result.Balances[0].BalanceAfter != 5_000_000 || result.Balances[0].SheetBalance != 7_045_000 {
		t.Fatalf("balances: %+v", result.Balances)
	}

	extracts := 0
	for _, request := range client.requests {
		user := request.Messages[len(request.Messages)-1]
		if strings.HasPrefix(user.Content, "Material") {
			extracts++
			if !strings.Contains(request.Messages[0].Content, "Today is 2026-10-10") || !strings.Contains(request.Messages[0].Content, `"VCB" is Vietcombank`) {
				t.Fatalf("extract system prompt not rendered: %q", request.Messages[0].Content)
			}
		}
		if strings.HasPrefix(user.Content, "Material momo ") && (len(user.Images) != 1 || user.Images[0].MediaType != "image/png") {
			t.Fatalf("image not attached: %+v", user)
		}
		if strings.HasPrefix(user.Content, "Material sms ") && (strings.Contains(user.Content, "Nguyen Van A") || strings.Contains(user.Content, "0011001234") || !strings.Contains(user.Content, "••••1234")) {
			t.Fatalf("text material must be masked: %q", user.Content)
		}
	}
	// junk is retried on the fallback model; categorise runs once.
	if extracts != 6 || len(client.requests) != 7 {
		t.Fatalf("expected 6 extraction calls and 1 categorise call, got %d / %d", extracts, len(client.requests))
	}
	extractEvents := 0
	for _, event := range progress {
		if event.Stage == "extract" && event.Status != "running" {
			extractEvents++
		}
	}
	if extractEvents != 5 {
		t.Fatalf("expected one finished extract event per material: %+v", progress)
	}
}

func TestCaptureFilesACardPaymentByItsLastFourDigits(t *testing.T) {
	client := &scriptedClient{}
	client.route = func(request domain.AIRequest) reply {
		user := request.Messages[len(request.Messages)-1].Content
		if strings.Contains(user, "Items:") {
			return reply{content: `{"items":[{"id":"s1","category":"Shopping","tag":"silver"},{"id":"s2","category":"Shopping"}]}`}
		}
		return reply{content: `{"events":[
			{"occurredAt":"2026-10-04","amount":300000,"currency":"VND","direction":"out","counterparty":"Shopee","sourceApp":"Acme","accountHint":"Visa *0123","confidence":0.8,"quote":"Shopee 300,000"},
			{"occurredAt":"2026-10-04","amount":90000,"currency":"VND","direction":"out","counterparty":"Grab","sourceApp":"Acme","accountHint":"Visa *7777","confidence":0.8,"quote":"Grab 90,000"}
		]}`}
	}
	result, err := newCaptureService(t, captureDatabase(), client).Capture(context.Background(), port.CaptureRequest{Materials: captureMaterials()[2:3]}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Suggestions) != 2 {
		t.Fatalf("expected two card payments: %+v", result.Suggestions)
	}
	gold, closed := result.Suggestions[0], result.Suggestions[1]
	if gold.Transaction.AccountName != "Credit Card Acme" || gold.AccountSource != domain.MatchSourceDigits || len(gold.Transaction.Tags) != 1 || gold.Transaction.Tags[0] != "gold" {
		t.Fatalf("the card's digits should file the payment to it, ahead of the model: %+v", gold)
	}
	// A closed card's digits match nothing: the row falls back to the name match.
	if closed.AccountSource == domain.MatchSourceDigits || (len(closed.Transaction.Tags) > 0 && closed.Transaction.Tags[0] == "old") {
		t.Fatalf("a closed card takes no payments: %+v", closed)
	}
	system := client.requests[0].Messages[0].Content
	if !strings.Contains(system, "Gold Card *0123") || strings.Contains(system, "7777") {
		t.Fatalf("the extract prompt should list open cards with their last 4 digits: %q", system)
	}
}

func TestCaptureLeavesRowsForReviewWhenCategorisingFails(t *testing.T) {
	client := &scriptedClient{}
	client.route = captureRoute(t, "not json")
	db := captureDatabase()
	svc := newCaptureService(t, db, client)
	result, err := svc.Capture(context.Background(), port.CaptureRequest{Materials: captureMaterials()[2:3]}, nil)
	if err != nil {
		t.Fatal(err)
	}
	card := result.Suggestions[0]
	if card.Transaction.Category != "Uncategorized" || card.CategorySource != domain.CategorySourceDefault || !strings.Contains(strings.Join(card.LowConfidence, ","), "category") || len(card.Notes) == 0 {
		t.Fatalf("expected an uncategorised row flagged for review: %+v", card)
	}
}

func TestCaptureUsesPayeeRulesBeforeHistory(t *testing.T) {
	client := &scriptedClient{}
	client.route = captureRoute(t, `{"items":[]}`)
	db := captureDatabase()
	db.CaptureRules = append(db.CaptureRules, domain.CaptureRule{Kind: domain.CaptureRulePayee, Pattern: "highlands coffee", Value: "Food"})
	result, err := newCaptureService(t, db, client).Capture(context.Background(), port.CaptureRequest{Materials: captureMaterials()[:1]}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if coffee := result.Suggestions[1]; coffee.Transaction.Category != "Food" || coffee.CategorySource != domain.CategorySourceRule {
		t.Fatalf("payee rule should win: %+v", coffee)
	}
}

func TestCaptureFailsWhenEveryExtractionFails(t *testing.T) {
	client := &scriptedClient{route: func(domain.AIRequest) reply {
		return reply{err: domain.ErrAIUnavailable}
	}}
	_, err := newCaptureService(t, captureDatabase(), client).Capture(context.Background(), port.CaptureRequest{Materials: captureMaterials()[:2]}, nil)
	if !errors.Is(err, domain.ErrAIUnavailable) {
		t.Fatalf("expected the provider error, got %v", err)
	}
}

func TestCaptureValidatesAndNeedsTheExtractPrompt(t *testing.T) {
	svc := newCaptureService(t, captureDatabase(), &scriptedClient{})
	if _, err := svc.Capture(context.Background(), port.CaptureRequest{}, nil); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
	db := captureDatabase()
	delete(db.AIContent.Prompts, keyCaptureExtract)
	_, err := newCaptureService(t, db, &scriptedClient{}).Capture(context.Background(), port.CaptureRequest{Materials: captureMaterials()[:1]}, nil)
	if !errors.Is(err, domain.ErrPromptMissing) {
		t.Fatalf("expected missing prompt, got %v", err)
	}
}

func TestCaptureTagsTheOnlyOpenCardOfAnAccount(t *testing.T) {
	db := captureDatabase()
	db.Cards.Cards = []domain.Card{db.Cards.Cards[0], db.Cards.Cards[2]}
	db.Cards.Cards[0].Last4 = ""
	client := &scriptedClient{}
	client.route = captureRoute(t, `{"items":[{"id":"s1","category":"shopping","tag":""}]}`)

	result, err := newCaptureService(t, db, client).Capture(context.Background(), port.CaptureRequest{Materials: captureMaterials()[2:3]}, func(port.CaptureProgress) {})
	if err != nil {
		t.Fatal(err)
	}

	card := result.Suggestions[0]
	if card.Transaction.AccountName != "Credit Card Acme" || len(card.Transaction.Tags) != 1 || card.Transaction.Tags[0] != "gold" {
		t.Fatalf("a card account with one open card should take its tag: %+v", card.Transaction)
	}
}

func TestCaptureRecordsATypedNoteDatedTodayInTheCategoryItAsksFor(t *testing.T) {
	// The model leaves out what a typed note does not say: no date, no sign.
	noteReply := `{"events":[{"occurredAt":"","amount":400000,"currency":"VND","direction":"","description":"tien gui xe","confidence":0.8,"quote":"400k tien gui xe","category":"%s"}]}`
	for name, test := range map[string]struct {
		hint, categorize, wantSource string
	}{
		"hint names a category": {hint: "transport", categorize: `{"items":[]}`, wantSource: domain.CategorySourceManual},
		"hint means a category": {hint: "transit", categorize: `{"items":[{"id":"s1","category":"Transport"}]}`, wantSource: domain.CategorySourceAI},
	} {
		t.Run(name, func(t *testing.T) {
			var categorizeRequest string
			client := &scriptedClient{route: func(request domain.AIRequest) reply {
				user := request.Messages[len(request.Messages)-1].Content
				if strings.Contains(user, "Items:") {
					categorizeRequest = user
					return reply{content: test.categorize}
				}
				if !strings.Contains(user, "Today is 2026-10-10.") {
					t.Errorf("extraction must state today's date: %q", user)
				}
				return reply{content: fmt.Sprintf(noteReply, test.hint)}
			}}
			materials := []domain.CaptureMaterial{{ID: "m1", Kind: domain.CaptureText, Text: "400k tien gui xe count as " + test.hint}}
			result, err := newCaptureService(t, captureDatabase(), client).Capture(context.Background(), port.CaptureRequest{Materials: materials}, nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Unreadable) != 0 || len(result.Suggestions) != 1 {
				t.Fatalf("expected one suggestion, got %+v", result)
			}
			row := result.Suggestions[0]
			tx := row.Transaction
			if tx.Date != "10/10/2026" || tx.Payment == nil || *tx.Payment != 400_000 || tx.Deposit != nil || tx.Payee != "tien gui xe" {
				t.Fatalf("expected a 400,000 payment to tien gui xe today: %+v", tx)
			}
			if tx.Category != "Transport" || row.CategorySource != test.wantSource {
				t.Fatalf("expected Transport from %s, got %q from %q", test.wantSource, tx.Category, row.CategorySource)
			}
			if !strings.Contains(strings.Join(row.LowConfidence, ","), "date") || !strings.Contains(strings.Join(row.Notes, " "), "dated today") {
				t.Fatalf("an assumed date must be flagged: %+v", row)
			}
			if test.wantSource == domain.CategorySourceAI && !strings.Contains(categorizeRequest, `"hint":"transit"`) {
				t.Fatalf("the categoriser must get the hint: %q", categorizeRequest)
			}
		})
	}
}

// A transfer screenshot names the person as the bank prints it; the note
// "lend" reads as no transaction but says what the money was for (WM-CAP-22).
func TestCaptureMatchesALendingPayeeAndReadsTheLendNote(t *testing.T) {
	const transferReply = `{"events":[{"occurredAt":"2026-10-09 20:15","amount":2000000,"currency":"VND","direction":"out","counterparty":"LE THU NGHIEM","description":"chuyen tien","sourceApp":"Vietcombank","accountHint":"VCB *1234","confidence":0.95,"quote":"-2,000,000 VND"}]}`
	for name, test := range map[string]struct {
		note       string
		wantSource string
	}{
		"with a lend note":    {note: "lend", wantSource: domain.CategorySourceManual},
		"without a note":      {wantSource: domain.CategorySourceHistory},
		"with any other note": {note: "dinner money", wantSource: domain.CategorySourceHistory},
	} {
		t.Run(name, func(t *testing.T) {
			db := captureDatabase()
			db.Categories = append(db.Categories, domain.Category{Name: "Cho vay", Type: domain.CategoryTypeNonBudget, Nature: domain.NatureLending})
			db.Transactions = append(db.Transactions, domain.Transaction{
				ID: "r9", AccountName: "Vietcombank", Payee: "Lê Thử Nghiệm", Category: "Cho vay", CategoryNature: domain.NatureLending,
				Date: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), Payment: money(1_000_000),
			})
			client := &scriptedClient{route: func(request domain.AIRequest) reply {
				user := request.Messages[len(request.Messages)-1].Content
				switch {
				case strings.Contains(user, "Items:"):
					t.Errorf("a lending payee needs no model call: %q", user)
					return reply{content: `{"items":[{"id":"s1","category":"Transfer"}]}`}
				case strings.HasPrefix(user, "Material shot "):
					return reply{content: transferReply}
				}
				return reply{content: `{"events":[],"unreadable":true,"reason":"no amount"}`}
			}}
			materials := []domain.CaptureMaterial{{ID: "shot", Kind: domain.CaptureImage, MediaType: "image/png", Data: []byte("\x89PNG")}}
			if test.note != "" {
				materials = append(materials, domain.CaptureMaterial{ID: "note", Kind: domain.CaptureText, Text: test.note})
			}

			result, err := newCaptureService(t, db, client).Capture(context.Background(), port.CaptureRequest{Materials: materials}, nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Suggestions) != 1 {
				t.Fatalf("expected one suggestion, got %+v", result.Suggestions)
			}
			row := result.Suggestions[0]
			if row.Transaction.Payee != "Lê Thử Nghiệm" || row.Transaction.Category != "Cho vay" || row.CategorySource != test.wantSource {
				t.Fatalf("expected Lê Thử Nghiệm under Cho vay from %s, got %q / %q from %q", test.wantSource, row.Transaction.Payee, row.Transaction.Category, row.CategorySource)
			}
		})
	}
}

func TestCaptureShowsANoteWithoutATransactionToTheCategoriser(t *testing.T) {
	var categorizeRequest string
	client := &scriptedClient{route: func(request domain.AIRequest) reply {
		user := request.Messages[len(request.Messages)-1].Content
		switch {
		case strings.Contains(user, "Items:"):
			categorizeRequest = user
			return reply{content: `{"items":[{"id":"s1","category":"Shopping"}]}`}
		case strings.HasPrefix(user, "Material card "):
			return reply{content: cardReply}
		}
		return reply{content: `{"events":[],"unreadable":true,"reason":"no amount"}`}
	}}
	png := []byte("\x89PNG")
	materials := []domain.CaptureMaterial{
		{ID: "card", Kind: domain.CaptureImage, MediaType: "image/png", Data: png},
		{ID: "note", Kind: domain.CaptureText, Text: "gift for mom"},
	}

	if _, err := newCaptureService(t, captureDatabase(), client).Capture(context.Background(), port.CaptureRequest{Materials: materials}, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(categorizeRequest, "The user's note:\ngift for mom") {
		t.Fatalf("the categoriser must see the note: %q", categorizeRequest)
	}
}

// splitCaptureDatabase gives Food an eating-out split tag and a definition.
func splitCaptureDatabase() *fakes.Database {
	db := captureDatabase()
	for i := range db.Categories {
		if db.Categories[i].Name == "Food" {
			db.Categories[i].Definition = "Food bought for home, and everyday meals out."
			db.Categories[i].SplitTags = []string{"eating-out"}
		}
	}
	return db
}

func TestGivenSplitTagsWhenTheModelFilesACaptureThenItAddsTheSplitTagNextToTheCardTag(t *testing.T) {
	var categorizeRequest string
	client := &scriptedClient{route: func(request domain.AIRequest) reply {
		user := request.Messages[len(request.Messages)-1].Content
		if strings.Contains(user, "Items:") {
			categorizeRequest = user
			return reply{content: `{"items":[{"id":"s1","category":"Food","tag":"silver","splitTag":"Eating-Out"}]}`}
		}
		return reply{content: cardReply}
	}}

	result, err := newCaptureService(t, splitCaptureDatabase(), client).Capture(context.Background(), port.CaptureRequest{Materials: captureMaterials()[2:3]}, nil)

	if err != nil {
		t.Fatal(err)
	}
	row := result.Suggestions[0]
	if row.Transaction.Category != "Food" || strings.Join(row.Transaction.Tags, ",") != "silver,eating-out" {
		t.Fatalf("row = %+v", row.Transaction)
	}
	if !strings.Contains(categorizeRequest, "Category guide:\n- Food: Food bought for home, and everyday meals out. Split tags: eating-out.") {
		t.Fatalf("the categoriser must see the guide: %s", categorizeRequest)
	}
}

func TestGivenAPayeeHistoryInASplitCategoryWhenCapturingThenHistoryDecidesTheSplitTagWithoutTheModel(t *testing.T) {
	db := splitCaptureDatabase()
	db.Transactions = append(db.Transactions,
		domain.Transaction{AccountName: "Credit Card Acme", Date: testToday.AddDate(0, 0, -6), Payee: "Shopee", Category: "Food", Tags: []string{"silver", "eating-out"}, Payment: money(100_000)},
		domain.Transaction{AccountName: "Credit Card Acme", Date: testToday.AddDate(0, 0, -5), Payee: "Shopee", Category: "Food", Tags: []string{"silver", "eating-out"}, Payment: money(100_000)},
	)
	client := &scriptedClient{}
	client.route = captureRoute(t, `{"items":[{"id":"s1","category":"Shopping","tag":"gold","splitTag":""}]}`)

	result, err := newCaptureService(t, db, client).Capture(context.Background(), port.CaptureRequest{Materials: captureMaterials()[2:3]}, nil)

	if err != nil {
		t.Fatal(err)
	}
	row := result.Suggestions[0]
	if row.Transaction.Category != "Food" || row.CategorySource != domain.CategorySourceHistory || strings.Join(row.Transaction.Tags, ",") != "silver,eating-out" {
		t.Fatalf("row = %+v (%s)", row.Transaction, row.CategorySource)
	}
	for _, request := range client.requests {
		if strings.Contains(request.Messages[len(request.Messages)-1].Content, "Items:") {
			t.Fatal("history decided everything; the categoriser must not be called")
		}
	}
}

func TestGivenARuleCategoryWithSplitTagsAndNoHistoryWhenCapturingThenTheModelPicksOnlyTheSplitTag(t *testing.T) {
	db := splitCaptureDatabase()
	db.CaptureRules = append(db.CaptureRules, domain.CaptureRule{Kind: domain.CaptureRulePayee, Pattern: "shopee", Value: "Food"})
	var categorizeRequest string
	client := &scriptedClient{route: func(request domain.AIRequest) reply {
		user := request.Messages[len(request.Messages)-1].Content
		if strings.Contains(user, "Items:") {
			categorizeRequest = user
			return reply{content: `{"items":[{"id":"s1","category":"Shopping","tag":"gold","splitTag":"eating-out"}]}`}
		}
		return reply{content: cardReply}
	}}

	result, err := newCaptureService(t, db, client).Capture(context.Background(), port.CaptureRequest{Materials: captureMaterials()[2:3]}, nil)

	if err != nil {
		t.Fatal(err)
	}
	row := result.Suggestions[0]
	if row.Transaction.Category != "Food" || row.CategorySource != domain.CategorySourceRule || strings.Join(row.Transaction.Tags, ",") != "gold,eating-out" {
		t.Fatalf("row = %+v (%s)", row.Transaction, row.CategorySource)
	}
	if !strings.Contains(categorizeRequest, `"category":"Food"`) {
		t.Fatalf("the categoriser must see the decided category: %s", categorizeRequest)
	}
}

func TestGivenAPayeeHistoryFromBeforeSplitTagsWhenCapturingThenTheModelPicksTheSplitTag(t *testing.T) {
	db := splitCaptureDatabase()
	db.Transactions = append(db.Transactions,
		domain.Transaction{AccountName: "Credit Card Acme", Date: testToday.AddDate(0, 0, -6), Payee: "Shopee", Category: "Food", Tags: []string{"silver"}, Payment: money(100_000)},
		domain.Transaction{AccountName: "Credit Card Acme", Date: testToday.AddDate(0, 0, -5), Payee: "Shopee", Category: "Food", Tags: []string{"silver"}, Payment: money(100_000)},
	)
	client := &scriptedClient{}
	client.route = captureRoute(t, `{"items":[{"id":"s1","category":"Food","tag":"silver","splitTag":"eating-out"}]}`)

	result, err := newCaptureService(t, db, client).Capture(context.Background(), port.CaptureRequest{Materials: captureMaterials()[2:3]}, nil)

	if err != nil {
		t.Fatal(err)
	}
	if row := result.Suggestions[0]; row.Transaction.Category != "Food" || row.CategorySource != domain.CategorySourceHistory || strings.Join(row.Transaction.Tags, ",") != "silver,eating-out" {
		t.Fatalf("row = %+v (%s)", row.Transaction, row.CategorySource)
	}
}
