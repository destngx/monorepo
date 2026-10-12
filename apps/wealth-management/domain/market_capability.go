package domain

import "time"

type TickerType string

const (
	TickerTypeEquity TickerType = "equity"
	TickerTypeIFC    TickerType = "ifc"
	TickerTypeGold   TickerType = "gold"
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
	CacheTTLSeconds     int
}

func DefaultMarketRoutingConfig() MarketRoutingConfig {
	return MarketRoutingConfig{
		GetTicker: map[TickerType][]string{
			TickerTypeEquity: {"vnstock", "fmarket"},
			TickerTypeIFC:    {"fmarket", "vnstock"},
			TickerTypeGold:   {"fmarket"},
		},
		GetExchangeRate: map[string][]string{
			"USD_VND": {"fmarket", "vnstock"},
		},
		GetPriceSeries: map[SeriesType][]string{
			SeriesTypeEquity:  {"vnstock", "fmarket"},
			SeriesTypeGoldUSD: {"fmarket"},
			SeriesTypeForex:   {"fmarket", "vnstock"},
		},
		GetBankInterestRate: []string{"fmarket"},
		CacheTTLSeconds:     int((5 * time.Minute).Seconds()),
	}
}
