package ai

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
)

const commandsReply = `{"executable_commands":[{"action":"allocate","exact_target":"VN30 ETF","execution_logic":"Trend is up."},{"action":"OBSERVE","target":"BTC","rationale":"Wait."}]}`

// thinkTankRoute fails the main think-tank prompt on both models so the
// fallback pair runs, and answers every other phase.
func thinkTankRoute(request domain.AIRequest) reply {
	system := request.Messages[0].Content
	switch {
	case strings.Contains(system, "Debate."):
		return reply{err: fmt.Errorf("%w: gateway down", domain.ErrAIUnavailable)}
	case strings.Contains(system, "Short debate"):
		return reply{content: "Experts disagree."}
	case strings.Contains(system, "Synthesise"):
		return reply{content: "Hold cash, add VN30."}
	case strings.Contains(system, "Act on"):
		return reply{content: commandsReply}
	}
	return reply{err: errors.New("unexpected prompt")}
}

func TestInvestmentAnalysisStreamsPhasesWithFallbackAndCommands(t *testing.T) {
	client := &scriptedClient{route: thinkTankRoute}
	search := &fakeSearch{results: newsResults}
	var events []port.InvestmentEvent
	err := newMarketService(t, marketDatabase(), client, search, nil).InvestmentAnalysis(context.Background(), port.InvestmentAnalysisRequest{}, func(event port.InvestmentEvent) error {
		events = append(events, event)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var trail []string
	for _, event := range events {
		trail = append(trail, strings.Trim(event.Type+":"+event.Phase+":"+event.Status, ":"))
	}
	want := []string{
		"phase:context:running", "context", "phase:context:ok",
		"phase:think-tank:running", "phase:think-tank:fallback", "phase:think-tank:ok", "message:think-tank",
		"phase:synthesis:running", "phase:synthesis:ok", "message:synthesis",
		"phase:actions:running", "phase:actions:ok", "actions:actions", "done",
	}
	if !slices.Equal(trail, want) {
		t.Fatalf("unexpected event trail:\n%v\nwant\n%v", trail, want)
	}
	commands := events[12].Commands
	if len(commands) != 2 || commands[0].Action != domain.ActionAllocate || commands[0].Target != "VN30 ETF" || commands[0].Rationale != "Trend is up." {
		t.Fatalf("unexpected commands %+v", commands)
	}
	if events[6].Content != "Experts disagree." || events[9].Content != "Hold cash, add VN30." {
		t.Fatalf("unexpected messages %+v %+v", events[6], events[9])
	}
	if len(search.queries) != 3 || !slices.Contains(search.queries, "crypto forecast BTC October 2026") {
		t.Fatalf("expected the three legacy searches, got %v", search.queries)
	}

	var debate string
	for _, request := range client.requests {
		if strings.Contains(request.Messages[0].Content, "Debate.") {
			debate = request.Messages[0].Content
		}
	}
	for _, fragment := range []string{"VND 40000000", "crypto 0.5 BTC", "p2p 26500", "US N/A: US market coverage", "VN Steady climb", "VN-Index: 1300 (1.25%)", "VNINDEX   : n/a | 0.50", `[1] "Rates cut"`, "Binance [name]", `"symbol":"FUND-A"`, `"annualisedReturnPct":10`, `"vsDeposit":"ahead"`, `"volatileSharePct":100`, `"cryptoShareOfNetWorthPct":75`, `"fallPct":30`} {
		if !strings.Contains(debate, fragment) {
			t.Errorf("think-tank prompt is missing %q:\n%s", fragment, debate)
		}
	}
	if strings.Contains(debate, "Nguyen Van A") || strings.Contains(debate, "Platform") {
		t.Fatalf("names must be masked: %s", debate)
	}
}

func TestInvestmentAnalysisFailsAPhaseWithoutFallbackAndToleratesMissingContext(t *testing.T) {
	db := marketDatabase()
	delete(db.AIContent.Prompts, phaseThinkTank.fallbackTask)
	client := &scriptedClient{route: thinkTankRoute}
	svc := newMarketService(t, db, client, nil, nil)
	svc.market = &fakeAnalysis{pulseErr: errors.New("vnstock down")}
	var events []port.InvestmentEvent
	err := svc.InvestmentAnalysis(context.Background(), port.InvestmentAnalysisRequest{}, func(event port.InvestmentEvent) error {
		events = append(events, event)
		return nil
	})
	if !errors.Is(err, domain.ErrAIUnavailable) || !strings.Contains(err.Error(), "think-tank phase") {
		t.Fatalf("expected the think-tank failure, got %v", err)
	}
	last := events[len(events)-1]
	if last.Status != "error" || last.Phase != "think-tank" {
		t.Fatalf("expected an error phase event, got %+v", last)
	}
	debate := client.requests[0].Messages[0].Content
	if !strings.Contains(debate, "NO WEB RESULTS") || !strings.Contains(debate, "market pulse failed (vnstock down)") {
		t.Fatalf("missing context must be stated: %s", debate)
	}

	svc.portfolio = nil
	if err := svc.InvestmentAnalysis(context.Background(), port.InvestmentAnalysisRequest{}, func(port.InvestmentEvent) error { return nil }); !errors.Is(err, domain.ErrAIUnavailable) {
		t.Fatalf("expected the portfolio to be required, got %v", err)
	}
}
