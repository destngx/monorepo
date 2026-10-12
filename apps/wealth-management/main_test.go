package main

import (
	fiberAdapter "apps/wealth-management/adapter/fiber"
	loggerAdapter "apps/wealth-management/adapter/logger"
	mcpAdapter "apps/wealth-management/adapter/mcp"
	"apps/wealth-management/domain"
	"apps/wealth-management/service"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestGivenTestModeWhenCheckingSpreadsheetThenOnlyCopiesPass(t *testing.T) {
	title := func(name string, err error) func() (string, error) {
		return func() (string, error) { return name, err }
	}
	if err := checkTestSheet("copy", title("wm-e2e-run-20261013T000000Z", nil)); err != nil {
		t.Fatalf("copy rejected: %v", err)
	}
	if err := checkTestSheet("prod", title("Wealth 2026", nil)); !errors.Is(err, domain.ErrNotTestSheet) {
		t.Fatalf("production title accepted: %v", err)
	}
	unreachable := errors.New("forbidden")
	if err := checkTestSheet("copy", title("", unreachable)); !errors.Is(err, unreachable) {
		t.Fatalf("title read error lost: %v", err)
	}
}

func TestGivenNoGoogleCredentialsWhenStartingThenSheetRoutesAnswer503NamingTheMissingVars(t *testing.T) {
	for _, name := range []string{"GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET", "GOOGLE_REFRESH_TOKEN", "GOOGLE_SHEETS_ID"} {
		t.Setenv(name, "")
	}
	log := loggerAdapter.NewTestLogger(t)
	health := service.NewHealthService()
	app := fiber.New()
	api := fiberAdapter.NewRouteRegistrar(app, "/api", "")

	dbSvc, analyticsSvc := setupSheets(context.Background(), log, health, api, fiberAdapter.NewWriteGuard("", "127.0.0.1"), mcpAdapter.NewServer(health, log), nil, domain.CardConfig{})

	if dbSvc != nil || analyticsSvc != nil {
		t.Fatal("services must be nil without credentials")
	}
	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/accounts", nil), -1)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusServiceUnavailable || !strings.Contains(string(body), `"code":"SHEETS_NOT_CONFIGURED"`) || !strings.Contains(string(body), "GOOGLE_SHEETS_ID") {
		t.Fatalf("status = %d, body = %s", response.StatusCode, body)
	}
}
