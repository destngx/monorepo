package fiber

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"errors"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/gofiber/fiber/v2"
)

// GoogleAuthHandler connects a Google account from the UI. Google redirects
// the browser to the callback; when that browser cannot reach the engine,
// the UI posts the address it landed on to the exchange route instead.
type GoogleAuthHandler struct {
	auth port.GoogleAuthService
	log  *logger.Logger
}

func NewGoogleAuthHandler(auth port.GoogleAuthService, log *logger.Logger) *GoogleAuthHandler {
	return &GoogleAuthHandler{auth: auth, log: log}
}

func (h *GoogleAuthHandler) Start(c *fiber.Ctx) error {
	var body struct {
		Purpose string `json:"purpose"`
	}
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&body); err != nil {
			return badRequest(c, "invalid request body")
		}
	}
	purpose, err := domain.ParseGoogleAuthPurpose(body.Purpose)
	if err != nil {
		return respondError(c, h.log, "google_auth_handler", err)
	}
	start, err := h.auth.Start(c.UserContext(), purpose, c.Get(fiber.HeaderOrigin))
	if err != nil {
		return respondError(c, h.log, "google_auth_handler", err)
	}
	return c.JSON(start)
}

func (h *GoogleAuthHandler) Exchange(c *fiber.Ctx) error {
	var body struct {
		RedirectedURL string `json:"redirectedUrl"`
	}
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	state, code, err := domain.ParseGoogleRedirect(body.RedirectedURL)
	if err != nil {
		return respondError(c, h.log, "google_auth_handler", err)
	}
	result, err := h.auth.Complete(c.UserContext(), state, code)
	if err != nil {
		return upstreamError(c, h.log, "google_auth_handler", err)
	}
	return c.JSON(result)
}

// Callback is where Google sends the browser. It is not behind the write
// guard (the browser has no API token); the one-shot state and the PKCE
// verifier held by the engine are what make it safe. It always answers by
// sending the browser back to the Settings page, relative so it stays on the
// origin Google returned to (the dev server or the engine), with
// google=connected&purpose=... or google=error&reason=... (a fixed code, so
// a crafted link cannot show arbitrary text).
func (h *GoogleAuthHandler) Callback(c *fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, "no-store")
	c.Set("Referrer-Policy", "no-referrer")

	state, code, err := domain.ParseGoogleRedirect(string(c.Request().URI().FullURI()))
	var result domain.GoogleAuthResult
	if err == nil {
		result, err = h.auth.Complete(c.UserContext(), state, code)
	}
	query := url.Values{}
	if err != nil {
		reason := callbackReason(err)
		if reason == "failed" {
			h.log.LogError(c.UserContext(), "google_auth_handler: callback failed", err, slog.String("endpoint", c.Path()))
		}
		query.Set("google", "error")
		query.Set("reason", reason)
	} else {
		query.Set("google", "connected")
		query.Set("purpose", string(result.Purpose))
	}
	return c.Redirect(UIMountPath+"/settings?"+query.Encode(), http.StatusSeeOther)
}

// callbackReason is the fixed code the Settings page explains.
func callbackReason(err error) string {
	switch {
	case errors.Is(err, domain.ErrGoogleConsentDenied):
		return "denied"
	case errors.Is(err, domain.ErrGoogleAuthState):
		return "expired"
	case errors.Is(err, domain.ErrGoogleCodeRejected):
		return "rejected"
	case errors.Is(err, domain.ErrGoogleNoRefreshToken):
		return "no_refresh_token"
	case errors.Is(err, domain.ErrValidation):
		return "invalid"
	default:
		return "failed"
	}
}

func RegisterGoogleAuthRoutes(api RouteRegistrar, h *GoogleAuthHandler, guard WriteGuard) {
	auth := api.Group("/auth/google", "settings")
	auth.Post("/start", guard.Wrap(h.Start), RouteMeta{
		Summary: "Starts connecting a Google account. Purpose is sheets (saves GOOGLE_REFRESH_TOKEN, spreadsheets scope only). Open authUrl in a browser; the sign-in expires after 10 minutes.",
		RequestBody: jsonBody(map[string]any{
			"purpose": map[string]any{"type": "string", "enum": []string{string(domain.GoogleAuthSheets)}},
		}),
	})
	auth.Get("/callback", h.Callback, RouteMeta{
		Summary: "Google redirects the browser here after consent. Saves the refresh token, then redirects (303) to the Settings page with google=connected&purpose=... or google=error&reason=denied|expired|rejected|no_refresh_token|invalid|failed.",
		QueryParams: []RouteParameter{
			{Name: "state", In: "query", Type: "string", Description: "Sign-in state from start."},
			{Name: "code", In: "query", Type: "string", Description: "Authorization code from Google."},
			{Name: "error", In: "query", Type: "string", Description: "Set by Google when consent failed."},
		},
	})
	auth.Post("/exchange", guard.Wrap(h.Exchange), RouteMeta{
		Summary: "Completes a sign-in from the address the browser landed on, when that browser could not reach the callback.",
		RequestBody: jsonBody(map[string]any{
			"redirectedUrl": map[string]any{"type": "string"},
		}, "redirectedUrl"),
	})
}
