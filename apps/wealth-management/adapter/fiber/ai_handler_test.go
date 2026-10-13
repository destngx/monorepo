package fiber

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/internal/fakes"
	"apps/wealth-management/port"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func newAIApp(t *testing.T, ai *fakes.AI) *fiber.App {
	t.Helper()
	app := fiber.New()
	RegisterAIRoutes(NewRouteRegistrar(app, "/api", ""), NewAIHandler(ai, logger.NewTestLogger(t)), NewWriteGuard(testToken, "0.0.0.0"))
	return app
}

func TestAIRoutes(t *testing.T) {
	ai := &fakes.AI{}
	app := newAIApp(t, ai)

	if status, body := call(t, app, http.MethodGet, "/api/ai/status", ""); status != http.StatusOK || !strings.Contains(body, `"missingPrompts":["tasks/loan-review"]`) {
		t.Fatalf("status: %d %s", status, body)
	}
	if status, body := call(t, app, http.MethodGet, "/api/ai/usage", ""); status != http.StatusOK || !strings.Contains(body, `"windows":[{"name":"5h","usedPercent":12}]`) {
		t.Fatalf("usage: %d %s", status, body)
	}
	if status, body := call(t, app, http.MethodPost, "/api/ai/review", `{"kind":"budget-review","month":"2026-09","force":true}`); status != http.StatusOK || len(ai.Reviews) != 1 || ai.Reviews[0].Month != "2026-09" || !ai.Reviews[0].Force {
		t.Fatalf("review: %d %s %+v", status, body, ai.Reviews)
	}
	if status, body := call(t, app, http.MethodPost, "/api/ai/chart-insight", `{"chartType":"line","data":[1]}`); status != http.StatusOK || body != `{"insight":"**flat**"}` {
		t.Fatalf("chart insight: %d %s", status, body)
	}
	if status, _ := call(t, app, http.MethodPost, "/api/ai/briefing", ""); status != http.StatusOK {
		t.Fatalf("briefing without body: %d", status)
	}
	if status, _ := call(t, app, http.MethodPost, "/api/ai/briefing", "", "Authorization", ""); status != http.StatusUnauthorized {
		t.Fatalf("AI calls must be guarded, got %d", status)
	}
}

func TestAIErrorsMapToStatus(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
	}{
		{fmt.Errorf("%w: no key", domain.ErrAIUnavailable), http.StatusServiceUnavailable},
		{fmt.Errorf("%w: tasks/x", domain.ErrPromptMissing), http.StatusServiceUnavailable},
		{fmt.Errorf("%w: not json", domain.ErrAIOutput), http.StatusBadGateway},
		{fmt.Errorf("%w: short", domain.ErrValidation), http.StatusBadRequest},
	} {
		app := newAIApp(t, &fakes.AI{Err: tc.err})
		if status, body := call(t, app, http.MethodPost, "/api/ai/suggest-category", `{"payee":"x"}`); status != tc.status {
			t.Fatalf("%v: got %d %s", tc.err, status, body)
		}
	}
}

func TestAIUsageUnavailableIs503(t *testing.T) {
	app := newAIApp(t, &fakes.AI{Err: fmt.Errorf("%w: provider returned 404: provider openai not configured", domain.ErrAIUnavailable)})
	if status, body := call(t, app, http.MethodGet, "/api/ai/usage", ""); status != http.StatusServiceUnavailable || !strings.Contains(body, "UNAVAILABLE") {
		t.Fatalf("usage: %d %s", status, body)
	}
}

func TestAIChatStreamsNDJSON(t *testing.T) {
	status, body := call(t, newAIApp(t, &fakes.AI{}), http.MethodPost, "/api/ai/chat", `{"messages":[{"role":"user","content":"hi"}]}`)
	if status != http.StatusOK || body != "{\"type\":\"tool\",\"tool\":\"getBudgetStatus\",\"status\":\"running\"}\n{\"type\":\"text\",\"delta\":\"Hel\"}\n{\"type\":\"text\",\"delta\":\"lo\"}\n{\"type\":\"done\",\"model\":\"m\"}\n" {
		t.Fatalf("chat: %d %q", status, body)
	}

	status, body = call(t, newAIApp(t, &fakes.AI{ChatErr: fmt.Errorf("%w: gateway down", domain.ErrAIUnavailable)}), http.MethodPost, "/api/ai/chat", `{"messages":[{"role":"user","content":"hi"}]}`)
	if status != http.StatusOK || !strings.HasSuffix(body, "{\"type\":\"error\",\"message\":\"ai unavailable: gateway down\"}\n") {
		t.Fatalf("chat error event: %d %q", status, body)
	}

	status, body = call(t, newAIApp(t, &fakes.AI{ChatErr: fmt.Errorf("boom")}), http.MethodPost, "/api/ai/chat", `{"messages":[{"role":"user","content":"hi"}]}`)
	if !strings.Contains(body, "The assistant failed") || strings.Contains(body, "boom") {
		t.Fatalf("internal errors must not leak: %d %q", status, body)
	}

	if status, _ := call(t, newAIApp(t, &fakes.AI{}), http.MethodPost, "/api/ai/chat", `{"messages":[]}`); status != http.StatusBadRequest {
		t.Fatalf("empty chat should be 400, got %d", status)
	}
}

func TestAICaptureStreamsProgressThenResult(t *testing.T) {
	ai := &fakes.AI{}
	body := `{"materials":[{"id":"m1","kind":"image","mediaType":"image/png","data":"iVBORw=="},{"id":"m2","kind":"text","text":"-50k"}]}`
	status, out := call(t, newAIApp(t, ai), http.MethodPost, "/api/ai/capture", body)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if status != http.StatusOK || len(lines) != 2 || lines[0] != `{"type":"progress","stage":"extract","materialId":"m1","status":"ok","events":1}` || !strings.HasPrefix(lines[1], `{"type":"result","result":{"suggestions":[{"id":"s1"`) {
		t.Fatalf("capture: %d %q", status, out)
	}
	if len(ai.Captures) != 1 || string(ai.Captures[0].Materials[0].Data) != "\x89PNG" {
		t.Fatalf("base64 image not decoded: %+v", ai.Captures)
	}

	if status, _ := call(t, newAIApp(t, &fakes.AI{}), http.MethodPost, "/api/ai/capture", `{"materials":[]}`); status != http.StatusBadRequest {
		t.Fatalf("empty batch should be 400, got %d", status)
	}
	if status, _ := call(t, newAIApp(t, &fakes.AI{}), http.MethodPost, "/api/ai/capture", body, "Authorization", ""); status != http.StatusUnauthorized {
		t.Fatalf("capture must be guarded, got %d", status)
	}

	status, out = call(t, newAIApp(t, &fakes.AI{Err: fmt.Errorf("boom")}), http.MethodPost, "/api/ai/capture", body)
	if status != http.StatusOK || !strings.HasSuffix(out, "{\"type\":\"error\",\"message\":\"Smart Capture failed. Try again.\"}\n") {
		t.Fatalf("internal errors must become a generic error event: %q", out)
	}
}

func TestAICaptureRespectsConfiguredLimits(t *testing.T) {
	app := fiber.New()
	handler := NewAIHandler(&fakes.AI{}, logger.NewTestLogger(t)).WithCaptureLimits(domain.CaptureLimits{MaxItems: 1, MaxItemBytes: 100, MaxBatchBytes: 100})
	RegisterAIRoutes(NewRouteRegistrar(app, "/api", ""), handler, NewWriteGuard(testToken, "0.0.0.0"))
	status, body := call(t, app, http.MethodPost, "/api/ai/capture", `{"materials":[{"id":"a","kind":"text","text":"x"},{"id":"b","kind":"text","text":"y"}]}`)
	if status != http.StatusBadRequest || !strings.Contains(body, "at most 1") {
		t.Fatalf("expected limit error, got %d %s", status, body)
	}
}

func TestCaptureProgressEventKeepsTheMessage(t *testing.T) {
	encoded, err := json.Marshal(captureProgressEvent(port.CaptureProgress{Stage: "extract", MaterialID: "m1", Status: "unreadable", Message: "Image too blurry."}))
	if err != nil || string(encoded) != `{"type":"progress","stage":"extract","materialId":"m1","status":"unreadable","message":"Image too blurry."}` {
		t.Fatalf("progress event: %s %v", encoded, err)
	}
}

func TestAIMarketRoutes(t *testing.T) {
	ai := &fakes.AI{}
	app := newAIApp(t, ai)
	if status, body := call(t, app, http.MethodPost, "/api/ai/news-sentiment", `{"topic":"Interest rates","force":true}`); status != http.StatusOK || !strings.Contains(body, `"bias":"BULLISH"`) || !ai.News[0].Force {
		t.Fatalf("news sentiment: %d %s %+v", status, body, ai.News)
	}
	if status, body := call(t, app, http.MethodPost, "/api/ai/ticker-insight", `{"symbol":"FPT","timeframe":"4h"}`); status != http.StatusOK || ai.Tickers[0].Timeframe != "4h" {
		t.Fatalf("ticker insight: %d %s %+v", status, body, ai.Tickers)
	}
	if status, _ := call(t, app, http.MethodPost, "/api/ai/ticker-insight", `{"symbol":" "}`); status != http.StatusBadRequest {
		t.Fatalf("a ticker insight needs a symbol, got %d", status)
	}

	status, body := call(t, app, http.MethodPost, "/api/ai/investment-analysis", "")
	want := "{\"type\":\"phase\",\"phase\":\"think-tank\",\"status\":\"ok\"}\n{\"type\":\"actions\",\"commands\":[{\"action\":\"OBSERVE\",\"target\":\"VN30\",\"rationale\":\"\"}]}\n{\"type\":\"done\"}\n"
	if status != http.StatusOK || body != want {
		t.Fatalf("investment analysis: %d %q", status, body)
	}
	status, body = call(t, newAIApp(t, &fakes.AI{Err: fmt.Errorf("%w: market/think-tank", domain.ErrPromptMissing)}), http.MethodPost, "/api/ai/investment-analysis", `{"model":"m"}`)
	if status != http.StatusOK || !strings.HasSuffix(body, "{\"type\":\"error\",\"message\":\"prompt missing from the Prompts sheet: market/think-tank\"}\n") {
		t.Fatalf("investment analysis error event: %d %q", status, body)
	}
}
