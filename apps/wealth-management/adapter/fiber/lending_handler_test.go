package fiber

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/internal/fakes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestGivenLendingTransactionsWhenGettingLendingThenTheLedgerIsReturned(t *testing.T) {
	lent := 2_000_000.0
	nonBudget := domain.CategoryTypeNonBudget
	db := &fakes.Database{Transactions: []domain.Transaction{{
		ID: "row-2", AccountName: "Bank", Date: time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC), Payee: "Person A",
		Category: "Lending / Repayment", CategoryType: &nonBudget, CategoryNature: domain.NatureLending, Payment: &lent,
	}}}
	app := newAnalyticsApp(t, db)

	status, body := call(t, app, http.MethodGet, "/api/lending", "")

	var ledger domain.MoneyLent
	if status != http.StatusOK || json.Unmarshal([]byte(body), &ledger) != nil {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if ledger.OwedToYou != 2_000_000 || len(ledger.People) != 1 || *ledger.People[0].DaysOutstanding != 10 {
		t.Fatalf("ledger = %+v", ledger)
	}
	if !strings.Contains(body, `"categoryNature":"lending"`) {
		t.Fatalf("transactions should carry their nature: %s", body)
	}
}

func TestGivenACategoryWhenPatchingItsNatureThenTheUpdateIsGuardedAndValidated(t *testing.T) {
	db := &fakes.Database{}
	app := newSheetsApp(t, db)

	status, body := call(t, app, http.MethodPatch, "/api/categories/nature", `{"category":"Lending / Repayment","nature":"lending"}`)
	if status != http.StatusOK || !strings.Contains(body, `"nature":"lending"`) || len(db.NatureUpdates) != 1 {
		t.Fatalf("patch: %d %s", status, body)
	}
	if status, _ := call(t, app, http.MethodPatch, "/api/categories/nature", `{"category":"Food","nature":"loan"}`); status != http.StatusBadRequest {
		t.Fatalf("an unknown nature must be a 400, got %d", status)
	}
	if status, _ := call(t, app, http.MethodPatch, "/api/categories/nature", `{"category":"Food","nature":"fixed"}`, "Authorization", ""); status != http.StatusUnauthorized {
		t.Fatalf("setting a nature must be guarded, got %d", status)
	}
}

func TestGivenACategoryWhenPatchingItsDefinitionThenTheUpdateIsGuardedAndValidated(t *testing.T) {
	db := &fakes.Database{}
	app := newSheetsApp(t, db)

	status, body := call(t, app, http.MethodPatch, "/api/categories/definition", `{"category":"Food","definition":"Food for home.","splitTags":["eating-out"]}`)
	if status != http.StatusOK || !strings.Contains(body, `"definition":"Food for home."`) || len(db.DefinitionUpdates) != 1 {
		t.Fatalf("patch: %d %s", status, body)
	}
	if status, _ := call(t, app, http.MethodPatch, "/api/categories/definition", `{"category":"Food","splitTags":["a,b"]}`); status != http.StatusBadRequest {
		t.Fatalf("a split tag with a comma must be a 400, got %d", status)
	}
	if status, _ := call(t, app, http.MethodPatch, "/api/categories/definition", `{"category":"Food"}`, "Authorization", ""); status != http.StatusUnauthorized {
		t.Fatalf("setting a definition must be guarded, got %d", status)
	}
}
