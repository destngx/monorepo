package fiber

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/internal/fakes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestGivenRecurringItemsWhenGettingBillsThenItemsCalendarAndSafeToSpendAreReturned(t *testing.T) {
	amount := 8_000_000.0
	db := &fakes.Database{
		Accounts: []domain.Account{{Name: "Bank", Type: domain.AccountTypeBank, Balance: 20_000_000}},
		Recurring: domain.RecurringSheet{Exists: true, Items: []domain.RecurringItem{
			{ID: "recurring-2", Name: "Rent", Kind: domain.RecurringBill, Amount: &amount, Frequency: domain.FrequencyMonthly, Day: 20, Match: "Landlord", Status: domain.RecurringActive},
		}},
	}
	app := newAnalyticsApp(t, db)

	status, body := call(t, app, http.MethodGet, "/api/bills", "")
	var bills domain.Bills
	if status != http.StatusOK || json.Unmarshal([]byte(body), &bills) != nil {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if len(bills.Items) != 1 || len(bills.Calendar.Occurrences) != 1 || bills.SafeToSpend.SafeToSpend != 12_000_000 || bills.Suggestions == nil {
		t.Fatalf("bills = %+v", bills)
	}

	status, body = call(t, app, http.MethodGet, "/api/bills/safe-to-spend", "")
	var summary domain.BillsSummary
	if status != http.StatusOK || json.Unmarshal([]byte(body), &summary) != nil || summary.Facts.UpcomingOut != 8_000_000 || len(summary.DueSoon) != 0 || summary.SafeToSpend.Until != "2026-11-11" {
		t.Fatalf("summary: %d %s", status, body)
	}
}

func TestGivenSpendableCashWhenGettingIdleCashThenTheIdleAmountAndVerdictAreReturned(t *testing.T) {
	db := &fakes.Database{Accounts: []domain.Account{{Name: "Bank", Type: domain.AccountTypeBank, Balance: 20_000_000}}}
	app := newAnalyticsApp(t, db)

	status, body := call(t, app, http.MethodGet, "/api/bills/idle-cash", "")
	var idle domain.IdleCash
	if status != http.StatusOK || json.Unmarshal([]byte(body), &idle) != nil {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	// No rate source in this app: the idle amount still shows, with the issue.
	if idle.Idle != 20_000_000 || idle.Verdict != domain.IdleCashNoRate || idle.RatesIssue == "" || idle.Best != nil {
		t.Fatalf("idle = %+v", idle)
	}
}

func TestGivenRecurringWritesWhenCallingTheAPIThenTheyAreGuardedValidatedAndPassedOn(t *testing.T) {
	db := &fakes.Database{}
	app := newSheetsApp(t, db)

	if status, body := call(t, app, http.MethodPost, "/api/bills/items", `{"name":"Water","kind":"bill","amount":200000,"day":5,"match":"Water Co"}`); status != http.StatusCreated || len(db.CreatedRecurring) != 1 {
		t.Fatalf("create: %d %s", status, body)
	}
	if status, _ := call(t, app, http.MethodPost, "/api/bills/items", `{"name":"Water","kind":"loan","amount":1,"day":5,"match":"x"}`); status != http.StatusBadRequest {
		t.Fatalf("an unknown kind must be a 400, got %d", status)
	}
	if status, body := call(t, app, http.MethodPost, "/api/bills/suggestions/confirm", `{"key":"out|water co|utilities","name":"Water"}`); status != http.StatusCreated || len(db.ConfirmedRecurring) != 1 {
		t.Fatalf("confirm: %d %s", status, body)
	}
	if status, body := call(t, app, http.MethodPatch, "/api/bills/items/recurring-3", `{"name":"Water","kind":"bill","amount":250000,"day":5,"match":"Water Co"}`, "If-Match", `"abc"`); status != http.StatusOK || db.UpdatedRecurring["recurring-3"].ETag != "abc" {
		t.Fatalf("update: %d %s", status, body)
	}
	if status, body := call(t, app, http.MethodPost, "/api/bills/items/recurring-3/status", `{"status":"paused","etag":"abc"}`); status != http.StatusOK || db.RecurringStatuses["recurring-3"].Status != domain.RecurringPaused {
		t.Fatalf("pause: %d %s", status, body)
	}
	if status, body := call(t, app, http.MethodDelete, "/api/bills/items/recurring-3?etag=abc", ""); status != http.StatusOK || db.DeletedRecurring["recurring-3"] != "abc" {
		t.Fatalf("delete: %d %s", status, body)
	}
	if status, _ := call(t, app, http.MethodPost, "/api/bills/items", `{"name":"Water","kind":"bill","amount":1,"day":5,"match":"x"}`, "Authorization", ""); status != http.StatusUnauthorized {
		t.Fatalf("bill writes must be guarded, got %d", status)
	}
}

func TestGivenAReadinessChangeWhenCallingTheAPIThenItIsGuardedETagCheckedAndSaved(t *testing.T) {
	db := &fakes.Database{}
	app := newSheetsApp(t, db)
	etag := domain.ParsePreferences(nil, 2).SafeToSpend.ETag

	status, body := call(t, app, http.MethodPost, "/api/bills/safe-to-spend/accounts", `{"account":"Broker B","ready":false}`, "If-Match", `"`+etag+`"`)
	var prefs domain.Preferences
	if status != http.StatusOK || json.Unmarshal([]byte(body), &prefs) != nil || len(prefs.SafeToSpend.NotReadyAccounts) != 1 || len(db.ReadinessChanges) != 1 {
		t.Fatalf("mark: %d %s", status, body)
	}
	if status, body := call(t, app, http.MethodPost, "/api/bills/safe-to-spend/accounts", `{"account":"Broker B","ready":true,"etag":"`+etag+`"}`); status != http.StatusConflict {
		t.Fatalf("a stale etag must be a 409: %d %s", status, body)
	}
	if status, _ := call(t, app, http.MethodPost, "/api/bills/safe-to-spend/accounts", `{"account":"","ready":true,"etag":"x"}`); status != http.StatusBadRequest {
		t.Fatalf("a missing account must be a 400, got %d", status)
	}
	if status, _ := call(t, app, http.MethodPost, "/api/bills/safe-to-spend/accounts", `{"account":"Broker B","ready":true,"etag":"x"}`, "Authorization", ""); status != http.StatusUnauthorized {
		t.Fatalf("readiness writes must be guarded, got %d", status)
	}
}
