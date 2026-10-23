package domain

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Goal Plan (D31, D32): goals are claims on a pooled savings balance, filled
// in priority order. The plan lives in the app-managed `Goal Plan` tab
// (REQUIREMENTS §2); the old `Goals` tab is read once, for the import.

// GoalStatus is the Goal Plan status column (F).
type GoalStatus string

const (
	GoalActive GoalStatus = "active"
	GoalPaused GoalStatus = "paused"
	GoalDone   GoalStatus = "done"
)

// GoalStatuses lists every status a Goal Plan row can have.
var GoalStatuses = []GoalStatus{GoalActive, GoalPaused, GoalDone}

// ParseGoalStatus reads column F ignoring case and spaces; blank is active.
func ParseGoalStatus(cell string) (GoalStatus, error) {
	status := GoalStatus(strings.ToLower(strings.TrimSpace(cell)))
	if status == "" {
		return GoalActive, nil
	}
	if !slices.Contains(GoalStatuses, status) {
		return "", fmt.Errorf("%w: status %q must be one of active, paused or done", ErrValidation, strings.TrimSpace(cell))
	}
	return status, nil
}

// PlannedGoal is one row of the Goal Plan tab. Target is nil only on the
// emergency fund row, whose target is then computed from monthly needs.
type PlannedGoal struct {
	// ID is "goal-<sheet row>"; with ETag it addresses the row for writes.
	ID     string   `json:"id"`
	Row    int      `json:"row"`
	Name   string   `json:"name"`
	Target *float64 `json:"target"`
	// TargetDate is "yyyy-mm"; empty means no deadline.
	TargetDate     string     `json:"targetDate,omitempty"`
	Priority       int        `json:"priority"`
	StartingAmount float64    `json:"startingAmount"`
	Status         GoalStatus `json:"status"`
	Note           string     `json:"note,omitempty"`
	// UseSlowMoney lets the goal draw on slow money (column H); off by
	// default.
	UseSlowMoney bool `json:"useSlowMoney"`
	// ETag fingerprints columns A-H as read, so edits detect a changed row.
	ETag string `json:"etag"`
}

// GoalPlanIssue is a Goal Plan row the reader rejected, with the reason.
type GoalPlanIssue struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

// GoalPlanSheet is the Goal Plan tab as read. When the tab does not exist
// yet, Exists is false and LegacyGoals counts the old `Goals` rows a setup
// would import.
type GoalPlanSheet struct {
	Exists      bool            `json:"exists"`
	Goals       []PlannedGoal   `json:"goals"`
	Issues      []GoalPlanIssue `json:"issues"`
	LegacyGoals int             `json:"legacyGoals"`
}

// EmergencyFundKeywords identify the emergency fund row: a goal whose name
// or note contains one of them, compared after FoldText (case and
// Vietnamese accents ignored). The first such active row by priority is the
// emergency fund; its target may be blank (computed).
var EmergencyFundKeywords = []string{"emergency", "khan cap", "du phong"}

// IsEmergencyFund applies the emergency fund rule to a name and note.
func IsEmergencyFund(name string, note string) bool {
	folded := " " + FoldText(name) + " " + FoldText(note) + " "
	for _, keyword := range EmergencyFundKeywords {
		if strings.Contains(folded, " "+keyword) {
			return true
		}
	}
	return false
}

var (
	goalYearMonth = regexp.MustCompile(`^(\d{4})-(\d{1,2})$`)
	goalMonthYear = regexp.MustCompile(`^(\d{1,2})/(\d{4})$`)
)

// ParseGoalMonth reads a target date cell as "yyyy-mm". Sheets turns a
// typed "2030-06" into a date serial, so serials are accepted, as are
// "mm/yyyy" and full dates. Blank means no deadline.
func ParseGoalMonth(cell string) (string, error) {
	trimmed := strings.TrimSpace(cell)
	if trimmed == "" {
		return "", nil
	}
	year, month := 0, 0
	if match := goalYearMonth.FindStringSubmatch(trimmed); match != nil {
		year, _ = strconv.Atoi(match[1])
		month, _ = strconv.Atoi(match[2])
	} else if match := goalMonthYear.FindStringSubmatch(trimmed); match != nil {
		month, _ = strconv.Atoi(match[1])
		year, _ = strconv.Atoi(match[2])
	} else if date, err := ParseSheetDate(trimmed); err == nil {
		year, month = date.Year(), int(date.Month())
	}
	if year < 1900 || month < 1 || month > 12 {
		return "", fmt.Errorf("%w: target date %q must be yyyy-mm", ErrValidation, trimmed)
	}
	return fmt.Sprintf("%04d-%02d", year, month), nil
}

// monthsBetween counts calendar months from a to b (b later is positive).
func monthsBetween(a time.Time, b time.Time) int {
	return (b.Year()-a.Year())*12 + int(b.Month()) - int(a.Month())
}

// GoalInput creates or replaces a Goal Plan row. On create a zero Priority
// puts the goal last and an empty Status is active.
type GoalInput struct {
	Name           string     `json:"name"`
	Target         *float64   `json:"target,omitempty"`
	TargetDate     string     `json:"targetDate,omitempty"`
	Priority       int        `json:"priority,omitempty"`
	StartingAmount *float64   `json:"startingAmount,omitempty"`
	Status         GoalStatus `json:"status,omitempty"`
	Note           string     `json:"note,omitempty"`
	UseSlowMoney   bool       `json:"useSlowMoney,omitempty"`
}

func (input GoalInput) Validate() error {
	var problems []string
	name := strings.TrimSpace(input.Name)
	if name == "" {
		problems = append(problems, "name is required")
	}
	switch {
	case input.Target != nil && *input.Target <= 0:
		problems = append(problems, "target must be positive")
	case input.Target == nil && name != "" && !IsEmergencyFund(name, input.Note):
		problems = append(problems, "target is required (only the emergency fund may leave it blank)")
	}
	if _, err := ParseGoalMonth(input.TargetDate); err != nil {
		problems = append(problems, fmt.Sprintf("targetDate %q must be yyyy-mm", input.TargetDate))
	}
	if input.Priority < 0 {
		problems = append(problems, "priority must be 1 or more")
	}
	if input.StartingAmount != nil && *input.StartingAmount < 0 {
		problems = append(problems, "startingAmount must not be negative")
	}
	if input.Status != "" && !slices.Contains(GoalStatuses, input.Status) {
		problems = append(problems, fmt.Sprintf("status %q must be one of active, paused or done", input.Status))
	}
	return validationError(problems)
}

// GoalUpdateInput replaces every column of a goal when its row still
// matches ETag.
type GoalUpdateInput struct {
	GoalInput
	ETag string `json:"etag"`
}

func (input GoalUpdateInput) Validate() error {
	if err := input.GoalInput.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(input.ETag) == "" {
		return fmt.Errorf("%w: etag is required", ErrValidation)
	}
	return nil
}

// GoalReorderInput lists goal IDs in the new priority order (first =
// priority 1); goals not listed follow in their current order.
type GoalReorderInput struct {
	IDs []string `json:"ids"`
}

func (input GoalReorderInput) Validate() error {
	if len(input.IDs) == 0 {
		return fmt.Errorf("%w: ids must list the goals in the new order", ErrValidation)
	}
	seen := map[string]bool{}
	for _, id := range input.IDs {
		key := strings.TrimSpace(id)
		if key == "" || seen[key] {
			return fmt.Errorf("%w: ids must be unique and not blank", ErrValidation)
		}
		seen[key] = true
	}
	return nil
}

// GoalPlanSetupResult reports the first-use setup: rows written, how many
// came from the old Goals tab, and the old rows left out (an emergency fund
// row is created by the setup itself).
type GoalPlanSetupResult struct {
	Created  int      `json:"created"`
	Imported int      `json:"imported"`
	Skipped  []string `json:"skipped"`
}
