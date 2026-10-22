package domain

import (
	"fmt"
	"math"
	"time"
)

// Moving money between instant-access places (WM-CASH-9, D41). A move
// earns the rate gap from the day the new place starts paying, but the
// money earns nothing in between: it stops earning in the old place the
// day it leaves (neither provider promises interest for the day of a
// withdrawal, so the cost errs high), and a product whose interest starts
// the next working day (Zalopay, MoMo) pays nothing for the days until
// then. The cost is what the money would have earned staying put for
// those days; the payback is how long the gap takes to earn it back.
// Public holidays are not known, so the working days are Monday to Friday.

// savingsMoveSearchDays is how far ahead the cheapest day to move is looked
// for: a week covers every weekday.
const savingsMoveSearchDays = 7

// workingHoursPerDay turns a withdrawal time stated in working hours into
// working days.
const workingHoursPerDay = 8

// SavingsMoveDay is the cost of moving on one day.
type SavingsMoveDay struct {
	Date    string `json:"date"`
	Weekday string `json:"weekday"`
	// DaysWithoutInterest are the days the money earns nothing: from the day
	// it leaves until the day before the new place starts paying.
	DaysWithoutInterest int `json:"daysWithoutInterest"`
	// InterestLost = amount x old rate x days / 365: what staying would have
	// earned in those days.
	InterestLost float64 `json:"interestLost"`
	// PaybackDays is how many days at the new rate earn the loss back; 0
	// when nothing is lost.
	PaybackDays int `json:"paybackDays"`
}

// SavingsMove is the plan to move money from an account to a better
// instant-access place.
type SavingsMove struct {
	From     string  `json:"from"`
	FromRate float64 `json:"fromRate"`
	// FromProduct is the account's linked product, when it has one.
	FromProduct string  `json:"fromProduct,omitempty"`
	Amount      float64 `json:"amount"`
	To          string  `json:"to"`
	ToRate      float64 `json:"toRate"`
	// ToBacking is the new place's backing when it is a flexible product.
	ToBacking string `json:"toBacking,omitempty"`
	// RateGapPoints = ToRate - FromRate.
	RateGapPoints float64 `json:"rateGapPoints"`
	ExtraPerMonth float64 `json:"extraPerMonth"`
	ExtraPerYear  float64 `json:"extraPerYear"`
	// Today is moving today; BestDay the cheapest day in the next week
	// (today when today is as cheap as any).
	Today   SavingsMoveDay `json:"today"`
	BestDay SavingsMoveDay `json:"bestDay"`
	// WorstCaseExtraDays are the days a slow withdrawal from the old place
	// can add (MoMo: up to 16 working hours); 0 when none is stated.
	WorstCaseExtraDays int `json:"worstCaseExtraDays"`
	// Steps say how to move, in order.
	Steps []string `json:"steps"`
}

// savingsMoveInput is one move to plan.
type savingsMoveInput struct {
	From        string
	FromRate    float64
	FromProduct *FlexibleSaving
	Amount      float64
	To          DepositRate
	Today       time.Time
}

// planSavingsMove prices a move; false when the new place does not pay
// more or there is nothing to move.
func planSavingsMove(in savingsMoveInput) (SavingsMove, bool) {
	gap := in.To.Rate - in.FromRate
	if in.Amount <= 0 || gap <= 0 {
		return SavingsMove{}, false
	}
	move := SavingsMove{
		From: in.From, FromRate: in.FromRate, Amount: math.Round(in.Amount), To: in.To.Place(), ToRate: in.To.Rate, ToBacking: in.To.Backing,
		RateGapPoints: round2(gap),
		ExtraPerYear:  math.Round(in.Amount * gap / 100),
		ExtraPerMonth: math.Round(in.Amount * gap / 100 / 12),
	}
	if in.FromProduct != nil {
		move.FromProduct = in.FromProduct.Name()
		if hours := in.FromProduct.WithdrawalMaxWorkingHours; hours != nil && *hours > 0 {
			move.WorstCaseExtraDays = (*hours + workingHoursPerDay - 1) / workingHoursPerDay
		}
	}
	day := dateOnly(in.Today)
	move.Today = in.moveDay(day)
	move.BestDay = move.Today
	for offset := 1; offset < savingsMoveSearchDays; offset++ {
		if candidate := in.moveDay(day.AddDate(0, 0, offset)); candidate.DaysWithoutInterest < move.BestDay.DaysWithoutInterest {
			move.BestDay = candidate
		}
	}
	move.Steps = move.steps(in)
	return move, true
}

// moveDay prices moving on one day.
func (in savingsMoveInput) moveDay(day time.Time) SavingsMoveDay {
	days := daysWithoutInterest(day, in.To.EarningStarts)
	lost := math.Round(in.Amount * in.FromRate / 100 * float64(days) / 365)
	result := SavingsMoveDay{Date: day.Format(time.DateOnly), Weekday: day.Weekday().String(), DaysWithoutInterest: days, InterestLost: lost}
	if days > 0 && in.FromRate > 0 {
		result.PaybackDays = int(math.Ceil(float64(days) * in.FromRate / (in.To.Rate - in.FromRate)))
	}
	return result
}

// daysWithoutInterest counts the days from moving on day until the new
// place starts paying: none when it pays from the same day (a bank deposit
// opened online), else until the next working day.
func daysWithoutInterest(day time.Time, earningStarts string) int {
	if earningStarts != EarningStartsNextWorkingDay {
		return 0
	}
	start := day.AddDate(0, 0, 1)
	for !isWorkingDay(start) {
		start = start.AddDate(0, 0, 1)
	}
	return int(start.Sub(day).Hours()/24 + 0.5)
}

func isWorkingDay(day time.Time) bool {
	return day.Weekday() != time.Saturday && day.Weekday() != time.Sunday
}

// steps is the strategy in plain words.
func (m SavingsMove) steps(in savingsMoveInput) []string {
	steps := []string{}
	when := "today"
	if m.BestDay.Date != m.Today.Date {
		when = fmt.Sprintf("on %s %s", m.BestDay.Weekday, m.BestDay.Date)
		steps = append(steps, fmt.Sprintf("Wait until %s %s: moving today (%s) leaves the money %s without interest (about %s VND), moving then only %s (about %s VND).",
			m.BestDay.Weekday, m.BestDay.Date, m.Today.Weekday, plural(m.Today.DaysWithoutInterest, "day", "days"), groupThousands(m.Today.InterestLost),
			plural(m.BestDay.DaysWithoutInterest, "day", "days"), groupThousands(m.BestDay.InterestLost)))
	}
	steps = append(steps, fmt.Sprintf("Move %s VND from %s to %s %s.", groupThousands(m.Amount), m.From, m.To, when))
	if m.WorstCaseExtraDays > 0 {
		steps = append(steps, fmt.Sprintf("A withdrawal from %s can take up to %d working hours, which may add %s without interest; start it early in a working day.",
			m.FromProduct, *in.FromProduct.WithdrawalMaxWorkingHours, plural(m.WorstCaseExtraDays, "working day", "working days")))
	}
	if in.To.EarningStarts == EarningStartsNextWorkingDay {
		steps = append(steps, fmt.Sprintf("%s starts paying the next working day: move Monday to Thursday (or Sunday), never on Friday or Saturday, and not on the last working day before a public holiday.", in.To.Place()))
	}
	if m.BestDay.InterestLost > 0 {
		steps = append(steps, fmt.Sprintf("The %s VND lost is earned back in %s at the new rate; after that it adds about %s VND a month.",
			groupThousands(m.BestDay.InterestLost), plural(m.BestDay.PaybackDays, "day", "days"), groupThousands(m.ExtraPerMonth)))
	} else {
		steps = append(steps, fmt.Sprintf("Nothing is lost moving then; it adds about %s VND a month.", groupThousands(m.ExtraPerMonth)))
	}
	if in.To.Backing == SavingsBackingBusinessCooperation {
		steps = append(steps, in.To.Place()+" is not a bank deposit: it is a business cooperation contract with the provider's partner, with financial risk. Keep emergency money in a deposit unless you accept that.")
	}
	return steps
}
