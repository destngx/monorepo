package main

import (
	"apps/wealth-management/domain"
	"errors"
	"testing"
)

func TestGivenTestModeWhenCheckingSpreadsheetThenOnlyCopiesPass(t *testing.T) {
	title := func(name string, err error) func() (string, error) {
		return func() (string, error) { return name, err }
	}
	if err := checkTestSheet("copy", title("wm-e2e-run-20261013T000000Z", nil)); err != nil {
		t.Fatalf("copy rejected: %v", err)
	}
	if err := checkTestSheet("prod", title("Wealth 2026", nil)); !errors.Is(err, domain.ErrNotTestSheet) {
		t.Fatalf("production title accepted: %v", err)
	}
	unreachable := errors.New("forbidden")
	if err := checkTestSheet("copy", title("", unreachable)); !errors.Is(err, unreachable) {
		t.Fatalf("title read error lost: %v", err)
	}
}
