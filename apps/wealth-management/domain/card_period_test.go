package domain

import (
	"errors"
	"testing"
)

func TestTheCurrentPeriodDetailExplainsTheEstimate(t *testing.T) {
	detail, err := ComputeCardPeriod(testCardConfig(), nil, marchLedger(), "Gold Card", "", day(2026, 3, 20))
	if err != nil {
		t.Fatal(err)
	}
	if detail.Card != "Gold Card" || detail.Month != "2026-03" || detail.Start != "2026-03-05" || detail.End != "2026-04-04" || !detail.Open {
		t.Fatalf("period %+v", detail)
	}
	// The same figures as the card's standing (WM-CARD-12).
	if detail.Spent != 17_300_000 || detail.Cashback != newCashbackAmount(200_000, 231_500) || detail.Status != ReconciliationOpen {
		t.Fatalf("totals %+v", detail)
	}
	if len(detail.Parts) != 1 {
		t.Fatalf("one version in effect: %+v", detail.Parts)
	}
	part := detail.Parts[0]
	if part.VersionFrom != "2025-01-01" || part.CardCap != 300_000 || part.CapPeriod != CapPeriodCycle || len(part.Rules) == 0 || len(part.Standings) == 0 {
		t.Fatalf("terms %+v", part)
	}
	if len(part.Payments) != 5 || part.Payments[0].Transaction.Date != day(2026, 3, 7) || part.Payments[4].Transaction.Date != day(2026, 4, 1) {
		t.Fatalf("payments oldest first: %+v", part.Payments)
	}
	groceries := part.Payments[0]
	if groceries.Rule == "" || groceries.Cashback <= 0 || groceries.MCC.Code != 5411 || groceries.MCC.Status != MCCConfirmed {
		t.Fatalf("a payment names its MCC, rule and cashback before caps: %+v", groceries)
	}
}

func TestAPastPeriodDetailCarriesItsRefund(t *testing.T) {
	ledger := append(marchLedger(), cashbackRefund(day(2026, 3, 18), 10_000, "2026-02", "gold"))
	detail, err := ComputeCardPeriod(testCardConfig(), nil, ledger, "gold card", "2026-02", day(2026, 3, 20))
	if err != nil {
		t.Fatal(err)
	}
	if detail.Start != "2026-02-05" || detail.End != "2026-03-04" || detail.Open || detail.Spent != 1_000_000 || detail.Refund != 10_000 || len(detail.Parts) != 1 || len(detail.Parts[0].Payments) != 1 {
		t.Fatalf("past period %+v", detail)
	}
}

func TestCardPeriodDetailErrors(t *testing.T) {
	config := testCardConfig()
	if _, err := ComputeCardPeriod(config, nil, marchLedger(), "Platinum", "", day(2026, 3, 20)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown card: %v", err)
	}
	if _, err := ComputeCardPeriod(config, nil, marchLedger(), "Gold Card", "2025-06", day(2026, 3, 20)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("a past month without payments: %v", err)
	}
	if _, err := ComputeCardPeriod(config, nil, marchLedger(), "Gold Card", "March", day(2026, 3, 20)); !errors.Is(err, ErrValidation) {
		t.Fatalf("bad month: %v", err)
	}
	closed := day(2026, 3, 10)
	config.Cards[0].ClosedFrom = &closed
	if _, err := ComputeCardPeriod(config, nil, marchLedger(), "Gold Card", "", day(2026, 3, 20)); !errors.Is(err, ErrValidation) {
		t.Fatalf("a closed card has no current period: %v", err)
	}
}
