package mcp

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"apps/wealth-management/service"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"
)

// ErrUnknownTool is returned by CallTool for a name that is not registered.
var ErrUnknownTool = errors.New("unknown tool")

const (
	jsonRPCParseError     = -32700
	jsonRPCInvalidParams  = -32602
	jsonRPCMethodNotFound = -32601
	maxRequestBytes       = 4 << 20
)

type toolHandler func(ctx context.Context, arguments json.RawMessage) (any, error)

type tool struct {
	definition domain.ToolDefinition
	handler    toolHandler
}

// Server exposes the HTTP API's capabilities as MCP tools over stdio. Writes
// go through the same domain validation as HTTP (WM-MCP-3), and tool failures
// are reported as isError results so the model can read and react to them.
type Server struct {
	healthService    port.HealthService
	databaseService  port.DatabaseService
	analyticsService port.AnalyticsService
	fmarketService   port.FmarketService
	marketService    port.MarketProviderService
	aiService        port.AIService
	log              *logger.Logger
	now              func() time.Time
}

func NewServer(healthService port.HealthService, log *logger.Logger) *Server {
	if healthService == nil {
		healthService = service.NewHealthService()
	}
	return &Server{healthService: healthService, log: log, now: time.Now}
}

func (s *Server) SetDatabaseService(databaseService port.DatabaseService) {
	s.databaseService = databaseService
}

func (s *Server) SetAnalyticsService(analyticsService port.AnalyticsService) {
	s.analyticsService = analyticsService
}

func (s *Server) SetFmarketService(fmarketService port.FmarketService) {
	s.fmarketService = fmarketService
}

func (s *Server) SetMarketService(marketService port.MarketProviderService) {
	s.marketService = marketService
}

func (s *Server) Run(r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), maxRequestBytes)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		response, shouldWrite := s.Handle(line)
		if !shouldWrite {
			continue
		}
		if _, err := fmt.Fprintln(w, string(response)); err != nil {
			return fmt.Errorf("write mcp response: %w", err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read mcp request: %w", err)
	}
	return nil
}

func (s *Server) Handle(raw []byte) ([]byte, bool) {
	var request jsonRPCRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		return mustMarshal(newErrorResponse(nil, jsonRPCParseError, "Parse error")), true
	}
	if request.ID == nil {
		// Notifications (e.g. notifications/initialized) get no response.
		return nil, false
	}
	ctx := context.Background()
	switch request.Method {
	case "initialize":
		return mustMarshal(newResultResponse(request.ID, map[string]any{
			"protocolVersion": "2024-11-05",
			"serverInfo":      map[string]any{"name": "wealth-management", "version": "0.2.0"},
			"capabilities":    map[string]any{"tools": map[string]any{}},
		})), true
	case "ping":
		return mustMarshal(newResultResponse(request.ID, map[string]any{})), true
	case "tools/list":
		return mustMarshal(newResultResponse(request.ID, map[string]any{"tools": toMCPTools(s.ListTools())})), true
	case "tools/call":
		var params struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(request.Params, &params); err != nil {
			return mustMarshal(newErrorResponse(request.ID, jsonRPCInvalidParams, "invalid tools/call params: "+err.Error())), true
		}
		result, err := s.CallTool(ctx, params.Name, params.Arguments)
		if errors.Is(err, ErrUnknownTool) {
			return mustMarshal(newErrorResponse(request.ID, jsonRPCInvalidParams, err.Error())), true
		}
		if err != nil {
			s.logToolFailure(ctx, params.Name, err)
			return mustMarshal(newResultResponse(request.ID, toolContent(err.Error(), true))), true
		}
		return mustMarshal(newResultResponse(request.ID, toolContent(result.Content, false))), true
	default:
		return mustMarshal(newErrorResponse(request.ID, jsonRPCMethodNotFound, "Method not found")), true
	}
}

// logToolFailure logs unexpected failures; validation, not-found and conflict
// errors are the caller's to fix and are only returned to them.
func (s *Server) logToolFailure(ctx context.Context, name string, err error) {
	if errors.Is(err, domain.ErrValidation) || errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrConflict) {
		return
	}
	s.log.LogError(ctx, "mcp: tool call failed", err, slog.String("tool_name", name))
}

func (s *Server) ListTools() []domain.ToolDefinition {
	tools := s.tools()
	definitions := make([]domain.ToolDefinition, 0, len(tools))
	for _, t := range tools {
		definitions = append(definitions, t.definition)
	}
	return definitions
}

func (s *Server) CallTool(ctx context.Context, name string, arguments json.RawMessage) (domain.ToolResult, error) {
	for _, t := range s.tools() {
		if t.definition.Name != name {
			continue
		}
		payload, err := t.handler(ctx, arguments)
		if err != nil {
			return domain.ToolResult{}, err
		}
		content, err := json.Marshal(payload)
		if err != nil {
			return domain.ToolResult{}, fmt.Errorf("encode %s result: %w", name, err)
		}
		return domain.ToolResult{Name: name, Backend: "mcp", Content: string(content)}, nil
	}
	return domain.ToolResult{}, fmt.Errorf("%w: %s", ErrUnknownTool, name)
}

// tools lists the tools available with the services configured right now.
func (s *Server) tools() []tool {
	tools := []tool{{
		definition: definition("EngineHealth", "Returns the overall status and per-dependency health (sheets, cache, market providers).", nil),
		handler: func(ctx context.Context, _ json.RawMessage) (any, error) {
			health := s.healthService.Check(ctx)
			return map[string]any{"status": health.Status, "checks": health.Checks, "timestamp": s.now().UTC().Format(time.RFC3339)}, nil
		},
	}}
	if s.databaseService != nil {
		tools = append(tools, s.databaseTools()...)
	}
	if s.analyticsService != nil {
		tools = append(tools, s.analyticsTools()...)
	}
	if s.marketService != nil {
		tools = append(tools, s.marketTools()...)
	}
	if s.aiService != nil {
		tools = append(tools, s.aiTools()...)
	}
	if s.fmarketService != nil {
		tools = append(tools, tool{
			definition: definition("RunFmarketAction", "Runs a read-only Fmarket action (fund lists, NAV history, gold, bank rates).", object(map[string]any{
				"action": map[string]any{"type": "string", "enum": domain.FmarketActions()},
				"params": map[string]any{"type": "object"},
				"force":  forceSchema,
			}, "action")),
			handler: func(ctx context.Context, arguments json.RawMessage) (any, error) {
				var args struct {
					Action string         `json:"action"`
					Params map[string]any `json:"params"`
					Force  bool           `json:"force"`
				}
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				return s.fmarketService.RunAction(ctx, domain.FmarketRequest{Action: domain.FmarketAction(args.Action), Params: args.Params}, args.Force)
			},
		})
	}
	return tools
}

func (s *Server) databaseTools() []tool {
	db := s.databaseService
	return []tool{
		readTool("GetAccounts", "Lists live accounts with type, balance, cleared balance and currency.", db.ListAccounts),
		{
			definition: definition("CreateAccount", "Creates an account. Currency defaults to USDT for Binance accounts, else VND.", object(map[string]any{
				"name": stringSchema, "type": accountTypeSchema, "balance": numberSchema, "currency": currencySchema, "note": stringSchema,
			}, "name", "type")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var input domain.AccountCreateInput
				if err := decodeValidated(arguments, &input); err != nil {
					return nil, err
				}
				if err := db.CreateAccount(input); err != nil {
					return nil, err
				}
				return map[string]any{"success": true}, nil
			},
		},
		{
			definition: definition("UpdateAccount", "Edits an account's type and/or note. Balances come from transactions and cannot be edited.", object(map[string]any{
				"name": stringSchema, "type": accountTypeSchema, "note": stringSchema,
			}, "name")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var args struct {
					Name string `json:"name"`
					domain.AccountUpdateInput
				}
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				if err := args.AccountUpdateInput.Validate(); err != nil {
					return nil, err
				}
				return db.UpdateAccount(args.Name, args.AccountUpdateInput)
			},
		},
		{
			definition: definition("DeleteAccount", "Soft-deletes an account (its name gets the [DELETED] prefix).", object(map[string]any{"name": stringSchema}, "name")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var args struct {
					Name string `json:"name"`
				}
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				if err := db.DeleteAccount(args.Name); err != nil {
					return nil, err
				}
				return map[string]any{"success": true}, nil
			},
		},
		readTool("GetTransactions", "Lists transactions newest first. Each has an id (row-N) and an etag needed to update or delete it.", db.ListTransactions),
		{
			definition: definition("CreateTransaction", "Appends a transaction and returns its id. Needs payment or deposit.", object(transactionSchema(), "accountName", "date", "payee", "category")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var input domain.TransactionCreateInput
				if err := decodeValidated(arguments, &input); err != nil {
					return nil, err
				}
				return db.CreateTransaction(input)
			},
		},
		{
			definition: definition("CreateTransactions", "Appends up to 100 transactions in one write. Retrying with the same batchId returns the first result without writing again.", object(map[string]any{
				"batchId":      stringSchema,
				"transactions": map[string]any{"type": "array", "maxItems": domain.MaxTransactionBatch, "items": object(transactionSchema(), "accountName", "date", "payee", "category")},
			}, "batchId", "transactions")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var input domain.TransactionBatchInput
				if err := decodeValidated(arguments, &input); err != nil {
					return nil, err
				}
				return db.CreateTransactions(input)
			},
		},
		{
			definition: definition("UpdateTransaction", "Replaces a transaction. Fails with a conflict when the row changed since its etag was read.", object(withProperties(transactionSchema(), map[string]any{
				"id": stringSchema, "etag": stringSchema,
			}), "id", "etag", "accountName", "date", "payee", "category")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var args struct {
					ID string `json:"id"`
					domain.TransactionUpdateInput
				}
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				if err := args.TransactionUpdateInput.Validate(); err != nil {
					return nil, err
				}
				return db.UpdateTransaction(args.ID, args.TransactionUpdateInput)
			},
		},
		{
			definition: definition("DeleteTransaction", "Clears a transaction row. Fails with a conflict when the row changed since its etag was read.", object(map[string]any{
				"id": stringSchema, "etag": stringSchema,
			}, "id", "etag")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var args struct {
					ID   string `json:"id"`
					ETag string `json:"etag"`
				}
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				if err := db.DeleteTransaction(args.ID, args.ETag); err != nil {
					return nil, err
				}
				return map[string]any{"success": true}, nil
			},
		},
		{
			definition: definition("GetBudget", "Reads a year's budget limits per category (defaults to the current year).", object(map[string]any{"year": yearSchema, "force": forceSchema})),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var args struct {
					Year  int  `json:"year"`
					Force bool `json:"force"`
				}
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				return db.ListBudget(args.Year, args.Force)
			},
		},
		{
			definition: definition("UpdateBudgetLimit", "Sets a category's monthly limit (month = YYYY-MM) or, without month, its yearly limit.", object(map[string]any{
				"year": yearSchema, "category": stringSchema, "month": map[string]any{"type": "string", "pattern": `^\d{4}-\d{2}$`}, "limit": numberSchema,
			}, "category", "limit")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var input domain.BudgetLimitUpdate
				if err := decodeValidated(arguments, &input); err != nil {
					return nil, err
				}
				if err := db.UpdateBudgetLimit(input); err != nil {
					return nil, err
				}
				return map[string]any{"success": true}, nil
			},
		},
		readTool("GetCategories", "Lists income, expense and non-budget categories.", db.ListCategories),
		readTool("GetGoals", "Lists savings goals with progress and status.", db.ListGoals),
		readTool("GetLoans", "Lists loans and debts with progress.", db.ListLoans),
		readTool("GetTags", "Lists normalised transaction tags.", db.ListTags),
		readTool("GetInvestmentAssets", "Returns crypto and fund holdings with symbol, quantity, price and value.", db.GetInvestmentAssets),
		readTool("WarmAIContent", "Loads AI prompts and knowledge from the sheet and returns row counts.", db.WarmAIContent),
		{
			definition: definition("SyncSheetsCache", "Invalidates every sheet-derived cache key so the next reads come from the sheet.", nil),
			handler: func(context.Context, json.RawMessage) (any, error) {
				return db.SyncCache()
			},
		},
	}
}

func (s *Server) analyticsTools() []tool {
	analytics := s.analyticsService
	return []tool{
		readTool("GetDashboard", "Balance sheet, month-over-month change, 30-day net worth trend, cash flow history and top accounts.", analytics.DashboardSummary),
		{
			definition: definition("GetBudgetReport", "Budget actuals for a month: spent, remaining and status per category.", object(map[string]any{
				"month": map[string]any{"type": "string", "description": "YYYY-MM; defaults to the current month."}, "force": forceSchema,
			})),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var args struct {
					Month string `json:"month"`
					Force bool   `json:"force"`
				}
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				return analytics.BudgetReport(args.Month, args.Force)
			},
		},
		readTool("GetCardReports", "Credit card cashback, statement cycle, utilisation and history.", analytics.CardReports),
		readTool("GetFinancialHealth", "Financial health score with per-pillar formulas and recommendations.", analytics.FinancialHealth),
		{
			definition: definition("GetAccountHistories", "Back-calculated daily balances per account.", object(map[string]any{
				"days": map[string]any{"type": "integer", "minimum": 1, "maximum": 366, "default": 60}, "force": forceSchema,
			})),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				args := struct {
					Days  int  `json:"days"`
					Force bool `json:"force"`
				}{Days: 60}
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				return analytics.AccountHistories(args.Days, args.Force)
			},
		},
	}
}

func (s *Server) marketTools() []tool {
	market := s.marketService
	type marketArgs struct {
		Provider string `json:"provider"`
		Symbol   string `json:"symbol"`
		Type     string `json:"type"`
		From     string `json:"from"`
		To       string `json:"to"`
	}
	marketTool := func(name, description string, schema map[string]any, call func(context.Context, marketArgs) (any, error)) tool {
		return tool{
			definition: definition(name, description, schema),
			handler: func(ctx context.Context, arguments json.RawMessage) (any, error) {
				var args marketArgs
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				return call(ctx, args)
			},
		}
	}
	return []tool{
		marketTool("GetMarketProviderHealth", "Reports a market data provider's health (fmarket or vnstock).",
			object(map[string]any{"provider": map[string]any{"type": "string", "enum": []string{"fmarket", "vnstock"}}}, "provider"),
			func(ctx context.Context, args marketArgs) (any, error) { return market.Health(ctx, args.Provider) }),
		marketTool("GetMarketTicker", "Returns the latest price for a fund certificate or gold.",
			object(map[string]any{"symbol": stringSchema, "type": map[string]any{"type": "string", "enum": []string{string(domain.TickerTypeIFC), string(domain.TickerTypeGold)}}}, "symbol", "type"),
			func(ctx context.Context, args marketArgs) (any, error) {
				return market.GetTicker(ctx, args.Symbol, domain.TickerType(args.Type))
			}),
		marketTool("GetExchangeRate", "Returns an exchange rate, e.g. USD to VND.",
			object(map[string]any{"from": stringSchema, "to": stringSchema}, "from", "to"),
			func(ctx context.Context, args marketArgs) (any, error) {
				return market.GetExchangeRate(ctx, args.From, args.To)
			}),
		marketTool("GetPriceSeries", "Returns a historical price series.",
			object(map[string]any{"symbol": stringSchema, "type": map[string]any{"type": "string", "enum": []string{string(domain.SeriesTypeGoldUSD)}}}, "type"),
			func(ctx context.Context, args marketArgs) (any, error) {
				return market.GetPriceSeries(ctx, args.Symbol, domain.SeriesType(args.Type))
			}),
		marketTool("GetBankRates", "Returns current bank deposit interest rates.", nil,
			func(ctx context.Context, _ marketArgs) (any, error) { return market.GetBankInterestRate(ctx) }),
	}
}

// readTool wraps a cached list call whose only argument is force.
func readTool[T any](name, description string, read func(forceFresh bool) (T, error)) tool {
	return tool{
		definition: definition(name, description, object(map[string]any{"force": forceSchema})),
		handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
			var args struct {
				Force bool `json:"force"`
			}
			if err := decodeArguments(arguments, &args); err != nil {
				return nil, err
			}
			return read(args.Force)
		},
	}
}

type validator interface{ Validate() error }

func decodeValidated(arguments json.RawMessage, target validator) error {
	if err := decodeArguments(arguments, target); err != nil {
		return err
	}
	return target.Validate()
}

// decodeArguments strictly decodes tool arguments; unknown fields and wrong
// types are validation errors, matching the advertised input schema.
func decodeArguments(arguments json.RawMessage, target any) error {
	if len(bytes.TrimSpace(arguments)) == 0 || bytes.Equal(bytes.TrimSpace(arguments), []byte("null")) {
		arguments = json.RawMessage("{}")
	}
	decoder := json.NewDecoder(bytes.NewReader(arguments))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("%w: invalid arguments: %v", domain.ErrValidation, err)
	}
	return nil
}

var (
	stringSchema      = map[string]any{"type": "string"}
	numberSchema      = map[string]any{"type": "number", "minimum": 0}
	forceSchema       = map[string]any{"type": "boolean", "description": "Bypass the cache and read the sheet directly."}
	yearSchema        = map[string]any{"type": "integer", "minimum": 2000, "maximum": 2100}
	accountTypeSchema = map[string]any{"type": "string", "enum": []string{
		string(domain.AccountTypeActiveUse), string(domain.AccountTypeRarelyUse), string(domain.AccountTypeLongHolding),
		string(domain.AccountTypeDeprecated), string(domain.AccountTypeNegativeActive), string(domain.AccountTypeBank),
		string(domain.AccountTypeCrypto), string(domain.AccountTypeCash), string(domain.AccountTypeInvestment),
	}}
	currencySchema = map[string]any{"type": "string", "enum": []string{string(domain.CurrencyVND), string(domain.CurrencyUSD), string(domain.CurrencyUSDT)}}
)

func transactionSchema() map[string]any {
	return map[string]any{
		"accountName":     stringSchema,
		"date":            map[string]any{"type": "string", "description": "dd/mm/yyyy or yyyy-mm-dd"},
		"payee":           stringSchema,
		"category":        stringSchema,
		"tags":            map[string]any{"type": "array", "items": stringSchema},
		"cleared":         map[string]any{"type": "boolean"},
		"payment":         numberSchema,
		"deposit":         numberSchema,
		"memo":            stringSchema,
		"referenceNumber": stringSchema,
	}
}

func withProperties(base map[string]any, extra map[string]any) map[string]any {
	for key, value := range extra {
		base[key] = value
	}
	return base
}

func object(properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{"type": "object", "properties": properties, "additionalProperties": false}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func definition(name, description string, inputSchema map[string]any) domain.ToolDefinition {
	if inputSchema == nil {
		inputSchema = object(map[string]any{})
	}
	return domain.ToolDefinition{Name: name, Description: description, Backend: "mcp", InputSchema: inputSchema}
}

func toolContent(text string, isError bool) map[string]any {
	result := map[string]any{"content": []map[string]any{{"type": "text", "text": text}}}
	if isError {
		result["isError"] = true
	}
	return result
}

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   any    `json:"error,omitempty"`
}

func newResultResponse(id any, result any) jsonRPCResponse {
	return jsonRPCResponse{JSONRPC: "2.0", ID: id, Result: result}
}

func newErrorResponse(id any, code int, message string) jsonRPCResponse {
	return jsonRPCResponse{JSONRPC: "2.0", ID: id, Error: map[string]any{"code": code, "message": message}}
}

func mustMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		b, _ = json.Marshal(newErrorResponse(nil, -32603, "encode response: "+err.Error()))
	}
	return b
}

func toMCPTools(definitions []domain.ToolDefinition) []map[string]any {
	tools := make([]map[string]any, 0, len(definitions))
	for _, definition := range definitions {
		tools = append(tools, map[string]any{"name": definition.Name, "description": definition.Description, "inputSchema": definition.InputSchema})
	}
	return tools
}
