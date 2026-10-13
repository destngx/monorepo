package domain

import (
	"fmt"
	"strings"
	"time"
)

// Portfolio sections are the holding ledgers of the Assets tab.
const (
	PortfolioSectionCrypto = "crypto"
	PortfolioSectionFunds  = "funds"
)

// LivePrice is a market quote for one holding, with its source.
type LivePrice struct {
	Price     float64  `json:"price"`
	Currency  Currency `json:"currency"`
	Provider  string   `json:"provider"`
	Timestamp string   `json:"timestamp,omitempty"`
}

// ValuedHolding is a sheet holding with its live price and VND value. Issue
// says why it has no VND value.
type ValuedHolding struct {
	Section string `json:"section"`
	Holding
	LivePrice *LivePrice `json:"livePrice,omitempty"`
	ValueVND  *float64   `json:"valueVND"`
	Issue     string     `json:"issue,omitempty"`
}

// PortfolioTotals follow WM-RULE-8: VND = VND accounts + funds, crypto =
// max(crypto account balances, crypto holdings).
type PortfolioTotals struct {
	VNDAccounts    float64 `json:"vndAccounts"`
	Funds          float64 `json:"funds"`
	VND            float64 `json:"vnd"`
	CryptoAccounts float64 `json:"cryptoAccounts"`
	CryptoHoldings float64 `json:"cryptoHoldings"`
	Crypto         float64 `json:"crypto"`
	Total          float64 `json:"total"`
	// Unvalued counts holdings and accounts left out of the totals.
	Unvalued int `json:"unvalued"`
}

// PortfolioValuation is the WM-INV-2/3/4 answer. Rates are VND per unit;
// there is no silent fallback rate, a missing one is listed in Issues.
type PortfolioValuation struct {
	Holdings    []ValuedHolding `json:"holdings"`
	Rates       []ExchangeRate  `json:"rates"`
	Totals      PortfolioTotals `json:"totals"`
	Issues      []string        `json:"issues"`
	GeneratedAt time.Time       `json:"generatedAt"`
}

// PortfolioInputs is everything ValuePortfolio needs, fetched by the caller.
// Prices are keyed by PriceKey; Rates by currency (VND per unit).
type PortfolioInputs struct {
	Assets      InvestmentAssets
	Accounts    []Account
	Prices      map[string]LivePrice
	PriceIssues map[string]string
	Rates       map[Currency]ExchangeRate
	RateIssues  map[Currency]string
	Now         time.Time
}

// PriceKey identifies a holding's live price.
func PriceKey(section string, symbol string) string {
	return section + ":" + strings.ToUpper(strings.TrimSpace(symbol))
}

// ValuePortfolio values every holding and computes the totals.
func ValuePortfolio(in PortfolioInputs) PortfolioValuation {
	result := PortfolioValuation{Holdings: []ValuedHolding{}, Rates: []ExchangeRate{}, Issues: []string{}, GeneratedAt: in.Now}
	for _, currency := range []Currency{CurrencyUSD, CurrencyUSDT} {
		if rate, ok := in.Rates[currency]; ok {
			result.Rates = append(result.Rates, rate)
		} else if issue := in.RateIssues[currency]; issue != "" {
			result.Issues = append(result.Issues, fmt.Sprintf("%s/VND rate unavailable: %s", currency, issue))
		}
	}

	sections := []struct {
		name     string
		holdings []Holding
	}{
		{PortfolioSectionCrypto, in.Assets.Crypto.Items},
		{PortfolioSectionFunds, in.Assets.Funds.Items},
	}
	totals := &result.Totals
	for _, section := range sections {
		for _, holding := range section.holdings {
			if strings.TrimSpace(holding.Symbol) == "" && holding.Value == nil {
				continue
			}
			valued := valueHolding(section.name, holding, in)
			result.Holdings = append(result.Holdings, valued)
			switch {
			case valued.ValueVND == nil:
				totals.Unvalued++
			case section.name == PortfolioSectionCrypto:
				totals.CryptoHoldings += *valued.ValueVND
			default:
				totals.Funds += *valued.ValueVND
			}
		}
	}

	for _, account := range in.Accounts {
		if account.Type == AccountTypeDeprecated || account.Balance == 0 {
			continue
		}
		// Balances are already VND (D25); the currency only marks USD/USDT
		// (exchange) accounts as crypto.
		isCrypto := account.Type == AccountTypeCrypto || account.Currency == CurrencyUSD || account.Currency == CurrencyUSDT
		vnd := account.Balance
		if isCrypto {
			totals.CryptoAccounts += vnd
		} else {
			totals.VNDAccounts += vnd
		}
	}

	totals.VND = totals.VNDAccounts + totals.Funds
	totals.Crypto = max(totals.CryptoAccounts, totals.CryptoHoldings)
	totals.Total = totals.VND + totals.Crypto
	return result
}

func valueHolding(section string, holding Holding, in PortfolioInputs) ValuedHolding {
	valued := ValuedHolding{Section: section, Holding: holding}
	key := PriceKey(section, holding.Symbol)
	if price, ok := in.Prices[key]; ok {
		valued.LivePrice = &price
	}

	if valued.Value == nil && valued.Quantity != nil && valued.LivePrice != nil {
		value := *valued.Quantity * valued.LivePrice.Price
		valued.Value = &value
		valued.Currency = valued.LivePrice.Currency
		valued.ValueSource = HoldingValueFromLivePrice
	}
	if valued.Value == nil {
		switch {
		case valued.Quantity == nil:
			valued.Issue = "no value or quantity in the sheet"
		case in.PriceIssues[key] != "":
			valued.Issue = "no live price: " + in.PriceIssues[key]
		default:
			valued.Issue = "no price in the sheet and no live price"
		}
		return valued
	}

	vnd, ok := toVND(*valued.Value, valued.Currency, in.Rates)
	if !ok {
		valued.Issue = fmt.Sprintf("no %s/VND rate", valued.Currency)
		return valued
	}
	valued.ValueVND = &vnd
	return valued
}

func toVND(value float64, currency Currency, rates map[Currency]ExchangeRate) (float64, bool) {
	if currency == CurrencyVND || currency == "" {
		return value, true
	}
	rate, ok := rates[currency]
	if !ok || rate.Rate <= 0 {
		return 0, false
	}
	return value * rate.Rate, true
}
