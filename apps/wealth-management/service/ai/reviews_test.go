package ai

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"strings"
	"testing"
	"time"
)

func TestTheCardSummaryReportsCapGroupRoomAndTheStatementEstimate(t *testing.T) {
	db := captureDatabase()
	refundMonth := "2026-09"
	cards := captureCards
	cards.Programs = []domain.Program{
		{Name: "Gold", Bank: "Acme", Versions: []domain.ProgramVersion{{Period: domain.CapPeriodCycle, Rules: []domain.CashbackRule{
			{Name: "Shopping", MCCs: []domain.MCCRange{{From: 5311, To: 5311}}, Rate: 10, Cap: 100_000, CapGroup: "Everyday"},
			{Name: "Others", Rate: 1, Fallback: true},
		}}}},
		captureCards.Programs[1],
	}
	cards.Merchants = []domain.MerchantMCC{{Pattern: "shopee", MCC: 5311, Confidence: domain.MCCConfirmed}}
	db.Cards = cards
	db.Transactions = append(db.Transactions, domain.Transaction{
		ID: "c1", AccountName: "Credit Card Acme", Payee: "Shopee", Category: "Shopping", Tags: []string{"gold"},
		Date: time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC), Payment: money(300_000),
	}, domain.Transaction{
		ID: "c0", AccountName: "Credit Card Acme", Payee: "Grab", Category: "Shopping", Tags: []string{"gold"},
		Date: time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC), Payment: money(120_000),
	}, domain.Transaction{
		ID: "c2", AccountName: "Credit Card Acme", Payee: "Acme Cashback", Category: "Refunds/Reimbursements", Tags: []string{"gold"},
		Date: time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC), Deposit: money(1_200), Memo: &refundMonth,
	})
	svc := newCaptureService(t, db, &scriptedClient{})
	snap, err := svc.loadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	vars, _, err := svc.cardSummaryInputs(snap, port.ReviewRequest{Kind: port.ReviewCreditCard, Card: "Gold Card"})
	if err != nil {
		t.Fatal(err)
	}
	stats := vars["cardStatsJson"]
	if !strings.Contains(stats, `"ruleCapsLeft":{"Everyday":70000}`) {
		t.Fatalf("the cap group's room should be reported: %s", stats)
	}
	// Pay-by 2026-10-29 settles the 2026-09-05 statement: only the Grab payment.
	if !strings.Contains(stats, `"statementEstimate":120000`) {
		t.Fatalf("the statement estimate should be reported: %s", stats)
	}
	// The open period only has an estimate; the refund that is actually paid is the last closed period's.
	if !strings.Contains(stats, `"cashbackEstimate":30000,"cashbackEstimateOnConfirmedMcc":30000`) || strings.Contains(stats, `"confirmedCashback"`) {
		t.Fatalf("the open period's cashback should read as an estimate: %s", stats)
	}
	if !strings.Contains(stats, `"previousPeriod":{"start":"2026-09-05","end":"2026-10-04","spent":120000,"cashbackEstimate":1200,"actualRefund":1200,"reconciliation":"match"}`) {
		t.Fatalf("the last closed period should carry its actual refund: %s", stats)
	}
}

func TestAPastMonthCardSummaryReportsThatPeriodAlone(t *testing.T) {
	db := captureDatabase()
	refundMonth := "2026-09"
	db.Transactions = append(db.Transactions, domain.Transaction{
		ID: "c0", AccountName: "Credit Card Acme", Payee: "Grab", Category: "Shopping", Tags: []string{"gold"},
		Date: time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC), Payment: money(120_000),
	}, domain.Transaction{
		ID: "c2", AccountName: "Credit Card Acme", Payee: "Acme Cashback", Category: "Refunds/Reimbursements", Tags: []string{"gold"},
		Date: time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC), Deposit: money(1_200), Memo: &refundMonth,
	})
	svc := newCaptureService(t, db, &scriptedClient{})
	snap, err := svc.loadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	vars, _, err := svc.cardSummaryInputs(snap, port.ReviewRequest{Kind: port.ReviewCreditCard, Card: "Gold Card", Month: "2026-09"})
	if err != nil {
		t.Fatal(err)
	}
	stats := vars["cardStatsJson"]
	if !strings.Contains(stats, `"closedPeriod":{"start":"2026-09-05","end":"2026-10-04","spent":120000,"cashbackEstimate":0,"actualRefund":1200,"reconciliation":"above"`) {
		t.Fatalf("the past period should be reported with its refund: %s", stats)
	}
	if strings.Contains(stats, "currentPeriod") || strings.Contains(stats, "previousPeriod") || strings.Contains(stats, "statementEstimate") {
		t.Fatalf("a past month carries no current figures: %s", stats)
	}
}
