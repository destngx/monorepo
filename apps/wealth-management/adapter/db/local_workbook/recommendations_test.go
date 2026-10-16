package local_workbook

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/service"
	"errors"
	"testing"
	"time"
)

// The Recommendations writes round-trip through the local workbook (§14.3):
// the tab is created on the first item, a status change rewrites only its
// status columns and every change is read back from the workbook.
func TestGivenTheLocalWorkbookWhenManagingRecommendationsThenEveryWriteReadsBack(t *testing.T) {
	client := openFixture(t)
	db := service.NewDatabaseService(client, nil, logger.NewTestLogger(t), service.DatabaseServiceOptions{})
	today := time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC)

	before, err := db.ReadRecommendations(true)
	if err != nil || before.Exists {
		t.Fatalf("before the first item: %+v %v", before, err)
	}
	recorded, err := db.UpdateRecommendations(func(sheet domain.RecommendationSheet) (domain.RecommendationChanges, error) {
		changes, _, _ := domain.PlanReviewRecord(sheet, domain.ReviewRecommendations{
			Source: "budget-review", Page: "/budget",
			Proposals: []domain.RecommendationProposal{
				{Title: "Cut dining out", Action: "Cook twice a week", ImpactVND: money(1_200_000), Evidence: "Dining over its limit"},
				{Title: "=Review subscriptions", Action: "Cancel one you do not use"},
			},
		}, today)
		return changes, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	assertRows(t, read(t, client, "'Recommendations'!A1:L1"), [][]string{{"Id", "Created", "Source", "Title", "Action", "Impact VND per year", "Evidence", "Status", "Snoozed until", "Status changed", "Outcome", "Page"}})
	created := recorded.Items(recorded.Added)
	if len(created) != 2 || created[0].Row != 2 || created[0].Created != "2026-10-17" || *created[0].ImpactVND != 1_200_000 ||
		created[0].EvidencePath != "/budget" || created[0].Evidence != "Dining over its limit" || created[1].Title != "=Review subscriptions" {
		t.Fatalf("created = %+v", created)
	}

	snooze := domain.RecommendationStatusInput{Status: domain.RecommendationSnoozed, SnoozeDays: 7, ETag: created[0].ETag}
	snoozed, err := db.UpdateRecommendations(func(sheet domain.RecommendationSheet) (domain.RecommendationChanges, error) {
		return domain.PlanStatusChange(sheet, created[0].ID, snooze, today)
	})
	if err != nil {
		t.Fatal(err)
	}
	if item := snoozed.Items(snoozed.Updated)[0]; item.Status != domain.RecommendationSnoozed || item.SnoozedUntil != "2026-10-24" || item.StatusChanged != "2026-10-17" || item.Title != "Cut dining out" {
		t.Fatalf("snoozed = %+v", item)
	}
	if _, err := db.UpdateRecommendations(func(sheet domain.RecommendationSheet) (domain.RecommendationChanges, error) {
		return domain.PlanStatusChange(sheet, created[0].ID, snooze, today)
	}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("a stale etag must conflict: %v", err)
	}

	after, err := db.ReadRecommendations(true)
	if err != nil || len(after.Items) != 2 || len(after.Issues) != 0 {
		t.Fatalf("after: %+v %v", after, err)
	}
}
