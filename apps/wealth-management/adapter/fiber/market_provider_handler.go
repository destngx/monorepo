package fiber

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/port"
	"context"
	"log/slog"
	"net/http"
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
		requestID := c.Get("X-Request-ID")
		h.log.LogError(c.UserContext(), "market_provider_handler: health check failed", err,
			slog.String("request_id", requestID),
			slog.String("endpoint", c.Path()),
			slog.String("provider", provider),
		)
		return writeError(c, http.StatusBadGateway, CodeUpstream, err.Error())
	}

	return c.JSON(health)
}
