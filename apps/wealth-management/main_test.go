package main

import (
	configAdapter "apps/wealth-management/adapter/config"
	fiberAdapter "apps/wealth-management/adapter/fiber"
	loggerAdapter "apps/wealth-management/adapter/logger"
	mcpAdapter "apps/wealth-management/adapter/mcp"
	"apps/wealth-management/service"
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestGivenATestWorkbookWhenBuildingTheSheetsClientThenGoogleIsNeverUsed(t *testing.T) {
	log := loggerAdapter.NewTestLogger(t)
	path := filepath.Join(t.TempDir(), "Book.xlsx")
	writeMinimalWorkbook(t, path)
	t.Setenv("WM_TEST_WORKBOOK", path)

	// Google settings are present but must be ignored.
	client, spreadsheetID, err := newSheetsClient(context.Background(), log, configAdapter.Values{"GOOGLE_SHEETS_ID": "production"})
	if err != nil {
		t.Fatal(err)
	}
	if spreadsheetID != "" {
		t.Fatalf("spreadsheet id = %q, want none for a local workbook", spreadsheetID)
	}
	if title, err := client.SpreadsheetTitle(); err != nil || title != "Book" {
		t.Fatalf("title = %q, %v", title, err)
	}
}

func TestGivenAnUnreadableTestWorkbookWhenBuildingTheSheetsClientThenItFailsWithoutFallingBack(t *testing.T) {
	log := loggerAdapter.NewTestLogger(t)
	t.Setenv("WM_TEST_WORKBOOK", filepath.Join(t.TempDir(), "missing.xlsx"))

	_, _, err := newSheetsClient(context.Background(), log, configAdapter.Values{})
	if !errors.Is(err, errTestWorkbook) {
		t.Fatalf("err = %v, want errTestWorkbook", err)
	}
}

// writeMinimalWorkbook writes an .xlsx with one empty tab.
func writeMinimalWorkbook(t *testing.T, path string) {
	t.Helper()
	parts := map[string]string{
		"xl/workbook.xml":            `<workbook xmlns:r="r"><sheets><sheet name="Accounts" r:id="rId1"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<Relationships><Relationship Id="rId1" Target="worksheets/sheet1.xml"/></Relationships>`,
		"xl/worksheets/sheet1.xml":   `<worksheet><sheetData/></worksheet>`,
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
	if err := os.WriteFile(path, buffer.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestGivenNoGoogleCredentialsWhenStartingThenSheetRoutesAnswer503NamingTheMissingSettings(t *testing.T) {
	log := loggerAdapter.NewTestLogger(t)
	health := service.NewHealthService()
	app := fiber.New()
	api := fiberAdapter.NewRouteRegistrar(app, "/api", "")

	sheets := setupSheets(context.Background(), log, health, api, fiberAdapter.NewWriteGuard(nil, "127.0.0.1"), mcpAdapter.NewServer(health, log), nil, nil, configAdapter.Values{})

	if sheets.db != nil || sheets.analytics != nil || sheets.portfolio != nil {
		t.Fatal("services must be nil without credentials")
	}
	for _, path := range []string{"/api/accounts", "/api/preferences", "/api/investments/allocation"} {
		response, err := app.Test(httptest.NewRequest(http.MethodGet, path, nil), -1)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusServiceUnavailable || !strings.Contains(string(body), `"code":"SHEETS_NOT_CONFIGURED"`) || !strings.Contains(string(body), "GOOGLE_SHEETS_ID") {
			t.Fatalf("%s: status = %d, body = %s", path, response.StatusCode, body)
		}
	}
}

func TestGivenAPortWhenDerivingTheGoogleRedirectThenItIsTheLoopbackCallback(t *testing.T) {
	if got := defaultGoogleRedirect(8080); got != "http://127.0.0.1:8080/api/auth/google/callback" {
		t.Fatalf("redirect = %q", got)
	}
}
