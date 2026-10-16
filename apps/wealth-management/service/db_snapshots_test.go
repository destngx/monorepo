package service

import (
	"apps/wealth-management/domain"
	"errors"
	"testing"
)

func testSnapshot(month string, netWorth float64) domain.MonthSnapshot {
	return domain.MonthSnapshot{
		Month: month, AsOf: month + "-28", NetWorth: netWorth, Assets: ptr(netWorth + 5_000_000), Liabilities: ptr(5_000_000.0),
		HealthScore: ptr(72.5), WrittenOn: "2026-11-05",
	}
}

// snapshotRow renders a sheet row as the unformatted read returns it.
func snapshotRow(month string, asOf string, netWorth string) []string {
	row := make([]string, 21)
	row[0], row[1], row[2], row[19] = month, asOf, netWorth, "2026-11-05"
	return row
}

func TestGivenNoSnapshotsTabWhenReadingThenItIsEmptyWithoutReadingTheRange(t *testing.T) {
	client := newFakeSheetClient()
	client.titles = []string{"Accounts"}

	sheet, err := newTestDBService(t, client, newFakeCache()).ReadSnapshots(true)

	if err != nil || sheet.Exists || len(sheet.Snapshots) != 0 {
		t.Fatalf("sheet %+v err %v", sheet, err)
	}
	for _, read := range client.reads {
		if read == snapshotsRange {
			t.Fatal("must not read a tab that does not exist")
		}
	}
}

func TestGivenHandEditedRowsWhenReadingSnapshotsThenTheyAreParsedTolerantly(t *testing.T) {
	client := newFakeSheetClient()
	client.titles = []string{snapshotsSheet}
	handTyped := []string{"46296", "", "30.000.000 ₫"} // 2026-10 typed by hand arrives as a date serial
	full := snapshotRow("2026-09", "2026-09-30", "1.2E+08")
	full[3], full[12], full[20] = "125000000", "35.5", "bonus month"
	client.ranges[snapshotsRange] = [][]string{
		{"Month", "As of", "Net worth"},
		{},
		full,
		handTyped,
		snapshotRow("2026-09", "2026-09-30", "1"),
		snapshotRow("2026-08", "2026-08-31", "lots"),
		snapshotRow("Aug", "", "1"),
	}

	sheet, err := newTestDBService(t, client, newFakeCache()).ReadSnapshots(true)

	if err != nil || !sheet.Exists || len(sheet.Snapshots) != 2 {
		t.Fatalf("sheet %+v err %v", sheet, err)
	}
	september := sheet.Snapshots[0]
	if september.Month != "2026-09" || september.Row != 4 || september.NetWorth != 120_000_000 || *september.Assets != 125_000_000 ||
		*september.SavingsRate != 35.5 || september.Note != "bonus month" || september.Liabilities != nil || september.WrittenOn != "2026-11-05" {
		t.Fatalf("september = %+v", september)
	}
	if october := sheet.Snapshots[1]; october.Month != "2026-10" || october.NetWorth != 30_000_000 || october.AsOf != "" {
		t.Fatalf("october = %+v", october)
	}
	if len(sheet.Issues) != 3 || sheet.Issues[0].Row != 6 || sheet.Issues[1].Row != 7 || sheet.Issues[2].Row != 8 {
		t.Fatalf("issues = %+v", sheet.Issues)
	}
	if sheet.Issues[1].Message != `month 2026-08: net worth "lots" is not an amount` {
		t.Fatalf("issue = %q", sheet.Issues[1].Message)
	}
}

func TestGivenNoTabWhenSavingASnapshotThenTheTabIsCreatedAndOneRowAppended(t *testing.T) {
	client := newFakeSheetClient()
	client.titles = []string{"Accounts"}
	client.ranges[snapshotsRange] = [][]string{}
	written := snapshotRow("2026-10", "2026-10-28", "100000000")
	client.ranges["'Snapshots'!A100:U100"] = [][]string{written}
	cache := newFakeCache()

	result, err := newTestDBService(t, client, cache).SaveSnapshot(testSnapshot("2026-10", 100_000_000))

	if err != nil || !result.Created || result.Snapshot.Row != 100 || result.Snapshot.NetWorth != 100_000_000 {
		t.Fatalf("result %+v err %v", result, err)
	}
	if len(client.created) != 1 || client.created[0] != snapshotsSheet || len(client.updates[0].values) != 21 {
		t.Fatalf("created %v updates %+v", client.created, client.updates)
	}
	if len(client.appends) != 1 || client.appends[0].anchor != snapshotsAnchor {
		t.Fatalf("appends %+v", client.appends)
	}
	row := client.appends[0].rows[0]
	if len(row) != 21 || row[0] != "'2026-10" || row[1] != "'2026-10-28" || row[2] != 100_000_000.0 || row[6] != "" || row[13] != 72.5 || row[19] != "'2026-11-05" {
		t.Fatalf("row %v", row)
	}
	if len(cache.invalidated) == 0 || cache.invalidated[len(cache.invalidated)-1] != cacheKeySnapshots {
		t.Fatalf("cache not invalidated: %v", cache.invalidated)
	}
}

func TestGivenTheMonthsRowWhenSavingASnapshotThenThatRowIsUpdatedAndTheNoteKept(t *testing.T) {
	client := newFakeSheetClient()
	client.titles = []string{snapshotsSheet}
	existing := snapshotRow("2026-10", "2026-10-15", "90000000")
	existing[20] = "my note"
	client.ranges[snapshotsRange] = [][]string{snapshotRow("2026-09", "2026-09-30", "80000000"), existing}
	updated := snapshotRow("2026-10", "2026-10-28", "100000000")
	updated[20] = "my note"
	client.ranges["'Snapshots'!A3:U3"] = [][]string{updated}

	result, err := newTestDBService(t, client, newFakeCache()).SaveSnapshot(testSnapshot("2026-10", 100_000_000))

	if err != nil || result.Created || result.Snapshot.Row != 3 || result.Snapshot.Note != "my note" {
		t.Fatalf("result %+v err %v", result, err)
	}
	if len(client.appends) != 0 || len(client.created) != 0 || len(client.updates) != 1 {
		t.Fatalf("appends %+v created %v updates %+v", client.appends, client.created, client.updates)
	}
	if update := client.updates[0]; update.rangeValue != "'Snapshots'!A3:T3" || len(update.values) != 20 {
		t.Fatalf("update %+v", update)
	}
}

func TestGivenABrokenRowForTheMonthWhenSavingThenItIsRepairedNotDuplicated(t *testing.T) {
	client := newFakeSheetClient()
	client.titles = []string{snapshotsSheet}
	client.ranges[snapshotsRange] = [][]string{snapshotRow("2026-10", "", "")}
	client.ranges["'Snapshots'!A2:U2"] = [][]string{snapshotRow("2026-10", "2026-10-28", "100000000")}

	result, err := newTestDBService(t, client, newFakeCache()).SaveSnapshot(testSnapshot("2026-10", 100_000_000))

	if err != nil || result.Created || len(client.appends) != 0 || client.updates[0].rangeValue != "'Snapshots'!A2:T2" {
		t.Fatalf("result %+v err %v updates %+v", result, err, client.updates)
	}
}

func TestGivenABadMonthOrAFailingWriteWhenSavingThenTheErrorIsReturned(t *testing.T) {
	client := newFakeSheetClient()
	client.titles = []string{snapshotsSheet}
	client.ranges[snapshotsRange] = [][]string{}
	svc := newTestDBService(t, client, newFakeCache())
	if _, err := svc.SaveSnapshot(testSnapshot("October", 1)); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("bad month: %v", err)
	}
	client.writeErr = errors.New("quota")
	if _, err := svc.SaveSnapshot(testSnapshot("2026-10", 1)); err == nil {
		t.Fatal("expected the write error")
	}
}
