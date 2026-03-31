package service

import (
	"apps/wealth-management-engine/domain"
	"apps/wealth-management-engine/port"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type toolIntegrationService struct {
	balanceTool port.BalanceTool
	marketSkill port.MarketAnalysisSkill
}

func NewToolIntegrationService(balanceTool port.BalanceTool, marketSkill port.MarketAnalysisSkill) port.ToolIntegrationService {
	return &toolIntegrationService{
		balanceTool: balanceTool,
		marketSkill: marketSkill,
	}
}

func (s *toolIntegrationService) RunConversation(ctx context.Context, prompt string) (domain.ToolConversation, error) {
	account := extractAccount(prompt)
	ticker := extractTicker(prompt)

	turns := []domain.ConversationTurn{
		{Role: "user", Type: "message", Content: prompt},
	}

	getBalanceCall := map[string]any{
		"name": "GetBalance",
		"arguments": map[string]any{
			"account": account,
		},
	}
	getBalanceJSON, _ := json.Marshal(getBalanceCall)
	turns = append(turns, domain.ConversationTurn{
		Role:         "assistant",
		Type:         "tool_call",
		ToolCallJSON: string(getBalanceJSON),
	})

	balance, err := s.balanceTool.GetBalance(ctx, account)
	if err != nil {
		return domain.ToolConversation{}, err
	}
	turns = append(turns, domain.ConversationTurn{
		Role:    "tool",
		Type:    "tool_result",
		Content: fmt.Sprintf(`{"name":"GetBalance","account":"%s","balance":%.2f}`, account, balance),
	})

	marketCall := map[string]any{
		"name": "MarketAnalysis",
		"arguments": map[string]any{
			"ticker": ticker,
		},
	}
	marketCallJSON, _ := json.Marshal(marketCall)
	turns = append(turns, domain.ConversationTurn{
		Role:         "assistant",
		Type:         "tool_call",
		ToolCallJSON: string(marketCallJSON),
	})

	analysis, err := s.marketSkill.Analyze(ctx, ticker)
	if err != nil {
		return domain.ToolConversation{}, err
	}
	turns = append(turns, domain.ConversationTurn{
		Role:    "tool",
		Type:    "tool_result",
		Content: fmt.Sprintf(`{"name":"MarketAnalysis","ticker":"%s","summary":%q}`, ticker, analysis),
	})

	turns = append(turns, domain.ConversationTurn{
		Role:    "assistant",
		Type:    "message",
		Content: fmt.Sprintf("For account %s, current balance is %.2f. Market analysis for %s: %s", account, balance, ticker, analysis),
	})

	return domain.ToolConversation{Turns: turns}, nil
}

func extractAccount(prompt string) string {
	lower := strings.ToLower(prompt)
	if strings.Contains(lower, "cash") {
		return "Cash"
	}
	if strings.Contains(lower, "zalo") {
		return "ZaloPay"
	}
	return "Cash"
}

func extractTicker(prompt string) string {
	upper := strings.ToUpper(prompt)
	tickers := []string{"BTC", "ETH", "VNINDEX", "AAPL", "TSLA"}
	for _, ticker := range tickers {
		if strings.Contains(upper, ticker) {
			return ticker
		}
	}
	return "BTC"
}
