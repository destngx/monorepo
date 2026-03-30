package fiber

import (
	"apps/wealth-management-engine/port"
	"context"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

type CacheHandler struct {
	cacheService port.CacheService
}

func NewCacheHandler(service port.CacheService) *CacheHandler {
	return &CacheHandler{cacheService: service}
}

func (h *CacheHandler) Ping(c *fiber.Ctx) error {
	if err := h.cacheService.Ping(context.Background()); err != nil {
		return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "OK"})
}

func (h *CacheHandler) Set(c *fiber.Ctx) error {
	var body struct {
		Key        string `json:"key"`
		Value      string `json:"value"`
		TTLSeconds int    `json:"ttlSeconds"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if body.Key == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "key is required"})
	}

	if err := h.cacheService.Set(context.Background(), body.Key, body.Value, body.TTLSeconds); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "OK"})
}

func (h *CacheHandler) Get(c *fiber.Ctx) error {
	key := c.Params("key")
	if key == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "key is required"})
	}

	entry, err := h.cacheService.Get(context.Background(), key)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(entry)
}

func (h *CacheHandler) Invalidate(c *fiber.Ctx) error {
	pattern := c.Query("pattern", "*")
	deleted, err := h.cacheService.Invalidate(context.Background(), pattern)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"pattern": pattern,
		"deleted": deleted,
	})
}
