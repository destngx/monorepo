package domain

import (
	"cmp"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Think Tank focus (ROADMAP M5, WM-INV-29): what the debate is about. The
// symbols the user asked for, else the user's current holdings. The focus
// names symbols only: no platforms, quantities or values.

type ThinkTankFocusSource string

const (
	// ThinkTankFocusHoldings is the default: the user's current holdings.
	ThinkTankFocusHoldings ThinkTankFocusSource = "holdings"
	// ThinkTankFocusRequest is a symbol list the user asked about.
	ThinkTankFocusRequest ThinkTankFocusSource = "request"

	// ThinkTankFocusLimit caps the symbols in a focus.
	ThinkTankFocusLimit = 10
	// RecommendationSourceThinkTank is the think tank's source id in the
	// recommendations list (WM-REC-8).
	RecommendationSourceThinkTank = "think-tank"
	// ThinkTankPage is the think tank's app route.
	ThinkTankPage = "/investments/think-tank"

	focusSymbolMax = 20
)

var focusSymbolPattern = regexp.MustCompile(`^\^?[A-Z0-9][A-Z0-9./_-]*$`)

// ThinkTankFocus is what one think tank run analyses.
type ThinkTankFocus struct {
	Source  ThinkTankFocusSource `json:"source"`
	Symbols []string             `json:"symbols"`
	// Note explains an empty holdings focus (e.g. the valuation failed).
	Note string `json:"note,omitempty"`
}

// NormalizeFocusSymbols trims and upper-cases requested symbols, drops
// blanks and repeats, and rejects anything that is not a ticker-like code
// or a list longer than ThinkTankFocusLimit (ErrValidation).
func NormalizeFocusSymbols(raw []string) ([]string, error) {
	out := []string{}
	for _, value := range raw {
		symbol := strings.ToUpper(strings.TrimSpace(value))
		if symbol == "" || slices.Contains(out, symbol) {
			continue
		}
		if len(symbol) > focusSymbolMax || !focusSymbolPattern.MatchString(symbol) {
			return nil, fmt.Errorf("%w: %q is not a symbol (letters, digits and . ^ / _ - only, at most %d characters)", ErrValidation, value, focusSymbolMax)
		}
		out = append(out, symbol)
	}
	if len(out) > ThinkTankFocusLimit {
		return nil, fmt.Errorf("%w: at most %d symbols, got %d", ErrValidation, ThinkTankFocusLimit, len(out))
	}
	return out, nil
}

// ResolveThinkTankFocus is the requested symbols (already normalized) when
// there are any, else the held symbols, largest VND value first, at most
// ThinkTankFocusLimit. Holdings sold to zero are left out. valuationErr
// explains a missing valuation.
func ResolveThinkTankFocus(requested []string, valuation *PortfolioValuation, valuationErr string) ThinkTankFocus {
	if len(requested) > 0 {
		return ThinkTankFocus{Source: ThinkTankFocusRequest, Symbols: slices.Clone(requested)}
	}
	focus := ThinkTankFocus{Source: ThinkTankFocusHoldings, Symbols: []string{}}
	if valuation == nil {
		focus.Note = "holdings could not be read"
		if valuationErr != "" {
			focus.Note += ": " + valuationErr
		}
		return focus
	}
	type held struct {
		symbol string
		value  float64
	}
	totals := []held{}
	index := map[string]int{}
	for _, holding := range valuation.Holdings {
		symbol := strings.ToUpper(strings.TrimSpace(holding.Symbol))
		if symbol == "" || (holding.Quantity != nil && *holding.Quantity <= 0) {
			continue
		}
		value := 0.0
		if holding.ValueVND != nil {
			value = *holding.ValueVND
		}
		if i, ok := index[symbol]; ok {
			totals[i].value += value
			continue
		}
		index[symbol] = len(totals)
		totals = append(totals, held{symbol, value})
	}
	slices.SortStableFunc(totals, func(a, b held) int { return cmp.Compare(b.value, a.value) })
	for _, item := range totals[:min(len(totals), ThinkTankFocusLimit)] {
		focus.Symbols = append(focus.Symbols, item.symbol)
	}
	return focus
}

// PromptBlock tells the models what to analyse; engine code appended to
// every phase, so the sheet prompts need no new placeholder.
func (f ThinkTankFocus) PromptBlock() string {
	switch {
	case f.Source == ThinkTankFocusRequest:
		return "\n\nFOCUS: analyse these symbols the user asked about: " + strings.Join(f.Symbols, ", ") + ". Relate them to the user's holdings and cash above."
	case len(f.Symbols) > 0:
		return "\n\nFOCUS: no topic was given, so analyse the user's current holdings: " + strings.Join(f.Symbols, ", ") + ". Keep every command about these or about cash."
	case f.Note != "":
		return "\n\nFOCUS: no topic was given and the user's " + f.Note + ". Analyse the market for the user's cash; do not invent holdings."
	}
	return "\n\nFOCUS: no topic was given and no priced holdings were found. Analyse the market for the user's cash; do not invent holdings."
}
