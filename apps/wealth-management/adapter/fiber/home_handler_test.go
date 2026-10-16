package fiber

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/internal/fakes"
	"apps/wealth-management/service"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

func newHomeApp(t *testing.T, db *fakes.Database) *fiber.App {
	t.Helper()
	app := fiber.New()
	now := func() time.Time { return time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC) }
	analytics := service.NewAnalyticsService(db, now)
	home := service.NewHomeService(service.HomeSources{
		Analytics: analytics, Ledger: db,
		Recommendations: service.NewRecommendationService(db, analytics, nil, logger.NewTestLogger(t), now),
	}, logger.NewTestLogger(t), now)
	RegisterHomeRoutes(NewRouteRegistrar(app, "/api", ""), NewHomeHandler(home, logger.NewTestLogger(t)))
	return app
}

func TestGivenAnUncategorizedRowWhenGettingNeedsYouThenTheRankedInboxIsReturned(t *testing.T) {
	payment := 150_000.0
	db := &fakes.Database{Transactions: []domain.Transaction{{ID: "row-2", AccountName: "Bank", Date: time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC), Payee: "Shop", Payment: &payment}}}
	status, body := call(t, newHomeApp(t, db), http.MethodGet, "/api/home/needs-you", "")
	var inbox domain.NeedsYou
	if status != http.StatusOK || json.Unmarshal([]byte(body), &inbox) != nil {
		t.Fatalf("needs you: %d %s", status, body)
	}
	if len(inbox.Items) != 1 || inbox.Items[0].Kind != domain.NeedsYouUncategorized || inbox.Items[0].Route != "/transactions?uncategorized=1" || len(inbox.Unavailable) != 0 {
		t.Fatalf("inbox = %+v", inbox)
	}
}

func TestGivenTheSheetUnreachableWhenGettingNeedsYouThenTheErrorIsMapped(t *testing.T) {
	db := &fakes.Database{Err: domain.ErrNotFound}
	if status, body := call(t, newHomeApp(t, db), http.MethodGet, "/api/home/needs-you", ""); status != http.StatusNotFound {
		t.Fatalf("every source failing must fail the call: %d %s", status, body)
	}
}
