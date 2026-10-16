package domain

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// Net worth history (ROADMAP P3, WM-NW-4..6): one point per month from the
// first month with data to this month. A month with a Snapshots row uses it;
// a month without one is estimated by taking the nearest later known value
// back by the flows booked in between (the back-calculation misses price
// moves, so those points are marked estimated). This month is always
// today's live net worth, which is newer than any snapshot written so far.

// NetWorthSource says where a point comes from.
type NetWorthSource string

const (
	NetWorthFromSnapshot NetWorthSource = "snapshot"
	NetWorthEstimated    NetWorthSource = "estimated"
	NetWorthLive         NetWorthSource = "live"
)

// maxHistoryMonths bounds the all-time view (50 years).
const maxHistoryMonths = 600

// NetWorthPoint is one month. Change is against the previous point. Savings
// and Market split the change when this point and the previous one are both
// measured (snapshot or live): Savings is income - expense booked between
// their dates (WM-RULE-3, lending left out), Market the rest (price moves,
// balance corrections and anything not recorded as income or expense).
type NetWorthPoint struct {
	Month     string         `json:"month"`
	AsOf      string         `json:"asOf"`
	NetWorth  float64        `json:"netWorth"`
	Source    NetWorthSource `json:"source"`
	Estimated bool           `json:"estimated"`
	Change    *float64       `json:"change"`
	Savings   *float64       `json:"savings"`
	Market    *float64       `json:"market"`
}

// MilestoneKind is a threshold crossed for the first time or the all-time high.
type MilestoneKind string

const (
	MilestoneThreshold   MilestoneKind = "threshold"
	MilestoneAllTimeHigh MilestoneKind = "all_time_high"
)

// NetWorthMilestone marks a month. Threshold is set for threshold milestones.
type NetWorthMilestone struct {
	Month     string        `json:"month"`
	Kind      MilestoneKind `json:"kind"`
	Threshold *float64      `json:"threshold"`
	NetWorth  float64       `json:"netWorth"`
	Label     string        `json:"label"`
	Estimated bool          `json:"estimated"`
}

// NetWorthHistory is every month, oldest first, with the milestones.
type NetWorthHistory struct {
	AsOf       string              `json:"asOf"`
	Current    float64             `json:"current"`
	Points     []NetWorthPoint     `json:"points"`
	Milestones []NetWorthMilestone `json:"milestones"`
	// Snapshots and Estimated count the points of each kind.
	Snapshots int `json:"snapshots"`
	Estimated int `json:"estimated"`
	// TabExists is false until the first snapshot is written.
	TabExists bool            `json:"tabExists"`
	Issues    []SnapshotIssue `json:"issues"`
}

// NetWorthHistoryInputs are the current position, the ledger and the tab.
type NetWorthHistoryInputs struct {
	Accounts     []Account
	Transactions []Transaction
	Snapshots    SnapshotSheet
	Today        time.Time
}

// ComputeNetWorthHistory builds the monthly history with the change split
// and the milestones.
func ComputeNetWorthHistory(in NetWorthHistoryInputs) NetWorthHistory {
	current := ComputeBalanceSheet(in.Accounts, in.Transactions).NetWorth
	history := NetWorthHistory{
		AsOf: in.Today.Format(time.DateOnly), Current: current, Points: []NetWorthPoint{}, Milestones: []NetWorthMilestone{},
		TabExists: in.Snapshots.Exists, Issues: append([]SnapshotIssue{}, in.Snapshots.Issues...),
	}
	thisMonth := firstOfMonthOf(in.Today)
	snapshots := map[string]MonthSnapshot{}
	for _, snapshot := range in.Snapshots.Snapshots {
		month, err := ParseMonthKey(snapshot.Month)
		if err != nil || month.After(thisMonth) {
			continue
		}
		snapshots[snapshot.Month] = snapshot
	}
	first := historyStart(in.Transactions, snapshots, thisMonth)
	flows := WithoutLending(in.Transactions)

	// Walk back from this month: each estimate starts from the nearest
	// later measured value (anchor) and undoes the flows in between.
	months := monthsFrom(first, thisMonth)
	points := make([]NetWorthPoint, len(months))
	anchorValue, anchorDate := current, in.Today
	for i := len(months) - 1; i >= 0; i-- {
		month := months[i]
		key := MonthKey(month)
		end := SnapshotAsOf(month, in.Today)
		point := NetWorthPoint{Month: key, AsOf: end.Format(time.DateOnly)}
		switch snapshot, ok := snapshots[key]; {
		case month.Equal(thisMonth):
			// Today's position is newer than a snapshot written earlier this month.
			point.NetWorth, point.Source = current, NetWorthLive
		case ok:
			point.NetWorth, point.Source = snapshot.NetWorth, NetWorthFromSnapshot
			if asOf, err := time.Parse(time.DateOnly, snapshot.AsOf); err == nil {
				point.AsOf, end = snapshot.AsOf, asOf
			}
		default:
			point.NetWorth = anchorValue - netFlowBetween(flows, end, anchorDate)
			point.Source, point.Estimated = NetWorthEstimated, true
		}
		anchorValue, anchorDate = point.NetWorth, end
		points[i] = point
	}
	for i := range points {
		if points[i].Estimated {
			history.Estimated++
		} else if points[i].Source == NetWorthFromSnapshot {
			history.Snapshots++
		}
		if i == 0 {
			continue
		}
		previous := points[i-1]
		change := points[i].NetWorth - previous.NetWorth
		points[i].Change = &change
		if previous.Estimated || points[i].Estimated {
			continue
		}
		from, errFrom := time.Parse(time.DateOnly, previous.AsOf)
		to, errTo := time.Parse(time.DateOnly, points[i].AsOf)
		if errFrom != nil || errTo != nil {
			continue
		}
		savings := savingsBetween(in.Transactions, from, to)
		market := change - savings
		points[i].Savings, points[i].Market = &savings, &market
	}
	history.Points = points
	history.Milestones = netWorthMilestones(points)
	return history
}

// historyStart is the earliest month with a usable transaction or a
// snapshot, at most maxHistoryMonths back.
func historyStart(transactions []Transaction, snapshots map[string]MonthSnapshot, thisMonth time.Time) time.Time {
	first := thisMonth
	for _, t := range transactions {
		if t.Usable() && t.Date.Before(first) {
			first = firstOfMonthOf(t.Date)
		}
	}
	for key := range snapshots {
		if month, err := ParseMonthKey(key); err == nil && month.Before(first) {
			first = month
		}
	}
	if limit := thisMonth.AddDate(0, -(maxHistoryMonths - 1), 0); first.Before(limit) {
		first = limit
	}
	return first
}

func monthsFrom(first time.Time, last time.Time) []time.Time {
	months := []time.Time{}
	for month := first; !month.After(last); month = month.AddDate(0, 1, 0) {
		months = append(months, month)
	}
	return months
}

// netFlowBetween is deposits - payments booked after from, up to and
// including to (booking dates: balances move when the money moves).
func netFlowBetween(transactions []Transaction, from time.Time, to time.Time) float64 {
	flow := 0.0
	for _, t := range transactions {
		if t.Usable() && t.Date.After(from) && !t.Date.After(to) {
			flow += t.DepositAmount() - t.PaymentAmount()
		}
	}
	return flow
}

// savingsBetween is income - expense booked after from, up to and including
// to (WM-RULE-3 predicates, lending left out).
func savingsBetween(transactions []Transaction, from time.Time, to time.Time) float64 {
	savings := 0.0
	for _, t := range transactions {
		if !t.Usable() || !t.Date.After(from) || t.Date.After(to) {
			continue
		}
		switch {
		case t.CountsAsIncome():
			savings += t.DepositAmount()
		case t.CountsAsExpense():
			savings -= t.PaymentAmount()
		}
	}
	return savings
}

// MilestoneThresholds are the round net worth levels: 1-2-5 steps from
// 100 million VND up to 1,000 billion.
func MilestoneThresholds() []float64 {
	thresholds := []float64{}
	for scale := 1e8; scale <= 1e12; scale *= 10 {
		for _, step := range []float64{1, 2, 5} {
			if value := step * scale; value <= 1e12 {
				thresholds = append(thresholds, value)
			}
		}
	}
	return thresholds
}

// netWorthMilestones: the first month each round threshold is crossed
// upwards (a level the history starts above is not a crossing), then the
// all-time high (the latest month at the highest value).
func netWorthMilestones(points []NetWorthPoint) []NetWorthMilestone {
	milestones := []NetWorthMilestone{}
	if len(points) == 0 {
		return milestones
	}
	for _, threshold := range MilestoneThresholds() {
		if points[0].NetWorth >= threshold {
			continue
		}
		for _, point := range points[1:] {
			if point.NetWorth >= threshold {
				value := threshold
				milestones = append(milestones, NetWorthMilestone{
					Month: point.Month, Kind: MilestoneThreshold, Threshold: &value, NetWorth: point.NetWorth,
					Label: "Crossed " + compactVND(threshold), Estimated: point.Estimated,
				})
				break
			}
		}
	}
	high := 0
	for i, point := range points {
		if point.NetWorth >= points[high].NetWorth {
			high = i
		}
	}
	if len(points) > 1 && points[high].NetWorth > 0 {
		milestones = append(milestones, NetWorthMilestone{
			Month: points[high].Month, Kind: MilestoneAllTimeHigh, NetWorth: points[high].NetWorth,
			Label: "All-time high", Estimated: points[high].Estimated,
		})
	}
	sort.SliceStable(milestones, func(i, j int) bool { return milestones[i].Month < milestones[j].Month })
	return milestones
}

// compactVND writes a round amount as "100 million" or "1.5 billion" VND.
func compactVND(value float64) string {
	format := func(amount float64, unit string) string {
		text := strings.TrimSuffix(strings.TrimRight(fmt.Sprintf("%.1f", amount), "0"), ".")
		return text + " " + unit + " VND"
	}
	switch {
	case math.Abs(value) >= 1e9:
		return format(value/1e9, "billion")
	case math.Abs(value) >= 1e6:
		return format(value/1e6, "million")
	default:
		return fmt.Sprintf("%.0f VND", value)
	}
}

// NetWorthHistoryFacts summarise the history for the AI briefing (nameless).
type NetWorthHistoryFacts struct {
	Months    int     `json:"months"`
	Snapshots int     `json:"snapshots"`
	Estimated int     `json:"estimated"`
	Current   float64 `json:"current"`
	// YearAgo is net worth 12 months back; nil with a shorter history.
	YearAgo          *float64 `json:"yearAgo,omitempty"`
	YearChange       *float64 `json:"yearChange,omitempty"`
	YearChangePct    *float64 `json:"yearChangePct,omitempty"`
	YearAgoEstimated bool     `json:"yearAgoEstimated,omitempty"`
	AllTimeHigh      *float64 `json:"allTimeHigh,omitempty"`
	AllTimeHighMonth string   `json:"allTimeHighMonth,omitempty"`
	// LastSplit is the latest month split into savings and market.
	LastSplit *NetWorthPoint `json:"lastSplit,omitempty"`
	Note      string         `json:"note"`
}

// Facts summarises the history for prompts.
func (h NetWorthHistory) Facts() NetWorthHistoryFacts {
	facts := NetWorthHistoryFacts{
		Months: len(h.Points), Snapshots: h.Snapshots, Estimated: h.Estimated, Current: math.Round(h.Current),
		Note: "Engine-computed. Snapshot months are measured at month end; estimated months are back-calculated from transactions and miss price moves. Market = net worth change - (income - expense).",
	}
	if n := len(h.Points); n > 12 {
		ago := h.Points[n-13]
		value := math.Round(ago.NetWorth)
		change := math.Round(h.Points[n-1].NetWorth - ago.NetWorth)
		facts.YearAgo, facts.YearChange, facts.YearAgoEstimated = &value, &change, ago.Estimated
		facts.YearChangePct = PercentChange(h.Points[n-1].NetWorth, ago.NetWorth)
	}
	for _, milestone := range h.Milestones {
		if milestone.Kind == MilestoneAllTimeHigh {
			value := math.Round(milestone.NetWorth)
			facts.AllTimeHigh, facts.AllTimeHighMonth = &value, milestone.Month
		}
	}
	for i := len(h.Points) - 1; i >= 0; i-- {
		if h.Points[i].Savings != nil {
			point := h.Points[i]
			facts.LastSplit = &point
			break
		}
	}
	return facts
}

// WithNetWorthHistory adds the history summary to the briefing facts.
func (f BriefingFacts) WithNetWorthHistory(history NetWorthHistory) BriefingFacts {
	facts := history.Facts()
	f.NetWorthHistory = &facts
	return f
}
