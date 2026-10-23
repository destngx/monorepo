package domain

import (
	"math"
	"slices"
	"sort"
	"strings"
	"time"
)

// Money lent (WM-LEND-1..5, D30). Lending is recorded as transactions in a
// category whose nature is lending: the payee is the person, a payment is
// money lent out and a deposit is money repaid. A deposit before any lending
// means the user borrowed, so a negative outstanding is owed by the user.

// unknownPerson names lending rows without a payee.
const unknownPerson = "Unknown"

// settledTolerance treats sub-dong leftovers (USD conversions) as settled.
const settledTolerance = 0.5

// MoneyLentPerson is one person's lending ledger.
type MoneyLentPerson struct {
	// Name is the most recent spelling of the payee.
	Name   string  `json:"name"`
	Lent   float64 `json:"lent"`
	Repaid float64 `json:"repaid"`
	// Outstanding is lent - repaid: positive is owed to the user, negative
	// is owed by the user, 0 is settled.
	Outstanding float64 `json:"outstanding"`
	// LastActivity is the date of the latest transaction, yyyy-mm-dd.
	LastActivity string `json:"lastActivity"`
	// OpenSince is the date the current open balance started; nil when settled.
	OpenSince       *string `json:"openSince"`
	DaysOutstanding *int    `json:"daysOutstanding"`
	// RepayBy is the soonest repayment date of the open loans (WM-LEND-6,
	// WM-LEND-7), yyyy-mm-dd; nil when none has one.
	RepayBy *string `json:"repayBy"`
	// DaysUntilRepay counts days from today to RepayBy; negative when overdue.
	DaysUntilRepay *int `json:"daysUntilRepay"`
	// RepayByAmount is what is left of the loans due on RepayBy.
	RepayByAmount float64 `json:"repayByAmount"`
	// Loans are the loans of the balance owed to the user, oldest first;
	// empty when the person owes nothing.
	Loans []LentLoan `json:"loans"`
	// Transactions are the person's lending transactions, newest first.
	Transactions []Transaction `json:"transactions"`
}

// LentLoan is one loan of a balance owed to the user (WM-LEND-7).
// Repayments pay off the loans with their Ref # first, then the oldest.
type LentLoan struct {
	// TransactionID is the lending transaction.
	TransactionID string `json:"transactionId"`
	// Date is the lending day, yyyy-mm-dd.
	Date string `json:"date"`
	// Reference is the lending row's Ref #; nil when blank. A repayment
	// with the same Ref # pays this loan first (WM-LEND-9).
	Reference *string `json:"reference"`
	Amount    float64 `json:"amount"`
	Repaid    float64 `json:"repaid"`
	// Outstanding is amount - repaid; 0 once repaid.
	Outstanding     float64 `json:"outstanding"`
	DaysOutstanding int     `json:"daysOutstanding"`
	// RepayBy is the newest repayment date written for the loan, yyyy-mm-dd;
	// nil when none or once repaid.
	RepayBy *string `json:"repayBy"`
	// DaysUntilRepay counts days from today to RepayBy; negative when overdue.
	DaysUntilRepay *int `json:"daysUntilRepay"`
	// Repayments are the parts of repayment transactions that paid the
	// loan, oldest first. One repayment can be split over several loans.
	Repayments []LoanRepayment `json:"repayments"`
}

// LoanRepayment is the part of a repayment transaction that paid one loan.
type LoanRepayment struct {
	TransactionID string `json:"transactionId"`
	// Date is the repayment day, yyyy-mm-dd.
	Date   string  `json:"date"`
	Amount float64 `json:"amount"`
}

// Open reports whether part of the loan is still owed.
func (l LentLoan) Open() bool {
	return l.Outstanding > 0
}

// MoneyLentFacts are the headline figures, also fed to the AI.
type MoneyLentFacts struct {
	OwedToYou float64 `json:"owedToYou"`
	YouOwe    float64 `json:"youOwe"`
	// PeopleOwing counts people with money owed to the user.
	PeopleOwing int `json:"peopleOwing"`
	// PeopleYouOwe counts people the user owes.
	PeopleYouOwe int `json:"peopleYouOwe"`
	// OldestDaysOutstanding is the longest open balance owed to the user; nil when none.
	OldestDaysOutstanding *int `json:"oldestDaysOutstanding"`
}

// MoneyLent is the per-person ledger. People owing the user come first
// (largest first), then people the user owes, then settled people (latest
// activity first).
type MoneyLent struct {
	AsOf string `json:"asOf"`
	MoneyLentFacts
	People []MoneyLentPerson `json:"people"`
}

// IsLending reports whether the transaction is in a lending category.
func (t Transaction) IsLending() bool {
	return t.CategoryNature == NatureLending
}

// CountsAsIncome is the WM-RULE-3 income predicate: a deposit in an income
// category that is not lending.
func (t Transaction) CountsAsIncome() bool {
	return t.HasCategoryType(CategoryTypeIncome) && !t.IsLending()
}

// CountsAsExpense is the WM-RULE-3 expense predicate: a payment in an
// expense category that is not lending.
func (t Transaction) CountsAsExpense() bool {
	return t.HasCategoryType(CategoryTypeExpense) && !t.IsLending()
}

// WithoutLending drops lending transactions. Lending moves money between an
// account and a receivable, so net worth series must not see it.
func WithoutLending(transactions []Transaction) []Transaction {
	out := make([]Transaction, 0, len(transactions))
	for _, t := range transactions {
		if !t.IsLending() {
			out = append(out, t)
		}
	}
	return out
}

// lendingKey groups payees ignoring case and surrounding spaces.
func lendingKey(payee string) string {
	return strings.ToLower(strings.TrimSpace(payee))
}

// lendingGroups returns the usable lending transactions per person.
func lendingGroups(transactions []Transaction) map[string][]Transaction {
	groups := map[string][]Transaction{}
	for _, t := range transactions {
		if t.Usable() && t.IsLending() {
			key := lendingKey(t.Payee)
			groups[key] = append(groups[key], t)
		}
	}
	return groups
}

func outstandingOf(transactions []Transaction) (lent float64, repaid float64, outstanding float64) {
	for _, t := range transactions {
		lent += t.PaymentAmount()
		repaid += t.DepositAmount()
	}
	outstanding = lent - repaid
	if math.Abs(outstanding) < settledTolerance {
		outstanding = 0
	}
	return lent, repaid, outstanding
}

// lendingPositions returns what people owe the user and what the user owes
// people, summed per person so one person's debt never offsets another's.
func lendingPositions(transactions []Transaction) (receivables float64, payables float64) {
	for _, group := range lendingGroups(transactions) {
		_, _, outstanding := outstandingOf(group)
		if outstanding > 0 {
			receivables += outstanding
		} else {
			payables += -outstanding
		}
	}
	return receivables, payables
}

// ComputeMoneyLent builds the per-person ledger as of today.
func ComputeMoneyLent(transactions []Transaction, today time.Time) MoneyLent {
	result := MoneyLent{AsOf: today.Format(time.DateOnly), People: []MoneyLentPerson{}}
	for _, group := range lendingGroups(transactions) {
		person := moneyLentPerson(group, today)
		result.People = append(result.People, person)
		switch {
		case person.Outstanding > 0:
			result.OwedToYou += person.Outstanding
			result.PeopleOwing++
			if result.OldestDaysOutstanding == nil || *person.DaysOutstanding > *result.OldestDaysOutstanding {
				days := *person.DaysOutstanding
				result.OldestDaysOutstanding = &days
			}
		case person.Outstanding < 0:
			result.YouOwe += -person.Outstanding
			result.PeopleYouOwe++
		}
	}
	sort.SliceStable(result.People, func(i, j int) bool {
		a, b := result.People[i], result.People[j]
		if rankA, rankB := lendingRank(a), lendingRank(b); rankA != rankB {
			return rankA < rankB
		}
		if a.Outstanding != b.Outstanding {
			return math.Abs(a.Outstanding) > math.Abs(b.Outstanding)
		}
		if a.LastActivity != b.LastActivity {
			return a.LastActivity > b.LastActivity
		}
		return a.Name < b.Name
	})
	return result
}

// lendingRank orders owed to the user, owed by the user, then settled.
func lendingRank(person MoneyLentPerson) int {
	switch {
	case person.Outstanding > 0:
		return 0
	case person.Outstanding < 0:
		return 1
	default:
		return 2
	}
}

func moneyLentPerson(group []Transaction, today time.Time) MoneyLentPerson {
	newestFirst := append([]Transaction(nil), group...)
	sort.SliceStable(newestFirst, func(i, j int) bool { return newestFirst[i].Date.After(newestFirst[j].Date) })
	lent, repaid, outstanding := outstandingOf(group)
	person := MoneyLentPerson{
		Name:         strings.TrimSpace(newestFirst[0].Payee),
		Lent:         lent,
		Repaid:       repaid,
		Outstanding:  outstanding,
		LastActivity: newestFirst[0].Date.Format(time.DateOnly),
		Transactions: newestFirst,
	}
	if person.Name == "" {
		person.Name = unknownPerson
	}
	person.Loans = []LentLoan{}
	if outstanding != 0 {
		since := openSince(group)
		value := since.Format(time.DateOnly)
		days := max(0, calendarDaysBetween(since, today))
		person.OpenSince, person.DaysOutstanding = &value, &days
	}
	if outstanding > 0 {
		person.Loans = openLoans(group, openSince(group), today)
		for _, loan := range person.Loans {
			switch {
			case !loan.Open() || loan.RepayBy == nil:
			case person.RepayBy == nil || *loan.RepayBy < *person.RepayBy:
				person.RepayBy, person.DaysUntilRepay, person.RepayByAmount = loan.RepayBy, loan.DaysUntilRepay, loan.Outstanding
			case *loan.RepayBy == *person.RepayBy:
				person.RepayByAmount += loan.Outstanding
			}
		}
	}
	return person
}

// openLoans splits the person's history into loans (WM-LEND-7). A
// repayment pays the loans with its Ref # first (WM-LEND-9), then a loan
// it repays exactly (WM-LEND-10), then the others, oldest first. A loan takes the date in its memo; a dated memo on a
// repayment moves the date of the first loan it would pay that is still
// open after it ("paid half, the rest by 30/12"). Money repaid beyond
// what was lent is owed to the person and offsets the next loan, its Ref
// # first. Each loan lists the repayments that paid it (WM-LEND-8). It
// returns the loans lent since the balance opened.
func openLoans(group []Transaction, openSince time.Time, today time.Time) []LentLoan {
	oldestFirst := append([]Transaction(nil), group...)
	sort.SliceStable(oldestFirst, func(i, j int) bool {
		a, b := oldestFirst[i], oldestFirst[j]
		if !a.Date.Equal(b.Date) {
			return a.Date.Before(b.Date)
		}
		return a.PaymentAmount() > b.PaymentAmount() // a day's lending before its repayments
	})
	var loans []LentLoan
	var lentOn []time.Time
	var dates []*time.Time
	// credit is money repaid beyond what was lent, oldest first.
	var credit []creditPart
	for _, t := range oldestFirst {
		repayment, reference := t.DepositAmount()-t.PaymentAmount(), referenceKey(t.ReferenceNumber)
		if repayment < 0 {
			loan := LentLoan{TransactionID: t.ID, Date: t.Date.Format(time.DateOnly), Reference: trimmedReference(t.ReferenceNumber),
				Amount: -repayment, Outstanding: -repayment, Repayments: []LoanRepayment{}}
			for _, i := range creditOrder(credit, reference) {
				if loan.Outstanding <= 0 {
					break
				}
				part := credit[i].LoanRepayment
				part.Amount = min(part.Amount, loan.Outstanding)
				loan.pay(part)
				credit[i].Amount -= part.Amount
			}
			credit = slices.DeleteFunc(credit, func(c creditPart) bool { return c.Amount <= 0 })
			loans = append(loans, loan)
			lentOn = append(lentOn, t.Date)
			var date *time.Time
			if repayBy, ok := RepaymentDate(t); ok {
				date = &repayBy
			}
			dates = append(dates, date)
			continue
		}
		order := payOrder(loans, reference, repayment)
		for _, i := range order {
			if repayment <= 0 {
				break
			}
			if paid := min(repayment, loans[i].Outstanding); paid > 0 {
				loans[i].pay(LoanRepayment{TransactionID: t.ID, Date: t.Date.Format(time.DateOnly), Amount: paid})
				repayment -= paid
			}
		}
		if repayment > 0 {
			credit = append(credit, creditPart{LoanRepayment{TransactionID: t.ID, Date: t.Date.Format(time.DateOnly), Amount: repayment}, reference})
		}
		if repayBy, ok := RepaymentDate(t); ok {
			for _, i := range order {
				if loans[i].Outstanding >= settledTolerance {
					dates[i] = &repayBy
					break
				}
			}
		}
	}
	open := []LentLoan{}
	for i, loan := range loans {
		if lentOn[i].Before(openSince) {
			continue
		}
		if loan.Outstanding < settledTolerance {
			loan.Repaid, loan.Outstanding = loan.Amount, 0
		} else {
			loan.DaysOutstanding = max(0, calendarDaysBetween(lentOn[i], today))
			if dates[i] != nil {
				date, until := dates[i].Format(time.DateOnly), calendarDaysBetween(today, *dates[i])
				loan.RepayBy, loan.DaysUntilRepay = &date, &until
			}
		}
		open = append(open, loan)
	}
	return open
}

// creditPart is money repaid ahead and the Ref # it was written with.
type creditPart struct {
	LoanRepayment
	reference string
}

// payOrder lists the loans a repayment pays: those with its Ref # first,
// then an open loan whose remaining amount it repays exactly (6m back for
// a 6m loan), then the others, each oldest first.
func payOrder(loans []LentLoan, reference string, amount float64) []int {
	return rankedOrder(len(loans), func(i int) int {
		switch {
		case reference != "" && referenceKey(loans[i].Reference) == reference:
			return 0
		case loans[i].Outstanding > 0 && math.Abs(loans[i].Outstanding-amount) < settledTolerance:
			return 1
		}
		return 2
	})
}

// creditOrder lists the money repaid ahead a new loan takes: parts with
// its Ref # first, then the others, each oldest first.
func creditOrder(credit []creditPart, reference string) []int {
	return rankedOrder(len(credit), func(i int) int {
		if reference != "" && credit[i].reference == reference {
			return 0
		}
		return 1
	})
}

// rankedOrder lists 0..n-1 by rank, keeping order within a rank.
func rankedOrder(n int, rank func(i int) int) []int {
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return rank(order[a]) < rank(order[b]) })
	return order
}

// referenceKey compares Ref # values ignoring case and surrounding spaces.
func referenceKey(reference *string) string {
	if reference == nil {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(*reference))
}

func trimmedReference(reference *string) *string {
	if reference == nil || strings.TrimSpace(*reference) == "" {
		return nil
	}
	value := strings.TrimSpace(*reference)
	return &value
}

func (l *LentLoan) pay(part LoanRepayment) {
	l.Repaid += part.Amount
	l.Outstanding -= part.Amount
	l.Repayments = append(l.Repayments, part)
}

// openSince walks the balance day by day (a day's lending and repayment net
// out first) and returns the day the current balance last left zero or
// changed sign.
func openSince(group []Transaction) time.Time {
	netByDay := map[time.Time]float64{}
	for _, t := range group {
		netByDay[t.Date] += t.PaymentAmount() - t.DepositAmount()
	}
	days := make([]time.Time, 0, len(netByDay))
	for date := range netByDay {
		days = append(days, date)
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Before(days[j]) })
	var balance float64
	var since time.Time
	for _, date := range days {
		previous := sign(balance)
		balance += netByDay[date]
		if current := sign(balance); current != 0 && current != previous {
			since = date
		}
	}
	return since
}

// calendarDaysBetween counts calendar days from a to b, ignoring the time of day.
func calendarDaysBetween(a time.Time, b time.Time) int {
	from := time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, time.UTC)
	to := time.Date(b.Year(), b.Month(), b.Day(), 0, 0, 0, 0, time.UTC)
	return int(math.Round(to.Sub(from).Hours() / 24))
}

func sign(value float64) int {
	switch {
	case value >= settledTolerance:
		return 1
	case value <= -settledTolerance:
		return -1
	default:
		return 0
	}
}
