package google_sheets

import (
	"apps/wealth-management-engine/adapter/logger"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"apps/wealth-management-engine/domain"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

func TestReadSheetRefreshesTokenAndNormalizesValues(t *testing.T) {
	var tokenHits atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			tokenHits.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`))
		case strings.HasPrefix(r.URL.Path, "/v4/spreadsheets/spreadsheet-123/values/Accounts!A2:I"):
			if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
				t.Fatalf("expected bearer token, got %q", got)
			}
			if got := r.URL.Query().Get("valueRenderOption"); got != "UNFORMATTED_VALUE" {
				t.Fatalf("expected valueRenderOption=UNFORMATTED_VALUE, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"range": "Accounts!A2:I",
				"values": [][]any{
					{"Cash Wallet", 1200, nil},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, domain.SheetsConfig{
		ClientID:      "client-id",
		ClientSecret:  "client-secret",
		RefreshToken:  "refresh-token",
		SpreadsheetID: "spreadsheet-123",
		RedirectURL:   "http://127.0.0.1:3000",
	})

	rows, err := client.ReadSheet("Accounts!A2:I")
	if err != nil {
		t.Fatalf("ReadSheet returned error: %v", err)
	}

	if tokenHits.Load() != 1 {
		t.Fatalf("expected token refresh to happen once, got %d", tokenHits.Load())
	}

	if len(rows) != 1 || len(rows[0]) != 3 {
		t.Fatalf("unexpected row shape: %#v", rows)
	}
	if rows[0][0] != "Cash Wallet" || rows[0][1] != "1200" || rows[0][2] != "" {
		t.Fatalf("unexpected normalized values: %#v", rows[0])
	}
}

func TestAppendRowUsesUserEnteredMode(t *testing.T) {
	var tokenHits atomic.Int32
	var capturedQuery url.Values
	var capturedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			tokenHits.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"append-token","token_type":"Bearer","expires_in":3600}`))
		case strings.HasPrefix(r.URL.Path, "/v4/spreadsheets/spreadsheet-123/values/Transactions!A2:append"):
			if got := r.Header.Get("Authorization"); got != "Bearer append-token" {
				t.Fatalf("expected bearer token, got %q", got)
			}
			capturedQuery = r.URL.Query()
			if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"updates": map[string]any{"updatedRows": 1}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, domain.SheetsConfig{
		ClientID:      "client-id",
		ClientSecret:  "client-secret",
		RefreshToken:  "refresh-token",
		SpreadsheetID: "spreadsheet-123",
		RedirectURL:   "http://127.0.0.1:3000",
	})

	err := client.AppendRow("Transactions!A2", []any{"Brokerage", 55.25, "Deposit"})
	if err != nil {
		t.Fatalf("AppendRow returned error: %v", err)
	}

	if tokenHits.Load() != 1 {
		t.Fatalf("expected token refresh to happen once, got %d", tokenHits.Load())
	}
	if got := capturedQuery.Get("valueInputOption"); got != "USER_ENTERED" {
		t.Fatalf("expected valueInputOption=USER_ENTERED, got %q", got)
	}

	values, ok := capturedBody["values"].([]any)
	if !ok || len(values) != 1 {
		t.Fatalf("unexpected append body: %#v", capturedBody)
	}
}

func TestUpdateRowUsesUserEnteredMode(t *testing.T) {
	var capturedQuery url.Values
	var capturedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"update-token","token_type":"Bearer","expires_in":3600}`))
		case strings.HasPrefix(r.URL.Path, "/v4/spreadsheets/spreadsheet-123/values/EmailNotifications!F9"):
			capturedQuery = r.URL.Query()
			if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"updatedRange": "EmailNotifications!F9"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, domain.SheetsConfig{
		ClientID:      "client-id",
		ClientSecret:  "client-secret",
		RefreshToken:  "refresh-token",
		SpreadsheetID: "spreadsheet-123",
		RedirectURL:   "http://127.0.0.1:3000",
	})

	if err := client.UpdateRow("EmailNotifications!F9", []any{"done"}); err != nil {
		t.Fatalf("UpdateRow returned error: %v", err)
	}
	if got := capturedQuery.Get("valueInputOption"); got != "USER_ENTERED" {
		t.Fatalf("expected valueInputOption=USER_ENTERED, got %q", got)
	}
	values, ok := capturedBody["values"].([]any)
	if !ok || len(values) != 1 {
		t.Fatalf("unexpected update body: %#v", capturedBody)
	}
}

func TestWriteToFirstEmptyRowUpdatesNextDataRow(t *testing.T) {
	var updatePath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"write-token","token_type":"Bearer","expires_in":3600}`))
		case strings.HasPrefix(r.URL.Path, "/v4/spreadsheets/spreadsheet-123/values/Transactions!A2:A"):
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"range":  "Transactions!A2:A",
				"values": [][]any{{"Wallet"}, {"Brokerage"}, {""}},
			})
		case strings.HasPrefix(r.URL.Path, "/v4/spreadsheets/spreadsheet-123/values/Transactions!A4"):
			updatePath = r.URL.Path
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"updatedRange": "Transactions!A4"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, domain.SheetsConfig{
		ClientID:      "client-id",
		ClientSecret:  "client-secret",
		RefreshToken:  "refresh-token",
		SpreadsheetID: "spreadsheet-123",
		RedirectURL:   "http://127.0.0.1:3000",
	})

	if err := client.WriteToFirstEmptyRow("Transactions", "Transactions!A2:A", []any{"Wallet", "31/03/2026"}); err != nil {
		t.Fatalf("WriteToFirstEmptyRow returned error: %v", err)
	}
	if updatePath == "" {
		t.Fatalf("expected update request for next data row")
	}
}

func newTestClient(t *testing.T, baseURL string, config domain.SheetsConfig) *SheetsClient {
	t.Helper()

	ctx := context.Background()
	oauthConfig := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Scopes:       []string{sheets.SpreadsheetsScope},
		Endpoint: oauth2.Endpoint{
			TokenURL: baseURL + "/token",
		},
	}
	httpClient := oauth2.NewClient(ctx, oauthConfig.TokenSource(ctx, &oauth2.Token{
		RefreshToken: config.RefreshToken,
	}))

	service, err := sheets.NewService(
		ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(baseURL+"/"),
	)
	if err != nil {
		t.Fatalf("failed to create sheets service: %v", err)
	}

	testLog := logger.NewTestLogger(t)
	return NewSheetsClientWithService(service, config.SpreadsheetID, testLog)
}
