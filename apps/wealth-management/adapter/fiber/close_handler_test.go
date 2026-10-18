package fiber

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/internal/fakes"
	"apps/wealth-management/service"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

func newCloseApp(t *testing.T, db *fakes.Database) *fiber.App {
	t.Helper()
	app := fiber.New()
	now := func() time.Time { return time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC) }
	log := logger.NewTestLogger(t)
	closeSvc := service.NewMonthlyCloseService(db, service.NewAnalyticsService(db, now), nil, nil, log, now)
	RegisterMonthlyCloseRoutes(NewRouteRegistrar(app, "/api", ""), NewCloseHandler(closeSvc, log), NewWriteGuard(staticToken(testToken), "0.0.0.0"))
	return app
}

func closeLedger() *fakes.Database {
	return &fakes.Database{Accounts: []domain.Account{{Name: "Bank", Type: domain.AccountTypeBank, Balance: 10_000_000}}}
}

func TestGivenTheLedgerWhenGettingTheCloseThenTheChecklistIsReturned(t *testing.T) {
	app := newCloseApp(t, closeLedger())

	status, body := call(t, app, http.MethodGet, "/api/close?month=2026-09", "")

	var close domain.MonthlyClose
	if status != http.StatusOK || json.Unmarshal([]byte(body), &close) != nil {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if close.Month != "2026-09" || len(close.Items) != 7 || !close.CanWriteSnapshot {
		t.Fatalf("close = %+v", close)
	}
	if status, _ := call(t, app, http.MethodGet, "/api/close?month=2027-01", ""); status != http.StatusBadRequest {
		t.Fatalf("a future month must be a 400, got %d", status)
	}
}

func TestGivenASnapshotWriteWhenPostingThenItIsGuardedCreatedThenUpdated(t *testing.T) {
	db := closeLedger()
	app := newCloseApp(t, db)

	if status, _ := call(t, app, http.MethodPost, "/api/close/snapshot", "", "Authorization", ""); status != http.StatusUnauthorized {
		t.Fatalf("writing a snapshot must be guarded, got %d", status)
	}
	status, body := call(t, app, http.MethodPost, "/api/close/snapshot", "")
	if status != http.StatusCreated || !strings.Contains(body, `"month":"2026-10"`) || !strings.Contains(body, `"netWorth":10000000`) {
		t.Fatalf("create: %d %s", status, body)
	}
	if status, body := call(t, app, http.MethodPost, "/api/close/snapshot", `{"month":"2026-10"}`); status != http.StatusOK || !strings.Contains(body, `"created":false`) {
		t.Fatalf("update: %d %s", status, body)
	}
	if status, _ := call(t, app, http.MethodPost, "/api/close/snapshot", `{"month":"2026-07"}`); status != http.StatusBadRequest {
		t.Fatalf("an old month must be a 400, got %d", status)
	}
	if len(db.Snapshots.Snapshots) != 1 {
		t.Fatalf("rows = %+v", db.Snapshots.Snapshots)
	}
}

func TestGivenSnapshotsWhenGettingTheHistoryThenPointsCarryTheirSource(t *testing.T) {
	db := closeLedger()
	db.Snapshots = domain.SnapshotSheet{Exists: true, Snapshots: []domain.MonthSnapshot{{Month: "2026-09", AsOf: "2026-09-30", NetWorth: 8_000_000}}}
	app := newCloseApp(t, db)

	status, body := call(t, app, http.MethodGet, "/api/net-worth/history", "")

	var history domain.NetWorthHistory
	if status != http.StatusOK || json.Unmarshal([]byte(body), &history) != nil {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if len(history.Points) != 2 || history.Points[0].Source != domain.NetWorthFromSnapshot || history.Points[1].Source != domain.NetWorthLive {
		t.Fatalf("history = %+v", history)
	}
}
