package local_workbook

import (
	"apps/wealth-management/port"
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sync"
	"testing"
)

var _ port.DatabaseClient = (*Client)(nil)

// fixtureParts is a minimal workbook as Google Sheets exports it: shared and
// inline strings, numbers, booleans, errors, formulas with and without a
// cached value, skipped rows and cells, and a tab whose sheet name ("Loan /
// Dept's") lost the "/" Excel forbids.
var fixtureParts = map[string]string{
	"xl/workbook.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <sheets>
    <sheet name="Accounts" sheetId="1" r:id="rId1"/>
    <sheet name="Loan  Dept's" sheetId="2" r:id="rId2"/>
  </sheets>
</workbook>`,
	"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="/xl/worksheets/sheet2.xml"/>
  <Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/sharedStrings" Target="sharedStrings.xml"/>
</Relationships>`,
	"xl/sharedStrings.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="4" uniqueCount="4">
  <si><t>Name</t></si>
  <si><t>Open</t></si>
  <si><t xml:space="preserve">Cash </t></si>
  <si><r><t>Ba</t></r><r><rPr><b/></rPr><t>nk</t></r><rPh sb="0" eb="1"><t>x</t></rPh></si>
</sst>`,
	"xl/worksheets/sheet1.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheetData>
    <row r="1">
      <c r="A1" t="s"><v>0</v></c>
      <c r="B1" t="inlineStr"><is><t>Balance</t></is></c>
      <c r="C1" t="s"><v>1</v></c>
    </row>
    <row r="2">
      <c r="A2" t="s"><v>2</v></c>
      <c r="B2"><v>1500000</v></c>
      <c r="C2" t="b"><v>1</v></c>
    </row>
    <row r="4">
      <c t="s"><v>3</v></c>
      <c><f>B2/3000000</f><v>0.5</v></c>
      <c t="b"><v>0</v></c>
      <c t="e"><v>#N/A</v></c>
      <c r="F4" t="str"><f>A4&amp;"!"</f><v>Bank!</v></c>
    </row>
    <row r="5">
      <c r="B5"><f>NOW()</f></c>
    </row>
  </sheetData>
</worksheet>`,
	"xl/worksheets/sheet2.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheetData><row r="1"><c r="A1" s="3"><v>46309</v></c></row></sheetData>
</worksheet>`,
}

func writeFixture(t *testing.T) string {
	t.Helper()
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	for name, content := range fixtureParts {
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
	path := filepath.Join(t.TempDir(), "Money Manager.xlsx")
	if err := os.WriteFile(path, buffer.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func openFixture(t *testing.T) *Client {
	t.Helper()
	client, err := Open(writeFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func read(t *testing.T, client *Client, rangeValue string) [][]string {
	t.Helper()
	rows, err := client.ReadSheet(rangeValue)
	if err != nil {
		t.Fatalf("ReadSheet(%q): %v", rangeValue, err)
	}
	return rows
}

func assertRows(t *testing.T, got [][]string, want [][]string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rows = %q, want %q", got, want)
	}
}

func TestGivenAnExportedWorkbookWhenReadingThenCellsRenderLikeUnformattedSheetsValues(t *testing.T) {
	client := openFixture(t)

	assertRows(t, read(t, client, "Accounts!A1:F"), [][]string{
		{"Name", "Balance", "Open"},
		{"Cash ", "1.5e+06", "true"},
		{},
		{"Bank", "0.5", "false", "#N/A", "", "Bank!"},
	})
}

func TestGivenARangeWhenReadingThenOnlyItsCellsComeBackWithoutTrailingBlanks(t *testing.T) {
	client := openFixture(t)

	cases := map[string][][]string{
		"Accounts!B2:C2":       {{"1.5e+06", "true"}},
		"Accounts!A2:A":        {{"Cash "}, {}, {"Bank"}},
		"Accounts!C1:D":        {{"Open"}, {"true"}, {}, {"false", "#N/A"}},
		"Accounts!A7:Z":        {},
		"accounts!A1":          {{"Name"}},
		"'Loan / Dept''s'!A1":  {{"46309"}},
		"'Loan / Dept''s'!A:A": {{"46309"}},
	}
	for rangeValue, want := range cases {
		t.Run(rangeValue, func(t *testing.T) {
			assertRows(t, read(t, client, rangeValue), want)
		})
	}
}

func TestGivenASheetTabNameExcelCannotHoldWhenReadingThenTheExportedTabMatches(t *testing.T) {
	client := openFixture(t)

	for _, rangeValue := range []string{"'Loan / Dept''s'!A1", "'Loan  Dept''s'!A1", "'loan / dept''s'!A1"} {
		assertRows(t, read(t, client, rangeValue), [][]string{{"46309"}})
	}
	if _, err := client.ReadSheet("'Loan Dept''s'!A1"); err == nil {
		t.Fatal("a different tab name matched")
	}
}

func TestGivenSeveralRangesWhenBatchReadingThenRowsKeepTheRequestOrder(t *testing.T) {
	client := openFixture(t)

	got, err := client.ReadSheets([]string{"'Loan / Dept''s'!A1", "Accounts!A1"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, [][][]string{{{"46309"}}, {{"Name"}}}) {
		t.Fatalf("ReadSheets = %q", got)
	}
}

func TestGivenAnUnknownTabOrBadRangeWhenReadingThenItFails(t *testing.T) {
	client := openFixture(t)

	for _, rangeValue := range []string{"Missing!A1", "Accounts!1A", "'Accounts!A1", "Accounts!A2:"} {
		if _, err := client.ReadSheet(rangeValue); err == nil {
			t.Errorf("ReadSheet(%q) succeeded", rangeValue)
		}
	}
	if _, err := client.ReadSheets([]string{"Accounts!A1", "Missing!A1"}); err == nil {
		t.Error("ReadSheets with an unknown tab succeeded")
	}
}

func TestGivenUserEnteredValuesWhenUpdatingThenTheyAreParsedLikeSheets(t *testing.T) {
	client := openFixture(t)

	err := client.UpdateRow("Accounts!A2:I2", []any{"14/10/2026", "42", "=SUM(B2)", "'0123", "1,5", true, 2000.0, 7, ""})
	if err != nil {
		t.Fatal(err)
	}

	assertRows(t, read(t, client, "Accounts!A2:I2"), [][]string{{"46309", "42", "", "0123", "1,5", "true", "2000", "7"}})
}

func TestGivenAnEmptyValueWhenUpdatingThenTheCellIsCleared(t *testing.T) {
	client := openFixture(t)

	if err := client.UpdateRow("Accounts!B2:C2", []any{"", nil}); err != nil {
		t.Fatal(err)
	}

	assertRows(t, read(t, client, "Accounts!A2:C2"), [][]string{{"Cash "}})
}

func TestGivenAnAnchorColumnWhenWritingRowsThenTheyLandBelowItsLastValue(t *testing.T) {
	client := openFixture(t)

	row, err := client.WriteRowsAtFirstEmptyRow("Accounts", "Accounts!A2:A", [][]any{{"Wallet", 10.0}, {"Card", 20.0}})
	if err != nil {
		t.Fatal(err)
	}

	if row != 5 {
		t.Fatalf("first row written = %d, want 5", row)
	}
	assertRows(t, read(t, client, "Accounts!A5:B6"), [][]string{{"Wallet", "10"}, {"Card", "20"}})
	if _, err := client.WriteRowsAtFirstEmptyRow("Accounts", "Accounts!A2:A", nil); err == nil {
		t.Fatal("writing no rows succeeded")
	}
}

func TestGivenConcurrentAppendsWhenWritingThenEveryRowGetsItsOwnLine(t *testing.T) {
	client := openFixture(t)

	const writers = 20
	rows := make([]int, writers)
	var group sync.WaitGroup
	for i := range writers {
		group.Go(func() {
			row, err := client.WriteRowsAtFirstEmptyRow("Accounts", "Accounts!A2:A", [][]any{{"concurrent"}})
			if err != nil {
				t.Error(err)
			}
			rows[i] = row
		})
	}
	group.Wait()

	slices.Sort(rows)
	for i, row := range rows {
		if row != 5+i {
			t.Fatalf("rows written = %v, want 5..%d", rows, 4+writers)
		}
	}
}

func TestGivenANewTabWhenCreatingThenItHasTheHeaderAndNamesMustBeUnique(t *testing.T) {
	client := openFixture(t)

	if err := client.CreateSheet("CaptureRules", []any{"Kind", "Pattern"}); err != nil {
		t.Fatal(err)
	}

	titles, err := client.SheetTitles()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(titles, []string{"Accounts", "Loan  Dept's", "CaptureRules"}) {
		t.Fatalf("titles = %q", titles)
	}
	assertRows(t, read(t, client, "CaptureRules!A1:D"), [][]string{{"Kind", "Pattern"}})
	if err := client.CreateSheet("capturerules", nil); err == nil {
		t.Fatal("a duplicate tab name was accepted")
	}
}

func TestGivenTheWorkbookWhenAskingItsTitleThenItIsTheFileName(t *testing.T) {
	client := openFixture(t)

	title, err := client.SpreadsheetTitle()
	if err != nil || title != "Money Manager" {
		t.Fatalf("title = %q, %v", title, err)
	}
}

func TestGivenWritesWhenTheRunEndsThenTheSourceFileIsUnchanged(t *testing.T) {
	path := writeFixture(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	client, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}

	if err := client.UpdateRow("Accounts!A1", []any{"changed"}); err != nil {
		t.Fatal(err)
	}
	if err := client.CreateSheet("New", []any{"x"}); err != nil {
		t.Fatal(err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("the source workbook was modified")
	}
}

func TestGivenAFileThatIsNotAWorkbookWhenOpeningThenItFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.xlsx")
	if err := os.WriteFile(path, []byte("not a zip"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, candidate := range []string{path, filepath.Join(t.TempDir(), "missing.xlsx")} {
		if _, err := Open(candidate); err == nil {
			t.Errorf("Open(%q) succeeded", candidate)
		}
	}
}
