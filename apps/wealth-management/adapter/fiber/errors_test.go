package fiber

import (
	configAdapter "apps/wealth-management/adapter/config"
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/internal/fakes"
	"apps/wealth-management/service"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/oauth2"
	"google.golang.org/api/googleapi"
)

func decodeEnvelope(t *testing.T, body string) ErrorBody {
	t.Helper()
	var envelope ErrorBody
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		t.Fatalf("body is not an error envelope: %s", body)
	}
	if envelope.Error == "" || envelope.Code == "" {
		t.Fatalf("envelope misses error or code: %s", body)
	}
	return envelope
}

func TestGivenEachErrorKindWhenClassifiedThenStatusAndCodeMatchTheContract(t *testing.T) {
	expiredToken := &url.Error{Op: "Post", URL: "https://oauth2.googleapis.com/token", Err: &oauth2.RetrieveError{ErrorCode: "invalid_grant"}}
	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"validation", fmt.Errorf("%w: bad", domain.ErrValidation), http.StatusBadRequest, CodeValidation},
		{"not found", fmt.Errorf("%w: gone", domain.ErrNotFound), http.StatusNotFound, CodeNotFound},
		{"conflict", fmt.Errorf("%w: changed", domain.ErrConflict), http.StatusConflict, CodeConflict},
		{"sheets missing", fmt.Errorf("%w: [GOOGLE_SHEETS_ID]", configAdapter.ErrMissingSheetsConfig), http.StatusServiceUnavailable, CodeSheetsNotConfigured},
		{"oauth typed", fmt.Errorf("read accounts: %w", expiredToken), http.StatusBadGateway, CodeOAuthExpired},
		{"oauth text", errors.New(`oauth2: "invalid_grant" "Token has been expired or revoked."`), http.StatusBadGateway, CodeOAuthExpired},
		{"google api", fmt.Errorf("read: %w", &googleapi.Error{Code: http.StatusForbidden, Message: "denied"}), http.StatusBadGateway, CodeUpstream},
		{"ai output", fmt.Errorf("%w: not json", domain.ErrAIOutput), http.StatusBadGateway, CodeUpstream},
		{"ai missing", fmt.Errorf("%w: key", configAdapter.ErrMissingAIConfig), http.StatusServiceUnavailable, CodeUnavailable},
		{"unknown", errors.New("boom"), http.StatusInternalServerError, CodeInternal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, code := classifyError(tc.err)
			if status != tc.status || code != tc.code {
				t.Fatalf("got %d %s, want %d %s", status, code, tc.status, tc.code)
			}
		})
	}
}

func TestGivenAFailingSheetWhenReadingThenTheEnvelopeCarriesTheCode(t *testing.T) {
	app := newSheetsApp(t, &fakes.Database{Err: fmt.Errorf("%w: no row", domain.ErrNotFound)})

	status, body := call(t, app, http.MethodGet, "/api/accounts", "")

	if envelope := decodeEnvelope(t, body); status != http.StatusNotFound || envelope.Code != CodeNotFound {
		t.Fatalf("status = %d, body = %s", status, body)
	}
}

func TestGivenNoTokenWhenWritingThenUnauthorizedEnvelope(t *testing.T) {
	app := newSheetsApp(t, &fakes.Database{})

	status, body := call(t, app, http.MethodPost, "/api/sync", "", "Authorization", "")

	if envelope := decodeEnvelope(t, body); status != http.StatusUnauthorized || envelope.Code != CodeUnauthorized {
		t.Fatalf("status = %d, body = %s", status, body)
	}
}

func TestGivenNoTokenOnPublicHostWhenWritingThenForbiddenEnvelope(t *testing.T) {
	app := fiber.New()
	RegisterSheetRoutes(NewRouteRegistrar(app, "/api", ""), NewDatabaseHandler(&fakes.Database{}, logger.NewTestLogger(t)), NewWriteGuard(staticToken(""), "0.0.0.0"))

	status, body := call(t, app, http.MethodPost, "/api/sync", "")

	if envelope := decodeEnvelope(t, body); status != http.StatusForbidden || envelope.Code != CodeForbidden {
		t.Fatalf("status = %d, body = %s", status, body)
	}
}

func TestGivenAnUnknownRouteWhenCalledThenNotFoundEnvelope(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: FiberErrorHandler(logger.NewTestLogger(t))})

	status, body := call(t, app, http.MethodGet, "/api/nope", "")

	if envelope := decodeEnvelope(t, body); status != http.StatusNotFound || envelope.Code != CodeNotFound {
		t.Fatalf("status = %d, body = %s", status, body)
	}
}

func TestGivenSheetsNotConfiguredWhenCallingSheetRoutesThen503NamesTheMissingVars(t *testing.T) {
	app := fiber.New()
	missing := fmt.Errorf("%w: [GOOGLE_SHEETS_ID GOOGLE_CLIENT_ID]", configAdapter.ErrMissingSheetsConfig)
	RegisterUnconfiguredSheetRoutes(NewRouteRegistrar(app, "/api", ""), missing, logger.NewTestLogger(t))

	for _, route := range []struct{ method, path string }{
		{http.MethodGet, "/api/accounts"},
		{http.MethodPost, "/api/transactions"},
		{http.MethodPost, "/api/budget/reallocate"},
		{http.MethodGet, "/api/sheets/info"},
		{http.MethodGet, "/api/analytics/dashboard"},
		{http.MethodGet, "/api/analytics/budget/history"},
	} {
		status, body := call(t, app, route.method, route.path, "")
		envelope := decodeEnvelope(t, body)
		if status != http.StatusServiceUnavailable || envelope.Code != CodeSheetsNotConfigured || !strings.Contains(envelope.Error, "GOOGLE_SHEETS_ID") {
			t.Fatalf("%s %s: status = %d, body = %s", route.method, route.path, status, body)
		}
	}
}

func TestGivenAReallocationWhenPostedThenBothLimitsAreReturned(t *testing.T) {
	db := &fakes.Database{}
	app := newSheetsApp(t, db)

	status, body := call(t, app, http.MethodPost, "/api/budget/reallocate", `{"year":2026,"month":"2026-10","from":"Food","to":"Fun","amount":500000}`)

	if status != http.StatusOK || body != `{"from":{"category":"Food","limit":0},"to":{"category":"Fun","limit":500000}}` {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if len(db.Reallocations) != 1 || db.Reallocations[0].Amount != 500000 {
		t.Fatalf("reallocations = %+v", db.Reallocations)
	}
}

func TestGivenAnInvalidReallocationWhenPostedThenValidationEnvelope(t *testing.T) {
	app := newSheetsApp(t, &fakes.Database{})

	status, body := call(t, app, http.MethodPost, "/api/budget/reallocate", `{"month":"2026-10","from":"Food","to":"food","amount":0}`)

	if envelope := decodeEnvelope(t, body); status != http.StatusBadRequest || envelope.Code != CodeValidation {
		t.Fatalf("status = %d, body = %s", status, body)
	}
}

func TestGivenNoTokenWhenReallocatingThenNothingIsWritten(t *testing.T) {
	db := &fakes.Database{}
	app := newSheetsApp(t, db)

	status, _ := call(t, app, http.MethodPost, "/api/budget/reallocate", `{"month":"2026-10","from":"Food","to":"Fun","amount":1}`, "Authorization", "")

	if status != http.StatusUnauthorized || len(db.Reallocations) != 0 {
		t.Fatalf("status = %d, reallocations = %+v", status, db.Reallocations)
	}
}

func TestGivenASyncedSheetWhenReadingInfoThenIdTitleAndLastSyncAreReturned(t *testing.T) {
	synced := "2026-10-12T09:00:00Z"
	app := newSheetsApp(t, &fakes.Database{Info: domain.SheetInfo{SpreadsheetID: "sheet-1", Title: "Wealth", LastSyncAt: &synced}})

	status, body := call(t, app, http.MethodGet, "/api/sheets/info", "")

	if status != http.StatusOK || body != `{"spreadsheetId":"sheet-1","title":"Wealth","lastSyncAt":"2026-10-12T09:00:00Z"}` {
		t.Fatalf("status = %d, body = %s", status, body)
	}
}

func TestGivenANeverSyncedSheetWhenReadingInfoThenLastSyncIsNull(t *testing.T) {
	app := newSheetsApp(t, &fakes.Database{Info: domain.SheetInfo{SpreadsheetID: "sheet-1", Title: "Wealth"}})

	status, body := call(t, app, http.MethodGet, "/api/sheets/info", "")

	if status != http.StatusOK || !strings.Contains(body, `"lastSyncAt":null`) {
		t.Fatalf("status = %d, body = %s", status, body)
	}
}

func newAnalyticsApp(t *testing.T, db *fakes.Database) *fiber.App {
	t.Helper()
	app := fiber.New()
	now := func() time.Time { return time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC) }
	analytics := service.NewAnalyticsService(db, now)
	RegisterAnalyticsRoutes(NewRouteRegistrar(app, "/api", ""), NewAnalyticsHandler(analytics, logger.NewTestLogger(t)))
	return app
}

func TestGivenBudgetsWhenReadingHistoryThenSixMonthsOldestFirstByDefault(t *testing.T) {
	expense := domain.CategoryTypeExpense
	db := &fakes.Database{Budget: []domain.BudgetItem{{Category: "Food", CategoryType: &expense, MonthlyLimit: 100}}}
	app := newAnalyticsApp(t, db)

	status, body := call(t, app, http.MethodGet, "/api/analytics/budget/history?month=2026-10", "")

	var history domain.BudgetHistory
	if status != http.StatusOK || json.Unmarshal([]byte(body), &history) != nil {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if len(history.Months) != 6 || history.Months[0] != "2026-05" || history.Months[5] != "2026-10" || len(history.Rows) != 1 || history.Rows[0].Values[5].Limit != 100 {
		t.Fatalf("history = %+v", history)
	}
}

func TestGivenANonNumericMonthsWhenReadingHistoryThenValidationEnvelope(t *testing.T) {
	app := newAnalyticsApp(t, &fakes.Database{})

	status, body := call(t, app, http.MethodGet, "/api/analytics/budget/history?months=six", "")

	if envelope := decodeEnvelope(t, body); status != http.StatusBadRequest || envelope.Code != CodeValidation {
		t.Fatalf("status = %d, body = %s", status, body)
	}
}

func TestGivenAYearWhenReadingYearlyCashFlowThenTwelveMonths(t *testing.T) {
	income := domain.CategoryTypeIncome
	deposit := 500.0
	db := &fakes.Database{Transactions: []domain.Transaction{{AccountName: "Bank", Date: time.Date(2025, 3, 3, 0, 0, 0, 0, time.UTC), Category: "Salary", CategoryType: &income, Deposit: &deposit}}}
	app := newAnalyticsApp(t, db)

	status, body := call(t, app, http.MethodGet, "/api/analytics/cash-flow/yearly?year=2025", "")

	var report domain.YearlyCashFlow
	if status != http.StatusOK || json.Unmarshal([]byte(body), &report) != nil {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if report.Year != 2025 || len(report.Months) != 12 || report.Months[2].Income != 500 || len(report.Income) != 1 || report.Years[0] != 2026 {
		t.Fatalf("report = %+v", report)
	}
}

func TestGivenANonNumericYearWhenReadingYearlyCashFlowThenValidationEnvelope(t *testing.T) {
	app := newAnalyticsApp(t, &fakes.Database{})

	for _, query := range []string{"year=last", "year=1800"} {
		status, body := call(t, app, http.MethodGet, "/api/analytics/cash-flow/yearly?"+query, "")
		if envelope := decodeEnvelope(t, body); status != http.StatusBadRequest || envelope.Code != CodeValidation {
			t.Fatalf("%s: status = %d, body = %s", query, status, body)
		}
	}
}

func cardAdviceDatabase() *fakes.Database {
	return &fakes.Database{
		Cards: domain.CardConfig{
			Programs: []domain.Program{{Name: "Gold", Bank: "Acme", Versions: []domain.ProgramVersion{{
				From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Period: domain.CapPeriodCycle,
				Rules: []domain.CashbackRule{{Name: "Dining", MCCs: []domain.MCCRange{{From: 5812, To: 5812}}, Rate: 3}, {Name: "Others", Rate: 1, Fallback: true}},
			}}}},
			Banks: []domain.Bank{{Name: "Acme", SharedAccount: "Credit Card Acme", ReferenceLimit: 10_000_000}},
			Cards: []domain.Card{{Name: "Gold Card", Program: "Gold", Keyword: "gold", StatementDay: 5, DueDay: 30, Default: true}},
		},
		Accounts: []domain.Account{{Name: "Credit Card Acme", Type: domain.AccountTypeNegativeActive}},
	}
}

func TestGivenAPaymentWhenSuggestingCardsThenTheRankedCardsAreReturned(t *testing.T) {
	app := newAnalyticsApp(t, cardAdviceDatabase())

	status, body := call(t, app, http.MethodGet, "/api/analytics/cards/suggest?payee=New+place&amount=1000000&mcc=5812", "")

	var suggestions []domain.CardSuggestion
	if status != http.StatusOK || json.Unmarshal([]byte(body), &suggestions) != nil {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if len(suggestions) != 1 || suggestions[0].Rule != "Dining" || suggestions[0].Cashback.Confirmed != 30_000 {
		t.Fatalf("suggestions = %+v", suggestions)
	}
}

func TestGivenABadPaymentWhenSuggestingCardsThenValidationEnvelope(t *testing.T) {
	app := newAnalyticsApp(t, cardAdviceDatabase())
	for _, query := range []string{"payee=Shop&amount=lots", "payee=Shop&amount=1&mcc=dining", "payee=Shop", "amount=1"} {
		status, body := call(t, app, http.MethodGet, "/api/analytics/cards/suggest?"+query, "")
		if envelope := decodeEnvelope(t, body); status != http.StatusBadRequest || envelope.Code != CodeValidation {
			t.Fatalf("%s: status = %d, body = %s", query, status, body)
		}
	}
}

func TestGivenCardsWhenComparingProgrammesThenTheReportIsReturned(t *testing.T) {
	status, body := call(t, newAnalyticsApp(t, cardAdviceDatabase()), http.MethodGet, "/api/analytics/cards/compare", "")
	var report domain.ProgramComparisonReport
	if status != http.StatusOK || json.Unmarshal([]byte(body), &report) != nil || len(report.Programs) != 1 || report.To != "2026-10-12" {
		t.Fatalf("status = %d, body = %s", status, body)
	}
}
