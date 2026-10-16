package domain

import (
	"errors"
	"strings"
	"testing"
)

func number(value float64) *float64 { return &value }

func TestGivenNoPreferencesWhenParsingThenDefaultsApplyAndTargetsAreUnset(t *testing.T) {
	prefs := ParsePreferences(nil, 2)
	if prefs.AllocationTargets != nil || prefs.GoalPool != DefaultGoalPoolSettings() || len(prefs.Issues) != 0 {
		t.Fatalf("prefs %+v", prefs)
	}
	if prefs.GoalPool.EmergencyMonths != 6 || prefs.GoalPool.SpendingBufferMonths != 1 {
		t.Fatalf("defaults %+v", prefs.GoalPool)
	}
}

func TestGivenATabEditedByHandWhenParsingThenValidRowsApplyAndBadRowsAreReported(t *testing.T) {
	rows := [][]string{
		{"Key", "Value", "Note"},
		{},
		{"allocation.target.cash", "60", "a note"},
		{"Allocation.Target.Funds", "30"},
		{"allocation.target.crypto", "10"},
		{"goal.emergencyMonths", "abc"},
		{"goal.spendingBufferMonths", "2"},
		{"goal.spendingBufferMonths", "3"},
		{"theme", "1"},
	}
	prefs := ParsePreferences(rows, 1)
	if prefs.AllocationTargets == nil || *prefs.AllocationTargets != (AllocationTargets{Cash: 60, Funds: 30, Crypto: 10}) {
		t.Fatalf("targets %+v", prefs.AllocationTargets)
	}
	// the bad emergency value falls back to the default; the first buffer row wins
	if prefs.GoalPool != (GoalPoolSettings{EmergencyMonths: 6, SpendingBufferMonths: 2}) {
		t.Fatalf("goal pool %+v", prefs.GoalPool)
	}
	if len(prefs.Issues) != 3 || !strings.Contains(prefs.Issues[0], "row 6") || !strings.Contains(prefs.Issues[1], "row 8") || !strings.Contains(prefs.Issues[2], `unknown key "theme"`) {
		t.Fatalf("issues %q", prefs.Issues)
	}
	if len(prefs.Entries) != 5 || prefs.Entries[0].Row != 3 || prefs.Entries[0].Note != "a note" || prefs.Entries[3].Value != nil {
		t.Fatalf("entries %+v", prefs.Entries)
	}
}

func TestGivenTargetsThatDoNotAddUpWhenParsingThenTargetsAreIgnoredWithAnIssue(t *testing.T) {
	for name, rows := range map[string][][]string{
		"sum":     {{"allocation.target.cash", "60"}, {"allocation.target.funds", "30"}, {"allocation.target.crypto", "20"}},
		"partial": {{"allocation.target.cash", "100"}},
		"range":   {{"allocation.target.cash", "120"}, {"allocation.target.funds", "-10"}, {"allocation.target.crypto", "-10"}},
	} {
		prefs := ParsePreferences(rows, 2)
		if prefs.AllocationTargets != nil || len(prefs.Issues) == 0 {
			t.Errorf("%s: prefs %+v", name, prefs)
		}
	}
}

func TestGivenDecimalTargetsWhenTheyAddUpTo100ThenTheyAreAccepted(t *testing.T) {
	if err := (AllocationTargets{Cash: 33.33, Funds: 33.33, Crypto: 33.34}).Validate(); err != nil {
		t.Fatal(err)
	}
	err := (AllocationTargets{Cash: 50, Funds: 30, Crypto: 10}).Validate()
	if !errors.Is(err, ErrValidation) || !strings.Contains(err.Error(), "add up to 100 (now 90)") {
		t.Fatalf("err = %v", err)
	}
}

func TestGivenAnUpdateWhenApplyingThenExistingRowsChangeAndNewKeysAreAppended(t *testing.T) {
	current := ParsePreferences([][]string{{"goal.emergencyMonths", "6", "kept"}}, 2)
	next, err := ApplyPreferencesUpdate(current, PreferencesUpdate{Values: []PreferenceValue{
		{Key: PrefGoalEmergencyMonths, Value: number(9)},
		{Key: PrefAllocationTargetCash, Value: number(70)},
		{Key: PrefAllocationTargetFunds, Value: number(20)},
		{Key: PrefAllocationTargetCrypto, Value: number(10)},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if next.GoalPool.EmergencyMonths != 9 || next.AllocationTargets == nil || next.AllocationTargets.Cash != 70 {
		t.Fatalf("next %+v", next)
	}
	if next.Entries[0].Row != 2 || next.Entries[0].Note != "kept" || next.Entries[1].Row != 0 || next.Entries[1].Note == "" {
		t.Fatalf("entries %+v", next.Entries)
	}
	if current.GoalPool.EmergencyMonths != 6 {
		t.Fatal("the current preferences must not change")
	}
}

func TestGivenAnInvalidPreferencesUpdateWhenApplyingThenItIsRejected(t *testing.T) {
	targets := ParsePreferences([][]string{{"allocation.target.cash", "50"}, {"allocation.target.funds", "50"}, {"allocation.target.crypto", "0"}}, 2)
	cases := map[string]PreferencesUpdate{
		"empty":       {},
		"unknown key": {Values: []PreferenceValue{{Key: "theme", Value: number(1)}}},
		"duplicate":   {Values: []PreferenceValue{{Key: PrefGoalEmergencyMonths, Value: number(3)}, {Key: PrefGoalEmergencyMonths, Value: number(4)}}},
		"range":       {Values: []PreferenceValue{{Key: PrefGoalEmergencyMonths, Value: number(0)}}},
		"buffer":      {Values: []PreferenceValue{{Key: PrefGoalSpendingBufferMonths, Value: number(13)}}},
		"sum":         {Values: []PreferenceValue{{Key: PrefAllocationTargetCash, Value: number(60)}}},
		"partial":     {Values: []PreferenceValue{{Key: PrefAllocationTargetCrypto, Value: nil}}},
	}
	for name, update := range cases {
		if _, err := ApplyPreferencesUpdate(targets, update); !errors.Is(err, ErrValidation) {
			t.Errorf("%s: err = %v, want validation", name, err)
		}
	}
}

func TestGivenTargetsWhenClearingAllThreeThenTargetsAreUnset(t *testing.T) {
	current := ParsePreferences([][]string{{"allocation.target.cash", "50"}, {"allocation.target.funds", "50"}, {"allocation.target.crypto", "0"}}, 2)
	next, err := ApplyPreferencesUpdate(current, PreferencesUpdate{Values: []PreferenceValue{
		{Key: PrefAllocationTargetCash}, {Key: PrefAllocationTargetFunds}, {Key: PrefAllocationTargetCrypto},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if next.AllocationTargets != nil || len(next.Entries) != 3 || next.Entries[0].Value != nil {
		t.Fatalf("next %+v", next)
	}
}

func TestGivenABadTargetRowWhenSavingAnotherKeyThenTheSaveIsNotBlocked(t *testing.T) {
	current := ParsePreferences([][]string{{"allocation.target.cash", "70"}}, 2)
	next, err := ApplyPreferencesUpdate(current, PreferencesUpdate{Values: []PreferenceValue{{Key: PrefGoalSpendingBufferMonths, Value: number(0)}}})
	if err != nil || next.GoalPool.SpendingBufferMonths != 0 {
		t.Fatalf("next %+v err %v", next, err)
	}
}
