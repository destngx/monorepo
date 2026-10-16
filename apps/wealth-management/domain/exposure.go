package domain

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"sort"
	"time"
)

// My exposure (ROADMAP M1, M2; WM-INV-24..26). The Investments page opens
// with what the user holds, read from the one allocation (WM-INV-20):
//
//   - volatile classes are fund certificates and crypto, the classes whose
//     value follows a market price; cash and deposits and money lent are
//     fixed VND amounts. The volatile share is their value over the whole
//     allocation (money lent included);
//   - today's change applies each holding's latest market move to its
//     current VND value: a coin's change over about 24 hours (hourly bars)
//     and a fund's NAV change since its previous published NAV. Only valued
//     holdings with a move count; Covered says how much of a class that is,
//     since a class may be valued from its accounts (WM-RULE-8);
//   - the crypto view lists the held coins and a drawdown scenario: if
//     crypto falls by a fixed percent, net worth (the balance sheet,
//     WM-RULE-1/2) falls by the crypto value times that percent. No
//     forecasting: the scenarios are a fixed set.

// VolatileClasses are the allocation classes whose value moves with markets.
var VolatileClasses = []AllocationClass{AllocationFunds, AllocationCrypto}

// CryptoDrawdowns are the fixed crypto falls of the scenario, in percent.
var CryptoDrawdowns = []float64{30, 50}

// Move bases say what a PriceMove compares.
const (
	MoveBasisDay     = "24h"
	MoveBasisLastNAV = "previous_nav"
)

// cryptoDayWindow is how far back the crypto move looks; cryptoDayTolerance
// is how far the reference bar may sit before that point (a gap in bars).
const (
	cryptoDayWindow    = 24 * time.Hour
	cryptoDayTolerance = 3 * time.Hour
)

// PriceMove is an instrument's latest price against an earlier one.
type PriceMove struct {
	Basis     string      `json:"basis"`
	Price     float64     `json:"price"`
	Previous  float64     `json:"previous"`
	ChangePct float64     `json:"changePct"`
	AsOf      string      `json:"asOf"`
	Since     string      `json:"since"`
	Source    *DataSource `json:"source,omitempty"`
}

// errNoMove explains a series too short for a move.
var errNoMove = errors.New("not enough prices for a change")

// CryptoDayMove is a coin's change over about 24 hours from hourly bars:
// the latest close against the last close at or before 24 hours earlier.
func CryptoDayMove(series CandleSeries) (PriceMove, error) {
	candles := pricedCandles(series.Candles)
	if len(candles) < 2 {
		return PriceMove{}, errNoMove
	}
	last := candles[len(candles)-1]
	cutoff := last.Time.Add(-cryptoDayWindow)
	reference := -1
	for i, candle := range candles[:len(candles)-1] {
		if !candle.Time.After(cutoff) {
			reference = i
		}
	}
	if reference < 0 || cutoff.Sub(candles[reference].Time) > cryptoDayTolerance {
		return PriceMove{}, fmt.Errorf("%w: no price about 24 hours back", errNoMove)
	}
	previous := candles[reference]
	return PriceMove{
		Basis: MoveBasisDay, Price: last.Close, Previous: previous.Close,
		ChangePct: round2((last.Close - previous.Close) / previous.Close * 100),
		AsOf:      last.Time.UTC().Format(time.RFC3339), Since: previous.Time.UTC().Format(time.RFC3339),
		Source: seriesSource(series),
	}, nil
}

// FundNAVMove is a fund's latest NAV against the previous published one.
func FundNAVMove(points []NavPoint) (PriceMove, error) {
	byDate := map[string]float64{}
	for _, point := range points {
		if point.NAV > 0 && point.Date != "" {
			byDate[point.Date] = point.NAV
		}
	}
	dates := make([]string, 0, len(byDate))
	for date := range byDate {
		dates = append(dates, date)
	}
	if len(dates) < 2 {
		return PriceMove{}, errNoMove
	}
	sort.Strings(dates)
	latest, previous := dates[len(dates)-1], dates[len(dates)-2]
	return PriceMove{
		Basis: MoveBasisLastNAV, Price: byDate[latest], Previous: byDate[previous],
		ChangePct: round2((byDate[latest] - byDate[previous]) / byDate[previous] * 100),
		AsOf:      latest, Since: previous,
	}, nil
}

// pricedCandles drops bars without a price and sorts them by time.
func pricedCandles(candles []Candle) []Candle {
	out := make([]Candle, 0, len(candles))
	for _, candle := range candles {
		if candle.Close > 0 && !candle.Time.IsZero() {
			out = append(out, candle)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	return out
}

func seriesSource(series CandleSeries) *DataSource {
	if series.Source != nil {
		return series.Source
	}
	if series.Provider == "" {
		return nil
	}
	return &DataSource{Provider: series.Provider}
}

// moveChange is the VND change behind value after a move of pct percent:
// value - value / (1 + pct/100).
func moveChange(value float64, pct float64) float64 {
	return value - value/(1+pct/100)
}

// ExposureClass is one allocation line with today's change.
type ExposureClass struct {
	Class    AllocationClass `json:"class"`
	Label    string          `json:"label"`
	Value    float64         `json:"value"`
	Share    float64         `json:"share"`
	Volatile bool            `json:"volatile"`
	// Change is today's move in VND of the holdings in Covered; nil for a
	// fixed class or when no holding has a move.
	Change    *float64 `json:"change"`
	ChangePct *float64 `json:"changePct"`
	Covered   float64  `json:"covered"`
}

// HeldPosition is a coin or fund code summed over every platform.
type HeldPosition struct {
	Symbol   string     `json:"symbol"`
	Quantity *float64   `json:"quantity"`
	Price    *LivePrice `json:"price,omitempty"`
	// Value is the VND value of the valued holdings; Unvalued counts the rest.
	Value    float64    `json:"value"`
	Unvalued int        `json:"unvalued"`
	Share    float64    `json:"share"`
	Move     *PriceMove `json:"move,omitempty"`
	Change   *float64   `json:"change"`
	Issue    string     `json:"issue,omitempty"`
}

// DrawdownScenario is net worth after crypto falls by FallPct.
type DrawdownScenario struct {
	FallPct       float64 `json:"fallPct"`
	Loss          float64 `json:"loss"`
	NetWorthAfter float64 `json:"netWorthAfter"`
	// NetWorthChangePct is -Loss over net worth; nil when net worth is not positive.
	NetWorthChangePct *float64 `json:"netWorthChangePct"`
}

// CryptoExposure is the crypto view (ROADMAP M2). Value is the allocation's
// crypto class; Holdings and Accounts are the two figures it is the larger of.
type CryptoExposure struct {
	Value           float64            `json:"value"`
	Holdings        float64            `json:"holdings"`
	Accounts        float64            `json:"accounts"`
	Share           float64            `json:"share"`
	ShareOfNetWorth *float64           `json:"shareOfNetWorth"`
	Coins           []HeldPosition     `json:"coins"`
	Change          *float64           `json:"change"`
	Scenarios       []DrawdownScenario `json:"scenarios"`
}

// Exposure is the Investments headline (ROADMAP M1, M2).
type Exposure struct {
	Total         float64         `json:"total"`
	NetWorth      float64         `json:"netWorth"`
	VolatileValue float64         `json:"volatileValue"`
	VolatileShare float64         `json:"volatileShare"`
	Change        *float64        `json:"change"`
	ChangePct     *float64        `json:"changePct"`
	Covered       float64         `json:"covered"`
	Classes       []ExposureClass `json:"classes"`
	Crypto        CryptoExposure  `json:"crypto"`
	Funds         []HeldPosition  `json:"funds"`
	Issues        []string        `json:"issues"`
	GeneratedAt   time.Time       `json:"generatedAt"`
}

// ExposureInputs are the figures exposure is built from. CryptoMoves are
// keyed by coin symbol and FundMoves by fund code (upper case); MoveIssues
// say why a symbol has no move, by the same keys.
type ExposureInputs struct {
	Allocation   Allocation
	Valuation    PortfolioValuation
	BalanceSheet BalanceSheet
	CryptoMoves  map[string]PriceMove
	FundMoves    map[string]PriceMove
	MoveIssues   map[string]string
	Now          time.Time
}

// ComputeExposure builds the exposure from the allocation, the valuation,
// the balance sheet and the market moves.
func ComputeExposure(in ExposureInputs) Exposure {
	result := Exposure{
		Total: in.Allocation.Total, NetWorth: in.BalanceSheet.NetWorth,
		Classes: []ExposureClass{}, Funds: []HeldPosition{}, Issues: []string{}, GeneratedAt: in.Now,
	}
	coins := heldPositions(in.Valuation.Holdings, PortfolioSectionCrypto, in.CryptoMoves, in.MoveIssues)
	funds := heldPositions(in.Valuation.Holdings, PortfolioSectionFunds, in.FundMoves, in.MoveIssues)
	moved := map[AllocationClass][]HeldPosition{AllocationCrypto: coins, AllocationFunds: funds}

	var change, covered float64
	anyChange := false
	for _, line := range in.Allocation.Lines {
		class := ExposureClass{Class: line.Class, Label: line.Label, Value: line.Value, Share: line.Share, Volatile: slices.Contains(VolatileClasses, line.Class)}
		if class.Volatile {
			result.VolatileValue += line.Value
			if classChange, classCovered, ok := sumChange(moved[line.Class]); ok {
				class.Change, class.Covered = &classChange, classCovered
				class.ChangePct = changePercent(classChange, classCovered)
				change += classChange
				covered += classCovered
				anyChange = true
			}
		}
		result.Classes = append(result.Classes, class)
	}
	if result.Total > 0 {
		result.VolatileShare = round2(result.VolatileValue / result.Total * 100)
	}
	if anyChange {
		result.Change, result.Covered = &change, covered
		result.ChangePct = changePercent(change, covered)
	}

	crypto := in.Allocation.Line(AllocationCrypto)
	result.Crypto = CryptoExposure{
		Value: crypto.Value, Holdings: in.Valuation.Totals.CryptoHoldings, Accounts: in.Valuation.Totals.CryptoAccounts,
		Share: crypto.Share, Coins: shareOf(coins), Scenarios: drawdowns(crypto.Value, result.NetWorth),
	}
	if result.NetWorth > 0 {
		share := round2(crypto.Value / result.NetWorth * 100)
		result.Crypto.ShareOfNetWorth = &share
	}
	if class := result.class(AllocationCrypto); class != nil {
		result.Crypto.Change = class.Change
	}
	result.Funds = shareOf(funds)

	keys := make([]string, 0, len(in.MoveIssues))
	for key := range in.MoveIssues {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		result.Issues = append(result.Issues, fmt.Sprintf("%s: no market change (%s)", key, in.MoveIssues[key]))
	}
	return result
}

func (e Exposure) class(class AllocationClass) *ExposureClass {
	for i := range e.Classes {
		if e.Classes[i].Class == class {
			return &e.Classes[i]
		}
	}
	return nil
}

// heldPositions sums a section's holdings per symbol, largest value first.
func heldPositions(holdings []ValuedHolding, section string, moves map[string]PriceMove, issues map[string]string) []HeldPosition {
	positions := []HeldPosition{}
	index := map[string]int{}
	for _, holding := range holdings {
		symbol := FundKey(holding.Symbol)
		if holding.Section != section || symbol == "" {
			continue
		}
		i, ok := index[symbol]
		if !ok {
			i = len(positions)
			index[symbol] = i
			position := HeldPosition{Symbol: symbol, Quantity: new(float64)}
			if move, ok := moves[symbol]; ok {
				position.Move = &move
			}
			position.Issue = issues[symbol]
			positions = append(positions, position)
		}
		position := &positions[i]
		if position.Quantity != nil {
			if holding.Quantity == nil {
				position.Quantity = nil
			} else {
				*position.Quantity += *holding.Quantity
			}
		}
		if position.Price == nil && holding.LivePrice != nil {
			price := *holding.LivePrice
			position.Price = &price
		}
		if holding.ValueVND == nil {
			position.Unvalued++
			if position.Issue == "" {
				position.Issue = holding.Issue
			}
			continue
		}
		position.Value += *holding.ValueVND
	}
	for i := range positions {
		if move := positions[i].Move; move != nil && positions[i].Value > 0 {
			value := moveChange(positions[i].Value, move.ChangePct)
			positions[i].Change = &value
		}
	}
	sort.SliceStable(positions, func(i, j int) bool { return positions[i].Value > positions[j].Value })
	return positions
}

// shareOf sets each position's share of the section's valued total.
func shareOf(positions []HeldPosition) []HeldPosition {
	total := 0.0
	for _, position := range positions {
		total += position.Value
	}
	for i := range positions {
		if total > 0 {
			positions[i].Share = round2(positions[i].Value / total * 100)
		}
	}
	return positions
}

func sumChange(positions []HeldPosition) (float64, float64, bool) {
	var change, covered float64
	ok := false
	for _, position := range positions {
		if position.Change != nil {
			change += *position.Change
			covered += position.Value
			ok = true
		}
	}
	return change, covered, ok
}

// changePercent is a change over the value before it.
func changePercent(change float64, value float64) *float64 {
	if before := value - change; before > 0 {
		pct := round2(change / before * 100)
		return &pct
	}
	return nil
}

func drawdowns(crypto float64, netWorth float64) []DrawdownScenario {
	scenarios := make([]DrawdownScenario, 0, len(CryptoDrawdowns))
	for _, fall := range CryptoDrawdowns {
		loss := crypto * fall / 100
		scenario := DrawdownScenario{FallPct: fall, Loss: loss, NetWorthAfter: netWorth - loss}
		if netWorth > 0 {
			pct := round2(-loss / netWorth * 100)
			scenario.NetWorthChangePct = &pct
		}
		scenarios = append(scenarios, scenario)
	}
	return scenarios
}

// ExposureFacts are the exposure figures the AI sees: classes only, with no
// account, platform or coin quantities.
type ExposureFacts struct {
	VolatileSharePct         float64             `json:"volatileSharePct"`
	Classes                  []ExposureClassFact `json:"classes"`
	TodayChangeVND           *float64            `json:"todayChangeVND,omitempty"`
	TodayChangePct           *float64            `json:"todayChangePct,omitempty"`
	CryptoValueVND           float64             `json:"cryptoValueVND"`
	CryptoShareOfNetWorthPct *float64            `json:"cryptoShareOfNetWorthPct,omitempty"`
	CryptoDrawdowns          []DrawdownScenario  `json:"cryptoDrawdowns"`
	Note                     string              `json:"note"`
}

// ExposureClassFact is one class of ExposureFacts.
type ExposureClassFact struct {
	Class     AllocationClass `json:"class"`
	ValueVND  float64         `json:"valueVND"`
	SharePct  float64         `json:"sharePct"`
	Volatile  bool            `json:"volatile"`
	ChangeVND *float64        `json:"todayChangeVND,omitempty"`
}

// Facts summarises the exposure for prompts.
func (e Exposure) Facts() ExposureFacts {
	facts := ExposureFacts{
		VolatileSharePct: e.VolatileShare, Classes: []ExposureClassFact{},
		TodayChangeVND: roundedVND(e.Change), TodayChangePct: e.ChangePct,
		CryptoValueVND: e.Crypto.Value, CryptoShareOfNetWorthPct: e.Crypto.ShareOfNetWorth, CryptoDrawdowns: e.Crypto.Scenarios,
		Note: "Engine-computed. Volatile = fund certificates + crypto. Today's change: coins over about 24h, funds since the previous NAV. Drawdowns are fixed scenarios, not forecasts.",
	}
	for _, class := range e.Classes {
		facts.Classes = append(facts.Classes, ExposureClassFact{Class: class.Class, ValueVND: class.Value, SharePct: class.Share, Volatile: class.Volatile, ChangeVND: roundedVND(class.Change)})
	}
	return facts
}

// WithExposure adds the exposure summary to the briefing facts.
func (f BriefingFacts) WithExposure(exposure Exposure) BriefingFacts {
	facts := exposure.Facts()
	f.Exposure = &facts
	return f
}

func roundedVND(value *float64) *float64 {
	if value == nil {
		return nil
	}
	rounded := math.Round(*value)
	return &rounded
}
