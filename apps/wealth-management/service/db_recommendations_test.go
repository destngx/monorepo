package service

import (
	"apps/wealth-management/domain"
	"errors"
	"strings"
	"testing"
)

var recommendationRow = []string{"rec-0a1b2c3d", "2026-10-01", "budget-review", "Cut dining out", "Cook twice a week", "1200000", "/budget: Dining over its limit", "open", "", "", "", "/budget"}

func recommendationsClient(rows [][]string) *fakeSheetClient {
	client := newFakeSheetClient()
	client.titles = []string{"Accounts", recommendationsSheet}
	client.ranges[recommendationsRange] = rows
	return client
}

func TestGivenARecommendationsTabWhenReadingThenRowsAreParsedAndBadRowsReported(t *testing.T) {
	client := recommendationsClient([][]string{
		{"Id", "Created", "Source", "Title"},
		recommendationRow,
		{},
		{"rec-11112222", "46312", "rule:rebalance", "Rebalance", "Move money", "", "/investments", "Snoozed", "2026-11-01", "2026-10-12", "", "/investments"},
		{"rec-33334444", "2026-10-01", "year-review", "Bad impact", "x", "lots"},
		{"rec-0a1b2c3d", "2026-10-02", "budget-review", "Repeated id", "x"},
		{"", "2026-10-01", "budget-review", "No id", "x"},
	})
	sheet, err := newTestDBService(t, client, newFakeCache()).ReadRecommendations(true)
	if err != nil || !sheet.Exists || len(sheet.Items) != 2 {
		t.Fatalf("sheet = %+v %v", sheet, err)
	}
	first, snoozed := sheet.Items[0], sheet.Items[1]
	if first.Row != 3 || *first.ImpactVND != 1_200_000 || first.EvidencePath != "/budget" || first.Evidence != "Dining over its limit" || first.ETag == "" {
		t.Fatalf("first = %+v", first)
	}
	if snoozed.Status != domain.RecommendationSnoozed || snoozed.Created != "2026-10-17" || snoozed.ImpactVND != nil || snoozed.EvidencePath != "/investments" {
		t.Fatalf("snoozed = %+v", snoozed)
	}
	want := map[int]string{6: "impact \"lots\"", 7: "repeats row 3", 8: "id \"\""}
	if len(sheet.Issues) != len(want) {
		t.Fatalf("issues = %+v", sheet.Issues)
	}
	for _, issue := range sheet.Issues {
		if !strings.Contains(issue.Message, want[issue.Row]) {
			t.Errorf("row %d: %q should mention %q", issue.Row, issue.Message, want[issue.Row])
		}
	}
}

func TestGivenNoRecommendationsTabWhenReadingThenThereAreNoItemsAndTheTabIsNotRead(t *testing.T) {
	client := newFakeSheetClient()
	client.titles = []string{"Accounts"}
	sheet, err := newTestDBService(t, client, newFakeCache()).ReadRecommendations(true)
	if err != nil || sheet.Exists || sheet.Items == nil || len(sheet.Items) != 0 {
		t.Fatalf("sheet = %+v %v", sheet, err)
	}
	if contains(client.reads, recommendationsRange) {
		t.Fatal("must not read a tab that does not exist")
	}
}

func TestGivenNoRecommendationsTabWhenTheFirstItemIsWrittenThenTheTabIsCreatedAndOnlyItIsWritten(t *testing.T) {
	client := newFakeSheetClient()
	client.titles = []string{"Accounts"}
	client.nextRow = 2
	cache := newFakeCache()
	impact := 1_200_000.0
	item := domain.Recommendation{
		ID: "rec-0a1b2c3d", Created: "2026-10-12", Source: "budget-review", Title: "=Cut dining", Action: "Cook", ImpactVND: &impact,
		EvidencePath: "/budget", Evidence: "Dining over", Status: domain.RecommendationOpen, Page: "/budget",
	}
	svc := newTestDBService(t, client, cache)
	// The fake does not keep writes: the read-back sees the row as the sheet would.
	client.ranges[recommendationsRange] = [][]string{recommendationRow}

	write, err := svc.UpdateRecommendations(func(sheet domain.RecommendationSheet) (domain.RecommendationChanges, error) {
		if sheet.Exists {
			t.Fatal("the plan must see the tab as missing")
		}
		return domain.RecommendationChanges{Add: []domain.Recommendation{item}}, nil
	})
	if err != nil || len(write.Added) != 1 || len(write.Sheet.Items) != 1 {
		t.Fatalf("write = %+v %v", write, err)
	}
	if len(client.created) != 1 || client.created[0] != recommendationsSheet || client.updates[0].values[0] != "Id" || len(client.updates[0].values) != recommendationsColumns {
		t.Fatalf("created = %v updates = %+v", client.created, client.updates)
	}
	if len(client.appends) != 1 || client.appends[0].sheetName != recommendationsSheet || client.appends[0].anchor != recommendationsAnchor {
		t.Fatalf("appends = %+v", client.appends)
	}
	row := client.appends[0].rows[0]
	if len(row) != recommendationsColumns || row[0] != "rec-0a1b2c3d" || row[1] != "'2026-10-12" || row[3] != "'=Cut dining" || row[5] != 1_200_000.0 || row[6] != "/budget: Dining over" || row[7] != "open" || row[8] != "" || row[11] != "/budget" {
		t.Fatalf("row = %#v", row)
	}
	if !contains(cache.invalidated, cacheKeyRecommendationsPattern) {
		t.Fatalf("caches not invalidated: %v", cache.invalidated)
	}
}

func TestGivenAnExistingItemWhenUpdatingThenOnlyItsStatusColumnsAreWritten(t *testing.T) {
	client := recommendationsClient([][]string{recommendationRow})
	svc := newTestDBService(t, client, newFakeCache())

	write, err := svc.UpdateRecommendations(func(sheet domain.RecommendationSheet) (domain.RecommendationChanges, error) {
		item := sheet.Items[0]
		return domain.RecommendationChanges{Update: []domain.RecommendationUpdate{{
			ID: item.ID, Row: item.Row, Status: domain.RecommendationSnoozed, SnoozedUntil: "2026-10-19", StatusChanged: "2026-10-12", Outcome: "+Not met",
		}}}, nil
	})
	if err != nil || len(write.Updated) != 1 {
		t.Fatalf("write = %+v %v", write, err)
	}
	if len(client.updates) != 1 || client.updates[0].rangeValue != "'Recommendations'!H2:K2" {
		t.Fatalf("updates = %+v", client.updates)
	}
	if values := client.updates[0].values; values[0] != "snoozed" || values[1] != "'2026-10-19" || values[2] != "'2026-10-12" || values[3] != "'+Not met" {
		t.Fatalf("values = %#v", values)
	}
	if len(client.appends) != 0 || len(client.created) != 0 {
		t.Fatal("an update appends nothing and creates no tab")
	}

	if _, err := svc.UpdateRecommendations(func(domain.RecommendationSheet) (domain.RecommendationChanges, error) {
		return domain.RecommendationChanges{Update: []domain.RecommendationUpdate{{ID: "rec-gone", Row: 9, Status: domain.RecommendationDone}}}, nil
	}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("a row that is not in the tab: %v", err)
	}
	planErr := errors.New("plan failed")
	if _, err := svc.UpdateRecommendations(func(domain.RecommendationSheet) (domain.RecommendationChanges, error) {
		return domain.RecommendationChanges{}, planErr
	}); !errors.Is(err, planErr) || len(client.updates) != 1 {
		t.Fatalf("a failing plan writes nothing: %v", err)
	}
}
