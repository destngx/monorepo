package mcp

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/internal/fakes"
	"apps/wealth-management/port"
	"apps/wealth-management/service"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

type rpcResponse struct {
	ID     any `json:"id"`
	Result struct {
		Tools []struct {
			Name        string         `json:"name"`
			InputSchema map[string]any `json:"inputSchema"`
		} `json:"tools"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	} `json:"result"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func newTestServer(t *testing.T, db port.DatabaseService) *Server {
	t.Helper()
	server := NewServer(nil, logger.NewTestLogger(t))
	if db != nil {
		server.SetDatabaseService(db)
	}
	return server
}

func handle(t *testing.T, server *Server, request string) rpcResponse {
	t.Helper()
	raw, ok := server.Handle([]byte(request))
	if !ok {
		t.Fatalf("expected a response to %s", request)
	}
	var response rpcResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		t.Fatalf("invalid json response %s: %v", raw, err)
	}
	return response
}

func callTool(t *testing.T, server *Server, name string, arguments string) rpcResponse {
	t.Helper()
	return handle(t, server, fmt.Sprintf(`{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":%q,"arguments":%s}}`, name, arguments))
}

func TestInitializeReturnsCapabilities(t *testing.T) {
	response := handle(t, newTestServer(t, nil), `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	if response.Error != nil {
		t.Fatalf("error = %+v", response.Error)
	}
}

func TestNotificationsGetNoResponse(t *testing.T) {
	if _, ok := newTestServer(t, nil).Handle([]byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)); ok {
		t.Fatal("notifications must not be answered")
	}
}

func TestToolsListDependsOnConfiguredServices(t *testing.T) {
	bare := handle(t, newTestServer(t, nil), `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	if len(bare.Result.Tools) != 1 || bare.Result.Tools[0].Name != "EngineHealth" {
		t.Fatalf("bare tools = %+v", bare.Result.Tools)
	}

	server := newTestServer(t, &fakes.Database{})
	server.SetFmarketService(&fakeFmarketServiceForMCP{})
	server.SetMarketService(&fakeMarketServiceForMCP{})
	response := handle(t, server, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)

	names := map[string]map[string]any{}
	for _, tool := range response.Result.Tools {
		names[tool.Name] = tool.InputSchema
	}
	for _, want := range []string{"CreateAccount", "UpdateAccount", "DeleteAccount", "CreateTransactions", "UpdateTransaction", "DeleteTransaction", "UpdateBudgetLimit", "GetMarketTicker", "RunFmarketAction"} {
		if _, ok := names[want]; !ok {
			t.Errorf("missing tool %s", want)
		}
	}
	if props := names["GetTransactions"]["properties"].(map[string]any); props["force"] == nil {
		t.Errorf("GetTransactions must accept force, schema = %+v", names["GetTransactions"])
	}
	if props := names["GetBudget"]["properties"].(map[string]any); props["year"] == nil {
		t.Errorf("GetBudget must accept year, schema = %+v", names["GetBudget"])
	}
}

func TestEngineHealthReportsChecks(t *testing.T) {
	response := callTool(t, newTestServer(t, nil), "EngineHealth", `{}`)
	if response.Error != nil || response.Result.IsError || !strings.Contains(response.Result.Content[0].Text, `"status":"ok"`) {
		t.Fatalf("response = %+v", response)
	}
}

func TestGetAccountsPassesForce(t *testing.T) {
	db := &fakes.Database{Accounts: []domain.Account{{Name: "Cash"}}}
	response := callTool(t, newTestServer(t, db), "GetAccounts", `{"force":true}`)
	if response.Result.IsError || !strings.Contains(response.Result.Content[0].Text, `"Cash"`) {
		t.Fatalf("response = %+v", response)
	}
}

func TestCreateTransactionValidatesLikeHTTP(t *testing.T) {
	db := &fakes.Database{}
	response := callTool(t, newTestServer(t, db), "CreateTransaction", `{"accountName":"Cash","date":"31/02/2026","payee":"Tea","category":"Food"}`)

	if !response.Result.IsError || response.Error != nil {
		t.Fatalf("expected an isError result, got %+v", response)
	}
	text := response.Result.Content[0].Text
	if !strings.Contains(text, "date") || !strings.Contains(text, "payment or deposit is required") {
		t.Fatalf("text = %s", text)
	}
	if len(db.CreatedTransactions) != 0 {
		t.Fatalf("invalid input must not reach the service: %+v", db.CreatedTransactions)
	}
}

func TestCreateTransactionReturnsID(t *testing.T) {
	db := &fakes.Database{}
	response := callTool(t, newTestServer(t, db), "CreateTransaction", `{"accountName":"Cash","date":"2026-03-31","payee":"Tea","category":"Food","payment":20000,"tags":["daily"]}`)
	if response.Result.IsError || !strings.Contains(response.Result.Content[0].Text, `"id":"row-100"`) {
		t.Fatalf("response = %+v", response)
	}
	if len(db.CreatedTransactions) != 1 || *db.CreatedTransactions[0].Payment != 20000 || db.CreatedTransactions[0].Tags[0] != "daily" {
		t.Fatalf("created = %+v", db.CreatedTransactions)
	}
}

func TestUnknownArgumentsAreRejected(t *testing.T) {
	response := callTool(t, newTestServer(t, &fakes.Database{}), "GetAccounts", `{"forse":true}`)
	if !response.Result.IsError || !strings.Contains(response.Result.Content[0].Text, "forse") {
		t.Fatalf("response = %+v", response)
	}
}

func TestUpdateAndDeleteTransactionForwardETag(t *testing.T) {
	db := &fakes.Database{}
	server := newTestServer(t, db)

	update := callTool(t, server, "UpdateTransaction", `{"id":"row-5","etag":"abc","accountName":"Cash","date":"2026-03-31","payee":"Tea","category":"Food","payment":1}`)
	remove := callTool(t, server, "DeleteTransaction", `{"id":"row-6","etag":"def"}`)

	if update.Result.IsError || remove.Result.IsError {
		t.Fatalf("update = %+v, delete = %+v", update, remove)
	}
	if db.UpdatedTransactions["row-5"].ETag != "abc" || db.DeletedTransactions["row-6"] != "def" {
		t.Fatalf("updated = %+v, deleted = %+v", db.UpdatedTransactions, db.DeletedTransactions)
	}
}

func TestServiceErrorsBecomeIsErrorResults(t *testing.T) {
	db := &fakes.Database{Err: fmt.Errorf("%w: row changed", domain.ErrConflict)}
	response := callTool(t, newTestServer(t, db), "DeleteTransaction", `{"id":"row-6","etag":"def"}`)
	if response.Error != nil || !response.Result.IsError || !strings.Contains(response.Result.Content[0].Text, "conflict") {
		t.Fatalf("response = %+v", response)
	}
}

func TestUnknownToolIsProtocolError(t *testing.T) {
	response := callTool(t, newTestServer(t, nil), "GetAccounts", `{}`)
	if response.Error == nil || response.Error.Code != jsonRPCInvalidParams {
		t.Fatalf("response = %+v", response)
	}
	if _, err := newTestServer(t, nil).CallTool(context.Background(), "Nope", nil); !errors.Is(err, ErrUnknownTool) {
		t.Fatalf("err = %v", err)
	}
}

func TestRunFmarketActionReturnsPayload(t *testing.T) {
	server := newTestServer(t, nil)
	server.SetFmarketService(&fakeFmarketServiceForMCP{})
	response := callTool(t, server, "RunFmarketAction", `{"action":"getProductsFilterNav","params":{"page":1}}`)
	if response.Result.IsError || len(response.Result.Content) != 1 {
		t.Fatalf("response = %+v", response)
	}
}

func TestMarketProviderHealthTool(t *testing.T) {
	server := newTestServer(t, nil)
	server.SetMarketService(&fakeMarketServiceForMCP{})
	response := callTool(t, server, "GetMarketProviderHealth", `{"provider":"fmarket"}`)
	if response.Result.IsError || !strings.Contains(response.Result.Content[0].Text, `"ok"`) {
		t.Fatalf("response = %+v", response)
	}
}

type fakeFmarketServiceForMCP struct{}
type fakeMarketServiceForMCP struct{}

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
func (f *fakeMarketServiceForMCP) GetBankInterestRate(context.Context) (domain.BankRates, error) {
	return domain.BankRates{Rates: []domain.BankRate{{Bank: "Test Bank", Rate: 6.5}}}, nil
}

var _ port.FmarketService = (*fakeFmarketServiceForMCP)(nil)
var _ port.MarketProviderService = (*fakeMarketServiceForMCP)(nil)

func TestAIToolsDelegateToTheService(t *testing.T) {
	server := newTestServer(t, nil)
	ai := &fakes.AI{}
	server.SetAIService(ai)

	result, err := server.CallTool(context.Background(), "ReviewFinances", json.RawMessage(`{"kind":"account-review","force":true}`))
	if err != nil || len(ai.Reviews) != 1 || ai.Reviews[0].Kind != port.ReviewAccounts || !ai.Reviews[0].Force || !strings.Contains(result.Content, `"summary":"ok"`) {
		t.Fatalf("review: %v %+v %s", err, ai.Reviews, result.Content)
	}
	result, err = server.CallTool(context.Background(), "Chat", json.RawMessage(`{"messages":[{"role":"user","content":"hi"}]}`))
	if err != nil || result.Content != `{"model":"m","reply":"Hello","tools":["getBudgetStatus"]}` {
		t.Fatalf("chat: %v %s", err, result.Content)
	}
	if _, err := server.CallTool(context.Background(), "ReviewFinances", json.RawMessage(`{"kind":"account-review","bogus":1}`)); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("unknown arguments should be rejected, got %v", err)
	}
}

func TestCaptureToolSuggestsWithoutWriting(t *testing.T) {
	db := &fakes.Database{}
	server := newTestServer(t, db)
	ai := &fakes.AI{}
	server.SetAIService(ai)

	result, err := server.CallTool(context.Background(), "SuggestTransactionsFromMaterials", json.RawMessage(`{"materials":[{"id":"a","kind":"image","mediaType":"image/png","data":"iVBORw=="},{"id":"b","kind":"text","text":"VCB -45,000VND"}]}`))
	if err != nil || !strings.Contains(result.Content, `"suggestions"`) {
		t.Fatalf("capture: %v %s", err, result.Content)
	}
	if len(ai.Captures) != 1 || string(ai.Captures[0].Materials[0].Data) != "\x89PNG" || ai.Captures[0].Materials[1].Text != "VCB -45,000VND" {
		t.Fatalf("materials not decoded: %+v", ai.Captures)
	}
	if len(db.CreatedTransactions) != 0 {
		t.Fatal("capture must not write")
	}
	if _, err := server.CallTool(context.Background(), "SuggestTransactionsFromMaterials", json.RawMessage(`{"materials":[]}`)); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("empty batch should be rejected, got %v", err)
	}
}

func TestCardProgramsTool(t *testing.T) {
	db := &fakes.Database{Cards: domain.CardConfig{Cards: []domain.Card{{Name: "Gold Card", Program: "Gold", Keyword: "gold"}}}}
	result, err := newTestServer(t, db).CallTool(context.Background(), "GetCardPrograms", nil)
	if err != nil || !strings.Contains(result.Content, `"keyword":"gold"`) {
		t.Fatalf("list: %v %s", err, result.Content)
	}
}

func TestCaptureRuleTools(t *testing.T) {
	db := &fakes.Database{CaptureRules: []domain.CaptureRule{{Kind: domain.CaptureRuleAlias, Pattern: "VCB", Value: "Vietcombank"}}}
	server := newTestServer(t, db)
	result, err := server.CallTool(context.Background(), "GetCaptureRules", nil)
	if err != nil || !strings.Contains(result.Content, `"pattern":"VCB"`) {
		t.Fatalf("list: %v %s", err, result.Content)
	}
	if _, err := server.CallTool(context.Background(), "SaveCaptureRules", json.RawMessage(`{"rules":[{"kind":"payee","pattern":"Grab","value":"Transport"}]}`)); err != nil {
		t.Fatal(err)
	}
	if len(db.SavedCaptureRules) != 1 || db.SavedCaptureRules[0][0].Value != "Transport" {
		t.Fatalf("saved: %+v", db.SavedCaptureRules)
	}
}

func TestRunAcceptsRequestsUpToTheConfiguredLimit(t *testing.T) {
	server := newTestServer(t, nil)
	server.SetMaxRequestBytes(8 << 20)
	padding := strings.Repeat("x", 5<<20)
	request := `{"jsonrpc":"2.0","id":1,"method":"ping","params":{"pad":"` + padding + `"}}` + "\n"
	var out strings.Builder
	if err := server.Run(strings.NewReader(request), &out); err != nil || !strings.Contains(out.String(), `"id":1`) {
		t.Fatalf("large request: %v %q", err, out.String()[:min(len(out.String()), 200)])
	}
}

func (f *fakeMarketServiceForMCP) GetCandles(context.Context, string, domain.Timeframe, int) (domain.CandleSeries, error) {
	return domain.CandleSeries{}, nil
}

func TestCardAdviceTools(t *testing.T) {
	db := &fakes.Database{Cards: domain.CardConfig{
		Programs: []domain.Program{{Name: "Gold", Bank: "Acme", Versions: []domain.ProgramVersion{{
			Period: domain.CapPeriodCycle, Rules: []domain.CashbackRule{{Name: "Others", Rate: 1, Fallback: true}},
		}}}},
		Banks: []domain.Bank{{Name: "Acme", SharedAccount: "Credit Card Acme"}},
		Cards: []domain.Card{{Name: "Gold Card", Program: "Gold", Keyword: "gold", StatementDay: 5, DueDay: 30, Default: true}},
	}}
	server := newTestServer(t, db)
	server.SetAnalyticsService(service.NewAnalyticsService(db, func() time.Time { return time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC) }))
	result, err := server.CallTool(context.Background(), "SuggestCardForPayment", json.RawMessage(`{"payee":"Shop","amount":1000000}`))
	if err != nil || !strings.Contains(result.Content, `"card":"Gold Card"`) {
		t.Fatalf("suggest: %v %s", err, result.Content)
	}
	if _, err := server.CallTool(context.Background(), "SuggestCardForPayment", json.RawMessage(`{"payee":"Shop"}`)); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("a payment without an amount should be rejected, got %v", err)
	}
	result, err = server.CallTool(context.Background(), "CompareCardPrograms", nil)
	if err != nil || !strings.Contains(result.Content, `"program":"Gold"`) {
		t.Fatalf("compare: %v %s", err, result.Content)
	}
}

func TestMoneyLentTools(t *testing.T) {
	lent := 3_000_000.0
	nonBudget := domain.CategoryTypeNonBudget
	db := &fakes.Database{Transactions: []domain.Transaction{{
		ID: "row-2", AccountName: "Bank", Date: time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC), Payee: "Person A",
		Category: "Lending / Repayment", CategoryType: &nonBudget, CategoryNature: domain.NatureLending, Payment: &lent,
	}}}
	server := newTestServer(t, db)
	server.SetAnalyticsService(service.NewAnalyticsService(db, func() time.Time { return time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC) }))

	result, err := server.CallTool(context.Background(), "GetMoneyLent", nil)
	if err != nil || !strings.Contains(result.Content, `"owedToYou":3000000`) {
		t.Fatalf("money lent: %v %s", err, result.Content)
	}
	result, err = server.CallTool(context.Background(), "SetCategoryNature", json.RawMessage(`{"category":"Lending / Repayment","nature":"lending"}`))
	if err != nil || len(db.NatureUpdates) != 1 || !strings.Contains(result.Content, `"nature":"lending"`) {
		t.Fatalf("set nature: %v %s", err, result.Content)
	}
	if _, err := server.CallTool(context.Background(), "SetCategoryNature", json.RawMessage(`{"category":"Food","nature":"loan"}`)); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("an unknown nature should be rejected, got %v", err)
	}
}

func TestPreferenceTools(t *testing.T) {
	db := &fakes.Database{}
	server := newTestServer(t, db)
	result, err := server.CallTool(context.Background(), "GetPreferences", nil)
	if err != nil || !strings.Contains(result.Content, `"emergencyMonths":6`) {
		t.Fatalf("list: %v %s", err, result.Content)
	}
	if _, err := server.CallTool(context.Background(), "SavePreferences", json.RawMessage(`{"values":[{"key":"goal.emergencyMonths","value":4}]}`)); err != nil {
		t.Fatal(err)
	}
	if len(db.SavedPreferences) != 1 || db.Preferences.GoalPool.EmergencyMonths != 4 {
		t.Fatalf("saved: %+v", db.SavedPreferences)
	}
	if _, err := server.CallTool(context.Background(), "SavePreferences", json.RawMessage(`{"values":[{"key":"allocation.target.cash","value":40}]}`)); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("targets not adding up must be a validation error: %v", err)
	}
}
