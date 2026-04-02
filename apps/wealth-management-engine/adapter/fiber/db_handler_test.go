package fiber

import (
	"apps/wealth-management-engine/adapter/logger"
	"apps/wealth-management-engine/domain"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestGivenAccountsRequestWhenDatabaseHandlerCalledThenReturnsTypedAccounts(t *testing.T) {
	app := fiber.New()
	testLog := logger.NewTestLogger(t)
	handler := NewDatabaseHandler(&fakeDatabaseServiceForHandler{
		accounts: []domain.Account{{Name: "Cash", Balance: 1000, ClearedBalance: 1000, Type: domain.AccountTypeCash, Currency: domain.CurrencyVND}},
	}, testLog)
	app.Get("/api/accounts", handler.GetAccounts)
	request := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("fiber test request failed: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}
	var payload []domain.Account
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload) != 1 || payload[0].Name != "Cash" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestGivenTransactionsCreateRequestWhenDatabaseHandlerCalledThenReturnsSuccess(t *testing.T) {
	app := fiber.New()
	fake := &fakeDatabaseServiceForHandler{}
	testLog := logger.NewTestLogger(t)
	handler := NewDatabaseHandler(fake, testLog)
	app.Post("/api/transactions", handler.CreateTransaction)
	request := httptest.NewRequest(http.MethodPost, "/api/transactions", strings.NewReader(`{"accountName":"Cash","date":"2026-03-31","payee":"Coffee","category":"Food","tags":["daily"],"cleared":true,"payment":42}`))
	request.Header.Set("Content-Type", "application/json")
	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("fiber test request failed: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}
	if fake.createdTransaction == nil || fake.createdTransaction.Payee != "Coffee" {
		t.Fatalf("expected transaction to be forwarded, got %+v", fake.createdTransaction)
	}
}

func TestGivenNotificationsPatchWhenDatabaseHandlerCalledThenMarksRowsDone(t *testing.T) {
	app := fiber.New()
	fake := &fakeDatabaseServiceForHandler{}
	testLog := logger.NewTestLogger(t)
	handler := NewDatabaseHandler(fake, testLog)
	app.Patch("/api/notifications", handler.MarkNotificationsDone)
	request := httptest.NewRequest(http.MethodPatch, "/api/notifications", strings.NewReader(`{"rowNumbers":[3,4]}`))
	request.Header.Set("Content-Type", "application/json")
	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("fiber test request failed: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}
	if len(fake.markedRows) != 2 || fake.markedRows[0] != 3 || fake.markedRows[1] != 4 {
		t.Fatalf("expected marked rows [3 4], got %+v", fake.markedRows)
	}
}

func TestGivenInitRequestWhenDatabaseHandlerCalledThenReturnsReadyPayload(t *testing.T) {
	app := fiber.New()
	testLog := logger.NewTestLogger(t)
	handler := NewDatabaseHandler(&fakeDatabaseServiceForHandler{initResult: domain.InitResult{Ready: true}}, testLog)
	app.Get("/api/init", handler.Init)
	request := httptest.NewRequest(http.MethodGet, "/api/init", nil)
	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("fiber test request failed: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}
	body, _ := io.ReadAll(response.Body)
	if !strings.Contains(string(body), `"ready":true`) {
		t.Fatalf("expected ready payload, got %s", string(body))
	}
}

type fakeDatabaseServiceForHandler struct {
	accounts           []domain.Account
	createdTransaction *domain.TransactionCreateInput
	markedRows         []int
	initResult         domain.InitResult
}

func (f *fakeDatabaseServiceForHandler) ReadAccounts() (domain.Accounts, error) {
	return domain.Accounts{Range: "Accounts!A2:I", Rows: [][]string{{"Cash"}}}, nil
}
func (f *fakeDatabaseServiceForHandler) ListAccounts() ([]domain.Account, error) {
	return f.accounts, nil
}
func (f *fakeDatabaseServiceForHandler) ListTransactions(bool) ([]domain.Transaction, error) {
	return []domain.Transaction{}, nil
}
func (f *fakeDatabaseServiceForHandler) CreateTransaction(input domain.TransactionCreateInput) error {
	f.createdTransaction = &input
	return nil
}
func (f *fakeDatabaseServiceForHandler) ListCategories(bool) ([]domain.Category, error) {
	return []domain.Category{}, nil
}
func (f *fakeDatabaseServiceForHandler) ListBudget(bool) ([]domain.BudgetItem, error) {
	return []domain.BudgetItem{}, nil
}
func (f *fakeDatabaseServiceForHandler) ListGoals(bool) ([]domain.Goal, error) {
	return []domain.Goal{}, nil
}
func (f *fakeDatabaseServiceForHandler) ListLoans(bool) ([]domain.Loan, error) {
	return []domain.Loan{}, nil
}
func (f *fakeDatabaseServiceForHandler) ListNotifications(bool) ([]domain.EmailNotification, error) {
	return []domain.EmailNotification{}, nil
}
func (f *fakeDatabaseServiceForHandler) ListPendingNotifications(bool) ([]domain.EmailNotification, error) {
	return []domain.EmailNotification{}, nil
}
func (f *fakeDatabaseServiceForHandler) MarkNotificationDone(rowNumber int) error {
	f.markedRows = append(f.markedRows, rowNumber)
	return nil
}
func (f *fakeDatabaseServiceForHandler) ListTags(bool) ([]string, error) {
	return []string{"salary"}, nil
}
func (f *fakeDatabaseServiceForHandler) GetInvestmentAssets(bool) (domain.InvestmentAssets, error) {
	return domain.InvestmentAssets{}, nil
}
func (f *fakeDatabaseServiceForHandler) WarmAIContent(bool) (domain.InitResult, error) {
	return f.initResult, nil
}
func (f *fakeDatabaseServiceForHandler) SyncCache() (domain.SyncResult, error) {
	return domain.SyncResult{Success: true}, nil
}

var _ context.Context
