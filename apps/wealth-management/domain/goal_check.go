package domain

import (
	"math"
	"time"
)

// checkPlan sums the required pace of dated, unfunded active goals against
// monthly savings. When short, each such goal, lowest priority first,
// gets the date move or target cut that alone would close the gap.
func checkPlan(goals []GoalProgress, savings float64, current time.Time) PlanCheck {
	check := PlanCheck{MonthlySavings: savings, TradeOffs: []GoalTradeOff{}}
	var dated []GoalProgress
	for _, goal := range goals {
		if goal.RequiredPerMonth != nil && goal.Pace != PaceDone && goal.Pace != PacePaused {
			dated = append(dated, goal)
			check.RequiredPerMonth += *goal.RequiredPerMonth
		}
	}
	check.DatedGoals = len(dated)
	check.Gap = check.RequiredPerMonth - savings
	switch {
	case len(dated) == 0:
		check.Verdict = PlanNoDeadlines
		return check
	case check.Gap <= 0:
		check.Verdict = PlanFits
		return check
	}
	check.Verdict = PlanShort
	for i := len(dated) - 1; i >= 0; i-- {
		goal := dated[i]
		tradeOff := GoalTradeOff{GoalID: goal.ID, Name: goal.Name, TargetDate: goal.TargetDate}
		if fitted := *goal.RequiredPerMonth - check.Gap; fitted > 0 {
			months := int(math.Ceil(goal.Remaining / fitted))
			moved := MonthKey(current.AddDate(0, months, 0))
			tradeOff.MoveTargetDateTo = &moved
		}
		if cut := check.Gap * float64(max(1, *goal.MonthsLeft)); cut <= goal.Remaining {
			tradeOff.LowerTargetBy = &cut
		}
		check.TradeOffs = append(check.TradeOffs, tradeOff)
	}
	return check
}

// GoalPlanFacts is the goal plan in numbers for AI prompts: no goal names.
type GoalPlanFacts struct {
	ActiveGoals      int         `json:"activeGoals"`
	Behind           int         `json:"behind"`
	Pool             float64     `json:"pool"`
	MonthlySavings   float64     `json:"monthlySavings"`
	RequiredPerMonth float64     `json:"requiredPerMonth"`
	Gap              float64     `json:"gap"`
	Verdict          PlanVerdict `json:"verdict"`
}

// Facts summarises the plan; nil before the Goal Plan tab is set up.
func (plan GoalPlan) Facts() *GoalPlanFacts {
	if plan.SetupNeeded {
		return nil
	}
	facts := GoalPlanFacts{Pool: plan.Pool.Total, MonthlySavings: plan.MonthlySavings.PerMonth, RequiredPerMonth: plan.Check.RequiredPerMonth, Gap: plan.Check.Gap, Verdict: plan.Check.Verdict}
	for _, goal := range plan.Goals {
		if goal.Status == GoalActive {
			facts.ActiveGoals++
		}
		if goal.Pace == PaceBehind {
			facts.Behind++
		}
	}
	return &facts
}
