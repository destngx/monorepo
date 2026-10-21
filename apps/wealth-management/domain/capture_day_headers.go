package domain

import (
	"regexp"
	"strconv"
	"time"
)

// History screens group rows under day headers ("Hôm nay", "Hôm qua",
// "19/10/2026"). The model reports the header each row sits under and the
// engine dates the row by it (WM-CAP-2), since the model dated rows under
// a covered header as today.

// Why a row's date needs a check (CaptureEvent.DateCheck).
const (
	// CaptureDateInferred: the header was hidden and the date is the one
	// day between the visible headers around it.
	CaptureDateInferred = "inferred"
	// CaptureDateHidden: the header was hidden and the headers around it
	// leave more than one day, so the model's date stays.
	CaptureDateHidden = "hidden"
)

const captureHiddenHeader = "hidden"

var dayHeaderDatePattern = regexp.MustCompile(`(\d{1,2})/(\d{1,2})(?:/(\d{4}))?`)

// resolveDayHeaders dates each event by its day header. Events are in
// screen order, so a hidden header lies between the nearest readable
// headers above and below it.
func resolveDayHeaders(events []CaptureEvent, today time.Time) {
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	dates := make([]*time.Time, len(events))
	for index, event := range events {
		if date, ok := dayHeaderDate(event.DayHeader, today); ok {
			dates[index] = &date
		}
	}
	for index := range events {
		event := &events[index]
		if dates[index] != nil {
			event.setDay(*dates[index])
			continue
		}
		if FoldText(event.DayHeader) != captureHiddenHeader {
			continue
		}
		above, below := nearestHeaderDate(dates, index, -1), nearestHeaderDate(dates, index, 1)
		if above != nil && below != nil {
			newer, older := *above, *below
			if older.After(newer) {
				newer, older = older, newer
			}
			if newer.Sub(older) == 48*time.Hour {
				event.setDay(older.AddDate(0, 0, 1))
				event.DateCheck = CaptureDateInferred
				continue
			}
		}
		event.DateCheck = CaptureDateHidden
	}
}

func nearestHeaderDate(dates []*time.Time, from, step int) *time.Time {
	for index := from + step; index >= 0 && index < len(dates); index += step {
		if dates[index] != nil {
			return dates[index]
		}
	}
	return nil
}

// dayHeaderDate reads "Hôm nay", "Hôm qua", "Hôm kia" or a dd/mm[/yyyy]
// date in a header; a date without a year is the latest one not after today.
func dayHeaderDate(header string, today time.Time) (time.Time, bool) {
	switch FoldText(header) {
	case "hom nay":
		return today, true
	case "hom qua":
		return today.AddDate(0, 0, -1), true
	case "hom kia":
		return today.AddDate(0, 0, -2), true
	}
	match := dayHeaderDatePattern.FindStringSubmatch(header)
	if match == nil {
		return time.Time{}, false
	}
	day, _ := strconv.Atoi(match[1])
	month, _ := strconv.Atoi(match[2])
	year := today.Year()
	if match[3] != "" {
		year, _ = strconv.Atoi(match[3])
	}
	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if date.Day() != day || date.Month() != time.Month(month) {
		return time.Time{}, false
	}
	if match[3] == "" && date.After(today) {
		date = date.AddDate(-1, 0, 0)
	}
	return date, true
}

// setDay moves the event to the day, keeping its time of day.
func (e *CaptureEvent) setDay(day time.Time) {
	e.At = time.Date(day.Year(), day.Month(), day.Day(), e.At.Hour(), e.At.Minute(), e.At.Second(), 0, time.UTC)
}
