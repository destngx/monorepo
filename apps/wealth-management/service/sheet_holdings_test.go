package service

import (
	"apps/wealth-management/domain"
	"testing"
)

func TestParseCryptoHoldingsComputesValueFromQuantityAndPrice(t *testing.T) {
	rows := [][]string{
		{"Crypto portfolio"},
		{"", "Token", "Amount", "Price (USD)"},
		{"Binance", "BTC", "0.5", "60000"},
		{"", "ETH", "2", ""},
		{"", "", "", ""},
	}

	section := parseHoldings(rows, cryptoSection)

	if len(section.Items) != 2 || len(section.Holdings) != 2 {
		t.Fatalf("items = %+v", section.Items)
	}
	btc := section.Items[0]
	if btc.Platform != "Binance" || btc.Symbol != "BTC" || btc.Currency != domain.CurrencyUSD ||
		btc.Value == nil || *btc.Value != 30000 || btc.ValueSource != domain.HoldingValueFromQuantityPrice {
		t.Fatalf("btc = %+v", btc)
	}
	eth := section.Items[1]
	if eth.Platform != "Binance" || eth.Value != nil || eth.Price != nil || *eth.Quantity != 2 {
		t.Fatalf("eth must inherit platform and have no value without a price, got %+v", eth)
	}
}

func TestParseFundHoldingsPrefersSheetVNDValue(t *testing.T) {
	rows := [][]string{
		{"Platform", "Certificate", "Total Unit", "NAV price", "Value VND"},
		{"Fmarket", "DCDS", "100", "50000", "5100000"},
	}

	item := parseHoldings(rows, fundSection).Items[0]

	if item.Symbol != "DCDS" || *item.Quantity != 100 || *item.Value != 5100000 ||
		item.ValueSource != domain.HoldingValueFromSheet || item.Currency != domain.CurrencyVND {
		t.Fatalf("item = %+v", item)
	}
}

func TestParseHoldingsWithoutHeaderIsEmpty(t *testing.T) {
	section := parseHoldings([][]string{{"nothing here"}}, fundSection)
	if section.Items == nil || len(section.Items) != 0 || section.Headers == nil {
		t.Fatalf("section = %+v", section)
	}
}

func TestGivenAFundTabWithAHistoryColumnWhenParsingThenTradesAreReadByHeaderName(t *testing.T) {
	rows := [][]string{
		{"Platform", "Index", "Note", "History", "Total Unit"},
		{"Platform A", "FUND-A", "monthly", "5-3-2024,10,5;1-6-2024,-2,5", "8"},
		{"", "FUND-B", "", "", "3"},
		{"", "FUND-C", "", "5-3-2024", "1"},
	}

	items := parseHoldings(rows, fundSection).Items

	if len(items) != 3 {
		t.Fatalf("items = %+v", items)
	}
	fundA := items[0]
	if fundA.Symbol != "FUND-A" || *fundA.Quantity != 8 || len(fundA.History) != 2 ||
		fundA.History[0] != (domain.FundTrade{Date: "2024-03-05", Units: 10.5}) || fundA.History[1].Units != -2.5 || len(fundA.HistoryIssues) != 0 {
		t.Fatalf("fund A = %+v", fundA)
	}
	if items[1].History != nil || items[1].HistoryIssues != nil {
		t.Fatalf("a holding without History has no trades: %+v", items[1])
	}
	if len(items[2].HistoryIssues) != 1 {
		t.Fatalf("an unreadable entry is an issue: %+v", items[2])
	}
}
