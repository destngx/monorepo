package domain

import (
	"testing"
	"time"
)

func testCardConfig() CardConfig {
	return CardConfig{Banks: []CardBank{{
		Name: "Sacombank", SharedAccount: "Credit Card Sacombank", DefaultCardKeyword: "platinum", SharedMonthlyCap: 600_000,
		Cards: []CardProgram{
			{
				Name: "UNIQ", Bank: "Sacombank", Keyword: "uniq", DefaultLimit: 40_000_000, AnnualFee: 599_000, MonthlyCap: 600_000,
				StatementDay: 5, DueDay: 30,
				Rules: []CashbackRule{
					{Name: "Supermarket & Transport", Tag: "uniq supermarket and transport", Rate: 20, CapMonthly: 300_000},
					{Name: "Others", Tag: "uniq others", Rate: 0.5, CapMonthly: 300_000, Fallback: true},
				},
			},
			{
				Name: "Platinum", Bank: "Sacombank", Keyword: "platinum", DefaultLimit: 40_000_000, AnnualFee: 599_000, MonthlyCap: 600_000,
				StatementDay: 5, DueDay: 30,
				Rules: []CashbackRule{
					{Name: "Online", Tag: "platinum online", Rate: 5, CapMonthly: 600_000},
					{Name: "Overseas", Tag: "platinum overseas", Rate: 3, CapMonthly: 600_000},
					{Name: "Others", Tag: "platinum", Rate: 0.5, CapMonthly: 600_000, Fallback: true},
				},
			},
		},
	}}}
}

func cardTxn(date time.Time, payment float64, tags ...string) Transaction {
	t := txn(date, "Shopping", CategoryTypeExpense, payment, 0)
	t.AccountName = "Credit Card Sacombank"
	t.Tags = tags
	return t
}

func TestTransactionCashback(t *testing.T) {
	platinum := testCardConfig().Banks[0].Cards[1]
	tests := []struct {
		name     string
		tx       Transaction
		amount   float64
		ruleName string
	}{
		{"exact tag", cardTxn(day(2026, 3, 6), 1_000_000, "Platinum Online"), 50_000, "Online"},
		{"untagged uses fallback", cardTxn(day(2026, 3, 6), 1_000_000), 5_000, "Others"},
		{"deposit earns nothing", txn(day(2026, 3, 6), "Shopping", CategoryTypeExpense, 0, 1_000), 0, ""},
		{"beginning balance excluded", txn(day(2026, 3, 6), "[Beginning Balance]", CategoryTypeNonBudget, 1_000_000, 0), 0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, ruleName := TransactionCashback(platinum, tt.tx)
			if amount != tt.amount || ruleName != tt.ruleName {
				t.Fatalf("got %v %q want %v %q", amount, ruleName, tt.amount, tt.ruleName)
			}
		})
	}
}

func TestComputeCardReportsSharedAccount(t *testing.T) {
	today := day(2026, 3, 20)
	accounts := []Account{{
		Name: "Credit Card Sacombank", Type: AccountTypeNegativeActive, ClearedBalance: -8_000_000, Balance: -8_000_000,
		GoalAmount: money(80_000_000), DueDate: text("5, 30"),
	}}
	refund := txn(day(2026, 3, 8), "Refunds/Reimbursements", CategoryTypeIncome, 0, 250_000)
	refund.AccountName = "Credit Card Sacombank"
	refund.Payee = "Sacombank Cashback"
	refund.Memo = text("2026-02")
	transactions := []Transaction{
		cardTxn(day(2026, 3, 6), 2_000_000, "uniq supermarket and transport"), // 400k, capped to 300k
		cardTxn(day(2026, 3, 7), 1_000_000, "uniq others"),                    // 5k
		cardTxn(day(2026, 3, 9), 4_000_000, "platinum online"),                // 200k
		cardTxn(day(2026, 3, 10), 2_000_000),                                  // untagged -> platinum, 10k
		cardTxn(day(2026, 3, 1), 9_000_000, "platinum online"),                // before cycle start
		refund,
	}
	reports := ComputeCardReports(testCardConfig(), accounts, transactions, today)
	if len(reports) != 1 || len(reports[0].Accounts) != 1 {
		t.Fatalf("expected one shared account group, got %+v", reports)
	}
	group := reports[0].Accounts[0]
	if !group.Shared || len(group.Cards) != 2 || group.Limit != 80_000_000 || group.Utilization != 10 {
		t.Fatalf("unexpected group %+v", group)
	}
	uniq, platinum := group.Cards[0], group.Cards[1]
	if uniq.EstimatedCashback != 305_000 || uniq.SpentThisCycle != 3_000_000 || uniq.TransactionCount != 2 {
		t.Fatalf("unexpected uniq %+v", uniq)
	}
	if platinum.EstimatedCashback != 210_000 || platinum.SpentThisCycle != 6_000_000 {
		t.Fatalf("unexpected platinum %+v", platinum)
	}
	if platinum.Cycle.CycleStart != "2026-03-05" || platinum.Cycle.NextDueDate != "2026-03-30" {
		t.Fatalf("unexpected cycle %+v", platinum.Cycle)
	}
	if platinum.RefundYTD != 250_000 || platinum.NetEarnYTD != 250_000-599_000 || uniq.RefundYTD != 0 {
		t.Fatalf("expected the untagged refund to belong to platinum, got %+v / %+v", platinum.RefundYTD, uniq.RefundYTD)
	}
	if group.EstimatedCashback != 515_000 || group.AnnualFees != 1_198_000 {
		t.Fatalf("unexpected group totals %+v", group)
	}
	if len(platinum.CycleTransactions) == 0 || !platinum.CycleTransactions[0].Date.Equal(day(2026, 3, 10)) {
		t.Fatal("expected cycle transactions newest first")
	}
}

func TestCardHistoryAttributesRefundsByEffectiveMonth(t *testing.T) {
	today := day(2026, 3, 20)
	refund := txn(day(2026, 3, 8), "Refunds/Reimbursements", CategoryTypeIncome, 0, 10_000)
	refund.AccountName = "Credit Card Sacombank"
	refund.Payee = "cashback"
	refund.Memo = text("2026-02")
	transactions := []Transaction{cardTxn(day(2026, 2, 10), 4_000_000), refund}
	platinum := ComputeCardReports(testCardConfig(), nil, transactions, today)[0].Accounts[0].Cards[1]

	var february CardHistoryMonth
	for _, month := range platinum.History {
		if month.Month == "2026-02" {
			february = month
		}
	}
	if february.Spent != 4_000_000 || february.PotentialCashback != 20_000 || february.ActualRefund != 10_000 || february.Missed != 10_000 {
		t.Fatalf("unexpected february %+v", february)
	}
	if february.Efficiency != 0.25 {
		t.Fatalf("expected efficiency 0.25%%, got %v", february.Efficiency)
	}
	if platinum.History[0].Month != "2026-03" {
		t.Fatalf("expected newest month first, got %s", platinum.History[0].Month)
	}
}

func TestCardUsesOwnAccountAndConfigDaysWhenSheetHasNone(t *testing.T) {
	accounts := []Account{{Name: "Card UNIQ", Type: AccountTypeNegativeActive, ClearedBalance: -1_000_000}}
	transactions := []Transaction{
		{AccountName: "Card UNIQ", Date: day(2026, 3, 6), Payment: money(100_000), Tags: []string{"platinum"}}, // other card tag
		{AccountName: "Card UNIQ", Date: day(2026, 3, 7), Payment: money(200_000)},
	}
	reports := ComputeCardReports(testCardConfig(), accounts, transactions, day(2026, 3, 20))
	uniq := reports[0].Accounts[0].Cards[0]
	if uniq.AccountName != "Card UNIQ" || uniq.Limit != 40_000_000 || uniq.Cycle.StatementDay != 5 {
		t.Fatalf("unexpected uniq %+v", uniq)
	}
	if uniq.SpentThisCycle != 200_000 {
		t.Fatalf("expected other card tag excluded, got %v", uniq.SpentThisCycle)
	}
	if uniq.LifetimeFees != 599_000 {
		t.Fatalf("expected one year of fees, got %v", uniq.LifetimeFees)
	}
}
