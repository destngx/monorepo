package service

import (
	"apps/wealth-management/domain"
	"fmt"
	"math"
	"strings"
)

// Recurring tab (ROADMAP R4, REQUIREMENTS §2): app-managed, created with its
// header on the first saved item. The engine writes only the rows the user
// changes in the app and never any other tab from here.

const (
	recurringSheet    = "Recurring"
	recurringRange    = "'Recurring'!A2:K"
	recurringAnchor   = "'Recurring'!A2:A"
	recurringFirstRow = 2
	recurringColumns  = 11 // A-K

	cacheKeyRecurring        = "recurring:items"
	cacheKeyRecurringPattern = "recurring:*"

	dismissedSuggestionNote = "Dismissed suggestion"
)

var recurringHeader = []any{"Name", "Kind", "Amount", "Frequency", "Day", "Month", "Account", "Category", "Match", "Status", "Note"}

func (s *dbService) ReadRecurring(forceFresh bool) (domain.RecurringSheet, error) {
	return cachedRead(s, cacheKeyRecurring, sheetDataTTL, forceFresh, s.readRecurring)
}

// readRecurring reads the tab; a missing tab has no items.
func (s *dbService) readRecurring() (domain.RecurringSheet, error) {
	sheet := domain.RecurringSheet{Items: []domain.RecurringItem{}, Issues: []domain.RecurringIssue{}}
	exists, err := s.hasSheet(recurringSheet)
	if err != nil {
		return domain.RecurringSheet{}, err
	}
	if !exists {
		return sheet, nil
	}
	rows, err := s.client.ReadSheet(recurringRange)
	if err != nil {
		return domain.RecurringSheet{}, fmt.Errorf("read recurring items: %w", err)
	}
	sheet.Exists = true
	sheet.Items, sheet.Issues = parseRecurring(rows)
	return sheet, nil
}

func (s *dbService) CreateRecurring(input domain.RecurringInput) (domain.RecurringItem, error) {
	if err := input.Validate(); err != nil {
		return domain.RecurringItem{}, err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.appendRecurring(input.Normalized())
}

// appendRecurring writes a validated item below the last row, creating the
// tab first when it is missing. Callers hold writeMu.
func (s *dbService) appendRecurring(input domain.RecurringInput) (domain.RecurringItem, error) {
	exists, err := s.hasSheet(recurringSheet)
	if err != nil {
		return domain.RecurringItem{}, err
	}
	if !exists {
		if err := s.client.CreateSheet(recurringSheet, recurringHeader); err != nil {
			return domain.RecurringItem{}, fmt.Errorf("create %s tab: %w", recurringSheet, err)
		}
	}
	row, err := s.client.WriteRowsAtFirstEmptyRow(recurringSheet, recurringAnchor, [][]any{recurringRowValues(input)})
	s.invalidate(cacheKeyRecurringPattern)
	if err != nil {
		return domain.RecurringItem{}, fmt.Errorf("write recurring item: %w", err)
	}
	return s.readRecurringRow(row)
}

// ConfirmRecurringSuggestion writes the suggestion with the given key,
// re-detected from the ledger and the tab under the write lock, with the
// overrides applied. ErrNotFound when the key is no longer suggested (for
// example, confirmed meanwhile).
func (s *dbService) ConfirmRecurringSuggestion(input domain.RecurringConfirmInput) (domain.RecurringItem, error) {
	if err := input.Validate(); err != nil {
		return domain.RecurringItem{}, err
	}
	transactions, err := s.ListTransactions(false)
	if err != nil {
		return domain.RecurringItem{}, err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	sheet, err := s.readRecurring()
	if err != nil {
		return domain.RecurringItem{}, err
	}
	for _, suggestion := range domain.DetectRecurring(transactions, sheet.Items, s.today()) {
		if suggestion.Key != strings.TrimSpace(input.Key) {
			continue
		}
		item := input.Apply(suggestion)
		if item.Status == domain.RecurringPaused && item.Note == "" {
			item.Note = dismissedSuggestionNote
		}
		if err := item.Validate(); err != nil {
			return domain.RecurringItem{}, err
		}
		return s.appendRecurring(item)
	}
	return domain.RecurringItem{}, fmt.Errorf("%w: %q is not a current suggestion; reload and retry", domain.ErrNotFound, input.Key)
}

// UpdateRecurring replaces columns A-K of a row that still matches the ETag.
func (s *dbService) UpdateRecurring(id string, input domain.RecurringUpdateInput) (domain.RecurringItem, error) {
	if err := input.Validate(); err != nil {
		return domain.RecurringItem{}, err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	row, _, err := s.checkRecurringETag(id, input.ETag)
	if err != nil {
		return domain.RecurringItem{}, err
	}
	if err := s.client.UpdateRow(recurringRowRange(row), recurringRowValues(input.RecurringInput.Normalized())); err != nil {
		s.invalidate(cacheKeyRecurringPattern)
		return domain.RecurringItem{}, fmt.Errorf("update recurring item: %w", err)
	}
	s.invalidate(cacheKeyRecurringPattern)
	return s.readRecurringRow(row)
}

// SetRecurringStatus pauses or resumes an item: only column J is written.
func (s *dbService) SetRecurringStatus(id string, input domain.RecurringStatusInput) (domain.RecurringItem, error) {
	if err := input.Validate(); err != nil {
		return domain.RecurringItem{}, err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	row, _, err := s.checkRecurringETag(id, input.ETag)
	if err != nil {
		return domain.RecurringItem{}, err
	}
	if err := s.client.UpdateRow(fmt.Sprintf("'%s'!J%d", recurringSheet, row), []any{string(input.Status)}); err != nil {
		s.invalidate(cacheKeyRecurringPattern)
		return domain.RecurringItem{}, fmt.Errorf("update recurring status: %w", err)
	}
	s.invalidate(cacheKeyRecurringPattern)
	return s.readRecurringRow(row)
}

// DeleteRecurring clears columns A-K; other item IDs do not shift.
func (s *dbService) DeleteRecurring(id string, etag string) error {
	if err := requireETag(etag); err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	row, _, err := s.checkRecurringETag(id, etag)
	if err != nil {
		return err
	}
	blank := make([]any, recurringColumns)
	for i := range blank {
		blank[i] = ""
	}
	if err := s.client.UpdateRow(recurringRowRange(row), blank); err != nil {
		s.invalidate(cacheKeyRecurringPattern)
		return fmt.Errorf("delete recurring item: %w", err)
	}
	s.invalidate(cacheKeyRecurringPattern)
	return nil
}

func requireETag(etag string) error {
	if strings.TrimSpace(etag) == "" {
		return fmt.Errorf("%w: etag is required", domain.ErrValidation)
	}
	return nil
}

// checkRecurringETag re-reads the row and returns it when it exists and is
// unchanged. A row the reader rejects can still be replaced or cleared.
// Callers hold writeMu.
func (s *dbService) checkRecurringETag(id string, etag string) (int, []string, error) {
	row, err := parseRecurringID(id)
	if err != nil {
		return 0, nil, err
	}
	exists, err := s.hasSheet(recurringSheet)
	if err != nil {
		return 0, nil, err
	}
	if !exists {
		return 0, nil, fmt.Errorf("%w: recurring item %s (the %s tab does not exist yet)", domain.ErrNotFound, id, recurringSheet)
	}
	rows, err := s.client.ReadSheet(recurringRowRange(row))
	if err != nil {
		return 0, nil, fmt.Errorf("read recurring item %s: %w", id, err)
	}
	current := rowAt(rows, 0)
	if strings.TrimSpace(strings.Join(current, "")) == "" {
		return 0, nil, fmt.Errorf("%w: recurring item %s", domain.ErrNotFound, id)
	}
	if rowETag(current, recurringColumns) != strings.TrimSpace(etag) {
		return 0, nil, fmt.Errorf("%w: recurring item %s changed since it was read; reload and retry", domain.ErrConflict, id)
	}
	return row, current, nil
}

// readRecurringRow reads one row back so the caller gets the sheet's own
// values and a matching ETag.
func (s *dbService) readRecurringRow(row int) (domain.RecurringItem, error) {
	rows, err := s.client.ReadSheet(recurringRowRange(row))
	if err != nil {
		return domain.RecurringItem{}, fmt.Errorf("read recurring item %s: %w", recurringID(row), err)
	}
	item, problem, ok := mapRecurringRow(rowAt(rows, 0), row)
	if problem != "" {
		return domain.RecurringItem{}, fmt.Errorf("%w: recurring item %s: %s", domain.ErrValidation, recurringID(row), problem)
	}
	if !ok {
		return domain.RecurringItem{}, fmt.Errorf("%w: recurring item %s", domain.ErrNotFound, recurringID(row))
	}
	return item, nil
}

// recurringRowValues renders columns A-K for USER_ENTERED. Text that Sheets
// would read as a formula is written as plain text.
func recurringRowValues(input domain.RecurringInput) []any {
	amount, day, month := any(""), any(""), any("")
	if input.Amount != nil {
		amount = *input.Amount
	}
	if input.Day > 0 {
		day = input.Day
	}
	if input.Month > 0 {
		month = input.Month
	}
	return []any{
		textCell(input.Name), string(input.Kind), amount, string(input.Frequency), day, month,
		textCell(input.Account), textCell(input.Category), textCell(input.Match), string(input.Status), textCell(input.Note),
	}
}

// textCell keeps user text from being read as a formula.
func textCell(value string) string {
	if value != "" && strings.ContainsAny(value[:1], "=+-@") {
		return "'" + value
	}
	return value
}

// Row parsing: tolerant of hand edits (REQUIREMENTS §2).

func parseRecurring(rows [][]string) ([]domain.RecurringItem, []domain.RecurringIssue) {
	items := []domain.RecurringItem{}
	issues := []domain.RecurringIssue{}
	for index, row := range rows {
		sheetRow := recurringFirstRow + index
		item, problem, ok := mapRecurringRow(row, sheetRow)
		switch {
		case problem != "":
			issues = append(issues, domain.RecurringIssue{Row: sheetRow, Message: problem})
		case ok:
			items = append(items, item)
		}
	}
	return items, issues
}

// mapRecurringRow maps one row. Blank rows and a header row are skipped
// (ok false, no problem); an invalid row returns its problems.
func mapRecurringRow(row []string, sheetRow int) (domain.RecurringItem, string, bool) {
	if strings.TrimSpace(strings.Join(row, "")) == "" {
		return domain.RecurringItem{}, "", false
	}
	name := strings.TrimSpace(cellAt(row, 0))
	if strings.EqualFold(name, "name") && strings.EqualFold(strings.TrimSpace(cellAt(row, 1)), "kind") {
		return domain.RecurringItem{}, "", false
	}
	var problems []string
	amount, ok := parseAmount(cellAt(row, 2))
	if !ok {
		problems = append(problems, fmt.Sprintf("amount %q is not an amount", strings.TrimSpace(cellAt(row, 2))))
	}
	day, dayProblem := wholeCell(cellAt(row, 4), "day")
	month, monthProblem := wholeCell(cellAt(row, 5), "month")
	problems = appendIf(appendIf(problems, dayProblem), monthProblem)
	input := domain.RecurringInput{
		Name: name, Kind: domain.RecurringKind(cellAt(row, 1)), Amount: amount, Frequency: domain.RecurringFrequency(cellAt(row, 3)),
		Day: day, Month: month, Account: cellAt(row, 6), Category: cellAt(row, 7), Match: cellAt(row, 8),
		Status: domain.RecurringStatus(cellAt(row, 9)), Note: cellAt(row, 10),
	}.Normalized()
	if !ok && input.Kind != domain.RecurringCard {
		placeholder := 1.0 // the amount problem is already reported
		input.Amount = &placeholder
	}
	problems = append(problems, input.Problems()...)
	if len(problems) > 0 {
		return domain.RecurringItem{}, strings.Join(problems, "; "), false
	}
	return domain.RecurringItem{
		ID: recurringID(sheetRow), Row: sheetRow, Name: input.Name, Kind: input.Kind, Amount: input.Amount,
		Frequency: input.Frequency, Day: input.Day, Month: input.Month, Account: input.Account, Category: input.Category,
		Match: input.Match, Status: input.Status, Note: input.Note, ETag: rowETag(row, recurringColumns),
	}, "", true
}

// wholeCell reads a whole-number cell; blank is 0.
func wholeCell(cell string, field string) (int, string) {
	value, ok := parseAmount(cell)
	if !ok || (value != nil && (*value < 0 || *value != math.Trunc(*value))) {
		return 0, fmt.Sprintf("%s %q must be a whole number", field, strings.TrimSpace(cell))
	}
	if value == nil {
		return 0, ""
	}
	return int(*value), ""
}

func appendIf(problems []string, problem string) []string {
	if problem != "" {
		return append(problems, problem)
	}
	return problems
}

func recurringID(row int) string {
	return fmt.Sprintf("recurring-%d", row)
}

func parseRecurringID(id string) (int, error) {
	var row int
	trimmed := strings.TrimSpace(id)
	if _, err := fmt.Sscanf(trimmed, "recurring-%d", &row); err != nil || row < recurringFirstRow || recurringID(row) != trimmed {
		return 0, fmt.Errorf("%w: recurring item id %q must look like recurring-N", domain.ErrValidation, id)
	}
	return row, nil
}

func recurringRowRange(row int) string {
	return fmt.Sprintf("'%s'!A%d:K%d", recurringSheet, row, row)
}
