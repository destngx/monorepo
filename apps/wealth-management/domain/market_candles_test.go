package domain

import (
	"errors"
	"testing"
)

func TestGivenTimeframeTextWhenParseTimeframeThenAcceptsKnownValuesOnly(t *testing.T) {
	for _, raw := range []string{"1h", "4H", " 1d ", "1w"} {
		if _, err := ParseTimeframe(raw); err != nil {
			t.Fatalf("ParseTimeframe(%q) error: %v", raw, err)
		}
	}
	if _, err := ParseTimeframe("15m"); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected a validation error, got %v", err)
	}
}

func TestGivenSymbolsWhenNormaliseMarketSymbolThenUppercasesAndDropsCaret(t *testing.T) {
	if got := NormaliseMarketSymbol(" ^vnIndex "); got != "VNINDEX" {
		t.Fatalf("got %q", got)
	}
}
