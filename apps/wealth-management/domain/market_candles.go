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

// SourceTimeframe is the resolution fetched from providers: 4h candles are
// built from 1h candles because neither vnstock nor Yahoo serve them.
func (t Timeframe) SourceTimeframe() Timeframe {
	if t == Timeframe4H {
		return Timeframe1H
	}
	return t
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

// CandleQuery asks a provider for bars of one source timeframe (1h, 1d or
// 1w) between From and To inclusive.
type CandleQuery struct {
	Timeframe Timeframe
	From      time.Time
	To        time.Time
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

// CleanCandles sorts bars by time, drops bars without a positive close and
// duplicates, and fills a missing high/low/open from the close.
func CleanCandles(candles []Candle) []Candle {
	cleaned := make([]Candle, 0, len(candles))
	for _, candle := range candles {
		if candle.Close <= 0 || candle.Time.IsZero() {
			continue
		}
		if candle.Open <= 0 {
			candle.Open = candle.Close
		}
		if candle.High <= 0 {
			candle.High = max(candle.Open, candle.Close)
		}
		if candle.Low <= 0 {
			candle.Low = min(candle.Open, candle.Close)
		}
		cleaned = append(cleaned, candle)
	}
	sortCandles(cleaned)
	unique := cleaned[:0]
	for i, candle := range cleaned {
		if i > 0 && candle.Time.Equal(cleaned[i-1].Time) {
			unique[len(unique)-1] = candle
			continue
		}
		unique = append(unique, candle)
	}
	return unique
}

func sortCandles(candles []Candle) {
	for i := 1; i < len(candles); i++ {
		for j := i; j > 0 && candles[j].Time.Before(candles[j-1].Time); j-- {
			candles[j], candles[j-1] = candles[j-1], candles[j]
		}
	}
}

// AggregateCandles merges shorter bars into timeframe bars aligned in
// location: 4h buckets start at 00/04/08/12/16/20 (one morning and one
// afternoon bar on HOSE), 1d at midnight and 1w on Monday. 1h input is
// returned unchanged.
func AggregateCandles(candles []Candle, timeframe Timeframe, location *time.Location) []Candle {
	if timeframe == Timeframe1H {
		return candles
	}
	result := []Candle{}
	var bucket time.Time
	for _, candle := range candles {
		start := bucketStart(candle.Time.In(location), timeframe)
		if len(result) == 0 || !start.Equal(bucket) {
			bucket = start
			result = append(result, Candle{Time: start, Open: candle.Open, High: candle.High, Low: candle.Low, Close: candle.Close, Volume: candle.Volume})
			continue
		}
		last := &result[len(result)-1]
		last.High = max(last.High, candle.High)
		last.Low = min(last.Low, candle.Low)
		last.Close = candle.Close
		last.Volume += candle.Volume
	}
	return result
}

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
