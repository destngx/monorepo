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
