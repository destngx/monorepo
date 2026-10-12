package domain

import (
	"errors"
	"testing"
)

func TestGivenCountsWhenBiasOfThenNeedsStrictPlurality(t *testing.T) {
	cases := []struct {
		counts NewsCounts
		want   NewsBias
	}{
		{NewsCounts{Positive: 3, Negative: 1, Neutral: 1}, NewsBullish},
		{NewsCounts{Positive: 1, Negative: 2, Neutral: 1}, NewsBearish},
		{NewsCounts{Positive: 2, Negative: 2, Neutral: 1}, NewsMixed},
		{NewsCounts{Positive: 2, Negative: 0, Neutral: 2}, NewsMixed},
		{NewsCounts{}, NewsMixed},
	}
	for _, tc := range cases {
		if got := BiasOf(tc.counts); got != tc.want {
			t.Errorf("BiasOf(%+v) = %s, want %s", tc.counts, got, tc.want)
		}
	}
}

func TestGivenVerdictsWhenClassifyNewsThenJoinsByIndexAndValidates(t *testing.T) {
	results := []SearchResult{{Title: "a", URL: "https://a.vn"}, {Title: "b", URL: "https://b.vn"}}
	articles, counts, err := ClassifyNews(results, []NewsClassification{
		{Index: 1, Sentiment: "negative", Confidence: 80.4, Impact: 60, Summary: " Rates up. "},
		{Index: 0, Sentiment: NewsPositive, Confidence: 90, Impact: 70},
	})
	if err != nil {
		t.Fatal(err)
	}
	if articles[1].Title != "b" || articles[1].Sentiment != NewsNegative || articles[1].Confidence != 80 || articles[1].Summary != "Rates up." {
		t.Fatalf("unexpected article %#v", articles[1])
	}
	if counts != (NewsCounts{Positive: 1, Negative: 1}) {
		t.Fatalf("unexpected counts %+v", counts)
	}
	for name, verdicts := range map[string][]NewsClassification{
		"missing":   {{Index: 0, Sentiment: NewsNeutral}},
		"repeated":  {{Index: 0, Sentiment: NewsNeutral}, {Index: 0, Sentiment: NewsNeutral}},
		"unknown":   {{Index: 0, Sentiment: "Bullish"}, {Index: 1, Sentiment: NewsNeutral}},
		"range":     {{Index: 0, Sentiment: NewsNeutral, Impact: 101}, {Index: 1, Sentiment: NewsNeutral}},
		"out index": {{Index: 2, Sentiment: NewsNeutral}},
	} {
		if _, _, err := ClassifyNews(results, verdicts); !errors.Is(err, ErrAIOutput) {
			t.Errorf("%s: expected ErrAIOutput, got %v", name, err)
		}
	}
}

func TestGivenLegacyFieldsWhenNormalizeCommandsThenFoldsAliases(t *testing.T) {
	commands, err := InvestmentCommands{Commands: []rawInvestmentCommand{
		{Action: "allocate", ExactTarget: "VESAF", ExecutionLogic: "DCA monthly"},
		{Action: "HEDGE", Target: "USDT", MacroHedgeReason: "VND weakness"},
	}}.Normalize()
	if err != nil {
		t.Fatal(err)
	}
	if commands[0] != (InvestmentCommand{Action: ActionAllocate, Target: "VESAF", Rationale: "DCA monthly"}) || commands[1].Rationale != "VND weakness" {
		t.Fatalf("unexpected commands %#v", commands)
	}
	if _, err := (InvestmentCommands{Commands: []rawInvestmentCommand{{Action: "BUY", Target: "FPT"}}}).Normalize(); !errors.Is(err, ErrAIOutput) {
		t.Fatalf("an unknown action must be rejected, got %v", err)
	}
	if _, err := (InvestmentCommands{Commands: []rawInvestmentCommand{{Action: "OBSERVE"}}}).Normalize(); !errors.Is(err, ErrAIOutput) {
		t.Fatalf("a command without a target must be rejected, got %v", err)
	}
}
