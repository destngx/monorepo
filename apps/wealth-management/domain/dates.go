package domain

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// SheetDateLayout is the only layout the app writes to the sheet (WM-DATA-3).
const SheetDateLayout = "02/01/2006"

// MonthKeyLayout formats a month as "YYYY-MM".
const MonthKeyLayout = "2006-01"

var ErrInvalidDate = errors.New("invalid date")

// AppLocation is the user's timezone; "today" and month boundaries use it.
var AppLocation = mustLoadLocation("Asia/Ho_Chi_Minh")

func mustLoadLocation(name string) *time.Location {
	location, err := time.LoadLocation(name)
	if err != nil {
		return time.FixedZone("ICT", 7*60*60)
	}
	return location
}

// sheetEpoch is day 0 of Google Sheets date serials.
var sheetEpoch = time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)

// SerialToDate converts a Sheets date serial to a UTC calendar date.
func SerialToDate(serial float64) time.Time {
	return sheetEpoch.AddDate(0, 0, int(math.Floor(serial)))
}

// ParseSheetDate parses a calendar date the way the sheet stores it. Day comes
// before month in every slash format (the app writes dd/MM/yyyy), so 05/03/2026
// is always 5 March. Unparseable input is an error, never "now".
func ParseSheetDate(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, fmt.Errorf("%w: empty", ErrInvalidDate)
	}
	if serial, err := strconv.ParseFloat(trimmed, 64); err == nil {
		if serial > 1000 && serial < 100000 {
			return SerialToDate(serial), nil
		}
		return time.Time{}, fmt.Errorf("%w: %q", ErrInvalidDate, trimmed)
	}
	for _, layout := range []string{"2/1/2006", "2/1/06", "2-1-2006", "2.1.2006", "2006-01-02", "2006/01/02"} {
		if parsed, err := time.Parse(layout, trimmed); err == nil {
			return normalizeYear(parsed), nil
		}
	}
	if parsed, err := time.Parse(time.RFC3339, trimmed); err == nil {
		local := parsed.In(AppLocation)
		return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC), nil
	}
	return time.Time{}, fmt.Errorf("%w: %q", ErrInvalidDate, trimmed)
}

func normalizeYear(value time.Time) time.Time {
	if value.Year() < 100 {
		return value.AddDate(2000, 0, 0)
	}
	return value
}

// Today returns the current calendar date in AppLocation as a UTC midnight value,
// matching how sheet dates are represented.
func Today(now time.Time) time.Time {
	local := now.In(AppLocation)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
}

// MonthKey returns "YYYY-MM" for a date.
func MonthKey(date time.Time) string {
	return date.Format(MonthKeyLayout)
}

// ParseMonthKey parses "YYYY-MM" into the first day of that month (UTC).
func ParseMonthKey(value string) (time.Time, error) {
	parsed, err := time.Parse(MonthKeyLayout, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: month %q, expected YYYY-MM", ErrInvalidDate, value)
	}
	return parsed, nil
}

// DaysInMonth returns the number of days in the month containing date.
func DaysInMonth(date time.Time) int {
	return time.Date(date.Year(), date.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// SameMonth reports whether two dates fall in the same calendar month.
func SameMonth(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month()
}
