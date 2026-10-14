package fiber

import (
	logger "apps/wealth-management/adapter/logger"
	"fmt"
	"net/http"
	"net/url"
	"time"

	fiberLib "github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"log/slog"
)

// RequestIDMiddleware adds a unique request ID to each request for tracing.
func RequestIDMiddleware(log *logger.Logger) fiberLib.Handler {
	return func(c *fiberLib.Ctx) error {
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("X-Request-ID", requestID)

		// Every log line written with the request context carries the ID.
		c.SetUserContext(logger.WithRequestID(c.UserContext(), requestID))

		return c.Next()
	}
}

// LoggingMiddleware writes one log line per request, at error level for
// 5xx and warn for 4xx. The request ID comes from the context.
func LoggingMiddleware(log *logger.Logger) fiberLib.Handler {
	return func(c *fiberLib.Ctx) error {
		start := time.Now()
		err := c.Next()
		duration := time.Since(start)

		statusCode := c.Response().StatusCode()
		attrs := []slog.Attr{
			slog.String("log_type", string(logger.LogTypeRequest)),
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.Int("status", statusCode),
			slog.Float64("latency_ms", float64(duration.Microseconds())/1000),
			slog.String("remote_addr", c.IP()),
			slog.String("user_agent", c.Get("User-Agent")),
			slog.Int("response_size_bytes", len(c.Response().Body())),
		}
		if query := c.Request().URI().QueryString(); len(query) > 0 {
			attrs = append(attrs, slog.String("query", redactQuery(string(query))))
		}

		level := slog.LevelInfo
		if statusCode >= 500 {
			level = slog.LevelError
		} else if statusCode >= 400 {
			level = slog.LevelWarn
		}
		log.LogAttrs(c.UserContext(), level, c.Method()+" "+c.Path(), attrs...)
		return err
	}
}

// RecoveryMiddleware logs panics and recovers gracefully.
func RecoveryMiddleware(log *logger.Logger) fiberLib.Handler {
	return func(c *fiberLib.Ctx) (err error) {
		defer func() {
			if r := recover(); r != nil {
				requestID := c.Get("X-Request-ID")
				log.LogAttrs(c.UserContext(), slog.LevelError,
					"panic_recovered",
					slog.String("request_id", requestID),
					slog.String("panic", fmt.Sprintf("%v", r)),
					slog.String("method", c.Method()),
					slog.String("path", c.Path()),
				)
				c.Status(http.StatusInternalServerError)
				_ = c.JSON(ErrorBody{Error: "internal server error", Code: CodeInternal})
			}
		}()
		return c.Next()
	}
}

// redactedQueryKeys never reach the log: the Google sign-in callback carries
// an authorization code and its state in the query.
var redactedQueryKeys = []string{"code", "state"}

func redactQuery(raw string) string {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return "[unparsable query]"
	}
	redacted := false
	for _, key := range redactedQueryKeys {
		if values.Has(key) {
			values.Set(key, "REDACTED")
			redacted = true
		}
	}
	if !redacted {
		return raw
	}
	return values.Encode()
}
