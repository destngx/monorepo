package domain

import "time"

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
	Symbol    string     `json:"symbol"`
	Type      TickerType `json:"type"`
	Provider  string     `json:"provider"`
	Currency  string     `json:"currency,omitempty"`
	Price     float64    `json:"price,omitempty"`
	Bid       float64    `json:"bid,omitempty"`
	Ask       float64    `json:"ask,omitempty"`
	Open      float64    `json:"open,omitempty"`
	High      float64    `json:"high,omitempty"`
	Low       float64    `json:"low,omitempty"`
	Close     float64    `json:"close,omitempty"`
	Timestamp string     `json:"timestamp,omitempty"`
}

type ExchangeRate struct {
	From      string  `json:"from"`
	To        string  `json:"to"`
	Rate      float64 `json:"rate"`
	Provider  string  `json:"provider"`
	Timestamp string  `json:"timestamp,omitempty"`
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
}

type BankRate struct {
	Bank       string  `json:"bank"`
	Tenor      string  `json:"tenor"`
	Rate       float64 `json:"rate"`
	Currency   string  `json:"currency,omitempty"`
	Timestamp  string  `json:"timestamp,omitempty"`
	UpdatedAt  string  `json:"updatedAt,omitempty"`
	RecordedAt string  `json:"recordedAt,omitempty"`
}

type MarketRoutingConfig struct {
	GetTicker           map[TickerType][]string
	GetExchangeRate     map[string][]string
	GetPriceSeries      map[SeriesType][]string
	GetBankInterestRate []string
	// GetCandles serves VN equities and indices, FX, gold and VIX
	// (WM-INV-11/12); providers skip symbols they do not cover.
	GetCandles []string
	// CacheTTL is how long each capability's answers stay cached
	// (WM-RULE-10).
	CacheTTL MarketCacheTTLs
}

// MarketCacheTTLs are the cache lifetimes, in seconds, per market
// capability (WM-RULE-10). A zero field uses its default.
type MarketCacheTTLs struct {
	Ticker       int
	ExchangeRate int
	PriceSeries  int
	Candles      int
	BankRates    int
}

// DefaultMarketCacheTTLs: tickers move within the minute, exchange rates
// within ten, series and candles are daily bars, and bank rates change
// rarely.
func DefaultMarketCacheTTLs() MarketCacheTTLs {
	return MarketCacheTTLs{
		Ticker:       int(time.Minute.Seconds()),
		ExchangeRate: int((10 * time.Minute).Seconds()),
		PriceSeries:  int((5 * time.Minute).Seconds()),
		Candles:      int((5 * time.Minute).Seconds()),
		BankRates:    int(time.Hour.Seconds()),
	}
}

// WithDefaults fills every zero lifetime with its default.
func (t MarketCacheTTLs) WithDefaults() MarketCacheTTLs {
	defaults := DefaultMarketCacheTTLs()
	pick := func(value, fallback int) int {
		if value > 0 {
			return value
		}
		return fallback
	}
	return MarketCacheTTLs{
		Ticker:       pick(t.Ticker, defaults.Ticker),
		ExchangeRate: pick(t.ExchangeRate, defaults.ExchangeRate),
		PriceSeries:  pick(t.PriceSeries, defaults.PriceSeries),
		Candles:      pick(t.Candles, defaults.Candles),
		BankRates:    pick(t.BankRates, defaults.BankRates),
	}
}

func DefaultMarketRoutingConfig() MarketRoutingConfig {
	return MarketRoutingConfig{
		GetTicker: map[TickerType][]string{
			TickerTypeEquity: {"vnstock", "fmarket"},
			TickerTypeIFC:    {"fmarket", "vnstock"},
			TickerTypeGold:   {"fmarket"},
			TickerTypeCrypto: {"coingecko"},
		},
		GetExchangeRate: map[string][]string{
			"USD_VND":  {"fmarket", "yahoo", "currencyapi"},
			"USDT_VND": {"coingecko", "currencyapi"},
		},
		GetPriceSeries: map[SeriesType][]string{
			SeriesTypeEquity:  {"vnstock", "fmarket"},
			SeriesTypeGoldUSD: {"fmarket"},
			SeriesTypeForex:   {"yahoo", "fmarket"},
		},
		GetBankInterestRate: []string{"fmarket"},
		GetCandles:          []string{"vnstock", "yahoo", "fmarket"},
		CacheTTL:            DefaultMarketCacheTTLs(),
	}
}
