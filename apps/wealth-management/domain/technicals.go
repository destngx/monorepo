package domain

import (
	"math"
	"slices"
	"time"
)

// Indicator periods (WM-INV-12).
const (
	RSIPeriod       = 14
	ATRPeriod       = 14
	BollingerPeriod = 20
	BollingerWidth  = 2.0
	StopLossATR     = 1.5
	TakeProfitATR   = 3.5
	// FlatThresholdPct is the ±0.05% band treated as no move (WM-RULE-9).
	FlatThresholdPct = 0.05
)

// SMA is the simple mean of the last period values.
func SMA(values []float64, period int) (float64, bool) {
	if period <= 0 || len(values) < period {
		return 0, false
	}
	return mean(values[len(values)-period:]), true
}

// EMA is the exponential moving average at the last value, seeded with the
// SMA of the first period values.
func EMA(values []float64, period int) (float64, bool) {
	if period <= 0 || len(values) < period {
		return 0, false
	}
	k := 2 / float64(period+1)
	ema := mean(values[:period])
	for _, value := range values[period:] {
		ema = value*k + ema*(1-k)
	}
	return ema, true
}

// RSI is Wilder's relative strength index at the last close.
func RSI(closes []float64, period int) (float64, bool) {
	if period <= 0 || len(closes) <= period {
		return 0, false
	}
	var gain, loss float64
	for i := 1; i <= period; i++ {
		change := closes[i] - closes[i-1]
		gain += max(change, 0)
		loss += max(-change, 0)
	}
	gain /= float64(period)
	loss /= float64(period)
	for i := period + 1; i < len(closes); i++ {
		change := closes[i] - closes[i-1]
		gain = (gain*float64(period-1) + max(change, 0)) / float64(period)
		loss = (loss*float64(period-1) + max(-change, 0)) / float64(period)
	}
	if loss == 0 {
		if gain == 0 {
			return 50, true
		}
		return 100, true
	}
	return 100 - 100/(1+gain/loss), true
}

// ATR is Wilder's average true range at the last bar.
func ATR(candles []Candle, period int) (float64, bool) {
	if period <= 0 || len(candles) <= period {
		return 0, false
	}
	trueRange := func(i int) float64 {
		previous := candles[i-1].Close
		bar := candles[i]
		return max(bar.High-bar.Low, math.Abs(bar.High-previous), math.Abs(bar.Low-previous))
	}
	var atr float64
	for i := 1; i <= period; i++ {
		atr += trueRange(i)
	}
	atr /= float64(period)
	for i := period + 1; i < len(candles); i++ {
		atr = (atr*float64(period-1) + trueRange(i)) / float64(period)
	}
	return atr, true
}

// Bollinger bands over the last period closes, population deviation.
type Bollinger struct {
	Upper  float64 `json:"upper"`
	Middle float64 `json:"middle"`
	Lower  float64 `json:"lower"`
}

func BollingerBands(closes []float64, period int, width float64) (Bollinger, bool) {
	middle, ok := SMA(closes, period)
	if !ok {
		return Bollinger{}, false
	}
	deviation := stdDev(closes[len(closes)-period:], middle)
	return Bollinger{Upper: middle + width*deviation, Middle: middle, Lower: middle - width*deviation}, true
}

// SupportResistance lists swing levels: support below the last close
// (nearest first) and resistance above it.
type SupportResistance struct {
	Support    []float64 `json:"support"`
	Resistance []float64 `json:"resistance"`
}

// swingWindow is how many bars on each side a swing high/low must beat.
const swingWindow = 2

// SwingLevels finds fractal swing highs and lows (a bar beating swingWindow
// bars on each side) and returns up to count levels on each side of the
// last close.
func SwingLevels(candles []Candle, count int) SupportResistance {
	levels := SupportResistance{Support: []float64{}, Resistance: []float64{}}
	if len(candles) == 0 {
		return levels
	}
	last := candles[len(candles)-1].Close
	for i := swingWindow; i < len(candles)-swingWindow; i++ {
		isHigh, isLow := true, true
		for j := i - swingWindow; j <= i+swingWindow; j++ {
			if j == i {
				continue
			}
			isHigh = isHigh && candles[i].High > candles[j].High
			isLow = isLow && candles[i].Low < candles[j].Low
		}
		if isHigh && candles[i].High > last {
			levels.Resistance = append(levels.Resistance, candles[i].High)
		}
		if isLow && candles[i].Low < last {
			levels.Support = append(levels.Support, candles[i].Low)
		}
	}
	slices.Sort(levels.Resistance)
	slices.Sort(levels.Support)
	slices.Reverse(levels.Support)
	levels.Resistance = firstN(slices.Compact(levels.Resistance), count)
	levels.Support = firstN(slices.Compact(levels.Support), count)
	return levels
}

func firstN[T any](values []T, n int) []T {
	return values[:min(n, len(values))]
}

// FairValueGap is a three-bar imbalance (ICT): bullish when a bar's low is
// above the high two bars earlier, bearish when its high is below that low.
type FairValueGap struct {
	Type   string  `json:"type"`
	Top    float64 `json:"top"`
	Bottom float64 `json:"bottom"`
	Time   string  `json:"time"`
}

const (
	FVGBullish = "BULLISH"
	FVGBearish = "BEARISH"
)

// OpenFairValueGaps returns up to count gaps not yet filled by a later bar,
// most recent first.
func OpenFairValueGaps(candles []Candle, count int) []FairValueGap {
	gaps := []FairValueGap{}
	for i := len(candles) - 1; i >= 2 && len(gaps) < count; i-- {
		var gap FairValueGap
		switch {
		case candles[i].Low > candles[i-2].High:
			gap = FairValueGap{Type: FVGBullish, Top: candles[i].Low, Bottom: candles[i-2].High}
		case candles[i].High < candles[i-2].Low:
			gap = FairValueGap{Type: FVGBearish, Top: candles[i-2].Low, Bottom: candles[i].High}
		default:
			continue
		}
		isFilled := false
		for _, later := range candles[i+1:] {
			if gap.Type == FVGBullish && later.Low <= gap.Bottom || gap.Type == FVGBearish && later.High >= gap.Top {
				isFilled = true
				break
			}
		}
		if !isFilled {
			gap.Time = candles[i].Time.Format(time.RFC3339)
			gaps = append(gaps, gap)
		}
	}
	return gaps
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}

// stdDev is the population standard deviation around avg.
func stdDev(values []float64, avg float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, value := range values {
		sum += (value - avg) * (value - avg)
	}
	return math.Sqrt(sum / float64(len(values)))
}
