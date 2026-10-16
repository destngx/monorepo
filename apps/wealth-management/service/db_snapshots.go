package service

import (
	"apps/wealth-management/domain"
	"fmt"
	"strings"
	"time"
)

// Snapshots tab (ROADMAP R8, P3; REQUIREMENTS §2): app-managed, created with
// its header on the first written snapshot. One row per month; writing a
// month that has a row updates that row (A-T, the note in U is kept), so a
// month never gets a second row. No other tab is written from here.

const (
	snapshotsSheet    = "Snapshots"
	snapshotsRange    = "'Snapshots'!A2:U"
	snapshotsAnchor   = "'Snapshots'!A2:A"
	snapshotsFirstRow = 2
	// snapshotsWritten are the columns the engine writes (A-T); U is the
	// user's note.
	snapshotsWritten = 20

	cacheKeySnapshots = "snapshots:rows"
)

var snapshotsHeader = []any{
	"Month", "As of", "Net worth", "Assets", "Liabilities", "Money lent", "Cash", "Funds", "Crypto",
	"Income", "Expense", "Savings", "Savings rate", "Health score", "Savings pillar", "Emergency pillar",
	"Debt pillar", "Diversification pillar", "Emergency months", "Written", "Note",
}

func (s *dbService) ReadSnapshots(forceFresh bool) (domain.SnapshotSheet, error) {
	return cachedRead(s, cacheKeySnapshots, sheetDataTTL, forceFresh, s.readSnapshots)
}

// readSnapshots reads the tab; a missing tab has no rows.
func (s *dbService) readSnapshots() (domain.SnapshotSheet, error) {
	sheet := domain.SnapshotSheet{Snapshots: []domain.MonthSnapshot{}, Issues: []domain.SnapshotIssue{}}
	exists, err := s.hasSheet(snapshotsSheet)
	if err != nil {
		return domain.SnapshotSheet{}, err
	}
	if !exists {
		return sheet, nil
	}
	rows, err := s.client.ReadSheet(snapshotsRange)
	if err != nil {
		return domain.SnapshotSheet{}, fmt.Errorf("read snapshots: %w", err)
	}
	sheet.Exists = true
	sheet.Snapshots, sheet.Issues = parseSnapshots(rows)
	return sheet, nil
}

// SaveSnapshot writes a month's row: the existing row for the month is
// updated, else a row is appended (the tab is created first when missing).
// The tab is re-read under the write lock, so a row added by hand since the
// last read is updated rather than duplicated.
func (s *dbService) SaveSnapshot(snapshot domain.MonthSnapshot) (domain.SnapshotSaveResult, error) {
	if _, err := domain.ParseMonthKey(snapshot.Month); err != nil {
		return domain.SnapshotSaveResult{}, fmt.Errorf("%w: snapshot month must be yyyy-mm", domain.ErrValidation)
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	sheet, err := s.readSnapshots()
	if err != nil {
		return domain.SnapshotSaveResult{}, err
	}
	values := snapshotRowValues(snapshot)
	row := 0
	if existing, ok := sheet.ForMonth(snapshot.Month); ok {
		row = existing.Row
	} else if found := issueRowForMonth(sheet.Issues, snapshot.Month); found > 0 {
		// A broken row for the month is replaced rather than duplicated.
		row = found
	}
	created := row == 0
	if created {
		if !sheet.Exists {
			if err := s.client.CreateSheet(snapshotsSheet, snapshotsHeader); err != nil {
				return domain.SnapshotSaveResult{}, fmt.Errorf("create %s tab: %w", snapshotsSheet, err)
			}
		}
		row, err = s.client.WriteRowsAtFirstEmptyRow(snapshotsSheet, snapshotsAnchor, [][]any{append(values, "")})
		s.invalidate(cacheKeySnapshots)
		if err != nil {
			return domain.SnapshotSaveResult{}, fmt.Errorf("write snapshot %s: %w", snapshot.Month, err)
		}
	} else {
		err = s.client.UpdateRow(fmt.Sprintf("'%s'!A%d:T%d", snapshotsSheet, row, row), values)
		s.invalidate(cacheKeySnapshots)
		if err != nil {
			return domain.SnapshotSaveResult{}, fmt.Errorf("update snapshot %s: %w", snapshot.Month, err)
		}
	}
	saved, err := s.readSnapshotRow(row)
	if err != nil {
		return domain.SnapshotSaveResult{}, err
	}
	return domain.SnapshotSaveResult{Snapshot: saved, Created: created}, nil
}

// issueRowForMonth finds a rejected row whose month cell parsed, so
// SaveSnapshot can repair it.
func issueRowForMonth(issues []domain.SnapshotIssue, month string) int {
	for _, issue := range issues {
		if strings.HasPrefix(issue.Message, "month "+month+":") {
			return issue.Row
		}
	}
	return 0
}

func (s *dbService) readSnapshotRow(row int) (domain.MonthSnapshot, error) {
	rows, err := s.client.ReadSheet(fmt.Sprintf("'%s'!A%d:U%d", snapshotsSheet, row, row))
	if err != nil {
		return domain.MonthSnapshot{}, fmt.Errorf("read snapshot row %d: %w", row, err)
	}
	snapshot, problem, ok := mapSnapshotRow(rowAt(rows, 0), row)
	if problem != "" {
		return domain.MonthSnapshot{}, fmt.Errorf("snapshot row %d did not read back: %s", row, problem)
	}
	if !ok {
		return domain.MonthSnapshot{}, fmt.Errorf("%w: snapshot row %d is empty after the write", domain.ErrNotFound, row)
	}
	return snapshot, nil
}

// snapshotRowValues renders A-T for USER_ENTERED; dates are written as text
// ('yyyy-mm) so Sheets does not turn them into date serials.
func snapshotRowValues(snapshot domain.MonthSnapshot) []any {
	optional := func(value *float64) any {
		if value == nil {
			return ""
		}
		return *value
	}
	date := func(value string) any {
		if value == "" {
			return ""
		}
		return "'" + value
	}
	return []any{
		"'" + snapshot.Month, date(snapshot.AsOf), snapshot.NetWorth,
		optional(snapshot.Assets), optional(snapshot.Liabilities), optional(snapshot.MoneyLent),
		optional(snapshot.Cash), optional(snapshot.Funds), optional(snapshot.Crypto),
		optional(snapshot.Income), optional(snapshot.Expense), optional(snapshot.Savings), optional(snapshot.SavingsRate),
		optional(snapshot.HealthScore), optional(snapshot.SavingsScore), optional(snapshot.EmergencyScore),
		optional(snapshot.DebtScore), optional(snapshot.DiversificationScore), optional(snapshot.EmergencyMonths),
		date(snapshot.WrittenOn),
	}
}

// Row parsing: tolerant of hand edits (REQUIREMENTS §2). The first row for
// a month wins; later ones are reported.
func parseSnapshots(rows [][]string) ([]domain.MonthSnapshot, []domain.SnapshotIssue) {
	snapshots := []domain.MonthSnapshot{}
	issues := []domain.SnapshotIssue{}
	seen := map[string]int{}
	for index, row := range rows {
		sheetRow := snapshotsFirstRow + index
		snapshot, problem, ok := mapSnapshotRow(row, sheetRow)
		switch {
		case problem != "":
			issues = append(issues, domain.SnapshotIssue{Row: sheetRow, Message: problem})
		case ok && seen[snapshot.Month] > 0:
			issues = append(issues, domain.SnapshotIssue{Row: sheetRow, Message: fmt.Sprintf("%s already has row %d; this row is ignored", snapshot.Month, seen[snapshot.Month])})
		case ok:
			seen[snapshot.Month] = sheetRow
			snapshots = append(snapshots, snapshot)
		}
	}
	return snapshots, issues
}

// mapSnapshotRow maps one row. Blank rows and the header are skipped (ok
// false, no problem). Month and net worth are required; the other figures
// may be blank. A problem names the month when it parsed ("month yyyy-mm:
// ..."), so a save can repair the row.
func mapSnapshotRow(row []string, sheetRow int) (domain.MonthSnapshot, string, bool) {
	if strings.TrimSpace(strings.Join(row, "")) == "" {
		return domain.MonthSnapshot{}, "", false
	}
	monthCell := strings.TrimSpace(cellAt(row, 0))
	if strings.EqualFold(monthCell, "month") {
		return domain.MonthSnapshot{}, "", false
	}
	month, err := domain.ParseGoalMonth(monthCell)
	if err != nil || month == "" {
		return domain.MonthSnapshot{}, fmt.Sprintf("month %q must be yyyy-mm", monthCell), false
	}
	snapshot := domain.MonthSnapshot{Month: month, Row: sheetRow, Note: strings.TrimSpace(cellAt(row, 20))}
	var problems []string
	if snapshot.AsOf, err = snapshotDate(cellAt(row, 1)); err != nil {
		problems = append(problems, fmt.Sprintf("as of %q is not a date", strings.TrimSpace(cellAt(row, 1))))
	}
	if snapshot.WrittenOn, err = snapshotDate(cellAt(row, 19)); err != nil {
		problems = append(problems, fmt.Sprintf("written %q is not a date", strings.TrimSpace(cellAt(row, 19))))
	}
	netWorth, ok := parseAmount(cellAt(row, 2))
	switch {
	case !ok:
		problems = append(problems, fmt.Sprintf("net worth %q is not an amount", strings.TrimSpace(cellAt(row, 2))))
	case netWorth == nil:
		problems = append(problems, "net worth is required")
	default:
		snapshot.NetWorth = *netWorth
	}
	fields := []struct {
		name   string
		column int
		target **float64
	}{
		{"assets", 3, &snapshot.Assets}, {"liabilities", 4, &snapshot.Liabilities}, {"money lent", 5, &snapshot.MoneyLent},
		{"cash", 6, &snapshot.Cash}, {"funds", 7, &snapshot.Funds}, {"crypto", 8, &snapshot.Crypto},
		{"income", 9, &snapshot.Income}, {"expense", 10, &snapshot.Expense}, {"savings", 11, &snapshot.Savings},
		{"savings rate", 12, &snapshot.SavingsRate}, {"health score", 13, &snapshot.HealthScore},
		{"savings pillar", 14, &snapshot.SavingsScore}, {"emergency pillar", 15, &snapshot.EmergencyScore},
		{"debt pillar", 16, &snapshot.DebtScore}, {"diversification pillar", 17, &snapshot.DiversificationScore},
		{"emergency months", 18, &snapshot.EmergencyMonths},
	}
	for _, field := range fields {
		value, ok := parseAmount(cellAt(row, field.column))
		if !ok {
			problems = append(problems, fmt.Sprintf("%s %q is not a number", field.name, strings.TrimSpace(cellAt(row, field.column))))
			continue
		}
		*field.target = value
	}
	if len(problems) > 0 {
		return domain.MonthSnapshot{}, fmt.Sprintf("month %s: %s", month, strings.Join(problems, "; ")), false
	}
	return snapshot, "", true
}

// snapshotDate reads a yyyy-mm-dd text, a date serial or a sheet date;
// blank stays blank.
func snapshotDate(cell string) (string, error) {
	trimmed := strings.TrimSpace(cell)
	if trimmed == "" {
		return "", nil
	}
	if parsed, err := time.Parse(time.DateOnly, trimmed); err == nil {
		return parsed.Format(time.DateOnly), nil
	}
	parsed, err := domain.ParseSheetDate(trimmed)
	if err != nil {
		return "", err
	}
	return parsed.Format(time.DateOnly), nil
}
