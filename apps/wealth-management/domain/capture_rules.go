package domain

import (
	"strings"
)

type CaptureRuleKind string

const (
	// CaptureRuleAlias maps an account hint (Pattern) to an account name (Value).
	CaptureRuleAlias CaptureRuleKind = "alias"
	// CaptureRulePayee maps a payee (Pattern) to a category (Value).
	CaptureRulePayee CaptureRuleKind = "payee"
	// CaptureRuleCounterparty maps a counterparty as a material shows it
	// (Pattern, matched by CounterpartyKey) to the payee the user writes
	// (Value), learned from written rows (WM-CAP-23).
	CaptureRuleCounterparty CaptureRuleKind = "counterparty"
)

// CaptureRule is a correction confirmed during review (WM-CAP-11).
type CaptureRule struct {
	Kind      CaptureRuleKind `json:"kind"`
	Pattern   string          `json:"pattern"`
	Value     string          `json:"value"`
	UpdatedAt string          `json:"updatedAt,omitempty"`
}

// Key identifies a rule for upserts: kind plus folded pattern, or the
// counterparty key for counterparty rules.
func (r CaptureRule) Key() string {
	return string(r.Kind) + ":" + r.matchText()
}

func (r CaptureRule) matchText() string {
	if r.Kind == CaptureRuleCounterparty {
		return CounterpartyKey(r.Pattern)
	}
	return FoldText(r.Pattern)
}

// CaptureRulesSaveResult counts what a rules upsert changed.
type CaptureRulesSaveResult struct {
	Created   int `json:"created"`
	Updated   int `json:"updated"`
	Unchanged int `json:"unchanged"`
}

func (r CaptureRule) Validate() error {
	var problems []string
	if r.Kind != CaptureRuleAlias && r.Kind != CaptureRulePayee && r.Kind != CaptureRuleCounterparty {
		problems = append(problems, "kind must be alias, payee or counterparty")
	}
	if r.matchText() == "" {
		problems = append(problems, "pattern is required")
	}
	if strings.TrimSpace(r.Value) == "" {
		problems = append(problems, "value is required")
	}
	return validationError(problems)
}
