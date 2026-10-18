package fiber

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/port"

	"github.com/gofiber/fiber/v2"
)

// APITokenHandler sets the write token from the Settings page (WM-SEC-1).
// The token is write-only: responses only say whether one is set.
type APITokenHandler struct {
	tokens port.APITokenService
	log    *logger.Logger
}

func NewAPITokenHandler(tokens port.APITokenService, log *logger.Logger) *APITokenHandler {
	return &APITokenHandler{tokens: tokens, log: log}
}

func (h *APITokenHandler) Status(c *fiber.Ctx) error {
	return c.JSON(h.tokens.Status())
}

func (h *APITokenHandler) Set(c *fiber.Ctx) error {
	var body domain.APITokenUpdate
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	status, err := h.tokens.Set(c.UserContext(), body)
	if err != nil {
		return respondError(c, h.log, "api_token_handler", err)
	}
	return c.JSON(status)
}

func RegisterAPITokenRoutes(api RouteRegistrar, h *APITokenHandler, guard WriteGuard) {
	settings := api.Group("/settings", "settings")
	settings.Get("/api-token", h.Status, RouteMeta{Summary: "Says whether an API token guards writes and whether the engine listens on loopback. The token is never returned."})
	settings.Post("/api-token", guard.Wrap(h.Set), RouteMeta{
		Summary:     "Replaces the API token. Needs the current token when one is set; the new token applies at once, no restart. It cannot be cleared.",
		RequestBody: jsonBody(map[string]any{"token": map[string]any{"type": "string", "minLength": 32, "maxLength": 256}}, "token"),
	})
}
