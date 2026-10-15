package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Programmes, banks, cards and merchant MCCs are data kept in the sheet
// (WM-CARD-3, N8); these checks hold whatever the source.

// VersionOn returns the version in effect on date: the latest starting on or
// before it.
func (p Program) VersionOn(date time.Time) (ProgramVersion, bool) {
	for i := len(p.Versions) - 1; i >= 0; i-- {
		if !p.Versions[i].From.After(date) {
			return p.Versions[i], true
		}
	}
	return ProgramVersion{}, false
}

// Program returns the programme named name.
func (c CardConfig) Program(name string) (Program, bool) {
	for _, program := range c.Programs {
		if strings.EqualFold(program.Name, name) {
			return program, true
		}
	}
	return Program{}, false
}

// Bank returns the bank named name.
func (c CardConfig) Bank(name string) (Bank, bool) {
	for _, bank := range c.Banks {
		if strings.EqualFold(bank.Name, name) {
			return bank, true
		}
	}
	return Bank{}, false
}

// BankOf returns the bank whose programme a card follows.
func (c CardConfig) BankOf(card Card) string {
	program, _ := c.Program(card.Program)
	return program.Bank
}

// Problems lists what makes a programme version unusable; its rules are
// checked separately.
func (v ProgramVersion) Problems() []string {
	var problems []string
	if v.From.IsZero() {
		problems = append(problems, "version from date is required")
	}
	if v.Period != CapPeriodCycle && v.Period != CapPeriodMonth {
		problems = append(problems, fmt.Sprintf("cap period must be %q or %q, got %q", CapPeriodCycle, CapPeriodMonth, v.Period))
	}
	if v.AnnualFee < 0 || v.Cap < 0 {
		problems = append(problems, "annual fee and card cap must not be negative")
	}
	return problems
}

// Problems lists what makes a cashback rule unusable.
func (r CashbackRule) Problems() []string {
	var problems []string
	if r.Name == "" {
		problems = append(problems, "rule name is required")
	}
	if !r.Fallback && len(r.MCCs) == 0 {
		problems = append(problems, "MCCs are required, unless the rule is the fallback")
	}
	if r.Fallback && len(r.MCCs) > 0 {
		problems = append(problems, "a fallback rule takes no MCCs: it applies to spend no other rule matches")
	}
	if r.Fallback && r.Exclude {
		problems = append(problems, "a rule cannot be both fallback and exclude")
	}
	if r.Exclude && (r.Rate != 0 || r.Cap != 0 || r.MinTransaction != 0 || r.MinCycleSpend != 0 || r.CapGroup != "") {
		problems = append(problems, "an exclude rule earns nothing: leave rate, cap, minimums and cap group empty")
	}
	if r.CapGroup != "" && !r.Exclude && r.Cap == 0 {
		problems = append(problems, "a cap group needs a rule cap: the cap its rules share")
	}
	for _, field := range []struct {
		label string
		value float64
	}{{"rate", r.Rate}, {"cap", r.Cap}, {"min transaction", r.MinTransaction}, {"min cycle spend", r.MinCycleSpend}} {
		if field.value < 0 {
			problems = append(problems, field.label+" must not be negative")
		}
	}
	return problems
}

// Problems lists what makes a bank unusable.
func (b Bank) Problems() []string {
	var problems []string
	if b.Name == "" {
		problems = append(problems, "bank is required")
	}
	if b.ReferenceLimit < 0 {
		problems = append(problems, "reference limit must not be negative")
	}
	return problems
}

// Problems lists what makes a held card unusable.
var last4Pattern = regexp.MustCompile(`^[0-9]{4}$`)

func (c Card) Problems() []string {
	var problems []string
	if c.Name == "" {
		problems = append(problems, "card name is required")
	}
	if c.Program == "" {
		problems = append(problems, "programme is required")
	}
	if c.Keyword == "" {
		problems = append(problems, "keyword is required")
	}
	if !validDayOfMonth(c.StatementDay) {
		problems = append(problems, fmt.Sprintf("statement day must be 1-31, got %d", c.StatementDay))
	}
	if !validDayOfMonth(c.DueDay) {
		problems = append(problems, fmt.Sprintf("due day must be 1-31, got %d", c.DueDay))
	}
	if c.Last4 != "" && !last4Pattern.MatchString(c.Last4) {
		problems = append(problems, fmt.Sprintf("last 4 digits must be 4 digits, got %q", c.Last4))
	}
	return problems
}

// Describe states a rule in words for AI prompts, e.g. "Groceries: 5% on
// MCC 5411, payments from 100000, cycle spend from 5000000, cap 300000".
func (r CashbackRule) Describe() string {
	target := "other spend"
	if !r.Fallback {
		target = "MCC " + FormatMCCList(r.MCCs)
	}
	if r.Exclude {
		return fmt.Sprintf("%s: no cashback on %s", r.Name, target)
	}
	parts := []string{fmt.Sprintf("%s: %.2f%% on %s", r.Name, r.Rate, target)}
	if r.MinTransaction > 0 {
		parts = append(parts, fmt.Sprintf("payments from %.0f", r.MinTransaction))
	}
	if r.MinCycleSpend > 0 {
		parts = append(parts, fmt.Sprintf("period spend from %.0f", r.MinCycleSpend))
	}
	switch {
	case r.CapGroup != "":
		parts = append(parts, fmt.Sprintf("cap %.0f shared by cap group %q", r.Cap, r.CapGroup))
	case r.Cap > 0:
		parts = append(parts, fmt.Sprintf("cap %.0f", r.Cap))
	}
	return strings.Join(parts, ", ")
}
