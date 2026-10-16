package domain

import (
	"errors"
	"strconv"
	"strings"
	"testing"
)

func TestGivenGoalNamesWhenCheckingTheEmergencyRuleThenNameOrNoteDecides(t *testing.T) {
	cases := []struct {
		name, note string
		want       bool
	}{
		{"Emergency fund", "", true},
		{"Quỹ khẩn cấp", "", true},
		{"Quỹ dự phòng", "", true},
		{"Safety net", "emergency money", true},
		{"New car", "", false},
		{"Trip", "fun", false},
	}
	for _, c := range cases {
		if got := IsEmergencyFund(c.name, c.note); got != c.want {
			t.Errorf("IsEmergencyFund(%q, %q) = %v, want %v", c.name, c.note, got, c.want)
		}
	}
}

func TestGivenTargetDateCellsWhenParsingThenTextAndDateSerialsAreAccepted(t *testing.T) {
	serial := DateToSerial(day(2030, 6, 1))
	cases := map[string]string{
		"2030-06":            "2030-06",
		" 2030-6 ":           "2030-06",
		"06/2030":            "2030-06",
		"2030-06-15":         "2030-06",
		"":                   "",
		strconv.Itoa(serial): "2030-06",
	}
	for cell, want := range cases {
		got, err := ParseGoalMonth(cell)
		if err != nil || got != want {
			t.Errorf("ParseGoalMonth(%q) = %q, %v; want %q", cell, got, err, want)
		}
	}
	for _, cell := range []string{"soon", "2030-13", "12"} {
		if _, err := ParseGoalMonth(cell); err == nil {
			t.Errorf("ParseGoalMonth(%q) should fail", cell)
		}
	}
}

func TestGivenStatusCellsWhenParsingThenBlankIsActiveAndUnknownIsRejected(t *testing.T) {
	for cell, want := range map[string]GoalStatus{"": GoalActive, "Active": GoalActive, " paused ": GoalPaused, "DONE": GoalDone} {
		if got, err := ParseGoalStatus(cell); err != nil || got != want {
			t.Errorf("ParseGoalStatus(%q) = %q, %v", cell, got, err)
		}
	}
	if _, err := ParseGoalStatus("finished"); err == nil || !strings.Contains(err.Error(), "active, paused or done") {
		t.Fatalf("unknown status must name the allowed values: %v", err)
	}
}

func TestGivenGoalInputsWhenValidatingThenProblemsAreNamed(t *testing.T) {
	valid := GoalInput{Name: "New car", Target: money(100_000_000), TargetDate: "2028-01", Priority: 2}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid input: %v", err)
	}
	emergency := GoalInput{Name: "Emergency fund"}
	if err := emergency.Validate(); err != nil {
		t.Fatalf("the emergency fund may leave its target blank: %v", err)
	}
	cases := map[string]GoalInput{
		"name is required":                {Target: money(1)},
		"target is required":              {Name: "Trip"},
		"target must be positive":         {Name: "Trip", Target: money(-5)},
		"targetDate":                      {Name: "Trip", Target: money(1), TargetDate: "next year"},
		"priority must be 1 or more":      {Name: "Trip", Target: money(1), Priority: -1},
		"startingAmount must not be":      {Name: "Trip", Target: money(1), StartingAmount: money(-1)},
		"status \"later\" must be one of": {Name: "Trip", Target: money(1), Status: "later"},
	}
	for want, input := range cases {
		err := input.Validate()
		if !errors.Is(err, ErrValidation) || !strings.Contains(err.Error(), want) {
			t.Errorf("%+v: error %v should contain %q", input, err, want)
		}
	}
}

func TestGivenReorderInputsWhenValidatingThenIDsMustBeUnique(t *testing.T) {
	if err := (GoalReorderInput{IDs: []string{"goal-2", "goal-3"}}).Validate(); err != nil {
		t.Fatal(err)
	}
	for _, input := range []GoalReorderInput{{}, {IDs: []string{"goal-2", "goal-2"}}} {
		if err := input.Validate(); !errors.Is(err, ErrValidation) {
			t.Errorf("%+v should be rejected, got %v", input, err)
		}
	}
}
