package ai

import (
	"apps/wealth-management/domain"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type chatTool struct {
	definition domain.AITool
	run        func(ctx context.Context, arguments json.RawMessage) (any, error)
}

// runTool runs a tool call and returns its result as is and as the JSON
// the model reads.
func (s *Service) runTool(ctx context.Context, tools map[string]chatTool, toolCall domain.AIToolCall) (any, string, error) {
	tool, ok := tools[toolCall.Name]
	if !ok {
		return nil, "", fmt.Errorf("%w: unknown tool %q", domain.ErrValidation, toolCall.Name)
	}
	arguments := json.RawMessage(strings.TrimSpace(toolCall.Arguments))
	if len(arguments) == 0 {
		arguments = json.RawMessage("{}")
	}
	result, err := tool.run(ctx, arguments)
	if err != nil {
		return nil, "", err
	}
	if form, ok := result.(domain.FillChatForm); ok {
		return form, s.masker.Mask(toJSON(fillToolAnswer(form))), nil
	}
	return result, toJSON(result), nil
}

// fillToolAnswer is what the model reads back from fillForm: the checked
// values and notes, and that the user saves the form, not the chat.
func fillToolAnswer(form domain.FillChatForm) map[string]any {
	return map[string]any{
		"form":   form.Title,
		"values": form.Result.Values,
		"rows":   form.Result.Rows,
		"filled": form.Result.Filled,
		"notes":  form.Result.Notes,
		"next":   "The user sees an Open the form button for this. Nothing is saved until they check the form and save it there. Mention any notes.",
	}
}

// fillFormTool runs a form fill for the chat (WM-FILL-7); the stream sends
// the result as a form event.
func (s *Service) fillFormTool() chatTool {
	forms := domain.ChatFillForms()
	keys := make([]string, len(forms))
	described := make([]string, len(forms))
	for index, form := range forms {
		keys[index] = form.Key
		described[index] = fmt.Sprintf("%s (%s)", form.Key, form.Title)
	}
	return chatTool{
		definition: domain.AITool{
			Name: "fillForm",
			Description: "Fills one of the app's forms from the user's words, for the user to check and save; it saves nothing. " +
				"Use it when the user asks to add, record, set or change something these forms cover: " + strings.Join(described, ", ") + ".",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"form": map[string]any{"type": "string", "enum": keys},
					"text": map[string]any{"type": "string", "description": "What to fill, in the user's words with every detail they gave."},
				},
				"required": []string{"form", "text"},
			},
		},
		run: func(ctx context.Context, arguments json.RawMessage) (any, error) {
			var input struct {
				Form string `json:"form"`
				Text string `json:"text"`
			}
			if err := decodeArguments(arguments, &input); err != nil {
				return nil, err
			}
			form, ok := domain.FillFormFor(input.Form)
			if !ok || form.Page == "" {
				return nil, fmt.Errorf("%w: the chat cannot fill the %q form", domain.ErrValidation, input.Form)
			}
			result, err := s.Fill(ctx, domain.FillRequest{Form: form.Key, Text: input.Text})
			if err != nil {
				return nil, err
			}
			return domain.FillChatForm{Form: form.Key, Title: form.Title, Page: form.Page, Text: strings.TrimSpace(input.Text), Result: result}, nil
		},
	}
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

// chatTools are the read-only ledger tools of the chat (WM-AI-11, WM-MCP-4),
// the market tools that are wired and fillForm, which fills a form the user
// saves (WM-FILL-7).
func (s *Service) chatTools() map[string]chatTool {
	str := func(description string) map[string]any {
		return map[string]any{"type": "string", "description": description}
	}
	tools := []chatTool{
		{
			definition: domain.AITool{Name: "getAccountBalances", Description: "Current balance of every account plus assets, liabilities and net worth; money lent that people owe is an asset (receivables), money owed to people a liability (payables).", Parameters: objectSchema(map[string]any{})},
			run: func(ctx context.Context, _ json.RawMessage) (any, error) {
				ledger, err := s.db.ReadLedger(false)
				if err != nil {
					return nil, err
				}
				return map[string]any{"accounts": s.accountContexts(ledger.Accounts), "balanceSheet": domain.ComputeBalanceSheet(ledger.Accounts, ledger.Transactions)}, nil
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
			definition: domain.AITool{Name: "getYearlyCashFlow", Description: "Income and expense of a calendar year: per month, per category per month, year total, monthly average, change vs the same months of the previous year, savings verdict, year-end projection and unusual months.", Parameters: objectSchema(map[string]any{"year": map[string]any{"type": "integer", "description": "Calendar year; 0 or empty is the current year"}})},
			run: func(ctx context.Context, arguments json.RawMessage) (any, error) {
				var input struct {
					Year int `json:"year"`
				}
				if err := decodeArguments(arguments, &input); err != nil {
					return nil, err
				}
				return s.analytics.YearlyCashFlow(input.Year, false)
			},
		},
		{
			definition: domain.AITool{Name: "getFinancialHealth", Description: "Deterministic financial health score, components and recommendations.", Parameters: objectSchema(map[string]any{})},
			run: func(ctx context.Context, _ json.RawMessage) (any, error) {
				return s.analytics.FinancialHealth(false)
			},
		},
		{
			definition: domain.AITool{Name: "getMoneyLent", Description: "Money lent to people: totals owed to the user and owed by the user, and each open balance (outstanding, lent, repaid, last activity, days outstanding; negative outstanding = the user owes that person).", Parameters: objectSchema(map[string]any{})},
			run: func(ctx context.Context, _ json.RawMessage) (any, error) {
				lent, err := s.analytics.MoneyLent(false)
				if err != nil {
					return nil, err
				}
				return s.moneyLentContext(lent), nil
			},
		},
		{
			definition: domain.AITool{Name: "getGoals", Description: "The goal plan: goals are claims on one pooled savings balance filled by priority (emergency fund first). Per goal target, funded, remaining, target date, required per month, projected finish and pace; the pool, average monthly savings, total required per month, the gap and trade-offs.", Parameters: objectSchema(map[string]any{})},
			run: func(ctx context.Context, _ json.RawMessage) (any, error) {
				plan, err := s.analytics.GoalPlan(false)
				if err != nil {
					return nil, err
				}
				return s.goalPlanContext(plan), nil
			},
		},
		{
			definition: domain.AITool{Name: "getBills", Description: "Bills and safe-to-spend in tiers until the next wage: ready money (bank, cash, active and rarely used accounts without crypto or accounts marked not ready to use), committed (unpaid bills and card payments due before the next income, overdue ones included, one due on payday being paid by that wage, plus fixed costs still to pay such as rent), essentials (needs such as groceries until payday) and freeToSpend (the rest, the only money for wants; use it for 'can I afford' questions), the calendar of recurring bills, income and card payments for the next 30 days (paid, due, overdue), the confirmed recurring items and the suggestions found in history.", Parameters: objectSchema(map[string]any{})},
			run: func(ctx context.Context, _ json.RawMessage) (any, error) {
				bills, err := s.analytics.Bills(false)
				if err != nil {
					return nil, err
				}
				return s.billsContext(bills), nil
			},
		},
	}
	if s.recommend != nil {
		tools = append(tools, chatTool{
			definition: domain.AITool{Name: "getRecommendations", Description: "The recommendations list: advice from the reviews and the engine rules with its action, estimated VND impact per year, evidence, the page it belongs to, the status the user set (open, snoozed, done, dismissed) and the outcome a later review found; plus the open count and their total yearly impact.", Parameters: objectSchema(map[string]any{
				"status": map[string]any{"type": "string", "enum": []string{"open", "snoozed", "done", "dismissed"}, "description": "Only items with this status; empty is every status"},
				"page":   str("Only items for this app route, e.g. /budget"),
			})},
			run: func(ctx context.Context, arguments json.RawMessage) (any, error) {
				var filter domain.RecommendationFilter
				if err := decodeArguments(arguments, &filter); err != nil {
					return nil, err
				}
				list, err := s.recommend.List(ctx, filter, false)
				if err != nil {
					return nil, err
				}
				for i, item := range list.Items {
					item.Title, item.Action, item.Evidence, item.Outcome = s.masker.Mask(item.Title), s.masker.Mask(item.Action), s.masker.Mask(item.Evidence), s.masker.Mask(item.Outcome)
					list.Items[i] = item
				}
				return list, nil
			},
		})
	}
	tools = append(tools, s.marketTools(str)...)
	tools = append(tools, s.fillFormTool())
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
