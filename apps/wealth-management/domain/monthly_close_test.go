package domain

import (
	"errors"
	"testing"
)

func closeItem(close MonthlyClose, key CloseItemKey) CloseItem {
	for _, item := range close.Items {
		if item.Key == key {
			return item
		}
	}
	return CloseItem{}
}

// closeInputs: closing October on 2026-11-05 with 2 drafts waiting, one
// uncategorized row, Food 3M over its limit, a goal plan needing 15M a
// month (October saved 12M), 1M lent 16 days ago and two open
// recommendations (one snoozed).
func closeInputs() CloseInputs {
	_, transactions := snapshotLedger()
	uncategorized := txn(day(2026, 10, 12), "Uncategorized", CategoryTypeExpense, 500_000, 0)
	transactions = append(transactions, uncategorized, Transaction{AccountName: "Cash", Date: day(2026, 10, 13), Category: "Unknown", Payment: money(100_000)})
	drafts := 2
	days := 16
	return CloseInputs{
		Month: day(2026, 10, 1), Today: day(2026, 11, 5), Transactions: transactions,
		PendingDrafts: &drafts,
		Budget: &BudgetReport{TotalLimit: 20_000_000, TotalSpent: 18_500_000, Lines: []BudgetLine{
			{Category: "Food", CategoryType: CategoryTypeExpense, Limit: 15_000_000, Spent: 18_000_000},
			{Category: "Fun", CategoryType: CategoryTypeExpense, Limit: 5_000_000, Spent: 500_000},
			{Category: "Gifts", CategoryType: CategoryTypeExpense, Spent: 200_000, Unbudgeted: true},
		}},
		Goals:           &GoalPlan{Check: PlanCheck{RequiredPerMonth: 15_000_000}, Goals: []GoalProgress{{Pace: PaceBehind}, {Pace: PaceOnTrack}}},
		Recommendations: &RecommendationSummary{Open: 2, OpenImpactVND: 5_000_000, Snoozed: 1},
		MoneyLent: &MoneyLent{MoneyLentFacts: MoneyLentFacts{OwedToYou: 1_000_000}, People: []MoneyLentPerson{
			lentPerson("Person A", 1_000_000, days, "", 0),
		}},
	}
}

func TestGivenAnOpenMonthWhenComputingTheCloseThenEachItemNeedsItsAction(t *testing.T) {
	close := ComputeMonthlyClose(closeInputs())

	if close.Month != "2026-10" || close.AsOf != "2026-10-31" || !close.MonthEnded || !close.CanWriteSnapshot {
		t.Fatalf("close = %+v", close)
	}
	keys := []CloseItemKey{CloseDrafts, CloseCategorize, CloseBudget, CloseGoals, CloseMoneyLent, CloseRecommendations, CloseSnapshot}
	if len(close.Items) != len(keys) {
		t.Fatalf("items = %+v", close.Items)
	}
	for i, key := range keys {
		if close.Items[i].Key != key || close.Items[i].Title == "" || close.Items[i].Detail == "" {
			t.Errorf("item %d = %+v", i, close.Items[i])
		}
	}
	if item := closeItem(close, CloseDrafts); item.State != CloseAction || *item.Count != 2 || item.Detail != "2 drafts waiting for approval." {
		t.Errorf("drafts = %+v", item)
	}
	if item := closeItem(close, CloseCategorize); item.State != CloseAction || *item.Count != 2 || *item.Amount != 600_000 {
		t.Errorf("categorize = %+v", item)
	}
	if item := closeItem(close, CloseBudget); item.State != CloseAction || *item.Count != 1 || *item.Amount != 3_000_000 || *item.Target != 20_000_000 {
		t.Errorf("budget = %+v", item)
	}
	// October saved 30M - 18M - 0.5M (the uncategorized expense) = 11.5M.
	if item := closeItem(close, CloseGoals); item.State != CloseAction || *item.Amount != 11_500_000 || *item.Target != 15_000_000 || *item.Count != 1 {
		t.Errorf("goals = %+v", item)
	}
	if item := closeItem(close, CloseMoneyLent); item.State != CloseDone || *item.Count != 0 {
		t.Errorf("money lent = %+v", item)
	}
	if item := closeItem(close, CloseRecommendations); item.State != CloseAction || *item.Count != 2 || *item.Amount != 5_000_000 ||
		item.Detail != "2 open recommendations to act on, dismiss or snooze. 1 snoozed." {
		t.Errorf("recommendations = %+v", item)
	}
	if item := closeItem(close, CloseSnapshot); item.State != CloseAction {
		t.Errorf("snapshot = %+v", item)
	}
	if close.Checked != 7 || close.Done != 1 {
		t.Fatalf("done %d of %d", close.Done, close.Checked)
	}
}

func TestGivenEverythingInOrderWhenComputingTheCloseThenEveryCheckedItemIsDone(t *testing.T) {
	in := closeInputs()
	none := 0
	in.PendingDrafts = &none
	in.Transactions = in.Transactions[:4]
	in.Budget.Lines = in.Budget.Lines[1:2]
	in.Goals = &GoalPlan{Check: PlanCheck{RequiredPerMonth: 5_000_000}}
	in.MoneyLent = &MoneyLent{}
	in.Recommendations = &RecommendationSummary{Done: 4}
	in.Snapshots = SnapshotSheet{Exists: true, Snapshots: []MonthSnapshot{{Month: "2026-10", AsOf: "2026-10-31", NetWorth: 1}}}

	close := ComputeMonthlyClose(in)

	if close.Done != close.Checked || close.Checked != 7 || close.Snapshot == nil {
		t.Fatalf("close = %+v", close)
	}
	if item := closeItem(close, CloseMoneyLent); item.Detail != "Nobody owes you money." {
		t.Fatalf("money lent = %+v", item)
	}
}

func TestGivenMissingContextWhenComputingTheCloseThenThoseItemsAreUnavailable(t *testing.T) {
	in := closeInputs()
	in.PendingDrafts, in.DraftsIssue = nil, "quick capture is not configured"
	in.Budget, in.BudgetIssue = nil, "no Budget_2026 tab"
	in.Goals, in.GoalsIssue = nil, "goal plan unreadable"
	in.MoneyLent = nil
	in.Recommendations, in.RecommendationsIssue = nil, "recommendations unreadable"

	close := ComputeMonthlyClose(in)

	for _, key := range []CloseItemKey{CloseDrafts, CloseBudget, CloseGoals, CloseMoneyLent, CloseRecommendations} {
		if item := closeItem(close, key); item.State != CloseUnavailable || item.Detail == "" {
			t.Errorf("%s = %+v", key, item)
		}
	}
	if item := closeItem(close, CloseBudget); item.Detail != "Could not be checked: no Budget_2026 tab" {
		t.Errorf("budget detail = %q", item.Detail)
	}
}

func TestGivenLongOpenLoansAndNoGoalPlanWhenComputingTheCloseThenTheyAreActions(t *testing.T) {
	in := closeInputs()
	days := 45
	in.MoneyLent.People = append(in.MoneyLent.People, lentPerson("Person B", 4_000_000, days, "", 0))
	in.Goals = &GoalPlan{SetupNeeded: true}

	close := ComputeMonthlyClose(in)

	if item := closeItem(close, CloseMoneyLent); item.State != CloseAction || *item.Count != 1 || *item.Amount != 4_000_000 {
		t.Fatalf("money lent = %+v", item)
	}
	if item := closeItem(close, CloseGoals); item.State != CloseAction || item.Amount != nil {
		t.Fatalf("goals = %+v", item)
	}
}

func TestGivenSnapshotTimingWhenComputingTheCloseThenTheSnapshotItemFollowsTheMonth(t *testing.T) {
	in := closeInputs()
	in.Month, in.Today = day(2026, 11, 1), day(2026, 11, 20)
	in.Snapshots = SnapshotSheet{Snapshots: []MonthSnapshot{{Month: "2026-11", AsOf: "2026-11-15", NetWorth: 1}}}
	if item := closeItem(ComputeMonthlyClose(in), CloseSnapshot); item.State != CloseDone {
		t.Fatalf("a snapshot during the month is done for now: %+v", item)
	}

	in.Month, in.Today = day(2026, 10, 1), day(2026, 11, 3)
	in.Snapshots = SnapshotSheet{Snapshots: []MonthSnapshot{{Month: "2026-10", AsOf: "2026-10-20", NetWorth: 1}}}
	if item := closeItem(ComputeMonthlyClose(in), CloseSnapshot); item.State != CloseAction {
		t.Fatalf("a snapshot taken before the month ended needs updating: %+v", item)
	}

	in.Month, in.Today = day(2026, 8, 1), day(2026, 11, 3)
	in.Snapshots = SnapshotSheet{}
	close := ComputeMonthlyClose(in)
	if item := closeItem(close, CloseSnapshot); item.State != CloseUnavailable || close.CanWriteSnapshot {
		t.Fatalf("an old month cannot be snapshotted: %+v", item)
	}
}

func TestGivenAMonthWhenParsingTheCloseMonthThenFutureAndBadMonthsAreRejected(t *testing.T) {
	if month, err := ParseCloseMonth("", day(2026, 11, 11)); err != nil || MonthKey(month) != "2026-11" {
		t.Fatalf("default after the grace days = %v %v", month, err)
	}
	if month, err := ParseCloseMonth("2025-03", day(2026, 11, 11)); err != nil || MonthKey(month) != "2025-03" {
		t.Fatalf("past month = %v %v", month, err)
	}
	for _, value := range []string{"2026-12", "March"} {
		if _, err := ParseCloseMonth(value, day(2026, 11, 11)); !errors.Is(err, ErrValidation) {
			t.Errorf("%q: %v", value, err)
		}
	}
}
