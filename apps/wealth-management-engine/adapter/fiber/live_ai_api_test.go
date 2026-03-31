package fiber

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestE2ELiveAIJSONAPI(t *testing.T) {
	ensureLiveAPITestsEnabled(t)

	baseURL := liveBaseURL()
	requestBody := map[string]any{
		"systemPrompt":     "You are a strict JSON-only financial assistant.",
		"userPrompt":       "Provide a concise investment briefing in one paragraph.",
		"assistantHistory": "User has moderate risk tolerance.",
	}

	response, body := performJSONRequest(t, http.MethodPost, baseURL+"/api/ai/json", requestBody)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from /api/ai/json, got %d body=%s", response.StatusCode, string(body))
	}

	var payload map[string]any
	decodeJSON(t, body, &payload)
	if strings.TrimSpace(toString(payload["persona"])) == "" {
		t.Fatalf("expected non-empty persona in AI JSON response, got %+v", payload)
	}
	if strings.TrimSpace(toString(payload["summary"])) == "" {
		t.Fatalf("expected non-empty summary in AI JSON response, got %+v", payload)
	}
	if _, ok := payload["actions"].([]any); !ok {
		t.Fatalf("expected actions array in AI JSON response, got %+v", payload)
	}
	if _, ok := payload["roles"].([]any); !ok {
		t.Fatalf("expected roles array in AI JSON response, got %+v", payload)
	}
}

func TestE2ELiveAIStreamAPI(t *testing.T) {
	ensureLiveAPITestsEnabled(t)

	baseURL := liveBaseURL()
	requestBody := map[string]any{
		"prompt": "Give one short sentence about portfolio risk management.",
	}

	response, body := performJSONRequest(t, http.MethodPost, baseURL+"/api/ai/stream", requestBody)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from /api/ai/stream, got %d body=%s", response.StatusCode, string(body))
	}

	contentType := response.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/x-ndjson") {
		t.Fatalf("expected ndjson content type, got %s", contentType)
	}

	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	if len(lines) == 0 {
		t.Fatalf("expected non-empty ndjson stream body")
	}

	parsedLines := 0
	doneSeen := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(line), &payload); err != nil {
			t.Fatalf("expected valid json line in ndjson stream, got line=%q err=%v", line, err)
		}
		parsedLines++
		if done, ok := payload["done"].(bool); ok && done {
			doneSeen = true
		}
	}

	if parsedLines == 0 {
		t.Fatalf("expected at least one parsed ndjson event")
	}
	if !doneSeen {
		t.Fatalf("expected stream to include done=true event, body=%s", string(body))
	}
}

func TestE2ELiveAIToolsAPI(t *testing.T) {
	ensureLiveAPITestsEnabled(t)

	baseURL := liveBaseURL()
	requestBody := map[string]any{
		"prompt": "Check cash balance and market outlook for BTC",
	}

	response, body := performJSONRequest(t, http.MethodPost, baseURL+"/api/ai/tools", requestBody)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from /api/ai/tools, got %d body=%s", response.StatusCode, string(body))
	}

	var payload map[string]any
	decodeJSON(t, body, &payload)
	turns, ok := payload["turns"].([]any)
	if !ok || len(turns) < 6 {
		t.Fatalf("expected tool conversation with at least 6 turns, got %+v", payload)
	}

	firstTurn, _ := turns[0].(map[string]any)
	if toString(firstTurn["role"]) != "user" {
		t.Fatalf("expected first turn role=user, got %+v", firstTurn)
	}

	lastTurn, _ := turns[len(turns)-1].(map[string]any)
	if toString(lastTurn["role"]) != "assistant" || toString(lastTurn["type"]) != "message" {
		t.Fatalf("expected final synthesized assistant message turn, got %+v", lastTurn)
	}
}
