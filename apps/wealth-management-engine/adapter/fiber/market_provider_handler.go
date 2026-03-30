package fiber

import (
	"apps/wealth-management-engine/port"
	"context"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

type MarketProviderHandler struct {
	service port.MarketProviderService
}

func NewMarketProviderHandler(service port.MarketProviderService) *MarketProviderHandler {
	return &MarketProviderHandler{service: service}
}

func (h *MarketProviderHandler) Health(c *fiber.Ctx) error {
	provider := c.Params("provider")
	health, err := h.service.Health(context.Background(), provider)
	if err != nil {
		return c.Status(http.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(health)
}
