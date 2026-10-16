package domain

import (
	"strings"
	"testing"
)

func plannedGoal(row int, name string, target *float64, date string, priority int, status GoalStatus) PlannedGoal {
	return PlannedGoal{ID: goalTestID(row), Row: row, Name: name, Target: target, TargetDate: date, Priority: priority, Status: status}
}

func goalTestID(row int) string {
	return "goal-" + string(rune('0'+row))
}

// examplePlan: pool 130M, needs 8M a month (emergency target 48M), savings
// 10M a month.
func examplePlan() GoalPlanSheet {
	car := plannedGoal(3, "New car", money(100_000_000), "2026-12", 2, GoalActive)
	car.StartingAmount = 10_000_000
	boat := plannedGoal(7, "Boat", money(50_000_000), "", 6, GoalPaused)
	boat.StartingAmount = 5_000_000
	return GoalPlanSheet{Exists: true, Goals: []PlannedGoal{
		plannedGoal(2, "Emergency fund", nil, "", 1, GoalActive),
		car,
		plannedGoal(4, "Trip", money(30_000_000), "2027-01", 3, GoalActive),
		plannedGoal(5, "House", money(1_000_000_000), "", 4, GoalActive),
		plannedGoal(6, "Laptop", money(20_000_000), "", 5, GoalDone),
		boat,
	}}
}

func computeExample(t *testing.T) GoalPlan {
	t.Helper()
	return ComputeGoalPlan(examplePlan(), poolAccounts(), monthlyHistory(), taggedCategories(), DefaultGoalPoolSettings(), planToday)
}

func goalNamed(t *testing.T, plan GoalPlan, name string) GoalProgress {
	t.Helper()
	for _, goal := range plan.Goals {
		if goal.Name == name {
			return goal
		}
	}
	t.Fatalf("no goal %q in %+v", name, plan.Goals)
	return GoalProgress{}
}

func TestGivenAPoolWhenFillingGoalsThenTheyFillByPriorityAfterTheirStartingAmount(t *testing.T) {
	plan := computeExample(t)

	if plan.Pool.Total != 130_000_000 || plan.MonthlySavings.PerMonth != 10_000_000 || plan.Needs.PerMonth != 8_000_000 {
		t.Fatalf("inputs: pool %v savings %+v needs %+v", plan.Pool.Total, plan.MonthlySavings, plan.Needs)
	}
	emergency := goalNamed(t, plan, "Emergency fund")
	if !emergency.EmergencyFund || !emergency.TargetComputed || emergency.TargetAmount != 48_000_000 || emergency.Funded != 48_000_000 || emergency.Pace != PaceFunded {
		t.Fatalf("emergency = %+v", emergency)
	}
	car := goalNamed(t, plan, "New car")
	if car.FromPool != 82_000_000 || car.Funded != 92_000_000 || car.Remaining != 8_000_000 {
		t.Fatalf("car = %+v", car)
	}
	if trip := goalNamed(t, plan, "Trip"); trip.Funded != 0 || trip.Remaining != 30_000_000 {
		t.Fatalf("trip = %+v", trip)
	}
	if laptop := goalNamed(t, plan, "Laptop"); laptop.Pace != PaceDone || laptop.Funded != 20_000_000 || laptop.FromPool != 0 {
		t.Fatalf("done goal = %+v", laptop)
	}
	if boat := goalNamed(t, plan, "Boat"); boat.Pace != PacePaused || boat.Funded != 5_000_000 || boat.FromPool != 0 || boat.ProjectedFinish != nil {
		t.Fatalf("paused goal = %+v", boat)
	}
	if plan.Unallocated != 0 {
		t.Fatalf("unallocated = %v", plan.Unallocated)
	}
}

func TestGivenMonthlySavingsWhenProjectingThenGoalsFinishInPriorityOrder(t *testing.T) {
	plan := computeExample(t)

	car := goalNamed(t, plan, "New car")
	if *car.RequiredPerMonth != 4_000_000 || *car.MonthsLeft != 2 || *car.ProjectedFinish != "2026-11" || car.Pace != PaceOnTrack {
		t.Fatalf("car = %+v", car)
	}
	trip := goalNamed(t, plan, "Trip")
	if *trip.RequiredPerMonth != 10_000_000 || *trip.ProjectedFinish != "2027-02" || trip.Pace != PaceBehind {
		t.Fatalf("trip = %+v", trip)
	}
	house := goalNamed(t, plan, "House")
	if house.Pace != PaceNoDeadline || house.RequiredPerMonth != nil || *house.MonthsToReach != 104 || *house.ProjectedFinish != "2035-06" {
		t.Fatalf("house = %+v", house)
	}
	if !house.LongHorizon || *house.YearsToReach != 8.7 {
		t.Fatalf("a long goal shows years to reach: %+v", house)
	}
}

func TestGivenDatedGoalsWhenCheckingThePlanThenTheGapAndTradeOffsAreData(t *testing.T) {
	check := computeExample(t).Check

	if check.RequiredPerMonth != 14_000_000 || check.MonthlySavings != 10_000_000 || check.Gap != 4_000_000 || check.Verdict != PlanShort || check.DatedGoals != 2 {
		t.Fatalf("check = %+v", check)
	}
	if len(check.TradeOffs) != 2 || check.TradeOffs[0].Name != "Trip" || check.TradeOffs[1].Name != "New car" {
		t.Fatalf("trade-offs should start with the lowest priority: %+v", check.TradeOffs)
	}
	trip := check.TradeOffs[0]
	if trip.MoveTargetDateTo == nil || *trip.MoveTargetDateTo != "2027-03" || trip.LowerTargetBy == nil || *trip.LowerTargetBy != 12_000_000 {
		t.Fatalf("trip trade-off = %+v", trip)
	}
	car := check.TradeOffs[1]
	if car.MoveTargetDateTo != nil || car.LowerTargetBy == nil || *car.LowerTargetBy != 8_000_000 {
		t.Fatalf("moving the car alone cannot close the gap: %+v", car)
	}
}

func TestGivenEnoughSavingsWhenCheckingThePlanThenItFits(t *testing.T) {
	sheet := GoalPlanSheet{Exists: true, Goals: []PlannedGoal{plannedGoal(2, "Trip", money(30_000_000), "2027-04", 1, GoalActive)}}

	plan := ComputeGoalPlan(sheet, nil, monthlyHistory(), taggedCategories(), DefaultGoalPoolSettings(), planToday)

	if plan.Check.Verdict != PlanFits || plan.Check.Gap != -5_000_000 || len(plan.Check.TradeOffs) != 0 {
		t.Fatalf("check = %+v", plan.Check)
	}
	if plan.Goals[0].Pace != PaceOnTrack || *plan.Goals[0].ProjectedFinish != "2027-01" {
		t.Fatalf("goal = %+v", plan.Goals[0])
	}
}

func TestGivenNoSavingsWhenProjectingThenUnfundedGoalsAreNotReachable(t *testing.T) {
	sheet := GoalPlanSheet{Exists: true, Goals: []PlannedGoal{
		plannedGoal(2, "Trip", money(30_000_000), "2027-04", 1, GoalActive),
		plannedGoal(3, "House", money(900_000_000), "", 2, GoalActive),
	}}

	plan := ComputeGoalPlan(sheet, nil, nil, nil, DefaultGoalPoolSettings(), planToday)

	trip, house := plan.Goals[0], plan.Goals[1]
	if trip.Pace != PaceBehind || trip.ProjectedFinish != nil || trip.MonthsToReach != nil {
		t.Fatalf("trip = %+v", trip)
	}
	if house.Pace != PaceNoDeadline || house.ProjectedFinish != nil {
		t.Fatalf("house = %+v", house)
	}
	if plan.Check.Verdict != PlanShort {
		t.Fatalf("check = %+v", plan.Check)
	}
}

func TestGivenAGoalBeyondTheHorizonWhenProjectingThenItIsNotReachable(t *testing.T) {
	sheet := GoalPlanSheet{Exists: true, Goals: []PlannedGoal{plannedGoal(2, "Island", money(1e13), "", 1, GoalActive)}}

	plan := ComputeGoalPlan(sheet, nil, monthlyHistory(), taggedCategories(), DefaultGoalPoolSettings(), planToday)

	if plan.Goals[0].ProjectedFinish != nil || plan.Goals[0].YearsToReach != nil {
		t.Fatalf("goal = %+v", plan.Goals[0])
	}
}

func TestGivenNoDatedGoalsWhenCheckingThePlanThenThereIsNoDeadlineVerdict(t *testing.T) {
	sheet := GoalPlanSheet{Exists: true, Goals: []PlannedGoal{plannedGoal(2, "House", money(900_000_000), "", 1, GoalActive)}}

	if plan := ComputeGoalPlan(sheet, nil, monthlyHistory(), nil, DefaultGoalPoolSettings(), planToday); plan.Check.Verdict != PlanNoDeadlines {
		t.Fatalf("check = %+v", plan.Check)
	}
}

func TestGivenTwoEmergencyRowsWithoutTargetWhenComputingThenOnlyTheFirstIsTheFund(t *testing.T) {
	sheet := GoalPlanSheet{Exists: true, Goals: []PlannedGoal{
		plannedGoal(2, "Emergency fund", nil, "", 1, GoalActive),
		plannedGoal(3, "Emergency fund 2", nil, "", 2, GoalActive),
	}}

	plan := ComputeGoalPlan(sheet, nil, monthlyHistory(), taggedCategories(), DefaultGoalPoolSettings(), planToday)

	if len(plan.Goals) != 1 || len(plan.Issues) != 1 || plan.Issues[0].Row != 3 || !strings.Contains(plan.Issues[0].Message, "only one emergency fund") {
		t.Fatalf("goals %+v issues %+v", plan.Goals, plan.Issues)
	}
}

func TestGivenAnEmergencyRowWithATargetWhenComputingThenTheTargetIsKept(t *testing.T) {
	sheet := GoalPlanSheet{Exists: true, Goals: []PlannedGoal{plannedGoal(2, "Emergency fund", money(20_000_000), "", 1, GoalActive)}}

	plan := ComputeGoalPlan(sheet, nil, monthlyHistory(), taggedCategories(), DefaultGoalPoolSettings(), planToday)

	if goal := plan.Goals[0]; !goal.EmergencyFund || goal.TargetComputed || goal.TargetAmount != 20_000_000 {
		t.Fatalf("goal = %+v", goal)
	}
}

func TestGivenNoTabWhenComputingThenSetupIsNeeded(t *testing.T) {
	plan := ComputeGoalPlan(GoalPlanSheet{LegacyGoals: 3}, poolAccounts(), monthlyHistory(), taggedCategories(), DefaultGoalPoolSettings(), planToday)

	if !plan.SetupNeeded || plan.LegacyGoals != 3 || len(plan.Goals) != 0 || plan.Pool.Total != 130_000_000 {
		t.Fatalf("plan = %+v", plan)
	}
}

func TestGivenAPlanWhenSummarisingForAIThenOnlyNumbersAreGiven(t *testing.T) {
	facts := computeExample(t).Facts()
	if facts == nil || facts.ActiveGoals != 4 || facts.Behind != 1 || facts.Gap != 4_000_000 || facts.Verdict != PlanShort {
		t.Fatalf("facts = %+v", facts)
	}
	if (GoalPlan{SetupNeeded: true}).Facts() != nil {
		t.Fatal("no facts before setup")
	}
}
