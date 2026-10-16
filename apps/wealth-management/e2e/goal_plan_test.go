package e2e

import (
	"apps/wealth-management/domain"
	"net/http"
	"testing"
)

// TestE2EGoalPlanRoundTrip sets the Goal Plan up on the in-memory workbook
// when it is missing, then creates, edits, reorders and deletes a goal and
// reads each change back (WM-GOAL-2..4).
func TestE2EGoalPlanRoundTrip(t *testing.T) {
	h := engine(t)
	var plan domain.GoalPlan
	h.request(t, http.MethodGet, "/api/goals?force=true", nil, http.StatusOK, &plan)
	if plan.SetupNeeded {
		var setup domain.GoalPlanSetupResult
		h.request(t, http.MethodPost, "/api/goals/setup", nil, http.StatusCreated, &setup)
		if setup.Created != setup.Imported+1 {
			t.Fatalf("setup should write the emergency fund plus the imported goals: %+v", setup)
		}
		h.request(t, http.MethodPost, "/api/goals/setup", nil, http.StatusConflict, nil)
	}

	name := "wm-e2e " + h.runID + " goal"
	var created domain.PlannedGoal
	h.request(t, http.MethodPost, "/api/goals", map[string]any{"name": name, "target": 10_000_000, "targetDate": "2030-06"}, http.StatusCreated, &created)
	if created.TargetDate != "2030-06" || created.Target == nil || *created.Target != 10_000_000 || created.ETag == "" {
		t.Fatalf("the new goal did not read back: %+v", created)
	}

	var edited domain.PlannedGoal
	h.request(t, http.MethodPatch, "/api/goals/"+created.ID, map[string]any{"name": name, "target": 12_000_000, "status": "paused", "etag": created.ETag}, http.StatusOK, &edited)
	if edited.Status != domain.GoalPaused || *edited.Target != 12_000_000 {
		t.Fatalf("the edit did not read back: %+v", edited)
	}
	h.request(t, http.MethodPatch, "/api/goals/"+created.ID, map[string]any{"name": name, "target": 1, "etag": created.ETag}, http.StatusConflict, nil)

	var reordered []domain.PlannedGoal
	h.request(t, http.MethodPost, "/api/goals/reorder", map[string]any{"ids": []string{created.ID}}, http.StatusOK, &reordered)
	if len(reordered) == 0 || reordered[0].ID != created.ID || reordered[0].Priority != 1 {
		t.Fatalf("the reorder did not read back: %+v", reordered)
	}

	h.request(t, http.MethodDelete, "/api/goals/"+created.ID, nil, http.StatusOK, nil, "If-Match", reordered[0].ETag)
	h.request(t, http.MethodGet, "/api/goals?force=true", nil, http.StatusOK, &plan)
	for _, goal := range plan.Goals {
		if goal.ID == created.ID {
			t.Fatalf("the deleted goal is still listed: %+v", goal)
		}
	}
}
