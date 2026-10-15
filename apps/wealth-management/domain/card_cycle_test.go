package domain

import (
	"testing"
	"time"
)

func TestComputeCardCycleAfterStatementDay(t *testing.T) {
	cycle := ComputeCardCycle(5, 30, day(2026, 3, 20))
	if cycle.CycleStart != "2026-03-05" || cycle.CycleEnd != "2026-04-05" || cycle.CycleDays != 31 || cycle.CycleElapsedDays != 15 {
		t.Fatalf("unexpected cycle %+v", cycle)
	}
	if cycle.DueDay != 30 || cycle.PayBy != "2026-03-29" || cycle.DaysUntilPayBy != 9 || cycle.PayByUrgent {
		t.Fatalf("unexpected pay-by info %+v", cycle)
	}
}

func TestComputeCardCycleBeforeStatementDayWrapsToLastMonth(t *testing.T) {
	cycle := ComputeCardCycle(5, 30, day(2026, 3, 2))
	if cycle.CycleStart != "2026-02-05" || cycle.CycleElapsedDays != 25 || cycle.CycleDays != 28 {
		t.Fatalf("unexpected cycle %+v", cycle)
	}
}

func TestPayByDateIsThe29thAndThe28thInFebruary(t *testing.T) {
	tests := map[time.Time]string{
		day(2026, 3, 20):  "2026-03-29",
		day(2026, 3, 29):  "2026-03-29",
		day(2026, 3, 30):  "2026-04-29",
		day(2026, 2, 10):  "2026-02-28",
		day(2028, 2, 10):  "2028-02-28", // leap year too
		day(2028, 1, 31):  "2028-02-28",
		day(2026, 12, 30): "2027-01-29",
	}
	for today, want := range tests {
		if got := PayByDate(today.Add(15 * time.Hour)).Format(time.DateOnly); got != want {
			t.Fatalf("%s: got %s want %s", today.Format(time.DateOnly), got, want)
		}
	}
}

func TestPeriodBounds(t *testing.T) {
	start, end := PeriodBounds(CapPeriodCycle, 5, day(2026, 3, 4))
	if !start.Equal(day(2026, 2, 5)) || !end.Equal(day(2026, 3, 5)) {
		t.Fatalf("cycle: %v - %v", start, end)
	}
	start, end = PeriodBounds(CapPeriodMonth, 5, day(2026, 3, 4))
	if !start.Equal(day(2026, 3, 1)) || !end.Equal(day(2026, 4, 1)) {
		t.Fatalf("month: %v - %v", start, end)
	}
	start, end = PeriodBounds(CapPeriodCycle, 31, day(2026, 2, 28))
	if !start.Equal(day(2026, 2, 28)) || !end.Equal(day(2026, 3, 31)) {
		t.Fatalf("clamped cycle: %v - %v", start, end)
	}
}

func TestUtilizationBandFor(t *testing.T) {
	if UtilizationBandFor(29.9) != UtilizationGood || UtilizationBandFor(30) != UtilizationFair || UtilizationBandFor(70) != UtilizationHigh {
		t.Fatal("unexpected utilisation bands")
	}
}
