package service

import (
	"apps/wealth-management/domain"
	"fmt"
	"strings"
)

// legacyGoal is a row of the old Goals tab (E name, G target, I current).
type legacyGoal struct {
	name   string
	target float64
}

// legacyGoals splits the old Goals tab into rows a setup imports and the
// names it leaves out: emergency fund rows (the setup writes its own) and
// rows without a target.
type legacyGoals struct {
	imported []legacyGoal
	skipped  []string
}

// readLegacyGoals reads the old Goals tab, only for the import; a missing
// tab has nothing to import.
func (s *dbService) readLegacyGoals(titles map[string]bool) (legacyGoals, error) {
	result := legacyGoals{skipped: []string{}}
	if !titles[legacyGoalsSheet] {
		return result, nil
	}
	rows, err := s.client.ReadSheet(legacyGoalsRange)
	if err != nil {
		return result, fmt.Errorf("read old goals: %w", err)
	}
	for _, row := range rows {
		name := strings.TrimSpace(cellAt(row, 4))
		if name == "" || name == "Instructions" || strings.Contains(name, "Insert more rows") {
			continue
		}
		target, _ := parseAmount(cellAt(row, 6))
		current, _ := parseAmount(cellAt(row, 8))
		hasTarget, hasCurrent := target != nil && *target > 0, current != nil && *current > 0
		switch {
		case !hasTarget && !hasCurrent:
			continue
		case !hasTarget || domain.IsEmergencyFund(name, ""):
			result.skipped = append(result.skipped, name)
		default:
			result.imported = append(result.imported, legacyGoal{name: name, target: *target})
		}
	}
	return result, nil
}

// SetupGoalPlan creates the Goal Plan tab (when missing) and writes the
// emergency fund at priority 1 with a blank, computed target, then the old
// goals in sheet order, all active. A tab that already holds rows is left
// alone (ErrConflict).
func (s *dbService) SetupGoalPlan() (domain.GoalPlanSetupResult, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	titles, err := s.sheetTitles()
	if err != nil {
		return domain.GoalPlanSetupResult{}, err
	}
	if titles[goalPlanSheet] {
		rows, err := s.client.ReadSheet(goalPlanRange)
		if err != nil {
			return domain.GoalPlanSetupResult{}, fmt.Errorf("read goal plan: %w", err)
		}
		for _, row := range rows {
			if strings.TrimSpace(strings.Join(row, "")) != "" {
				return domain.GoalPlanSetupResult{}, fmt.Errorf("%w: the %s tab already has goals", domain.ErrConflict, goalPlanSheet)
			}
		}
	}
	legacy, err := s.readLegacyGoals(titles)
	if err != nil {
		return domain.GoalPlanSetupResult{}, err
	}
	if !titles[goalPlanSheet] {
		if err := s.client.CreateSheet(goalPlanSheet, goalPlanHeader); err != nil {
			return domain.GoalPlanSetupResult{}, fmt.Errorf("create %s tab: %w", goalPlanSheet, err)
		}
	}
	rows := [][]any{goalRowValues(domain.GoalInput{Name: emergencyFundName, Note: emergencyFundNote}, 1, domain.GoalActive)}
	for index, goal := range legacy.imported {
		target := goal.target
		rows = append(rows, goalRowValues(domain.GoalInput{Name: goal.name, Target: &target}, index+2, domain.GoalActive))
	}
	if _, err := s.client.WriteRowsAtFirstEmptyRow(goalPlanSheet, goalPlanAnchor, rows); err != nil {
		s.invalidate(cacheKeyGoalPattern)
		return domain.GoalPlanSetupResult{}, fmt.Errorf("write goal plan: %w", err)
	}
	s.invalidate(cacheKeyGoalPattern)
	return domain.GoalPlanSetupResult{Created: len(rows), Imported: len(legacy.imported), Skipped: legacy.skipped}, nil
}
