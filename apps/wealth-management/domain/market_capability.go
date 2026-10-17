package domain

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
