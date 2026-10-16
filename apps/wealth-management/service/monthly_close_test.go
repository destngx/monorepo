package service

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/internal/fakes"
	"context"
	"errors"
	"testing"
	"time"
)

type fakeDraftCounter struct {
	count int
	err   error
}

func (f fakeDraftCounter) PendingDraftCount(context.Context) (int, error) { return f.count, f.err }

type fakeAllocation struct {
	allocation domain.Allocation
	err        error
	forced     []bool
}

func (f *fakeAllocation) Allocation(_ context.Context, forceFresh bool) (domain.Allocation, error) {
	f.forced = append(f.forced, forceFresh)
	return f.allocation, f.err
}

// closeDB: today is 2026-10-12 (fixedNow); a 50M bank account, 10M income
// and 4M expense in October, one uncategorized row.
func closeDB() *fakes.Database {
	income, expense := domain.CategoryTypeIncome, domain.CategoryTypeExpense
	salary, food, unknown := 10_000_000.0, 4_000_000.0, 200_000.0
	return &fakes.Database{
		Accounts: []domain.Account{{Name: "Bank", Type: domain.AccountTypeBank, Balance: 50_000_000}},
		Transactions: []domain.Transaction{
			{AccountName: "Bank", Date: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), Category: "Salary", CategoryType: &income, Deposit: &salary},
			{AccountName: "Bank", Date: time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC), Category: "Food", CategoryType: &expense, Payment: &food},
			{AccountName: "Bank", Date: time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC), Category: "Uncategorized", Payment: &unknown},
		},
		Budget: []domain.BudgetItem{expenseBudget("Food", 5_000_000)},
		Recommendations: domain.RecommendationSheet{Exists: true, Items: []domain.Recommendation{
			{ID: "rec-1", Row: 2, Title: "Move idle cash", Status: domain.RecommendationOpen, ImpactVND: ptr(1_200_000.0)},
			{ID: "rec-2", Row: 3, Title: "Review a subscription", Status: domain.RecommendationDone},
		}},
	}
}

func newCloseService(t *testing.T, db *fakes.Database, allocation *fakeAllocation) *MonthlyCloseService {
	t.Helper()
	now := func() time.Time { return fixedNow }
	var source interface {
		Allocation(context.Context, bool) (domain.Allocation, error)
	}
	if allocation != nil {
		source = allocation
	}
	analytics := NewAnalyticsService(db, now)
	recommendations := NewRecommendationService(db, analytics, nil, logger.NewTestLogger(t), now)
	return NewMonthlyCloseService(db, analytics, source, recommendations, logger.NewTestLogger(t), now)
}

func itemOf(close domain.MonthlyClose, key domain.CloseItemKey) domain.CloseItem {
	for _, item := range close.Items {
		if item.Key == key {
			return item
		}
	}
	return domain.CloseItem{}
}

func TestGivenTheLedgerWhenReadingTheCloseThenEveryItemIsDerived(t *testing.T) {
	svc := newCloseService(t, closeDB(), nil)
	svc.SetDraftCounter(fakeDraftCounter{count: 3})

	close, err := svc.MonthlyClose(context.Background(), "", false)

	if err != nil || close.Month != "2026-10" || close.MonthEnded {
		t.Fatalf("close %+v err %v", close, err)
	}
	if item := itemOf(close, domain.CloseDrafts); item.State != domain.CloseAction || *item.Count != 3 {
		t.Fatalf("drafts = %+v", item)
	}
	if item := itemOf(close, domain.CloseCategorize); *item.Count != 1 {
		t.Fatalf("categorize = %+v", item)
	}
	if item := itemOf(close, domain.CloseBudget); item.State != domain.CloseDone {
		t.Fatalf("budget = %+v", item)
	}
	if item := itemOf(close, domain.CloseGoals); item.State != domain.CloseAction || item.Detail != "Set up the goal plan so the surplus has somewhere to go." {
		t.Fatalf("goals = %+v", item)
	}
	if item := itemOf(close, domain.CloseRecommendations); item.State != domain.CloseAction || *item.Count != 1 || *item.Amount != 1_200_000 {
		t.Fatalf("recommendations = %+v", item)
	}
}

func TestGivenFailingContextWhenReadingTheCloseThenThoseItemsAreUnavailableNotErrors(t *testing.T) {
	db := closeDB()
	db.GoalPlanErr = errors.New("goal tab unreadable")
	svc := newCloseService(t, db, nil)
	svc.SetDraftCounter(fakeDraftCounter{err: errors.New("store locked")})

	close, err := svc.MonthlyClose(context.Background(), "2026-10", false)

	if err != nil {
		t.Fatal(err)
	}
	if item := itemOf(close, domain.CloseDrafts); item.State != domain.CloseUnavailable || item.Detail != "Could not be checked: store locked" {
		t.Fatalf("drafts = %+v", item)
	}
	if item := itemOf(close, domain.CloseGoals); item.State != domain.CloseUnavailable {
		t.Fatalf("goals = %+v", item)
	}
	svc.SetDraftCounter(nil)
	close, _ = svc.MonthlyClose(context.Background(), "2026-10", false)
	if item := itemOf(close, domain.CloseDrafts); item.State != domain.CloseUnavailable {
		t.Fatalf("without quick capture = %+v", item)
	}
}

func TestGivenARequiredReadFailingWhenReadingTheCloseThenTheErrorIsReturned(t *testing.T) {
	db := closeDB()
	db.SnapshotsErr = errors.New("quota")
	if _, err := newCloseService(t, db, nil).MonthlyClose(context.Background(), "", false); err == nil {
		t.Fatal("expected the snapshots error")
	}
	if _, err := newCloseService(t, closeDB(), nil).MonthlyClose(context.Background(), "2027-01", false); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("a future month must be rejected: %v", err)
	}
}

func TestGivenThisMonthWhenWritingTheSnapshotTwiceThenTheRowIsUpdatedNotDuplicated(t *testing.T) {
	db := closeDB()
	allocation := &fakeAllocation{allocation: domain.ComputeAllocation(domain.AllocationInputs{Funds: 7_000_000})}
	svc := newCloseService(t, db, allocation)

	first, err := svc.WriteSnapshot(context.Background(), "2026-10")
	if err != nil || !first.Created || first.Snapshot.NetWorth != 50_000_000 || *first.Snapshot.Funds != 7_000_000 || first.Snapshot.AsOf != "2026-10-12" {
		t.Fatalf("first %+v err %v", first, err)
	}
	second, err := svc.WriteSnapshot(context.Background(), "")
	if err != nil || second.Created || len(db.Snapshots.Snapshots) != 1 || len(db.SavedSnapshots) != 2 {
		t.Fatalf("second %+v err %v rows %d", second, err, len(db.Snapshots.Snapshots))
	}
	if len(allocation.forced) != 2 || !allocation.forced[0] {
		t.Fatalf("holdings must be valued afresh: %v", allocation.forced)
	}
	close, _ := svc.MonthlyClose(context.Background(), "2026-10", false)
	if close.Snapshot == nil || itemOf(close, domain.CloseSnapshot).State != domain.CloseDone {
		t.Fatalf("the close must see the snapshot: %+v", close)
	}
}

func TestGivenAnOldMonthOrAFailingValuationWhenWritingTheSnapshotThenNothingIsWritten(t *testing.T) {
	db := closeDB()
	if _, err := newCloseService(t, db, nil).WriteSnapshot(context.Background(), "2026-08"); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("old month: %v", err)
	}
	if _, err := newCloseService(t, db, &fakeAllocation{err: errors.New("prices down")}).WriteSnapshot(context.Background(), ""); err == nil {
		t.Fatal("a failing valuation must fail the write")
	}
	if len(db.SavedSnapshots) != 0 {
		t.Fatalf("nothing may be written: %+v", db.SavedSnapshots)
	}
}

func TestGivenSnapshotsWhenReadingTheHistoryThenTheyAnchorTheMonths(t *testing.T) {
	db := closeDB()
	db.Snapshots = domain.SnapshotSheet{Exists: true, Snapshots: []domain.MonthSnapshot{{Month: "2026-09", AsOf: "2026-09-30", NetWorth: 40_000_000, Row: 2}}}

	history, err := newCloseService(t, db, nil).NetWorthHistory(context.Background(), false)

	if err != nil || len(history.Points) != 2 || history.Snapshots != 1 || history.Points[1].Source != domain.NetWorthLive {
		t.Fatalf("history %+v err %v", history, err)
	}
	// October: 10M change, 6M income - expense (the uncategorized row is
	// neither), so 4M market and other.
	if october := history.Points[1]; *october.Change != 10_000_000 || *october.Savings != 6_000_000 || *october.Market != 4_000_000 {
		t.Fatalf("october = %+v", october)
	}
}
