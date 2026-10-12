package fiber

import (
	configAdapter "apps/wealth-management/adapter/config"
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// statusForError maps domain sentinel errors to HTTP status codes.
func statusForError(err error) int {
	switch {
	case errors.Is(err, domain.ErrValidation):
		return http.StatusBadRequest
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrConflict):
		return http.StatusConflict
	case errors.Is(err, configAdapter.ErrMissingSheetsConfig),
		errors.Is(err, configAdapter.ErrMissingAIConfig),
		errors.Is(err, domain.ErrAIUnavailable),
		errors.Is(err, domain.ErrPromptMissing):
		return http.StatusServiceUnavailable
	case errors.Is(err, domain.ErrAIOutput):
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

// respondError logs the failure with the request ID and writes {"error": ...}.
// Client errors (4xx) are expected outcomes and are not logged as errors.
func respondError(c *fiber.Ctx, log *logger.Logger, scope string, err error) error {
	status := statusForError(err)
	if status >= http.StatusInternalServerError {
		log.LogError(c.UserContext(), scope+": operation failed", err,
			slog.String("request_id", c.Get("X-Request-ID")),
			slog.String("endpoint", c.Path()),
		)
	}
	return c.Status(status).JSON(fiber.Map{"error": err.Error()})
}

func badRequest(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": message})
}
