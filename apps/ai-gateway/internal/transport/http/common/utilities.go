package common

import (
	"encoding/json"
	"io"
	"log/slog"
	"math"
	"net/http"
	"runtime/debug"

	"apps/ai-gateway/internal/domain"
)

// WriteError writes the gateway's standard JSON error envelope.
func WriteError(w http.ResponseWriter, r *http.Request, code int, msg string) {
	rid, _ := r.Context().Value(domain.RequestIDKey).(string)
	provider, model, mapping, reasoningEffort := "", "", "", ""
	if meta, ok := r.Context().Value(domain.LogMetaKey).(*domain.RequestLogMeta); ok && meta != nil {
		provider = meta.Provider
		model = meta.Model
		mapping = meta.Mapping
		reasoningEffort = meta.ReasoningEffort
	}
	slog.Error("HTTP Error",
		"rid", rid,
		"status", code,
		"method", r.Method,
		"path", r.URL.Path,
		"provider", provider,
		"model", model,
		"reasoning_effort", reasoningEffort,
		"mapping", mapping,
		"message", msg,
	)
	w.Header().Set(HeaderContentType, ContentTypeJSON)
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": msg, "stack": string(debug.Stack())})
}

// StreamLogWriter logs provider stream chunks while forwarding them.
type StreamLogWriter struct {
	W   io.Writer
	RID string
}

func (sw *StreamLogWriter) Write(p []byte) (int, error) {
	slog.Debug("Stream chunk", "rid", sw.RID, "content", string(p))
	return sw.W.Write(p)
}

func (sw *StreamLogWriter) Flush() {
	if f, ok := sw.W.(http.Flusher); ok {
		f.Flush()
	}
}

// RawStreamLogWriter logs provider bytes before protocol conversion.
type RawStreamLogWriter struct {
	W   io.Writer
	RID string
}

func (sw *RawStreamLogWriter) Write(p []byte) (int, error) {
	slog.Info("Raw provider SSE chunk", "rid", sw.RID, "content", string(p))
	return sw.W.Write(p)
}

// SetMetrics records provider usage in the request metrics payload.
func SetMetrics(r *http.Request, provider, model, inputModel string, usage domain.Usage, stream bool, err error) {
	cached, writes := 0, 0
	metadataPresent := usage.PromptTokensDetails != nil
	if metadataPresent {
		cached = usage.PromptTokensDetails.CachedTokens
		writes = usage.PromptTokensDetails.CacheWriteTokens
	}
	ratio := 0.0
	if usage.PromptTokens > 0 {
		ratio = math.Round(float64(cached)/float64(usage.PromptTokens)*1000) / 10
	}
	level := slog.LevelDebug
	if cached > 0 || writes > 0 {
		level = slog.LevelInfo
	}
	slog.Log(r.Context(), level, "Prompt cache usage", "provider", provider, "model", model, "input_tokens", usage.PromptTokens, "cache_read_tokens", cached, "cache_write_tokens", writes, "cache_read_ratio", ratio, "metadata_present", metadataPresent)
	payload, ok := r.Context().Value(domain.MetricsPayloadKey).(*domain.MetricsPayload)
	if !ok || payload == nil {
		return
	}
	payload.Provider, payload.Model, payload.InputModel = provider, model, inputModel
	payload.Usage, payload.Stream = usage, stream
	if err != nil {
		payload.Error = err.Error()
	}
}
