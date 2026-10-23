package local_workbook

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/service"
	"testing"
)

// The writes for inputs that were sheet-only (D47) round-trip through the
// local workbook: each tab is created on first use and read back.
func TestGivenTheLocalWorkbookWhenWritingCardConfigThenTheStrictReadAcceptsIt(t *testing.T) {
	client := openFixture(t)
	db := service.NewDatabaseService(client, nil, logger.NewTestLogger(t), service.DatabaseServiceOptions{})

	if _, err := db.CreateCardProgram(domain.ProgramInput{
		Program: "Gold", Bank: "Acme", SharedAccount: "Bank", From: "2026-01-01", CapPeriod: "cycle",
		Rules: []domain.ProgramRuleInput{{Rule: "Dining", MCCs: "5812-5814", Rate: 3}, {Rule: "Others", Rate: 0.5, Fallback: true}},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateCard(domain.CardInput{Name: "Gold Card", Program: "Gold", Keyword: "gold", Expiry: "08/29", StatementDay: 5, DueDay: 25, Last4: "0123"}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SaveMerchant(domain.MerchantInput{Pattern: "Bakery", MCC: 742}); err != nil {
		t.Fatal(err)
	}

	config, err := db.ListCardConfig(true)
	if err != nil {
		t.Fatal(err)
	}
	if len(config.Programs) != 1 || len(config.Programs[0].Versions) != 1 || len(config.Programs[0].Versions[0].Rules) != 2 {
		t.Fatalf("programs = %+v", config.Programs)
	}
	if len(config.Cards) != 1 || config.Cards[0].Expiry != "08/29" || config.Cards[0].Last4 != "0123" {
		t.Fatalf("cards = %+v", config.Cards)
	}
	if len(config.Banks) != 1 || config.Banks[0].SharedAccount != "Bank" {
		t.Fatalf("banks = %+v", config.Banks)
	}
	if len(config.Merchants) != 1 || config.Merchants[0].MCC != 742 {
		t.Fatalf("merchants = %+v", config.Merchants)
	}
}

func TestGivenTheLocalWorkbookWhenWritingHoldingsThenTheAssetsReadThemBack(t *testing.T) {
	client := openFixture(t)
	db := service.NewDatabaseService(client, nil, logger.NewTestLogger(t), service.DatabaseServiceOptions{})
	if err := client.CreateSheet("InvestmentFundCertificate", []any{"Fund certificates"}); err != nil {
		t.Fatal(err)
	}
	if err := client.UpdateCells([]domain.CellUpdate{{Range: "InvestmentFundCertificate!B3", Values: []any{"Index", "Total Unit", "Note", "History"}}}); err != nil {
		t.Fatal(err)
	}

	if _, err := db.RecordFundTrade(domain.FundTradeInput{Code: "FUNDA", Date: "2026-01-02", Units: 10.5}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.RecordFundTrade(domain.FundTradeInput{Code: "FUNDA", Date: "2026-02-02", Units: -0.5}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SetCryptoHoldings(domain.CryptoHoldingsInput{Holdings: []domain.CryptoHoldingInput{{Platform: "Exchange A", Symbol: "BTC", Quantity: 0.25}}}); err != nil {
		t.Fatal(err)
	}

	assets, err := db.GetInvestmentAssets(true)
	if err != nil {
		t.Fatal(err)
	}
	if len(assets.Funds.Items) != 1 || assets.Funds.Items[0].Symbol != "FUNDA" || *assets.Funds.Items[0].Quantity != 10 {
		t.Fatalf("funds = %+v", assets.Funds.Items)
	}
	if trades, issues := domain.ParseFundHistory(read(t, client, "InvestmentFundCertificate!E4")[0][0]); len(trades) != 2 || len(issues) != 0 {
		t.Fatalf("history = %+v %v", trades, issues)
	}
	if len(assets.Crypto.Items) != 1 || assets.Crypto.Items[0].Symbol != "BTC" || *assets.Crypto.Items[0].Quantity != 0.25 {
		t.Fatalf("crypto = %+v", assets.Crypto.Items)
	}
}
