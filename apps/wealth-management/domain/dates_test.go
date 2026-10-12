package domain

import (
	"errors"
	"testing"
	"time"
)

func TestParseSheetDate(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  time.Time
	}{
		{"day before month in the written format", "05/03/2026", time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)},
		{"single digit day and month", "5/3/2026", time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)},
		{"day above 12 still day first", "25/12/2025", time.Date(2025, 12, 25, 0, 0, 0, 0, time.UTC)},
		{"two digit year", "05/03/26", time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)},
		{"iso date", "2026-03-05", time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)},
		{"sheets serial", "46086", time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)},
		{"sheets serial with time fraction", "46086.75", time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)},
		{"rfc3339 converted to local calendar day", "2026-03-04T18:30:00Z", time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSheetDate(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.Equal(tt.want) {
				t.Fatalf("got %s, want %s", got.Format(time.DateOnly), tt.want.Format(time.DateOnly))
			}
		})
	}
}

func TestParseSheetDateRejectsGarbage(t *testing.T) {
	for _, input := range []string{"", "  ", "tomorrow", "13/13/2026", "31/02/2026", "12"} {
		t.Run(input, func(t *testing.T) {
			if _, err := ParseSheetDate(input); !errors.Is(err, ErrInvalidDate) {
				t.Fatalf("expected ErrInvalidDate for %q, got %v", input, err)
			}
		})
	}
}

func TestGivenWrittenDateWhenReadBackThenSameCalendarDay(t *testing.T) {
	date := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)
	got, err := ParseSheetDate(date.Format(SheetDateLayout))
	if err != nil || !got.Equal(date) {
		t.Fatalf("round trip failed: got %v err %v", got, err)
	}
}

func TestTodayUsesVietnamTimezone(t *testing.T) {
	// 20:00 UTC on 4 March is 03:00 on 5 March in Ho Chi Minh City.
	now := time.Date(2026, 3, 4, 20, 0, 0, 0, time.UTC)
	if got := Today(now); !got.Equal(time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("got %s", got)
	}
}

func TestDaysInMonth(t *testing.T) {
	if DaysInMonth(time.Date(2028, 2, 10, 0, 0, 0, 0, time.UTC)) != 29 {
		t.Fatal("expected leap February to have 29 days")
	}
	if DaysInMonth(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)) != 30 {
		t.Fatal("expected April to have 30 days")
	}
}
