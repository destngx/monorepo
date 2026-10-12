package fiber

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

type HealthHandler struct {
	healthService port.HealthService
}

func NewHealthHandler(s port.HealthService) *HealthHandler {
	return &HealthHandler{healthService: s}
}

// HealthCheck returns 200 when every dependency is up, else 503 with details.
func (h *HealthHandler) HealthCheck(c *fiber.Ctx) error {
	status := h.healthService.Check(c.UserContext())
	code := http.StatusOK
	if status.Status != domain.HealthOK {
		code = http.StatusServiceUnavailable
	}
	return c.Status(code).JSON(status)
}
