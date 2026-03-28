package fiber

import (
	"apps/wealth-management-engine/port"
	"github.com/gofiber/fiber/v2"
)

type HealthHandler struct {
	healthService port.HealthService
}

func NewHealthHandler(s port.HealthService) *HealthHandler {
	return &HealthHandler{healthService: s}
}

func (h *HealthHandler) HealthCheck(c *fiber.Ctx) error {
	status := h.healthService.Check()
	return c.JSON(status)
}
