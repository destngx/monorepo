package service

import (
	"apps/wealth-management/domain"
	"fmt"
	"strings"
)

// Recommendations tab (ROADMAP R7, REQUIREMENTS §2): app-managed, created
// with its header on the first item. The engine appends new items and
// rewrites only the status columns (H-K) of existing ones; it never writes
// another tab from here.

const (
	recommendationsSheet    = "Recommendations"
	recommendationsRange    = "'Recommendations'!A2:L"
	recommendationsAnchor   = "'Recommendations'!A2:A"
	recommendationsFirstRow = 2
	recommendationsColumns  = 12 // A-L

	cacheKeyRecommendations        = "recommendations:items"
	cacheKeyRecommendationsPattern = "recommendations:*"
)

var recommendationsHeader = []any{"Id", "Created", "Source", "Title", "Action", "Impact VND per year", "Evidence", "Status", "Snoozed until", "Status changed", "Outcome", "Page"}

func (s *dbService) ReadRecommendations(forceFresh bool) (domain.RecommendationSheet, error) {
	return cachedRead(s, cacheKeyRecommendations, sheetDataTTL, forceFresh, s.readRecommendations)
}

// readRecommendations reads the tab; a missing tab has no items.
func (s *dbService) readRecommendations() (domain.RecommendationSheet, error) {
	sheet := domain.RecommendationSheet{Items: []domain.Recommendation{}, Issues: []domain.RecommendationIssue{}}
	exists, err := s.hasSheet(recommendationsSheet)
	if err != nil {
		return domain.RecommendationSheet{}, err
	}
	if !exists {
		return sheet, nil
	}
	rows, err := s.client.ReadSheet(recommendationsRange)
	if err != nil {
		return domain.RecommendationSheet{}, fmt.Errorf("read recommendations: %w", err)
	}
	sheet.Exists = true
	sheet.Items, sheet.Issues = parseRecommendations(rows)
	return sheet, nil
}

func (s *dbService) UpdateRecommendations(plan domain.RecommendationPlan) (domain.RecommendationWrite, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	sheet, err := s.readRecommendations()
	if err != nil {
		return domain.RecommendationWrite{}, err
	}
	changes, err := plan(sheet)
	if err != nil {
		return domain.RecommendationWrite{}, err
	}
	if changes.Empty() {
		return domain.RecommendationWrite{Sheet: sheet, Added: []string{}, Updated: []string{}}, nil
	}
	write := domain.RecommendationWrite{Added: []string{}, Updated: []string{}}
	rows := map[int]bool{}
	for _, item := range sheet.Items {
		rows[item.Row] = true
	}
	for _, update := range changes.Update {
		if !rows[update.Row] {
			return domain.RecommendationWrite{}, fmt.Errorf("%w: recommendation %s is not in the tab", domain.ErrNotFound, update.ID)
		}
	}
	for _, update := range changes.Update {
		values := []any{string(update.Status), dateCell(update.SnoozedUntil), dateCell(update.StatusChanged), textCell(update.Outcome)}
		if err := s.client.UpdateRow(fmt.Sprintf("'%s'!H%d:K%d", recommendationsSheet, update.Row, update.Row), values); err != nil {
			s.invalidate(cacheKeyRecommendationsPattern)
			return domain.RecommendationWrite{}, fmt.Errorf("update recommendation %s: %w", update.ID, err)
		}
		write.Updated = append(write.Updated, update.ID)
	}
	if len(changes.Add) > 0 {
		if err := s.appendRecommendations(sheet.Exists, changes.Add); err != nil {
			s.invalidate(cacheKeyRecommendationsPattern)
			return domain.RecommendationWrite{}, err
		}
		for _, item := range changes.Add {
			write.Added = append(write.Added, item.ID)
		}
	}
	s.invalidate(cacheKeyRecommendationsPattern)
	if write.Sheet, err = s.readRecommendations(); err != nil {
		return domain.RecommendationWrite{}, fmt.Errorf("read recommendations back: %w", err)
	}
	return write, nil
}

// appendRecommendations writes new rows below the last one, creating the
// tab first when it is missing. Callers hold writeMu.
func (s *dbService) appendRecommendations(exists bool, items []domain.Recommendation) error {
	if !exists {
		if err := s.client.CreateSheet(recommendationsSheet, recommendationsHeader); err != nil {
			return fmt.Errorf("create %s tab: %w", recommendationsSheet, err)
		}
	}
	values := make([][]any, 0, len(items))
	for _, item := range items {
		values = append(values, recommendationRowValues(item.SheetRow()))
	}
	if _, err := s.client.WriteRowsAtFirstEmptyRow(recommendationsSheet, recommendationsAnchor, values); err != nil {
		return fmt.Errorf("write recommendations: %w", err)
	}
	return nil
}

// recommendationRowValues renders columns A-L for USER_ENTERED: dates as
// text so they read back as written, model text never as a formula.
func recommendationRowValues(row domain.RecommendationRow) []any {
	impact := any("")
	if row.ImpactVND != nil {
		impact = *row.ImpactVND
	}
	return []any{
		row.ID, dateCell(row.Created), textCell(row.Source), textCell(row.Title), textCell(row.Action), impact,
		textCell(row.Evidence), string(row.Status), dateCell(row.SnoozedUntil), dateCell(row.StatusChanged),
		textCell(row.Outcome), textCell(row.Page),
	}
}

// dateCell writes yyyy-mm-dd as text; blank stays blank.
func dateCell(value string) string {
	if value == "" {
		return ""
	}
	return "'" + value
}

func parseRecommendations(rows [][]string) ([]domain.Recommendation, []domain.RecommendationIssue) {
	items := []domain.Recommendation{}
	issues := []domain.RecommendationIssue{}
	seen := map[string]int{}
	for index, row := range rows {
		sheetRow := recommendationsFirstRow + index
		if strings.TrimSpace(strings.Join(row, "")) == "" {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(cellAt(row, 0)), "id") && strings.EqualFold(strings.TrimSpace(cellAt(row, 1)), "created") {
			continue
		}
		impact, ok := parseAmount(cellAt(row, 5))
		var problems []string
		if !ok {
			problems = append(problems, fmt.Sprintf("impact %q is not an amount", strings.TrimSpace(cellAt(row, 5))))
		}
		item, rowProblems := domain.ParseRecommendationRow(domain.RecommendationRow{
			ID: cellAt(row, 0), Created: cellAt(row, 1), Source: cellAt(row, 2), Title: cellAt(row, 3), Action: cellAt(row, 4),
			ImpactVND: impact, Evidence: cellAt(row, 6), Status: domain.RecommendationStatus(cellAt(row, 7)),
			SnoozedUntil: cellAt(row, 8), StatusChanged: cellAt(row, 9), Outcome: cellAt(row, 10), Page: cellAt(row, 11),
		}, sheetRow, rowETag(row, recommendationsColumns))
		problems = append(problems, rowProblems...)
		if first, repeated := seen[item.ID]; repeated && len(problems) == 0 {
			problems = append(problems, fmt.Sprintf("id %s repeats row %d", item.ID, first))
		}
		if len(problems) > 0 {
			issues = append(issues, domain.RecommendationIssue{Row: sheetRow, Message: strings.Join(problems, "; ")})
			continue
		}
		seen[item.ID] = sheetRow
		items = append(items, item)
	}
	return items, issues
}
