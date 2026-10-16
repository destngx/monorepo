package ai

import "apps/wealth-management/domain"

// goalPlanContext is the goal plan as AI facts (D31): the pool, monthly
// savings, the plan check and each goal's funding and pace. Names are masked.
type goalPlanContext struct {
	SetupNeeded      bool                  `json:"setupNeeded,omitempty"`
	Pool             float64               `json:"pool"`
	PoolExcludes     string                `json:"poolExcludes"`
	MonthlySavings   float64               `json:"monthlySavings"`
	RequiredPerMonth float64               `json:"requiredPerMonth"`
	Gap              float64               `json:"gap"`
	Verdict          domain.PlanVerdict    `json:"verdict"`
	Goals            []goalContext         `json:"goals"`
	TradeOffs        []domain.GoalTradeOff `json:"tradeOffs,omitempty"`
}

type goalContext struct {
	Name             string            `json:"name"`
	Priority         int               `json:"priority"`
	Status           domain.GoalStatus `json:"status"`
	Pace             domain.GoalPace   `json:"pace"`
	EmergencyFund    bool              `json:"emergencyFund,omitempty"`
	Target           float64           `json:"target"`
	Funded           float64           `json:"funded"`
	Remaining        float64           `json:"remaining"`
	TargetDate       string            `json:"targetDate,omitempty"`
	RequiredPerMonth *float64          `json:"requiredPerMonth,omitempty"`
	ProjectedFinish  *string           `json:"projectedFinish,omitempty"`
	YearsToReach     *float64          `json:"yearsToReach,omitempty"`
}

func (s *Service) goalPlanContext(plan domain.GoalPlan) goalPlanContext {
	out := goalPlanContext{
		SetupNeeded:      plan.SetupNeeded,
		Pool:             plan.Pool.Total,
		PoolExcludes:     "crypto, archived accounts and money lent; card debt and a spending buffer are taken off",
		MonthlySavings:   plan.MonthlySavings.PerMonth,
		RequiredPerMonth: plan.Check.RequiredPerMonth,
		Gap:              plan.Check.Gap,
		Verdict:          plan.Check.Verdict,
		Goals:            make([]goalContext, 0, len(plan.Goals)),
	}
	for _, goal := range plan.Goals {
		out.Goals = append(out.Goals, goalContext{
			Name: s.masker.Mask(goal.Name), Priority: goal.Priority, Status: goal.Status, Pace: goal.Pace, EmergencyFund: goal.EmergencyFund,
			Target: goal.TargetAmount, Funded: goal.Funded, Remaining: goal.Remaining, TargetDate: goal.TargetDate,
			RequiredPerMonth: goal.RequiredPerMonth, ProjectedFinish: goal.ProjectedFinish, YearsToReach: goal.YearsToReach,
		})
	}
	for _, tradeOff := range plan.Check.TradeOffs {
		tradeOff.Name = s.masker.Mask(tradeOff.Name)
		out.TradeOffs = append(out.TradeOffs, tradeOff)
	}
	return out
}
