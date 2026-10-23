package service

import (
	"apps/wealth-management/domain"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Goal Plan row parsing: tolerant of hand edits (REQUIREMENTS §2).

func parseGoalPlan(rows [][]string) ([]domain.PlannedGoal, []domain.GoalPlanIssue) {
	goals := []domain.PlannedGoal{}
	issues := []domain.GoalPlanIssue{}
	for index, row := range rows {
		sheetRow := goalPlanFirstRow + index
		goal, problem, ok := mapGoalPlanRow(row, sheetRow)
		switch {
		case problem != "":
			issues = append(issues, domain.GoalPlanIssue{Row: sheetRow, Message: problem})
		case ok:
			goals = append(goals, goal)
		}
	}
	return goals, issues
}

// mapGoalPlanRow maps one row. Blank rows and a header row are skipped
// (ok false, no problem); an invalid row returns the problem.
func mapGoalPlanRow(row []string, sheetRow int) (domain.PlannedGoal, string, bool) {
	if strings.TrimSpace(strings.Join(row, "")) == "" {
		return domain.PlannedGoal{}, "", false
	}
	name := strings.TrimSpace(cellAt(row, 0))
	if strings.EqualFold(name, "name") {
		return domain.PlannedGoal{}, "", false
	}
	goal := domain.PlannedGoal{ID: goalID(sheetRow), Row: sheetRow, Name: name, Note: strings.TrimSpace(cellAt(row, 6)), ETag: rowETag(row, goalPlanColumns)}
	var problems []string
	if name == "" {
		problems = append(problems, "name is required")
	}
	target, ok := parseAmount(cellAt(row, 1))
	switch {
	case !ok:
		problems = append(problems, fmt.Sprintf("target %q is not an amount", strings.TrimSpace(cellAt(row, 1))))
	case target != nil && *target <= 0:
		problems = append(problems, "target must be positive")
	case target == nil && name != "" && !domain.IsEmergencyFund(name, goal.Note):
		problems = append(problems, "target is required (only the emergency fund may leave it blank)")
	}
	goal.Target = target
	date, err := domain.ParseGoalMonth(cellAt(row, 2))
	if err != nil {
		problems = append(problems, fmt.Sprintf("target date %q must be yyyy-mm", strings.TrimSpace(cellAt(row, 2))))
	}
	goal.TargetDate = date
	if priority, ok := parseAmount(cellAt(row, 3)); !ok || (priority != nil && (*priority < 1 || *priority != math.Trunc(*priority))) {
		problems = append(problems, fmt.Sprintf("priority %q must be a whole number from 1", strings.TrimSpace(cellAt(row, 3))))
	} else if priority != nil {
		goal.Priority = int(*priority)
	}
	if starting, ok := parseAmount(cellAt(row, 4)); !ok || (starting != nil && *starting < 0) {
		problems = append(problems, fmt.Sprintf("starting amount %q must be an amount of 0 or more", strings.TrimSpace(cellAt(row, 4))))
	} else if starting != nil {
		goal.StartingAmount = *starting
	}
	status, err := domain.ParseGoalStatus(cellAt(row, 5))
	if err != nil {
		problems = append(problems, fmt.Sprintf("status %q must be one of active, paused or done", strings.TrimSpace(cellAt(row, 5))))
	}
	goal.Status = status
	switch slow := strings.ToLower(strings.TrimSpace(cellAt(row, 7))); slow {
	case "", "no", "n", "false":
	case "yes", "y", "x", "true", "✓":
		goal.UseSlowMoney = true
	default:
		problems = append(problems, fmt.Sprintf("slow money %q must be yes or blank", strings.TrimSpace(cellAt(row, 7))))
	}
	if len(problems) > 0 {
		return domain.PlannedGoal{}, strings.Join(problems, "; "), false
	}
	return goal, "", true
}

// parseAmount reads a number cell: blank is nil; unformatted numbers
// ("1e+08") and VND text ("30.000.000 ₫") are accepted; ok is false for
// anything else.
func parseAmount(cell string) (*float64, bool) {
	trimmed := strings.TrimSpace(cell)
	if trimmed == "" {
		return nil, true
	}
	if value := parseOptionalNumber(trimmed); value != nil {
		return value, true
	}
	clean := strings.NewReplacer(".", "", "₫", "", " ", "", ",", "").Replace(trimmed)
	value, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		return nil, false
	}
	return &value, true
}

func goalID(row int) string {
	return fmt.Sprintf("goal-%d", row)
}

func parseGoalID(id string) (int, error) {
	var row int
	trimmed := strings.TrimSpace(id)
	if _, err := fmt.Sscanf(trimmed, "goal-%d", &row); err != nil || row < goalPlanFirstRow || goalID(row) != trimmed {
		return 0, fmt.Errorf("%w: goal id %q must look like goal-N", domain.ErrValidation, id)
	}
	return row, nil
}

func goalRowRange(row int) string {
	return fmt.Sprintf("'%s'!A%d:H%d", goalPlanSheet, row, row)
}
