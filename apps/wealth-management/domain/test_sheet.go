package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// TestSheetPrefix starts the name of every disposable spreadsheet copy made
// by the live test harness (WM-TEST-1).
const TestSheetPrefix = "wm-e2e-"

// ErrNotTestSheet stops a test-mode engine or harness from touching a
// spreadsheet that is not a disposable copy (WM-TEST-2).
var ErrNotTestSheet = errors.New("not a disposable test spreadsheet")

// TestSheetName is "wm-e2e-<runID>-<UTC time>".
func TestSheetName(runID string, now time.Time) string {
	return TestSheetPrefix + runID + "-" + now.UTC().Format("20060102T150405Z")
}

// IsTestSheetTitle reports whether title names a disposable copy.
func IsTestSheetTitle(title string) bool {
	return strings.HasPrefix(title, TestSheetPrefix)
}

// CheckTestSheet allows a spreadsheet only when it is a disposable copy
// distinct from the production spreadsheet.
func CheckTestSheet(id string, title string, productionID string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: empty spreadsheet id", ErrNotTestSheet)
	}
	if id == productionID {
		return fmt.Errorf("%w: it is the production spreadsheet", ErrNotTestSheet)
	}
	if !IsTestSheetTitle(title) {
		return fmt.Errorf("%w: title %q does not start with %q", ErrNotTestSheet, title, TestSheetPrefix)
	}
	return nil
}
