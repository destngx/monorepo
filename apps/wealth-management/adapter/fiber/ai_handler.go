package fiber

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/port"
	"bufio"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
)

const aiScope = "ai_handler"

type AIHandler struct {
	ai  port.AIService
	log *logger.Logger
}

func NewAIHandler(ai port.AIService, log *logger.Logger) *AIHandler {
	return &AIHandler{ai: ai, log: log}
}

// aiOptions are the fields every AI request may carry.
type aiOptions struct {
	Force bool   `json:"force"`
	Model string `json:"model"`
}

func (h *AIHandler) GetStatus(c *fiber.Ctx) error {
	return c.JSON(h.ai.Status(c.UserContext()))
}

func (h *AIHandler) SuggestCategory(c *fiber.Ctx) error {
	var body struct {
		Payee string `json:"payee"`
		Model string `json:"model"`
	}
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	return respondWith(c, h.log, aiScope)(h.ai.SuggestCategory(c.UserContext(), body.Payee, body.Model))
}

func (h *AIHandler) Review(c *fiber.Ctx) error {
	var body port.ReviewRequest
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	return respondWith(c, h.log, aiScope)(h.ai.Review(c.UserContext(), body))
}

func (h *AIHandler) HealthCommentary(c *fiber.Ctx) error {
	var body aiOptions
	if err := parseOptionalBody(c, &body); err != nil {
		return badRequest(c, "invalid request body")
	}
	return respondWith(c, h.log, aiScope)(h.ai.HealthCommentary(c.UserContext(), body.Force, body.Model))
}

func (h *AIHandler) ChartInsight(c *fiber.Ctx) error {
	var body port.ChartInsightRequest
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	insight, err := h.ai.ChartInsight(c.UserContext(), body)
	if err != nil {
		return respondError(c, h.log, aiScope, err)
	}
	return c.JSON(fiber.Map{"insight": insight})
}

func (h *AIHandler) Briefing(c *fiber.Ctx) error {
	var body aiOptions
	if err := parseOptionalBody(c, &body); err != nil {
		return badRequest(c, "invalid request body")
	}
	return respondWith(c, h.log, aiScope)(h.ai.Briefing(c.UserContext(), body.Force, body.Model))
}

func (h *AIHandler) BudgetAdvice(c *fiber.Ctx) error {
	var body aiOptions
	if err := parseOptionalBody(c, &body); err != nil {
		return badRequest(c, "invalid request body")
	}
	return respondWith(c, h.log, aiScope)(h.ai.BudgetAdvice(c.UserContext(), body.Force, body.Model))
}

// Chat streams port.ChatEvent values as NDJSON (WM-AI-11). Validation errors
// are answered with a status code; later failures become an "error" event.
func (h *AIHandler) Chat(c *fiber.Ctx) error {
	var body port.ChatRequest
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	if err := body.Validate(); err != nil {
		return respondError(c, h.log, aiScope, err)
	}
	// The stream writer runs after the handler returns, when c is recycled:
	// capture everything it needs now.
	ctx := c.UserContext()
	requestID := c.Get("X-Request-ID")
	c.Set(fiber.HeaderContentType, "application/x-ndjson")
	c.Set(fiber.HeaderCacheControl, "no-cache")
	c.Set("X-Accel-Buffering", "no")
	c.Status(http.StatusOK).Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		encoder := json.NewEncoder(w)
		emit := func(event port.ChatEvent) error {
			if err := encoder.Encode(event); err != nil {
				return err
			}
			return w.Flush()
		}
		err := h.ai.Chat(ctx, body, emit)
		if err == nil {
			return
		}
		h.log.LogError(ctx, aiScope+": chat failed", err, slog.String("request_id", requestID))
		message := "The assistant failed. Try again."
		if statusForError(err) != http.StatusInternalServerError {
			message = err.Error()
		}
		_ = emit(port.ChatEvent{Type: "error", Message: message})
	})
	return nil
}

// parseOptionalBody accepts an empty body as the zero value.
func parseOptionalBody(c *fiber.Ctx, target any) error {
	if len(strings.TrimSpace(string(c.Body()))) == 0 {
		return nil
	}
	return c.BodyParser(target)
}
