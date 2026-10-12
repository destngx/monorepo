package ai

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// chatMode frames the assistant for conversation; engine code like the directive.
const chatMode = `## CHAT MODE
You are chatting with the owner of this ledger. Answer in concise markdown. Use the tools to fetch figures you do not have in the snapshot; never guess ledger numbers.`

const (
	defaultToolTransactions = 20
	maxToolTransactions     = 100
)

// Chat runs a tool-using conversation and streams events (WM-AI-11).
func (s *Service) Chat(ctx context.Context, request port.ChatRequest, emit func(port.ChatEvent) error) error {
	if err := request.Validate(); err != nil {
		return err
	}
	if err := s.requireConfigured(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, chatTimeout)
	defer cancel()

	snap, err := s.loadSnapshot()
	if err != nil {
		return err
	}
	p, err := s.loadPrompts()
	if err != nil {
		return err
	}
	task := chatMode
	if extra, ok := p.lookup(keyChatTask, nil); ok {
		task += "\n\n" + extra
	}
	if page := request.Page; page != nil && strings.TrimSpace(page.Path) != "" {
		task += fmt.Sprintf("\n\n## CURRENT PAGE\nThe user is looking at %s", page.Path)
		if page.Title != "" {
			task += " (" + s.masker.Mask(page.Title) + ")"
		}
		if page.Summary != "" {
			task += ":\n" + s.masker.Mask(page.Summary)
		}
	}
	system, err := s.systemPrompt(p, snap, task)
	if err != nil {
		return err
	}

	history := request.Messages
	if len(history) > maxChatHistory {
		history = history[len(history)-maxChatHistory:]
	}
	messages := []domain.AIMessage{{Role: domain.AIRoleSystem, Content: system}}
	for _, message := range history {
		messages = append(messages, domain.AIMessage{Role: domain.AIRole(message.Role), Content: message.Content})
	}
	tools := s.chatTools()
	definitions := make([]domain.AITool, 0, len(tools))
	for _, tool := range tools {
		definitions = append(definitions, tool.definition)
	}

	for step := 0; step < maxChatSteps; step++ {
		lastStep := step == maxChatSteps-1
		turn := domain.AIRequest{Messages: messages}
		if !lastStep {
			turn.Tools = definitions
		}
		response, err := s.streamStep(ctx, turn, p.used, request.Model, emit)
		if err != nil {
			return err
		}
		if len(response.ToolCalls) == 0 || lastStep {
			return emit(port.ChatEvent{Type: "done", Model: response.Model})
		}
		messages = append(messages, domain.AIMessage{Role: domain.AIRoleAssistant, Content: response.Content, ToolCalls: response.ToolCalls})
		for _, toolCall := range response.ToolCalls {
			if err := emit(port.ChatEvent{Type: "tool", Tool: toolCall.Name, Status: "running"}); err != nil {
				return err
			}
			result, toolErr := s.runTool(ctx, tools, toolCall)
			status := "ok"
			if toolErr != nil {
				status = "error"
				s.log.LogError(ctx, "ai chat tool failed", toolErr)
				result = toJSON(map[string]string{"error": toolMessage(toolErr)})
			}
			if err := emit(port.ChatEvent{Type: "tool", Tool: toolCall.Name, Status: status}); err != nil {
				return err
			}
			messages = append(messages, domain.AIMessage{Role: domain.AIRoleTool, ToolCallID: toolCall.ID, Content: result})
		}
	}
	return nil
}

// streamStep streams one model turn. The fallback model is tried only when
// the primary fails before any text reached the user.
func (s *Service) streamStep(ctx context.Context, request domain.AIRequest, promptKeys []string, override string, emit func(port.ChatEvent) error) (domain.AIResponse, error) {
	var lastErr error
	for _, model := range s.models(domain.AIUseCaseChat, override) {
		request.Model = model
		streamed := false
		var emitErr error
		attemptCtx, cancel := context.WithTimeout(ctx, s.config.Timeout)
		started := time.Now()
		response, err := s.client.Stream(attemptCtx, request, func(delta string) error {
			streamed = true
			emitErr = emit(port.ChatEvent{Type: "text", Delta: delta})
			return emitErr
		})
		cancel()
		s.logCall(ctx, "chat", model, promptKeys, time.Since(started), response.Usage, err)
		if emitErr != nil {
			return domain.AIResponse{}, emitErr
		}
		if err == nil {
			if response.Model == "" {
				response.Model = model
			}
			return response, nil
		}
		lastErr = err
		if streamed || ctx.Err() != nil {
			break
		}
	}
	return domain.AIResponse{}, lastErr
}

func toolMessage(err error) string {
	if IsUserFacing(err) {
		return err.Error()
	}
	return "the tool failed; tell the user the data is unavailable"
}

type chatTool struct {
	definition domain.AITool
	run        func(ctx context.Context, arguments json.RawMessage) (any, error)
}

func (s *Service) runTool(ctx context.Context, tools map[string]chatTool, toolCall domain.AIToolCall) (string, error) {
	tool, ok := tools[toolCall.Name]
	if !ok {
		return "", fmt.Errorf("%w: unknown tool %q", domain.ErrValidation, toolCall.Name)
	}
	arguments := json.RawMessage(strings.TrimSpace(toolCall.Arguments))
	if len(arguments) == 0 {
		arguments = json.RawMessage("{}")
	}
	result, err := tool.run(ctx, arguments)
	if err != nil {
		return "", err
	}
	return toJSON(result), nil
}

func decodeArguments(arguments json.RawMessage, target any) error {
	if err := json.Unmarshal(arguments, target); err != nil {
		return fmt.Errorf("%w: invalid tool arguments: %v", domain.ErrValidation, err)
	}
	return nil
}

func objectSchema(properties map[string]any) map[string]any {
	return map[string]any{"type": "object", "properties": properties}
}

// chatTools are the read-only ledger tools of the chat (WM-AI-11, WM-MCP-4).
func (s *Service) chatTools() map[string]chatTool {
	str := func(description string) map[string]any {
		return map[string]any{"type": "string", "description": description}
	}
	tools := []chatTool{
		{
			definition: domain.AITool{Name: "getAccountBalances", Description: "Current balance of every account plus assets, liabilities and net worth.", Parameters: objectSchema(map[string]any{})},
			run: func(ctx context.Context, _ json.RawMessage) (any, error) {
				accounts, err := s.db.ListAccounts(false)
				if err != nil {
					return nil, err
				}
				loans, err := s.db.ListLoans(false)
				if err != nil {
					return nil, err
				}
				return map[string]any{"accounts": s.accountContexts(accounts), "balanceSheet": domain.ComputeBalanceSheet(accounts, loans)}, nil
			},
		},
		{
			definition: domain.AITool{Name: "getRecentTransactions", Description: "Transactions newest first, optionally filtered.", Parameters: objectSchema(map[string]any{
				"limit":    map[string]any{"type": "integer", "description": fmt.Sprintf("Maximum rows, default %d, at most %d", defaultToolTransactions, maxToolTransactions)},
				"from":     str("Earliest date, YYYY-MM-DD"),
				"to":       str("Latest date, YYYY-MM-DD"),
				"category": str("Exact category name"),
				"payee":    str("Text contained in the payee"),
				"account":  str("Text contained in the account name"),
			})},
			run: s.transactionsTool,
		},
		{
			definition: domain.AITool{Name: "getBudgetStatus", Description: "Budget limits, spending and status per category for a month.", Parameters: objectSchema(map[string]any{"month": str("Month YYYY-MM; empty is the current month")})},
			run: func(ctx context.Context, arguments json.RawMessage) (any, error) {
				var input struct {
					Month string `json:"month"`
				}
				if err := decodeArguments(arguments, &input); err != nil {
					return nil, err
				}
				return s.analytics.BudgetReport(strings.TrimSpace(input.Month), false)
			},
		},
		{
			definition: domain.AITool{Name: "getFinancialHealth", Description: "Deterministic financial health score, components and recommendations.", Parameters: objectSchema(map[string]any{})},
			run: func(ctx context.Context, _ json.RawMessage) (any, error) {
				return s.analytics.FinancialHealth(false)
			},
		},
		{
			definition: domain.AITool{Name: "getLoans", Description: "Loans with monthly and yearly debt, paid and remaining.", Parameters: objectSchema(map[string]any{})},
			run: func(ctx context.Context, _ json.RawMessage) (any, error) {
				loans, err := s.db.ListLoans(false)
				if err != nil {
					return nil, err
				}
				return s.loanContexts(loans), nil
			},
		},
		{
			definition: domain.AITool{Name: "getGoals", Description: "Savings goals with target, current amount, deadline and status.", Parameters: objectSchema(map[string]any{})},
			run: func(ctx context.Context, _ json.RawMessage) (any, error) {
				goals, err := s.db.ListGoals(false)
				if err != nil {
					return nil, err
				}
				return s.goalContexts(goals), nil
			},
		},
	}
	byName := make(map[string]chatTool, len(tools))
	for _, tool := range tools {
		byName[tool.definition.Name] = tool
	}
	return byName
}

func (s *Service) transactionsTool(ctx context.Context, arguments json.RawMessage) (any, error) {
	var input struct {
		Limit    int    `json:"limit"`
		From     string `json:"from"`
		To       string `json:"to"`
		Category string `json:"category"`
		Payee    string `json:"payee"`
		Account  string `json:"account"`
	}
	if err := decodeArguments(arguments, &input); err != nil {
		return nil, err
	}
	if input.Limit <= 0 {
		input.Limit = defaultToolTransactions
	}
	input.Limit = min(input.Limit, maxToolTransactions)
	from, to := time.Time{}, time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
	for _, bound := range []struct {
		raw    string
		target *time.Time
	}{{input.From, &from}, {input.To, &to}} {
		if raw := strings.TrimSpace(bound.raw); raw != "" {
			parsed, err := time.Parse(time.DateOnly, raw)
			if err != nil {
				return nil, fmt.Errorf("%w: dates must be YYYY-MM-DD", domain.ErrValidation)
			}
			*bound.target = parsed
		}
	}
	transactions, err := s.db.ListTransactions(false)
	if err != nil {
		return nil, err
	}
	payee := strings.ToLower(strings.TrimSpace(input.Payee))
	account := strings.ToLower(strings.TrimSpace(input.Account))
	matched := []domain.Transaction{}
	for _, t := range recent(usable(transactions), from, to, 0) {
		if input.Category != "" && !strings.EqualFold(t.Category, strings.TrimSpace(input.Category)) {
			continue
		}
		if account != "" && !strings.Contains(strings.ToLower(t.AccountName), account) {
			continue
		}
		if payee != "" && !strings.Contains(strings.ToLower(t.Payee), payee) {
			continue
		}
		matched = append(matched, t)
	}
	total := len(matched)
	if total > input.Limit {
		matched = matched[:input.Limit]
	}
	return map[string]any{"total": total, "transactions": s.txContexts(matched)}, nil
}
