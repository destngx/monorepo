package httptransport

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"apps/ai-gateway/internal/domain"
	"apps/ai-gateway/internal/service"
	"github.com/fatih/color"
)

const (
	errMsgInternalServer = "internal server error"
)

var (
	methodColors = map[string]func(string, ...any) string{
		"GET":    color.CyanString,
		"POST":   color.GreenString,
		"PUT":    color.YellowString,
		"DELETE": color.RedString,
	}
)

func statusColor(code int) func(string, ...any) string {
	switch {
	case code >= 500:
		return color.RedString
	case code >= 400:
		return color.YellowString
	default:
		return color.GreenString
	}
}

const HeaderAIProvider = domain.HeaderAIProvider

// Chain applies middlewares in order.
func Chain(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

// Logger logs method, path, provider, model, status, and duration, including a unique Request ID.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Generate simple RequestID
		tid := make([]byte, 4)
		rand.Read(tid)
		requestID := hex.EncodeToString(tid)

		// Add to context
		ctx := context.WithValue(r.Context(), domain.RequestIDKey, requestID)
		ctx = context.WithValue(ctx, domain.LogMetaKey, &domain.RequestLogMeta{})
		r = r.WithContext(ctx)

		provider := r.Header.Get(HeaderAIProvider)
		if provider == "" {
			provider = domain.ProviderGitHubCopilot
		}

		rw := &ResponseWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(rw, r)

		meta, _ := r.Context().Value(domain.LogMetaKey).(*domain.RequestLogMeta)
		mapping := ""
		model := ""
		if meta != nil {
			mapping = meta.Mapping
			model = meta.Model
		}

		method := r.Method
		if c, ok := methodColors[method]; ok {
			method = c(method)
		}

		status := fmt.Sprintf("%d", rw.status)
		status = statusColor(rw.status)(status)

		slog.Info("Request",
			"method", method,
			"path", r.URL.Path,
			"status", status,
			"provider", provider,
			"model", model,
			"duration", time.Since(start),
			"mapping", mapping,
			"rid", requestID,
		)
	})
}

// Metrics captures request metrics and records them to the collector.
func Metrics(collector *service.MetricsCollector) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Inject mutable payload pointer BEFORE handler runs
			payload := &domain.MetricsPayload{}
			ctx := context.WithValue(r.Context(), domain.MetricsPayloadKey, payload)

			rw := &ResponseWriter{ResponseWriter: w, status: 200}
			next.ServeHTTP(rw, r.WithContext(ctx))

			// AFTER handler returns — read the populated pointer.
			// Only record if an AI provider was actually used (set by handlers).
			if payload.Provider != "" {
				collector.Record(domain.RequestRecord{
					Timestamp:  start,
					Route:      r.URL.Path,
					Provider:   payload.Provider,
					Model:      payload.Model,
					InputModel: payload.InputModel,
					Stream:     payload.Stream,
					StatusCode: rw.status,
					DurationMs: time.Since(start).Milliseconds(),
					Usage:      payload.Usage,
					Error:      payload.Error,
				})
			}
		})
	}
}

// SetLogMapping attaches model mapping metadata to the request context.
func SetLogMapping(r *http.Request, mapping string) *http.Request {
	if meta, ok := r.Context().Value(domain.LogMetaKey).(*domain.RequestLogMeta); ok && meta != nil {
		meta.Mapping = mapping
	}
	return r
}

// SetLogModel attaches the resolved model metadata to the request context.
func SetLogModel(r *http.Request, model string) *http.Request {
	if meta, ok := r.Context().Value(domain.LogMetaKey).(*domain.RequestLogMeta); ok && meta != nil {
		meta.Model = model
	}
	return r
}

// Recovery catches panics and returns 500.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered", "error", rec)
				http.Error(w, errMsgInternalServer, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// CORS allows all local origins (suitable for local-only gateway).
func CORS(next http.Handler, allowOrigin, allowMethods, allowHeaders string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		w.Header().Set("Access-Control-Allow-Methods", allowMethods)
		w.Header().Set("Access-Control-Allow-Headers", allowHeaders)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type ResponseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *ResponseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *ResponseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
