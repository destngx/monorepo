// Package fiber is the HTTP adapter: routes, auth middleware, the error
// envelope and the UI mount.
package fiber

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gofiber/fiber/v2"

	"apps/health-watcher/domain"
)

// Error codes of the {"error", "code"} envelope. Clients branch on the
// code, never on the message.
const (
	CodeValidation   = "VALIDATION"
	CodeUnauthorized = "UNAUTHORIZED"
	CodeForbidden    = "FORBIDDEN"
	CodeNotFound     = "NOT_FOUND"
	CodeConflict     = "CONFLICT"
	CodeRateLimited  = "RATE_LIMITED"
	CodeTooLarge     = "TOO_LARGE"
	CodeInternal     = "INTERNAL"
)

// ErrorBody is the JSON body of every error response.
type ErrorBody struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func classify(err error) (int, string) {
	switch {
	case errors.Is(err, domain.ErrValidation):
		return http.StatusBadRequest, CodeValidation
	case errors.Is(err, domain.ErrUnauthorized):
		return http.StatusUnauthorized, CodeUnauthorized
	case errors.Is(err, domain.ErrForbidden):
		return http.StatusForbidden, CodeForbidden
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, CodeNotFound
	case errors.Is(err, domain.ErrConflict):
		return http.StatusConflict, CodeConflict
	case errors.Is(err, domain.ErrRateLimited):
		return http.StatusTooManyRequests, CodeRateLimited
	default:
		return http.StatusInternalServerError, CodeInternal
	}
}

func writeError(c *fiber.Ctx, status int, code, message string) error {
	return c.Status(status).JSON(ErrorBody{Error: message, Code: code})
}

// respond writes the envelope for err. Server errors are logged with the
// request id and answered with a generic message, so internals never leak
// to the client (HW-OPS-5).
func respond(c *fiber.Ctx, log *slog.Logger, err error) error {
	status, code := classify(err)
	if status >= http.StatusInternalServerError {
		log.Error("request failed", "request_id", c.Locals(localRequestID), "method", c.Method(), "route", c.Route().Path, "error", err)
		return writeError(c, status, code, "Something went wrong on the server. Check the engine log.")
	}
	return writeError(c, status, code, err.Error())
}

// ErrorHandler answers errors that escape handlers (unknown routes,
// oversized bodies) with the envelope.
func ErrorHandler(log *slog.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		var fe *fiber.Error
		if errors.As(err, &fe) && fe.Code < http.StatusInternalServerError {
			// Framework messages can quote the raw request, body included
			// (a password on login), so only the status text goes back.
			code := CodeValidation
			switch fe.Code {
			case http.StatusRequestEntityTooLarge:
				return writeError(c, fe.Code, CodeTooLarge, "The upload is too large.")
			case http.StatusNotFound:
				code = CodeNotFound
			case http.StatusUnauthorized:
				code = CodeUnauthorized
			case http.StatusForbidden:
				code = CodeForbidden
			}
			return writeError(c, fe.Code, code, http.StatusText(fe.Code)+".")
		}
		return respond(c, log, err)
	}
}
