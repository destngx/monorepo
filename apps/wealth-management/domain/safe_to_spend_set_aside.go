package domain

import (
	"cmp"
	"math"
	"slices"
	"time"
)

// Money set aside before safe-to-spend (D36): ready money is split into
// committed (bills, card payments and the fixed costs still to pay), the
// essentials until the horizon and what is free to spend. Fixed and
// essential are category natures (Categories column B); untagged and
// flexible categories are never set aside.

// SetAsideBasis says where a set-aside amount comes from.
type SetAsideBasis string

const (
	// BasisBudget: the category's monthly budget limit.
	BasisBudget SetAsideBasis = "budget"
	// BasisAverage: the category's spending over the last completed months.
	BasisAverage SetAsideBasis = "average"
)

// setAsideWindowMonths is the history behind the averages, as monthly needs.
const setAsideWindowMonths = needsWindowMonths

// SafeToSpendSetAsideRule documents what is set aside before free money.
const SafeToSpendSetAsideRule = "Committed money cannot be spent: unpaid bills and card payments due before the next " +
	"income, plus the fixed costs still to pay (categories tagged fixed, such as rent or insurance). A fixed category " +
	"expects its monthly budget limit, else its average monthly spending over the last 6 completed months, for each " +
	"month the window touches; what is already spent that month and its unpaid recurring bills that month are taken " +
	"off, so nothing counts twice, and a fixed cost is held in full since its day is unknown. Essentials (categories " +
	"tagged essential, such as groceries or transport) are money for needs only: each category's average daily spending " +
	"over the last 6 completed months (recurring bills left out, they are counted already), else its monthly budget " +
	"spread over the month, times the days to the horizon. Free to spend = ready money - committed - essentials, " +
	"floored at 0: money for flexible spending. Untagged and flexible categories are never set aside."

// FixedCost is a fixed category's cost still to pay in one month of the
// window: max(0, Expected - Spent - InBills).
type FixedCost struct {
	Category string `json:"category"`
	// Month is yyyy-mm.
	Month    string        `json:"month"`
	Expected float64       `json:"expected"`
	Basis    SetAsideBasis `json:"basis"`
	// Spent is the category's spending that month so far.
	Spent float64 `json:"spent"`
	// InBills is its unpaid recurring bills that month: counted in the
	// bills when due before the horizon, paid by the wage when after it.
	InBills float64 `json:"inBills"`
	Amount  float64 `json:"amount"`
}

// EssentialNeed is an essential category's spending until the horizon.
type EssentialNeed struct {
	Category string        `json:"category"`
	PerDay   float64       `json:"perDay"`
	Basis    SetAsideBasis `json:"basis"`
	Days     int           `json:"days"`
	Amount   float64       `json:"amount"`
}

// setAside fills the fixed costs, the essentials and the tiers of result,
// whose horizon, ready money and bills are set.
func setAside(result *SafeToSpend, input BillsInput, active []RecurringItem, today, lastDay time.Time) {
	result.Fixed, result.Essential = []FixedCost{}, []EssentialNeed{}
	fixed, essential := []Category{}, []Category{}
	for _, category := range input.Categories {
		if category.Type != CategoryTypeExpense {
			continue
		}
		switch category.Nature {
		case NatureFixed:
			fixed = append(fixed, category)
		case NatureEssential:
			essential = append(essential, category)
		}
	}
	result.NatureTagged = len(fixed)+len(essential) > 0

	history := windowMonths(input.Transactions, today, setAsideWindowMonths)
	categories := itemCategories(active, input.Transactions, today)
	for _, category := range fixed {
		for month := monthOf(today); !month.After(lastDay); month = month.AddDate(0, 1, 0) {
			if line, ok := fixedCost(category.Name, month, input, active, categories, history, today); ok {
				result.Fixed = append(result.Fixed, line)
				result.FixedCosts += line.Amount
			}
		}
	}
	for _, category := range essential {
		if line, ok := essentialNeed(category.Name, input, active, history, today, result.Days); ok {
			result.Essential = append(result.Essential, line)
			result.Essentials += line.Amount
		}
	}
	slices.SortStableFunc(result.Fixed, func(a, b FixedCost) int {
		return cmp.Or(cmp.Compare(a.Month, b.Month), cmp.Compare(b.Amount, a.Amount), cmp.Compare(a.Category, b.Category))
	})
	slices.SortStableFunc(result.Essential, func(a, b EssentialNeed) int {
		return cmp.Or(cmp.Compare(b.Amount, a.Amount), cmp.Compare(a.Category, b.Category))
	})
}

// fixedCost is a fixed category's cost still to pay in month; false when
// nothing is expected (no budget and no history).
func fixedCost(category string, month time.Time, input BillsInput, active []RecurringItem, categories map[string]string, history []time.Time, today time.Time) (FixedCost, bool) {
	line := FixedCost{Category: category, Month: MonthKey(month), Basis: BasisBudget}
	line.Expected = budgetLimit(input.Budgets, category, month)
	if line.Expected <= 0 {
		line.Basis = BasisAverage
		line.Expected = average(categorySpend(input.Transactions, category, history, nil), len(history))
	}
	if line.Expected <= 0 {
		return FixedCost{}, false
	}
	line.Spent = math.Round(clampZero(categorySpend(input.Transactions, category, []time.Time{month}, nil)))
	monthEnd := month.AddDate(0, 1, -1)
	for _, item := range active {
		if item.Kind.IsIncome() || categories[item.ID] != categoryKey(category) {
			continue
		}
		for _, date := range item.occurrencesBetween(month, monthEnd) {
			if occurrence := occurrenceOf(item, date, input.Transactions, today); occurrence.Status != BillPaid {
				line.InBills += occurrence.Remaining
			}
		}
	}
	line.Amount = math.Round(math.Max(0, line.Expected-line.Spent-line.InBills))
	return line, true
}

// essentialNeed is an essential category's spending for days; false when
// it has neither history nor a budget.
func essentialNeed(category string, input BillsInput, active []RecurringItem, history []time.Time, today time.Time, days int) (EssentialNeed, bool) {
	line := EssentialNeed{Category: category, Basis: BasisAverage, Days: days}
	historyDays := 0
	for _, month := range history {
		historyDays += DaysInMonth(month)
	}
	// Recurring bills are counted on their own dates, so they do not set the pace.
	isBill := func(t Transaction) bool {
		return slices.ContainsFunc(active, func(item RecurringItem) bool { return !item.Kind.IsIncome() && item.Matches(t) })
	}
	perDay := 0.0
	if spent := categorySpend(input.Transactions, category, history, isBill); historyDays > 0 && spent > 0 {
		perDay = spent / float64(historyDays)
	} else if limit := budgetLimit(input.Budgets, category, today); limit > 0 {
		line.Basis = BasisBudget
		perDay = limit / float64(DaysInMonth(today))
	}
	if perDay <= 0 {
		return EssentialNeed{}, false
	}
	line.PerDay = math.Round(perDay)
	line.Amount = math.Round(perDay * float64(days))
	return line, true
}

// categorySpend is the category's net spending (payment - deposit) in the
// months, leaving out the transactions skip reports.
func categorySpend(transactions []Transaction, category string, months []time.Time, skip func(Transaction) bool) float64 {
	key := categoryKey(category)
	total := 0.0
	for _, t := range transactions {
		if !t.Usable() || t.IsLending() || categoryKey(t.Category) != key || !inMonths(EffectiveDate(t), months) {
			continue
		}
		if skip != nil && skip(t) {
			continue
		}
		total += t.PaymentAmount() - t.DepositAmount()
	}
	return total
}

// budgetLimit is the category's budget limit for month (0 without one).
func budgetLimit(budgets map[int][]BudgetItem, category string, month time.Time) float64 {
	for _, item := range budgets[month.Year()] {
		if categoryKey(item.Category) == categoryKey(category) {
			return item.LimitFor(month)
		}
	}
	return 0
}

// itemCategories maps each item ID to its category key: the item's own,
// else the category of its latest matching payment up to today.
func itemCategories(items []RecurringItem, transactions []Transaction, today time.Time) map[string]string {
	categories := map[string]string{}
	for _, item := range items {
		if item.Category != "" {
			categories[item.ID] = categoryKey(item.Category)
			continue
		}
		var latest time.Time
		for _, t := range transactions {
			if !t.Date.After(today) && t.Category != "" && item.Matches(t) && !t.Date.Before(latest) {
				latest, categories[item.ID] = t.Date, categoryKey(t.Category)
			}
		}
	}
	return categories
}

func monthOf(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, time.UTC)
}
