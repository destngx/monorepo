package fiber

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/port"
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type ToolIntegrationHandler struct {
	service port.ToolIntegrationService
	log     *logger.Logger
}

func NewToolIntegrationHandler(service port.ToolIntegrationService, log *logger.Logger) *ToolIntegrationHandler {
	return &ToolIntegrationHandler{service: service, log: log}
}

func (h *ToolIntegrationHandler) Run(c *fiber.Ctx) error {
	var body struct {
		Prompt string `json:"prompt"`
	}
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	if strings.TrimSpace(body.Prompt) == "" {
		body.Prompt = "Check cash balance and market outlook for BTC"
	}

	conversation, err := h.service.RunConversation(context.Background(), body.Prompt)
	if err != nil {
		return respondError(c, h.log, "tool_integration_handler: conversation", err)
	}

	return c.JSON(conversation)
}
