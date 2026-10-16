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

func newRecommendationsApp(t *testing.T, db *fakes.Database) *fiber.App {
	t.Helper()
	app := fiber.New()
	now := func() time.Time { return time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC) }
	recommendations := service.NewRecommendationService(db, service.NewAnalyticsService(db, now), nil, logger.NewTestLogger(t), now)
	RegisterRecommendationRoutes(NewRouteRegistrar(app, "/api", ""), NewRecommendationHandler(recommendations, logger.NewTestLogger(t)), NewWriteGuard(testToken, "0.0.0.0"))
	return app
}

func TestGivenRecommendationsWhenListingAndChangingStatusThenWritesAreGuardedAndETagChecked(t *testing.T) {
	impact := 1_200_000.0
	db := &fakes.Database{Recommendations: domain.RecommendationSheet{Exists: true, Items: []domain.Recommendation{
		{ID: "rec-0a1b2c3d", Row: 2, Created: "2026-10-01", Source: "budget-review", Title: "Cut dining", Action: "Cook", ImpactVND: &impact, Status: domain.RecommendationOpen, Page: "/budget", ETag: "etag-1"},
		{ID: "rec-11112222", Row: 3, Created: "2026-10-01", Source: "year-review", Title: "Save more", Action: "Move money", Status: domain.RecommendationOpen, Page: "/cash-flow", ETag: "etag-2"},
	}}}
	app := newRecommendationsApp(t, db)

	status, body := call(t, app, http.MethodGet, "/api/recommendations?page=/budget", "")
	var list domain.RecommendationList
	if status != http.StatusOK || json.Unmarshal([]byte(body), &list) != nil || len(list.Items) != 1 || list.Summary.OpenImpactVND != 1_200_000 {
		t.Fatalf("list: %d %s", status, body)
	}
	if status, _ := call(t, app, http.MethodGet, "/api/recommendations?status=later", ""); status != http.StatusBadRequest {
		t.Fatalf("a bad filter must be a 400, got %d", status)
	}

	if status, _ := call(t, app, http.MethodPost, "/api/recommendations/rec-0a1b2c3d/status", `{"status":"done","etag":"etag-1"}`, "Authorization", ""); status != http.StatusUnauthorized {
		t.Fatalf("a status change must be guarded, got %d", status)
	}
	status, body = call(t, app, http.MethodPost, "/api/recommendations/rec-0a1b2c3d/status", `{"status":"snoozed","snoozeDays":7}`, "If-Match", `"etag-1"`)
	if status != http.StatusOK || !strings.Contains(body, `"status":"snoozed"`) || !strings.Contains(body, `"snoozedUntil":"2026-10-19"`) {
		t.Fatalf("snooze: %d %s", status, body)
	}
	if status, _ := call(t, app, http.MethodPost, "/api/recommendations/rec-0a1b2c3d/status", `{"status":"done","etag":"etag-1"}`); status != http.StatusConflict {
		t.Fatalf("a stale etag must be a 409, got %d", status)
	}
	if status, _ := call(t, app, http.MethodPost, "/api/recommendations/rec-ffffffff/status", `{"status":"done","etag":"x"}`); status != http.StatusNotFound {
		t.Fatalf("an unknown id must be a 404, got %d", status)
	}
	if status, _ := call(t, app, http.MethodPost, "/api/recommendations/refresh", "", "Authorization", ""); status != http.StatusUnauthorized {
		t.Fatalf("refresh writes, so it must be guarded, got %d", status)
	}
	if status, body := call(t, app, http.MethodPost, "/api/recommendations/refresh?status=open", ""); status != http.StatusOK || !strings.Contains(body, `"open":1`) {
		t.Fatalf("refresh: %d %s", status, body)
	}
}
