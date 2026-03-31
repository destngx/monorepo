package fiber

import (
	"apps/wealth-management-engine/domain"
	"apps/wealth-management-engine/port"
	"context"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type MarketHandler struct {
	service port.MarketProviderService
}

func NewMarketHandler(service port.MarketProviderService) *MarketHandler {
	return &MarketHandler{service: service}
}

func (h *MarketHandler) GetTicker(c *fiber.Ctx) error {
	symbol := strings.TrimSpace(c.Query("symbol"))
	tickerType := domain.TickerType(strings.ToLower(strings.TrimSpace(c.Query("type"))))
	if symbol == "" || tickerType == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "symbol and type are required"})
	}
	ctx := withBypassCache(c)

	ticker, err := h.service.GetTicker(ctx, symbol, tickerType)
	if err != nil {
		return c.Status(http.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(ticker)
}

func (h *MarketHandler) GetExchangeRate(c *fiber.Ctx) error {
	from := strings.TrimSpace(c.Query("from"))
	to := strings.TrimSpace(c.Query("to"))
	if from == "" || to == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "from and to are required"})
	}
	ctx := withBypassCache(c)

	rate, err := h.service.GetExchangeRate(ctx, from, to)
	if err != nil {
		return c.Status(http.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(rate)
}

func (h *MarketHandler) GetPriceSeries(c *fiber.Ctx) error {
	symbol := strings.TrimSpace(c.Query("symbol"))
	seriesType := domain.SeriesType(strings.ToLower(strings.TrimSpace(c.Query("type"))))
	if symbol == "" || seriesType == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "symbol and type are required"})
	}
	ctx := withBypassCache(c)

	series, err := h.service.GetPriceSeries(ctx, symbol, seriesType)
	if err != nil {
		return c.Status(http.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(series)
}

func (h *MarketHandler) GetBankInterestRate(c *fiber.Ctx) error {
	ctx := withBypassCache(c)
	rates, err := h.service.GetBankInterestRate(ctx)
	if err != nil {
		return c.Status(http.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(rates)
}

func withBypassCache(c *fiber.Ctx) context.Context {
	skipCache := strings.EqualFold(c.Query("skipCache"), "true")
	return domain.WithBypassCache(context.Background(), skipCache)
}
