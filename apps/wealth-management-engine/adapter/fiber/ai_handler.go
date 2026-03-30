package fiber

import (
	"apps/wealth-management-engine/port"
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type AIHandler struct {
	aiService port.AIService
}

func NewAIHandler(service port.AIService) *AIHandler {
	return &AIHandler{aiService: service}
}

func (h *AIHandler) Stream(c *fiber.Ctx) error {
	var body struct {
		Prompt string `json:"prompt"`
		Model  string `json:"model"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if strings.TrimSpace(body.Prompt) == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "prompt is required"})
	}

	stream, err := h.aiService.Stream(context.Background(), body.Prompt, body.Model)
	if err != nil {
		return c.Status(http.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
	}
	defer stream.Close()

	c.Set("Content-Type", "application/x-ndjson")

	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" {
			continue
		}
		if data == "[DONE]" {
			donePayload, _ := json.Marshal(fiber.Map{"done": true})
			if _, err := c.Write(append(donePayload, '\n')); err != nil {
				return err
			}
			continue
		}

		var event map[string]any
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		payload, _ := json.Marshal(event)
		if _, err := c.Write(append(payload, '\n')); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil && err != io.EOF {
		return c.Status(http.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
	}

	return nil
}

func (h *AIHandler) JSON(c *fiber.Ctx) error {
	var body struct {
		SystemPrompt     string `json:"systemPrompt"`
		UserPrompt       string `json:"userPrompt"`
		AssistantHistory string `json:"assistantHistory"`
		Model            string `json:"model"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if strings.TrimSpace(body.SystemPrompt) == "" {
		body.SystemPrompt = "You are a strict JSON-only financial assistant."
	}
	if strings.TrimSpace(body.UserPrompt) == "" {
		body.UserPrompt = "Provide a concise investment briefing."
	}
	if strings.TrimSpace(body.AssistantHistory) == "" {
		body.AssistantHistory = "Previous context: user prefers conservative risk."
	}

	parsed, err := h.aiService.GenerateStructuredJSON(
		context.Background(),
		body.SystemPrompt,
		body.UserPrompt,
		body.AssistantHistory,
		body.Model,
	)
	if err != nil {
		return c.Status(http.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(parsed)
}
