package domain

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Repayment dates (WM-LEND-6): the user writes the agreed date in the memo
// of a lending transaction ("repay 30/11", "hẹn trả 2026-12-15", "dự kiến
// cuối tháng 11 hoàn trả"). Dates are ISO yyyy-mm-dd or day first dd/mm,
// dd/mm/yy, dd/mm/yyyy ("-" and "." only with the year, so "1.5%" or "2-3
// triệu" is never a date), or a Vietnamese month phrase: "cuối tháng 11"
// (the last day), "giữa tháng 11" (the 15th), "đầu tháng 11" (the 1st) and
// "tháng 11" alone (the last day), with or without accents, optionally with
// the year ("tháng 11/2026", "tháng 11 năm 2026"), a day of a month
// ("ngày 15 tháng 11", "mùng 5 tháng 1 năm 2027"), or a day alone ("lương
// ngày 24", "mùng 5"): that day of the transaction's month, or of the next
// month when it is not after the transaction day, but never the
// transaction day itself. A date without a month or year is
// the next one after the transaction day ("05/01" written on 20 Dec is next
// January, "31/10" on a loan from March is this October), unless it falls
// within recentPastDays before it: "05/10" written on 12 Oct is a week ago.
// The memo may be written long after the loan, so the transaction day is
// the only anchor. A date on or before the transaction day is when
// something happened, not a repayment date, so it is ignored.

// LentDueSoonDays: a repayment date this close is worth a reminder.
const LentDueSoonDays = 3

var (
	isoMemoDate      = regexp.MustCompile(`\b(\d{4})-(\d{1,2})-(\d{1,2})\b`)
	dayFirstMemoDate = regexp.MustCompile(`\b(\d{1,2})([/.-])(\d{1,2})(?:([/.-])(\d{4}|\d{2}))?\b`)
	// monthMemoDate: an optional part of the month, the month and an optional
	// year. The leading group stands in for a word boundary, which \b does not
	// give before a non-ASCII letter such as "đ".
	monthMemoDate = regexp.MustCompile(`(?i)(?:^|[^\p{L}\d])(?:(cuối|cuoi|giữa|giua|đầu|dau)\s+)?(?:tháng|thang)\s+(\d{1,2})(?:\s*(?:/|năm|nam)\s*(\d{4}|\d{2}))?\b`)
	// dayFollowers after a day alone make it something else: the rest of a
	// day-first date or an amount ("ngay" also means "at once").
	dayFollowers = regexp.MustCompile(`(?i)^(?:[/.-]\d|\s*(?:triệu|trieu|tr|nghìn|nghin|ngàn|ngan|k|đ|vnd)(?:[^\p{L}]|$))`)
	// dayMemoDate: a day word, the day and an optional month and year.
	dayMemoDate = regexp.MustCompile(`(?i)(?:^|[^\p{L}\d])(?:ngày|ngay|mùng|mồng|mung)\s+(\d{1,2})\b(?:\s+(?:tháng|thang)\s+(\d{1,2})(?:\s*(?:/|năm|nam)\s*(\d{4}|\d{2}))?\b)?`)
)

// recentPastDays: a date without a year this close before the transaction
// day is a note of what happened, not next year's repayment date.
const recentPastDays = 31

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
	dayMatches := [][]int{}
	for _, match := range dayMemoDate.FindAllStringSubmatchIndex(memo, -1) {
		day := atoi(memo, match, 1)
		switch {
		case insideAny(match, isoMatches):
			continue
		case match[4] < 0 && dayFollowers.MatchString(memo[match[3]:]):
			continue // "ngày 24/11" is a day-first date, "trả ngay 2 triệu" an amount
		case match[4] < 0:
			dayMatches = append(dayMatches, match)
			if date, ok := nextDayOfMonth(after, day); ok {
				consider(date)
			}
		case match[6] >= 0:
			dayMatches = append(dayMatches, match)
			if date, ok := validDate(fullYear(atoi(memo, match, 3)), atoi(memo, match, 2), day); ok {
				consider(date)
			}
		default:
			dayMatches = append(dayMatches, match)
			month := atoi(memo, match, 2)
			if date, ok := nextOccurrence(after, func(year int) (time.Time, bool) { return validDate(year, month, day) }); ok {
				consider(date)
			}
		}
	}
	monthMatches := [][]int{}
	for _, match := range monthMemoDate.FindAllStringSubmatchIndex(memo, -1) {
		if insideAny(match, dayMatches) {
			continue // the month of "ngày 15 tháng 11"
		}
		monthMatches = append(monthMatches, match)
		part, month := "", atoi(memo, match, 2)
		if match[2] >= 0 {
			part = memo[match[2]:match[3]]
		}
		if month < 1 || month > 12 {
			continue
		}
		on := func(year int) (time.Time, bool) { return partOfMonth(year, month, part), true }
		if match[6] >= 0 {
			consider(partOfMonth(fullYear(atoi(memo, match, 3)), month, part))
		} else if date, ok := nextOccurrence(after, on); ok {
			consider(date)
		}
	}
	for _, match := range dayFirstMemoDate.FindAllStringSubmatchIndex(memo, -1) {
		if insideAny(match, isoMatches) || insideAny(match, monthMatches) || insideAny(match, dayMatches) {
			continue // the tail of an ISO date ("01-12" in 2026-01-12) or a month's year ("11/2026")
		}
		day, month, separator := atoi(memo, match, 1), atoi(memo, match, 3), memo[match[4]:match[5]]
		hasYear := match[10] >= 0
		switch {
		case !hasYear && separator != "/":
			continue // "1.5" and "2-3" are numbers
		case hasYear && memo[match[8]:match[9]] != separator:
			continue
		case hasYear:
			if date, ok := validDate(fullYear(atoi(memo, match, 5)), month, day); ok {
				consider(date)
			}
		default:
			if date, ok := nextOccurrence(after, func(year int) (time.Time, bool) { return validDate(year, month, day) }); ok {
				consider(date)
			}
		}
	}
	return earliest, !earliest.IsZero()
}

// Due reports whether the open loan's repayment date is within
// LentDueSoonDays or has passed.
func (l LentLoan) Due() bool {
	return l.Open() && l.DaysUntilRepay != nil && *l.DaysUntilRepay <= LentDueSoonDays
}

// Overdue reports whether the open loan is past its repayment date.
func (l LentLoan) Overdue() bool {
	return l.Open() && l.DaysUntilRepay != nil && *l.DaysUntilRepay < 0
}

// nextOccurrence is the first date on(year) after from, or none when the
// latest one on or before from is within recentPastDays of it. A 29 Feb
// may be up to 4 years away.
func nextOccurrence(from time.Time, on func(year int) (time.Time, bool)) (time.Time, bool) {
	for year := from.Year(); year >= from.Year()-4; year-- {
		date, ok := on(year)
		if !ok || date.After(from) {
			continue
		}
		if from.Sub(date) <= recentPastDays*24*time.Hour {
			return time.Time{}, false
		}
		break
	}
	for year := from.Year(); year <= from.Year()+4; year++ {
		if date, ok := on(year); ok && date.After(from) {
			return date, true
		}
	}
	return time.Time{}, false
}

// nextDayOfMonth is the first day of that number after from: in from's
// month or a later one, skipping months too short for it. From's own day
// is when the money moved, so none.
func nextDayOfMonth(from time.Time, day int) (time.Time, bool) {
	if day == from.Day() {
		return time.Time{}, false
	}
	for months := range 12 {
		first := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, months, 0)
		if date, ok := validDate(first.Year(), int(first.Month()), day); ok && date.After(from) {
			return date, true
		}
	}
	return time.Time{}, false
}

// partOfMonth is the day a month phrase means: the 1st for "đầu" (early),
// the 15th for "giữa" (mid) and the last day for "cuối" (end) or none.
func partOfMonth(year int, month int, part string) time.Time {
	first := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	switch strings.ToLower(part) {
	case "đầu", "dau":
		return first
	case "giữa", "giua":
		return first.AddDate(0, 0, 14)
	}
	return first.AddDate(0, 1, -1)
}

// fullYear reads a two-digit year as 20yy.
func fullYear(year int) int {
	if year < 100 {
		return year + 2000
	}
	return year
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
