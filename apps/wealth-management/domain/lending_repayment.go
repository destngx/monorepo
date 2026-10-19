package domain

import (
	"regexp"
	"strconv"
	"time"
)

// Repayment dates (WM-LEND-6): the user writes the agreed date in the memo
// of a lending transaction ("repay 30/11", "hẹn trả 2026-12-15"). Dates are
// ISO yyyy-mm-dd or day first dd/mm, dd/mm/yy, dd/mm/yyyy ("-" and "." only
// with the year, so "1.5%" or "2-3 triệu" is never a date). A date without
// a year is the one nearest the transaction day ("05/01" written on 20 Dec
// is next January, "05/10" written on 12 Oct is a week ago). A date on or
// before the transaction day is when something happened, not a repayment
// date, so it is ignored.

// LentDueSoonDays: a repayment date this close is worth a reminder.
const LentDueSoonDays = 3

var (
	isoMemoDate      = regexp.MustCompile(`\b(\d{4})-(\d{1,2})-(\d{1,2})\b`)
	dayFirstMemoDate = regexp.MustCompile(`\b(\d{1,2})([/.-])(\d{1,2})(?:([/.-])(\d{4}|\d{2}))?\b`)
)

// RepaymentDate returns the earliest date after the transaction day written
// in its memo.
func RepaymentDate(t Transaction) (time.Time, bool) {
	if t.Memo == nil {
		return time.Time{}, false
	}
	after := time.Date(t.Date.Year(), t.Date.Month(), t.Date.Day(), 0, 0, 0, 0, time.UTC)
	var earliest time.Time
	consider := func(date time.Time) {
		if date.After(after) && (earliest.IsZero() || date.Before(earliest)) {
			earliest = date
		}
	}
	memo := *t.Memo
	isoMatches := isoMemoDate.FindAllStringSubmatchIndex(memo, -1)
	for _, match := range isoMatches {
		if date, ok := validDate(atoi(memo, match, 1), atoi(memo, match, 2), atoi(memo, match, 3)); ok {
			consider(date)
		}
	}
	for _, match := range dayFirstMemoDate.FindAllStringSubmatchIndex(memo, -1) {
		if insideAny(match, isoMatches) {
			continue // the tail of an ISO date ("01-12" in 2026-01-12)
		}
		day, month, separator := atoi(memo, match, 1), atoi(memo, match, 3), memo[match[4]:match[5]]
		hasYear := match[10] >= 0
		switch {
		case !hasYear && separator != "/":
			continue // "1.5" and "2-3" are numbers
		case hasYear && memo[match[8]:match[9]] != separator:
			continue
		case hasYear:
			year := atoi(memo, match, 5)
			if year < 100 {
				year += 2000
			}
			if date, ok := validDate(year, month, day); ok {
				consider(date)
			}
		default:
			if date, ok := nearestDayMonth(after, day, month); ok {
				consider(date)
			}
		}
	}
	return earliest, !earliest.IsZero()
}

// openRepaymentDate is the repayment date of the open balance: the one in
// the newest memo with a date since the balance opened, so a later note
// ("paid half, the rest by 30/12") replaces an earlier one.
func openRepaymentDate(newestFirst []Transaction, openSince time.Time) (time.Time, bool) {
	for _, t := range newestFirst {
		if t.Date.Before(openSince) {
			break
		}
		if date, ok := RepaymentDate(t); ok {
			return date, true
		}
	}
	return time.Time{}, false
}

// RepaymentDue reports whether the person owes the user and the repayment
// date is within LentDueSoonDays or has passed.
func (p MoneyLentPerson) RepaymentDue() bool {
	return p.Outstanding > 0 && p.DaysUntilRepay != nil && *p.DaysUntilRepay <= LentDueSoonDays
}

// RepaymentOverdue reports whether the person owes the user past the repayment date.
func (p MoneyLentPerson) RepaymentOverdue() bool {
	return p.Outstanding > 0 && p.DaysUntilRepay != nil && *p.DaysUntilRepay < 0
}

// nearestDayMonth is the day and month in the year before, of or after
// from that is nearest to it.
func nearestDayMonth(from time.Time, day int, month int) (time.Time, bool) {
	var nearest time.Time
	for year := from.Year() - 1; year <= from.Year()+1; year++ {
		date, ok := validDate(year, month, day)
		if ok && (nearest.IsZero() || absDuration(date.Sub(from)) < absDuration(nearest.Sub(from))) {
			nearest = date
		}
	}
	return nearest, !nearest.IsZero()
}

func insideAny(match []int, ranges [][]int) bool {
	for _, r := range ranges {
		if match[0] < r[1] && r[0] < match[1] {
			return true
		}
	}
	return false
}

// atoi reads a digits-only regexp group.
func atoi(s string, match []int, group int) int {
	value, _ := strconv.Atoi(s[match[2*group]:match[2*group+1]]) // the group is \d+
	return value
}

// validDate rejects days that time.Date would roll over (31/02).
func validDate(year int, month int, day int) (time.Time, bool) {
	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	return date, date.Year() == year && int(date.Month()) == month && date.Day() == day
}
