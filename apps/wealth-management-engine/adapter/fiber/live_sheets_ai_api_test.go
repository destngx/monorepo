package fiber

import (
	"net/http"
	"strings"
	"testing"
)

func TestE2ELiveSheetsAccountsAPI(t *testing.T) {
	ensureLiveAPITestsEnabled(t)

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
