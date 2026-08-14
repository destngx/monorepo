package anthropic

import (
	"apps/ai-gateway/internal/transport/http/common"
	"io"
	"log/slog"
	"net/http"

	"apps/ai-gateway/internal/domain"
	"apps/ai-gateway/internal/providers/shared"
)

func (h *AnthropicHandler) handleStream(w http.ResponseWriter, r *http.Request, p shared.Provider, req domain.ChatRequest, inputModel string) {
	rid, _ := r.Context().Value(domain.RequestIDKey).(string)
	if h.registry.Config.Verbose >= 2 {
		slog.Debug("Entering Anthropic handleStream", "rid", rid)
	}
	w.Header().Set(common.HeaderContentType, common.ContentTypeEventStream)
	w.Header().Set(common.HeaderCacheControl, common.ValueNoCache)
	w.Header().Set(common.HeaderConnection, common.ValueKeepAlive)
	w.Header().Set(common.HeaderXAccelBuffering, common.ValueNo)
	if _, ok := w.(http.Flusher); !ok {
		common.WriteError(w, r, http.StatusInternalServerError, common.ErrMsgStreamNotSupp)
		return
	}
	pr, pw := io.Pipe()
	errCh := make(chan error, 1)
	go func() {
		defer pw.Close()
		providerWriter := io.Writer(pw)
		if h.registry.Config.Verbose >= 2 {
			providerWriter = &common.RawStreamLogWriter{W: pw, RID: rid}
		}
		usage, err := p.ChatStream(r.Context(), req, providerWriter)
		errCh <- err
		common.SetMetrics(r, p.Name(), req.Model, inputModel, usage, req.Stream, err)
	}()
	var writer io.Writer = w
	if h.registry.Config.Verbose >= 1 {
		writer = &common.StreamLogWriter{W: w, RID: rid}
	}
	eventCount, convertErr := convertToAnthropicStream(pr, writer, inputModel)
	if convertErr != nil {
		slog.Error("Stream convert error", "rid", rid, "error", convertErr)
		sendAnthroEvent(writer, eventError, map[string]any{"error": map[string]any{"type": typeAnthroAPIError, "message": convertErr.Error()}})
		if flusher, ok := writer.(interface{ Flush() }); ok {
			flusher.Flush()
		}
	}
	if err := <-errCh; err != nil {
		slog.Error("Anthropic stream error", "rid", rid, "error", err)
		if eventCount == 0 && convertErr == nil {
			sendAnthroEvent(writer, eventError, map[string]any{"error": map[string]any{"type": typeAnthroAPIError, "message": err.Error()}})
			if flusher, ok := writer.(interface{ Flush() }); ok {
				flusher.Flush()
			}
		}
	}
}
