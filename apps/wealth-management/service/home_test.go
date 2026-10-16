package service

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/internal/fakes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// stubDrafts counts pending drafts, or fails.
type stubDrafts struct {
	pending int
	err     error
}

func (s stubDrafts) PendingDraftCount(context.Context) (int, error) {
	return s.pending, s.err
}

// stubClose serves a fixed checklist, or fails.
type stubClose struct {
	close domain.MonthlyClose
	err   error
}

func (s stubClose) MonthlyClose(context.Context, string, bool) (domain.MonthlyClose, error) {
	return s.close, s.err
}

// failingBills fails the bills read only.
type failingBills struct {
	HomeAnalytics
}

func (failingBills) Bills(bool) (domain.Bills, error) {
	return domain.Bills{}, errors.New("recurring tab unreadable")
}

func homeSources(db *fakes.Database) HomeSources {
	now := func() time.Time { return fixedNow }
	return HomeSources{
		Analytics:       NewAnalyticsService(db, now),
		Ledger:          db,
		Recommendations: NewRecommendationService(db, NewAnalyticsService(db, now), nil, nil, now),
	}
}

func homeKinds(result domain.NeedsYou) string {
	kinds := make([]string, len(result.Items))
	for i, item := range result.Items {
		kinds[i] = string(item.Kind)
	}
	return strings.Join(kinds, ",")
}

func TestGivenSheetDataAndDraftsWhenListingNeedsYouThenEachSourceAddsItsItemsRanked(t *testing.T) {
	db := lendingDatabase(120)
	db.Transactions = append(db.Transactions, domain.Transaction{ID: "row-3", AccountName: "Bank", Date: fixedNow.AddDate(0, 0, -2), Payee: "Shop", Payment: new(150_000.0)})
	sources := homeSources(db)
	sources.Drafts = stubDrafts{pending: 2}
	sources.Close = stubClose{close: domain.MonthlyClose{Month: "2026-09", MonthEnded: true, Items: []domain.CloseItem{{Key: domain.CloseSnapshot, State: domain.CloseAction}}}}
	svc := NewHomeService(sources, logger.NewTestLogger(t), func() time.Time { return fixedNow })

	result, err := svc.NeedsYou(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if got := homeKinds(result); got != "lent-overdue,month-close,drafts,uncategorized" {
		t.Fatalf("kinds = %s", got)
	}
	if len(result.Unavailable) != 0 || result.AsOf != "2026-10-12" {
		t.Fatalf("result = %+v", result)
	}
	if strings.Contains(result.Items[0].Headline+result.Items[0].Detail, "Person A") {
		t.Fatalf("a person's name leaks: %+v", result.Items[0])
	}
}

func TestGivenARuleItemDismissedWhenListingNeedsYouThenTheLiveFindingStaysQuiet(t *testing.T) {
	db := lendingDatabase(120)
	db.Recommendations = domain.RecommendationSheet{Exists: true, Items: []domain.Recommendation{{
		ID: "rec-1a2b3c4d", Row: 2, Created: "2026-10-01", Source: "rule:lent-overdue", Title: "Chase money lent",
		Status: domain.RecommendationDismissed, StatusChanged: "2026-10-05",
	}}}
	svc := NewHomeService(homeSources(db), logger.NewTestLogger(t), func() time.Time { return fixedNow })

	result, err := svc.NeedsYou(context.Background(), false)
	if err != nil || len(result.Items) != 0 {
		t.Fatalf("result = %+v %v", result, err)
	}
	if db.RecommendationWrites != 0 {
		t.Fatal("the inbox reads only")
	}
}

func TestGivenSomeSourcesFailingWhenListingNeedsYouThenTheRestStillListsAndTheFailuresAreNamed(t *testing.T) {
	db := lendingDatabase(120)
	sources := homeSources(db)
	sources.Analytics = failingBills{sources.Analytics}
	sources.Drafts = stubDrafts{err: errors.New("capture store locked")}
	sources.Allocation = &stubAllocation{err: errors.New("market data down")}
	sources.Close = stubClose{err: errors.New("snapshots tab unreadable")}
	log, records := logger.NewRecordingTestLogger(t)
	svc := NewHomeService(sources, log, func() time.Time { return fixedNow })

	result, err := svc.NeedsYou(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if got := homeKinds(result); got != "lent-overdue" {
		t.Fatalf("kinds = %s", got)
	}
	parts := []string{}
	for _, missing := range result.Unavailable {
		parts = append(parts, string(missing.Part))
	}
	if got := strings.Join(parts, ","); got != "bills,allocation,drafts,close" {
		t.Fatalf("unavailable = %s", got)
	}
	if logged := records.String(); !strings.Contains(logged, "needs you: bills unavailable") || !strings.Contains(logged, "capture store locked") {
		t.Fatalf("failures must be logged: %s", logged)
	}
}

func TestGivenEverySourceFailingWhenListingNeedsYouThenTheErrorIsReturned(t *testing.T) {
	sheetDown := errors.New("sheet unreachable")
	db := &fakes.Database{Err: sheetDown}
	svc := NewHomeService(homeSources(db), logger.NewTestLogger(t), func() time.Time { return fixedNow })
	if _, err := svc.NeedsYou(context.Background(), false); !errors.Is(err, sheetDown) {
		t.Fatalf("err = %v", err)
	}
}
