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

func ensureLiveAPITestsEnabled(t *testing.T) {
	t.Helper()
	if strings.ToLower(os.Getenv("RUN_LIVE_API_TESTS")) != "true" {
		t.Skip("set RUN_LIVE_API_TESTS=true to run live API e2e tests")
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
