package domain

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestGivenRequestedSymbolsWhenNormalizingThenTheyAreTrimmedUpperCasedAndDeduplicated(t *testing.T) {
	got, err := NormalizeFocusSymbols([]string{" btc ", "FUND-A", "", "Btc", "^vnindex", "usd/vnd"})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"BTC", "FUND-A", "^VNINDEX", "USD/VND"}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if empty, err := NormalizeFocusSymbols(nil); err != nil || len(empty) != 0 {
		t.Fatalf("no symbols is not an error: %v %v", empty, err)
	}
}

func TestGivenABadOrTooLongSymbolListWhenNormalizingThenItIsAValidationError(t *testing.T) {
	for _, raw := range [][]string{
		{"BTC; drop"},
		{strings.Repeat("A", 21)},
		{"A1", "A2", "A3", "A4", "A5", "A6", "A7", "A8", "A9", "A10", "A11"},
	} {
		if _, err := NormalizeFocusSymbols(raw); !errors.Is(err, ErrValidation) {
			t.Errorf("%v: expected a validation error, got %v", raw, err)
		}
	}
}

func TestGivenNoRequestedSymbolsWhenResolvingTheFocusThenItIsTheHoldingsLargestFirstSymbolsOnly(t *testing.T) {
	small, large, zero := 2_000_000.0, 9_000_000.0, 0.0
	valuation := &PortfolioValuation{Holdings: []ValuedHolding{
		{Section: PortfolioSectionFunds, Holding: Holding{Platform: "Platform X", Symbol: "FUND-A"}, ValueVND: &small},
		{Section: PortfolioSectionCrypto, Holding: Holding{Platform: "Platform Y", Symbol: "BTC"}, ValueVND: &large},
		{Section: PortfolioSectionCrypto, Holding: Holding{Platform: "Platform Z", Symbol: "btc"}, ValueVND: &small},
		{Section: PortfolioSectionCrypto, Holding: Holding{Symbol: "SOLD", Quantity: &zero}},
		{Section: PortfolioSectionFunds, Holding: Holding{Symbol: "FUND-B"}},
		{Section: PortfolioSectionFunds, Holding: Holding{Symbol: " "}},
	}}

	focus := ResolveThinkTankFocus(nil, valuation, "")
	if focus.Source != ThinkTankFocusHoldings || !slices.Equal(focus.Symbols, []string{"BTC", "FUND-A", "FUND-B"}) {
		t.Fatalf("focus = %+v", focus)
	}
	block := focus.PromptBlock()
	if !strings.Contains(block, "current holdings: BTC, FUND-A, FUND-B") || strings.Contains(block, "Platform") || strings.Contains(block, "9000000") {
		t.Fatalf("the block names symbols only: %s", block)
	}
}

func TestGivenRequestedSymbolsWhenResolvingTheFocusThenTheyWinOverTheHoldings(t *testing.T) {
	value := 1.0
	valuation := &PortfolioValuation{Holdings: []ValuedHolding{{Holding: Holding{Symbol: "BTC"}, ValueVND: &value}}}
	focus := ResolveThinkTankFocus([]string{"FPT", "VCB"}, valuation, "")
	if focus.Source != ThinkTankFocusRequest || !slices.Equal(focus.Symbols, []string{"FPT", "VCB"}) {
		t.Fatalf("focus = %+v", focus)
	}
	if block := focus.PromptBlock(); !strings.Contains(block, "the user asked about: FPT, VCB") {
		t.Fatalf("block = %s", block)
	}
}

func TestGivenNoHoldingsWhenResolvingTheFocusThenThePromptSaysSoInsteadOfInventingSymbols(t *testing.T) {
	empty := ResolveThinkTankFocus(nil, &PortfolioValuation{}, "")
	if empty.Source != ThinkTankFocusHoldings || len(empty.Symbols) != 0 || !strings.Contains(empty.PromptBlock(), "no priced holdings") {
		t.Fatalf("empty = %+v %s", empty, empty.PromptBlock())
	}
	failed := ResolveThinkTankFocus(nil, nil, "sheet down")
	if failed.Symbols == nil || failed.Note != "holdings could not be read: sheet down" || !strings.Contains(failed.PromptBlock(), "could not be read") {
		t.Fatalf("failed = %+v %s", failed, failed.PromptBlock())
	}
}

func TestGivenACommandsReplyWithRecommendationsWhenDecodingThenTheyAreKeptBesideTheCommands(t *testing.T) {
	var raw InvestmentCommands
	reply := `{"executable_commands":[{"action":"OBSERVE","target":"BTC"}],
		"recommendations":[{"title":"Cap crypto at 20%","action":"Move 5.000.000 to a deposit","impactVnd":null,"evidence":"Crypto is 30% of net worth"}],
		"outcomes":[{"id":"rec-1","verdict":"met"}]}`
	if err := ParseAIJSON(reply, &raw); err != nil {
		t.Fatal(err)
	}
	if len(raw.Recommendations) != 1 || raw.Recommendations[0].Title != "Cap crypto at 20%" || len(raw.Outcomes) != 1 {
		t.Fatalf("raw = %+v", raw)
	}
	if commands, err := raw.Normalize(); err != nil || len(commands) != 1 {
		t.Fatalf("commands = %+v %v", commands, err)
	}
}
