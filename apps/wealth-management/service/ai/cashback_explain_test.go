package ai

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/internal/fakes"
	"apps/wealth-management/port"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func explainDatabase() *fakes.Database {
	db := captureDatabase()
	cards := captureCards
	cards.Programs = []domain.Program{
		{Name: "Gold", Bank: "Acme", Versions: []domain.ProgramVersion{{Period: domain.CapPeriodCycle, Rules: []domain.CashbackRule{
			{Name: "Shopping", MCCs: []domain.MCCRange{{From: 5311, To: 5311}}, Rate: 10, Cap: 100_000, MinTransaction: 100_000},
			{Name: "Others", Rate: 1, Fallback: true},
		}}}},
		captureCards.Programs[1],
	}
	cards.Merchants = []domain.MerchantMCC{{Pattern: "shopee", MCC: 5311, Confidence: domain.MCCConfirmed}}
	db.Cards = cards
	db.Transactions = append(db.Transactions, domain.Transaction{
		ID: "c1", AccountName: "Credit Card Acme", Payee: "Shopee Nguyen Van A", Category: "Shopping", Tags: []string{"gold"},
		Date: time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC), Payment: money(300_000),
	}, domain.Transaction{
		ID: "c0", AccountName: "Credit Card Acme", Payee: "Grab", Category: "Transport", Tags: []string{"gold"},
		Date: time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC), Payment: money(120_000),
	})
	db.AIContent.Prompts["tasks/cashback-explain"] = "Card {{card}} ({{program}}), {{start}} to {{end}}, open {{open}}: {{periodJson}}"
	db.AIContent.Prompts["actions/cashback-explain"] = "Explain {{card}} from {{start}} to {{end}}."
	return db
}

func TestExplainCashbackWalksThroughTheCurrentPeriod(t *testing.T) {
	client := &scriptedClient{replies: []reply{{content: "  **30,000** from Shopee.  "}}}
	text, err := newCaptureService(t, explainDatabase(), client).ExplainCashback(context.Background(), port.CashbackExplainRequest{Card: "Gold Card"})
	if err != nil || text != "**30,000** from Shopee." {
		t.Fatalf("unexpected explanation %q %v", text, err)
	}
	prompt := client.requests[0].Messages[0].Content + "\n" + client.requests[0].Messages[1].Content
	for _, want := range []string{
		"Card Gold Card (Gold), 2026-10-05 to 2026-11-04, open true",
		"Explain Gold Card from 2026-10-05 to 2026-11-04.",
		`"date":"2026-10-06"`, `"rule":"Shopping"`, `"cashbackBeforeCaps":30000`, `"mcc":5311`,
		"Shopping: 10.00% on MCC 5311, payments from 100000, cap 100000",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("the prompt should carry %q:\n%s", want, prompt)
		}
	}
	if strings.Contains(prompt, "Nguyen Van A") {
		t.Fatalf("payees must be masked:\n%s", prompt)
	}
	if strings.Contains(prompt, "Grab") {
		t.Fatalf("only the period's payments belong in the prompt:\n%s", prompt)
	}
}

func TestExplainCashbackCoversAPastPeriod(t *testing.T) {
	client := &scriptedClient{replies: []reply{{content: "Grab earned 1%."}}}
	if _, err := newCaptureService(t, explainDatabase(), client).ExplainCashback(context.Background(), port.CashbackExplainRequest{Card: "Gold Card", Month: "2026-09"}); err != nil {
		t.Fatal(err)
	}
	if prompt := client.requests[0].Messages[0].Content; !strings.Contains(prompt, "2026-09-05 to 2026-10-04, open false") || !strings.Contains(prompt, `"rule":"Others"`) {
		t.Fatalf("the past period should be explained:\n%s", prompt)
	}
}

func TestExplainCashbackRejectsBadRequests(t *testing.T) {
	svc := newCaptureService(t, explainDatabase(), &scriptedClient{})
	if _, err := svc.ExplainCashback(context.Background(), port.CashbackExplainRequest{}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("a missing card should be a validation error, got %v", err)
	}
	if _, err := svc.ExplainCashback(context.Background(), port.CashbackExplainRequest{Card: "Platinum"}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("an unknown card should be not found, got %v", err)
	}
}
