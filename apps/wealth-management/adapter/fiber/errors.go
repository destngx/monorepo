package fiber

import (
	configAdapter "apps/wealth-management/adapter/config"
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/oauth2"
	"google.golang.org/api/googleapi"
)

// Error codes of the {"error", "code"} envelope every engine error uses
// (WM-UI-4). Clients branch on the code, never on the message.
const (
	CodeValidation          = "VALIDATION"
	CodeUnauthorized        = "UNAUTHORIZED"
	CodeForbidden           = "FORBIDDEN"
	CodeNotFound            = "NOT_FOUND"
	CodeConflict            = "CONFLICT"
	CodeSheetsNotConfigured = "SHEETS_NOT_CONFIGURED"
	CodeUnavailable         = "UNAVAILABLE"
	CodeOAuthExpired        = "OAUTH_EXPIRED"
	CodeUpstream            = "UPSTREAM"
	CodeInternal            = "INTERNAL"
)

// ErrorCodes lists every code in the envelope, for the OpenAPI document.
var ErrorCodes = []string{
	CodeValidation, CodeUnauthorized, CodeForbidden, CodeNotFound, CodeConflict,
	CodeSheetsNotConfigured, CodeUnavailable, CodeOAuthExpired, CodeUpstream, CodeInternal,
}

// ErrorBody is the JSON body of every error response.
type ErrorBody struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// oauthExpiredMarkers are the phrases Google uses when a refresh token is no
// longer valid; they are matched when the typed error was flattened to text.
var oauthExpiredMarkers = []string{"invalid_grant", "token has been expired or revoked", "token expired"}

// classifyError maps an error to its HTTP status and envelope code.
func classifyError(err error) (int, string) {
	switch {
	case errors.Is(err, domain.ErrValidation):
		return http.StatusBadRequest, CodeValidation
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, CodeNotFound
	case errors.Is(err, domain.ErrConflict):
		return http.StatusConflict, CodeConflict
	case errors.Is(err, configAdapter.ErrMissingSheetsConfig):
		return http.StatusServiceUnavailable, CodeSheetsNotConfigured
	case isOAuthExpired(err):
		return http.StatusBadGateway, CodeOAuthExpired
	case errors.Is(err, configAdapter.ErrMissingAIConfig),
		errors.Is(err, domain.ErrAIUnavailable),
		errors.Is(err, domain.ErrPromptMissing):
		return http.StatusServiceUnavailable, CodeUnavailable
	case errors.Is(err, domain.ErrAIOutput), isUpstreamFailure(err):
		return http.StatusBadGateway, CodeUpstream
	default:
		return http.StatusInternalServerError, CodeInternal
	}
}

// statusForError maps domain sentinel errors to HTTP status codes.
func statusForError(err error) int {
	status, _ := classifyError(err)
	return status
}

func isOAuthExpired(err error) bool {
	var retrieve *oauth2.RetrieveError
	if errors.As(err, &retrieve) && retrieve.ErrorCode == "invalid_grant" {
		return true
	}
	message := strings.ToLower(err.Error())
	for _, marker := range oauthExpiredMarkers {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}

// isUpstreamFailure reports failures of an external provider: a Google API
// error, a failed token exchange or a network error.
func isUpstreamFailure(err error) bool {
	var apiErr *googleapi.Error
	var retrieve *oauth2.RetrieveError
	var urlErr *url.Error
	return errors.As(err, &apiErr) || errors.As(err, &retrieve) || errors.As(err, &urlErr)
}

// writeError writes the error envelope.
func writeError(c *fiber.Ctx, status int, code, message string) error {
	return c.Status(status).JSON(ErrorBody{Error: message, Code: code})
}

// respondError logs the failure with the request ID and writes the envelope.
// Client errors (4xx) are expected outcomes and are not logged as errors.
func respondError(c *fiber.Ctx, log *logger.Logger, scope string, err error) error {
	status, code := classifyError(err)
	if status >= http.StatusInternalServerError {
		log.LogError(c.UserContext(), scope+": operation failed", err,
			slog.String("request_id", c.Get("X-Request-ID")),
			slog.String("endpoint", c.Path()),
		)
	}
	return writeError(c, status, code, err.Error())
}

func badRequest(c *fiber.Ctx, message string) error {
	return writeError(c, http.StatusBadRequest, CodeValidation, message)
}

// upstreamError answers like respondError, except that an unclassified
// failure is a provider failure (502 UPSTREAM) rather than INTERNAL.
func upstreamError(c *fiber.Ctx, log *logger.Logger, scope string, err error) error {
	status, code := classifyError(err)
	if code == CodeInternal {
		status, code = http.StatusBadGateway, CodeUpstream
	}
	if status >= http.StatusInternalServerError {
		log.LogError(c.UserContext(), scope+" failed", err,
			slog.String("request_id", c.Get("X-Request-ID")),
			slog.String("endpoint", c.Path()),
		)
	}
	return writeError(c, status, code, err.Error())
}

// FiberErrorHandler answers errors that escape handlers (unknown routes,
// oversized bodies, framework errors) with the envelope.
func FiberErrorHandler(log *logger.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		var fiberErr *fiber.Error
		if errors.As(err, &fiberErr) {
			return writeError(c, fiberErr.Code, codeForStatus(fiberErr.Code), fiberErr.Message)
		}
		return respondError(c, log, "fiber", err)
	}
}

// codeForStatus picks the envelope code for a bare HTTP status.
func codeForStatus(status int) string {
	switch {
	case status == http.StatusUnauthorized:
		return CodeUnauthorized
	case status == http.StatusForbidden:
		return CodeForbidden
	case status == http.StatusNotFound:
		return CodeNotFound
	case status == http.StatusConflict:
		return CodeConflict
	case status == http.StatusBadGateway:
		return CodeUpstream
	case status == http.StatusServiceUnavailable:
		return CodeUnavailable
	case status >= http.StatusBadRequest && status < http.StatusInternalServerError:
		return CodeValidation
	default:
		return CodeInternal
	}
}
