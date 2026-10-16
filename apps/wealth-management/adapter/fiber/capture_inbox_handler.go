package fiber

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

const captureInboxScope = "capture_inbox_handler"

// CaptureInboxHandler serves quick capture (§6.3): the phone posts a
// screenshot, the UI reviews the drafts read from it.
type CaptureInboxHandler struct {
	inbox port.CaptureInboxService
	log   *logger.Logger
}

func NewCaptureInboxHandler(inbox port.CaptureInboxService, log *logger.Logger) *CaptureInboxHandler {
	return &CaptureInboxHandler{inbox: inbox, log: log}
}

// QuickCaptureReceipt is the answer to POST /api/capture/inbox.
type QuickCaptureReceipt struct {
	domain.QuickCapture
	// Duplicate is set when the same image and note were sent before.
	Duplicate bool `json:"duplicate"`
}

// Submit takes a multipart form with an "image" file and a "note" field,
// either optional but not both, and answers 202 once they are stored.
func (h *CaptureInboxHandler) Submit(c *fiber.Ctx) error {
	form, err := c.MultipartForm()
	if err != nil {
		return badRequest(c, "send a multipart form with an image file and/or a note field")
	}
	var input domain.QuickCaptureInput
	if notes := form.Value["note"]; len(notes) > 0 {
		input.Note = notes[0]
	}
	if files := form.File["image"]; len(files) > 0 {
		file, err := files[0].Open()
		if err != nil {
			return respondError(c, h.log, captureInboxScope, fmt.Errorf("open uploaded image: %w", err))
		}
		input.Image, err = io.ReadAll(file)
		if err := errors.Join(err, file.Close()); err != nil {
			return respondError(c, h.log, captureInboxScope, fmt.Errorf("read uploaded image: %w", err))
		}
	}
	capture, duplicate, err := h.inbox.Submit(c.UserContext(), input)
	if err != nil {
		return respondError(c, h.log, captureInboxScope, err)
	}
	return c.Status(http.StatusAccepted).JSON(QuickCaptureReceipt{QuickCapture: capture, Duplicate: duplicate})
}

func (h *CaptureInboxHandler) List(c *fiber.Ctx) error {
	return respondWith(c, h.log, captureInboxScope)(h.inbox.Overview(c.UserContext()))
}

func (h *CaptureInboxHandler) Retry(c *fiber.Ctx) error {
	if err := h.inbox.Retry(c.UserContext(), c.Params("id")); err != nil {
		return respondError(c, h.log, captureInboxScope, err)
	}
	return c.Status(http.StatusAccepted).JSON(fiber.Map{"status": string(domain.QuickCaptureQueued)})
}

func (h *CaptureInboxHandler) Discard(c *fiber.Ctx) error {
	if err := h.inbox.Discard(c.UserContext(), c.Params("id")); err != nil {
		return respondError(c, h.log, captureInboxScope, err)
	}
	return c.SendStatus(http.StatusNoContent)
}

func (h *CaptureInboxHandler) Drafts(c *fiber.Ctx) error {
	return respondWith(c, h.log, captureInboxScope)(h.inbox.Drafts(c.UserContext()))
}

func (h *CaptureInboxHandler) Approve(c *fiber.Ctx) error {
	var body domain.CaptureDraftApproval
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	return respondWith(c, h.log, captureInboxScope)(h.inbox.Approve(c.UserContext(), body))
}

func (h *CaptureInboxHandler) Reject(c *fiber.Ctx) error {
	var body struct {
		IDs []string `json:"ids"`
	}
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	if err := h.inbox.Reject(c.UserContext(), body.IDs); err != nil {
		return respondError(c, h.log, captureInboxScope, err)
	}
	return c.SendStatus(http.StatusNoContent)
}

// Material serves a stored screenshot or note; never cached by the browser.
func (h *CaptureInboxHandler) Material(c *fiber.Ctx) error {
	material, err := h.inbox.Material(c.UserContext(), c.Params("id"))
	if err != nil {
		return respondError(c, h.log, captureInboxScope, err)
	}
	c.Set(fiber.HeaderCacheControl, "private, no-store")
	c.Set(fiber.HeaderXContentTypeOptions, "nosniff")
	if material.Kind == domain.CaptureText {
		c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
		return c.SendString(material.Text)
	}
	c.Set(fiber.HeaderContentType, material.MediaType)
	return c.Send(material.Data)
}

var (
	schemaQuickCapture = map[string]any{"type": "object", "properties": map[string]any{
		"id":            schemaString,
		"createdAt":     map[string]any{"type": "string", "format": "date-time"},
		"status":        map[string]any{"type": "string", "enum": []string{"queued", "processing", "done", "failed"}},
		"note":          schemaString,
		"hasImage":      map[string]any{"type": "boolean"},
		"attempts":      map[string]any{"type": "integer"},
		"nextAttemptAt": map[string]any{"type": "string", "format": "date-time"},
		"message":       schemaString,
		"newDrafts":     map[string]any{"type": "integer"},
	}}
	captureIDParam  = []RouteParameter{{Name: "id", In: "path", Type: "string", Required: true, Description: "Capture id."}}
	materialIDParam = []RouteParameter{{Name: "id", In: "path", Type: "string", Required: true, Description: "Material id: <captureId>-image or <captureId>-note."}}
)

func RegisterCaptureInboxRoutes(api RouteRegistrar, h *CaptureInboxHandler, guard WriteGuard) {
	capture := api.Group("/capture", "capture")
	capture.Post("/inbox", guard.Wrap(h.Submit), RouteMeta{
		Summary: "Quick capture (WM-CAP-15): stores a screenshot and/or a note and answers 202 before any AI work; the engine reads it into drafts in the background. Re-sending the same image and note returns the first capture with duplicate set. Built for an iOS Shortcut.",
		RequestBody: map[string]any{
			"required": true,
			"content": map[string]any{"multipart/form-data": map[string]any{"schema": map[string]any{"type": "object", "properties": map[string]any{
				"image": map[string]any{"type": "string", "format": "binary", "description": "PNG, JPEG, WebP or GIF screenshot."},
				"note":  map[string]any{"type": "string", "description": "Optional context, added to the drafts' memo and read with the image."},
			}}}},
		},
		Responses: map[string]any{"202": map[string]any{"description": "Stored and queued.", "content": map[string]any{"application/json": map[string]any{"schema": schemaQuickCapture}}}},
	})
	capture.Get("/inbox", h.List, RouteMeta{Summary: "The latest 50 quick captures, newest first, with status (queued, processing, done, failed), attempts, next retry and outcome message, and the number of drafts waiting for review (pendingDrafts)."})
	capture.Post("/inbox/:id/retry", guard.Wrap(h.Retry), RouteMeta{Summary: "Queues a failed capture again with fresh retries. 409 unless the capture failed.", PathParams: captureIDParam})
	capture.Delete("/inbox/:id", guard.Wrap(h.Discard), RouteMeta{Summary: "Deletes a capture and its material. 409 when drafts came from it (approve or reject them instead) or while it is being read.", PathParams: captureIDParam})
	capture.Get("/drafts", h.Drafts, RouteMeta{Summary: "Pending drafts (WM-CAP-19) with ledger dedup run against the current sheet, the materials they came from (fetch content from /api/capture/materials/{id}) and the latest balance each capture showed per account."})
	capture.Post("/drafts/approve", guard.Wrap(h.Approve), RouteMeta{
		Summary: "Writes the edited drafts through the batch create path (WM-TX-10) and marks the written ones approved. Replaying a batchId does not write twice and answers the first result. 409 when a draft was rejected or approved by another batch.",
		RequestBody: jsonBody(map[string]any{
			"batchId": schemaString,
			"drafts": map[string]any{"type": "array", "items": map[string]any{"type": "object", "properties": map[string]any{
				"id":          schemaString,
				"transaction": map[string]any{"type": "object", "description": "The row to write, as for POST /api/transactions/batch."},
			}, "required": []string{"id", "transaction"}}},
		}, "batchId", "drafts"),
	})
	capture.Post("/drafts/reject", guard.Wrap(h.Reject), RouteMeta{
		Summary:     "Rejects pending drafts; a rejected row is not offered again by later captures. 409 when one was already approved.",
		RequestBody: jsonBody(map[string]any{"ids": map[string]any{"type": "array", "items": schemaString}}, "ids"),
	})
	capture.Get("/materials/:id", h.Material, RouteMeta{Summary: "A stored screenshot (its image type) or note (text/plain) while a pending draft or an unfinished capture needs it; 404 once deleted.", PathParams: materialIDParam})
}

// RegisterUnavailableCaptureInboxRoutes registers the quick capture routes
// answering with cause when the sheet or the store cannot be used.
func RegisterUnavailableCaptureInboxRoutes(api RouteRegistrar, cause error, log *logger.Logger) {
	unavailable := api.Answering(func(c *fiber.Ctx) error {
		return respondError(c, log, captureInboxScope, cause)
	})
	RegisterCaptureInboxRoutes(unavailable, NewCaptureInboxHandler(nil, log), WriteGuard{})
}
