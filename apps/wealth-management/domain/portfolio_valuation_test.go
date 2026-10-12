package domain

import (
	"strings"
	"testing"
	"time"
)

func ptr(value float64) *float64 { return &value }

func portfolioInputs() PortfolioInputs {
	return PortfolioInputs{
		Assets: InvestmentAssets{
			Crypto: InvestmentHoldingSection{Items: []Holding{
				{Platform: "Binance", Symbol: "BTC", Quantity: ptr(0.5), Price: ptr(80000), Currency: CurrencyUSD, Value: ptr(40000), ValueSource: HoldingValueFromQuantityPrice},
				{Platform: "Binance", Symbol: "ETH", Quantity: ptr(2), Currency: CurrencyUSD},
				{Platform: "Ledger", Symbol: "PEPE", Quantity: ptr(1000), Currency: CurrencyUSD},
				{Platform: "Ledger", Symbol: "", Currency: CurrencyUSD},
			}},
			Funds: InvestmentHoldingSection{Items: []Holding{
				{Platform: "Fmarket", Symbol: "VESAF", Value: ptr(31_000_000), Currency: CurrencyVND, ValueSource: HoldingValueFromSheet},
				{Platform: "Fmarket", Symbol: "DCDS", Quantity: ptr(100), Currency: CurrencyVND},
			}},
		},
		Accounts: []Account{
			{Name: "Techcombank", Type: AccountTypeBank, Currency: CurrencyVND, Balance: 50_000_000},
			{Name: "Credit Card", Type: AccountTypeNegativeActive, Currency: CurrencyVND, Balance: -5_000_000},
			{Name: "Old", Type: AccountTypeDeprecated, Currency: CurrencyVND, Balance: 999},
			{Name: "Binance", Type: AccountTypeCrypto, Currency: CurrencyUSDT, Balance: 60000},
		},
		Prices: map[string]LivePrice{
			PriceKey(PortfolioSectionCrypto, "btc"): {Price: 82000, Currency: CurrencyUSD, Provider: "coingecko"},
			PriceKey(PortfolioSectionCrypto, "ETH"): {Price: 3000, Currency: CurrencyUSD, Provider: "coingecko"},
			PriceKey(PortfolioSectionFunds, "DCDS"): {Price: 100_000, Currency: CurrencyVND, Provider: "fmarket"},
		},
		PriceIssues: map[string]string{PriceKey(PortfolioSectionCrypto, "PEPE"): "rate limited"},
		Rates: map[Currency]ExchangeRate{
			CurrencyUSD:  {From: "USD", To: "VND", Rate: 26000, Provider: "yahoo"},
			CurrencyUSDT: {From: "USDT", To: "VND", Rate: 25900, Provider: "coingecko"},
		},
		Now: time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC),
	}
}

func TestGivenSheetValuesWhenValuePortfolioThenPrefersSheetOverLivePrice(t *testing.T) {
	result := ValuePortfolio(portfolioInputs())
	btc := result.Holdings[0]
	if btc.ValueSource != HoldingValueFromQuantityPrice || *btc.Value != 40000 || *btc.ValueVND != 40000*26000 {
		t.Fatalf("expected the sheet qty × price in VND, got %#v", btc)
	}
	if btc.LivePrice == nil || btc.LivePrice.Price != 82000 {
		t.Fatalf("expected the live price shown for reference, got %#v", btc.LivePrice)
	}
}

func TestGivenMissingSheetPriceWhenValuePortfolioThenUsesLivePriceOrExplainsWhy(t *testing.T) {
	result := ValuePortfolio(portfolioInputs())
	if len(result.Holdings) != 5 {
		t.Fatalf("expected the blank row skipped, got %d holdings", len(result.Holdings))
	}
	eth := result.Holdings[1]
	if eth.ValueSource != HoldingValueFromLivePrice || *eth.Value != 6000 || *eth.ValueVND != 6000*26000 {
		t.Fatalf("expected quantity × live price, got %#v", eth)
	}
	pepe := result.Holdings[2]
	if pepe.ValueVND != nil || pepe.Issue != "no live price: rate limited" {
		t.Fatalf("expected an unvalued holding with the reason, got %#v", pepe)
	}
	dcds := result.Holdings[4]
	if *dcds.ValueVND != 10_000_000 || dcds.Currency != CurrencyVND {
		t.Fatalf("expected the fund valued from NAV, got %#v", dcds)
	}
}

func TestGivenAccountsAndHoldingsWhenValuePortfolioThenTotalsFollowRule8(t *testing.T) {
	totals := ValuePortfolio(portfolioInputs()).Totals
	if totals.VNDAccounts != 45_000_000 || totals.Funds != 41_000_000 || totals.VND != 86_000_000 {
		t.Fatalf("unexpected VND totals: %#v", totals)
	}
	if totals.CryptoHoldings != 46000*26000 || totals.CryptoAccounts != 60000*25900 {
		t.Fatalf("unexpected crypto parts: %#v", totals)
	}
	if totals.Crypto != 60000*25900 || totals.Total != totals.VND+totals.Crypto || totals.Unvalued != 1 {
		t.Fatalf("expected crypto = max(accounts, holdings), got %#v", totals)
	}
}

func TestGivenMissingRateWhenValuePortfolioThenNothingIsSilentlyConverted(t *testing.T) {
	in := portfolioInputs()
	delete(in.Rates, CurrencyUSDT)
	in.RateIssues = map[Currency]string{CurrencyUSDT: "coingecko: 429"}
	result := ValuePortfolio(in)
	if result.Totals.CryptoAccounts != 0 || result.Totals.Unvalued != 2 {
		t.Fatalf("expected the USDT account left out, got %#v", result.Totals)
	}
	joined := strings.Join(result.Issues, " | ")
	if !strings.Contains(joined, "USDT/VND rate unavailable: coingecko: 429") || !strings.Contains(joined, "account Binance (USDT) left out") {
		t.Fatalf("expected both issues, got %q", joined)
	}
	if len(result.Rates) != 1 || result.Rates[0].From != "USD" {
		t.Fatalf("expected only the USD rate, got %#v", result.Rates)
	}
}
