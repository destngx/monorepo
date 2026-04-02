package mcp

import (
	"apps/wealth-management-engine/adapter/logger"
	"apps/wealth-management-engine/domain"
	"apps/wealth-management-engine/port"
	"context"
	"encoding/json"
	"testing"
)

func TestGivenInitializeRequestWhenHandleThenReturnsCapabilities(t *testing.T) {
	server := NewServer(nil, nil, nil, nil, logger.NewTestLogger(t))
	responseRaw, ok := server.Handle([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`))
	if !ok {
		t.Fatalf("expected response")
	}
	var response map[string]any
	if err := json.Unmarshal(responseRaw, &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["error"] != nil {
		t.Fatalf("expected no error, got %v", response["error"])
	}
}

func TestGivenToolsListWhenHandleThenReturnsDatabaseToolsWhenConfigured(t *testing.T) {
	server := NewServer(nil, &fakeDatabaseServiceForMCP{}, &fakeFmarketServiceForMCP{}, &fakeMarketServiceForMCP{}, logger.NewTestLogger(t))
	responseRaw, ok := server.Handle([]byte(`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`))
	if !ok {
		t.Fatalf("expected response")
	}
	var response struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(responseRaw, &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if len(response.Result.Tools) < 10 {
		t.Fatalf("expected expanded MCP tool list, got %+v", response.Result.Tools)
	}
}

func TestGivenToolsCallEngineHealthWhenHandleThenReturnsHealthPayload(t *testing.T) {
	server := NewServer(nil, nil, nil, nil, logger.NewTestLogger(t))
	responseRaw, ok := server.Handle([]byte(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"EngineHealth","arguments":{}}}`))
	if !ok {
		t.Fatalf("expected response")
	}
	var response map[string]any
	if err := json.Unmarshal(responseRaw, &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["error"] != nil {
		t.Fatalf("expected no error, got %v", response["error"])
	}
}

func TestGivenToolsCallGetAccountsWhenHandleThenReturnsDatabasePayload(t *testing.T) {
	server := NewServer(nil, &fakeDatabaseServiceForMCP{}, nil, nil, logger.NewTestLogger(t))
	responseRaw, ok := server.Handle([]byte(`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"GetAccounts","arguments":{}}}`))
	if !ok {
		t.Fatalf("expected response")
	}
	var response map[string]any
	if err := json.Unmarshal(responseRaw, &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["error"] != nil {
		t.Fatalf("expected no error, got %v", response["error"])
	}
}

func TestGivenToolsCallRunFmarketActionWhenHandleThenReturnsFmarketPayload(t *testing.T) {
	server := NewServer(nil, nil, &fakeFmarketServiceForMCP{}, nil, logger.NewTestLogger(t))
	responseRaw, ok := server.Handle([]byte(`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"RunFmarketAction","arguments":{"action":"getProductsFilterNav","params":{"page":1}}}}`))
	if !ok {
		t.Fatalf("expected response")
	}
	var response map[string]any
	if err := json.Unmarshal(responseRaw, &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["error"] != nil {
		t.Fatalf("expected no error, got %v", response["error"])
	}
	result, ok := response["result"].(map[string]any)
	if !ok || result["content"] == nil {
		t.Fatalf("expected MCP content payload, got %+v", response)
	}
}

func TestGivenToolsCallGetFmarketHealthWhenHandleThenReturnsMarketHealth(t *testing.T) {
	server := NewServer(nil, nil, nil, &fakeMarketServiceForMCP{}, logger.NewTestLogger(t))
	responseRaw, ok := server.Handle([]byte(`{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"GetFmarketHealth","arguments":{}}}`))
	if !ok {
		t.Fatalf("expected response")
	}
	var response map[string]any
	if err := json.Unmarshal(responseRaw, &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["error"] != nil {
		t.Fatalf("expected no error, got %v", response["error"])
	}
}

type fakeDatabaseServiceForMCP struct{}
type fakeFmarketServiceForMCP struct{}
type fakeMarketServiceForMCP struct{}

func (f *fakeDatabaseServiceForMCP) ReadAccounts() (domain.Accounts, error) {
	return domain.Accounts{}, nil
}
func (f *fakeDatabaseServiceForMCP) ListAccounts() ([]domain.Account, error) {
	return []domain.Account{{Name: "Cash", Balance: 1000, ClearedBalance: 1000, Type: domain.AccountTypeCash, Currency: domain.CurrencyVND}}, nil
}
func (f *fakeDatabaseServiceForMCP) ListTransactions(bool) ([]domain.Transaction, error) {
	return []domain.Transaction{}, nil
}
func (f *fakeDatabaseServiceForMCP) CreateTransaction(domain.TransactionCreateInput) error {
	return nil
}
func (f *fakeDatabaseServiceForMCP) ListCategories(bool) ([]domain.Category, error) {
	return []domain.Category{{Name: "Salary", Type: domain.CategoryTypeIncome}}, nil
}
func (f *fakeDatabaseServiceForMCP) ListBudget(bool) ([]domain.BudgetItem, error) {
	return []domain.BudgetItem{}, nil
}
func (f *fakeDatabaseServiceForMCP) ListGoals(bool) ([]domain.Goal, error) {
	return []domain.Goal{}, nil
}
func (f *fakeDatabaseServiceForMCP) ListLoans(bool) ([]domain.Loan, error) {
	return []domain.Loan{}, nil
}
func (f *fakeDatabaseServiceForMCP) ListNotifications(bool) ([]domain.EmailNotification, error) {
	return []domain.EmailNotification{}, nil
}
func (f *fakeDatabaseServiceForMCP) ListPendingNotifications(bool) ([]domain.EmailNotification, error) {
	return []domain.EmailNotification{}, nil
}
func (f *fakeDatabaseServiceForMCP) MarkNotificationDone(int) error  { return nil }
func (f *fakeDatabaseServiceForMCP) ListTags(bool) ([]string, error) { return []string{"salary"}, nil }
func (f *fakeDatabaseServiceForMCP) GetInvestmentAssets(bool) (domain.InvestmentAssets, error) {
	return domain.InvestmentAssets{}, nil
}
func (f *fakeDatabaseServiceForMCP) WarmAIContent(bool) (domain.InitResult, error) {
	return domain.InitResult{Ready: true}, nil
}
func (f *fakeDatabaseServiceForMCP) SyncCache() (domain.SyncResult, error) {
	return domain.SyncResult{Success: true}, nil
}

func (f *fakeFmarketServiceForMCP) RunAction(context.Context, domain.FmarketRequest, bool) (any, error) {
	return map[string]any{"data": map[string]any{"rows": []any{}}}, nil
}

func (f *fakeMarketServiceForMCP) Health(context.Context, string) (domain.MarketProviderHealth, error) {
	return domain.MarketProviderHealth{Provider: "fmarket", Status: "ok"}, nil
}
func (f *fakeMarketServiceForMCP) GetTicker(context.Context, string, domain.TickerType) (domain.Ticker, error) {
	return domain.Ticker{Symbol: "VFMVF1", Type: domain.TickerTypeIFC, Provider: "fmarket"}, nil
}
func (f *fakeMarketServiceForMCP) GetExchangeRate(context.Context, string, string) (domain.ExchangeRate, error) {
	return domain.ExchangeRate{From: "USD", To: "VND", Provider: "fmarket", Rate: 25000}, nil
}
func (f *fakeMarketServiceForMCP) GetPriceSeries(context.Context, string, domain.SeriesType) (domain.PriceSeries, error) {
	return domain.PriceSeries{Provider: "fmarket", SeriesType: domain.SeriesTypeGoldUSD}, nil
}
func (f *fakeMarketServiceForMCP) GetBankInterestRate(context.Context) ([]domain.BankRate, error) {
	return []domain.BankRate{{Bank: "Test Bank", Rate: 6.5}}, nil
}

var _ port.DatabaseService = (*fakeDatabaseServiceForMCP)(nil)
var _ port.FmarketService = (*fakeFmarketServiceForMCP)(nil)
var _ port.MarketProviderService = (*fakeMarketServiceForMCP)(nil)
