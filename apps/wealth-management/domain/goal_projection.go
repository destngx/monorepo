package domain

import (
	"cmp"
	"math"
	"slices"
	"time"
)

// GoalPace is a goal's standing in the plan.
type GoalPace string

const (
	PaceDone   GoalPace = "done"
	PacePaused GoalPace = "paused"
	// PaceFunded: the pool covers the whole target today.
	PaceFunded GoalPace = "funded"
	// PaceOnTrack: projected to finish by the target date.
	PaceOnTrack GoalPace = "on_track"
	// PaceBehind: projected after the target date, or not reachable.
	PaceBehind GoalPace = "behind"
	// PaceNoDeadline: no target date; the projection is the only pace.
	PaceNoDeadline GoalPace = "no_deadline"
)

const (
	// goalHorizonMonths caps the projection (50 years); beyond it a goal is
	// not reachable at the current pace.
	goalHorizonMonths = 600
	// longHorizonMonths marks goals shown in years to reach rather than a
	// progress ring near 0%.
	longHorizonMonths = 36
)

// GoalProgress is one goal in the computed plan.
type GoalProgress struct {
	PlannedGoal
	EmergencyFund bool `json:"emergencyFund"`
	// TargetAmount is the target used: the row's, or EmergencyMonths x
	// monthly needs for an emergency fund with a blank target.
	TargetAmount   float64 `json:"targetAmount"`
	TargetComputed bool    `json:"targetComputed"`
	// Funded = starting amount + FromPool (the target for a done goal).
	Funded    float64 `json:"funded"`
	FromPool  float64 `json:"fromPool"`
	Remaining float64 `json:"remaining"`
	Percent   float64 `json:"percent"`
	// MonthsLeft counts months from this month to the target month.
	MonthsLeft *int `json:"monthsLeft"`
	// RequiredPerMonth = remaining / months left, for dated active goals.
	RequiredPerMonth *float64 `json:"requiredPerMonth"`
	// ProjectedFinish ("yyyy-mm") adds monthly savings in priority order;
	// nil when not reachable within 50 years, paused or done.
	ProjectedFinish *string  `json:"projectedFinish"`
	MonthsToReach   *int     `json:"monthsToReach"`
	YearsToReach    *float64 `json:"yearsToReach"`
	LongHorizon     bool     `json:"longHorizon"`
	Pace            GoalPace `json:"pace"`
}

// PlanVerdict answers "does my saving cover my goals?".
type PlanVerdict string

const (
	PlanFits        PlanVerdict = "fits"
	PlanShort       PlanVerdict = "short"
	PlanNoDeadlines PlanVerdict = "no_deadlines"
)

// GoalTradeOff is one way to close the gap by changing a single goal:
// move its target date (nil when moving it alone is not enough) or lower its
// target (nil when lowering it alone is not enough).
type GoalTradeOff struct {
	GoalID           string   `json:"goalId"`
	Name             string   `json:"name"`
	TargetDate       string   `json:"targetDate"`
	MoveTargetDateTo *string  `json:"moveTargetDateTo"`
	LowerTargetBy    *float64 `json:"lowerTargetBy"`
}

// PlanCheck compares what dated goals need per month with what is saved.
// Gap > 0 is the monthly shortfall.
type PlanCheck struct {
	RequiredPerMonth float64        `json:"requiredPerMonth"`
	MonthlySavings   float64        `json:"monthlySavings"`
	Gap              float64        `json:"gap"`
	Verdict          PlanVerdict    `json:"verdict"`
	DatedGoals       int            `json:"datedGoals"`
	TradeOffs        []GoalTradeOff `json:"tradeOffs"`
}

// GoalPlan is the computed goal plan (WM-GOAL-1..).
type GoalPlan struct {
	AsOf           string           `json:"asOf"`
	SetupNeeded    bool             `json:"setupNeeded"`
	LegacyGoals    int              `json:"legacyGoals"`
	Settings       GoalPoolSettings `json:"settings"`
	Pool           GoalPool         `json:"pool"`
	Needs          MonthlyNeeds     `json:"needs"`
	MonthlySavings MonthlySavings   `json:"monthlySavings"`
	// Goals are in priority order (sheet row on ties).
	Goals []GoalProgress `json:"goals"`
	// Unallocated is pool money left after every active goal is filled.
	Unallocated float64         `json:"unallocated"`
	Check       PlanCheck       `json:"check"`
	Issues      []GoalPlanIssue `json:"issues"`
}

// ComputeGoalPlan fills the goals from the pool by priority (goals with the
// same priority share it equally), projects the rest from monthly savings
// and checks the plan.
func ComputeGoalPlan(sheet GoalPlanSheet, accounts []Account, transactions []Transaction, categories []Category, settings GoalPoolSettings, today time.Time) GoalPlan {
	plan := GoalPlan{
		AsOf:           today.Format(time.DateOnly),
		SetupNeeded:    !sheet.Exists,
		LegacyGoals:    sheet.LegacyGoals,
		Settings:       settings,
		Pool:           ComputeGoalPool(accounts, transactions, settings, today),
		Needs:          ComputeMonthlyNeeds(transactions, categories, today),
		MonthlySavings: ComputeMonthlySavings(transactions, today),
		Goals:          []GoalProgress{},
		Issues:         append([]GoalPlanIssue{}, sheet.Issues...),
	}
	ordered := slices.Clone(sheet.Goals)
	slices.SortStableFunc(ordered, func(a, b PlannedGoal) int {
		return cmp.Or(cmp.Compare(priorityKey(a.Priority), priorityKey(b.Priority)), cmp.Compare(a.Row, b.Row))
	})
	emergencyFound := false
	for _, goal := range ordered {
		progress := GoalProgress{PlannedGoal: goal}
		if IsEmergencyFund(goal.Name, goal.Note) && !emergencyFound {
			progress.EmergencyFund, emergencyFound = true, true
		}
		switch {
		case goal.Target != nil:
			progress.TargetAmount = *goal.Target
		case progress.EmergencyFund:
			progress.TargetAmount, progress.TargetComputed = math.Round(settings.EmergencyMonths*plan.Needs.PerMonth), true
		default:
			plan.Issues = append(plan.Issues, GoalPlanIssue{Row: goal.Row, Message: "only one emergency fund may leave its target blank; give this goal a target"})
			continue
		}
		plan.Goals = append(plan.Goals, progress)
	}
	plan.Unallocated = fillFromPool(plan.Goals, plan.Pool.Total)
	current := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	projectGoals(plan.Goals, plan.MonthlySavings.PerMonth, current)
	plan.Check = checkPlan(plan.Goals, plan.MonthlySavings.PerMonth, current)
	return plan
}

// priorityKey puts a missing priority (0) after every numbered one.
func priorityKey(priority int) int {
	if priority <= 0 {
		return math.MaxInt
	}
	return priority
}

// priorityGroups splits goals in priority order into runs that share
// money equally: goals with the same priority. A goal without a priority is
// a run of its own, so those fill in sheet order.
func priorityGroups(goals []GoalProgress) [][]int {
	var groups [][]int
	for i, goal := range goals {
		if last := len(groups) - 1; last >= 0 && goal.Priority > 0 && goals[groups[last][0]].Priority == goal.Priority {
			groups[last] = append(groups[last], i)
			continue
		}
		groups = append(groups, []int{i})
	}
	return groups
}

// shareEqually splits budget equally across needs. A need smaller than its
// share takes only what it needs and the rest is split among the others.
// It returns each take, in the order of needs.
func shareEqually(needs []float64, budget float64) []float64 {
	takes := make([]float64, len(needs))
	order := make([]int, len(needs))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int { return cmp.Compare(needs[a], needs[b]) })
	for k, i := range order {
		takes[i] = math.Min(needs[i], budget/float64(len(order)-k))
		budget -= takes[i]
	}
	return takes
}

// fillFromPool funds the goals by priority group and returns the pool left.
// Done goals count as fully funded and take nothing; paused goals keep only
// their starting amount.
func fillFromPool(goals []GoalProgress, pool float64) float64 {
	for _, group := range priorityGroups(goals) {
		var active []int
		var needs []float64
		for _, i := range group {
			progress := &goals[i]
			target := progress.TargetAmount
			switch progress.Status {
			case GoalDone:
				progress.Funded, progress.Pace = target, PaceDone
			case GoalPaused:
				progress.Funded, progress.Pace = math.Min(progress.StartingAmount, target), PacePaused
			default:
				progress.Funded = math.Min(progress.StartingAmount, target)
				active = append(active, i)
				needs = append(needs, target-progress.Funded)
			}
		}
		for k, take := range shareEqually(needs, pool) {
			progress := &goals[active[k]]
			progress.FromPool = take
			progress.Funded += take
			pool -= take
		}
		for _, i := range group {
			progress := &goals[i]
			progress.Remaining = math.Max(0, progress.TargetAmount-progress.Funded)
			if progress.TargetAmount > 0 {
				progress.Percent = math.Round(progress.Funded/progress.TargetAmount*1000) / 10
			} else {
				progress.Percent = 100
			}
		}
	}
	return pool
}

// projectGoals simulates month by month: each month's savings fill the
// unfinished active goals in priority order, shared equally within a
// priority, then sets each goal's required pace, projection and status.
func projectGoals(goals []GoalProgress, savings float64, current time.Time) {
	remaining := make([]float64, len(goals))
	finish := make([]int, len(goals))
	unfinished := 0
	for i, goal := range goals {
		finish[i] = -1
		if goal.Pace == PaceDone || goal.Pace == PacePaused {
			continue
		}
		if remaining[i] = goal.Remaining; remaining[i] <= 0 {
			finish[i] = 0
			continue
		}
		unfinished++
	}
	groups := priorityGroups(goals)
	for month := 1; month <= goalHorizonMonths && unfinished > 0 && savings > 0; month++ {
		budget := savings
		for _, group := range groups {
			if budget <= 0 {
				break
			}
			var open []int
			var needs []float64
			for _, i := range group {
				if finish[i] < 0 && remaining[i] > 0 {
					open = append(open, i)
					needs = append(needs, remaining[i])
				}
			}
			for k, take := range shareEqually(needs, budget) {
				i := open[k]
				remaining[i] -= take
				budget -= take
				if remaining[i] <= 0 {
					finish[i] = month
					unfinished--
				}
			}
		}
	}
	for i := range goals {
		paceGoal(&goals[i], finish[i], current)
	}
}

func paceGoal(goal *GoalProgress, finish int, current time.Time) {
	if goal.Pace == PaceDone || goal.Pace == PacePaused {
		return
	}
	if finish >= 0 {
		months := finish
		years := math.Round(float64(finish)/12*10) / 10
		finishMonth := MonthKey(current.AddDate(0, finish, 0))
		goal.MonthsToReach, goal.YearsToReach, goal.ProjectedFinish = &months, &years, &finishMonth
	}
	if goal.TargetDate != "" {
		if target, err := ParseMonthKey(goal.TargetDate); err == nil {
			left := monthsBetween(current, target)
			goal.MonthsLeft = &left
			if goal.Remaining > 0 {
				required := math.Ceil(goal.Remaining / float64(max(1, left)))
				goal.RequiredPerMonth = &required
			}
		}
	}
	goal.LongHorizon = (goal.MonthsToReach != nil && *goal.MonthsToReach >= longHorizonMonths) ||
		(goal.MonthsLeft != nil && *goal.MonthsLeft >= longHorizonMonths)
	switch {
	case goal.Remaining <= 0:
		goal.Pace = PaceFunded
	case goal.MonthsLeft == nil:
		goal.Pace = PaceNoDeadline
	case goal.MonthsToReach != nil && *goal.MonthsToReach <= *goal.MonthsLeft:
		goal.Pace = PaceOnTrack
	default:
		goal.Pace = PaceBehind
	}
}
