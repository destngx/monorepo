package domain

import (
	"math"
	"strconv"
	"strings"
	"time"
)

// Credit card statement cycle and due-date maths (WM-RULE-11).

const (
	minimumPaymentRate = 0.05
	dueUrgentDays      = 5
)

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

// CardCycle describes where today falls in the statement and payment cycle.
type CardCycle struct {
	StatementDay      int     `json:"statementDay"`
	DueDay            int     `json:"dueDay"`
	CycleStart        string  `json:"cycleStart"`
	CycleEnd          string  `json:"cycleEnd"`
	CycleDays         int     `json:"cycleDays"`
	CycleElapsedDays  int     `json:"cycleElapsedDays"`
	CycleProgress     float64 `json:"cycleProgress"`
	NextDueDate       string  `json:"nextDueDate"`
	DaysUntilDue      int     `json:"daysUntilDue"`
	DueUrgent         bool    `json:"dueUrgent"`
	MinimumPayment    float64 `json:"minimumPayment"`
	OutstandingAmount float64 `json:"outstandingAmount"`
}

// ParseCardDays reads the Accounts dueDate cell, e.g. "5, 30" = statement day 5, due day 30.
func ParseCardDays(value *string) (statementDay int, dueDay int, ok bool) {
	if value == nil {
		return 0, 0, false
	}
	parts := strings.FieldsFunc(*value, func(r rune) bool { return r == ',' || r == ';' || r == ' ' })
	if len(parts) != 2 {
		return 0, 0, false
	}
	statement, errStatement := strconv.Atoi(parts[0])
	due, errDue := strconv.Atoi(parts[1])
	if errStatement != nil || errDue != nil || !validDayOfMonth(statement) || !validDayOfMonth(due) {
		return 0, 0, false
	}
	return statement, due, true
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

// CycleStartFor returns the most recent statement date on or before today.
func CycleStartFor(statementDay int, today time.Time) time.Time {
	start := clampedDate(today.Year(), today.Month(), statementDay)
	if start.After(today) {
		previous := today.AddDate(0, 0, -today.Day())
		start = clampedDate(previous.Year(), previous.Month(), statementDay)
	}
	return start
}

// ComputeCardCycle derives cycle progress, the next due date and the minimum payment.
func ComputeCardCycle(statementDay int, dueDay int, outstanding float64, today time.Time) CardCycle {
	start := CycleStartFor(statementDay, today)
	nextMonth := time.Date(start.Year(), start.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	end := clampedDate(nextMonth.Year(), nextMonth.Month(), statementDay)
	cycleDays := int(end.Sub(start).Hours() / 24)
	elapsed := int(today.Sub(start).Hours() / 24)

	due := clampedDate(today.Year(), today.Month(), dueDay)
	if due.Before(today) {
		following := time.Date(today.Year(), today.Month()+1, 1, 0, 0, 0, 0, time.UTC)
		due = clampedDate(following.Year(), following.Month(), dueDay)
	}
	daysUntilDue := int(due.Sub(today).Hours() / 24)
	amount := math.Abs(outstanding)

	return CardCycle{
		StatementDay:      statementDay,
		DueDay:            dueDay,
		CycleStart:        start.Format(time.DateOnly),
		CycleEnd:          end.Format(time.DateOnly),
		CycleDays:         cycleDays,
		CycleElapsedDays:  elapsed,
		CycleProgress:     round2(float64(elapsed) / float64(cycleDays) * 100),
		NextDueDate:       due.Format(time.DateOnly),
		DaysUntilDue:      daysUntilDue,
		DueUrgent:         amount > 0 && daysUntilDue <= dueUrgentDays,
		MinimumPayment:    math.Round(amount * minimumPaymentRate),
		OutstandingAmount: amount,
	}
}
