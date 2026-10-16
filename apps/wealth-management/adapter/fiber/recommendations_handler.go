package fiber

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/port"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
)

// Recommendations list (ROADMAP R7, WM-REC-1..). Reading writes nothing;
// refresh (which runs the engine rules) and status changes go through the
// WriteGuard and write only the app-managed Recommendations tab.

const recommendationsScope = "recommendations_handler"

const recommendationsSummary = "The recommendations list. items: id (rec-<code>), created, source (a review: account-review, budget-review, " +
	"transaction-review, credit-summary, year-review, financial-health, intelligence-briefing, think-tank; or an engine rule: rule:rebalance, " +
	"rule:goal-plan-short, rule:lent-overdue, rule:idle-cash), title, action, impactVnd (estimated VND per year, null when not estimated), " +
	"evidencePath (app route) and evidence (facts), status (open, snoozed, done, dismissed; a snooze that ended reads as open), " +
	"snoozedUntil, statusChanged, outcome (written by a later review, or resolved when a rule's condition is gone), page (the " +
	"app route it belongs to) and etag. Ordered open (largest impact first), snoozed, done, dismissed. summary counts each " +
	"status and sums the open items' impact. issues lists rows the reader left out. tabExists is false until the first item."

type RecommendationHandler struct {
	recommendations port.RecommendationService
	log             *logger.Logger
}

func NewRecommendationHandler(recommendations port.RecommendationService, log *logger.Logger) *RecommendationHandler {
	return &RecommendationHandler{recommendations: recommendations, log: log}
}

func recommendationFilterParams() []RouteParameter {
	return []RouteParameter{
		QueryParam("status", "string", "open, snoozed, done or dismissed; omit for every status."),
		QueryParam("page", "string", "App route, e.g. /budget; omit for every page."),
		QueryParam("source", "string", "Review kind or rule:<id>; omit for every source."),
	}
}

func RegisterRecommendationRoutes(api RouteRegistrar, h *RecommendationHandler, guard WriteGuard) {
	recommendations := api.Group("/recommendations", "recommendations")
	recommendations.Get("/", h.List, RouteMeta{
		Summary:     recommendationsSummary + " Reads only; POST /api/recommendations/refresh also runs the engine rules.",
		QueryParams: append(recommendationFilterParams(), forceParam()),
	})
	recommendations.Post("/refresh", guard.Wrap(h.Refresh), RouteMeta{
		Summary: "Runs the engine rules, then returns the list as GET /api/recommendations does. Rules: rebalance (an allocation " +
			"class drifts beyond the threshold), goal-plan-short (dated goals need more a month than is saved; impact = the gap x 12), " +
			"lent-overdue (money owed to the user open longer than 90 days), idle-cash (idle cash worth moving to the best published " +
			"rate, GET /api/bills/idle-cash; impact = the extra interest a year; skipped without a rate). A rule adds one item while its condition holds, unless " +
			"one is pending, was dismissed in the last 90 days or marked done in the last 30, and closes its pending item (done, outcome resolved) when the " +
			"condition is gone. A rule whose figures cannot be loaded is skipped. The tab is created on the first item.",
		QueryParams: recommendationFilterParams(),
	})
	recommendations.Post("/:id/status", guard.Wrap(h.SetStatus), RouteMeta{
		Summary:    "Marks a recommendation done, dismissed, snoozed (until a date, or for a number of days) or open again; writes its Status, Snoozed until and Status changed columns. Returns 409 when the row changed since it was read.",
		PathParams: []RouteParameter{PathParam("id", "string", "Recommendation id, rec-<code>.")},
		RequestBody: jsonBody(map[string]any{
			"status":       map[string]any{"type": "string", "enum": []string{"open", "done", "dismissed", "snoozed"}},
			"snoozedUntil": map[string]any{"type": "string", "description": "yyyy-mm-dd after today, for status snoozed."},
			"snoozeDays":   map[string]any{"type": "integer", "description": "1-365 days from today, for status snoozed (instead of snoozedUntil)."},
			"etag":         map[string]any{"type": "string", "description": "ETag from the list; may be sent as If-Match instead."},
		}, "status"),
	})
}

func recommendationFilter(c *fiber.Ctx) domain.RecommendationFilter {
	return domain.RecommendationFilter{
		Status: domain.RecommendationStatus(utils.CopyString(c.Query("status"))),
		Page:   utils.CopyString(c.Query("page")),
		Source: utils.CopyString(c.Query("source")),
	}
}

func (h *RecommendationHandler) List(c *fiber.Ctx) error {
	return respondWith(c, h.log, recommendationsScope)(h.recommendations.List(c.UserContext(), recommendationFilter(c), forceFresh(c)))
}

func (h *RecommendationHandler) Refresh(c *fiber.Ctx) error {
	return respondWith(c, h.log, recommendationsScope)(h.recommendations.Refresh(c.UserContext(), recommendationFilter(c)))
}

func (h *RecommendationHandler) SetStatus(c *fiber.Ctx) error {
	var body domain.RecommendationStatusInput
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	body.ETag = etagFrom(c, body.ETag)
	return respondWith(c, h.log, recommendationsScope)(h.recommendations.SetStatus(c.UserContext(), pathParam(c, "id"), body))
}
