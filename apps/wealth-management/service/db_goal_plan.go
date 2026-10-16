package service

import (
	"apps/wealth-management/domain"
	"fmt"
	"math"
	"slices"
	"strings"
)

// Goal Plan (D31, REQUIREMENTS §2): an app-managed tab created on first
// use. The engine writes only the goal rows the user changes; the old Goals
// tab is read once, for the import, and never written.

const (
	goalPlanSheet    = "Goal Plan"
	goalPlanRange    = "'Goal Plan'!A2:G"
	goalPlanAnchor   = "'Goal Plan'!A2:A"
	goalPlanFirstRow = 2
	goalPlanColumns  = 7 // A-G

	legacyGoalsSheet = "Goals"
	legacyGoalsRange = "Goals!A1:Z50"

	cacheKeyGoalPlan    = "goals:plan"
	cacheKeyGoalPattern = "goals:*"

	emergencyFundName = "Emergency fund"
	emergencyFundNote = "Blank target = computed from monthly needs"
)

var goalPlanHeader = []any{"Name", "Target", "Target date", "Priority", "Starting amount", "Status", "Note"}

func (s *dbService) ReadGoalPlan(forceFresh bool) (domain.GoalPlanSheet, error) {
	return cachedRead(s, cacheKeyGoalPlan, sheetDataTTL, forceFresh, func() (domain.GoalPlanSheet, error) {
		titles, err := s.sheetTitles()
		if err != nil {
			return domain.GoalPlanSheet{}, err
		}
		sheet := domain.GoalPlanSheet{Exists: titles[goalPlanSheet], Goals: []domain.PlannedGoal{}, Issues: []domain.GoalPlanIssue{}}
		if !sheet.Exists {
			legacy, err := s.readLegacyGoals(titles)
			if err != nil {
				return domain.GoalPlanSheet{}, err
			}
			sheet.LegacyGoals = len(legacy.imported)
			return sheet, nil
		}
		rows, err := s.client.ReadSheet(goalPlanRange)
		if err != nil {
			return domain.GoalPlanSheet{}, fmt.Errorf("read goal plan: %w", err)
		}
		sheet.Goals, sheet.Issues = parseGoalPlan(rows)
		return sheet, nil
	})
}

// goalRowValues renders columns A-G for USER_ENTERED. The target date is
// written as text ('yyyy-mm) so Sheets does not turn it into a date.
func goalRowValues(input domain.GoalInput, priority int, status domain.GoalStatus) []any {
	target, starting, date := any(""), any(""), any("")
	if input.Target != nil {
		target = *input.Target
	}
	if input.StartingAmount != nil && *input.StartingAmount > 0 {
		starting = *input.StartingAmount
	}
	if month, _ := domain.ParseGoalMonth(input.TargetDate); month != "" {
		date = "'" + month
	}
	return []any{strings.TrimSpace(input.Name), target, date, priority, starting, string(status), strings.TrimSpace(input.Note)}
}

// readGoalRow reads one goal row back so the caller gets the sheet's own
// values and a matching ETag.
func (s *dbService) readGoalRow(row int) (domain.PlannedGoal, error) {
	rows, err := s.client.ReadSheet(goalRowRange(row))
	if err != nil {
		return domain.PlannedGoal{}, fmt.Errorf("read goal %s: %w", goalID(row), err)
	}
	goal, problem, ok := mapGoalPlanRow(rowAt(rows, 0), row)
	if problem != "" {
		return domain.PlannedGoal{}, fmt.Errorf("%w: goal %s: %s", domain.ErrValidation, goalID(row), problem)
	}
	if !ok {
		return domain.PlannedGoal{}, fmt.Errorf("%w: goal %s", domain.ErrNotFound, goalID(row))
	}
	return goal, nil
}

// requireGoalPlan reads the tab's goals; ErrNotFound before setup.
// Callers hold writeMu.
func (s *dbService) requireGoalPlan() ([]domain.PlannedGoal, error) {
	exists, err := s.hasSheet(goalPlanSheet)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("%w: the %s tab does not exist yet; set up the goal plan first", domain.ErrNotFound, goalPlanSheet)
	}
	rows, err := s.client.ReadSheet(goalPlanRange)
	if err != nil {
		return nil, fmt.Errorf("read goal plan: %w", err)
	}
	goals, _ := parseGoalPlan(rows)
	return goals, nil
}

func (s *dbService) CreateGoal(input domain.GoalInput) (domain.PlannedGoal, error) {
	if err := input.Validate(); err != nil {
		return domain.PlannedGoal{}, err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	goals, err := s.requireGoalPlan()
	if err != nil {
		return domain.PlannedGoal{}, err
	}
	priority := input.Priority
	if priority == 0 {
		for _, goal := range goals {
			priority = max(priority, goal.Priority)
		}
		priority++
	}
	status := input.Status
	if status == "" {
		status = domain.GoalActive
	}
	row, err := s.client.WriteRowsAtFirstEmptyRow(goalPlanSheet, goalPlanAnchor, [][]any{goalRowValues(input, priority, status)})
	if err != nil {
		return domain.PlannedGoal{}, fmt.Errorf("write goal: %w", err)
	}
	s.invalidate(cacheKeyGoalPattern)
	return s.readGoalRow(row)
}

// UpdateGoal replaces columns A-G. A zero Priority or empty Status keeps
// the row's current value.
func (s *dbService) UpdateGoal(id string, input domain.GoalUpdateInput) (domain.PlannedGoal, error) {
	if err := input.Validate(); err != nil {
		return domain.PlannedGoal{}, err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	row, current, err := s.checkGoalETag(id, input.ETag)
	if err != nil {
		return domain.PlannedGoal{}, err
	}
	priority, status := input.Priority, input.Status
	if priority == 0 {
		priority = current.Priority
	}
	if status == "" {
		status = current.Status
	}
	if err := s.client.UpdateRow(goalRowRange(row), goalRowValues(input.GoalInput, priority, status)); err != nil {
		return domain.PlannedGoal{}, fmt.Errorf("update goal: %w", err)
	}
	s.invalidate(cacheKeyGoalPattern)
	return s.readGoalRow(row)
}

// DeleteGoal clears columns A-G; other goal IDs do not shift.
func (s *dbService) DeleteGoal(id string, etag string) error {
	if strings.TrimSpace(etag) == "" {
		return fmt.Errorf("%w: etag is required", domain.ErrValidation)
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	row, _, err := s.checkGoalETag(id, etag)
	if err != nil {
		return err
	}
	blank := make([]any, goalPlanColumns)
	for i := range blank {
		blank[i] = ""
	}
	if err := s.client.UpdateRow(goalRowRange(row), blank); err != nil {
		return fmt.Errorf("delete goal: %w", err)
	}
	s.invalidate(cacheKeyGoalPattern)
	return nil
}

// checkGoalETag re-reads the row and returns it when it exists and is
// unchanged. A row the reader rejects can still be replaced or cleared.
// Callers hold writeMu.
func (s *dbService) checkGoalETag(id string, etag string) (int, domain.PlannedGoal, error) {
	row, err := parseGoalID(id)
	if err != nil {
		return 0, domain.PlannedGoal{}, err
	}
	rows, err := s.client.ReadSheet(goalRowRange(row))
	if err != nil {
		return 0, domain.PlannedGoal{}, fmt.Errorf("read goal %s: %w", id, err)
	}
	current := rowAt(rows, 0)
	if strings.TrimSpace(strings.Join(current, "")) == "" {
		return 0, domain.PlannedGoal{}, fmt.Errorf("%w: goal %s", domain.ErrNotFound, id)
	}
	if rowETag(current, goalPlanColumns) != strings.TrimSpace(etag) {
		return 0, domain.PlannedGoal{}, fmt.Errorf("%w: goal %s changed since it was read; reload and retry", domain.ErrConflict, id)
	}
	goal, _, _ := mapGoalPlanRow(current, row)
	return row, goal, nil
}

// ReorderGoals writes priorities 1..n (column D) in the order given, only
// where they change. Goals not listed (e.g. added meanwhile) follow in their
// current order; an ID that is not a goal of the tab is a conflict.
func (s *dbService) ReorderGoals(input domain.GoalReorderInput) ([]domain.PlannedGoal, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	goals, err := s.requireGoalPlan()
	if err != nil {
		return nil, err
	}
	byID := make(map[string]domain.PlannedGoal, len(goals))
	for _, goal := range goals {
		byID[goal.ID] = goal
	}
	order := make([]domain.PlannedGoal, 0, len(goals))
	listed := map[string]bool{}
	for _, id := range input.IDs {
		goal, ok := byID[strings.TrimSpace(id)]
		if !ok {
			return nil, fmt.Errorf("%w: goal %s is not in the goal plan; reload and retry", domain.ErrConflict, id)
		}
		order = append(order, goal)
		listed[goal.ID] = true
	}
	rest := make([]domain.PlannedGoal, 0, len(goals))
	for _, goal := range goals {
		if !listed[goal.ID] {
			rest = append(rest, goal)
		}
	}
	slices.SortStableFunc(rest, func(a, b domain.PlannedGoal) int { return goalPriorityKey(a) - goalPriorityKey(b) })
	priorities := map[string]int{}
	for index, goal := range append(order, rest...) {
		priorities[goal.ID] = index + 1
	}
	for _, goal := range goals {
		priority := priorities[goal.ID]
		if goal.Priority == priority {
			continue
		}
		if err := s.client.UpdateRow(fmt.Sprintf("'%s'!D%d", goalPlanSheet, goal.Row), []any{priority}); err != nil {
			s.invalidate(cacheKeyGoalPattern)
			return nil, fmt.Errorf("update goal priority: %w", err)
		}
	}
	s.invalidate(cacheKeyGoalPattern)
	rows, err := s.client.ReadSheet(goalPlanRange)
	if err != nil {
		return nil, fmt.Errorf("read goal plan: %w", err)
	}
	reordered, _ := parseGoalPlan(rows)
	slices.SortStableFunc(reordered, func(a, b domain.PlannedGoal) int { return a.Priority - b.Priority })
	return reordered, nil
}

// goalPriorityKey puts a goal without a priority after numbered ones.
func goalPriorityKey(goal domain.PlannedGoal) int {
	if goal.Priority <= 0 {
		return math.MaxInt32
	}
	return goal.Priority
}
