package domain

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// YearlyCashFlow backs the yearly cash flow page (WM-DASH-8): income and
// expense per month of a calendar year, split by category (WM-RULE-3/4).
type YearlyCashFlow struct {
	Year int `json:"year"`
	// Years with income or expense, newest first, plus the current and requested year.
	Years []int `json:"years"`
	// MonthsElapsed is 12 for a past year, the current month for this year, 0 for a future one.
	MonthsElapsed int        `json:"monthsElapsed"`
	Months        []CashFlow `json:"months"`
	Total         CashFlow   `json:"total"`
	// MonthlyAverage is Total over the elapsed months.
	MonthlyAverage CashFlow `json:"monthlyAverage"`
	// PreviousYear totals the same elapsed months of the year before.
	PreviousYear         CashFlow         `json:"previousYear"`
	IncomeChangePercent  *float64         `json:"incomeChangePercent"`
	ExpenseChangePercent *float64         `json:"expenseChangePercent"`
	Income               []CategoryMonths `json:"income"`
	Expense              []CategoryMonths `json:"expense"`
	Verdict              YearVerdict      `json:"verdict"`
	// Projection estimates the year end; only for the current year.
	Projection *YearProjection `json:"projection"`
	// Spikes are elapsed months whose expense stands out, oldest first.
	Spikes        []ExpenseSpike `json:"spikes"`
	SavingsTarget SavingsTarget  `json:"savingsTarget"`
	ExpenseSplit  ExpenseSplit   `json:"expenseSplit"`
}

// ExpenseSplit splits the year's expense by the Categories tag. Needs are
// fixed plus essential; flexible is wants. Room is what the essential and
// flexible categories could save, judged by your own leaner months.
type ExpenseSplit struct {
	Fixed     CategoryMonths `json:"fixed"`
	Essential CategoryMonths `json:"essential"`
	Flexible  CategoryMonths `json:"flexible"`
	Untagged  CategoryMonths `json:"untagged"`
	// NeedsShareOfIncome is fixed plus essential expense over income in percent; nil without income.
	NeedsShareOfIncome *float64 `json:"needsShareOfIncome"`
	// NeedsShareComfortable is the share of income needs should stay under.
	NeedsShareComfortable float64 `json:"needsShareComfortable"`
	// UntaggedCategories have expense this year but no tag, largest first.
	UntaggedCategories []string `json:"untaggedCategories"`
	// Room lists essential and flexible categories that could cost less over
	// the completed months, most room first. The rows do not add up to the total: categories rarely have
	// their lean month together.
	Room []CategoryRoom `json:"room"`
	// RoomPerMonth compares whole months of essential plus flexible spending,
	// typical against lean, so the total is a month already lived.
	RoomPerMonth float64 `json:"roomPerMonth"`
	RoomPerYear  float64 `json:"roomPerYear"`
}

// CategoryRoom is how much less a category could cost: the typical (median)
// month against the lean (25th percentile) one, a level already lived on, so
// essentials are lowered rather than cut.
type CategoryRoom struct {
	Category string         `json:"category"`
	Nature   CategoryNature `json:"nature"`
	Typical  float64        `json:"typical"`
	Lean     float64        `json:"lean"`
	PerMonth float64        `json:"perMonth"`
	PerYear  float64        `json:"perYear"`
}

// NeedsShareComfortable follows the 50/30/20 rule of thumb: needs within half of income.
const NeedsShareComfortable = 50.0

// leanPercentile picks the lean month among the months with spending.
const leanPercentile = 0.25

const untaggedNature = "untagged"

// Savings rate bands for the yearly verdict, in percent of income.
const (
	SavingsRateHealthy = 20.0
	SavingsRateWatch   = 10.0
)

// Verdict statuses, best first.
const (
	VerdictOnTarget = "onTarget"
	VerdictHealthy  = "healthy"
	VerdictWatch    = "watch"
	VerdictAtRisk   = "atRisk"
	VerdictNoData   = "noData"
)

// SavingsTarget is the savings rate band the page draws: healthy from Healthy, on target from Target.
type SavingsTarget struct {
	Watch   float64 `json:"watch"`
	Healthy float64 `json:"healthy"`
	Target  float64 `json:"target"`
}

// YearVerdict grades the year by its savings rate: on target from
// TargetSavingsRate, healthy from 20%, watch from 10%, else at risk; noData
// without income. DeficitMonths counts elapsed months that spent more than they earned.
type YearVerdict struct {
	Status        string  `json:"status"`
	SavingsRate   float64 `json:"savingsRate"`
	DeficitMonths int     `json:"deficitMonths"`
}

// YearProjection is the year end at the current pace: completed months as
// recorded, the current month as at least the average, the rest at the
// average of the completed months (the current month alone in January).
type YearProjection struct {
	CashFlow
	BasisMonths int `json:"basisMonths"`
}

// ExpenseSpike is a month whose expense is at least spikeRatio times the
// median of the other months, with the category that grew the most.
type ExpenseSpike struct {
	Month           string  `json:"month"`
	Expense         float64 `json:"expense"`
	Typical         float64 `json:"typical"`
	Ratio           float64 `json:"ratio"`
	Category        string  `json:"category"`
	CategoryAmount  float64 `json:"categoryAmount"`
	CategoryTypical float64 `json:"categoryTypical"`
}

const (
	spikeRatio = 1.5
	// typicalMinMonths months with spending are needed to say what a typical month is.
	typicalMinMonths = 4
)

// CategoryMonths is one category's amount per month (January first) and its year total.
type CategoryMonths struct {
	Category string    `json:"category"`
	Values   []float64 `json:"values"`
	Total    float64   `json:"total"`
}

// ComputeYearlyCashFlow summarises income and expense for `year`. Category
// rows use the same attribution as ComputeCashFlow, so each month's rows add
// up to that month's income and expense.
func ComputeYearlyCashFlow(transactions []Transaction, categories []Category, year int, today time.Time) YearlyCashFlow {
	first := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	elapsed := monthsElapsed(year, today)
	months := MonthlyCashFlows(transactions, first, first.AddDate(0, 11, 0))
	total := sumCashFlows(strconv.Itoa(year), months)
	previous := sumCashFlows(strconv.Itoa(year-1), nil)
	if elapsed > 0 {
		previous = sumCashFlows(previous.Month, MonthlyCashFlows(transactions, first.AddDate(-1, 0, 0), first.AddDate(-1, elapsed-1, 0)))
	}
	income, expense := categoryMonths(transactions, year)
	return YearlyCashFlow{
		Year:                 year,
		Years:                cashFlowYears(transactions, year, today.Year()),
		MonthsElapsed:        elapsed,
		Months:               months,
		Total:                total,
		MonthlyAverage:       averageCashFlow(total, elapsed),
		PreviousYear:         previous,
		IncomeChangePercent:  PercentChange(total.Income, previous.Income),
		ExpenseChangePercent: PercentChange(total.Expense, previous.Expense),
		Income:               income,
		Expense:              expense,
		Verdict:              yearVerdict(total, months[:elapsed]),
		Projection:           projectYear(months, year, today),
		Spikes:               expenseSpikes(months[:elapsed], expense),
		SavingsTarget:        SavingsTarget{Watch: SavingsRateWatch, Healthy: SavingsRateHealthy, Target: TargetSavingsRate},
		ExpenseSplit:         expenseSplit(expense, categories, total.Income, months[:completedMonths(year, today)]),
	}
}

// expenseSplit sums the expense category rows by their tag, matching names
// ignoring case and spaces like the category type join.
func expenseSplit(expense []CategoryMonths, categories []Category, income float64, completed []CashFlow) ExpenseSplit {
	natures := make(map[string]CategoryNature, len(categories))
	for _, category := range categories {
		natures[strings.ToLower(strings.TrimSpace(category.Name))] = category.Nature
	}
	split := ExpenseSplit{
		Fixed:                 CategoryMonths{Category: string(NatureFixed), Values: make([]float64, 12)},
		Essential:             CategoryMonths{Category: string(NatureEssential), Values: make([]float64, 12)},
		Flexible:              CategoryMonths{Category: string(NatureFlexible), Values: make([]float64, 12)},
		Untagged:              CategoryMonths{Category: untaggedNature, Values: make([]float64, 12)},
		NeedsShareComfortable: NeedsShareComfortable,
		UntaggedCategories:    []string{},
		Room:                  []CategoryRoom{},
	}
	active := activeMonths(completed)
	for _, row := range expense {
		nature := natures[strings.ToLower(strings.TrimSpace(row.Category))]
		bucket := &split.Untagged
		switch nature {
		case NatureFixed:
			bucket = &split.Fixed
		case NatureEssential:
			bucket = &split.Essential
		case NatureFlexible:
			bucket = &split.Flexible
		default:
			split.UntaggedCategories = append(split.UntaggedCategories, row.Category)
		}
		for month, value := range row.Values {
			bucket.Values[month] += value
		}
		bucket.Total += row.Total
		if (nature == NatureEssential || nature == NatureFlexible) && len(active) >= typicalMinMonths {
			if room := categoryRoom(row, nature, active); room.PerMonth > 0 {
				split.Room = append(split.Room, room)
			}
		}
	}
	if len(active) >= typicalMinMonths {
		variable := CategoryMonths{Values: make([]float64, 12)}
		for month := range variable.Values {
			variable.Values[month] = split.Essential.Values[month] + split.Flexible.Values[month]
		}
		whole := categoryRoom(variable, "", active)
		split.RoomPerMonth, split.RoomPerYear = whole.PerMonth, whole.PerYear
	}
	sort.SliceStable(split.Room, func(i, j int) bool { return split.Room[i].PerMonth > split.Room[j].PerMonth })
	if income > 0 {
		share := round2((split.Fixed.Total + split.Essential.Total) / income * 100)
		split.NeedsShareOfIncome = &share
	}
	return split
}

func categoryRoom(row CategoryMonths, nature CategoryNature, active []int) CategoryRoom {
	values := make([]float64, len(active))
	for i, month := range active {
		values[i] = row.Values[month]
	}
	typical := math.Round(median(values))
	lean := math.Round(percentile(values, leanPercentile))
	return CategoryRoom{Category: row.Category, Nature: nature, Typical: typical, Lean: lean, PerMonth: typical - lean, PerYear: (typical - lean) * 12}
}

// activeMonths are the indexes of the months with any expense; a month with
// none is more likely unrecorded than lived on nothing.
func activeMonths(elapsed []CashFlow) []int {
	active := []int{}
	for i, month := range elapsed {
		if month.Expense > 0 {
			active = append(active, i)
		}
	}
	return active
}

// percentile interpolates linearly between the closest ranks.
func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	rank := p * float64(len(sorted)-1)
	low := int(math.Floor(rank))
	high := int(math.Ceil(rank))
	return sorted[low] + (sorted[high]-sorted[low])*(rank-float64(low))
}

func yearVerdict(total CashFlow, elapsed []CashFlow) YearVerdict {
	verdict := YearVerdict{Status: VerdictNoData}
	for _, month := range elapsed {
		if month.CashFlow < 0 {
			verdict.DeficitMonths++
		}
	}
	if total.Income <= 0 {
		return verdict
	}
	verdict.SavingsRate = total.SavingsRate
	switch rate := total.SavingsRate; {
	case rate >= TargetSavingsRate:
		verdict.Status = VerdictOnTarget
	case rate >= SavingsRateHealthy:
		verdict.Status = VerdictHealthy
	case rate >= SavingsRateWatch:
		verdict.Status = VerdictWatch
	default:
		verdict.Status = VerdictAtRisk
	}
	return verdict
}

func projectYear(months []CashFlow, year int, today time.Time) *YearProjection {
	if year != today.Year() {
		return nil
	}
	current := int(today.Month()) - 1
	completed := months[:current]
	pace := sumCashFlows("", completed)
	if current > 0 {
		pace.Income /= float64(current)
		pace.Expense /= float64(current)
	} else {
		pace = months[current]
	}
	remaining := float64(11 - current)
	projected := sumCashFlows(strconv.Itoa(year), completed)
	projected.Income += math.Max(months[current].Income, pace.Income) + pace.Income*remaining
	projected.Expense += math.Max(months[current].Expense, pace.Expense) + pace.Expense*remaining
	return &YearProjection{CashFlow: withDerived(projected), BasisMonths: current}
}

// expenseSpikes compares each month with spending against the median of the
// other such months; it needs typicalMinMonths of them to say what is typical.
func expenseSpikes(elapsed []CashFlow, categories []CategoryMonths) []ExpenseSpike {
	spikes := []ExpenseSpike{}
	active := activeMonths(elapsed)
	if len(active) < typicalMinMonths {
		return spikes
	}
	othersMedian := func(i int, value func(int) float64) float64 {
		values := make([]float64, 0, len(active)-1)
		for _, j := range active {
			if j != i {
				values = append(values, value(j))
			}
		}
		return median(values)
	}
	for _, i := range active {
		typical := othersMedian(i, func(j int) float64 { return elapsed[j].Expense })
		if typical <= 0 || elapsed[i].Expense < typical*spikeRatio {
			continue
		}
		spike := ExpenseSpike{Month: elapsed[i].Month, Expense: elapsed[i].Expense, Typical: typical, Ratio: round2(elapsed[i].Expense / typical)}
		growth := math.Inf(-1)
		for _, row := range categories {
			rowTypical := othersMedian(i, func(j int) float64 { return row.Values[j] })
			if delta := row.Values[i] - rowTypical; delta > growth {
				growth = delta
				spike.Category, spike.CategoryAmount, spike.CategoryTypical = row.Category, row.Values[i], rowTypical
			}
		}
		spikes = append(spikes, spike)
	}
	return spikes
}

func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

// completedMonths leaves out the month under way: half a month looks lean.
func completedMonths(year int, today time.Time) int {
	if year == today.Year() {
		return int(today.Month()) - 1
	}
	return monthsElapsed(year, today)
}

func monthsElapsed(year int, today time.Time) int {
	switch {
	case year < today.Year():
		return 12
	case year == today.Year():
		return int(today.Month())
	default:
		return 0
	}
}

// sumCashFlows totals the months under `label` and derives the rate from the totals.
func sumCashFlows(label string, months []CashFlow) CashFlow {
	total := CashFlow{Month: label}
	for _, month := range months {
		total.Income += month.Income
		total.Expense += month.Expense
	}
	return withDerived(total)
}

func averageCashFlow(total CashFlow, months int) CashFlow {
	if months == 0 {
		return CashFlow{Month: total.Month}
	}
	return withDerived(CashFlow{Month: total.Month, Income: total.Income / float64(months), Expense: total.Expense / float64(months)})
}

func withDerived(flow CashFlow) CashFlow {
	flow.CashFlow = flow.Income - flow.Expense
	flow.SavingsRate = 0
	if flow.Income > 0 {
		flow.SavingsRate = round2(flow.CashFlow / flow.Income * 100)
	}
	return flow
}

// categoryMonths splits the year's income deposits and expense payments by
// category, largest year total first.
func categoryMonths(transactions []Transaction, year int) (income []CategoryMonths, expense []CategoryMonths) {
	incomeRows := map[string]*CategoryMonths{}
	expenseRows := map[string]*CategoryMonths{}
	for _, t := range transactions {
		date := EffectiveDate(t)
		if !t.Usable() || date.Year() != year {
			continue
		}
		rows, amount := incomeRows, t.DepositAmount()
		switch {
		case t.CountsAsIncome():
		case t.CountsAsExpense():
			rows, amount = expenseRows, t.PaymentAmount()
		default:
			continue
		}
		row, ok := rows[t.Category]
		if !ok {
			row = &CategoryMonths{Category: t.Category, Values: make([]float64, 12)}
			rows[t.Category] = row
		}
		row.Values[date.Month()-1] += amount
		row.Total += amount
	}
	return rankedCategoryMonths(incomeRows), rankedCategoryMonths(expenseRows)
}

func rankedCategoryMonths(rows map[string]*CategoryMonths) []CategoryMonths {
	ranked := make([]CategoryMonths, 0, len(rows))
	for _, row := range rows {
		if row.Total != 0 {
			ranked = append(ranked, *row)
		}
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].Total != ranked[j].Total {
			return ranked[i].Total > ranked[j].Total
		}
		return ranked[i].Category < ranked[j].Category
	})
	return ranked
}

// cashFlowYears lists the years with usable income or expense, newest first,
// always including the current and the requested year.
func cashFlowYears(transactions []Transaction, requested int, current int) []int {
	seen := map[int]bool{requested: true, current: true}
	for _, t := range transactions {
		if t.Usable() && (t.CountsAsIncome() || t.CountsAsExpense()) {
			seen[EffectiveDate(t).Year()] = true
		}
	}
	years := make([]int, 0, len(seen))
	for year := range seen {
		years = append(years, year)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(years)))
	return years
}
