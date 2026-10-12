package fiber

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/internal/fakes"
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
