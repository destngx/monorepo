package local_workbook

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/service"
	"testing"
)

// The Snapshots writes round-trip through the local workbook (§14.3): the
// tab is created on the first snapshot, and writing the same month again
// updates its row instead of adding one.
func TestGivenTheLocalWorkbookWhenWritingSnapshotsThenEachMonthKeepsOneRow(t *testing.T) {
	client := openFixture(t)
	db := service.NewDatabaseService(client, nil, logger.NewTestLogger(t), service.DatabaseServiceOptions{})
	score := 70.5

	before, err := db.ReadSnapshots(true)
	if err != nil || before.Exists {
		t.Fatalf("before the first snapshot: %+v %v", before, err)
	}
	first, err := db.SaveSnapshot(domain.MonthSnapshot{Month: "2026-10", AsOf: "2026-10-15", NetWorth: 90_000_000, HealthScore: &score, WrittenOn: "2026-10-15"})
	if err != nil || !first.Created || first.Snapshot.Row != 2 || first.Snapshot.AsOf != "2026-10-15" || *first.Snapshot.HealthScore != 70.5 {
		t.Fatalf("first = %+v %v", first, err)
	}
	header := read(t, client, "'Snapshots'!A1:U1")
	if len(header) != 1 || len(header[0]) != 21 || header[0][0] != "Month" || header[0][20] != "Note" {
		t.Fatalf("header = %q", header)
	}
	if _, err := db.SaveSnapshot(domain.MonthSnapshot{Month: "2026-09", AsOf: "2026-09-30", NetWorth: 80_000_000, WrittenOn: "2026-10-15"}); err != nil {
		t.Fatal(err)
	}
	again, err := db.SaveSnapshot(domain.MonthSnapshot{Month: "2026-10", AsOf: "2026-10-31", NetWorth: 95_000_000, WrittenOn: "2026-11-01"})
	if err != nil || again.Created || again.Snapshot.Row != 2 || again.Snapshot.NetWorth != 95_000_000 || again.Snapshot.HealthScore != nil {
		t.Fatalf("again = %+v %v", again, err)
	}

	after, err := db.ReadSnapshots(true)
	if err != nil || len(after.Snapshots) != 2 || len(after.Issues) != 0 {
		t.Fatalf("after = %+v %v", after, err)
	}
	if october, _ := after.ForMonth("2026-10"); october.AsOf != "2026-10-31" || october.WrittenOn != "2026-11-01" {
		t.Fatalf("october = %+v", october)
	}
}
