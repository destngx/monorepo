package domain

import (
	"fmt"
	"strings"
	"time"
)

// Timeframe is a candle resolution used by technicals and the market pulse
// (WM-RULE-9).
type Timeframe string

const (
	Timeframe1H Timeframe = "1h"
	Timeframe4H Timeframe = "4h"
	Timeframe1D Timeframe = "1d"
	Timeframe1W Timeframe = "1w"
)

// Timeframes are the supported resolutions, shortest first.
var Timeframes = []Timeframe{Timeframe1H, Timeframe4H, Timeframe1D, Timeframe1W}

func ParseTimeframe(raw string) (Timeframe, error) {
	value := Timeframe(strings.ToLower(strings.TrimSpace(raw)))
	for _, known := range Timeframes {
		if value == known {
			return value, nil
		}
	}
	return "", fmt.Errorf("%w: timeframe must be one of 1h, 4h, 1d, 1w", ErrValidation)
}

// PulseLookbackDays is the history the market pulse reads per timeframe
// (WM-RULE-9: 1h→7d, 4h→14d, 1d→60d, 1w→90d).
func (t Timeframe) PulseLookbackDays() int {
	switch t {
	case Timeframe1H:
		return 7
	case Timeframe4H:
		return 14
	case Timeframe1W:
		return 90
	default:
		return 60
	}
}

// AnalysisLookbackDays is long enough for EMA200 and seasonality on each
// timeframe (about 200+ candles).
func (t Timeframe) AnalysisLookbackDays() int {
	switch t {
	case Timeframe1H:
		return 60
	case Timeframe4H:
		return 180
	case Timeframe1W:
		return 5 * 365
	default:
		return 2 * 365
	}
}

// Candle is one OHLCV bar. Time is the bar's open time.
type Candle struct {
	Time   time.Time `json:"time"`
	Open   float64   `json:"open"`
	High   float64   `json:"high"`
	Low    float64   `json:"low"`
	Close  float64   `json:"close"`
	Volume float64   `json:"volume,omitempty"`
}

// CandleSeries is a provider answer with its data-source indicator
// (WM-INV-15): Fallback is set when the first provider in the chain failed.
type CandleSeries struct {
	Symbol    string    `json:"symbol"`
	Timeframe Timeframe `json:"timeframe"`
	Provider  string    `json:"provider"`
	Currency  string    `json:"currency,omitempty"`
	Candles   []Candle  `json:"candles"`
	Fallback  string    `json:"fallback,omitempty"`
}

// Closes returns the close prices in order.
func (s CandleSeries) Closes() []float64 {
	closes := make([]float64, len(s.Candles))
	for i, candle := range s.Candles {
		closes[i] = candle.Close
	}
	return closes
}

// NormaliseMarketSymbol is the canonical form used in cache keys and
// provider calls (WM-MKT-5): upper case, no "^" index prefix.
func NormaliseMarketSymbol(symbol string) string {
	return strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(symbol)), "^")
}

// bucketStart is the start of the timeframe bar holding local: 4h buckets
// start at 00/04/08/12/16/20, 1d at midnight and 1w on Monday.
func bucketStart(local time.Time, timeframe Timeframe) time.Time {
	day := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location())
	switch timeframe {
	case Timeframe4H:
		return day.Add(time.Duration(local.Hour()/4*4) * time.Hour)
	case Timeframe1W:
		return day.AddDate(0, 0, -((int(day.Weekday()) + 6) % 7))
	default:
		return day
	}
}
