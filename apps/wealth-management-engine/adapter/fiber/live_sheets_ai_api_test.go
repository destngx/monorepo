package fiber

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestE2ELiveSheetsAccountsAPI(t *testing.T) {
	if strings.ToLower(os.Getenv("RUN_LIVE_API_TESTS")) != "true" {
		t.Skip("set RUN_LIVE_API_TESTS=true to run live API e2e tests")
	}

	baseURL := liveBaseURL()
	response, body := performRequest(t, http.MethodGet, baseURL+"/api/sheets/accounts")

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from /api/sheets/accounts, got %d body=%s", response.StatusCode, string(body))
	}

	var payload map[string]any
	decodeJSON(t, body, &payload)

	rangeValue, _ := payload["range"].(string)
	if !strings.Contains(rangeValue, "Accounts!") {
		t.Fatalf("expected accounts range in response, got %+v", payload)
	}

	rowsRaw, ok := payload["rows"].([]any)
	if !ok {
		t.Fatalf("expected rows array in response, got %+v", payload)
	}
	if len(rowsRaw) > 0 {
		firstRow, ok := rowsRaw[0].([]any)
		if !ok || len(firstRow) == 0 {
			t.Fatalf("expected first row to be non-empty, got %+v", rowsRaw[0])
		}
	}
}

func TestE2ELiveAIJSONAPI(t *testing.T) {
	if strings.ToLower(os.Getenv("RUN_LIVE_API_TESTS")) != "true" {
		t.Skip("set RUN_LIVE_API_TESTS=true to run live API e2e tests")
	}

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
	if strings.ToLower(os.Getenv("RUN_LIVE_API_TESTS")) != "true" {
		t.Skip("set RUN_LIVE_API_TESTS=true to run live API e2e tests")
	}

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
	if strings.ToLower(os.Getenv("RUN_LIVE_API_TESTS")) != "true" {
		t.Skip("set RUN_LIVE_API_TESTS=true to run live API e2e tests")
	}

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

func performJSONRequest(t *testing.T, method string, url string, payload map[string]any) (*http.Response, []byte) {
	t.Helper()
	rawBody, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal json payload: %v", err)
	}
	request, err := http.NewRequest(method, url, bytes.NewReader(rawBody))
	if err != nil {
		t.Fatalf("request build failed: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request failed for %s %s: %v", method, url, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("failed reading response body: %v", err)
	}
	return response, body
}

func liveBaseURL() string {
	baseURL := os.Getenv("LIVE_API_BASE_URL")
	if strings.TrimSpace(baseURL) == "" {
		return "http://localhost:8080"
	}
	return baseURL
}

func toString(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}
