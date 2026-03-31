package fiber

import (
	"apps/wealth-management-engine/port"
	"context"
	"net/http"
	"strings"

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
	if provider == "" {
		path := strings.ToLower(c.Path())
		switch {
		case strings.Contains(path, "/vnstock/"):
			provider = "vnstock"
		case strings.Contains(path, "/fmarket/"):
			provider = "fmarket"
		}
	}
	health, err := h.service.Health(context.Background(), provider)
	if err != nil {
		return c.Status(http.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(health)
}
