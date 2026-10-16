package domain

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Engine rules (WM-REC-3): deterministic recommendations, no AI. Each rule
// has a stable id; it adds one item while its condition holds and closes
// that item (done, outcome "resolved") once the condition is gone.

const (
	RuleRebalance     = "rebalance"
	RuleGoalPlanShort = "goal-plan-short"
	RuleLentOverdue   = "lent-overdue"
	RuleIdleCash      = "idle-cash"

	// LentOverdueDays: money lent open longer than this is worth chasing.
	LentOverdueDays = 90

	// ruleResolvedOutcome is written when a rule's condition no longer holds.
	ruleResolvedOutcome = "resolved"
)

// RecommendationRuleSource is the Source column of a rule's items.
func RecommendationRuleSource(rule string) string {
	return RecommendationRuleSourcePrefix + rule
}

// RuleFinding is a rule whose condition holds now.
type RuleFinding struct {
	Rule      string
	Title     string
	Action    string
	ImpactVND *float64
	Evidence  string
	Page      string
}

// RuleInputs are the figures the rules read; a nil input means it could not
// be loaded, and its rule is not evaluated (its items stay as they are).
type RuleInputs struct {
	Allocation *Allocation
	GoalPlan   *GoalPlan
	MoneyLent  *MoneyLent
	IdleCash   *IdleCash
}

// EvaluateRecommendationRules returns the rules that were evaluated and the
// findings among them.
func EvaluateRecommendationRules(in RuleInputs) ([]string, []RuleFinding) {
	evaluated := []string{}
	findings := []RuleFinding{}
	if in.Allocation != nil {
		evaluated = append(evaluated, RuleRebalance)
		if finding, ok := rebalanceFinding(*in.Allocation); ok {
			findings = append(findings, finding)
		}
	}
	if in.GoalPlan != nil && !in.GoalPlan.SetupNeeded {
		evaluated = append(evaluated, RuleGoalPlanShort)
		if finding, ok := goalPlanShortFinding(*in.GoalPlan); ok {
			findings = append(findings, finding)
		}
	}
	if in.MoneyLent != nil {
		evaluated = append(evaluated, RuleLentOverdue)
		if finding, ok := lentOverdueFinding(*in.MoneyLent); ok {
			findings = append(findings, finding)
		}
	}
	// Without a published rate the idle cash check cannot judge, so its
	// items stay as they are.
	if in.IdleCash != nil && in.IdleCash.Verdict != IdleCashNoRate {
		evaluated = append(evaluated, RuleIdleCash)
		if finding, ok := idleCashFinding(*in.IdleCash); ok {
			findings = append(findings, finding)
		}
	}
	return evaluated, findings
}

// idleCashFinding: idle cash would earn materially more at the best
// published rate (ROADMAP R6, WM-CASH-1..5). About the rate only (D31).
func idleCashFinding(idle IdleCash) (RuleFinding, bool) {
	if idle.Verdict != IdleCashWorthMoving || idle.Best == nil {
		return RuleFinding{}, false
	}
	extra := math.Round(idle.ExtraPerYear)
	return RuleFinding{
		Rule:  RuleIdleCash,
		Title: "Move idle cash to a higher savings rate",
		Action: fmt.Sprintf("Put about %s of idle cash in a savings deposit at %s (%.1f%%, %s); it would earn about %s more a year.",
			FormatVND(idle.Idle), idle.Best.Bank, idle.Best.Rate, idle.Best.Tenor, FormatVND(extra)),
		ImpactVND: &extra,
		Evidence: fmt.Sprintf("%s idle beyond what stays liquid; earning %.1f%% now vs %.1f%% best published",
			FormatVND(idle.Idle), idle.CurrentRate, idle.Best.Rate),
		Page: "/bills",
	}, true
}

// rebalanceFinding: a target class drifts beyond the threshold (WM-INV-20).
func rebalanceFinding(allocation Allocation) (RuleFinding, bool) {
	if !allocation.Rebalance || allocation.TargetedTotal <= 0 {
		return RuleFinding{}, false
	}
	var widest *AllocationLine
	toMove := 0.0
	parts := []string{}
	for i, line := range allocation.Lines {
		if !line.Targeted || line.Drift == nil || line.Current == nil || line.Target == nil {
			continue
		}
		if *line.Drift > 0 {
			toMove += *line.Drift / 100 * allocation.TargetedTotal
		}
		parts = append(parts, fmt.Sprintf("%s %.0f%% vs target %.0f%%", line.Label, *line.Current, *line.Target))
		if widest == nil || math.Abs(*line.Drift) > math.Abs(*widest.Drift) {
			widest = &allocation.Lines[i]
		}
	}
	if widest == nil {
		return RuleFinding{}, false
	}
	direction := "over"
	if *widest.Drift < 0 {
		direction = "under"
	}
	return RuleFinding{
		Rule:     RuleRebalance,
		Title:    fmt.Sprintf("Rebalance: %s is %.0f points %s target", widest.Label, math.Abs(*widest.Drift), direction),
		Action:   fmt.Sprintf("Move about %s from the classes over target to those under it, or change the targets on Investments.", FormatVND(toMove)),
		Evidence: fmt.Sprintf("%s; rebalance beyond %.0f points", strings.Join(parts, ", "), allocation.DriftThreshold),
		Page:     "/investments",
	}, true
}

// goalPlanShortFinding: dated goals need more a month than is saved (D31).
func goalPlanShortFinding(plan GoalPlan) (RuleFinding, bool) {
	check := plan.Check
	if check.Verdict != PlanShort || check.Gap <= 0 {
		return RuleFinding{}, false
	}
	action := fmt.Sprintf("Save %s more a month", FormatVND(check.Gap))
	for _, tradeOff := range check.TradeOffs {
		switch {
		case tradeOff.MoveTargetDateTo != nil:
			action += fmt.Sprintf(", or move %s to %s", tradeOff.Name, *tradeOff.MoveTargetDateTo)
		case tradeOff.LowerTargetBy != nil:
			action += fmt.Sprintf(", or lower %s by %s", tradeOff.Name, FormatVND(*tradeOff.LowerTargetBy))
		default:
			continue
		}
		break
	}
	yearly := math.Round(check.Gap * 12)
	return RuleFinding{
		Rule:      RuleGoalPlanShort,
		Title:     fmt.Sprintf("Goal plan is short by %s a month", FormatVND(check.Gap)),
		Action:    action + ".",
		ImpactVND: &yearly,
		Evidence:  fmt.Sprintf("dated goals need %s a month; average savings %s a month", FormatVND(check.RequiredPerMonth), FormatVND(check.MonthlySavings)),
		Page:      "/accounts/goals",
	}, true
}

// lentOverdueFinding: money owed to the user open longer than 90 days (D30).
func lentOverdueFinding(lent MoneyLent) (RuleFinding, bool) {
	people, total, oldest := 0, 0.0, 0
	for _, person := range lent.People {
		if person.Outstanding <= 0 || person.DaysOutstanding == nil || *person.DaysOutstanding <= LentOverdueDays {
			continue
		}
		people++
		total += person.Outstanding
		oldest = max(oldest, *person.DaysOutstanding)
	}
	if people == 0 {
		return RuleFinding{}, false
	}
	who := "1 person owes"
	if people > 1 {
		who = fmt.Sprintf("%d people owe", people)
	}
	return RuleFinding{
		Rule:     RuleLentOverdue,
		Title:    fmt.Sprintf("Chase money lent: %s %s for over %d days", who, FormatVND(total), LentOverdueDays),
		Action:   "Ask for repayment or agree a date, and record repayments as deposits in the lending category.",
		Evidence: fmt.Sprintf("%s open longer than %d days; oldest %d days", FormatVND(total), LentOverdueDays, oldest),
		Page:     "/accounts/lent",
	}, true
}

// PlanRuleRecommendations adds an item for each finding whose rule has no
// pending item, no item dismissed in the last 90 days and no item the user
// marked done in the last 30, and closes the pending items of evaluated
// rules without a finding.
func PlanRuleRecommendations(sheet RecommendationSheet, evaluated []string, findings []RuleFinding, today time.Time) RecommendationChanges {
	date := today.Format(time.DateOnly)
	dismissedSince := today.AddDate(0, 0, -RecommendationDismissQuietDays).Format(time.DateOnly)
	doneSince := today.AddDate(0, 0, -RecommendationDoneQuietDays).Format(time.DateOnly)
	found := map[string]RuleFinding{}
	for _, finding := range findings {
		found[finding.Rule] = finding
	}
	changes := RecommendationChanges{}
	taken := takenIDs(sheet)
	for _, rule := range evaluated {
		source := RecommendationRuleSource(rule)
		finding, holds := found[rule]
		quiet := false
		for _, item := range sheet.Items {
			if item.Source != source {
				continue
			}
			switch {
			case item.Pending() && holds:
				quiet = true
			case item.Pending():
				changes.Update = append(changes.Update, RecommendationUpdate{
					ID: item.ID, Row: item.Row, Status: RecommendationDone, StatusChanged: date, Outcome: ruleResolvedOutcome,
				})
			case item.Status == RecommendationDismissed && item.StatusChanged >= dismissedSince:
				quiet = true
			case item.Status == RecommendationDone && item.Outcome != ruleResolvedOutcome && item.StatusChanged >= doneSince:
				quiet = true
			}
		}
		if holds && !quiet {
			changes.Add = append(changes.Add, Recommendation{
				ID: newRecommendationID(source, finding.Title, date, taken), Created: date, Source: source,
				Title: clip(finding.Title, recommendationTitleMax), Action: clip(finding.Action, recommendationActionMax),
				ImpactVND: finding.ImpactVND, EvidencePath: finding.Page, Evidence: clip(finding.Evidence, recommendationEvidenceMax),
				Status: RecommendationOpen, Page: finding.Page,
			})
		}
	}
	return changes
}
