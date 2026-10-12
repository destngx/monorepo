package domain

import "testing"

func TestParseCardDays(t *testing.T) {
	tests := []struct {
		input     *string
		statement int
		due       int
		ok        bool
	}{
		{text("5, 30"), 5, 30, true},
		{text("5,30"), 5, 30, true},
		{text("12 28"), 12, 28, true},
		{text("1/4/2026"), 0, 0, false},
		{text("5, 40"), 0, 0, false},
		{nil, 0, 0, false},
	}
	for _, tt := range tests {
		statement, due, ok := ParseCardDays(tt.input)
		if statement != tt.statement || due != tt.due || ok != tt.ok {
			t.Fatalf("input %v: got %d %d %v", tt.input, statement, due, ok)
		}
	}
}

func TestComputeCardCycleAfterStatementDay(t *testing.T) {
	cycle := ComputeCardCycle(5, 30, -4_000_000, day(2026, 3, 20))
	if cycle.CycleStart != "2026-03-05" || cycle.CycleEnd != "2026-04-05" || cycle.CycleDays != 31 || cycle.CycleElapsedDays != 15 {
		t.Fatalf("unexpected cycle %+v", cycle)
	}
	if cycle.NextDueDate != "2026-03-30" || cycle.DaysUntilDue != 10 || cycle.DueUrgent {
		t.Fatalf("unexpected due info %+v", cycle)
	}
	if cycle.MinimumPayment != 200_000 || cycle.OutstandingAmount != 4_000_000 {
		t.Fatalf("unexpected payment info %+v", cycle)
	}
}

func TestComputeCardCycleBeforeStatementDayWrapsToLastMonth(t *testing.T) {
	cycle := ComputeCardCycle(5, 30, -1, day(2026, 3, 2))
	if cycle.CycleStart != "2026-02-05" || cycle.CycleElapsedDays != 25 || cycle.CycleDays != 28 {
		t.Fatalf("unexpected cycle %+v", cycle)
	}
}

func TestComputeCardCycleDueDayClampsToMonthLengthAndRollsOver(t *testing.T) {
	// Due day 30 in February clamps to the 28th; on 1 March the next due is 30 March.
	feb := ComputeCardCycle(5, 30, -100, day(2026, 2, 26))
	if feb.NextDueDate != "2026-02-28" || feb.DaysUntilDue != 2 || !feb.DueUrgent {
		t.Fatalf("unexpected february due %+v", feb)
	}
	rolled := ComputeCardCycle(5, 10, -100, day(2026, 3, 11))
	if rolled.NextDueDate != "2026-04-10" {
		t.Fatalf("expected rollover to next month, got %+v", rolled)
	}
}

func TestDueNotUrgentWhenNothingOwed(t *testing.T) {
	if ComputeCardCycle(5, 30, 0, day(2026, 3, 29)).DueUrgent {
		t.Fatal("expected no urgency with zero balance")
	}
}

func TestUtilizationBandFor(t *testing.T) {
	if UtilizationBandFor(29.9) != UtilizationGood || UtilizationBandFor(30) != UtilizationFair || UtilizationBandFor(70) != UtilizationHigh {
		t.Fatal("unexpected utilisation bands")
	}
}
