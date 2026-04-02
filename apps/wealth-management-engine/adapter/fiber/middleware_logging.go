package fiber

import (
	logger "apps/wealth-management-engine/adapter/logger"
	"context"
	"fmt"
	"net/http"
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

		// Store request ID in context for downstream handlers
		ctx := context.WithValue(c.UserContext(), "request_id", requestID)
		c.SetUserContext(ctx)

		return c.Next()
	}
}

// LoggingMiddleware logs HTTP request and response details.
func LoggingMiddleware(log *logger.Logger) fiberLib.Handler {
	return func(c *fiberLib.Ctx) error {
		// Extract request context
		requestID := c.Get("X-Request-ID")
		method := c.Method()
		path := c.Path()
		query := c.Request().URI().QueryString()
		remoteAddr := c.IP()
		userAgent := c.Get("User-Agent")

		// Measure request latency
		start := time.Now()
		err := c.Next()
		duration := time.Since(start)

		// Extract response details
		statusCode := c.Response().StatusCode()
		responseSize := len(c.Response().Body())

		// Build log attributes
		attrs := []slog.Attr{
			slog.String("request_id", requestID),
			slog.String("method", method),
			slog.String("path", path),
			slog.String("status", fmt.Sprintf("%d", statusCode)),
			slog.String("latency_ms", fmt.Sprintf("%.2f", float64(duration.Milliseconds()))),
			slog.String("remote_addr", remoteAddr),
			slog.String("user_agent", userAgent),
			slog.Int("response_size_bytes", responseSize),
		}

		if query != nil {
			attrs = append(attrs, slog.String("query", string(query)))
		}

		// Log at appropriate level based on status code
		if statusCode >= 500 {
			log.LogAttrs(c.UserContext(), slog.LevelError, "http_request", attrs...)
		} else if statusCode >= 400 {
			log.LogAttrs(c.UserContext(), slog.LevelWarn, "http_request", attrs...)
		} else {
			log.LogAttrs(c.UserContext(), slog.LevelInfo, "http_request", attrs...)
		}

		// Use LogRequestEvent for structured request logging
		log.LogRequestEvent(c.UserContext(),
			fmt.Sprintf("%s %s", method, path),
			slog.String("request_id", requestID),
			slog.Int("status_code", statusCode),
			slog.String("latency_ms", fmt.Sprintf("%.2f", float64(duration.Milliseconds()))),
			slog.String("remote_addr", remoteAddr),
		)

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
				_ = c.JSON(fiberLib.Map{"error": "internal server error"})
			}
		}()
		return c.Next()
	}
}
