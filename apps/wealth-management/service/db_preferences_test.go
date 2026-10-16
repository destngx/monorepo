package service

import (
	"apps/wealth-management/adapter/db/local_workbook"
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"archive/zip"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func preferenceValue(key domain.PreferenceKey, value float64) domain.PreferenceValue {
	return domain.PreferenceValue{Key: key, Value: &value}
}

func targetsUpdate(cash, funds, crypto float64) domain.PreferencesUpdate {
	return domain.PreferencesUpdate{Values: []domain.PreferenceValue{
		preferenceValue(domain.PrefAllocationTargetCash, cash),
		preferenceValue(domain.PrefAllocationTargetFunds, funds),
		preferenceValue(domain.PrefAllocationTargetCrypto, crypto),
	}}
}

func TestGivenNoPreferencesTabWhenListingThenDefaultsAreReturnedWithoutReadingIt(t *testing.T) {
	client := newFakeSheetClient()
	client.titles = []string{"Accounts"}
	prefs, err := newTestDBService(t, client, newFakeCache()).ListPreferences(true)
	if err != nil || prefs.TabExists || prefs.AllocationTargets != nil || prefs.GoalPool != domain.DefaultGoalPoolSettings() {
		t.Fatalf("prefs %+v err %v", prefs, err)
	}
	for _, read := range client.reads {
		if read == preferencesRange {
			t.Fatal("must not read a tab that does not exist")
		}
	}
}

func TestGivenAPreferencesTabWhenListingThenItIsParsedAndCached(t *testing.T) {
	client := newFakeSheetClient()
	client.titles = []string{preferencesSheet}
	client.ranges[preferencesRange] = [][]string{{"goal.emergencyMonths", "4"}}
	cache := newFakeCache()
	svc := newTestDBService(t, client, cache)
	prefs, err := svc.ListPreferences(false)
	if err != nil || !prefs.TabExists || prefs.GoalPool.EmergencyMonths != 4 || prefs.Entries[0].Row != 2 {
		t.Fatalf("prefs %+v err %v", prefs, err)
	}
	if _, ok := cache.values[cacheKeyPreferences]; !ok {
		t.Fatal("preferences must be cached like the other tabs")
	}
}

func TestGivenNoTabWhenSavingThenTheTabIsCreatedAndOnlyItsRowsAreWritten(t *testing.T) {
	client := newFakeSheetClient()
	client.titles = []string{"Accounts"}
	client.ranges[preferencesRange] = [][]string{}
	cache := newFakeCache()
	if _, err := newTestDBService(t, client, cache).SavePreferences(targetsUpdate(60, 30, 10)); err != nil {
		t.Fatal(err)
	}
	if len(client.created) != 1 || client.created[0] != preferencesSheet {
		t.Fatalf("created %v", client.created)
	}
	if len(client.appends) != 1 || client.appends[0].sheetName != preferencesSheet || len(client.appends[0].rows) != 3 {
		t.Fatalf("appends %+v", client.appends)
	}
	row := client.appends[0].rows[0]
	if row[0] != "allocation.target.cash" || row[1] != 60.0 || row[2] == "" {
		t.Fatalf("row %v", row)
	}
	for _, update := range client.updates {
		if update.rangeValue != preferencesSheet+"!A1" {
			t.Fatalf("only the new tab's header may be written: %+v", update)
		}
	}
	if len(cache.invalidated) == 0 || cache.invalidated[len(cache.invalidated)-1] != cacheKeyPreferences {
		t.Fatalf("cache not invalidated: %v", cache.invalidated)
	}
}

func TestGivenAnExistingKeyWhenSavingThenItsRowIsUpdatedAndTheNoteKept(t *testing.T) {
	client := newFakeSheetClient()
	client.titles = []string{preferencesSheet}
	client.ranges[preferencesRange] = [][]string{{"Key", "Value", "Note"}, {"goal.emergencyMonths", "6", "my note"}}
	update := domain.PreferencesUpdate{Values: []domain.PreferenceValue{
		preferenceValue(domain.PrefGoalEmergencyMonths, 8),
		preferenceValue(domain.PrefGoalSpendingBufferMonths, 2),
	}}
	if _, err := newTestDBService(t, client, newFakeCache()).SavePreferences(update); err != nil {
		t.Fatal(err)
	}
	if len(client.created) != 0 || len(client.updates) != 1 || client.updates[0].rangeValue != "Preferences!A3:B3" || client.updates[0].values[1] != 8.0 {
		t.Fatalf("updates %+v", client.updates)
	}
	if len(client.appends) != 1 || client.appends[0].rows[0][0] != "goal.spendingBufferMonths" {
		t.Fatalf("appends %+v", client.appends)
	}
}

func TestGivenAnInvalidSaveWhenSavingThenNothingIsWritten(t *testing.T) {
	client := newFakeSheetClient()
	client.titles = []string{"Accounts"}
	_, err := newTestDBService(t, client, newFakeCache()).SavePreferences(targetsUpdate(60, 30, 20))
	if !errors.Is(err, domain.ErrValidation) || len(client.created) != 0 || len(client.appends) != 0 || len(client.updates) != 0 {
		t.Fatalf("err %v created %v appends %v", err, client.created, client.appends)
	}
}

func TestGivenAFailingWriteWhenSavingThenTheErrorIsReturned(t *testing.T) {
	client := newFakeSheetClient()
	client.titles = []string{preferencesSheet}
	client.ranges[preferencesRange] = [][]string{{"goal.emergencyMonths", "6"}}
	client.writeErr = errors.New("quota")
	update := domain.PreferencesUpdate{Values: []domain.PreferenceValue{preferenceValue(domain.PrefGoalEmergencyMonths, 3)}}
	if _, err := newTestDBService(t, client, newFakeCache()).SavePreferences(update); err == nil {
		t.Fatal("expected the write error")
	}
}

// emptyWorkbook writes an .xlsx with one Accounts tab, as the local
// workbook adapter serves it in tests (WM-TEST-1).
func emptyWorkbook(t *testing.T) string {
	t.Helper()
	parts := map[string]string{
		"xl/workbook.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <sheets><sheet name="Accounts" sheetId="1" r:id="rId1"/></sheets>
</workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
</Relationships>`,
		"xl/worksheets/sheet1.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheetData><row r="1"><c r="A1" t="inlineStr"><is><t>Account</t></is></c></row></sheetData>
</worksheet>`,
	}
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	for name, content := range parts {
		part, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "book.xlsx")
	if err := os.WriteFile(path, buffer.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestGivenTheLocalWorkbookWhenSavingPreferencesThenTheyRoundTripAndOnlyThePreferencesTabChanges(t *testing.T) {
	client, err := local_workbook.Open(emptyWorkbook(t))
	if err != nil {
		t.Fatal(err)
	}
	svc := NewDatabaseService(client, nil, logger.NewTestLogger(t), DatabaseServiceOptions{Now: func() time.Time { return fixedNow }})

	saved, err := svc.SavePreferences(targetsUpdate(70, 20, 10))
	if err != nil {
		t.Fatal(err)
	}
	if !saved.TabExists || saved.AllocationTargets == nil || *saved.AllocationTargets != (domain.AllocationTargets{Cash: 70, Funds: 20, Crypto: 10}) {
		t.Fatalf("saved %+v", saved)
	}
	header, err := client.ReadSheet("Preferences!A1:C1")
	if err != nil || len(header) != 1 || header[0][0] != "Key" || header[0][2] != "Note" {
		t.Fatalf("header %v err %v", header, err)
	}

	// a second save updates the rows in place and appends the new key
	update := targetsUpdate(50, 40, 10)
	update.Values = append(update.Values, preferenceValue(domain.PrefGoalEmergencyMonths, 9))
	if _, err := svc.SavePreferences(update); err != nil {
		t.Fatal(err)
	}
	prefs, err := svc.ListPreferences(true)
	if err != nil {
		t.Fatal(err)
	}
	if prefs.AllocationTargets.Cash != 50 || prefs.GoalPool.EmergencyMonths != 9 || prefs.GoalPool.SpendingBufferMonths != 1 || len(prefs.Entries) != 4 || len(prefs.Issues) != 0 {
		t.Fatalf("prefs %+v", prefs)
	}

	// clearing the targets blanks their values
	clear := domain.PreferencesUpdate{Values: []domain.PreferenceValue{
		{Key: domain.PrefAllocationTargetCash}, {Key: domain.PrefAllocationTargetFunds}, {Key: domain.PrefAllocationTargetCrypto},
	}}
	if prefs, err = svc.SavePreferences(clear); err != nil || prefs.AllocationTargets != nil {
		t.Fatalf("prefs %+v err %v", prefs, err)
	}

	titles, err := client.SheetTitles()
	if err != nil || len(titles) != 2 {
		t.Fatalf("titles %v err %v", titles, err)
	}
	accounts, err := client.ReadSheet("Accounts!A1:Z10")
	if err != nil || len(accounts) != 1 || len(accounts[0]) != 1 {
		t.Fatalf("the Accounts tab must be untouched: %v err %v", accounts, err)
	}
}
