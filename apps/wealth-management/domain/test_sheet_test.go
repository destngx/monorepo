package domain

import (
	"errors"
	"testing"
	"time"
)

func TestGivenRunWhenNamingCopyThenUsesPrefixRunAndUTCTime(t *testing.T) {
	now := time.Date(2026, 10, 13, 8, 30, 5, 0, time.FixedZone("ICT", 7*3600))
	if got := TestSheetName("abc123", now); got != "wm-e2e-abc123-20261013T013005Z" {
		t.Fatalf("got %q", got)
	}
}

func TestGivenSpreadsheetWhenCheckingTestSheetThenOnlyAllowsDisposableCopies(t *testing.T) {
	cases := []struct {
		name, id, title string
		ok              bool
	}{
		{"copy", "copy-1", "wm-e2e-run-20261013T000000Z", true},
		{"production id", "prod", "wm-e2e-run", false},
		{"production title", "copy-1", "Wealth 2026", false},
		{"prefix not at start", "copy-1", "Copy of wm-e2e-run", false},
		{"empty id", "", "wm-e2e-run", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckTestSheet(tc.id, tc.title, "prod")
			if tc.ok != (err == nil) || (err != nil && !errors.Is(err, ErrNotTestSheet)) {
				t.Fatalf("CheckTestSheet(%q, %q) = %v", tc.id, tc.title, err)
			}
		})
	}
}
