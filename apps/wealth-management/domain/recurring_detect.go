package domain

import (
	"math"
	"slices"
	"sort"
	"strings"
	"time"
)

// Recurring detection (WM-BILL-2): suggests recurring items from the
// transaction history for the user to confirm.
//
// Monthly: transactions grouped by payee (case, accents and spacing
// ignored), category and direction (in or out) that appear in at least
// detectMinMonths of the last detectMonths completed months, mostly once a
// month, with a stable amount (within detectAmountTolerance of the median
// in at least two thirds of those months) and a typical day (within
// detectDayTolerance days, across month ends, in two thirds of the stable
// months).
//
// Yearly: a group seen in one month a year, in the same month of at least
// two consecutive years (the latest within the last 13 months), amounts
// within detectYearlyTolerance of their median. Quarterly items are not
// detected; the user adds them by hand.

const (
	detectMonths           = 6
	detectMinMonths        = 3
	detectAmountTolerance  = 0.15
	detectYearlyTolerance  = 0.25
	detectDayTolerance     = 5
	detectYearlyDayDrift   = 7
	detectStableShare      = 2.0 / 3
	detectYearlyMinYears   = 2
	detectYearlyRecentDays = 400
	// A group without a payee is only suggested when it is this stable.
	detectBlankAmountSpread = 0.02
	detectBlankDaySpread    = 1
	detectBlankMinMonths    = 4
)

// SuggestionConfidence grades a suggestion.
type SuggestionConfidence string

const (
	ConfidenceHigh   SuggestionConfidence = "high"
	ConfidenceMedium SuggestionConfidence = "medium"
	ConfidenceLow    SuggestionConfidence = "low"
)

// RecurringSuggestion is a recurring item found in the history, ready to
// be confirmed into the Recurring tab.
type RecurringSuggestion struct {
	// Key identifies the group (direction, payee, category) for confirm.
	Key       string             `json:"key"`
	Name      string             `json:"name"`
	Kind      RecurringKind      `json:"kind"`
	Amount    float64            `json:"amount"`
	Frequency RecurringFrequency `json:"frequency"`
	Day       int                `json:"day"`
	Month     int                `json:"month,omitempty"`
	Account   string             `json:"account,omitempty"`
	Category  string             `json:"category"`
	Match     string             `json:"match"`
	// LastSeen is the latest month it was seen, yyyy-mm.
	LastSeen string `json:"lastSeen"`
	// Seen counts the months (monthly) or years (yearly) it was seen in;
	// Checked is the span looked at (6 months, or the years of history).
	Seen    int `json:"seen"`
	Checked int `json:"checked"`
	// AmountSpread is the largest gap from the median among the stable
	// occurrences, in percent; DaySpread the largest gap from the typical
	// day, in days.
	AmountSpread float64              `json:"amountSpread"`
	DaySpread    int                  `json:"daySpread"`
	Confidence   SuggestionConfidence `json:"confidence"`
}

// detectEntry is one transaction of a group.
type detectEntry struct {
	date    time.Time
	amount  float64
	account string
	payee   string
	cat     string
}

// detectPeriod is a group's money in one month: the total and the day of
// its first transaction.
type detectPeriod struct {
	month   time.Time
	amount  float64
	day     int
	count   int
	entries []detectEntry
}

type detectGroup struct {
	key       string
	direction string
	payee     string
	entries   []detectEntry
}

// DetectRecurring suggests recurring items from the transactions. Lending,
// transfers (non-budget), refunds and uncategorised rows are left out, and
// groups an existing Recurring row already covers are not suggested again.
func DetectRecurring(transactions []Transaction, items []RecurringItem, today time.Time) []RecurringSuggestion {
	today = dateOnly(today)
	monthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	windowStart := monthStart.AddDate(0, -detectMonths, 0)

	groups := map[string]*detectGroup{}
	order := []string{}
	for _, t := range transactions {
		direction, amount := detectFlow(t)
		if direction == "" || t.Date.After(today) {
			continue
		}
		payee := FoldText(t.Payee)
		key := direction + "|" + payee + "|" + FoldText(t.Category)
		group, ok := groups[key]
		if !ok {
			group = &detectGroup{key: key, direction: direction, payee: payee}
			groups[key] = group
			order = append(order, key)
		}
		group.entries = append(group.entries, detectEntry{date: dateOnly(t.Date), amount: amount, account: strings.TrimSpace(t.AccountName), payee: strings.TrimSpace(t.Payee), cat: strings.TrimSpace(t.Category)})
	}

	suggestions := []RecurringSuggestion{}
	for _, key := range order {
		group := groups[key]
		sort.SliceStable(group.entries, func(i, j int) bool { return group.entries[i].date.Before(group.entries[j].date) })
		suggestion, ok := detectMonthly(group, windowStart, monthStart)
		if !ok {
			suggestion, ok = detectYearly(group, monthStart, today)
		}
		if !ok || coveredByItem(group, items) {
			continue
		}
		suggestions = append(suggestions, suggestion)
	}
	rank := map[SuggestionConfidence]int{ConfidenceHigh: 0, ConfidenceMedium: 1, ConfidenceLow: 2}
	sort.SliceStable(suggestions, func(i, j int) bool {
		a, b := suggestions[i], suggestions[j]
		if rank[a.Confidence] != rank[b.Confidence] {
			return rank[a.Confidence] < rank[b.Confidence]
		}
		return a.Amount > b.Amount
	})
	return suggestions
}

// detectFlow returns the direction ("in" for income, "out" for expense)
// and amount of a transaction that can be part of a recurring item.
func detectFlow(t Transaction) (string, float64) {
	if !t.Usable() || t.IsLending() || IsRefundCategory(t.Category) {
		return "", 0
	}
	switch {
	case t.CountsAsIncome() && t.DepositAmount() > 0:
		return "in", t.DepositAmount()
	case t.CountsAsExpense() && t.PaymentAmount() > 0:
		return "out", t.PaymentAmount()
	}
	return "", 0
}

// periods buckets the group's entries in [from, to) by month.
func (group *detectGroup) periods(from, to time.Time) []detectPeriod {
	byMonth := map[time.Time]*detectPeriod{}
	months := []time.Time{}
	for _, entry := range group.entries {
		if entry.date.Before(from) || !entry.date.Before(to) {
			continue
		}
		month := time.Date(entry.date.Year(), entry.date.Month(), 1, 0, 0, 0, 0, time.UTC)
		period, ok := byMonth[month]
		if !ok {
			period = &detectPeriod{month: month, day: entry.date.Day()}
			byMonth[month] = period
			months = append(months, month)
		}
		period.amount += entry.amount
		period.count++
		period.entries = append(period.entries, entry)
	}
	slices.SortFunc(months, func(a, b time.Time) int { return a.Compare(b) })
	out := make([]detectPeriod, 0, len(months))
	for _, month := range months {
		out = append(out, *byMonth[month])
	}
	return out
}

func detectMonthly(group *detectGroup, from, to time.Time) (RecurringSuggestion, bool) {
	periods := group.periods(from, to)
	if len(periods) < detectMinMonths {
		return RecurringSuggestion{}, false
	}
	// Spending at the same place several times a month is a habit, not a bill.
	frequent := 0
	for _, period := range periods {
		if period.count > 1 {
			frequent++
		}
	}
	if frequent*2 > len(periods) {
		return RecurringSuggestion{}, false
	}
	stable, spread := stablePeriods(periods, detectAmountTolerance)
	if len(stable) < detectMinMonths || float64(len(stable)) < detectStableShare*float64(len(periods)) {
		return RecurringSuggestion{}, false
	}
	days := make([]int, len(stable))
	for i, period := range stable {
		days[i] = period.day
	}
	day, onDay, daySpread := typicalDay(days, detectDayTolerance)
	if onDay < detectMinMonths || float64(onDay) < detectStableShare*float64(len(stable)) {
		return RecurringSuggestion{}, false
	}
	allStable := len(stable) == len(periods)
	confidence := ConfidenceLow
	switch {
	case len(periods) >= 5 && allStable && spread <= 0.05 && daySpread <= 2:
		confidence = ConfidenceHigh
	case len(periods) >= 4 || (spread <= 0.05 && daySpread <= 3):
		confidence = ConfidenceMedium
	}
	if group.payee == "" {
		if !allStable || len(periods) < detectBlankMinMonths || spread > detectBlankAmountSpread || daySpread > detectBlankDaySpread {
			return RecurringSuggestion{}, false
		}
		confidence = ConfidenceLow
	}
	suggestion := group.suggestion(stable, periods[len(periods)-1].month)
	suggestion.Frequency = FrequencyMonthly
	suggestion.Day = day
	suggestion.Seen, suggestion.Checked = len(periods), detectMonths
	suggestion.AmountSpread, suggestion.DaySpread = round2(spread*100), daySpread
	suggestion.Confidence = confidence
	return suggestion, true
}

func detectYearly(group *detectGroup, monthStart, today time.Time) (RecurringSuggestion, bool) {
	if group.payee == "" {
		return RecurringSuggestion{}, false
	}
	periods := group.periods(time.Time{}, monthStart.AddDate(0, 1, 0))
	if len(periods) < detectYearlyMinYears {
		return RecurringSuggestion{}, false
	}
	byYear := map[int]detectPeriod{}
	for _, period := range periods {
		if _, seen := byYear[period.month.Year()]; seen {
			return RecurringSuggestion{}, false // more than one month a year
		}
		byYear[period.month.Year()] = period
	}
	latest := periods[len(periods)-1]
	if today.Sub(latest.month).Hours()/24 > detectYearlyRecentDays {
		return RecurringSuggestion{}, false
	}
	// Consecutive years ending at the latest, in the same month.
	run := []detectPeriod{latest}
	for year := latest.month.Year() - 1; ; year-- {
		period, ok := byYear[year]
		if !ok || period.month.Month() != latest.month.Month() {
			break
		}
		run = append([]detectPeriod{period}, run...)
	}
	if len(run) < detectYearlyMinYears {
		return RecurringSuggestion{}, false
	}
	stable, spread := stablePeriods(run, detectYearlyTolerance)
	if len(stable) != len(run) {
		return RecurringSuggestion{}, false
	}
	days := make([]int, len(run))
	for i, period := range run {
		days[i] = period.day
	}
	day, onDay, daySpread := typicalDay(days, detectYearlyDayDrift)
	if onDay != len(run) {
		return RecurringSuggestion{}, false
	}
	suggestion := group.suggestion(run, latest.month)
	suggestion.Frequency = FrequencyYearly
	suggestion.Day = day
	suggestion.Month = int(latest.month.Month())
	suggestion.Seen = len(run)
	suggestion.Checked = periods[len(periods)-1].month.Year() - periods[0].month.Year() + 1
	suggestion.AmountSpread, suggestion.DaySpread = round2(spread*100), daySpread
	suggestion.Confidence = ConfidenceLow
	if len(run) >= 3 {
		suggestion.Confidence = ConfidenceMedium
	}
	return suggestion, true
}

// suggestion fills the fields shared by monthly and yearly suggestions:
// the latest spelling of payee and category, the median amount of the
// stable periods and the account used most (the latest on ties).
func (group *detectGroup) suggestion(stable []detectPeriod, lastSeen time.Time) RecurringSuggestion {
	last := group.entries[len(group.entries)-1]
	amounts := make([]float64, len(stable))
	accounts := map[string]int{}
	account, best := "", 0
	for i, period := range stable {
		amounts[i] = period.amount
		for _, entry := range period.entries {
			accounts[entry.account]++
			if accounts[entry.account] >= best {
				account, best = entry.account, accounts[entry.account]
			}
		}
	}
	name := last.payee
	if name == "" {
		name = last.cat
	}
	return RecurringSuggestion{
		Key:      group.key,
		Name:     name,
		Kind:     guessKind(group.direction, last.payee, last.cat),
		Amount:   math.Round(median(amounts)),
		Account:  account,
		Category: last.cat,
		Match:    last.payee,
		LastSeen: MonthKey(lastSeen),
	}
}

// stablePeriods keeps the periods whose amount is within tolerance of the
// median and returns the largest relative gap among them.
func stablePeriods(periods []detectPeriod, tolerance float64) ([]detectPeriod, float64) {
	amounts := make([]float64, len(periods))
	for i, period := range periods {
		amounts[i] = period.amount
	}
	center := median(amounts)
	if center <= 0 {
		return nil, 0
	}
	stable := []detectPeriod{}
	spread := 0.0
	for _, period := range periods {
		gap := math.Abs(period.amount-center) / center
		if gap <= tolerance {
			stable = append(stable, period)
			spread = math.Max(spread, gap)
		}
	}
	return stable, spread
}

// typicalDay picks the day closest (across month ends) to the others and
// returns it with the number of days within tolerance of it and the largest
// gap among those.
func typicalDay(days []int, tolerance int) (int, int, int) {
	best, bestCost := 0, math.MaxInt
	for _, candidate := range days {
		cost := 0
		for _, day := range days {
			cost += monthDayGap(candidate, day)
		}
		if cost < bestCost || (cost == bestCost && candidate < best) {
			best, bestCost = candidate, cost
		}
	}
	within, spread := 0, 0
	for _, day := range days {
		if gap := monthDayGap(best, day); gap <= tolerance {
			within++
			spread = max(spread, gap)
		}
	}
	return best, within, spread
}

// dayDistance is the gap between two days of the month, wrapping at month
// end so the 30th and the 1st are close.
func monthDayGap(a, b int) int {
	gap := a - b
	if gap < 0 {
		gap = -gap
	}
	return min(gap, 31-gap)
}

var (
	insuranceWords    = []string{"insurance", "bao hiem"}
	subscriptionWords = []string{"subscription", "subscriptions", "membership"}
)

func guessKind(direction, payee, category string) RecurringKind {
	if direction == "in" {
		return RecurringIncome
	}
	folded := FoldText(payee + " " + category)
	for _, word := range insuranceWords {
		if containsWords(folded, word) {
			return RecurringInsurance
		}
	}
	for _, word := range subscriptionWords {
		if containsWords(folded, word) {
			return RecurringSubscription
		}
	}
	return RecurringBill
}

// coveredByItem reports whether a Recurring row (any status, so a paused
// row also records a dismissed suggestion) already covers the group: its
// match text or name is in the payee and its category, when set, agrees.
func coveredByItem(group *detectGroup, items []RecurringItem) bool {
	last := group.entries[len(group.entries)-1]
	for _, item := range items {
		if item.Kind == RecurringCard || item.Kind.IsIncome() != (group.direction == "in") {
			continue
		}
		if item.Category != "" && FoldText(item.Category) != FoldText(last.cat) {
			continue
		}
		match := FoldText(item.Match)
		if match == "" {
			match = FoldText(item.Name)
		}
		if match != "" && group.payee != "" && containsWords(group.payee, match) {
			return true
		}
		if group.payee == "" && item.Match == "" && item.Category != "" {
			return true
		}
	}
	return false
}
