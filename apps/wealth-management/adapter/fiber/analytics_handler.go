package fiber

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/port"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

const (
	defaultHistoryDays   = 60
	defaultBudgetHistory = 6
)

type AnalyticsHandler struct {
	analytics port.AnalyticsService
	log       *logger.Logger
}

func NewAnalyticsHandler(analytics port.AnalyticsService, log *logger.Logger) *AnalyticsHandler {
	return &AnalyticsHandler{analytics: analytics, log: log}
}

func (h *AnalyticsHandler) respond(c *fiber.Ctx) func(value any, err error) error {
	return respondWith(c, h.log, "analytics_handler")
}

func (h *AnalyticsHandler) GetDashboard(c *fiber.Ctx) error {
	return h.respond(c)(h.analytics.DashboardSummary(forceFresh(c)))
}

func (h *AnalyticsHandler) GetBudgetReport(c *fiber.Ctx) error {
	return h.respond(c)(h.analytics.BudgetReport(strings.TrimSpace(c.Query("month")), forceFresh(c)))
}

func (h *AnalyticsHandler) GetBudgetHistory(c *fiber.Ctx) error {
	months := defaultBudgetHistory
	if raw := strings.TrimSpace(c.Query("months")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return badRequest(c, "months must be a number")
		}
		months = parsed
	}
	return h.respond(c)(h.analytics.BudgetHistory(strings.TrimSpace(c.Query("month")), months, forceFresh(c)))
}

func (h *AnalyticsHandler) GetCards(c *fiber.Ctx) error {
	return h.respond(c)(h.analytics.CardReports(forceFresh(c)))
}

func (h *AnalyticsHandler) GetFinancialHealth(c *fiber.Ctx) error {
	return h.respond(c)(h.analytics.FinancialHealth(forceFresh(c)))
}

func (h *AnalyticsHandler) GetAccountHistories(c *fiber.Ctx) error {
	days := defaultHistoryDays
	if raw := strings.TrimSpace(c.Query("days")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return badRequest(c, "days must be a number")
		}
		days = parsed
	}
	return h.respond(c)(h.analytics.AccountHistories(days, forceFresh(c)))
}
