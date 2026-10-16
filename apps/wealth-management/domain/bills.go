package domain

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// Bills and safe-to-spend (ROADMAP R4, WM-BILL-3..6): the confirmed
// recurring items matched against the ledger, a 30-day calendar with the
// card payments, and how much can be spent until the next income.

const (
	// BillsCalendarDays is how far ahead the calendar looks.
	BillsCalendarDays = 30
	// safeToSpendDefaultDays is the horizon when no income is expected.
	safeToSpendDefaultDays = 30
	// incomeSearchDays bounds the search for the next income (a yearly
	// income can be up to a year away).
	incomeSearchDays = 366
)

// BillStatus is where an occurrence stands.
type BillStatus string

const (
	// BillPaid: a matching transaction settled it (received, for income).
	BillPaid BillStatus = "paid"
	// BillDue: not paid yet and its date is today or later.
	BillDue BillStatus = "due"
	// BillOverdue: not paid and its date has passed (late, for income).
	BillOverdue BillStatus = "overdue"
)

// BillSource says where an occurrence comes from.
type BillSource string

const (
	BillFromRecurring BillSource = "recurring"
	// BillFromCard is a card payment from the card statement (WM-CARD-15).
	BillFromCard BillSource = "card"
)

// BillMatch is a transaction recognised as a payment of an item.
type BillMatch struct {
	TransactionID string  `json:"transactionId"`
	Date          string  `json:"date"`
	Amount        float64 `json:"amount"`
	Payee         string  `json:"payee"`
	Account       string  `json:"account"`
}

// BillOccurrence is one expected payment (or income) on the calendar.
type BillOccurrence struct {
	// ItemID is the Recurring row, empty for a card payment without one.
	ItemID         string        `json:"itemId,omitempty"`
	Name           string        `json:"name"`
	Kind           RecurringKind `json:"kind"`
	Source         BillSource    `json:"source"`
	ExpectedDate   string        `json:"expectedDate"`
	ExpectedAmount float64       `json:"expectedAmount"`
	Status         BillStatus    `json:"status"`
	// DaysUntil is negative once the date has passed.
	DaysUntil int    `json:"daysUntil"`
	Account   string `json:"account,omitempty"`
	Note      string `json:"note,omitempty"`
	// PaidAmount is the total of the matching transactions (for a card,
	// the repayments since the statement closed).
	PaidAmount float64     `json:"paidAmount"`
	PaidDate   string      `json:"paidDate,omitempty"`
	Matches    []BillMatch `json:"matches,omitempty"`
	// Remaining is what is still to pay (or receive): 0 once paid.
	Remaining float64 `json:"remaining"`
	// Estimate is true when the amount can still change: a card statement
	// whose cycle has not closed, or a card balance without statement
	// settings.
	Estimate bool `json:"estimate,omitempty"`
}

// IsIncome reports whether the occurrence brings money in.
func (o BillOccurrence) IsIncome() bool { return o.Kind.IsIncome() }

// BillItem is a Recurring row with its current cycle and last payment.
type BillItem struct {
	RecurringItem
	// Current is the occurrence whose cycle contains today; nil when
	// today falls between the cycles of a quarterly or yearly item.
	Current *BillOccurrence `json:"current"`
	// NextDate is the next expected date on or after today, yyyy-mm-dd.
	NextDate string `json:"nextDate,omitempty"`
	// LastMatch is the latest transaction that matched the item.
	LastMatch *BillMatch `json:"lastMatch"`
	// MatchCount counts matching transactions in the last 12 months.
	MatchCount int `json:"matchCount"`
}

// BillCalendar lists the occurrences from the first of this month to
// BillsCalendarDays days ahead, oldest first.
type BillCalendar struct {
	From        string           `json:"from"`
	To          string           `json:"to"`
	Occurrences []BillOccurrence `json:"occurrences"`
	// UpcomingOut is what is still to pay from today to To (bills and
	// cards); UpcomingIn what is still to be received.
	UpcomingOut   float64 `json:"upcomingOut"`
	UpcomingCount int     `json:"upcomingCount"`
	UpcomingIn    float64 `json:"upcomingIn"`
	// Overdue counts unpaid bills whose date has passed this cycle.
	OverdueCount  int     `json:"overdueCount"`
	OverdueAmount float64 `json:"overdueAmount"`
	PaidOut       float64 `json:"paidOut"`
}

// SpendableAccount is an account counted in spendable cash.
type SpendableAccount struct {
	Name    string  `json:"name"`
	Balance float64 `json:"balance"`
}

// SafeToSpend is the money free to spend until the next income:
// spendable cash - unpaid bills and card payments due by then - this
// month's goal set-aside. Amount can be negative; SafeToSpend floors it at
// 0 and ShortBy says by how much it fell below.
type SafeToSpend struct {
	AsOf string `json:"asOf"`
	// Until is the horizon, yyyy-mm-dd: the next expected income, else
	// 30 days ahead (UntilIncome false).
	Until       string `json:"until"`
	UntilIncome bool   `json:"untilIncome"`
	IncomeName  string `json:"incomeName,omitempty"`
	Days        int    `json:"days"`
	// SpendableCash is the positive balances of active, rarely used, bank
	// and cash accounts (the liquid balance, WM-HEA-2).
	SpendableCash    float64            `json:"spendableCash"`
	CashAccounts     []SpendableAccount `json:"cashAccounts"`
	BillsDue         float64            `json:"billsDue"`
	CardPayments     float64            `json:"cardPayments"`
	GoalSetAside     float64            `json:"goalSetAside"`
	GoalPlanUsed     bool               `json:"goalPlanUsed"`
	Amount           float64            `json:"amount"`
	SafeToSpend      float64            `json:"safeToSpend"`
	ShortBy          float64            `json:"shortBy"`
	PerDay           float64            `json:"perDay"`
	Counted          []BillOccurrence   `json:"counted"`
	Explanation      string             `json:"explanation"`
	GoalSetAsideRule string             `json:"goalSetAsideRule"`
}

// Bills is the bills page: items, suggestions, calendar and safe-to-spend.
type Bills struct {
	AsOf        string                `json:"asOf"`
	TabExists   bool                  `json:"tabExists"`
	Items       []BillItem            `json:"items"`
	Issues      []RecurringIssue      `json:"issues"`
	Suggestions []RecurringSuggestion `json:"suggestions"`
	Calendar    BillCalendar          `json:"calendar"`
	SafeToSpend SafeToSpend           `json:"safeToSpend"`
}

// BillsFacts is the nameless summary for the AI briefing.
type BillsFacts struct {
	Items         int     `json:"items"`
	UpcomingOut   float64 `json:"upcoming30Days"`
	UpcomingCount int     `json:"upcomingCount"`
	OverdueCount  int     `json:"overdueCount"`
	OverdueAmount float64 `json:"overdueAmount"`
	SafeToSpend   float64 `json:"safeToSpend"`
	ShortBy       float64 `json:"shortBy"`
	Until         string  `json:"until"`
	UntilIncome   bool    `json:"untilIncome"`
	Suggestions   int     `json:"suggestions"`
}

// Facts summarises the bills without names.
func (b Bills) Facts() BillsFacts {
	active := 0
	for _, item := range b.Items {
		if item.Status == RecurringActive {
			active++
		}
	}
	return BillsFacts{
		Items:         active,
		UpcomingOut:   b.Calendar.UpcomingOut,
		UpcomingCount: b.Calendar.UpcomingCount,
		OverdueCount:  b.Calendar.OverdueCount,
		OverdueAmount: b.Calendar.OverdueAmount,
		SafeToSpend:   b.SafeToSpend.SafeToSpend,
		ShortBy:       b.SafeToSpend.ShortBy,
		Until:         b.SafeToSpend.Until,
		UntilIncome:   b.SafeToSpend.UntilIncome,
		Suggestions:   len(b.Suggestions),
	}
}

// BillsInput is what ComputeBills needs. Goals may be nil (no goal plan).
type BillsInput struct {
	Sheet        RecurringSheet
	Accounts     []Account
	Transactions []Transaction
	Cards        CardConfig
	Goals        *GoalPlan
	Today        time.Time
}

// GoalSetAsideRule documents the goal part of safe-to-spend.
const GoalSetAsideRule = "The full monthly amount the goal plan requires (dated active goals), " +
	"not prorated: goal money is put aside once a month, whatever the day."

// ComputeBills matches the items, builds the calendar and safe-to-spend
// and suggests new items.
func ComputeBills(input BillsInput) Bills {
	today := dateOnly(input.Today)
	monthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	calendarEnd := today.AddDate(0, 0, BillsCalendarDays)
	bills := Bills{
		AsOf:        today.Format(time.DateOnly),
		TabExists:   input.Sheet.Exists,
		Items:       []BillItem{},
		Issues:      append([]RecurringIssue{}, input.Sheet.Issues...),
		Suggestions: DetectRecurring(input.Transactions, input.Sheet.Items, today),
	}

	var active []RecurringItem
	for _, item := range input.Sheet.Items {
		bills.Items = append(bills.Items, describeItem(item, input.Transactions, today))
		if item.Status == RecurringActive && item.Kind != RecurringCard {
			active = append(active, item)
		}
	}

	cards, cardIssues := cardPayments(input, today)
	bills.Issues = append(bills.Issues, cardIssues...)

	occurrences := []BillOccurrence{}
	for _, item := range active {
		for _, date := range item.occurrencesBetween(monthStart, calendarEnd) {
			occurrences = append(occurrences, occurrenceOf(item, date, input.Transactions, today))
		}
	}
	for _, card := range cards {
		if expected, _ := time.Parse(time.DateOnly, card.ExpectedDate); !expected.After(calendarEnd) {
			occurrences = append(occurrences, card)
		}
	}
	sortOccurrences(occurrences)
	bills.Calendar = summarizeCalendar(occurrences, monthStart, calendarEnd, today)
	bills.SafeToSpend = computeSafeToSpend(input, active, cards, today)
	return bills
}

// describeItem gives an item its current cycle, next date and last match.
func describeItem(item RecurringItem, transactions []Transaction, today time.Time) BillItem {
	described := BillItem{RecurringItem: item}
	yearAgo := today.AddDate(-1, 0, 0)
	for _, t := range transactions {
		if t.Date.After(today) || !item.Matches(t) {
			continue
		}
		if !t.Date.Before(yearAgo) {
			described.MatchCount++
		}
		if described.LastMatch == nil || !t.Date.Before(parseDay(described.LastMatch.Date)) {
			match := matchOf(item, t)
			described.LastMatch = &match
		}
	}
	if item.Kind == RecurringCard {
		return described
	}
	// The cycle containing today: this month's date for monthly items,
	// the nearest date whose window holds today for the others.
	for _, date := range item.occurrencesBetween(today.AddDate(0, -2, 0), today.AddDate(0, 2, 0)) {
		start, end := item.cycleWindow(date)
		if !today.Before(start) && today.Before(end) {
			current := occurrenceOf(item, date, transactions, today)
			described.Current = &current
			break
		}
	}
	for _, date := range item.occurrencesBetween(today, today.AddDate(0, 0, incomeSearchDays)) {
		described.NextDate = date.Format(time.DateOnly)
		break
	}
	return described
}

func parseDay(value string) time.Time {
	date, _ := time.Parse(time.DateOnly, value)
	return date
}

func matchOf(item RecurringItem, t Transaction) BillMatch {
	return BillMatch{TransactionID: t.ID, Date: dateOnly(t.Date).Format(time.DateOnly), Amount: item.flowAmount(t), Payee: t.Payee, Account: t.AccountName}
}

// occurrenceOf settles the occurrence due on date with the matching
// transactions in its cycle window.
func occurrenceOf(item RecurringItem, date time.Time, transactions []Transaction, today time.Time) BillOccurrence {
	expected := 0.0
	if item.Amount != nil {
		expected = *item.Amount
	}
	occurrence := BillOccurrence{
		ItemID: item.ID, Name: item.Name, Kind: item.Kind, Source: BillFromRecurring,
		ExpectedDate: date.Format(time.DateOnly), ExpectedAmount: expected,
		DaysUntil: daysBetween(today, date), Account: item.Account, Note: item.Note,
	}
	start, end := item.cycleWindow(date)
	for _, t := range transactions {
		if t.Date.Before(start) || !t.Date.Before(end) || t.Date.After(today) || !item.Matches(t) {
			continue
		}
		match := matchOf(item, t)
		occurrence.Matches = append(occurrence.Matches, match)
		occurrence.PaidAmount += match.Amount
		if match.Date > occurrence.PaidDate {
			occurrence.PaidDate = match.Date
		}
	}
	switch {
	case len(occurrence.Matches) > 0:
		occurrence.Status = BillPaid
	case date.Before(today):
		occurrence.Status = BillOverdue
	default:
		occurrence.Status = BillDue
	}
	if occurrence.Status != BillPaid {
		occurrence.Remaining = expected
	}
	return occurrence
}

func daysBetween(from, to time.Time) int {
	return int(math.Round(to.Sub(from).Hours() / 24))
}

// cardPayments returns one payment per credit card account due on the next
// pay-by date (D26): the statements of the held cards billed to it
// (WM-CARD-15), else the balance owed for a card account without card
// settings. A `card` Recurring row naming the account (Account, else Name)
// renames the payment and adds its note; a paused one hides it.
func cardPayments(input BillsInput, today time.Time) ([]BillOccurrence, []RecurringIssue) {
	payBy := PayByDate(today)
	type billed struct {
		amount  float64
		open    bool
		closed  time.Time
		hasCard bool
	}
	byAccount := map[string]*billed{}
	order := []string{}
	add := func(account string) *billed {
		if _, ok := byAccount[account]; !ok {
			byAccount[account] = &billed{}
			order = append(order, account)
		}
		return byAccount[account]
	}
	for _, bank := range ComputeCardReports(input.Cards, input.Accounts, input.Transactions, today) {
		for _, card := range bank.Cards {
			if card.Statement == nil || card.AccountName == "" {
				continue
			}
			entry := add(card.AccountName)
			entry.hasCard = true
			entry.amount += card.Statement.Amount
			entry.open = entry.open || card.Statement.Open
			if end := parseDay(card.Statement.End); end.After(entry.closed) {
				entry.closed = end
			}
		}
	}
	for _, account := range input.Accounts {
		if account.Type != AccountTypeNegativeActive {
			continue
		}
		if _, ok := byAccount[account.Name]; ok {
			continue
		}
		if owed := owedAmount(account.Balance); owed > 0 {
			entry := add(account.Name)
			entry.amount = owed
			entry.open = true
		}
	}

	cardRows := map[string]RecurringItem{}
	var issues []RecurringIssue
	for _, item := range input.Sheet.Items {
		if item.Kind != RecurringCard {
			continue
		}
		key := FoldText(item.Account)
		if key == "" {
			key = FoldText(item.Name)
		}
		found := false
		for _, account := range input.Accounts {
			if account.Type == AccountTypeNegativeActive && FoldText(account.Name) == key {
				cardRows[account.Name] = item
				found = true
			}
		}
		if !found {
			issues = append(issues, RecurringIssue{Row: item.Row, Message: fmt.Sprintf("card row %q names no credit card account", item.Name)})
		}
	}

	payments := []BillOccurrence{}
	for _, account := range order {
		entry := byAccount[account]
		row, hasRow := cardRows[account]
		if hasRow && row.Status == RecurringPaused {
			continue
		}
		if entry.amount <= 0 {
			continue
		}
		payment := BillOccurrence{
			Name: account, Kind: RecurringCard, Source: BillFromCard,
			ExpectedDate: payBy.Format(time.DateOnly), ExpectedAmount: math.Round(entry.amount),
			DaysUntil: daysBetween(today, payBy), Account: account, Estimate: entry.open,
		}
		if !entry.hasCard {
			payment.Note = "No card settings: the balance owed is used"
		}
		if hasRow {
			payment.ItemID, payment.Name = row.ID, row.Name
			if row.Note != "" {
				payment.Note = row.Note
			}
		}
		if entry.hasCard && !entry.open {
			for _, t := range input.Transactions {
				if t.AccountName != account || !t.Usable() || t.IsLending() || t.DepositAmount() <= 0 || IsCashbackRefund(t) || IsRefundCategory(t.Category) {
					continue
				}
				if t.Date.After(entry.closed) && !t.Date.After(today) {
					payment.PaidAmount += t.DepositAmount()
					payment.Matches = append(payment.Matches, BillMatch{TransactionID: t.ID, Date: dateOnly(t.Date).Format(time.DateOnly), Amount: t.DepositAmount(), Payee: t.Payee, Account: t.AccountName})
					if date := dateOnly(t.Date).Format(time.DateOnly); date > payment.PaidDate {
						payment.PaidDate = date
					}
				}
			}
		}
		payment.Remaining = math.Max(0, payment.ExpectedAmount-payment.PaidAmount)
		// Never more than the account owes: money paid in ahead settles it.
		for _, held := range input.Accounts {
			if held.Name == account {
				payment.Remaining = math.Min(payment.Remaining, math.Round(owedAmount(held.Balance)))
			}
		}
		if payment.Remaining == 0 {
			payment.Status = BillPaid
		} else {
			payment.Status = BillDue
		}
		payments = append(payments, payment)
	}
	return payments, issues
}

func sortOccurrences(occurrences []BillOccurrence) {
	sort.SliceStable(occurrences, func(i, j int) bool {
		if occurrences[i].ExpectedDate != occurrences[j].ExpectedDate {
			return occurrences[i].ExpectedDate < occurrences[j].ExpectedDate
		}
		return occurrences[i].Name < occurrences[j].Name
	})
}

func summarizeCalendar(occurrences []BillOccurrence, from, to, today time.Time) BillCalendar {
	calendar := BillCalendar{From: from.Format(time.DateOnly), To: to.Format(time.DateOnly), Occurrences: occurrences}
	for _, o := range occurrences {
		switch {
		case o.Status == BillPaid && !o.IsIncome():
			calendar.PaidOut += o.PaidAmount
		case o.Status == BillOverdue && !o.IsIncome():
			calendar.OverdueCount++
			calendar.OverdueAmount += o.Remaining
		case o.Status == BillDue && o.IsIncome():
			calendar.UpcomingIn += o.Remaining
		case o.Status == BillDue:
			calendar.UpcomingOut += o.Remaining
			calendar.UpcomingCount++
		}
	}
	return calendar
}

// computeSafeToSpend: horizon = the next unpaid income on or after today
// (else 30 days); bills counted = unpaid outgoing occurrences from the
// first of this month up to and including the horizon (overdue ones too)
// and the card payments due by then.
func computeSafeToSpend(input BillsInput, active []RecurringItem, cards []BillOccurrence, today time.Time) SafeToSpend {
	monthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	result := SafeToSpend{AsOf: today.Format(time.DateOnly), CashAccounts: []SpendableAccount{}, Counted: []BillOccurrence{}, GoalSetAsideRule: GoalSetAsideRule}

	horizon := today.AddDate(0, 0, safeToSpendDefaultDays)
	for _, item := range active {
		if !item.Kind.IsIncome() {
			continue
		}
		for _, date := range item.occurrencesBetween(today, today.AddDate(0, 0, incomeSearchDays)) {
			if occurrenceOf(item, date, input.Transactions, today).Status == BillPaid {
				continue
			}
			if !result.UntilIncome || date.Before(horizon) {
				horizon, result.UntilIncome, result.IncomeName = date, true, item.Name
			}
			break
		}
	}
	result.Until = horizon.Format(time.DateOnly)
	result.Days = max(1, daysBetween(today, horizon))

	for _, account := range input.Accounts {
		switch account.Type {
		case AccountTypeActiveUse, AccountTypeRarelyUse, AccountTypeBank, AccountTypeCash:
			if account.Balance > 0 {
				result.CashAccounts = append(result.CashAccounts, SpendableAccount{Name: account.Name, Balance: account.Balance})
			}
		}
	}
	result.SpendableCash = liquidBalance(input.Accounts)

	for _, item := range active {
		if item.Kind.IsIncome() {
			continue
		}
		for _, date := range item.occurrencesBetween(monthStart, horizon) {
			occurrence := occurrenceOf(item, date, input.Transactions, today)
			if occurrence.Status != BillPaid {
				result.BillsDue += occurrence.Remaining
				result.Counted = append(result.Counted, occurrence)
			}
		}
	}
	for _, card := range cards {
		if card.Status != BillPaid && !parseDay(card.ExpectedDate).After(horizon) {
			result.CardPayments += card.Remaining
			result.Counted = append(result.Counted, card)
		}
	}
	sortOccurrences(result.Counted)

	if input.Goals != nil && !input.Goals.SetupNeeded {
		result.GoalPlanUsed = true
		result.GoalSetAside = math.Round(input.Goals.Check.RequiredPerMonth)
	}
	result.Amount = math.Round(result.SpendableCash - result.BillsDue - result.CardPayments - result.GoalSetAside)
	result.SafeToSpend = math.Max(0, result.Amount)
	result.ShortBy = math.Max(0, -result.Amount)
	result.PerDay = math.Floor(result.SafeToSpend / float64(result.Days))
	until := "in the next 30 days (no income expected)"
	if result.UntilIncome {
		until = "up to the next income (" + result.IncomeName + ", " + result.Until + ")"
	}
	result.Explanation = fmt.Sprintf("Spendable cash minus unpaid bills and card payments due %s, minus this month's goal set-aside.", until)
	return result
}

// billsDueSoonDays is the window of BillsSummary.DueSoon.
const billsDueSoonDays = 7

// BillsSummary is the short form for the home page (ROADMAP R5):
// safe-to-spend with its math, the bills facts, overdue bills and the
// unpaid bills and card payments due within a week.
type BillsSummary struct {
	SafeToSpend SafeToSpend      `json:"safeToSpend"`
	Facts       BillsFacts       `json:"facts"`
	Overdue     []BillOccurrence `json:"overdue"`
	DueSoon     []BillOccurrence `json:"dueSoon"`
}

// Summary keeps the headline parts of the bills.
func (b Bills) Summary() BillsSummary {
	summary := BillsSummary{SafeToSpend: b.SafeToSpend, Facts: b.Facts(), Overdue: []BillOccurrence{}, DueSoon: []BillOccurrence{}}
	for _, o := range b.Calendar.Occurrences {
		switch {
		case o.IsIncome():
		case o.Status == BillOverdue:
			summary.Overdue = append(summary.Overdue, o)
		case o.Status == BillDue && o.DaysUntil <= billsDueSoonDays:
			summary.DueSoon = append(summary.DueSoon, o)
		}
	}
	return summary
}
