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
	defaultMaxRequest     = 4 << 20
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
	marketAnalysis   port.MarketAnalysisService
	portfolio        port.PortfolioService
	recommendations  port.RecommendationService
	monthlyClose     port.MonthlyCloseService
	log              *logger.Logger
	now              func() time.Time
	maxRequestBytes  int
}

func NewServer(healthService port.HealthService, log *logger.Logger) *Server {
	if healthService == nil {
		healthService = service.NewHealthService()
	}
	return &Server{healthService: healthService, log: log, now: time.Now, maxRequestBytes: defaultMaxRequest}
}

// SetMaxRequestBytes bounds one stdio request line; Smart Capture batches
// carry base64 images and need more than the default.
func (s *Server) SetMaxRequestBytes(limit int) {
	s.maxRequestBytes = max(limit, defaultMaxRequest)
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
	scanner.Buffer(make([]byte, 0, 64*1024), s.maxRequestBytes)
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
	if s.monthlyClose != nil {
		tools = append(tools, s.closeTools()...)
	}
	if s.marketService != nil {
		tools = append(tools, s.marketTools()...)
	}
	if s.marketAnalysis != nil {
		tools = append(tools, s.marketAnalysisTools()...)
	}
	if s.portfolio != nil {
		tools = append(tools, s.portfolioTools()...)
	}
	if s.recommendations != nil {
		tools = append(tools, s.recommendationTools()...)
	}
	if s.aiService != nil {
		tools = append(tools, s.aiTools()...)
		tools = append(tools, s.aiInvestmentTools()...)
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
