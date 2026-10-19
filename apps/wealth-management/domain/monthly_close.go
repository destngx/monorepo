package domain

import (
	"fmt"
	"strings"
	"time"
)

// Monthly close (ROADMAP R8, WM-CLOSE-1..4): a short checklist for a month,
// each item's state derived from the data, so nothing is ticked by hand and
// no tab is needed for it.

// CloseGraceDays: during the first days of a month the close is for last
// month; afterwards it is for this month.
const CloseGraceDays = 10

// LendingChaseDays: money lent open this long is worth chasing.
const LendingChaseDays = 30

// CloseItemKey names a checklist item.
type CloseItemKey string

const (
	CloseDrafts          CloseItemKey = "drafts"
	CloseCategorize      CloseItemKey = "categorize"
	CloseBudget          CloseItemKey = "budget"
	CloseGoals           CloseItemKey = "goals"
	CloseMoneyLent       CloseItemKey = "moneyLent"
	CloseRecommendations CloseItemKey = "recommendations"
	CloseSnapshot        CloseItemKey = "snapshot"
)

// CloseItemState is derived from the data: done, action (something to do)
// or unavailable (its data could not be read; Detail says why).
type CloseItemState string

const (
	CloseDone        CloseItemState = "done"
	CloseAction      CloseItemState = "action"
	CloseUnavailable CloseItemState = "unavailable"
)

// CloseItem is one checklist line. Detail carries no amounts (the UI masks
// money); Count, Amount and Target hold the figures when they apply:
// drafts and categorize count rows (Amount = their total), budget counts the
// categories over their limit (Amount = over by, Target = total limit),
// goals counts goals behind (Amount = saved this month, Target = required a
// month), money lent counts people to chase (Amount = what they owe) and
// recommendations counts the open ones (Amount = their estimated impact a
// year).
type CloseItem struct {
	Key    CloseItemKey   `json:"key"`
	Title  string         `json:"title"`
	State  CloseItemState `json:"state"`
	Detail string         `json:"detail"`
	Count  *int           `json:"count"`
	Amount *float64       `json:"amount"`
	Target *float64       `json:"target"`
}

// MonthlyClose is the checklist for a month: Done of Checked items are done.
type MonthlyClose struct {
	Month string `json:"month"`
	AsOf  string `json:"asOf"`
	// MonthEnded is false while the month is still running.
	MonthEnded bool `json:"monthEnded"`
	// CanWriteSnapshot is true for this month and last month.
	CanWriteSnapshot bool           `json:"canWriteSnapshot"`
	Items            []CloseItem    `json:"items"`
	Done             int            `json:"done"`
	Checked          int            `json:"checked"`
	Snapshot         *MonthSnapshot `json:"snapshot"`
}

// CloseInputs are the data behind the checklist. A nil optional part with
// its Issue set makes that item unavailable.
type CloseInputs struct {
	Month        time.Time
	Today        time.Time
	Transactions []Transaction
	Snapshots    SnapshotSheet
	// PendingDrafts is the number of capture drafts waiting for approval.
	PendingDrafts  *int
	DraftsIssue    string
	Budget         *BudgetReport
	BudgetIssue    string
	Goals          *GoalPlan
	GoalsIssue     string
	MoneyLent      *MoneyLent
	MoneyLentIssue string
	// Recommendations is the recommendations list's summary (ROADMAP R7).
	Recommendations      *RecommendationSummary
	RecommendationsIssue string
}

// CloseMonthFor is the month a close is for by default: last month during
// the first CloseGraceDays days, else this month.
func CloseMonthFor(today time.Time) time.Time {
	current := firstOfMonthOf(today)
	if today.Day() <= CloseGraceDays {
		return current.AddDate(0, -1, 0)
	}
	return current
}

// ParseCloseMonth parses a yyyy-mm month up to this month; blank means
// CloseMonthFor(today).
func ParseCloseMonth(value string, today time.Time) (time.Time, error) {
	return closeMonth(value, today)
}

func closeMonth(value string, today time.Time) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return CloseMonthFor(today), nil
	}
	month, err := ParseMonthKey(value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: month must be yyyy-mm", ErrValidation)
	}
	if month.After(firstOfMonthOf(today)) {
		return time.Time{}, fmt.Errorf("%w: month %s has not started", ErrValidation, MonthKey(month))
	}
	return month, nil
}

// ComputeMonthlyClose derives every item's state.
func ComputeMonthlyClose(in CloseInputs) MonthlyClose {
	month := firstOfMonthOf(in.Month)
	asOf := SnapshotAsOf(month, in.Today)
	current := firstOfMonthOf(in.Today)
	result := MonthlyClose{
		Month: MonthKey(month), AsOf: asOf.Format(time.DateOnly),
		MonthEnded:       month.Before(current),
		CanWriteSnapshot: !month.Before(current.AddDate(0, -SnapshotMonthsBack, 0)),
		Items:            []CloseItem{},
	}
	if snapshot, ok := in.Snapshots.ForMonth(result.Month); ok {
		result.Snapshot = &snapshot
	}
	result.Items = append(result.Items,
		draftsItem(in),
		categorizeItem(in.Transactions, month),
		budgetItem(in),
		goalsItem(in, month, asOf),
		moneyLentItem(in),
		recommendationsItem(in),
		snapshotItem(result, month),
	)
	result.Checked = len(result.Items)
	for _, item := range result.Items {
		if item.State == CloseDone {
			result.Done++
		}
	}
	return result
}

func unavailableItem(key CloseItemKey, title string, issue string) CloseItem {
	if issue == "" {
		issue = "not available"
	}
	return CloseItem{Key: key, Title: title, State: CloseUnavailable, Detail: "Could not be checked: " + issue}
}

func draftsItem(in CloseInputs) CloseItem {
	const title = "Approve capture drafts"
	if in.PendingDrafts == nil {
		return unavailableItem(CloseDrafts, title, in.DraftsIssue)
	}
	count := *in.PendingDrafts
	item := CloseItem{Key: CloseDrafts, Title: title, Count: &count}
	if count == 0 {
		item.State, item.Detail = CloseDone, "No drafts waiting."
		return item
	}
	item.State, item.Detail = CloseAction, fmt.Sprintf("%s waiting for approval.", plural(count, "draft", "drafts"))
	return item
}

// categorizeItem counts the month's rows still Uncategorized or in a
// category the Categories tab does not know.
func categorizeItem(transactions []Transaction, month time.Time) CloseItem {
	count, amount := 0, 0.0
	for _, t := range transactions {
		if !t.Usable() || !SameMonth(t.Date, month) || !NeedsCategory(t) {
			continue
		}
		count++
		amount += t.PaymentAmount() + t.DepositAmount()
	}
	item := CloseItem{Key: CloseCategorize, Title: "Categorize transactions", Count: &count, Amount: &amount}
	if count == 0 {
		item.State, item.Detail = CloseDone, "Every transaction this month has a category."
		return item
	}
	item.State, item.Detail = CloseAction, fmt.Sprintf("%s without a category.", plural(count, "transaction", "transactions"))
	return item
}

// NeedsCategory: Uncategorized (or blank), or a category the Categories tab
// does not list.
func NeedsCategory(t Transaction) bool {
	category := strings.TrimSpace(t.Category)
	return category == "" || strings.EqualFold(category, "Uncategorized") || t.CategoryType == nil
}

func budgetItem(in CloseInputs) CloseItem {
	const title = "Budget vs actual"
	if in.Budget == nil {
		return unavailableItem(CloseBudget, title, in.BudgetIssue)
	}
	over, overBy, unbudgeted := 0, 0.0, 0
	for _, line := range in.Budget.Lines {
		if !line.CountsAsExpense() {
			continue
		}
		switch {
		case line.Unbudgeted && line.Spent > 0:
			unbudgeted++
		case line.Limit > 0 && line.Spent > line.Limit:
			over++
			overBy += line.Spent - line.Limit
		}
	}
	limit := in.Budget.TotalLimit
	item := CloseItem{Key: CloseBudget, Title: title, Count: &over, Amount: &overBy, Target: &limit}
	extra := ""
	if unbudgeted > 0 {
		extra = fmt.Sprintf(" %s spending without a budget.", plural(unbudgeted, "category has", "categories have"))
	}
	if over == 0 {
		item.State, item.Detail = CloseDone, "Every category is within its limit."+extra
		return item
	}
	item.State = CloseAction
	item.Detail = fmt.Sprintf("%s over the limit.%s", plural(over, "category", "categories"), extra)
	return item
}

// goalsItem compares the month's savings with what the goal plan needs a
// month.
func goalsItem(in CloseInputs, month time.Time, asOf time.Time) CloseItem {
	const title = "Surplus to goals"
	if in.Goals == nil {
		return unavailableItem(CloseGoals, title, in.GoalsIssue)
	}
	if in.Goals.SetupNeeded {
		return CloseItem{Key: CloseGoals, Title: title, State: CloseAction, Detail: "Set up the goal plan so the surplus has somewhere to go."}
	}
	saved := CashFlowToDay(in.Transactions, month, asOf.Day()).CashFlow
	required := in.Goals.Check.RequiredPerMonth
	behind := 0
	for _, goal := range in.Goals.Goals {
		if goal.Pace == PaceBehind {
			behind++
		}
	}
	item := CloseItem{Key: CloseGoals, Title: title, Amount: &saved, Target: &required, Count: &behind}
	covered := saved >= required
	switch {
	case covered && behind == 0:
		item.State, item.Detail = CloseDone, "This month's savings cover what the goals need a month."
	case covered:
		item.State, item.Detail = CloseAction, fmt.Sprintf("Savings cover the monthly need, but %s behind.", plural(behind, "goal is", "goals are"))
	default:
		item.State, item.Detail = CloseAction, "This month's savings are short of what the goals need a month."
		if behind > 0 {
			item.Detail += fmt.Sprintf(" %s behind.", plural(behind, "goal is", "goals are"))
		}
	}
	return item
}

func moneyLentItem(in CloseInputs) CloseItem {
	const title = "Chase money lent"
	if in.MoneyLent == nil {
		return unavailableItem(CloseMoneyLent, title, in.MoneyLentIssue)
	}
	count, amount := 0, 0.0
	for _, person := range in.MoneyLent.People {
		if person.Outstanding > settledTolerance && closeChase(person) {
			count++
			amount += person.Outstanding
		}
	}
	item := CloseItem{Key: CloseMoneyLent, Title: title, Count: &count, Amount: &amount}
	switch {
	case count > 0:
		item.State = CloseAction
		item.Detail = fmt.Sprintf("%s owed you money for %d days or more, or past or close to the repayment date in the memo.", plural(count, "person has", "people have"), LendingChaseDays)
	case in.MoneyLent.OwedToYou > settledTolerance:
		item.State = CloseDone
		item.Detail = fmt.Sprintf("Money owed to you was lent less than %d days ago or is not due back yet.", LendingChaseDays)
	default:
		item.State, item.Detail = CloseDone, "Nobody owes you money."
	}
	return item
}

// closeChase: the repayment date in the memo is close or has passed, or,
// with no date, the balance is LendingChaseDays old (WM-LEND-6).
func closeChase(person MoneyLentPerson) bool {
	if person.RepayBy != nil {
		return person.RepaymentDue()
	}
	return person.DaysOutstanding != nil && *person.DaysOutstanding >= LendingChaseDays
}

func recommendationsItem(in CloseInputs) CloseItem {
	const title = "Review recommendations"
	if in.Recommendations == nil {
		return unavailableItem(CloseRecommendations, title, in.RecommendationsIssue)
	}
	open := in.Recommendations.Open
	impact := in.Recommendations.OpenImpactVND
	item := CloseItem{Key: CloseRecommendations, Title: title, Count: &open, Amount: &impact}
	snoozed := ""
	if in.Recommendations.Snoozed > 0 {
		snoozed = fmt.Sprintf(" %d snoozed.", in.Recommendations.Snoozed)
	}
	if open == 0 {
		item.State, item.Detail = CloseDone, "No open recommendations."+snoozed
		return item
	}
	item.State = CloseAction
	item.Detail = fmt.Sprintf("%s to act on, dismiss or snooze.%s", plural(open, "open recommendation", "open recommendations"), snoozed)
	return item
}

func snapshotItem(close MonthlyClose, month time.Time) CloseItem {
	item := CloseItem{Key: CloseSnapshot, Title: "Write the snapshot"}
	monthEnd := month.AddDate(0, 1, -1).Format(time.DateOnly)
	switch snapshot := close.Snapshot; {
	case snapshot != nil && snapshot.AsOf >= monthEnd:
		item.State, item.Detail = CloseDone, fmt.Sprintf("Written for %s.", snapshot.AsOf)
	case snapshot != nil && !close.MonthEnded:
		item.State = CloseDone
		item.Detail = fmt.Sprintf("Written for %s; write it again at the month end to update the row.", snapshot.AsOf)
	case snapshot != nil:
		item.State = CloseAction
		item.Detail = fmt.Sprintf("Written for %s, before the month ended; write it again to update the row.", snapshot.AsOf)
	case !close.CanWriteSnapshot:
		item.State, item.Detail = CloseUnavailable, "Snapshots are written for this month or last month; type older months into the Snapshots tab by hand."
	default:
		item.State, item.Detail = CloseAction, "Record this month's net worth, balances and health score in the Snapshots tab."
	}
	if item.State == CloseAction && !close.CanWriteSnapshot {
		item.State = CloseUnavailable
	}
	return item
}

func plural(count int, one string, many string) string {
	if count == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", count, many)
}
