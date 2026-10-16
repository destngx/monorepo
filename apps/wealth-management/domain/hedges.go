package domain

import (
	"time"
)

// Gold and USD as protection (ROADMAP M4, WM-INV-27). The user's cash is
// VND; gold and the US dollar are the usual ways to hold value outside it.
// The answer frames their trend against the cash share ("your cash is X%
// of your assets; gold rose Y% and USD Z% over 12 months"), not as trading
// charts: one daily close per day, the change over fixed periods, and what
// the cash would have gained or lost in each over the same period.

// Hedge assets and the market data server symbols they are read from.
const (
	HedgeGold = "gold"
	HedgeUSD  = "usd"
	// HedgeGoldSymbol is SJC gold bars (fmarket's SJC sell price, VND per tael).
	HedgeGoldSymbol = "SJC"
	// HedgeUSDSymbol is the USD/VND market rate.
	HedgeUSDSymbol = "USDVND"
)

// HedgePeriod is a lookback the trend reports a change over.
type HedgePeriod struct {
	Label  string
	Months int
}

// HedgePeriods are the fixed lookbacks, shortest first. The longest is the
// span of the points.
var HedgePeriods = []HedgePeriod{{Label: "3M", Months: 3}, {Label: "1Y", Months: 12}}

// HedgeLookbackDays is how much daily history to request: the longest
// period plus a margin so a close at or before its start exists.
const HedgeLookbackDays = 380

// TrendChange is the change over one period. Partial means the history
// starts after the period start, so From is the first close. OnCash is what
// the VND cash would have gained (or lost) held in the asset instead.
type TrendChange struct {
	Period    string  `json:"period"`
	FromDate  string  `json:"fromDate"`
	From      float64 `json:"from"`
	ChangePct float64 `json:"changePct"`
	Partial   bool    `json:"partial"`
	OnCash    float64 `json:"onCash"`
}

// DailyClose is a daily close (date in Vietnam time).
type DailyClose struct {
	Date  string  `json:"date"`
	Close float64 `json:"close"`
}

// HedgeTrend is one protection asset over the longest period.
type HedgeTrend struct {
	Asset      string        `json:"asset"`
	Label      string        `json:"label"`
	Symbol     string        `json:"symbol"`
	Unit       string        `json:"unit"`
	Latest     float64       `json:"latest"`
	LatestDate string        `json:"latestDate"`
	Changes    []TrendChange `json:"changes"`
	Points     []DailyClose  `json:"points"`
	Source     *DataSource   `json:"source,omitempty"`
}

// Change returns the change of a period, if any.
func (t HedgeTrend) Change(period string) *TrendChange {
	for i := range t.Changes {
		if t.Changes[i].Period == period {
			return &t.Changes[i]
		}
	}
	return nil
}

// Hedges is the answer. CashShare is the cash class share of the
// allocation; VNDShare adds every other VND-denominated class (funds and
// money lent), i.e. everything but crypto.
type Hedges struct {
	Cash        float64     `json:"cash"`
	CashShare   float64     `json:"cashShare"`
	VNDShare    float64     `json:"vndShare"`
	Gold        *HedgeTrend `json:"gold"`
	USD         *HedgeTrend `json:"usd"`
	Issues      []string    `json:"issues"`
	GeneratedAt time.Time   `json:"generatedAt"`
}

// HedgeInputs are the allocation and the two daily series; a nil series
// comes with its issue.
type HedgeInputs struct {
	Allocation Allocation
	Gold       *CandleSeries
	GoldIssue  string
	USD        *CandleSeries
	USDIssue   string
	Now        time.Time
}

// ComputeHedges builds the gold and USD trends against the cash share.
func ComputeHedges(in HedgeInputs) Hedges {
	cash := in.Allocation.Line(AllocationCash)
	result := Hedges{Cash: cash.Value, CashShare: cash.Share, Issues: []string{}, GeneratedAt: in.Now}
	if in.Allocation.Total > 0 {
		result.VNDShare = round2((in.Allocation.Total - in.Allocation.Line(AllocationCrypto).Value) / in.Allocation.Total * 100)
	}
	assets := []struct {
		asset, label, symbol, unit string
		series                     *CandleSeries
		issue                      string
		target                     **HedgeTrend
	}{
		{HedgeGold, "SJC gold", HedgeGoldSymbol, "VND per tael (SJC sell price)", in.Gold, in.GoldIssue, &result.Gold},
		{HedgeUSD, "US dollar", HedgeUSDSymbol, "VND per USD", in.USD, in.USDIssue, &result.USD},
	}
	for _, asset := range assets {
		if asset.series == nil {
			result.Issues = append(result.Issues, asset.label+" trend unavailable: "+asset.issue)
			continue
		}
		trend, ok := hedgeTrend(*asset.series, result.Cash)
		if !ok {
			result.Issues = append(result.Issues, asset.label+" trend unavailable: "+errNoMove.Error())
			continue
		}
		trend.Asset, trend.Label, trend.Symbol, trend.Unit = asset.asset, asset.label, asset.symbol, asset.unit
		*asset.target = &trend
	}
	return result
}

// hedgeTrend keeps one close per Vietnam day over the longest period and
// the change over each period.
func hedgeTrend(series CandleSeries, cash float64) (HedgeTrend, bool) {
	days := []DailyClose{}
	times := []time.Time{}
	for _, candle := range pricedCandles(series.Candles) {
		date := candle.Time.In(AppLocation).Format(time.DateOnly)
		if n := len(days); n > 0 && days[n-1].Date == date {
			days[n-1].Close, times[n-1] = candle.Close, candle.Time
			continue
		}
		days = append(days, DailyClose{Date: date, Close: candle.Close})
		times = append(times, candle.Time)
	}
	if len(days) < 2 {
		return HedgeTrend{}, false
	}
	last := len(days) - 1
	trend := HedgeTrend{Latest: days[last].Close, LatestDate: days[last].Date, Changes: []TrendChange{}, Source: seriesSource(series)}
	start := 0
	for _, period := range HedgePeriods {
		anchor := times[last].AddDate(0, -period.Months, 0)
		reference, partial := 0, true
		for i := range last {
			if !times[i].After(anchor) {
				reference, partial = i, false
			}
		}
		from := days[reference]
		pct := round2((days[last].Close - from.Close) / from.Close * 100)
		trend.Changes = append(trend.Changes, TrendChange{
			Period: period.Label, FromDate: from.Date, From: from.Close, ChangePct: pct, Partial: partial,
			OnCash: cash * (days[last].Close - from.Close) / from.Close,
		})
		start = reference
	}
	trend.Points = days[start:]
	return trend, true
}
