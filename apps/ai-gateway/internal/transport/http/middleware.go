package httptransport

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"time"

	"apps/ai-gateway/internal/domain"
)

const (
	logFormatRequest = "[%s] %s %s [ID:%s] provider=%s status=%d duration=%s%s"
	logFormatPanic   = "panic recovered: %v"

	errMsgInternalServer = "internal server error"
)

const HeaderAIProvider = "X-AI-Provider"
const ProviderGitHub = "github"

// Chain applies middlewares in order.
func Chain(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

// Logger logs method, path, provider, status, and duration, including a unique Request ID.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Generate simple RequestID
		tid := make([]byte, 4)
		rand.Read(tid)
		requestID := hex.EncodeToString(tid)

		// Add to context
		ctx := context.WithValue(r.Context(), domain.RequestIDKey, requestID)
		r = r.WithContext(ctx)

		provider := r.Header.Get(HeaderAIProvider)
		if provider == "" {
			provider = ProviderGitHub
		}

		rw := &ResponseWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(rw, r)

		mapping, _ := r.Context().Value(domain.LogMappingKey).(string)
		mappingStr := ""
		if mapping != "" {
			mappingStr = " mapping=" + mapping
		}

		log.Printf(logFormatRequest,
			r.Method, r.URL.Path, r.RemoteAddr, requestID, provider, rw.status, time.Since(start), mappingStr)
	})
}

// SetLogMapping attaches model mapping metadata to the request context.
func SetLogMapping(r *http.Request, mapping string) *http.Request {
	ctx := context.WithValue(r.Context(), domain.LogMappingKey, mapping)
	return r.WithContext(ctx)
}

// Recovery catches panics and returns 500.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf(logFormatPanic, rec)
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
