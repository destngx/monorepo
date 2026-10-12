package domain

import (
	"errors"
	"testing"
)

func TestRenderPromptSubstitutesKnownPlaceholders(t *testing.T) {
	got := RenderPrompt("Net {{ netWorth }} for {{payee}} and {{unknown}} $1", map[string]string{"netWorth": "10", "payee": "$1 Shop"})
	want := "Net 10 for $1 Shop and {{unknown}} $1"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestParseAIJSON(t *testing.T) {
	var value struct {
		Summary string `json:"summary"`
	}
	if err := ParseAIJSON("```json\n{\"summary\":\"ok\"}\n```", &value); err != nil || value.Summary != "ok" {
		t.Fatalf("fenced JSON: %v %q", err, value.Summary)
	}
	if err := ParseAIJSON(`Here you go: {"summary":"x"}`, &value); !errors.Is(err, ErrAIOutput) {
		t.Fatalf("prose before JSON must fail, got %v", err)
	}
	if err := ParseAIJSON(`{"summary":"x"} trailing`, &value); !errors.Is(err, ErrAIOutput) {
		t.Fatalf("text after JSON must fail, got %v", err)
	}
}

func TestPIIMaskerMasksIdentifiersButNotSmallAmounts(t *testing.T) {
	masker := NewPIIMasker([]string{"Nguyen Van A"})
	got := masker.Mask("CK 500000 tu 0912 345 678 STK 190312345678 nguyen van a mail a.b@x.vn")
	want := "CK 500000 tu ••••5678 STK ••••5678 [name] mail [email]"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestPIIMaskerMatchesNamesAtUnicodeBoundaries(t *testing.T) {
	masker := NewPIIMasker([]string{"Đức", "Ann"})
	got := masker.Mask("đức chuyển cho Ann, không phải Annabelle hay Đứcanh")
	want := "[name] chuyển cho [name], không phải Annabelle hay Đứcanh"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestResolveCategory(t *testing.T) {
	categories := []string{"Food & Dining", "Transport"}
	if got, ok := ResolveCategory(`"food & dining".`, categories); !ok || got != "Food & Dining" {
		t.Fatalf("case-insensitive match failed: %q %v", got, ok)
	}
	if _, ok := ResolveCategory("Groceries", categories); ok {
		t.Fatal("unknown category must not resolve")
	}
}

func TestInsightNormalize(t *testing.T) {
	insight := Insight{
		Summary: " Spending is on track ",
		Sections: []InsightSection{
			{Icon: "Rocket", Title: "Pace", Content: "Fine", Severity: "CRITICAL"},
			{Title: " ", Content: ""},
		},
		ActionItems: []string{"a", " ", "b", "c", "d", "e"},
	}
	if err := insight.Normalize(); err != nil {
		t.Fatal(err)
	}
	if len(insight.Sections) != 1 || insight.Sections[0].Icon != "tip" || insight.Sections[0].Severity != InsightCritical {
		t.Fatalf("unexpected sections %+v", insight.Sections)
	}
	if len(insight.ActionItems) != 4 {
		t.Fatalf("action items not capped: %v", insight.ActionItems)
	}
	empty := Insight{Summary: "x"}
	if err := empty.Normalize(); !errors.Is(err, ErrAIOutput) {
		t.Fatalf("no sections must fail, got %v", err)
	}
}
