package fiber

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/port"
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type MarketProviderHandler struct {
	service port.MarketProviderService
	log     *logger.Logger
}

func NewMarketProviderHandler(service port.MarketProviderService, log *logger.Logger) *MarketProviderHandler {
	return &MarketProviderHandler{service: service, log: log}
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
		return upstreamError(c, h.log, "market_provider_handler: health check", err)
	}

	return c.JSON(health)
}
