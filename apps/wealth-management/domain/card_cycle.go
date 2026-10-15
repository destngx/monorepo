package domain

import (
	"math"
	"time"
)

// Credit card statement cycle and due-date maths (WM-RULE-11).

const dueUrgentDays = 5

// UtilizationBand colours the utilisation ring: <30 good, <70 fair, else high.
type UtilizationBand string

const (
	UtilizationGood UtilizationBand = "good"
	UtilizationFair UtilizationBand = "fair"
	UtilizationHigh UtilizationBand = "high"
)

func UtilizationBandFor(percent float64) UtilizationBand {
	switch {
	case percent < 30:
		return UtilizationGood
	case percent < 70:
		return UtilizationFair
	default:
		return UtilizationHigh
	}
}

// CardCycle describes where today falls in the statement cycle and when the
// balance must be paid.
type CardCycle struct {
	StatementDay     int     `json:"statementDay"`
	DueDay           int     `json:"dueDay"`
	CycleStart       string  `json:"cycleStart"`
	CycleEnd         string  `json:"cycleEnd"`
	CycleDays        int     `json:"cycleDays"`
	CycleElapsedDays int     `json:"cycleElapsedDays"`
	CycleProgress    float64 `json:"cycleProgress"`
	// PayBy is the next pay-by date (WM-CARD-15); the bank's DueDay is later.
	PayBy          string `json:"payBy"`
	DaysUntilPayBy int    `json:"daysUntilPayBy"`
	// PayByUrgent: the statement it settles is owed and due within days.
	PayByUrgent bool `json:"payByUrgent"`
}

func validDayOfMonth(value int) bool {
	return value >= 1 && value <= 31
}

// clampedDate returns the given day in the month, clamped to the month length.
func clampedDate(year int, month time.Month, dayOfMonth int) time.Time {
	first := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	if dayOfMonth > DaysInMonth(first) {
		dayOfMonth = DaysInMonth(first)
	}
	return time.Date(year, month, dayOfMonth, 0, 0, 0, 0, time.UTC)
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// CycleStartFor returns the most recent statement date on or before today.
func CycleStartFor(statementDay int, today time.Time) time.Time {
	start := clampedDate(today.Year(), today.Month(), statementDay)
	if start.After(today) {
		previous := today.AddDate(0, 0, -today.Day())
		start = clampedDate(previous.Year(), previous.Month(), statementDay)
	}
	return start
}

// PeriodBounds returns the counting period containing date (C7): from the
// statement date or the 1st of the month, to the start of the next period
// (exclusive).
func PeriodBounds(period CapPeriod, statementDay int, date time.Time) (time.Time, time.Time) {
	if period == CapPeriodMonth {
		start := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 1, 0)
	}
	start := CycleStartFor(statementDay, date)
	next := time.Date(start.Year(), start.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	return start, clampedDate(next.Year(), next.Month(), statementDay)
}

// Pay-by day (WM-CARD-15, D26): every balance is paid by the 29th, the 28th
// in February (leap years too), whatever the bank's due day.
const (
	payByDay         = 29
	payByDayFebruary = 28
)

// PayByDate returns the next pay-by date on or after today.
func PayByDate(today time.Time) time.Time {
	today = dateOnly(today)
	date := payByIn(today.Year(), today.Month())
	if date.Before(today) {
		next := time.Date(today.Year(), today.Month()+1, 1, 0, 0, 0, 0, time.UTC)
		date = payByIn(next.Year(), next.Month())
	}
	return date
}

func payByIn(year int, month time.Month) time.Time {
	day := payByDay
	if month == time.February {
		day = payByDayFebruary
	}
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// ComputeCardCycle derives cycle progress and the pay-by date; whether
// paying is urgent depends on the statement, set with it.
func ComputeCardCycle(statementDay int, dueDay int, today time.Time) CardCycle {
	today = dateOnly(today)
	start, end := PeriodBounds(CapPeriodCycle, statementDay, today)
	cycleDays := int(end.Sub(start).Hours() / 24)
	elapsed := int(today.Sub(start).Hours() / 24)
	payBy := PayByDate(today)

	return CardCycle{
		StatementDay:     statementDay,
		DueDay:           dueDay,
		CycleStart:       start.Format(time.DateOnly),
		CycleEnd:         end.Format(time.DateOnly),
		CycleDays:        cycleDays,
		CycleElapsedDays: elapsed,
		CycleProgress:    round2(float64(elapsed) / float64(cycleDays) * 100),
		PayBy:            payBy.Format(time.DateOnly),
		DaysUntilPayBy:   int(payBy.Sub(today).Hours() / 24),
	}
}

// StatementBounds returns the statement cycle a pay-by date settles: the
// last one ending (exclusive) on or before it.
func StatementBounds(statementDay int, payBy time.Time) (time.Time, time.Time) {
	end := CycleStartFor(statementDay, payBy)
	return CycleStartFor(statementDay, end.AddDate(0, 0, -1)), end
}

// owedAmount is what a card account owes: a positive balance is money paid
// in ahead, which is room, not debt (C9).
func owedAmount(balance float64) float64 {
	return math.Max(0, -balance)
}
