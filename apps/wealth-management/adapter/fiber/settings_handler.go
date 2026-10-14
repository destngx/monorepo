package fiber

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// SettingsHandler edits the stored engine settings, which replace the env
// file. Secrets are write-only: responses say whether they are set.
type SettingsHandler struct {
	settings port.SettingsService
	log      *logger.Logger
}

func NewSettingsHandler(settings port.SettingsService, log *logger.Logger) *SettingsHandler {
	return &SettingsHandler{settings: settings, log: log}
}

func (h *SettingsHandler) Get(c *fiber.Ctx) error {
	view, err := h.settings.View(c.UserContext())
	if err != nil {
		return respondError(c, h.log, "settings_handler", err)
	}
	return c.JSON(view)
}

func (h *SettingsHandler) Update(c *fiber.Ctx) error {
	var body domain.SettingsUpdate
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	view, err := h.settings.Update(c.UserContext(), body)
	if err != nil {
		return respondError(c, h.log, "settings_handler", err)
	}
	return c.JSON(view)
}

// Restart answers 202; the engine restarts once the response is sent.
func (h *SettingsHandler) Restart(c *fiber.Ctx) error {
	if err := h.settings.Restart(c.UserContext()); err != nil {
		return respondError(c, h.log, "settings_handler", err)
	}
	return c.Status(http.StatusAccepted).JSON(fiber.Map{"status": "restarting"})
}

func RegisterSettingsRoutes(api RouteRegistrar, h *SettingsHandler, guard WriteGuard) {
	settings := api.Group("/settings", "settings")
	settings.Get("/", h.Get, RouteMeta{Summary: "Lists every engine setting by group with its stored value. Secrets are never returned, only whether they are set."})
	settings.Patch("/", guard.Wrap(h.Update), RouteMeta{
		Summary: "Saves settings. Values map keys to strings; null or an empty string clears a setting. Applies after POST /api/settings/restart.",
		RequestBody: jsonBody(map[string]any{
			"values": map[string]any{"type": "object", "additionalProperties": map[string]any{"type": []string{"string", "null"}}},
		}, "values"),
	})
	settings.Post("/restart", guard.Wrap(h.Restart), RouteMeta{Summary: "Restarts the engine so saved settings take effect. Answers 202, then the engine is briefly unavailable."})
}
