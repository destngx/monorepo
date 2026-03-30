package fiber

import (
	"apps/wealth-management-engine/port"
	"context"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type ToolIntegrationHandler struct {
	service port.ToolIntegrationService
}

func NewToolIntegrationHandler(service port.ToolIntegrationService) *ToolIntegrationHandler {
	return &ToolIntegrationHandler{service: service}
}

func (h *ToolIntegrationHandler) Run(c *fiber.Ctx) error {
	var body struct {
		Prompt string `json:"prompt"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if strings.TrimSpace(body.Prompt) == "" {
		body.Prompt = "Check cash balance and market outlook for BTC"
	}

	conversation, err := h.service.RunConversation(context.Background(), body.Prompt)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(conversation)
}
