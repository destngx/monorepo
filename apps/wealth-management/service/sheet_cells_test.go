package service

import "testing"

func TestClearedCellIsWrittenAsX(t *testing.T) {
	if got := clearedCell(true); got != "x" {
		t.Fatalf("cleared = %q", got)
	}
	if got := clearedCell(false); got != "" {
		t.Fatalf("uncleared = %q", got)
	}
}

func TestParseClearedAcceptsXAndLegacyMarks(t *testing.T) {
	for _, value := range []string{"x", " X ", "y", "Yes", "true", "✓"} {
		if !parseCleared(value) {
			t.Errorf("parseCleared(%q) = false", value)
		}
	}
	for _, value := range []string{"", "n", "no", "-"} {
		if parseCleared(value) {
			t.Errorf("parseCleared(%q) = true", value)
		}
	}
}
