package fiber

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/port"
	"log/slog"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// CacheHandler exposes cache health only; generic get/set/invalidate were
// removed (WM-SEC-1). Use POST /api/sync to invalidate sheet data.
type CacheHandler struct {
	cacheService port.CacheService
	log          *logger.Logger
}

func NewCacheHandler(service port.CacheService, log *logger.Logger) *CacheHandler {
	return &CacheHandler{cacheService: service, log: log}
}

func (h *CacheHandler) Ping(c *fiber.Ctx) error {
	if err := h.cacheService.Ping(c.UserContext()); err != nil {
		h.log.LogError(c.UserContext(), "cache_handler: ping failed", err,
			slog.String("request_id", c.Get("X-Request-ID")),
			slog.String("endpoint", c.Path()),
		)
		return writeError(c, http.StatusServiceUnavailable, CodeUnavailable, err.Error())
	}
	return c.JSON(fiber.Map{"status": "OK"})
}
