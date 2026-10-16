package local_workbook

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/service"
	"errors"
	"testing"
)

// The Recurring writes round-trip through the local workbook (§14.3): the
// tab is created on the first item and every change is read back from it.
func TestGivenTheLocalWorkbookWhenManagingRecurringItemsThenEveryWriteReadsBack(t *testing.T) {
	client := openFixture(t)
	db := service.NewDatabaseService(client, nil, logger.NewTestLogger(t), service.DatabaseServiceOptions{})

	before, err := db.ReadRecurring(true)
	if err != nil || before.Exists {
		t.Fatalf("before the first item: %+v %v", before, err)
	}
	rent, err := db.CreateRecurring(domain.RecurringInput{Name: "Rent", Kind: domain.RecurringBill, Amount: money(8_000_000), Day: 1, Category: "Rent", Match: "Landlord"})
	if err != nil {
		t.Fatal(err)
	}
	assertRows(t, read(t, client, "'Recurring'!A1:K1"), [][]string{{"Name", "Kind", "Amount", "Frequency", "Day", "Month", "Account", "Category", "Match", "Status", "Note"}})
	if rent.ID != "recurring-2" || *rent.Amount != 8_000_000 || rent.Frequency != domain.FrequencyMonthly || rent.Day != 1 || rent.Status != domain.RecurringActive {
		t.Fatalf("created = %+v", rent)
	}
	cover, err := db.CreateRecurring(domain.RecurringInput{Name: "Cover", Kind: domain.RecurringInsurance, Amount: money(2_000_000), Frequency: domain.FrequencyYearly, Day: 5, Month: 11, Match: "Cover Co"})
	if err != nil || cover.ID != "recurring-3" || cover.Month != 11 {
		t.Fatalf("second = %+v %v", cover, err)
	}

	updated, err := db.UpdateRecurring(rent.ID, domain.RecurringUpdateInput{RecurringInput: domain.RecurringInput{Name: "Rent", Kind: domain.RecurringBill, Amount: money(9_000_000), Day: 2, Match: "Landlord"}, ETag: rent.ETag})
	if err != nil || *updated.Amount != 9_000_000 || updated.Day != 2 || updated.Category != "" {
		t.Fatalf("updated = %+v %v", updated, err)
	}
	if _, err := db.UpdateRecurring(rent.ID, domain.RecurringUpdateInput{RecurringInput: domain.RecurringInput{Name: "Rent", Kind: domain.RecurringBill, Amount: money(1), Day: 2, Match: "x"}, ETag: rent.ETag}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("a stale etag must conflict: %v", err)
	}
	paused, err := db.SetRecurringStatus(rent.ID, domain.RecurringStatusInput{Status: domain.RecurringPaused, ETag: updated.ETag})
	if err != nil || paused.Status != domain.RecurringPaused || *paused.Amount != 9_000_000 {
		t.Fatalf("paused = %+v %v", paused, err)
	}

	if err := db.DeleteRecurring(rent.ID, paused.ETag); err != nil {
		t.Fatal(err)
	}
	after, err := db.ReadRecurring(true)
	if err != nil || !after.Exists || len(after.Items) != 1 || after.Items[0].Name != "Cover" || len(after.Issues) != 0 {
		t.Fatalf("after delete: %+v %v", after, err)
	}
}
