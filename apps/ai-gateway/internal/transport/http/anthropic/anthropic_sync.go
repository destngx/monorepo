package anthropic

import (
	"apps/ai-gateway/internal/transport/http/common"
	"encoding/json"
	"log/slog"
	"net/http"

	"apps/ai-gateway/internal/domain"
	"apps/ai-gateway/internal/providers/shared"
)

func (h *AnthropicHandler) handleSync(w http.ResponseWriter, r *http.Request, p shared.Provider, req domain.ChatRequest, inputModel string, wasStream bool) {
	rid, _ := r.Context().Value(domain.RequestIDKey).(string)
	if h.registry.Config.Verbose >= 2 {
		slog.Debug("Entering Anthropic handleSync", "rid", rid)
	}

	resp, err := p.Chat(r.Context(), req)
	if err != nil {
		if h.registry.Config.Verbose >= 1 {
			slog.Warn("Provider returned error", "rid", rid, "error", err)
		}
		common.WriteError(w, r, http.StatusBadGateway, err.Error())
		common.SetMetrics(r, p.Name(), req.Model, inputModel, domain.Usage{}, req.Stream, err)
		return
	}

	common.SetMetrics(r, p.Name(), req.Model, inputModel, resp.Usage, req.Stream, nil)
	if wasStream {
		h.writeAnthroStreamResponse(w, resp, inputModel)
		return
	}

	anthroResp := convertToAnthropicResponse(resp, inputModel)
	if h.registry.Config.Verbose >= 1 {
		body, _ := json.MarshalIndent(anthroResp, "", "  ")
		slog.Info("FULL ANTHROPIC RESPONSE", "rid", rid, "body", string(body))
	}
	w.Header().Set(common.HeaderContentType, common.ContentTypeJSON)
	_ = json.NewEncoder(w).Encode(anthroResp)
}
