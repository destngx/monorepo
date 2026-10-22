package service

import (
	"apps/wealth-management/domain"
	"errors"
	"strings"
	"testing"
)

func recurringClient(rows [][]string) *fakeSheetClient {
	client := newFakeSheetClient()
	client.titles = []string{"Accounts", recurringSheet}
	client.ranges[recurringRange] = rows
	return client
}

func TestGivenARecurringTabWhenReadingThenRowsAreParsedAndBadRowsReported(t *testing.T) {
	client := recurringClient([][]string{
		{"Name", "Kind", "Amount", "Frequency", "Day", "Month", "Account", "Category", "Match", "Status", "Note"},
		{"Salary", "Income", "20000000", "", "25", "", "Main Bank", "Salary", "Employer A"},
		{},
		{"Health cover", "insurance", "2.000.000 ₫", "Yearly", "5", "11", "", "", "Cover Co", "paused", "renews"},
		{"Card A", "card", "", "", "", "", "Card Account"},
		{"Phone", "bill", "", "", "20", "", "", "", "Phone Co"},
		{"Water", "utility", "100000", "", "40", "", "", "", "Water Co"},
		{"Cover", "insurance", "1000", "yearly", "5", "", "", "", "Cover"},
		{"", "bill", "1000", "", "1", "", "", "", "x"},
		{"Gym", "bill", "abc", "", "1.5"},
	})
	sheet, err := newTestDBService(t, client, newFakeCache()).ReadRecurring(true)
	if err != nil || !sheet.Exists || len(sheet.Items) != 3 {
		t.Fatalf("sheet = %+v %v", sheet, err)
	}
	salary, cover, card := sheet.Items[0], sheet.Items[1], sheet.Items[2]
	if salary.ID != "recurring-3" || salary.Kind != domain.RecurringIncome || *salary.Amount != 20_000_000 || salary.Frequency != domain.FrequencyMonthly || salary.Day != 25 || salary.Status != domain.RecurringActive || salary.ETag == "" {
		t.Fatalf("salary = %+v", salary)
	}
	if cover.Row != 5 || *cover.Amount != 2_000_000 || cover.Frequency != domain.FrequencyYearly || cover.Month != 11 || cover.Status != domain.RecurringPaused || cover.Note != "renews" {
		t.Fatalf("cover = %+v", cover)
	}
	if card.Kind != domain.RecurringCard || card.Amount != nil || card.Account != "Card Account" {
		t.Fatalf("card = %+v", card)
	}
	want := map[int]string{7: "amount is required", 8: "kind \"utility\"", 9: "month (1-12)", 10: "name is required", 11: "amount \"abc\""}
	if len(sheet.Issues) != len(want) {
		t.Fatalf("issues = %+v", sheet.Issues)
	}
	for _, issue := range sheet.Issues {
		if !strings.Contains(issue.Message, want[issue.Row]) {
			t.Errorf("row %d: %q should mention %q", issue.Row, issue.Message, want[issue.Row])
		}
	}
	if !strings.Contains(sheet.Issues[1].Message, "day must be") || !strings.Contains(sheet.Issues[4].Message, "day \"1.5\"") {
		t.Fatalf("day problems: %+v", sheet.Issues)
	}
}

func TestGivenNoRecurringTabWhenReadingThenThereAreNoItemsAndTheTabIsNotRead(t *testing.T) {
	client := newFakeSheetClient()
	client.titles = []string{"Accounts"}
	sheet, err := newTestDBService(t, client, newFakeCache()).ReadRecurring(true)
	if err != nil || sheet.Exists || len(sheet.Items) != 0 || sheet.Items == nil {
		t.Fatalf("sheet = %+v %v", sheet, err)
	}
	if contains(client.reads, recurringRange) {
		t.Fatal("must not read a tab that does not exist")
	}
}

func TestGivenNoRecurringTabWhenCreatingAnItemThenTheTabIsCreatedAndOnlyItIsWritten(t *testing.T) {
	client := newFakeSheetClient()
	client.titles = []string{"Accounts"}
	client.nextRow = 2
	client.ranges["'Recurring'!A2:L2"] = [][]string{{"Rent", "bill", "8000000", "monthly", "1", "", "", "Rent", "=Landlord", "active"}}
	cache := newFakeCache()

	item, err := newTestDBService(t, client, cache).CreateRecurring(domain.RecurringInput{Name: " Rent ", Kind: "Bill", Amount: ptr(8_000_000.0), Day: 1, Category: "Rent", Match: "=Landlord"})
	if err != nil || item.ID != "recurring-2" || item.Name != "Rent" || item.ETag == "" {
		t.Fatalf("item = %+v %v", item, err)
	}
	if len(client.created) != 1 || client.created[0] != recurringSheet || len(client.updates) != 1 || client.updates[0].values[0] != "Name" {
		t.Fatalf("created = %v updates = %+v", client.created, client.updates)
	}
	append0 := client.appends[0]
	if append0.sheetName != recurringSheet || append0.anchor != recurringAnchor {
		t.Fatalf("append = %+v", append0)
	}
	row := append0.rows[0]
	if len(row) != recurringColumns || row[0] != "Rent" || row[1] != "bill" || row[2] != 8_000_000.0 || row[3] != "monthly" || row[4] != 1 || row[5] != "" || row[8] != "'=Landlord" || row[9] != "active" {
		t.Fatalf("row = %#v", row)
	}
	if !contains(cache.invalidated, cacheKeyRecurringPattern) {
		t.Fatalf("recurring caches not invalidated: %v", cache.invalidated)
	}

	client.created = nil
	client.nextRow = 2
	if _, err := newTestDBService(t, client, cache).CreateRecurring(domain.RecurringInput{Name: "Rent", Kind: domain.RecurringBill, Amount: ptr(1.0), Day: 1, Match: "x"}); err != nil || len(client.created) != 0 {
		t.Fatalf("an existing tab is not created again: %v %v", err, client.created)
	}
}

func TestGivenARecurringItemWhenUpdatingPausingAndDeletingThenTheETagGuardsTheRow(t *testing.T) {
	current := []string{"Rent", "bill", "8000000", "monthly", "1", "", "", "Rent", "Landlord", "active"}
	client := recurringClient(nil)
	client.ranges["'Recurring'!A4:L4"] = [][]string{current}
	client.ranges["'Recurring'!A5:L5"] = [][]string{{}}
	svc := newTestDBService(t, client, newFakeCache())
	etag := rowETag(current, recurringColumns)

	input := domain.RecurringUpdateInput{RecurringInput: domain.RecurringInput{Name: "Rent", Kind: domain.RecurringBill, Amount: ptr(9_000_000.0), Day: 2, Match: "Landlord"}, ETag: etag}
	if _, err := svc.UpdateRecurring("recurring-4", input); err != nil {
		t.Fatal(err)
	}
	if update := client.updates[0]; update.rangeValue != "'Recurring'!A4:L4" || update.values[2] != 9_000_000.0 || update.values[4] != 2 {
		t.Fatalf("update = %+v", update)
	}
	if _, err := svc.SetRecurringStatus("recurring-4", domain.RecurringStatusInput{Status: domain.RecurringPaused, ETag: etag}); err != nil {
		t.Fatal(err)
	}
	if update := client.updates[1]; update.rangeValue != "'Recurring'!J4" || update.values[0] != "paused" {
		t.Fatalf("pause writes column J only: %+v", update)
	}
	input.ETag = "stale"
	if _, err := svc.UpdateRecurring("recurring-4", input); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("stale etag: %v", err)
	}
	if _, err := svc.UpdateRecurring("goal-4", domain.RecurringUpdateInput{RecurringInput: input.RecurringInput, ETag: etag}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("bad id: %v", err)
	}
	if err := svc.DeleteRecurring("recurring-4", etag); err != nil {
		t.Fatal(err)
	}
	if last := client.updates[len(client.updates)-1]; last.rangeValue != "'Recurring'!A4:L4" || len(last.values) != recurringColumns || last.values[0] != "" {
		t.Fatalf("delete clears A-L: %+v", last)
	}
	if err := svc.DeleteRecurring("recurring-5", "x"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("blank row: %v", err)
	}
	for _, update := range client.updates {
		if !strings.HasPrefix(update.rangeValue, "'Recurring'!") {
			t.Fatalf("only the Recurring tab may be written: %+v", update)
		}
	}
}

func TestGivenASuggestionWhenConfirmingThenItIsWrittenAndAnUnknownKeyIsNotFound(t *testing.T) {
	client := newFakeSheetClient()
	client.titles = []string{"Accounts", "Transactions", "Categories"}
	client.ranges[categoriesSheetRange] = [][]string{{"Expense categories"}, {"Utilities"}}
	rows := [][]string{}
	for _, date := range []string{"05/04/2026", "05/05/2026", "06/06/2026", "05/07/2026"} {
		rows = append(rows, []string{"Main Bank", date, "", "Water Co", "", "", "Utilities", "", "200000"})
	}
	client.ranges[transactionsSheetRange] = rows
	client.nextRow = 2
	client.ranges["'Recurring'!A2:L2"] = [][]string{{"Water", "bill", "200000", "monthly", "5", "", "Main Bank", "Utilities", "Water Co", "paused", "Dismissed suggestion"}}
	svc := newTestDBService(t, client, newFakeCache())

	if _, err := svc.ConfirmRecurringSuggestion(domain.RecurringConfirmInput{Key: "out|gas co|utilities"}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("unknown key: %v", err)
	}
	item, err := svc.ConfirmRecurringSuggestion(domain.RecurringConfirmInput{Key: "out|water co|utilities", Name: "Water", Status: domain.RecurringPaused})
	if err != nil || item.Status != domain.RecurringPaused {
		t.Fatalf("confirm = %+v %v", item, err)
	}
	row := client.appends[0].rows[0]
	if row[0] != "Water" || row[2] != 200_000.0 || row[4] != 5 || row[6] != "Main Bank" || row[8] != "Water Co" || row[9] != "paused" || row[10] != dismissedSuggestionNote {
		t.Fatalf("row = %#v", row)
	}
	if len(client.created) != 1 {
		t.Fatalf("the tab is created on the first confirm: %v", client.created)
	}
}

func TestGivenGraceDaysInColumnLWhenReadingAndWritingThenTheyRoundTrip(t *testing.T) {
	client := recurringClient([][]string{
		{"Rent", "bill", "8000000", "monthly", "28", "", "", "", "Landlord", "active", "", "14"},
		{"Water", "bill", "200000", "monthly", "5", "", "", "", "Water Co", "active", "", "two weeks"},
		{"Phone", "bill", "200000", "monthly", "20", "", "", "", "Phone Co", "active", "", "30"},
	})
	sheet, err := newTestDBService(t, client, newFakeCache()).ReadRecurring(true)
	if err != nil || len(sheet.Items) != 1 || sheet.Items[0].GraceDays != 14 {
		t.Fatalf("sheet = %+v %v", sheet, err)
	}
	if len(sheet.Issues) != 2 || !strings.Contains(sheet.Issues[0].Message, `grace days "two weeks"`) || !strings.Contains(sheet.Issues[1].Message, "grace days must be from 0 to 21") {
		t.Fatalf("issues = %+v", sheet.Issues)
	}

	values := recurringRowValues(domain.RecurringInput{Name: "Rent", Kind: domain.RecurringBill, Amount: ptr(8_000_000.0), Day: 28, Match: "Landlord", GraceDays: 14}.Normalized())
	if len(values) != recurringColumns || values[11] != 14 {
		t.Fatalf("row values = %+v", values)
	}
	if values := recurringRowValues(domain.RecurringInput{Name: "Rent", Kind: domain.RecurringBill, Amount: ptr(1.0), Day: 1, Match: "x"}); values[11] != "" {
		t.Fatalf("no grace is a blank cell: %+v", values)
	}
}
