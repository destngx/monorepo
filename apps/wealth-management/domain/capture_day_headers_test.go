package domain

import (
	"strconv"
	"strings"
	"testing"
)

func headerEvents(t *testing.T, events string) []CaptureEvent {
	t.Helper()
	extraction, err := ParseCaptureExtraction(`{"events":[`+events+`]}`, captureScreenshot, captureToday)
	if err != nil {
		t.Fatal(err)
	}
	return extraction.Events
}

func headerEvent(header, occurredAt string, amount int) string {
	return `{"occurredAt":"` + occurredAt + `","amount":` + strconv.Itoa(amount) + `,"direction":"out","dayHeader":"` + header + `"}`
}

func TestGivenDayHeadersWhenParsingThenEachRowTakesItsHeadersDate(t *testing.T) {
	// Today is 2026-10-19; the model dated the "Hôm qua" row today.
	events := headerEvents(t, strings.Join([]string{
		headerEvent("Hôm nay", "2026-10-19 08:00", 1),
		headerEvent("Hôm qua", "2026-10-19 22:15", 2),
		headerEvent("17/10", "2026-10-17", 3),
		headerEvent("Thứ Hai", "2026-10-13", 4),
	}, ","))

	want := []string{"2026-10-19 08:00", "2026-10-18 22:15", "2026-10-17", "2026-10-13"}
	for index, event := range events {
		got := event.At.Format("2006-01-02 15:04")
		if !event.HasTime {
			got = event.At.Format("2006-01-02")
		}
		if got != want[index] || event.DateCheck != "" {
			t.Fatalf("row %d: %s check=%q, want %s", index, got, event.DateCheck, want[index])
		}
	}
}

func TestGivenAHiddenHeaderWithOneDayBetweenItsNeighboursWhenParsingThenThatDayIsUsed(t *testing.T) {
	events := headerEvents(t, strings.Join([]string{
		headerEvent("Hôm nay", "2026-10-19 08:00", 1),
		headerEvent("hidden", "2026-10-19 21:00", 2),
		headerEvent("hidden", "2026-10-19", 3),
		headerEvent("17/10/2026", "2026-10-17", 4),
	}, ","))

	for _, event := range events[1:3] {
		if event.Date() != captureToday.AddDate(0, 0, -1) || event.DateCheck != CaptureDateInferred {
			t.Fatalf("a covered Hôm qua is the only day between: %+v", event)
		}
	}
	if !events[1].HasTime || events[1].At.Hour() != 21 {
		t.Fatalf("the time of day is kept: %+v", events[1])
	}
}

func TestGivenAHiddenHeaderWithSeveralDaysPossibleWhenParsingThenTheDateIsLeftToCheck(t *testing.T) {
	events := headerEvents(t, strings.Join([]string{
		headerEvent("Hôm nay", "2026-10-19 08:00", 1),
		headerEvent("hidden", "2026-10-19", 2),
		headerEvent("16/10/2026", "2026-10-16", 3),
		headerEvent("hidden", "2026-10-15", 4),
	}, ","))

	if events[1].Date() != captureToday || events[1].DateCheck != CaptureDateHidden {
		t.Fatalf("18 or 17: the model's date stays, flagged: %+v", events[1])
	}
	if events[3].DateCheck != CaptureDateHidden {
		t.Fatalf("a hidden header with nothing below is flagged: %+v", events[3])
	}
}

func TestGivenACheckedDateWhenScoringThenItCostsPoints(t *testing.T) {
	inferred, hidden := confident(0.95), confident(0.95)
	inferred.DateCheck, hidden.DateCheck = CaptureDateInferred, CaptureDateHidden
	suggestions := []CaptureSuggestion{inferred, hidden}

	ApplyCaptureConfidence(suggestions)

	if got := suggestions[0]; got.Overall != 85 || !strings.Contains(reasonTexts(got), "Date worked out from the day headers around it: check it") {
		t.Fatalf("inferred: overall=%d reasons=%q", got.Overall, reasonTexts(got))
	}
	if got := suggestions[1]; got.Overall != 75 || !strings.Contains(reasonTexts(got), "Its day header is hidden: check the date") {
		t.Fatalf("hidden: overall=%d reasons=%q", got.Overall, reasonTexts(got))
	}
}
