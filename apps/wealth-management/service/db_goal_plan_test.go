package service

import (
	"apps/wealth-management/domain"
	"errors"
	"strings"
	"testing"
)

func goalPlanClient(rows [][]string) *fakeSheetClient {
	client := newFakeSheetClient()
	client.titles = []string{"Accounts", legacyGoalsSheet, goalPlanSheet}
	client.ranges[goalPlanRange] = rows
	return client
}

func TestGivenAGoalPlanTabWhenReadingThenRowsAreParsedAndBadRowsReported(t *testing.T) {
	client := goalPlanClient([][]string{
		{"Emergency fund", "", "", "1", "", "active", "computed"},
		{"New car", "1e+08", "47269", "2", "10000000", "Active"},
		{},
		{"Name", "Target", "Target date", "Priority", "Starting amount", "Status", "Note"},
		{"Trip", "30.000.000", "2027-01", "3", "", "paused"},
		{"Boat", "50000000", "", "4", "", "later"},
		{"House", "", "", "5"},
		{"", "1000"},
		{"Laptop", "20000000", "soon", "x"},
	})
	sheet, err := newTestDBService(t, client, newFakeCache()).ReadGoalPlan(true)
	if err != nil || !sheet.Exists {
		t.Fatalf("read: %+v %v", sheet, err)
	}
	if len(sheet.Goals) != 3 {
		t.Fatalf("goals = %+v", sheet.Goals)
	}
	emergency, car, trip := sheet.Goals[0], sheet.Goals[1], sheet.Goals[2]
	if emergency.ID != "goal-2" || emergency.Target != nil || emergency.Status != domain.GoalActive || emergency.ETag == "" {
		t.Fatalf("emergency = %+v", emergency)
	}
	if *car.Target != 100_000_000 || car.TargetDate != "2029-05" || car.Priority != 2 || car.StartingAmount != 10_000_000 {
		t.Fatalf("car = %+v", car)
	}
	if trip.Row != 6 || *trip.Target != 30_000_000 || trip.Status != domain.GoalPaused {
		t.Fatalf("trip = %+v", trip)
	}
	want := map[int]string{7: "status \"later\"", 8: "target is required", 9: "name is required", 10: "target date"}
	if len(sheet.Issues) != len(want) {
		t.Fatalf("issues = %+v", sheet.Issues)
	}
	for _, issue := range sheet.Issues {
		if !strings.Contains(issue.Message, want[issue.Row]) {
			t.Errorf("row %d: %q should mention %q", issue.Row, issue.Message, want[issue.Row])
		}
	}
}

func TestGivenNoGoalPlanTabWhenReadingThenSetupIsNeededAndOldGoalsAreCounted(t *testing.T) {
	client := newFakeSheetClient()
	client.titles = []string{"Accounts", legacyGoalsSheet}
	client.ranges[legacyGoalsRange] = legacyGoalRows()

	sheet, err := newTestDBService(t, client, newFakeCache()).ReadGoalPlan(true)
	if err != nil || sheet.Exists || sheet.LegacyGoals != 2 || len(sheet.Goals) != 0 {
		t.Fatalf("sheet = %+v, %v", sheet, err)
	}
	for _, read := range client.reads {
		if read == goalPlanRange {
			t.Fatal("must not read a tab that does not exist")
		}
	}
}

// legacyGoalRows mimics the old Goals tab: name in E, target in G, current in I.
func legacyGoalRows() [][]string {
	return [][]string{
		{"", "", "", "", "Instructions"},
		{"", "", "", "", "New car", "", "100.000.000 ₫", "", "20000000"},
		{"", "", "", "", "Emergency money", "", "30000000"},
		{"", "", "", "", "Trip", "", "1.5e+07"},
		{"", "", "", "", "Someday", "", "0", "", "5000000"},
		{"", "", "", "", "Insert more rows above"},
	}
}

func TestGivenOldGoalsWhenSettingUpThenTheTabIsCreatedWithTheEmergencyFundFirstAndOldGoalsAreOnlyRead(t *testing.T) {
	client := newFakeSheetClient()
	client.titles = []string{"Accounts", legacyGoalsSheet}
	client.ranges[legacyGoalsRange] = legacyGoalRows()
	cache := newFakeCache()

	result, err := newTestDBService(t, client, cache).SetupGoalPlan()
	if err != nil || result.Created != 3 || result.Imported != 2 {
		t.Fatalf("setup = %+v, %v", result, err)
	}
	if len(result.Skipped) != 2 || result.Skipped[0] != "Emergency money" || result.Skipped[1] != "Someday" {
		t.Fatalf("skipped = %v", result.Skipped)
	}
	if len(client.created) != 1 || client.created[0] != goalPlanSheet {
		t.Fatalf("created = %v", client.created)
	}
	if len(client.appends) != 1 || client.appends[0].sheetName != goalPlanSheet || client.appends[0].anchor != goalPlanAnchor {
		t.Fatalf("appends = %+v", client.appends)
	}
	rows := client.appends[0].rows
	if rows[0][0] != "Emergency fund" || rows[0][1] != "" || rows[0][3] != 1 || rows[0][5] != "active" {
		t.Fatalf("emergency row = %v", rows[0])
	}
	if rows[1][0] != "New car" || rows[1][1] != 100_000_000.0 || rows[1][3] != 2 || rows[2][0] != "Trip" || rows[2][1] != 15_000_000.0 || rows[2][3] != 3 {
		t.Fatalf("imported rows = %v", rows[1:])
	}
	for _, update := range client.updates {
		if strings.HasPrefix(update.rangeValue, legacyGoalsSheet+"!") {
			t.Fatalf("the old Goals tab must never be written: %+v", update)
		}
	}
	if !contains(cache.invalidated, cacheKeyGoalPattern) {
		t.Fatalf("goal caches not invalidated: %v", cache.invalidated)
	}
}

func TestGivenAGoalPlanWithGoalsWhenSettingUpAgainThenItConflicts(t *testing.T) {
	client := goalPlanClient([][]string{{"Trip", "1000", "", "1"}})
	if _, err := newTestDBService(t, client, newFakeCache()).SetupGoalPlan(); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("err = %v", err)
	}
	if len(client.created) != 0 || len(client.appends) != 0 {
		t.Fatal("nothing may be written")
	}
}

func TestGivenAnEmptyGoalPlanTabWhenSettingUpThenRowsAreWrittenWithoutRecreatingIt(t *testing.T) {
	client := goalPlanClient(nil)
	client.titles = []string{goalPlanSheet}
	result, err := newTestDBService(t, client, newFakeCache()).SetupGoalPlan()
	if err != nil || result.Created != 1 || len(client.created) != 0 || len(client.appends) != 1 {
		t.Fatalf("setup = %+v %v created %v", result, err, client.created)
	}
}

func TestGivenANewGoalWhenCreatingThenItGoesLastAndIsReadBack(t *testing.T) {
	client := goalPlanClient([][]string{{"Emergency fund", "", "", "1"}, {"Car", "1000", "", "4"}})
	client.nextRow = 4
	client.ranges["'Goal Plan'!A4:H4"] = [][]string{{"Trip", "30000000", "2027-01", "5", "", "active"}}

	goal, err := newTestDBService(t, client, newFakeCache()).CreateGoal(domain.GoalInput{Name: " Trip ", Target: ptr(30_000_000.0), TargetDate: "2027-1"})
	if err != nil || goal.ID != "goal-4" || goal.Priority != 5 || goal.ETag == "" {
		t.Fatalf("goal = %+v, %v", goal, err)
	}
	row := client.appends[0].rows[0]
	if row[0] != "Trip" || row[1] != 30_000_000.0 || row[2] != "'2027-01" || row[3] != 5 || row[4] != "" || row[5] != "active" {
		t.Fatalf("row = %v", row)
	}
}

func TestGivenNoGoalPlanTabWhenCreatingAGoalThenItIsNotFound(t *testing.T) {
	client := newFakeSheetClient()
	client.titles = []string{"Accounts"}
	_, err := newTestDBService(t, client, newFakeCache()).CreateGoal(domain.GoalInput{Name: "Trip", Target: ptr(1.0)})
	if !errors.Is(err, domain.ErrNotFound) || len(client.appends) != 0 {
		t.Fatalf("err = %v", err)
	}
}

func TestGivenAGoalWhenUpdatingThenTheETagGuardsTheRow(t *testing.T) {
	current := []string{"Trip", "30000000", "2027-01", "3", "", "active"}
	client := goalPlanClient(nil)
	client.ranges["'Goal Plan'!A4:H4"] = [][]string{current}
	svc := newTestDBService(t, client, newFakeCache())
	etag := rowETag(current, goalPlanColumns)

	input := domain.GoalUpdateInput{GoalInput: domain.GoalInput{Name: "Trip", Target: ptr(30_000_000.0), Status: domain.GoalPaused}, ETag: etag}
	if _, err := svc.UpdateGoal("goal-4", input); err != nil {
		t.Fatal(err)
	}
	update := client.updates[0]
	if update.rangeValue != "'Goal Plan'!A4:H4" || update.values[3] != 3 || update.values[5] != "paused" || update.values[2] != "" {
		t.Fatalf("update = %+v (priority 0 keeps the row's)", update)
	}

	input.ETag = "stale"
	if _, err := svc.UpdateGoal("goal-4", input); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("stale etag: %v", err)
	}
	if _, err := svc.UpdateGoal("row-4", input); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("bad id: %v", err)
	}
}

func TestGivenAGoalWhenDeletingThenItsRowIsCleared(t *testing.T) {
	current := []string{"Trip", "30000000", "", "3"}
	client := goalPlanClient(nil)
	client.ranges["'Goal Plan'!A4:H4"] = [][]string{current}
	client.ranges["'Goal Plan'!A5:H5"] = [][]string{{}}
	svc := newTestDBService(t, client, newFakeCache())

	if err := svc.DeleteGoal("goal-4", rowETag(current, goalPlanColumns)); err != nil {
		t.Fatal(err)
	}
	if len(client.updates) != 1 || len(client.updates[0].values) != goalPlanColumns || client.updates[0].values[0] != "" {
		t.Fatalf("updates = %+v", client.updates)
	}
	if err := svc.DeleteGoal("goal-5", "x"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("blank row: %v", err)
	}
}

func TestGivenGoalsWhenReorderingThenOnlyChangedPrioritiesAreWritten(t *testing.T) {
	client := goalPlanClient([][]string{
		{"Emergency fund", "", "", "1"},
		{"Car", "1000", "", "2"},
		{"Trip", "2000", "", "3"},
	})
	svc := newTestDBService(t, client, newFakeCache())

	goals, err := svc.ReorderGoals(domain.GoalReorderInput{IDs: []string{"goal-2", "goal-4", "goal-3"}})
	if err != nil || len(goals) != 3 {
		t.Fatalf("reorder: %+v %v", goals, err)
	}
	if len(client.updates) != 2 || client.updates[0].rangeValue != "'Goal Plan'!D3" || client.updates[0].values[0] != 3 || client.updates[1].rangeValue != "'Goal Plan'!D4" {
		t.Fatalf("updates = %+v", client.updates)
	}
	client.updates = nil
	if _, err := svc.ReorderGoals(domain.GoalReorderInput{IDs: []string{"goal-4"}}); err != nil {
		t.Fatal(err)
	}
	if len(client.updates) != 3 || client.updates[0].values[0] != 2 || client.updates[1].values[0] != 3 || client.updates[2].values[0] != 1 {
		t.Fatalf("unlisted goals follow in their order: %+v", client.updates)
	}
	if _, err := svc.ReorderGoals(domain.GoalReorderInput{IDs: []string{"goal-9"}}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("an unknown goal must conflict: %v", err)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestGivenASlowMoneyColumnWhenReadingThenOnlyMarkedGoalsUseSlowMoney(t *testing.T) {
	client := goalPlanClient([][]string{
		{"House", "1000", "", "1", "", "", "", "Yes"},
		{"Car", "1000", "", "2", "", "", "", "no"},
		{"Trip", "1000", "", "3"},
		{"Boat", "1000", "", "4", "", "", "", "maybe"},
	})
	sheet, err := newTestDBService(t, client, newFakeCache()).ReadGoalPlan(true)
	if err != nil || len(sheet.Goals) != 3 {
		t.Fatalf("read: %+v %v", sheet, err)
	}
	if !sheet.Goals[0].UseSlowMoney || sheet.Goals[1].UseSlowMoney || sheet.Goals[2].UseSlowMoney {
		t.Fatalf("goals = %+v", sheet.Goals)
	}
	if len(sheet.Issues) != 1 || sheet.Issues[0].Row != 5 || !strings.Contains(sheet.Issues[0].Message, "slow money") {
		t.Fatalf("issues = %+v", sheet.Issues)
	}
}

func TestGivenAGoalThatUsesSlowMoneyWhenCreatingThenColumnHAndItsHeaderAreWritten(t *testing.T) {
	client := goalPlanClient(nil)
	client.nextRow = 4
	client.ranges["'Goal Plan'!A4:H4"] = [][]string{{"House", "1000", "", "1", "", "active", "", "yes"}}

	goal, err := newTestDBService(t, client, newFakeCache()).CreateGoal(domain.GoalInput{Name: "House", Target: ptr(1000.0), UseSlowMoney: true})
	if err != nil || !goal.UseSlowMoney {
		t.Fatalf("goal = %+v, %v", goal, err)
	}
	if row := client.appends[0].rows[0]; len(row) != goalPlanColumns || row[7] != "yes" {
		t.Fatalf("row = %v", row)
	}
	if len(client.updates) != 1 || client.updates[0].rangeValue != "'Goal Plan'!H1" || client.updates[0].values[0] != "Slow money" {
		t.Fatalf("an older tab gets the column header: %+v", client.updates)
	}
}
