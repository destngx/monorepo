package domain

import (
	"strings"
	"testing"
)

func expectOneProblem(t *testing.T, problems []string, want string) {
	t.Helper()
	if len(problems) != 1 || !strings.Contains(problems[0], want) {
		t.Fatalf("problems %v should be one mentioning %q", problems, want)
	}
}

func TestProgramVersionOn(t *testing.T) {
	program := Program{Versions: []ProgramVersion{{From: day(2026, 1, 1), Cap: 1}, {From: day(2026, 6, 1), Cap: 2}}}
	if _, found := program.VersionOn(day(2025, 12, 31)); found {
		t.Fatal("no version before the first one starts")
	}
	if version, _ := program.VersionOn(day(2026, 5, 31)); version.Cap != 1 {
		t.Fatalf("got %+v", version)
	}
	if version, _ := program.VersionOn(day(2026, 6, 1)); version.Cap != 2 {
		t.Fatalf("a version applies from its date, got %+v", version)
	}
}

func TestProgramVersionProblems(t *testing.T) {
	valid := ProgramVersion{From: day(2026, 1, 1), Period: CapPeriodCycle}
	if problems := valid.Problems(); len(problems) != 0 {
		t.Fatalf("valid version: %v", problems)
	}
	tests := map[string]struct {
		edit func(*ProgramVersion)
		want string
	}{
		"no from":      {func(v *ProgramVersion) { v.From = day(1, 1, 1) }, "from date"},
		"no period":    {func(v *ProgramVersion) { v.Period = "" }, "cap period"},
		"weekly":       {func(v *ProgramVersion) { v.Period = "week" }, "cap period"},
		"negative fee": {func(v *ProgramVersion) { v.AnnualFee = -1 }, "negative"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			version := valid
			test.edit(&version)
			expectOneProblem(t, version.Problems(), test.want)
		})
	}
}

func TestCardProblems(t *testing.T) {
	valid := Card{Name: "Gold", Program: "Gold", Keyword: "gold", StatementDay: 5, DueDay: 30}
	if problems := valid.Problems(); len(problems) != 0 {
		t.Fatalf("valid card: %v", problems)
	}
	tests := map[string]struct {
		edit func(*Card)
		want string
	}{
		"no name":     {func(c *Card) { c.Name = "" }, "card name"},
		"no program":  {func(c *Card) { c.Program = "" }, "programme"},
		"no keyword":  {func(c *Card) { c.Keyword = "" }, "keyword"},
		"statement 0": {func(c *Card) { c.StatementDay = 0 }, "statement day"},
		"due 32":      {func(c *Card) { c.DueDay = 32 }, "due day"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			card := valid
			test.edit(&card)
			expectOneProblem(t, card.Problems(), test.want)
		})
	}
}

func TestBankProblems(t *testing.T) {
	expectOneProblem(t, Bank{ReferenceLimit: 1}.Problems(), "bank")
	expectOneProblem(t, Bank{Name: "Acme", ReferenceLimit: -1}.Problems(), "negative")
}

func TestCashbackRuleProblems(t *testing.T) {
	groceries := []MCCRange{{From: 5411, To: 5411}}
	valid := []CashbackRule{
		{Name: "Groceries", MCCs: groceries, Rate: 5, MinTransaction: 100_000, MinCycleSpend: 5_000_000},
		{Name: "Others", Rate: 0.5, Fallback: true},
		{Name: "Utilities", MCCs: []MCCRange{{From: 4900, To: 4900}}, Exclude: true},
	}
	for _, rule := range valid {
		if problems := rule.Problems(); len(problems) != 0 {
			t.Fatalf("valid rule %q: %v", rule.Name, problems)
		}
	}
	tests := map[string]struct {
		rule CashbackRule
		want []string
	}{
		"nothing to match":     {CashbackRule{Name: "Online", Rate: 5}, []string{"MCCs are required"}},
		"fallback with MCCs":   {CashbackRule{Name: "x", MCCs: groceries, Fallback: true}, []string{"takes no MCCs"}},
		"negative numbers":     {CashbackRule{Name: "x", MCCs: groceries, Rate: -1, Cap: -1, MinTransaction: -1, MinCycleSpend: -1}, []string{"rate", "cap", "min transaction", "min cycle spend"}},
		"no name":              {CashbackRule{MCCs: groceries}, []string{"rule name"}},
		"fallback and exclude": {CashbackRule{Name: "x", Fallback: true, Exclude: true}, []string{"both"}},
		"exclude with a rate":  {CashbackRule{Name: "x", MCCs: groceries, Rate: 1, Exclude: true}, []string{"earns nothing"}},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			problems := strings.Join(test.rule.Problems(), "; ")
			for _, want := range test.want {
				if !strings.Contains(problems, want) {
					t.Fatalf("problems %q should mention %q", problems, want)
				}
			}
		})
	}
}

func TestCashbackRuleDescribe(t *testing.T) {
	tests := map[string]CashbackRule{
		`Groceries: 5.00% on MCC 5411, 5812-5814, payments from 100000, period spend from 5000000, cap 300000`: {
			Name: "Groceries", MCCs: []MCCRange{{From: 5411, To: 5411}, {From: 5812, To: 5814}},
			Rate: 5, MinTransaction: 100_000, MinCycleSpend: 5_000_000, Cap: 300_000,
		},
		"Others: 0.50% on other spend":       {Name: "Others", Rate: 0.5, Fallback: true},
		"Utilities: no cashback on MCC 4900": {Name: "Utilities", MCCs: []MCCRange{{From: 4900, To: 4900}}, Exclude: true},
	}
	for want, rule := range tests {
		if got := rule.Describe(); got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	}
}
