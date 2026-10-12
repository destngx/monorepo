package domain

import (
	"fmt"
	"math"
	"slices"
	"time"
)

// Market symbols outside the VN equity board, resolved by each provider.
const (
	MarketSymbolVNIndex = "VNINDEX"
	MarketSymbolVN30    = "VN30"
	MarketSymbolUSDVND  = "USDVND"
	MarketSymbolSJC     = "SJC"
	MarketSymbolXAUUSD  = "XAUUSD"
	MarketSymbolVIX     = "VIX"
)

// IsVNListing reports whether symbol is a HOSE/HNX/UPCoM ticker or index,
// as opposed to an FX pair, gold or a global index.
func IsVNListing(symbol string) bool {
	switch NormaliseMarketSymbol(symbol) {
	case MarketSymbolUSDVND, MarketSymbolSJC, MarketSymbolXAUUSD, MarketSymbolVIX:
		return false
	}
	return true
}

// PulseAsset is one WM-INV-11 instrument.
type PulseAsset struct {
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
}

// PulseAssets are the WM-INV-11 instruments. VIX is the global risk gauge
// the capital-flow and sentiment rules read (WM-RULE-9).
var PulseAssets = []PulseAsset{
	{MarketSymbolVNIndex, "VN-Index"},
	{MarketSymbolVN30, "VN30"},
	{"VCB", "Vietcombank"},
	{"VNM", "Vinamilk"},
	{"ACB", "ACB"},
	{"TCB", "Techcombank"},
	{"MBB", "MB Bank"},
	{"FPT", "FPT"},
	{"HPG", "Hoa Phat"},
	{"SAB", "Sabeco"},
	{"GAS", "PV Gas"},
	{MarketSymbolUSDVND, "USD/VND"},
	{MarketSymbolSJC, "SJC gold"},
	{MarketSymbolVIX, "VIX"},
}

// Momentum labels (WM-RULE-9).
const (
	MomentumFire   = "fire"
	MomentumStable = "stable"
	MomentumSleep  = "sleep"
)

// PulseQuote is an instrument's move over the pulse timeframe. Change is
// against the previous bar; day/week changes compare with the last bar at
// least 24 h / 7 days older and are nil when the history is too short.
type PulseQuote struct {
	PulseAsset
	Provider      string   `json:"provider,omitempty"`
	Fallback      string   `json:"fallback,omitempty"`
	Currency      string   `json:"currency,omitempty"`
	Price         float64  `json:"price"`
	Time          string   `json:"time,omitempty"`
	ChangePct     float64  `json:"changePct"`
	DayChangePct  *float64 `json:"dayChangePct"`
	WeekChangePct *float64 `json:"weekChangePct"`
	Direction     string   `json:"direction"`
	Momentum      string   `json:"momentum"`
	Candles       int      `json:"candles"`
	Error         string   `json:"error,omitempty"`
}

// QuotePulse computes the quote from a cleaned series.
func QuotePulse(asset PulseAsset, series CandleSeries) PulseQuote {
	quote := PulseQuote{PulseAsset: asset, Provider: series.Provider, Fallback: series.Fallback, Currency: series.Currency, Candles: len(series.Candles)}
	candles := series.Candles
	if len(candles) < 2 {
		quote.Error = "not enough candles"
		return quote
	}
	last := candles[len(candles)-1]
	quote.Price = last.Close
	quote.Time = last.Time.Format(time.RFC3339)
	quote.ChangePct = percentChange(candles[len(candles)-2].Close, last.Close)
	if series.Timeframe != Timeframe1W {
		quote.DayChangePct = changeSince(candles, last.Time.Add(-24*time.Hour))
	}
	quote.WeekChangePct = changeSince(candles, last.Time.AddDate(0, 0, -7))
	quote.Direction = Direction(quote.ChangePct)
	dayChange := 0.0
	if quote.DayChangePct != nil {
		dayChange = *quote.DayChangePct
	}
	switch {
	case math.Abs(quote.ChangePct) > 2 || quote.Direction == TrendUp && dayChange > 5:
		quote.Momentum = MomentumFire
	case math.Abs(quote.ChangePct) < 0.1:
		quote.Momentum = MomentumSleep
	default:
		quote.Momentum = MomentumStable
	}
	return quote
}

// changeSince compares the last close with the last bar at or before cutoff.
func changeSince(candles []Candle, cutoff time.Time) *float64 {
	last := candles[len(candles)-1].Close
	for i := len(candles) - 2; i >= 0; i-- {
		if !candles[i].Time.After(cutoff) {
			change := percentChange(candles[i].Close, last)
			return &change
		}
	}
	return nil
}

func percentChange(from, to float64) float64 {
	if from == 0 {
		return 0
	}
	return (to/from - 1) * 100
}

// CorrelationMatrix holds Pearson correlations of bar returns aligned on
// bar time; nil where two series share fewer than minCorrelationPoints.
type CorrelationMatrix struct {
	Symbols []string     `json:"symbols"`
	Values  [][]*float64 `json:"values"`
}

const minCorrelationPoints = 5

// Correlate aligns each pair of series on their bar start in location for
// the timeframe, so daily bars from different exchanges meet on the date.
func Correlate(series []CandleSeries, timeframe Timeframe, location *time.Location) CorrelationMatrix {
	returns := make([]map[int64]float64, len(series))
	matrix := CorrelationMatrix{Symbols: make([]string, len(series)), Values: make([][]*float64, len(series))}
	for i, s := range series {
		matrix.Symbols[i] = s.Symbol
		returns[i] = map[int64]float64{}
		for j := 1; j < len(s.Candles); j++ {
			key := bucketStart(s.Candles[j].Time.In(location), timeframe).Unix()
			returns[i][key] = s.Candles[j].Close/s.Candles[j-1].Close - 1
		}
	}
	for i := range series {
		matrix.Values[i] = make([]*float64, len(series))
		for j := range series {
			if i == j {
				one := 1.0
				matrix.Values[i][j] = &one
				continue
			}
			if j < i {
				matrix.Values[i][j] = matrix.Values[j][i]
				continue
			}
			var x, y []float64
			for key, value := range returns[i] {
				if other, ok := returns[j][key]; ok {
					x, y = append(x, value), append(y, other)
				}
			}
			if len(x) < minCorrelationPoints {
				continue
			}
			if r, ok := PearsonCorrelation(x, y); ok {
				rounded := math.Round(r*1000) / 1000
				matrix.Values[i][j] = &rounded
			}
		}
	}
	return matrix
}

// Capital flow signals (WM-RULE-9).
const (
	FlowDefensive = "DEFENSIVE"
	FlowRiskOn    = "RISK-ON"
	FlowMixed     = "MIXED"
)

type CapitalFlow struct {
	Signal  string `json:"signal"`
	Summary string `json:"summary"`
}

// Scenario is the rule-based market regime read.
type Scenario struct {
	Name    string `json:"name"`
	Regime  string `json:"regime"`
	Summary string `json:"summary"`
	Action  string `json:"action"`
}

// MarketPulse is the WM-INV-11 answer.
type MarketPulse struct {
	Timeframe   Timeframe         `json:"timeframe"`
	Assets      []PulseQuote      `json:"assets"`
	TopMovers   []PulseQuote      `json:"topMovers"`
	Correlation CorrelationMatrix `json:"correlation"`
	CapitalFlow CapitalFlow       `json:"capitalFlow"`
	Scenario    Scenario          `json:"scenario"`
	Index       *Technicals       `json:"index"`
	Sentiment   *Sentiment        `json:"sentiment"`
	Notes       []string          `json:"notes"`
	GeneratedAt time.Time         `json:"generatedAt"`
}

// BuildPulse assembles the pulse from fetched series; failures holds the
// error per symbol for instruments that could not be fetched.
func BuildPulse(timeframe Timeframe, series map[string]CandleSeries, failures map[string]string, location *time.Location, now time.Time) MarketPulse {
	pulse := MarketPulse{Timeframe: timeframe, Assets: []PulseQuote{}, TopMovers: []PulseQuote{}, Notes: []string{}, GeneratedAt: now}
	quotes := map[string]PulseQuote{}
	correlated := []CandleSeries{}
	for _, asset := range PulseAssets {
		s, ok := series[asset.Symbol]
		if !ok {
			pulse.Assets = append(pulse.Assets, PulseQuote{PulseAsset: asset, Error: failures[asset.Symbol]})
			continue
		}
		quote := QuotePulse(asset, s)
		pulse.Assets = append(pulse.Assets, quote)
		if quote.Error == "" {
			quotes[asset.Symbol] = quote
			correlated = append(correlated, s)
		}
	}

	for _, quote := range pulse.Assets {
		if quote.Error == "" && quote.Symbol != MarketSymbolVIX {
			pulse.TopMovers = append(pulse.TopMovers, quote)
		}
	}
	slices.SortStableFunc(pulse.TopMovers, func(a, b PulseQuote) int {
		return compareDesc(math.Abs(a.ChangePct), math.Abs(b.ChangePct))
	})
	pulse.TopMovers = firstN(pulse.TopMovers, 5)
	pulse.Correlation = Correlate(correlated, timeframe, location)

	index, hasIndex := quotes[MarketSymbolVNIndex]
	gold, hasGold := quotes[MarketSymbolSJC]
	vix, hasVIX := quotes[MarketSymbolVIX]
	if !hasVIX {
		pulse.Notes = append(pulse.Notes, "VIX unavailable: capital flow ignores it and sentiment is not computed")
	}
	pulse.CapitalFlow = capitalFlow(index, hasIndex, gold, hasGold, vix, hasVIX)
	vn30, hasVN30 := quotes[MarketSymbolVN30]
	if !hasVN30 {
		pulse.Notes = append(pulse.Notes, "VN30 unavailable: the falling-knife check is skipped")
	}
	pulse.Scenario = scenario(index, hasIndex, vn30, hasVN30, vix, hasVIX)

	if s, ok := series[MarketSymbolVNIndex]; ok && len(s.Candles) > 0 {
		technicals := ComputeTechnicals(s.Candles)
		pulse.Index = &technicals
		if hasVIX && technicals.Indicators.RSI != nil {
			sentiment := SentimentScore(vix.Price, *technicals.Indicators.RSI)
			pulse.Sentiment = &sentiment
		}
	}
	return pulse
}

func capitalFlow(index PulseQuote, hasIndex bool, gold PulseQuote, hasGold bool, vix PulseQuote, hasVIX bool) CapitalFlow {
	isIndexDown := hasIndex && index.ChangePct < 0
	isVIXJump := hasVIX && vix.ChangePct > 2
	if hasGold && gold.ChangePct > 0.5 && (isIndexDown || isVIXJump) {
		return CapitalFlow{FlowDefensive, fmt.Sprintf("Defensive: SJC gold %+.2f%% while equities are under pressure.", gold.ChangePct)}
	}
	if hasIndex && index.ChangePct > 0.5 && hasVIX && vix.ChangePct < 0 {
		return CapitalFlow{FlowRiskOn, fmt.Sprintf("Risk-on: VN-Index %+.2f%% with VIX easing.", index.ChangePct)}
	}
	return CapitalFlow{FlowMixed, "Mixed signals: no clear direction in capital flow."}
}

func scenario(index PulseQuote, hasIndex bool, vn30 PulseQuote, hasVN30 bool, vix PulseQuote, hasVIX bool) Scenario {
	switch {
	case hasVN30 && vn30.ChangePct < -1.5:
		return Scenario{"Falling knife", "Crisis", fmt.Sprintf("VN30 %+.2f%%: heavy selling; oversold readings are unreliable.", vn30.ChangePct), "Do not average down; wait for accumulation to confirm a bottom."}
	case hasVIX && vix.Price > 30:
		return Scenario{"Crisis mode", "Crisis", fmt.Sprintf("VIX %.1f: extreme global fear.", vix.Price), "Favour cash and defensive assets."}
	case hasVIX && vix.Price < 15 && hasIndex && index.ChangePct > 0:
		return Scenario{"Normal growth", "Risk-ON", fmt.Sprintf("VIX %.1f and VN-Index rising.", vix.Price), "Keep exposure to strong-momentum names."}
	default:
		return Scenario{"Neutral / mixed", "Risk-OFF", "Transitional or uncertain market.", "Hold positions; wait for a clear breakout."}
	}
}
