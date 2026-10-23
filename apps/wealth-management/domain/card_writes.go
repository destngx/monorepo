package domain

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Card configuration writes (WM-FILL-12..14). The engine also reads these
// tabs strictly (WM-CARD-3), so a write is checked by reading the tab as it
// would be after the write; these inputs only check their own shape.

// CardInput is a Cards row; Limit, when set, is the shared account's credit
// limit (Accounts C).
type CardInput struct {
	Name         string   `json:"name"`
	Program      string   `json:"program"`
	Keyword      string   `json:"keyword"`
	Expiry       string   `json:"expiry,omitempty"`
	StatementDay int      `json:"statementDay"`
	DueDay       int      `json:"dueDay"`
	Default      bool     `json:"default,omitempty"`
	ClosedFrom   string   `json:"closedFrom,omitempty"`
	Last4        string   `json:"last4,omitempty"`
	Limit        *float64 `json:"limit,omitempty"`
}

func (input CardInput) Normalized() CardInput {
	input.Name = strings.TrimSpace(input.Name)
	input.Program = strings.TrimSpace(input.Program)
	input.Keyword = strings.TrimSpace(input.Keyword)
	input.Expiry = strings.TrimSpace(input.Expiry)
	input.ClosedFrom = strings.TrimSpace(input.ClosedFrom)
	input.Last4 = strings.TrimSpace(input.Last4)
	return input
}

func (input CardInput) Validate() error {
	switch {
	case input.Name == "":
		return fmt.Errorf("%w: card name is required", ErrValidation)
	case input.Program == "":
		return fmt.Errorf("%w: programme is required", ErrValidation)
	case input.StatementDay < 0 || input.StatementDay > 31 || input.DueDay < 0 || input.DueDay > 31:
		return fmt.Errorf("%w: statement and due days are days of the month, 1-31", ErrValidation)
	case input.Limit != nil && (*input.Limit < 0 || math.IsNaN(*input.Limit) || math.IsInf(*input.Limit, 0)):
		return fmt.Errorf("%w: credit limit must be 0 or more", ErrValidation)
	}
	if input.ClosedFrom != "" {
		if _, err := time.Parse(time.DateOnly, input.ClosedFrom); err != nil {
			return fmt.Errorf("%w: closed from %q must be yyyy-mm-dd", ErrValidation, input.ClosedFrom)
		}
	}
	return nil
}

// SheetRow is the Cards row A-I as written (USER_ENTERED).
func (input CardInput) SheetRow() []any {
	flag := ""
	if input.Default {
		flag = "x"
	}
	return []any{
		input.Name, input.Program, input.Keyword, textCell(input.Expiry), dayCell(input.StatementDay), dayCell(input.DueDay),
		flag, input.ClosedFrom, textCell(input.Last4),
	}
}

// textCell keeps a value as written: Sheets would read 08/2029 as a date
// and 0123 as 123 (USER_ENTERED).
func textCell(value string) string {
	if value == "" {
		return ""
	}
	return "'" + value
}

func dayCell(day int) any {
	if day == 0 {
		return ""
	}
	return day
}

// ProgramRuleInput is a ProgramRules row of a programme version.
type ProgramRuleInput struct {
	Rule           string  `json:"rule"`
	MCCs           string  `json:"mccs,omitempty"`
	Rate           float64 `json:"rate"`
	Cap            float64 `json:"cap,omitempty"`
	MinTransaction float64 `json:"minTransaction,omitempty"`
	MinCycleSpend  float64 `json:"minCycleSpend,omitempty"`
	Fallback       bool    `json:"fallback,omitempty"`
	Exclude        bool    `json:"exclude,omitempty"`
	CapGroup       string  `json:"capGroup,omitempty"`
}

// ProgramInput is a programme version (a Programs row) and its rules.
// SharedAccount, for a bank with no Banks row yet, adds that row: the
// account the bank's cards bill to.
type ProgramInput struct {
	Program       string             `json:"program"`
	Bank          string             `json:"bank"`
	SharedAccount string             `json:"sharedAccount,omitempty"`
	From          string             `json:"from"`
	AnnualFee     float64            `json:"annualFee,omitempty"`
	Cap           float64            `json:"cap,omitempty"`
	CapPeriod     string             `json:"capPeriod"`
	Rules         []ProgramRuleInput `json:"rules"`
}

const maxProgramRules = 50

func (input ProgramInput) Normalized() ProgramInput {
	input.Program = strings.TrimSpace(input.Program)
	input.Bank = strings.TrimSpace(input.Bank)
	input.SharedAccount = strings.TrimSpace(input.SharedAccount)
	input.From = strings.TrimSpace(input.From)
	input.CapPeriod = strings.ToLower(strings.TrimSpace(input.CapPeriod))
	rules := make([]ProgramRuleInput, len(input.Rules))
	for index, rule := range input.Rules {
		rule.Rule = strings.TrimSpace(rule.Rule)
		rule.MCCs = strings.TrimSpace(rule.MCCs)
		rule.CapGroup = strings.TrimSpace(rule.CapGroup)
		rules[index] = rule
	}
	input.Rules = rules
	return input
}

func (input ProgramInput) Validate() (time.Time, error) {
	switch {
	case input.Program == "":
		return time.Time{}, fmt.Errorf("%w: programme is required", ErrValidation)
	case input.Bank == "":
		return time.Time{}, fmt.Errorf("%w: bank is required", ErrValidation)
	case len(input.Rules) == 0:
		return time.Time{}, fmt.Errorf("%w: at least one rule is required", ErrValidation)
	case len(input.Rules) > maxProgramRules:
		return time.Time{}, fmt.Errorf("%w: at most %d rules", ErrValidation, maxProgramRules)
	}
	from, err := time.Parse(time.DateOnly, input.From)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: from %q must be yyyy-mm-dd", ErrValidation, input.From)
	}
	for index, rule := range input.Rules {
		if rule.Rule == "" {
			return time.Time{}, fmt.Errorf("%w: rule %d needs a name", ErrValidation, index+1)
		}
	}
	return from, nil
}

// ProgramRow is the Programs row A-F.
func (input ProgramInput) ProgramRow() []any {
	return []any{input.Program, input.Bank, input.From, input.AnnualFee, input.Cap, input.CapPeriod}
}

// RuleRows are the ProgramRules rows A-K.
func (input ProgramInput) RuleRows() [][]any {
	rows := make([][]any, len(input.Rules))
	for index, rule := range input.Rules {
		rows[index] = []any{
			input.Program, input.From, rule.Rule, textCell(rule.MCCs), rule.Rate, optionalAmount(rule.Cap),
			optionalAmount(rule.MinTransaction), optionalAmount(rule.MinCycleSpend), flagCell(rule.Fallback), flagCell(rule.Exclude), rule.CapGroup,
		}
	}
	return rows
}

func optionalAmount(value float64) any {
	if value == 0 {
		return ""
	}
	return value
}

func flagCell(set bool) string {
	if set {
		return "x"
	}
	return ""
}

// MerchantInput is a Merchants row: the MCC a payee pattern is billed with.
type MerchantInput struct {
	Pattern    string `json:"pattern"`
	MCC        int    `json:"mcc"`
	From       string `json:"from,omitempty"`
	Confidence string `json:"confidence,omitempty"`
	Note       string `json:"note,omitempty"`
}

func (input MerchantInput) Normalized() MerchantInput {
	input.Pattern = strings.TrimSpace(input.Pattern)
	input.From = strings.TrimSpace(input.From)
	input.Confidence = strings.ToLower(strings.TrimSpace(input.Confidence))
	if input.Confidence == "" {
		input.Confidence = string(MCCAssumed)
	}
	input.Note = strings.TrimSpace(input.Note)
	return input
}

func (input MerchantInput) Validate() error {
	switch {
	case FoldText(input.Pattern) == "":
		return fmt.Errorf("%w: payee pattern is required", ErrValidation)
	case input.MCC < 1 || input.MCC > 9999:
		return fmt.Errorf("%w: MCC must be a 4-digit code", ErrValidation)
	case input.Confidence != string(MCCAssumed) && input.Confidence != string(MCCConfirmed):
		return fmt.Errorf("%w: confidence must be %q or %q", ErrValidation, MCCAssumed, MCCConfirmed)
	case len([]rune(input.Note)) > 200:
		return fmt.Errorf("%w: source note is longer than 200 characters", ErrValidation)
	}
	if input.From != "" {
		if _, err := time.Parse(time.DateOnly, input.From); err != nil {
			return fmt.Errorf("%w: from %q must be yyyy-mm-dd", ErrValidation, input.From)
		}
	}
	return nil
}

// SheetRow is the Merchants row A-E; the MCC is text so 0742 keeps its zero.
func (input MerchantInput) SheetRow() []any {
	return []any{input.Pattern, textCell(fmt.Sprintf("%04d", input.MCC)), input.From, input.Confidence, input.Note}
}
