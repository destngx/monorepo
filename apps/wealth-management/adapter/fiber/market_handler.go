package fiber

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type MarketHandler struct {
	service port.MarketProviderService
	log     *logger.Logger
}

func NewMarketHandler(service port.MarketProviderService, log *logger.Logger) *MarketHandler {
	return &MarketHandler{service: service, log: log}
}

func (h *MarketHandler) GetTicker(c *fiber.Ctx) error {
	symbol := strings.TrimSpace(c.Query("symbol"))
	tickerType := domain.TickerType(strings.ToLower(strings.TrimSpace(c.Query("type"))))
	if symbol == "" || tickerType == "" {
		return badRequest(c, "symbol and type are required")
	}
	ctx := withBypassCache(c)

	ticker, err := h.service.GetTicker(ctx, symbol, tickerType)
	if errors.Is(err, domain.ErrValidation) {
		return badRequest(c, err.Error())
	}
	if err != nil {
		requestID := c.Get("X-Request-ID")
		h.log.LogError(c.UserContext(), "market_handler: get ticker failed", err,
			slog.String("request_id", requestID),
			slog.String("endpoint", c.Path()),
			slog.String("symbol", symbol),
		)
		return writeError(c, http.StatusBadGateway, CodeUpstream, err.Error())
	}

	return c.JSON(ticker)
}

func (h *MarketHandler) GetExchangeRate(c *fiber.Ctx) error {
	from := strings.TrimSpace(c.Query("from"))
	to := strings.TrimSpace(c.Query("to"))
	if from == "" || to == "" {
		return badRequest(c, "from and to are required")
	}
	ctx := withBypassCache(c)

	rate, err := h.service.GetExchangeRate(ctx, from, to)
	if err != nil {
		requestID := c.Get("X-Request-ID")
		h.log.LogError(c.UserContext(), "market_handler: get exchange rate failed", err,
			slog.String("request_id", requestID),
			slog.String("endpoint", c.Path()),
			slog.String("from", from),
			slog.String("to", to),
		)
		return writeError(c, http.StatusBadGateway, CodeUpstream, err.Error())
	}

	return c.JSON(rate)
}

func (h *MarketHandler) GetPriceSeries(c *fiber.Ctx) error {
	symbol := strings.TrimSpace(c.Query("symbol"))
	seriesType := domain.SeriesType(strings.ToLower(strings.TrimSpace(c.Query("type"))))
	if symbol == "" || seriesType == "" {
		return badRequest(c, "symbol and type are required")
	}
	ctx := withBypassCache(c)

	series, err := h.service.GetPriceSeries(ctx, symbol, seriesType)
	if err != nil {
		requestID := c.Get("X-Request-ID")
		h.log.LogError(c.UserContext(), "market_handler: get price series failed", err,
			slog.String("request_id", requestID),
			slog.String("endpoint", c.Path()),
			slog.String("symbol", symbol),
		)
		return writeError(c, http.StatusBadGateway, CodeUpstream, err.Error())
	}

	return c.JSON(series)
}

func (h *MarketHandler) GetBankInterestRate(c *fiber.Ctx) error {
	ctx := withBypassCache(c)
	rates, err := h.service.GetBankInterestRate(ctx)
	if err != nil {
		requestID := c.Get("X-Request-ID")
		h.log.LogError(c.UserContext(), "market_handler: get bank interest rate failed", err,
			slog.String("request_id", requestID),
			slog.String("endpoint", c.Path()),
		)
		return writeError(c, http.StatusBadGateway, CodeUpstream, err.Error())
	}

	return c.JSON(rates)
}

// GetCandles serves OHLCV bars with their data source (WM-INV-14/15).
func (h *MarketHandler) GetCandles(c *fiber.Ctx) error {
	symbol := strings.TrimSpace(c.Query("symbol"))
	if symbol == "" {
		return badRequest(c, "symbol is required")
	}
	timeframe, err := domain.ParseTimeframe(c.Query("timeframe", string(domain.Timeframe1D)))
	if err != nil {
		return badRequest(c, err.Error())
	}
	days := c.QueryInt("days", 0)
	if days < 0 {
		return badRequest(c, "days must be positive")
	}

	series, err := h.service.GetCandles(withBypassCache(c), symbol, timeframe, days)
	if err != nil {
		return h.providerError(c, "get candles", err)
	}
	return c.JSON(series)
}

func (h *MarketHandler) providerError(c *fiber.Ctx, scope string, err error) error {
	return upstreamError(c, h.log, "market_handler: "+scope, err)
}

func withBypassCache(c *fiber.Ctx) context.Context {
	skipCache := strings.EqualFold(c.Query("skipCache"), "true")
	return domain.WithBypassCache(context.Background(), skipCache)
}
