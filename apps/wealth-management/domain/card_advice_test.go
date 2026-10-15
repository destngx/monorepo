package domain

import (
	"testing"
)

func adviceAccounts() []Account {
	return []Account{{Name: "Credit Card Acme", Type: AccountTypeNegativeActive, ClearedBalance: -8_000_000}}
}

func TestSuggestCardsRanksByWhatThePaymentAdds(t *testing.T) {
	ledger := marchLedger()[:8]
	suggest := func(query CardPaymentQuery) []CardSuggestion {
		t.Helper()
		suggestions := SuggestCards(testCardConfig(), adviceAccounts(), ledger, query, day(2026, 3, 20))
		if len(suggestions) != 2 {
			t.Fatalf("expected both held cards, got %+v", suggestions)
		}
		return suggestions
	}

	ranked := suggest(CardPaymentQuery{Payee: "Coop", Amount: 1_000_000})
	if ranked[0].Card != "Gold Card" || ranked[0].Rule != "Groceries" || ranked[0].Cashback != newCashbackAmount(50_000, 50_000) || ranked[0].MCC != (PaymentMCC{5411, MCCConfirmed}) {
		t.Fatalf("first %+v", ranked[0])
	}
	if ranked[1].Card != "Silver Card" || ranked[1].Rule != "Others" || ranked[1].Cashback.Total != 5_000 || !ranked[1].FitsLimit || ranked[1].RemainingLimit != 12_000_000 {
		t.Fatalf("second %+v", ranked[1])
	}

	// Reaching the tier lifts the groceries already paid this period too, up to the rule cap.
	tier := suggest(CardPaymentQuery{Payee: "Coop", Amount: 2_000_000})[0]
	if tier.Rule != "Groceries tier 2" || tier.Cashback != newCashbackAmount(100_000, 100_000) {
		t.Fatalf("tier %+v", tier)
	}

	mcc := 5812
	known := suggest(CardPaymentQuery{Payee: "New place", Amount: 1_000_000, MCC: &mcc})[0]
	if known.Rule != "Dining" || known.Cashback != newCashbackAmount(30_000, 30_000) {
		t.Fatalf("a known MCC is confirmed: %+v", known)
	}

	unknown := suggest(CardPaymentQuery{Payee: "New place", Amount: 1_000_000})[0]
	if unknown.MCC.Status != MCCUnknown || unknown.Cashback != newCashbackAmount(0, 10_000) {
		t.Fatalf("an unknown MCC earns the fallback, estimated: %+v", unknown)
	}

	for _, suggestion := range suggest(CardPaymentQuery{Payee: "Coop", Amount: 13_000_000}) {
		if suggestion.FitsLimit {
			t.Fatalf("13M exceeds the 12M left: %+v", suggestion)
		}
	}
}

func TestCompareProgramsReplaysTheLastYear(t *testing.T) {
	report := ComparePrograms(testCardConfig(), adviceAccounts(), marchLedger()[:8], day(2026, 3, 20))
	if report.From != "2025-03-21" || report.To != "2026-03-20" || report.Payments != 7 || len(report.Programs) != 3 {
		t.Fatalf("report %+v", report)
	}
	want := []ProgramComparison{
		{Program: "Silver", Bank: "Acme", Held: true, VersionFrom: "2025-01-01", Spent: 15_300_000, Cashback: newCashbackAmount(65_000, 121_500), Net: 121_500},
		{Program: "Gold", Bank: "Acme", Held: true, VersionFrom: "2025-01-01", Spent: 15_300_000, Cashback: newCashbackAmount(250_000, 291_500), AnnualFee: 599_000, Net: -307_500},
		{Program: "Platinum", Bank: "Beta", VersionFrom: "2026-01-01", Spent: 15_300_000, Cashback: newCashbackAmount(260_000, 306_000), AnnualFee: 1_000_000, Net: -694_000},
	}
	for i, program := range report.Programs {
		if program != want[i] {
			t.Fatalf("rank %d: got %+v want %+v", i, program, want[i])
		}
	}
}
