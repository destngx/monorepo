package fiber

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/internal/fakes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

const testToken = "secret"

// newSheetsApp registers the real sheet routes behind a token guard.
func newSheetsApp(t *testing.T, db *fakes.Database) *fiber.App {
	t.Helper()
	app := fiber.New()
	api := NewRouteRegistrar(app, "/api", "")
	RegisterSheetRoutes(api, NewDatabaseHandler(db, logger.NewTestLogger(t)), NewWriteGuard(testToken, "0.0.0.0"))
	return app
}

func call(t *testing.T, app *fiber.App, method, target, body string, headers ...string) (int, string) {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	request := httptest.NewRequest(method, target, reader)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+testToken)
	for i := 0; i+1 < len(headers); i += 2 {
		request.Header.Set(headers[i], headers[i+1])
	}
	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("%s %s: %v", method, target, err)
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return response.StatusCode, string(payload)
}

func TestGetAccountsReturnsTypedAccounts(t *testing.T) {
	app := newSheetsApp(t, &fakes.Database{
		Accounts: []domain.Account{{Name: "Cash", Balance: 1000, Type: domain.AccountTypeCash, Currency: domain.CurrencyVND}},
	})

	status, body := call(t, app, http.MethodGet, "/api/accounts", "")

	var payload []domain.Account
	if status != http.StatusOK || json.Unmarshal([]byte(body), &payload) != nil || len(payload) != 1 || payload[0].Name != "Cash" {
		t.Fatalf("status = %d, body = %s", status, body)
	}
}

func TestCreateTransactionReturnsCreatedID(t *testing.T) {
	db := &fakes.Database{}
	app := newSheetsApp(t, db)

	status, body := call(t, app, http.MethodPost, "/api/transactions", `{"accountName":"Cash","date":"2026-03-31","payee":"Coffee","category":"Food","tags":["daily"],"payment":42}`)

	if status != http.StatusCreated || !strings.Contains(body, `"id":"row-100"`) {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if len(db.CreatedTransactions) != 1 || db.CreatedTransactions[0].Payee != "Coffee" {
		t.Fatalf("created = %+v", db.CreatedTransactions)
	}
}

func TestWritesRequireToken(t *testing.T) {
	db := &fakes.Database{}
	app := newSheetsApp(t, db)
	request := httptest.NewRequest(http.MethodPost, "/api/sync", nil)

	response, err := app.Test(request, -1)

	if err != nil || response.StatusCode != http.StatusUnauthorized || db.Syncs != 0 {
		t.Fatalf("err = %v, status = %d, syncs = %d", err, response.StatusCode, db.Syncs)
	}
}

func TestDomainErrorsMapToStatusCodes(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{fmt.Errorf("%w: payee is required", domain.ErrValidation), http.StatusBadRequest},
		{fmt.Errorf("%w: row-9", domain.ErrNotFound), http.StatusNotFound},
		{fmt.Errorf("%w: row changed", domain.ErrConflict), http.StatusConflict},
		{errors.New("sheets api down"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.err.Error(), func(t *testing.T) {
			app := newSheetsApp(t, &fakes.Database{Err: tc.err})

			status, body := call(t, app, http.MethodDelete, "/api/transactions/row-9", "", "If-Match", `"abc"`)

			if status != tc.want || !strings.Contains(body, `"error"`) {
				t.Fatalf("status = %d, want %d, body = %s", status, tc.want, body)
			}
		})
	}
}

func TestTransactionETagSources(t *testing.T) {
	db := &fakes.Database{}
	app := newSheetsApp(t, db)
	update := `{"accountName":"Cash","date":"2026-03-31","payee":"Tea","category":"Food","payment":1}`

	if status, body := call(t, app, http.MethodPatch, "/api/transactions/row-5", update, "If-Match", "tag-1"); status != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", status, body)
	}
	if status, _ := call(t, app, http.MethodDelete, "/api/transactions/row-6?etag=tag-2", ""); status != http.StatusOK {
		t.Fatalf("delete status = %d", status)
	}

	if db.UpdatedTransactions["row-5"].ETag != "tag-1" || db.DeletedTransactions["row-6"] != "tag-2" {
		t.Fatalf("updated = %+v, deleted = %+v", db.UpdatedTransactions, db.DeletedTransactions)
	}
}

func TestAccountRoutesDecodeName(t *testing.T) {
	db := &fakes.Database{}
	app := newSheetsApp(t, db)

	if status, body := call(t, app, http.MethodPatch, "/api/accounts/VCB%20Saving", `{"note":"rainy day"}`); status != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", status, body)
	}
	if status, _ := call(t, app, http.MethodDelete, "/api/accounts/VCB%20Saving", ""); status != http.StatusOK {
		t.Fatalf("delete status = %d", status)
	}

	if _, ok := db.UpdatedAccounts["VCB Saving"]; !ok || len(db.DeletedAccounts) != 1 || db.DeletedAccounts[0] != "VCB Saving" {
		t.Fatalf("updated = %+v, deleted = %v", db.UpdatedAccounts, db.DeletedAccounts)
	}
}

func TestGetBudgetYearParameter(t *testing.T) {
	db := &fakes.Database{}
	app := newSheetsApp(t, db)

	if status, _ := call(t, app, http.MethodGet, "/api/budget?year=2025", ""); status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}
	if status, _ := call(t, app, http.MethodGet, "/api/budget?year=abc", ""); status != http.StatusBadRequest {
		t.Fatalf("invalid year status = %d", status)
	}
	if len(db.BudgetYears) != 1 || db.BudgetYears[0] != 2025 {
		t.Fatalf("years = %v", db.BudgetYears)
	}
}

func TestInitReportsReadiness(t *testing.T) {
	app := newSheetsApp(t, &fakes.Database{Init: domain.InitResult{Ready: true}})
	if status, body := call(t, app, http.MethodGet, "/api/init", ""); status != http.StatusOK || !strings.Contains(body, `"ready":true`) {
		t.Fatalf("status = %d, body = %s", status, body)
	}

	failing := newSheetsApp(t, &fakes.Database{Err: errors.New("boom")})
	if status, body := call(t, failing, http.MethodGet, "/api/init", ""); status != http.StatusInternalServerError || !strings.Contains(body, `"ready":false`) {
		t.Fatalf("status = %d, body = %s", status, body)
	}
}

func TestCaptureRulesRoutes(t *testing.T) {
	db := &fakes.Database{CaptureRules: []domain.CaptureRule{{Kind: domain.CaptureRuleAlias, Pattern: "VCB", Value: "Vietcombank"}}}
	app := newSheetsApp(t, db)
	if status, body := call(t, app, http.MethodGet, "/api/capture/rules", ""); status != http.StatusOK || !strings.Contains(body, `"pattern":"VCB"`) {
		t.Fatalf("list: %d %s", status, body)
	}
	status, body := call(t, app, http.MethodPost, "/api/capture/rules", `{"rules":[{"kind":"payee","pattern":"Grab","value":"Transport"}]}`)
	if status != http.StatusOK || body != `{"created":1,"updated":0,"unchanged":0}` || len(db.SavedCaptureRules) != 1 || db.SavedCaptureRules[0][0].Value != "Transport" {
		t.Fatalf("save: %d %s", status, body)
	}
	if status, _ := call(t, app, http.MethodPost, "/api/capture/rules", `{"rules":[]}`, "Authorization", ""); status != http.StatusUnauthorized {
		t.Fatalf("saving rules must be guarded, got %d", status)
	}
}
