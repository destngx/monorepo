package fiber

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type FmarketHandler struct {
	service port.FmarketService
	market  port.MarketProviderService
	log     *logger.Logger
}

func NewFmarketHandler(service port.FmarketService, market port.MarketProviderService, log *logger.Logger) *FmarketHandler {
	return &FmarketHandler{service: service, market: market, log: log}
}

func (h *FmarketHandler) Action(c *fiber.Ctx) error {
	var body domain.FmarketRequest
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	if strings.TrimSpace(string(body.Action)) == "" {
		return badRequest(c, "action is required")
	}
	result, err := h.service.RunAction(context.Background(), body, strings.EqualFold(c.Query("force"), "true"))
	if err != nil {
		requestID := c.Get("X-Request-ID")
		h.log.LogError(c.UserContext(), "fmarket_handler: action failed", err,
			slog.String("request_id", requestID),
			slog.String("endpoint", c.Path()),
			slog.String("action", string(body.Action)),
		)
		return writeError(c, http.StatusBadGateway, CodeUpstream, err.Error())
	}
	return c.JSON(result)
}

func (h *FmarketHandler) Health(c *fiber.Ctx) error {
	if h.market == nil {
		return writeError(c, http.StatusServiceUnavailable, CodeUnavailable, "market service unavailable")
	}
	health, err := h.market.Health(context.Background(), "fmarket")
	if err != nil {
		requestID := c.Get("X-Request-ID")
		h.log.LogError(c.UserContext(), "fmarket_handler: health check failed", err,
			slog.String("request_id", requestID),
			slog.String("endpoint", c.Path()),
		)
		return writeError(c, http.StatusBadGateway, CodeUpstream, err.Error())
	}
	return c.JSON(health)
}

func (h *FmarketHandler) GetTicker(c *fiber.Ctx) error {
	if h.market == nil {
		return writeError(c, http.StatusServiceUnavailable, CodeUnavailable, "market service unavailable")
	}
	symbol := strings.TrimSpace(c.Query("symbol"))
	tickerType := domain.TickerType(strings.ToLower(strings.TrimSpace(c.Query("type"))))
	if symbol == "" || tickerType == "" {
		return badRequest(c, "symbol and type are required")
	}
	ctx := withBypassCache(c)
	ticker, err := h.market.GetTicker(ctx, symbol, tickerType)
	if err != nil {
		requestID := c.Get("X-Request-ID")
		h.log.LogError(c.UserContext(), "fmarket_handler: get ticker failed", err,
			slog.String("request_id", requestID),
			slog.String("endpoint", c.Path()),
			slog.String("symbol", symbol),
		)
		return writeError(c, http.StatusBadGateway, CodeUpstream, err.Error())
	}
	return c.JSON(ticker)
}

func (h *FmarketHandler) GetExchangeRate(c *fiber.Ctx) error {
	if h.market == nil {
		return writeError(c, http.StatusServiceUnavailable, CodeUnavailable, "market service unavailable")
	}
	from := strings.TrimSpace(c.Query("from"))
	to := strings.TrimSpace(c.Query("to"))
	if from == "" || to == "" {
		return badRequest(c, "from and to are required")
	}
	rate, err := h.market.GetExchangeRate(withBypassCache(c), from, to)
	if err != nil {
		requestID := c.Get("X-Request-ID")
		h.log.LogError(c.UserContext(), "fmarket_handler: get exchange rate failed", err,
			slog.String("request_id", requestID),
			slog.String("endpoint", c.Path()),
			slog.String("from", from),
			slog.String("to", to),
		)
		return writeError(c, http.StatusBadGateway, CodeUpstream, err.Error())
	}
	return c.JSON(rate)
}

func (h *FmarketHandler) GetPriceSeries(c *fiber.Ctx) error {
	if h.market == nil {
		return writeError(c, http.StatusServiceUnavailable, CodeUnavailable, "market service unavailable")
	}
	symbol := strings.TrimSpace(c.Query("symbol"))
	seriesType := domain.SeriesType(strings.ToLower(strings.TrimSpace(c.Query("type"))))
	if symbol == "" || seriesType == "" {
		return badRequest(c, "symbol and type are required")
	}
	series, err := h.market.GetPriceSeries(withBypassCache(c), symbol, seriesType)
	if err != nil {
		requestID := c.Get("X-Request-ID")
		h.log.LogError(c.UserContext(), "fmarket_handler: get price series failed", err,
			slog.String("request_id", requestID),
			slog.String("endpoint", c.Path()),
			slog.String("symbol", symbol),
		)
		return writeError(c, http.StatusBadGateway, CodeUpstream, err.Error())
	}
	return c.JSON(series)
}

func (h *FmarketHandler) GetBankRates(c *fiber.Ctx) error {
	if h.market == nil {
		return writeError(c, http.StatusServiceUnavailable, CodeUnavailable, "market service unavailable")
	}
	rates, err := h.market.GetBankInterestRate(withBypassCache(c))
	if err != nil {
		requestID := c.Get("X-Request-ID")
		h.log.LogError(c.UserContext(), "fmarket_handler: get bank rates failed", err,
			slog.String("request_id", requestID),
			slog.String("endpoint", c.Path()),
		)
		return writeError(c, http.StatusBadGateway, CodeUpstream, err.Error())
	}
	return c.JSON(rates)
}
