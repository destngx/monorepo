package domain

import (
	"slices"
	"strings"
	"testing"
)

// nextWorkingDayProduct is a wallet product that pays from the next
// working day; slowWithdrawal adds MoMo's 16 working hours.
func nextWorkingDayProduct(id string, rate float64, slowWithdrawal bool) FlexibleSaving {
	product := FlexibleSaving{
		ID: id, Provider: "Wallet " + strings.ToUpper(id[:1]), Product: "Earn", Rate: rate,
		Backing: SavingsBackingDeposit, EarningStarts: EarningStartsNextWorkingDay, URL: "https://example.test/" + id,
	}
	if slowWithdrawal {
		hours := 16
		product.WithdrawalMaxWorkingHours = &hours
	}
	return product
}

func TestGivenANextWorkingDayProductWhenMovingOnEachWeekdayThenTheWeekendCostsMore(t *testing.T) {
	// 2026-10-19 is a Monday.
	want := map[string]int{"Monday": 1, "Tuesday": 1, "Wednesday": 1, "Thursday": 1, "Friday": 3, "Saturday": 2, "Sunday": 1}
	for offset := range 7 {
		moveDay := day(2026, 10, 19+offset)
		if got := daysWithoutInterest(moveDay, EarningStartsNextWorkingDay); got != want[moveDay.Weekday().String()] {
			t.Errorf("%s: %d days without interest, want %d", moveDay.Weekday(), got, want[moveDay.Weekday().String()])
		}
		if got := daysWithoutInterest(moveDay, ""); got != 0 {
			t.Errorf("%s: a deposit paying from the same day loses %d days", moveDay.Weekday(), got)
		}
	}
}

func TestGivenAFridayWhenPlanningAMoveThenSundayIsTheCheapestDayAndThePaybackIsPriced(t *testing.T) {
	to := FlexibleDepositRate(nextWorkingDayProduct("b", 5.5, false))
	move, ok := planSavingsMove(savingsMoveInput{From: "Wallet A", FromRate: 4, Amount: 73_000_000, To: to, Today: day(2026, 10, 23)})
	if !ok {
		t.Fatal("no move planned")
	}
	// 73M x 1.5 points = 1.095M a year, 91.250 a month.
	if move.RateGapPoints != 1.5 || move.ExtraPerYear != 1_095_000 || move.ExtraPerMonth != 91_250 {
		t.Fatalf("gain = %+v", move)
	}
	// Friday: 3 days at 4% on 73M = 24.000; earned back in ceil(3 x 4 / 1.5) = 8 days.
	if move.Today.Weekday != "Friday" || move.Today.DaysWithoutInterest != 3 || move.Today.InterestLost != 24_000 || move.Today.PaybackDays != 8 {
		t.Fatalf("today = %+v", move.Today)
	}
	if move.BestDay.Date != "2026-10-25" || move.BestDay.DaysWithoutInterest != 1 || move.BestDay.InterestLost != 8_000 || move.BestDay.PaybackDays != 3 {
		t.Fatalf("best day = %+v", move.BestDay)
	}
	if !strings.HasPrefix(move.Steps[0], "Wait until Sunday 2026-10-25") || !slices.ContainsFunc(move.Steps, func(s string) bool { return strings.Contains(s, "never on Friday or Saturday") }) {
		t.Fatalf("steps = %q", move.Steps)
	}
}

func TestGivenASlowWithdrawalWhenPlanningAMoveThenTheWorstCaseIsStated(t *testing.T) {
	from := nextWorkingDayProduct("m", 3, true)
	to := FlexibleDepositRate(nextWorkingDayProduct("z", 4.5, false))
	move, ok := planSavingsMove(savingsMoveInput{From: "Wallet M", FromRate: 3, FromProduct: &from, Amount: 50_000_000, To: to, Today: day(2026, 10, 20)})
	if !ok || move.WorstCaseExtraDays != 2 || move.FromProduct != "Wallet M Earn" {
		t.Fatalf("move = %+v", move)
	}
	if move.BestDay.Date != move.Today.Date {
		t.Fatalf("a Tuesday is as cheap as any day, best = %+v", move.BestDay)
	}
	if !slices.ContainsFunc(move.Steps, func(s string) bool { return strings.Contains(s, "up to 16 working hours") }) {
		t.Fatalf("steps = %q", move.Steps)
	}
}

func TestGivenAFromRateOfZeroWhenPlanningThenNothingIsLost(t *testing.T) {
	to := FlexibleDepositRate(nextWorkingDayProduct("z", 4.5, false))
	move, ok := planSavingsMove(savingsMoveInput{From: "Bank A", Amount: 10_000_000, To: to, Today: day(2026, 10, 23)})
	if !ok || move.Today.InterestLost != 0 || move.Today.PaybackDays != 0 {
		t.Fatalf("move = %+v", move)
	}
}

func TestGivenANotBetterPlaceWhenPlanningThenNoMove(t *testing.T) {
	to := FlexibleDepositRate(nextWorkingDayProduct("z", 4.5, false))
	if _, ok := planSavingsMove(savingsMoveInput{From: "A", FromRate: 4.5, Amount: 10_000_000, To: to, Today: billsToday}); ok {
		t.Fatal("moved to an equal rate")
	}
	if _, ok := planSavingsMove(savingsMoveInput{From: "A", FromRate: 1, To: to, Today: billsToday}); ok {
		t.Fatal("moved nothing")
	}
}

func TestGivenABusinessCooperationProductWhenPlanningThenTheStepsSayItIsNotADeposit(t *testing.T) {
	product := nextWorkingDayProduct("m", 6, false)
	product.Backing = SavingsBackingBusinessCooperation
	move, _ := planSavingsMove(savingsMoveInput{From: "A", FromRate: 1, Amount: 10_000_000, To: FlexibleDepositRate(product), Today: billsToday})
	if !strings.Contains(move.Steps[len(move.Steps)-1], "is not a bank deposit") {
		t.Fatalf("steps = %q", move.Steps)
	}
}
