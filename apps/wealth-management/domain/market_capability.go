package domain

import "strings"

type TickerType string

const (
	TickerTypeEquity TickerType = "equity"
	TickerTypeIFC    TickerType = "ifc"
	TickerTypeGold   TickerType = "gold"
	// TickerTypeCrypto prices a coin in USD (WM-INV-2).
	TickerTypeCrypto TickerType = "crypto"
)

type SeriesType string

const (
	SeriesTypeEquity  SeriesType = "equity"
	SeriesTypeGoldUSD SeriesType = "gold_usd"
	SeriesTypeForex   SeriesType = "forex"
)

type Ticker struct {
	Symbol    string      `json:"symbol"`
	Type      TickerType  `json:"type"`
	Provider  string      `json:"provider"`
	Currency  string      `json:"currency,omitempty"`
	Price     float64     `json:"price,omitempty"`
	Bid       float64     `json:"bid,omitempty"`
	Ask       float64     `json:"ask,omitempty"`
	Open      float64     `json:"open,omitempty"`
	High      float64     `json:"high,omitempty"`
	Low       float64     `json:"low,omitempty"`
	Close     float64     `json:"close,omitempty"`
	Timestamp string      `json:"timestamp,omitempty"`
	Source    *DataSource `json:"source,omitempty"`
}

// RateBasis is what an exchange rate means. Sources in one fallback chain
// quote different rates for the same pair, so the basis travels with the
// rate. Empty means unknown (copies cached before the server sent it).
type RateBasis string

const (
	RateBasisBankSell     RateBasis = "bank_sell"
	RateBasisMarket       RateBasis = "market"
	RateBasisReference    RateBasis = "reference"
	RateBasisCryptoMarket RateBasis = "crypto_market"
)

type ExchangeRate struct {
	From      string      `json:"from"`
	To        string      `json:"to"`
	Rate      float64     `json:"rate"`
	Provider  string      `json:"provider"`
	Timestamp string      `json:"timestamp,omitempty"`
	Basis     RateBasis   `json:"basis,omitempty"`
	Source    *DataSource `json:"source,omitempty"`
}

type PricePoint struct {
	Price     float64 `json:"price"`
	Timestamp string  `json:"timestamp"`
}

type PriceSeries struct {
	Provider   string      `json:"provider"`
	SeriesType SeriesType  `json:"seriesType"`
	Currency   string      `json:"currency,omitempty"`
	Data       interface{} `json:"data"`
	Source     *DataSource `json:"source,omitempty"`
}

// BankRate is one published VND savings rate. TenorMonths is 0 for a
// no-term deposit and nil when the source publishes no tenor (fmarket's
// headline rates); InstantAccess is true only for no-term deposits
// (withdrawable any time without losing interest). Channel is "counter" or
// "online" when the source says which.
type BankRate struct {
	Bank          string  `json:"bank"`
	Tenor         string  `json:"tenor"`
	TenorMonths   *int    `json:"tenorMonths,omitempty"`
	InstantAccess bool    `json:"instantAccess,omitempty"`
	Channel       string  `json:"channel,omitempty"`
	Rate          float64 `json:"rate"`
	Currency      string  `json:"currency,omitempty"`
	Timestamp     string  `json:"timestamp,omitempty"`
	UpdatedAt     string  `json:"updatedAt,omitempty"`
	RecordedAt    string  `json:"recordedAt,omitempty"`
}

// BankRates is the published savings rates with their data source.
type BankRates struct {
	Rates  []BankRate  `json:"rates"`
	Source *DataSource `json:"source,omitempty"`
}

// What holds the money of a flexible savings product.
const (
	// SavingsBackingDeposit: a bank deposit; the rate is interest.
	SavingsBackingDeposit = "deposit"
	// SavingsBackingBusinessCooperation: a business cooperation contract
	// with a company that invests the money (MoMo Túi Thần Tài); the rate
	// is the company's promise and the money is not a deposit.
	SavingsBackingBusinessCooperation = "business_cooperation"
)

// EarningStartsNextWorkingDay: money moved in Monday to Thursday starts
// earning the next day, Friday to Sunday the next Monday, before a public
// holiday the first working day after it.
const EarningStartsNextWorkingDay = "next_working_day"

// FlexibleSaving is a product that pays interest on money that can be spent
// or withdrawn at any time, such as an e-wallet's earning balance, as the
// market data server reads it from the provider's page. ID links an account
// to it (idleCash.accountProducts). WithdrawalMaxWorkingHours is the
// longest a withdrawal takes as the provider states it (nil: none stated).
// AsOf and Stale say when the server read the page and whether that read
// is an old copy kept because the page failed since.
type FlexibleSaving struct {
	ID                        string  `json:"id"`
	Provider                  string  `json:"provider"`
	Product                   string  `json:"product"`
	Rate                      float64 `json:"rate"`
	Backing                   string  `json:"backing"`
	EarningStarts             string  `json:"earningStarts"`
	WithdrawalMaxWorkingHours *int    `json:"withdrawalMaxWorkingHours,omitempty"`
	Currency                  string  `json:"currency,omitempty"`
	UpdatedAt                 string  `json:"updatedAt,omitempty"`
	URL                       string  `json:"url"`
	AsOf                      string  `json:"asOf,omitempty"`
	Stale                     bool    `json:"stale,omitempty"`
}

// Name is "Zalopay Số dư sinh lời".
func (p FlexibleSaving) Name() string {
	return strings.TrimSpace(p.Provider + " " + p.Product)
}

// FlexibleSavings is every provider's flexible savings products; Failed
// names the providers that could not be read (each "provider: reason").
type FlexibleSavings struct {
	Products []FlexibleSaving `json:"products"`
	Failed   []string         `json:"failed"`
	Source   *DataSource      `json:"source,omitempty"`
}

// Product returns the product with the ID, ignoring case.
func (s FlexibleSavings) Product(id string) (FlexibleSaving, bool) {
	for _, product := range s.Products {
		if strings.EqualFold(product.ID, strings.TrimSpace(id)) {
			return product, true
		}
	}
	return FlexibleSaving{}, false
}
