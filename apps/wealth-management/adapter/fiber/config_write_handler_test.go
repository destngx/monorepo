package fiber

import (
	"apps/wealth-management/internal/fakes"
	"net/http"
	"testing"
)

func TestGivenConfigWritesWhenCallingTheAPIThenTheyAreValidatedAndPassedOn(t *testing.T) {
	db := &fakes.Database{}
	app := newSheetsApp(t, db)

	steps := []struct {
		method, path, body string
		want               int
	}{
		{http.MethodPost, "/api/investments/funds/trades", `{"code":"funda","date":"2026-01-02","units":10}`, http.StatusCreated},
		{http.MethodPost, "/api/investments/funds/trades", `{"code":"funda","date":"2026-01-02","units":0}`, http.StatusBadRequest},
		{http.MethodPost, "/api/investments/crypto/holdings", `{"holdings":[{"platform":"Exchange A","symbol":"btc","quantity":0.5}]}`, http.StatusOK},
		{http.MethodPost, "/api/investments/crypto/holdings", `{"holdings":[]}`, http.StatusBadRequest},
		{http.MethodPost, "/api/cards", `{"name":"Gold Card","program":"Gold","statementDay":5,"dueDay":25}`, http.StatusCreated},
		{http.MethodPatch, "/api/cards/Gold%20Card", `{"name":"Gold Card","program":"Gold","statementDay":6}`, http.StatusOK},
		{http.MethodPost, "/api/cards", `{"name":"","program":"Gold"}`, http.StatusBadRequest},
		{http.MethodPost, "/api/cards/programs", `{"program":"Gold","bank":"Acme","from":"2027-01-01","capPeriod":"cycle","rules":[{"rule":"Others","rate":1,"fallback":true}]}`, http.StatusCreated},
		{http.MethodPost, "/api/cards/merchants", `{"pattern":"Bakery","mcc":742}`, http.StatusOK},
		{http.MethodPost, "/api/cards/merchants", `{"pattern":"Bakery","mcc":99999}`, http.StatusBadRequest},
	}
	for _, step := range steps {
		if status, body := call(t, app, step.method, step.path, step.body); status != step.want {
			t.Fatalf("%s %s: %d %s", step.method, step.path, status, body)
		}
	}
	if len(db.FundTrades) != 1 || db.FundTrades[0].Code != "FUNDA" || len(db.CryptoWrites) != 1 || len(db.CreatedCards) != 1 ||
		db.UpdatedCards["Gold Card"].StatementDay != 6 || len(db.CreatedPrograms) != 1 || len(db.SavedMerchants) != 1 {
		t.Fatalf("db: %+v", db)
	}
}
