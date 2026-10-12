package domain

import (
	"math"
	"slices"
	"time"
)

// Alignment between two adjacent timeframes.
const (
	AlignmentAligned  = "ALIGNED"
	AlignmentConflict = "CONFLICT"
	AlignmentMixed    = "MIXED"
)

// TimeframeView is one row of the WM-INV-13 confluence matrix.
type TimeframeView struct {
	Timeframe Timeframe `json:"timeframe"`
	Provider  string    `json:"provider,omitempty"`
	Candles   int       `json:"candles"`
	Price     float64   `json:"price"`
	Trend     string    `json:"trend"`
	Phase     string    `json:"phase"`
	RSI       *float64  `json:"rsi"`
	Error     string    `json:"error,omitempty"`
}

type TimeframePair struct {
	Higher Timeframe `json:"higher"`
	Lower  Timeframe `json:"lower"`
	Status string    `json:"status"`
}

// Confluence weighs each timeframe's trend (1w and 1d count 3, 4h and 1h
// count 2). Score 0-10 is how strongly the available timeframes agree on
// Bias.
type Confluence struct {
	Views []TimeframeView `json:"views"`
	Pairs []TimeframePair `json:"pairs"`
	Bias  string          `json:"bias"`
	Score int             `json:"score"`
}

var confluenceWeights = map[Timeframe]float64{Timeframe1W: 3, Timeframe1D: 3, Timeframe4H: 2, Timeframe1H: 2}

// BuildConfluence orders views from the longest timeframe down.
func BuildConfluence(views map[Timeframe]TimeframeView) Confluence {
	result := Confluence{Views: []TimeframeView{}, Pairs: []TimeframePair{}, Bias: TrendSideways}
	ordered := slices.Clone(Timeframes)
	slices.Reverse(ordered)
	var sum, weight float64
	for _, timeframe := range ordered {
		view, ok := views[timeframe]
		if !ok {
			continue
		}
		result.Views = append(result.Views, view)
		if view.Error != "" {
			continue
		}
		weight += confluenceWeights[timeframe]
		switch view.Trend {
		case TrendUp:
			sum += confluenceWeights[timeframe]
		case TrendDown:
			sum -= confluenceWeights[timeframe]
		}
	}
	for i := 1; i < len(result.Views); i++ {
		higher, lower := result.Views[i-1], result.Views[i]
		if higher.Error != "" || lower.Error != "" {
			continue
		}
		result.Pairs = append(result.Pairs, TimeframePair{Higher: higher.Timeframe, Lower: lower.Timeframe, Status: alignment(higher.Trend, lower.Trend)})
	}
	if weight > 0 {
		result.Score = int(math.Round(10 * math.Abs(sum) / weight))
		switch {
		case sum > 0:
			result.Bias = TrendUp
		case sum < 0:
			result.Bias = TrendDown
		}
	}
	return result
}

func alignment(higher, lower string) string {
	switch {
	case higher == lower && higher != TrendSideways:
		return AlignmentAligned
	case higher != TrendSideways && lower != TrendSideways:
		return AlignmentConflict
	default:
		return AlignmentMixed
	}
}

// MarketAnalysis is the WM-INV-12/13/14/16 ticker deep-dive. Technicals
// are for the requested timeframe; seasonality and risk use daily candles.
type MarketAnalysis struct {
	Symbol      string              `json:"symbol"`
	Timeframe   Timeframe           `json:"timeframe"`
	Provider    string              `json:"provider"`
	Fallback    string              `json:"fallback,omitempty"`
	Currency    string              `json:"currency,omitempty"`
	Technicals  Technicals          `json:"technicals"`
	Confluence  Confluence          `json:"confluence"`
	Seasonality []SeasonalityBucket `json:"seasonality"`
	Risk        *RiskStats          `json:"risk"`
	Notes       []string            `json:"notes"`
	GeneratedAt time.Time           `json:"generatedAt"`
}
