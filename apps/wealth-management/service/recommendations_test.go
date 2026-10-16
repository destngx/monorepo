package service

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/internal/fakes"
	"apps/wealth-management/port"
	"context"
	"errors"
	"testing"
	"time"
)

// stubAllocation serves a fixed allocation, or fails.
type stubAllocation struct {
	allocation domain.Allocation
	err        error
}

func (s *stubAllocation) Allocation(context.Context, bool) (domain.Allocation, error) {
	return s.allocation, s.err
}

func lendingDatabase(days int) *fakes.Database {
	nonBudget := domain.CategoryTypeNonBudget
	lent := 2_000_000.0
	return &fakes.Database{
		Categories: []domain.Category{{Name: "Lending / Repayment", Type: domain.CategoryTypeNonBudget, Nature: domain.NatureLending}},
		Transactions: []domain.Transaction{{
			ID: "row-2", AccountName: "Bank", Date: fixedNow.AddDate(0, 0, -days), Payee: "Person A",
			Category: "Lending / Repayment", CategoryType: &nonBudget, CategoryNature: domain.NatureLending, Payment: &lent,
		}},
	}
}

func newRecommendations(t *testing.T, db *fakes.Database, allocation port.AllocationSource) port.RecommendationService {
	t.Helper()
	now := func() time.Time { return fixedNow }
	return NewRecommendationService(db, NewAnalyticsService(db, now), allocation, logger.NewTestLogger(t), now)
}

func TestGivenMoneyLentLongAgoWhenRefreshingTwiceThenOneRuleItemIsAddedAndClosedOnceRepaid(t *testing.T) {
	db := lendingDatabase(120)
	svc := newRecommendations(t, db, &stubAllocation{err: errors.New("market data down")})
	ctx := context.Background()

	list, err := svc.Refresh(ctx, domain.RecommendationFilter{})
	if err != nil || len(list.Items) != 1 || !list.TabExists {
		t.Fatalf("list = %+v %v", list, err)
	}
	item := list.Items[0]
	if item.Source != "rule:lent-overdue" || item.Page != "/accounts/lent" || item.Status != domain.RecommendationOpen || list.Summary.Open != 1 {
		t.Fatalf("item = %+v", item)
	}
	if again, err := svc.Refresh(ctx, domain.RecommendationFilter{}); err != nil || len(again.Items) != 1 || db.RecommendationWrites != 1 {
		t.Fatalf("a second refresh must not add again: %+v %v (writes %d)", again, err, db.RecommendationWrites)
	}

	repaid := 2_000_000.0
	db.Transactions = append(db.Transactions, domain.Transaction{
		ID: "row-3", AccountName: "Bank", Date: fixedNow, Payee: "Person A", Category: "Lending / Repayment",
		CategoryType: db.Transactions[0].CategoryType, CategoryNature: domain.NatureLending, Deposit: &repaid,
	})
	closed, err := svc.Refresh(ctx, domain.RecommendationFilter{Status: domain.RecommendationDone})
	if err != nil || len(closed.Items) != 1 || closed.Items[0].Outcome != "resolved" || closed.Items[0].StatusChanged != "2026-10-12" {
		t.Fatalf("closed = %+v %v", closed, err)
	}
}

func TestGivenAnOpenItemWhenTheUserSnoozesItThenTheStatusIsWrittenAndAStaleETagConflicts(t *testing.T) {
	db := lendingDatabase(120)
	svc := newRecommendations(t, db, nil)
	ctx := context.Background()
	list, err := svc.Refresh(ctx, domain.RecommendationFilter{})
	if err != nil {
		t.Fatal(err)
	}
	item := list.Items[0]

	snoozed, err := svc.SetStatus(ctx, item.ID, domain.RecommendationStatusInput{Status: domain.RecommendationSnoozed, SnoozeDays: 30, ETag: item.ETag})
	if err != nil || snoozed.Status != domain.RecommendationSnoozed || snoozed.SnoozedUntil != "2026-11-11" || snoozed.ETag == item.ETag {
		t.Fatalf("snoozed = %+v %v", snoozed, err)
	}
	if _, err := svc.SetStatus(ctx, item.ID, domain.RecommendationStatusInput{Status: domain.RecommendationDone, ETag: item.ETag}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("a stale etag must conflict: %v", err)
	}
	if _, err := svc.List(ctx, domain.RecommendationFilter{Status: "later"}, false); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("a bad filter: %v", err)
	}
	if again, err := svc.Refresh(ctx, domain.RecommendationFilter{}); err != nil || len(again.Items) != 1 {
		t.Fatalf("a snoozed rule item is still pending, so it is not added again: %+v %v", again, err)
	}
}

func TestGivenAReviewWhenRecordingThenNewItemsAndCheckedOutcomesAreReturned(t *testing.T) {
	db := &fakes.Database{}
	svc := newRecommendations(t, db, nil)
	ctx := context.Background()

	first, err := svc.Record(ctx, domain.ReviewRecommendations{Source: "year-review", Page: "/cash-flow", Proposals: []domain.RecommendationProposal{{Title: "Raise savings", Action: "Move 1.000.000 ₫ a month to savings"}}})
	if err != nil || len(first.Created) != 1 || first.Created[0].Page != "/cash-flow" {
		t.Fatalf("first = %+v %v", first, err)
	}
	recent, err := svc.RecentForReview(ctx, "year-review")
	if err != nil || len(recent) != 1 {
		t.Fatalf("recent = %+v %v", recent, err)
	}
	second, err := svc.Record(ctx, domain.ReviewRecommendations{
		Source: "year-review", Page: "/cash-flow",
		Proposals: []domain.RecommendationProposal{{Title: "raise savings!", Action: "Again"}},
		Outcomes:  []domain.RecommendationOutcome{{ID: recent[0].ID, Verdict: domain.OutcomeMet, Note: "Savings rate is up"}, {ID: "rec-unknown", Verdict: domain.OutcomeMet}},
	})
	if err != nil || len(second.Created) != 0 || second.Duplicates != 1 || second.IgnoredOutcomes != 1 || len(second.Checked) != 1 {
		t.Fatalf("second = %+v %v", second, err)
	}
	if checked := second.Checked[0]; checked.Status != domain.RecommendationDone || checked.Outcome != "Met: Savings rate is up" {
		t.Fatalf("checked = %+v", checked)
	}
	if empty, err := svc.Record(ctx, domain.ReviewRecommendations{Source: "year-review"}); err != nil || db.RecommendationWrites != 2 || empty.Created == nil {
		t.Fatalf("an empty review writes nothing: %+v %v", empty, err)
	}
}
