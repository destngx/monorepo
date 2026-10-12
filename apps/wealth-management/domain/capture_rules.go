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
)

// CaptureRule is a correction confirmed during review (WM-CAP-11).
type CaptureRule struct {
	Kind      CaptureRuleKind `json:"kind"`
	Pattern   string          `json:"pattern"`
	Value     string          `json:"value"`
	UpdatedAt string          `json:"updatedAt,omitempty"`
}

// Key identifies a rule for upserts: kind plus folded pattern.
func (r CaptureRule) Key() string {
	return string(r.Kind) + ":" + FoldText(r.Pattern)
}

// CaptureRulesSaveResult counts what a rules upsert changed.
type CaptureRulesSaveResult struct {
	Created   int `json:"created"`
	Updated   int `json:"updated"`
	Unchanged int `json:"unchanged"`
}

func (r CaptureRule) Validate() error {
	var problems []string
	if r.Kind != CaptureRuleAlias && r.Kind != CaptureRulePayee {
		problems = append(problems, "kind must be alias or payee")
	}
	if FoldText(r.Pattern) == "" {
		problems = append(problems, "pattern is required")
	}
	if strings.TrimSpace(r.Value) == "" {
		problems = append(problems, "value is required")
	}
	return validationError(problems)
}
