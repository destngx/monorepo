package domain

import (
	"math"
)

// Trend directions.
const (
	TrendUp       = "up"
	TrendDown     = "down"
	TrendSideways = "sideways"
)

// Wyckoff phases (WM-RULE-9). Decline and Consolidation cover the states
// the rule leaves open.
const (
	PhaseMarkup        = "Markup"
	PhaseMarkdown      = "Mark-Down"
	PhaseDistribution  = "Distribution"
	PhaseAccumulation  = "Accumulation"
	PhaseDecline       = "Decline"
	PhaseConsolidation = "Consolidation"
	PhaseUnknown       = "Unknown"
)

// Signal actions.
const (
	ActionLong       = "LONG"
	ActionShort      = "SHORT"
	ActionTakeProfit = "TAKE PROFIT"
	ActionReduce     = "REDUCE"
	ActionExit       = "EXIT"
	ActionHoldWatch  = "HOLD/WATCH"
)

// Indicators are the last values; nil when the series is too short.
type Indicators struct {
	RSI       *float64   `json:"rsi"`
	EMA20     *float64   `json:"ema20"`
	EMA50     *float64   `json:"ema50"`
	EMA200    *float64   `json:"ema200"`
	ATR       *float64   `json:"atr"`
	Bollinger *Bollinger `json:"bollinger"`
}

// Signal is the rule-table action for the phase. Stop loss and take profit
// are set only for LONG / HOLD/WATCH (long side) and SHORT.
type Signal struct {
	Action     string   `json:"action"`
	Confidence int      `json:"confidence"`
	Reasons    []string `json:"reasons"`
	Entry      float64  `json:"entry"`
	StopLoss   *float64 `json:"stopLoss,omitempty"`
	TakeProfit *float64 `json:"takeProfit,omitempty"`
	RiskReward *float64 `json:"riskReward,omitempty"`
}

// Technicals is the WM-INV-12 snapshot of one series. Phase, signal and
// confidence come from fixed rules, not a fitted model: they are heuristics.
type Technicals struct {
	Price             float64           `json:"price"`
	Candles           int               `json:"candles"`
	Indicators        Indicators        `json:"indicators"`
	Trend             string            `json:"trend"`
	Phase             string            `json:"phase"`
	Signal            Signal            `json:"signal"`
	SupportResistance SupportResistance `json:"supportResistance"`
	FairValueGaps     []FairValueGap    `json:"fairValueGaps"`
	Notes             []string          `json:"notes"`
}

// ComputeTechnicals runs every WM-INV-12 indicator and rule on candles
// (oldest first, cleaned).
func ComputeTechnicals(candles []Candle) Technicals {
	result := Technicals{Candles: len(candles), FairValueGaps: []FairValueGap{}, Notes: []string{}, Phase: PhaseUnknown, Trend: TrendSideways}
	result.SupportResistance = SupportResistance{Support: []float64{}, Resistance: []float64{}}
	if len(candles) == 0 {
		result.Notes = append(result.Notes, "no candles")
		return result
	}
	closes := make([]float64, len(candles))
	for i, candle := range candles {
		closes[i] = candle.Close
	}
	price := closes[len(closes)-1]
	result.Price = price

	optional := func(value float64, ok bool) *float64 {
		if !ok {
			return nil
		}
		return &value
	}
	indicators := Indicators{
		RSI:    optional(RSI(closes, RSIPeriod)),
		EMA20:  optional(EMA(closes, 20)),
		EMA50:  optional(EMA(closes, 50)),
		EMA200: optional(EMA(closes, 200)),
		ATR:    optional(ATR(candles, ATRPeriod)),
	}
	if bands, ok := BollingerBands(closes, BollingerPeriod, BollingerWidth); ok {
		indicators.Bollinger = &bands
	}
	result.Indicators = indicators
	result.Trend = trendOf(price, indicators)
	result.Phase = wyckoffPhase(price, indicators)
	if indicators.EMA200 == nil {
		result.Notes = append(result.Notes, "fewer than 200 candles: EMA200 is missing, so the phase uses EMA20/EMA50 only")
	}
	result.Signal = signalFor(result.Phase, price, indicators)
	result.SupportResistance = SwingLevels(candles, 3)
	result.FairValueGaps = OpenFairValueGaps(candles, 3)
	return result
}

func trendOf(price float64, indicators Indicators) string {
	if indicators.EMA20 == nil || indicators.EMA50 == nil {
		return TrendSideways
	}
	ema20, ema50 := *indicators.EMA20, *indicators.EMA50
	switch {
	case price > ema20 && ema20 > ema50:
		return TrendUp
	case price < ema20 && ema20 < ema50:
		return TrendDown
	default:
		return TrendSideways
	}
}

// wyckoffPhase applies WM-RULE-9 in order. A missing EMA200 is left out of
// the "above/below all" checks and disables Accumulation.
func wyckoffPhase(price float64, indicators Indicators) string {
	if indicators.EMA20 == nil || indicators.EMA50 == nil || indicators.RSI == nil {
		return PhaseUnknown
	}
	isAbove20, isAbove50 := price > *indicators.EMA20, price > *indicators.EMA50
	hasEMA200 := indicators.EMA200 != nil
	isAbove200 := hasEMA200 && price > *indicators.EMA200
	isBelow200 := hasEMA200 && price < *indicators.EMA200
	rsi := *indicators.RSI
	switch {
	case isAbove20 && isAbove50 && (isAbove200 || !hasEMA200):
		return PhaseMarkup
	case !isAbove20 && !isAbove50 && (isBelow200 || !hasEMA200):
		return PhaseMarkdown
	case isAbove50 && rsi > 70:
		return PhaseDistribution
	case isBelow200 && rsi < 30:
		return PhaseAccumulation
	case isAbove50 && !isAbove20:
		return PhaseDecline
	default:
		return PhaseConsolidation
	}
}

// signalFor is the phase/RSI action table carried over from the legacy app.
func signalFor(phase string, price float64, indicators Indicators) Signal {
	rsi := 50.0
	if indicators.RSI != nil {
		rsi = *indicators.RSI
	}
	signal := Signal{Action: ActionHoldWatch, Confidence: 50, Entry: price, Reasons: []string{"no clear phase"}}
	switch phase {
	case PhaseMarkup:
		if rsi < 65 {
			signal = Signal{Action: ActionLong, Confidence: 85, Reasons: []string{"price above EMA20/50/200", "RSI below 65 leaves room to run"}}
		} else {
			signal = Signal{Action: ActionTakeProfit, Confidence: 75, Reasons: []string{"extended markup", "RSI 65 or higher"}}
		}
	case PhaseMarkdown:
		if rsi > 35 {
			signal = Signal{Action: ActionShort, Confidence: 80, Reasons: []string{"price below EMA20/50/200", "RSI not yet oversold"}}
		} else {
			signal = Signal{Action: ActionExit, Confidence: 90, Reasons: []string{"extreme bearish momentum", "RSI 35 or lower: oversold bounce likely"}}
		}
	case PhaseDistribution:
		if rsi > 60 {
			signal = Signal{Action: ActionReduce, Confidence: 70, Reasons: []string{"above EMA50 with RSI over 70", "de-risk into strength"}}
		} else {
			signal = Signal{Action: ActionExit, Confidence: 65, Reasons: []string{"distribution losing momentum"}}
		}
	case PhaseAccumulation:
		if rsi < 30 {
			signal = Signal{Action: ActionLong, Confidence: 60, Reasons: []string{"below EMA200 with RSI under 30", "possible supply exhaustion"}}
		} else {
			signal = Signal{Action: ActionHoldWatch, Confidence: 55, Reasons: []string{"range: wait for a breakout"}}
		}
	case PhaseDecline:
		signal = Signal{Action: ActionHoldWatch, Confidence: 50, Reasons: []string{"lost EMA20 while above EMA50: momentum slowing"}}
	case PhaseConsolidation:
		signal = Signal{Action: ActionHoldWatch, Confidence: 50, Reasons: []string{"mixed EMA alignment: no trend"}}
	}
	signal.Entry = price
	if indicators.ATR == nil {
		return signal
	}
	atr := *indicators.ATR
	var stop, target float64
	switch signal.Action {
	case ActionLong, ActionHoldWatch:
		stop, target = price-StopLossATR*atr, price+TakeProfitATR*atr
	case ActionShort:
		stop, target = price+StopLossATR*atr, price-TakeProfitATR*atr
	default:
		return signal
	}
	ratio := TakeProfitATR / StopLossATR
	signal.StopLoss, signal.TakeProfit, signal.RiskReward = &stop, &target, &ratio
	return signal
}

// Direction is up/down/flat with the ±0.05% band (WM-RULE-9).
func Direction(changePct float64) string {
	switch {
	case changePct > FlatThresholdPct:
		return TrendUp
	case changePct < -FlatThresholdPct:
		return TrendDown
	default:
		return "flat"
	}
}

// Sentiment is the WM-RULE-9 fear/greed gauge from VIX and RSI.
type Sentiment struct {
	Score int    `json:"score"`
	Label string `json:"label"`
}

func SentimentScore(vix float64, rsi float64) Sentiment {
	score := int(math.Round(math.Max(0, math.Min(100, 100-vix*2.5+(rsi-50)))))
	label := "Neutral"
	if score > 65 {
		label = "Greed"
	} else if score < 35 {
		label = "Fear"
	}
	return Sentiment{Score: score, Label: label}
}
