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
	// RuleEmergencyFundRate: the emergency accounts earn materially less
	// than the best instant-access rate (D34).
	RuleEmergencyFundRate = "emergency-fund-rate"

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
	// AmountVND is the money the finding is about (to move, owed, short a
	// month, idle); the home inbox shows it, the sheet does not store it.
	AmountVND *float64
	Evidence  string
	Page      string
	// DueDate (yyyy-mm-dd) is when the finding falls due; the home inbox
	// shows it, the sheet does not store it.
	DueDate string
	// Urgent raises the home inbox item from warning to serious.
	Urgent bool
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
	// Without an instant-access rate the idle cash check cannot judge, so
	// its items stay as they are; an untenored or long-term rate is never
	// quoted for money kept within reach.
	if in.IdleCash != nil && in.IdleCash.Verdict != IdleCashNoRate && in.IdleCash.Best != nil {
		evaluated = append(evaluated, RuleIdleCash, RuleEmergencyFundRate)
		if finding, ok := idleCashFinding(*in.IdleCash); ok {
			findings = append(findings, finding)
		}
		if finding, ok := emergencyFundRateFinding(*in.IdleCash); ok {
			findings = append(findings, finding)
		}
	}
	return evaluated, findings
}

// idleCashFinding: the idle part above the day-to-day float would earn
// materially more in an instant-access deposit (ROADMAP R6, WM-CASH-1..5,
// D34). About the rate only (D31); the money stays within reach.
func idleCashFinding(idle IdleCash) (RuleFinding, bool) {
	if idle.Verdict != IdleCashWorthMoving || idle.Best == nil {
		return RuleFinding{}, false
	}
	extra := math.Round(idle.ExtraPerYear)
	keep := math.Max(0, idle.Components.ReadyMoney-idle.Idle)
	action := fmt.Sprintf("Keep %s ready in your everyday accounts. The other %s could earn about %s a year more in %s at %s (%s%%, %s)",
		FormatVND(keep), FormatVND(idle.Idle), FormatVND(extra), idle.Best.InstantAccessKind(), idle.Best.Bank, formatRate(idle.Best.Rate), idle.Best.InstantAccessTerms())
	if ladder := idle.Ladder; ladder != nil {
		action += fmt.Sprintf(", for example split into %d parts of about %s opened %d days apart, so one matures every %d days",
			ladder.Parts, FormatVND(ladder.PartAmount), ladder.SpacingDays, ladder.SpacingDays)
	}
	action += ". It stays reachable."
	return RuleFinding{
		Rule:      RuleIdleCash,
		Title:     "Earn more on idle cash and keep it within reach",
		Action:    action,
		ImpactVND: &extra,
		AmountVND: vndAmount(idle.Idle),
		Evidence: fmt.Sprintf("%s idle above the day-to-day float (%s) and what stays liquid; earning %s%% now vs %s%% best instant-access: %s",
			FormatVND(idle.Idle), FormatVND(idle.Components.DayToDayFloat), formatRate(idle.CurrentRate), formatRate(idle.Best.Rate), strings.TrimPrefix(idle.Best.InstantAccessKind(), "a ")),
		Page: "/bills",
	}, true
}

// emergencyFundRateFinding: the emergency accounts earn materially less
// than the best instant-access rate (D34). Its own wording: it is about
// where the emergency fund sits, never about moving all cash.
func emergencyFundRateFinding(idle IdleCash) (RuleFinding, bool) {
	e := idle.Emergency
	if !e.BetterRate || idle.Best == nil || e.Balance <= 0 {
		return RuleFinding{}, false
	}
	extra := math.Round(e.ExtraPerYear)
	kind := idle.Best.InstantAccessKind()
	holding := 0
	for _, account := range e.Accounts {
		if account.Balance > 0 {
			holding++
		}
	}
	return RuleFinding{
		Rule:  RuleEmergencyFundRate,
		Title: "Your emergency fund could earn more and stay within reach",
		Action: fmt.Sprintf("Your emergency fund (%s in %s) earns %s%%. %s at %s (%s%%, %s) would earn about %s a year more and keep it within reach. "+
			"Move it only if the new place is as quick to reach in an emergency.",
			FormatVND(e.Balance), plural(holding, "emergency account", "emergency accounts"), formatRate(e.CurrentRate),
			strings.ToUpper(kind[:1])+kind[1:], idle.Best.Bank, formatRate(idle.Best.Rate), idle.Best.InstantAccessTerms(), FormatVND(extra)),
		ImpactVND: &extra,
		AmountVND: vndAmount(e.Balance),
		Evidence: fmt.Sprintf("emergency accounts earn %s%% vs %s%% best instant-access: %s", formatRate(e.CurrentRate), formatRate(idle.Best.Rate),
			strings.TrimPrefix(kind, "a ")),
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
		Rule:      RuleRebalance,
		Title:     fmt.Sprintf("Rebalance: %s is %.0f points %s target", widest.Label, math.Abs(*widest.Drift), direction),
		Action:    fmt.Sprintf("Move about %s from the classes over target to those under it, or change the targets on Investments.", FormatVND(toMove)),
		AmountVND: vndAmount(toMove),
		Evidence:  fmt.Sprintf("%s; rebalance beyond %.0f points", strings.Join(parts, ", "), allocation.DriftThreshold),
		Page:      "/investments",
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
		AmountVND: vndAmount(check.Gap),
		Evidence:  fmt.Sprintf("dated goals need %s a month; average savings %s a month", FormatVND(check.RequiredPerMonth), FormatVND(check.MonthlySavings)),
		Page:      "/accounts/goals",
	}, true
}

// lentOverdueFinding: money owed to the user whose repayment date written
// in a memo is within LentDueSoonDays or has passed (WM-LEND-6), or, with no
// date, open longer than 90 days (D30). A date further out is not chased.
func lentOverdueFinding(lent MoneyLent) (RuleFinding, bool) {
	var overdue, dueSoon, stale lentGroup
	for _, person := range lent.People {
		switch {
		case person.Outstanding <= 0:
		case person.RepaymentOverdue():
			overdue.add(person)
		case person.RepaymentDue():
			dueSoon.add(person)
		case person.RepayBy == nil && person.DaysOutstanding != nil && *person.DaysOutstanding > LentOverdueDays:
			stale.add(person)
		}
	}
	people, total := overdue.people+dueSoon.people+stale.people, overdue.total+dueSoon.total+stale.total
	if people == 0 {
		return RuleFinding{}, false
	}
	who := plural(people, "person owes", "people owe")
	const record = "record repayments as deposits in the lending category."
	finding := RuleFinding{
		Rule:      RuleLentOverdue,
		AmountVND: vndAmount(total),
		Page:      "/accounts/lent",
		Urgent:    overdue.people > 0,
	}
	evidence := []string{}
	if overdue.people > 0 {
		evidence = append(evidence, fmt.Sprintf("%s past the repayment date (due %s, %s ago)", FormatVND(overdue.total), overdue.earliest, plural(-overdue.soonest, "day", "days")))
	}
	if dueSoon.people > 0 {
		evidence = append(evidence, fmt.Sprintf("%s due back by %s", FormatVND(dueSoon.total), dueSoon.latest))
	}
	if stale.people > 0 {
		evidence = append(evidence, fmt.Sprintf("%s open longer than %d days with no repayment date; oldest %s", FormatVND(stale.total), LentOverdueDays, plural(stale.oldest, "day", "days")))
	}
	finding.Evidence = strings.Join(evidence, "; ")
	switch {
	case overdue.people > 0:
		finding.Title = fmt.Sprintf("Chase money lent: %s %s, repayment overdue since %s", who, FormatVND(total), overdue.earliest)
		finding.Action = "Remind them the agreed date has passed, write the new date in the memo if it moves, and " + record
		finding.DueDate = overdue.earliest
	case dueSoon.people > 0:
		finding.Title = fmt.Sprintf("Chase money lent: %s %s, repayment due %s", who, FormatVND(total), dueIn(dueSoon.soonest))
		finding.Action = fmt.Sprintf("Remind them repayment is due on %s, and %s", dueSoon.earliest, record)
		finding.DueDate = dueSoon.earliest
	default:
		finding.Title = fmt.Sprintf("Chase money lent: %s %s for over %d days", who, FormatVND(total), LentOverdueDays)
		finding.Action = "Ask for repayment or agree a date (write it in the memo, e.g. \"repay 30/11/2026\"), and " + record
	}
	return finding, true
}

// lentGroup sums the people of one chase reason.
type lentGroup struct {
	people int
	total  float64
	// earliest and latest repayment dates, soonest days until repayment, oldest days outstanding.
	earliest, latest string
	soonest, oldest  int
}

func (g *lentGroup) add(person MoneyLentPerson) {
	if person.RepayBy != nil {
		if g.people == 0 || *person.RepayBy < g.earliest {
			g.earliest, g.soonest = *person.RepayBy, *person.DaysUntilRepay
		}
		g.latest = max(g.latest, *person.RepayBy)
	}
	if person.DaysOutstanding != nil {
		g.oldest = max(g.oldest, *person.DaysOutstanding)
	}
	g.people++
	g.total += person.Outstanding
}

func dueIn(n int) string {
	switch n {
	case 0:
		return "today"
	case 1:
		return "tomorrow"
	}
	return "in " + plural(n, "day", "days")
}

// vndAmount rounds a figure to whole VND.
func vndAmount(value float64) *float64 {
	rounded := math.Round(value)
	return &rounded
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
