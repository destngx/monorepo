package domain

import (
	"fmt"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Fund performance and the deposit comparison (ROADMAP M3, WM-INV-21..23).
// A fund holding's cost comes from its History cell priced at the fund's
// NAV history; nothing is guessed when the two disagree.

// FundTrade is one History entry: units bought (positive) or sold
// (negative) on a day ("YYYY-MM-DD").
type FundTrade struct {
	Date  string  `json:"date"`
	Units float64 `json:"units"`
}

// NavPoint is one published NAV per unit, in VND.
type NavPoint struct {
	Date string  `json:"date"`
	NAV  float64 `json:"nav"`
}

// FundNAVHistory is a fund's published NAV history with its data source.
type FundNAVHistory struct {
	Symbol string      `json:"symbol"`
	Points []NavPoint  `json:"points"`
	Source *DataSource `json:"source,omitempty"`
}

// FundCashFlow is a trade priced at the NAV it was matched at. Amount is
// the cash seen from the investor: negative for a buy, positive for a sale.
type FundCashFlow struct {
	Date    string  `json:"date"`
	Units   float64 `json:"units"`
	NAV     float64 `json:"nav"`
	NAVDate string  `json:"navDate"`
	Amount  float64 `json:"amount"`
}

// DepositRate is the bank deposit rate used as the comparison. Historical
// deposit rates are not published, so today's rate is a labelled proxy.
type DepositRate struct {
	Bank        string `json:"bank"`
	Tenor       string `json:"tenor"`
	TenorMonths *int   `json:"tenorMonths,omitempty"`
	Channel     string `json:"channel,omitempty"`
	// Access is set for an instant-access comparison (idle cash): no_term
	// or one_month, with AccessLabel saying what it means.
	Access      DepositAccess `json:"access,omitempty"`
	AccessLabel string        `json:"accessLabel,omitempty"`
	Rate        float64       `json:"rate"` // percent a year
	UpdatedAt   string        `json:"updatedAt,omitempty"`
	Basis       string        `json:"basis"`
}

// Comparison with the deposit.
const (
	VsDepositAhead  = "ahead"
	VsDepositBehind = "behind"
	VsDepositEven   = "even"
)

// Why a holding's cost is (not) known.
const (
	FundCostKnown          = "known"
	FundCostNoHistory      = "no_history"
	FundCostHistoryInvalid = "history_invalid"
	FundCostUnitsMismatch  = "units_mismatch"
	FundCostNAVUnavailable = "nav_unavailable"
)

// FundReturn is the performance of a set of cash flows ending in a value.
// Percentages are percent; AnnualisedReturn is the money-weighted return
// (XIRR), left out when the money has been invested for under
// minAnnualisedDays. Deposit fields are what the same cash flows would be
// worth in a deposit at the comparison rate.
type FundReturn struct {
	TotalBought      float64  `json:"totalBought"`
	TotalSold        float64  `json:"totalSold"`
	NetInvested      float64  `json:"netInvested"`
	Value            float64  `json:"value"`
	Gain             float64  `json:"gain"`
	SimpleReturn     *float64 `json:"simpleReturn"`
	AnnualisedReturn *float64 `json:"annualisedReturn"`
	FirstDate        string   `json:"firstDate"`
	AsOf             string   `json:"asOf"`
	DepositValue     *float64 `json:"depositValue"`
	DepositGain      *float64 `json:"depositGain"`
	DifferenceVND    *float64 `json:"differenceVND"`
	DifferencePoints *float64 `json:"differencePoints"`
	VsDeposit        string   `json:"vsDeposit,omitempty"`
}

// FundHoldingPerformance is one row of the fund tab. Value is current units
// × the latest NAV. Return, CostBasis (average cost of the units held) and
// UnrealisedGain are set only when CostStatus is known.
type FundHoldingPerformance struct {
	Platform       string         `json:"platform"`
	Symbol         string         `json:"symbol"`
	Units          *float64       `json:"units"`
	HistoryUnits   float64        `json:"historyUnits"`
	NAV            *float64       `json:"nav"`
	NAVDate        string         `json:"navDate,omitempty"`
	Value          *float64       `json:"value"`
	CostStatus     string         `json:"costStatus"`
	CostBasis      *float64       `json:"costBasis"`
	UnrealisedGain *float64       `json:"unrealisedGain"`
	CashFlows      []FundCashFlow `json:"cashFlows"`
	Return         *FundReturn    `json:"return"`
	Issues         []string       `json:"issues"`
}

// FundCodePerformance aggregates the holdings of one fund code.
type FundCodePerformance struct {
	Symbol     string      `json:"symbol"`
	Holdings   int         `json:"holdings"`
	Value      float64     `json:"value"`
	CostStatus string      `json:"costStatus"`
	Return     *FundReturn `json:"return"`
}

// FundPerformanceTotals: Value counts every valued holding, ValueCostKnown
// and Return only the holdings with a known cost.
type FundPerformanceTotals struct {
	Holdings       int         `json:"holdings"`
	CostKnown      int         `json:"costKnown"`
	CostUnknown    int         `json:"costUnknown"`
	Value          float64     `json:"value"`
	ValueCostKnown float64     `json:"valueCostKnown"`
	Return         *FundReturn `json:"return"`
}

// FundPerformance is the WM-INV-21 answer.
type FundPerformance struct {
	Holdings     []FundHoldingPerformance `json:"holdings"`
	Funds        []FundCodePerformance    `json:"funds"`
	Totals       FundPerformanceTotals    `json:"totals"`
	Deposit      *DepositRate             `json:"deposit"`
	DepositIssue string                   `json:"depositIssue,omitempty"`
	GeneratedAt  time.Time                `json:"generatedAt"`
}

// FundPerformanceInputs is everything ComputeFundPerformance needs. NAVs
// and NAVIssues are keyed by FundKey.
type FundPerformanceInputs struct {
	Holdings     []Holding
	NAVs         map[string][]NavPoint
	NAVIssues    map[string]string
	Deposit      *DepositRate
	DepositIssue string
	Now          time.Time
}

const (
	// unitsTolerance absorbs rounding of units to two decimals.
	unitsTolerance = 0.01
	// minAnnualisedDays: annualising a few days of returns is noise.
	minAnnualisedDays = 30
	// evenShare: within 0.5% of the money bought counts as even.
	evenShare = 0.005
)

// FundKey identifies a fund code.
func FundKey(symbol string) string {
	return strings.ToUpper(strings.TrimSpace(symbol))
}

// ParseFundHistory reads a History cell: entries separated by `;`, each
// `d-m-yyyy,units`. Each entry is split on its first comma; units use a
// comma as the decimal separator (a dot then groups thousands) and a
// negative value is a sale. Unreadable entries are returned as issues.
func ParseFundHistory(cell string) ([]FundTrade, []string) {
	trades := []FundTrade{}
	issues := []string{}
	for index, raw := range strings.Split(cell, ";") {
		entry := strings.TrimSpace(raw)
		if entry == "" {
			continue
		}
		datePart, unitsPart, found := strings.Cut(entry, ",")
		if !found {
			issues = append(issues, fmt.Sprintf("History entry %d %q has no units", index+1, entry))
			continue
		}
		date, err := parseHistoryDate(datePart)
		if err != nil {
			issues = append(issues, fmt.Sprintf("History entry %d %q has an unreadable date", index+1, entry))
			continue
		}
		value, ok := parseHistoryUnits(unitsPart)
		if !ok || value == 0 {
			issues = append(issues, fmt.Sprintf("History entry %d %q has unreadable units", index+1, entry))
			continue
		}
		trades = append(trades, FundTrade{Date: date.Format(time.DateOnly), Units: value})
	}
	return trades, issues
}

func parseHistoryDate(value string) (time.Time, error) {
	trimmed := strings.ReplaceAll(strings.TrimSpace(value), "/", "-")
	return time.Parse("2-1-2006", trimmed)
}

// parseHistoryUnits: with a comma, `1.234,5` is 1234.5; without one the
// value is read as written (`12.5`), so a units total that does not add up
// is caught by the reconciliation with Total Unit.
func parseHistoryUnits(value string) (float64, bool) {
	trimmed := strings.ReplaceAll(strings.TrimSpace(value), " ", "")
	if strings.Contains(trimmed, ",") {
		trimmed = strings.ReplaceAll(strings.ReplaceAll(trimmed, ".", ""), ",", ".")
	}
	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
		return 0, false
	}
	return parsed, true
}

// ComputeFundPerformance values each fund holding at its latest NAV and,
// when its History reconciles with Total Unit, prices every trade at the
// NAV it was matched at (the NAV of the trade day, else the next published
// one) to get gain, simple and money-weighted return, and the same cash
// flows in a deposit.
func ComputeFundPerformance(in FundPerformanceInputs) FundPerformance {
	result := FundPerformance{Holdings: []FundHoldingPerformance{}, Funds: []FundCodePerformance{}, Deposit: in.Deposit, DepositIssue: in.DepositIssue, GeneratedAt: in.Now}
	navs := map[string][]NavPoint{}
	for key, points := range in.NAVs {
		navs[FundKey(key)] = sortedNavs(points)
	}
	navIssues := map[string]string{}
	for key, issue := range in.NAVIssues {
		navIssues[FundKey(key)] = issue
	}

	type fundGroup struct {
		code    FundCodePerformance
		flows   []datedFlow
		unknown bool
	}
	groups := []*fundGroup{}
	byCode := map[string]*fundGroup{}
	var knownFlows []datedFlow
	totals := &result.Totals

	for _, holding := range in.Holdings {
		if strings.TrimSpace(holding.Symbol) == "" {
			continue
		}
		performance, flows := holdingPerformance(holding, navs[FundKey(holding.Symbol)], navIssues[FundKey(holding.Symbol)], in.Deposit)
		result.Holdings = append(result.Holdings, performance)

		key := FundKey(holding.Symbol)
		group := byCode[key]
		if group == nil {
			group = &fundGroup{code: FundCodePerformance{Symbol: key}}
			byCode[key] = group
			groups = append(groups, group)
		}
		group.code.Holdings++
		totals.Holdings++
		if performance.Value != nil {
			group.code.Value += *performance.Value
			totals.Value += *performance.Value
		}
		if performance.CostStatus != FundCostKnown {
			totals.CostUnknown++
			if !group.unknown {
				group.unknown = true
				group.code.CostStatus = performance.CostStatus
			}
			continue
		}
		totals.CostKnown++
		totals.ValueCostKnown += *performance.Value
		group.flows = append(group.flows, flows...)
		knownFlows = append(knownFlows, flows...)
	}

	for _, group := range groups {
		if !group.unknown {
			group.code.CostStatus = FundCostKnown
			group.code.Return = evaluateFlows(group.flows, in.Deposit)
		}
		result.Funds = append(result.Funds, group.code)
	}
	if totals.CostKnown > 0 {
		totals.Return = evaluateFlows(knownFlows, in.Deposit)
	}
	return result
}

// datedFlow is a cash flow for XIRR and the deposit. terminal marks the
// value at the end, which is not money bought or sold.
type datedFlow struct {
	date     time.Time
	amount   float64
	terminal bool
	// asOf is the day the holding of a trade is valued, where its deposit
	// stops growing.
	asOf time.Time
}

func holdingPerformance(holding Holding, navs []NavPoint, navIssue string, deposit *DepositRate) (FundHoldingPerformance, []datedFlow) {
	performance := FundHoldingPerformance{
		Platform:  holding.Platform,
		Symbol:    holding.Symbol,
		Units:     holding.Quantity,
		CashFlows: []FundCashFlow{},
		Issues:    slices.Clone(holding.HistoryIssues),
	}
	if performance.Issues == nil {
		performance.Issues = []string{}
	}
	for _, trade := range holding.History {
		performance.HistoryUnits += trade.Units
	}
	performance.HistoryUnits = math.Round(performance.HistoryUnits*1e6) / 1e6

	if len(navs) == 0 {
		performance.CostStatus = FundCostNAVUnavailable
		reason := "no NAV history"
		if navIssue != "" {
			reason += ": " + navIssue
		}
		performance.Issues = append(performance.Issues, reason)
		return performance, nil
	}
	latest := navs[len(navs)-1]
	performance.NAV, performance.NAVDate = &latest.NAV, latest.Date
	if holding.Quantity != nil {
		value := *holding.Quantity * latest.NAV
		performance.Value = &value
	} else {
		performance.Issues = append(performance.Issues, "no Total Unit in the sheet")
	}

	switch {
	case len(holding.HistoryIssues) > 0:
		performance.CostStatus = FundCostHistoryInvalid
		return performance, nil
	case len(holding.History) == 0:
		performance.CostStatus = FundCostNoHistory
		return performance, nil
	case holding.Quantity == nil:
		performance.CostStatus = FundCostUnitsMismatch
		return performance, nil
	case math.Abs(performance.HistoryUnits-*holding.Quantity) > unitsTolerance:
		performance.CostStatus = FundCostUnitsMismatch
		performance.Issues = append(performance.Issues, fmt.Sprintf(
			"History adds up to %s units but Total Unit is %s; units bought outside the History cannot be priced, so cost is unknown",
			formatUnits(performance.HistoryUnits), formatUnits(*holding.Quantity)))
		return performance, nil
	}

	trades := slices.Clone(holding.History)
	sort.SliceStable(trades, func(i, j int) bool { return trades[i].Date < trades[j].Date })
	flows := make([]datedFlow, 0, len(trades)+1)
	heldUnits, heldCost := 0.0, 0.0
	for _, trade := range trades {
		point, ok := matchedNAV(navs, trade.Date)
		if !ok {
			performance.CostStatus = FundCostNAVUnavailable
			performance.CashFlows = []FundCashFlow{}
			performance.Issues = append(performance.Issues, fmt.Sprintf("no NAV published on or after %s to price that trade", trade.Date))
			return performance, nil
		}
		amount := -trade.Units * point.NAV
		performance.CashFlows = append(performance.CashFlows, FundCashFlow{Date: trade.Date, Units: trade.Units, NAV: point.NAV, NAVDate: point.Date, Amount: amount})
		flows = append(flows, datedFlow{date: mustDay(point.Date), amount: amount, asOf: mustDay(latest.Date)})
		if trade.Units > 0 {
			heldUnits += trade.Units
			heldCost += -amount
		} else if heldUnits > 0 {
			// Average cost: a sale removes its share of the cost held.
			heldCost -= heldCost * math.Min(1, -trade.Units/heldUnits)
			heldUnits += trade.Units
		}
	}
	flows = append(flows, datedFlow{date: mustDay(latest.Date), amount: *performance.Value, terminal: true})

	performance.CostStatus = FundCostKnown
	costBasis := heldCost
	unrealised := *performance.Value - costBasis
	performance.CostBasis, performance.UnrealisedGain = &costBasis, &unrealised
	performance.Return = evaluateFlows(flows, deposit)
	return performance, flows
}

// matchedNAV is the NAV a trade on date was matched at: that day's NAV,
// else the next published one.
func matchedNAV(navs []NavPoint, date string) (NavPoint, bool) {
	index := sort.Search(len(navs), func(i int) bool { return navs[i].Date >= date })
	if index == len(navs) {
		return NavPoint{}, false
	}
	return navs[index], true
}

func sortedNavs(points []NavPoint) []NavPoint {
	byDate := map[string]NavPoint{}
	for _, point := range points {
		if _, err := time.Parse(time.DateOnly, point.Date); err == nil && point.NAV > 0 {
			byDate[point.Date] = point
		}
	}
	sorted := make([]NavPoint, 0, len(byDate))
	for _, point := range byDate {
		sorted = append(sorted, point)
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Date < sorted[j].Date })
	return sorted
}

func mustDay(date string) time.Time {
	parsed, err := time.Parse(time.DateOnly, date)
	if err != nil {
		panic(fmt.Sprintf("fund performance: unchecked date %q", date))
	}
	return parsed
}

func formatUnits(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

// evaluateFlows computes a FundReturn from trades and terminal values.
func evaluateFlows(flows []datedFlow, deposit *DepositRate) *FundReturn {
	if len(flows) == 0 {
		return nil
	}
	result := &FundReturn{}
	first, last := flows[0].date, flows[0].date
	for _, flow := range flows {
		if flow.date.Before(first) {
			first = flow.date
		}
		if flow.date.After(last) {
			last = flow.date
		}
		switch {
		case flow.terminal:
			result.Value += flow.amount
		case flow.amount < 0:
			result.TotalBought += -flow.amount
		default:
			result.TotalSold += flow.amount
		}
	}
	result.FirstDate, result.AsOf = first.Format(time.DateOnly), last.Format(time.DateOnly)
	result.NetInvested = result.TotalBought - result.TotalSold
	result.Gain = result.Value - result.NetInvested
	if result.TotalBought > 0 {
		simple := result.Gain / result.TotalBought * 100
		result.SimpleReturn = &simple
	}
	if last.Sub(first) >= minAnnualisedDays*24*time.Hour {
		if rate, ok := xirr(flows); ok {
			annualised := rate * 100
			result.AnnualisedReturn = &annualised
		}
	}

	if deposit == nil {
		return result
	}
	// The deposit takes the same cash on the same days and grows, compounded
	// yearly, to the day each holding is valued.
	depositValue := 0.0
	for _, flow := range flows {
		if flow.terminal {
			continue
		}
		depositValue += -flow.amount * math.Pow(1+deposit.Rate/100, yearsBetween(flow.date, flow.asOf))
	}
	depositGain := depositValue - result.NetInvested
	difference := result.Value - depositValue
	result.DepositValue, result.DepositGain, result.DifferenceVND = &depositValue, &depositGain, &difference
	if result.AnnualisedReturn != nil {
		points := *result.AnnualisedReturn - deposit.Rate
		result.DifferencePoints = &points
	}
	switch {
	case math.Abs(difference) <= evenShare*result.TotalBought:
		result.VsDeposit = VsDepositEven
	case difference > 0:
		result.VsDeposit = VsDepositAhead
	default:
		result.VsDeposit = VsDepositBehind
	}
	return result
}

func yearsBetween(from time.Time, to time.Time) float64 {
	return to.Sub(from).Hours() / 24 / 365
}

// xirr is the annual rate r with Σ amount / (1+r)^(years from the first
// flow) = 0, by bisection; it needs money both in and out.
func xirr(flows []datedFlow) (float64, bool) {
	hasIn, hasOut := false, false
	start := flows[0].date
	for _, flow := range flows {
		hasIn = hasIn || flow.amount < 0
		hasOut = hasOut || flow.amount > 0
		if flow.date.Before(start) {
			start = flow.date
		}
	}
	if !hasIn || !hasOut {
		return 0, false
	}
	npv := func(rate float64) float64 {
		total := 0.0
		for _, flow := range flows {
			total += flow.amount / math.Pow(1+rate, yearsBetween(start, flow.date))
		}
		return total
	}
	low, high := -0.999999, 1.0
	for npv(low)*npv(high) > 0 {
		high *= 2
		if high > 1e6 {
			return 0, false
		}
	}
	for range 300 {
		mid := (low + high) / 2
		if npv(low)*npv(mid) <= 0 {
			high = mid
		} else {
			low = mid
		}
		if high-low < 1e-12 {
			break
		}
	}
	return (low + high) / 2, true
}
