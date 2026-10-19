package domain

import (
	"math"
	"slices"
	"sort"
	"strings"
	"time"
)

// Spending windows (WM-BUD-12): how the user is spending today, over the
// last 30 days and over the last 3 months, each a rolling window ending
// today, with recommendations drawn from the comparison.
const (
	SpendingRecentDays   = 30
	SpendingBaselineDays = 90
	// spendingAverageMonthDays turns a daily average into a month (365/12).
	spendingAverageMonthDays = 30.4
	// spendingPaceChange is how far the 30-day daily average must move from
	// the 3-month one to be called faster or slower (as the WM-BUD-9 trend).
	spendingPaceChange = 0.15
	// spendingRiseChange is how far a category or split must rise over its
	// 3-month month to be flagged, and spendingRiseShare the least share of
	// the 30 days' day-to-day spend the extra must be, so small categories
	// do not raise noise.
	spendingRiseChange = 0.25
	spendingRiseShare  = 0.05
	// spendingHeavyDay is today's day-to-day spend against a typical day.
	spendingHeavyDay        = 2.0
	maxSpendingRecommends   = 5
	maxSpendingRisingAlerts = 3
)

// SpendingWindows is the Budget page's "How am I spending?" card.
type SpendingWindows struct {
	AsOf  string        `json:"asOf"`
	Today SpendingToday `json:"today"`
	// Windows are the last 30 days and the last 3 months (90 days).
	Windows []SpendingWindow `json:"windows"`
	// Daily is each of the last 90 days, oldest first.
	Daily           []SpendingDay            `json:"daily"`
	Recommendations []SpendingRecommendation `json:"recommendations"`
}

// SpendingToday is today's expense against a typical day of the 30 days
// before it.
type SpendingToday struct {
	Date     string  `json:"date"`
	Spent    float64 `json:"spent"`
	DayToDay float64 `json:"dayToDay"`
	// TypicalDay is the day-to-day average of the 30 days before today.
	TypicalDay float64 `json:"typicalDay"`
	// ChangePercent compares DayToDay with TypicalDay; nil without a typical day.
	ChangePercent *float64 `json:"changePercent"`
}

// SpendingDay is one day's expense: fixed costs (paid once a month) apart
// from day-to-day spend.
type SpendingDay struct {
	Date     string  `json:"date"`
	Fixed    float64 `json:"fixed"`
	DayToDay float64 `json:"dayToDay"`
}

// SpendingWindow is the expense of a rolling window ending today.
type SpendingWindow struct {
	Days int    `json:"days"`
	From string `json:"from"`
	To   string `json:"to"`
	// Spent = Fixed + DayToDay. Fixed categories (Categories tag fixed) are
	// paid once, so averages per day use DayToDay only.
	Spent    float64 `json:"spent"`
	Fixed    float64 `json:"fixed"`
	DayToDay float64 `json:"dayToDay"`
	// DailyAverage is DayToDay / Days; MonthlyAverage is Spent per 30.4 days.
	DailyAverage   float64 `json:"dailyAverage"`
	MonthlyAverage float64 `json:"monthlyAverage"`
	// Previous is the same number of days just before From.
	Previous SpendingPeriod `json:"previous"`
	// DailyAverageChangePercent compares DailyAverage with the previous
	// period's; nil when it had no day-to-day spend.
	DailyAverageChangePercent *float64 `json:"dailyAverageChangePercent"`
	// Categories are the expense categories with spend, largest first.
	Categories []SpendingCategory `json:"categories"`
}

// SpendingPeriod is the window before a SpendingWindow.
type SpendingPeriod struct {
	From         string  `json:"from"`
	To           string  `json:"to"`
	Spent        float64 `json:"spent"`
	DayToDay     float64 `json:"dayToDay"`
	DailyAverage float64 `json:"dailyAverage"`
}

// SpendingCategory is one expense category in a window.
type SpendingCategory struct {
	Category string         `json:"category"`
	Nature   CategoryNature `json:"nature,omitempty"`
	Spent    float64        `json:"spent"`
	// PercentOfSpent is the category's share of the window's Spent.
	PercentOfSpent float64 `json:"percentOfSpent"`
	MonthlyAverage float64 `json:"monthlyAverage"`
	PreviousSpent  float64 `json:"previousSpent"`
	// ChangePercent compares Spent with PreviousSpent; nil without previous spend.
	ChangePercent *float64 `json:"changePercent"`
	// Splits are the category's split tags in the window (WM-CAT-3).
	Splits []SplitSpend `json:"splits,omitempty"`
}

// SpendingRecommendationKind names a spending recommendation rule.
type SpendingRecommendationKind string

const (
	// SpendingFaster: the last 30 days' day-to-day average is above the 3-month one.
	SpendingFaster SpendingRecommendationKind = "faster"
	// SpendingSlower: below it; the difference can go to goals.
	SpendingSlower SpendingRecommendationKind = "slower"
	// SpendingCategoryRising: a category's last 30 days are above its 3-month month.
	SpendingCategoryRising SpendingRecommendationKind = "category_rising"
	// SpendingSplitRising: the same for a split tag inside a category.
	SpendingSplitRising SpendingRecommendationKind = "split_rising"
	// SpendingHeavyDay: today's day-to-day spend is well above a typical day.
	SpendingHeavyDay SpendingRecommendationKind = "heavy_day"
)

// SpendingRecommendation is one deterministic piece of advice; the UI words
// it so amounts stay maskable. Tone is warning (act), info (look) or good
// (keep it up). Current and Usual are daily averages for faster and slower,
// the 30 days' spend and the usual 30 days (3-month spend / 3) for a rising
// category or split, and today against a typical day for heavy_day.
type SpendingRecommendation struct {
	Kind     SpendingRecommendationKind `json:"kind"`
	Tone     string                     `json:"tone"`
	Category string                     `json:"category,omitempty"`
	Tag      string                     `json:"tag,omitempty"`
	Nature   CategoryNature             `json:"nature,omitempty"`
	Current  float64                    `json:"current"`
	Usual    float64                    `json:"usual"`
	// Amount is the extra a month (the saving for slower; today's extra for
	// heavy_day), rounded to VND.
	Amount        float64 `json:"amount"`
	ChangePercent float64 `json:"changePercent"`
}

// spendingRow is an expense transaction reduced to what the windows need.
type spendingRow struct {
	day      time.Time
	category string
	name     string
	nature   CategoryNature
	tags     []string
	net      float64
}

// ComputeSpendingWindows builds the card from the ledger. Expense is
// WM-RULE-3 (expense categories, lending left out) attributed by effective
// date (WM-RULE-4); a category's net spend (payment - deposit, WM-RULE-5)
// in a window never goes below zero. Rows dated after today are left out.
func ComputeSpendingWindows(transactions []Transaction, categories []Category, today time.Time) SpendingWindows {
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	first := today.AddDate(0, 0, -2*SpendingBaselineDays+1)
	natures := map[string]CategoryNature{}
	for _, category := range categories {
		if category.Nature != "" {
			natures[normalizeCategory(category.Name)] = category.Nature
		}
	}
	rows := []spendingRow{}
	for _, t := range transactions {
		if !t.Usable() || !t.CountsAsExpense() {
			continue
		}
		effective := EffectiveDate(t)
		effective = time.Date(effective.Year(), effective.Month(), effective.Day(), 0, 0, 0, 0, time.UTC)
		if effective.Before(first) || effective.After(today) {
			continue
		}
		key := normalizeCategory(t.Category)
		nature, ok := natures[key]
		if !ok {
			nature = t.CategoryNature
		}
		rows = append(rows, spendingRow{
			day: effective, category: key, name: strings.TrimSpace(t.Category),
			nature: nature, tags: t.Tags, net: t.PaymentAmount() - t.DepositAmount(),
		})
	}
	splits := NewSplitTagIndex(categories)

	result := SpendingWindows{AsOf: today.Format(time.DateOnly), Recommendations: []SpendingRecommendation{}}
	result.Today = spendingToday(rows, today)
	recent := spendingWindow(rows, splits, today, SpendingRecentDays)
	baseline := spendingWindow(rows, splits, today, SpendingBaselineDays)
	result.Windows = []SpendingWindow{recent, baseline}
	result.Daily = spendingDaily(rows, today, SpendingBaselineDays)
	result.Recommendations = spendingRecommendations(result.Today, recent, baseline)
	return result
}

func spendingToday(rows []spendingRow, today time.Time) SpendingToday {
	todayTotals := windowTotals(rows, today, today)
	before := windowTotals(rows, today.AddDate(0, 0, -SpendingRecentDays), today.AddDate(0, 0, -1))
	typical := round2(before.dayToDay / SpendingRecentDays)
	return SpendingToday{
		Date:          today.Format(time.DateOnly),
		Spent:         todayTotals.spent,
		DayToDay:      todayTotals.dayToDay,
		TypicalDay:    typical,
		ChangePercent: PercentChange(todayTotals.dayToDay, typical),
	}
}

func spendingWindow(rows []spendingRow, splits SplitTagIndex, today time.Time, days int) SpendingWindow {
	from := today.AddDate(0, 0, -days+1)
	previousTo := from.AddDate(0, 0, -1)
	previousFrom := previousTo.AddDate(0, 0, -days+1)
	current := windowTotals(rows, from, today)
	previous := windowTotals(rows, previousFrom, previousTo)

	window := SpendingWindow{
		Days:           days,
		From:           from.Format(time.DateOnly),
		To:             today.Format(time.DateOnly),
		Spent:          current.spent,
		Fixed:          current.fixed,
		DayToDay:       current.dayToDay,
		DailyAverage:   round2(current.dayToDay / float64(days)),
		MonthlyAverage: math.Round(current.spent / float64(days) * spendingAverageMonthDays),
		Previous: SpendingPeriod{
			From: previousFrom.Format(time.DateOnly), To: previousTo.Format(time.DateOnly),
			Spent: previous.spent, DayToDay: previous.dayToDay, DailyAverage: round2(previous.dayToDay / float64(days)),
		},
		Categories: []SpendingCategory{},
	}
	window.DailyAverageChangePercent = PercentChange(window.DailyAverage, window.Previous.DailyAverage)

	for _, key := range current.order {
		spent := current.categories[key]
		if spent <= 0 {
			continue
		}
		previousSpent := clampZero(previous.categories[key])
		category := SpendingCategory{
			Category:       current.names[key],
			Nature:         current.natures[key],
			Spent:          spent,
			PercentOfSpent: percentOf(spent, current.spent),
			MonthlyAverage: math.Round(spent / float64(days) * spendingAverageMonthDays),
			PreviousSpent:  previousSpent,
			ChangePercent:  PercentChange(spent, previousSpent),
		}
		if tags := splits.For(key); len(tags) > 0 {
			category.Splits = make([]SplitSpend, 0, len(tags))
			for _, tag := range tags {
				amount := clampZero(splitNet(rows, key, tag, from, today))
				category.Splits = append(category.Splits, SplitSpend{Tag: tag, Spent: amount, PercentOfSpent: percentOf(amount, spent)})
			}
		}
		window.Categories = append(window.Categories, category)
	}
	sort.SliceStable(window.Categories, func(i, j int) bool { return window.Categories[i].Spent > window.Categories[j].Spent })
	return window
}

func spendingDaily(rows []spendingRow, today time.Time, days int) []SpendingDay {
	daily := make([]SpendingDay, days)
	for i := range daily {
		date := today.AddDate(0, 0, i-days+1)
		totals := windowTotals(rows, date, date)
		daily[i] = SpendingDay{Date: date.Format(time.DateOnly), Fixed: totals.fixed, DayToDay: totals.dayToDay}
	}
	return daily
}

// spendingTotals is a window's spend: per category net, clamped at zero,
// then summed into fixed and day-to-day.
type spendingTotals struct {
	spent, fixed, dayToDay float64
	categories             map[string]float64
	names                  map[string]string
	natures                map[string]CategoryNature
	order                  []string
}

func windowTotals(rows []spendingRow, from, to time.Time) spendingTotals {
	totals := spendingTotals{categories: map[string]float64{}, names: map[string]string{}, natures: map[string]CategoryNature{}}
	for _, row := range rows {
		if row.day.Before(from) || row.day.After(to) {
			continue
		}
		if _, seen := totals.categories[row.category]; !seen {
			totals.order = append(totals.order, row.category)
			totals.names[row.category] = row.name
		}
		totals.categories[row.category] += row.net
		if row.nature != "" {
			totals.natures[row.category] = row.nature
		}
	}
	for key, net := range totals.categories {
		net = clampZero(net)
		totals.categories[key] = net
		if totals.natures[key] == NatureFixed {
			totals.fixed += net
		} else {
			totals.dayToDay += net
		}
	}
	totals.spent = totals.fixed + totals.dayToDay
	return totals
}

func splitNet(rows []spendingRow, category, tag string, from, to time.Time) float64 {
	net := 0.0
	for _, row := range rows {
		if row.category != category || row.day.Before(from) || row.day.After(to) {
			continue
		}
		if slices.ContainsFunc(row.tags, func(value string) bool { return strings.EqualFold(strings.TrimSpace(value), tag) }) {
			net += row.net
		}
	}
	return net
}

// spendingRecommendations compares the last 30 days with the 3-month
// window. Fixed costs are left out (paid once a month, not a habit). Needs
// are never told to be cut: a rising essential gets the info tone (look at
// what changed), a rising want or split tag the warning tone.
func spendingRecommendations(today SpendingToday, recent, baseline SpendingWindow) []SpendingRecommendation {
	recommendations := []SpendingRecommendation{}
	if baseline.DailyAverage > 0 {
		change := recent.DailyAverage/baseline.DailyAverage - 1
		pace := SpendingRecommendation{
			Current: recent.DailyAverage, Usual: baseline.DailyAverage, ChangePercent: round2(change * 100),
			Amount: math.Round(math.Abs(recent.DailyAverage-baseline.DailyAverage) * spendingAverageMonthDays),
		}
		switch {
		case change > spendingPaceChange:
			pace.Kind, pace.Tone = SpendingFaster, "warning"
			recommendations = append(recommendations, pace)
		case change < -spendingPaceChange:
			pace.Kind, pace.Tone = SpendingSlower, "good"
			recommendations = append(recommendations, pace)
		}
	}

	recommendations = append(recommendations, risingSpend(recent, baseline)...)

	if today.TypicalDay > 0 && today.DayToDay >= today.TypicalDay*spendingHeavyDay {
		recommendations = append(recommendations, SpendingRecommendation{
			Kind: SpendingHeavyDay, Tone: "info", Current: today.DayToDay, Usual: today.TypicalDay,
			Amount: math.Round(today.DayToDay - today.TypicalDay), ChangePercent: round2((today.DayToDay/today.TypicalDay - 1) * 100),
		})
	}
	if len(recommendations) > maxSpendingRecommends {
		recommendations = recommendations[:maxSpendingRecommends]
	}
	return recommendations
}

// risingSpend flags the categories and split tags whose last 30 days are
// well above their usual 30 days (the 3-month window's spend / 3), largest
// extra first.
func risingSpend(recent, baseline SpendingWindow) []SpendingRecommendation {
	floor := recent.DayToDay * spendingRiseShare
	usualCategories := map[string]SpendingCategory{}
	for _, category := range baseline.Categories {
		usualCategories[normalizeCategory(category.Category)] = category
	}
	scale := float64(recent.Days) / float64(baseline.Days)
	rising := []SpendingRecommendation{}
	add := func(kind SpendingRecommendationKind, category SpendingCategory, tag string, spent, usual float64) {
		extra := spent - usual
		if usual <= 0 || extra < floor || extra <= usual*spendingRiseChange {
			return
		}
		tone := "warning"
		if kind == SpendingCategoryRising && category.Nature == NatureEssential {
			tone = "info"
		}
		rising = append(rising, SpendingRecommendation{
			Kind: kind, Tone: tone, Category: category.Category, Tag: tag, Nature: category.Nature,
			Current: spent, Usual: math.Round(usual), Amount: math.Round(extra / float64(recent.Days) * spendingAverageMonthDays),
			ChangePercent: round2(extra / usual * 100),
		})
	}
	for _, category := range recent.Categories {
		if category.Nature == NatureFixed {
			continue
		}
		usualCategory := usualCategories[normalizeCategory(category.Category)]
		add(SpendingCategoryRising, category, "", category.Spent, usualCategory.Spent*scale)
		for _, split := range category.Splits {
			for _, usualSplit := range usualCategory.Splits {
				if usualSplit.Tag == split.Tag {
					add(SpendingSplitRising, category, split.Tag, split.Spent, usualSplit.Spent*scale)
				}
			}
		}
	}
	sort.SliceStable(rising, func(i, j int) bool { return rising[i].Amount > rising[j].Amount })
	if len(rising) > maxSpendingRisingAlerts {
		rising = rising[:maxSpendingRisingAlerts]
	}
	return rising
}
