package fiber

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/internal/fakes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestGivenNoGoalPlanTabWhenGettingGoalsThenSetupIsNeeded(t *testing.T) {
	app := newAnalyticsApp(t, &fakes.Database{GoalPlan: domain.GoalPlanSheet{LegacyGoals: 2}})

	status, body := call(t, app, http.MethodGet, "/api/goals", "")

	var plan domain.GoalPlan
	if status != http.StatusOK || json.Unmarshal([]byte(body), &plan) != nil || !plan.SetupNeeded || plan.LegacyGoals != 2 {
		t.Fatalf("status = %d, body = %s", status, body)
	}
}

func TestGivenAGoalPlanWhenGettingGoalsThenTheComputedPlanIsReturned(t *testing.T) {
	target := 30_000_000.0
	db := &fakes.Database{
		Accounts: []domain.Account{{Name: "Bank", Type: domain.AccountTypeActiveUse, Currency: domain.CurrencyVND, Balance: 10_000_000}},
		GoalPlan: domain.GoalPlanSheet{Exists: true, Goals: []domain.PlannedGoal{{ID: "goal-2", Row: 2, Name: "Trip", Target: &target, Priority: 1, Status: domain.GoalActive}}},
	}
	app := newAnalyticsApp(t, db)

	status, body := call(t, app, http.MethodGet, "/api/goals", "")

	var plan domain.GoalPlan
	if status != http.StatusOK || json.Unmarshal([]byte(body), &plan) != nil || len(plan.Goals) != 1 {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if goal := plan.Goals[0]; goal.Funded != 10_000_000 || goal.Remaining != 20_000_000 || goal.Pace != domain.PaceNoDeadline {
		t.Fatalf("goal = %+v", goal)
	}
}

func TestGivenGoalWritesWhenCallingTheAPIThenTheyAreGuardedValidatedAndPassedOn(t *testing.T) {
	db := &fakes.Database{}
	app := newSheetsApp(t, db)

	if status, body := call(t, app, http.MethodPost, "/api/goals/setup", ""); status != http.StatusCreated || db.GoalSetups != 1 {
		t.Fatalf("setup: %d %s", status, body)
	}
	if status, body := call(t, app, http.MethodPost, "/api/goals", `{"name":"Trip","target":30000000,"targetDate":"2027-06"}`); status != http.StatusCreated || len(db.CreatedGoals) != 1 {
		t.Fatalf("create: %d %s", status, body)
	}
	if status, _ := call(t, app, http.MethodPost, "/api/goals", `{"name":"Trip"}`); status != http.StatusBadRequest {
		t.Fatalf("a goal without a target must be a 400, got %d", status)
	}
	if status, body := call(t, app, http.MethodPatch, "/api/goals/goal-3", `{"name":"Trip","target":1000,"status":"paused"}`, "If-Match", `"abc"`); status != http.StatusOK || db.UpdatedGoals["goal-3"].ETag != "abc" {
		t.Fatalf("update: %d %s %+v", status, body, db.UpdatedGoals)
	}
	if status, body := call(t, app, http.MethodDelete, "/api/goals/goal-3?etag=abc", ""); status != http.StatusOK || db.DeletedGoals["goal-3"] != "abc" {
		t.Fatalf("delete: %d %s", status, body)
	}
	if status, body := call(t, app, http.MethodPost, "/api/goals/reorder", `{"ids":["goal-3","goal-2"]}`); status != http.StatusOK || len(db.GoalOrders) != 1 {
		t.Fatalf("reorder: %d %s", status, body)
	}
	if status, _ := call(t, app, http.MethodPost, "/api/goals", `{"name":"Trip","target":1}`, "Authorization", ""); status != http.StatusUnauthorized {
		t.Fatalf("goal writes must be guarded, got %d", status)
	}
}
