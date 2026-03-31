package fiber

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestE2ELiveMarketAPISmoke(t *testing.T) {
	if strings.ToLower(os.Getenv("RUN_LIVE_API_TESTS")) != "true" {
		t.Skip("set RUN_LIVE_API_TESTS=true to run live market API smoke tests")
	}

	baseURL := os.Getenv("LIVE_API_BASE_URL")
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "http://localhost:8080"
	}

	clearMarketCache(t, baseURL)

	assertJSONEndpoint200(t, baseURL, "/api/external/fmarket/health", func(t *testing.T, body []byte) {
		var payload map[string]any
		decodeJSON(t, body, &payload)
		if payload["provider"] != "fmarket" || payload["status"] != "ok" {
			t.Fatalf("expected provider=fmarket, got %+v", payload)
		}
	})

	assertJSONEndpoint200(t, baseURL, "/api/external/vnstock/health", func(t *testing.T, body []byte) {
		var payload map[string]any
		decodeJSON(t, body, &payload)
		if payload["provider"] != "vnstock" || payload["status"] != "ok" {
			t.Fatalf("expected provider=vnstock, got %+v", payload)
		}
	})

	assertJSONEndpoint200(t, baseURL, "/api/market/ticker?symbol=VFMVF1&type=ifc", func(t *testing.T, body []byte) {
		var payload map[string]any
		decodeJSON(t, body, &payload)
		if payload["provider"] != "fmarket" || payload["type"] != "ifc" || payload["currency"] != "VND" {
			t.Fatalf("expected provider=fmarket for IFC ticker, got %+v", payload)
		}
		if price := asFloat(payload["price"]); price <= 0 {
			t.Fatalf("expected IFC ticker price > 0, got %+v", payload)
		}
		if symbol := strings.TrimSpace(fmt.Sprintf("%v", payload["symbol"])); symbol != "VFMVF1" {
			t.Fatalf("expected IFC ticker symbol VFMVF1, got %+v", payload)
		}
	})

	assertJSONEndpoint200(t, baseURL, "/api/market/ticker?symbol=ACB&type=equity", func(t *testing.T, body []byte) {
		var payload map[string]any
		decodeJSON(t, body, &payload)
		if payload["provider"] != "vnstock" || payload["type"] != "equity" || payload["currency"] != "VND" {
			t.Fatalf("expected provider=vnstock for equity ticker, got %+v", payload)
		}
		if price := asFloat(payload["price"]); price <= 0 {
			t.Fatalf("expected equity ticker price > 0, got %+v", payload)
		}
		if timestamp := strings.TrimSpace(fmt.Sprintf("%v", payload["timestamp"])); timestamp == "" {
			t.Fatalf("expected equity ticker timestamp, got %+v", payload)
		}
	})

	assertJSONEndpoint200(t, baseURL, "/api/market/ticker?symbol=SJC&type=gold", func(t *testing.T, body []byte) {
		var payload map[string]any
		decodeJSON(t, body, &payload)
		if payload["provider"] != "fmarket" || payload["type"] != "gold" || payload["currency"] != "VND" {
			t.Fatalf("expected provider=fmarket for gold ticker, got %+v", payload)
		}
		bid := asFloat(payload["bid"])
		ask := asFloat(payload["ask"])
		if bid <= 0 || ask <= 0 || ask < bid {
			t.Fatalf("expected valid gold bid/ask (ask >= bid > 0), got %+v", payload)
		}
	})

	assertJSONEndpoint200(t, baseURL, "/api/market/exchange-rate?from=USD&to=VND", func(t *testing.T, body []byte) {
		var payload map[string]any
		decodeJSON(t, body, &payload)
		if payload["provider"] != "fmarket" || payload["from"] != "USD" || payload["to"] != "VND" {
			t.Fatalf("expected provider=fmarket for exchange rate, got %+v", payload)
		}
		rate := asFloat(payload["rate"])
		if rate < 10000 || rate > 50000 {
			t.Fatalf("expected plausible USD_VND rate, got %+v", payload)
		}
	})

	assertJSONEndpoint200(t, baseURL, "/api/market/price-series?symbol=GOLD&type=gold_usd", func(t *testing.T, body []byte) {
		var payload map[string]any
		decodeJSON(t, body, &payload)
		if payload["provider"] != "fmarket" || payload["seriesType"] != "gold_usd" || payload["currency"] != "USD" {
			t.Fatalf("unexpected price-series payload: %+v", payload)
		}
		data, ok := payload["data"].(map[string]any)
		if !ok {
			t.Fatalf("expected price-series data object, got %+v", payload)
		}
		points, ok := data["points"].([]any)
		if !ok || len(points) == 0 {
			t.Fatalf("expected non-empty gold_usd points, got %+v", payload)
		}
		firstPoint, ok := points[0].(map[string]any)
		if !ok || asFloat(firstPoint["price"]) <= 0 {
			t.Fatalf("expected first point price > 0, got %+v", payload)
		}
	})

	assertJSONEndpoint200(t, baseURL, "/api/market/bank-rates", func(t *testing.T, body []byte) {
		var payload []map[string]any
		decodeJSON(t, body, &payload)
		if len(payload) == 0 {
			t.Fatalf("expected non-empty bank rates payload")
		}
		validRows := 0
		for _, row := range payload {
			bank := strings.TrimSpace(fmt.Sprintf("%v", row["bank"]))
			rate := asFloat(row["rate"])
			if bank != "" && rate > 0 && !math.IsNaN(rate) {
				validRows++
			}
		}
		if validRows == 0 {
			t.Fatalf("expected at least one valid bank-rate row, got %+v", payload)
		}
	})
}

func assertJSONEndpoint200(t *testing.T, baseURL string, path string, assertFn func(t *testing.T, body []byte)) {
	t.Helper()
	response, body := performRequest(t, http.MethodGet, baseURL+path)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for %s, got %d body=%s", path, response.StatusCode, string(body))
	}
	assertFn(t, body)
}

func clearMarketCache(t *testing.T, baseURL string) {
	t.Helper()
	response, body := performRequest(t, http.MethodDelete, baseURL+"/api/cache/invalidate?pattern=market:*")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("failed clearing market cache: status=%d body=%s", response.StatusCode, string(body))
	}
}

func performRequest(t *testing.T, method string, url string) (*http.Response, []byte) {
	t.Helper()
	request, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatalf("request build failed: %v", err)
	}
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

func decodeJSON(t *testing.T, raw []byte, target interface{}) {
	t.Helper()
	if err := json.Unmarshal(raw, target); err != nil {
		t.Fatalf("json decode failed: %v; body=%s", err, string(raw))
	}
}

func asFloat(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		parsed, _ := v.Float64()
		return parsed
	default:
		return 0
	}
}
