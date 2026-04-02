package mcp

import (
	"apps/wealth-management-engine/adapter/logger"
	"apps/wealth-management-engine/domain"
	"apps/wealth-management-engine/port"
	"apps/wealth-management-engine/service"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"time"
)

type Server struct {
	healthService   port.HealthService
	databaseService port.DatabaseService
	fmarketService  port.FmarketService
	marketService   port.MarketProviderService
	log             *logger.Logger
}

func NewServer(healthService port.HealthService, databaseService port.DatabaseService, fmarketService port.FmarketService, marketService port.MarketProviderService, log *logger.Logger) *Server {
	if healthService == nil {
		healthService = service.NewHealthService()
	}
	return &Server{healthService: healthService, databaseService: databaseService, fmarketService: fmarketService, marketService: marketService, log: log}
}

func (s *Server) SetDatabaseService(databaseService port.DatabaseService) {
	s.databaseService = databaseService
}

func (s *Server) SetFmarketService(fmarketService port.FmarketService) {
	s.fmarketService = fmarketService
}

func (s *Server) SetMarketService(marketService port.MarketProviderService) {
	s.marketService = marketService
}

func (s *Server) Run(r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		response, shouldWrite := s.Handle(line)
		if !shouldWrite {
			continue
		}
		if _, err := fmt.Fprintln(w, string(response)); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (s *Server) Handle(raw []byte) ([]byte, bool) {
	var request jsonRPCRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		payload, _ := json.Marshal(newErrorResponse(nil, -32700, "Parse error"))
		return payload, true
	}
	if request.ID == nil {
		return nil, false
	}
	switch request.Method {
	case "initialize":
		return mustMarshal(jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Result: map[string]any{
				"protocolVersion": "2024-11-05",
				"serverInfo":      map[string]any{"name": "wealth-management-engine", "version": "0.1.0"},
				"capabilities":    map[string]any{"tools": map[string]any{}},
			},
		}), true
	case "tools/list":
		tools, err := s.ListTools(context.Background())
		if err != nil {
			return mustMarshal(newErrorResponse(request.ID, -32603, err.Error())), true
		}
		return mustMarshal(jsonRPCResponse{JSONRPC: "2.0", ID: request.ID, Result: map[string]any{"tools": toMCPTools(tools)}}), true
	case "tools/call":
		var params struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		_ = json.Unmarshal(request.Params, &params)
		result, err := s.CallTool(context.Background(), params.Name, params.Arguments)
		if err != nil {
			s.log.LogError(context.Background(), "mcp: tool call failed", err,
				slog.String("tool_name", params.Name),
				slog.Int("arg_count", len(params.Arguments)),
			)
			return mustMarshal(newErrorResponse(request.ID, -32602, err.Error())), true
		}
		return mustMarshal(jsonRPCResponse{JSONRPC: "2.0", ID: request.ID, Result: map[string]any{"content": []map[string]any{{"type": "text", "text": result.Content}}}}), true
	default:
		return mustMarshal(newErrorResponse(request.ID, -32601, "Method not found")), true
	}
}

func (s *Server) ListTools(_ context.Context) ([]domain.ToolDefinition, error) {
	tools := []domain.ToolDefinition{engineHealthToolDefinition()}
	if s.databaseService != nil {
		tools = append(tools,
			newToolDefinition("GetAccounts", "Returns typed accounts from Google Sheets."),
			newToolDefinition("GetTransactions", "Returns transactions from Google Sheets."),
			newToolDefinition("CreateTransaction", "Creates a transaction row in Google Sheets.", map[string]any{
				"accountName":     jsonSchemaString(),
				"date":            jsonSchemaString(),
				"payee":           jsonSchemaString(),
				"category":        jsonSchemaString(),
				"tags":            map[string]any{"type": "array", "items": jsonSchemaString()},
				"cleared":         map[string]any{"type": "boolean"},
				"payment":         map[string]any{"type": "number"},
				"deposit":         map[string]any{"type": "number"},
				"memo":            jsonSchemaString(),
				"referenceNumber": jsonSchemaString(),
			}),
			newToolDefinition("GetBudget", "Returns budget items from Google Sheets."),
			newToolDefinition("GetCategories", "Returns categories from Google Sheets."),
			newToolDefinition("GetGoals", "Returns goals from Google Sheets."),
			newToolDefinition("GetLoans", "Returns loans from Google Sheets."),
			newToolDefinition("GetNotifications", "Returns pending notifications from Google Sheets."),
			newToolDefinition("MarkNotificationsDone", "Marks notification rows as done.", map[string]any{"rowNumbers": map[string]any{"type": "array", "items": map[string]any{"type": "integer"}}}),
			newToolDefinition("GetTags", "Returns normalized transaction tags."),
			newToolDefinition("SyncSheetsCache", "Invalidates Sheets-related cache keys."),
			newToolDefinition("WarmAIContent", "Warms AI prompts and knowledge content from Google Sheets."),
			newToolDefinition("GetInvestmentAssets", "Returns investment asset holdings from Google Sheets."),
		)
	}
	if s.marketService != nil {
		tools = append(tools,
			newToolDefinition("GetFmarketHealth", "Returns Fmarket provider health status."),
			newToolDefinition("GetFmarketTicker", "Returns an explicit Fmarket ticker.", map[string]any{"symbol": jsonSchemaString(), "type": map[string]any{"type": "string", "enum": []string{"ifc", "gold"}}}),
			newToolDefinition("GetFmarketExchangeRate", "Returns USD/VND exchange rate from Fmarket.", map[string]any{"from": jsonSchemaString(), "to": jsonSchemaString()}),
			newToolDefinition("GetFmarketPriceSeries", "Returns explicit Fmarket price series.", map[string]any{"symbol": jsonSchemaString(), "type": map[string]any{"type": "string", "enum": []string{"gold_usd"}}}),
			newToolDefinition("GetFmarketBankRates", "Returns explicit Fmarket bank rates."),
		)
	}
	if s.fmarketService != nil {
		tools = append(tools,
			newToolDefinition("RunFmarketAction", "Runs a legacy-style Fmarket action.", map[string]any{"action": jsonSchemaString(), "params": map[string]any{"type": "object"}, "force": map[string]any{"type": "boolean"}}),
		)
	}
	return tools, nil
}

func (s *Server) CallTool(_ context.Context, name string, arguments map[string]any) (domain.ToolResult, error) {
	switch name {
	case "EngineHealth":
		health := s.healthService.Check()
		resultJSON, _ := json.Marshal(map[string]any{"status": health.Status, "timestamp": time.Now().UTC().Format(time.RFC3339)})
		return domain.ToolResult{Name: name, Backend: "mcp", Content: string(resultJSON)}, nil
	}
	if s.databaseService == nil {
		switch name {
		case "GetFmarketHealth", "GetFmarketTicker", "GetFmarketExchangeRate", "GetFmarketPriceSeries", "GetFmarketBankRates", "RunFmarketAction":
		default:
			return domain.ToolResult{}, fmt.Errorf("database service unavailable")
		}
	}
	var payload any
	var err error
	switch name {
	case "GetAccounts":
		payload, err = s.databaseService.ListAccounts()
	case "GetTransactions":
		payload, err = s.databaseService.ListTransactions(argumentBool(arguments, "force"))
	case "CreateTransaction":
		input := domain.TransactionCreateInput{
			AccountName:     argumentString(arguments, "accountName"),
			Date:            argumentString(arguments, "date"),
			ReferenceNumber: argumentOptionalString(arguments, "referenceNumber"),
			Payee:           argumentString(arguments, "payee"),
			Tags:            argumentStringSlice(arguments, "tags"),
			Memo:            argumentOptionalString(arguments, "memo"),
			Category:        argumentString(arguments, "category"),
			Cleared:         argumentBool(arguments, "cleared"),
			Payment:         argumentOptionalFloat(arguments, "payment"),
			Deposit:         argumentOptionalFloat(arguments, "deposit"),
		}
		err = s.databaseService.CreateTransaction(input)
		payload = map[string]any{"success": err == nil}
	case "GetBudget":
		payload, err = s.databaseService.ListBudget(argumentBool(arguments, "force"))
	case "GetCategories":
		payload, err = s.databaseService.ListCategories(argumentBool(arguments, "force"))
	case "GetGoals":
		payload, err = s.databaseService.ListGoals(argumentBool(arguments, "force"))
	case "GetLoans":
		payload, err = s.databaseService.ListLoans(argumentBool(arguments, "force"))
	case "GetNotifications":
		payload, err = s.databaseService.ListPendingNotifications(argumentBool(arguments, "force"))
	case "MarkNotificationsDone":
		rowNumbers := argumentIntSlice(arguments, "rowNumbers")
		for _, rowNumber := range rowNumbers {
			if err = s.databaseService.MarkNotificationDone(rowNumber); err != nil {
				break
			}
		}
		payload = map[string]any{"success": err == nil, "rowNumbers": rowNumbers}
	case "GetTags":
		payload, err = s.databaseService.ListTags(argumentBool(arguments, "force"))
	case "SyncSheetsCache":
		payload, err = s.databaseService.SyncCache()
	case "WarmAIContent":
		payload, err = s.databaseService.WarmAIContent(argumentBool(arguments, "force"))
	case "GetInvestmentAssets":
		payload, err = s.databaseService.GetInvestmentAssets(argumentBool(arguments, "force"))
	case "GetFmarketHealth":
		if s.marketService == nil {
			return domain.ToolResult{}, fmt.Errorf("market service unavailable")
		}
		payload, err = s.marketService.Health(context.Background(), "fmarket")
	case "GetFmarketTicker":
		if s.marketService == nil {
			return domain.ToolResult{}, fmt.Errorf("market service unavailable")
		}
		payload, err = s.marketService.GetTicker(context.Background(), argumentString(arguments, "symbol"), domain.TickerType(argumentString(arguments, "type")))
	case "GetFmarketExchangeRate":
		if s.marketService == nil {
			return domain.ToolResult{}, fmt.Errorf("market service unavailable")
		}
		payload, err = s.marketService.GetExchangeRate(context.Background(), argumentString(arguments, "from"), argumentString(arguments, "to"))
	case "GetFmarketPriceSeries":
		if s.marketService == nil {
			return domain.ToolResult{}, fmt.Errorf("market service unavailable")
		}
		payload, err = s.marketService.GetPriceSeries(context.Background(), argumentString(arguments, "symbol"), domain.SeriesType(argumentString(arguments, "type")))
	case "GetFmarketBankRates":
		if s.marketService == nil {
			return domain.ToolResult{}, fmt.Errorf("market service unavailable")
		}
		payload, err = s.marketService.GetBankInterestRate(context.Background())
	case "RunFmarketAction":
		if s.fmarketService == nil {
			return domain.ToolResult{}, fmt.Errorf("fmarket service unavailable")
		}
		params, _ := arguments["params"].(map[string]any)
		payload, err = s.fmarketService.RunAction(context.Background(), domain.FmarketRequest{Action: domain.FmarketAction(argumentString(arguments, "action")), Params: params}, argumentBool(arguments, "force"))
	default:
		return domain.ToolResult{}, fmt.Errorf("unknown tool: %s", name)
	}
	if err != nil {
		return domain.ToolResult{}, err
	}
	resultJSON, _ := json.Marshal(payload)
	return domain.ToolResult{Name: name, Backend: "mcp", Content: string(resultJSON)}, nil
}

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   any    `json:"error,omitempty"`
}

func newErrorResponse(id any, code int, message string) jsonRPCResponse {
	return jsonRPCResponse{JSONRPC: "2.0", ID: id, Error: map[string]any{"code": code, "message": message}}
}

func mustMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func engineHealthToolDefinition() domain.ToolDefinition {
	return newToolDefinition("EngineHealth", "Returns engine health status.")
}

func newToolDefinition(name string, description string, properties ...map[string]any) domain.ToolDefinition {
	inputSchema := map[string]any{"type": "object", "properties": map[string]any{}, "additionalProperties": false}
	if len(properties) > 0 {
		inputSchema["properties"] = properties[0]
	}
	return domain.ToolDefinition{Name: name, Description: description, Backend: "mcp", InputSchema: inputSchema}
}

func jsonSchemaString() map[string]any {
	return map[string]any{"type": "string"}
}

func toMCPTools(definitions []domain.ToolDefinition) []map[string]any {
	tools := make([]map[string]any, 0, len(definitions))
	for _, definition := range definitions {
		tools = append(tools, map[string]any{"name": definition.Name, "description": definition.Description, "inputSchema": definition.InputSchema})
	}
	return tools
}

func argumentString(arguments map[string]any, key string) string {
	if value, ok := arguments[key]; ok {
		return fmt.Sprintf("%v", value)
	}
	return ""
}

func argumentOptionalString(arguments map[string]any, key string) *string {
	value := argumentString(arguments, key)
	if value == "" {
		return nil
	}
	return &value
}

func argumentBool(arguments map[string]any, key string) bool {
	value, ok := arguments[key]
	if !ok {
		return false
	}
	boolValue, ok := value.(bool)
	return ok && boolValue
}

func argumentOptionalFloat(arguments map[string]any, key string) *float64 {
	value, ok := arguments[key]
	if !ok {
		return nil
	}
	switch typed := value.(type) {
	case float64:
		return &typed
	case float32:
		converted := float64(typed)
		return &converted
	case int:
		converted := float64(typed)
		return &converted
	case int64:
		converted := float64(typed)
		return &converted
	default:
		return nil
	}
}

func argumentStringSlice(arguments map[string]any, key string) []string {
	value, ok := arguments[key]
	if !ok {
		return []string{}
	}
	rawValues, ok := value.([]any)
	if !ok {
		return []string{}
	}
	result := make([]string, 0, len(rawValues))
	for _, item := range rawValues {
		result = append(result, fmt.Sprintf("%v", item))
	}
	return result
}

func argumentIntSlice(arguments map[string]any, key string) []int {
	value, ok := arguments[key]
	if !ok {
		return []int{}
	}
	rawValues, ok := value.([]any)
	if !ok {
		return []int{}
	}
	result := make([]int, 0, len(rawValues))
	for _, item := range rawValues {
		switch typed := item.(type) {
		case float64:
			result = append(result, int(typed))
		case int:
			result = append(result, typed)
		}
	}
	return result
}
