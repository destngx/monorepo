package local_workbook

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/service"
	"errors"
	"testing"
)

// The Goal Plan writes round-trip through the local workbook (§14.3): the
// tab is created on first use and every change is read back from it.
func TestGivenTheLocalWorkbookWhenManagingTheGoalPlanThenEveryWriteReadsBack(t *testing.T) {
	client := openFixture(t)
	db := service.NewDatabaseService(client, nil, logger.NewTestLogger(t), service.DatabaseServiceOptions{})

	before, err := db.ReadGoalPlan(true)
	if err != nil || before.Exists {
		t.Fatalf("before setup: %+v %v", before, err)
	}
	if _, err := db.SetupGoalPlan(); err != nil {
		t.Fatal(err)
	}
	assertRows(t, read(t, client, "'Goal Plan'!A1:H1"), [][]string{{"Name", "Target", "Target date", "Priority", "Starting amount", "Status", "Note", "Slow money"}})

	car, err := db.CreateGoal(domain.GoalInput{Name: "New car", Target: money(100_000_000), TargetDate: "2030-06", StartingAmount: money(10_000_000)})
	if err != nil {
		t.Fatal(err)
	}
	if car.ID != "goal-3" || car.Priority != 2 || car.TargetDate != "2030-06" || *car.Target != 100_000_000 || car.StartingAmount != 10_000_000 {
		t.Fatalf("created = %+v", car)
	}

	paused, err := db.UpdateGoal(car.ID, domain.GoalUpdateInput{GoalInput: domain.GoalInput{Name: "New car", Target: money(120_000_000), Status: domain.GoalPaused, UseSlowMoney: true}, ETag: car.ETag})
	if err != nil || paused.Status != domain.GoalPaused || *paused.Target != 120_000_000 || paused.TargetDate != "" || paused.Priority != 2 || !paused.UseSlowMoney {
		t.Fatalf("updated = %+v %v", paused, err)
	}
	if _, err := db.UpdateGoal(car.ID, domain.GoalUpdateInput{GoalInput: domain.GoalInput{Name: "New car", Target: money(1)}, ETag: car.ETag}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("a stale etag must conflict: %v", err)
	}

	reordered, err := db.ReorderGoals(domain.GoalReorderInput{IDs: []string{"goal-3", "goal-2"}})
	if err != nil || reordered[0].ID != "goal-3" || reordered[0].Priority != 1 || reordered[1].Priority != 2 {
		t.Fatalf("reordered = %+v %v", reordered, err)
	}

	if err := db.DeleteGoal(car.ID, reordered[0].ETag); err != nil {
		t.Fatal(err)
	}
	after, err := db.ReadGoalPlan(true)
	if err != nil || !after.Exists || len(after.Goals) != 1 || after.Goals[0].Name != "Emergency fund" || len(after.Issues) != 0 {
		t.Fatalf("after delete: %+v %v", after, err)
	}
}

func money(value float64) *float64 { return &value }
